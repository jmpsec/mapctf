package backend

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// pgTestDSNEnv is the environment variable that, when set to a Postgres DSN,
// enables the Postgres-backed integration tests in this file. These tests
// reproduce the original concurrent-AutoMigrate race against a real Postgres
// instance; they are skipped by default so `make test` (sqlite-only) stays
// hermetic.
const pgTestDSNEnv = "MAPCTF_PG_TEST_DSN"

// testMigrateModel is a throwaway model used only by SafeAutoMigrate tests so
// the migration has a table to create. Each Postgres integration test uses a
// distinct model (and therefore table) to avoid cross-test interference.
type testMigrateModel struct {
	gorm.Model
	Name string `gorm:"index"`
}

// raceModelA/B/C are distinct models used by the Postgres concurrent-migration
// test so each goroutine migrates a different table, mirroring how multiple
// services each migrate their own tables on a shared database.
type raceModelA struct {
	gorm.Model
	Value string
}

type raceModelB struct {
	gorm.Model
	Value string
}

type raceModelC struct {
	gorm.Model
	Value string
}

func openSQLiteForMigrateTest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	return db
}

// openPostgresForMigrateTest opens a connection to the Postgres instance
// identified by the MAPCTF_PG_TEST_DSN env var. It drops the supplied models
// first so each test starts from a clean (table-less) state, which is required
// to reproduce the original first-boot race. The test is skipped if the env
// var is unset.
func openPostgresForMigrateTest(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	dsn := os.Getenv(pgTestDSNEnv)
	if dsn == "" {
		t.Skipf("skipping Postgres integration test: %s not set", pgTestDSNEnv)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open postgres: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	// Drop tables up front so the test reproduces the fresh-DB condition.
	for _, m := range models {
		_ = db.Migrator().DropTable(m)
	}
	return db
}

// TestSafeAutoMigrateNilDB ensures the helper guards against nil connections.
func TestSafeAutoMigrateNilDB(t *testing.T) {
	if err := SafeAutoMigrate(nil, &testMigrateModel{}); err == nil {
		t.Fatal("expected error when passing nil db, got nil")
	}
}

// TestSafeAutoMigrateSQLiteDelegates verifies that on non-Postgres drivers the
// helper simply delegates to gorm's AutoMigrate and creates the table.
func TestSafeAutoMigrateSQLiteDelegates(t *testing.T) {
	db := openSQLiteForMigrateTest(t)

	if err := SafeAutoMigrate(db, &testMigrateModel{}); err != nil {
		t.Fatalf("SafeAutoMigrate failed on sqlite: %v", err)
	}
	if !db.Migrator().HasTable(&testMigrateModel{}) {
		t.Fatal("expected test_migrate_models table to exist after SafeAutoMigrate")
	}
}

// TestSafeAutoMigrateIdempotent verifies that calling SafeAutoMigrate twice on
// a non-Postgres driver succeeds (mirroring AutoMigrate's idempotence).
func TestSafeAutoMigrateIdempotent(t *testing.T) {
	db := openSQLiteForMigrateTest(t)

	for i := 0; i < 2; i++ {
		if err := SafeAutoMigrate(db, &testMigrateModel{}); err != nil {
			t.Fatalf("SafeAutoMigrate pass %d failed: %v", i+1, err)
		}
	}
	if !db.Migrator().HasTable(&testMigrateModel{}) {
		t.Fatal("expected test_migrate_models table to exist after repeated SafeAutoMigrate")
	}
}

// TestSafeAutoMigratePropagatesMigrateError verifies that underlying AutoMigrate
// errors are surfaced. We force an error by passing a model whose table cannot
// be created because the connection is closed.
func TestSafeAutoMigratePropagatesMigrateError(t *testing.T) {
	db := openSQLiteForMigrateTest(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sqlDB: %v", err)
	}
	sqlDB.Close()

	if err := SafeAutoMigrate(db, &testMigrateModel{}); err == nil {
		t.Fatal("expected error when AutoMigrate runs against a closed sqlite connection, got nil")
	}
}

// TestSafeAutoMigrateConcurrentSQLite verifies that sequential SafeAutoMigrate
// calls against a shared in-memory sqlite database succeed. Concurrent
// migration is intentionally only serialized on Postgres (where the
// pg_type duplicate-key race occurs); sqlite is used only in tests, so we
// only exercise the sequential path here.
func TestSafeAutoMigrateConcurrentSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open shared sqlite: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	for i := 0; i < 4; i++ {
		if err := SafeAutoMigrate(db, &testMigrateModel{}); err != nil {
			t.Fatalf("sequential SafeAutoMigrate pass %d failed: %v", i+1, err)
		}
	}
	if !db.Migrator().HasTable(&testMigrateModel{}) {
		t.Fatal("expected test_migrate_models table to exist after sequential SafeAutoMigrate")
	}
}

// TestIsPostgres verifies the dialector detection used to decide whether to
// take the advisory-lock path.
func TestIsPostgres(t *testing.T) {
	db := openSQLiteForMigrateTest(t)
	if isPostgres(db) {
		t.Fatal("expected isPostgres to be false for sqlite connection")
	}
	if isPostgres(nil) {
		t.Fatal("expected isPostgres(nil) to be false")
	}
}

// TestWithMigrateLockTimeoutConstant sanity-checks the configured wait is
// positive so we never return ErrMigrationLockTimeout instantly.
func TestWithMigrateLockTimeoutConstant(t *testing.T) {
	if lockWait <= 0 {
		t.Fatalf("lockWait must be positive, got %v", lockWait)
	}
	if lockWait > 5*time.Minute {
		t.Fatalf("lockWait unreasonably large: %v", lockWait)
	}
}

// TestSafeAutoMigratePostgresConcurrent is an integration test that reproduces
// the original first-boot race condition: multiple services booting
// simultaneously each call AutoMigrate against the same fresh Postgres
// database. Without the advisory lock this previously failed with
// "duplicate key value violates unique constraint pg_type_typname_nsp_index".
//
// With SafeAutoMigrate, the Postgres advisory lock serializes the concurrent
// migrations so all callers succeed and all tables are created.
//
// To run: set MAPCTF_PG_TEST_DSN to a Postgres DSN, e.g.
//
//	MAPCTF_PG_TEST_DSN="host=127.0.0.1 port=5432 dbname=mapctf-db user=mapctf password=mapctf sslmode=disable" go test ./pkg/backend/ -run TestSafeAutoMigratePostgresConcurrent
func TestSafeAutoMigratePostgresConcurrent(t *testing.T) {
	db := openPostgresForMigrateTest(t, &raceModelA{}, &raceModelB{}, &raceModelC{})

	if !isPostgres(db) {
		t.Fatal("expected isPostgres to be true for postgres connection")
	}

	// Each goroutine migrates a distinct model, mirroring how mapctf-api and
	// mapctf-map each migrate their own tables on a shared database at boot.
	type job struct {
		name  string
		model interface{}
	}
	jobs := []job{
		{"raceModelA", &raceModelA{}},
		{"raceModelB", &raceModelB{}},
		{"raceModelC", &raceModelC{}},
	}

	workers := len(jobs)
	var wg sync.WaitGroup
	wg.Add(workers)
	errs := make(chan error, workers)
	start := make(chan struct{})

	for _, j := range jobs {
		go func(j job) {
			defer wg.Done()
			<-start
			if err := SafeAutoMigrate(db, j.model); err != nil {
				errs <- fmt.Errorf("migrate %s: %w", j.name, err)
			}
		}(j)
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent SafeAutoMigrate failed: %v", err)
	}
	for _, j := range jobs {
		if !db.Migrator().HasTable(j.model) {
			t.Fatalf("expected table for %s to exist after concurrent SafeAutoMigrate", j.name)
		}
	}
}

// TestSafeAutoMigratePostgresIdempotent verifies that repeated SafeAutoMigrate
// calls on Postgres succeed when the table already exists (the steady-state
// restart path).
func TestSafeAutoMigratePostgresIdempotent(t *testing.T) {
	db := openPostgresForMigrateTest(t, &testMigrateModel{})

	for i := 0; i < 3; i++ {
		if err := SafeAutoMigrate(db, &testMigrateModel{}); err != nil {
			t.Fatalf("SafeAutoMigrate pass %d failed on postgres: %v", i+1, err)
		}
	}
	if !db.Migrator().HasTable(&testMigrateModel{}) {
		t.Fatal("expected test_migrate_models table to exist after repeated SafeAutoMigrate")
	}
}

// TestSafeAutoMigratePostgresSerializes verifies that the advisory lock
// actually serializes concurrent callers: when one goroutine holds the lock
// inside a long-running fn, a second caller blocks until the first releases
// it, and the observed ordering proves serialization (no overlap).
func TestSafeAutoMigratePostgresSerializes(t *testing.T) {
	db := openPostgresForMigrateTest(t, &raceModelA{}, &raceModelB{})

	var overlapping atomic.Int32
	var inLock atomic.Int32
	var maxConcurrent atomic.Int32

	// slowFn wraps AutoMigrate with a delay so we can observe overlap. It
	// records how many callers were inside the locked section at once.
	slowFn := func(tx *gorm.DB) error {
		cur := inLock.Add(1)
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		defer inLock.Add(-1)
		time.Sleep(200 * time.Millisecond)
		// Actually run the migration so the table is created.
		return tx.AutoMigrate(&raceModelA{})
	}

	var wg sync.WaitGroup
	wg.Add(2)
	start := make(chan struct{})
	go func() {
		defer wg.Done()
		<-start
		_ = withMigrateLock(db, slowFn)
	}()
	go func() {
		defer wg.Done()
		<-start
		// Migrate a different model; this call must wait for the first to
		// release the advisory lock.
		_ = withMigrateLock(db, func(tx *gorm.DB) error {
			// Track overlap explicitly: if both callers are here at once,
			// inLock will be 2.
			cur := inLock.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			defer inLock.Add(-1)
			return tx.AutoMigrate(&raceModelB{})
		})
	}()
	close(start)
	wg.Wait()

	if maxConcurrent.Load() > 1 {
		overlapping.Store(maxConcurrent.Load())
	}
	if got := overlapping.Load(); got > 1 {
		t.Fatalf("advisory lock did not serialize callers: %d concurrent inside locked section", got)
	}
}

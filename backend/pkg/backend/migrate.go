package backend

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// migrateLockKey is a stable session-level advisory lock key used to serialize
// AutoMigrate runs across concurrent services (e.g. mapctf-api and mapctf-map)
// that share the same database. Session-level locks are released when the
// transaction ends, so a single AutoMigrate call holds the lock for its
// duration and the next caller blocks until it finishes.
const migrateLockKey = 0x4D415043 // "MAPC"

// ErrMigrationLockTimeout is returned when SafeAutoMigrate cannot acquire the
// migration advisory lock within lockWait.
var ErrMigrationLockTimeout = errors.New("timed out waiting for migration advisory lock")

// lockWait is the maximum time SafeAutoMigrate will wait to acquire the
// migration advisory lock before giving up.
const lockWait = 60 * time.Second

// SafeAutoMigrate runs gorm's AutoMigrate for the given models while holding a
// serializing transaction-level advisory lock on Postgres-backed connections.
// On other drivers (MySQL, SQLite) it delegates directly to AutoMigrate,
// since those drivers do not suffer from the same pg_type duplicate-key race
// during concurrent CREATE TABLE attempts.
//
// Holding the lock ensures that when two services boot simultaneously against
// a fresh database, only one performs the initial CREATE TABLE; the other
// waits, then AutoMigrate observes the table already exists and skips
// creation, avoiding the "duplicate key value violates unique constraint
// pg_type_typname_nsp_index" error. The transaction-level lock
// (pg_try_advisory_xact_lock) is released automatically when the surrounding
// transaction commits or rolls back, so no explicit unlock is required.
func SafeAutoMigrate(db *gorm.DB, models ...interface{}) error {
	if db == nil {
		return fmt.Errorf("database connection cannot be nil")
	}
	if !isPostgres(db) {
		return db.AutoMigrate(models...)
	}
	return withMigrateLock(db, func(tx *gorm.DB) error {
		return tx.AutoMigrate(models...)
	})
}

// withMigrateLock acquires a transaction-level advisory lock on the supplied
// Postgres connection, runs fn inside that transaction, and commits. The lock
// is released automatically on commit/rollback. If the lock cannot be
// acquired within lockWait it returns ErrMigrationLockTimeout.
func withMigrateLock(db *gorm.DB, fn func(*gorm.DB) error) error {
	deadline := time.Now().Add(lockWait)
	tx := db.Begin()
	defer func() {
		// Rollback is a no-op if the transaction was already committed; it
		// also ensures the transaction-level advisory lock is released on
		// the error/timeout paths.
		_ = tx.Rollback().Error
	}()
	for {
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", migrateLockKey).Scan(&locked).Error; err != nil {
			return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
		}
		if locked {
			break
		}
		if time.Now().After(deadline) {
			return ErrMigrationLockTimeout
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit().Error
}

// isPostgres reports whether the supplied gorm.DB is backed by the Postgres
// driver, by inspecting the dialector name.
func isPostgres(db *gorm.DB) bool {
	return db != nil && db.Name() == DBTypePostgres
}

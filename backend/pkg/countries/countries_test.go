package countries

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testUUID1 = "test-uuid-1"
	testUUID2 = "test-uuid-2"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

func writeSeedFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "countries-seed.json")
	if err := os.WriteFile(file, []byte(contents), 0o644); err != nil {
		t.Fatalf("Failed to write seed file: %v", err)
	}
	return file
}

func TestCreateCountries(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create CountriesManager: %v", err)
	}
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	if manager.DB == nil {
		t.Fatal("Expected non-nil DB in manager")
	}
	if manager.UUID != testUUID1 {
		t.Fatalf("Expected UUID %q, got %q", testUUID1, manager.UUID)
	}
	if !db.Migrator().HasTable(&MapCountry{}) {
		t.Fatal("Expected map_countries table to be created")
	}
}

func TestCreateCountriesWithNilDB(t *testing.T) {
	_, err := CreateCountries(nil, testUUID1)
	if err == nil {
		t.Fatal("Expected error when DB is nil")
	}
}

func TestCreateCountriesAutoMigrateError(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get raw db: %v", err)
	}
	_ = sqlDB.Close()

	_, err = CreateCountries(db, testUUID1)
	if err == nil {
		t.Fatal("Expected AutoMigrate error with closed DB")
	}
}

func TestLoadCountriesSeedData(t *testing.T) {
	seed := writeSeedFile(t, `[
	  {"Name":"Spain","CountryCode":"ES","LandPath":"L1","LandClass":"lc1","LandStyle":"ls1","MarkerPath":"M1","MarkerClass":"mc1","MarkerStyle":"ms1","MarkerTransform":"mt1"},
	  {"Name":"France","CountryCode":"FR","LandPath":"L2","LandClass":"lc2","LandStyle":"ls2","MarkerPath":"M2","MarkerClass":"mc2","MarkerStyle":"ms2","MarkerTransform":"mt2"}
	]`)

	items, err := loadCountriesSeedData(seed)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}
	if items[0].CountryCode != "ES" || items[1].CountryCode != "FR" {
		t.Fatalf("Unexpected country codes: %+v", items)
	}
}

func TestLoadCountriesSeedDataMissingFile(t *testing.T) {
	_, err := loadCountriesSeedData("/tmp/does-not-exist-countries-seed.json")
	if err == nil {
		t.Fatal("Expected error for missing seed file")
	}
}

func TestLoadCountriesSeedDataInvalidJSON(t *testing.T) {
	seed := writeSeedFile(t, `{invalid-json}`)
	_, err := loadCountriesSeedData(seed)
	if err == nil {
		t.Fatal("Expected JSON unmarshal error")
	}
}

func TestInitializeCountries(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	seed := writeSeedFile(t, `[
	  {"Name":"Spain","CountryCode":"ES","LandPath":"L1","LandClass":"lc1","LandStyle":"ls1","MarkerPath":"M1","MarkerClass":"mc1","MarkerStyle":"ms1","MarkerTransform":"mt1"},
	  {"Name":"France","CountryCode":"FR","LandPath":"L2","LandClass":"lc2","LandStyle":"ls2","MarkerPath":"M2","MarkerClass":"mc2","MarkerStyle":"ms2","MarkerTransform":"mt2"}
	]`)

	stats, err := manager.InitializeCountries(seed)
	if err != nil {
		t.Fatalf("InitializeCountries failed: %v", err)
	}
	if stats.TotalCountries != 2 || stats.InsertedCountries != 2 || stats.ExistingCountries != 0 {
		t.Fatalf("Unexpected stats on first initialize: %+v", stats)
	}

	stats, err = manager.InitializeCountries(seed)
	if err != nil {
		t.Fatalf("InitializeCountries second run failed: %v", err)
	}
	if stats.TotalCountries != 2 || stats.InsertedCountries != 0 || stats.ExistingCountries != 2 {
		t.Fatalf("Unexpected stats on second initialize: %+v", stats)
	}
}

func TestInitializeCountriesInvalidSeed(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	stats, err := manager.InitializeCountries("/tmp/missing-seed.json")
	if err == nil {
		t.Fatal("Expected error for missing seed file")
	}
	if stats == nil {
		t.Fatal("Expected non-nil stats even on error")
	}
	if stats.TotalCountries != 0 || stats.InsertedCountries != 0 || stats.ExistingCountries != 0 {
		t.Fatalf("Expected zero stats on seed load failure, got: %+v", stats)
	}
}

func TestCreateAndQueryHelpers(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Create(MapCountry{Name: "Spain", CountryCode: "ES", Active: true, Assigned: false})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	err = manager.Create(MapCountry{Name: "France", CountryCode: "FR", Active: true, Assigned: true, ChallengeID: 10})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	err = manager.Create(MapCountry{Name: "Germany", CountryCode: "DE", Active: false, Assigned: false})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	all, err := manager.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("Expected 3 countries, got %d", len(all))
	}

	active, err := manager.GetActive()
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("Expected 2 active countries, got %d", len(active))
	}

	available, err := manager.GetAvailable()
	if err != nil {
		t.Fatalf("GetAvailable failed: %v", err)
	}
	if len(available) != 1 {
		t.Fatalf("Expected 1 available country, got %d", len(available))
	}
	if available[0].CountryCode != "ES" {
		t.Fatalf("Expected ES to be available, got %s", available[0].CountryCode)
	}
}

func TestExistsAndGetByCode(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if err := manager.Create(MapCountry{Name: "Spain", CountryCode: "ES", Active: true}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	exists, err := manager.Exists("ES")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("Expected ES to exist")
	}

	exists, err = manager.Exists("FR")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("Expected FR not to exist")
	}

	c, err := manager.GetByCode("ES")
	if err != nil {
		t.Fatalf("GetByCode failed: %v", err)
	}
	if c.Name != "Spain" {
		t.Fatalf("Expected country Spain, got %s", c.Name)
	}

	_, err = manager.GetByCode("FR")
	if err == nil {
		t.Fatal("Expected error for missing country")
	}
}

func TestCreateSetsManagerUUID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Create(MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		UUID:        "wrong-uuid",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var c MapCountry
	if err := db.Where("country_code = ?", "ES").First(&c).Error; err != nil {
		t.Fatalf("Failed to load country: %v", err)
	}
	if c.UUID != testUUID1 {
		t.Fatalf("Expected UUID %q, got %q", testUUID1, c.UUID)
	}
}

func TestSetActiveByID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if err := manager.Create(MapCountry{Name: "Spain", CountryCode: "ES", Active: true}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	c, err := manager.GetByCode("ES")
	if err != nil {
		t.Fatalf("GetByCode failed: %v", err)
	}

	if err := manager.SetActiveByID(c.ID, false); err != nil {
		t.Fatalf("SetActiveByID failed: %v", err)
	}
	updated, err := manager.GetByCode("ES")
	if err != nil {
		t.Fatalf("GetByCode after update failed: %v", err)
	}
	if updated.Active {
		t.Fatal("Expected Active=false after update")
	}

	err = manager.SetActiveByID(99999, true)
	if err == nil {
		t.Fatal("Expected not found error for unknown id")
	}
	if !strings.Contains(err.Error(), "country not found") {
		t.Fatalf("Expected 'country not found' error, got: %v", err)
	}
}

func TestAssignAndReleaseCountry(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	if err := manager.Create(MapCountry{Name: "Spain", CountryCode: "ES", Active: true}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := manager.AssignCountryToChallenge("ES", 42); err != nil {
		t.Fatalf("AssignCountryToChallenge failed: %v", err)
	}
	c, err := manager.GetByCode("ES")
	if err != nil {
		t.Fatalf("GetByCode failed: %v", err)
	}
	if c.ChallengeID != 42 || !c.Assigned {
		t.Fatalf("Expected challenge_id=42 and assigned=true, got challenge_id=%d assigned=%v", c.ChallengeID, c.Assigned)
	}

	if err := manager.ReleaseCountry("ES"); err != nil {
		t.Fatalf("ReleaseCountry failed: %v", err)
	}
	c, err = manager.GetByCode("ES")
	if err != nil {
		t.Fatalf("GetByCode failed: %v", err)
	}
	if c.ChallengeID != NoChallengeID || c.Assigned {
		t.Fatalf("Expected challenge_id=%d and assigned=false, got challenge_id=%d assigned=%v", NoChallengeID, c.ChallengeID, c.Assigned)
	}
}

func TestAssignAndReleaseCountryNotFound(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.AssignCountryToChallenge("ZZ", 1)
	if err == nil {
		t.Fatal("Expected error assigning unknown country")
	}
	if !strings.Contains(err.Error(), "country not found") {
		t.Fatalf("Expected 'country not found' error, got: %v", err)
	}

	err = manager.ReleaseCountry("ZZ")
	if err == nil {
		t.Fatal("Expected error releasing unknown country")
	}
	if !strings.Contains(err.Error(), "country not found") {
		t.Fatalf("Expected 'country not found' error, got: %v", err)
	}
}

func TestUUIDIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager1, err := CreateCountries(db, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create manager1: %v", err)
	}
	manager2, err := CreateCountries(db, testUUID2)
	if err != nil {
		t.Fatalf("Failed to create manager2: %v", err)
	}

	if err := manager1.Create(MapCountry{Name: "Spain", CountryCode: "ES", Active: true}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	exists, err := manager2.Exists("ES")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("Expected ES not to exist under manager2 UUID")
	}

	_, err = manager2.GetByCode("ES")
	if err == nil {
		t.Fatal("Expected GetByCode to fail for manager2 UUID")
	}

	if err := manager2.Create(MapCountry{Name: "Spain 2", CountryCode: "ES", Active: true}); err != nil {
		t.Fatalf("Create failed for manager2: %v", err)
	}

	all1, err := manager1.GetAll()
	if err != nil {
		t.Fatalf("manager1 GetAll failed: %v", err)
	}
	all2, err := manager2.GetAll()
	if err != nil {
		t.Fatalf("manager2 GetAll failed: %v", err)
	}
	if len(all1) != 1 || len(all2) != 1 {
		t.Fatalf("Expected 1 country per UUID, got manager1=%d manager2=%d", len(all1), len(all2))
	}
}

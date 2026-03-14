package logs

import (
	"database/sql"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testUUIDA = "tenant-a"
	testUUIDB = "tenant-b"
)

func newTestDB(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}

	return db, sqlDB
}

func newTestManager(t *testing.T) (*LogManager, *sql.DB) {
	t.Helper()

	db, sqlDB := newTestDB(t)
	m, err := CreateLogManager(db)
	if err != nil {
		t.Fatalf("failed to create log manager: %v", err)
	}
	return m, sqlDB
}

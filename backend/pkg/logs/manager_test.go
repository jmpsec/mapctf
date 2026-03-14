package logs

import (
	"strings"
	"testing"
)

func TestCreateLogManager(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		m, err := CreateLogManager(nil)
		if m != nil {
			t.Fatal("expected nil manager")
		}
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "database connection cannot be nil") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("success and migrates all tables", func(t *testing.T) {
		db, sqlDB := newTestDB(t)
		defer func() { _ = sqlDB.Close() }()

		m, err := CreateLogManager(db)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m == nil || m.DB == nil {
			t.Fatal("expected non-nil manager and db")
		}

		if !db.Migrator().HasTable(&ActivityLog{}) {
			t.Fatal("activity_logs table was not migrated")
		}
		if !db.Migrator().HasTable(&Announcement{}) {
			t.Fatal("announcements table was not migrated")
		}
		if !db.Migrator().HasTable(&ScoreboardLog{}) {
			t.Fatal("scoreboard_logs table was not migrated")
		}
		if !db.Migrator().HasTable(&HintsLog{}) {
			t.Fatal("hints_logs table was not migrated")
		}
		if !db.Migrator().HasTable(&FailuresLog{}) {
			t.Fatal("failures_logs table was not migrated")
		}
		if !db.Migrator().HasTable(&RegistrationLog{}) {
			t.Fatal("registration_logs table was not migrated")
		}
	})

	t.Run("automigrate failure", func(t *testing.T) {
		db, sqlDB := newTestDB(t)
		_ = sqlDB.Close()

		m, err := CreateLogManager(db)
		if m != nil {
			t.Fatal("expected nil manager")
		}
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to AutoMigrate table") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

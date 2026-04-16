package chat

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testUUIDA = "chat-test-uuid-a"
	testUUIDB = "chat-test-uuid-b"
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

func setupTestManager(t *testing.T, uuid string) (*ChatManager, *gorm.DB) {
	t.Helper()

	db := setupTestDB(t)
	manager, err := CreateChatManager(db, uuid)
	if err != nil {
		t.Fatalf("Failed to create ChatManager: %v", err)
	}

	return manager, db
}

func TestCreateChatManager(t *testing.T) {
	t.Run("nil db returns error", func(t *testing.T) {
		manager, err := CreateChatManager(nil, testUUIDA)
		if err == nil {
			t.Fatal("expected error when database is nil")
		}
		if manager != nil {
			t.Fatal("expected nil manager when database is nil")
		}
	})

	t.Run("successfully migrates", func(t *testing.T) {
		db := setupTestDB(t)
		manager, err := CreateChatManager(db, testUUIDA)
		if err != nil {
			t.Fatalf("unexpected error creating manager: %v", err)
		}
		if manager.UUID != testUUIDA {
			t.Fatalf("expected UUID %s, got %s", testUUIDA, manager.UUID)
		}
		if !db.Migrator().HasTable(&ChatEntry{}) {
			t.Fatal("expected chat_entries table to be created")
		}
	})

	t.Run("automigrate failure returns error", func(t *testing.T) {
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("failed to get sql db: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("failed to close sql db: %v", err)
		}

		manager, err := CreateChatManager(db, testUUIDA)
		if err == nil {
			t.Fatal("expected automigrate error on closed db")
		}
		if manager != nil {
			t.Fatal("expected nil manager on automigrate failure")
		}
	})
}

func TestNewChatEntry(t *testing.T) {
	manager, _ := setupTestManager(t, testUUIDA)

	entry, err := manager.New("alice", "hello world", testUUIDA, 7, 5)
	if err != nil {
		t.Fatalf("unexpected error from New: %v", err)
	}

	if entry.Username != "alice" {
		t.Fatalf("expected username alice, got %s", entry.Username)
	}
	if entry.Body != "hello" {
		t.Fatalf("expected truncated body hello, got %s", entry.Body)
	}
	if entry.UUID != testUUIDA {
		t.Fatalf("expected uuid %s, got %s", testUUIDA, entry.UUID)
	}
	if entry.TeamID != 7 {
		t.Fatalf("expected team id 7, got %d", entry.TeamID)
	}
}

func TestNewChatEntryRejectsEmptyFields(t *testing.T) {
	manager, _ := setupTestManager(t, testUUIDA)

	tests := []struct {
		name     string
		username string
		body     string
	}{
		{name: "empty username", username: "", body: "hello"},
		{name: "empty body", username: "alice", body: ""},
		{name: "empty both", username: "", body: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.New(tc.username, tc.body, testUUIDA, NoTeamID, DefaultMaxLen)
			if err == nil {
				t.Fatal("expected error for empty username/body")
			}
		})
	}
}

func TestCreateAndGetAllScopeByUUID(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	entryA, err := manager.New("alice", "hello", testUUIDA, 1, DefaultMaxLen)
	if err != nil {
		t.Fatalf("unexpected error creating entryA: %v", err)
	}
	if err := manager.Create(entryA); err != nil {
		t.Fatalf("unexpected error saving entryA: %v", err)
	}

	if err := db.Create(&ChatEntry{
		Username: "bob",
		Body:     "other",
		TeamID:   2,
		UUID:     testUUIDB,
	}).Error; err != nil {
		t.Fatalf("unexpected error creating entryB: %v", err)
	}

	entries, err := manager.GetAll()
	if err != nil {
		t.Fatalf("unexpected error from GetAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 scoped entry, got %d", len(entries))
	}
	if entries[0].Username != "alice" || entries[0].UUID != testUUIDA {
		t.Fatalf("unexpected scoped entry: %+v", entries[0])
	}
}

func TestGetAllReturnsEntriesInCreationOrder(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	baseTime := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	seed := []ChatEntry{
		{Username: "second", Body: "later", TeamID: 1, UUID: testUUIDA, Model: gorm.Model{CreatedAt: baseTime.Add(2 * time.Minute)}},
		{Username: "first", Body: "earlier", TeamID: 1, UUID: testUUIDA, Model: gorm.Model{CreatedAt: baseTime}},
	}
	for _, entry := range seed {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("unexpected seed error: %v", err)
		}
	}

	entries, err := manager.GetAll()
	if err != nil {
		t.Fatalf("unexpected error from GetAll: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Username != "first" || entries[1].Username != "second" {
		t.Fatalf("expected chronological order, got %+v", entries)
	}
}

func TestDeleteByID(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	entry := ChatEntry{Username: "alice", Body: "hello", TeamID: 1, UUID: testUUIDA}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("unexpected error creating entry: %v", err)
	}

	if err := manager.Delete(entry.ID); err != nil {
		t.Fatalf("unexpected error deleting by id: %v", err)
	}

	var count int64
	if err := db.Model(&ChatEntry{}).Where("id = ?", entry.ID).Count(&count).Error; err != nil {
		t.Fatalf("unexpected count error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted entry count 0, got %d", count)
	}
}

func TestDeleteAllByManagerUUID(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	seed := []ChatEntry{
		{Username: "alice", Body: "a", TeamID: 1, UUID: testUUIDA},
		{Username: "bob", Body: "b", TeamID: 2, UUID: testUUIDA},
		{Username: "carol", Body: "c", TeamID: 3, UUID: testUUIDB},
	}
	for _, entry := range seed {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("unexpected seed error: %v", err)
		}
	}

	if err := manager.DeleteAll(); err != nil {
		t.Fatalf("unexpected error deleting all scoped entries: %v", err)
	}

	var countA, countB int64
	if err := db.Model(&ChatEntry{}).Where("uuid = ?", testUUIDA).Count(&countA).Error; err != nil {
		t.Fatalf("unexpected countA error: %v", err)
	}
	if err := db.Model(&ChatEntry{}).Where("uuid = ?", testUUIDB).Count(&countB).Error; err != nil {
		t.Fatalf("unexpected countB error: %v", err)
	}
	if countA != 0 {
		t.Fatalf("expected uuid A entries deleted, got %d", countA)
	}
	if countB != 1 {
		t.Fatalf("expected uuid B entries preserved, got %d", countB)
	}
}

func TestDeleteAllByTeamID(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	seed := []ChatEntry{
		{Username: "alice", Body: "a", TeamID: 1, UUID: testUUIDA},
		{Username: "bob", Body: "b", TeamID: 1, UUID: testUUIDA},
		{Username: "carol", Body: "c", TeamID: 2, UUID: testUUIDA},
		{Username: "dave", Body: "d", TeamID: 1, UUID: testUUIDB},
	}
	for _, entry := range seed {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("unexpected seed error: %v", err)
		}
	}

	if err := manager.DeleteAllByTeamID(1); err != nil {
		t.Fatalf("unexpected error deleting by team id: %v", err)
	}

	var countScopedTeam1, countScopedTeam2, countOtherUUID int64
	_ = db.Model(&ChatEntry{}).Where("uuid = ? AND team_id = ?", testUUIDA, 1).Count(&countScopedTeam1).Error
	_ = db.Model(&ChatEntry{}).Where("uuid = ? AND team_id = ?", testUUIDA, 2).Count(&countScopedTeam2).Error
	_ = db.Model(&ChatEntry{}).Where("uuid = ? AND team_id = ?", testUUIDB, 1).Count(&countOtherUUID).Error

	if countScopedTeam1 != 0 {
		t.Fatalf("expected scoped team 1 entries deleted, got %d", countScopedTeam1)
	}
	if countScopedTeam2 != 1 {
		t.Fatalf("expected scoped team 2 entries preserved, got %d", countScopedTeam2)
	}
	if countOtherUUID != 1 {
		t.Fatalf("expected other uuid entries preserved, got %d", countOtherUUID)
	}
}

func TestDeleteAllByUsername(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	seed := []ChatEntry{
		{Username: "alice", Body: "a", TeamID: 1, UUID: testUUIDA},
		{Username: "alice", Body: "b", TeamID: 2, UUID: testUUIDA},
		{Username: "bob", Body: "c", TeamID: 1, UUID: testUUIDA},
		{Username: "alice", Body: "d", TeamID: 1, UUID: testUUIDB},
	}
	for _, entry := range seed {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("unexpected seed error: %v", err)
		}
	}

	if err := manager.DeleteAllByUsername("alice"); err != nil {
		t.Fatalf("unexpected error deleting by username: %v", err)
	}

	var countScopedAlice, countScopedBob, countOtherUUIDAlice int64
	_ = db.Model(&ChatEntry{}).Where("uuid = ? AND username = ?", testUUIDA, "alice").Count(&countScopedAlice).Error
	_ = db.Model(&ChatEntry{}).Where("uuid = ? AND username = ?", testUUIDA, "bob").Count(&countScopedBob).Error
	_ = db.Model(&ChatEntry{}).Where("uuid = ? AND username = ?", testUUIDB, "alice").Count(&countOtherUUIDAlice).Error

	if countScopedAlice != 0 {
		t.Fatalf("expected scoped alice entries deleted, got %d", countScopedAlice)
	}
	if countScopedBob != 1 {
		t.Fatalf("expected scoped bob entry preserved, got %d", countScopedBob)
	}
	if countOtherUUIDAlice != 1 {
		t.Fatalf("expected other uuid alice entry preserved, got %d", countOtherUUIDAlice)
	}
}

func TestClosedDBErrorPaths(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close sql db: %v", err)
	}

	if err := manager.Create(ChatEntry{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := manager.GetAll(); err == nil {
		t.Fatal("expected getall error on closed db")
	}
	if err := manager.Delete(1); err == nil {
		t.Fatal("expected delete error on closed db")
	}
	if err := manager.DeleteAll(); err == nil {
		t.Fatal("expected deleteall error on closed db")
	}
	if err := manager.DeleteAllByTeamID(1); err == nil {
		t.Fatal("expected deleteall by team id error on closed db")
	}
	if err := manager.DeleteAllByUsername("alice"); err == nil {
		t.Fatal("expected deleteall by username error on closed db")
	}
}

func TestCreateNew(t *testing.T) {
	manager, db := setupTestManager(t, testUUIDA)

	if err := manager.CreateNew("alice", "hello world", 3, 5); err != nil {
		t.Fatalf("unexpected error from CreateNew: %v", err)
	}

	var entries []ChatEntry
	if err := db.Where("uuid = ?", testUUIDA).Find(&entries).Error; err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Body != "hello" {
		t.Fatalf("expected truncated stored body hello, got %s", entries[0].Body)
	}
	if entries[0].TeamID != 3 {
		t.Fatalf("expected team id 3, got %d", entries[0].TeamID)
	}
}

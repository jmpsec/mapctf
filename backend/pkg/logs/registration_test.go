package logs

import "testing"

func TestRegistrationLogMethods(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	entry, err := m.NewRegistrationLog("alice", "Alice", "A-Team", "alice@local", "logo.png", "127.0.0.1", testUUIDA, true)
	if err != nil {
		t.Fatalf("unexpected error from NewRegistrationLog: %v", err)
	}
	if entry.Username != "alice" || entry.Name != "Alice" || entry.Team != "A-Team" || entry.Email != "alice@local" || entry.Logo != "logo.png" || entry.IPAddress != "127.0.0.1" || entry.Password != true || entry.UUID != testUUIDA {
		t.Fatalf("unexpected registration entry: %+v", entry)
	}

	if err := m.CreateRegistrationLog(entry); err != nil {
		t.Fatalf("unexpected error from CreateRegistrationLog: %v", err)
	}
	if err := m.CreateRegistrationLog(RegistrationLog{Username: "bob", UUID: testUUIDB}); err != nil {
		t.Fatalf("unexpected error from CreateRegistrationLog: %v", err)
	}

	logs, err := m.AllRegistrationLogs(testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from AllRegistrationLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 registration log, got %d", len(logs))
	}
	if logs[0].UUID != testUUIDA || logs[0].Username != "alice" {
		t.Fatalf("unexpected registration row: %+v", logs[0])
	}
}

func TestRegistrationLogErrorPaths(t *testing.T) {
	m, sqlDB := newTestManager(t)
	_ = sqlDB.Close()

	if err := m.CreateRegistrationLog(RegistrationLog{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := m.AllRegistrationLogs(testUUIDA); err == nil {
		t.Fatal("expected query error on closed db")
	}
}

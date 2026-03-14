package logs

import "testing"

func TestFailuresLogMethods(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	entry, err := m.NewFailuresLog(11, 22, "flag{bad}", testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from NewFailuresLog: %v", err)
	}
	if entry.ChallengeID != 11 || entry.TeamID != 22 || entry.Flag != "flag{bad}" || entry.UUID != testUUIDA {
		t.Fatalf("unexpected failures entry: %+v", entry)
	}

	if err := m.CreateFailuresLog(entry); err != nil {
		t.Fatalf("unexpected error from CreateFailuresLog: %v", err)
	}
	if err := m.CreateFailuresLog(FailuresLog{ChallengeID: 1, TeamID: 2, Flag: "x", UUID: testUUIDB}); err != nil {
		t.Fatalf("unexpected error from CreateFailuresLog: %v", err)
	}

	failures, err := m.AllFailuresLogs(testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from AllFailuresLogs: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failures log, got %d", len(failures))
	}
	if failures[0].UUID != testUUIDA || failures[0].Flag != "flag{bad}" {
		t.Fatalf("unexpected failures row: %+v", failures[0])
	}
}

func TestFailuresLogErrorPaths(t *testing.T) {
	m, sqlDB := newTestManager(t)
	_ = sqlDB.Close()

	if err := m.CreateFailuresLog(FailuresLog{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := m.AllFailuresLogs(testUUIDA); err == nil {
		t.Fatal("expected query error on closed db")
	}
}

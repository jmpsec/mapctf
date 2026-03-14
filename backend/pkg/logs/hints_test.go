package logs

import "testing"

func TestHintsLogMethods(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	entry, err := m.NewHintsLog(9, 3, 25, testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from NewHintsLog: %v", err)
	}
	if entry.ChallengeID != 9 || entry.TeamID != 3 || entry.Penalty != 25 || entry.UUID != testUUIDA {
		t.Fatalf("unexpected hints entry: %+v", entry)
	}

	if err := m.CreateHintsLog(entry); err != nil {
		t.Fatalf("unexpected error from CreateHintsLog: %v", err)
	}
	if err := m.CreateHintsLog(HintsLog{ChallengeID: 1, TeamID: 1, Penalty: 1, UUID: testUUIDB}); err != nil {
		t.Fatalf("unexpected error from CreateHintsLog: %v", err)
	}

	hints, err := m.AllHintsLogs(testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from AllHintsLogs: %v", err)
	}
	if len(hints) != 1 {
		t.Fatalf("expected 1 hints log, got %d", len(hints))
	}
	if hints[0].UUID != testUUIDA || hints[0].Penalty != 25 {
		t.Fatalf("unexpected hints row: %+v", hints[0])
	}
}

func TestHintsLogErrorPaths(t *testing.T) {
	m, sqlDB := newTestManager(t)
	_ = sqlDB.Close()

	if err := m.CreateHintsLog(HintsLog{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := m.AllHintsLogs(testUUIDA); err == nil {
		t.Fatal("expected query error on closed db")
	}
}

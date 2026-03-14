package logs

import "testing"

func TestScoreboardLogMethods(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	entry, err := m.NewScoreboardLog("A-Team", 150, 3, testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from NewScoreboardLog: %v", err)
	}
	if entry.Team != "A-Team" || entry.Points != 150 || entry.Iteration != 3 || entry.UUID != testUUIDA {
		t.Fatalf("unexpected scoreboard entry: %+v", entry)
	}

	if err := m.CreateScoreboardLog(entry); err != nil {
		t.Fatalf("unexpected error from CreateScoreboardLog: %v", err)
	}
	if err := m.CreateScoreboardLog(ScoreboardLog{Team: "B-Team", Points: 100, Iteration: 1, UUID: testUUIDB}); err != nil {
		t.Fatalf("unexpected error from CreateScoreboardLog: %v", err)
	}

	logs, err := m.AllScoreboardLogs(testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from AllScoreboardLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 scoreboard log, got %d", len(logs))
	}
	if logs[0].UUID != testUUIDA || logs[0].Team != "A-Team" {
		t.Fatalf("unexpected scoreboard row: %+v", logs[0])
	}
}

func TestScoreboardLogErrorPaths(t *testing.T) {
	m, sqlDB := newTestManager(t)
	_ = sqlDB.Close()

	if err := m.CreateScoreboardLog(ScoreboardLog{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := m.AllScoreboardLogs(testUUIDA); err == nil {
		t.Fatal("expected query error on closed db")
	}
}

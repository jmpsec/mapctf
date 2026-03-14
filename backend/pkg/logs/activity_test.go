package logs

import "testing"

func TestActivityLogMethods(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	entry, err := m.NewActivity("team", "solve", "solved challenge", "challenge=web-1", testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from NewActivity: %v", err)
	}
	if entry.Subject != "team" || entry.Action != "solve" || entry.Message != "solved challenge" || entry.Arguments != "challenge=web-1" || entry.UUID != testUUIDA {
		t.Fatalf("unexpected activity entry: %+v", entry)
	}

	if err := m.CreateActivity(entry); err != nil {
		t.Fatalf("unexpected error from CreateActivity: %v", err)
	}
	if err := m.CreateActivity(ActivityLog{Subject: "team2", Action: "join", UUID: testUUIDB}); err != nil {
		t.Fatalf("unexpected error from CreateActivity: %v", err)
	}

	activities, err := m.AllActivity(testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from AllActivity: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].UUID != testUUIDA || activities[0].Subject != "team" {
		t.Fatalf("unexpected activity row: %+v", activities[0])
	}
}

func TestActivityLogErrorPaths(t *testing.T) {
	m, sqlDB := newTestManager(t)
	_ = sqlDB.Close()

	if err := m.CreateActivity(ActivityLog{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := m.AllActivity(testUUIDA); err == nil {
		t.Fatal("expected query error on closed db")
	}
}

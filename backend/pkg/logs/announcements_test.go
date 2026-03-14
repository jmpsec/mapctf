package logs

import "testing"

func TestAnnouncementMethods(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	entry, err := m.NewAnnouncement("platform maintenance", "admin", testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from NewAnnouncement: %v", err)
	}
	if entry.Entry != "platform maintenance" || entry.MadeBy != "admin" || entry.UUID != testUUIDA {
		t.Fatalf("unexpected announcement entry: %+v", entry)
	}

	if err := m.CreateAnnouncement(entry); err != nil {
		t.Fatalf("unexpected error from CreateAnnouncement: %v", err)
	}
	if err := m.CreateAnnouncement(Announcement{Entry: "other", MadeBy: "bob", UUID: testUUIDB}); err != nil {
		t.Fatalf("unexpected error from CreateAnnouncement: %v", err)
	}

	announcements, err := m.AllAnnouncements(testUUIDA)
	if err != nil {
		t.Fatalf("unexpected error from AllAnnouncements: %v", err)
	}
	if len(announcements) != 1 {
		t.Fatalf("expected 1 announcement, got %d", len(announcements))
	}
	if announcements[0].UUID != testUUIDA || announcements[0].MadeBy != "admin" {
		t.Fatalf("unexpected announcement row: %+v", announcements[0])
	}
}

func TestAnnouncementErrorPaths(t *testing.T) {
	m, sqlDB := newTestManager(t)
	_ = sqlDB.Close()

	if err := m.CreateAnnouncement(Announcement{}); err == nil {
		t.Fatal("expected create error on closed db")
	}
	if _, err := m.AllAnnouncements(testUUIDA); err == nil {
		t.Fatal("expected query error on closed db")
	}
}

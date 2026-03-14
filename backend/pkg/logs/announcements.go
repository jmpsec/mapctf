package logs

import "gorm.io/gorm"

// Announcement to hold each announcement entry in the system
type Announcement struct {
	gorm.Model
	Entry  string
	MadeBy string
	UUID   string `gorm:"index"`
}

// CreateAnnouncement to create a new announcement entry
func (l *LogManager) CreateAnnouncement(announcement Announcement) error {
	if err := l.DB.Create(&announcement).Error; err != nil {
		return err
	}
	return nil
}

// NewAnnouncement to create a new announcement struct
func (l *LogManager) NewAnnouncement(entry, madeBy, uuid string) (Announcement, error) {
	return Announcement{
		Entry:  entry,
		MadeBy: madeBy,
		UUID:   uuid,
	}, nil
}

// AllAnnouncements to get all announcements for a given UUID
func (l *LogManager) AllAnnouncements(uuid string) ([]Announcement, error) {
	var announcements []Announcement
	if err := l.DB.Where("uuid = ?", uuid).Find(&announcements).Error; err != nil {
		return announcements, err
	}
	return announcements, nil
}

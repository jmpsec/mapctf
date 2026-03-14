package logs

import (
	"fmt"

	"gorm.io/gorm"
)

// ActivityLog to hold each activity entry in the system
type ActivityLog struct {
	gorm.Model
	Subject   string
	Action    string
	Message   string
	Arguments string
	UUID      string `gorm:"index"`
}

// CreateActivity logs a new activity
func (l *LogManager) CreateActivity(activity ActivityLog) error {
	if err := l.DB.Create(&activity).Error; err != nil {
		return fmt.Errorf("Create ActivityLog %w", err)
	}
	return nil
}

// NewActivity to create a new activity log entry
func (l *LogManager) NewActivity(subject, action, message, arguments, uuid string) (ActivityLog, error) {
	return ActivityLog{
		Subject:   subject,
		Action:    action,
		Message:   message,
		Arguments: arguments,
		UUID:      uuid,
	}, nil
}

// AllActivity to get all activity logs for a given UUID
func (l *LogManager) AllActivity(uuid string) ([]ActivityLog, error) {
	var activities []ActivityLog
	if err := l.DB.Where("uuid = ?", uuid).Find(&activities).Error; err != nil {
		return activities, fmt.Errorf("Get All Activity Logs for UUID: %w", err)
	}
	return activities, nil
}

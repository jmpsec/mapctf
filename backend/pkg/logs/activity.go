package logs

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	// ActionEnabled is the action type for enabling a challenge
	ActionEnabled = "enabled"
	// ActionDisabled is the action type for disabling a challenge
	ActionDisabled = "disabled"
	// ActionCompleted is the action type for a team completing a challenge
	ActionCompleted = "completed"
)

// ActivityLog to hold each activity entry in the system
type ActivityLog struct {
	gorm.Model
	Subject     string
	Action      string
	ChallengeID uint
	Message     string
	UUID        string `gorm:"index"`
}

// CreateActivity logs a new activity
func (l *LogManager) CreateActivity(activity ActivityLog) error {
	if err := l.DB.Create(&activity).Error; err != nil {
		return fmt.Errorf("Create ActivityLog %w", err)
	}
	return nil
}

// NewActivity to create a new activity log entry
func (l *LogManager) NewActivity(subject, action, message string, challengeID uint, uuid string) (ActivityLog, error) {
	return ActivityLog{
		Subject:     subject,
		Action:      action,
		ChallengeID: challengeID,
		Message:     message,
		UUID:        uuid,
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

// ActivityMessage generates a formatted message for an activity log entry
func (a *ActivityLog) ActivityMessage(activity ActivityLog) string {
	switch activity.Action {
	case ActionEnabled, ActionDisabled, ActionCompleted:
		if activity.Message != "" {
			return activity.Message
		}
		if activity.Action != "" {
			return activity.Action
		}
		return "Unknown activity action"
	default:
		return "Unknown activity action"
	}
}

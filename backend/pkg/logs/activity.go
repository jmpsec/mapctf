package logs

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	// ActionCreated is the action type for creating a new challenge or team
	ActionCreated = "created"
	// ActionUpdated is the action type for updating a challenge or team
	ActionUpdated = "updated"
	// ActionEnabled is the action type for enabling a challenge
	ActionEnabled = "enabled"
	// ActionDisabled is the action type for disabling a challenge
	ActionDisabled = "disabled"
	// ActionCompleted is the action type for a team completing a challenge
	ActionCompleted = "completed"
	// ActionAnnouncement is the action type for an announcement (no challenge ID)
	ActionAnnouncement = "announcement"
	// NoChallengeID is a constant to represent no challenge ID for an activity log entry
	NoChallengeID = 0
)

// ActivityLog to hold each activity entry in the system
type ActivityLog struct {
	gorm.Model
	Visible     bool
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
func (l *LogManager) NewActivity(visible bool, subject, action, message string, challengeID uint, uuid string) (ActivityLog, error) {
	return ActivityLog{
		Visible:     visible,
		Subject:     subject,
		Action:      action,
		ChallengeID: challengeID,
		Message:     message,
		UUID:        uuid,
	}, nil
}

// NewScoreActivity to create a new score log entry
func (l *LogManager) NewScoreActivity(teamName string, points int, countryCode, categoryName string, challengeID uint, uuid string) (ActivityLog, error) {
	message := ScoreMessage(teamName, points, countryCode, categoryName)
	return ActivityLog{
		Visible:     true,
		Subject:     teamName,
		Action:      ActionCompleted,
		ChallengeID: challengeID,
		Message:     message,
		UUID:        uuid,
	}, nil
}

// NewEnabledActivity to create a new enabled log entry
func (l *LogManager) NewEnabledActivity(countryCode, categoryName string, points int, challengeID uint, uuid string) (ActivityLog, error) {
	message := EnableMessage(countryCode, categoryName, points)
	return ActivityLog{
		Visible:     true,
		Subject:     fmt.Sprintf("%s (%s)", countryCode, categoryName),
		Action:      ActionEnabled,
		ChallengeID: challengeID,
		Message:     message,
		UUID:        uuid,
	}, nil
}

// NewAnnouncement to create a new announcement log entry (no challenge ID)
func (l *LogManager) NewAnnouncement(subject, message string, uuid string) (ActivityLog, error) {
	return ActivityLog{
		Visible:     true,
		Subject:     subject,
		Action:      ActionAnnouncement,
		ChallengeID: NoChallengeID,
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

// DeleteActivity deletes a single activity log by ID scoped to the given UUID.
func (l *LogManager) DeleteActivity(id uint, uuid string) error {
	if err := l.DB.Where("id = ? AND uuid = ?", id, uuid).Delete(&ActivityLog{}).Error; err != nil {
		return fmt.Errorf("Delete ActivityLog: %w", err)
	}
	return nil
}

// GetActivityByChallengeID to get all activity logs for a given challenge ID and UUID
func (l *LogManager) GetActivityByChallengeID(challengeID uint, uuid string) ([]ActivityLog, error) {
	var activities []ActivityLog
	if err := l.DB.Where("challenge_id = ? AND uuid = ?", challengeID, uuid).Find(&activities).Error; err != nil {
		return activities, fmt.Errorf("Get Activity Logs by Challenge ID: %w", err)
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

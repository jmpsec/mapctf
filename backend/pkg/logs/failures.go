package logs

import "gorm.io/gorm"

// FailuresLog to hold each scoring failure entry in the system
type FailuresLog struct {
	gorm.Model
	ChallengeID uint
	TeamID      uint
	Flag        string
	UUID        string `gorm:"index"`
}

// CreateFailuresLog to create a new failures log entry
func (l *LogManager) CreateFailuresLog(failuresLog FailuresLog) error {
	if err := l.DB.Create(&failuresLog).Error; err != nil {
		return err
	}
	return nil
}

// NewFailuresLog to create a new failures log struct
func (l *LogManager) NewFailuresLog(challengeID, teamID uint, flag, uuid string) (FailuresLog, error) {
	return FailuresLog{
		ChallengeID: challengeID,
		TeamID:      teamID,
		Flag:        flag,
		UUID:        uuid,
	}, nil
}

// AllFailuresLogs to get all failures logs for a given UUID
func (l *LogManager) AllFailuresLogs(uuid string) ([]FailuresLog, error) {
	var failuresLogs []FailuresLog
	if err := l.DB.Where("uuid = ?", uuid).Find(&failuresLogs).Error; err != nil {
		return failuresLogs, err
	}
	return failuresLogs, nil
}

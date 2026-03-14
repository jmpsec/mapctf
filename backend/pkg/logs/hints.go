package logs

import "gorm.io/gorm"

// HintsLog to hold each hint entry in the system
type HintsLog struct {
	gorm.Model
	ChallengeID uint
	TeamID      uint
	Penalty     int
	UUID        string `gorm:"index"`
}

// CreateHintsLog to create a new hints log entry
func (l *LogManager) CreateHintsLog(hintsLog HintsLog) error {
	if err := l.DB.Create(&hintsLog).Error; err != nil {
		return err
	}
	return nil
}

// NewHintsLog to create a new hints log struct
func (l *LogManager) NewHintsLog(challengeID, teamID uint, penalty int, uuid string) (HintsLog, error) {
	return HintsLog{
		ChallengeID: challengeID,
		TeamID:      teamID,
		Penalty:     penalty,
		UUID:        uuid,
	}, nil
}

// AllHintsLogs to get all hints logs for a given UUID
func (l *LogManager) AllHintsLogs(uuid string) ([]HintsLog, error) {
	var hintsLogs []HintsLog
	if err := l.DB.Where("uuid = ?", uuid).Find(&hintsLogs).Error; err != nil {
		return hintsLogs, err
	}
	return hintsLogs, nil
}

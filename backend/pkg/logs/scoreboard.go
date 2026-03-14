package logs

import "gorm.io/gorm"

// ScoreboardLog to hold each scoreboard entry in the system
type ScoreboardLog struct {
	gorm.Model
	Team      string
	Points    int
	Iteration int
	UUID      string `gorm:"index"`
}

// CreateScoreboardLog to create a new scoreboard log entry
func (l *LogManager) CreateScoreboardLog(scoreboardLog ScoreboardLog) error {
	if err := l.DB.Create(&scoreboardLog).Error; err != nil {
		return err
	}
	return nil
}

// NewScoreboardLog to create a new scoreboard log struct
func (l *LogManager) NewScoreboardLog(team string, points, iteration int, uuid string) (ScoreboardLog, error) {
	return ScoreboardLog{
		Team:      team,
		Points:    points,
		Iteration: iteration,
		UUID:      uuid,
	}, nil
}

// AllScoreboardLogs to get all scoreboard logs for a given UUID
func (l *LogManager) AllScoreboardLogs(uuid string) ([]ScoreboardLog, error) {
	var scoreboardLogs []ScoreboardLog
	if err := l.DB.Where("uuid = ?", uuid).Find(&scoreboardLogs).Error; err != nil {
		return scoreboardLogs, err
	}
	return scoreboardLogs, nil
}

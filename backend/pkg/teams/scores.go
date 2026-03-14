package teams

import "gorm.io/gorm"

// TeamScore to hold all team scores over time
type TeamScore struct {
	gorm.Model
	TeamID      uint `gorm:"index"`
	ChallengeID uint
	Points      int
	UUID        string `gorm:"index"`
	ScoredBy    string
}

// GetScores to get all scores for a team and UUID
func (m *TeamManager) GetScores(teamID uint, uuid string) ([]TeamScore, error) {
	var scores []TeamScore
	if err := m.DB.Where("team_id = ? AND uuid = ?", teamID, uuid).Find(&scores).Error; err != nil {
		return scores, err
	}
	return scores, nil
}

// NewScore to create a new team score
func (m *TeamManager) NewScore(teamID, challengeID uint, points int, uuid, scoredBy string) (TeamScore, error) {
	return TeamScore{
		TeamID:      teamID,
		ChallengeID: challengeID,
		Points:      points,
		UUID:        uuid,
		ScoredBy:    scoredBy,
	}, nil
}

// CreateScore to save a new team score
func (m *TeamManager) CreateScore(score TeamScore) error {
	if err := m.DB.Create(&score).Error; err != nil {
		return err
	}
	return nil
}

// GetScoreTotal to get the total score for a team and UUID
func (m *TeamManager) GetScoreTotal(teamID uint, uuid string) (int, error) {
	var total int
	if err := m.DB.Model(&TeamScore{}).Where("team_id = ? AND uuid = ?", teamID, uuid).Select("SUM(points)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

package teams

import "gorm.io/gorm"

// TeamScore to hold all team scores over time
type TeamScore struct {
	gorm.Model
	TeamID      uint `gorm:"index"`
	ChallengeID uint
	Points      int
	EntID       uint
	ScoredBy    uint
}

// GetScores to get all scores for a team and entity ID
func (m *TeamManager) GetScores(teamID, eID uint) ([]TeamScore, error) {
	var scores []TeamScore
	if err := m.DB.Where("team_id = ? AND ent_id = ?", teamID, eID).Find(&scores).Error; err != nil {
		return scores, err
	}
	return scores, nil
}

// NewScore to create a new team score
func (m *TeamManager) NewScore(teamID, challengeID uint, points int, eID, scoredBy uint) (TeamScore, error) {
	return TeamScore{
		TeamID:      teamID,
		ChallengeID: challengeID,
		Points:      points,
		EntID:       eID,
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

// GetScoreTotal to get the total score for a team and entity ID
func (m *TeamManager) GetScoreTotal(teamID, eID uint) (int, error) {
	var total int
	if err := m.DB.Model(&TeamScore{}).Where("team_id = ? AND ent_id = ?", teamID, eID).Select("SUM(points)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

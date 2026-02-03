package teams

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TeamManager to handle all teams of the platform
type TeamManager struct {
	DB *gorm.DB
}

// PlatformTeam to hold all teams of the platform
type PlatformTeam struct {
	gorm.Model
	Name      string `gorm:"index"`
	Logo      string
	Points    int
	LastScore time.Time
	Protected bool
	Visible   bool
	Active    bool
	EntID     uint
}

// TeamMembership to link users to teams
type TeamMembership struct {
	gorm.Model
	TeamID     uint `gorm:"index"`
	UserID     uint `gorm:"index"`
	EntID      uint
	AssignedBy uint
}

// TeamScore to hold all team scores over time
type TeamScore struct {
	gorm.Model
	TeamID      uint `gorm:"index"`
	ChallengeID uint
	Points      int
	EntID       uint
	ScoredBy    uint
}

// CreateTeams to initialize the teams struct and its tables
func CreateTeams(backend *gorm.DB) (*TeamManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	t := &TeamManager{
		DB: backend,
	}
	// table platform_teams
	if err := backend.AutoMigrate(&PlatformTeam{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (platform_teams): %w", err)
	}
	// table team_memberships
	if err := backend.AutoMigrate(&TeamMembership{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (team_memberships): %w", err)
	}
	// table team_scores
	if err := backend.AutoMigrate(&TeamScore{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (team_scores): %w", err)
	}
	return t, nil
}

// Create new team
func (m *TeamManager) Create(team PlatformTeam) error {
	if err := m.DB.Create(&team).Error; err != nil {
		return fmt.Errorf("Create PlatformTeam %w", err)
	}
	return nil
}

// Exists checks if user exists
func (m *TeamManager) Exists(name string) bool {
	var results int64
	m.DB.Model(&PlatformTeam{}).Where("name = ?", name).Count(&results)
	return (results > 0)
}

// Get user by username including service users
func (m *TeamManager) Get(name string) (PlatformTeam, error) {
	var team PlatformTeam
	if err := m.DB.Where("name = ?", name).First(&team).Error; err != nil {
		return team, err
	}
	return team, nil
}

// Get user by username and by entity ID, including service users
func (m *TeamManager) GetByEntID(name string, entID uint) (PlatformTeam, error) {
	var team PlatformTeam
	if err := m.DB.Where("name = ? AND ent_id = ?", name, entID).First(&team).Error; err != nil {
		return team, err
	}
	return team, nil
}

// ExistsGet checks if user exists and returns the user
func (m *TeamManager) ExistsGet(name string) (bool, PlatformTeam) {
	team, err := m.Get(name)
	if err != nil {
		return false, PlatformTeam{}
	}
	return true, team
}

// ExistsGetByEntID checks if user exists and returns the user
func (m *TeamManager) ExistsGetByEntID(name string, entID uint) (bool, PlatformTeam) {
	team, err := m.GetByEntID(name, entID)
	if err != nil {
		return false, PlatformTeam{}
	}
	return true, team
}

// New empty user
func (m *TeamManager) New(name, logo, email string, protected, visible bool, eID uint) (PlatformTeam, error) {
	if !m.Exists(name) {
		return PlatformTeam{
			Name:      name,
			Logo:      logo,
			Protected: protected,
			Visible:   visible,
			Active:    true,
			EntID:     eID,
		}, nil
	}
	return PlatformTeam{}, fmt.Errorf("%s already exists", name)
}

package teams

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	// NoTeamID is the default team ID when no team is assigned
	NoTeamID uint = 0
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
	// table team_logos
	if err := backend.AutoMigrate(&TeamLogo{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (team_logos): %w", err)
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

// Exists checks if team exists
func (m *TeamManager) Exists(name string, entID uint) bool {
	var results int64
	m.DB.Model(&PlatformTeam{}).Where("name = ? AND ent_id = ?", name, entID).Count(&results)
	return (results > 0)
}

// Get team by name
func (m *TeamManager) Get(name string, entID uint) (PlatformTeam, error) {
	var team PlatformTeam
	if err := m.DB.Where("name = ? AND ent_id = ?", name, entID).First(&team).Error; err != nil {
		return team, err
	}
	return team, nil
}

// Get user by name and by entity ID
func (m *TeamManager) GetByEntID(name string, entID uint) (PlatformTeam, error) {
	var team PlatformTeam
	if err := m.DB.Where("name = ? AND ent_id = ?", name, entID).First(&team).Error; err != nil {
		return team, err
	}
	return team, nil
}

// ExistsGet checks if user exists and returns the team
func (m *TeamManager) ExistsGet(name string, entID uint) (bool, PlatformTeam) {
	team, err := m.Get(name, entID)
	if err != nil {
		return false, PlatformTeam{}
	}
	return true, team
}

// ExistsGetByEntID checks if team exists and returns the team by name and by entity ID
func (m *TeamManager) ExistsGetByEntID(name string, entID uint) (bool, PlatformTeam) {
	team, err := m.GetByEntID(name, entID)
	if err != nil {
		return false, PlatformTeam{}
	}
	return true, team
}

// New empty team
func (m *TeamManager) New(name, logo, email string, protected, visible bool, eID uint) (PlatformTeam, error) {
	if !m.Exists(name, eID) {
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

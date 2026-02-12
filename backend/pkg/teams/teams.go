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
	UUID      string `gorm:"index"`
}

// TeamMembership to link users to teams
type TeamMembership struct {
	gorm.Model
	TeamID     uint   `gorm:"index"`
	UserID     uint   `gorm:"index"`
	UUID       string `gorm:"index"`
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
func (m *TeamManager) Exists(name string, uuid string) bool {
	var results int64
	m.DB.Model(&PlatformTeam{}).Where("name = ? AND uuid = ?", name, uuid).Count(&results)
	return (results > 0)
}

// Get team by name
func (m *TeamManager) Get(name string, uuid string) (PlatformTeam, error) {
	var team PlatformTeam
	if err := m.DB.Where("name = ? AND uuid = ?", name, uuid).First(&team).Error; err != nil {
		return team, err
	}
	return team, nil
}

// Get all teams
func (m *TeamManager) GetAll(uuid string) ([]PlatformTeam, error) {
	var teams []PlatformTeam
	if err := m.DB.Where("uuid = ?", uuid).Find(&teams).Error; err != nil {
		return teams, err
	}
	return teams, nil
}

// GetByUUID gets team by name and UUID (alias for Get)
func (m *TeamManager) GetByUUID(name string, uuid string) (PlatformTeam, error) {
	return m.Get(name, uuid)
}

// ExistsGet checks if team exists and returns the team
func (m *TeamManager) ExistsGet(name string, uuid string) (bool, PlatformTeam) {
	team, err := m.Get(name, uuid)
	if err != nil {
		return false, PlatformTeam{}
	}
	return true, team
}

// ExistsGetByUUID checks if team exists and returns the team by name and UUID
func (m *TeamManager) ExistsGetByUUID(name string, uuid string) (bool, PlatformTeam) {
	team, err := m.Get(name, uuid)
	if err != nil {
		return false, PlatformTeam{}
	}
	return true, team
}

// New empty team
func (m *TeamManager) New(name, logo, email string, protected, visible bool, uuid string) (PlatformTeam, error) {
	if !m.Exists(name, uuid) {
		return PlatformTeam{
			Name:      name,
			Logo:      logo,
			Protected: protected,
			Visible:   visible,
			Active:    true,
			UUID:      uuid,
		}, nil
	}
	return PlatformTeam{}, fmt.Errorf("%s already exists", name)
}

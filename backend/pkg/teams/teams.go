package teams

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
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
	TeamID   uint `gorm:"index"`
	Points   int
	EntID    uint
	ScoredBy uint
}

// CreateTeams to initialize the teams struct and its tables
func CreateTeams(backend *gorm.DB) *TeamManager {
	t := &TeamManager{
		DB: backend,
	}
	// table platform_teams
	if err := backend.AutoMigrate(&PlatformTeam{}); err != nil {
		log.Fatal().Msgf("Failed to AutoMigrate table (platform_teams): %v", err)
	}
	// table team_memberships
	if err := backend.AutoMigrate(&TeamMembership{}); err != nil {
		log.Fatal().Msgf("Failed to AutoMigrate table (team_memberships): %v", err)
	}
	// table team_scores
	if err := backend.AutoMigrate(&TeamScore{}); err != nil {
		log.Fatal().Msgf("Failed to AutoMigrate table (team_scores): %v", err)
	}
	return t
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

// Get user by username and by tenant ID, including service users
func (m *TeamManager) GetByTenantID(name string, tenantID uint) (PlatformTeam, error) {
	var team PlatformTeam
	if err := m.DB.Where("name = ? AND ent_id = ?", name, tenantID).First(&team).Error; err != nil {
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

// ExistsGetByTenantID checks if user exists and returns the user
func (m *TeamManager) ExistsGetByTenantID(name string, tenantID uint) (bool, PlatformTeam) {
	team, err := m.GetByTenantID(name, tenantID)
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

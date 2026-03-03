package teams

import (
	"fmt"

	"gorm.io/gorm"
)

// TeamLogo to hold team logos
type TeamLogo struct {
	gorm.Model
	Name      string `gorm:"index"`
	Logo      string
	Used      bool
	Enabled   bool
	Custom    bool
	UUID      string `gorm:"index"`
	Protected bool
	CreatedBy uint
}

// GetLogo to get a team logo by name and UUID
func (m *TeamManager) GetLogo(name string, uuid string) (TeamLogo, error) {
	var logo TeamLogo
	if err := m.DB.Where("name = ? AND uuid = ?", name, uuid).First(&logo).Error; err != nil {
		return logo, err
	}
	return logo, nil
}

// NewLogo to create a new team logo
func (m *TeamManager) NewLogo(name, logo string, enabled, custom bool, uuid string, createdBy uint) (TeamLogo, error) {
	return TeamLogo{
		Name:      name,
		Logo:      logo,
		Enabled:   enabled,
		Custom:    custom,
		UUID:      uuid,
		CreatedBy: createdBy,
	}, nil
}

// CreateLogo to save a new team logo
func (m *TeamManager) CreateLogo(logo TeamLogo) error {
	if err := m.DB.Create(&logo).Error; err != nil {
		return fmt.Errorf("Create Team Logo: %w", err)
	}
	return nil
}

// ExistsLogo checks if team logo exists
func (m *TeamManager) ExistsLogo(name string, uuid string) bool {
	var results int64
	m.DB.Model(&TeamLogo{}).Where("name = ? AND uuid = ?", name, uuid).Count(&results)
	return (results > 0)
}

// ExistsLogoGet checks if team logo exists and returns the logo
func (m *TeamManager) ExistsLogoGet(name string, uuid string) (bool, TeamLogo) {
	logo, err := m.GetLogo(name, uuid)
	if err != nil {
		return false, TeamLogo{}
	}
	return true, logo
}

// Random logo for team
func (m *TeamManager) RandomLogo(uuid string) (TeamLogo, error) {
	var logo TeamLogo
	if err := m.DB.Where("enabled = ? AND uuid = ?", true, uuid).Order("RANDOM()").First(&logo).Error; err != nil {
		return logo, err
	}
	return logo, nil
}

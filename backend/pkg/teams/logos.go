package teams

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

const (
	// ImporterCreatorID is the default user ID for created logos during initialization
	ImporterCreatorID uint = 0
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

// JSONLogo represents the structure of the team logo data in the seed JSON file
type JSONLogo struct {
	Name   string `json:"name"`
	Logo   string `json:"logo"`
	Custom bool   `json:"custom"`
}

// InitializationStats to hold the statistics of the initialization process
type InitializationStats struct {
	TotalLogos    int
	InsertedLogos int
	ExistingLogos int
}

func loadLogosSeedData(seedFile string) ([]JSONLogo, error) {
	var logos []JSONLogo
	data, err := os.ReadFile(seedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read logos seed file: %w", err)
	}
	if err := json.Unmarshal(data, &logos); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logos seed data: %w", err)
	}
	return logos, nil
}

// InitializeLogos to initialize the logos data in the database
func (s *TeamManager) InitializeLogos(seedFile string) (*InitializationStats, error) {
	stats := &InitializationStats{}
	logosData, err := loadLogosSeedData(seedFile)
	if err != nil {
		return stats, fmt.Errorf("failed to load logos seed data: %w", err)
	}
	for _, logoData := range logosData {
		stats.TotalLogos++
		if s.ExistsLogo(logoData.Name) {
			stats.ExistingLogos++
			continue
		}
		logo, err := s.NewLogo(logoData.Name, logoData.Logo, true, logoData.Custom, 0)
		if err != nil {
			return stats, fmt.Errorf("failed to create logo object %s: %w", logoData.Name, err)
		}
		logo.CreatedBy = ImporterCreatorID
		if err := s.CreateLogo(logo); err != nil {
			return stats, fmt.Errorf("failed to create logo %s: %w", logoData.Name, err)
		}
		stats.InsertedLogos++
	}
	return stats, nil
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
func (m *TeamManager) NewLogo(name, logo string, enabled, custom bool, createdBy uint) (TeamLogo, error) {
	return TeamLogo{
		Name:      name,
		Logo:      logo,
		Enabled:   enabled,
		Custom:    custom,
		UUID:      m.UUID,
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
func (m *TeamManager) ExistsLogo(name string) bool {
	var results int64
	m.DB.Model(&TeamLogo{}).Where("name = ? AND uuid = ?", name, m.UUID).Count(&results)
	return results > 0
}

// ExistsLogoGet checks if team logo exists and returns the logo
func (m *TeamManager) ExistsLogoGet(name string) (bool, TeamLogo) {
	logo, err := m.GetLogo(name, m.UUID)
	if err != nil {
		return false, TeamLogo{}
	}
	return true, logo
}

// RandomLogo for team
func (m *TeamManager) RandomLogo() (TeamLogo, error) {
	var logo TeamLogo
	if err := m.DB.Where("enabled = ? AND uuid = ?", true, m.UUID).Order("RANDOM()").First(&logo).Error; err != nil {
		return logo, err
	}
	return logo, nil
}

func normalizeLogoSymbol(logo string) string {
	logo = strings.TrimSpace(logo)
	if logo == "" {
		return ""
	}
	logo = strings.TrimPrefix(logo, "#icon--badge-")
	logo = strings.TrimPrefix(logo, "icon--badge-")
	if idx := strings.LastIndex(logo, "/"); idx >= 0 && idx < len(logo)-1 {
		logo = logo[idx+1:]
	}
	logo = strings.TrimSuffix(logo, ".svg")
	logo = strings.TrimPrefix(logo, "badge-")
	return strings.TrimSpace(logo)
}

// SyncLogoUsage recalculates the Used flag of all logos for the manager UUID.
func (m *TeamManager) SyncLogoUsage() error {
	var teams []PlatformTeam
	if err := m.DB.Where("uuid = ?", m.UUID).Find(&teams).Error; err != nil {
		return fmt.Errorf("failed to load teams: %w", err)
	}

	var logos []TeamLogo
	if err := m.DB.Where("uuid = ?", m.UUID).Find(&logos).Error; err != nil {
		return fmt.Errorf("failed to load logos: %w", err)
	}

	usedBySymbol := make(map[string]bool, len(teams))
	for _, team := range teams {
		symbol := normalizeLogoSymbol(team.Logo)
		if symbol == "" || strings.EqualFold(symbol, "random") {
			continue
		}
		usedBySymbol[symbol] = true
	}

	for _, logo := range logos {
		symbol := normalizeLogoSymbol(logo.Logo)
		shouldBeUsed := usedBySymbol[symbol]
		if logo.Used == shouldBeUsed {
			continue
		}
		if err := m.DB.Model(&TeamLogo{}).
			Where("id = ? AND uuid = ?", logo.ID, m.UUID).
			Update("used", shouldBeUsed).Error; err != nil {
			return fmt.Errorf("failed to update logo usage: %w", err)
		}
	}

	return nil
}

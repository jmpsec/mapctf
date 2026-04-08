package countries

import (
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/gorm"
)

const (
	// NoChallengeID is a constant to represent no challenge assigned to a country
	NoChallengeID uint = 0
)

// MapCountry represents a country in the map with its vector data and styling information
type MapCountry struct {
	gorm.Model
	Name            string `gorm:"index"`
	CountryCode     string `gorm:"index"`
	Active          bool
	Assigned        bool
	ChallengeID     uint
	LandPath        string
	LandClass       string
	LandStyle       string
	MarkerPath      string
	MarkerClass     string
	MarkerStyle     string
	MarkerTransform string
	UUID            string `gorm:"index"`
}

// JSONCountry represents the structure of the country data in the seed JSON file
type JSONCountry struct {
	Name            string `json:"Name"`
	CountryCode     string `json:"CountryCode"`
	LandPath        string `json:"LandPath"`
	LandClass       string `json:"LandClass"`
	LandStyle       string `json:"LandStyle"`
	MarkerPath      string `json:"MarkerPath"`
	MarkerClass     string `json:"MarkerClass"`
	MarkerStyle     string `json:"MarkerStyle"`
	MarkerTransform string `json:"MarkerTransform"`
}

// InitializationStats to hold the statistics of the initialization process
type InitializationStats struct {
	TotalCountries    int
	InsertedCountries int
	ExistingCountries int
}

// CountriesManager have all settings of the system
type CountriesManager struct {
	DB   *gorm.DB
	UUID string
}

// CreateCountries to initialize the countries struct and tables
func CreateCountries(backend *gorm.DB, uuid string) (*CountriesManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	s := &CountriesManager{DB: backend, UUID: uuid}
	// table map_countries
	if err := backend.AutoMigrate(&MapCountry{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (map_countries): %w", err)
	}
	return s, nil
}

// loadCountriesSeedData to load the countries seed data from a JSON file
func loadCountriesSeedData(seedFile string) ([]JSONCountry, error) {
	var countries []JSONCountry
	// Load the JSON file
	data, err := os.ReadFile(seedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read countries seed file: %w", err)
	}
	// Unmarshal the JSON data into the countries slice
	if err := json.Unmarshal(data, &countries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal countries seed data: %w", err)
	}
	return countries, nil
}

// InitializeCountries to initialize the countries data in the database
func (s *CountriesManager) InitializeCountries(seedFile string) (*InitializationStats, error) {
	stats := &InitializationStats{
		TotalCountries:    0,
		InsertedCountries: 0,
		ExistingCountries: 0,
	}
	// Load JSON data from the seed file
	countriesData, err := loadCountriesSeedData(seedFile)
	if err != nil {
		return stats, fmt.Errorf("failed to load countries seed data: %w", err)
	}
	// Insert countries into the database
	for _, c := range countriesData {
		stats.TotalCountries++
		// Check if the country already exists in the database
		exists, err := s.Exists(c.CountryCode)
		if err != nil {
			return stats, fmt.Errorf("failed to check if country exists: %w", err)
		}
		if exists {
			stats.ExistingCountries++
			continue // Skip existing countries
		}
		// Create a new MapCountry instance and insert it into the database
		country := MapCountry{
			Name:            c.Name,
			CountryCode:     c.CountryCode,
			Active:          true,
			Assigned:        false,
			ChallengeID:     NoChallengeID,
			LandPath:        c.LandPath,
			LandClass:       c.LandClass,
			LandStyle:       c.LandStyle,
			MarkerPath:      c.MarkerPath,
			MarkerClass:     c.MarkerClass,
			MarkerStyle:     c.MarkerStyle,
			MarkerTransform: c.MarkerTransform,
			UUID:            s.UUID,
		}
		if err := s.Create(country); err != nil {
			return stats, fmt.Errorf("failed to create country %s: %w", country.Name, err)
		}
		stats.InsertedCountries++
	}
	return stats, nil
}

// GetAll to get all countries from the database
func (s *CountriesManager) GetAll() ([]MapCountry, error) {
	var countries []MapCountry
	if err := s.DB.Where("uuid = ?", s.UUID).Find(&countries).Error; err != nil {
		return nil, fmt.Errorf("failed to get countries: %w", err)
	}
	return countries, nil
}

// GetActive to get all active countries from the database
func (s *CountriesManager) GetActive() ([]MapCountry, error) {
	var countries []MapCountry
	if err := s.DB.Where("uuid = ? AND active = ?", s.UUID, true).Find(&countries).Error; err != nil {
		return nil, fmt.Errorf("failed to get active countries: %w", err)
	}
	return countries, nil
}

// GetAvailable to get all available (active and not assigned) countries from the database
func (s *CountriesManager) GetAvailable() ([]MapCountry, error) {
	var countries []MapCountry
	if err := s.DB.Where("uuid = ? AND active = ? AND assigned = ?", s.UUID, true, false).Find(&countries).Error; err != nil {
		return nil, fmt.Errorf("failed to get available countries: %w", err)
	}
	return countries, nil
}

// Exists to check if a country exists in the database by its country code and uuid
func (s *CountriesManager) Exists(countryCode string) (bool, error) {
	var count int64
	if err := s.DB.Model(&MapCountry{}).Where("country_code = ? AND uuid = ?", countryCode, s.UUID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check if country exists: %w", err)
	}
	return count > 0, nil
}

// GetByCode returns one country by country code and manager UUID
func (s *CountriesManager) GetByCode(countryCode string) (MapCountry, error) {
	var country MapCountry
	if err := s.DB.Where("country_code = ? AND uuid = ?", countryCode, s.UUID).First(&country).Error; err != nil {
		return MapCountry{}, fmt.Errorf("failed to get country by code: %w", err)
	}
	return country, nil
}

// Create to create a new country in the database
func (s *CountriesManager) Create(country MapCountry) error {
	country.UUID = s.UUID
	if err := s.DB.Create(&country).Error; err != nil {
		return fmt.Errorf("failed to create country: %w", err)
	}
	return nil
}

// SetActiveByID updates the active status of a country by ID for the manager UUID
func (s *CountriesManager) SetActiveByID(id uint, active bool) error {
	result := s.DB.Model(&MapCountry{}).
		Where("id = ? AND uuid = ?", id, s.UUID).
		Update("active", active)
	if result.Error != nil {
		return fmt.Errorf("failed to update country active status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("country not found")
	}
	return nil
}

// AssignCountryToChallenge assigns a challenge to a country by country code and challenge ID
func (s *CountriesManager) AssignCountryToChallenge(countryCode string, challengeID uint) error {
	result := s.DB.Model(&MapCountry{}).
		Where("country_code = ? AND uuid = ?", countryCode, s.UUID).
		Update("challenge_id", challengeID).
		Update("assigned", challengeID != NoChallengeID)
	if result.Error != nil {
		return fmt.Errorf("failed to assign challenge to country: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("country not found")
	}
	return nil
}

// ReleaseCountry releases a country from its assigned challenge by country code
func (s *CountriesManager) ReleaseCountry(countryCode string) error {
	result := s.DB.Model(&MapCountry{}).
		Where("country_code = ? AND uuid = ?", countryCode, s.UUID).
		Update("challenge_id", NoChallengeID).
		Update("assigned", false)
	if result.Error != nil {
		return fmt.Errorf("failed to release country: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("country not found")
	}
	return nil
}

package challenges

import (
	"fmt"

	"gorm.io/gorm"
)

// Challenge to hold all platform challenges
type Challenge struct {
	gorm.Model
	Title       string
	Description string
	CategoryID  uint
	Active      bool
	Points      int
	Bonus       int
	BonusDecay  int
	Flag        string
	Hint        string
	Penalty     int
	EntID       uint `gorm:"index"`
}

// Category to hold all challenge categories
type Category struct {
	gorm.Model
	Name        string `gorm:"index"`
	Description string
	Logo        string
	EntID       uint `gorm:"index"`
}

// ChallengeManager to handle all challenges of the platform
type ChallengeManager struct {
	DB *gorm.DB
}

// CreateChallengeManager to initialize the challenges struct and its tables
func CreateChallengeManager(backend *gorm.DB) (*ChallengeManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	c := &ChallengeManager{
		DB: backend,
	}
	// table challenges
	if err := backend.AutoMigrate(&Challenge{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (challenges): %w", err)
	}
	// table categories
	if err := backend.AutoMigrate(&Category{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (categories): %w", err)
	}
	return c, nil
}

// Create challenge
func (m *ChallengeManager) Create(challenge Challenge) error {
	if err := m.DB.Create(&challenge).Error; err != nil {
		return fmt.Errorf("Create Challenge %w", err)
	}
	return nil
}

// Create category
func (m *ChallengeManager) CreateCategory(category Category) error {
	if err := m.DB.Create(&category).Error; err != nil {
		return fmt.Errorf("Create Category: %w", err)
	}
	return nil
}

// GetByID to get a challenge by id and entity
func (m *ChallengeManager) GetByID(id uint, entID uint) (Challenge, error) {
	var challenge Challenge
	if err := m.DB.Where("id = ? AND ent_id = ?", id, entID).First(&challenge).Error; err != nil {
		return Challenge{}, fmt.Errorf("Get Challenge by ID and Entity: %w", err)
	}
	return challenge, nil
}

// GetCategoryByID to get a category by id and entity id
func (m *ChallengeManager) GetCategoryByID(id uint, entID uint) (Category, error) {
	var category Category
	if err := m.DB.Where("id = ? AND ent_id = ?", id, entID).First(&category).Error; err != nil {
		return Category{}, fmt.Errorf("Get Category by ID and Entity: %w", err)
	}
	return category, nil
}

// ExistCategory to check if a category exists name
func (m *ChallengeManager) ExistCategory(name string, entID uint) bool {
	var count int64
	if err := m.DB.Model(&Category{}).Where("name = ? AND ent_id = ?", name, entID).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// New empty challenge
func (m *ChallengeManager) New(title, description string, categoryID uint, active bool, points, bonus, bonusDecay, penalty int, flag, hint string, entID uint) Challenge {
	return Challenge{
		Title:       title,
		Description: description,
		CategoryID:  categoryID,
		Active:      active,
		Points:      points,
		Bonus:       bonus,
		BonusDecay:  bonusDecay,
		Flag:        flag,
		Hint:        hint,
		Penalty:     penalty,
		EntID:       entID,
	}
}

// New empty category
func (m *ChallengeManager) NewCategory(name, description, logo string, entID uint) (Category, error) {
	if !m.ExistCategory(name, entID) {
		return Category{
			Name:        name,
			Description: description,
			Logo:        logo,
			EntID:       entID,
		}, nil
	}
	return Category{}, fmt.Errorf("Category with name '%s' already exists", name)
}

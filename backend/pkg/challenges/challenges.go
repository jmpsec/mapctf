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
	Country     string
	Active      bool
	Points      int
	Bonus       int
	BonusDecay  int
	Flag        string
	Hint        string
	HintPenalty int
	HelpPenalty int
	UUID        string `gorm:"index"`
}

// Category to hold all challenge categories
type Category struct {
	gorm.Model
	Name        string `gorm:"index"`
	Description string
	Logo        string
	UUID        string `gorm:"index"`
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
	if err := backend.AutoMigrate(&Challenge{}); err != nil {
		return nil, fmt.Errorf("Failed to AutoMigrate table (challenges): %w", err)
	}
	// Best-effort legacy migration: copy old penalty values into hint_penalty.
	if backend.Migrator().HasColumn(&Challenge{}, "penalty") {
		if err := backend.Exec("UPDATE challenges SET hint_penalty = penalty WHERE hint_penalty = 0").Error; err != nil {
			return nil, fmt.Errorf("failed to migrate legacy challenge penalty values: %w", err)
		}
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

// CreateAndReturn challenge and populate its generated fields (ID, timestamps)
func (m *ChallengeManager) CreateAndReturn(challenge *Challenge) error {
	if err := m.DB.Create(challenge).Error; err != nil {
		return fmt.Errorf("Create Challenge %w", err)
	}
	return nil
}

// Update challenge
func (m *ChallengeManager) Update(challenge Challenge) error {
	if err := m.DB.Model(&Challenge{}).
		Where("id = ? AND uuid = ?", challenge.ID, challenge.UUID).
		Updates(map[string]interface{}{
			"title":        challenge.Title,
			"description":  challenge.Description,
			"category_id":  challenge.CategoryID,
			"country":      challenge.Country,
			"active":       challenge.Active,
			"points":       challenge.Points,
			"bonus":        challenge.Bonus,
			"bonus_decay":  challenge.BonusDecay,
			"flag":         challenge.Flag,
			"hint":         challenge.Hint,
			"hint_penalty": challenge.HintPenalty,
			"help_penalty": challenge.HelpPenalty,
		}).Error; err != nil {
		return fmt.Errorf("Update Challenge %w", err)
	}
	return nil
}

// Delete challenge
func (m *ChallengeManager) Delete(id uint, uuid string) error {
	if err := m.DB.Where("id = ? AND uuid = ?", id, uuid).Delete(&Challenge{}).Error; err != nil {
		return fmt.Errorf("Delete Challenge %w", err)
	}
	return nil
}

// DeleteAll soft-deletes all challenges for a specific UUID
func (m *ChallengeManager) DeleteAll(uuid string) (int64, error) {
	result := m.DB.Where("uuid = ?", uuid).Delete(&Challenge{})
	if result.Error != nil {
		return 0, fmt.Errorf("Delete All Challenges %w", result.Error)
	}
	return result.RowsAffected, nil
}

// SetAllActive updates active state for all challenges of a specific UUID
func (m *ChallengeManager) SetAllActive(uuid string, active bool) (int64, error) {
	result := m.DB.Model(&Challenge{}).Where("uuid = ?", uuid).Update("active", active)
	if result.Error != nil {
		return 0, fmt.Errorf("Set All Challenge Active %w", result.Error)
	}
	return result.RowsAffected, nil
}

// Create category
func (m *ChallengeManager) CreateCategory(category Category) error {
	if err := m.DB.Create(&category).Error; err != nil {
		return fmt.Errorf("Create Category: %w", err)
	}
	return nil
}

// UpdateCategory updates an existing category by id and uuid
func (m *ChallengeManager) UpdateCategory(category Category) error {
	if err := m.DB.Model(&Category{}).
		Where("id = ? AND uuid = ?", category.ID, category.UUID).
		Updates(map[string]interface{}{
			"name":        category.Name,
			"description": category.Description,
			"logo":        category.Logo,
		}).Error; err != nil {
		return fmt.Errorf("Update Category: %w", err)
	}
	return nil
}

// DeleteCategory deletes a category by id and uuid
func (m *ChallengeManager) DeleteCategory(id uint, uuid string) error {
	if err := m.DB.Where("id = ? AND uuid = ?", id, uuid).Delete(&Category{}).Error; err != nil {
		return fmt.Errorf("Delete Category: %w", err)
	}
	return nil
}

// CategoryHasChallenges checks whether any challenge references the category
func (m *ChallengeManager) CategoryHasChallenges(categoryID uint, uuid string) (bool, error) {
	var count int64
	if err := m.DB.Model(&Challenge{}).
		Where("category_id = ? AND uuid = ?", categoryID, uuid).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("Category Has Challenges: %w", err)
	}
	return count > 0, nil
}

// GetByID to get a challenge by id and entity
func (m *ChallengeManager) GetByID(id uint, uuid string) (Challenge, error) {
	var challenge Challenge
	if err := m.DB.Where("id = ? AND uuid = ?", id, uuid).First(&challenge).Error; err != nil {
		return Challenge{}, fmt.Errorf("Get Challenge by ID and Entity: %w", err)
	}
	return challenge, nil
}

// GetAll to get all challenges for a specific entity ID
func (m *ChallengeManager) GetAll(uuid string) ([]Challenge, error) {
	var challenges []Challenge
	if err := m.DB.Where("uuid = ?", uuid).Find(&challenges).Error; err != nil {
		return challenges, fmt.Errorf("Get All Challenges by Entity: %w", err)
	}
	return challenges, nil
}

// GetActive to get all active challenges
func (m *ChallengeManager) GetActive(uuid string) ([]Challenge, error) {
	var challenges []Challenge
	if err := m.DB.Where("uuid = ? AND active = ?", uuid, true).Find(&challenges).Error; err != nil {
		return challenges, fmt.Errorf("Get Active Challenges by Entity: %w", err)
	}
	return challenges, nil
}

// GetAllCategories to get all categories for a specific entity ID
func (m *ChallengeManager) GetAllCategories(uuid string) ([]Category, error) {
	var categories []Category
	if err := m.DB.Where("uuid = ?", uuid).Find(&categories).Error; err != nil {
		return categories, fmt.Errorf("Get All Categories by Entity: %w", err)
	}
	return categories, nil
}

// DeleteAllCategories soft-deletes all categories for a specific UUID
func (m *ChallengeManager) DeleteAllCategories(uuid string) (int64, error) {
	result := m.DB.Where("uuid = ?", uuid).Delete(&Category{})
	if result.Error != nil {
		return 0, fmt.Errorf("Delete All Categories %w", result.Error)
	}
	return result.RowsAffected, nil
}

// GetCategoryByID to get a category by id and entity id
func (m *ChallengeManager) GetCategoryByID(id uint, uuid string) (Category, error) {
	var category Category
	if err := m.DB.Where("id = ? AND uuid = ?", id, uuid).First(&category).Error; err != nil {
		return Category{}, fmt.Errorf("Get Category by ID and Entity: %w", err)
	}
	return category, nil
}

// ExistCategory to check if a category exists name
func (m *ChallengeManager) ExistCategory(name string, uuid string) bool {
	var count int64
	if err := m.DB.Model(&Category{}).Where("name = ? AND uuid = ?", name, uuid).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// New empty challenge
func (m *ChallengeManager) New(title, description string, categoryID uint, country string, active bool, points, bonus, bonusDecay, hintPenalty, helpPenalty int, flag, hint string, uuid string) Challenge {
	return Challenge{
		Title:       title,
		Description: description,
		CategoryID:  categoryID,
		Country:     country,
		Active:      active,
		Points:      points,
		Bonus:       bonus,
		BonusDecay:  bonusDecay,
		Flag:        flag,
		Hint:        hint,
		HintPenalty: hintPenalty,
		HelpPenalty: helpPenalty,
		UUID:        uuid,
	}
}

// New empty category
func (m *ChallengeManager) NewCategory(name, description, logo string, uuid string) (Category, error) {
	if !m.ExistCategory(name, uuid) {
		return Category{
			Name:        name,
			Description: description,
			Logo:        logo,
			UUID:        uuid,
		}, nil
	}
	return Category{}, fmt.Errorf("Category with name '%s' already exists", name)
}

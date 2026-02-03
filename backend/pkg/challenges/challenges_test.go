package challenges

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

// TestCreateChallengeManager tests the CreateChallengeManager function
func TestCreateChallengeManager(t *testing.T) {
	db := setupTestDB(t)

	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil ChallengeManager")
	}

	if manager.DB == nil {
		t.Fatal("Expected non-nil DB in ChallengeManager")
	}

	// Verify tables were created
	if !db.Migrator().HasTable(&Challenge{}) {
		t.Error("Expected challenges table to be created")
	}

	if !db.Migrator().HasTable(&Category{}) {
		t.Error("Expected categories table to be created")
	}
}

// TestCreateChallengeManagerWithNilDB tests CreateChallengeManager with nil database
func TestCreateChallengeManagerWithNilDB(t *testing.T) {
	_, err := CreateChallengeManager(nil)
	if err == nil {
		t.Error("Expected error when creating ChallengeManager with nil DB")
	}
}

// TestCreateChallengeManagerAutoMigrateErrors tests AutoMigrate error handling
func TestCreateChallengeManagerAutoMigrateErrors(t *testing.T) {
	db := setupTestDB(t)

	// Close the database connection to cause AutoMigrate errors
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	// Now try to create challenge manager - should fail on AutoMigrate
	_, err = CreateChallengeManager(db)
	if err == nil {
		t.Error("Expected error when AutoMigrate fails")
	}
}

// TestCreate tests creating a challenge
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	challenge := Challenge{
		Title:       "Test Challenge",
		Description: "Test Description",
		CategoryID:  1,
		Active:      true,
		Points:      100,
		Bonus:       10,
		BonusDecay:  5,
		Flag:        "flag{test}",
		Hint:        "Test hint",
		Penalty:     5,
		EntID:       1,
	}

	err = manager.Create(challenge)
	if err != nil {
		t.Errorf("Failed to create challenge: %v", err)
	}

	// Verify challenge was created
	var retrieved Challenge
	if err := db.First(&retrieved).Error; err != nil {
		t.Errorf("Failed to retrieve created challenge: %v", err)
	}

	if retrieved.Title != "Test Challenge" {
		t.Errorf("Expected title 'Test Challenge', got '%s'", retrieved.Title)
	}
	if retrieved.Points != 100 {
		t.Errorf("Expected points 100, got %d", retrieved.Points)
	}
}

// TestCreateCategory tests creating a category
func TestCreateCategory(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	category := Category{
		Name:        "Web",
		Description: "Web challenges",
		Logo:        "web.png",
		EntID:       1,
	}

	err = manager.CreateCategory(category)
	if err != nil {
		t.Errorf("Failed to create category: %v", err)
	}

	// Verify category was created
	var retrieved Category
	if err := db.First(&retrieved).Error; err != nil {
		t.Errorf("Failed to retrieve created category: %v", err)
	}

	if retrieved.Name != "Web" {
		t.Errorf("Expected name 'Web', got '%s'", retrieved.Name)
	}
}

// TestGetByID tests retrieving a challenge by ID and entity ID
func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create a challenge
	challenge := Challenge{
		Title:  "Test Challenge",
		Points: 100,
		EntID:  1,
	}
	if err := manager.Create(challenge); err != nil {
		t.Fatalf("Failed to create challenge: %v", err)
	}

	// Retrieve by ID
	retrieved, err := manager.GetByID(1, 1)
	if err != nil {
		t.Errorf("Failed to get challenge by ID: %v", err)
	}

	if retrieved.Title != "Test Challenge" {
		t.Errorf("Expected title 'Test Challenge', got '%s'", retrieved.Title)
	}
}

// TestGetByIDWrongEntity tests retrieving a challenge with wrong entity ID
func TestGetByIDWrongEntity(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create a challenge with entity ID 1
	challenge := Challenge{
		Title:  "Test Challenge",
		Points: 100,
		EntID:  1,
	}
	if err := manager.Create(challenge); err != nil {
		t.Fatalf("Failed to create challenge: %v", err)
	}

	// Try to retrieve with different entity ID
	_, err = manager.GetByID(1, 2)
	if err == nil {
		t.Error("Expected error when retrieving challenge with wrong entity ID")
	}
}

// TestGetByIDNotFound tests retrieving a non-existent challenge
func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	_, err = manager.GetByID(999, 1)
	if err == nil {
		t.Error("Expected error when retrieving non-existent challenge")
	}
}

// TestGetCategoryByID tests retrieving a category by ID and entity ID
func TestGetCategoryByID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create a category
	category := Category{
		Name:  "Crypto",
		EntID: 1,
	}
	if err := manager.CreateCategory(category); err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Retrieve by ID
	retrieved, err := manager.GetCategoryByID(1, 1)
	if err != nil {
		t.Errorf("Failed to get category by ID: %v", err)
	}

	if retrieved.Name != "Crypto" {
		t.Errorf("Expected name 'Crypto', got '%s'", retrieved.Name)
	}
}

// TestGetCategoryByIDWrongEntity tests retrieving a category with wrong entity ID
func TestGetCategoryByIDWrongEntity(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create a category with entity ID 1
	category := Category{
		Name:  "Reversing",
		EntID: 1,
	}
	if err := manager.CreateCategory(category); err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Try to retrieve with different entity ID
	_, err = manager.GetCategoryByID(1, 2)
	if err == nil {
		t.Error("Expected error when retrieving category with wrong entity ID")
	}
}

// TestGetCategoryByIDNotFound tests retrieving a non-existent category
func TestGetCategoryByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	_, err = manager.GetCategoryByID(999, 1)
	if err == nil {
		t.Error("Expected error when retrieving non-existent category")
	}
}

// TestExistCategory tests checking if a category exists
func TestExistCategory(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Category should not exist initially
	if manager.ExistCategory("Forensics", 1) {
		t.Error("Expected category 'Forensics' to not exist")
	}

	// Create a category
	category := Category{
		Name:  "Forensics",
		EntID: 1,
	}
	if err := manager.CreateCategory(category); err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Now it should exist
	if !manager.ExistCategory("Forensics", 1) {
		t.Error("Expected category 'Forensics' to exist")
	}

	// Should not exist for different entity
	if manager.ExistCategory("Forensics", 2) {
		t.Error("Expected category 'Forensics' to not exist for entity 2")
	}
}

// TestNew tests creating a new challenge object
func TestNew(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	challenge := manager.New(
		"SQL Injection",
		"Find the SQL injection vulnerability",
		1,
		true,
		200,
		20,
		10,
		5,
		"flag{sql_injection}",
		"Check the login form",
		1,
	)

	if challenge.Title != "SQL Injection" {
		t.Errorf("Expected title 'SQL Injection', got '%s'", challenge.Title)
	}
	if challenge.Description != "Find the SQL injection vulnerability" {
		t.Errorf("Expected description to match, got '%s'", challenge.Description)
	}
	if challenge.CategoryID != 1 {
		t.Errorf("Expected category ID 1, got %d", challenge.CategoryID)
	}
	if !challenge.Active {
		t.Error("Expected challenge to be active")
	}
	if challenge.Points != 200 {
		t.Errorf("Expected points 200, got %d", challenge.Points)
	}
	if challenge.Bonus != 20 {
		t.Errorf("Expected bonus 20, got %d", challenge.Bonus)
	}
	if challenge.BonusDecay != 10 {
		t.Errorf("Expected bonus decay 10, got %d", challenge.BonusDecay)
	}
	if challenge.Penalty != 5 {
		t.Errorf("Expected penalty 5, got %d", challenge.Penalty)
	}
	if challenge.Flag != "flag{sql_injection}" {
		t.Errorf("Expected flag 'flag{sql_injection}', got '%s'", challenge.Flag)
	}
	if challenge.Hint != "Check the login form" {
		t.Errorf("Expected hint 'Check the login form', got '%s'", challenge.Hint)
	}
	if challenge.EntID != 1 {
		t.Errorf("Expected entity ID 1, got %d", challenge.EntID)
	}
}

// TestNewCategory tests creating a new category object
func TestNewCategory(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	category, err := manager.NewCategory("Binary", "Binary exploitation", "binary.png", 1)
	if err != nil {
		t.Errorf("Failed to create new category: %v", err)
	}

	if category.Name != "Binary" {
		t.Errorf("Expected name 'Binary', got '%s'", category.Name)
	}
	if category.Description != "Binary exploitation" {
		t.Errorf("Expected description to match, got '%s'", category.Description)
	}
	if category.Logo != "binary.png" {
		t.Errorf("Expected logo 'binary.png', got '%s'", category.Logo)
	}
	if category.EntID != 1 {
		t.Errorf("Expected entity ID 1, got %d", category.EntID)
	}
}

// TestNewCategoryDuplicate tests creating a duplicate category
func TestNewCategoryDuplicate(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create first category
	category := Category{
		Name:  "Pwn",
		EntID: 1,
	}
	if err := manager.CreateCategory(category); err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Try to create duplicate
	_, err = manager.NewCategory("Pwn", "Pwning challenges", "pwn.png", 1)
	if err == nil {
		t.Error("Expected error when creating duplicate category")
	}
}

// TestNewCategorySameNameDifferentEntity tests creating same-named category for different entities
func TestNewCategorySameNameDifferentEntity(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create category for entity 1
	category1 := Category{
		Name:  "Misc",
		EntID: 1,
	}
	if err := manager.CreateCategory(category1); err != nil {
		t.Fatalf("Failed to create category for entity 1: %v", err)
	}

	// Create same-named category for entity 2 should succeed
	category2, err := manager.NewCategory("Misc", "Miscellaneous", "misc.png", 2)
	if err != nil {
		t.Errorf("Expected to create category with same name for different entity, got error: %v", err)
	}

	if category2.Name != "Misc" || category2.EntID != 2 {
		t.Error("Expected category to be created for entity 2")
	}
}

// TestMultiEntityChallengeIsolation tests entity isolation for challenges
func TestMultiEntityChallengeIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create challenges for different entities
	challenge1 := Challenge{
		Title:  "Entity 1 Challenge",
		EntID:  1,
		Points: 100,
	}
	challenge2 := Challenge{
		Title:  "Entity 2 Challenge",
		EntID:  2,
		Points: 200,
	}

	if err := manager.Create(challenge1); err != nil {
		t.Fatalf("Failed to create challenge 1: %v", err)
	}
	if err := manager.Create(challenge2); err != nil {
		t.Fatalf("Failed to create challenge 2: %v", err)
	}

	// Entity 1 should only see its challenge
	c1, err := manager.GetByID(1, 1)
	if err != nil {
		t.Errorf("Failed to get challenge for entity 1: %v", err)
	}
	if c1.Title != "Entity 1 Challenge" {
		t.Errorf("Expected 'Entity 1 Challenge', got '%s'", c1.Title)
	}

	// Entity 2 should only see its challenge
	c2, err := manager.GetByID(2, 2)
	if err != nil {
		t.Errorf("Failed to get challenge for entity 2: %v", err)
	}
	if c2.Title != "Entity 2 Challenge" {
		t.Errorf("Expected 'Entity 2 Challenge', got '%s'", c2.Title)
	}

	// Entity 1 should not see entity 2's challenge
	_, err = manager.GetByID(2, 1)
	if err == nil {
		t.Error("Expected error when entity 1 tries to access entity 2's challenge")
	}
}

// TestMultiEntityCategoryIsolation tests entity isolation for categories
func TestMultiEntityCategoryIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateChallengeManager(db)
	if err != nil {
		t.Fatalf("Failed to create ChallengeManager: %v", err)
	}

	// Create categories for different entities
	category1 := Category{
		Name:  "Entity 1 Category",
		EntID: 1,
	}
	category2 := Category{
		Name:  "Entity 2 Category",
		EntID: 2,
	}

	if err := manager.CreateCategory(category1); err != nil {
		t.Fatalf("Failed to create category 1: %v", err)
	}
	if err := manager.CreateCategory(category2); err != nil {
		t.Fatalf("Failed to create category 2: %v", err)
	}

	// Entity 1 should only see its category
	cat1, err := manager.GetCategoryByID(1, 1)
	if err != nil {
		t.Errorf("Failed to get category for entity 1: %v", err)
	}
	if cat1.Name != "Entity 1 Category" {
		t.Errorf("Expected 'Entity 1 Category', got '%s'", cat1.Name)
	}

	// Entity 2 should only see its category
	cat2, err := manager.GetCategoryByID(2, 2)
	if err != nil {
		t.Errorf("Failed to get category for entity 2: %v", err)
	}
	if cat2.Name != "Entity 2 Category" {
		t.Errorf("Expected 'Entity 2 Category', got '%s'", cat2.Name)
	}

	// Entity 1 should not see entity 2's category
	_, err = manager.GetCategoryByID(2, 1)
	if err == nil {
		t.Error("Expected error when entity 1 tries to access entity 2's category")
	}
}

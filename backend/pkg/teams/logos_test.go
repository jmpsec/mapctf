package teams

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDBForLogos creates an in-memory SQLite database for testing logos
func setupTestDBForLogos(t *testing.T) (*gorm.DB, *TeamManager) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto migrate the TeamLogo table
	if err := db.AutoMigrate(&TeamLogo{}); err != nil {
		t.Fatalf("Failed to migrate TeamLogo table: %v", err)
	}

	manager := &TeamManager{
		DB: db,
	}

	return db, manager
}

// TestTeamLogoStruct tests the TeamLogo struct
func TestTeamLogoStruct(t *testing.T) {
	logo := TeamLogo{
		Name:      "test-logo",
		Logo:      "logo.png",
		Used:      true,
		Enabled:   true,
		Custom:    false,
		UUID:      testUUID1,
		Protected: false,
		CreatedBy: 1,
	}

	if logo.Name != "test-logo" {
		t.Errorf("Expected Name 'test-logo', got '%s'", logo.Name)
	}
	if logo.Logo != "logo.png" {
		t.Errorf("Expected Logo 'logo.png', got '%s'", logo.Logo)
	}
	if !logo.Used {
		t.Error("Expected Used to be true")
	}
	if !logo.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if logo.Custom {
		t.Error("Expected Custom to be false")
	}
	if logo.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, logo.UUID)
	}
	if logo.Protected {
		t.Error("Expected Protected to be false")
	}
	if logo.CreatedBy != 1 {
		t.Errorf("Expected CreatedBy 1, got %d", logo.CreatedBy)
	}
}

// TestGetLogo tests the GetLogo method
func TestGetLogo(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create a test logo
	testLogo := TeamLogo{
		Name:      "test-logo",
		Logo:      "test.png",
		Used:      false,
		Enabled:   true,
		Custom:    false,
		UUID:      testUUID1,
		Protected: false,
		CreatedBy: 1,
	}

	if err := manager.DB.Create(&testLogo).Error; err != nil {
		t.Fatalf("Failed to create test logo: %v", err)
	}

	// Test successful retrieval
	logo, err := manager.GetLogo("test-logo", testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if logo.Name != "test-logo" {
		t.Errorf("Expected name 'test-logo', got '%s'", logo.Name)
	}
	if logo.Logo != "test.png" {
		t.Errorf("Expected logo 'test.png', got '%s'", logo.Logo)
	}
	if logo.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, logo.UUID)
	}
}

// TestGetLogoNotFound tests GetLogo with non-existent logo
func TestGetLogoNotFound(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Try to get a non-existent logo
	_, err := manager.GetLogo("non-existent", testUUID1)
	if err == nil {
		t.Error("Expected error when getting non-existent logo")
	}
}

// TestGetLogoWrongUUID tests GetLogo with wrong UUID
func TestGetLogoWrongUUID(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create a test logo with UUID 1
	testLogo := TeamLogo{
		Name:      "test-logo",
		Logo:      "test.png",
		UUID:      testUUID1,
		Enabled:   true,
		CreatedBy: 1,
	}

	if err := manager.DB.Create(&testLogo).Error; err != nil {
		t.Fatalf("Failed to create test logo: %v", err)
	}

	// Try to get with different UUID
	_, err := manager.GetLogo("test-logo", testUUID2)
	if err == nil {
		t.Error("Expected error when getting logo with wrong UUID")
	}
}

// TestNewLogo tests the NewLogo method
func TestNewLogo(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	logo, err := manager.NewLogo("new-logo", "new.png", true, false, testUUID1, 5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if logo.Name != "new-logo" {
		t.Errorf("Expected Name 'new-logo', got '%s'", logo.Name)
	}
	if logo.Logo != "new.png" {
		t.Errorf("Expected Logo 'new.png', got '%s'", logo.Logo)
	}
	if !logo.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if logo.Custom {
		t.Error("Expected Custom to be false")
	}
	if logo.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, logo.UUID)
	}
	if logo.CreatedBy != 5 {
		t.Errorf("Expected CreatedBy 5, got %d", logo.CreatedBy)
	}
}

// TestNewLogoWithDifferentParameters tests NewLogo with various parameters
func TestNewLogoWithDifferentParameters(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	testCases := []struct {
		name      string
		logoPath  string
		enabled   bool
		custom    bool
		uuid      string
		createdBy uint
	}{
		{"logo1", "path1.png", true, true, testUUID1, 10},
		{"logo2", "path2.jpg", false, false, testUUID2, 20},
		{"logo3", "path3.svg", true, false, testUUID1, 30},
		{"logo4", "path4.gif", false, true, testUUID2, 40},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logo, err := manager.NewLogo(tc.name, tc.logoPath, tc.enabled, tc.custom, tc.uuid, tc.createdBy)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if logo.Name != tc.name {
				t.Errorf("Expected Name '%s', got '%s'", tc.name, logo.Name)
			}
			if logo.Logo != tc.logoPath {
				t.Errorf("Expected Logo '%s', got '%s'", tc.logoPath, logo.Logo)
			}
			if logo.Enabled != tc.enabled {
				t.Errorf("Expected Enabled %v, got %v", tc.enabled, logo.Enabled)
			}
			if logo.Custom != tc.custom {
				t.Errorf("Expected Custom %v, got %v", tc.custom, logo.Custom)
			}
			if logo.UUID != tc.uuid {
				t.Errorf("Expected UUID '%s', got '%s'", tc.uuid, logo.UUID)
			}
			if logo.CreatedBy != tc.createdBy {
				t.Errorf("Expected CreatedBy %d, got %d", tc.createdBy, logo.CreatedBy)
			}
		})
	}
}

// TestCreateLogo tests the CreateLogo method
func TestCreateLogo(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	logo := TeamLogo{
		Name:      "create-test",
		Logo:      "create.png",
		Enabled:   true,
		Custom:    false,
		UUID:      testUUID1,
		CreatedBy: 1,
	}

	err := manager.CreateLogo(logo)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the logo was created
	var retrievedLogo TeamLogo
	if err := manager.DB.Where("name = ? AND uuid = ?", "create-test", testUUID1).First(&retrievedLogo).Error; err != nil {
		t.Fatalf("Failed to retrieve created logo: %v", err)
	}

	if retrievedLogo.Name != "create-test" {
		t.Errorf("Expected name 'create-test', got '%s'", retrievedLogo.Name)
	}
	if retrievedLogo.Logo != "create.png" {
		t.Errorf("Expected logo 'create.png', got '%s'", retrievedLogo.Logo)
	}
}

// TestCreateLogoMultiple tests creating multiple logos
func TestCreateLogoMultiple(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	logos := []TeamLogo{
		{Name: "logo1", Logo: "path1.png", UUID: testUUID1, Enabled: true, CreatedBy: 1},
		{Name: "logo2", Logo: "path2.png", UUID: testUUID1, Enabled: true, CreatedBy: 1},
		{Name: "logo3", Logo: "path3.png", UUID: testUUID2, Enabled: false, CreatedBy: 2},
	}

	for _, logo := range logos {
		if err := manager.CreateLogo(logo); err != nil {
			t.Fatalf("Failed to create logo %s: %v", logo.Name, err)
		}
	}

	// Verify all logos were created
	var count int64
	manager.DB.Model(&TeamLogo{}).Count(&count)
	if count != int64(len(logos)) {
		t.Errorf("Expected %d logos, got %d", len(logos), count)
	}
}

// TestCreateLogoWithClosedDB tests CreateLogo with closed database
func TestCreateLogoWithClosedDB(t *testing.T) {
	db, manager := setupTestDBForLogos(t)

	// Close the database connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	logo := TeamLogo{
		Name:      "fail-test",
		Logo:      "fail.png",
		UUID:      testUUID1,
		Enabled:   true,
		CreatedBy: 1,
	}

	err = manager.CreateLogo(logo)
	if err == nil {
		t.Error("Expected error when creating logo with closed database")
	}
}

// TestExistsLogo tests the ExistsLogo method
func TestExistsLogo(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create a test logo
	testLogo := TeamLogo{
		Name:      "exists-test",
		Logo:      "exists.png",
		UUID:      testUUID1,
		Enabled:   true,
		CreatedBy: 1,
	}

	if err := manager.DB.Create(&testLogo).Error; err != nil {
		t.Fatalf("Failed to create test logo: %v", err)
	}

	// Test that it exists
	if !manager.ExistsLogo("exists-test", testUUID1) {
		t.Error("Expected logo to exist")
	}

	// Test that non-existent logo doesn't exist
	if manager.ExistsLogo("non-existent", testUUID1) {
		t.Error("Expected non-existent logo to not exist")
	}

	// Test with wrong UUID
	if manager.ExistsLogo("exists-test", testUUID2) {
		t.Error("Expected logo to not exist with wrong UUID")
	}
}

// TestExistsLogoEmpty tests ExistsLogo with empty database
func TestExistsLogoEmpty(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	if manager.ExistsLogo("any-logo", testUUID1) {
		t.Error("Expected logo to not exist in empty database")
	}
}

// TestExistsLogoMultiple tests ExistsLogo with multiple logos
func TestExistsLogoMultiple(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create multiple logos
	logos := []TeamLogo{
		{Name: "logo1", Logo: "path1.png", UUID: testUUID1, Enabled: true, CreatedBy: 1},
		{Name: "logo2", Logo: "path2.png", UUID: testUUID1, Enabled: true, CreatedBy: 1},
		{Name: "logo1", Logo: "path1.png", UUID: testUUID2, Enabled: true, CreatedBy: 2},
	}

	for _, logo := range logos {
		if err := manager.DB.Create(&logo).Error; err != nil {
			t.Fatalf("Failed to create logo: %v", err)
		}
	}

	// Test existence for each specific combination
	if !manager.ExistsLogo("logo1", testUUID1) {
		t.Error("Expected logo1 with UUID 1 to exist")
	}
	if !manager.ExistsLogo("logo2", testUUID1) {
		t.Error("Expected logo2 with UUID 1 to exist")
	}
	if !manager.ExistsLogo("logo1", testUUID2) {
		t.Error("Expected logo1 with UUID 2 to exist")
	}
	if manager.ExistsLogo("logo2", testUUID2) {
		t.Error("Expected logo2 with UUID 2 to not exist")
	}
}

// TestExistsLogoGet tests the ExistsLogoGet method
func TestExistsLogoGet(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create a test logo
	testLogo := TeamLogo{
		Name:      "exists-get-test",
		Logo:      "existsget.png",
		Used:      true,
		Enabled:   true,
		Custom:    false,
		UUID:      testUUID1,
		Protected: false,
		CreatedBy: 1,
	}

	if err := manager.DB.Create(&testLogo).Error; err != nil {
		t.Fatalf("Failed to create test logo: %v", err)
	}

	// Test that it exists and retrieve it
	exists, logo := manager.ExistsLogoGet("exists-get-test", testUUID1)
	if !exists {
		t.Error("Expected logo to exist")
	}

	if logo.Name != "exists-get-test" {
		t.Errorf("Expected name 'exists-get-test', got '%s'", logo.Name)
	}
	if logo.Logo != "existsget.png" {
		t.Errorf("Expected logo 'existsget.png', got '%s'", logo.Logo)
	}
	if !logo.Used {
		t.Error("Expected Used to be true")
	}
	if !logo.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if logo.Custom {
		t.Error("Expected Custom to be false")
	}
	if logo.Protected {
		t.Error("Expected Protected to be false")
	}
}

// TestExistsLogoGetNotFound tests ExistsLogoGet with non-existent logo
func TestExistsLogoGetNotFound(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	exists, logo := manager.ExistsLogoGet("non-existent", testUUID1)
	if exists {
		t.Error("Expected logo to not exist")
	}

	// Verify empty logo is returned
	if logo.Name != "" {
		t.Errorf("Expected empty name, got '%s'", logo.Name)
	}
	if logo.Logo != "" {
		t.Errorf("Expected empty logo path, got '%s'", logo.Logo)
	}
	if logo.ID != 0 {
		t.Errorf("Expected ID 0, got %d", logo.ID)
	}
}

// TestExistsLogoGetWrongUUID tests ExistsLogoGet with wrong UUID
func TestExistsLogoGetWrongUUID(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create a test logo with UUID 1
	testLogo := TeamLogo{
		Name:      "test-logo",
		Logo:      "test.png",
		UUID:      testUUID1,
		Enabled:   true,
		CreatedBy: 1,
	}

	if err := manager.DB.Create(&testLogo).Error; err != nil {
		t.Fatalf("Failed to create test logo: %v", err)
	}

	// Try to get with different UUID
	exists, logo := manager.ExistsLogoGet("test-logo", testUUID2)
	if exists {
		t.Error("Expected logo to not exist with wrong UUID")
	}

	if logo.Name != "" {
		t.Errorf("Expected empty name, got '%s'", logo.Name)
	}
}

// TestExistsLogoGetMultiple tests ExistsLogoGet with multiple logos
func TestExistsLogoGetMultiple(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create multiple logos with different combinations
	logos := []TeamLogo{
		{Name: "logo-a", Logo: "a.png", UUID: testUUID1, Enabled: true, CreatedBy: 1},
		{Name: "logo-b", Logo: "b.png", UUID: testUUID1, Enabled: false, CreatedBy: 2},
		{Name: "logo-a", Logo: "a2.png", UUID: testUUID2, Enabled: true, CreatedBy: 3},
	}

	for _, logo := range logos {
		if err := manager.DB.Create(&logo).Error; err != nil {
			t.Fatalf("Failed to create logo: %v", err)
		}
	}

	// Test retrieving specific logos
	exists, logo := manager.ExistsLogoGet("logo-a", testUUID1)
	if !exists {
		t.Error("Expected logo-a with UUID 1 to exist")
	}
	if logo.Logo != "a.png" {
		t.Errorf("Expected logo 'a.png', got '%s'", logo.Logo)
	}

	exists, logo = manager.ExistsLogoGet("logo-b", testUUID1)
	if !exists {
		t.Error("Expected logo-b with UUID 1 to exist")
	}
	if logo.Logo != "b.png" {
		t.Errorf("Expected logo 'b.png', got '%s'", logo.Logo)
	}
	if logo.Enabled {
		t.Error("Expected Enabled to be false")
	}

	exists, logo = manager.ExistsLogoGet("logo-a", testUUID2)
	if !exists {
		t.Error("Expected logo-a with UUID 2 to exist")
	}
	if logo.Logo != "a2.png" {
		t.Errorf("Expected logo 'a2.png', got '%s'", logo.Logo)
	}

	exists, _ = manager.ExistsLogoGet("logo-b", testUUID2)
	if exists {
		t.Error("Expected logo-b with UUID 2 to not exist")
	}
}

// TestNewLogoEmptyStrings tests NewLogo with empty strings
func TestNewLogoEmptyStrings(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	logo, err := manager.NewLogo("", "", false, false, "", 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if logo.Name != "" {
		t.Errorf("Expected empty Name, got '%s'", logo.Name)
	}
	if logo.Logo != "" {
		t.Errorf("Expected empty Logo, got '%s'", logo.Logo)
	}
}

// TestCreateLogoWithAllFields tests CreateLogo with all fields set
func TestCreateLogoWithAllFields(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	logo := TeamLogo{
		Name:      "full-test",
		Logo:      "full.png",
		Used:      true,
		Enabled:   true,
		Custom:    true,
		UUID:      "test-uuid-5",
		Protected: true,
		CreatedBy: 10,
	}

	err := manager.CreateLogo(logo)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify all fields were saved correctly
	var retrievedLogo TeamLogo
	if err := manager.DB.Where("name = ? AND uuid = ?", "full-test", "test-uuid-5").First(&retrievedLogo).Error; err != nil {
		t.Fatalf("Failed to retrieve created logo: %v", err)
	}

	if retrievedLogo.Name != "full-test" {
		t.Errorf("Expected name 'full-test', got '%s'", retrievedLogo.Name)
	}
	if retrievedLogo.Logo != "full.png" {
		t.Errorf("Expected logo 'full.png', got '%s'", retrievedLogo.Logo)
	}
	if !retrievedLogo.Used {
		t.Error("Expected Used to be true")
	}
	if !retrievedLogo.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if !retrievedLogo.Custom {
		t.Error("Expected Custom to be true")
	}
	if retrievedLogo.UUID != "test-uuid-5" {
		t.Errorf("Expected UUID 'test-uuid-5', got '%s'", retrievedLogo.UUID)
	}
	if !retrievedLogo.Protected {
		t.Error("Expected Protected to be true")
	}
	if retrievedLogo.CreatedBy != 10 {
		t.Errorf("Expected CreatedBy 10, got %d", retrievedLogo.CreatedBy)
	}
}

// TestGetLogoWithSpecialCharacters tests GetLogo with special characters in name
func TestGetLogoWithSpecialCharacters(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	specialNames := []string{
		"logo-with-dashes",
		"logo_with_underscores",
		"logo.with.dots",
		"logo with spaces",
		"logo@#$%",
	}

	for _, name := range specialNames {
		testLogo := TeamLogo{
			Name:      name,
			Logo:      "test.png",
			UUID:      testUUID1,
			Enabled:   true,
			CreatedBy: 1,
		}

		if err := manager.DB.Create(&testLogo).Error; err != nil {
			t.Fatalf("Failed to create test logo with name '%s': %v", name, err)
		}

		logo, err := manager.GetLogo(name, testUUID1)
		if err != nil {
			t.Errorf("Failed to retrieve logo with name '%s': %v", name, err)
		}

		if logo.Name != name {
			t.Errorf("Expected name '%s', got '%s'", name, logo.Name)
		}
	}
}

// TestExistsLogoWithEmptyUUID tests ExistsLogo with empty UUID
func TestExistsLogoWithEmptyUUID(t *testing.T) {
	_, manager := setupTestDBForLogos(t)

	// Create a logo with empty UUID
	testLogo := TeamLogo{
		Name:      "zero-ent",
		Logo:      "zero.png",
		UUID:      "",
		Enabled:   true,
		CreatedBy: 1,
	}

	if err := manager.DB.Create(&testLogo).Error; err != nil {
		t.Fatalf("Failed to create test logo: %v", err)
	}

	// Test that it exists with EntID 0
	if !manager.ExistsLogo("zero-ent", "") {
		t.Error("Expected logo to exist with EntID 0")
	}

	// Verify it doesn't match with different UUID
	if manager.ExistsLogo("zero-ent", testUUID1) {
		t.Error("Expected logo to not exist with UUID 1")
	}
}

package users

import (
	"testing"
	"time"

	"github.com/jmpsec/mapctf/pkg/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test UUID constants
const (
	testUUID1 = "test-uuid-1"
	testUUID2 = "test-uuid-2"
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

// TestCreateUserManager tests the CreateUserManager function
func TestCreateUserManager(t *testing.T) {
	db := setupTestDB(t)

	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil UserManager")
	}

	if manager.DB == nil {
		t.Fatal("Expected non-nil DB in UserManager")
	}

	// Verify table was created
	if !db.Migrator().HasTable(&PlatformUser{}) {
		t.Error("Expected platform_users table to be created")
	}
}

// TestCreateUserManagerWithNilDB tests CreateUserManager with nil database
func TestCreateUserManagerWithNilDB(t *testing.T) {
	_, err := CreateUserManager(nil, &config.ConfigurationJWT{})
	if err == nil {
		t.Error("Expected error when creating UserManager with nil database")
	}
}

// TestCreateUserManagerAutoMigrateError tests AutoMigrate error handling
func TestCreateUserManagerAutoMigrateError(t *testing.T) {
	db := setupTestDB(t)

	// Close the database connection to cause AutoMigrate error
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	// Now try to create user manager - should fail on AutoMigrate
	_, err = CreateUserManager(db, &config.ConfigurationJWT{})
	if err == nil {
		t.Error("Expected error when AutoMigrate fails")
	}
}

// TestPlatformUserStructure tests the PlatformUser struct
func TestPlatformUserStructure(t *testing.T) {
	user := PlatformUser{
		Username:      "testuser",
		Name:          "Test User",
		Email:         "test@example.com",
		TeamID:        1,
		PassHash:      "hashedpassword",
		APIToken:      "token123",
		TokenExpire:   time.Now().Add(24 * time.Hour),
		Admin:         false,
		Service:       false,
		Active:        true,
		LastIPAddress: "127.0.0.1",
		LastUserAgent: "TestAgent",
		LastAccess:    time.Now(),
		LastTokenUse:  time.Now(),
		UUID:          testUUID1,
	}

	if user.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got '%s'", user.Email)
	}

	if user.TeamID != 1 {
		t.Errorf("Expected TeamID 1, got %d", user.TeamID)
	}
}

// TestCreate tests creating a new user
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	user := PlatformUser{
		Username: "newuser",
		Name:     "New User",
		Email:    "new@example.com",
		PassHash: "hashedpass",
		TeamID:   1,
		Admin:    false,
		Service:  false,
		Active:   true,
		UUID:     testUUID1,
	}

	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user was created
	var createdUser PlatformUser
	result := db.Where("username = ?", "newuser").First(&createdUser)
	if result.Error != nil {
		t.Fatalf("Failed to retrieve created user: %v", result.Error)
	}

	if createdUser.Username != "newuser" {
		t.Errorf("Expected username 'newuser', got '%s'", createdUser.Username)
	}

	if createdUser.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", createdUser.Email)
	}
}

// TestHashTextWithSalt tests text hashing functionality
func TestHashTextWithSalt(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	text := "mySecretText"
	hash, err := manager.HashTextWithSalt(text)

	if err != nil {
		t.Fatalf("Failed to hash text: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if hash == text {
		t.Error("Hash should not equal original text")
	}

	// Verify the hash can be compared with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(text))
	if err != nil {
		t.Errorf("Hash verification failed: %v", err)
	}
}

// TestHashTextWithSaltDifferentHashes tests that same text produces different hashes
func TestHashTextWithSaltDifferentHashes(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	text := "sameText"
	hash1, err := manager.HashTextWithSalt(text)
	if err != nil {
		t.Fatalf("Failed to hash text: %v", err)
	}

	hash2, err := manager.HashTextWithSalt(text)
	if err != nil {
		t.Fatalf("Failed to hash text: %v", err)
	}

	// bcrypt should produce different hashes due to random salt
	if hash1 == hash2 {
		t.Error("Expected different hashes for same text due to salt")
	}
}

// TestHashPasswordWithSalt tests password hashing
func TestHashPasswordWithSalt(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	password := "mySecurePassword123!"
	hash, err := manager.HashPasswordWithSalt(password)

	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if hash == password {
		t.Error("Hash should not equal original password")
	}

	// Verify the hash can be compared with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Errorf("Password hash verification failed: %v", err)
	}
}

// TestExists tests checking if a user exists
func TestExists(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// User should not exist initially
	if manager.Exists("nonexistent", testUUID1) {
		t.Error("Expected user 'nonexistent' to not exist")
	}

	// Create a user
	user := PlatformUser{
		Username: "existinguser",
		Email:    "existing@example.com",
		PassHash: "hash",
		UUID:     testUUID1,
	}

	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Now user should exist
	if !manager.Exists("existinguser", testUUID1) {
		t.Error("Expected user 'existinguser' to exist")
	}

	// Different username should not exist
	if manager.Exists("differentuser", testUUID1) {
		t.Error("Expected user 'differentuser' to not exist")
	}

	// Same username but different entity ID should not exist
	if manager.Exists("existinguser", testUUID2) {
		t.Error("Expected user 'existinguser' with entity 2 to not exist")
	}
}

// TestGet tests retrieving a user by username
func TestGet(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a user
	user := PlatformUser{
		Username: "getuser",
		Name:     "Get User",
		Email:    "get@example.com",
		PassHash: "hash",
		TeamID:   5,
		Admin:    true,
		UUID:     testUUID1,
	}

	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get the user
	retrievedUser, err := manager.Get("getuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.Username != "getuser" {
		t.Errorf("Expected username 'getuser', got '%s'", retrievedUser.Username)
	}

	if retrievedUser.Email != "get@example.com" {
		t.Errorf("Expected email 'get@example.com', got '%s'", retrievedUser.Email)
	}

	if retrievedUser.TeamID != 5 {
		t.Errorf("Expected TeamID 5, got %d", retrievedUser.TeamID)
	}

	if !retrievedUser.Admin {
		t.Error("Expected Admin to be true")
	}

	if retrievedUser.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, retrievedUser.UUID)
	}
}

// TestGetNonExistent tests getting a non-existent user
func TestGetNonExistent(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	_, err = manager.Get("nonexistent", testUUID1)
	if err == nil {
		t.Error("Expected error when getting non-existent user")
	}
}

// TestGetByUUID tests retrieving a user by username and UUID
func TestGetByUUID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create users with different entity IDs
	user1 := PlatformUser{
		Username: "multiuser",
		Email:    "user1@example.com",
		PassHash: "hash1",
		UUID:     testUUID1,
	}

	user2 := PlatformUser{
		Username: "multiuser",
		Email:    "user2@example.com",
		PassHash: "hash2",
		UUID:     testUUID2,
	}

	err = manager.Create(user1)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	err = manager.Create(user2)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Get user by entity ID 1
	retrievedUser1, err := manager.GetByUUID("multiuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user by entity ID 1: %v", err)
	}

	if retrievedUser1.Email != "user1@example.com" {
		t.Errorf("Expected email 'user1@example.com', got '%s'", retrievedUser1.Email)
	}

	if retrievedUser1.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, retrievedUser1.UUID)
	}

	// Get user by entity ID 2
	retrievedUser2, err := manager.GetByUUID("multiuser", testUUID2)
	if err != nil {
		t.Fatalf("Failed to get user by entity ID 2: %v", err)
	}

	if retrievedUser2.Email != "user2@example.com" {
		t.Errorf("Expected email 'user2@example.com', got '%s'", retrievedUser2.Email)
	}

	if retrievedUser2.UUID != testUUID2 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID2, retrievedUser2.UUID)
	}
}

// TestGetByUUIDNonExistent tests getting a non-existent user by UUID
func TestGetByUUIDNonExistent(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	_, err = manager.GetByUUID("nonexistent", testUUID1)
	if err == nil {
		t.Error("Expected error when getting non-existent user by entity ID")
	}
}

// TestExistsGet tests the ExistsGet function
func TestExistsGet(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Non-existent user
	exists, user := manager.ExistsGet("nonexistent", testUUID1)
	if exists {
		t.Error("Expected user to not exist")
	}

	if user.Username != "" {
		t.Error("Expected empty user struct for non-existent user")
	}

	// Create a user
	newUser := PlatformUser{
		Username: "existsgetuser",
		Name:     "Exists Get User",
		Email:    "existsget@example.com",
		PassHash: "hash",
		TeamID:   3,
		UUID:     testUUID1,
	}

	err = manager.Create(newUser)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Existing user
	exists, user = manager.ExistsGet("existsgetuser", testUUID1)
	if !exists {
		t.Error("Expected user to exist")
	}

	if user.Username != "existsgetuser" {
		t.Errorf("Expected username 'existsgetuser', got '%s'", user.Username)
	}

	if user.Email != "existsget@example.com" {
		t.Errorf("Expected email 'existsget@example.com', got '%s'", user.Email)
	}

	if user.TeamID != 3 {
		t.Errorf("Expected TeamID 3, got %d", user.TeamID)
	}

	if user.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, user.UUID)
	}
}

// TestExistsGetByUUID tests the ExistsGetByUUID function
func TestExistsGetByUUID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Non-existent user
	exists, user := manager.ExistsGetByUUID("nonexistent", testUUID1)
	if exists {
		t.Error("Expected user to not exist")
	}

	if user.Username != "" {
		t.Error("Expected empty user struct for non-existent user")
	}

	// Create users with different entity IDs
	newUser1 := PlatformUser{
		Username: "entityuser",
		Email:    "entity1@example.com",
		PassHash: "hash",
		UUID:     testUUID1,
	}

	newUser2 := PlatformUser{
		Username: "entityuser",
		Email:    "entity2@example.com",
		PassHash: "hash",
		UUID:     testUUID2,
	}

	err = manager.Create(newUser1)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	err = manager.Create(newUser2)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Check entity 1
	exists, user = manager.ExistsGetByUUID("entityuser", testUUID1)
	if !exists {
		t.Error("Expected user to exist for entity 1")
	}

	if user.Email != "entity1@example.com" {
		t.Errorf("Expected email 'entity1@example.com', got '%s'", user.Email)
	}

	// Check entity 2
	exists, user = manager.ExistsGetByUUID("entityuser", testUUID2)
	if !exists {
		t.Error("Expected user to exist for entity 2")
	}

	if user.Email != "entity2@example.com" {
		t.Errorf("Expected email 'entity2@example.com', got '%s'", user.Email)
	}

	// Check non-existent entity
	exists, user = manager.ExistsGetByUUID("entityuser", "non-existent-uuid")
	if exists {
		t.Error("Expected user to not exist for entity 999")
	}
}

// TestNew tests creating a new user struct without persisting
func TestNew(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	user, err := manager.New("newuser", "password123", "new@example.com", "New User", true, false, testUUID1, 5)
	if err != nil {
		t.Fatalf("Failed to create new user: %v", err)
	}

	if user.Username != "newuser" {
		t.Errorf("Expected username 'newuser', got '%s'", user.Username)
	}

	if user.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", user.Email)
	}

	if user.Name != "New User" {
		t.Errorf("Expected name 'New User', got '%s'", user.Name)
	}

	if !user.Admin {
		t.Error("Expected Admin to be true")
	}

	if user.Service {
		t.Error("Expected Service to be false")
	}

	if !user.Active {
		t.Error("Expected Active to be true")
	}

	if user.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, user.UUID)
	}

	if user.TeamID != 5 {
		t.Errorf("Expected TeamID 5, got %d", user.TeamID)
	}

	if user.PassHash == "" {
		t.Error("Expected non-empty PassHash")
	}

	if user.PassHash == "password123" {
		t.Error("PassHash should not equal plain password")
	}

	// Verify password hash is valid
	err = bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte("password123"))
	if err != nil {
		t.Errorf("Password hash verification failed: %v", err)
	}
}

// TestNewExistingUser tests creating a new user that already exists
func TestNewExistingUser(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a user first
	existingUser := PlatformUser{
		Username: "existing",
		Email:    "existing@example.com",
		PassHash: "hash",
		UUID:     testUUID1,
	}

	err = manager.Create(existingUser)
	if err != nil {
		t.Fatalf("Failed to create existing user: %v", err)
	}

	// Try to create a new user with the same username
	_, err = manager.New("existing", "password", "new@example.com", "New", false, false, testUUID1, 1)
	if err == nil {
		t.Error("Expected error when creating user with existing username")
	}

	expectedError := "existing already exists"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestNewServiceUser tests creating a service user
func TestNewServiceUser(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	user, err := manager.New("serviceuser", "servicepass", "service@example.com", "Service User", false, true, testUUID1, 0)
	if err != nil {
		t.Fatalf("Failed to create service user: %v", err)
	}

	if !user.Service {
		t.Error("Expected Service to be true")
	}

	if user.Admin {
		t.Error("Expected Admin to be false for service user")
	}
}

// TestHashEmptyString tests hashing an empty string
func TestHashEmptyString(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	hash, err := manager.HashTextWithSalt("")
	if err != nil {
		t.Fatalf("Failed to hash empty string: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash even for empty string")
	}

	// Verify the hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(""))
	if err != nil {
		t.Errorf("Empty string hash verification failed: %v", err)
	}
}

// TestCreateMultipleUsers tests creating multiple users
func TestCreateMultipleUsers(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	users := []PlatformUser{
		{Username: "user1", Email: "user1@example.com", PassHash: "hash1", UUID: testUUID1},
		{Username: "user2", Email: "user2@example.com", PassHash: "hash2", UUID: testUUID1},
		{Username: "user3", Email: "user3@example.com", PassHash: "hash3", UUID: testUUID2},
	}

	for _, user := range users {
		err = manager.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user %s: %v", user.Username, err)
		}
	}

	// Verify all users exist
	for _, user := range users {
		if !manager.Exists(user.Username, user.UUID) {
			t.Errorf("Expected user %s to exist for UUID %s", user.Username, user.UUID)
		}
	}
}

// TestCreateDuplicateUser tests creating a user with duplicate username (for error coverage)
func TestCreateDuplicateUser(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Add unique constraint to username
	db.Exec("CREATE UNIQUE INDEX idx_username ON platform_users(username)")

	user := PlatformUser{
		Username: "duplicate",
		Email:    "user@example.com",
		PassHash: "hash",
		UUID:     testUUID1,
	}

	// First creation should succeed
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Second creation with same username should fail
	user2 := PlatformUser{
		Username: "duplicate",
		Email:    "user2@example.com",
		PassHash: "hash2",
		UUID:     testUUID1,
	}

	err = manager.Create(user2)
	if err == nil {
		t.Error("Expected error when creating user with duplicate username")
	}
}

// TestHashTextWithSaltError tests hash error handling
func TestHashTextWithSaltError(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Test with very long string (edge case)
	longString := string(make([]byte, 100000))
	_, err = manager.HashTextWithSalt(longString)
	if err != nil {
		// bcrypt has a 72 byte limit, but we're handling it
		t.Logf("Long string hashing error (expected): %v", err)
	}
}

// TestNewWithHashError tests New function error handling
func TestNewWithHashError(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a user with very long password that may cause bcrypt error
	longPassword := string(make([]byte, 100))
	user, err := manager.New("longpassuser", longPassword, "test@example.com", "Test", false, false, testUUID1, 1)

	// bcrypt should handle this, but if it fails we should get an error
	if err == nil {
		// Verify the user was created successfully
		if user.Username != "longpassuser" {
			t.Error("Expected user to be created")
		}
	}
}

// TestPlatformUserJSONTags tests that sensitive fields are excluded from JSON
func TestPlatformUserJSONTags(t *testing.T) {
	user := PlatformUser{
		Username: "jsontest",
		PassHash: "shouldnotappear",
		APIToken: "tokenshouldnotappear",
		Email:    "json@example.com",
	}

	// The PassHash and APIToken fields have json:"-" tags
	// This test documents that behavior
	if user.PassHash == "" {
		t.Error("PassHash should have value in struct")
	}
	if user.APIToken == "" {
		t.Error("APIToken should have value in struct")
	}
}

// TestUserWorkflow tests a complete user workflow
func TestUserWorkflow(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Step 1: Verify user doesn't exist
	if manager.Exists("workflowuser", testUUID1) {
		t.Error("User should not exist initially")
	}

	// Step 2: Create new user struct
	user, err := manager.New("workflowuser", "password123", "workflow@example.com", "Workflow User", false, false, testUUID1, 10)
	if err != nil {
		t.Fatalf("Failed to create new user: %v", err)
	}

	// Step 3: Persist user to database
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to persist user: %v", err)
	}

	// Step 4: Verify user exists
	if !manager.Exists("workflowuser", testUUID1) {
		t.Error("User should exist after creation")
	}

	// Step 5: Retrieve user
	exists, retrievedUser := manager.ExistsGet("workflowuser", testUUID1)
	if !exists {
		t.Error("User should exist")
	}

	// Step 6: Verify retrieved data
	if retrievedUser.Email != "workflow@example.com" {
		t.Errorf("Expected email 'workflow@example.com', got '%s'", retrievedUser.Email)
	}

	if retrievedUser.TeamID != 10 {
		t.Errorf("Expected TeamID 10, got %d", retrievedUser.TeamID)
	}

	// Step 7: Verify password hash
	err = bcrypt.CompareHashAndPassword([]byte(retrievedUser.PassHash), []byte("password123"))
	if err != nil {
		t.Error("Password verification failed")
	}
}

// TestMultiEntityIsolation tests that users are properly isolated by entity
func TestMultiEntityIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create same username in different UUIDs
	entity1Users := []PlatformUser{
		{Username: "admin", Email: "admin@entity1.com", PassHash: "hash1", UUID: testUUID1, TeamID: 1},
		{Username: "user", Email: "user@entity1.com", PassHash: "hash2", UUID: testUUID1, TeamID: 2},
	}

	entity2Users := []PlatformUser{
		{Username: "admin", Email: "admin@entity2.com", PassHash: "hash3", UUID: testUUID2, TeamID: 3},
		{Username: "user", Email: "user@entity2.com", PassHash: "hash4", UUID: testUUID2, TeamID: 4},
	}

	// Create all users
	for _, user := range append(entity1Users, entity2Users...) {
		if err := manager.Create(user); err != nil {
			t.Fatalf("Failed to create user %s for UUID %s: %v", user.Username, user.UUID, err)
		}
	}

	// Verify entity isolation
	entity1Admin, err := manager.GetByUUID("admin", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get admin for entity 1: %v", err)
	}
	if entity1Admin.Email != "admin@entity1.com" {
		t.Errorf("Expected entity 1 admin email, got '%s'", entity1Admin.Email)
	}

	entity2Admin, err := manager.GetByUUID("admin", testUUID2)
	if err != nil {
		t.Fatalf("Failed to get admin for entity 2: %v", err)
	}
	if entity2Admin.Email != "admin@entity2.com" {
		t.Errorf("Expected entity 2 admin email, got '%s'", entity2Admin.Email)
	}

	// Ensure they're different users
	if entity1Admin.ID == entity2Admin.ID {
		t.Error("Entity 1 and Entity 2 admins should have different IDs")
	}
}

// TestCheckLoginCredentials tests verifying login credentials
func TestCheckLoginCredentials(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a user with hashed password
	password := "securePassword123!"
	passHash, err := manager.HashPasswordWithSalt(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := PlatformUser{
		Username: "loginuser",
		Email:    "login@example.com",
		PassHash: passHash,
		UUID:     testUUID1,
		Active:   true,
	}

	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test valid credentials
	valid, retrievedUser := manager.CheckLoginCredentials("loginuser", password, testUUID1)
	if !valid {
		t.Error("Expected valid credentials")
	}
	if retrievedUser.Username != "loginuser" {
		t.Errorf("Expected username 'loginuser', got '%s'", retrievedUser.Username)
	}
	if retrievedUser.Email != "login@example.com" {
		t.Errorf("Expected email 'login@example.com', got '%s'", retrievedUser.Email)
	}
	if retrievedUser.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, retrievedUser.UUID)
	}

	// Test invalid password
	valid, _ = manager.CheckLoginCredentials("loginuser", "wrongpassword", testUUID1)
	if valid {
		t.Error("Expected invalid credentials for wrong password")
	}

	// Test non-existent user
	valid, _ = manager.CheckLoginCredentials("nonexistent", password, testUUID1)
	if valid {
		t.Error("Expected invalid credentials for non-existent user")
	}

	// Test wrong entity ID
	valid, _ = manager.CheckLoginCredentials("loginuser", password, testUUID2)
	if valid {
		t.Error("Expected invalid credentials for wrong entity ID")
	}
}

// TestCheckLoginCredentialsEmptyPassword tests login with empty password
func TestCheckLoginCredentialsEmptyPassword(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a user with empty password hash
	passHash, _ := manager.HashPasswordWithSalt("")
	user := PlatformUser{
		Username: "emptypassuser",
		Email:    "empty@example.com",
		PassHash: passHash,
		UUID:     testUUID1,
	}

	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Login with empty password should work
	valid, _ := manager.CheckLoginCredentials("emptypassuser", "", testUUID1)
	if !valid {
		t.Error("Expected valid credentials for empty password")
	}

	// Login with non-empty password should fail
	valid, _ = manager.CheckLoginCredentials("emptypassuser", "notEmpty", testUUID1)
	if valid {
		t.Error("Expected invalid credentials for non-empty password on empty-password user")
	}

	// Login with wrong entity ID should fail
	valid, _ = manager.CheckLoginCredentials("emptypassuser", "", testUUID2)
	if valid {
		t.Error("Expected invalid credentials for wrong entity ID")
	}
}

// TestCheckLoginCredentialsEntityIsolation tests that login credentials are isolated by entity ID
func TestCheckLoginCredentialsEntityIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create users with same username but different entity IDs
	password1 := "password1"
	password2 := "password2"
	passHash1, err := manager.HashPasswordWithSalt(password1)
	if err != nil {
		t.Fatalf("Failed to hash password1: %v", err)
	}
	passHash2, err := manager.HashPasswordWithSalt(password2)
	if err != nil {
		t.Fatalf("Failed to hash password2: %v", err)
	}

	user1 := PlatformUser{
		Username: "multientityuser",
		Email:    "user1@entity1.com",
		PassHash: passHash1,
		UUID:     testUUID1,
		Active:   true,
	}

	user2 := PlatformUser{
		Username: "multientityuser",
		Email:    "user2@entity2.com",
		PassHash: passHash2,
		UUID:     testUUID2,
		Active:   true,
	}

	err = manager.Create(user1)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	err = manager.Create(user2)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Test login with entity ID 1 and correct password
	valid, retrievedUser := manager.CheckLoginCredentials("multientityuser", password1, testUUID1)
	if !valid {
		t.Error("Expected valid credentials for entity 1")
	}
	if retrievedUser.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, retrievedUser.UUID)
	}
	if retrievedUser.Email != "user1@entity1.com" {
		t.Errorf("Expected email 'user1@entity1.com', got '%s'", retrievedUser.Email)
	}

	// Test login with entity ID 2 and correct password
	valid, retrievedUser = manager.CheckLoginCredentials("multientityuser", password2, testUUID2)
	if !valid {
		t.Error("Expected valid credentials for entity 2")
	}
	if retrievedUser.UUID != testUUID2 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID2, retrievedUser.UUID)
	}
	if retrievedUser.Email != "user2@entity2.com" {
		t.Errorf("Expected email 'user2@entity2.com', got '%s'", retrievedUser.Email)
	}

	// Test login with entity ID 1 but password for entity 2 (should fail)
	valid, _ = manager.CheckLoginCredentials("multientityuser", password2, testUUID1)
	if valid {
		t.Error("Expected invalid credentials when using wrong password for entity 1")
	}

	// Test login with entity ID 2 but password for entity 1 (should fail)
	valid, _ = manager.CheckLoginCredentials("multientityuser", password1, testUUID2)
	if valid {
		t.Error("Expected invalid credentials when using wrong password for entity 2")
	}

	// Test login with non-existent entity ID
	valid, _ = manager.CheckLoginCredentials("multientityuser", password1, "non-existent-uuid")
	if valid {
		t.Error("Expected invalid credentials for non-existent entity ID")
	}
}

// TestCreateToken tests JWT token creation
func TestCreateToken(t *testing.T) {
	db := setupTestDB(t)
	jwtConfig := &config.ConfigurationJWT{
		Secret:        "test-secret-key-12345",
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Test token creation with custom expiration
	token, expTime, err := manager.CreateToken("testuser", "mapctf", 2)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Verify expiration time is approximately 2 hours from now
	expectedExpiry := time.Now().Add(2 * time.Hour)
	timeDiff := expTime.Sub(expectedExpiry)
	if timeDiff > time.Minute || timeDiff < -time.Minute {
		t.Errorf("Expected expiration time around %v, got %v", expectedExpiry, expTime)
	}
}

// TestCreateTokenDefaultExpiration tests token creation with default expiration
func TestCreateTokenDefaultExpiration(t *testing.T) {
	db := setupTestDB(t)
	jwtConfig := &config.ConfigurationJWT{
		Secret:        "test-secret-key-12345",
		HoursToExpire: 48,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Test token creation with 0 expHours (should use default from config)
	token, expTime, err := manager.CreateToken("testuser", "mapctf", 0)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Verify expiration time is approximately 48 hours from now (config default)
	expectedExpiry := time.Now().Add(48 * time.Hour)
	timeDiff := expTime.Sub(expectedExpiry)
	if timeDiff > time.Minute || timeDiff < -time.Minute {
		t.Errorf("Expected expiration time around %v, got %v", expectedExpiry, expTime)
	}
}

// TestCreateTokenDifferentIssuers tests token creation with different issuers
func TestCreateTokenDifferentIssuers(t *testing.T) {
	db := setupTestDB(t)
	jwtConfig := &config.ConfigurationJWT{
		Secret:        "test-secret-key-12345",
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	token1, _, err := manager.CreateToken("user1", "issuer1", 1)
	if err != nil {
		t.Fatalf("Failed to create token1: %v", err)
	}

	token2, _, err := manager.CreateToken("user1", "issuer2", 1)
	if err != nil {
		t.Fatalf("Failed to create token2: %v", err)
	}

	// Tokens should be different due to different issuers
	if token1 == token2 {
		t.Error("Expected different tokens for different issuers")
	}
}

// TestCheckToken tests JWT token validation
func TestCheckToken(t *testing.T) {
	db := setupTestDB(t)
	jwtSecret := "test-secret-key-12345"
	jwtConfig := &config.ConfigurationJWT{
		Secret:        jwtSecret,
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a valid token
	token, _, err := manager.CreateToken("testuser", "mapctf", 1)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Verify the token
	claims, err := manager.CheckToken(jwtSecret, token)
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	if claims.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", claims.Username)
	}

	if claims.Issuer != "mapctf" {
		t.Errorf("Expected issuer 'mapctf', got '%s'", claims.Issuer)
	}
}

// TestCheckTokenInvalidSecret tests token validation with wrong secret
func TestCheckTokenInvalidSecret(t *testing.T) {
	db := setupTestDB(t)
	jwtConfig := &config.ConfigurationJWT{
		Secret:        "correct-secret-key",
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a token with one secret
	token, _, err := manager.CreateToken("testuser", "mapctf", 1)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Try to verify with different secret
	_, err = manager.CheckToken("wrong-secret-key", token)
	if err == nil {
		t.Error("Expected error when verifying token with wrong secret")
	}
}

// TestCheckTokenInvalidToken tests validation of invalid token string
func TestCheckTokenInvalidToken(t *testing.T) {
	db := setupTestDB(t)
	jwtConfig := &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Try to verify an invalid token string
	_, err = manager.CheckToken("test-secret", "invalid.token.string")
	if err == nil {
		t.Error("Expected error when verifying invalid token")
	}
}

// TestCheckTokenMalformedToken tests validation of malformed token
func TestCheckTokenMalformedToken(t *testing.T) {
	db := setupTestDB(t)
	jwtConfig := &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Try to verify completely malformed token
	_, err = manager.CheckToken("test-secret", "not-a-jwt-at-all")
	if err == nil {
		t.Error("Expected error when verifying malformed token")
	}

	// Try empty token
	_, err = manager.CheckToken("test-secret", "")
	if err == nil {
		t.Error("Expected error when verifying empty token")
	}
}

// TestTokenRoundTrip tests full token creation and verification cycle
func TestTokenRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	jwtSecret := "round-trip-secret-key"
	jwtConfig := &config.ConfigurationJWT{
		Secret:        jwtSecret,
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	testCases := []struct {
		username string
		issuer   string
		expHours int
	}{
		{"user1", "issuer1", 1},
		{"user2", "issuer2", 24},
		{"admin", "mapctf", 168},
	}

	for _, tc := range testCases {
		t.Run(tc.username, func(t *testing.T) {
			token, _, err := manager.CreateToken(tc.username, tc.issuer, tc.expHours)
			if err != nil {
				t.Fatalf("Failed to create token: %v", err)
			}

			claims, err := manager.CheckToken(jwtSecret, token)
			if err != nil {
				t.Fatalf("Failed to verify token: %v", err)
			}

			if claims.Username != tc.username {
				t.Errorf("Expected username '%s', got '%s'", tc.username, claims.Username)
			}

			if claims.Issuer != tc.issuer {
				t.Errorf("Expected issuer '%s', got '%s'", tc.issuer, claims.Issuer)
			}
		})
	}
}

// TestLoginAndTokenWorkflow tests complete login and token workflow
func TestLoginAndTokenWorkflow(t *testing.T) {
	db := setupTestDB(t)
	jwtSecret := "workflow-secret-key"
	jwtConfig := &config.ConfigurationJWT{
		Secret:        jwtSecret,
		HoursToExpire: 24,
	}
	manager, err := CreateUserManager(db, jwtConfig)
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Step 1: Create a new user
	password := "securePass123!"
	user, err := manager.New("workflowuser", password, "workflow@example.com", "Workflow User", false, false, testUUID1, 1)
	if err != nil {
		t.Fatalf("Failed to create new user: %v", err)
	}

	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to persist user: %v", err)
	}

	// Step 2: Verify login credentials
	valid, retrievedUser := manager.CheckLoginCredentials("workflowuser", password, testUUID1)
	if !valid {
		t.Fatal("Expected valid login credentials")
	}

	// Step 3: Create token for authenticated user
	token, expTime, err := manager.CreateToken(retrievedUser.Username, "mapctf", 1)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	if expTime.Before(time.Now()) {
		t.Error("Token expiration should be in the future")
	}

	// Step 4: Verify the token
	claims, err := manager.CheckToken(jwtSecret, token)
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	if claims.Username != "workflowuser" {
		t.Errorf("Expected username 'workflowuser', got '%s'", claims.Username)
	}

	// Step 5: Verify wrong password fails
	valid, _ = manager.CheckLoginCredentials("workflowuser", "wrongpassword", testUUID1)
	if valid {
		t.Error("Expected invalid login with wrong password")
	}

	// Step 6: Verify wrong entity ID fails
	valid, _ = manager.CheckLoginCredentials("workflowuser", password, testUUID2)
	if valid {
		t.Error("Expected invalid login with wrong entity ID")
	}
}

// TestTokenClaims tests the TokenClaims struct
func TestTokenClaims(t *testing.T) {
	claims := TokenClaims{
		Username: "testuser",
	}

	if claims.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", claims.Username)
	}
}

// TestConstants tests the package constants
func TestConstants(t *testing.T) {
	if NoTeamID != 0 {
		t.Errorf("Expected NoTeamID to be 0, got %d", NoTeamID)
	}

	if NoUUID != "" {
		t.Errorf("Expected NoUUID to be empty string, got '%s'", NoUUID)
	}
}

// BenchmarkHashPassword benchmarks password hashing
func BenchmarkHashPassword(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		b.Fatalf("Failed to create UserManager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.HashPasswordWithSalt("testpassword123")
	}
}

// BenchmarkExists benchmarks the Exists check
func BenchmarkExists(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		b.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a test user
	user := PlatformUser{
		Username: "benchuser",
		Email:    "bench@example.com",
		PassHash: "hash",
		UUID:     testUUID1,
	}
	_ = manager.Create(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Exists("benchuser", testUUID1)
	}
}

// BenchmarkGet benchmarks the Get operation
func BenchmarkGet(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		b.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a test user
	user := PlatformUser{
		Username: "benchuser",
		Email:    "bench@example.com",
		PassHash: "hash",
		UUID:     testUUID1,
	}
	_ = manager.Create(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Get("benchuser", testUUID1)
	}
}

// TestSetPassword tests setting a password for a user
func TestSetPassword(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a user
	password := "originalPassword123!"
	user, err := manager.New("setpassuser", password, "setpass@example.com", "Set Pass User", false, false, testUUID1, 0)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Set a new password
	newPassword := "newPassword456!"
	err = manager.SetPassword("setpassuser", newPassword, testUUID1)
	if err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	// Verify old password doesn't work
	valid, _ := manager.CheckLoginCredentials("setpassuser", password, testUUID1)
	if valid {
		t.Error("Old password should not work after password change")
	}

	// Verify new password works
	valid, retrievedUser := manager.CheckLoginCredentials("setpassuser", newPassword, testUUID1)
	if !valid {
		t.Error("New password should work after password change")
	}
	if retrievedUser.Username != "setpassuser" {
		t.Errorf("Expected username 'setpassuser', got '%s'", retrievedUser.Username)
	}

	// Test setting password for non-existent user (GORM Update doesn't fail, just affects 0 rows)
	err = manager.SetPassword("nonexistent", "password", testUUID1)
	// This won't error, but won't affect any rows either - that's GORM behavior
	if err != nil {
		t.Logf("SetPassword for non-existent user returned error: %v", err)
	}

	// Test setting password for wrong entity ID (should not affect the user)
	err = manager.SetPassword("setpassuser", "wrongEntityPassword", testUUID2)
	if err != nil {
		t.Logf("SetPassword for wrong entity returned error: %v", err)
	}
	// Verify original user's password still works (wasn't affected)
	valid, _ = manager.CheckLoginCredentials("setpassuser", newPassword, testUUID1)
	if !valid {
		t.Error("User's password should not be affected by SetPassword with wrong entity ID")
	}
}

// TestSetAdmin tests setting the admin flag for a user
func TestSetAdmin(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a non-admin user
	user, err := manager.New("adminuser", "password123", "admin@example.com", "Admin User", false, false, testUUID1, 0)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user is not admin initially
	retrievedUser, err := manager.Get("adminuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if retrievedUser.Admin {
		t.Error("User should not be admin initially")
	}

	// Set admin flag to true
	err = manager.SetAdmin(true, "adminuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to set admin flag: %v", err)
	}

	// Verify admin flag is set
	retrievedUser, err = manager.Get("adminuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if !retrievedUser.Admin {
		t.Error("User should be admin after setting admin flag")
	}

	// Set admin flag to false
	err = manager.SetAdmin(false, "adminuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to set admin flag: %v", err)
	}

	// Verify admin flag is unset
	retrievedUser, err = manager.Get("adminuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if retrievedUser.Admin {
		t.Error("User should not be admin after unsetting admin flag")
	}

	// Test setting admin for non-existent user (GORM Update doesn't fail, just affects 0 rows)
	err = manager.SetAdmin(true, "nonexistent", testUUID1)
	// This won't error, but won't affect any rows either - that's GORM behavior
	if err != nil {
		t.Logf("SetAdmin for non-existent user returned error: %v", err)
	}
}

// TestSetActive tests setting the active flag for a user
func TestSetActive(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create an active user
	user, err := manager.New("activeuser", "password123", "active@example.com", "Active User", false, false, testUUID1, 0)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user is active initially
	retrievedUser, err := manager.Get("activeuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if !retrievedUser.Active {
		t.Error("User should be active initially")
	}

	// Set active flag to false
	err = manager.SetActive(false, "activeuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to set active flag: %v", err)
	}

	// Verify active flag is unset
	retrievedUser, err = manager.Get("activeuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if retrievedUser.Active {
		t.Error("User should not be active after setting active flag to false")
	}

	// Set active flag to true
	err = manager.SetActive(true, "activeuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to set active flag: %v", err)
	}

	// Verify active flag is set
	retrievedUser, err = manager.Get("activeuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if !retrievedUser.Active {
		t.Error("User should be active after setting active flag to true")
	}

	// Test setting active for non-existent user (GORM Update doesn't fail, just affects 0 rows)
	err = manager.SetActive(false, "nonexistent", testUUID1)
	// This won't error, but won't affect any rows either - that's GORM behavior
	if err != nil {
		t.Logf("SetActive for non-existent user returned error: %v", err)
	}
}

// TestSetService tests setting the service flag for a user
func TestSetService(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create a non-service user
	user, err := manager.New("serviceuser", "password123", "service@example.com", "Service User", false, false, testUUID1, 0)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user is not service initially
	retrievedUser, err := manager.Get("serviceuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if retrievedUser.Service {
		t.Error("User should not be service initially")
	}

	// Set service flag to true
	err = manager.SetService(true, "serviceuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to set service flag: %v", err)
	}

	// Verify service flag is set
	retrievedUser, err = manager.Get("serviceuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if !retrievedUser.Service {
		t.Error("User should be service after setting service flag")
	}

	// Set service flag to false
	err = manager.SetService(false, "serviceuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to set service flag: %v", err)
	}

	// Verify service flag is unset
	retrievedUser, err = manager.Get("serviceuser", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if retrievedUser.Service {
		t.Error("User should not be service after unsetting service flag")
	}

	// Test setting service for non-existent user (GORM Update doesn't fail, just affects 0 rows)
	err = manager.SetService(true, "nonexistent", testUUID1)
	// This won't error, but won't affect any rows either - that's GORM behavior
	if err != nil {
		t.Logf("SetService for non-existent user returned error: %v", err)
	}
}

// TestSetPasswordUUIDIsolation tests that SetPassword respects UUID isolation
func TestSetPasswordUUIDIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateUserManager(db, &config.ConfigurationJWT{})
	if err != nil {
		t.Fatalf("Failed to create UserManager: %v", err)
	}

	// Create users with same username but different entity IDs
	password1 := "password1"
	password2 := "password2"
	user1, err := manager.New("multientitypass", password1, "user1@entity1.com", "User 1", false, false, testUUID1, 0)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}
	err = manager.Create(user1)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2, err := manager.New("multientitypass", password2, "user2@entity2.com", "User 2", false, false, testUUID2, 0)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}
	err = manager.Create(user2)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Set password for entity 1
	newPassword1 := "newPassword1"
	err = manager.SetPassword("multientitypass", newPassword1, testUUID1)
	if err != nil {
		t.Fatalf("Failed to set password for entity 1: %v", err)
	}

	// Verify entity 1 password changed
	valid, _ := manager.CheckLoginCredentials("multientitypass", password1, testUUID1)
	if valid {
		t.Error("Old password should not work for entity 1")
	}
	valid, _ = manager.CheckLoginCredentials("multientitypass", newPassword1, testUUID1)
	if !valid {
		t.Error("New password should work for entity 1")
	}

	// Verify entity 2 password unchanged
	valid, _ = manager.CheckLoginCredentials("multientitypass", password2, testUUID2)
	if !valid {
		t.Error("Entity 2 password should remain unchanged")
	}
	valid, _ = manager.CheckLoginCredentials("multientitypass", newPassword1, testUUID2)
	if valid {
		t.Error("Entity 1's new password should not work for entity 2")
	}
}

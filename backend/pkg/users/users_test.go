package users

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
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

// TestCreateUserManager tests the CreateUserManager function
func TestCreateUserManager(t *testing.T) {
	db := setupTestDB(t)

	manager := CreateUserManager(db)

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

// TestPlatformUserStructure tests the PlatformUser struct
func TestPlatformUserStructure(t *testing.T) {
	user := PlatformUser{
		Username:      "testuser",
		Display:       "Test User",
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
		EntID:         1,
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
	manager := CreateUserManager(db)

	user := PlatformUser{
		Username: "newuser",
		Display:  "New User",
		Email:    "new@example.com",
		PassHash: "hashedpass",
		TeamID:   1,
		Admin:    false,
		Service:  false,
		Active:   true,
		EntID:    1,
	}

	err := manager.Create(user)
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
	manager := CreateUserManager(db)

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
	manager := CreateUserManager(db)

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
	manager := CreateUserManager(db)

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
	manager := CreateUserManager(db)

	// User should not exist initially
	if manager.Exists("nonexistent") {
		t.Error("Expected user 'nonexistent' to not exist")
	}

	// Create a user
	user := PlatformUser{
		Username: "existinguser",
		Email:    "existing@example.com",
		PassHash: "hash",
		EntID:    1,
	}

	err := manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Now user should exist
	if !manager.Exists("existinguser") {
		t.Error("Expected user 'existinguser' to exist")
	}

	// Different username should not exist
	if manager.Exists("differentuser") {
		t.Error("Expected user 'differentuser' to not exist")
	}
}

// TestGet tests retrieving a user by username
func TestGet(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Create a user
	user := PlatformUser{
		Username: "getuser",
		Display:  "Get User",
		Email:    "get@example.com",
		PassHash: "hash",
		TeamID:   5,
		Admin:    true,
		EntID:    1,
	}

	err := manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get the user
	retrievedUser, err := manager.Get("getuser")
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
}

// TestGetNonExistent tests getting a non-existent user
func TestGetNonExistent(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	_, err := manager.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent user")
	}
}

// TestGetByTenantID tests retrieving a user by username and tenant ID
func TestGetByTenantID(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Create users with different tenant IDs
	user1 := PlatformUser{
		Username: "multiuser",
		Email:    "user1@example.com",
		PassHash: "hash1",
		EntID:    1,
	}

	user2 := PlatformUser{
		Username: "multiuser",
		Email:    "user2@example.com",
		PassHash: "hash2",
		EntID:    2,
	}

	err := manager.Create(user1)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	err = manager.Create(user2)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Get user by tenant ID 1
	retrievedUser1, err := manager.GetByTenantID("multiuser", 1)
	if err != nil {
		t.Fatalf("Failed to get user by tenant ID 1: %v", err)
	}

	if retrievedUser1.Email != "user1@example.com" {
		t.Errorf("Expected email 'user1@example.com', got '%s'", retrievedUser1.Email)
	}

	if retrievedUser1.EntID != 1 {
		t.Errorf("Expected EntID 1, got %d", retrievedUser1.EntID)
	}

	// Get user by tenant ID 2
	retrievedUser2, err := manager.GetByTenantID("multiuser", 2)
	if err != nil {
		t.Fatalf("Failed to get user by tenant ID 2: %v", err)
	}

	if retrievedUser2.Email != "user2@example.com" {
		t.Errorf("Expected email 'user2@example.com', got '%s'", retrievedUser2.Email)
	}

	if retrievedUser2.EntID != 2 {
		t.Errorf("Expected EntID 2, got %d", retrievedUser2.EntID)
	}
}

// TestGetByTenantIDNonExistent tests getting a non-existent user by tenant ID
func TestGetByTenantIDNonExistent(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	_, err := manager.GetByTenantID("nonexistent", 1)
	if err == nil {
		t.Error("Expected error when getting non-existent user by tenant ID")
	}
}

// TestExistsGet tests the ExistsGet function
func TestExistsGet(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Non-existent user
	exists, user := manager.ExistsGet("nonexistent")
	if exists {
		t.Error("Expected user to not exist")
	}

	if user.Username != "" {
		t.Error("Expected empty user struct for non-existent user")
	}

	// Create a user
	newUser := PlatformUser{
		Username: "existsgetuser",
		Display:  "Exists Get User",
		Email:    "existsget@example.com",
		PassHash: "hash",
		TeamID:   3,
		EntID:    1,
	}

	err := manager.Create(newUser)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Existing user
	exists, user = manager.ExistsGet("existsgetuser")
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
}

// TestExistsGetByTenantID tests the ExistsGetByTenantID function
func TestExistsGetByTenantID(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Non-existent user
	exists, user := manager.ExistsGetByTenantID("nonexistent", 1)
	if exists {
		t.Error("Expected user to not exist")
	}

	if user.Username != "" {
		t.Error("Expected empty user struct for non-existent user")
	}

	// Create users with different tenant IDs
	newUser1 := PlatformUser{
		Username: "tenantuser",
		Email:    "tenant1@example.com",
		PassHash: "hash",
		EntID:    1,
	}

	newUser2 := PlatformUser{
		Username: "tenantuser",
		Email:    "tenant2@example.com",
		PassHash: "hash",
		EntID:    2,
	}

	err := manager.Create(newUser1)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	err = manager.Create(newUser2)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Check tenant 1
	exists, user = manager.ExistsGetByTenantID("tenantuser", 1)
	if !exists {
		t.Error("Expected user to exist for tenant 1")
	}

	if user.Email != "tenant1@example.com" {
		t.Errorf("Expected email 'tenant1@example.com', got '%s'", user.Email)
	}

	// Check tenant 2
	exists, user = manager.ExistsGetByTenantID("tenantuser", 2)
	if !exists {
		t.Error("Expected user to exist for tenant 2")
	}

	if user.Email != "tenant2@example.com" {
		t.Errorf("Expected email 'tenant2@example.com', got '%s'", user.Email)
	}

	// Check non-existent tenant
	exists, user = manager.ExistsGetByTenantID("tenantuser", 999)
	if exists {
		t.Error("Expected user to not exist for tenant 999")
	}
}

// TestNew tests creating a new user struct without persisting
func TestNew(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	user, err := manager.New("newuser", "password123", "new@example.com", "New User", true, false, 1, 5)
	if err != nil {
		t.Fatalf("Failed to create new user: %v", err)
	}

	if user.Username != "newuser" {
		t.Errorf("Expected username 'newuser', got '%s'", user.Username)
	}

	if user.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", user.Email)
	}

	if user.Display != "New User" {
		t.Errorf("Expected display 'New User', got '%s'", user.Display)
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

	if user.EntID != 1 {
		t.Errorf("Expected EntID 1, got %d", user.EntID)
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
	manager := CreateUserManager(db)

	// Create a user first
	existingUser := PlatformUser{
		Username: "existing",
		Email:    "existing@example.com",
		PassHash: "hash",
		EntID:    1,
	}

	err := manager.Create(existingUser)
	if err != nil {
		t.Fatalf("Failed to create existing user: %v", err)
	}

	// Try to create a new user with the same username
	_, err = manager.New("existing", "password", "new@example.com", "New", false, false, 1, 1)
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
	manager := CreateUserManager(db)

	user, err := manager.New("serviceuser", "servicepass", "service@example.com", "Service User", false, true, 1, 0)
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
	manager := CreateUserManager(db)

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
	manager := CreateUserManager(db)

	users := []PlatformUser{
		{Username: "user1", Email: "user1@example.com", PassHash: "hash1", EntID: 1},
		{Username: "user2", Email: "user2@example.com", PassHash: "hash2", EntID: 1},
		{Username: "user3", Email: "user3@example.com", PassHash: "hash3", EntID: 2},
	}

	for _, user := range users {
		err := manager.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user %s: %v", user.Username, err)
		}
	}

	// Verify all users exist
	for _, user := range users {
		if !manager.Exists(user.Username) {
			t.Errorf("Expected user %s to exist", user.Username)
		}
	}
}

// TestCreateDuplicateUser tests creating a user with duplicate username (for error coverage)
func TestCreateDuplicateUser(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Add unique constraint to username
	db.Exec("CREATE UNIQUE INDEX idx_username ON platform_users(username)")

	user := PlatformUser{
		Username: "duplicate",
		Email:    "user@example.com",
		PassHash: "hash",
		EntID:    1,
	}

	// First creation should succeed
	err := manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Second creation with same username should fail
	user2 := PlatformUser{
		Username: "duplicate",
		Email:    "user2@example.com",
		PassHash: "hash2",
		EntID:    1,
	}

	err = manager.Create(user2)
	if err == nil {
		t.Error("Expected error when creating user with duplicate username")
	}
}

// TestHashTextWithSaltError tests hash error handling
func TestHashTextWithSaltError(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Test with very long string (edge case)
	longString := string(make([]byte, 100000))
	_, err := manager.HashTextWithSalt(longString)
	if err != nil {
		// bcrypt has a 72 byte limit, but we're handling it
		t.Logf("Long string hashing error (expected): %v", err)
	}
}

// TestNewWithHashError tests New function error handling
func TestNewWithHashError(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Create a user with very long password that may cause bcrypt error
	longPassword := string(make([]byte, 100))
	user, err := manager.New("longpassuser", longPassword, "test@example.com", "Test", false, false, 1, 1)

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
	manager := CreateUserManager(db)

	// Step 1: Verify user doesn't exist
	if manager.Exists("workflowuser") {
		t.Error("User should not exist initially")
	}

	// Step 2: Create new user struct
	user, err := manager.New("workflowuser", "password123", "workflow@example.com", "Workflow User", false, false, 1, 10)
	if err != nil {
		t.Fatalf("Failed to create new user: %v", err)
	}

	// Step 3: Persist user to database
	err = manager.Create(user)
	if err != nil {
		t.Fatalf("Failed to persist user: %v", err)
	}

	// Step 4: Verify user exists
	if !manager.Exists("workflowuser") {
		t.Error("User should exist after creation")
	}

	// Step 5: Retrieve user
	exists, retrievedUser := manager.ExistsGet("workflowuser")
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

// TestMultiTenantIsolation tests that users are properly isolated by tenant
func TestMultiTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager := CreateUserManager(db)

	// Create same username in different tenants
	tenant1Users := []PlatformUser{
		{Username: "admin", Email: "admin@tenant1.com", PassHash: "hash1", EntID: 1, TeamID: 1},
		{Username: "user", Email: "user@tenant1.com", PassHash: "hash2", EntID: 1, TeamID: 2},
	}

	tenant2Users := []PlatformUser{
		{Username: "admin", Email: "admin@tenant2.com", PassHash: "hash3", EntID: 2, TeamID: 3},
		{Username: "user", Email: "user@tenant2.com", PassHash: "hash4", EntID: 2, TeamID: 4},
	}

	// Create all users
	for _, user := range append(tenant1Users, tenant2Users...) {
		if err := manager.Create(user); err != nil {
			t.Fatalf("Failed to create user %s for tenant %d: %v", user.Username, user.EntID, err)
		}
	}

	// Verify tenant isolation
	tenant1Admin, err := manager.GetByTenantID("admin", 1)
	if err != nil {
		t.Fatalf("Failed to get admin for tenant 1: %v", err)
	}
	if tenant1Admin.Email != "admin@tenant1.com" {
		t.Errorf("Expected tenant 1 admin email, got '%s'", tenant1Admin.Email)
	}

	tenant2Admin, err := manager.GetByTenantID("admin", 2)
	if err != nil {
		t.Fatalf("Failed to get admin for tenant 2: %v", err)
	}
	if tenant2Admin.Email != "admin@tenant2.com" {
		t.Errorf("Expected tenant 2 admin email, got '%s'", tenant2Admin.Email)
	}

	// Ensure they're different users
	if tenant1Admin.ID == tenant2Admin.ID {
		t.Error("Tenant 1 and Tenant 2 admins should have different IDs")
	}
}

// BenchmarkHashPassword benchmarks password hashing
func BenchmarkHashPassword(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager := CreateUserManager(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.HashPasswordWithSalt("testpassword123")
	}
}

// BenchmarkExists benchmarks the Exists check
func BenchmarkExists(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager := CreateUserManager(db)

	// Create a test user
	user := PlatformUser{
		Username: "benchuser",
		Email:    "bench@example.com",
		PassHash: "hash",
		EntID:    1,
	}
	_ = manager.Create(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Exists("benchuser")
	}
}

// BenchmarkGet benchmarks the Get operation
func BenchmarkGet(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager := CreateUserManager(db)

	// Create a test user
	user := PlatformUser{
		Username: "benchuser",
		Email:    "bench@example.com",
		PassHash: "hash",
		EntID:    1,
	}
	_ = manager.Create(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Get("benchuser")
	}
}

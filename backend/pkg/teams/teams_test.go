package teams

import (
	"fmt"
	"testing"
	"time"

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

// TestCreateTeams tests the CreateTeams function
func TestCreateTeams(t *testing.T) {
	db := setupTestDB(t)

	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil TeamManager")
	}

	if manager.DB == nil {
		t.Fatal("Expected non-nil DB in TeamManager")
	}

	// Verify tables were created
	if !db.Migrator().HasTable(&PlatformTeam{}) {
		t.Error("Expected platform_teams table to be created")
	}

	if !db.Migrator().HasTable(&TeamMembership{}) {
		t.Error("Expected team_memberships table to be created")
	}

	if !db.Migrator().HasTable(&TeamScore{}) {
		t.Error("Expected team_scores table to be created")
	}
}

// TestCreateTeamsWithNilDB tests CreateTeams with nil database
func TestCreateTeamsWithNilDB(t *testing.T) {
	_, err := CreateTeams(nil)
	if err == nil {
		t.Error("Expected error when creating TeamManager with nil DB")
	}
}

// TestCreateTeamsAutoMigrateErrors tests AutoMigrate error handling
func TestCreateTeamsAutoMigrateErrors(t *testing.T) {
	db := setupTestDB(t)

	// Close the database connection to cause AutoMigrate errors
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	// Now try to create teams - should fail on AutoMigrate
	_, err = CreateTeams(db)
	if err == nil {
		t.Error("Expected error when AutoMigrate fails")
	}
}

// TestCreateTeamsSecondAutoMigrateError tests the second AutoMigrate error path
func TestCreateTeamsSecondAutoMigrateError(t *testing.T) {
	// Skip this test as SQLite's AutoMigrate is very permissive and doesn't fail
	// with conflicting table structures - it simply adds missing columns
	t.Skip("SQLite AutoMigrate doesn't reliably fail with conflicting schemas")
}

// TestCreateTeamsThirdAutoMigrateError tests the third AutoMigrate error path
func TestCreateTeamsThirdAutoMigrateError(t *testing.T) {
	// Skip this test as SQLite's AutoMigrate is very permissive and doesn't fail
	// with conflicting table structures - it simply adds missing columns
	t.Skip("SQLite AutoMigrate doesn't reliably fail with conflicting schemas")
}

// TestPlatformTeamStructure tests the PlatformTeam struct
func TestPlatformTeamStructure(t *testing.T) {
	now := time.Now()
	team := PlatformTeam{
		Name:      "Test Team",
		Logo:      "logo.png",
		Points:    100,
		LastScore: now,
		Protected: false,
		Visible:   true,
		Active:    true,
		UUID:      testUUID1,
	}

	if team.Name != "Test Team" {
		t.Errorf("Expected Name 'Test Team', got '%s'", team.Name)
	}

	if team.Logo != "logo.png" {
		t.Errorf("Expected Logo 'logo.png', got '%s'", team.Logo)
	}

	if team.Points != 100 {
		t.Errorf("Expected Points 100, got %d", team.Points)
	}

	if !team.Visible {
		t.Error("Expected Visible to be true")
	}

	if !team.Active {
		t.Error("Expected Active to be true")
	}

	if team.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, team.UUID)
	}
}

// TestTeamMembershipStructure tests the TeamMembership struct
func TestTeamMembershipStructure(t *testing.T) {
	membership := TeamMembership{
		TeamID:     1,
		UserID:     2,
		UUID:       testUUID1,
		AssignedBy: 4,
	}

	if membership.TeamID != 1 {
		t.Errorf("Expected TeamID 1, got %d", membership.TeamID)
	}

	if membership.UserID != 2 {
		t.Errorf("Expected UserID 2, got %d", membership.UserID)
	}

	if membership.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, membership.UUID)
	}

	if membership.AssignedBy != 4 {
		t.Errorf("Expected AssignedBy 4, got %d", membership.AssignedBy)
	}
}

// TestTeamScoreStructure tests the TeamScore struct
func TestTeamScoreStructure(t *testing.T) {
	score := TeamScore{
		TeamID:   1,
		Points:   50,
		UUID:     testUUID2,
		ScoredBy: 3,
	}

	if score.TeamID != 1 {
		t.Errorf("Expected TeamID 1, got %d", score.TeamID)
	}

	if score.Points != 50 {
		t.Errorf("Expected Points 50, got %d", score.Points)
	}

	if score.UUID != testUUID2 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID2, score.UUID)
	}

	if score.ScoredBy != 3 {
		t.Errorf("Expected ScoredBy 3, got %d", score.ScoredBy)
	}
}

// TestCreate tests creating a new team
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	team := PlatformTeam{
		Name:      "New Team",
		Logo:      "new-logo.png",
		Points:    0,
		Protected: false,
		Visible:   true,
		Active:    true,
		UUID:      testUUID1,
	}

	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Verify team was created
	var createdTeam PlatformTeam
	result := db.Where("name = ?", "New Team").First(&createdTeam)
	if result.Error != nil {
		t.Fatalf("Failed to retrieve created team: %v", result.Error)
	}

	if createdTeam.Name != "New Team" {
		t.Errorf("Expected name 'New Team', got '%s'", createdTeam.Name)
	}

	if createdTeam.Logo != "new-logo.png" {
		t.Errorf("Expected logo 'new-logo.png', got '%s'", createdTeam.Logo)
	}

	if !createdTeam.Active {
		t.Error("Expected Active to be true")
	}
}

// TestExists tests checking if a team exists
func TestExists(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Team should not exist initially
	if manager.Exists("nonexistent", testUUID1) {
		t.Error("Expected team 'nonexistent' to not exist")
	}

	// Create a team
	team := PlatformTeam{
		Name:   "Existing Team",
		Logo:   "logo.png",
		UUID:   testUUID1,
		Active: true,
	}

	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Now team should exist
	if !manager.Exists("Existing Team", testUUID1) {
		t.Error("Expected team 'Existing Team' to exist")
	}

	// Different team name should not exist
	if manager.Exists("Different Team", testUUID1) {
		t.Error("Expected team 'Different Team' to not exist")
	}
}

// TestGet tests retrieving a team by name
func TestGet(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create a team
	team := PlatformTeam{
		Name:      "Get Team",
		Logo:      "get-logo.png",
		Points:    150,
		Protected: true,
		Visible:   true,
		Active:    true,
		UUID:      testUUID1,
	}

	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Get the team
	retrievedTeam, err := manager.Get("Get Team", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}

	if retrievedTeam.Name != "Get Team" {
		t.Errorf("Expected name 'Get Team', got '%s'", retrievedTeam.Name)
	}

	if retrievedTeam.Logo != "get-logo.png" {
		t.Errorf("Expected logo 'get-logo.png', got '%s'", retrievedTeam.Logo)
	}

	if retrievedTeam.Points != 150 {
		t.Errorf("Expected Points 150, got %d", retrievedTeam.Points)
	}

	if !retrievedTeam.Protected {
		t.Error("Expected Protected to be true")
	}
}

// TestGetNonExistent tests getting a non-existent team
func TestGetNonExistent(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	_, err = manager.Get("nonexistent", testUUID1)
	if err == nil {
		t.Error("Expected error when getting non-existent team")
	}
}

// TestGetByUUID tests retrieving a team by name and UUID
func TestGetByUUID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create teams with different entity IDs
	team1 := PlatformTeam{
		Name:   "Multi Team",
		Logo:   "logo1.png",
		Points: 100,
		UUID:   testUUID1,
		Active: true,
	}

	team2 := PlatformTeam{
		Name:   "Multi Team",
		Logo:   "logo2.png",
		Points: 200,
		UUID:   testUUID2,
		Active: true,
	}

	err = manager.Create(team1)
	if err != nil {
		t.Fatalf("Failed to create team1: %v", err)
	}

	err = manager.Create(team2)
	if err != nil {
		t.Fatalf("Failed to create team2: %v", err)
	}

	// Get team by entity ID 1
	retrievedTeam1, err := manager.GetByUUID("Multi Team", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get team by entity ID 1: %v", err)
	}

	if retrievedTeam1.Logo != "logo1.png" {
		t.Errorf("Expected logo 'logo1.png', got '%s'", retrievedTeam1.Logo)
	}

	if retrievedTeam1.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, retrievedTeam1.UUID)
	}

	if retrievedTeam1.Points != 100 {
		t.Errorf("Expected Points 100, got %d", retrievedTeam1.Points)
	}

	// Get team by entity ID 2
	retrievedTeam2, err := manager.GetByUUID("Multi Team", testUUID2)
	if err != nil {
		t.Fatalf("Failed to get team by entity ID 2: %v", err)
	}

	if retrievedTeam2.Logo != "logo2.png" {
		t.Errorf("Expected logo 'logo2.png', got '%s'", retrievedTeam2.Logo)
	}

	if retrievedTeam2.UUID != testUUID2 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID2, retrievedTeam2.UUID)
	}

	if retrievedTeam2.Points != 200 {
		t.Errorf("Expected Points 200, got %d", retrievedTeam2.Points)
	}
}

// TestGetByUUIDNonExistent tests getting a non-existent team by UUID
func TestGetByUUIDNonExistent(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	_, err = manager.GetByUUID("nonexistent", testUUID1)
	if err == nil {
		t.Error("Expected error when getting non-existent team by entity ID")
	}
}

// TestExistsGet tests the ExistsGet function
func TestExistsGet(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Non-existent team
	exists, team := manager.ExistsGet("nonexistent", testUUID1)
	if exists {
		t.Error("Expected team to not exist")
	}

	if team.Name != "" {
		t.Error("Expected empty team struct for non-existent team")
	}

	// Create a team
	newTeam := PlatformTeam{
		Name:      "ExistsGet Team",
		Logo:      "exists-logo.png",
		Points:    75,
		Protected: false,
		Visible:   true,
		Active:    true,
		UUID:      testUUID1,
	}

	err = manager.Create(newTeam)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Existing team
	exists, team = manager.ExistsGet("ExistsGet Team", testUUID1)
	if !exists {
		t.Error("Expected team to exist")
	}

	if team.Name != "ExistsGet Team" {
		t.Errorf("Expected name 'ExistsGet Team', got '%s'", team.Name)
	}

	if team.Logo != "exists-logo.png" {
		t.Errorf("Expected logo 'exists-logo.png', got '%s'", team.Logo)
	}

	if team.Points != 75 {
		t.Errorf("Expected Points 75, got %d", team.Points)
	}
}

// TestExistsGetByUUID tests the ExistsGetByUUID function
func TestExistsGetByUUID(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Non-existent team
	exists, team := manager.ExistsGetByUUID("nonexistent", testUUID1)
	if exists {
		t.Error("Expected team to not exist")
	}

	if team.Name != "" {
		t.Error("Expected empty team struct for non-existent team")
	}

	// Create teams with different entity IDs
	newTeam1 := PlatformTeam{
		Name:   "Entity Team",
		Logo:   "entity1-logo.png",
		Points: 50,
		UUID:   testUUID1,
		Active: true,
	}

	newTeam2 := PlatformTeam{
		Name:   "Entity Team",
		Logo:   "entity2-logo.png",
		Points: 60,
		UUID:   testUUID2,
		Active: true,
	}

	err = manager.Create(newTeam1)
	if err != nil {
		t.Fatalf("Failed to create team1: %v", err)
	}

	err = manager.Create(newTeam2)
	if err != nil {
		t.Fatalf("Failed to create team2: %v", err)
	}

	// Check entity 1
	exists, team = manager.ExistsGetByUUID("Entity Team", testUUID1)
	if !exists {
		t.Error("Expected team to exist for entity 1")
	}

	if team.Logo != "entity1-logo.png" {
		t.Errorf("Expected logo 'entity1-logo.png', got '%s'", team.Logo)
	}

	if team.Points != 50 {
		t.Errorf("Expected Points 50, got %d", team.Points)
	}

	// Check entity 2
	exists, team = manager.ExistsGetByUUID("Entity Team", testUUID2)
	if !exists {
		t.Error("Expected team to exist for entity 2")
	}

	if team.Logo != "entity2-logo.png" {
		t.Errorf("Expected logo 'entity2-logo.png', got '%s'", team.Logo)
	}

	if team.Points != 60 {
		t.Errorf("Expected Points 60, got %d", team.Points)
	}

	// Check non-existent entity
	exists, team = manager.ExistsGetByUUID("Entity Team", "non-existent-uuid")
	if exists {
		t.Error("Expected team to not exist for entity 999")
	}
}

// TestNew tests creating a new team struct without persisting
func TestNew(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	team, err := manager.New("New Team", "new-logo.png", "team@example.com", true, true, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create new team: %v", err)
	}

	if team.Name != "New Team" {
		t.Errorf("Expected name 'New Team', got '%s'", team.Name)
	}

	if team.Logo != "new-logo.png" {
		t.Errorf("Expected logo 'new-logo.png', got '%s'", team.Logo)
	}

	if !team.Protected {
		t.Error("Expected Protected to be true")
	}

	if !team.Visible {
		t.Error("Expected Visible to be true")
	}

	if !team.Active {
		t.Error("Expected Active to be true")
	}

	if team.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, team.UUID)
	}
}

// TestNewExistingTeam tests creating a new team that already exists
func TestNewExistingTeam(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create a team first
	existingTeam := PlatformTeam{
		Name:   "Existing",
		Logo:   "existing-logo.png",
		UUID:   testUUID1,
		Active: true,
	}

	err = manager.Create(existingTeam)
	if err != nil {
		t.Fatalf("Failed to create existing team: %v", err)
	}

	// Try to create a new team with the same name
	_, err = manager.New("Existing", "new-logo.png", "team@example.com", false, true, testUUID1)
	if err == nil {
		t.Error("Expected error when creating team with existing name")
	}

	expectedError := "Existing already exists"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestNewProtectedTeam tests creating a protected team
func TestNewProtectedTeam(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	team, err := manager.New("Protected Team", "protected-logo.png", "protected@example.com", true, false, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create protected team: %v", err)
	}

	if !team.Protected {
		t.Error("Expected Protected to be true")
	}

	if team.Visible {
		t.Error("Expected Visible to be false for protected team")
	}
}

// TestCreateMultipleTeams tests creating multiple teams
func TestCreateMultipleTeams(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	teams := []PlatformTeam{
		{Name: "Team 1", Logo: "logo1.png", UUID: testUUID1, Active: true},
		{Name: "Team 2", Logo: "logo2.png", UUID: testUUID1, Active: true},
		{Name: "Team 3", Logo: "logo3.png", UUID: testUUID2, Active: true},
	}

	for _, team := range teams {
		err = manager.Create(team)
		if err != nil {
			t.Fatalf("Failed to create team %s: %v", team.Name, err)
		}
	}

	// Verify all teams exist
	for _, team := range teams {
		if !manager.Exists(team.Name, team.UUID) {
			t.Errorf("Expected team %s to exist", team.Name)
		}
	}
}

// TestCreateDuplicateTeam tests creating a team with duplicate name
func TestCreateDuplicateTeam(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Add unique constraint to name
	db.Exec("CREATE UNIQUE INDEX idx_team_name ON platform_teams(name)")

	team := PlatformTeam{
		Name:   "Duplicate",
		Logo:   "logo.png",
		UUID:   testUUID1,
		Active: true,
	}

	// First creation should succeed
	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create first team: %v", err)
	}

	// Second creation with same name should fail
	team2 := PlatformTeam{
		Name:   "Duplicate",
		Logo:   "logo2.png",
		UUID:   testUUID1,
		Active: true,
	}

	err = manager.Create(team2)
	if err == nil {
		t.Error("Expected error when creating team with duplicate name")
	}
}

// TestTeamWorkflow tests a complete team workflow
func TestTeamWorkflow(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Step 1: Verify team doesn't exist
	if manager.Exists("Workflow Team", testUUID1) {
		t.Error("Team should not exist initially")
	}

	// Step 2: Create new team struct
	team, err := manager.New("Workflow Team", "workflow-logo.png", "workflow@example.com", false, true, testUUID1)
	if err != nil {
		t.Fatalf("Failed to create new team: %v", err)
	}

	// Step 3: Persist team to database
	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to persist team: %v", err)
	}

	// Step 4: Verify team exists
	if !manager.Exists("Workflow Team", testUUID1) {
		t.Error("Team should exist after creation")
	}

	// Step 5: Retrieve team
	exists, retrievedTeam := manager.ExistsGet("Workflow Team", testUUID1)
	if !exists {
		t.Error("Team should exist")
	}

	// Step 6: Verify retrieved data
	if retrievedTeam.Name != "Workflow Team" {
		t.Errorf("Expected name 'Workflow Team', got '%s'", retrievedTeam.Name)
	}

	if retrievedTeam.Logo != "workflow-logo.png" {
		t.Errorf("Expected logo 'workflow-logo.png', got '%s'", retrievedTeam.Logo)
	}

	if !retrievedTeam.Active {
		t.Error("Expected Active to be true")
	}
}

// TestMultiEntityTeamIsolation tests that teams are properly isolated by entity
func TestMultiEntityTeamIsolation(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create same team name in different UUIDs
	entity1Teams := []PlatformTeam{
		{Name: "Admin Team", Logo: "admin1.png", UUID: testUUID1, Points: 100, Active: true},
		{Name: "User Team", Logo: "user1.png", UUID: testUUID1, Points: 50, Active: true},
	}

	entity2Teams := []PlatformTeam{
		{Name: "Admin Team", Logo: "admin2.png", UUID: testUUID2, Points: 200, Active: true},
		{Name: "User Team", Logo: "user2.png", UUID: testUUID2, Points: 75, Active: true},
	}

	// Create all teams
	for _, team := range append(entity1Teams, entity2Teams...) {
		if err := manager.Create(team); err != nil {
			t.Fatalf("Failed to create team %s for UUID %s: %v", team.Name, team.UUID, err)
		}
	}

	// Verify entity isolation
	entity1Admin, err := manager.GetByUUID("Admin Team", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get admin team for entity 1: %v", err)
	}
	if entity1Admin.Logo != "admin1.png" {
		t.Errorf("Expected entity 1 admin logo, got '%s'", entity1Admin.Logo)
	}
	if entity1Admin.Points != 100 {
		t.Errorf("Expected entity 1 admin points 100, got %d", entity1Admin.Points)
	}

	entity2Admin, err := manager.GetByUUID("Admin Team", testUUID2)
	if err != nil {
		t.Fatalf("Failed to get admin team for entity 2: %v", err)
	}
	if entity2Admin.Logo != "admin2.png" {
		t.Errorf("Expected entity 2 admin logo, got '%s'", entity2Admin.Logo)
	}
	if entity2Admin.Points != 200 {
		t.Errorf("Expected entity 2 admin points 200, got %d", entity2Admin.Points)
	}

	// Ensure they're different teams
	if entity1Admin.ID == entity2Admin.ID {
		t.Error("Entity 1 and Entity 2 admin teams should have different IDs")
	}
}

// TestTeamWithMembership tests team and membership relationship
func TestTeamWithMembership(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create a team
	team := PlatformTeam{
		Name:   "Membership Team",
		Logo:   "member-logo.png",
		UUID:   testUUID1,
		Active: true,
	}

	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Get the created team to get its ID
	createdTeam, err := manager.Get("Membership Team", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}

	// Create memberships
	membership1 := TeamMembership{
		TeamID:     createdTeam.ID,
		UserID:     1,
		UUID:       testUUID1,
		AssignedBy: 1,
	}

	membership2 := TeamMembership{
		TeamID:     createdTeam.ID,
		UserID:     2,
		UUID:       testUUID1,
		AssignedBy: 1,
	}

	if err := db.Create(&membership1).Error; err != nil {
		t.Fatalf("Failed to create membership1: %v", err)
	}

	if err := db.Create(&membership2).Error; err != nil {
		t.Fatalf("Failed to create membership2: %v", err)
	}

	// Verify memberships exist
	var count int64
	db.Model(&TeamMembership{}).Where("team_id = ?", createdTeam.ID).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 memberships, got %d", count)
	}
}

// TestTeamWithScores tests team and score relationship
func TestTeamWithScores(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create a team
	team := PlatformTeam{
		Name:   "Score Team",
		Logo:   "score-logo.png",
		Points: 0,
		UUID:   testUUID1,
		Active: true,
	}

	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Get the created team to get its ID
	createdTeam, err := manager.Get("Score Team", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}

	// Create scores
	scores := []TeamScore{
		{TeamID: createdTeam.ID, Points: 10, UUID: testUUID1, ScoredBy: 1},
		{TeamID: createdTeam.ID, Points: 20, UUID: testUUID1, ScoredBy: 2},
		{TeamID: createdTeam.ID, Points: 15, UUID: testUUID1, ScoredBy: 3},
	}

	for _, score := range scores {
		if err := db.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create score: %v", err)
		}
	}

	// Verify scores exist
	var count int64
	db.Model(&TeamScore{}).Where("team_id = ?", createdTeam.ID).Count(&count)
	if count != 3 {
		t.Errorf("Expected 3 scores, got %d", count)
	}

	// Calculate total points
	var totalPoints int
	db.Model(&TeamScore{}).Where("team_id = ?", createdTeam.ID).Select("SUM(points)").Scan(&totalPoints)
	if totalPoints != 45 {
		t.Errorf("Expected total points 45, got %d", totalPoints)
	}
}

// TestTeamPointsTracking tests tracking team points over time
func TestTeamPointsTracking(t *testing.T) {
	db := setupTestDB(t)
	manager, err := CreateTeams(db)
	if err != nil {
		t.Fatalf("Failed to create TeamManager: %v", err)
	}

	team := PlatformTeam{
		Name:   "Points Team",
		Logo:   "points-logo.png",
		Points: 0,
		UUID:   testUUID1,
		Active: true,
	}

	err = manager.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Update points
	var updatedTeam PlatformTeam
	db.Model(&updatedTeam).Where("name = ?", "Points Team").Update("points", 100)

	// Retrieve and verify
	retrievedTeam, err := manager.Get("Points Team", testUUID1)
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}

	if retrievedTeam.Points != 100 {
		t.Errorf("Expected points 100, got %d", retrievedTeam.Points)
	}
}

// BenchmarkCreate benchmarks team creation
func BenchmarkCreate(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager, err := CreateTeams(db)
	if err != nil {
		b.Fatalf("Failed to create TeamManager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		team := PlatformTeam{
			Name:   fmt.Sprintf("Bench Team %d", i),
			Logo:   "bench-logo.png",
			UUID:   testUUID1,
			Active: true,
		}
		_ = manager.Create(team)
	}
}

// BenchmarkExists benchmarks the Exists check
func BenchmarkExists(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager, err := CreateTeams(db)
	if err != nil {
		b.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create a test team
	team := PlatformTeam{
		Name:   "Bench Team",
		Logo:   "bench-logo.png",
		UUID:   testUUID1,
		Active: true,
	}
	_ = manager.Create(team)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Exists("Bench Team", testUUID1)
	}
}

// BenchmarkGet benchmarks the Get operation
func BenchmarkGet(b *testing.B) {
	db := setupTestDB(&testing.T{})
	manager, err := CreateTeams(db)
	if err != nil {
		b.Fatalf("Failed to create TeamManager: %v", err)
	}

	// Create a test team
	team := PlatformTeam{
		Name:   "Bench Team",
		Logo:   "bench-logo.png",
		UUID:   testUUID1,
		Active: true,
	}
	_ = manager.Create(team)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Get("Bench Team", testUUID1)
	}
}

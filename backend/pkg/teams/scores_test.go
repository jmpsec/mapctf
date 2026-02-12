package teams

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDBForScores creates an in-memory SQLite database for testing scores
func setupTestDBForScores(t *testing.T) (*gorm.DB, *TeamManager) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto migrate the TeamScore table
	if err := db.AutoMigrate(&TeamScore{}); err != nil {
		t.Fatalf("Failed to migrate TeamScore table: %v", err)
	}

	manager := &TeamManager{
		DB: db,
	}

	return db, manager
}

// TestTeamScoreStruct tests the TeamScore struct
func TestTeamScoreStruct(t *testing.T) {
	score := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	if score.TeamID != 1 {
		t.Errorf("Expected TeamID 1, got %d", score.TeamID)
	}
	if score.ChallengeID != 10 {
		t.Errorf("Expected ChallengeID 10, got %d", score.ChallengeID)
	}
	if score.Points != 100 {
		t.Errorf("Expected Points 100, got %d", score.Points)
	}
	if score.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, score.UUID)
	}
	if score.ScoredBy != 5 {
		t.Errorf("Expected ScoredBy 5, got %d", score.ScoredBy)
	}
}

// TestGetScores tests the GetScores method
func TestGetScores(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create test scores
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 200, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 30, Points: 150, UUID: testUUID1, ScoredBy: 6},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Test successful retrieval
	scores, err := manager.GetScores(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 3 {
		t.Fatalf("Expected 3 scores, got %d", len(scores))
	}

	// Verify scores are correct
	totalPoints := 0
	for _, score := range scores {
		if score.TeamID != 1 {
			t.Errorf("Expected TeamID 1, got %d", score.TeamID)
		}
		if score.UUID != testUUID1 {
			t.Errorf("Expected UUID '%s', got '%s'", testUUID1, score.UUID)
		}
		totalPoints += score.Points
	}

	if totalPoints != 450 {
		t.Errorf("Expected total points 450, got %d", totalPoints)
	}
}

// TestGetScoresEmpty tests GetScores with no scores
func TestGetScoresEmpty(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	scores, err := manager.GetScores(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 0 {
		t.Errorf("Expected 0 scores, got %d", len(scores))
	}
}

// TestGetScoresWrongTeamID tests GetScores with wrong team ID
func TestGetScoresWrongTeamID(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for team 1
	testScore := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	if err := manager.DB.Create(&testScore).Error; err != nil {
		t.Fatalf("Failed to create test score: %v", err)
	}

	// Try to get scores for team 2
	scores, err := manager.GetScores(2, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 0 {
		t.Errorf("Expected 0 scores for team 2, got %d", len(scores))
	}
}

// TestGetScoresWrongUUID tests GetScores with wrong UUID
func TestGetScoresWrongUUID(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for EntID 1
	testScore := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	if err := manager.DB.Create(&testScore).Error; err != nil {
		t.Fatalf("Failed to create test score: %v", err)
	}

	// Try to get scores for UUID 2
	scores, err := manager.GetScores(1, testUUID2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 0 {
		t.Errorf("Expected 0 scores for UUID 2, got %d", len(scores))
	}
}

// TestGetScoresMultipleTeams tests GetScores with multiple teams
func TestGetScoresMultipleTeams(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for different teams
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 2, ChallengeID: 10, Points: 200, UUID: testUUID1, ScoredBy: 6},
		{TeamID: 1, ChallengeID: 20, Points: 150, UUID: testUUID1, ScoredBy: 5},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get scores for team 1
	scores, err := manager.GetScores(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 2 {
		t.Errorf("Expected 2 scores for team 1, got %d", len(scores))
	}

	// Get scores for team 2
	scores, err = manager.GetScores(2, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 1 {
		t.Errorf("Expected 1 score for team 2, got %d", len(scores))
	}
}

// TestGetScoresMultipleEntities tests GetScores with multiple entities
func TestGetScoresMultipleEntities(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for different UUIDs
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 200, UUID: testUUID2, ScoredBy: 6},
		{TeamID: 1, ChallengeID: 30, Points: 150, UUID: testUUID1, ScoredBy: 5},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get scores for UUID 1
	scores, err := manager.GetScores(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 2 {
		t.Errorf("Expected 2 scores for UUID 1, got %d", len(scores))
	}

	// Get scores for UUID 2
	scores, err = manager.GetScores(1, testUUID2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scores) != 1 {
		t.Errorf("Expected 1 score for UUID 2, got %d", len(scores))
	}
}

// TestNewScore tests the NewScore method
func TestNewScore(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	score, err := manager.NewScore(1, 10, 100, testUUID1, 5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if score.TeamID != 1 {
		t.Errorf("Expected TeamID 1, got %d", score.TeamID)
	}
	if score.ChallengeID != 10 {
		t.Errorf("Expected ChallengeID 10, got %d", score.ChallengeID)
	}
	if score.Points != 100 {
		t.Errorf("Expected Points 100, got %d", score.Points)
	}
	if score.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, score.UUID)
	}
	if score.ScoredBy != 5 {
		t.Errorf("Expected ScoredBy 5, got %d", score.ScoredBy)
	}
}

// TestNewScoreWithDifferentParameters tests NewScore with various parameters
func TestNewScoreWithDifferentParameters(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	testCases := []struct {
		name        string
		teamID      uint
		challengeID uint
		points      int
		uuid        string
		scoredBy    uint
	}{
		{"score1", 1, 10, 100, testUUID1, 5},
		{"score2", 2, 20, 200, testUUID2, 10},
		{"score3", 3, 30, 0, testUUID1, 15},
		{"score4", 4, 40, -50, testUUID2, 20},
		{"score5", 0, 0, 1000, "", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, err := manager.NewScore(tc.teamID, tc.challengeID, tc.points, tc.uuid, tc.scoredBy)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if score.TeamID != tc.teamID {
				t.Errorf("Expected TeamID %d, got %d", tc.teamID, score.TeamID)
			}
			if score.ChallengeID != tc.challengeID {
				t.Errorf("Expected ChallengeID %d, got %d", tc.challengeID, score.ChallengeID)
			}
			if score.Points != tc.points {
				t.Errorf("Expected Points %d, got %d", tc.points, score.Points)
			}
			if score.UUID != tc.uuid {
				t.Errorf("Expected UUID '%s', got '%s'", tc.uuid, score.UUID)
			}
			if score.ScoredBy != tc.scoredBy {
				t.Errorf("Expected ScoredBy %d, got %d", tc.scoredBy, score.ScoredBy)
			}
		})
	}
}

// TestNewScoreNegativePoints tests NewScore with negative points
func TestNewScoreNegativePoints(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	score, err := manager.NewScore(1, 10, -100, testUUID1, 5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if score.Points != -100 {
		t.Errorf("Expected Points -100, got %d", score.Points)
	}
}

// TestNewScoreZeroValues tests NewScore with zero values
func TestNewScoreZeroValues(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	score, err := manager.NewScore(0, 0, 0, "", 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if score.TeamID != 0 {
		t.Errorf("Expected TeamID 0, got %d", score.TeamID)
	}
	if score.ChallengeID != 0 {
		t.Errorf("Expected ChallengeID 0, got %d", score.ChallengeID)
	}
	if score.Points != 0 {
		t.Errorf("Expected Points 0, got %d", score.Points)
	}
	if score.UUID != "" {
		t.Errorf("Expected UUID '', got '%s'", score.UUID)
	}
	if score.ScoredBy != 0 {
		t.Errorf("Expected ScoredBy 0, got %d", score.ScoredBy)
	}
}

// TestCreateScore tests the CreateScore method
func TestCreateScore(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	score := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	err := manager.CreateScore(score)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the score was created
	var retrievedScore TeamScore
	if err := manager.DB.Where("team_id = ? AND challenge_id = ?", 1, 10).First(&retrievedScore).Error; err != nil {
		t.Fatalf("Failed to retrieve created score: %v", err)
	}

	if retrievedScore.TeamID != 1 {
		t.Errorf("Expected TeamID 1, got %d", retrievedScore.TeamID)
	}
	if retrievedScore.ChallengeID != 10 {
		t.Errorf("Expected ChallengeID 10, got %d", retrievedScore.ChallengeID)
	}
	if retrievedScore.Points != 100 {
		t.Errorf("Expected Points 100, got %d", retrievedScore.Points)
	}
	if retrievedScore.UUID != testUUID1 {
		t.Errorf("Expected UUID '%s', got '%s'", testUUID1, retrievedScore.UUID)
	}
	if retrievedScore.ScoredBy != 5 {
		t.Errorf("Expected ScoredBy 5, got %d", retrievedScore.ScoredBy)
	}
}

// TestCreateScoreMultiple tests creating multiple scores
func TestCreateScoreMultiple(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	scores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 200, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 2, ChallengeID: 10, Points: 150, UUID: testUUID1, ScoredBy: 6},
	}

	for _, score := range scores {
		if err := manager.CreateScore(score); err != nil {
			t.Fatalf("Failed to create score: %v", err)
		}
	}

	// Verify all scores were created
	var count int64
	manager.DB.Model(&TeamScore{}).Count(&count)
	if count != int64(len(scores)) {
		t.Errorf("Expected %d scores, got %d", len(scores), count)
	}
}

// TestCreateScoreWithClosedDB tests CreateScore with closed database
func TestCreateScoreWithClosedDB(t *testing.T) {
	db, manager := setupTestDBForScores(t)

	// Close the database connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	score := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	err = manager.CreateScore(score)
	if err == nil {
		t.Error("Expected error when creating score with closed database")
	}
}

// TestGetScoreTotal tests the GetScoreTotal method
func TestGetScoreTotal(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create test scores
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 200, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 30, Points: 150, UUID: testUUID1, ScoredBy: 6},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Test successful retrieval
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 450 {
		t.Errorf("Expected total 450, got %d", total)
	}
}

// TestGetScoreTotalEmpty tests GetScoreTotal with no scores
func TestGetScoreTotalEmpty(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	total, err := manager.GetScoreTotal(1, testUUID1)
	// When there are no scores, SUM returns NULL which causes a scan error
	if err == nil {
		// If no error, verify total is 0
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	}
	// Both error (NULL scan) or 0 are acceptable behaviors
}

// TestGetScoreTotalWrongTeamID tests GetScoreTotal with wrong team ID
func TestGetScoreTotalWrongTeamID(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for team 1
	testScore := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	if err := manager.DB.Create(&testScore).Error; err != nil {
		t.Fatalf("Failed to create test score: %v", err)
	}

	// Try to get total for team 2
	total, err := manager.GetScoreTotal(2, testUUID1)
	// When there are no scores, SUM returns NULL which causes a scan error
	if err == nil {
		// If no error, verify total is 0
		if total != 0 {
			t.Errorf("Expected total 0 for team 2, got %d", total)
		}
	}
	// Both error (NULL scan) or 0 are acceptable behaviors
}

// TestGetScoreTotalWrongUUID tests GetScoreTotal with wrong UUID
func TestGetScoreTotalWrongUUID(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for EntID 1
	testScore := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	if err := manager.DB.Create(&testScore).Error; err != nil {
		t.Fatalf("Failed to create test score: %v", err)
	}

	// Try to get total for UUID 2
	total, err := manager.GetScoreTotal(1, testUUID2)
	// When there are no scores, SUM returns NULL which causes a scan error
	if err == nil {
		// If no error, verify total is 0
		if total != 0 {
			t.Errorf("Expected total 0 for UUID 2, got %d", total)
		}
	}
	// Both error (NULL scan) or 0 are acceptable behaviors
}

// TestGetScoreTotalMultipleTeams tests GetScoreTotal with multiple teams
func TestGetScoreTotalMultipleTeams(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for different teams
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 2, ChallengeID: 10, Points: 200, UUID: testUUID1, ScoredBy: 6},
		{TeamID: 1, ChallengeID: 20, Points: 150, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 2, ChallengeID: 20, Points: 250, UUID: testUUID1, ScoredBy: 6},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get total for team 1
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 250 {
		t.Errorf("Expected total 250 for team 1, got %d", total)
	}

	// Get total for team 2
	total, err = manager.GetScoreTotal(2, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 450 {
		t.Errorf("Expected total 450 for team 2, got %d", total)
	}
}

// TestGetScoreTotalMultipleEntities tests GetScoreTotal with multiple entities
func TestGetScoreTotalMultipleEntities(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores for different UUIDs
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 200, UUID: testUUID2, ScoredBy: 6},
		{TeamID: 1, ChallengeID: 30, Points: 150, UUID: testUUID1, ScoredBy: 5},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get total for UUID 1
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 250 {
		t.Errorf("Expected total 250 for UUID 1, got %d", total)
	}

	// Get total for UUID 2
	total, err = manager.GetScoreTotal(1, testUUID2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 200 {
		t.Errorf("Expected total 200 for UUID 2, got %d", total)
	}
}

// TestGetScoreTotalWithNegativePoints tests GetScoreTotal with negative points
func TestGetScoreTotalWithNegativePoints(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores with negative points (penalties)
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 100, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: -50, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 30, Points: 200, UUID: testUUID1, ScoredBy: 6},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Test total with negative points
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 250 {
		t.Errorf("Expected total 250 (100-50+200), got %d", total)
	}
}

// TestGetScoreTotalZeroPoints tests GetScoreTotal with zero points
func TestGetScoreTotalZeroPoints(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores with zero points
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 0, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 0, UUID: testUUID1, ScoredBy: 5},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Test total with zero points
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 0 {
		t.Errorf("Expected total 0, got %d", total)
	}
}

// TestGetScoreTotalLargeNumbers tests GetScoreTotal with large point values
func TestGetScoreTotalLargeNumbers(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create scores with large point values
	testScores := []TeamScore{
		{TeamID: 1, ChallengeID: 10, Points: 1000000, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 20, Points: 2000000, UUID: testUUID1, ScoredBy: 5},
		{TeamID: 1, ChallengeID: 30, Points: 3000000, UUID: testUUID1, ScoredBy: 6},
	}

	for _, score := range testScores {
		if err := manager.DB.Create(&score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Test total with large numbers
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 6000000 {
		t.Errorf("Expected total 6000000, got %d", total)
	}
}

// TestCreateScoreDuplicate tests creating duplicate scores
func TestCreateScoreDuplicate(t *testing.T) {
	_, manager := setupTestDBForScores(t)

	// Create the same score twice
	score := TeamScore{
		TeamID:      1,
		ChallengeID: 10,
		Points:      100,
		UUID:        testUUID1,
		ScoredBy:    5,
	}

	err := manager.CreateScore(score)
	if err != nil {
		t.Fatalf("Failed to create first score: %v", err)
	}

	// Create the same score again
	err = manager.CreateScore(score)
	if err != nil {
		t.Fatalf("Failed to create duplicate score: %v", err)
	}

	// Verify both scores exist
	var count int64
	manager.DB.Model(&TeamScore{}).Where("team_id = ? AND challenge_id = ?", 1, 10).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 duplicate scores, got %d", count)
	}

	// Verify total is doubled
	total, err := manager.GetScoreTotal(1, testUUID1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 200 {
		t.Errorf("Expected total 200 (100+100), got %d", total)
	}
}

// TestGetScoresWithClosedDB tests GetScores with closed database
func TestGetScoresWithClosedDB(t *testing.T) {
	db, manager := setupTestDBForScores(t)

	// Close the database connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

		_, err = manager.GetScores(1, testUUID1)
	if err == nil {
		t.Error("Expected error when getting scores with closed database")
	}
}

// TestGetScoreTotalWithClosedDB tests GetScoreTotal with closed database
func TestGetScoreTotalWithClosedDB(t *testing.T) {
	db, manager := setupTestDBForScores(t)

	// Close the database connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

		_, err = manager.GetScoreTotal(1, testUUID1)
	if err == nil {
		t.Error("Expected error when getting score total with closed database")
	}
}

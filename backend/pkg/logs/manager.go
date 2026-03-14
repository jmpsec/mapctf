package logs

import (
	"fmt"

	"gorm.io/gorm"
)

// LogManager manage all the logs of the system
type LogManager struct {
	DB *gorm.DB
}

// CreateLogManager to initialize the logs struct and tables
func CreateLogManager(backend *gorm.DB) (*LogManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	l := &LogManager{DB: backend}
	// table activity_logs
	if err := backend.AutoMigrate(&ActivityLog{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (activity_logs): %w", err)
	}
	// table announcements
	if err := backend.AutoMigrate(&Announcement{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (announcements): %w", err)
	}
	// table scoreboard_logs
	if err := backend.AutoMigrate(&ScoreboardLog{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (scoreboard_logs): %w", err)
	}
	// table hints_logs
	if err := backend.AutoMigrate(&HintsLog{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (hints_logs): %w", err)
	}
	// table failures_logs
	if err := backend.AutoMigrate(&FailuresLog{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (failures_logs): %w", err)
	}
	// table registration_logs
	if err := backend.AutoMigrate(&RegistrationLog{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (registration_logs): %w", err)
	}
	return l, nil
}

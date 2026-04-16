package chat

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	// NoTeamID is the default team ID when no team is assigned
	NoTeamID uint = 0
	// DefaultMaxLen is the default maximum length of a chat entry
	DefaultMaxLen int = 200
)

// ChatEntry to hold all chat entries
type ChatEntry struct {
	gorm.Model
	Username string `gorm:"index"`
	Body     string
	TeamID   uint
	UUID     string `gorm:"index"`
}

// ChatManager have all chat entries of the system
type ChatManager struct {
	DB     *gorm.DB
	UUID   string
	MaxLen int
}

// CreateChatManager to initialize the chat manager struct and tables
func CreateChatManager(backend *gorm.DB, uuid string, maxLen int) (*ChatManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	if maxLen <= 0 {
		maxLen = DefaultMaxLen
	}
	c := &ChatManager{DB: backend, UUID: uuid, MaxLen: maxLen}
	// table chat_entries
	if err := backend.AutoMigrate(&ChatEntry{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (chat_entries): %w", err)
	}
	return c, nil
}

// Create new chat entry
func (m *ChatManager) Create(entry ChatEntry) error {
	if err := m.DB.Create(&entry).Error; err != nil {
		return fmt.Errorf("Create ChatEntry %w", err)
	}
	return nil
}

// GetAll users by UUID
func (m *ChatManager) GetAll() ([]ChatEntry, error) {
	var entries []ChatEntry
	if err := m.DB.Where("uuid = ?", m.UUID).Find(&entries).Error; err != nil {
		return entries, err
	}
	return entries, nil
}

// New chat entry with all the required fields and UUID
func (m *ChatManager) New(username, body string, uuid string, teamID uint) (ChatEntry, error) {
	if username == "" || body == "" {
		return ChatEntry{}, fmt.Errorf("username and body cannot be empty")
	}
	// Truncate body if it exceeds MaxLen
	if len(body) > m.MaxLen {
		body = body[:m.MaxLen]
	}
	return ChatEntry{
		Username: username,
		Body:     body,
		UUID:     uuid,
		TeamID:   teamID,
	}, nil
}

// Delete chat entry by ID
func (m *ChatManager) Delete(id uint) error {
	if err := m.DB.Delete(&ChatEntry{}, id).Error; err != nil {
		return fmt.Errorf("Delete ChatEntry %w", err)
	}
	return nil
}

// DeleteAll chat entries by UUID
func (m *ChatManager) DeleteAll() error {
	if err := m.DB.Where("uuid = ?", m.UUID).Delete(&ChatEntry{}).Error; err != nil {
		return fmt.Errorf("DeleteAll ChatEntries %w", err)
	}
	return nil
}

// DeleteAllByTeamID chat entries by team ID and UUID
func (m *ChatManager) DeleteAllByTeamID(teamID uint) error {
	if err := m.DB.Where("team_id = ? AND uuid = ?", teamID, m.UUID).Delete(&ChatEntry{}).Error; err != nil {
		return fmt.Errorf("DeleteAllByTeamID ChatEntries %w", err)
	}
	return nil
}

// DeleteAllByUsername chat entries by username and UUID
func (m *ChatManager) DeleteAllByUsername(username string) error {
	if err := m.DB.Where("username = ? AND uuid = ?", username, m.UUID).Delete(&ChatEntry{}).Error; err != nil {
		return fmt.Errorf("DeleteAllByUsername ChatEntries %w", err)
	}
	return nil
}

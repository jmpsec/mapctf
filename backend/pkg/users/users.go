package users

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// NoTeamID is the default team ID when no team is assigned
	NoTeamID uint = 0
	// NoEntID is the default entity ID when no entity is assigned
	NoEntID uint = 0
)

// User to hold all platform users
type PlatformUser struct {
	gorm.Model
	Username      string `gorm:"index"`
	Display       string
	Email         string
	TeamID        uint
	PassHash      string `json:"-"`
	APIToken      string `json:"-"`
	TokenExpire   time.Time
	Admin         bool
	Service       bool
	Active        bool
	LastIPAddress string
	LastUserAgent string
	LastAccess    time.Time
	LastTokenUse  time.Time
	EntID         uint
}

// UserManager have all users of the system
type UserManager struct {
	DB *gorm.DB
}

// CreateUserManager to initialize the users struct and tables
func CreateUserManager(backend *gorm.DB) *UserManager {
	u := &UserManager{DB: backend}
	// table platform_users
	if err := backend.AutoMigrate(&PlatformUser{}); err != nil {
		log.Fatal().Msgf("Failed to AutoMigrate table (platform_users): %v", err)
	}
	return u
}

// Create new user
func (m *UserManager) Create(user PlatformUser) error {
	if err := m.DB.Create(&user).Error; err != nil {
		return fmt.Errorf("Create PlatformUser %w", err)
	}
	return nil
}

// HashTextWithSalt to hash text before store it
func (m *UserManager) HashTextWithSalt(text string) (string, error) {
	saltedBytes := []byte(text)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hash := string(hashedBytes)
	return hash, nil
}

// HashPasswordWithSalt to hash a password before store it
func (m *UserManager) HashPasswordWithSalt(password string) (string, error) {
	return m.HashTextWithSalt(password)
}

// Exists checks if user exists
func (m *UserManager) Exists(username string) bool {
	var results int64
	m.DB.Model(&PlatformUser{}).Where("username = ?", username).Count(&results)
	return (results > 0)
}

// Get user by username including service users
func (m *UserManager) Get(username string) (PlatformUser, error) {
	var user PlatformUser
	if err := m.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// Get user by username and by tenant ID, including service users
func (m *UserManager) GetByTenantID(username string, tenantID uint) (PlatformUser, error) {
	var user PlatformUser
	if err := m.DB.Where("username = ? AND ent_id = ?", username, tenantID).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// ExistsGet checks if user exists and returns the user
func (m *UserManager) ExistsGet(username string) (bool, PlatformUser) {
	user, err := m.Get(username)
	if err != nil {
		return false, PlatformUser{}
	}
	return true, user
}

// ExistsGetByTenantID checks if user exists and returns the user
func (m *UserManager) ExistsGetByTenantID(username string, tenantID uint) (bool, PlatformUser) {
	user, err := m.GetByTenantID(username, tenantID)
	if err != nil {
		return false, PlatformUser{}
	}
	return true, user
}

// New empty user
func (m *UserManager) New(username, password, email, display string, admin, service bool, eID, teamID uint) (PlatformUser, error) {
	if !m.Exists(username) {
		passhash, err := m.HashPasswordWithSalt(password)
		if err != nil {
			return PlatformUser{}, err
		}
		return PlatformUser{
			Username: username,
			PassHash: passhash,
			Admin:    admin,
			Service:  service,
			Email:    email,
			Display:  display,
			TeamID:   teamID,
			Active:   true,
			EntID:    eID,
		}, nil
	}
	return PlatformUser{}, fmt.Errorf("%s already exists", username)
}

// Update

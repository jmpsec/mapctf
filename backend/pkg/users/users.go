package users

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jmpsec/mapctf/pkg/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// NoTeamID is the default team ID when no team is assigned
	NoTeamID uint = 0
	// NoUUID is the default UUID when no entity is specified
	NoUUID string = ""
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
	UUID          string `gorm:"index"`
}

// TokenClaims to hold user claims when using JWT
type TokenClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// UserManager have all users of the system
type UserManager struct {
	DB        *gorm.DB
	JWTConfig *config.ConfigurationJWT
}

// CreateUserManager to initialize the users struct and tables
func CreateUserManager(backend *gorm.DB, jwtConfig *config.ConfigurationJWT) (*UserManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	u := &UserManager{DB: backend, JWTConfig: jwtConfig}
	// table platform_users
	if err := backend.AutoMigrate(&PlatformUser{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (platform_users): %w", err)
	}
	return u, nil
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
func (m *UserManager) Exists(username string, uuid string) bool {
	var results int64
	m.DB.Model(&PlatformUser{}).Where("username = ? AND uuid = ?", username, uuid).Count(&results)
	return (results > 0)
}

// Get user by username including service users
func (m *UserManager) Get(username string, uuid string) (PlatformUser, error) {
	var user PlatformUser
	if err := m.DB.Where("username = ? AND uuid = ?", username, uuid).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// ExistsGet checks if user exists and returns the user
func (m *UserManager) ExistsGet(username string, uuid string) (bool, PlatformUser) {
	user, err := m.Get(username, uuid)
	if err != nil {
		return false, PlatformUser{}
	}
	return true, user
}

// GetByUUID gets user by username and UUID (alias for Get)
func (m *UserManager) GetByUUID(username string, uuid string) (PlatformUser, error) {
	return m.Get(username, uuid)
}

// ExistsGetByUUID checks if user exists and returns the user
func (m *UserManager) ExistsGetByUUID(username string, uuid string) (bool, PlatformUser) {
	user, err := m.Get(username, uuid)
	if err != nil {
		return false, PlatformUser{}
	}
	return true, user
}

// New empty user
func (m *UserManager) New(username, password, email, display string, admin, service bool, uuid string, teamID uint) (PlatformUser, error) {
	if !m.Exists(username, uuid) {
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
			UUID:     uuid,
		}, nil
	}
	return PlatformUser{}, fmt.Errorf("%s already exists", username)
}

// Update

// CheckLoginCredentials to check provided login credentials by matching hashes
func (m *UserManager) CheckLoginCredentials(username, password string, uuid string) (bool, PlatformUser) {
	user, err := m.Get(username, uuid)
	if err != nil {
		return false, PlatformUser{}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		return false, PlatformUser{}
	}
	return true, user
}

// CreateToken to create a new JWT token for a given user
func (m *UserManager) CreateToken(username, issuer string, expHours int) (string, time.Time, error) {
	tDuration := time.Duration(expHours)
	if expHours == 0 {
		tDuration = time.Duration(m.JWTConfig.HoursToExpire)
	}
	expirationTime := time.Now().Add(time.Hour * tDuration)
	// Create the JWT claims, which includes the username, level and expiry time
	claims := &TokenClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    issuer,
		},
	}
	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString([]byte(m.JWTConfig.Secret))
	if err != nil {
		return "", time.Now(), err
	}
	return tokenString, expirationTime, nil
}

// CheckToken to verify if a token used is valid
func (m *UserManager) CheckToken(jwtSecret, tokenStr string) (TokenClaims, error) {
	claims := &TokenClaims{}
	tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return *claims, fmt.Errorf("error parsing token claims: %w", err)
	}
	if !tkn.Valid {
		return *claims, fmt.Errorf("invalid token")
	}
	return *claims, nil
}

// SetPassword to set a new password for a given user by username and UUID
func (m *UserManager) SetPassword(username, password string, uuid string) error {
	passHash, err := m.HashPasswordWithSalt(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	if err := m.DB.Model(&PlatformUser{}).
		Where("username = ? AND uuid = ?", username, uuid).
		Update("pass_hash", passHash).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// SetAdmin to set the admin flag for a given user by username and UUID
func (m *UserManager) SetAdmin(admin bool, username string, uuid string) error {
	if err := m.DB.Model(&PlatformUser{}).
		Where("username = ? AND uuid = ?", username, uuid).
		Update("admin", admin).Error; err != nil {
		return fmt.Errorf("failed to update admin flag: %w", err)
	}
	return nil
}

// SetActive to set the active flag for a given user by username and UUID
func (m *UserManager) SetActive(active bool, username string, uuid string) error {
	if err := m.DB.Model(&PlatformUser{}).
		Where("username = ? AND uuid = ?", username, uuid).
		Update("active", active).Error; err != nil {
		return fmt.Errorf("failed to update active flag: %w", err)
	}
	return nil
}

// SetService to set the service flag for a given user by username and UUID
func (m *UserManager) SetService(service bool, username string, uuid string) error {
	if err := m.DB.Model(&PlatformUser{}).
		Where("username = ? AND uuid = ?", username, uuid).
		Update("service", service).Error; err != nil {
		return fmt.Errorf("failed to update service flag: %w", err)
	}
	return nil
}

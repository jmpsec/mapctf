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

// Get user by username and by entity ID, including service users
func (m *UserManager) GetByEntID(username string, entID uint) (PlatformUser, error) {
	var user PlatformUser
	if err := m.DB.Where("username = ? AND ent_id = ?", username, entID).First(&user).Error; err != nil {
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

// ExistsGetByEntID checks if user exists and returns the user
func (m *UserManager) ExistsGetByEntID(username string, entID uint) (bool, PlatformUser) {
	user, err := m.GetByEntID(username, entID)
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

// CheckLoginCredentials to check provided login credentials by matching hashes
func (m *UserManager) CheckLoginCredentials(username, password string) (bool, PlatformUser) {
	user, err := m.Get(username)
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

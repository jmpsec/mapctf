package settings

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	// LoginEnabled is the setting name for login enabled/disabled
	LoginEnabled string = "login_enabled"
	// LoginStrongPasswords is the setting name for login strong password requirement enabled/disabled
	LoginStrongPasswords string = "login_strong_passwords"
	// RegistrationEnabled is the setting name for registration enabled/disabled
	RegistrationEnabled string = "registration_enabled"
	// RegistrationNames is the setting name for registration name field enabled/disabled
	RegistrationNames string = "registration_names"
	// RegistrationEmails is the setting name for registration email field enabled/disabled
	RegistrationEmails string = "registration_emails"
	// RegistrationType is the setting name for registration type (open, token)
	RegistrationType string = "registration_type"
	// RegistrationToken is the setting name for the registration token when registration type is token-based
	RegistrationToken string = "registration_token"
	// ScoringEnabled is the setting name for scoring enabled/disabled
	ScoringEnabled string = "scoring_enabled"
	// GamePaused is the setting name for game paused/unpaused
	GamePaused string = "game_paused"
	// GameStarted is the setting name for game started/not started
	GameStarted string = "game_started"
	// GameStartTime is the setting name for game start time
	GameStartTime string = "game_start_time"
	// GameEndTime is the setting name for game end time
	GameEndTime string = "game_end_time"
	// CustomOrg is the setting name for the custom branding organization
	CustomOrg string = "custom_org"
	// CustomLogo is the setting name for the custom branding logo
	CustomLogo string = "custom_logo"
	// Language is the setting name for platform language
	Language string = "language"
	// LeaderboardLimit is the setting name for the number of teams to show on the leaderboard
	LeaderboardLimit string = "leaderboard_limit"
	// GameboardShowTeamMembers is the setting name for showing team members on the gameboard
	GameboardShowTeamMembers string = "gameboard_show_team_members"
	// GameboardChatMaxLen is the setting name for the maximum length of chat messages on the gameboard
	GameboardChatMaxLen string = "gameboard_chat_max_len"
)

const (
	// OpenRegistration is the registration type for open registration
	OpenRegistration int = 0
	// TokenRegistration is the registration type for token-based registration
	TokenRegistration int = 1
)

// BooleanSettings to be used as check for valid setting and to keep default value, if needed
var BooleanSettings = map[string]bool{
	LoginEnabled:             false,
	LoginStrongPasswords:     false,
	RegistrationEnabled:      false,
	RegistrationNames:        false,
	RegistrationEmails:       false,
	ScoringEnabled:           false,
	GamePaused:               false,
	GameStarted:              false,
	GameboardShowTeamMembers: false,
}

// StringSettings to be used as check for valid setting and to keep default value
var StringSettings = map[string]string{
	CustomOrg:         "",
	CustomLogo:        "",
	Language:          "en",
	RegistrationToken: "",
}

// DateSettings to be used as check for valid setting and to keep default value
var DateSettings = map[string]time.Time{
	GameStartTime: time.Time{},
	GameEndTime:   time.Time{},
}

// IntSettings to be used as check for valid setting and to keep default value
var IntSettings = map[string]int{
	RegistrationType:    OpenRegistration,
	LeaderboardLimit:    10,
	GameboardChatMaxLen: 200,
}

const (
	// TypeString is the string type for settings
	TypeString string = "string"
	// TypeInt is the integer type for settings
	TypeInt string = "int"
	// TypeBool is the boolean type for settings
	TypeBool string = "bool"
	// TypeFloat is the float type for settings
	TypeFloat string = "float"
	// TypeDate is the date type for settings
	TypeDate string = "date"
	// EventCreate is the event type for creating a setting
	EventCreate string = "create"
	// EventUpdate is the event type for updating a setting
	EventUpdate string = "update"
	// EventDelete is the event type for deleting a setting
	EventDelete string = "delete"
)

// PlatformSetting to hold all platform settings
type PlatformSetting struct {
	gorm.Model
	Name        string `gorm:"index"`
	ValueString string
	ValueType   string
	ValueInt    int
	ValueBool   bool
	ValueFloat  float64
	ValueDate   time.Time
	Description string
	UUID        string `gorm:"index"`
}

// SettingLog to hold all setting changes for auditing
type SettingLog struct {
	gorm.Model
	SettingID uint
	Event     string
	ChangedBy string
	UUID      string `gorm:"index"`
}

// SettingsManager have all settings of the system
type SettingsManager struct {
	DB      *gorm.DB
	UUID    string
	Service string
}

// CreateSettingsManager to initialize the settings struct and tables
func CreateSettingsManager(backend *gorm.DB, service, uuid string) (*SettingsManager, error) {
	if backend == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	s := &SettingsManager{DB: backend, UUID: uuid, Service: service}
	// table platform_settings
	if err := backend.AutoMigrate(&PlatformSetting{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (platform_settings): %w", err)
	}
	// table setting_logs
	if err := backend.AutoMigrate(&SettingLog{}); err != nil {
		return nil, fmt.Errorf("failed to AutoMigrate table (setting_logs): %w", err)
	}
	return s, nil
}

// Initialization with default settings if they don't exist
func (m *SettingsManager) Initialization() error {
	allSettings, err := m.GetAll(m.UUID)
	if err != nil {
		return fmt.Errorf("failed to get all settings: %w", err)
	}
	// Convert slice to map for easier lookup
	existingSettings := make(map[string]PlatformSetting)
	for _, setting := range allSettings {
		existingSettings[setting.Name] = setting
	}
	// Create default boolean settings if they don't exist
	for name, defaultValue := range BooleanSettings {
		if _, exists := existingSettings[name]; !exists {
			newSetting := PlatformSetting{
				Name:        name,
				ValueType:   TypeBool,
				ValueBool:   defaultValue,
				UUID:        m.UUID,
				Description: name + " boolean setting",
			}
			if err := m.Create(newSetting); err != nil {
				return fmt.Errorf("failed to create default boolean setting %s: %w", name, err)
			}
		}
	}
	// Create default string settings if they don't exist
	for name, defaultValue := range StringSettings {
		if _, exists := existingSettings[name]; !exists {
			newSetting := PlatformSetting{
				Name:        name,
				ValueType:   TypeString,
				ValueString: defaultValue,
				UUID:        m.UUID,
				Description: name + " string setting",
			}
			if err := m.Create(newSetting); err != nil {
				return fmt.Errorf("failed to create default string setting %s: %w", name, err)
			}
		}
	}
	// Create default date settings if they don't exist
	for name, defaultValue := range DateSettings {
		if _, exists := existingSettings[name]; !exists {
			newSetting := PlatformSetting{
				Name:        name,
				ValueType:   TypeDate,
				ValueDate:   defaultValue,
				UUID:        m.UUID,
				Description: name + " date setting",
			}
			if err := m.Create(newSetting); err != nil {
				return fmt.Errorf("failed to create default date setting %s: %w", name, err)
			}
		}
	}
	// Create default int settings if they don't exist
	for name, defaultValue := range IntSettings {
		if _, exists := existingSettings[name]; !exists {
			newSetting := PlatformSetting{
				Name:        name,
				ValueType:   TypeInt,
				ValueInt:    defaultValue,
				UUID:        m.UUID,
				Description: name + " int setting",
			}
			if err := m.Create(newSetting); err != nil {
				return fmt.Errorf("failed to create default int setting %s: %w", name, err)
			}
		}
	}
	return nil
}

// Create new setting
func (m *SettingsManager) Create(setting PlatformSetting) error {
	if err := m.DB.Create(&setting).Error; err != nil {
		return fmt.Errorf("Create PlatformSetting %w", err)
	}
	// Log the creation event
	if err := m.LogEvent(setting.ID, EventCreate, m.Service, setting.UUID); err != nil {
		return fmt.Errorf("LogEvent PlatformSetting %w", err)
	}
	return nil
}

// Exists checks if setting exists
func (m *SettingsManager) Exists(name string, uuid string) bool {
	var results int64
	m.DB.Model(&PlatformSetting{}).Where("name = ? AND uuid = ?", name, uuid).Count(&results)
	return (results > 0)
}

// Get setting by name including service settings
func (m *SettingsManager) Get(name string, uuid string) (PlatformSetting, error) {
	var setting PlatformSetting
	if err := m.DB.Where("name = ? AND uuid = ?", name, uuid).First(&setting).Error; err != nil {
		return setting, err
	}
	return setting, nil
}

// GetAll settings for a given uuid
func (m *SettingsManager) GetAll(uuid string) ([]PlatformSetting, error) {
	var settings []PlatformSetting
	if err := m.DB.Where("uuid = ?", uuid).Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// ExistsGet checks if setting exists and returns the setting
func (m *SettingsManager) ExistsGet(name string, uuid string) (bool, PlatformSetting) {
	setting, err := m.Get(name, uuid)
	if err != nil {
		return false, PlatformSetting{}
	}
	return true, setting
}

// LogEvent logs a setting change for auditing, only log write events (create, update, delete)
func (m *SettingsManager) LogEvent(settingID uint, event string, changedBy string, uuid string) error {
	log := SettingLog{
		SettingID: settingID,
		Event:     event,
		ChangedBy: changedBy,
		UUID:      uuid,
	}
	if err := m.DB.Create(&log).Error; err != nil {
		return fmt.Errorf("Create SettingLog %w", err)
	}
	return nil
}

// New empty setting
func (m *SettingsManager) New(name, valueType, description string, uuid string, value any) (PlatformSetting, error) {
	if !m.Exists(name, uuid) {
		new := PlatformSetting{
			Name:        name,
			ValueType:   valueType,
			Description: description,
			UUID:        uuid,
		}
		// Initialize the correct value field based on the value type
		switch valueType {
		case TypeString:
			new.ValueString = value.(string)
		case TypeInt:
			new.ValueInt = value.(int)
		case TypeBool:
			new.ValueBool = value.(bool)
		case TypeFloat:
			new.ValueFloat = value.(float64)
		case TypeDate:
			new.ValueDate = value.(time.Time)
		default:
			return PlatformSetting{}, fmt.Errorf("invalid value type: %s", valueType)
		}
		return new, nil
	}
	return PlatformSetting{}, fmt.Errorf("%s already exists", name)
}

// Save new setting
func (m *SettingsManager) Save(setting PlatformSetting, username string) error {
	if err := m.DB.Save(&setting).Error; err != nil {
		return fmt.Errorf("Save PlatformSetting %w", err)
	}
	if err := m.LogEvent(setting.ID, EventUpdate, username, setting.UUID); err != nil {
		return fmt.Errorf("LogEvent PlatformSetting %w", err)
	}
	return nil
}

// Change changes the value of a setting and logs the change
func (m *SettingsManager) Change(name, valueType string, uuid string, value any, username string) error {
	setting, err := m.Get(name, uuid)
	if err != nil {
		return fmt.Errorf("failed to get setting: %w", err)
	}
	column, err := valueColumnByType(valueType)
	if err != nil {
		return fmt.Errorf("failed to resolve setting value column: %w", err)
	}
	if err := m.DB.Model(&PlatformSetting{}).
		Where("name = ? AND uuid = ?", name, uuid).
		Update(column, value).Error; err != nil {
		return fmt.Errorf("failed to update setting value: %w", err)
	}
	if err := m.LogEvent(setting.ID, EventUpdate, username, setting.UUID); err != nil {
		return fmt.Errorf("LogEvent PlatformSetting %w", err)
	}
	return nil
}

func valueColumnByType(valueType string) (string, error) {
	switch valueType {
	case TypeString:
		return "value_string", nil
	case TypeInt:
		return "value_int", nil
	case TypeBool:
		return "value_bool", nil
	case TypeFloat:
		return "value_float", nil
	case TypeDate:
		return "value_date", nil
	default:
		return "", fmt.Errorf("invalid value type: %s", valueType)
	}
}

func (m *SettingsManager) upsertSetting(name, valueType, description, username string, setValue func(*PlatformSetting)) error {
	setting, err := m.Get(name, m.UUID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get setting %s: %w", name, err)
		}
		newSetting := PlatformSetting{
			Name:        name,
			ValueType:   valueType,
			Description: description,
			UUID:        m.UUID,
		}
		setValue(&newSetting)
		if err := m.Create(newSetting); err != nil {
			return fmt.Errorf("failed to create setting %s: %w", name, err)
		}
		return nil
	}

	setting.ValueType = valueType
	if setting.Description == "" {
		setting.Description = description
	}
	setValue(&setting)
	if err := m.Save(setting, username); err != nil {
		return fmt.Errorf("failed to save setting %s: %w", name, err)
	}
	return nil
}

func (m *SettingsManager) getBoolSetting(name string) (bool, error) {
	setting, err := m.Get(name, m.UUID)
	if err != nil {
		return false, fmt.Errorf("failed to get setting %s: %w", name, err)
	}
	if setting.ValueType != TypeBool {
		return false, fmt.Errorf("setting %s has unexpected type %s", name, setting.ValueType)
	}
	return setting.ValueBool, nil
}

func (m *SettingsManager) getStringSetting(name string) (string, error) {
	setting, err := m.Get(name, m.UUID)
	if err != nil {
		return "", fmt.Errorf("failed to get setting %s: %w", name, err)
	}
	if setting.ValueType != TypeString {
		return "", fmt.Errorf("setting %s has unexpected type %s", name, setting.ValueType)
	}
	return setting.ValueString, nil
}

func (m *SettingsManager) getIntSetting(name string) (int, error) {
	setting, err := m.Get(name, m.UUID)
	if err != nil {
		return 0, fmt.Errorf("failed to get setting %s: %w", name, err)
	}
	if setting.ValueType != TypeInt {
		return 0, fmt.Errorf("setting %s has unexpected type %s", name, setting.ValueType)
	}
	return setting.ValueInt, nil
}

func (m *SettingsManager) getDateSetting(name string) (time.Time, error) {
	setting, err := m.Get(name, m.UUID)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get setting %s: %w", name, err)
	}
	if setting.ValueType != TypeDate {
		return time.Time{}, fmt.Errorf("setting %s has unexpected type %s", name, setting.ValueType)
	}
	return setting.ValueDate, nil
}

func (m *SettingsManager) SetLoginEnabled(enabled bool, username string) error {
	return m.upsertSetting(LoginEnabled, TypeBool, LoginEnabled+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = enabled
	})
}

func (m *SettingsManager) GetLoginEnabled() (bool, error) {
	return m.getBoolSetting(LoginEnabled)
}

func (m *SettingsManager) SetLoginStrongPasswords(enabled bool, username string) error {
	return m.upsertSetting(LoginStrongPasswords, TypeBool, LoginStrongPasswords+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = enabled
	})
}

func (m *SettingsManager) GetLoginStrongPasswords() (bool, error) {
	return m.getBoolSetting(LoginStrongPasswords)
}

func (m *SettingsManager) SetRegistrationEnabled(enabled bool, username string) error {
	return m.upsertSetting(RegistrationEnabled, TypeBool, RegistrationEnabled+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = enabled
	})
}

func (m *SettingsManager) GetRegistrationEnabled() (bool, error) {
	return m.getBoolSetting(RegistrationEnabled)
}

func (m *SettingsManager) SetRegistrationNames(enabled bool, username string) error {
	return m.upsertSetting(RegistrationNames, TypeBool, RegistrationNames+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = enabled
	})
}

func (m *SettingsManager) GetRegistrationNames() (bool, error) {
	return m.getBoolSetting(RegistrationNames)
}

func (m *SettingsManager) SetRegistrationEmails(enabled bool, username string) error {
	return m.upsertSetting(RegistrationEmails, TypeBool, RegistrationEmails+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = enabled
	})
}

func (m *SettingsManager) GetRegistrationEmails() (bool, error) {
	return m.getBoolSetting(RegistrationEmails)
}

func (m *SettingsManager) SetRegistrationType(regType int, username string) error {
	return m.upsertSetting(RegistrationType, TypeInt, RegistrationType+" int setting", username, func(s *PlatformSetting) {
		s.ValueInt = regType
	})
}

func (m *SettingsManager) GetRegistrationType() (int, error) {
	return m.getIntSetting(RegistrationType)
}

func (m *SettingsManager) SetScoringEnabled(enabled bool, username string) error {
	return m.upsertSetting(ScoringEnabled, TypeBool, ScoringEnabled+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = enabled
	})
}

func (m *SettingsManager) GetScoringEnabled() (bool, error) {
	return m.getBoolSetting(ScoringEnabled)
}

func (m *SettingsManager) SetGamePaused(paused bool, username string) error {
	return m.upsertSetting(GamePaused, TypeBool, GamePaused+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = paused
	})
}

func (m *SettingsManager) GetGamePaused() (bool, error) {
	return m.getBoolSetting(GamePaused)
}

func (m *SettingsManager) SetGameStarted(started bool, username string) error {
	return m.upsertSetting(GameStarted, TypeBool, GameStarted+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = started
	})
}

func (m *SettingsManager) GetGameStarted() (bool, error) {
	return m.getBoolSetting(GameStarted)
}

func (m *SettingsManager) SetGameStartTime(startTime time.Time, username string) error {
	return m.upsertSetting(GameStartTime, TypeDate, GameStartTime+" date setting", username, func(s *PlatformSetting) {
		s.ValueDate = startTime
	})
}

func (m *SettingsManager) GetGameStartTime() (time.Time, error) {
	return m.getDateSetting(GameStartTime)
}

func (m *SettingsManager) SetGameEndTime(endTime time.Time, username string) error {
	return m.upsertSetting(GameEndTime, TypeDate, GameEndTime+" date setting", username, func(s *PlatformSetting) {
		s.ValueDate = endTime
	})
}

func (m *SettingsManager) GetGameEndTime() (time.Time, error) {
	return m.getDateSetting(GameEndTime)
}

func (m *SettingsManager) SetCustomOrg(org string, username string) error {
	return m.upsertSetting(CustomOrg, TypeString, CustomOrg+" string setting", username, func(s *PlatformSetting) {
		s.ValueString = org
	})
}

func (m *SettingsManager) GetCustomOrg() (string, error) {
	return m.getStringSetting(CustomOrg)
}

func (m *SettingsManager) SetCustomLogo(logo, username string) error {
	return m.upsertSetting(CustomLogo, TypeString, CustomLogo+" string setting", username, func(s *PlatformSetting) {
		s.ValueString = logo
	})
}

func (m *SettingsManager) GetCustomLogo() (string, error) {
	return m.getStringSetting(CustomLogo)
}

func (m *SettingsManager) SetLanguage(language, username string) error {
	return m.upsertSetting(Language, TypeString, Language+" string setting", username, func(s *PlatformSetting) {
		s.ValueString = language
	})
}

func (m *SettingsManager) GetLanguage() (string, error) {
	return m.getStringSetting(Language)
}

func (m *SettingsManager) SetLeaderboardLimit(limit int, username string) error {
	return m.upsertSetting(LeaderboardLimit, TypeInt, LeaderboardLimit+" int setting", username, func(s *PlatformSetting) {
		s.ValueInt = limit
	})
}

func (m *SettingsManager) GetLeaderboardLimit() (int, error) {
	return m.getIntSetting(LeaderboardLimit)
}

func (m *SettingsManager) SetGameboardShowTeamMembers(show bool, username string) error {
	return m.upsertSetting(GameboardShowTeamMembers, TypeBool, GameboardShowTeamMembers+" boolean setting", username, func(s *PlatformSetting) {
		s.ValueBool = show
	})
}

func (m *SettingsManager) GetGameboardShowTeamMembers() (bool, error) {
	return m.getBoolSetting(GameboardShowTeamMembers)
}

func (m *SettingsManager) SetRegistrationToken(token, username string) error {
	return m.upsertSetting(RegistrationToken, TypeString, RegistrationToken+" string setting", username, func(s *PlatformSetting) {
		s.ValueString = token
	})
}

func (m *SettingsManager) GetRegistrationToken() (string, error) {
	return m.getStringSetting(RegistrationToken)
}

func (m *SettingsManager) SetGameboardChatMaxLen(maxLen int, username string) error {
	return m.upsertSetting(GameboardChatMaxLen, TypeInt, GameboardChatMaxLen+" int setting", username, func(s *PlatformSetting) {
		s.ValueInt = maxLen
	})
}

func (m *SettingsManager) GetGameboardChatMaxLen() (int, error) {
	return m.getIntSetting(GameboardChatMaxLen)
}

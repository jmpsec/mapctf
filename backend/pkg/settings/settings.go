package settings

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	// LoginEnabled is the setting name for login enabled/disabled
	LoginEnabled string = "login_enabled"
	// RegistrationEnabled is the setting name for registration enabled/disabled
	RegistrationEnabled string = "registration_enabled"
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
	// CustomOrg is the setting name for the custom organization
	CustomOrg string = "custom_org"
)

// BooleanSettings to be used as check for valid setting and to keep default value, if needed
var BooleanSettings = map[string]bool{
	LoginEnabled:        false,
	RegistrationEnabled: false,
	ScoringEnabled:      false,
	GamePaused:          false,
	GameStarted:         false,
}

// StringSettings to be used as check for valid setting and to keep default value
var StringSettings = map[string]string{
	CustomOrg: "",
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
	if err := m.DB.Model(&PlatformSetting{}).
		Where("name = ? AND uuid = ?", name, uuid).
		Update("value", value).Error; err != nil {
		return fmt.Errorf("failed to update setting value: %w", err)
	}
	if err := m.LogEvent(setting.ID, EventUpdate, username, setting.UUID); err != nil {
		return fmt.Errorf("LogEvent PlatformSetting %w", err)
	}
	return nil
}

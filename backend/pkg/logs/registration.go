package logs

import "gorm.io/gorm"

// RegistrationLog to hold each registration entry in the system
type RegistrationLog struct {
	gorm.Model
	Username  string
	Password  bool
	Name      string
	Team      string
	Email     string
	Logo      string
	IPAddress string
	UUID      string `gorm:"index"`
}

// CreateRegistrationLog to create a new registration log entry
func (l *LogManager) CreateRegistrationLog(registrationLog RegistrationLog) error {
	if err := l.DB.Create(&registrationLog).Error; err != nil {
		return err
	}
	return nil
}

// NewRegistrationLog to create a new registration log struct
func (l *LogManager) NewRegistrationLog(username, name, team, email, logo, ipAddress, uuid string, password bool) (RegistrationLog, error) {
	return RegistrationLog{
		Username:  username,
		Password:  password,
		Name:      name,
		Team:      team,
		Email:     email,
		Logo:      logo,
		IPAddress: ipAddress,
		UUID:      uuid,
	}, nil
}

// AllRegistrationLogs to get all registration logs for a given UUID
func (l *LogManager) AllRegistrationLogs(uuid string) ([]RegistrationLog, error) {
	var registrationLogs []RegistrationLog
	if err := l.DB.Where("uuid = ?", uuid).Find(&registrationLogs).Error; err != nil {
		return registrationLogs, err
	}
	return registrationLogs, nil
}

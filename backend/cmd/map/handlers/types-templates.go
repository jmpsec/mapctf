package handlers

import (
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
)

// LoginTemplateData for passing data to the login template
type LoginTemplateData struct {
	Title         string
	LoginType     string
	LoginMsg      string
	LoginURL      string
	UUID          string
	TeamLogin     bool
	Authenticated bool
	Admin         bool
}

// IndexTemplateData for passing data to the index template
type IndexTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// CountdownTemplateData for passing data to the countdown template
type CountdownTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// RulesTemplateData for passing data to the rules template
type RulesTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// GameboardTemplateData for passing data to the gameboard template
type GameboardTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
}

// RegistrationTemplateData for passing data to the registration template
type RegistrationTemplateData struct {
	Title            string
	RegistrationType string
	RegistrationMsg  string
	RegisterURL      string
	UUID             string
	OpenRegistration bool
	Authenticated    bool
	Admin            bool
}

// AdminSettingsTemplateData for passing data to the admin settings template
type AdminSettingsTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string

	LoginEnabled        bool
	RegistrationEnabled bool
	ScoringEnabled      bool
	GamePaused          bool
	GameStarted         bool
	CustomOrg           string
	Language            string
	GameStartTime       string
	GameEndTime         string
}

// AdminTemplateData for passing data to the admin template
type AdminTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
}

// AdminControlsTemplateData for passing data to the admin controls template
type AdminControlsTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
}

// AdminUsersTemplateData for passing data to the admin users template
type AdminUsersTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
	Users         []users.PlatformUser
}

// AdminTeamsTemplateData for passing data to the admin teams template
type AdminTeamsTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
	Teams         []teams.PlatformTeam
}

// AdminChallengesTemplateData for passing data to the admin challenges template
type AdminChallengesTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
	Challenges    []challenges.Challenge
	Categories    []challenges.Category
}

// ErrorTemplateData for passing data to the error template
type ErrorTemplateData struct {
	Title  string
	Error  string
	Status string
	Header string
}

package handlers

import (
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
)

// LoginTemplateData for passing data to the login template
type LoginTemplateData struct {
	Title                string
	LoginType            string
	LoginMsg             string
	LoginURL             string
	UUID                 string
	LoginEnabled         bool
	LoginStrongPasswords bool
	Authenticated        bool
	Admin                bool
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
	Title                    string
	UUID                     string
	Authenticated            bool
	Admin                    bool
	CurrentUsername          string
	GameboardChatMaxLen      int
	GameboardShowTeamMembers bool
	Countries                []countries.MapCountry
}

// RegistrationTemplateData for passing data to the registration template
type RegistrationTemplateData struct {
	Title               string
	RegistrationMsg     string
	RegisterURL         string
	UUID                string
	RegistrationEnabled bool
	RegistrationNames   bool
	RegistrationEmails  bool
	RegistrationType    int
	RegistrationTypeStr string
	Authenticated       bool
	Admin               bool
}

// AdminSettingsTemplateData for passing data to the admin settings template
type AdminSettingsTemplateData struct {
	Title                    string
	UUID                     string
	Authenticated            bool
	Admin                    bool
	Status                   string
	Message                  string
	LoginEnabled             bool
	LoginStrongPasswords     bool
	RegistrationEnabled      bool
	RegistrationNames        bool
	RegistrationEmails       bool
	RegistrationType         int
	RegistrationToken        string
	ScoringEnabled           bool
	GamePaused               bool
	GameStarted              bool
	CustomOrg                string
	CustomLogo               string
	Language                 string
	LeaderboardLimit         int
	GameboardShowTeamMembers bool
	GameStartTime            string
	GameEndTime              string
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
	TeamNames     map[uint]string
	Teams         []teams.PlatformTeam
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
	Logos         []teams.TeamLogo
	AllLogos      []teams.TeamLogo
	Users         []users.PlatformUser
	TeamMembers   map[uint][]users.PlatformUser
}

// AdminChallengesTemplateData for passing data to the admin challenges template
type AdminChallengesTemplateData struct {
	Title                   string
	UUID                    string
	Authenticated           bool
	Admin                   bool
	Status                  string
	Message                 string
	Challenges              []challenges.Challenge
	AllCountries            []countries.MapCountry
	AvailableCountries      []countries.MapCountry
	ChallengeCountryOptions map[uint][]countries.MapCountry
	CountryFlag             map[string]string
	Categories              []challenges.Category
	Solves                  []logs.ActivityLog
	ChallengeSolves         map[uint][]logs.ActivityLog
}

// AdminActivityTemplateData for passing data to the admin activity template
type AdminActivityTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
	Activity      []logs.ActivityLog
}

// AdminAnnouncementsTemplateData for passing data to the admin announcements template
type AdminAnnouncementsTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
	Announcements []logs.Announcement
}

// AdminCountriesTemplateData for passing data to the admin countries template
type AdminCountriesTemplateData struct {
	Title         string
	UUID          string
	Authenticated bool
	Admin         bool
	Status        string
	Message       string
	Countries     []countries.MapCountry
	ChallengeName map[uint]string
	CountryFlag   map[string]string
}

// ErrorTemplateData for passing data to the error template
type ErrorTemplateData struct {
	Title  string
	Error  string
	Status string
	Header string
}

package handlers

import "time"

// ApiLoginRequest to receive login requests
type ApiLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ApiErrorResponse to be returned to API requests with the error message
type ApiErrorResponse struct {
	Error string `json:"error"`
}

// ApiLoginResponse to be returned for login requests
type ApiLoginResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message,omitempty"`
	Token   string    `json:"token"`
	ExpTime time.Time `json:"expTime"`
}

// Team represents a team in the gameboard
type Team struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
	Logo  string `json:"logo,omitempty"`
}

// AdminTeam represents a team in the admin panel
type AdminTeam struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Score   int    `json:"score"`
	Members int    `json:"members"`
	Active  bool   `json:"active"`
}

// Challenge represents a challenge in the gameboard
type Challenge struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Points      int    `json:"points"`
	Solved      bool   `json:"solved"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// AdminChallenge represents a challenge in the admin panel
type AdminChallenge struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Points      int    `json:"points"`
	Flag        string `json:"flag"`
	Active      bool   `json:"active"`
}

// AdminStats represents admin dashboard statistics
type AdminStats struct {
	TotalTeams      int `json:"totalTeams"`
	TotalChallenges int `json:"totalChallenges"`
	ActivePlayers   int `json:"activePlayers"`
	TotalFlags      int `json:"totalFlags"`
}

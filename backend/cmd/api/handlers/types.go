package handlers

import "time"

// ApiLoginRequest to receive login requests
type ApiLoginRequest struct {
	Username string `json:"username"`
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

// AdminTeam represents a team in the admin panel
type AdminTeam struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Score   int    `json:"score"`
	Members int    `json:"members"`
	Active  bool   `json:"active"`
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

// CreateTeamRequest represents a request to create a team
type CreateTeamRequest struct {
	Name      string `json:"name"`
	Logo      string `json:"logo"`
	Protected bool   `json:"protected"`
	Visible   bool   `json:"visible"`
}

// CreateChallengeRequest represents a request to create a challenge
type CreateChallengeRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CategoryID  uint   `json:"categoryID"`
	Active      bool   `json:"active"`
	Points      int    `json:"points"`
	Bonus       int    `json:"bonus"`
	BonusDecay  int    `json:"bonusDecay"`
	Flag        string `json:"flag"`
	Hint        string `json:"hint"`
	Penalty     int    `json:"penalty"`
}

// Challenge represents a challenge in the gameboard (public API)
type Challenge struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Points      int    `json:"points"`
	Solved      bool   `json:"solved"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

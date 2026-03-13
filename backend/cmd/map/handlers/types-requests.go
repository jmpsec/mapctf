package handlers

type MapRegistrationRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Logo     string `json:"logo"`
	Team     string `json:"team"`
}

type MapRegistrationResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Redirect string `json:"redirect,omitempty"`
}

// MapLoginRequest to receive login requests
type MapLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// MapLoginResponse to be returned to map requests after a successful login
type MapLoginResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Redirect string `json:"redirect,omitempty"`
}

// MapLogoutResponse to be returned to map requests after a successful logout
type MapLogoutResponse MapLoginResponse

// AdminSettingsRequest to receive admin settings update requests
type AdminSettingsRequest struct {
	SettingName  string `json:"setting_name,omitempty"`
	SettingValue string `json:"setting_value,omitempty"`
	Name         string `json:"name,omitempty"`
	Value        string `json:"value,omitempty"`
}

// AdminChallengeCreateRequest to receive admin challenge creation requests
type AdminChallengeCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CategoryID  string `json:"category_id"`
	Active      string `json:"active"`
	Points      string `json:"points"`
	Bonus       string `json:"bonus"`
	BonusDecay  string `json:"bonus_decay"`
	Penalty     string `json:"penalty"`
	Flag        string `json:"flag"`
	Hint        string `json:"hint"`
}

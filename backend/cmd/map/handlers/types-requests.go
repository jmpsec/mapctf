package handlers

type MapRegistrationRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
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

// MapErrorResponse to be returned to map requests with the error message
type MapErrorResponse struct {
	Error string `json:"error"`
}

// MapChatResponse to be returned to map requests after a successful chat message creation
type MapChatResponse struct {
	Success bool `json:"success"`
}

// MapScoreRequest to receive challenge scoring requests
type MapScoreRequest struct {
	CountryCode string `json:"country_code"`
	Flag        string `json:"flag"`
}

// MapScoreResponse to be returned to map requests after a scoring attempt
type MapScoreResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	CountryCode   string `json:"country_code,omitempty"`
	ChallengeID   uint   `json:"challenge_id,omitempty"`
	PointsAwarded int    `json:"points_awarded,omitempty"`
	TotalPoints   int    `json:"total_points,omitempty"`
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
	Country     string `json:"country"`
	Active      string `json:"active"`
	Points      string `json:"points"`
	Bonus       string `json:"bonus"`
	BonusDecay  string `json:"bonus_decay"`
	Penalty     string `json:"penalty"`
	Flag        string `json:"flag"`
	Hint        string `json:"hint"`
}

// AdminCategoryCreateRequest to receive admin category creation requests
type AdminCategoryCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
}

// AdminUserCreateRequest to receive admin user creation requests
type AdminUserCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	TeamID   string `json:"team_id"`
	Admin    string `json:"admin"`
	Service  string `json:"service"`
	Active   string `json:"active"`
}

// AdminUserUpdateRequest to receive admin user update requests
type AdminUserUpdateRequest struct {
	TeamID  string `json:"team_id"`
	Admin   string `json:"admin"`
	Service string `json:"service"`
	Active  string `json:"active"`
}

// AdminTeamCreateRequest to receive admin team creation requests
type AdminTeamCreateRequest struct {
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// AdminTeamUpdateRequest to receive admin team update requests
type AdminTeamUpdateRequest struct {
	Name      string `json:"name"`
	Logo      string `json:"logo"`
	Active    string `json:"active"`
	Visible   string `json:"visible"`
	Protected string `json:"protected"`
}

// AdminLogoCreateRequest to receive admin team logo creation requests
type AdminLogoCreateRequest struct {
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// AdminLogoUpdateRequest to receive admin team logo update requests
type AdminLogoUpdateRequest struct {
	Name      string `json:"name"`
	Enabled   string `json:"enabled"`
	Protected string `json:"protected"`
}

// ChatEntryRequest to receive chat entry requests
type ChatEntryRequest struct {
	Message string `json:"message"`
}

// AdminActivityCreateRequest to receive admin custom activity entry requests
type AdminActivityCreateRequest struct {
	Subject string `json:"subject"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

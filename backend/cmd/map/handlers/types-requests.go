package handlers

type MapRegistrationRequest struct {
	Name     string `json:"name"`
	Alias    string `json:"alias"`
	Email    string `json:"email"`
	Team     string `json:"team"`
	Password string `json:"password"`
}

type MapRegistrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
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

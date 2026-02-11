package handlers

// MapLoginRequest to receive login requests
type MapLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	EntID    uint   `json:"entID"`
}

// MapLoginResponse to be returned to map requests after a successful login
type MapLoginResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Redirect string `json:"redirect,omitempty"`
}

// MapLogoutResponse to be returned to map requests after a successful logout
type MapLogoutResponse MapLoginResponse

// MapErrorResponse to be returned to map requests with the error message
type MapErrorResponse struct {
	Error string `json:"error"`
}

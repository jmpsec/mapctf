package handlers

// ApiLoginRequest to receive login requests
type ApiLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ApiErrorResponse to be returned to API requests with the error message
type ApiErrorResponse struct {
	Error string `json:"error"`
}

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

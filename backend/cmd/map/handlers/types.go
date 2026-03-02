package handlers

// MapErrorResponse to be returned to map requests with the error message
type MapErrorResponse struct {
	Error string `json:"error"`
}

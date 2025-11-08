package handlers

import "net/http"

// GetTeamsHandler - Handle get teams requests
func (h *HandlersAPI) GetTeamsHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Teams List", http.StatusOK, []byte(okContent))
}

// GetAdminTeamsHandler - Handle get admin teams requests
func (h *HandlersAPI) GetAdminTeamsHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Admin Teams List", http.StatusOK, []byte(okContent))
}

// CreateAdminTeamHandler - Handle create team requests
func (h *HandlersAPI) CreateAdminTeamHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Team Created", http.StatusOK, []byte(okContent))
}

// UpdateTeamHandler - Handle update team requests
func (h *HandlersAPI) UpdateTeamHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Team Updated", http.StatusOK, []byte(okContent))
}

// DeleteAdminTeamHandler - Handle delete admin team requests
func (h *HandlersAPI) DeleteAdminTeamHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Admin Team Deleted", http.StatusOK, []byte(okContent))
}

package handlers

import "net/http"

// GetChallengesHandler - Handle get challenges requests
func (h *HandlersAPI) GetChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Challenges List", http.StatusOK, []byte(okContent))
}

// GetAdminChallengesHandler - Handle get admin challenges requests
func (h *HandlersAPI) GetAdminChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Admin Challenges List", http.StatusOK, []byte(okContent))
}

// CreateAdminChallengeHandler - Handle create challenge requests
func (h *HandlersAPI) CreateAdminChallengeHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Challenge Created", http.StatusOK, []byte(okContent))
}

// UpdateChallengeHandler - Handle update challenge requests
func (h *HandlersAPI) UpdateChallengeHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Challenge Updated", http.StatusOK, []byte(okContent))
}

// DeleteAdminChallengeHandler - Handle delete admin challenge requests
func (h *HandlersAPI) DeleteAdminChallengeHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Admin Challenge Deleted", http.StatusOK, []byte(okContent))
}

// UpdateAdminChallengeHandler - Handle update admin challenge requests
func (h *HandlersAPI) UpdateAdminChallengeHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Admin Challenge Updated", http.StatusOK, []byte(okContent))
}

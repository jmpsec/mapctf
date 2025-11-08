package handlers

import "net/http"

// GetAdminStatsHandler - Handle get admin stats requests
func (h *HandlersAPI) GetAdminStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "Admin Stats", http.StatusOK, []byte(okContent))
}

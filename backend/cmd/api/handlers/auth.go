package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// LoginHandler - Handle login requests
func (h *HandlersAPI) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	var l ApiLoginRequest
	// Parse request JSON body
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		log.Err(err).Msg("error parsing POST body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "error parsing POST body"})
		return
	}

	// Mock authentication - accept any email/password for development
	// In production, this would check against the database
	if l.Email == "" || l.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "email and password are required"})
		return
	}

	// Mock successful login
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, ApiLoginResponse{
		Success: true,
		Message: "Login successful",
	})
}

// LogoutHandler - Handle logout requests
func (h *HandlersAPI) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "Checked", http.StatusOK, []byte(okContent))
}

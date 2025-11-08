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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error parsing POST body"})
		return
	}
	// Send response
	HTTPResponse(w, "Checked", http.StatusOK, []byte(okContent))
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

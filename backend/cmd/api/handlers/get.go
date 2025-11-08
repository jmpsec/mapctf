package handlers

import (
	"net/http"
)

// ErrorHandler for error requests
func (h *HandlersAPI) ErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "", http.StatusInternalServerError, []byte(errorContent))
}

// ForbiddenHandler for forbidden error requests
func (h *HandlersAPI) ForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP for environment
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, errorContent)
}

// RootHandler - Handler for the root path
func (h *HandlersAPI) RootHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// HealthHandler - Handle health requests
func (h *HandlersAPI) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusOK, []byte(okContent))
}

// CheckHandlerNoAuth - Handle unauthenticated check requests
func (h *HandlersAPI) CheckHandlerNoAuth(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "Checked", http.StatusOK, []byte(okContent))
}


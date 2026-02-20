package handlers

import (
	"net/http"
)

// ErrorHandler for error requests
func (h *HandlersMap) ErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "", http.StatusInternalServerError, []byte(errorContent))
}

// ForbiddenHandler for forbidden error requests
func (h *HandlersMap) ForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP for environment
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, forbiddenContent)
}

// RootHandler for root requests
func (h *HandlersMap) RootHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, rootContent)
}

// HealthHandler for health requests
func (h *HandlersMap) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusOK, []byte(okContent))
}

// FaviconHandler for the favicon
func (h *HandlersMap) FaviconHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, "/static/img/favicon.png")
}

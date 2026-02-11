package handlers

import (
	"net/http"
)

func (h *HandlersMap) ErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "", http.StatusInternalServerError, []byte(errorContent))
}

func (h *HandlersMap) ForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP for environment
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, forbiddenContent)
}

func (h *HandlersMap) RootHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, rootContent)
}

func (h *HandlersMap) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusOK, []byte(okContent))
}

package handlers

import (
	"net/http"
)

// ErrorHandler for error requests
// @Summary      Error page
// @Description  Returns a 500 Internal Server Error response
// @Tags         system
// @Produce      text/plain
// @Success      500  {string}  string  "Internal server error"
// @Router       /error [get]
func (h *HandlersAPI) ErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Send response
	HTTPResponse(w, "", http.StatusInternalServerError, []byte(errorContent))
}

// ForbiddenHandler for forbidden error requests
// @Summary      Forbidden page
// @Description  Returns a 403 Forbidden response
// @Tags         system
// @Produce      text/plain
// @Success      403  {string}  string  "Forbidden"
// @Router       /forbidden [get]
func (h *HandlersAPI) ForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP for environment
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, forbiddenContent)
}

// RootHandler - Handler for the root path
// @Summary      Root endpoint
// @Description  Redirects to dashboard
// @Tags         system
// @Success      302  {string}  string  "Redirect to dashboard"
// @Router       / [get]
func (h *HandlersAPI) RootHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, rootContent)
}

// HealthHandler - Handle health requests
// @Summary      Health check
// @Description  Returns health status of the API service
// @Tags         system
// @Produce      text/plain
// @Success      200  {string}  string  "Service is healthy"
// @Router       /health [get]
func (h *HandlersAPI) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusOK, []byte(okContent))
}

// CheckHandlerNoAuth - Handle unauthenticated check requests
// @Summary      Unauthenticated check
// @Description  Check endpoint that does not require authentication
// @Tags         system
// @Produce      text/plain
// @Success      200  {string}  string  "Check successful"
// @Router       /api/v1/checks-no-auth [get]
func (h *HandlersAPI) CheckHandlerNoAuth(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "Checked", http.StatusOK, []byte(checkedNoAuthContent))
}

// CheckHandlerAuth - Handle authenticated check requests
// @Summary      Authenticated check
// @Description  Check endpoint that requires authentication
// @Tags         system
// @Produce      text/plain
// @Success      200  {string}  string  "Check successful"
// @Router       /api/v1/checks-auth [get]
func (h *HandlersAPI) CheckHandlerAuth(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "Checked", http.StatusOK, []byte(checkedAuthContent))
}

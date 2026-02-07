package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// LoginHandler - Handle login requests
// @Summary      User login
// @Description  Authenticate user with email and password, returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ApiLoginRequest  true  "Login credentials"
// @Success      200      {object}  ApiLoginResponse  "Login successful"
// @Failure      400      {object}  ApiErrorResponse  "Bad request - invalid input"
// @Failure      401      {object}  ApiErrorResponse  "Unauthorized - invalid credentials"
// @Failure      500      {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/auth/login [post]
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
	if l.Email == "" || l.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "email and password are required"})
		return
	}
	valid, user := h.Users.CheckLoginCredentials(l.Email, l.Password)
	if !valid {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, ApiErrorResponse{Error: "invalid credentials"})
		return
	}
	token, expTime, err := h.Users.CreateToken(user.Username, h.ServiceName, h.Config.JWT.HoursToExpire)
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error creating token"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, ApiLoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		ExpTime: expTime,
	})
}

// LogoutHandler - Handle logout requests
// @Summary      User logout
// @Description  Logout user session
// @Tags         auth
// @Produce      text/plain
// @Success      200  {string}  string  "Logout successful"
// @Router       /api/v1/auth/logout [get]
func (h *HandlersAPI) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "Checked", http.StatusOK, []byte(okContent))
}

// Middleware to check access to a resource based on the authentication enabled
func (h *HandlersAPI) checkAuth(handler http.Handler, auth, jwtSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractHeaderToken(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		_, err := h.Users.CheckToken(jwtSecret, token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Access granted
		// TODO: Add context with claims
		handler.ServeHTTP(w, r.WithContext(nil))
	})
}

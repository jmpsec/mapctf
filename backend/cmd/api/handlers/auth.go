package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// ContextKey type for context values
type ContextKey string

const (
	// ContextKeyUser is the key for storing username in request context
	ContextKeyUser ContextKey = "user"
	// ContextKeyClaims is the key for storing full token claims in request context
	ContextKeyClaims ContextKey = "claims"
)

// LoginHandler - Handle login requests
// @Summary      User login
// @Description  Authenticate user with username, password, and entity ID, returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ApiLoginRequest  true  "Login credentials (username, password, entID)"
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
	if l.Username == "" || l.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "username and password are required"})
		return
	}
	if l.EntID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "entity ID is required"})
		return
	}
	valid, user := h.Users.CheckLoginCredentials(l.Username, l.Password, l.EntID)
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

// GetUserFromContext extracts the username from request context
// Returns empty string if not found
func GetUserFromContext(r *http.Request) string {
	if user, ok := r.Context().Value(ContextKeyUser).(string); ok {
		return user
	}
	return ""
}

// GetClaimsFromContext extracts the token claims from request context
// Returns nil if not found
func GetClaimsFromContext(r *http.Request) interface{} {
	return r.Context().Value(ContextKeyClaims)
}

// Middleware to check access to a resource based on the authentication enabled
func (h *HandlersAPI) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token := extractHeaderToken(r)
		if token == "" {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized,
				ApiErrorResponse{Error: "Authorization token required"})
			return
		}

		// Validate token and get claims
		claims, err := h.Users.CheckToken(h.Config.JWT.Secret, token)
		if err != nil {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized,
				ApiErrorResponse{Error: "Invalid or expired token"})
			return
		}

		// Store user info in context for use in handlers
		ctx := context.WithValue(r.Context(), ContextKeyUser, claims.Username)
		ctx = context.WithValue(ctx, ContextKeyClaims, claims)

		// Continue to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

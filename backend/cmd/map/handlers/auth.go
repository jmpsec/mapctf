package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const (
	// ContextKeyUser is the key for storing username in request context
	ContextKeyUser string = "username"
	// ContextKeyAdmin is the key for storing admin status in request context
	ContextKeyAdmin string = "isAdmin"
)

func (h *HandlersMap) LoginPOSTHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Get UUID from URL path parameters and validate it
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	var l MapLoginRequest
	// Parse request JSON body
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		log.Err(err).Msg(h.T(r.Context())("auth.error_parse_body"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("auth.error_parse_body")})
		return
	}
	if l.Username == "" || l.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("auth.user_pass_required")})
		return
	}
	valid, user := h.Users.CheckLoginCredentials(l.Username, l.Password, uuid)
	if !valid {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("auth.invalid_credentials")})
		return
	}
	if !user.Admin {
		if h.Settings == nil {
			log.Err(errors.New("settings manager not initialized")).Msg("error checking login_enabled")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("auth.login_unavailable")})
			return
		}
		loginEnabled, err := h.Settings.GetLoginEnabled(uuid)
		if err != nil {
			log.Err(err).Msg("error getting login enabled setting")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("auth.login_unavailable")})
			return
		}
		if !loginEnabled {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusForbidden, MapErrorResponse{Error: h.T(r.Context())("auth.login_disabled")})
			return
		}
	}
	err := h.Sessions.RenewToken(r.Context())
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("auth.error_renew_session")})
		return
	}
	h.Sessions.Put(r.Context(), string(ContextKeyUser), user.Username)
	h.Sessions.Put(r.Context(), string(ContextKeyAdmin), user.Admin)
	// Update last login time for the user and other relevant info
	if err := h.Users.UpdateUserSession(user.Username, getRealIP(r), r.UserAgent(), uuid); err != nil {
		log.Err(err).Msg("error updating user session")
	}
	redirectTo := "/" + uuid + "/gameboard"
	if user.Admin {
		redirectTo = "/" + uuid + "/admin"
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapLoginResponse{
		Success:  true,
		Message:  h.T(r.Context())("login.success_msg"),
		Redirect: redirectTo,
	})
}

func (h *HandlersMap) LogoutPOSTHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Get UUID from URL path
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		log.Err(errors.New(h.T(r.Context())("auth.uuid_required"))).Msg(h.T(r.Context())("auth.uuid_required"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("auth.uuid_required")})
		return
	}
	if err := h.Sessions.Destroy(r.Context()); err != nil {
		log.Err(err).Msg(h.T(r.Context())("auth.error_destroy_session"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("auth.error_destroy_session")})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapLogoutResponse{
		Success:  true,
		Message:  h.T(r.Context())("login.logout_msg"),
		Redirect: "/" + uuid + "/login",
	})
}

func (h *HandlersMap) IsAuthenticated(ctx context.Context) bool {
	return h.Sessions.GetString(ctx, string(ContextKeyUser)) != ""
}

func (h *HandlersMap) IsAdmin(ctx context.Context) bool {
	return h.Sessions.GetBool(ctx, string(ContextKeyAdmin))
}

func (h *HandlersMap) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Sessions.GetString(r.Context(), string(ContextKeyUser)) == "" {
			http.Redirect(w, r, "/"+h.Config.Map.UUID+"/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *HandlersMap) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.Sessions.GetBool(r.Context(), string(ContextKeyAdmin)) {
			http.Error(w, forbiddenContent, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

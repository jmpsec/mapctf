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
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	var l MapLoginRequest
	// Parse request JSON body
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		log.Err(err).Msg("error parsing POST body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "error parsing POST body"})
		return
	}
	if l.Username == "" || l.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "username and password are required"})
		return
	}
	valid, user := h.Users.CheckLoginCredentials(l.Username, l.Password, uuid)
	if !valid {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: "invalid credentials"})
		return
	}
	err := h.Sessions.RenewToken(r.Context())
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error renewing session"})
		return
	}
	h.Sessions.Put(r.Context(), string(ContextKeyUser), user.Username)
	h.Sessions.Put(r.Context(), string(ContextKeyAdmin), user.Admin)
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapLoginResponse{
		Success:  true,
		Message:  "Login successful",
		Redirect: "/" + uuid + "/gameboard",
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
		log.Err(errors.New("UUID is required")).Msg("UUID is required")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "UUID is required"})
		return
	}
	if err := h.Sessions.Destroy(r.Context()); err != nil {
		log.Err(err).Msg("error destroying session")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error destroying session"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapLogoutResponse{
		Success:  true,
		Message:  "Logout successful",
		Redirect: "/" + uuid + "/login",
	})
}

func (h *HandlersMap) IsAuthenticated(ctx context.Context) bool {
	return h.Sessions.GetString(ctx, string(ContextKeyUser)) != ""
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

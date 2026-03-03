package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *HandlersMap) RegistrationPOSTHandler(w http.ResponseWriter, r *http.Request) {
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
	// Parse request body
	var req MapRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "invalid request body"})
		return
	}
	if req.Username == "" || req.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "username and password are required"})
		return
	}
	if req.Email == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "email is required"})
		return
	}
	if req.Team == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "team is required"})
		return
	}
	// Register team
	nTeam, err := h.Teams.Register(req.Team, req.Logo, uuid)
	if err != nil {
		log.Err(err).Msg("error registering team")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to register team"})
		return
	}
	// Register user
	err = h.Users.Register(req.Username, req.Password, req.Name, req.Email, nTeam.ID, uuid)
	if err != nil {
		log.Err(err).Msg("error registering user")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to register user"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapRegistrationResponse{
		Success:  true,
		Message:  "Registration successful",
		Redirect: "/" + uuid + "/login",
	})
}

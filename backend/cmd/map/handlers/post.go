package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/settings"
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
	regType, err := h.Settings.GetRegistrationType()
	if err != nil {
		log.Err(err).Msg("error getting registration type setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to get registration type setting"})
		return
	}
	if req.Token == "" && regType == settings.TokenRegistration {
		log.Err(errors.New("registration token is required")).Msg("registration token is required")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "registration token is required"})
		return
	}
	if regType == settings.TokenRegistration {
		regToken, err := h.Settings.GetRegistrationToken()
		if err != nil {
			log.Err(err).Msg("error getting registration token setting")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to get registration token setting"})
			return
		}
		if req.Token != regToken {
			log.Err(errors.New("invalid registration token")).Msg("invalid registration token")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "invalid registration token"})
			return
		}
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
	nTeam, err := h.Teams.Register(req.Team, req.Logo)
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

func (h *HandlersMap) ChatPOSTHandler(w http.ResponseWriter, r *http.Request) {
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
	var req ChatEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "invalid request body"})
		return
	}
	chatMaxLen, err := h.Settings.GetGameboardChatMaxLen()
	if err != nil {
		log.Err(err).Msg("error getting gameboard chat max length setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to get gameboard chat max length	 setting"})
		return
	}
	// Get user from session
	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		log.Err(errors.New("user not authenticated")).Msg("user not authenticated")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: "user not authenticated"})
		return
	}
	// Get user team
	teamID, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error getting user for chat message")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to get user for chat message"})
		return
	}
	if err := h.Chat.CreateNew(username, req.Message, teamID.TeamID, chatMaxLen); err != nil {
		log.Err(err).Msg("error creating chat message")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to create chat message"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapChatResponse{
		Success: true,
	})
}

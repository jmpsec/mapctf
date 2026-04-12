package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/rs/zerolog/log"
)

// JSONActivityHandler to return all activity logs for a given UUID in JSON format
func (h *HandlersMap) JSONActivityHandler(w http.ResponseWriter, r *http.Request) {
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
	// Get all activity logs for the given UUID
	activityLogs, err := h.Logs.AllActivity(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving activity logs")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving activity logs"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, activityLogs)
}

// JSONAnnouncementsHandler to return all announcements for a given UUID in JSON format
func (h *HandlersMap) JSONAnnouncementsHandler(w http.ResponseWriter, r *http.Request) {
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
	// Get all announcements for the given UUID
	announcements, err := h.Logs.AllAnnouncements(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving announcements")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving announcements"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, announcements)
}

// JSONTeamsHandler to return all teams for a given UUID in JSON format
func (h *HandlersMap) JSONTeamsHandler(w http.ResponseWriter, r *http.Request) {
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
	// Get all teams for the given UUID
	allTeams, err := h.Teams.GetAll()
	if err != nil {
		log.Err(err).Msg("error retrieving teams")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving teams"})
		return
	}
	filteredTeams := make([]teams.PlatformTeam, 0, len(allTeams))
	for _, team := range allTeams {
		if !team.Active || !team.Visible {
			continue
		}
		filteredTeams = append(filteredTeams, team)
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, filteredTeams)
}

// JSONChallengesHandler to return all challenges for a given UUID in JSON format
func (h *HandlersMap) JSONChallengesHandler(w http.ResponseWriter, r *http.Request) {
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
	// Get all active challenges for the given UUID
	challenges, err := h.Challenges.GetActive(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving challenges")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving challenges"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, challenges)
}

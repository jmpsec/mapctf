package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type JSONTeamResponse struct {
	Name        string    `json:"name"`
	Logo        string    `json:"logo"`
	Points      int       `json:"points"`
	LastScore   time.Time `json:"last_score"`
	TeamMembers []string  `json:"team_members,omitempty"`
}

type JSONCountryDataResponse struct {
	CountryCode     string   `json:"country_code"`
	Active          bool     `json:"active"`
	Points          int      `json:"points"`
	Category        string   `json:"category"`
	Owner           string   `json:"owner"`
	Completed       []string `json:"completed"`
	Intro           string   `json:"intro"`
	Hint            string   `json:"hint"`
	LandPath        string   `json:"land_path"`
	LandClass       string   `json:"land_class"`
	LandStyle       string   `json:"land_style"`
	MarkerPath      string   `json:"marker_path"`
	MarkerClass     string   `json:"marker_class"`
	MarkerStyle     string   `json:"marker_style"`
	MarkerTransform string   `json:"marker_transform"`
}

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

	showTeamMembers, err := h.Settings.GetGameboardShowTeamMembers()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Err(err).Msg("error retrieving gameboard_show_team_members setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving team settings"})
		return
	}

	membersByTeamID := make(map[uint][]string)
	if showTeamMembers {
		allUsers, err := h.Users.GetAll(uuid)
		if err != nil {
			log.Err(err).Msg("error retrieving users for teams JSON")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving team members"})
			return
		}
		for _, user := range allUsers {
			if user.TeamID == 0 || !user.Active || user.Service {
				continue
			}
			membersByTeamID[user.TeamID] = append(membersByTeamID[user.TeamID], user.Username)
		}
	}

	filteredTeams := make([]JSONTeamResponse, 0, len(allTeams))
	for _, team := range allTeams {
		if !team.Active || !team.Visible {
			continue
		}
		filteredTeams = append(filteredTeams, JSONTeamResponse{
			Name:        team.Name,
			Logo:        team.Logo,
			Points:      team.Points,
			LastScore:   team.LastScore,
			TeamMembers: membersByTeamID[team.ID],
		})
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

// JSONCountriesHandler returns live gameboard country data for all countries,
// marking only challenge-backed countries as active.
func (h *HandlersMap) JSONCountriesHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		log.Err(errors.New("UUID is required")).Msg("UUID is required")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "UUID is required"})
		return
	}

	if h.Countries == nil || h.Challenges == nil {
		log.Err(errors.New("countries or challenges manager not initialized")).Msg("error retrieving country data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
		return
	}

	allCountries, err := h.Countries.GetAll()
	if err != nil {
		log.Err(err).Msg("error retrieving countries")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
		return
	}

	activeChallenges, err := h.Challenges.GetActive(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving active challenges for country data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
		return
	}

	categoriesByID := map[uint]challenges.Category{}
	allCategories, err := h.Challenges.GetAllCategories(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving categories for country data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
		return
	}
	for _, category := range allCategories {
		categoriesByID[category.ID] = category
	}

	activeChallengesByCode := make(map[string]challenges.Challenge, len(activeChallenges))
	for _, challenge := range activeChallenges {
		countryCode := strings.ToUpper(strings.TrimSpace(challenge.Country))
		if countryCode == "" {
			continue
		}
		if _, exists := activeChallengesByCode[countryCode]; exists {
			continue
		}
		activeChallengesByCode[countryCode] = challenge
	}

	response := make(map[string]JSONCountryDataResponse, len(allCountries))
	for _, country := range allCountries {
		countryCode := strings.ToUpper(strings.TrimSpace(country.CountryCode))
		challenge, hasChallenge := activeChallengesByCode[countryCode]

		categoryName := ""
		if hasChallenge {
			if category, exists := categoriesByID[challenge.CategoryID]; exists {
				categoryName = category.Name
			}
		}

		response[country.Name] = JSONCountryDataResponse{
			CountryCode:     country.CountryCode,
			Active:          hasChallenge,
			Points:          challenge.Points,
			Category:        categoryName,
			Owner:           "",
			Completed:       []string{},
			Intro:           challenge.Description,
			Hint:            challenge.Hint,
			LandPath:        country.LandPath,
			LandClass:       country.LandClass,
			LandStyle:       country.LandStyle,
			MarkerPath:      country.MarkerPath,
			MarkerClass:     country.MarkerClass,
			MarkerStyle:     country.MarkerStyle,
			MarkerTransform: country.MarkerTransform,
		}
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, response)
}

// JSONChatHandler to return all chat entries for a given UUID in JSON format
func (h *HandlersMap) JSONChatHandler(w http.ResponseWriter, r *http.Request) {
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
	// Get all chat entries for the given UUID
	chatEntries, err := h.Chat.GetVisible()
	if err != nil {
		log.Err(err).Msg("error retrieving chat entries")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving chat entries"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, chatEntries)
}

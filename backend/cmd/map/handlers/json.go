package handlers

import (
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/teams"
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
	FlagEmoji       string   `json:"flag_emoji"`
	Active          bool     `json:"active"`
	SolvedByCurrent bool     `json:"solved_by_current"`
	Points          int      `json:"points"`
	HintPenalty     int      `json:"hint_penalty"`
	HelpPenalty     int      `json:"help_penalty"`
	Category        string   `json:"category"`
	Owner           string   `json:"owner"`
	Completed       []string `json:"completed"`
	Intro           string   `json:"intro"`
	LandPath        string   `json:"land_path"`
	LandClass       string   `json:"land_class"`
	LandStyle       string   `json:"land_style"`
	MarkerPath      string   `json:"marker_path"`
	MarkerClass     string   `json:"marker_class"`
	MarkerStyle     string   `json:"marker_style"`
	MarkerTransform string   `json:"marker_transform"`
}

type JSONWorldDominationResponse struct {
	CurrentTeam         string `json:"current_team"`
	CompletedChallenges int    `json:"completed_challenges"`
	TotalChallenges     int    `json:"total_challenges"`
	CompletionPct       int    `json:"completion_pct"`
	WinRatePct          int    `json:"win_rate_pct"`
	LoseRatePct         int    `json:"lose_rate_pct"`
}

func (h *HandlersMap) validatedJSONUUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		log.Err(errors.New("UUID is required")).Msg("UUID is required")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "UUID is required"})
		return "", false
	}
	if uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "invalid UUID"})
		return "", false
	}
	return uuid, true
}

// JSONActivityHandler to return all activity logs for a given UUID in JSON format
func (h *HandlersMap) JSONActivityHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
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

// JSONTeamsHandler to return all teams for a given UUID in JSON format
func (h *HandlersMap) JSONTeamsHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
		return
	}
	// Get all teams for the given UUID
	allTeams, err := h.Teams.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving teams")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving teams"})
		return
	}

	showTeamMembers, err := h.Settings.GetGameboardShowTeamMembers(uuid)
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
	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
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

	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
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
	activeChallengeIDs := make(map[uint]string, len(activeChallenges))
	for _, challenge := range activeChallenges {
		countryCode := strings.ToUpper(strings.TrimSpace(challenge.Country))
		if countryCode == "" {
			continue
		}
		if _, exists := activeChallengesByCode[countryCode]; exists {
			continue
		}
		activeChallengesByCode[countryCode] = challenge
		activeChallengeIDs[challenge.ID] = countryCode
	}

	completedByCountry := make(map[string][]string, len(activeChallengesByCode))
	ownerByCountry := make(map[string]string, len(activeChallengesByCode))
	solvedByCurrentCountry := make(map[string]bool, len(activeChallengesByCode))
	currentTeamID := uint(0)
	if h.Teams != nil {
		if h.Sessions != nil && h.Users != nil {
			username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
			if username != "" {
				user, err := h.Users.Get(username, uuid)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					log.Err(err).Msg("error retrieving current user for country data")
					HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
					return
				}
				currentTeamID = user.TeamID
			}
		}

		var allTeams []teams.PlatformTeam
		if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&allTeams).Error; err != nil {
			log.Err(err).Msg("error retrieving teams for country data")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
			return
		}

		teamNamesByID := make(map[uint]string, len(allTeams))
		for _, team := range allTeams {
			teamNamesByID[team.ID] = team.Name
		}

		var teamScores []teams.TeamScore
		if err := h.Teams.DB.Where("uuid = ?", uuid).Order("created_at ASC").Find(&teamScores).Error; err != nil {
			log.Err(err).Msg("error retrieving team scores for country data")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving country data"})
			return
		}

		seenCompletedByCountry := make(map[string]map[string]struct{}, len(activeChallengesByCode))
		for _, score := range teamScores {
			countryCode, ok := activeChallengeIDs[score.ChallengeID]
			if !ok {
				continue
			}

			teamName := strings.TrimSpace(teamNamesByID[score.TeamID])
			if teamName == "" {
				continue
			}

			if seenCompletedByCountry[countryCode] == nil {
				seenCompletedByCountry[countryCode] = make(map[string]struct{})
			}
			if _, seen := seenCompletedByCountry[countryCode][teamName]; !seen {
				completedByCountry[countryCode] = append(completedByCountry[countryCode], teamName)
				seenCompletedByCountry[countryCode][teamName] = struct{}{}
			}

			if currentTeamID != 0 && score.TeamID == currentTeamID {
				solvedByCurrentCountry[countryCode] = true
			}

			ownerByCountry[countryCode] = teamName
		}
	}

	response := make(map[string]JSONCountryDataResponse, len(allCountries))
	for _, country := range allCountries {
		countryCode := strings.ToUpper(strings.TrimSpace(country.CountryCode))
		challenge, hasChallenge := activeChallengesByCode[countryCode]
		completed := completedByCountry[countryCode]
		if completed == nil {
			completed = []string{}
		}

		categoryName := ""
		if hasChallenge {
			if category, exists := categoriesByID[challenge.CategoryID]; exists {
				categoryName = category.Name
			}
		}

		response[country.Name] = JSONCountryDataResponse{
			CountryCode:     country.CountryCode,
			FlagEmoji:       countryCodeToFlagEmoji(country.CountryCode),
			Active:          hasChallenge,
			SolvedByCurrent: solvedByCurrentCountry[countryCode],
			Points:          challenge.Points,
			HintPenalty:     challenge.HintPenalty,
			HelpPenalty:     challenge.HelpPenalty,
			Category:        categoryName,
			Owner:           ownerByCountry[countryCode],
			Completed:       completed,
			Intro:           challenge.Description,
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

// JSONWorldDominationHandler returns aggregate completion metrics for the authenticated user's team
func (h *HandlersMap) JSONWorldDominationHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
		return
	}

	if h.Challenges == nil || h.Teams == nil || h.Users == nil || h.Sessions == nil {
		log.Err(errors.New("world domination dependencies not initialized")).Msg("error retrieving world domination data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving world domination data"})
		return
	}

	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		log.Err(errors.New("user not authenticated")).Msg("user not authenticated")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: "user not authenticated"})
		return
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving user for world domination data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving world domination data"})
		return
	}

	activeChallenges, err := h.Challenges.GetActive(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving active challenges for world domination data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving world domination data"})
		return
	}

	response := JSONWorldDominationResponse{
		CompletedChallenges: 0,
		TotalChallenges:     len(activeChallenges),
		CompletionPct:       0,
		WinRatePct:          0,
		LoseRatePct:         0,
	}

	if user.TeamID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, response)
		return
	}

	var currentTeam teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", user.TeamID, uuid).First(&currentTeam).Error; err == nil {
		response.CurrentTeam = currentTeam.Name
	}

	activeChallengeIDs := make(map[uint]struct{}, len(activeChallenges))
	for _, challenge := range activeChallenges {
		activeChallengeIDs[challenge.ID] = struct{}{}
	}

	if len(activeChallengeIDs) == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, response)
		return
	}

	var scores []teams.TeamScore
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&scores).Error; err != nil {
		log.Err(err).Msg("error retrieving team scores for world domination data")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "error retrieving world domination data"})
		return
	}

	currentTeamCompleted := make(map[uint]struct{})
	currentTeamSolveCount := 0
	otherTeamsSolveCount := 0

	for _, score := range scores {
		if _, ok := activeChallengeIDs[score.ChallengeID]; !ok {
			continue
		}

		if score.TeamID == user.TeamID {
			currentTeamSolveCount++
			currentTeamCompleted[score.ChallengeID] = struct{}{}
			continue
		}

		otherTeamsSolveCount++
	}

	response.CompletedChallenges = len(currentTeamCompleted)
	response.CompletionPct = calculatePercentage(response.CompletedChallenges, response.TotalChallenges)

	totalSolveCount := currentTeamSolveCount + otherTeamsSolveCount
	response.WinRatePct = calculatePercentage(currentTeamSolveCount, totalSolveCount)
	response.LoseRatePct = calculatePercentage(otherTeamsSolveCount, totalSolveCount)

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, response)
}

func calculatePercentage(numerator, denominator int) int {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}

	return int(math.Round(float64(numerator) * 100 / float64(denominator)))
}

// JSONChatHandler to return all chat entries for a given UUID in JSON format
func (h *HandlersMap) JSONChatHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	_, ok := h.validatedJSONUUID(w, r)
	if !ok {
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

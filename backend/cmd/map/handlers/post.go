package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
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

func (h *HandlersMap) ScorePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	if h.Settings == nil || h.Users == nil || h.Teams == nil || h.Challenges == nil || h.Logs == nil || h.Sessions == nil {
		log.Err(errors.New("score handler dependencies not initialized")).Msg("error scoring challenge")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "scoring is unavailable"})
		return
	}

	var req MapScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing score request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "invalid request body"})
		return
	}

	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	submittedFlag := strings.TrimSpace(req.Flag)
	if countryCode == "" || submittedFlag == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "country_code and flag are required"})
		return
	}

	scoringEnabled, err := h.Settings.GetScoringEnabled()
	if err != nil {
		log.Err(err).Msg("error retrieving scoring setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to retrieve scoring setting"})
		return
	}
	if !scoringEnabled {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusForbidden, MapErrorResponse{Error: "scoring is disabled"})
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
		log.Err(err).Msg("error retrieving user for score request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to retrieve user"})
		return
	}
	if user.TeamID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: "user is not assigned to a team"})
		return
	}

	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", user.TeamID, uuid).First(&team).Error; err != nil {
		log.Err(err).Msg("error retrieving team for score request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to retrieve team"})
		return
	}

	var challenge challenges.Challenge
	if err := h.Challenges.DB.Where("uuid = ? AND active = ? AND country = ?", uuid, true, countryCode).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: "no active challenge found for country"})
			return
		}
		log.Err(err).Msg("error retrieving challenge for score request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to retrieve challenge"})
		return
	}

	var existingScore teams.TeamScore
	if err := h.Teams.DB.Where("uuid = ? AND team_id = ? AND challenge_id = ?", uuid, user.TeamID, challenge.ID).First(&existingScore).Error; err == nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusConflict, MapScoreResponse{
			Success:     false,
			Message:     "Your team already completed this challenge",
			CountryCode: countryCode,
			ChallengeID: challenge.ID,
			TotalPoints: team.Points,
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Err(err).Msg("error checking existing team score")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to evaluate current score"})
		return
	}

	if submittedFlag != strings.TrimSpace(challenge.Flag) {
		failureLog, err := h.Logs.NewFailuresLog(challenge.ID, user.TeamID, submittedFlag, uuid)
		if err != nil {
			log.Err(err).Msg("error building failure log")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to record scoring attempt"})
			return
		}
		if err := h.Logs.CreateFailuresLog(failureLog); err != nil {
			log.Err(err).Msg("error creating failure log")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to record scoring attempt"})
			return
		}

		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapScoreResponse{
			Success:     false,
			Message:     "Incorrect flag",
			CountryCode: countryCode,
			ChallengeID: challenge.ID,
			TotalPoints: team.Points,
		})
		return
	}

	awardedPoints := challenge.Points
	updatedTotalPoints := team.Points + awardedPoints
	now := time.Now().UTC()
	if err := h.Teams.DB.Transaction(func(tx *gorm.DB) error {
		score, err := h.Teams.NewScore(user.TeamID, challenge.ID, awardedPoints, uuid, username)
		if err != nil {
			return err
		}
		if err := tx.Create(&score).Error; err != nil {
			return err
		}

		if err := tx.Model(&teams.PlatformTeam{}).
			Where("id = ? AND uuid = ?", user.TeamID, uuid).
			Updates(map[string]any{
				"points":     gorm.Expr("points + ?", awardedPoints),
				"last_score": now,
			}).Error; err != nil {
			return err
		}

		var scoreboardCount int64
		if err := tx.Model(&logs.ScoreboardLog{}).Where("uuid = ?", uuid).Count(&scoreboardCount).Error; err != nil {
			return err
		}
		scoreboardLog, err := h.Logs.NewScoreboardLog(team.Name, updatedTotalPoints, int(scoreboardCount)+1, uuid)
		if err != nil {
			return err
		}
		if err := tx.Create(&scoreboardLog).Error; err != nil {
			return err
		}

		activity, err := h.Logs.NewActivity(team.Name, "completed", challenge.Title, challenge.ID, uuid)
		if err != nil {
			return err
		}
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Err(err).Msg("error storing successful score")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: "failed to score challenge"})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapScoreResponse{
		Success:       true,
		Message:       "Challenge completed",
		CountryCode:   countryCode,
		ChallengeID:   challenge.ID,
		PointsAwarded: awardedPoints,
		TotalPoints:   updatedTotalPoints,
	})
}

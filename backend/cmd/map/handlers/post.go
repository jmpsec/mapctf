package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
		log.Err(errors.New("invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	// Parse request body
	var req MapRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.invalid_body")})
		return
	}
	regType, err := h.Settings.GetRegistrationType(uuid)
	if err != nil {
		log.Err(err).Msg("error getting registration type setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("reg.failed_type_setting")})
		return
	}
	if req.Token == "" && regType == settings.TokenRegistration {
		log.Err(errors.New(h.T(r.Context())("reg.token_required"))).Msg(h.T(r.Context())("reg.token_required"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.token_required")})
		return
	}
	if regType == settings.TokenRegistration {
		regToken, err := h.Settings.GetRegistrationToken(uuid)
		if err != nil {
			log.Err(err).Msg("error getting registration token setting")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("reg.failed_token_setting")})
			return
		}
		if req.Token != regToken {
			log.Err(errors.New(h.T(r.Context())("reg.invalid_token"))).Msg(h.T(r.Context())("reg.invalid_token"))
			HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.invalid_token")})
			return
		}
	}
	if req.Username == "" || req.Password == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("auth.user_pass_required")})
		return
	}
	if req.Email == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.email_required")})
		return
	}
	if req.Team == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.team_required")})
		return
	}
	// Register team
	nTeam, err := h.Teams.Register(req.Team, req.Logo, uuid)
	if err != nil {
		log.Err(err).Msg("error registering team")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("reg.failed_register_team")})
		return
	}
	// Register user
	err = h.Users.Register(req.Username, req.Password, req.Name, req.Email, nTeam.ID, uuid)
	if err != nil {
		log.Err(err).Msg("error registering user")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("reg.failed_register_user")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapRegistrationResponse{
		Success:  true,
		Message:  h.T(r.Context())("reg.success"),
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
		log.Err(errors.New("invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	// Parse request body
	var req ChatEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.invalid_body")})
		return
	}
	chatMaxLen, err := h.Settings.GetGameboardChatMaxLen(uuid)
	if err != nil {
		log.Err(err).Msg("error getting gameboard chat max length setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("chat.failed_maxlen_setting")})
		return
	}
	// Get user from session
	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		log.Err(errors.New(h.T(r.Context())("chat.not_authenticated"))).Msg(h.T(r.Context())("chat.not_authenticated"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("chat.not_authenticated")})
		return
	}
	// Get user team
	teamID, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error getting user for chat message")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("chat.failed_get_user")})
		return
	}
	if err := h.Chat.CreateNew(username, req.Message, teamID.TeamID, chatMaxLen); err != nil {
		log.Err(err).Msg("error creating chat message")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("chat.failed_create")})
		return
	}
	h.invalidateFeed("chat", uuid)
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
		log.Err(errors.New("invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	if h.Settings == nil || h.Users == nil || h.Teams == nil || h.Challenges == nil || h.Logs == nil || h.Sessions == nil {
		log.Err(errors.New("score handler dependencies not initialized")).Msg("error scoring challenge")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.unavailable")})
		return
	}

	var req MapScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing score request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.invalid_body")})
		return
	}

	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	submittedFlag := strings.TrimSpace(req.Flag)
	if countryCode == "" || submittedFlag == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("score.country_flag_required")})
		return
	}

	scoringEnabled, err := h.Settings.GetScoringEnabled(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving scoring setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_setting")})
		return
	}
	if !scoringEnabled {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusForbidden, MapErrorResponse{Error: h.T(r.Context())("score.disabled")})
		return
	}

	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		log.Err(errors.New(h.T(r.Context())("chat.not_authenticated"))).Msg(h.T(r.Context())("chat.not_authenticated"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("chat.not_authenticated")})
		return
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving user for score request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_user")})
		return
	}
	if user.TeamID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("score.no_team")})
		return
	}

	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", user.TeamID, uuid).First(&team).Error; err != nil {
		log.Err(err).Msg("error retrieving team for score request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_team")})
		return
	}

	var challenge challenges.Challenge
	if err := h.Challenges.DB.Where("uuid = ? AND active = ? AND country = ?", uuid, true, countryCode).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: h.T(r.Context())("score.no_active_challenge")})
			return
		}
		log.Err(err).Msg("error retrieving challenge for score request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_challenge")})
		return
	}

	var existingScore teams.TeamScore
	if err := h.Teams.DB.Where("uuid = ? AND team_id = ? AND challenge_id = ?", uuid, user.TeamID, challenge.ID).First(&existingScore).Error; err == nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusConflict, MapScoreResponse{
			Success:     false,
			Message:     h.T(r.Context())("score.already_completed"),
			CountryCode: countryCode,
			ChallengeID: challenge.ID,
			TotalPoints: team.Points,
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Err(err).Msg("error checking existing team score")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_evaluate")})
		return
	}

	if submittedFlag != strings.TrimSpace(challenge.Flag) {
		failureLog, err := h.Logs.NewFailuresLog(challenge.ID, user.TeamID, submittedFlag, uuid)
		if err != nil {
			log.Err(err).Msg("error building failure log")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_record")})
			return
		}
		if err := h.Logs.CreateFailuresLog(failureLog); err != nil {
			log.Err(err).Msg("error creating failure log")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_record")})
			return
		}

		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapScoreResponse{
			Success:     false,
			Message:     h.T(r.Context())("score.incorrect_flag"),
			CountryCode: countryCode,
			ChallengeID: challenge.ID,
			TotalPoints: team.Points,
		})
		return
	}

	awardedPoints := challenge.Points
	updatedTotalPoints := team.Points + awardedPoints
	now := time.Now().UTC()
	categoryName := ""
	if challenge.CategoryID != 0 {
		category, err := h.Challenges.GetCategoryByID(challenge.CategoryID, uuid)
		if err != nil {
			log.Err(err).Msg("error retrieving challenge category for score activity")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_category")})
			return
		}
		categoryName = category.Name
	}

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
		activity, err := h.Logs.NewScoreActivity(team.Name, awardedPoints, countryCode, categoryName, challenge.ID, uuid)
		if err != nil {
			return err
		}
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Err(err).Msg("error storing successful score")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_score")})
		return
	}

	h.invalidateFeed("teams", uuid)
	h.invalidateFeed("activity", uuid)
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapScoreResponse{
		Success:       true,
		Message:       h.T(r.Context())("score.completed"),
		CountryCode:   countryCode,
		ChallengeID:   challenge.ID,
		PointsAwarded: awardedPoints,
		TotalPoints:   updatedTotalPoints,
	})
}

func (h *HandlersMap) HintPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	if h.Settings == nil || h.Users == nil || h.Teams == nil || h.Challenges == nil || h.Logs == nil || h.Sessions == nil {
		log.Err(errors.New("hint handler dependencies not initialized")).Msg("error requesting hint")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("hint.requests_unavailable")})
		return
	}

	var req MapHintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing hint request body")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("reg.invalid_body")})
		return
	}

	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if countryCode == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("hint.country_required")})
		return
	}

	scoringHintsEnabled, err := h.Settings.GetScoringHints(uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving scoring_hints setting")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("hint.failed_setting")})
		return
	}
	if !scoringHintsEnabled {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusForbidden, MapErrorResponse{Error: h.T(r.Context())("hint.requests_disabled")})
		return
	}

	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		log.Err(errors.New(h.T(r.Context())("chat.not_authenticated"))).Msg(h.T(r.Context())("chat.not_authenticated"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("chat.not_authenticated")})
		return
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving user for hint request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_user")})
		return
	}
	if user.TeamID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("score.no_team")})
		return
	}

	var challenge challenges.Challenge
	if err := h.Challenges.DB.Where("uuid = ? AND active = ? AND country = ?", uuid, true, countryCode).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: h.T(r.Context())("score.no_active_challenge")})
			return
		}
		log.Err(err).Msg("error retrieving challenge for hint request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_challenge")})
		return
	}

	if strings.TrimSpace(challenge.Hint) == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: h.T(r.Context())("hint.not_available")})
		return
	}

	penalty := challenge.HintPenalty
	if penalty < 0 {
		penalty = 0
	}

	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", user.TeamID, uuid).First(&team).Error; err != nil {
		log.Err(err).Msg("error retrieving team for hint request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_team")})
		return
	}

	var existingHint logs.HintsLog
	if err := h.Logs.DB.Where("uuid = ? AND team_id = ? AND challenge_id = ?", uuid, user.TeamID, challenge.ID).First(&existingHint).Error; err == nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapHintResponse{
			Success:         true,
			Message:         h.T(r.Context())("hint.already_unlocked"),
			CountryCode:     countryCode,
			ChallengeID:     challenge.ID,
			Hint:            challenge.Hint,
			Penalty:         existingHint.Penalty,
			TotalPoints:     team.Points,
			AlreadyUnlocked: true,
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Err(err).Msg("error checking existing hint log")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("hint.failed_evaluate")})
		return
	}

	if team.Points < penalty {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusConflict, MapHintResponse{
			Success:     false,
			Message:     h.T(r.Context())("hint.not_enough_points"),
			CountryCode: countryCode,
			ChallengeID: challenge.ID,
			Penalty:     penalty,
			TotalPoints: team.Points,
		})
		return
	}

	updatedTotalPoints := team.Points - penalty
	categoryName := ""
	if challenge.CategoryID != 0 {
		category, err := h.Challenges.GetCategoryByID(challenge.CategoryID, uuid)
		if err != nil {
			log.Err(err).Msg("error retrieving challenge category for hint activity")
			HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_category")})
			return
		}
		categoryName = category.Name
	}

	if err := h.Teams.DB.Transaction(func(tx *gorm.DB) error {
		var currentTeam teams.PlatformTeam
		if err := tx.Where("id = ? AND uuid = ?", user.TeamID, uuid).First(&currentTeam).Error; err != nil {
			return err
		}
		if currentTeam.Points < penalty {
			return fmt.Errorf("insufficient points")
		}

		hintLog, err := h.Logs.NewHintsLog(challenge.ID, user.TeamID, penalty, uuid)
		if err != nil {
			return err
		}
		if err := tx.Create(&hintLog).Error; err != nil {
			return err
		}

		if err := tx.Model(&teams.PlatformTeam{}).
			Where("id = ? AND uuid = ?", user.TeamID, uuid).
			Updates(map[string]any{
				"points": gorm.Expr("points - ?", penalty),
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

		activityMessage := fmt.Sprintf("Team %s spent %d points for a hint on challenge %s (%s)", team.Name, penalty, countryCode, categoryName)
		activity, err := h.Logs.NewActivity(true, team.Name, "hint", activityMessage, challenge.ID, uuid)
		if err != nil {
			return err
		}
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		if err.Error() == "insufficient points" {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusConflict, MapHintResponse{
				Success:     false,
				Message:     h.T(r.Context())("hint.not_enough_points"),
				CountryCode: countryCode,
				ChallengeID: challenge.ID,
				Penalty:     penalty,
				TotalPoints: team.Points,
			})
			return
		}
		log.Err(err).Msg("error storing successful hint request")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("hint.failed_unlock")})
		return
	}

	h.invalidateFeed("teams", uuid)
	h.invalidateFeed("activity", uuid)
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapHintResponse{
		Success:         true,
		Message:         h.T(r.Context())("hint.unlocked"),
		CountryCode:     countryCode,
		ChallengeID:     challenge.ID,
		Hint:            challenge.Hint,
		Penalty:         penalty,
		TotalPoints:     updatedTotalPoints,
		AlreadyUnlocked: false,
	})
}

func (h *HandlersMap) HintGETHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	if h.Users == nil || h.Teams == nil || h.Challenges == nil || h.Logs == nil || h.Sessions == nil {
		log.Err(errors.New("hint get handler dependencies not initialized")).Msg("error retrieving unlocked hint")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("hint.retrieval_unavailable")})
		return
	}

	countryCode := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country_code")))
	if countryCode == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("hint.country_required")})
		return
	}

	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		log.Err(errors.New(h.T(r.Context())("chat.not_authenticated"))).Msg(h.T(r.Context())("chat.not_authenticated"))
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("chat.not_authenticated")})
		return
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving user for hint retrieval")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_user")})
		return
	}
	if user.TeamID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("score.no_team")})
		return
	}

	var challenge challenges.Challenge
	if err := h.Challenges.DB.Where("uuid = ? AND active = ? AND country = ?", uuid, true, countryCode).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: h.T(r.Context())("score.no_active_challenge")})
			return
		}
		log.Err(err).Msg("error retrieving challenge for hint retrieval")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_challenge")})
		return
	}

	if strings.TrimSpace(challenge.Hint) == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: h.T(r.Context())("hint.not_available")})
		return
	}

	var existingHint logs.HintsLog
	if err := h.Logs.DB.Where("uuid = ? AND team_id = ? AND challenge_id = ?", uuid, user.TeamID, challenge.ID).First(&existingHint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusNotFound, MapErrorResponse{Error: h.T(r.Context())("hint.not_unlocked")})
			return
		}
		log.Err(err).Msg("error checking unlocked hint state")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("hint.failed_evaluate")})
		return
	}

	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", user.TeamID, uuid).First(&team).Error; err != nil {
		log.Err(err).Msg("error retrieving team for hint retrieval")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("score.failed_retrieve_team")})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapHintResponse{
		Success:         true,
		Message:         h.T(r.Context())("hint.already_unlocked"),
		CountryCode:     countryCode,
		ChallengeID:     challenge.ID,
		Hint:            challenge.Hint,
		Penalty:         existingHint.Penalty,
		TotalPoints:     team.Points,
		AlreadyUnlocked: true,
	})
}

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type adminActionResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func wantsJSONResponse(r *http.Request) bool {
	return strings.Contains(r.Header.Get(ContentType), JSONApplication) ||
		strings.Contains(r.Header.Get("Accept"), JSONApplication) ||
		strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest")
}

// AdminTemplateHandler for admin dashboard page for GET requests
func (h *HandlersMap) AdminTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/admin/index.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminTemplateData{
		Title:         "Admin",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminSettingsTemplateHandler for admin settings page for GET requests
func (h *HandlersMap) AdminSettingsTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/admin/settings.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminSettingsTemplateData{
		Title:         "Admin",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}

	loginEnabled, err := h.Settings.GetLoginEnabled()
	if err == nil {
		templateData.LoginEnabled = loginEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading login_enabled")
	}

	registrationEnabled, err := h.Settings.GetRegistrationEnabled()
	if err == nil {
		templateData.RegistrationEnabled = registrationEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_enabled")
	}

	scoringEnabled, err := h.Settings.GetScoringEnabled()
	if err == nil {
		templateData.ScoringEnabled = scoringEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading scoring_enabled")
	}

	gamePaused, err := h.Settings.GetGamePaused()
	if err == nil {
		templateData.GamePaused = gamePaused
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_paused")
	}

	gameStarted, err := h.Settings.GetGameStarted()
	if err == nil {
		templateData.GameStarted = gameStarted
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_started")
	}

	customOrg, err := h.Settings.GetCustomOrg()
	if err == nil {
		templateData.CustomOrg = customOrg
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading custom_org")
	}

	language, err := h.Settings.GetLanguage()
	if err == nil {
		templateData.Language = language
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading language")
	}

	gameStartTime, err := h.Settings.GetGameStartTime()
	if err == nil && !gameStartTime.IsZero() {
		templateData.GameStartTime = gameStartTime.Format("2006-01-02T15:04")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_start_time")
	}

	gameEndTime, err := h.Settings.GetGameEndTime()
	if err == nil && !gameEndTime.IsZero() {
		templateData.GameEndTime = gameEndTime.Format("2006-01-02T15:04")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_end_time")
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminSettingsPOSTHandler for admin settings page for POST requests
func (h *HandlersMap) AdminSettingsPOSTHandler(w http.ResponseWriter, r *http.Request) {
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

	jsonResponse := wantsJSONResponse(r)
	redirectBase := "/" + uuid + "/admin/settings"
	writeError := func(code int, msg string) {
		if jsonResponse {
			HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
				Success: false,
				Status:  "error",
				Message: msg,
			})
			return
		}
		http.Redirect(w, r, redirectBase+"?status=error&msg="+url.QueryEscape(msg), http.StatusFound)
	}
	writeSuccess := func(msg string) {
		if jsonResponse {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
				Success: true,
				Status:  "ok",
				Message: msg,
			})
			return
		}
		http.Redirect(w, r, redirectBase+"?status=ok&msg="+url.QueryEscape(msg), http.StatusFound)
	}

	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		username = h.ServiceName
	}

	var req AdminSettingsRequest
	if strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Err(err).Msg("error parsing admin settings JSON payload")
			writeError(http.StatusBadRequest, "Invalid JSON payload")
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Err(err).Msg("error parsing admin settings form")
			writeError(http.StatusBadRequest, "Invalid form payload")
			return
		}
		req.SettingName = r.FormValue("setting_name")
		req.SettingValue = r.FormValue("setting_value")
	}

	settingName := strings.TrimSpace(req.SettingName)
	if settingName == "" {
		settingName = strings.TrimSpace(req.Name)
	}
	settingValue := strings.TrimSpace(req.SettingValue)
	if settingValue == "" {
		settingValue = strings.TrimSpace(req.Value)
	}
	if settingName == "" {
		writeError(http.StatusBadRequest, "Missing setting_name")
		return
	}

	setBoolSetting := func(setter func(bool, string) error, setting string) bool {
		parsed, err := strconv.ParseBool(strings.ToLower(settingValue))
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid boolean for "+setting)
			return false
		}
		if err := setter(parsed, username); err != nil {
			log.Err(err).Msgf("error updating %s", setting)
			writeError(http.StatusInternalServerError, "Failed to update "+setting)
			return false
		}
		return true
	}

	switch settingName {
	case "login_enabled":
		if !setBoolSetting(h.Settings.SetLoginEnabled, settingName) {
			return
		}
	case "registration_enabled":
		if !setBoolSetting(h.Settings.SetRegistrationEnabled, settingName) {
			return
		}
	case "scoring_enabled":
		if !setBoolSetting(h.Settings.SetScoringEnabled, settingName) {
			return
		}
	case "game_paused":
		if !setBoolSetting(h.Settings.SetGamePaused, settingName) {
			return
		}
	case "game_started":
		if !setBoolSetting(h.Settings.SetGameStarted, settingName) {
			return
		}
	case "custom_org":
		if err := h.Settings.SetCustomOrg(settingValue, username); err != nil {
			log.Err(err).Msg("error updating custom_org")
			writeError(http.StatusInternalServerError, "Failed to update custom_org")
			return
		}
	case "language":
		if err := h.Settings.SetLanguage(settingValue, username); err != nil {
			log.Err(err).Msg("error updating language")
			writeError(http.StatusInternalServerError, "Failed to update language")
			return
		}
	case "game_start_time":
		gameStartTime, err := time.ParseInLocation("2006-01-02T15:04", settingValue, time.Local)
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid game_start_time format")
			return
		}
		if err := h.Settings.SetGameStartTime(gameStartTime, username); err != nil {
			log.Err(err).Msg("error updating game_start_time")
			writeError(http.StatusInternalServerError, "Failed to update game_start_time")
			return
		}
	case "game_end_time":
		gameEndTime, err := time.ParseInLocation("2006-01-02T15:04", settingValue, time.Local)
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid game_end_time format")
			return
		}
		if err := h.Settings.SetGameEndTime(gameEndTime, username); err != nil {
			log.Err(err).Msg("error updating game_end_time")
			writeError(http.StatusInternalServerError, "Failed to update game_end_time")
			return
		}
	default:
		writeError(http.StatusBadRequest, "Unsupported setting")
		return
	}

	writeSuccess("Updated " + settingName)
}

// AdminControlsTemplateHandler for admin controls page for GET requests
func (h *HandlersMap) AdminControlsTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/admin/controls.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminControlsTemplateData{
		Title:         "Admin",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminTeamsTemplateHandler for admin teams page for GET requests
func (h *HandlersMap) AdminTeamsTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/admin/teams.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminTeamsTemplateData{
		Title:         "Admin",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	teams, err := h.Teams.GetAll(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading teams")
	} else {
		templateData.Teams = teams
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminUsersTemplateHandler for admin users page for GET requests
func (h *HandlersMap) AdminUsersTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/admin/users.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminUsersTemplateData{
		Title:         "Admin",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	users, err := h.Users.GetAll(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading users")
	} else {
		templateData.Users = users
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminChallengesTemplateHandler for admin challenges page for GET requests
func (h *HandlersMap) AdminChallengesTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/admin/challenges.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminChallengesTemplateData{
		Title:         "Admin",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	challenges, err := h.Challenges.GetAll(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading challenges")
	} else {
		templateData.Challenges = challenges
	}
	categories, err := h.Challenges.GetAllCategories(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading categories")
	} else {
		templateData.Categories = categories
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminChallengesPOSTHandler for admin challenges page for POST requests
func (h *HandlersMap) AdminChallengesPOSTHandler(w http.ResponseWriter, r *http.Request) {
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

	jsonResponse := wantsJSONResponse(r)
	redirectBase := "/" + uuid + "/admin/challenges"
	writeError := func(code int, msg string) {
		if jsonResponse {
			HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
				Success: false,
				Status:  "error",
				Message: msg,
			})
			return
		}
		http.Redirect(w, r, redirectBase+"?status=error&msg="+url.QueryEscape(msg), http.StatusFound)
	}
	writeSuccess := func(msg string) {
		if jsonResponse {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
				Success: true,
				Status:  "ok",
				Message: msg,
			})
			return
		}
		http.Redirect(w, r, redirectBase+"?status=ok&msg="+url.QueryEscape(msg), http.StatusFound)
	}

	var req AdminChallengeCreateRequest
	if strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Err(err).Msg("error parsing admin challenges JSON payload")
			writeError(http.StatusBadRequest, "Invalid JSON payload")
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Err(err).Msg("error parsing admin challenges form")
			writeError(http.StatusBadRequest, "Invalid form payload")
			return
		}
		req.Title = r.FormValue("title")
		req.Description = r.FormValue("description")
		req.CategoryID = r.FormValue("category_id")
		req.Active = r.FormValue("active")
		req.Points = r.FormValue("points")
		req.Bonus = r.FormValue("bonus")
		req.BonusDecay = r.FormValue("bonus_decay")
		req.Penalty = r.FormValue("penalty")
		req.Flag = r.FormValue("flag")
		req.Hint = r.FormValue("hint")
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	categoryIDStr := strings.TrimSpace(req.CategoryID)
	activeStr := strings.TrimSpace(req.Active)
	pointsStr := strings.TrimSpace(req.Points)
	bonusStr := strings.TrimSpace(req.Bonus)
	bonusDecayStr := strings.TrimSpace(req.BonusDecay)
	penaltyStr := strings.TrimSpace(req.Penalty)
	flag := strings.TrimSpace(req.Flag)
	hint := strings.TrimSpace(req.Hint)

	if title == "" || flag == "" {
		writeError(http.StatusBadRequest, "Title and flag are required")
		return
	}
	categoryID, _ := strconv.ParseUint(categoryIDStr, 10, 64)
	active, _ := strconv.ParseBool(activeStr)
	points, _ := strconv.ParseInt(pointsStr, 10, 64)
	bonus, _ := strconv.ParseInt(bonusStr, 10, 64)
	bonusDecay, _ := strconv.ParseInt(bonusDecayStr, 10, 64)
	penalty, _ := strconv.ParseInt(penaltyStr, 10, 64)

	challenge := h.Challenges.New(
		title,
		description,
		uint(categoryID),
		active,
		int(points),
		int(bonus),
		int(bonusDecay),
		int(penalty),
		flag,
		hint,
		uuid,
	)

	if err := h.Challenges.Create(challenge); err != nil {
		log.Err(err).Msg("error creating challenge")
		writeError(http.StatusInternalServerError, "Failed to create challenge")
		return
	}

	writeSuccess("Challenge created")
}

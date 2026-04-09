package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type adminActionResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type adminChallengesTransferPayload struct {
	Version    int                               `json:"version"`
	ExportedAt string                            `json:"exported_at"`
	Categories []adminChallengesTransferCategory `json:"categories"`
	Challenges []adminChallengesTransferItem     `json:"challenges"`
}

type adminChallengesTransferCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
}

type adminChallengesTransferItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Country     string `json:"country"`
	Active      bool   `json:"active"`
	Points      int    `json:"points"`
	Bonus       int    `json:"bonus"`
	BonusDecay  int    `json:"bonus_decay"`
	Penalty     int    `json:"penalty"`
	Flag        string `json:"flag"`
	Hint        string `json:"hint"`
}

type adminTeamsTransferPayload struct {
	Version    int                      `json:"version"`
	ExportedAt string                   `json:"exported_at"`
	Logos      []adminTeamsTransferLogo `json:"logos"`
	Teams      []adminTeamsTransferTeam `json:"teams"`
}

type adminTeamsTransferLogo struct {
	Name      string `json:"name"`
	Logo      string `json:"logo"`
	Enabled   bool   `json:"enabled"`
	Custom    bool   `json:"custom"`
	Protected bool   `json:"protected"`
	Used      bool   `json:"used"`
}

type adminTeamsTransferTeam struct {
	Name      string `json:"name"`
	Logo      string `json:"logo"`
	Active    bool   `json:"active"`
	Visible   bool   `json:"visible"`
	Protected bool   `json:"protected"`
}

type adminUsersTransferPayload struct {
	Version    int                      `json:"version"`
	ExportedAt string                   `json:"exported_at"`
	Users      []adminUsersTransferUser `json:"users"`
}

type adminUsersTransferUser struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	TeamID   uint   `json:"team_id"`
	Admin    bool   `json:"admin"`
	Service  bool   `json:"service"`
	Active   bool   `json:"active"`
	PassHash string `json:"pass_hash,omitempty"`
}

type adminSettingsTransferPayload struct {
	Version    int                         `json:"version"`
	ExportedAt string                      `json:"exported_at"`
	Settings   []adminSettingsTransferItem `json:"settings"`
}

type adminSettingsTransferItem struct {
	Name        string  `json:"name"`
	ValueType   string  `json:"value_type"`
	ValueString string  `json:"value_string,omitempty"`
	ValueInt    int     `json:"value_int,omitempty"`
	ValueBool   bool    `json:"value_bool,omitempty"`
	ValueFloat  float64 `json:"value_float,omitempty"`
	ValueDate   string  `json:"value_date,omitempty"`
}

const maxCustomLogoUploadBytes int64 = 512 * 1024

var (
	logoSlugCleaner     = regexp.MustCompile(`[^a-z0-9-]+`)
	logoSlugMultiDash   = regexp.MustCompile(`-+`)
	svgScriptTagPattern = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	svgViewBoxPatternD  = regexp.MustCompile(`(?i)viewBox\s*=\s*"([^"]+)"`)
	svgViewBoxPatternS  = regexp.MustCompile(`(?i)viewBox\s*=\s*'([^']+)'`)
)

func countryCodeToFlagEmoji(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 {
		return ""
	}
	first := code[0]
	second := code[1]
	if first < 'A' || first > 'Z' || second < 'A' || second > 'Z' {
		return ""
	}
	return string([]rune{
		rune(first-'A') + 0x1F1E6,
		rune(second-'A') + 0x1F1E6,
	})
}

func normalizeLogoSymbolName(logo string) string {
	logo = strings.TrimSpace(logo)
	if logo == "" {
		return "invader"
	}

	logo = strings.TrimPrefix(logo, "#icon--badge-")
	logo = strings.TrimPrefix(logo, "icon--badge-")

	if idx := strings.LastIndex(logo, "/"); idx >= 0 && idx < len(logo)-1 {
		logo = logo[idx+1:]
	}

	logo = strings.TrimSuffix(logo, ".svg")
	logo = strings.TrimPrefix(logo, "badge-")

	if logo == "" {
		return "invader"
	}
	return logo
}

func sanitizeLogoSlug(raw string) string {
	slug := strings.ToLower(strings.TrimSpace(raw))
	slug = strings.TrimPrefix(slug, "badge-")
	slug = strings.TrimSuffix(slug, ".svg")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = logoSlugCleaner.ReplaceAllString(slug, "-")
	slug = logoSlugMultiDash.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func buildUploadedLogoSymbol(slug string, svgData []byte) (string, error) {
	source := strings.TrimSpace(string(svgData))
	if source == "" {
		return "", fmt.Errorf("empty svg file")
	}

	lower := strings.ToLower(source)
	start := strings.Index(lower, "<svg")
	if start < 0 {
		return "", fmt.Errorf("invalid svg file")
	}

	openEndOffset := strings.Index(lower[start:], ">")
	if openEndOffset < 0 {
		return "", fmt.Errorf("invalid svg file")
	}
	openEnd := start + openEndOffset
	closeIdx := strings.LastIndex(lower, "</svg>")
	if closeIdx <= openEnd {
		return "", fmt.Errorf("invalid svg file")
	}

	openTag := source[start : openEnd+1]
	inner := strings.TrimSpace(source[openEnd+1 : closeIdx])
	inner = svgScriptTagPattern.ReplaceAllString(inner, "")
	if inner == "" {
		return "", fmt.Errorf("svg has no drawable content")
	}

	viewBox := "0 0 64 48"
	if match := svgViewBoxPatternD.FindStringSubmatch(openTag); len(match) > 1 {
		viewBox = strings.TrimSpace(match[1])
	} else if match := svgViewBoxPatternS.FindStringSubmatch(openTag); len(match) > 1 {
		viewBox = strings.TrimSpace(match[1])
	}

	symbolID := "icon--badge-" + slug
	return fmt.Sprintf(`<symbol id="%s" viewBox="%s">%s</symbol>`, symbolID, viewBox, inner), nil
}

func appendSymbolToSprite(spritePath, symbolID, symbolMarkup string) error {
	data, err := os.ReadFile(spritePath)
	if err != nil {
		return fmt.Errorf("failed to read sprite file: %w", err)
	}

	content := string(data)
	if strings.Contains(content, `id="`+symbolID+`"`) {
		return fmt.Errorf("logo symbol already exists")
	}

	closeIdx := strings.LastIndex(strings.ToLower(content), "</svg>")
	if closeIdx < 0 {
		return fmt.Errorf("invalid sprite file")
	}

	updated := content[:closeIdx] + "\n" + symbolMarkup + "\n" + content[closeIdx:]
	if err := os.WriteFile(spritePath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("failed to update sprite file: %w", err)
	}
	return nil
}

func saveUploadedLogoFile(staticDir, slug string, svgData []byte) error {
	customDir := filepath.Join(staticDir, "svg", "icons", "custom")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		return fmt.Errorf("failed to create custom logo dir: %w", err)
	}
	outputPath := filepath.Join(customDir, "badge-"+slug+".svg")
	if err := os.WriteFile(outputPath, svgData, 0o644); err != nil {
		return fmt.Errorf("failed to save custom logo file: %w", err)
	}
	return nil
}

func wantsJSONResponse(r *http.Request) bool {
	return strings.Contains(r.Header.Get(ContentType), JSONApplication) ||
		strings.Contains(r.Header.Get("Accept"), JSONApplication) ||
		strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest")
}

func (h *HandlersMap) decodeChallengeImportPayload(r *http.Request, payload *adminChallengesTransferPayload) error {
	if strings.Contains(r.Header.Get(ContentType), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			return err
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			return err
		}
		defer file.Close()
		return json.NewDecoder(file).Decode(payload)
	}
	return json.NewDecoder(r.Body).Decode(payload)
}

func (h *HandlersMap) decodeTeamsImportPayload(r *http.Request, payload *adminTeamsTransferPayload) error {
	if strings.Contains(r.Header.Get(ContentType), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			return err
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			return err
		}
		defer file.Close()
		return json.NewDecoder(file).Decode(payload)
	}
	return json.NewDecoder(r.Body).Decode(payload)
}

func (h *HandlersMap) decodeUsersImportPayload(r *http.Request, payload *adminUsersTransferPayload) error {
	if strings.Contains(r.Header.Get(ContentType), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			return err
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			return err
		}
		defer file.Close()
		return json.NewDecoder(file).Decode(payload)
	}
	return json.NewDecoder(r.Body).Decode(payload)
}

func (h *HandlersMap) decodeSettingsImportPayload(r *http.Request, payload *adminSettingsTransferPayload) error {
	if strings.Contains(r.Header.Get(ContentType), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			return err
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			return err
		}
		defer file.Close()
		return json.NewDecoder(file).Decode(payload)
	}
	return json.NewDecoder(r.Body).Decode(payload)
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
		Title:         "MapCTF Admin: Dashboard",
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
		Title:         "MapCTF Admin: Settings",
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

	loginStrongPasswords, err := h.Settings.GetLoginStrongPasswords()
	if err == nil {
		templateData.LoginStrongPasswords = loginStrongPasswords
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading login_strong_passwords")
	}

	registrationEnabled, err := h.Settings.GetRegistrationEnabled()
	if err == nil {
		templateData.RegistrationEnabled = registrationEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_enabled")
	}

	registrationNames, err := h.Settings.GetRegistrationNames()
	if err == nil {
		templateData.RegistrationNames = registrationNames
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_names")
	}

	registrationEmails, err := h.Settings.GetRegistrationEmails()
	if err == nil {
		templateData.RegistrationEmails = registrationEmails
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_emails")
	}

	registrationType, err := h.Settings.GetRegistrationType()
	if err == nil {
		templateData.RegistrationType = registrationType
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_type")
	}

	registrationToken, err := h.Settings.GetRegistrationToken()
	if err == nil {
		templateData.RegistrationToken = registrationToken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_token")
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

	customLogo, err := h.Settings.GetCustomLogo()
	if err == nil {
		templateData.CustomLogo = customLogo
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading custom_logo")
	}

	language, err := h.Settings.GetLanguage()
	if err == nil {
		templateData.Language = language
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading language")
	}

	leaderboardLimit, err := h.Settings.GetLeaderboardLimit()
	if err == nil {
		templateData.LeaderboardLimit = leaderboardLimit
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading leaderboard_limit")
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
	setIntSetting := func(setter func(int, string) error, setting string) bool {
		parsed, err := strconv.Atoi(settingValue)
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid integer for "+setting)
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
	case "login_strong_passwords":
		if !setBoolSetting(h.Settings.SetLoginStrongPasswords, settingName) {
			return
		}
	case "registration_enabled":
		if !setBoolSetting(h.Settings.SetRegistrationEnabled, settingName) {
			return
		}
	case "registration_names":
		if !setBoolSetting(h.Settings.SetRegistrationNames, settingName) {
			return
		}
	case "registration_emails":
		if !setBoolSetting(h.Settings.SetRegistrationEmails, settingName) {
			return
		}
	case "registration_type":
		if !setIntSetting(h.Settings.SetRegistrationType, settingName) {
			return
		}
	case "registration_token":
		if err := h.Settings.SetRegistrationToken(settingValue, username); err != nil {
			log.Err(err).Msg("error updating registration_token")
			writeError(http.StatusInternalServerError, "Failed to update registration_token")
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
	case "custom_logo":
		if err := h.Settings.SetCustomLogo(settingValue, username); err != nil {
			log.Err(err).Msg("error updating custom_logo")
			writeError(http.StatusInternalServerError, "Failed to update custom_logo")
			return
		}
	case "language":
		if err := h.Settings.SetLanguage(settingValue, username); err != nil {
			log.Err(err).Msg("error updating language")
			writeError(http.StatusInternalServerError, "Failed to update language")
			return
		}
	case "leaderboard_limit":
		if !setIntSetting(h.Settings.SetLeaderboardLimit, settingName) {
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

// AdminSettingsExportHandler exports settings as JSON
func (h *HandlersMap) AdminSettingsExportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	settingsList, err := h.Settings.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error loading settings for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load settings",
		})
		return
	}

	sort.Slice(settingsList, func(i, j int) bool {
		return settingsList[i].Name < settingsList[j].Name
	})

	payload := adminSettingsTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Settings:   make([]adminSettingsTransferItem, 0, len(settingsList)),
	}
	for _, s := range settingsList {
		item := adminSettingsTransferItem{
			Name:        s.Name,
			ValueType:   s.ValueType,
			ValueString: s.ValueString,
			ValueInt:    s.ValueInt,
			ValueBool:   s.ValueBool,
			ValueFloat:  s.ValueFloat,
		}
		if !s.ValueDate.IsZero() {
			item.ValueDate = s.ValueDate.UTC().Format(time.RFC3339)
		}
		payload.Settings = append(payload.Settings, item)
	}

	filename := "mapctf-settings-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, payload)
}

// AdminSettingsImportHandler imports settings from JSON payload/file
func (h *HandlersMap) AdminSettingsImportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	contentType := strings.ToLower(r.Header.Get(ContentType))
	isJSON := strings.Contains(contentType, JSONApplication)
	isMultipart := strings.Contains(contentType, "multipart/form-data")
	if !isJSON && !isMultipart {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json or multipart/form-data")
		return
	}

	var payload adminSettingsTransferPayload
	if err := h.decodeSettingsImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing settings import payload")
		writeError(http.StatusBadRequest, "Invalid settings import payload")
		return
	}

	username := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if username == "" {
		username = h.ServiceName
	}

	updatedSettings := 0
	skippedSettings := 0

	for _, in := range payload.Settings {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			skippedSettings++
			continue
		}

		var err error
		switch name {
		case "login_enabled":
			err = h.Settings.SetLoginEnabled(in.ValueBool, username)
		case "login_strong_passwords":
			err = h.Settings.SetLoginStrongPasswords(in.ValueBool, username)
		case "registration_enabled":
			err = h.Settings.SetRegistrationEnabled(in.ValueBool, username)
		case "registration_names":
			err = h.Settings.SetRegistrationNames(in.ValueBool, username)
		case "registration_emails":
			err = h.Settings.SetRegistrationEmails(in.ValueBool, username)
		case "registration_type":
			err = h.Settings.SetRegistrationType(in.ValueInt, username)
		case "registration_token":
			err = h.Settings.SetRegistrationToken(in.ValueString, username)
		case "scoring_enabled":
			err = h.Settings.SetScoringEnabled(in.ValueBool, username)
		case "game_paused":
			err = h.Settings.SetGamePaused(in.ValueBool, username)
		case "game_started":
			err = h.Settings.SetGameStarted(in.ValueBool, username)
		case "game_start_time":
			var t time.Time
			if strings.TrimSpace(in.ValueDate) != "" {
				t, err = time.Parse(time.RFC3339, strings.TrimSpace(in.ValueDate))
				if err != nil {
					t, err = time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(in.ValueDate), time.Local)
				}
				if err != nil {
					writeError(http.StatusBadRequest, "Invalid game_start_time format in import")
					return
				}
			}
			err = h.Settings.SetGameStartTime(t, username)
		case "game_end_time":
			var t time.Time
			if strings.TrimSpace(in.ValueDate) != "" {
				t, err = time.Parse(time.RFC3339, strings.TrimSpace(in.ValueDate))
				if err != nil {
					t, err = time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(in.ValueDate), time.Local)
				}
				if err != nil {
					writeError(http.StatusBadRequest, "Invalid game_end_time format in import")
					return
				}
			}
			err = h.Settings.SetGameEndTime(t, username)
		case "custom_org":
			err = h.Settings.SetCustomOrg(in.ValueString, username)
		case "custom_logo":
			err = h.Settings.SetCustomLogo(in.ValueString, username)
		case "language":
			err = h.Settings.SetLanguage(in.ValueString, username)
		case "leaderboard_limit":
			err = h.Settings.SetLeaderboardLimit(in.ValueInt, username)
		default:
			skippedSettings++
			continue
		}

		if err != nil {
			log.Err(err).Msgf("error importing setting %s", name)
			writeError(http.StatusInternalServerError, "Failed importing setting "+name)
			return
		}
		updatedSettings++
	}

	messageParts := []string{"settings updated " + strconv.Itoa(updatedSettings)}
	if skippedSettings > 0 {
		messageParts = append(messageParts, "settings skipped "+strconv.Itoa(skippedSettings))
	}
	writeSuccess("Import complete: " + strings.Join(messageParts, ", "))
}

// AdminSettingsResetDefaultsPOSTHandler resets settings to default values
func (h *HandlersMap) AdminSettingsResetDefaultsPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	username := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if username == "" {
		username = h.ServiceName
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	defaultStartTime := time.Time{}
	defaultEndTime := time.Time{}
	updates := []func() error{
		func() error { return h.Settings.SetLoginEnabled(false, username) },
		func() error { return h.Settings.SetLoginStrongPasswords(false, username) },
		func() error { return h.Settings.SetRegistrationEnabled(false, username) },
		func() error { return h.Settings.SetRegistrationNames(false, username) },
		func() error { return h.Settings.SetRegistrationEmails(false, username) },
		func() error { return h.Settings.SetRegistrationType(0, username) },
		func() error { return h.Settings.SetRegistrationToken("", username) },
		func() error { return h.Settings.SetScoringEnabled(false, username) },
		func() error { return h.Settings.SetGamePaused(false, username) },
		func() error { return h.Settings.SetGameStarted(false, username) },
		func() error { return h.Settings.SetGameStartTime(defaultStartTime, username) },
		func() error { return h.Settings.SetGameEndTime(defaultEndTime, username) },
		func() error { return h.Settings.SetCustomOrg("", username) },
		func() error { return h.Settings.SetCustomLogo("", username) },
		func() error { return h.Settings.SetLanguage("en", username) },
		func() error { return h.Settings.SetLeaderboardLimit(10, username) },
	}

	for _, update := range updates {
		if err := update(); err != nil {
			log.Err(err).Msg("error resetting settings to defaults")
			writeError(http.StatusInternalServerError, "Failed resetting settings to defaults")
			return
		}
	}

	writeSuccess("Settings reset to defaults")
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
		Title:         "MapCTF Admin: Controls",
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
		Title:         "MapCTF Admin: Teams",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	teamList, err := h.Teams.GetAll()
	if err != nil {
		log.Warn().Err(err).Msg("error loading teams")
	} else {
		for i := range teamList {
			teamList[i].Logo = normalizeLogoSymbolName(teamList[i].Logo)
		}
		templateData.Teams = teamList
	}
	var logos []teams.TeamLogo
	if err := h.Teams.DB.Where("enabled = ? AND uuid = ?", true, uuid).Order("name ASC").Find(&logos).Error; err != nil {
		log.Warn().Err(err).Msg("error loading team logos")
	} else {
		for i := range logos {
			logos[i].Logo = normalizeLogoSymbolName(logos[i].Logo)
		}
		templateData.Logos = logos
	}
	var allLogos []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ?", uuid).Order("name ASC").Find(&allLogos).Error; err != nil {
		log.Warn().Err(err).Msg("error loading all team logos")
	} else {
		for i := range allLogos {
			allLogos[i].Logo = normalizeLogoSymbolName(allLogos[i].Logo)
		}
		templateData.AllLogos = allLogos
	}
	teamUsers, err := h.Users.GetAll(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading users for teams view")
	} else {
		templateData.Users = teamUsers
		templateData.TeamMembers = make(map[uint][]users.PlatformUser)
		for _, user := range teamUsers {
			templateData.TeamMembers[user.TeamID] = append(templateData.TeamMembers[user.TeamID], user)
		}
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminTeamsPOSTHandler for admin teams page for POST requests
func (h *HandlersMap) AdminTeamsPOSTHandler(w http.ResponseWriter, r *http.Request) {
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
	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}
	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	var req AdminTeamCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin teams JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	name := strings.TrimSpace(req.Name)
	logo := strings.TrimSpace(req.Logo)
	if name == "" {
		writeError(http.StatusBadRequest, "Team name is required")
		return
	}
	if _, err := h.Teams.Register(name, logo); err != nil {
		log.Err(err).Msg("error creating team")
		writeError(http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess("Team created")
}

func (h *HandlersMap) importAdminTeamLogosFromPayload(uuid string, logos []adminTeamsTransferLogo) (int, int, int, error) {
	var existingLogos []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&existingLogos).Error; err != nil {
		return 0, 0, 0, err
	}
	logosByName := make(map[string]teams.TeamLogo, len(existingLogos))
	for _, l := range existingLogos {
		key := strings.ToLower(strings.TrimSpace(l.Name))
		if key == "" {
			continue
		}
		logosByName[key] = l
	}

	createdLogos := 0
	updatedLogos := 0
	skippedLogos := 0
	for _, inLogo := range logos {
		name := strings.TrimSpace(inLogo.Name)
		logo := normalizeLogoSymbolName(strings.TrimSpace(inLogo.Logo))
		if name == "" || logo == "" {
			skippedLogos++
			continue
		}
		key := strings.ToLower(name)
		if existing, ok := logosByName[key]; ok {
			result := h.Teams.DB.Model(&teams.TeamLogo{}).
				Where("id = ? AND uuid = ?", existing.ID, uuid).
				Updates(map[string]interface{}{
					"logo":      logo,
					"enabled":   inLogo.Enabled,
					"custom":    inLogo.Custom,
					"protected": inLogo.Protected,
					"used":      inLogo.Used,
				})
			if result.Error != nil {
				return createdLogos, updatedLogos, skippedLogos, result.Error
			}
			updatedLogos++
			continue
		}

		newLogo, err := h.Teams.NewLogo(name, logo, inLogo.Enabled, inLogo.Custom, 0)
		if err != nil {
			return createdLogos, updatedLogos, skippedLogos, err
		}
		newLogo.Protected = inLogo.Protected
		newLogo.Used = inLogo.Used
		if err := h.Teams.CreateLogo(newLogo); err != nil {
			return createdLogos, updatedLogos, skippedLogos, err
		}
		createdLogos++
	}
	return createdLogos, updatedLogos, skippedLogos, nil
}

func (h *HandlersMap) importAdminTeamsFromPayload(uuid string, inTeams []adminTeamsTransferTeam) (int, int, int, error) {
	var existingTeams []teams.PlatformTeam
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&existingTeams).Error; err != nil {
		return 0, 0, 0, err
	}
	teamsByName := make(map[string]teams.PlatformTeam, len(existingTeams))
	for _, t := range existingTeams {
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if key == "" {
			continue
		}
		teamsByName[key] = t
	}

	createdTeams := 0
	updatedTeams := 0
	skippedTeams := 0
	for _, inTeam := range inTeams {
		name := strings.TrimSpace(inTeam.Name)
		if name == "" {
			skippedTeams++
			continue
		}
		logo := normalizeLogoSymbolName(strings.TrimSpace(inTeam.Logo))
		if logo == "" || strings.EqualFold(logo, "random") {
			randomLogo, err := h.Teams.RandomLogo()
			if err != nil {
				logo = "invader"
			} else {
				logo = normalizeLogoSymbolName(randomLogo.Logo)
			}
		}

		key := strings.ToLower(name)
		if existing, ok := teamsByName[key]; ok {
			result := h.Teams.DB.Model(&teams.PlatformTeam{}).
				Where("id = ? AND uuid = ?", existing.ID, uuid).
				Updates(map[string]interface{}{
					"logo":      logo,
					"active":    inTeam.Active,
					"visible":   inTeam.Visible,
					"protected": inTeam.Protected,
				})
			if result.Error != nil {
				return createdTeams, updatedTeams, skippedTeams, result.Error
			}
			updatedTeams++
			continue
		}

		newTeam, err := h.Teams.New(name, logo, inTeam.Protected, inTeam.Visible)
		if err != nil {
			return createdTeams, updatedTeams, skippedTeams, err
		}
		newTeam.Active = inTeam.Active
		if err := h.Teams.Create(newTeam); err != nil {
			return createdTeams, updatedTeams, skippedTeams, err
		}
		createdTeams++
	}
	return createdTeams, updatedTeams, skippedTeams, nil
}

// AdminTeamsExportHandler exports teams and logos as JSON
func (h *HandlersMap) AdminTeamsExportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	teamsList, err := h.Teams.GetAll()
	if err != nil {
		log.Err(err).Msg("error loading teams for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load teams",
		})
		return
	}
	var logosList []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&logosList).Error; err != nil {
		log.Err(err).Msg("error loading logos for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load logos",
		})
		return
	}

	sort.Slice(teamsList, func(i, j int) bool {
		return strings.ToLower(teamsList[i].Name) < strings.ToLower(teamsList[j].Name)
	})
	sort.Slice(logosList, func(i, j int) bool {
		return strings.ToLower(logosList[i].Name) < strings.ToLower(logosList[j].Name)
	})

	payload := adminTeamsTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Logos:      make([]adminTeamsTransferLogo, 0, len(logosList)),
		Teams:      make([]adminTeamsTransferTeam, 0, len(teamsList)),
	}

	for _, logo := range logosList {
		payload.Logos = append(payload.Logos, adminTeamsTransferLogo{
			Name:      strings.TrimSpace(logo.Name),
			Logo:      normalizeLogoSymbolName(strings.TrimSpace(logo.Logo)),
			Enabled:   logo.Enabled,
			Custom:    logo.Custom,
			Protected: logo.Protected,
			Used:      logo.Used,
		})
	}
	for _, team := range teamsList {
		payload.Teams = append(payload.Teams, adminTeamsTransferTeam{
			Name:      strings.TrimSpace(team.Name),
			Logo:      normalizeLogoSymbolName(strings.TrimSpace(team.Logo)),
			Active:    team.Active,
			Visible:   team.Visible,
			Protected: team.Protected,
		})
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling teams export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to generate export JSON",
		})
		return
	}

	fileName := "mapctf-teams-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// AdminTeamsExportTeamsHandler exports teams only as JSON
func (h *HandlersMap) AdminTeamsExportTeamsHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	teamsList, err := h.Teams.GetAll()
	if err != nil {
		log.Err(err).Msg("error loading teams for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load teams",
		})
		return
	}
	sort.Slice(teamsList, func(i, j int) bool {
		return strings.ToLower(teamsList[i].Name) < strings.ToLower(teamsList[j].Name)
	})

	payload := adminTeamsTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Teams:      make([]adminTeamsTransferTeam, 0, len(teamsList)),
	}
	for _, team := range teamsList {
		payload.Teams = append(payload.Teams, adminTeamsTransferTeam{
			Name:      strings.TrimSpace(team.Name),
			Logo:      normalizeLogoSymbolName(strings.TrimSpace(team.Logo)),
			Active:    team.Active,
			Visible:   team.Visible,
			Protected: team.Protected,
		})
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling teams export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to generate export JSON",
		})
		return
	}

	fileName := "mapctf-teams-only-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// AdminTeamsExportLogosHandler exports team logos only as JSON
func (h *HandlersMap) AdminTeamsExportLogosHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	var logosList []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&logosList).Error; err != nil {
		log.Err(err).Msg("error loading logos for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load logos",
		})
		return
	}
	sort.Slice(logosList, func(i, j int) bool {
		return strings.ToLower(logosList[i].Name) < strings.ToLower(logosList[j].Name)
	})

	payload := adminTeamsTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Logos:      make([]adminTeamsTransferLogo, 0, len(logosList)),
	}
	for _, logo := range logosList {
		payload.Logos = append(payload.Logos, adminTeamsTransferLogo{
			Name:      strings.TrimSpace(logo.Name),
			Logo:      normalizeLogoSymbolName(strings.TrimSpace(logo.Logo)),
			Enabled:   logo.Enabled,
			Custom:    logo.Custom,
			Protected: logo.Protected,
			Used:      logo.Used,
		})
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling logos export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to generate export JSON",
		})
		return
	}

	fileName := "mapctf-logos-only-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// AdminTeamsImportHandler imports teams and logos from a JSON payload/file
func (h *HandlersMap) AdminTeamsImportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	var payload adminTeamsTransferPayload
	if err := h.decodeTeamsImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing teams import payload")
		writeError(http.StatusBadRequest, "Invalid import payload")
		return
	}
	if len(payload.Logos) == 0 && len(payload.Teams) == 0 {
		writeError(http.StatusBadRequest, "No teams or logos found in import payload")
		return
	}

	var existingLogos []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&existingLogos).Error; err != nil {
		log.Err(err).Msg("error loading existing logos for import")
		writeError(http.StatusInternalServerError, "Failed to load logos")
		return
	}
	logosByName := make(map[string]teams.TeamLogo, len(existingLogos))
	for _, l := range existingLogos {
		key := strings.ToLower(strings.TrimSpace(l.Name))
		if key == "" {
			continue
		}
		logosByName[key] = l
	}

	createdLogos := 0
	updatedLogos := 0
	skippedLogos := 0
	for _, inLogo := range payload.Logos {
		name := strings.TrimSpace(inLogo.Name)
		logo := normalizeLogoSymbolName(strings.TrimSpace(inLogo.Logo))
		if name == "" || logo == "" {
			skippedLogos++
			continue
		}
		key := strings.ToLower(name)
		if existing, ok := logosByName[key]; ok {
			result := h.Teams.DB.Model(&teams.TeamLogo{}).
				Where("id = ? AND uuid = ?", existing.ID, uuid).
				Updates(map[string]interface{}{
					"logo":      logo,
					"enabled":   inLogo.Enabled,
					"custom":    inLogo.Custom,
					"protected": inLogo.Protected,
					"used":      inLogo.Used,
				})
			if result.Error != nil {
				log.Err(result.Error).Msg("error updating logo from import")
				writeError(http.StatusInternalServerError, "Failed to import logos")
				return
			}
			updatedLogos++
			continue
		}

		newLogo, err := h.Teams.NewLogo(name, logo, inLogo.Enabled, inLogo.Custom, 0)
		if err != nil {
			log.Err(err).Msg("error creating logo from import")
			writeError(http.StatusBadRequest, "Failed to import logos")
			return
		}
		newLogo.Protected = inLogo.Protected
		newLogo.Used = inLogo.Used
		if err := h.Teams.CreateLogo(newLogo); err != nil {
			log.Err(err).Msg("error saving logo from import")
			writeError(http.StatusInternalServerError, "Failed to import logos")
			return
		}
		createdLogos++
	}

	var existingTeams []teams.PlatformTeam
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&existingTeams).Error; err != nil {
		log.Err(err).Msg("error loading existing teams for import")
		writeError(http.StatusInternalServerError, "Failed to load teams")
		return
	}
	teamsByName := make(map[string]teams.PlatformTeam, len(existingTeams))
	for _, t := range existingTeams {
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if key == "" {
			continue
		}
		teamsByName[key] = t
	}

	createdTeams := 0
	updatedTeams := 0
	skippedTeams := 0
	for _, inTeam := range payload.Teams {
		name := strings.TrimSpace(inTeam.Name)
		if name == "" {
			skippedTeams++
			continue
		}
		logo := normalizeLogoSymbolName(strings.TrimSpace(inTeam.Logo))
		if logo == "" || strings.EqualFold(logo, "random") {
			randomLogo, err := h.Teams.RandomLogo()
			if err != nil {
				log.Err(err).Msg("error resolving random logo during team import")
				logo = "invader"
			} else {
				logo = normalizeLogoSymbolName(randomLogo.Logo)
			}
		}

		key := strings.ToLower(name)
		if existing, ok := teamsByName[key]; ok {
			result := h.Teams.DB.Model(&teams.PlatformTeam{}).
				Where("id = ? AND uuid = ?", existing.ID, uuid).
				Updates(map[string]interface{}{
					"logo":      logo,
					"active":    inTeam.Active,
					"visible":   inTeam.Visible,
					"protected": inTeam.Protected,
				})
			if result.Error != nil {
				log.Err(result.Error).Msg("error updating team from import")
				writeError(http.StatusInternalServerError, "Failed to import teams")
				return
			}
			updatedTeams++
			continue
		}

		newTeam, err := h.Teams.New(name, logo, inTeam.Protected, inTeam.Visible)
		if err != nil {
			log.Err(err).Msg("error creating team object from import")
			writeError(http.StatusBadRequest, "Failed to import teams")
			return
		}
		newTeam.Active = inTeam.Active
		if err := h.Teams.Create(newTeam); err != nil {
			log.Err(err).Msg("error saving team from import")
			writeError(http.StatusInternalServerError, "Failed to import teams")
			return
		}
		createdTeams++
	}

	messageParts := []string{
		"logos created " + strconv.Itoa(createdLogos),
		"logos updated " + strconv.Itoa(updatedLogos),
		"teams created " + strconv.Itoa(createdTeams),
		"teams updated " + strconv.Itoa(updatedTeams),
	}
	if skippedLogos > 0 {
		messageParts = append(messageParts, "logos skipped "+strconv.Itoa(skippedLogos))
	}
	if skippedTeams > 0 {
		messageParts = append(messageParts, "teams skipped "+strconv.Itoa(skippedTeams))
	}
	if err := h.Teams.SyncLogoUsage(); err != nil {
		log.Err(err).Msg("error syncing logo usage after teams import")
		writeError(http.StatusInternalServerError, "Import completed but failed to sync logo usage")
		return
	}

	writeSuccess("Import complete: " + strings.Join(messageParts, ", "))
}

// AdminTeamsImportTeamsHandler imports teams only from JSON payload/file
func (h *HandlersMap) AdminTeamsImportTeamsHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{Success: false, Status: "error", Message: msg})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: msg})
	}

	var payload adminTeamsTransferPayload
	if err := h.decodeTeamsImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing teams-only import payload")
		writeError(http.StatusBadRequest, "Invalid import payload")
		return
	}
	if len(payload.Teams) == 0 {
		writeError(http.StatusBadRequest, "No teams found in import payload")
		return
	}

	createdTeams, updatedTeams, skippedTeams, err := h.importAdminTeamsFromPayload(uuid, payload.Teams)
	if err != nil {
		log.Err(err).Msg("error importing teams")
		writeError(http.StatusInternalServerError, "Failed to import teams")
		return
	}

	messageParts := []string{
		"teams created " + strconv.Itoa(createdTeams),
		"teams updated " + strconv.Itoa(updatedTeams),
	}
	if skippedTeams > 0 {
		messageParts = append(messageParts, "teams skipped "+strconv.Itoa(skippedTeams))
	}
	writeSuccess("Import complete: " + strings.Join(messageParts, ", "))
}

// AdminTeamsImportLogosHandler imports logos only from JSON payload/file
func (h *HandlersMap) AdminTeamsImportLogosHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{Success: false, Status: "error", Message: msg})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: msg})
	}

	var payload adminTeamsTransferPayload
	if err := h.decodeTeamsImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing logos-only import payload")
		writeError(http.StatusBadRequest, "Invalid import payload")
		return
	}
	if len(payload.Logos) == 0 {
		writeError(http.StatusBadRequest, "No logos found in import payload")
		return
	}

	createdLogos, updatedLogos, skippedLogos, err := h.importAdminTeamLogosFromPayload(uuid, payload.Logos)
	if err != nil {
		log.Err(err).Msg("error importing logos")
		writeError(http.StatusInternalServerError, "Failed to import logos")
		return
	}

	messageParts := []string{
		"logos created " + strconv.Itoa(createdLogos),
		"logos updated " + strconv.Itoa(updatedLogos),
	}
	if skippedLogos > 0 {
		messageParts = append(messageParts, "logos skipped "+strconv.Itoa(skippedLogos))
	}
	writeSuccess("Import complete: " + strings.Join(messageParts, ", "))
}

// AdminTeamsEnableAllPOSTHandler enables all teams
func (h *HandlersMap) AdminTeamsEnableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("uuid = ?", uuid).Update("active", true)
	if result.Error != nil {
		log.Err(result.Error).Msg("error enabling all teams")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to enable all teams"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Enabled " + strconv.FormatInt(result.RowsAffected, 10) + " team(s)"})
}

// AdminTeamsDisableAllPOSTHandler disables all teams
func (h *HandlersMap) AdminTeamsDisableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("uuid = ?", uuid).Update("active", false)
	if result.Error != nil {
		log.Err(result.Error).Msg("error disabling all teams")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to disable all teams"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Disabled " + strconv.FormatInt(result.RowsAffected, 10) + " team(s)"})
}

// AdminTeamsVisibleAllPOSTHandler sets all teams visible
func (h *HandlersMap) AdminTeamsVisibleAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("uuid = ?", uuid).Update("visible", true)
	if result.Error != nil {
		log.Err(result.Error).Msg("error setting all teams visible")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to make all teams visible"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Set visible for " + strconv.FormatInt(result.RowsAffected, 10) + " team(s)"})
}

// AdminTeamsInvisibleAllPOSTHandler sets all teams invisible
func (h *HandlersMap) AdminTeamsInvisibleAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("uuid = ?", uuid).Update("visible", false)
	if result.Error != nil {
		log.Err(result.Error).Msg("error setting all teams invisible")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to make all teams invisible"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Set invisible for " + strconv.FormatInt(result.RowsAffected, 10) + " team(s)"})
}

// AdminTeamLogosEnableAllPOSTHandler enables all team logos
func (h *HandlersMap) AdminTeamLogosEnableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Model(&teams.TeamLogo{}).Where("uuid = ?", uuid).Update("enabled", true)
	if result.Error != nil {
		log.Err(result.Error).Msg("error enabling all logos")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to enable all logos"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Enabled " + strconv.FormatInt(result.RowsAffected, 10) + " logo(s)"})
}

// AdminTeamLogosDisableAllPOSTHandler disables all team logos
func (h *HandlersMap) AdminTeamLogosDisableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Model(&teams.TeamLogo{}).Where("uuid = ?", uuid).Update("enabled", false)
	if result.Error != nil {
		log.Err(result.Error).Msg("error disabling all logos")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to disable all logos"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Disabled " + strconv.FormatInt(result.RowsAffected, 10) + " logo(s)"})
}

// AdminTeamLogosDeleteAllPOSTHandler deletes all team logos
func (h *HandlersMap) AdminTeamLogosDeleteAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	result := h.Teams.DB.Where("uuid = ?", uuid).Delete(&teams.TeamLogo{})
	if result.Error != nil {
		log.Err(result.Error).Msg("error deleting all logos")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: "Failed to delete all logos"})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: "Deleted " + strconv.FormatInt(result.RowsAffected, 10) + " logo(s)"})
}

// AdminTeamsDeleteAllPOSTHandler deletes all teams and unassigns related users
func (h *HandlersMap) AdminTeamsDeleteAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	tx := h.Teams.DB.Begin()
	if tx.Error != nil {
		log.Err(tx.Error).Msg("error starting delete-all-teams transaction")
		writeError(http.StatusInternalServerError, "Failed to delete all teams")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Model(&users.PlatformUser{}).Where("uuid = ?", uuid).Update("team_id", users.NoTeamID).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error clearing user teams before bulk delete")
		writeError(http.StatusInternalServerError, "Failed to delete all teams")
		return
	}
	if err := tx.Where("uuid = ?", uuid).Delete(&teams.TeamMembership{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting memberships before bulk delete")
		writeError(http.StatusInternalServerError, "Failed to delete all teams")
		return
	}
	if err := tx.Where("uuid = ?", uuid).Delete(&teams.TeamScore{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting scores before bulk delete")
		writeError(http.StatusInternalServerError, "Failed to delete all teams")
		return
	}
	deleteResult := tx.Where("uuid = ?", uuid).Delete(&teams.PlatformTeam{})
	if deleteResult.Error != nil {
		tx.Rollback()
		log.Err(deleteResult.Error).Msg("error deleting all teams")
		writeError(http.StatusInternalServerError, "Failed to delete all teams")
		return
	}
	if err := tx.Commit().Error; err != nil {
		log.Err(err).Msg("error committing delete-all-teams transaction")
		writeError(http.StatusInternalServerError, "Failed to delete all teams")
		return
	}
	if err := h.Teams.SyncLogoUsage(); err != nil {
		log.Err(err).Msg("error syncing logo usage after delete-all-teams")
		writeError(http.StatusInternalServerError, "Deleted teams but failed to sync logo usage")
		return
	}

	writeSuccess("Deleted " + strconv.FormatInt(deleteResult.RowsAffected, 10) + " team(s)")
}

// AdminTeamUpdatePOSTHandler updates editable team settings from admin view
func (h *HandlersMap) AdminTeamUpdatePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	teamIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil || teamID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Invalid team id",
		})
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req AdminTeamUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin team update JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(http.StatusBadRequest, "Team name is required")
		return
	}
	var duplicateCount int64
	if err := h.Teams.DB.Model(&teams.PlatformTeam{}).
		Where("uuid = ? AND name = ? AND id <> ?", uuid, name, uint(teamID)).
		Count(&duplicateCount).Error; err != nil {
		log.Err(err).Msg("error validating team name uniqueness")
		writeError(http.StatusInternalServerError, "Failed to validate team name")
		return
	}
	if duplicateCount > 0 {
		writeError(http.StatusBadRequest, "Team name already exists")
		return
	}

	logoInput := strings.TrimSpace(req.Logo)
	logo := normalizeLogoSymbolName(logoInput)
	if logoInput == "" {
		writeError(http.StatusBadRequest, "Logo is required")
		return
	}
	if strings.EqualFold(logoInput, "random") {
		randomLogo, err := h.Teams.RandomLogo()
		if err != nil {
			log.Err(err).Msg("error getting random logo for team update")
			writeError(http.StatusInternalServerError, "Failed to resolve random logo")
			return
		}
		logo = normalizeLogoSymbolName(randomLogo.Logo)
	}

	visible, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Visible)))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid visible value")
		return
	}

	protected, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Protected)))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid protected value")
		return
	}

	updateResult := h.Teams.DB.Model(&teams.PlatformTeam{}).
		Where("id = ? AND uuid = ?", uint(teamID), uuid).
		Updates(map[string]interface{}{
			"name":      name,
			"logo":      logo,
			"visible":   visible,
			"protected": protected,
		})
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating team")
		writeError(http.StatusInternalServerError, "Failed to update team")
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, "Team not found")
		return
	}
	if err := h.Teams.SyncLogoUsage(); err != nil {
		log.Err(err).Msg("error syncing logo usage after team update")
		writeError(http.StatusInternalServerError, "Team updated but failed to sync logo usage")
		return
	}

	writeSuccess("Team updated")
}

// AdminTeamDeletePOSTHandler deletes a team from the admin view
func (h *HandlersMap) AdminTeamDeletePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	teamIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	teamID64, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil || teamID64 == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Invalid team id",
		})
		return
	}
	teamID := uint(teamID64)

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}

	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", teamID, uuid).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(http.StatusNotFound, "Team not found")
			return
		}
		log.Err(err).Msg("error loading team to delete")
		writeError(http.StatusInternalServerError, "Failed to load team")
		return
	}

	tx := h.Teams.DB.Begin()
	if tx.Error != nil {
		log.Err(tx.Error).Msg("error starting team delete transaction")
		writeError(http.StatusInternalServerError, "Failed to delete team")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Model(&users.PlatformUser{}).
		Where("team_id = ? AND uuid = ?", teamID, uuid).
		Update("team_id", users.NoTeamID).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error clearing users team assignment before delete")
		writeError(http.StatusInternalServerError, "Failed to delete team")
		return
	}

	if err := tx.Where("team_id = ? AND uuid = ?", teamID, uuid).Delete(&teams.TeamMembership{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting team memberships before delete")
		writeError(http.StatusInternalServerError, "Failed to delete team")
		return
	}

	if err := tx.Where("team_id = ? AND uuid = ?", teamID, uuid).Delete(&teams.TeamScore{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting team scores before delete")
		writeError(http.StatusInternalServerError, "Failed to delete team")
		return
	}

	deleteResult := tx.Where("id = ? AND uuid = ?", teamID, uuid).Delete(&teams.PlatformTeam{})
	if deleteResult.Error != nil {
		tx.Rollback()
		log.Err(deleteResult.Error).Msg("error deleting team")
		writeError(http.StatusInternalServerError, "Failed to delete team")
		return
	}
	if deleteResult.RowsAffected == 0 {
		tx.Rollback()
		writeError(http.StatusNotFound, "Team not found")
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Err(err).Msg("error committing team delete transaction")
		writeError(http.StatusInternalServerError, "Failed to delete team")
		return
	}
	if err := h.Teams.SyncLogoUsage(); err != nil {
		log.Err(err).Msg("error syncing logo usage after team delete")
		writeError(http.StatusInternalServerError, "Team deleted but failed to sync logo usage")
		return
	}

	writeSuccess("Team deleted")
}

// AdminTeamLogosPOSTHandler for admin team logos creation via POST requests
func (h *HandlersMap) AdminTeamLogosPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}
	contentType := strings.ToLower(r.Header.Get(ContentType))
	isJSON := strings.Contains(contentType, JSONApplication)
	isMultipart := strings.Contains(contentType, "multipart/form-data")
	if !isJSON && !isMultipart {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json or multipart/form-data")
		return
	}

	var (
		name               string
		rawLogo            string
		uploadedLogoSlug   string
		uploadedLogoData   []byte
		uploadedLogoSymbol string
	)

	if isMultipart {
		if err := r.ParseMultipartForm(maxCustomLogoUploadBytes * 2); err != nil {
			writeError(http.StatusBadRequest, "Invalid multipart form payload")
			return
		}
		name = strings.TrimSpace(r.FormValue("name"))
		rawLogo = strings.TrimSpace(r.FormValue("logo"))

		file, fileHeader, err := r.FormFile("logo_file")
		if err == nil && file != nil {
			defer file.Close()

			svgData, readErr := io.ReadAll(io.LimitReader(file, maxCustomLogoUploadBytes+1))
			if readErr != nil {
				writeError(http.StatusBadRequest, "Failed to read uploaded logo file")
				return
			}
			if int64(len(svgData)) > maxCustomLogoUploadBytes {
				writeError(http.StatusBadRequest, "Uploaded logo file is too large")
				return
			}

			slugInput := strings.TrimSpace(r.FormValue("logo_slug"))
			if slugInput == "" && fileHeader != nil {
				slugInput = strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename))
			}
			slug := sanitizeLogoSlug(slugInput)
			if slug == "" {
				writeError(http.StatusBadRequest, "Invalid logo slug")
				return
			}

			symbolMarkup, symbolErr := buildUploadedLogoSymbol(slug, svgData)
			if symbolErr != nil {
				writeError(http.StatusBadRequest, "Invalid SVG file")
				return
			}

			uploadedLogoSlug = slug
			uploadedLogoData = svgData
			uploadedLogoSymbol = symbolMarkup
			rawLogo = slug
		} else if err != nil && !errors.Is(err, http.ErrMissingFile) {
			writeError(http.StatusBadRequest, "Invalid uploaded logo file")
			return
		}
	} else {
		var req AdminLogoCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Err(err).Msg("error parsing admin logos JSON payload")
			writeError(http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		name = strings.TrimSpace(req.Name)
		rawLogo = strings.TrimSpace(req.Logo)
	}

	if name == "" {
		writeError(http.StatusBadRequest, "Logo name is required")
		return
	}
	if strings.EqualFold(rawLogo, "random") {
		writeError(http.StatusBadRequest, "Random is not a valid logo for logo creation")
		return
	}
	logo := normalizeLogoSymbolName(rawLogo)
	if rawLogo == "" || logo == "" {
		writeError(http.StatusBadRequest, "Logo symbol is required")
		return
	}
	if h.Teams.ExistsLogo(name) {
		writeError(http.StatusBadRequest, "Logo already exists")
		return
	}
	if uploadedLogoSlug != "" {
		spritePath := filepath.Join(h.Config.Map.StaticDir, "svg", "icons", "icons.svg")
		if err := appendSymbolToSprite(spritePath, "icon--badge-"+uploadedLogoSlug, uploadedLogoSymbol); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "already exists") {
				writeError(http.StatusBadRequest, "Logo symbol already exists")
				return
			}
			log.Err(err).Msg("error appending custom logo symbol")
			writeError(http.StatusInternalServerError, "Failed to register custom logo")
			return
		}
		if err := saveUploadedLogoFile(h.Config.Map.StaticDir, uploadedLogoSlug, uploadedLogoData); err != nil {
			log.Err(err).Msg("error saving custom logo file")
			writeError(http.StatusInternalServerError, "Failed to store custom logo file")
			return
		}
	}

	newLogo, err := h.Teams.NewLogo(name, logo, true, true, 0)
	if err != nil {
		log.Err(err).Msg("error creating logo object")
		writeError(http.StatusBadRequest, "Failed to create logo")
		return
	}
	if err := h.Teams.CreateLogo(newLogo); err != nil {
		log.Err(err).Msg("error saving logo")
		writeError(http.StatusInternalServerError, "Failed to create logo")
		return
	}

	writeSuccess("Logo created")
}

// AdminTeamLogoUpdatePOSTHandler updates editable team logo settings from admin view
func (h *HandlersMap) AdminTeamLogoUpdatePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	logoIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	logoID, err := strconv.ParseUint(logoIDStr, 10, 64)
	if err != nil || logoID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Invalid logo id",
		})
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req AdminLogoUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin logo update JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(http.StatusBadRequest, "Logo name is required")
		return
	}

	enabled, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Enabled)))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid enabled value")
		return
	}

	protected, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Protected)))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid protected value")
		return
	}

	updateResult := h.Teams.DB.Model(&teams.TeamLogo{}).
		Where("id = ? AND uuid = ?", uint(logoID), uuid).
		Updates(map[string]interface{}{
			"name":      name,
			"enabled":   enabled,
			"protected": protected,
		})
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating logo")
		writeError(http.StatusInternalServerError, "Failed to update logo")
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, "Logo not found")
		return
	}

	writeSuccess("Logo updated")
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
		Title:         "MapCTF Admin: Users",
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
	templateData.TeamNames = map[uint]string{
		0: "None",
	}
	teams, err := h.Teams.GetAll()
	if err != nil {
		log.Warn().Err(err).Msg("error loading teams for users view")
	} else {
		templateData.Teams = teams
		for _, team := range teams {
			templateData.TeamNames[team.ID] = team.Name
		}
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminUsersPOSTHandler for admin users page for POST requests
func (h *HandlersMap) AdminUsersPOSTHandler(w http.ResponseWriter, r *http.Request) {
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

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req AdminUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin users JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)
	teamIDStr := strings.TrimSpace(req.TeamID)
	adminStr := strings.TrimSpace(req.Admin)
	serviceStr := strings.TrimSpace(req.Service)
	activeStr := strings.TrimSpace(req.Active)

	if username == "" || password == "" {
		writeError(http.StatusBadRequest, "Username and password are required")
		return
	}

	teamID := uint(0)
	if teamIDStr != "" {
		parsedTeamID, err := strconv.ParseUint(teamIDStr, 10, 64)
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid team_id")
			return
		}
		teamID = uint(parsedTeamID)
	}

	admin := false
	if adminStr != "" {
		parsedAdmin, err := strconv.ParseBool(strings.ToLower(adminStr))
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid admin value")
			return
		}
		admin = parsedAdmin
	}

	service := false
	if serviceStr != "" {
		parsedService, err := strconv.ParseBool(strings.ToLower(serviceStr))
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid service value")
			return
		}
		service = parsedService
	}

	active := true
	if activeStr != "" {
		parsedActive, err := strconv.ParseBool(strings.ToLower(activeStr))
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid active value")
			return
		}
		active = parsedActive
	}

	user, err := h.Users.New(username, password, email, name, admin, service, uuid, teamID)
	if err != nil {
		log.Err(err).Msg("error creating user object")
		writeError(http.StatusBadRequest, err.Error())
		return
	}

	if err := h.Users.Create(user); err != nil {
		log.Err(err).Msg("error saving user")
		writeError(http.StatusInternalServerError, "Failed to create user")
		return
	}

	if !active {
		if err := h.Users.SetActive(false, username, uuid); err != nil {
			log.Err(err).Msg("error setting active flag for new user")
			writeError(http.StatusInternalServerError, "User created but failed to set active flag")
			return
		}
	}

	writeSuccess("User created")
}

// AdminUserUpdatePOSTHandler updates editable user settings from admin view
func (h *HandlersMap) AdminUserUpdatePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	userIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil || userID == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Invalid user id",
		})
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, "AJAX requests only")
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req AdminUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin user update JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	teamIDStr := strings.TrimSpace(req.TeamID)
	if teamIDStr == "" {
		teamIDStr = "0"
	}
	adminStr := strings.TrimSpace(req.Admin)
	serviceStr := strings.TrimSpace(req.Service)
	if adminStr == "" || serviceStr == "" {
		writeError(http.StatusBadRequest, "Admin and service values are required")
		return
	}

	teamID64, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid team_id")
		return
	}
	teamID := uint(teamID64)
	adminValue, err := strconv.ParseBool(strings.ToLower(adminStr))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid admin value")
		return
	}
	serviceValue, err := strconv.ParseBool(strings.ToLower(serviceStr))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid service value")
		return
	}

	if teamID != users.NoTeamID {
		var teamCount int64
		if err := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("id = ? AND uuid = ?", teamID, uuid).Count(&teamCount).Error; err != nil {
			log.Err(err).Msg("error validating team for user update")
			writeError(http.StatusInternalServerError, "Failed to validate team")
			return
		}
		if teamCount == 0 {
			writeError(http.StatusBadRequest, "Team not found")
			return
		}
	}

	updateResult := h.Users.DB.Model(&users.PlatformUser{}).
		Where("id = ? AND uuid = ?", uint(userID), uuid).
		Updates(map[string]interface{}{
			"team_id": teamID,
			"admin":   adminValue,
			"service": serviceValue,
		})
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating user")
		writeError(http.StatusInternalServerError, "Failed to update user")
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, "User not found")
		return
	}

	writeSuccess("User updated")
}

// AdminUsersExportHandler exports users as JSON
func (h *HandlersMap) AdminUsersExportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	usersList, err := h.Users.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error loading users for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load users",
		})
		return
	}

	sort.Slice(usersList, func(i, j int) bool {
		return strings.ToLower(usersList[i].Username) < strings.ToLower(usersList[j].Username)
	})

	payload := adminUsersTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Users:      make([]adminUsersTransferUser, 0, len(usersList)),
	}
	for _, user := range usersList {
		payload.Users = append(payload.Users, adminUsersTransferUser{
			Username: strings.TrimSpace(user.Username),
			Name:     strings.TrimSpace(user.Name),
			Email:    strings.TrimSpace(user.Email),
			TeamID:   user.TeamID,
			Admin:    user.Admin,
			Service:  user.Service,
			Active:   user.Active,
			PassHash: strings.TrimSpace(user.PassHash),
		})
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling users export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to generate export JSON",
		})
		return
	}

	fileName := "mapctf-users-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// AdminUsersImportHandler imports users from JSON payload/file
func (h *HandlersMap) AdminUsersImportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	var payload adminUsersTransferPayload
	if err := h.decodeUsersImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing users import payload")
		writeError(http.StatusBadRequest, "Invalid import payload")
		return
	}
	if len(payload.Users) == 0 {
		writeError(http.StatusBadRequest, "No users found in import payload")
		return
	}

	createdUsers := 0
	updatedUsers := 0
	skippedUsers := 0

	for _, inUser := range payload.Users {
		username := strings.TrimSpace(inUser.Username)
		if username == "" {
			skippedUsers++
			continue
		}

		teamID := inUser.TeamID
		if teamID != users.NoTeamID {
			var teamCount int64
			if err := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("id = ? AND uuid = ?", teamID, uuid).Count(&teamCount).Error; err != nil {
				log.Err(err).Msg("error validating team during users import")
				writeError(http.StatusInternalServerError, "Failed to validate teams for import")
				return
			}
			if teamCount == 0 {
				teamID = users.NoTeamID
			}
		}

		exists, existingUser := h.Users.ExistsGet(username, uuid)
		if exists {
			updates := map[string]interface{}{
				"name":    strings.TrimSpace(inUser.Name),
				"email":   strings.TrimSpace(inUser.Email),
				"team_id": teamID,
				"admin":   inUser.Admin,
				"service": inUser.Service,
				"active":  inUser.Active,
			}
			if strings.TrimSpace(inUser.PassHash) != "" {
				updates["pass_hash"] = strings.TrimSpace(inUser.PassHash)
			}
			result := h.Users.DB.Model(&users.PlatformUser{}).
				Where("id = ? AND uuid = ?", existingUser.ID, uuid).
				Updates(updates)
			if result.Error != nil {
				log.Err(result.Error).Msg("error updating user from import")
				writeError(http.StatusInternalServerError, "Failed to import users")
				return
			}
			updatedUsers++
			continue
		}

		passHash := strings.TrimSpace(inUser.PassHash)
		if passHash == "" {
			skippedUsers++
			continue
		}
		newUser := users.PlatformUser{
			Username: username,
			Name:     strings.TrimSpace(inUser.Name),
			Email:    strings.TrimSpace(inUser.Email),
			TeamID:   teamID,
			PassHash: passHash,
			Admin:    inUser.Admin,
			Service:  inUser.Service,
			Active:   inUser.Active,
			UUID:     uuid,
		}
		if err := h.Users.Create(newUser); err != nil {
			log.Err(err).Msg("error creating user from import")
			writeError(http.StatusInternalServerError, "Failed to import users")
			return
		}
		createdUsers++
	}

	messageParts := []string{
		"users created " + strconv.Itoa(createdUsers),
		"users updated " + strconv.Itoa(updatedUsers),
	}
	if skippedUsers > 0 {
		messageParts = append(messageParts, "users skipped "+strconv.Itoa(skippedUsers))
	}

	writeSuccess("Import complete: " + strings.Join(messageParts, ", "))
}

// AdminUsersEnableAllPOSTHandler enables all users
func (h *HandlersMap) AdminUsersEnableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	result := h.Users.DB.Model(&users.PlatformUser{}).Where("uuid = ?", uuid).Update("active", true)
	if result.Error != nil {
		log.Err(result.Error).Msg("error enabling all users")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to enable all users",
		})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: "Enabled " + strconv.FormatInt(result.RowsAffected, 10) + " user(s)",
	})
}

// AdminUsersDisableAllPOSTHandler disables all users
func (h *HandlersMap) AdminUsersDisableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	currentUsername := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if currentUsername == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Not authenticated",
		})
		return
	}

	result := h.Users.DB.Model(&users.PlatformUser{}).
		Where("uuid = ? AND username <> ?", uuid, currentUsername).
		Update("active", false)
	if result.Error != nil {
		log.Err(result.Error).Msg("error disabling all users")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to disable all users",
		})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: "Disabled " + strconv.FormatInt(result.RowsAffected, 10) + " user(s). Current user not modified.",
	})
}

// AdminUsersDeleteAllPOSTHandler deletes all users
func (h *HandlersMap) AdminUsersDeleteAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	currentUsername := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if currentUsername == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Not authenticated",
		})
		return
	}

	result := h.Users.DB.Where("uuid = ? AND username <> ?", uuid, currentUsername).Delete(&users.PlatformUser{})
	if result.Error != nil {
		log.Err(result.Error).Msg("error deleting all users")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to delete all users",
		})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: "Deleted " + strconv.FormatInt(result.RowsAffected, 10) + " user(s). Current user preserved.",
	})
}

// AdminChallengesExportHandler exports all categories and challenges as JSON
func (h *HandlersMap) AdminChallengesExportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	categoriesList, err := h.Challenges.GetAllCategories(uuid)
	if err != nil {
		log.Err(err).Msg("error loading categories for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load categories",
		})
		return
	}
	challengesList, err := h.Challenges.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenges for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to load challenges",
		})
		return
	}

	sort.Slice(categoriesList, func(i, j int) bool {
		return strings.ToLower(categoriesList[i].Name) < strings.ToLower(categoriesList[j].Name)
	})
	sort.Slice(challengesList, func(i, j int) bool {
		return strings.ToLower(challengesList[i].Title) < strings.ToLower(challengesList[j].Title)
	})

	categoriesByID := make(map[uint]adminChallengesTransferCategory, len(categoriesList))
	transferCategories := make([]adminChallengesTransferCategory, 0, len(categoriesList))
	for _, category := range categoriesList {
		entry := adminChallengesTransferCategory{
			Name:        strings.TrimSpace(category.Name),
			Description: strings.TrimSpace(category.Description),
			Logo:        strings.TrimSpace(category.Logo),
		}
		categoriesByID[category.ID] = entry
		transferCategories = append(transferCategories, entry)
	}

	transferChallenges := make([]adminChallengesTransferItem, 0, len(challengesList))
	for _, challenge := range challengesList {
		transferItem := adminChallengesTransferItem{
			Title:       strings.TrimSpace(challenge.Title),
			Description: strings.TrimSpace(challenge.Description),
			Country:     strings.ToUpper(strings.TrimSpace(challenge.Country)),
			Active:      challenge.Active,
			Points:      challenge.Points,
			Bonus:       challenge.Bonus,
			BonusDecay:  challenge.BonusDecay,
			Penalty:     challenge.Penalty,
			Flag:        strings.TrimSpace(challenge.Flag),
			Hint:        strings.TrimSpace(challenge.Hint),
		}
		if category, ok := categoriesByID[challenge.CategoryID]; ok {
			transferItem.Category = category.Name
		}
		transferChallenges = append(transferChallenges, transferItem)
	}

	payload := adminChallengesTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Categories: transferCategories,
		Challenges: transferChallenges,
	}
	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling challenges export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: "Failed to generate export JSON",
		})
		return
	}

	fileName := "mapctf-challenges-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// AdminChallengesImportHandler imports categories and challenges from a JSON payload/file
func (h *HandlersMap) AdminChallengesImportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	var payload adminChallengesTransferPayload
	if err := h.decodeChallengeImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing challenges import payload")
		writeError(http.StatusBadRequest, "Invalid import payload")
		return
	}
	if len(payload.Challenges) == 0 {
		writeError(http.StatusBadRequest, "No challenges found in import payload")
		return
	}

	existingCategories, err := h.Challenges.GetAllCategories(uuid)
	if err != nil {
		log.Err(err).Msg("error loading existing categories for import")
		writeError(http.StatusInternalServerError, "Failed to load categories")
		return
	}

	type categoryRef struct {
		ID          uint
		Description string
		Logo        string
	}
	categoriesByName := make(map[string]categoryRef, len(existingCategories))
	for _, category := range existingCategories {
		key := strings.ToLower(strings.TrimSpace(category.Name))
		if key == "" {
			continue
		}
		categoriesByName[key] = categoryRef{
			ID:          category.ID,
			Description: strings.TrimSpace(category.Description),
			Logo:        strings.TrimSpace(category.Logo),
		}
	}

	createdCategories := 0
	ensureCategory := func(name, description, logo string) (uint, error) {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			return 0, errors.New("category name is required")
		}
		if existing, ok := categoriesByName[key]; ok {
			return existing.ID, nil
		}

		newCategory, err := h.Challenges.NewCategory(strings.TrimSpace(name), strings.TrimSpace(description), strings.TrimSpace(logo), uuid)
		if err != nil {
			return 0, err
		}
		if err := h.Challenges.CreateCategory(newCategory); err != nil {
			return 0, err
		}
		refreshedCategories, err := h.Challenges.GetAllCategories(uuid)
		if err != nil {
			return 0, err
		}
		var createdID uint
		for _, refreshed := range refreshedCategories {
			refreshedKey := strings.ToLower(strings.TrimSpace(refreshed.Name))
			if refreshedKey != key {
				continue
			}
			createdID = refreshed.ID
			break
		}
		if createdID == 0 {
			return 0, errors.New("created category not found")
		}
		createdCategories++
		categoriesByName[key] = categoryRef{
			ID:          createdID,
			Description: strings.TrimSpace(description),
			Logo:        strings.TrimSpace(logo),
		}
		return createdID, nil
	}

	for _, category := range payload.Categories {
		if strings.TrimSpace(category.Name) == "" {
			continue
		}
		if _, err := ensureCategory(category.Name, category.Description, category.Logo); err != nil {
			log.Err(err).Msg("error creating category from import payload")
			writeError(http.StatusBadRequest, "Failed to import categories")
			return
		}
	}

	importedChallenges := 0
	skippedChallenges := 0
	unassignedCountries := 0
	for _, item := range payload.Challenges {
		title := strings.TrimSpace(item.Title)
		flag := strings.TrimSpace(item.Flag)
		categoryName := strings.TrimSpace(item.Category)
		if title == "" || flag == "" || categoryName == "" {
			skippedChallenges++
			continue
		}

		categoryID, err := ensureCategory(categoryName, "", "")
		if err != nil || categoryID == 0 {
			log.Err(err).Msg("error resolving category during challenge import")
			skippedChallenges++
			continue
		}

		countryCode := strings.ToUpper(strings.TrimSpace(item.Country))
		if countryCode != "" {
			selectedCountry, err := h.Countries.GetByCode(countryCode)
			if err != nil || !selectedCountry.Active || selectedCountry.Assigned {
				countryCode = ""
				unassignedCountries++
			}
		}

		challenge := h.Challenges.New(
			title,
			strings.TrimSpace(item.Description),
			categoryID,
			countryCode,
			item.Active,
			item.Points,
			item.Bonus,
			item.BonusDecay,
			item.Penalty,
			flag,
			strings.TrimSpace(item.Hint),
			uuid,
		)

		if err := h.Challenges.CreateAndReturn(&challenge); err != nil {
			log.Err(err).Msg("error creating imported challenge")
			writeError(http.StatusInternalServerError, "Failed to import challenges")
			return
		}
		if countryCode != "" {
			if err := h.Countries.AssignCountryToChallenge(countryCode, challenge.ID); err != nil {
				log.Err(err).Msg("error assigning imported challenge country")
				_ = h.Challenges.Delete(challenge.ID, uuid)
				writeError(http.StatusInternalServerError, "Failed to assign imported challenge country")
				return
			}
		}
		importedChallenges++
	}

	messageParts := []string{
		"Imported " + strconv.Itoa(importedChallenges) + " challenge(s)",
		"created " + strconv.Itoa(createdCategories) + " category(ies)",
	}
	if skippedChallenges > 0 {
		messageParts = append(messageParts, "skipped "+strconv.Itoa(skippedChallenges)+" invalid challenge(s)")
	}
	if unassignedCountries > 0 {
		messageParts = append(messageParts, strconv.Itoa(unassignedCountries)+" challenge(s) imported without country assignment")
	}
	writeSuccess(strings.Join(messageParts, ", "))
}

// AdminChallengesDeleteAllPOSTHandler deletes all challenges and releases assigned countries
func (h *HandlersMap) AdminChallengesDeleteAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	challengesList, err := h.Challenges.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenges before bulk delete")
		writeError(http.StatusInternalServerError, "Failed to load challenges")
		return
	}

	for _, challenge := range challengesList {
		countryCode := strings.ToUpper(strings.TrimSpace(challenge.Country))
		if countryCode == "" {
			continue
		}
		exists, err := h.Countries.Exists(countryCode)
		if err != nil {
			log.Err(err).Msg("error checking challenge country before bulk delete")
			writeError(http.StatusInternalServerError, "Failed to release challenge countries")
			return
		}
		if !exists {
			continue
		}
		if err := h.Countries.ReleaseCountry(countryCode); err != nil {
			log.Err(err).Msg("error releasing challenge country before bulk delete")
			writeError(http.StatusInternalServerError, "Failed to release challenge countries")
			return
		}
	}

	deletedCount, err := h.Challenges.DeleteAll(uuid)
	if err != nil {
		log.Err(err).Msg("error deleting all challenges")
		writeError(http.StatusInternalServerError, "Failed to delete all challenges")
		return
	}

	writeSuccess("Deleted " + strconv.FormatInt(deletedCount, 10) + " challenge(s)")
}

// AdminChallengesEnableAllPOSTHandler enables all challenges
func (h *HandlersMap) AdminChallengesEnableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	updatedCount, err := h.Challenges.SetAllActive(uuid, true)
	if err != nil {
		log.Err(err).Msg("error enabling all challenges")
		writeError(http.StatusInternalServerError, "Failed to enable all challenges")
		return
	}

	writeSuccess("Enabled " + strconv.FormatInt(updatedCount, 10) + " challenge(s)")
}

// AdminChallengesDisableAllPOSTHandler disables all challenges
func (h *HandlersMap) AdminChallengesDisableAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	updatedCount, err := h.Challenges.SetAllActive(uuid, false)
	if err != nil {
		log.Err(err).Msg("error disabling all challenges")
		writeError(http.StatusInternalServerError, "Failed to disable all challenges")
		return
	}

	writeSuccess("Disabled " + strconv.FormatInt(updatedCount, 10) + " challenge(s)")
}

// AdminChallengeCategoriesDeleteAllPOSTHandler deletes all categories after ensuring no challenges exist
func (h *HandlersMap) AdminChallengeCategoriesDeleteAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	challengesList, err := h.Challenges.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenges before category bulk delete")
		writeError(http.StatusInternalServerError, "Failed to load challenges")
		return
	}
	if len(challengesList) > 0 {
		writeError(http.StatusBadRequest, "Delete all challenges before deleting categories")
		return
	}

	deletedCount, err := h.Challenges.DeleteAllCategories(uuid)
	if err != nil {
		log.Err(err).Msg("error deleting all categories")
		writeError(http.StatusInternalServerError, "Failed to delete all categories")
		return
	}

	writeSuccess("Deleted " + strconv.FormatInt(deletedCount, 10) + " categor(ies)")
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
		Title:         "MapCTF Admin: Challenges",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
		CountryFlag:   make(map[string]string),
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
	activity, err := h.Logs.AllActivity(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading activity for solves view")
	} else {
		for _, entry := range activity {
			if strings.Contains(strings.ToLower(entry.Action), "solve") {
				templateData.Solves = append(templateData.Solves, entry)
			}
		}
	}
	templateData.ChallengeCountryOptions = make(map[uint][]countries.MapCountry)
	allCountries, err := h.Countries.GetAll()
	if err != nil {
		log.Warn().Err(err).Msg("error loading all countries")
	} else {
		sort.Slice(allCountries, func(i, j int) bool {
			return allCountries[i].Name < allCountries[j].Name
		})
		templateData.AllCountries = allCountries
	}
	availableCountries, err := h.Countries.GetAvailable()
	if err != nil {
		log.Warn().Err(err).Msg("error loading available countries")
	} else {
		sort.Slice(availableCountries, func(i, j int) bool {
			return availableCountries[i].Name < availableCountries[j].Name
		})
		templateData.AvailableCountries = availableCountries
	}

	allCountriesByCode := make(map[string]countries.MapCountry, len(templateData.AllCountries))
	allCountriesByName := make(map[string]countries.MapCountry, len(templateData.AllCountries))
	for _, c := range templateData.AllCountries {
		code := strings.ToUpper(strings.TrimSpace(c.CountryCode))
		allCountriesByCode[code] = c
		allCountriesByName[strings.ToLower(strings.TrimSpace(c.Name))] = c
		templateData.CountryFlag[code] = countryCodeToFlagEmoji(code)
	}
	availableByCode := make(map[string]countries.MapCountry, len(templateData.AvailableCountries))
	for _, c := range templateData.AvailableCountries {
		availableByCode[strings.ToUpper(strings.TrimSpace(c.CountryCode))] = c
	}

	templateData.ChallengeSolves = make(map[uint][]logs.ActivityLog, len(templateData.Challenges))
	for i := range templateData.Challenges {
		challenge := templateData.Challenges[i]
		challengeTitle := strings.ToLower(challenge.Title)
		challengeID := strconv.Itoa(int(challenge.ID))
		for _, solve := range templateData.Solves {
			searchText := strings.ToLower(solve.Subject + " " + solve.Action + " " + solve.Message + " " + solve.Arguments)
			if strings.Contains(searchText, challengeTitle) ||
				strings.Contains(searchText, "challenge="+challengeID) ||
				strings.Contains(searchText, "challenge_id="+challengeID) ||
				strings.Contains(searchText, "challengeid="+challengeID) {
				templateData.ChallengeSolves[challenge.ID] = append(templateData.ChallengeSolves[challenge.ID], solve)
			}
		}

		normalizedChallengeCountryCode := strings.ToUpper(strings.TrimSpace(challenge.Country))
		if normalizedChallengeCountryCode != "" {
			if _, exists := allCountriesByCode[normalizedChallengeCountryCode]; !exists {
				if countryByName, found := allCountriesByName[strings.ToLower(strings.TrimSpace(challenge.Country))]; found {
					normalizedChallengeCountryCode = strings.ToUpper(strings.TrimSpace(countryByName.CountryCode))
					templateData.Challenges[i].Country = normalizedChallengeCountryCode
				}
			} else {
				templateData.Challenges[i].Country = normalizedChallengeCountryCode
			}
		}

		options := make([]countries.MapCountry, 0, len(templateData.AvailableCountries)+1)
		added := make(map[string]bool, len(templateData.AvailableCountries)+1)
		if normalizedChallengeCountryCode != "" {
			if currentCountry, ok := allCountriesByCode[normalizedChallengeCountryCode]; ok {
				options = append(options, currentCountry)
				added[normalizedChallengeCountryCode] = true
			}
		}
		for _, availableCountry := range templateData.AvailableCountries {
			code := strings.ToUpper(strings.TrimSpace(availableCountry.CountryCode))
			if added[code] {
				continue
			}
			options = append(options, availableCountry)
			added[code] = true
		}
		if len(options) == 0 && normalizedChallengeCountryCode != "" {
			options = append(options, countries.MapCountry{
				Name:        templateData.Challenges[i].Country,
				CountryCode: templateData.Challenges[i].Country,
			})
		}
		templateData.ChallengeCountryOptions[challenge.ID] = options
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

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req AdminChallengeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin challenges JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	categoryIDStr := strings.TrimSpace(req.CategoryID)
	country := strings.ToUpper(strings.TrimSpace(req.Country))
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
	if country != "" {
		selectedCountry, err := h.Countries.GetByCode(country)
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid country code")
			return
		}
		if !selectedCountry.Active || selectedCountry.Assigned {
			writeError(http.StatusBadRequest, "Country must be active and available")
			return
		}
	}
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, "Please select a category")
		return
	}
	if _, err := h.Challenges.GetCategoryByID(uint(categoryID), uuid); err != nil {
		writeError(http.StatusBadRequest, "Please select a valid category")
		return
	}
	active, err := strconv.ParseBool(activeStr)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid active value")
		return
	}
	points, err := strconv.ParseInt(pointsStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid points value")
		return
	}
	bonus, err := strconv.ParseInt(bonusStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid bonus value")
		return
	}
	bonusDecay, err := strconv.ParseInt(bonusDecayStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid bonus_decay value")
		return
	}
	penalty, err := strconv.ParseInt(penaltyStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid penalty value")
		return
	}

	challenge := h.Challenges.New(
		title,
		description,
		uint(categoryID),
		country,
		active,
		int(points),
		int(bonus),
		int(bonusDecay),
		int(penalty),
		flag,
		hint,
		uuid,
	)

	if err := h.Challenges.CreateAndReturn(&challenge); err != nil {
		log.Err(err).Msg("error creating challenge")
		writeError(http.StatusInternalServerError, "Failed to create challenge")
		return
	}
	if country != "" {
		if err := h.Countries.AssignCountryToChallenge(country, challenge.ID); err != nil {
			log.Err(err).Msg("error assigning country to challenge")
			_ = h.Challenges.Delete(challenge.ID, uuid)
			writeError(http.StatusInternalServerError, "Failed to assign country to challenge")
			return
		}
	}

	writeSuccess("Challenge created")
}

// AdminChallengeUpdatePOSTHandler for updating an existing challenge via POST requests
func (h *HandlersMap) AdminChallengeUpdatePOSTHandler(w http.ResponseWriter, r *http.Request) {
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

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	challengeIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	challengeID, err := strconv.ParseUint(challengeIDStr, 10, 64)
	if err != nil || challengeID == 0 {
		writeError(http.StatusBadRequest, "Invalid challenge id")
		return
	}

	var req AdminChallengeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin challenge update JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	categoryIDStr := strings.TrimSpace(req.CategoryID)
	country := strings.ToUpper(strings.TrimSpace(req.Country))
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
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, "Valid category_id is required")
		return
	}
	active, err := strconv.ParseBool(activeStr)
	if err != nil {
		switch strings.ToLower(activeStr) {
		case "active", "on":
			active = true
		case "inactive", "off":
			active = false
		default:
			writeError(http.StatusBadRequest, "Invalid active value")
			return
		}
	}
	points, err := strconv.ParseInt(pointsStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid points value")
		return
	}
	bonus, err := strconv.ParseInt(bonusStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid bonus value")
		return
	}
	bonusDecay, err := strconv.ParseInt(bonusDecayStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid bonus_decay value")
		return
	}
	penalty, err := strconv.ParseInt(penaltyStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid penalty value")
		return
	}

	challenge, err := h.Challenges.GetByID(uint(challengeID), uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenge to update")
		writeError(http.StatusNotFound, "Challenge not found")
		return
	}
	previousCountry := strings.ToUpper(strings.TrimSpace(challenge.Country))
	if country != previousCountry && country != "" {
		selectedCountry, err := h.Countries.GetByCode(country)
		if err != nil {
			writeError(http.StatusBadRequest, "Invalid country code")
			return
		}
		if !selectedCountry.Active || selectedCountry.Assigned {
			writeError(http.StatusBadRequest, "Country must be active and available")
			return
		}
	}

	challenge.Title = title
	challenge.Description = description
	challenge.CategoryID = uint(categoryID)
	challenge.Country = country
	challenge.Active = active
	challenge.Points = int(points)
	challenge.Bonus = int(bonus)
	challenge.BonusDecay = int(bonusDecay)
	challenge.Penalty = int(penalty)
	challenge.Flag = flag
	challenge.Hint = hint

	if err := h.Challenges.Update(challenge); err != nil {
		log.Err(err).Msg("error updating challenge")
		writeError(http.StatusInternalServerError, "Failed to update challenge")
		return
	}
	if previousCountry != country {
		if previousCountry != "" {
			exists, err := h.Countries.Exists(previousCountry)
			if err != nil {
				log.Err(err).Msg("error checking previous challenge country")
				writeError(http.StatusInternalServerError, "Failed to update challenge country assignment")
				return
			}
			if exists {
				if err := h.Countries.ReleaseCountry(previousCountry); err != nil {
					log.Err(err).Msg("error releasing previous challenge country")
					writeError(http.StatusInternalServerError, "Failed to update challenge country assignment")
					return
				}
			}
		}
		if country != "" {
			if err := h.Countries.AssignCountryToChallenge(country, challenge.ID); err != nil {
				log.Err(err).Msg("error assigning updated challenge country")
				writeError(http.StatusInternalServerError, "Failed to update challenge country assignment")
				return
			}
		}
	}

	writeSuccess("Challenge updated")
}

// AdminChallengeDeletePOSTHandler for deleting an existing challenge via POST requests
func (h *HandlersMap) AdminChallengeDeletePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	challengeIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	challengeID, err := strconv.ParseUint(challengeIDStr, 10, 64)
	if err != nil || challengeID == 0 {
		writeError(http.StatusBadRequest, "Invalid challenge id")
		return
	}

	challenge, err := h.Challenges.GetByID(uint(challengeID), uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenge to delete")
		writeError(http.StatusNotFound, "Challenge not found")
		return
	}
	country := strings.ToUpper(strings.TrimSpace(challenge.Country))
	if country != "" {
		exists, err := h.Countries.Exists(country)
		if err != nil {
			log.Err(err).Msg("error checking challenge country before delete")
			writeError(http.StatusInternalServerError, "Failed to release challenge country")
			return
		}
		if exists {
			if err := h.Countries.ReleaseCountry(country); err != nil {
				log.Err(err).Msg("error releasing challenge country before delete")
				writeError(http.StatusInternalServerError, "Failed to release challenge country")
				return
			}
		}
	}

	if err := h.Challenges.Delete(uint(challengeID), uuid); err != nil {
		log.Err(err).Msg("error deleting challenge")
		writeError(http.StatusInternalServerError, "Failed to delete challenge")
		return
	}

	writeSuccess("Challenge deleted")
}

// AdminChallengeCategoriesPOSTHandler for admin challenge categories creation via POST requests
func (h *HandlersMap) AdminChallengeCategoriesPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var req AdminCategoryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin category JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	logo := strings.TrimSpace(req.Logo)

	if name == "" {
		writeError(http.StatusBadRequest, "Category name is required")
		return
	}

	category, err := h.Challenges.NewCategory(name, description, logo, uuid)
	if err != nil {
		log.Err(err).Msg("error creating category object")
		writeError(http.StatusBadRequest, err.Error())
		return
	}

	if err := h.Challenges.CreateCategory(category); err != nil {
		log.Err(err).Msg("error creating category")
		writeError(http.StatusInternalServerError, "Failed to create category")
		return
	}

	writeSuccess("Category created")
}

// AdminChallengeCategoryUpdatePOSTHandler updates an existing challenge category via POST requests
func (h *HandlersMap) AdminChallengeCategoryUpdatePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	if !strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	categoryIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, "Invalid category id")
		return
	}

	var req AdminCategoryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin category update JSON payload")
		writeError(http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	logo := strings.TrimSpace(req.Logo)

	if name == "" {
		writeError(http.StatusBadRequest, "Category name is required")
		return
	}

	category, err := h.Challenges.GetCategoryByID(uint(categoryID), uuid)
	if err != nil {
		log.Err(err).Msg("error loading category to update")
		writeError(http.StatusNotFound, "Category not found")
		return
	}

	category.Name = name
	category.Description = description
	category.Logo = logo

	if err := h.Challenges.UpdateCategory(category); err != nil {
		log.Err(err).Msg("error updating category")
		writeError(http.StatusInternalServerError, "Failed to update category")
		return
	}

	writeSuccess("Category updated")
}

// AdminChallengeCategoryDeletePOSTHandler deletes a category if no challenges are assigned
func (h *HandlersMap) AdminChallengeCategoryDeletePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	writeError := func(code int, msg string) {
		HTTPResponse(w, JSONApplicationUTF8, code, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: msg,
		})
	}
	writeSuccess := func(msg string) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
			Success: true,
			Status:  "ok",
			Message: msg,
		})
	}

	categoryIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, "Invalid category id")
		return
	}

	if _, err := h.Challenges.GetCategoryByID(uint(categoryID), uuid); err != nil {
		log.Err(err).Msg("error loading category to delete")
		writeError(http.StatusNotFound, "Category not found")
		return
	}

	hasChallenges, err := h.Challenges.CategoryHasChallenges(uint(categoryID), uuid)
	if err != nil {
		log.Err(err).Msg("error checking category usage")
		writeError(http.StatusInternalServerError, "Failed to verify category usage")
		return
	}
	if hasChallenges {
		writeError(http.StatusConflict, "Category is assigned to challenges and was not deleted")
		return
	}

	if err := h.Challenges.DeleteCategory(uint(categoryID), uuid); err != nil {
		log.Err(err).Msg("error deleting category")
		writeError(http.StatusInternalServerError, "Failed to delete category")
		return
	}

	writeSuccess("Category deleted")
}

// AdminActivityTemplateHandler for admin activity page for GET requests
func (h *HandlersMap) AdminActivityTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/admin/activity.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminActivityTemplateData{
		Title:         "MapCTF Admin: Activity",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	activity, err := h.Logs.AllActivity(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading activity")
	} else {
		templateData.Activity = activity
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminAnnouncementsTemplateHandler for admin announcements page for GET requests
func (h *HandlersMap) AdminAnnouncementsTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/admin/announcements.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminAnnouncementsTemplateData{
		Title:         "MapCTF Admin: Announcements",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	announcements, err := h.Logs.AllAnnouncements(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading announcements")
	} else {
		templateData.Announcements = announcements
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminCountriesTemplateHandler for admin countries page for GET requests
func (h *HandlersMap) AdminCountriesTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/admin/countries.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := AdminCountriesTemplateData{
		Title:         "MapCTF Admin: Countries",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
		ChallengeName: make(map[uint]string),
		CountryFlag:   make(map[string]string),
	}
	var countriesList []countries.MapCountry
	if h.Countries != nil {
		countriesList, err = h.Countries.GetAll()
	} else {
		err = errors.New("countries manager not initialized")
	}
	if err != nil {
		log.Warn().Err(err).Msg("error loading countries")
	} else {
		templateData.Countries = countriesList
		for _, country := range countriesList {
			templateData.CountryFlag[country.CountryCode] = countryCodeToFlagEmoji(country.CountryCode)
		}
	}
	challengesList, err := h.Challenges.GetAll(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading challenges for countries view")
	} else {
		for _, challenge := range challengesList {
			templateData.ChallengeName[challenge.ID] = challenge.Title
		}
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminCountryUpdatePOSTHandler for updating country active status via POST requests
func (h *HandlersMap) AdminCountryUpdatePOSTHandler(w http.ResponseWriter, r *http.Request) {
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
	redirectBase := "/" + uuid + "/admin/countries"
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
	countryIDParam := strings.TrimSpace(chi.URLParam(r, "id"))
	if countryIDParam == "" {
		writeError(http.StatusBadRequest, "Missing country ID")
		return
	}
	countryID, err := strconv.ParseUint(countryIDParam, 10, 32)
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid country ID")
		return
	}
	if h.Countries == nil {
		writeError(http.StatusInternalServerError, "Countries manager is not initialized")
		return
	}
	activeValue := "false"
	if strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		var req struct {
			Active bool `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		if req.Active {
			activeValue = "true"
		}
	} else {
		if err := r.ParseForm(); err != nil {
			writeError(http.StatusBadRequest, "Invalid form payload")
			return
		}
		activeValue = strings.TrimSpace(r.FormValue("active"))
		if activeValue == "" {
			activeValue = "false"
		}
	}
	active, err := strconv.ParseBool(strings.ToLower(activeValue))
	if err != nil {
		writeError(http.StatusBadRequest, "Invalid active value")
		return
	}
	if err := h.Countries.SetActiveByID(uint(countryID), active); err != nil {
		log.Err(err).Msg("error updating country active status")
		writeError(http.StatusInternalServerError, "Failed to update country status")
		return
	}
	writeSuccess("Country status updated")
}

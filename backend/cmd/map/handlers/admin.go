package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io"
	"net/http"
	"net/mail"
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
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/chat"
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

type adminChatVisibilityRequest struct {
	Hidden json.RawMessage `json:"hidden"`
}

func (h *HandlersMap) createAdminActivityLog(r *http.Request, action, message string, challengeID uint) {
	h.createAdminActivityLogVisible(r, true, action, message, challengeID)
}

func (h *HandlersMap) createAdminActivityLogVisible(r *http.Request, visible bool, action, message string, challengeID uint) {
	if h.Logs == nil || h.Sessions == nil {
		return
	}

	uuid := chi.URLParam(r, "uuid")
	username := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if username == "" {
		username = h.ServiceName
	}

	activity, err := h.Logs.NewActivity(visible, username, action, message, challengeID, uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error building admin activity log")
		return
	}
	if err := h.Logs.CreateActivity(activity); err != nil {
		log.Warn().Err(err).Msg("error creating admin activity log")
	}
}

func parseAdminChatHiddenValue(raw json.RawMessage) (bool, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false, errors.New("missing hidden value")
	}

	var boolValue bool
	if err := json.Unmarshal(raw, &boolValue); err == nil {
		return boolValue, nil
	}

	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		stringValue = strings.TrimSpace(stringValue)
		switch {
		case strings.EqualFold(stringValue, "true"):
			return true, nil
		case strings.EqualFold(stringValue, "false"):
			return false, nil
		}
	}

	return false, errors.New("invalid hidden value")
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
	URL         string `json:"url,omitempty"`
	Category    string `json:"category"`
	Country     string `json:"country"`
	Active      bool   `json:"active"`
	Points      int    `json:"points"`
	Bonus       int    `json:"bonus"`
	BonusDecay  int    `json:"bonus_decay"`
	HintPenalty int    `json:"hint_penalty"`
	HelpPenalty int    `json:"help_penalty"`
	Penalty     int    `json:"penalty,omitempty"`
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

type adminGameTransferPayload struct {
	Version    int                            `json:"version"`
	ExportedAt string                         `json:"exported_at"`
	Settings   adminSettingsTransferPayload   `json:"settings"`
	Users      adminUsersTransferPayload      `json:"users"`
	Teams      adminTeamsTransferPayload      `json:"teams"`
	Challenges adminChallengesTransferPayload `json:"challenges"`
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
	logoSlugCleaner          = regexp.MustCompile(`[^a-z0-9-]+`)
	logoSlugMultiDash        = regexp.MustCompile(`-+`)
	customLogoAssetPathRegex = regexp.MustCompile(`^/static/img/team-logos/badge-[a-z0-9-]+\.(gif|jpe?g|png|svg)$`)
	svgScriptTagPattern      = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	svgViewBoxPatternD       = regexp.MustCompile(`(?i)viewBox\s*=\s*"([^"]+)"`)
	svgViewBoxPatternS       = regexp.MustCompile(`(?i)viewBox\s*=\s*'([^']+)'`)
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

func normalizeLogoValue(logo string) string {
	logo = strings.TrimSpace(logo)
	if logo == "" {
		return "invader"
	}

	if isCustomLogoAssetPath(logo) {
		if !strings.HasPrefix(logo, "/") {
			logo = "/" + logo
		}
		return logo
	}

	return normalizeLogoSymbolName(logo)
}

func isCustomLogoAssetPath(logo string) bool {
	logo = strings.TrimSpace(logo)
	if strings.HasPrefix(logo, "static/") {
		logo = "/" + logo
	}
	return customLogoAssetPathRegex.MatchString(logo)
}

func logoFilePath(custom bool, logo string) string {
	logo = normalizeLogoValue(logo)
	if isCustomLogoAssetPath(logo) {
		return logo
	}
	slug := normalizeLogoSymbolName(logo)
	if custom {
		return "/static/svg/icons/custom/badge-" + slug + ".svg"
	}
	return "/static/svg/icons/badges/badge-" + slug + ".svg"
}

func (h *HandlersMap) adminTemplateFuncs(r *http.Request) template.FuncMap {
	return template.FuncMap{
		"logoFilePath": logoFilePath,
		"logoIsImage":  isCustomLogoAssetPath,
		"logoSymbol":   normalizeLogoSymbolName,
		"T":            h.T(r.Context()),
	}
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

func customLogoAssetPath(slug, ext string) string {
	return "/static/img/team-logos/badge-" + slug + ext
}

func rasterLogoContentTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return ""
	}
}

func validateRasterLogoUpload(ext string, data []byte) error {
	expectedContentType := rasterLogoContentTypeForExt(ext)
	if expectedContentType == "" {
		return fmt.Errorf("unsupported raster logo extension")
	}
	contentType := http.DetectContentType(data)
	if contentType != expectedContentType {
		return fmt.Errorf("uploaded logo content type %s does not match %s", contentType, expectedContentType)
	}
	return nil
}

func saveUploadedLogoAssetFile(staticDir, slug, ext string, data []byte) error {
	if ext == "" {
		return fmt.Errorf("missing custom logo extension")
	}
	customDir := filepath.Join(staticDir, "img", "team-logos")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		return fmt.Errorf("failed to create custom logo dir: %w", err)
	}
	outputPath := filepath.Join(customDir, "badge-"+slug+ext)
	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("custom logo file already exists: %w", err)
		}
		return fmt.Errorf("failed to save custom logo file: %w", err)
	}
	defer file.Close()

	if n, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write custom logo file: %w", err)
	} else if n != len(data) {
		return fmt.Errorf("failed to write custom logo file: %w", io.ErrShortWrite)
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

func (h *HandlersMap) decodeGameImportPayload(r *http.Request, payload *adminGameTransferPayload) error {
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
	t, err := template.New("index.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/index.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminTemplateData{
		Title:         tr("admin.title.dashboard"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
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
	t, err := template.New("settings.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/settings.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminSettingsTemplateData{
		Title:         tr("admin.title.settings"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}

	loginEnabled, err := h.Settings.GetLoginEnabled(uuid)
	if err == nil {
		templateData.LoginEnabled = loginEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading login_enabled")
	}

	loginStrongPasswords, err := h.Settings.GetLoginStrongPasswords(uuid)
	if err == nil {
		templateData.LoginStrongPasswords = loginStrongPasswords
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading login_strong_passwords")
	}

	registrationEnabled, err := h.Settings.GetRegistrationEnabled(uuid)
	if err == nil {
		templateData.RegistrationEnabled = registrationEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_enabled")
	}

	registrationNames, err := h.Settings.GetRegistrationNames(uuid)
	if err == nil {
		templateData.RegistrationNames = registrationNames
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_names")
	}

	registrationEmails, err := h.Settings.GetRegistrationEmails(uuid)
	if err == nil {
		templateData.RegistrationEmails = registrationEmails
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_emails")
	}

	registrationType, err := h.Settings.GetRegistrationType(uuid)
	if err == nil {
		templateData.RegistrationType = registrationType
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_type")
	}

	registrationToken, err := h.Settings.GetRegistrationToken(uuid)
	if err == nil {
		templateData.RegistrationToken = registrationToken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading registration_token")
	}

	scoringEnabled, err := h.Settings.GetScoringEnabled(uuid)
	if err == nil {
		templateData.ScoringEnabled = scoringEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading scoring_enabled")
	}

	scoringHints, err := h.Settings.GetScoringHints(uuid)
	if err == nil {
		templateData.ScoringHints = scoringHints
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading scoring_hints")
	}

	scoringHelp, err := h.Settings.GetScoringHelp(uuid)
	if err == nil {
		templateData.ScoringHelp = scoringHelp
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading scoring_help")
	}

	gamePaused, err := h.Settings.GetGamePaused(uuid)
	if err == nil {
		templateData.GamePaused = gamePaused
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_paused")
	}

	gameStarted, err := h.Settings.GetGameStarted(uuid)
	if err == nil {
		templateData.GameStarted = gameStarted
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_started")
	}

	customOrg, err := h.Settings.GetCustomOrg(uuid)
	if err == nil {
		templateData.CustomOrg = customOrg
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading custom_org")
	}

	customLogo, err := h.Settings.GetCustomLogo(uuid)
	if err == nil {
		templateData.CustomLogo = customLogo
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading custom_logo")
	}

	language, err := h.Settings.GetLanguage(uuid)
	if err == nil {
		templateData.Language = language
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading language")
	}

	leaderboardLimit, err := h.Settings.GetLeaderboardLimit(uuid)
	if err == nil {
		templateData.LeaderboardLimit = leaderboardLimit
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading leaderboard_limit")
	}

	gameboardShowTeamMembers, err := h.Settings.GetGameboardShowTeamMembers(uuid)
	if err == nil {
		templateData.GameboardShowTeamMembers = gameboardShowTeamMembers
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading gameboard_show_team_members")
	}

	gameStartTime, err := h.Settings.GetGameStartTime(uuid)
	if err == nil && !gameStartTime.IsZero() {
		templateData.GameStartTime = gameStartTime.Format("2006-01-02T15:04")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_start_time")
	}

	gameEndTime, err := h.Settings.GetGameEndTime(uuid)
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
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Err(err).Msg("error parsing admin settings form")
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_form_payload"))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.missing_setting_name"))
		return
	}

	setBoolSetting := func(setter func(bool, string, string) error, setting string) bool {
		parsed, err := strconv.ParseBool(strings.ToLower(settingValue))
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_boolean_for", setting))
			return false
		}
		if err := setter(parsed, username, uuid); err != nil {
			log.Err(err).Msgf("error updating %s", setting)
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_setting", setting))
			return false
		}
		return true
	}
	setIntSetting := func(setter func(int, string, string) error, setting string) bool {
		parsed, err := strconv.Atoi(settingValue)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_integer_for", setting))
			return false
		}
		if err := setter(parsed, username, uuid); err != nil {
			log.Err(err).Msgf("error updating %s", setting)
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_setting", setting))
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
		if err := h.Settings.SetRegistrationToken(settingValue, username, uuid); err != nil {
			log.Err(err).Msg("error updating registration_token")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_reg_token"))
			return
		}
	case "scoring_enabled":
		if !setBoolSetting(h.Settings.SetScoringEnabled, settingName) {
			return
		}
	case "scoring_hints":
		if !setBoolSetting(h.Settings.SetScoringHints, settingName) {
			return
		}
	case "scoring_help":
		if !setBoolSetting(h.Settings.SetScoringHelp, settingName) {
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
		if err := h.Settings.SetCustomOrg(settingValue, username, uuid); err != nil {
			log.Err(err).Msg("error updating custom_org")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_custom_org"))
			return
		}
	case "custom_logo":
		if err := h.Settings.SetCustomLogo(settingValue, username, uuid); err != nil {
			log.Err(err).Msg("error updating custom_logo")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_custom_logo"))
			return
		}
	case "language":
		if err := h.Settings.SetLanguage(settingValue, username, uuid); err != nil {
			log.Err(err).Msg("error updating language")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_language"))
			return
		}
	case "leaderboard_limit":
		if !setIntSetting(h.Settings.SetLeaderboardLimit, settingName) {
			return
		}
	case "gameboard_show_team_members":
		if !setBoolSetting(h.Settings.SetGameboardShowTeamMembers, settingName) {
			return
		}
	case "game_start_time":
		gameStartTime, err := time.ParseInLocation("2006-01-02T15:04", settingValue, time.Local)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_game_start_format"))
			return
		}
		if err := h.Settings.SetGameStartTime(gameStartTime, username, uuid); err != nil {
			log.Err(err).Msg("error updating game_start_time")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_game_start"))
			return
		}
	case "game_end_time":
		gameEndTime, err := time.ParseInLocation("2006-01-02T15:04", settingValue, time.Local)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_game_end_format"))
			return
		}
		if err := h.Settings.SetGameEndTime(gameEndTime, username, uuid); err != nil {
			log.Err(err).Msg("error updating game_end_time")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_game_end"))
			return
		}
	default:
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.unsupported_setting"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.updated_setting", settingName))
}

func (h *HandlersMap) buildAdminSettingsTransferPayload(uuid string) (adminSettingsTransferPayload, error) {
	settingsList, err := h.Settings.GetAll(uuid)
	if err != nil {
		return adminSettingsTransferPayload{}, err
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

	return payload, nil
}

func (h *HandlersMap) buildAdminUsersTransferPayload(uuid string) (adminUsersTransferPayload, error) {
	usersList, err := h.Users.GetAll(uuid)
	if err != nil {
		return adminUsersTransferPayload{}, err
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

	return payload, nil
}

func (h *HandlersMap) buildAdminTeamsTransferPayload(uuid string) (adminTeamsTransferPayload, error) {
	teamsList, err := h.Teams.GetAll(uuid)
	if err != nil {
		return adminTeamsTransferPayload{}, err
	}

	var logosList []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&logosList).Error; err != nil {
		return adminTeamsTransferPayload{}, err
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
			Logo:      normalizeLogoValue(strings.TrimSpace(logo.Logo)),
			Enabled:   logo.Enabled,
			Custom:    logo.Custom,
			Protected: logo.Protected,
			Used:      logo.Used,
		})
	}
	for _, team := range teamsList {
		payload.Teams = append(payload.Teams, adminTeamsTransferTeam{
			Name:      strings.TrimSpace(team.Name),
			Logo:      normalizeLogoValue(strings.TrimSpace(team.Logo)),
			Active:    team.Active,
			Visible:   team.Visible,
			Protected: team.Protected,
		})
	}

	return payload, nil
}

func (h *HandlersMap) buildAdminChallengesTransferPayload(uuid string) (adminChallengesTransferPayload, error) {
	categoriesList, err := h.Challenges.GetAllCategories(uuid)
	if err != nil {
		return adminChallengesTransferPayload{}, err
	}
	challengesList, err := h.Challenges.GetAll(uuid)
	if err != nil {
		return adminChallengesTransferPayload{}, err
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
		challengeURL, _ := challenges.NormalizeChallengeURL(challenge.URL)
		transferItem := adminChallengesTransferItem{
			Title:       strings.TrimSpace(challenge.Title),
			Description: strings.TrimSpace(challenge.Description),
			URL:         challengeURL,
			Country:     strings.ToUpper(strings.TrimSpace(challenge.Country)),
			Active:      challenge.Active,
			Points:      challenge.Points,
			Bonus:       challenge.Bonus,
			BonusDecay:  challenge.BonusDecay,
			HintPenalty: challenge.HintPenalty,
			HelpPenalty: challenge.HelpPenalty,
			Flag:        strings.TrimSpace(challenge.Flag),
			Hint:        strings.TrimSpace(challenge.Hint),
		}
		if category, ok := categoriesByID[challenge.CategoryID]; ok {
			transferItem.Category = category.Name
		}
		transferChallenges = append(transferChallenges, transferItem)
	}

	return adminChallengesTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Categories: transferCategories,
		Challenges: transferChallenges,
	}, nil
}

func (h *HandlersMap) importAdminSettingsFromPayload(uuid string, payload adminSettingsTransferPayload, username string) (int, int, error) {
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
			err = h.Settings.SetLoginEnabled(in.ValueBool, username, uuid)
		case "login_strong_passwords":
			err = h.Settings.SetLoginStrongPasswords(in.ValueBool, username, uuid)
		case "registration_enabled":
			err = h.Settings.SetRegistrationEnabled(in.ValueBool, username, uuid)
		case "registration_names":
			err = h.Settings.SetRegistrationNames(in.ValueBool, username, uuid)
		case "registration_emails":
			err = h.Settings.SetRegistrationEmails(in.ValueBool, username, uuid)
		case "registration_type":
			err = h.Settings.SetRegistrationType(in.ValueInt, username, uuid)
		case "registration_token":
			err = h.Settings.SetRegistrationToken(in.ValueString, username, uuid)
		case "scoring_enabled":
			err = h.Settings.SetScoringEnabled(in.ValueBool, username, uuid)
		case "scoring_hints":
			err = h.Settings.SetScoringHints(in.ValueBool, username, uuid)
		case "scoring_help":
			err = h.Settings.SetScoringHelp(in.ValueBool, username, uuid)
		case "game_paused":
			err = h.Settings.SetGamePaused(in.ValueBool, username, uuid)
		case "game_started":
			err = h.Settings.SetGameStarted(in.ValueBool, username, uuid)
		case "game_start_time":
			var t time.Time
			if strings.TrimSpace(in.ValueDate) != "" {
				t, err = time.Parse(time.RFC3339, strings.TrimSpace(in.ValueDate))
				if err != nil {
					t, err = time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(in.ValueDate), time.Local)
				}
				if err != nil {
					return updatedSettings, skippedSettings, fmt.Errorf("invalid game_start_time format in import")
				}
			}
			err = h.Settings.SetGameStartTime(t, username, uuid)
		case "game_end_time":
			var t time.Time
			if strings.TrimSpace(in.ValueDate) != "" {
				t, err = time.Parse(time.RFC3339, strings.TrimSpace(in.ValueDate))
				if err != nil {
					t, err = time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(in.ValueDate), time.Local)
				}
				if err != nil {
					return updatedSettings, skippedSettings, fmt.Errorf("invalid game_end_time format in import")
				}
			}
			err = h.Settings.SetGameEndTime(t, username, uuid)
		case "custom_org":
			err = h.Settings.SetCustomOrg(in.ValueString, username, uuid)
		case "custom_logo":
			err = h.Settings.SetCustomLogo(in.ValueString, username, uuid)
		case "language":
			err = h.Settings.SetLanguage(in.ValueString, username, uuid)
		case "leaderboard_limit":
			err = h.Settings.SetLeaderboardLimit(in.ValueInt, username, uuid)
		case "gameboard_show_team_members":
			err = h.Settings.SetGameboardShowTeamMembers(in.ValueBool, username, uuid)
		default:
			skippedSettings++
			continue
		}

		if err != nil {
			return updatedSettings, skippedSettings, fmt.Errorf("failed importing setting %s", name)
		}
		updatedSettings++
	}

	return updatedSettings, skippedSettings, nil
}

func (h *HandlersMap) importAdminUsersFromPayload(uuid string, inUsers []adminUsersTransferUser) (int, int, int, error) {
	createdUsers := 0
	updatedUsers := 0
	skippedUsers := 0

	for _, inUser := range inUsers {
		username := strings.TrimSpace(inUser.Username)
		if username == "" {
			skippedUsers++
			continue
		}

		teamID := inUser.TeamID
		if teamID != users.NoTeamID {
			var teamCount int64
			if err := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("id = ? AND uuid = ?", teamID, uuid).Count(&teamCount).Error; err != nil {
				return createdUsers, updatedUsers, skippedUsers, err
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
				return createdUsers, updatedUsers, skippedUsers, result.Error
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
			return createdUsers, updatedUsers, skippedUsers, err
		}
		createdUsers++
	}

	return createdUsers, updatedUsers, skippedUsers, nil
}

func (h *HandlersMap) importAdminChallengesFromPayload(uuid string, payload adminChallengesTransferPayload) (int, int, int, int, error) {
	existingCategories, err := h.Challenges.GetAllCategories(uuid)
	if err != nil {
		return 0, 0, 0, 0, err
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
			return 0, createdCategories, 0, 0, err
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
		hintPenalty := item.HintPenalty
		if hintPenalty == 0 && item.Penalty != 0 {
			hintPenalty = item.Penalty
		}
		challengeURL, err := challenges.NormalizeChallengeURL(item.URL)
		if err != nil {
			skippedChallenges++
			continue
		}

		challenge := h.Challenges.New(
			title,
			strings.TrimSpace(item.Description),
			challengeURL,
			categoryID,
			countryCode,
			item.Active,
			item.Points,
			item.Bonus,
			item.BonusDecay,
			hintPenalty,
			item.HelpPenalty,
			flag,
			strings.TrimSpace(item.Hint),
			uuid,
		)

		if err := h.Challenges.CreateAndReturn(&challenge); err != nil {
			return importedChallenges, createdCategories, skippedChallenges, unassignedCountries, err
		}
		if countryCode != "" {
			if err := h.Countries.AssignCountryToChallenge(countryCode, challenge.ID); err != nil {
				_ = h.Challenges.Delete(challenge.ID, uuid)
				return importedChallenges, createdCategories, skippedChallenges, unassignedCountries, err
			}
		}
		importedChallenges++
	}

	return importedChallenges, createdCategories, skippedChallenges, unassignedCountries, nil
}

// AdminGameExportHandler exports settings, users, teams/logos and challenges/categories as one JSON payload
func (h *HandlersMap) AdminGameExportHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	settingsPayload, err := h.buildAdminSettingsTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading settings for full-game export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_load_settings")})
		return
	}
	usersPayload, err := h.buildAdminUsersTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading users for full-game export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_load_users")})
		return
	}
	teamsPayload, err := h.buildAdminTeamsTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading teams for full-game export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_load_teams_logos")})
		return
	}
	challengesPayload, err := h.buildAdminChallengesTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenges for full-game export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_load_challenges")})
		return
	}

	payload := adminGameTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Settings:   settingsPayload,
		Users:      usersPayload,
		Teams:      teamsPayload,
		Challenges: challengesPayload,
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling full-game export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_export_json")})
		return
	}

	fileName := "mapctf-full-game-export-" + time.Now().UTC().Format("20060102-150405") + ".json"
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

// AdminGameImportHandler imports a full game payload with settings, users, teams/logos and challenges/categories
func (h *HandlersMap) AdminGameImportHandler(w http.ResponseWriter, r *http.Request) {
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("admin.msg.content_type_form"))
		return
	}

	var payload adminGameTransferPayload
	if err := h.decodeGameImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing full-game import payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_full_game_payload"))
		return
	}

	username := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if username == "" {
		username = h.ServiceName
	}

	settingsUpdated, settingsSkipped, err := h.importAdminSettingsFromPayload(uuid, payload.Settings, username)
	if err != nil {
		log.Err(err).Msg("error importing full-game settings")
		writeError(http.StatusBadRequest, err.Error())
		return
	}

	logosCreated, logosUpdated, logosSkipped, err := h.importAdminTeamLogosFromPayload(uuid, payload.Teams.Logos)
	if err != nil {
		log.Err(err).Msg("error importing full-game logos")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_importing_logos"))
		return
	}
	teamsCreated, teamsUpdated, teamsSkipped, err := h.importAdminTeamsFromPayload(uuid, payload.Teams.Teams)
	if err != nil {
		log.Err(err).Msg("error importing full-game teams")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_importing_teams"))
		return
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after full-game teams import")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.import_sync_fail"))
		return
	}

	usersCreated, usersUpdated, usersSkipped, err := h.importAdminUsersFromPayload(uuid, payload.Users.Users)
	if err != nil {
		log.Err(err).Msg("error importing full-game users")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_importing_users"))
		return
	}

	challengesImported, categoriesCreated, challengesSkipped, challengesNoCountry, err := h.importAdminChallengesFromPayload(uuid, payload.Challenges)
	if err != nil {
		log.Err(err).Msg("error importing full-game challenges")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_importing_challenges"))
		return
	}

	messageParts := []string{
		"settings updated " + strconv.Itoa(settingsUpdated),
		"logos created " + strconv.Itoa(logosCreated),
		"logos updated " + strconv.Itoa(logosUpdated),
		h.T(r.Context())("admin.msg.teams_created", teamsCreated),
		h.T(r.Context())("admin.msg.teams_updated", teamsUpdated),
		h.T(r.Context())("admin.msg.users_created", usersCreated),
		h.T(r.Context())("admin.msg.users_updated", usersUpdated),
		"challenges imported " + strconv.Itoa(challengesImported),
		"categories created " + strconv.Itoa(categoriesCreated),
	}
	if settingsSkipped > 0 {
		messageParts = append(messageParts, "settings skipped "+strconv.Itoa(settingsSkipped))
	}
	if logosSkipped > 0 {
		messageParts = append(messageParts, h.T(r.Context())("admin.msg.logos_skipped", logosSkipped))
	}
	if teamsSkipped > 0 {
		messageParts = append(messageParts, h.T(r.Context())("admin.msg.teams_skipped", teamsSkipped))
	}
	if usersSkipped > 0 {
		messageParts = append(messageParts, "users skipped "+strconv.Itoa(usersSkipped))
	}
	if challengesSkipped > 0 {
		messageParts = append(messageParts, "challenges skipped "+strconv.Itoa(challengesSkipped))
	}
	if challengesNoCountry > 0 {
		messageParts = append(messageParts, "challenges without country "+strconv.Itoa(challengesNoCountry))
	}

	writeSuccess(h.T(r.Context())("admin.msg.full_import_complete") + strings.Join(messageParts, ", "))
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

	payload, err := h.buildAdminSettingsTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading settings for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_load_settings"),
		})
		return
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("admin.msg.content_type_form"))
		return
	}

	var payload adminSettingsTransferPayload
	if err := h.decodeSettingsImportPayload(r, &payload); err != nil {
		log.Err(err).Msg("error parsing settings import payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_settings_payload"))
		return
	}

	username := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	if username == "" {
		username = h.ServiceName
	}

	updatedSettings, skippedSettings, err := h.importAdminSettingsFromPayload(uuid, payload, username)
	if err != nil {
		log.Err(err).Msg("error importing settings")
		writeError(http.StatusBadRequest, err.Error())
		return
	}

	messageParts := []string{"settings updated " + strconv.Itoa(updatedSettings)}
	if skippedSettings > 0 {
		messageParts = append(messageParts, "settings skipped "+strconv.Itoa(skippedSettings))
	}
	writeSuccess(h.T(r.Context())("admin.msg.import_complete") + strings.Join(messageParts, ", "))
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
		func() error { return h.Settings.SetLoginEnabled(false, username, uuid) },
		func() error { return h.Settings.SetLoginStrongPasswords(false, username, uuid) },
		func() error { return h.Settings.SetRegistrationEnabled(false, username, uuid) },
		func() error { return h.Settings.SetRegistrationNames(false, username, uuid) },
		func() error { return h.Settings.SetRegistrationEmails(false, username, uuid) },
		func() error { return h.Settings.SetRegistrationType(0, username, uuid) },
		func() error { return h.Settings.SetRegistrationToken("", username, uuid) },
		func() error { return h.Settings.SetScoringEnabled(false, username, uuid) },
		func() error { return h.Settings.SetScoringHints(false, username, uuid) },
		func() error { return h.Settings.SetScoringHelp(false, username, uuid) },
		func() error { return h.Settings.SetGamePaused(false, username, uuid) },
		func() error { return h.Settings.SetGameStarted(false, username, uuid) },
		func() error { return h.Settings.SetGameStartTime(defaultStartTime, username, uuid) },
		func() error { return h.Settings.SetGameEndTime(defaultEndTime, username, uuid) },
		func() error { return h.Settings.SetCustomOrg("", username, uuid) },
		func() error { return h.Settings.SetCustomLogo("", username, uuid) },
		func() error { return h.Settings.SetLanguage("en", username, uuid) },
		func() error { return h.Settings.SetLeaderboardLimit(10, username, uuid) },
		func() error { return h.Settings.SetGameboardShowTeamMembers(false, username, uuid) },
	}

	for _, update := range updates {
		if err := update(); err != nil {
			log.Err(err).Msg("error resetting settings to defaults")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_resetting_defaults"))
			return
		}
	}

	writeSuccess(h.T(r.Context())("admin.msg.settings_reset_defaults"))
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
	t, err := template.New("controls.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/controls.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminControlsTemplateData{
		Title:         tr("admin.title.controls"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
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
	t, err := template.New("teams.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/teams.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminTeamsTemplateData{
		Title:         tr("admin.title.teams"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	teamList, err := h.Teams.GetAll(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading teams")
	} else {
		for i := range teamList {
			teamList[i].Logo = normalizeLogoValue(teamList[i].Logo)
		}
		templateData.Teams = teamList
	}
	// Full catalog for logo dropdowns (include disabled so existing assignments stay selectable).
	logos, err := h.Teams.GetAllLogos(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading team logos")
	} else {
		for i := range logos {
			logos[i].Logo = normalizeLogoValue(logos[i].Logo)
		}
		templateData.Logos = logos
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

// AdminTeamLogosTemplateHandler serves the admin team logos management page.
func (h *HandlersMap) AdminTeamLogosTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	t, err := template.New("team-logos.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/team-logos.html")
	if err != nil {
		log.Err(err).Msg("error getting admin team-logos template")
		return
	}
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminTeamLogosTemplateData{
		Title:         tr("admin.title.team_logos"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
	}
	allLogos, err := h.Teams.GetAllLogos(uuid)
	if err != nil {
		log.Err(err).Msg("error getting admin team-logos template")
		return
	}
	platformLogos := make([]teams.TeamLogo, 0, len(allLogos))
	customLogos := make([]teams.TeamLogo, 0, len(allLogos))
	for i := range allLogos {
		allLogos[i].Logo = normalizeLogoValue(allLogos[i].Logo)
		if allLogos[i].Protected {
			platformLogos = append(platformLogos, allLogos[i])
			continue
		}
		customLogos = append(customLogos, allLogos[i])
	}
	templateData.AllLogos = allLogos
	templateData.CustomLogos = customLogos
	templateData.PlatformLogos = platformLogos
	t.Execute(w, templateData)
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}
	var req AdminTeamCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin teams JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}
	name := strings.TrimSpace(req.Name)
	logo := strings.TrimSpace(req.Logo)
	if name == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.team_name_required"))
		return
	}
	if _, err := h.Teams.Register(name, logo, uuid); err != nil {
		log.Err(err).Msg("error creating team")
		writeError(http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess(h.T(r.Context())("admin.msg.team_created"))
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
		logo := normalizeLogoValue(strings.TrimSpace(inLogo.Logo))
		if name == "" || logo == "" {
			skippedLogos++
			continue
		}
		key := strings.ToLower(name)
		if existing, ok := logosByName[key]; ok {
			if !existing.Custom {
				result := h.Teams.DB.Model(&teams.TeamLogo{}).
					Where("id = ? AND uuid = ?", existing.ID, uuid).
					Updates(map[string]interface{}{
						"enabled":   inLogo.Enabled,
						"protected": inLogo.Protected,
					})
				if result.Error != nil {
					return createdLogos, updatedLogos, skippedLogos, result.Error
				}
				updatedLogos++
				continue
			}
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

		if !inLogo.Custom {
			skippedLogos++
			continue
		}

		newLogo, err := h.Teams.NewLogo(name, logo, inLogo.Enabled, inLogo.Custom, 0, uuid)
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
		logo := normalizeLogoValue(strings.TrimSpace(inTeam.Logo))
		if logo == "" || strings.EqualFold(logo, "random") {
			randomLogo, err := h.Teams.RandomLogo(uuid)
			if err != nil {
				logo = "invader"
			} else {
				logo = normalizeLogoValue(randomLogo.Logo)
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

		newTeam, err := h.Teams.New(name, logo, inTeam.Protected, inTeam.Visible, uuid)
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

	payload, err := h.buildAdminTeamsTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading teams/logos for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_load_teams_logos"),
		})
		return
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling teams export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_export_json"),
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

	teamsList, err := h.Teams.GetAll(uuid)
	if err != nil {
		log.Err(err).Msg("error loading teams for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_load_teams"),
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
			Logo:      normalizeLogoValue(strings.TrimSpace(team.Logo)),
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
			Message: h.T(r.Context())("admin.msg.failed_export_json"),
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
			Message: h.T(r.Context())("admin.msg.failed_load_logos"),
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
			Logo:      normalizeLogoValue(strings.TrimSpace(logo.Logo)),
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
			Message: h.T(r.Context())("admin.msg.failed_export_json"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_import_payload"))
		return
	}
	if len(payload.Logos) == 0 && len(payload.Teams) == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.no_teams_or_logos_in_payload"))
		return
	}

	createdLogos, updatedLogos, skippedLogos, err := h.importAdminTeamLogosFromPayload(uuid, payload.Logos)
	if err != nil {
		log.Err(err).Msg("error importing logos from combined import")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_logos"))
		return
	}

	var existingTeams []teams.PlatformTeam
	if err := h.Teams.DB.Where("uuid = ?", uuid).Find(&existingTeams).Error; err != nil {
		log.Err(err).Msg("error loading existing teams for import")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_load_teams"))
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
		logo := normalizeLogoValue(strings.TrimSpace(inTeam.Logo))
		if logo == "" || strings.EqualFold(logo, "random") {
			randomLogo, err := h.Teams.RandomLogo(uuid)
			if err != nil {
				log.Err(err).Msg("error resolving random logo during team import")
				logo = "invader"
			} else {
				logo = normalizeLogoValue(randomLogo.Logo)
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
				writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_teams"))
				return
			}
			updatedTeams++
			continue
		}

		newTeam, err := h.Teams.New(name, logo, inTeam.Protected, inTeam.Visible, uuid)
		if err != nil {
			log.Err(err).Msg("error creating team object from import")
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.failed_import_teams"))
			return
		}
		newTeam.Active = inTeam.Active
		if err := h.Teams.Create(newTeam); err != nil {
			log.Err(err).Msg("error saving team from import")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_teams"))
			return
		}
		createdTeams++
	}

	messageParts := []string{
		"logos created " + strconv.Itoa(createdLogos),
		"logos updated " + strconv.Itoa(updatedLogos),
		h.T(r.Context())("admin.msg.teams_created", createdTeams),
		h.T(r.Context())("admin.msg.teams_updated", updatedTeams),
	}
	if skippedLogos > 0 {
		messageParts = append(messageParts, h.T(r.Context())("admin.msg.logos_skipped", skippedLogos))
	}
	if skippedTeams > 0 {
		messageParts = append(messageParts, h.T(r.Context())("admin.msg.teams_skipped", skippedTeams))
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after teams import")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.import_sync_fail"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.import_complete") + strings.Join(messageParts, ", "))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_import_payload"))
		return
	}
	if len(payload.Teams) == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.no_teams_in_payload"))
		return
	}

	createdTeams, updatedTeams, skippedTeams, err := h.importAdminTeamsFromPayload(uuid, payload.Teams)
	if err != nil {
		log.Err(err).Msg("error importing teams")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_teams"))
		return
	}

	messageParts := []string{
		h.T(r.Context())("admin.msg.teams_created", createdTeams),
		h.T(r.Context())("admin.msg.teams_updated", updatedTeams),
	}
	if skippedTeams > 0 {
		messageParts = append(messageParts, h.T(r.Context())("admin.msg.teams_skipped", skippedTeams))
	}
	writeSuccess(h.T(r.Context())("admin.msg.import_complete") + strings.Join(messageParts, ", "))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_import_payload"))
		return
	}
	if len(payload.Logos) == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.no_logos_in_payload"))
		return
	}

	createdLogos, updatedLogos, skippedLogos, err := h.importAdminTeamLogosFromPayload(uuid, payload.Logos)
	if err != nil {
		log.Err(err).Msg("error importing logos")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_logos"))
		return
	}

	messageParts := []string{
		"logos created " + strconv.Itoa(createdLogos),
		"logos updated " + strconv.Itoa(updatedLogos),
	}
	if skippedLogos > 0 {
		messageParts = append(messageParts, h.T(r.Context())("admin.msg.logos_skipped", skippedLogos))
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after logos import")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.import_sync_fail"))
		return
	}
	writeSuccess(h.T(r.Context())("admin.msg.import_complete") + strings.Join(messageParts, ", "))
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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_enable_teams")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.enabled_teams", result.RowsAffected)})
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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_disable_teams")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.disabled_teams", result.RowsAffected)})
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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_visible_teams")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.visible_teams", result.RowsAffected)})
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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_invisible_teams")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.invisible_teams", result.RowsAffected)})
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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_enable_logos")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.enabled_logos", result.RowsAffected)})
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
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_disable_logos")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.disabled_logos", result.RowsAffected)})
}

// AdminTeamLogosDeleteAllPOSTHandler deletes all custom team logos for the map UUID.
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
	var customs []teams.TeamLogo
	if err := h.Teams.DB.Where("uuid = ? AND custom = ?", uuid, true).Find(&customs).Error; err != nil {
		log.Err(err).Msg("error loading custom logos for delete-all")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_load_custom_logos")})
		return
	}
	for _, row := range customs {
		h.removeCustomUploadedLogoFile(row.Logo)
	}
	result := h.Teams.DB.Where("uuid = ? AND custom = ?", uuid, true).Delete(&teams.TeamLogo{})
	if result.Error != nil {
		log.Err(result.Error).Msg("error deleting custom logos")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.failed_delete_logos")})
		return
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after delete-all custom logos")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{Success: false, Status: "error", Message: h.T(r.Context())("admin.msg.logos_deleted_sync_fail")})
		return
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{Success: true, Status: "ok", Message: h.T(r.Context())("admin.msg.deleted_logos", result.RowsAffected)})
}

func (h *HandlersMap) removeCustomUploadedLogoFile(logoSymbol string) {
	staticDir := strings.TrimSpace(h.Config.Map.StaticDir)
	if staticDir == "" {
		return
	}
	if isCustomLogoAssetPath(logoSymbol) {
		logoPath := strings.TrimSpace(logoSymbol)
		logoPath = strings.TrimPrefix(logoPath, "/static/")
		logoPath = strings.TrimPrefix(logoPath, "static/")
		path := filepath.Join(staticDir, filepath.FromSlash(logoPath))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", path).Msg("failed to remove custom logo asset file")
		}
		return
	}
	slug := normalizeLogoSymbolName(logoSymbol)
	if slug == "" {
		return
	}
	path := filepath.Join(staticDir, "svg", "icons", "custom", "badge-"+slug+".svg")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Str("path", path).Msg("failed to remove custom logo file")
	}
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_teams"))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_teams"))
		return
	}
	if err := tx.Where("uuid = ?", uuid).Delete(&teams.TeamMembership{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting memberships before bulk delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_teams"))
		return
	}
	if err := tx.Where("uuid = ?", uuid).Delete(&teams.TeamScore{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting scores before bulk delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_teams"))
		return
	}
	deleteResult := tx.Where("uuid = ?", uuid).Delete(&teams.PlatformTeam{})
	if deleteResult.Error != nil {
		tx.Rollback()
		log.Err(deleteResult.Error).Msg("error deleting all teams")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_teams"))
		return
	}
	if err := tx.Commit().Error; err != nil {
		log.Err(err).Msg("error committing delete-all-teams transaction")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_teams"))
		return
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after delete-all-teams")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.deleted_team_sync_fail"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.deleted_team_count", deleteResult.RowsAffected))
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
			Message: h.T(r.Context())("admin.msg.invalid_team_id"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminTeamUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin team update JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.team_name_required"))
		return
	}
	var duplicateCount int64
	if err := h.Teams.DB.Model(&teams.PlatformTeam{}).
		Where("uuid = ? AND name = ? AND id <> ?", uuid, name, uint(teamID)).
		Count(&duplicateCount).Error; err != nil {
		log.Err(err).Msg("error validating team name uniqueness")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_validate_team_name"))
		return
	}
	if duplicateCount > 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.team_name_already_exists"))
		return
	}

	logoInput := strings.TrimSpace(req.Logo)
	logo := normalizeLogoValue(logoInput)
	if logoInput == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_is_required"))
		return
	}
	if strings.EqualFold(logoInput, "random") {
		randomLogo, err := h.Teams.RandomLogo(uuid)
		if err != nil {
			log.Err(err).Msg("error getting random logo for team update")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_resolve_random_logo"))
			return
		}
		logo = normalizeLogoValue(randomLogo.Logo)
	}

	active, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Active)))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_active"))
		return
	}

	visible, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Visible)))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_visible"))
		return
	}

	protected, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Protected)))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_protected"))
		return
	}

	updateResult := h.Teams.DB.Model(&teams.PlatformTeam{}).
		Where("id = ? AND uuid = ?", uint(teamID), uuid).
		Updates(map[string]interface{}{
			"name":      name,
			"logo":      logo,
			"active":    active,
			"visible":   visible,
			"protected": protected,
		})
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating team")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_team"))
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.team_not_found"))
		return
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after team update")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.team_updated_sync_fail"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.team_updated"))
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
			Message: h.T(r.Context())("admin.msg.invalid_team_id"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}

	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", teamID, uuid).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.team_not_found"))
			return
		}
		log.Err(err).Msg("error loading team to delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_load_team"))
		return
	}

	tx := h.Teams.DB.Begin()
	if tx.Error != nil {
		log.Err(tx.Error).Msg("error starting team delete transaction")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_team"))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_team"))
		return
	}

	if err := tx.Where("team_id = ? AND uuid = ?", teamID, uuid).Delete(&teams.TeamMembership{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting team memberships before delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_team"))
		return
	}

	if err := tx.Where("team_id = ? AND uuid = ?", teamID, uuid).Delete(&teams.TeamScore{}).Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error deleting team scores before delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_team"))
		return
	}

	deleteResult := tx.Where("id = ? AND uuid = ?", teamID, uuid).Delete(&teams.PlatformTeam{})
	if deleteResult.Error != nil {
		tx.Rollback()
		log.Err(deleteResult.Error).Msg("error deleting team")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_team"))
		return
	}
	if deleteResult.RowsAffected == 0 {
		tx.Rollback()
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.team_not_found"))
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Err(err).Msg("error committing team delete transaction")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_team"))
		return
	}
	if err := h.Teams.SyncLogoUsage(uuid); err != nil {
		log.Err(err).Msg("error syncing logo usage after team delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.team_deleted_sync_fail"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.team_deleted"))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	contentType := strings.ToLower(r.Header.Get(ContentType))
	isJSON := strings.Contains(contentType, JSONApplication)
	isMultipart := strings.Contains(contentType, "multipart/form-data")
	if !isJSON && !isMultipart {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("admin.msg.content_type_form"))
		return
	}

	var (
		name              string
		rawLogo           string
		uploadedLogoSlug  string
		uploadedLogoData  []byte
		uploadedLogoExt   string
		uploadedLogoAsset bool
	)

	if isMultipart {
		if err := r.ParseMultipartForm(maxCustomLogoUploadBytes * 2); err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_multipart"))
			return
		}
		name = strings.TrimSpace(r.FormValue("name"))
		rawLogo = strings.TrimSpace(r.FormValue("logo"))

		file, fileHeader, err := r.FormFile("logo_file")
		if err == nil && file != nil {
			defer file.Close()

			uploadData, readErr := io.ReadAll(io.LimitReader(file, maxCustomLogoUploadBytes+1))
			if readErr != nil {
				writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.failed_read_logo_file"))
				return
			}
			if int64(len(uploadData)) > maxCustomLogoUploadBytes {
				writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.uploaded_logo_too_large"))
				return
			}

			slugInput := strings.TrimSpace(r.FormValue("logo_slug"))
			if slugInput == "" && fileHeader != nil {
				slugInput = strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename))
			}
			slug := sanitizeLogoSlug(slugInput)
			if slug == "" {
				writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_logo_slug"))
				return
			}

			uploadedLogoSlug = slug
			uploadedLogoData = uploadData

			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			if rasterLogoContentTypeForExt(ext) != "" {
				if rasterErr := validateRasterLogoUpload(ext, uploadData); rasterErr != nil {
					writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_raster_logo"))
					return
				}
				uploadedLogoExt = ext
				uploadedLogoAsset = true
				rawLogo = customLogoAssetPath(slug, ext)
			} else if ext != "" && ext != ".svg" {
				writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_uploaded_logo_type"))
				return
			} else {
				if _, svgErr := buildUploadedLogoSymbol(slug, uploadData); svgErr != nil {
					writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_svg"))
					return
				}
				uploadedLogoData = []byte(svgScriptTagPattern.ReplaceAllString(string(uploadData), ""))
				uploadedLogoExt = ".svg"
				uploadedLogoAsset = true
				rawLogo = customLogoAssetPath(slug, uploadedLogoExt)
			}
		} else if err != nil {
			if errors.Is(err, http.ErrMissingFile) {
				writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_file_required"))
				return
			}
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_uploaded_logo"))
			return
		}
	} else {
		var req AdminLogoCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Err(err).Msg("error parsing admin logos JSON payload")
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
			return
		}
		name = strings.TrimSpace(req.Name)
		rawLogo = strings.TrimSpace(req.Logo)
	}

	if name == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_name_required"))
		return
	}
	if strings.EqualFold(rawLogo, "random") {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.random_not_valid_logo"))
		return
	}
	logo := normalizeLogoValue(rawLogo)
	if rawLogo == "" || logo == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_symbol_required"))
		return
	}
	if h.Teams.ExistsLogo(name, uuid) {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_already_exists"))
		return
	}
	if uploadedLogoSlug != "" && uploadedLogoAsset {
		if err := saveUploadedLogoAssetFile(h.Config.Map.StaticDir, uploadedLogoSlug, uploadedLogoExt, uploadedLogoData); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "already exists") {
				writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_file_already_exists"))
				return
			}
			log.Err(err).Msg("error saving custom logo file")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_store_logo_file"))
			return
		}
	}

	newLogo, err := h.Teams.NewLogo(name, logo, true, true, 0, uuid)
	if err != nil {
		log.Err(err).Msg("error creating logo object")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.failed_create_logo"))
		return
	}
	if err := h.Teams.CreateLogo(newLogo); err != nil {
		log.Err(err).Msg("error saving logo")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_create_logo"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.logo_created"))
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
			Message: h.T(r.Context())("admin.msg.invalid_logo_id"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminLogoUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin logo update JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.logo_name_required"))
		return
	}

	enabled, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Enabled)))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_enabled"))
		return
	}

	protected, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(req.Protected)))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_protected"))
		return
	}

	var existing teams.TeamLogo
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", uint(logoID), uuid).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.logo_not_found"))
			return
		}
		log.Err(err).Msg("error loading logo for update")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_load_logo"))
		return
	}

	if !existing.Custom && name != strings.TrimSpace(existing.Name) {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.platform_logo_no_rename"))
		return
	}

	updates := map[string]interface{}{
		"enabled":   enabled,
		"protected": protected,
	}
	if existing.Custom {
		updates["name"] = name
	}
	updateResult := h.Teams.DB.Model(&teams.TeamLogo{}).
		Where("id = ? AND uuid = ?", uint(logoID), uuid).
		Updates(updates)
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating logo")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_logo"))
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.logo_not_found"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.logo_updated"))
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
	t, err := template.New("users.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/users.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminUsersTemplateData{
		Title:         tr("admin.title.users"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
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
	teams, err := h.Teams.GetAll(uuid)
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin users JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.user_pass_required"))
		return
	}

	teamID := uint(0)
	if teamIDStr != "" {
		parsedTeamID, err := strconv.ParseUint(teamIDStr, 10, 64)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_team_id"))
			return
		}
		teamID = uint(parsedTeamID)
	}

	admin := false
	if adminStr != "" {
		parsedAdmin, err := strconv.ParseBool(strings.ToLower(adminStr))
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_admin"))
			return
		}
		admin = parsedAdmin
	}

	service := false
	if serviceStr != "" {
		parsedService, err := strconv.ParseBool(strings.ToLower(serviceStr))
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_service"))
			return
		}
		service = parsedService
	}

	active := true
	if activeStr != "" {
		parsedActive, err := strconv.ParseBool(strings.ToLower(activeStr))
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_active"))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_create_user"))
		return
	}

	if !active {
		if err := h.Users.SetActive(false, username, uuid); err != nil {
			log.Err(err).Msg("error setting active flag for new user")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.user_created_active_fail"))
			return
		}
	}

	writeSuccess(h.T(r.Context())("admin.msg.user_created"))
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
			Message: h.T(r.Context())("admin.msg.invalid_user_id"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin user update JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email != "" {
			parsedEmail, err := mail.ParseAddress(email)
			if err != nil || parsedEmail.Address != email {
				writeError(http.StatusBadRequest, h.T(r.Context())("profile.email_invalid"))
				return
			}
		}
		updates["email"] = email
	}

	teamIDStr := strings.TrimSpace(req.TeamID)
	if teamIDStr == "" {
		teamIDStr = "0"
	}
	adminStr := strings.TrimSpace(req.Admin)
	serviceStr := strings.TrimSpace(req.Service)
	activeStr := strings.TrimSpace(req.Active)
	if adminStr == "" || serviceStr == "" || activeStr == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.admin_service_active_required"))
		return
	}

	teamID64, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_team_id"))
		return
	}
	teamID := uint(teamID64)
	adminValue, err := strconv.ParseBool(strings.ToLower(adminStr))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_admin"))
		return
	}
	serviceValue, err := strconv.ParseBool(strings.ToLower(serviceStr))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_service"))
		return
	}
	activeValue, err := strconv.ParseBool(strings.ToLower(activeStr))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_active"))
		return
	}

	if teamID != users.NoTeamID {
		var teamCount int64
		if err := h.Teams.DB.Model(&teams.PlatformTeam{}).Where("id = ? AND uuid = ?", teamID, uuid).Count(&teamCount).Error; err != nil {
			log.Err(err).Msg("error validating team for user update")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_validate_team"))
			return
		}
		if teamCount == 0 {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.team_not_found"))
			return
		}
	}

	updates["team_id"] = teamID
	updates["admin"] = adminValue
	updates["service"] = serviceValue
	updates["active"] = activeValue

	updateResult := h.Users.DB.Model(&users.PlatformUser{}).
		Where("id = ? AND uuid = ?", uint(userID), uuid).
		Updates(updates)
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating user")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_user"))
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.user_not_found"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.user_updated"))
}

// AdminUserPasswordPOSTHandler resets a user's password from the admin view
func (h *HandlersMap) AdminUserPasswordPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, false)
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
			Message: h.T(r.Context())("admin.msg.invalid_user_id"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin user password JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	switch {
	case strings.TrimSpace(req.NewPassword) == "":
		writeError(http.StatusBadRequest, h.T(r.Context())("profile.new_required"))
		return
	}

	passHash, err := h.Users.HashPasswordWithSalt(req.NewPassword)
	if err != nil {
		log.Err(err).Msg("error hashing admin user password")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_password"))
		return
	}

	updateResult := h.Users.DB.Model(&users.PlatformUser{}).
		Where("id = ? AND uuid = ?", uint(userID), uuid).
		Update("pass_hash", passHash)
	if updateResult.Error != nil {
		log.Err(updateResult.Error).Msg("error updating admin user password")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_password"))
		return
	}
	if updateResult.RowsAffected == 0 {
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.user_not_found"))
		return
	}

	writeSuccess(h.T(r.Context())("profile.password_updated_msg"))
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

	payload, err := h.buildAdminUsersTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading users for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_load_users"),
		})
		return
	}

	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling users export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_export_json"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_import_payload"))
		return
	}
	if len(payload.Users) == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.no_users_in_payload"))
		return
	}

	createdUsers, updatedUsers, skippedUsers, err := h.importAdminUsersFromPayload(uuid, payload.Users)
	if err != nil {
		log.Err(err).Msg("error importing users")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_users"))
		return
	}

	messageParts := []string{
		"users created " + strconv.Itoa(createdUsers),
		"users updated " + strconv.Itoa(updatedUsers),
	}
	if skippedUsers > 0 {
		messageParts = append(messageParts, "users skipped "+strconv.Itoa(skippedUsers))
	}

	writeSuccess(h.T(r.Context())("admin.msg.import_complete") + strings.Join(messageParts, ", "))
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
			Message: h.T(r.Context())("admin.msg.failed_enable_users"),
		})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: h.T(r.Context())("admin.msg.enabled_users", result.RowsAffected),
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
			Message: h.T(r.Context())("admin.msg.not_authenticated"),
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
			Message: h.T(r.Context())("admin.msg.failed_disable_users"),
		})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: h.T(r.Context())("admin.msg.disabled_users", result.RowsAffected),
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
			Message: h.T(r.Context())("admin.msg.not_authenticated"),
		})
		return
	}

	result := h.Users.DB.Where("uuid = ? AND username <> ?", uuid, currentUsername).Delete(&users.PlatformUser{})
	if result.Error != nil {
		log.Err(result.Error).Msg("error deleting all users")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_delete_users"),
		})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: h.T(r.Context())("admin.msg.deleted_users", result.RowsAffected),
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

	payload, err := h.buildAdminChallengesTransferPayload(uuid)
	if err != nil {
		log.Err(err).Msg("error loading categories/challenges for export")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_load_challenges"),
		})
		return
	}
	output, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Err(err).Msg("error marshaling challenges export JSON")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, adminActionResponse{
			Success: false,
			Status:  "error",
			Message: h.T(r.Context())("admin.msg.failed_export_json"),
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_import_payload"))
		return
	}
	if len(payload.Challenges) == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.no_challenges_in_payload"))
		return
	}

	importedChallenges, createdCategories, skippedChallenges, unassignedCountries, err := h.importAdminChallengesFromPayload(uuid, payload)
	if err != nil {
		log.Err(err).Msg("error importing challenges")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_import_challenges"))
		return
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_load_challenges"))
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
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_release_country_refs"))
			return
		}
		if !exists {
			continue
		}
		if err := h.Countries.ReleaseCountry(countryCode); err != nil {
			log.Err(err).Msg("error releasing challenge country before bulk delete")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_release_country_refs"))
			return
		}
	}

	deletedCount, err := h.Challenges.DeleteAll(uuid)
	if err != nil {
		log.Err(err).Msg("error deleting all challenges")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_challenges"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.deleted_challenge_count", deletedCount))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_enable_all_challenges"))
		return
	}

	if updatedCount > 0 {
		h.createAdminActivityLog(r, "enabled", fmt.Sprintf("enabled all challenges (%d)", updatedCount), 0)
	}

	writeSuccess(h.T(r.Context())("admin.msg.enabled_challenge_count", updatedCount))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_disable_all_challenges"))
		return
	}

	if updatedCount > 0 {
		h.createAdminActivityLog(r, "disabled", fmt.Sprintf("disabled all challenges (%d)", updatedCount), 0)
	}

	writeSuccess(h.T(r.Context())("admin.msg.disabled_challenge_count", updatedCount))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_load_challenges"))
		return
	}
	if len(challengesList) > 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.delete_challenges_before_categories"))
		return
	}

	deletedCount, err := h.Challenges.DeleteAllCategories(uuid)
	if err != nil {
		log.Err(err).Msg("error deleting all categories")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_categories"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.deleted_category_count", deletedCount))
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
	t, err := template.New("challenges.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/challenges.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminChallengesTemplateData{
		Title:         tr("admin.title.challenges"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
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

	templateData.ChallengeActivity = make(map[uint][]AdminChallengeActivityEntry, len(templateData.Challenges))
	teamNamesByID := make(map[uint]string)
	if h.Teams != nil {
		allTeams, teamErr := h.Teams.GetAll(uuid)
		if teamErr != nil {
			log.Warn().Err(teamErr).Msg("error loading teams for challenge activity")
		} else {
			for _, team := range allTeams {
				teamNamesByID[team.ID] = team.Name
			}
		}
	}
	activity, err := h.Logs.AllActivity(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading activity for challenge view")
	}
	hintsLogs, err := h.Logs.AllHintsLogs(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading hint logs for challenge view")
		hintsLogs = nil
	}
	failuresLogs, err := h.Logs.AllFailuresLogs(uuid)
	if err != nil {
		log.Warn().Err(err).Msg("error loading failure logs for challenge view")
		failuresLogs = nil
	}
	for i := range templateData.Challenges {
		challenge := templateData.Challenges[i]
		challengeTitle := strings.ToLower(strings.TrimSpace(challenge.Title))
		challengeID := strconv.Itoa(int(challenge.ID))

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

		for _, entry := range activity {
			searchText := strings.ToLower(strings.TrimSpace(entry.Subject + " " + entry.Action + " " + entry.Message))
			if strings.Contains(searchText, challengeTitle) ||
				strconv.FormatUint(uint64(entry.ChallengeID), 10) == challengeID {
				templateData.ChallengeActivity[challenge.ID] = append(templateData.ChallengeActivity[challenge.ID], AdminChallengeActivityEntry{
					Label:   "Activity",
					Subject: entry.Subject,
					Action:  entry.Action,
					Message: entry.Message,
					At:      entry.CreatedAt,
				})
			}
		}

		for _, hintEntry := range hintsLogs {
			if hintEntry.ChallengeID != challenge.ID {
				continue
			}
			templateData.ChallengeActivity[challenge.ID] = append(templateData.ChallengeActivity[challenge.ID], AdminChallengeActivityEntry{
				Label:     "Hint",
				Subject:   teamNamesByID[hintEntry.TeamID],
				Action:    "hint",
				Message:   h.T(r.Context())("hint.requested"),
				Arguments: "penalty=" + strconv.Itoa(hintEntry.Penalty),
				At:        hintEntry.CreatedAt,
			})
		}

		for _, failureEntry := range failuresLogs {
			if failureEntry.ChallengeID != challenge.ID {
				continue
			}
			templateData.ChallengeActivity[challenge.ID] = append(templateData.ChallengeActivity[challenge.ID], AdminChallengeActivityEntry{
				Label:     "Failure",
				Subject:   teamNamesByID[failureEntry.TeamID],
				Action:    "failure",
				Message:   h.T(r.Context())("score.incorrect_submission"),
				Arguments: "flag=" + failureEntry.Flag,
				At:        failureEntry.CreatedAt,
			})
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminChallengeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin challenges JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	challengeURL, err := challenges.NormalizeChallengeURL(req.URL)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_challenge_url"))
		return
	}
	categoryIDStr := strings.TrimSpace(req.CategoryID)
	country := strings.ToUpper(strings.TrimSpace(req.Country))
	activeStr := strings.TrimSpace(req.Active)
	pointsStr := strings.TrimSpace(req.Points)
	bonusStr := strings.TrimSpace(req.Bonus)
	bonusDecayStr := strings.TrimSpace(req.BonusDecay)
	hintPenaltyStr := strings.TrimSpace(req.HintPenalty)
	helpPenaltyStr := strings.TrimSpace(req.HelpPenalty)
	flag := strings.TrimSpace(req.Flag)
	hint := strings.TrimSpace(req.Hint)

	if title == "" || flag == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.title_flag_required"))
		return
	}
	if hintPenaltyStr == "" {
		hintPenaltyStr = "0"
	}
	if helpPenaltyStr == "" {
		helpPenaltyStr = "0"
	}
	if country != "" {
		selectedCountry, err := h.Countries.GetByCode(country)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_country_code"))
			return
		}
		if !selectedCountry.Active || selectedCountry.Assigned {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.country_active_available"))
			return
		}
	}
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.select_category"))
		return
	}
	category, err := h.Challenges.GetCategoryByID(uint(categoryID), uuid)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.select_valid_category"))
		return
	}
	active := false
	if activeStr != "" {
		parsedActive, err := strconv.ParseBool(activeStr)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_active"))
			return
		}
		active = parsedActive
	}
	points, err := strconv.ParseInt(pointsStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_points"))
		return
	}
	bonus, err := strconv.ParseInt(bonusStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_bonus"))
		return
	}
	bonusDecay, err := strconv.ParseInt(bonusDecayStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_bonus_decay"))
		return
	}
	hintPenalty, err := strconv.ParseInt(hintPenaltyStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_hint_penalty"))
		return
	}
	helpPenalty, err := strconv.ParseInt(helpPenaltyStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_help_penalty"))
		return
	}

	challenge := h.Challenges.New(
		title,
		description,
		challengeURL,
		uint(categoryID),
		country,
		active,
		int(points),
		int(bonus),
		int(bonusDecay),
		int(hintPenalty),
		int(helpPenalty),
		flag,
		hint,
		uuid,
	)

	if err := h.Challenges.CreateAndReturn(&challenge); err != nil {
		log.Err(err).Msg("error creating challenge")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_create_challenge"))
		return
	}
	if country != "" {
		if err := h.Countries.AssignCountryToChallenge(country, challenge.ID); err != nil {
			log.Err(err).Msg("error assigning country to challenge")
			_ = h.Challenges.Delete(challenge.ID, uuid)
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_assign_country"))
			return
		}
	}

	countryLabel := strings.TrimSpace(challenge.Country)
	if countryLabel == "" {
		countryLabel = strings.TrimSpace(challenge.Title)
	}
	createMsg := fmt.Sprintf(logs.ActivityCreateChallenge, countryLabel, category.Name, challenge.Points)
	h.createAdminActivityLogVisible(r, false, "created", createMsg, challenge.ID)

	writeSuccess(h.T(r.Context())("admin.msg.challenge_created"))
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	challengeIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	challengeID, err := strconv.ParseUint(challengeIDStr, 10, 64)
	if err != nil || challengeID == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_challenge_id"))
		return
	}

	var req AdminChallengeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin challenge update JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	challengeURL, err := challenges.NormalizeChallengeURL(req.URL)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_challenge_url"))
		return
	}
	categoryIDStr := strings.TrimSpace(req.CategoryID)
	country := strings.ToUpper(strings.TrimSpace(req.Country))
	activeStr := strings.TrimSpace(req.Active)
	pointsStr := strings.TrimSpace(req.Points)
	bonusStr := strings.TrimSpace(req.Bonus)
	bonusDecayStr := strings.TrimSpace(req.BonusDecay)
	hintPenaltyStr := strings.TrimSpace(req.HintPenalty)
	helpPenaltyStr := strings.TrimSpace(req.HelpPenalty)
	flag := strings.TrimSpace(req.Flag)
	hint := strings.TrimSpace(req.Hint)

	if title == "" || flag == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.title_flag_required"))
		return
	}
	if hintPenaltyStr == "" {
		hintPenaltyStr = "0"
	}
	if helpPenaltyStr == "" {
		helpPenaltyStr = "0"
	}
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.valid_category_id_required"))
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
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_active"))
			return
		}
	}
	points, err := strconv.ParseInt(pointsStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_points"))
		return
	}
	bonus, err := strconv.ParseInt(bonusStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_bonus"))
		return
	}
	bonusDecay, err := strconv.ParseInt(bonusDecayStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_bonus_decay"))
		return
	}
	hintPenalty, err := strconv.ParseInt(hintPenaltyStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_hint_penalty"))
		return
	}
	helpPenalty, err := strconv.ParseInt(helpPenaltyStr, 10, 64)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_help_penalty"))
		return
	}

	challenge, err := h.Challenges.GetByID(uint(challengeID), uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenge to update")
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.challenge_not_found"))
		return
	}
	previousActive := challenge.Active
	previousCountry := strings.ToUpper(strings.TrimSpace(challenge.Country))
	if country != previousCountry && country != "" {
		selectedCountry, err := h.Countries.GetByCode(country)
		if err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_country_code"))
			return
		}
		if !selectedCountry.Active || selectedCountry.Assigned {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.country_active_available"))
			return
		}
	}

	challenge.Title = title
	challenge.Description = description
	challenge.URL = challengeURL
	challenge.CategoryID = uint(categoryID)
	challenge.Country = country
	challenge.Active = active
	challenge.Points = int(points)
	challenge.Bonus = int(bonus)
	challenge.BonusDecay = int(bonusDecay)
	challenge.HintPenalty = int(hintPenalty)
	challenge.HelpPenalty = int(helpPenalty)
	challenge.Flag = flag
	challenge.Hint = hint

	if err := h.Challenges.Update(challenge); err != nil {
		log.Err(err).Msg("error updating challenge")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_challenge"))
		return
	}
	if previousCountry != country {
		if previousCountry != "" {
			exists, err := h.Countries.Exists(previousCountry)
			if err != nil {
				log.Err(err).Msg("error checking previous challenge country")
				writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_country_assignment"))
				return
			}
			if exists {
				if err := h.Countries.ReleaseCountry(previousCountry); err != nil {
					log.Err(err).Msg("error releasing previous challenge country")
					writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_country_assignment"))
					return
				}
			}
		}
		if country != "" {
			if err := h.Countries.AssignCountryToChallenge(country, challenge.ID); err != nil {
				log.Err(err).Msg("error assigning updated challenge country")
				writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_country_assignment"))
				return
			}
		}
	}

	if previousActive != active {
		countryLabel := strings.TrimSpace(challenge.Country)
		if countryLabel == "" {
			countryLabel = strings.TrimSpace(challenge.Title)
		}

		categoryName := ""
		if challenge.CategoryID != 0 {
			category, err := h.Challenges.GetCategoryByID(challenge.CategoryID, uuid)
			if err != nil {
				log.Err(err).Msg("error retrieving challenge category for admin activity")
				writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_retrieve_category"))
				return
			}
			categoryName = category.Name
		}

		action := "disabled"
		actionMsg := logs.DisableMessage(countryLabel, categoryName, challenge.Points)
		if active {
			action = "enabled"
			actionMsg = logs.EnableMessage(countryLabel, categoryName, challenge.Points)
		}
		h.createAdminActivityLog(r, action, actionMsg, challenge.ID)
	}

	writeSuccess(h.T(r.Context())("admin.msg.challenge_updated"))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_challenge_id"))
		return
	}

	challenge, err := h.Challenges.GetByID(uint(challengeID), uuid)
	if err != nil {
		log.Err(err).Msg("error loading challenge to delete")
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.challenge_not_found"))
		return
	}
	country := strings.ToUpper(strings.TrimSpace(challenge.Country))
	if country != "" {
		exists, err := h.Countries.Exists(country)
		if err != nil {
			log.Err(err).Msg("error checking challenge country before delete")
			writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_release_country"))
			return
		}
		if exists {
			if err := h.Countries.ReleaseCountry(country); err != nil {
				log.Err(err).Msg("error releasing challenge country before delete")
				writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_release_country"))
				return
			}
		}
	}

	if err := h.Challenges.Delete(uint(challengeID), uuid); err != nil {
		log.Err(err).Msg("error deleting challenge")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_challenge"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.challenge_deleted"))
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	var req AdminCategoryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin category JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	logo := strings.TrimSpace(req.Logo)

	if name == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.category_name_required"))
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
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_create_category"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.category_created"))
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	categoryIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil || categoryID == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_category_id"))
		return
	}

	var req AdminCategoryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin category update JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	logo := strings.TrimSpace(req.Logo)

	if name == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.category_name_required"))
		return
	}

	category, err := h.Challenges.GetCategoryByID(uint(categoryID), uuid)
	if err != nil {
		log.Err(err).Msg("error loading category to update")
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.category_not_found"))
		return
	}

	category.Name = name
	category.Description = description
	category.Logo = logo

	if err := h.Challenges.UpdateCategory(category); err != nil {
		log.Err(err).Msg("error updating category")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_category"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.category_updated"))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_category_id"))
		return
	}

	if _, err := h.Challenges.GetCategoryByID(uint(categoryID), uuid); err != nil {
		log.Err(err).Msg("error loading category to delete")
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.category_not_found"))
		return
	}

	hasChallenges, err := h.Challenges.CategoryHasChallenges(uint(categoryID), uuid)
	if err != nil {
		log.Err(err).Msg("error checking category usage")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_verify_category_usage"))
		return
	}
	if hasChallenges {
		writeError(http.StatusConflict, h.T(r.Context())("admin.msg.category_assigned_not_deleted"))
		return
	}

	if err := h.Challenges.DeleteCategory(uint(categoryID), uuid); err != nil {
		log.Err(err).Msg("error deleting category")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_category"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.category_deleted"))
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
	t, err := template.New("activity.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/activity.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	currentUsername := strings.TrimSpace(h.Sessions.GetString(r.Context(), string(ContextKeyUser)))
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminActivityTemplateData{
		Title:           tr("admin.title.activity"),
		Lang:            h.Locale(r.Context()).String(),
		I18NJSON:        htmltemplate.JS(i18nJSON),
		UUID:            uuid,
		Authenticated:   authenticated,
		Admin:           h.IsAdmin(r.Context()),
		CurrentUsername: template.HTMLEscapeString(currentUsername),
		Status:          r.URL.Query().Get("status"),
		Message:         r.URL.Query().Get("msg"),
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

// AdminActivityPOSTHandler creates a custom admin activity log entry
func (h *HandlersMap) AdminActivityPOSTHandler(w http.ResponseWriter, r *http.Request) {
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
		writeError(http.StatusUnsupportedMediaType, h.T(r.Context())("feed.content_type"))
		return
	}

	if h.Logs == nil || h.Sessions == nil {
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.activity_logging_unavailable"))
		return
	}

	var req AdminActivityCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Err(err).Msg("error parsing admin activity JSON payload")
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
		return
	}

	subject := strings.TrimSpace(req.Subject)
	action := strings.TrimSpace(req.Action)
	message := strings.TrimSpace(req.Message)
	if subject == "" && message == "" {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.subject_message_required"))
		return
	}

	activity, err := h.Logs.NewActivity(req.Visible, subject, action, message, 0, uuid)
	if err != nil {
		log.Err(err).Msg("error building custom admin activity log")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_build_activity"))
		return
	}
	if err := h.Logs.CreateActivity(activity); err != nil {
		log.Err(err).Msg("error creating custom admin activity log")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_create_activity"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.activity_entry_created"))
}

// AdminActivityDeletePOSTHandler deletes a custom activity entry.
func (h *HandlersMap) AdminActivityDeletePOSTHandler(w http.ResponseWriter, r *http.Request) {
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

	if h.Logs == nil {
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.activity_logging_unavailable"))
		return
	}

	idValue := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil || id == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_activity_id"))
		return
	}

	if err := h.Logs.DeleteActivity(uint(id), uuid); err != nil {
		log.Err(err).Msg("error deleting activity entry")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_activity"))
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
		Success: true,
		Status:  "ok",
		Message: h.T(r.Context())("admin.msg.activity_deleted"),
	})
}

// AdminChatTemplateHandler for admin chat page for GET requests
func (h *HandlersMap) AdminChatTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}
	t, err := template.New("chat.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/chat.html")
	if err != nil {
		log.Err(err).Msg("error getting admin chat template")
		return
	}
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminChatTemplateData{
		Title:         tr("admin.title.chat"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
		UUID:          uuid,
		Authenticated: h.IsAuthenticated(r.Context()),
		Admin:         h.IsAdmin(r.Context()),
		Status:        r.URL.Query().Get("status"),
		Message:       r.URL.Query().Get("msg"),
		ChatTeamNames: map[uint]string{},
	}
	if h.Chat != nil {
		chatEntries, err := h.Chat.GetAll()
		if err != nil {
			log.Warn().Err(err).Msg("error loading chat for admin chat")
		} else {
			start := 0
			if len(chatEntries) > 100 {
				start = len(chatEntries) - 100
			}
			recentChat := append([]chat.ChatEntry(nil), chatEntries[start:]...)
			for left, right := 0, len(recentChat)-1; left < right; left, right = left+1, right-1 {
				recentChat[left], recentChat[right] = recentChat[right], recentChat[left]
			}
			templateData.RecentChat = recentChat
		}
	}
	if h.Teams != nil {
		allTeams, err := h.Teams.GetAll(uuid)
		if err != nil {
			log.Warn().Err(err).Msg("error loading teams for admin chat")
		} else {
			for _, team := range allTeams {
				templateData.ChatTeamNames[team.ID] = team.Name
			}
		}
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// AdminChatSetHiddenPOSTHandler toggles chat visibility for an entry
func (h *HandlersMap) AdminChatSetHiddenPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	redirectBase := "/" + uuid + "/admin/chat"
	writeError := func(code int, msg string) {
		if wantsJSONResponse(r) {
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
		if wantsJSONResponse(r) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
				Success: true,
				Status:  "ok",
				Message: msg,
			})
			return
		}
		http.Redirect(w, r, redirectBase+"?status=ok&msg="+url.QueryEscape(msg), http.StatusFound)
	}

	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_chat_id"))
		return
	}

	entry, err := h.Chat.GetByID(uint(id))
	if err != nil {
		log.Err(err).Msg("error loading chat entry")
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.chat_message_not_found"))
		return
	}

	hiddenValue := false
	if strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		var req adminChatVisibilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Err(err).Msg("error parsing admin chat visibility JSON payload")
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
			return
		}
		parsedHiddenValue, err := parseAdminChatHiddenValue(req.Hidden)
		if err != nil {
			log.Err(err).Msg("error parsing admin chat hidden value")
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_hidden"))
			return
		}
		hiddenValue = parsedHiddenValue
	} else {
		hiddenValue = strings.EqualFold(strings.TrimSpace(r.FormValue("hidden")), "true")
	}

	if err := h.Chat.SetHiddenByID(uint(id), hiddenValue); err != nil {
		log.Err(err).Msg("error updating chat visibility")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_chat_visibility"))
		return
	}

	if hiddenValue {
		writeSuccess(h.T(r.Context())("admin.msg.chat_message_hidden"))
		return
	}
	if entry.Hidden {
		writeSuccess(h.T(r.Context())("admin.msg.chat_message_unhidden"))
		return
	}
	writeSuccess(h.T(r.Context())("admin.msg.chat_visibility_updated"))
}

// AdminChatDeletePOSTHandler deletes a chat entry
func (h *HandlersMap) AdminChatDeletePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" || uuid != h.Config.Map.UUID {
		log.Err(errors.New("Invalid UUID")).Msgf("UUID: %s", uuid)
		h.ErrorInvalidUUID(w, r)
		return
	}

	redirectBase := "/" + uuid + "/admin/chat"
	writeError := func(code int, msg string) {
		if wantsJSONResponse(r) {
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
		if wantsJSONResponse(r) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminActionResponse{
				Success: true,
				Status:  "ok",
				Message: msg,
			})
			return
		}
		http.Redirect(w, r, redirectBase+"?status=ok&msg="+url.QueryEscape(msg), http.StatusFound)
	}

	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_chat_id"))
		return
	}

	if _, err := h.Chat.GetByID(uint(id)); err != nil {
		log.Err(err).Msg("error loading chat entry for delete")
		writeError(http.StatusNotFound, h.T(r.Context())("admin.msg.chat_message_not_found"))
		return
	}

	if err := h.Chat.Delete(uint(id)); err != nil {
		log.Err(err).Msg("error deleting chat entry")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_chat"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.chat_message_deleted"))
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
	t, err := template.New("countries.html").Funcs(h.adminTemplateFuncs(r)).ParseFiles(h.Config.Map.TemplatesDir + "/admin/countries.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	tr := h.T(r.Context())
	i18nJSON, _ := json.Marshal(h.LocaleMessages(r.Context()))
	templateData := AdminCountriesTemplateData{
		Title:         tr("admin.title.countries"),
		Lang:          h.Locale(r.Context()).String(),
		I18NJSON:      htmltemplate.JS(i18nJSON),
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

// AdminCountriesDeleteAllPOSTHandler deletes all countries and clears challenge country references
func (h *HandlersMap) AdminCountriesDeleteAllPOSTHandler(w http.ResponseWriter, r *http.Request) {
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

	if h.Countries == nil || h.Challenges == nil {
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.countries_or_challenges_not_initialized"))
		return
	}
	if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.ajax_only"))
		return
	}

	tx := h.Countries.DB.Begin()
	if tx.Error != nil {
		log.Err(tx.Error).Msg("error starting delete-all-countries transaction")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_countries"))
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			tx.Rollback()
			panic(recovered)
		}
	}()

	if err := tx.Model(&challenges.Challenge{}).Where("uuid = ?", uuid).Update("country", "").Error; err != nil {
		tx.Rollback()
		log.Err(err).Msg("error clearing challenge country references before bulk country delete")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_clear_country_refs"))
		return
	}

	deleteResult := tx.Where("uuid = ?", uuid).Delete(&countries.MapCountry{})
	if deleteResult.Error != nil {
		tx.Rollback()
		log.Err(deleteResult.Error).Msg("error deleting all countries")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_countries"))
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Err(err).Msg("error committing delete-all-countries transaction")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_delete_all_countries"))
		return
	}

	writeSuccess(h.T(r.Context())("admin.msg.deleted_country_count", deleteResult.RowsAffected))
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
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.missing_country_id"))
		return
	}
	countryID, err := strconv.ParseUint(countryIDParam, 10, 32)
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_country_id"))
		return
	}
	if h.Countries == nil {
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.countries_not_initialized"))
		return
	}
	activeValue := "false"
	if strings.Contains(r.Header.Get(ContentType), JSONApplication) {
		var req struct {
			Active bool `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_json_payload"))
			return
		}
		if req.Active {
			activeValue = "true"
		}
	} else {
		if err := r.ParseForm(); err != nil {
			writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_form_payload"))
			return
		}
		activeValue = strings.TrimSpace(r.FormValue("active"))
		if activeValue == "" {
			activeValue = "false"
		}
	}
	active, err := strconv.ParseBool(strings.ToLower(activeValue))
	if err != nil {
		writeError(http.StatusBadRequest, h.T(r.Context())("admin.msg.invalid_active"))
		return
	}
	if err := h.Countries.SetActiveByID(uint(countryID), active); err != nil {
		log.Err(err).Msg("error updating country active status")
		writeError(http.StatusInternalServerError, h.T(r.Context())("admin.msg.failed_update_country_status"))
		return
	}
	writeSuccess(h.T(r.Context())("admin.msg.country_status_updated"))
}

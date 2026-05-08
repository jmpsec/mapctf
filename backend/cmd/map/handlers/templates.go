package handlers

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// IndexTemplateHandler for root requests
func (h *HandlersMap) IndexTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/index.html")
	if err != nil {
		log.Err(err).Msg("error getting index template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	templateData := IndexTemplateData{
		Title:         "MapCTF: Welcome to the platform",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         isAdmin,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

func (h *HandlersMap) currentGameboardTeamName(username, uuid string) string {
	username = strings.TrimSpace(username)
	if username == "" || h.Users == nil || h.Teams == nil {
		return ""
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().Err(err).Str("username", username).Msg("error loading current gameboard user")
		}
		return ""
	}
	if user.TeamID == 0 {
		return ""
	}

	var currentTeam teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ? AND active = ? AND visible = ?", user.TeamID, uuid, true, true).First(&currentTeam).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().Err(err).Uint("team_id", user.TeamID).Msg("error loading current gameboard team")
		}
		return ""
	}

	return currentTeam.Name
}

// LoginHandler for login page for GET requests
func (h *HandlersMap) LoginHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/login.html")
	if err != nil {
		log.Err(err).Msg("error getting login template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	loginEnabled, err := h.Settings.GetLoginEnabled(uuid)
	if err != nil {
		log.Err(err).Msg("error getting login enabled setting")
		loginEnabled = false
	}
	loginStrongPasswords, err := h.Settings.GetLoginStrongPasswords(uuid)
	if err != nil {
		log.Err(err).Msg("error getting login strong passwords setting")
		loginStrongPasswords = false
	}
	loginMsg := "Login to play Capture The Flag here."
	loginType := "Team Login"
	if !loginEnabled {
		loginMsg = "Team login is currently disabled. Only admins can login at this time."
		loginType = "Admin Login"
	}
	templateData := LoginTemplateData{
		Title:                "MapCTF: Login to platform",
		LoginType:            loginType,
		LoginMsg:             loginMsg,
		LoginURL:             "/" + uuid + "/login",
		UUID:                 uuid,
		LoginEnabled:         loginEnabled,
		LoginStrongPasswords: loginStrongPasswords,
		Authenticated:        authenticated,
		Admin:                isAdmin,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// RegistrationTemplateHandler for registration page for GET requests
func (h *HandlersMap) RegistrationTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/registration.html")
	if err != nil {
		log.Err(err).Msg("error getting registration template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	rMsg := "Register to play Capture The Flag here. Once you have registered, simply login for future site visits."
	regEnabled, err := h.Settings.GetRegistrationEnabled(uuid)
	if err != nil {
		log.Err(err).Msg("error getting registration enabled setting")
		regEnabled = false
	}
	if !regEnabled {
		rMsg = "Team Registration will be open soon, stay tuned!"
	}
	regNames, err := h.Settings.GetRegistrationNames(uuid)
	if err != nil {
		log.Err(err).Msg("error getting registration names setting")
		regNames = false
	}
	regEmails, err := h.Settings.GetRegistrationEmails(uuid)
	if err != nil {
		log.Err(err).Msg("error getting registration emails setting")
		regEmails = false
	}
	regType, err := h.Settings.GetRegistrationType(uuid)
	if err != nil {
		log.Err(err).Msg("error getting registration type setting")
		regType = settings.OpenRegistration
	}
	rTypeStr := "Open Registration"
	if regType == settings.TokenRegistration {
		rTypeStr = "Registration with Token"
	}
	templateData := RegistrationTemplateData{
		Title:               "MapCTF: Register to platform",
		RegistrationMsg:     rMsg,
		RegisterURL:         "/" + uuid + "/registration",
		UUID:                uuid,
		RegistrationEnabled: regEnabled,
		RegistrationNames:   regNames,
		RegistrationEmails:  regEmails,
		RegistrationType:    regType,
		RegistrationTypeStr: rTypeStr,
		Authenticated:       authenticated,
		Admin:               isAdmin,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// CountdownTemplateHandler for countdown page for GET requests
func (h *HandlersMap) CountdownTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/countdown.html")
	if err != nil {
		log.Err(err).Msg("error getting countdown template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	startTime, err := h.Settings.GetGameStartTime(uuid)
	if err != nil {
		log.Err(err).Msg("error getting game start time")
		startTime = time.Time{}
	}
	// Calculate countdown units if start time is set and in the future
	var units CountdownUnits
	if !startTime.IsZero() && startTime.After(time.Now()) {
		duration := time.Until(startTime)
		units = CountdownUnits{
			Days:    fmt.Sprintf("%02d", int(duration.Hours())/24),
			Hours:   fmt.Sprintf("%02d", int(duration.Hours())%24),
			Minutes: fmt.Sprintf("%02d", int(duration.Minutes())%60),
			Seconds: fmt.Sprintf("%02d", int(duration.Seconds())%60),
		}
	}
	// Get if game has already started to show message on countdown page
	alreadyStarted, err := h.Settings.GetGameStarted(uuid)
	if err != nil {
		log.Err(err).Msg("error getting game started setting")
		alreadyStarted = false
	}
	endTime, err := h.Settings.GetGameEndTime(uuid)
	if err != nil {
		log.Err(err).Msg("error getting game end time")
		endTime = time.Time{}
	}
	if alreadyStarted && !endTime.IsZero() && endTime.After(time.Now()) {
		duration := time.Until(endTime)
		units = CountdownUnits{
			Days:    fmt.Sprintf("%02d", int(duration.Hours())/24),
			Hours:   fmt.Sprintf("%02d", int(duration.Hours())%24),
			Minutes: fmt.Sprintf("%02d", int(duration.Minutes())%60),
			Seconds: fmt.Sprintf("%02d", int(duration.Seconds())%60),
		}
	}
	if !alreadyStarted && !startTime.IsZero() && startTime.Before(time.Now()) {
		startTime = time.Time{}
	}
	templateData := CountdownTemplateData{
		Title:          "MapCTF: Countdown to event",
		UUID:           uuid,
		StartTime:      startTime,
		EndTime:        endTime,
		EndSet:         !endTime.IsZero(),
		AlreadyStarted: alreadyStarted,
		StartSet:       !startTime.IsZero(),
		Units:          units,
		Authenticated:  authenticated,
		Admin:          isAdmin,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// RulesTemplateHandler for rules page for GET requests
func (h *HandlersMap) RulesTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		h.Config.Map.TemplatesDir + "/rules.html")
	if err != nil {
		log.Err(err).Msg("error getting rules template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	templateData := RulesTemplateData{
		Title:         "MapCTF: Rules of the game",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         isAdmin,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// GameboardTemplateHandler for gameboard page for GET requests
func (h *HandlersMap) GameboardTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
	t, err := template.New("gameboard.html").Funcs(template.FuncMap{
		"htmlAttr": html.EscapeString,
	}).ParseFiles(h.Config.Map.TemplatesDir + "/gameboard.html")
	if err != nil {
		log.Err(err).Msg("error getting gameboard template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	currentUsername := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	templateData := GameboardTemplateData{
		Title:               "MapCTF: Gameboard",
		UUID:                uuid,
		Authenticated:       authenticated,
		Admin:               isAdmin,
		CurrentUsername:     currentUsername,
		CurrentTeam:         h.currentGameboardTeamName(currentUsername, uuid),
		GameboardChatMaxLen: chat.DefaultMaxLen,
	}
	chatMaxLen, err := h.Settings.GetGameboardChatMaxLen(uuid)
	if err == nil {
		templateData.GameboardChatMaxLen = chatMaxLen
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading gameboard_chat_max_len")
	}
	showTeamMembers, err := h.Settings.GetGameboardShowTeamMembers(uuid)
	if err == nil {
		templateData.GameboardShowTeamMembers = showTeamMembers
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading gameboard_show_team_members")
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
	gameStarted, err := h.Settings.GetGameStarted(uuid)
	if err == nil {
		templateData.GameStarted = gameStarted
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_started")
	}
	gamePaused, err := h.Settings.GetGamePaused(uuid)
	if err == nil {
		templateData.GamePaused = gamePaused
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_paused")
	}
	gameStartTime, err := h.Settings.GetGameStartTime(uuid)
	if err == nil {
		templateData.GameStartTime = gameStartTime
		templateData.GameStartSet = !gameStartTime.IsZero()
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_start_time")
	}
	gameEndTime, err := h.Settings.GetGameEndTime(uuid)
	if err == nil {
		templateData.GameEndTime = gameEndTime
		templateData.GameEndSet = !gameEndTime.IsZero()
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading game_end_time")
	}
	if h.Countries != nil {
		countriesList, err := h.Countries.GetAll()
		if err != nil {
			log.Warn().Err(err).Msg("error loading countries for gameboard")
		} else if h.Challenges != nil {
			activeChallenges, challengeErr := h.Challenges.GetActive(uuid)
			if challengeErr != nil {
				log.Warn().Err(challengeErr).Msg("error loading active challenges for gameboard countries")
				templateData.Countries = countriesList
			} else {
				activeCountryCodes := make(map[string]struct{}, len(activeChallenges))
				for _, challenge := range activeChallenges {
					countryCode := strings.ToUpper(strings.TrimSpace(challenge.Country))
					if countryCode == "" {
						continue
					}
					activeCountryCodes[countryCode] = struct{}{}
				}

				renderCountries := make([]countries.MapCountry, 0, len(countriesList))
				for _, country := range countriesList {
					renderCountry := country
					if _, ok := activeCountryCodes[strings.ToUpper(strings.TrimSpace(country.CountryCode))]; ok {
						if !strings.Contains(renderCountry.LandClass, "active") {
							renderCountry.LandClass = strings.TrimSpace(renderCountry.LandClass + " active")
						}
					}
					renderCountries = append(renderCountries, renderCountry)
				}
				templateData.Countries = renderCountries
			}
		} else {
			templateData.Countries = countriesList
		}
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

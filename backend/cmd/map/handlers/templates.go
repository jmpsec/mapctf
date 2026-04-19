package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/settings"
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
	loginEnabled, err := h.Settings.GetLoginEnabled()
	if err != nil {
		log.Err(err).Msg("error getting login enabled setting")
		loginEnabled = false
	}
	loginStrongPasswords, err := h.Settings.GetLoginStrongPasswords()
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
	regEnabled, err := h.Settings.GetRegistrationEnabled()
	if err != nil {
		log.Err(err).Msg("error getting registration enabled setting")
		regEnabled = false
	}
	if !regEnabled {
		rMsg = "Team Registration will be open soon, stay tuned!"
	}
	regNames, err := h.Settings.GetRegistrationNames()
	if err != nil {
		log.Err(err).Msg("error getting registration names setting")
		regNames = false
	}
	regEmails, err := h.Settings.GetRegistrationEmails()
	if err != nil {
		log.Err(err).Msg("error getting registration emails setting")
		regEmails = false
	}
	regType, err := h.Settings.GetRegistrationType()
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
	startTime, err := h.Settings.GetGameStartTime()
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
	alreadyStarted, err := h.Settings.GetGameStarted()
	if err != nil {
		log.Err(err).Msg("error getting game started setting")
		alreadyStarted = false
	}
	endTime, err := h.Settings.GetGameEndTime()
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
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/gameboard.html")
	if err != nil {
		log.Err(err).Msg("error getting gameboard template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	isAdmin := h.IsAdmin(r.Context())
	templateData := GameboardTemplateData{
		Title:               "MapCTF: Gameboard",
		UUID:                uuid,
		Authenticated:       authenticated,
		Admin:               isAdmin,
		CurrentUsername:     h.Sessions.GetString(r.Context(), string(ContextKeyUser)),
		GameboardChatMaxLen: chat.DefaultMaxLen,
	}
	chatMaxLen, err := h.Settings.GetGameboardChatMaxLen()
	if err == nil {
		templateData.GameboardChatMaxLen = chatMaxLen
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading gameboard_chat_max_len")
	}
	showTeamMembers, err := h.Settings.GetGameboardShowTeamMembers()
	if err == nil {
		templateData.GameboardShowTeamMembers = showTeamMembers
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn().Err(err).Msg("error loading gameboard_show_team_members")
	}
	if h.Countries != nil {
		countriesList, err := h.Countries.GetAll()
		if err != nil {
			log.Warn().Err(err).Msg("error loading countries for gameboard")
		} else {
			templateData.Countries = countriesList
		}
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

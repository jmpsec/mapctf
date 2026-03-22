package handlers

import (
	"errors"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/rs/zerolog/log"
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
		Title:         "Welcome to mapctf",
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
	templateData := LoginTemplateData{
		Title:         "Login to mapctf",
		LoginType:     "Admin Login",
		LoginMsg:      "Team login is disabled. Only admins can login at this time.",
		LoginURL:      "/" + uuid + "/login",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         isAdmin,
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
		Title:               "Register to mapctf",
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
	templateData := CountdownTemplateData{
		Title:         "Countdown to mapctf",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         isAdmin,
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
		Title:         "Rules of mapctf",
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
		Title:         "Gameboard of mapctf",
		UUID:          uuid,
		Authenticated: authenticated,
		Admin:         isAdmin,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

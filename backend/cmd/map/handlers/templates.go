package handlers

import (
	"net/http"
	"text/template"

	"github.com/rs/zerolog/log"
)

// LoginHandler for login page for GET requests
func (h *HandlersMap) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Prepare template
	t, err := template.ParseFiles(
		h.Config.Map.TemplatesDir + "/login.html")
	if err != nil {
		log.Err(err).Msg("error getting login template")
		return
	}
	// Prepare template data
	templateData := LoginTemplateData{
		Title:   "Login to osctrl",
		Project: "osctrl",
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

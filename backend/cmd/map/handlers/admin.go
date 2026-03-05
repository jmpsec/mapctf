package handlers

import (
	"errors"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// AdminTemplateHandler for admin page for GET requests
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
		h.Config.Map.TemplatesDir + "/admin.html")
	if err != nil {
		log.Err(err).Msg("error getting admin template")
		return
	}
	// Prepare template data
	authenticated := h.IsAuthenticated(r.Context())
	templateData := RulesTemplateData{
		Title:         "Rules of mapctf",
		UUID:          uuid,
		Authenticated: authenticated,
	}
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

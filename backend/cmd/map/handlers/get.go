package handlers

import (
	"net/http"
	"net/url"
	"text/template"

	"github.com/rs/zerolog/log"
)

// ErrorHandler for error requests
func (h *HandlersMap) ErrorHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	t, err := template.ParseFiles(h.Config.Map.TemplatesDir + "/error.html")
	if err != nil {
		log.Err(err).Msg("error getting error template")
		HTTPResponse(w, "", http.StatusInternalServerError, []byte(errorContent))
		return
	}
	msg := "An unexpected issue occurred while processing your request."
	if raw := r.URL.Query().Get("message"); raw != "" {
		if unescaped, decodeErr := url.QueryUnescape(raw); decodeErr == nil && unescaped != "" {
			msg = unescaped
		} else {
			msg = raw
		}
	}
	status := "error"
	if raw := r.URL.Query().Get("status"); raw != "" {
		if unescaped, decodeErr := url.QueryUnescape(raw); decodeErr == nil && unescaped != "" {
			status = unescaped
		} else {
			status = raw
		}
	}
	header := "Issue detected"
	if raw := r.URL.Query().Get("header"); raw != "" {
		if unescaped, decodeErr := url.QueryUnescape(raw); decodeErr == nil && unescaped != "" {
			header = unescaped
		} else {
			header = raw
		}
	}
	templateData := ErrorTemplateData{
		Title: "Error in mapctf",
		Error: msg,
		Status: status,
		Header: header,
	}
	w.WriteHeader(http.StatusInternalServerError)
	if err := t.Execute(w, templateData); err != nil {
		log.Err(err).Msg("template error")
		return
	}
}

// ErrorInvalidUUID for invalid UUID error
func (h *HandlersMap) ErrorInvalidUUID(w http.ResponseWriter, r *http.Request) {
	rErr := r.Clone(r.Context())
	q := rErr.URL.Query()
	q.Set("message", "Valid UUID is required")
	rErr.URL.RawQuery = q.Encode()
	h.ErrorHandler(w, rErr)
}

// ErrorForbidden for forbidden error
func (h *HandlersMap) ErrorForbidden(w http.ResponseWriter, r *http.Request) {
	rErr := r.Clone(r.Context())
	q := rErr.URL.Query()
	q.Set("message", "You do not have permission to access this resource")
	q.Set("status", "forbidden")
	q.Set("header", "Access forbidden")
	rErr.URL.RawQuery = q.Encode()
	h.ErrorHandler(w, rErr)
}

// ForbiddenHandler for forbidden error requests
func (h *HandlersMap) ForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP for environment
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, forbiddenContent)
}

// RootHandler for root requests
func (h *HandlersMap) RootHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusForbidden, rootContent)
}

// HealthHandler for health requests
func (h *HandlersMap) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Send response
	HTTPResponse(w, "", http.StatusOK, []byte(okContent))
}

// FaviconHandler for the favicon
func (h *HandlersMap) FaviconHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, "/static/img/favicon.png")
}

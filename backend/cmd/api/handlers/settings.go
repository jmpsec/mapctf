package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/users"
)

// SettingsHandler - Handle settings requests
// @Summary      Settings
// @Description  Get all settings for a specific UUID
// @Tags         settings
// @Produce      json
// @Param        uuid  path      string  true  "UUID"
// @Success      200    {array}   Setting  "Settings"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid UUID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/settings/{uuid} [get]
func (h *HandlersAPI) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Extract UUID from path
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		uuid = users.NoUUID
	}
	// Get settings from database filtered by UUID
	settings, err := h.Settings.GetAll(uuid)
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error getting settings"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, settings)
}

package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/users"
)

// TeamsHandler - Handle teams requests
// @Summary      Teams
// @Description  Get all teams for a specific UUID
// @Tags         teams
// @Produce      json
// @Param        uuid  path      string  true  "UUID"
// @Success      200    {array}   Team  "Teams"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid UUID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/teams/{uuid} [get]
func (h *HandlersAPI) TeamsHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Extract UUID from path
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		uuid = users.NoUUID
	}
	// Get teams from database filtered by UUID
	teams, err := h.Teams.GetAll(uuid)
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error getting teams"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, teams)
}

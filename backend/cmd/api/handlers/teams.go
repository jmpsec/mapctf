package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// TeamsHandler - Handle teams requests
// @Summary      Teams
// @Description  Get all teams for a specific entity ID
// @Tags         teams
// @Produce      json
// @Param        entID  path      int  true  "Entity ID"
// @Success      200    {array}   Team  "Teams"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid entity ID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/teams/{entID} [get]
func (h *HandlersAPI) TeamsHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Extract entID from path
	entIDStr := chi.URLParam(r, "entID")
	entID, err := strconv.ParseUint(entIDStr, 10, 32)
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "invalid entity ID"})
		return
	}
	// Get teams from database filtered by entity ID
	teams, err := h.Teams.GetAll(uint(entID))
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error getting teams"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, teams)
}

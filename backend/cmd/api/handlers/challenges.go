package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ChallengesHandler - Handle challenges requests
// @Summary      Challenges
// @Description  Get all challenges for a specific entity ID
// @Tags         challenges
// @Produce      json
// @Param        entID  path      int  true  "Entity ID"
// @Success      200    {array}   Challenge  "Challenges"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid entity ID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/challenges/{entID} [get]
func (h *HandlersAPI) ChallengesHandler(w http.ResponseWriter, r *http.Request) {
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
	// Get challenges from database filtered by entity ID
	challenges, err := h.Challenges.GetAll(uint(entID))
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error getting challenges"})
		return
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, challenges)
}

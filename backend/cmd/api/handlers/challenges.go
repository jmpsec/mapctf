package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ChallengesHandler - Handle challenges requests
// @Summary      Challenges
// @Description  Get all challenges for a specific UUID
// @Tags         challenges
// @Produce      json
// @Param        uuid  path      string  true  "UUID"
// @Success      200    {array}   Challenge  "Challenges"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid UUID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/challenges/{uuid} [get]
func (h *HandlersAPI) ChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Debug HTTP if enabled
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	// Extract UUID from path
	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "invalid UUID"})
		return
	}
	// Get challenges from database filtered by UUID
	dbChallenges, err := h.Challenges.GetAll(uuid)
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error getting challenges"})
		return
	}
	// Convert internal Challenge structs to API Challenge structs
	apiChallenges := make([]Challenge, 0, len(dbChallenges))
	for _, dbChallenge := range dbChallenges {
		// Get category name (placeholder for now - will need to fetch from Category table)
		categoryName := "Uncategorized"
		if dbChallenge.CategoryID > 0 {
			category, err := h.Challenges.GetCategoryByID(dbChallenge.CategoryID, uuid)
			if err == nil {
				categoryName = category.Name
			}
		}

		apiChallenge := Challenge{
			ID:          strconv.FormatUint(uint64(dbChallenge.ID), 10),
			Title:       dbChallenge.Title,
			Category:    categoryName,
			Points:      dbChallenge.Points,
			Solved:      false, // TODO: Check if current user/team solved this challenge
			Country:     "",    // TODO: Add country field to Challenge model
			CountryCode: "",    // TODO: Add countryCode field to Challenge model
		}
		apiChallenges = append(apiChallenges, apiChallenge)
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, apiChallenges)
}

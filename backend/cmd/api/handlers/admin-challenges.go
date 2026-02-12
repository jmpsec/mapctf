package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// AdminChallengesHandler - Handle admin challenges requests
// @Summary      Admin Challenges
// @Description  Get all challenges for admin panel for a specific UUID
// @Tags         admin
// @Produce      json
// @Param        uuid   path      string  true  "UUID"
// @Success      200    {array}   AdminChallenge  "Admin Challenges"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid UUID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/admin/challenges/{uuid} [get]
func (h *HandlersAPI) AdminChallengesHandler(w http.ResponseWriter, r *http.Request) {
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

	// Convert internal Challenge structs to AdminChallenge structs
	adminChallenges := make([]AdminChallenge, 0, len(dbChallenges))
	for _, dbChallenge := range dbChallenges {
		// Get category name if category ID exists
		categoryName := ""
		if dbChallenge.CategoryID > 0 {
			category, catErr := h.Challenges.GetCategoryByID(dbChallenge.CategoryID, uuid)
			if catErr == nil {
				categoryName = category.Name
			}
		}

		adminChallenges = append(adminChallenges, AdminChallenge{
			ID:          strconv.FormatUint(uint64(dbChallenge.ID), 10),
			Title:       dbChallenge.Title,
			Description: dbChallenge.Description,
			Category:    categoryName,
			Points:      dbChallenge.Points,
			Flag:        dbChallenge.Flag,
			Active:      dbChallenge.Active,
		})
	}

	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminChallenges)
}

// CreateChallengeHandler - Handle create challenge requests
// @Summary      Create Challenge
// @Description  Create a new challenge for a specific UUID
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        uuid      path      string                    true  "UUID"
// @Param        request   body      CreateChallengeRequest  true  "Challenge data"
// @Success      201       {object}  AdminChallenge          "Challenge created"
// @Failure      400       {object}  ApiErrorResponse        "Bad request - invalid input"
// @Failure      500       {object}  ApiErrorResponse        "Internal server error"
// @Router       /api/v1/admin/challenges/{uuid} [post]
func (h *HandlersAPI) CreateChallengeHandler(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body
	var req CreateChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "error parsing request body"})
		return
	}

	// Validate required fields
	if req.Title == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "title is required"})
		return
	}
	if req.Flag == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "flag is required"})
		return
	}

	// Create challenge using the New method
	challenge := h.Challenges.New(
		req.Title,
		req.Description,
		req.CategoryID,
		req.Active,
		req.Points,
		req.Bonus,
		req.BonusDecay,
		req.Penalty,
		req.Flag,
		req.Hint,
		uuid,
	)

	// Save challenge to database
	if err := h.Challenges.Create(challenge); err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error creating challenge"})
		return
	}

	// Get category name if category ID exists
	categoryName := ""
	if challenge.CategoryID > 0 {
		category, catErr := h.Challenges.GetCategoryByID(challenge.CategoryID, uuid)
		if catErr == nil {
			categoryName = category.Name
		}
	}

	// Return created challenge
	adminChallenge := AdminChallenge{
		ID:          strconv.FormatUint(uint64(challenge.ID), 10),
		Title:       challenge.Title,
		Description: challenge.Description,
		Category:    categoryName,
		Points:      challenge.Points,
		Flag:        challenge.Flag,
		Active:      challenge.Active,
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusCreated, adminChallenge)
}

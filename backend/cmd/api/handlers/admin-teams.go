package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/teams"
)

// AdminTeamsHandler - Handle admin teams requests
// @Summary      Admin Teams
// @Description  Get all teams for admin panel with member counts for a specific entity ID
// @Tags         admin
// @Produce      json
// @Param        entID  path      int  true  "Entity ID"
// @Success      200    {array}   AdminTeam  "Admin Teams"
// @Failure      400    {object}  ApiErrorResponse  "Bad request - invalid entity ID"
// @Failure      500    {object}  ApiErrorResponse  "Internal server error"
// @Router       /api/v1/admin/teams/{entID} [get]
func (h *HandlersAPI) AdminTeamsHandler(w http.ResponseWriter, r *http.Request) {
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
	teamsList, err := h.Teams.GetAll(uint(entID))
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error getting teams"})
		return
	}
	// Convert PlatformTeam to AdminTeam with member counts
	adminTeams := make([]AdminTeam, 0, len(teamsList))
	for _, team := range teamsList {
		// Count team members
		var memberCount int64
		h.DB.Model(&teams.TeamMembership{}).Where("team_id = ?", team.ID).Count(&memberCount)

		adminTeams = append(adminTeams, AdminTeam{
			ID:      strconv.FormatUint(uint64(team.ID), 10),
			Name:    team.Name,
			Email:   "", // Email not stored in PlatformTeam
			Score:   team.Points,
			Members: int(memberCount),
			Active:  team.Active,
		})
	}
	// Send response
	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, adminTeams)
}

// CreateTeamHandler - Handle create team requests
// @Summary      Create Team
// @Description  Create a new team for a specific entity ID
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        entID    path      int                true  "Entity ID"
// @Param        request  body      CreateTeamRequest true  "Team data"
// @Success      201      {object}  AdminTeam         "Team created"
// @Failure      400      {object}  ApiErrorResponse   "Bad request - invalid input"
// @Failure      500      {object}  ApiErrorResponse   "Internal server error"
// @Router       /api/v1/admin/teams/{entID} [post]
func (h *HandlersAPI) CreateTeamHandler(w http.ResponseWriter, r *http.Request) {
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
	// Parse request body
	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "error parsing request body"})
		return
	}
	// Validate required fields
	if req.Name == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "name is required"})
		return
	}
	// Check if team already exists
	if h.Teams.Exists(req.Name, uint(entID)) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: "team already exists"})
		return
	}
	// Create team using the New method
	team, err := h.Teams.New(req.Name, req.Logo, "", req.Protected, req.Visible, uint(entID))
	if err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, ApiErrorResponse{Error: err.Error()})
		return
	}
	// Save team to database
	if err := h.Teams.Create(team); err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, ApiErrorResponse{Error: "error creating team"})
		return
	}
	// Return created team (member count is 0 for new team)
	adminTeam := AdminTeam{
		ID:      strconv.FormatUint(uint64(team.ID), 10),
		Name:    team.Name,
		Email:   "", // Email not stored in PlatformTeam
		Score:   team.Points,
		Members: 0, // New team has no members
		Active:  team.Active,
	}
	HTTPResponse(w, JSONApplicationUTF8, http.StatusCreated, adminTeam)
}

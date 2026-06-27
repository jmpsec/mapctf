package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const minProfilePasswordLength = 8

func (h *HandlersMap) ProfileGETHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)
	}
	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
		return
	}
	username, ok := h.profileSessionUsername(w, r)
	if !ok {
		return
	}
	if h.Users == nil {
		log.Error().Msg("users manager not initialized")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.unavailable")})
		return
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("profile.auth_required")})
			return
		}
		log.Err(err).Msg("error retrieving profile user")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.unavailable")})
		return
	}

	resp := MapProfileResponse{
		Success: true,
		Account: profileAccountResponse(user, h.T(r.Context())),
	}

	team, found, err := h.currentProfileTeam(user.TeamID, uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving profile team")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.unavailable")})
		return
	}
	if found {
		resp.Team = &MapProfileTeamResponse{
			ID:        team.ID,
			Name:      team.Name,
			Logo:      team.Logo,
			Points:    team.Points,
			Rank:      h.profileTeamRank(team.ID, uuid),
			Visible:   team.Visible,
			Active:    team.Active,
			LastScore: profileTime(team.LastScore),
		}
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, resp)
}

func (h *HandlersMap) ProfilePOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, false)
	}
	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
		return
	}
	username, ok := h.profileSessionUsername(w, r)
	if !ok {
		return
	}
	if h.Users == nil {
		log.Error().Msg("users manager not initialized")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.unavailable")})
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnsupportedMediaType, MapErrorResponse{Error: h.T(r.Context())("feed.content_type")})
		return
	}

	var req MapProfileAccountUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("feed.invalid_json")})
		return
	}

	fullName := strings.TrimSpace(req.FullName)
	email := strings.TrimSpace(req.Email)
	if email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("profile.email_invalid")})
			return
		}
	}

	result := h.Users.DB.Model(&users.PlatformUser{}).
		Where("username = ? AND uuid = ?", username, uuid).
		Updates(map[string]interface{}{
			"name":  fullName,
			"email": email,
		})
	if result.Error != nil {
		log.Err(result.Error).Msg("error updating profile account")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.not_updated")})
		return
	}
	if result.RowsAffected == 0 {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("profile.auth_required")})
		return
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		log.Err(err).Msg("error retrieving updated profile user")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.not_updated")})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapProfileAccountUpdateResponse{
		Success: true,
		Message: h.T(r.Context())("profile.updated_msg"),
		Account: profileAccountResponse(user, h.T(r.Context())),
	})
}

func (h *HandlersMap) ProfilePasswordPOSTHandler(w http.ResponseWriter, r *http.Request) {
	if h.Config.DebugHTTP.Enabled {
		DebugHTTPDump(h.DebugHTTP, r, false)
	}
	uuid, ok := h.validatedJSONUUID(w, r)
	if !ok {
		return
	}
	username, ok := h.profileSessionUsername(w, r)
	if !ok {
		return
	}
	if h.Users == nil {
		log.Error().Msg("users manager not initialized")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.unavailable")})
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get(ContentType)), JSONApplication) {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnsupportedMediaType, MapErrorResponse{Error: h.T(r.Context())("feed.content_type")})
		return
	}

	var req MapProfilePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("feed.invalid_json")})
		return
	}

	switch {
	case req.CurrentPassword == "":
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("profile.current_required")})
		return
	case strings.TrimSpace(req.NewPassword) == "":
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("profile.new_required")})
		return
	case utf8.RuneCountInString(req.NewPassword) < minProfilePasswordLength:
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("profile.new_min_length")})
		return
	case req.NewPassword != req.ConfirmPassword:
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("profile.new_mismatch")})
		return
	case req.CurrentPassword == req.NewPassword:
		HTTPResponse(w, JSONApplicationUTF8, http.StatusBadRequest, MapErrorResponse{Error: h.T(r.Context())("profile.new_must_differ")})
		return
	}

	valid, _ := h.Users.CheckLoginCredentials(username, req.CurrentPassword, uuid)
	if !valid {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("profile.current_incorrect")})
		return
	}

	if err := h.Users.SetPassword(username, req.NewPassword, uuid); err != nil {
		log.Err(err).Msg("error updating profile password")
		HTTPResponse(w, JSONApplicationUTF8, http.StatusInternalServerError, MapErrorResponse{Error: h.T(r.Context())("profile.password_not_updated")})
		return
	}

	HTTPResponse(w, JSONApplicationUTF8, http.StatusOK, MapProfilePasswordResponse{
		Success: true,
		Message: h.T(r.Context())("profile.password_updated_msg"),
	})
}

func (h *HandlersMap) profileSessionUsername(w http.ResponseWriter, r *http.Request) (username string, ok bool) {
	if h.Sessions == nil {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("profile.auth_required")})
		return "", false
	}
	// SCS panics if the LoadAndSave middleware has not prepared the context.
	defer func() {
		if recover() != nil {
			HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("profile.auth_required")})
			username = ""
			ok = false
		}
	}()
	username = h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		HTTPResponse(w, JSONApplicationUTF8, http.StatusUnauthorized, MapErrorResponse{Error: h.T(r.Context())("profile.auth_required")})
		return "", false
	}
	return username, true
}

func (h *HandlersMap) currentProfileTeam(teamID uint, uuid string) (teams.PlatformTeam, bool, error) {
	if teamID == 0 || h.Teams == nil {
		return teams.PlatformTeam{}, false, nil
	}
	var team teams.PlatformTeam
	if err := h.Teams.DB.Where("id = ? AND uuid = ?", teamID, uuid).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return teams.PlatformTeam{}, false, nil
		}
		return teams.PlatformTeam{}, false, err
	}
	return team, true, nil
}

func (h *HandlersMap) profileTeamRank(teamID uint, uuid string) int {
	if teamID == 0 || h.Teams == nil {
		return 0
	}
	var rankedTeams []teams.PlatformTeam
	if err := h.Teams.DB.
		Where("uuid = ? AND active = ? AND visible = ?", uuid, true, true).
		Order("points DESC").
		Order("name ASC").
		Find(&rankedTeams).Error; err != nil {
		log.Err(err).Msg("error calculating profile team rank")
		return 0
	}
	for i, team := range rankedTeams {
		if team.ID == teamID {
			return i + 1
		}
	}
	return 0
}

func profileRole(admin, service bool, tr func(string, ...any) string) string {
	if admin {
		return tr("profile.role_admin")
	}
	if service {
		return tr("profile.role_service")
	}
	return tr("profile.role_player")
}

func profileAccountResponse(user users.PlatformUser, tr func(string, ...any) string) MapProfileAccountResponse {
	return MapProfileAccountResponse{
		Username: user.Username,
		Name:     user.Name,
		Email:    user.Email,
		Role:     profileRole(user.Admin, user.Service, tr),
		Status:   profileStatus(user.Active, tr),
	}
}

func profileStatus(active bool, tr func(string, ...any) string) string {
	if active {
		return tr("profile.status_active")
	}
	return tr("profile.status_inactive")
}

func profileTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

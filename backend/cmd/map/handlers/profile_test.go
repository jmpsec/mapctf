package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/i18n"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
)

func newProfileHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *users.UserManager, *teams.TeamManager) {
	t.Helper()

	db := newJSONTestDB(t)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, userManager, teamManager
}

func TestProfileGETHandlerReturnsCurrentUserAndTeam(t *testing.T) {
	handler, sessions, userManager, teamManager := newProfileHandler(t)

	alpha := teams.PlatformTeam{
		Name:      "Alpha",
		Logo:      "/static/img/team-logos/badge-alpha.png",
		Points:    100,
		LastScore: time.Date(2026, 5, 3, 10, 30, 0, 0, time.UTC),
		Visible:   true,
		Active:    true,
		UUID:      jsonTestUUID,
	}
	bravo := teams.PlatformTeam{
		Name:    "Bravo",
		Logo:    "rocket",
		Points:  200,
		Visible: true,
		Active:  true,
		UUID:    jsonTestUUID,
	}
	require.NoError(t, teamManager.DB.Create(&alpha).Error)
	require.NoError(t, teamManager.DB.Create(&bravo).Error)

	user, err := userManager.New("alice", "password123", "alice@example.com", "Alice Doe", false, false, jsonTestUUID, alpha.ID)
	require.NoError(t, err)
	user.LastIPAddress = "203.0.113.10"
	user.LastUserAgent = "Firefox Test"
	user.LastAccess = time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	require.NoError(t, userManager.Create(user))

	req := newRequestWithUUID(http.MethodGet, "/profile", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ProfileGETHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MapProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "alice", resp.Account.Username)
	require.Equal(t, "Alice Doe", resp.Account.Name)
	require.Equal(t, "alice@example.com", resp.Account.Email)
	require.Equal(t, "Player", resp.Account.Role)
	require.Equal(t, "Active", resp.Account.Status)
	require.NotNil(t, resp.Team)
	require.Equal(t, alpha.ID, resp.Team.ID)
	require.Equal(t, "Alpha", resp.Team.Name)
	require.Equal(t, "/static/img/team-logos/badge-alpha.png", resp.Team.Logo)
	require.Equal(t, 100, resp.Team.Points)
	require.Equal(t, 2, resp.Team.Rank)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	account, ok := raw["account"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, account, "last_access")
	require.NotContains(t, account, "last_ip_address")
	require.NotContains(t, account, "last_user_agent")
}

func TestProfileGETHandlerRequiresSessionUser(t *testing.T) {
	handler, _, _, _ := newProfileHandler(t)

	req := newRequestWithUUID(http.MethodGet, "/profile", jsonTestUUID)
	rec := httptest.NewRecorder()

	handler.ProfileGETHandler(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProfilePOSTHandlerUpdatesCurrentUserNameAndEmail(t *testing.T) {
	handler, sessions, userManager, _ := newProfileHandler(t)

	user, err := userManager.New("alice", "password123", "alice@example.com", "Alice", false, false, jsonTestUUID, 0)
	require.NoError(t, err)
	user.TeamID = 7
	require.NoError(t, userManager.Create(user))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/profile", jsonTestUUID, MapProfileAccountUpdateRequest{
		FullName: "  Alice Updated  ",
		Email:    "  alice.updated@example.com  ",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ProfilePOSTHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MapProfileAccountUpdateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "Profile updated", resp.Message)
	require.Equal(t, "Alice Updated", resp.Account.Name)
	require.Equal(t, "alice.updated@example.com", resp.Account.Email)

	updated, err := userManager.Get("alice", jsonTestUUID)
	require.NoError(t, err)
	require.Equal(t, "Alice Updated", updated.Name)
	require.Equal(t, "alice.updated@example.com", updated.Email)
	require.Equal(t, uint(7), updated.TeamID)
	require.False(t, updated.Admin)
	require.True(t, updated.Active)
}

func TestProfilePOSTHandlerRejectsInvalidEmail(t *testing.T) {
	handler, sessions, userManager, _ := newProfileHandler(t)

	user, err := userManager.New("alice", "password123", "alice@example.com", "Alice", false, false, jsonTestUUID, 0)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(user))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/profile", jsonTestUUID, MapProfileAccountUpdateRequest{
		FullName: "Alice Updated",
		Email:    "not an email",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ProfilePOSTHandler(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp MapErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "email is invalid", resp.Error)

	unchanged, err := userManager.Get("alice", jsonTestUUID)
	require.NoError(t, err)
	require.Equal(t, "Alice", unchanged.Name)
	require.Equal(t, "alice@example.com", unchanged.Email)
}

func TestProfilePOSTHandlerRequiresSessionUser(t *testing.T) {
	handler, _, _, _ := newProfileHandler(t)

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/profile", jsonTestUUID, MapProfileAccountUpdateRequest{
		FullName: "Alice",
		Email:    "alice@example.com",
	})
	rec := httptest.NewRecorder()

	handler.ProfilePOSTHandler(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProfilePasswordPOSTHandlerChangesPasswordWithCurrentPassword(t *testing.T) {
	handler, sessions, userManager, _ := newProfileHandler(t)

	user, err := userManager.New("alice", "oldpass123", "alice@example.com", "Alice", false, false, jsonTestUUID, 0)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(user))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/profile/password", jsonTestUUID, MapProfilePasswordRequest{
		CurrentPassword: "oldpass123",
		NewPassword:     "newpass123!",
		ConfirmPassword: "newpass123!",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ProfilePasswordPOSTHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MapProfilePasswordResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "Password updated", resp.Message)

	validOld, _ := userManager.CheckLoginCredentials("alice", "oldpass123", jsonTestUUID)
	require.False(t, validOld)
	validNew, _ := userManager.CheckLoginCredentials("alice", "newpass123!", jsonTestUUID)
	require.True(t, validNew)
}

func TestProfilePasswordPOSTHandlerRejectsWrongCurrentPassword(t *testing.T) {
	handler, sessions, userManager, _ := newProfileHandler(t)

	user, err := userManager.New("alice", "oldpass123", "alice@example.com", "Alice", false, false, jsonTestUUID, 0)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(user))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/profile/password", jsonTestUUID, MapProfilePasswordRequest{
		CurrentPassword: "wrongpass",
		NewPassword:     "newpass123!",
		ConfirmPassword: "newpass123!",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ProfilePasswordPOSTHandler(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)

	validOld, _ := userManager.CheckLoginCredentials("alice", "oldpass123", jsonTestUUID)
	require.True(t, validOld)
	validNew, _ := userManager.CheckLoginCredentials("alice", "newpass123!", jsonTestUUID)
	require.False(t, validNew)
}

func TestProfilePasswordPOSTHandlerSuppressesDebugBodyDump(t *testing.T) {
	source := string(mustReadFile(t, "profile.go"))
	accountStart := strings.Index(source, "func (h *HandlersMap) ProfilePOSTHandler")
	require.NotEqual(t, -1, accountStart)
	accountEnd := strings.Index(source[accountStart:], "func (h *HandlersMap) ProfilePasswordPOSTHandler")
	require.NotEqual(t, -1, accountEnd)
	accountHandlerSource := source[accountStart : accountStart+accountEnd]

	require.Contains(t, accountHandlerSource, `DebugHTTPDump(h.DebugHTTP, r, false)`)
	require.NotContains(t, accountHandlerSource, `DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)`)

	start := strings.Index(source, "func (h *HandlersMap) ProfilePasswordPOSTHandler")
	require.NotEqual(t, -1, start)
	end := strings.Index(source[start:], "func (h *HandlersMap) profileSessionUsername")
	require.NotEqual(t, -1, end)
	handlerSource := source[start : start+end]

	require.Contains(t, handlerSource, `DebugHTTPDump(h.DebugHTTP, r, false)`)
	require.NotContains(t, handlerSource, `DebugHTTPDump(h.DebugHTTP, r, h.Config.DebugHTTP.ShowBody)`)
}

func TestGameboardProfileNavigationAndModalAssets(t *testing.T) {
	gameboard := string(mustReadFile(t, filepath.Join("..", "templates", "gameboard.html")))
	require.Contains(t, gameboard, `id="gameboard-profile-link"`)
	require.Contains(t, gameboard, `data-profile-url="/{{ .UUID }}/profile"`)
	require.Contains(t, gameboard, `data-profile-password-url="/{{ .UUID }}/profile/password"`)
	require.NotContains(t, gameboard, `href="/{{ .UUID }}/registration">Registration</a>`)
	require.NotContains(t, gameboard, `.js-profile-last-access`)
	require.NotContains(t, gameboard, `.js-profile-last-ip`)
	require.NotContains(t, gameboard, `.js-profile-last-agent`)

	modal := string(mustReadFile(t, filepath.Join("..", "templates", "static", "inc", "modals", "profile.html")))
	require.Contains(t, modal, `profile-modal`)
	require.Contains(t, modal, `profile-team-metrics`)
	require.Contains(t, modal, `js-profile-team-name`)
	require.Contains(t, modal, `js-profile-team-rank`)
	require.Contains(t, modal, `js-profile-team-last-score`)
	require.Contains(t, modal, `profile-actions-grid`)
	require.Contains(t, modal, `profile-account-panel`)
	require.Contains(t, modal, `profile-password-panel`)
	require.Contains(t, modal, `js-profile-account-form`)
	require.Contains(t, modal, `name="full_name"`)
	require.Contains(t, modal, `name="email"`)
	require.Contains(t, modal, `js-profile-password-form`)
	require.Contains(t, modal, `profile-password-current`)
	require.Contains(t, modal, `current_password`)
	require.NotContains(t, modal, `profile-password-form-wide`)
	require.NotContains(t, modal, `<span class="highlighted">team</span>`)
	require.NotContains(t, modal, `>session<`)
	require.NotContains(t, modal, `js-profile-last-access`)
	require.NotContains(t, modal, `js-profile-last-ip`)
	require.NotContains(t, modal, `js-profile-last-agent`)

	css := string(mustReadFile(t, filepath.Join("..", "templates", "static", "css", "mapctf.css")))
	profileContentStart := strings.Index(css, `.modal--profile .mctf-modal-content {`)
	require.NotEqual(t, -1, profileContentStart)
	profileContentEnd := strings.Index(css[profileContentStart:], `}`)
	require.NotEqual(t, -1, profileContentEnd)
	profileContentCSS := css[profileContentStart : profileContentStart+profileContentEnd]
	require.Contains(t, profileContentCSS, `max-width: 920px;`)
	require.Contains(t, profileContentCSS, `max-height: calc(100vh - 28px);`)
	require.Contains(t, profileContentCSS, `overflow-y: auto;`)
}

func TestProfileGETHandlerTranslatesRoleAndStatus(t *testing.T) {
	db := newJSONTestDB(t)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)
	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{Secret: "test-secret", HoursToExpire: 24})
	require.NoError(t, err)
	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	require.NoError(t, settingsManager.Initialization(jsonTestUUID))
	require.NoError(t, settingsManager.SetLanguage("es", jsonSettingsAuthor, jsonTestUUID))

	catalog, err := i18n.New()
	require.NoError(t, err)
	sessions := scs.New()
	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID, TemplatesDir: filepath.Join("..", "templates")}}),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSettings(settingsManager),
		WithSessions(sessions),
		WithI18N(catalog),
	)

	user, err := userManager.New("alice", "password123", "alice@example.com", "Alice Doe", false, false, jsonTestUUID, 0)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(user))

	req := newRequestWithUUID(http.MethodGet, "/profile", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.ProfileGETHandler)).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp MapProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Jugador", resp.Account.Role)
	require.Equal(t, "Activo", resp.Account.Status)
}

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
)

func newAuthHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *users.UserManager, *settings.SettingsManager) {
	t.Helper()

	db := newJSONTestDB(t)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service", jsonTestUUID)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID: jsonTestUUID,
			},
		}),
		WithUsers(userManager),
		WithSettings(settingsManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, userManager, settingsManager
}

func TestLoginPOSTHandlerBlocksNonAdminWhenLoginDisabled(t *testing.T) {
	handler, sessions, userManager, settingsManager := newAuthHandler(t)

	require.NoError(t, settingsManager.SetLoginEnabled(false, jsonSettingsAuthor))

	user, err := userManager.New("alice", "password123", "alice@example.com", "Alice", false, false, jsonTestUUID, 5)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(user))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/login", jsonTestUUID, MapLoginRequest{
		Username: "alice",
		Password: "password123",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.LoginPOSTHandler(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)

	var resp MapErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "login is disabled", resp.Error)
	require.Empty(t, sessions.GetString(req.Context(), string(ContextKeyUser)))
}

func TestLoginPOSTHandlerAllowsAdminWhenLoginDisabled(t *testing.T) {
	handler, sessions, userManager, settingsManager := newAuthHandler(t)

	require.NoError(t, settingsManager.SetLoginEnabled(false, jsonSettingsAuthor))

	adminUser, err := userManager.New("admin", "password123", "admin@example.com", "Admin", true, false, jsonTestUUID, 0)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(adminUser))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/login", jsonTestUUID, MapLoginRequest{
		Username: "admin",
		Password: "password123",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.LoginPOSTHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MapLoginResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "/"+jsonTestUUID+"/admin", resp.Redirect)
	require.Equal(t, "admin", sessions.GetString(req.Context(), string(ContextKeyUser)))
	require.True(t, sessions.GetBool(req.Context(), string(ContextKeyAdmin)))
}

func TestLoginPOSTHandlerAllowsNonAdminWhenLoginEnabled(t *testing.T) {
	handler, sessions, userManager, settingsManager := newAuthHandler(t)

	require.NoError(t, settingsManager.SetLoginEnabled(true, jsonSettingsAuthor))

	user, err := userManager.New("alice", "password123", "alice@example.com", "Alice", false, false, jsonTestUUID, 5)
	require.NoError(t, err)
	require.NoError(t, userManager.Create(user))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/login", jsonTestUUID, MapLoginRequest{
		Username: "alice",
		Password: "password123",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.LoginPOSTHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MapLoginResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "/"+jsonTestUUID+"/gameboard", resp.Redirect)
	require.Equal(t, "alice", sessions.GetString(req.Context(), string(ContextKeyUser)))
	require.False(t, sessions.GetBool(req.Context(), string(ContextKeyAdmin)))
}

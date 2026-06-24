package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAPISettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	return db
}

func newAPISettingsHandler(t *testing.T) (*HandlersAPI, *users.UserManager) {
	t.Helper()

	const secret = "settings-handler-secret"

	db := newAPISettingsTestDB(t)
	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        secret,
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	require.NoError(t, settingsManager.Initialization("entity-a"))

	handler := CreateHandlersAPI(
		WithUsers(userManager),
		WithSettings(settingsManager),
		WithConfig(config.MapCTFConfiguration{
			JWT: config.ConfigurationJWT{
				Secret:        secret,
				HoursToExpire: 24,
			},
		}),
	)

	return handler, userManager
}

func withAPISettingsUUID(r *http.Request, uuid string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", uuid)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)
	return r.WithContext(ctx)
}

func TestSettingsHandlerRejectsNonAdminToken(t *testing.T) {
	handler, userManager := newAPISettingsHandler(t)

	token, _, err := userManager.CreateTokenForUser("player", "entity-a", false, "mapctf-api", 1)
	require.NoError(t, err)

	req := withAPISettingsUUID(httptest.NewRequest(http.MethodGet, "/api/v1/entity-a/admin/settings", nil), "entity-a")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected := handler.AuthMiddleware(handler.RequireAdmin(http.HandlerFunc(handler.SettingsHandler)))
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "Admin access required")
}

func TestSettingsHandlerAllowsAdminToken(t *testing.T) {
	handler, userManager := newAPISettingsHandler(t)

	token, _, err := userManager.CreateTokenForUser("admin", "entity-a", true, "mapctf-api", 1)
	require.NoError(t, err)

	req := withAPISettingsUUID(httptest.NewRequest(http.MethodGet, "/api/v1/entity-a/admin/settings", nil), "entity-a")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected := handler.AuthMiddleware(handler.RequireAdmin(http.HandlerFunc(handler.SettingsHandler)))
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "Admin access required")
}

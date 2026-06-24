package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTeamHandlerRejectsNonAdminToken(t *testing.T) {
	handler, userManager := newAPISettingsHandler(t)

	token, _, err := userManager.CreateTokenForUser("player", "entity-a", false, "mapctf-api", 1)
	require.NoError(t, err)

	req := withAPISettingsUUID(
		httptest.NewRequest(http.MethodPost, "/api/v1/entity-a/admin/teams", strings.NewReader(`{"name":"red-team"}`)),
		"entity-a",
	)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected := handler.AuthMiddleware(handler.RequireAdmin(http.HandlerFunc(handler.CreateTeamHandler)))
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "Admin access required")
}

func TestCreateChallengeHandlerRejectsNonAdminToken(t *testing.T) {
	handler, userManager := newAPISettingsHandler(t)

	token, _, err := userManager.CreateTokenForUser("player", "entity-a", false, "mapctf-api", 1)
	require.NoError(t, err)

	req := withAPISettingsUUID(
		httptest.NewRequest(http.MethodPost, "/api/v1/entity-a/admin/challenges", strings.NewReader(`{"title":"challenge","flag":"flag{test}"}`)),
		"entity-a",
	)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected := handler.AuthMiddleware(handler.RequireAdmin(http.HandlerFunc(handler.CreateChallengeHandler)))
	protected.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "Admin access required")
}

func TestAdminMutationRoutesAllowAdminToken(t *testing.T) {
	handler, userManager := newAPISettingsHandler(t)

	token, _, err := userManager.CreateTokenForUser("admin", "entity-a", true, "mapctf-api", 1)
	require.NoError(t, err)

	for _, path := range []string{
		"/api/v1/entity-a/admin/teams",
		"/api/v1/entity-a/admin/challenges",
	} {
		req := withAPISettingsUUID(httptest.NewRequest(http.MethodPost, path, nil), "entity-a")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		nextCalled := false

		protected := handler.AuthMiddleware(handler.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusNoContent)
		})))
		protected.ServeHTTP(rec, req)

		require.True(t, nextCalled, "expected admin token to pass auth gate for %s", path)
		require.Equal(t, http.StatusNoContent, rec.Code)
	}
}

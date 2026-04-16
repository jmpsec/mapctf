package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/stretchr/testify/require"
)

func newGameboardTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *settings.SettingsManager) {
	t.Helper()

	db := newJSONTestDB(t)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service", jsonTestUUID)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithSettings(settingsManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, settingsManager
}

func newTemplateRequestWithUUID(method, target, uuid string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", uuid)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestGameboardTemplateHandlerIncludesChatTemplateData(t *testing.T) {
	handler, sessions, settingsManager := newGameboardTemplateHandler(t)

	require.NoError(t, settingsManager.SetGameboardChatMaxLen(64, jsonSettingsAuthor))
	require.NoError(t, settingsManager.SetGameboardShowTeamMembers(true, jsonSettingsAuthor))

	req := newTemplateRequestWithUUID(http.MethodGet, "/gameboard", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GameboardTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, `data-chat-max-len="64"`)
	require.Contains(t, body, `data-current-username="alice"`)
	require.Contains(t, body, `data-module="world-chat"`)
}

func TestGameboardTemplateHandlerFallsBackToDefaultChatMaxLen(t *testing.T) {
	handler, sessions, _ := newGameboardTemplateHandler(t)

	req := newTemplateRequestWithUUID(http.MethodGet, "/gameboard", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GameboardTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `data-chat-max-len="`+strconv.Itoa(chat.DefaultMaxLen)+`"`)
}

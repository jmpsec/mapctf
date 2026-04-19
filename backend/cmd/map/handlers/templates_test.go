package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

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
	gameStartTime := time.Date(2030, time.January, 2, 9, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	gameEndTime := time.Date(2030, time.January, 2, 18, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, settingsManager.SetGameStarted(true, jsonSettingsAuthor))
	require.NoError(t, settingsManager.SetGameStartTime(gameStartTime, jsonSettingsAuthor))
	require.NoError(t, settingsManager.SetGameEndTime(gameEndTime, jsonSettingsAuthor))

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
	require.Contains(t, body, `data-game-started="true"`)
	require.Contains(t, body, `data-game-start-time="2030-01-02T09:00:00+02:00"`)
	require.Contains(t, body, `data-game-end-time="2030-01-02T18:30:00+02:00"`)
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

func TestCountdownTemplateHandlerUsesStartTimeBeforeGameStarts(t *testing.T) {
	handler, sessions, settingsManager := newGameboardTemplateHandler(t)

	startTime := time.Date(2030, time.January, 2, 15, 4, 5, 0, time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, settingsManager.SetGameStartTime(startTime, jsonSettingsAuthor))
	require.NoError(t, settingsManager.SetGameStarted(false, jsonSettingsAuthor))

	req := newTemplateRequestWithUUID(http.MethodGet, "/countdown", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.CountdownTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, `data-countdown-mode="start"`)
	require.Contains(t, body, `data-target-time="2030-01-02T15:04:05+02:00"`)
	require.Contains(t, body, "Starts January 2, 2030 at 15:04 +0200.")
	require.Contains(t, body, "Game on Standby")
}

func TestCountdownTemplateHandlerUsesEndTimeAfterGameStarts(t *testing.T) {
	handler, sessions, settingsManager := newGameboardTemplateHandler(t)

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Date(2030, time.January, 2, 18, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, settingsManager.SetGameStartTime(startTime, jsonSettingsAuthor))
	require.NoError(t, settingsManager.SetGameEndTime(endTime, jsonSettingsAuthor))
	require.NoError(t, settingsManager.SetGameStarted(true, jsonSettingsAuthor))

	req := newTemplateRequestWithUUID(http.MethodGet, "/countdown", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.CountdownTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, `data-countdown-mode="end"`)
	require.Contains(t, body, `data-target-time="2030-01-02T18:30:00+02:00"`)
	require.Contains(t, body, "Ends January 2, 2030 at 18:30 +0200.")
	require.Contains(t, body, "Countdown to game end")
}

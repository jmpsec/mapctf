package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/stretchr/testify/require"
)

func newAdminTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *chat.ChatManager, *teams.TeamManager) {
	t.Helper()

	db := newJSONTestDB(t)

	chatManager, err := chat.CreateChatManager(db, jsonTestUUID)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db, jsonTestUUID)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithChat(chatManager),
		WithTeams(teamManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, chatManager, teamManager
}

func newAdminRequestWithUUID(method, target, uuid string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", uuid)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestAdminChatTemplateHandlerIncludesRecentChatSection(t *testing.T) {
	handler, sessions, chatManager, teamManager := newAdminTemplateHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:   "blue-team",
		UUID:   jsonTestUUID,
		Active: true,
	}))

	allTeams, err := teamManager.GetAll()
	require.NoError(t, err)
	require.Len(t, allTeams, 1)

	require.NoError(t, chatManager.CreateNew("alice", "hello admin chat", allTeams[0].ID, chat.DefaultMaxLen))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/chat", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "World Chat")
	require.Contains(t, body, "hello admin chat")
	require.Contains(t, body, "blue-team")
	require.Contains(t, body, "alice")
}

func TestAdminChatTemplateHandlerShowsEmptyChatState(t *testing.T) {
	handler, sessions, _, _ := newAdminTemplateHandler(t)

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/chat", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "No chat messages yet.")
}

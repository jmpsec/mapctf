package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func TestAdminChatTemplateHandlerIncludesModerationControls(t *testing.T) {
	handler, sessions, chatManager, teamManager := newAdminTemplateHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:   "blue-team",
		UUID:   jsonTestUUID,
		Active: true,
	}))
	allTeams, err := teamManager.GetAll()
	require.NoError(t, err)

	require.NoError(t, chatManager.CreateNew("alice", "needs moderation", allTeams[0].ID, chat.DefaultMaxLen))

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
	require.Contains(t, body, "/admin/chat/")
	require.Contains(t, body, "/visibility")
	require.Contains(t, body, "/delete")
	require.Contains(t, body, "admin-chat-list")
	require.Contains(t, body, "data-admin-reload-on-success=\"true\"")
	require.Contains(t, body, "Hide")
	require.Contains(t, body, "Delete")
}

func TestAdminChatSetHiddenPOSTHandlerHidesEntry(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "hide me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/visibility", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)
	req.PostForm = map[string][]string{"hidden": {"true"}}

	rr := httptest.NewRecorder()
	handler.AdminChatSetHiddenPOSTHandler(rr, req)

	require.Equal(t, http.StatusFound, rr.Code)
	updated, err := chatManager.GetByID(entries[0].ID)
	require.NoError(t, err)
	require.True(t, updated.Hidden)
	require.Contains(t, rr.Header().Get("Location"), "status=ok")
}

func TestAdminChatSetHiddenPOSTHandlerHidesEntryForAJAX(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "hide me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	reqBody := bytes.NewBufferString(`{"hidden":true}`)
	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/visibility", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	req.Header.Set(ContentType, JSONApplication)
	req.Header.Set("Accept", JSONApplication)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Body = io.NopCloser(reqBody)
	req.ContentLength = int64(reqBody.Len())

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatSetHiddenPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Chat message hidden", resp.Message)

	updated, err := chatManager.GetByID(entries[0].ID)
	require.NoError(t, err)
	require.True(t, updated.Hidden)
}

func TestAdminChatSetHiddenPOSTHandlerHidesEntryForAJAXStringValue(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "hide me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	reqBody := bytes.NewBufferString(`{"hidden":"true"}`)
	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/visibility", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	req.Header.Set(ContentType, JSONApplication)
	req.Header.Set("Accept", JSONApplication)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Body = io.NopCloser(reqBody)
	req.ContentLength = int64(reqBody.Len())

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatSetHiddenPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Chat message hidden", resp.Message)

	updated, err := chatManager.GetByID(entries[0].ID)
	require.NoError(t, err)
	require.True(t, updated.Hidden)
}

func TestAdminChatDeletePOSTHandlerDeletesScopedEntry(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "delete me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/delete", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatDeletePOSTHandler(rr, req)

	require.Equal(t, http.StatusFound, rr.Code)
	remaining, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, remaining, 0)
	require.Contains(t, rr.Header().Get("Location"), "status=ok")
}

func TestAdminChatDeletePOSTHandlerDeletesScopedEntryForAJAX(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "delete me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/delete", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	req.Header.Set(ContentType, JSONApplication)
	req.Header.Set("Accept", JSONApplication)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Body = io.NopCloser(bytes.NewBufferString(`{}`))

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatDeletePOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Chat message deleted", resp.Message)

	remaining, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, remaining, 0)
}

func TestJSONChatHandlerExcludesHiddenEntries(t *testing.T) {
	db := newJSONTestDB(t)

	chatManager, err := chat.CreateChatManager(db, jsonTestUUID)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{}),
		WithChat(chatManager),
	)

	require.NoError(t, chatManager.Create(chat.ChatEntry{
		Username: "visible",
		Body:     "shown",
		TeamID:   1,
		UUID:     jsonTestUUID,
		Hidden:   false,
	}))
	require.NoError(t, chatManager.Create(chat.ChatEntry{
		Username: "hidden",
		Body:     "secret",
		TeamID:   1,
		UUID:     jsonTestUUID,
		Hidden:   true,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/chat", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONChatHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []chat.ChatEntry
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	require.Equal(t, "visible", resp[0].Username)
}

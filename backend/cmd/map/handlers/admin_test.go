package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAdminTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *chat.ChatManager, *teams.TeamManager) {
	t.Helper()

	db := newJSONTestDB(t)

	chatManager, err := chat.CreateChatManager(db, jsonTestUUID)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db)
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

func newAdminCountryActionHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *countries.CountriesManager, *challenges.ChallengeManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, countriesManager, challengesManager
}

func newAdminChallengeActivityHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *countries.CountriesManager, *challenges.ChallengeManager, *logs.LogManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithLogs(logManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, countriesManager, challengesManager, logManager
}

func newAdminActivityTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *logs.LogManager) {
	t.Helper()

	db := newJSONTestDB(t)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithLogs(logManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, logManager
}

func newAdminChallengesTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *countries.CountriesManager, *challenges.ChallengeManager, *logs.LogManager, *teams.TeamManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithLogs(logManager),
		WithTeams(teamManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, countriesManager, challengesManager, logManager, teamManager
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

	allTeams, err := teamManager.GetAll(jsonTestUUID)
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

func TestAdminActivityTemplateHandlerIncludesActivityEntries(t *testing.T) {
	handler, sessions, logManager := newAdminActivityTemplateHandler(t)

	activity, err := logManager.NewActivity(true, "Blue Team", "completed", "Captured Spain", 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, logManager.CreateActivity(activity))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/activity", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminActivityTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Activity Log")
	require.Contains(t, body, "Blue Team")
	require.Contains(t, body, "completed")
	require.Contains(t, body, "Captured Spain")
}

func TestAdminActivityPOSTHandlerCreatesCustomEntry(t *testing.T) {
	handler, sessions, logManager := newAdminActivityTemplateHandler(t)

	payload := AdminActivityCreateRequest{
		Subject: "Blue Team",
		Action:  "custom",
		Message: "Custom activity",
		Visible: true,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/activity", bytes.NewReader(body))
	req.Header.Set(ContentType, JSONApplicationUTF8)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", jsonTestUUID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminActivityPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 1)
	require.True(t, activityEntries[0].Visible)
	require.Equal(t, "Blue Team", activityEntries[0].Subject)
	require.Equal(t, "custom", activityEntries[0].Action)
	require.Equal(t, "Custom activity", activityEntries[0].Message)
}

func TestAdminActivityDeletePOSTHandlerDeletesEntry(t *testing.T) {
	handler, sessions, logManager := newAdminActivityTemplateHandler(t)

	activity, err := logManager.NewActivity(true, "Blue Team", "announcement", "Delete me", 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, logManager.CreateActivity(activity))

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/activity/1/delete", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", strconv.FormatUint(uint64(activityEntries[0].ID), 10))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminActivityDeletePOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err = logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 0)
}

func TestAdminChallengesTemplateHandlerShowsChallengeRelatedActivity(t *testing.T) {
	handler, sessions, countriesManager, challengesManager, logManager, teamManager := newAdminChallengesTemplateHandler(t)

	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
	}))
	require.NoError(t, challengesManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 3},
		Name:  "Web",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Model:      gorm.Model{ID: 77},
		Title:      "Spanish Challenge",
		CategoryID: 3,
		Country:    "ES",
		Active:     true,
		Points:     100,
		Flag:       "MAP{es}",
		UUID:       jsonTestUUID,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 5},
		Name:    "Blue Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))

	activity, err := logManager.NewActivity(true, "Blue Team", "completed", "Spanish Challenge", 77, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, logManager.CreateActivity(activity))
	require.NoError(t, logManager.CreateFailuresLog(logs.FailuresLog{
		ChallengeID: 77,
		TeamID:      5,
		Flag:        "wrong-flag",
		UUID:        jsonTestUUID,
	}))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/challenges", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChallengesTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Spanish Challenge")
	require.Contains(t, body, "Blue Team")
	require.Contains(t, body, "completed")
	require.Contains(t, body, "Failure")
	require.Contains(t, body, "wrong-flag")
}

func TestAdminChallengeUpdatePOSTHandlerLogsEnableAndDisableStateChanges(t *testing.T) {
	handler, sessions, _, challengesManager, logManager := newAdminChallengeActivityHandler(t)

	require.NoError(t, challengesManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 1},
		Name:  "Web",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Model:       gorm.Model{ID: 50},
		Title:       "Spain",
		CategoryID:  1,
		Active:      false,
		Points:      100,
		Bonus:       0,
		BonusDecay:  0,
		HintPenalty: 0,
		HelpPenalty: 0,
		Flag:        "MAP{es}",
		Hint:        "hint",
		UUID:        jsonTestUUID,
	}))

	makeRequest := func(activeValue string) *http.Request {
		payload := AdminChallengeCreateRequest{
			Title:       "Spain",
			Description: "",
			CategoryID:  "1",
			Country:     "",
			Active:      activeValue,
			Points:      "100",
			Bonus:       "0",
			BonusDecay:  "0",
			HintPenalty: "0",
			HelpPenalty: "0",
			Flag:        "MAP{es}",
			Hint:        "hint",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/admin/challenges/50", bytes.NewReader(body))
		req.Header.Set(ContentType, JSONApplicationUTF8)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("uuid", jsonTestUUID)
		routeCtx.URLParams.Add("id", "50")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
		ctx, err := sessions.Load(req.Context(), "")
		require.NoError(t, err)
		sessions.Put(ctx, string(ContextKeyUser), "admin")
		sessions.Put(ctx, string(ContextKeyAdmin), true)
		return req.WithContext(ctx)
	}

	rr := httptest.NewRecorder()
	handler.AdminChallengeUpdatePOSTHandler(rr, makeRequest("true"))
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.AdminChallengeUpdatePOSTHandler(rr, makeRequest("false"))
	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 2)
	require.Equal(t, "admin", activityEntries[0].Subject)
	require.Equal(t, "enabled", activityEntries[0].Action)
	require.Equal(t, "Challenge Spain (Web) was enabled: 100 points", activityEntries[0].Message)
	require.Equal(t, uint(50), activityEntries[0].ChallengeID)
	require.Equal(t, "disabled", activityEntries[1].Action)
}

func TestAdminChallengesBulkStateChangeHandlersLogActivity(t *testing.T) {
	handler, sessions, _, challengesManager, logManager := newAdminChallengeActivityHandler(t)

	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:  "One",
		Active: false,
		Flag:   "MAP{one}",
		UUID:   jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:  "Two",
		Active: false,
		Flag:   "MAP{two}",
		UUID:   jsonTestUUID,
	}))

	makeRequest := func() *http.Request {
		req := newAdminRequestWithUUID(http.MethodPost, "/admin/challenges", jsonTestUUID)
		ctx, err := sessions.Load(req.Context(), "")
		require.NoError(t, err)
		sessions.Put(ctx, string(ContextKeyUser), "admin")
		sessions.Put(ctx, string(ContextKeyAdmin), true)
		return req.WithContext(ctx)
	}

	rr := httptest.NewRecorder()
	handler.AdminChallengesEnableAllPOSTHandler(rr, makeRequest())
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.AdminChallengesDisableAllPOSTHandler(rr, makeRequest())
	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 2)
	require.Equal(t, "enabled", activityEntries[0].Action)
	require.Equal(t, "enabled all challenges (2)", activityEntries[0].Message)
	require.Equal(t, "disabled", activityEntries[1].Action)
	require.Equal(t, "disabled all challenges (2)", activityEntries[1].Message)
}

func TestAdminChatTemplateHandlerIncludesModerationControls(t *testing.T) {
	handler, sessions, chatManager, teamManager := newAdminTemplateHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:   "blue-team",
		UUID:   jsonTestUUID,
		Active: true,
	}))
	allTeams, err := teamManager.GetAll(jsonTestUUID)
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
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
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

func TestAdminCountriesDeleteAllPOSTHandlerDeletesScopedCountriesAndClearsChallengeCountries(t *testing.T) {
	handler, sessions, countriesManager, challengesManager := newAdminCountryActionHandler(t)

	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
	}))
	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "France",
		CountryCode: "FR",
		Active:      true,
	}))
	require.NoError(t, countriesManager.DB.Create(&countries.MapCountry{
		Name:        "Other UUID Country",
		CountryCode: "DE",
		Active:      true,
		UUID:        jsonOtherTestUUID,
	}).Error)

	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Scoped challenge",
		Country: "ES",
		Active:  true,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Other UUID challenge",
		Country: "DE",
		Active:  true,
		UUID:    jsonOtherTestUUID,
	}))

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/countries/delete-all", jsonTestUUID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set(ContentType, JSONApplication)
	req.Body = io.NopCloser(bytes.NewBufferString(`{}`))

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminCountriesDeleteAllPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Deleted 2 country(ies)", resp.Message)

	remainingCountries, err := countriesManager.GetAll()
	require.NoError(t, err)
	require.Empty(t, remainingCountries)

	var otherCountries []countries.MapCountry
	require.NoError(t, countriesManager.DB.Where("uuid = ?", jsonOtherTestUUID).Find(&otherCountries).Error)
	require.Len(t, otherCountries, 1)
	require.Equal(t, "DE", otherCountries[0].CountryCode)

	scopedChallenge, err := challengesManager.GetByID(1, jsonTestUUID)
	require.NoError(t, err)
	require.Empty(t, scopedChallenge.Country)

	otherChallenge, err := challengesManager.GetByID(2, jsonOtherTestUUID)
	require.NoError(t, err)
	require.Equal(t, "DE", otherChallenge.Country)
}

func TestAdminCountriesDeleteAllPOSTHandlerReturnsErrorWithoutManagers(t *testing.T) {
	sessionManager := scs.New()
	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithSessions(sessionManager),
	)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/countries/delete-all", jsonTestUUID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set(ContentType, JSONApplication)
	req.Body = io.NopCloser(bytes.NewBufferString(`{}`))

	ctx, err := sessionManager.Load(req.Context(), "")
	require.NoError(t, err)
	sessionManager.Put(ctx, string(ContextKeyUser), "admin")
	sessionManager.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminCountriesDeleteAllPOSTHandler(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "error", resp.Status)
	require.Equal(t, "Countries or challenges manager is not initialized", resp.Message)
}

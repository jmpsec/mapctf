package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
)

func newGameboardTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *settings.SettingsManager, *countries.CountriesManager, *challenges.ChallengeManager) {
	t.Helper()

	db := newJSONTestDB(t)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)
	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)
	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)
	userManager, err := users.CreateUserManager(db, nil)
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
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, settingsManager, countriesManager, challengesManager
}

func newTemplateRequestWithUUID(method, target, uuid string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", uuid)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestGameboardTemplateHandlerIncludesChatTemplateData(t *testing.T) {
	handler, sessions, settingsManager, _, _ := newGameboardTemplateHandler(t)

	require.NoError(t, settingsManager.SetGameboardChatMaxLen(64, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameboardShowTeamMembers(true, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetScoringHints(true, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetScoringHelp(false, jsonSettingsAuthor, jsonTestUUID))
	gameStartTime := time.Date(2030, time.January, 2, 9, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	gameEndTime := time.Date(2030, time.January, 2, 18, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, settingsManager.SetGameStarted(true, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameStartTime(gameStartTime, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameEndTime(gameEndTime, jsonSettingsAuthor, jsonTestUUID))

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
	require.Contains(t, body, `data-scoring-hints="true"`)
	require.Contains(t, body, `data-scoring-help="false"`)
	require.Contains(t, body, `data-game-started="true"`)
	require.Contains(t, body, `data-game-start-time="2030-01-02T09:00:00+02:00"`)
	require.Contains(t, body, `data-game-end-time="2030-01-02T18:30:00+02:00"`)
	require.Contains(t, body, `data-module="world-chat"`)
}

func TestGameboardTemplateHandlerIncludesCurrentTeam(t *testing.T) {
	handler, sessions, _, _, _ := newGameboardTemplateHandler(t)

	require.NoError(t, handler.Teams.Create(teams.PlatformTeam{
		Name:    "Blue Team",
		Logo:    "bee",
		Active:  true,
		Visible: true,
		UUID:    jsonTestUUID,
	}))
	team, err := handler.Teams.Get("Blue Team", jsonTestUUID)
	require.NoError(t, err)
	user, err := handler.Users.New("alice", "password123", "alice@example.com", "Alice", false, false, jsonTestUUID, team.ID)
	require.NoError(t, err)
	require.NoError(t, handler.Users.Create(user))

	req := newTemplateRequestWithUUID(http.MethodGet, "/gameboard", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GameboardTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `data-current-team="Blue Team"`)
}

func TestGameboardActivityRendererIncludesTimestamps(t *testing.T) {
	js, err := os.ReadFile(filepath.Join("..", "templates", "static", "js", "mapctf.js"))
	require.NoError(t, err)
	css, err := os.ReadFile(filepath.Join("..", "templates", "static", "css", "mapctf.css"))
	require.NoError(t, err)

	jsBody := string(js)
	require.Contains(t, jsBody, "function formatActivityTime")
	require.Contains(t, jsBody, "activity-time")
	require.Contains(t, jsBody, "CreatedAt")
	require.Contains(t, jsBody, "hour12: false")
	require.Contains(t, string(css), ".activity-time")
}

func TestGameboardTemplateHandlerFallsBackToDefaultChatMaxLen(t *testing.T) {
	handler, sessions, _, _, _ := newGameboardTemplateHandler(t)

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
	handler, sessions, settingsManager, _, _ := newGameboardTemplateHandler(t)

	startTime := time.Date(2030, time.January, 2, 15, 4, 5, 0, time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, settingsManager.SetGameStartTime(startTime, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameStarted(false, jsonSettingsAuthor, jsonTestUUID))

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
	handler, sessions, settingsManager, _, _ := newGameboardTemplateHandler(t)

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Date(2030, time.January, 2, 18, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, settingsManager.SetGameStartTime(startTime, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameEndTime(endTime, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameStarted(true, jsonSettingsAuthor, jsonTestUUID))

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

func TestGameboardTemplateHandlerRendersAllCountriesAndMarksChallengeBackedOnesActive(t *testing.T) {
	handler, sessions, _, countriesManager, challengesManager := newGameboardTemplateHandler(t)

	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
		LandPath:    "L1",
		LandClass:   "land",
		LandStyle:   "stroke-width:1px",
		MarkerPath:  "M1",
		MarkerClass: "map-indicator",
	}))
	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "France",
		CountryCode: "FR",
		Active:      true,
		LandPath:    "L2",
		LandClass:   "land",
		LandStyle:   "stroke-width:1px",
		MarkerPath:  "M2",
		MarkerClass: "map-indicator",
	}))
	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "Germany",
		CountryCode: "DE",
		Active:      false,
		LandPath:    "L3",
		LandClass:   "land",
		LandStyle:   "stroke-width:1px",
		MarkerPath:  "M3",
		MarkerClass: "map-indicator",
	}))

	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Active Spain",
		Country: "ES",
		Active:  true,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Inactive France",
		Country: "FR",
		Active:  false,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Active Germany inactive country",
		Country: "DE",
		Active:  true,
		UUID:    jsonTestUUID,
	}))

	req := newTemplateRequestWithUUID(http.MethodGet, "/gameboard", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GameboardTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, `id="ES" title="Spain" class="land active"`)
	require.Contains(t, body, `id="FR" title="France" class="land"`)
	require.Contains(t, body, `id="DE" title="Germany" class="land active"`)
}

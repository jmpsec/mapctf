package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	jsonTestUUID       = "json-test-uuid"
	jsonOtherTestUUID  = "json-other-test-uuid"
	jsonSettingsAuthor = "test-admin"
)

func newJSONTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return db
}

func newJSONTeamsHandler(t *testing.T) (*HandlersMap, *teams.TeamManager, *users.UserManager, *settings.SettingsManager) {
	t.Helper()

	db := newJSONTestDB(t)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSettings(settingsManager),
	)

	return handler, teamManager, userManager, settingsManager
}

func newJSONActivityHandler(t *testing.T) (*HandlersMap, *logs.LogManager) {
	t.Helper()

	db := newJSONTestDB(t)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithLogs(logManager),
	)

	return handler, logManager
}

func newJSONCountryDataHandler(t *testing.T) (*HandlersMap, *countries.CountriesManager, *challenges.ChallengeManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countryManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithCountries(countryManager),
		WithChallenges(challengeManager),
	)

	return handler, countryManager, challengeManager
}

func newJSONWorldDominationHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *teams.TeamManager, *users.UserManager, *challenges.ChallengeManager) {
	t.Helper()

	db := newJSONTestDB(t)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithChallenges(challengeManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, teamManager, userManager, challengeManager
}

func newJSONGameClockHandler(t *testing.T) (*HandlersMap, *settings.SettingsManager) {
	t.Helper()

	db := newJSONTestDB(t)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithSettings(settingsManager),
	)

	return handler, settingsManager
}

func newRequestWithUUID(method, target, uuid string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	routeCtx := chi.NewRouteContext()
	if uuid != "" {
		routeCtx.URLParams.Add("uuid", uuid)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func decodeJSONTeamResponses(t *testing.T, body []byte) []JSONTeamResponse {
	t.Helper()

	var resp []JSONTeamResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func decodeJSONMapSlice(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	var resp []map[string]any
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestJSONActivityHandlerExcludesHiddenEntries(t *testing.T) {
	handler, logManager := newJSONActivityHandler(t)

	require.NoError(t, logManager.CreateActivity(logs.ActivityLog{
		Visible: true,
		Subject: "Blue Team",
		Action:  "completed",
		Message: "Shown",
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, logManager.CreateActivity(logs.ActivityLog{
		Visible: false,
		Subject: "admin",
		Action:  "created",
		Message: "Hidden",
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, logManager.CreateActivity(logs.ActivityLog{
		Visible: true,
		Subject: "Other",
		Action:  "completed",
		Message: "Other UUID",
		UUID:    jsonOtherTestUUID,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/activity", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONActivityHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp []logs.ActivityLog
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	require.Equal(t, "Shown", resp[0].Message)
	require.True(t, resp[0].Visible)
}

func TestJSONGameClockHandlerReturnsServerClockSettings(t *testing.T) {
	handler, settingsManager := newJSONGameClockHandler(t)
	startTime := time.Date(2030, 1, 2, 9, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	endTime := startTime.Add(3*time.Hour + 30*time.Minute)

	require.NoError(t, settingsManager.SetGameStarted(true, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGamePaused(true, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameStartTime(startTime, jsonSettingsAuthor, jsonTestUUID))
	require.NoError(t, settingsManager.SetGameEndTime(endTime, jsonSettingsAuthor, jsonTestUUID))

	req := newRequestWithUUID(http.MethodGet, "/json/game-clock", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONGameClockHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp JSONGameClockResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.GameStarted)
	require.True(t, resp.GamePaused)
	require.NotZero(t, resp.ServerTime)
	require.NotNil(t, resp.GameStartTime)
	require.NotNil(t, resp.GameEndTime)
	require.Equal(t, startTime.Format(time.RFC3339), resp.GameStartTime.Format(time.RFC3339))
	require.Equal(t, endTime.Format(time.RFC3339), resp.GameEndTime.Format(time.RFC3339))
	require.Greater(t, resp.DurationMS, int64(0))
	require.GreaterOrEqual(t, resp.RemainingMS, int64(0))
}

func TestJSONTeamsHandlerRequiresUUID(t *testing.T) {
	handler, _, _, _ := newJSONTeamsHandler(t)

	req := newRequestWithUUID(http.MethodGet, "/json/teams", "")
	rr := httptest.NewRecorder()

	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp MapErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "UUID is required", resp.Error)
}

func TestJSONTeamsHandlerRejectsInvalidUUID(t *testing.T) {
	handler, _, _, _ := newJSONTeamsHandler(t)

	req := newRequestWithUUID(http.MethodGet, "/json/teams", "wrong-uuid")
	rr := httptest.NewRecorder()

	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp MapErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "invalid UUID", resp.Error)
}

func TestJSONTeamsHandlerFiltersInactiveAndInvisibleTeams(t *testing.T) {
	handler, teamManager, _, _ := newJSONTeamsHandler(t)

	visibleLastScore := time.Date(2026, 4, 16, 10, 30, 0, 0, time.UTC)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:      "visible-team",
		Logo:      "alpha.svg",
		Points:    120,
		LastScore: visibleLastScore,
		Visible:   true,
		Active:    true,
		UUID:      jsonTestUUID,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:    "hidden-team",
		Logo:    "hidden.svg",
		Points:  80,
		Visible: false,
		Active:  true,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:    "inactive-team",
		Logo:    "inactive.svg",
		Points:  60,
		Visible: true,
		Active:  false,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:    "other-uuid-team",
		Logo:    "other.svg",
		Points:  999,
		Visible: true,
		Active:  true,
		UUID:    jsonOtherTestUUID,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/teams", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	resp := decodeJSONMapSlice(t, rr.Body.Bytes())
	require.Len(t, resp, 1)
	require.Equal(t, "visible-team", resp[0]["name"])
	require.Equal(t, "alpha.svg", resp[0]["logo"])
	require.Equal(t, float64(120), resp[0]["points"])
	_, hasMembers := resp[0]["team_members"]
	require.False(t, hasMembers)

	parsed := decodeJSONTeamResponses(t, rr.Body.Bytes())
	require.NotNil(t, parsed[0].LastScore)
	require.Equal(t, visibleLastScore.UTC(), parsed[0].LastScore.UTC())
}

func TestJSONTeamsHandlerOmitsZeroLastScore(t *testing.T) {
	handler, teamManager, _, _ := newJSONTeamsHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:    "new-team",
		Logo:    "alpha.svg",
		Points:  0,
		Visible: true,
		Active:  true,
		UUID:    jsonTestUUID,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/teams", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	resp := decodeJSONMapSlice(t, rr.Body.Bytes())
	require.Len(t, resp, 1)
	_, hasLastScore := resp[0]["last_score"]
	require.False(t, hasLastScore)
}

func TestJSONTeamsHandlerIncludesMembersOnlyWhenSettingEnabled(t *testing.T) {
	handler, teamManager, userManager, settingsManager := newJSONTeamsHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 10},
		Name:    "alpha",
		Logo:    "alpha.svg",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))

	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "alice",
		TeamID:   10,
		Active:   true,
		UUID:     jsonTestUUID,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "bob",
		TeamID:   10,
		Active:   true,
		UUID:     jsonTestUUID,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/teams", jsonTestUUID)
	rr := httptest.NewRecorder()
	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	respWithoutMembers := decodeJSONMapSlice(t, rr.Body.Bytes())
	require.Len(t, respWithoutMembers, 1)
	_, hasMembers := respWithoutMembers[0]["team_members"]
	require.False(t, hasMembers)

	require.NoError(t, settingsManager.SetGameboardShowTeamMembers(true, jsonSettingsAuthor, jsonTestUUID))

	rr = httptest.NewRecorder()
	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	respWithMembers := decodeJSONTeamResponses(t, rr.Body.Bytes())
	require.Len(t, respWithMembers, 1)
	require.ElementsMatch(t, []string{"alice", "bob"}, respWithMembers[0].TeamMembers)
}

func TestJSONTeamsHandlerSkipsUsersThatShouldNotAppearAsMembers(t *testing.T) {
	handler, teamManager, userManager, settingsManager := newJSONTeamsHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 25},
		Name:    "blue-team",
		Logo:    "blue.svg",
		Points:  42,
		Visible: true,
		Active:  true,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, settingsManager.SetGameboardShowTeamMembers(true, jsonSettingsAuthor, jsonTestUUID))

	seedUsers := []users.PlatformUser{
		{Username: "valid-member", TeamID: 25, Active: true, UUID: jsonTestUUID},
		{Username: "inactive-member", TeamID: 25, Active: false, UUID: jsonTestUUID},
		{Username: "service-member", TeamID: 25, Active: true, Service: true, UUID: jsonTestUUID},
		{Username: "no-team-member", TeamID: 0, Active: true, UUID: jsonTestUUID},
		{Username: "other-uuid-member", TeamID: 25, Active: true, UUID: jsonOtherTestUUID},
	}
	for _, user := range seedUsers {
		require.NoError(t, userManager.Create(user))
	}

	req := newRequestWithUUID(http.MethodGet, "/json/teams", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONTeamsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	resp := decodeJSONTeamResponses(t, rr.Body.Bytes())
	require.Len(t, resp, 1)
	require.Equal(t, "blue-team", resp[0].Name)
	require.Equal(t, 42, resp[0].Points)
	require.Equal(t, []string{"valid-member"}, resp[0].TeamMembers)
}

func TestJSONCountriesHandlerRequiresUUID(t *testing.T) {
	handler, _, _ := newJSONCountryDataHandler(t)

	req := newRequestWithUUID(http.MethodGet, "/json/countries", "")
	rr := httptest.NewRecorder()

	handler.JSONCountriesHandler(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp MapErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "UUID is required", resp.Error)
}

func TestJSONCountriesHandlerReturnsAllCountriesAndMarksChallengeBackedOnesActive(t *testing.T) {
	handler, countryManager, challengeManager := newJSONCountryDataHandler(t)

	require.NoError(t, countryManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
	}))
	require.NoError(t, countryManager.Create(countries.MapCountry{
		Name:        "France",
		CountryCode: "FR",
		Active:      false,
	}))
	require.NoError(t, countryManager.Create(countries.MapCountry{
		Name:        "Italy",
		CountryCode: "IT",
		Active:      true,
	}))
	require.NoError(t, countryManager.DB.Create(&countries.MapCountry{
		Name:        "Germany",
		CountryCode: "DE",
		Active:      true,
		UUID:        jsonOtherTestUUID,
	}).Error)

	category := challenges.Category{
		Model:       gorm.Model{ID: 7},
		Name:        "Web",
		Description: "Web category",
		UUID:        jsonTestUUID,
	}
	require.NoError(t, challengeManager.CreateCategory(category))
	require.NoError(t, challengeManager.DB.Create(&challenges.Category{
		Model: gorm.Model{ID: 8},
		Name:  "Other",
		UUID:  jsonOtherTestUUID,
	}).Error)

	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Title:       "Spanish challenge",
		Description: "Live intro",
		URL:         "https://example.com/challenges/spain",
		CategoryID:  7,
		Country:     "ES",
		Active:      true,
		Points:      250,
		HintPenalty: 15,
		HelpPenalty: 40,
		Hint:        "Live hint",
		UUID:        jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Title:      "Inactive country challenge",
		CategoryID: 7,
		Country:    "FR",
		Active:     true,
		Points:     100,
		UUID:       jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Title:      "Inactive challenge",
		CategoryID: 7,
		Country:    "ES",
		Active:     false,
		Points:     500,
		UUID:       jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Title:      "Other UUID challenge",
		CategoryID: 8,
		Country:    "DE",
		Active:     true,
		Points:     999,
		UUID:       jsonOtherTestUUID,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/countries", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONCountriesHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]JSONCountryDataResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp, 3)

	spain, ok := resp["Spain"]
	require.True(t, ok)
	require.True(t, spain.Active)
	require.Equal(t, 250, spain.Points)
	require.Equal(t, 15, spain.HintPenalty)
	require.Equal(t, 40, spain.HelpPenalty)
	require.Equal(t, "Web", spain.Category)
	require.Equal(t, "Live intro", spain.Intro)
	require.Equal(t, "https://example.com/challenges/spain", spain.URL)
	require.Equal(t, "", spain.Owner)
	require.Empty(t, spain.Completed)
	require.False(t, spain.SolvedByCurrent)

	france, ok := resp["France"]
	require.True(t, ok)
	require.True(t, france.Active)
	require.Equal(t, 100, france.Points)
	require.Equal(t, "Web", france.Category)

	italy, ok := resp["Italy"]
	require.True(t, ok)
	require.False(t, italy.Active)
	require.Equal(t, 0, italy.Points)
	require.Empty(t, italy.Category)
	require.Empty(t, italy.Intro)
}

func TestJSONCountriesHandlerIncludesOwnerAndCompletedTeams(t *testing.T) {
	db := newJSONTestDB(t)

	countryManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithCountries(countryManager),
		WithChallenges(challengeManager),
		WithTeams(teamManager),
	)

	require.NoError(t, countryManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
	}))

	require.NoError(t, challengeManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 2},
		Name:  "Crypto",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:      gorm.Model{ID: 101},
		Title:      "Spanish",
		CategoryID: 2,
		Country:    "ES",
		Active:     true,
		Points:     300,
		UUID:       jsonTestUUID,
	}))

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 10},
		Name:    "Blue Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 11},
		Name:    "Red Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))

	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      10,
		ChallengeID: 101,
		Points:      300,
		UUID:        jsonTestUUID,
		ScoredBy:    "alice",
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      11,
		ChallengeID: 101,
		Points:      300,
		UUID:        jsonTestUUID,
		ScoredBy:    "bob",
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      10,
		ChallengeID: 101,
		Points:      300,
		UUID:        jsonTestUUID,
		ScoredBy:    "alice",
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/countries", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONCountriesHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]JSONCountryDataResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Blue Team", resp["Spain"].Owner)
	require.Equal(t, []string{"Blue Team", "Red Team"}, resp["Spain"].Completed)
	require.False(t, resp["Spain"].SolvedByCurrent)
}

func TestJSONCountriesHandlerMarksCountriesSolvedByCurrentTeam(t *testing.T) {
	db := newJSONTestDB(t)

	countryManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithCountries(countryManager),
		WithChallenges(challengeManager),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSessions(sessionManager),
	)

	require.NoError(t, countryManager.Create(countries.MapCountry{
		Name:        "France",
		CountryCode: "FR",
		Active:      true,
	}))
	require.NoError(t, challengeManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 3},
		Name:  "Web",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:      gorm.Model{ID: 202},
		Title:      "French",
		CategoryID: 3,
		Country:    "FR",
		Active:     true,
		Points:     150,
		UUID:       jsonTestUUID,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 31},
		Name:    "Blue Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "alice",
		TeamID:   31,
		Active:   true,
		UUID:     jsonTestUUID,
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      31,
		ChallengeID: 202,
		Points:      150,
		UUID:        jsonTestUUID,
		ScoredBy:    "alice",
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/countries", jsonTestUUID)
	ctx, err := sessionManager.Load(req.Context(), "")
	require.NoError(t, err)
	sessionManager.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.JSONCountriesHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]JSONCountryDataResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp["France"].SolvedByCurrent)
}

func TestJSONWorldDominationHandlerReturnsCurrentTeamMetrics(t *testing.T) {
	handler, sessions, teamManager, userManager, challengeManager := newJSONWorldDominationHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 10},
		Name:    "Blue Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 11},
		Name:    "Red Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))

	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "alice",
		TeamID:   10,
		Active:   true,
		UUID:     jsonTestUUID,
	}))

	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:  gorm.Model{ID: 101},
		Title:  "One",
		Active: true,
		UUID:   jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:  gorm.Model{ID: 102},
		Title:  "Two",
		Active: true,
		UUID:   jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:  gorm.Model{ID: 103},
		Title:  "Three",
		Active: true,
		UUID:   jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:  gorm.Model{ID: 104},
		Title:  "Inactive",
		Active: false,
		UUID:   jsonTestUUID,
	}))

	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      10,
		ChallengeID: 101,
		Points:      100,
		UUID:        jsonTestUUID,
		ScoredBy:    "alice",
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      10,
		ChallengeID: 102,
		Points:      100,
		UUID:        jsonTestUUID,
		ScoredBy:    "alice",
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      11,
		ChallengeID: 103,
		Points:      100,
		UUID:        jsonTestUUID,
		ScoredBy:    "bob",
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      11,
		ChallengeID: 104,
		Points:      100,
		UUID:        jsonTestUUID,
		ScoredBy:    "bob",
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/domination", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.JSONWorldDominationHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp JSONWorldDominationResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Blue Team", resp.CurrentTeam)
	require.Equal(t, 2, resp.CompletedChallenges)
	require.Equal(t, 3, resp.TotalChallenges)
	require.Equal(t, 67, resp.CompletionPct)
	require.Equal(t, 67, resp.WinRatePct)
	require.Equal(t, 33, resp.LoseRatePct)
}

func TestJSONChallengesFeedCacheServesL1AndInvalidates(t *testing.T) {
	db := newJSONTestDB(t)
	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Title:   "Cached challenge",
		Country: "ES",
		Active:  true,
		Points:  100,
		UUID:    jsonTestUUID,
	}))

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithChallenges(challengeManager),
	)
	// L1-only cache (no Redis) keeps the test network-free.
	handler.feeds = &respCache{}

	fetch := func() []challenges.Challenge {
		req := newRequestWithUUID(http.MethodGet, "/json/challenges", jsonTestUUID)
		rec := httptest.NewRecorder()
		handler.JSONChallengesHandler(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var out []challenges.Challenge
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		return out
	}

	// First fetch misses and populates the cache.
	require.Len(t, fetch(), 1)

	// Deactivate the challenge directly in the DB, bypassing the handler/cache.
	require.NoError(t, challengeManager.DB.Model(&challenges.Challenge{}).
		Where("uuid = ?", jsonTestUUID).Update("active", false).Error)

	// Cached read still returns the stale challenge, proving L1 served it.
	cached := fetch()
	require.Len(t, cached, 1, "cached feed should not reflect a bypassing DB write")

	// After invalidation the next fetch reflects the DB state.
	handler.invalidateFeed("challenges", jsonTestUUID)
	require.Empty(t, fetch(), "invalidated feed should reflect the DB state")
}

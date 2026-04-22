package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/countries"
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

	teamManager, err := teams.CreateTeams(db, jsonTestUUID)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service", jsonTestUUID)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{}),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSettings(settingsManager),
	)

	return handler, teamManager, userManager, settingsManager
}

func newJSONCountryDataHandler(t *testing.T) (*HandlersMap, *countries.CountriesManager, *challenges.ChallengeManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countryManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{}),
		WithCountries(countryManager),
		WithChallenges(challengeManager),
	)

	return handler, countryManager, challengeManager
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
	require.Equal(t, visibleLastScore.UTC(), parsed[0].LastScore.UTC())
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

	require.NoError(t, settingsManager.SetGameboardShowTeamMembers(true, jsonSettingsAuthor))

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
	require.NoError(t, settingsManager.SetGameboardShowTeamMembers(true, jsonSettingsAuthor))

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
		CategoryID:  7,
		Country:     "ES",
		Active:      true,
		Points:      250,
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
	require.Equal(t, "Web", spain.Category)
	require.Equal(t, "Live intro", spain.Intro)
	require.Equal(t, "Live hint", spain.Hint)
	require.Equal(t, "", spain.Owner)
	require.Empty(t, spain.Completed)

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
	require.Empty(t, italy.Hint)
}

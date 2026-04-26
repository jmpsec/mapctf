package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newScorePostHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *teams.TeamManager, *users.UserManager, *challenges.ChallengeManager, *settings.SettingsManager, *logs.LogManager) {
	t.Helper()

	db := newJSONTestDB(t)

	teamManager, err := teams.CreateTeams(db, jsonTestUUID)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	challengeManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	settingsManager, err := settings.CreateSettingsManager(db, "test-service", jsonTestUUID)
	require.NoError(t, err)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID: jsonTestUUID,
			},
		}),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithChallenges(challengeManager),
		WithSettings(settingsManager),
		WithLogs(logManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, teamManager, userManager, challengeManager, settingsManager, logManager
}

func newJSONBodyRequestWithUUID(method, target, uuid string, body any) *http.Request {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, target, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	if uuid != "" {
		routeCtx.URLParams.Add("uuid", uuid)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestScorePOSTHandlerCreatesScoreAndLogs(t *testing.T) {
	handler, sessions, teamManager, userManager, challengeManager, settingsManager, logManager := newScorePostHandler(t)

	require.NoError(t, settingsManager.SetScoringEnabled(true, jsonSettingsAuthor))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 5},
		Name:    "Blue Team",
		Points:  100,
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "alice",
		TeamID:   5,
		Active:   true,
		UUID:     jsonTestUUID,
	}))
	require.NoError(t, challengeManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 7},
		Name:  "Web",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:      gorm.Model{ID: 20},
		Title:      "Spanish Challenge",
		CategoryID: 7,
		Country:    "ES",
		Active:     true,
		Points:     75,
		Flag:       "MAP{correct}",
		UUID:       jsonTestUUID,
	}))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/gameboard/score", jsonTestUUID, MapScoreRequest{
		CountryCode: "ES",
		Flag:        "MAP{correct}",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ScorePOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp MapScoreResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "Challenge completed", resp.Message)
	require.Equal(t, "ES", resp.CountryCode)
	require.Equal(t, uint(20), resp.ChallengeID)
	require.Equal(t, 75, resp.PointsAwarded)
	require.Equal(t, 175, resp.TotalPoints)

	var updatedTeam teams.PlatformTeam
	require.NoError(t, teamManager.DB.Where("id = ? AND uuid = ?", 5, jsonTestUUID).First(&updatedTeam).Error)
	require.Equal(t, 175, updatedTeam.Points)
	require.False(t, updatedTeam.LastScore.IsZero())

	var teamScores []teams.TeamScore
	require.NoError(t, teamManager.DB.Where("uuid = ? AND team_id = ?", jsonTestUUID, 5).Find(&teamScores).Error)
	require.Len(t, teamScores, 1)
	require.Equal(t, uint(20), teamScores[0].ChallengeID)
	require.Equal(t, "alice", teamScores[0].ScoredBy)

	scoreboardLogs, err := logManager.AllScoreboardLogs(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, scoreboardLogs, 1)
	require.Equal(t, "Blue Team", scoreboardLogs[0].Team)
	require.Equal(t, 175, scoreboardLogs[0].Points)
	require.Equal(t, 1, scoreboardLogs[0].Iteration)

	activityLogs, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityLogs, 1)
	require.Equal(t, "Blue Team", activityLogs[0].Subject)
	require.Equal(t, "completed", activityLogs[0].Action)
	require.Equal(t, "Spanish Challenge", activityLogs[0].Message)
	require.Equal(t, uint(20), activityLogs[0].ChallengeID)
}

func TestScorePOSTHandlerRejectsDuplicateSolve(t *testing.T) {
	handler, sessions, teamManager, userManager, challengeManager, settingsManager, _ := newScorePostHandler(t)

	require.NoError(t, settingsManager.SetScoringEnabled(true, jsonSettingsAuthor))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:  gorm.Model{ID: 7},
		Name:   "Blue Team",
		Points: 200,
		UUID:   jsonTestUUID,
		Active: true,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "alice",
		TeamID:   7,
		Active:   true,
		UUID:     jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:   gorm.Model{ID: 44},
		Title:   "France",
		Country: "FR",
		Active:  true,
		Points:  50,
		Flag:    "MAP{fr}",
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, teamManager.CreateScore(teams.TeamScore{
		TeamID:      7,
		ChallengeID: 44,
		Points:      50,
		UUID:        jsonTestUUID,
		ScoredBy:    "alice",
	}))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/gameboard/score", jsonTestUUID, MapScoreRequest{
		CountryCode: "FR",
		Flag:        "MAP{fr}",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "alice")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ScorePOSTHandler(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)

	var resp MapScoreResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "Your team already completed this challenge", resp.Message)
	require.Equal(t, 200, resp.TotalPoints)
}

func TestScorePOSTHandlerRecordsFailureForWrongFlag(t *testing.T) {
	handler, sessions, teamManager, userManager, challengeManager, settingsManager, logManager := newScorePostHandler(t)

	require.NoError(t, settingsManager.SetScoringEnabled(true, jsonSettingsAuthor))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:  gorm.Model{ID: 9},
		Name:   "Red Team",
		Points: 40,
		UUID:   jsonTestUUID,
		Active: true,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "bob",
		TeamID:   9,
		Active:   true,
		UUID:     jsonTestUUID,
	}))
	require.NoError(t, challengeManager.Create(challenges.Challenge{
		Model:   gorm.Model{ID: 55},
		Title:   "Italy",
		Country: "IT",
		Active:  true,
		Points:  30,
		Flag:    "MAP{it}",
		UUID:    jsonTestUUID,
	}))

	req := newJSONBodyRequestWithUUID(http.MethodPost, "/gameboard/score", jsonTestUUID, MapScoreRequest{
		CountryCode: "IT",
		Flag:        "wrong-flag",
	})
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "bob")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ScorePOSTHandler(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp MapScoreResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "Incorrect flag", resp.Message)
	require.Equal(t, 40, resp.TotalPoints)

	var teamScores []teams.TeamScore
	require.NoError(t, teamManager.DB.Where("uuid = ? AND team_id = ?", jsonTestUUID, 9).Find(&teamScores).Error)
	require.Empty(t, teamScores)

	failureLogs, err := logManager.AllFailuresLogs(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, failureLogs, 1)
	require.Equal(t, uint(55), failureLogs[0].ChallengeID)
	require.Equal(t, uint(9), failureLogs[0].TeamID)
	require.Equal(t, "wrong-flag", failureLogs[0].Flag)
}

func TestPerTeamThrottleBacklogLimitsWithinTeamOnly(t *testing.T) {
	handler, sessions, teamManager, userManager, _, _, _ := newScorePostHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:  gorm.Model{ID: 21},
		Name:   "Blue Team",
		UUID:   jsonTestUUID,
		Active: true,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:  gorm.Model{ID: 22},
		Name:   "Red Team",
		UUID:   jsonTestUUID,
		Active: true,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "alice",
		TeamID:   21,
		Active:   true,
		UUID:     jsonTestUUID,
	}))
	require.NoError(t, userManager.Create(users.PlatformUser{
		Username: "bob",
		TeamID:   22,
		Active:   true,
		UUID:     jsonTestUUID,
	}))

	firstStarted := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case firstStarted <- struct{}{}:
		default:
		}
		<-releaseFirst
		w.WriteHeader(http.StatusOK)
	})

	wrapped := handler.PerTeamThrottleBacklog(1, 0, 50*time.Millisecond)(next)

	reqAlice1 := newJSONBodyRequestWithUUID(http.MethodPost, "/gameboard/score", jsonTestUUID, nil)
	ctxAlice1, err := sessions.Load(reqAlice1.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctxAlice1, string(ContextKeyUser), "alice")
	reqAlice1 = reqAlice1.WithContext(ctxAlice1)

	rrAlice1 := httptest.NewRecorder()
	doneAlice1 := make(chan struct{})
	go func() {
		defer close(doneAlice1)
		wrapped.ServeHTTP(rrAlice1, reqAlice1)
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first alice request to start")
	}

	reqAlice2 := newJSONBodyRequestWithUUID(http.MethodPost, "/gameboard/score", jsonTestUUID, nil)
	ctxAlice2, err := sessions.Load(reqAlice2.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctxAlice2, string(ContextKeyUser), "alice")
	reqAlice2 = reqAlice2.WithContext(ctxAlice2)

	rrAlice2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rrAlice2, reqAlice2)
	require.Equal(t, http.StatusTooManyRequests, rrAlice2.Code)

	reqBob := newJSONBodyRequestWithUUID(http.MethodPost, "/gameboard/score", jsonTestUUID, nil)
	ctxBob, err := sessions.Load(reqBob.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctxBob, string(ContextKeyUser), "bob")
	reqBob = reqBob.WithContext(ctxBob)

	rrBob := httptest.NewRecorder()
	doneBob := make(chan struct{})
	go func() {
		defer close(doneBob)
		wrapped.ServeHTTP(rrBob, reqBob)
	}()

	time.Sleep(20 * time.Millisecond)
	close(releaseFirst)

	select {
	case <-doneAlice1:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first alice request to finish")
	}
	select {
	case <-doneBob:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bob request to finish")
	}

	require.Equal(t, http.StatusOK, rrAlice1.Code)
	require.Equal(t, http.StatusOK, rrBob.Code)
}

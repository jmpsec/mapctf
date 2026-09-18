package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDashboardHandler(t *testing.T) *HandlersMap {
	t.Helper()
	h, _, _, _ := newAdminTemplateHandler(t)
	var err error
	h.Settings, err = settings.CreateSettingsManager(h.Teams.DB, "test")
	require.NoError(t, err)
	require.NoError(t, h.Settings.Initialization(jsonTestUUID))
	h.Challenges, err = challenges.CreateChallengeManager(h.Teams.DB)
	require.NoError(t, err)
	h.Logs, err = logs.CreateLogManager(h.Teams.DB)
	require.NoError(t, err)
	return h
}

func renderDashboard(t *testing.T, h *HandlersMap) *httptest.ResponseRecorder {
	t.Helper()
	req := newAdminRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/admin", jsonTestUUID)
	ctx, err := h.Sessions.Load(req.Context(), "")
	require.NoError(t, err)
	h.Sessions.Put(ctx, string(ContextKeyUser), "admin")
	h.Sessions.Put(ctx, string(ContextKeyAdmin), true)
	rec := httptest.NewRecorder()
	h.RequireAdmin(http.HandlerFunc(h.AdminTemplateHandler)).ServeHTTP(rec, req.WithContext(ctx))
	return rec
}

func TestDashboardCompetitionState(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name            string
		started, paused bool
		start, end      time.Time
		state           string
	}{
		{name: "not started"},
		{name: "scheduled", started: true, start: now.Add(time.Hour)},
		{name: "ended", started: true, end: now.Add(-time.Minute), state: "Ended"},
		{name: "never started with past schedule", end: now.Add(-time.Minute)},
		{name: "running", started: true, start: now.Add(-time.Hour), end: now.Add(time.Hour), state: "Live"},
		{name: "paused", started: true, paused: true, state: "Paused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newDashboardHandler(t)
			require.NoError(t, h.Settings.SetGameStarted(tc.started, "admin", jsonTestUUID))
			require.NoError(t, h.Settings.SetGamePaused(tc.paused, "admin", jsonTestUUID))
			require.NoError(t, h.Settings.SetGameStartTime(tc.start, "admin", jsonTestUUID))
			require.NoError(t, h.Settings.SetGameEndTime(tc.end, "admin", jsonTestUUID))
			rec := renderDashboard(t, h)
			require.Equal(t, http.StatusOK, rec.Code)
			body := strings.Split(rec.Body.String(), "<script")[0]
			if tc.state == "" {
				require.Contains(t, body, "There is no ongoing competition.")
				require.NotContains(t, body, `class="dashboard-kpi-grid"`)
			} else {
				require.Contains(t, body, `data-metric="state">`+tc.state+`</strong>`)
				require.Contains(t, body, `data-metric="coverage">0%</strong>`)
				require.Contains(t, body, "No teams yet")
				require.Contains(t, body, "No activity yet")
			}
		})
	}
}

func TestDashboardUsesCompetitionDataAndEscapesContent(t *testing.T) {
	h := newDashboardHandler(t)
	now := time.Now().UTC()
	require.NoError(t, h.Settings.SetGameStarted(true, "admin", jsonTestUUID))
	require.NoError(t, h.Settings.SetGameStartTime(now.Add(-2*time.Hour), "admin", jsonTestUUID))
	for _, team := range []teams.PlatformTeam{
		{Model: gorm.Model{ID: 1}, Name: "Runner up", Points: 100, Active: true, Visible: true, UUID: jsonTestUUID},
		{Model: gorm.Model{ID: 2}, Name: "<img src=x onerror=alert(1)>", Points: 275, Active: true, Visible: true, UUID: jsonTestUUID},
		{Name: "Hidden team", Points: 999, Active: true, UUID: jsonTestUUID},
		{Name: "Inactive team", Points: 999, Visible: true, UUID: jsonTestUUID},
		{Name: "Other competition", Points: 999, Active: true, Visible: true, UUID: jsonOtherTestUUID},
	} {
		require.NoError(t, h.Teams.Create(team))
	}
	for _, challenge := range []challenges.Challenge{
		{Model: gorm.Model{ID: 1}, Title: "Solved", Active: true, Flag: "secret-flag", UUID: jsonTestUUID},
		{Model: gorm.Model{ID: 2}, Title: "Unsolved", Active: true, UUID: jsonTestUUID},
		{Model: gorm.Model{ID: 3}, Title: "Disabled", UUID: jsonTestUUID},
		{Title: "Other challenge", Active: true, UUID: jsonOtherTestUUID},
	} {
		require.NoError(t, h.Challenges.Create(challenge))
	}
	for _, score := range []teams.TeamScore{
		{Model: gorm.Model{CreatedAt: now.Add(-5 * time.Minute)}, TeamID: 1, ChallengeID: 1, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-30 * time.Minute)}, TeamID: 2, ChallengeID: 1, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-90 * time.Minute)}, TeamID: 2, ChallengeID: 3, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-3 * time.Hour)}, TeamID: 1, ChallengeID: 2, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-time.Minute)}, TeamID: 1, ChallengeID: 2, UUID: jsonOtherTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-time.Minute), DeletedAt: gorm.DeletedAt{Time: now, Valid: true}}, TeamID: 1, ChallengeID: 2, UUID: jsonTestUUID},
	} {
		require.NoError(t, h.Teams.CreateScore(score))
	}
	for _, activity := range []logs.ActivityLog{
		{Model: gorm.Model{CreatedAt: now.Add(-time.Minute)}, Visible: true, Message: "<script>alert(1)</script>", UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-time.Minute)}, Visible: false, Message: "Private activity", UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-time.Minute)}, Visible: true, Message: "Foreign activity", UUID: jsonOtherTestUUID},
		{Model: gorm.Model{CreatedAt: now.Add(-3 * time.Hour)}, Visible: true, Message: "Previous game", UUID: jsonTestUUID},
	} {
		require.NoError(t, h.Logs.CreateActivity(activity))
	}
	rec := renderDashboard(t, h)
	require.Equal(t, http.StatusOK, rec.Code)
	body := strings.Split(rec.Body.String(), "<script")[0]
	for _, value := range []string{
		`data-metric="teams">2</strong>`, `data-metric="captures-hour">2</strong>`,
		`data-metric="captures-total">3</strong>`, `data-metric="coverage">50%</strong>`,
		`data-metric="captures-10m">1</strong>`, `data-metric="active-challenges">2</strong>`,
		`data-metric="inactive-challenges">1</strong>`, `275`,
		`&lt;img src=x onerror=alert(1)&gt;`, `&lt;script&gt;alert(1)&lt;/script&gt;`,
	} {
		require.Contains(t, body, value)
	}
	require.Less(t, strings.Index(body, "&lt;img"), strings.Index(body, "Runner up"))
	for _, absent := range []string{"<img src=x", "<script>alert", "Private activity", "Foreign activity", "Previous game", "Other competition", "Hidden team", "Inactive team", "secret-flag", "p95 118ms", "Orion Labs", "Submission burst"} {
		require.NotContains(t, body, absent)
	}
}

func TestDashboardDatabaseFailureIsNotAnEmptyCompetition(t *testing.T) {
	h := newDashboardHandler(t)
	require.NoError(t, h.Settings.SetGameStarted(true, "admin", jsonTestUUID))
	require.NoError(t, h.Teams.DB.Migrator().DropTable(&teams.TeamScore{}))
	rec := renderDashboard(t, h)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.NotContains(t, rec.Body.String(), "There is no ongoing competition.")
}

func TestDashboardEndedSummaryStopsAtEndTime(t *testing.T) {
	h := newDashboardHandler(t)
	now := time.Now().UTC()
	start, end := now.Add(-4*time.Hour), now.Add(-2*time.Hour)
	require.NoError(t, h.Settings.SetGameStarted(true, "admin", jsonTestUUID))
	require.NoError(t, h.Settings.SetGameStartTime(start, "admin", jsonTestUUID))
	require.NoError(t, h.Settings.SetGameEndTime(end, "admin", jsonTestUUID))
	require.NoError(t, h.Settings.SetGamePaused(true, "admin", jsonTestUUID))
	require.NoError(t, h.Teams.Create(teams.PlatformTeam{Model: gorm.Model{ID: 1}, Name: "Winning team", Points: 9999, Active: true, Visible: true, UUID: jsonTestUUID}))
	for _, name := range []string{"Team B", "Team C", "Team D", "Team E", "Team F"} {
		require.NoError(t, h.Teams.Create(teams.PlatformTeam{Name: name, Active: true, Visible: true, UUID: jsonTestUUID}))
	}
	require.NoError(t, h.Challenges.Create(challenges.Challenge{Model: gorm.Model{ID: 1}, Active: true, UUID: jsonTestUUID}))
	require.NoError(t, h.Challenges.Create(challenges.Challenge{Model: gorm.Model{ID: 2}, Active: true, UUID: jsonTestUUID}))
	for _, score := range []teams.TeamScore{
		{Model: gorm.Model{CreatedAt: start.Add(-time.Minute)}, TeamID: 1, ChallengeID: 2, Points: 500, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: end}, TeamID: 1, ChallengeID: 1, Points: 100, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: end.Add(time.Minute)}, TeamID: 1, ChallengeID: 2, Points: 900, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: end}, TeamID: 1, ChallengeID: 2, Points: 500, UUID: jsonOtherTestUUID},
	} {
		require.NoError(t, h.Teams.CreateScore(score))
	}
	for _, hint := range []logs.HintsLog{
		{Model: gorm.Model{CreatedAt: start.Add(-time.Minute)}, TeamID: 1, Penalty: 5, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: end.Add(-time.Minute)}, TeamID: 1, Penalty: 20, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: end.Add(time.Minute)}, TeamID: 1, Penalty: 30, UUID: jsonTestUUID},
		{Model: gorm.Model{CreatedAt: end}, TeamID: 1, Penalty: 50, UUID: jsonOtherTestUUID},
	} {
		require.NoError(t, h.Logs.CreateHintsLog(hint))
	}
	require.NoError(t, h.Logs.CreateActivity(logs.ActivityLog{Model: gorm.Model{CreatedAt: end}, Visible: true, Message: "Last game capture", UUID: jsonTestUUID}))
	require.NoError(t, h.Logs.CreateActivity(logs.ActivityLog{Model: gorm.Model{CreatedAt: end.Add(time.Minute)}, Visible: true, Message: "After-game activity", UUID: jsonTestUUID}))

	data, err := h.loadDashboard(jsonTestUUID, now)
	require.NoError(t, err)
	require.False(t, data.Ongoing)
	require.Equal(t, int64(1), data.Captures)
	require.Equal(t, int64(1), data.Hour)
	require.Equal(t, int64(1), data.TenMins)
	require.Equal(t, 50, data.Coverage)
	require.Len(t, data.Leaders, 6)
	require.Equal(t, 80, data.Leaders[0].Points)
	require.Equal(t, "Winning team", data.Leaders[0].Name)
	require.Equal(t, 0, data.Leaders[5].Points)
	require.Len(t, data.Activity, 1)
	require.Equal(t, "Last game capture", data.Activity[0].Message)
	later, err := h.loadDashboard(jsonTestUUID, now.Add(24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, data.Captures, later.Captures)
	require.Equal(t, data.Hour, later.Hour)
	require.Equal(t, data.Leaders, later.Leaders)
	atEnd, err := h.loadDashboard(jsonTestUUID, end)
	require.NoError(t, err)
	require.True(t, atEnd.Ended)
	require.False(t, atEnd.Ongoing)
	require.Equal(t, data.Leaders, atEnd.Leaders)

	rec := renderDashboard(t, h)
	require.Equal(t, http.StatusOK, rec.Code)
	body := strings.Split(rec.Body.String(), "<script")[0]
	for _, value := range []string{"Competition summary", "Competition standings", "Winning team", "Team F", "80 Points", "Last game capture", `data-metric="state">Ended</strong>`} {
		require.Contains(t, body, value)
	}
	for _, absent := range []string{"There is no ongoing competition.", "Live game metrics", "After-game activity", "Captures in the last hour", "Last 10m", "9999", ">Paused</strong>"} {
		require.NotContains(t, body, absent)
	}
}

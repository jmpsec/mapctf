package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/teams"
)

type dashboardData struct {
	Ongoing, Paused                           bool
	Start, End, Updated                       time.Time
	TeamCount, Coverage                       int
	ActiveChallenges, InactiveChallenges      int64
	SolvedChallenges, Captures, Hour, TenMins int64
	Leaders                                   []dashboardLeader
	Activity                                  []logs.ActivityLog
}

type dashboardLeader struct {
	Rank, Points int
	Name         string
}

func (h *HandlersMap) loadDashboard(uuid string, now time.Time) (dashboardData, error) {
	d := dashboardData{Updated: now}
	if h.Settings == nil {
		return d, fmt.Errorf("dashboard settings unavailable")
	}
	started, err := h.Settings.GetGameStarted(uuid)
	if err != nil || !started {
		return d, err
	}
	if d.Start, err = h.Settings.GetGameStartTime(uuid); err != nil {
		return d, err
	}
	if d.End, err = h.Settings.GetGameEndTime(uuid); err != nil {
		return d, err
	}
	if d.Start.After(now) || (!d.End.IsZero() && !d.End.After(now)) {
		return d, nil
	}
	d.Ongoing = true
	if d.Paused, err = h.Settings.GetGamePaused(uuid); err != nil {
		return d, err
	}
	if h.Teams == nil || h.Challenges == nil || h.Logs == nil {
		return d, fmt.Errorf("dashboard data unavailable")
	}
	allTeams, err := h.Teams.GetAll(uuid)
	if err != nil {
		return d, err
	}
	for _, team := range allTeams {
		if team.Active && team.Visible {
			d.Leaders = append(d.Leaders, dashboardLeader{Name: team.Name, Points: team.Points})
		}
	}
	d.TeamCount = len(d.Leaders)
	sort.SliceStable(d.Leaders, func(i, j int) bool {
		if d.Leaders[i].Points != d.Leaders[j].Points {
			return d.Leaders[i].Points > d.Leaders[j].Points
		}
		return strings.ToLower(d.Leaders[i].Name) < strings.ToLower(d.Leaders[j].Name)
	})
	if len(d.Leaders) > 5 {
		d.Leaders = d.Leaders[:5]
	}
	for i := range d.Leaders {
		d.Leaders[i].Rank = i + 1
	}
	challengeQuery := h.Challenges.DB.Model(&challenges.Challenge{}).Where("uuid = ?", uuid)
	if err := challengeQuery.Where("active = ?", true).Count(&d.ActiveChallenges).Error; err != nil {
		return d, err
	}
	if err := h.Challenges.DB.Model(&challenges.Challenge{}).Where("uuid = ? AND active = ?", uuid, false).Count(&d.InactiveChallenges).Error; err != nil {
		return d, err
	}
	// Score rows record successful captures, including zero-point solves. Hint
	// penalties change team totals separately and must not count as captures.
	for _, metric := range []struct {
		since time.Time
		count *int64
	}{
		{d.Start, &d.Captures},
		{now.Add(-time.Hour), &d.Hour},
		{now.Add(-10 * time.Minute), &d.TenMins},
	} {
		since := metric.since
		if since.Before(d.Start) {
			since = d.Start
		}
		if err := h.Teams.DB.Model(&teams.TeamScore{}).
			Where("uuid = ? AND created_at >= ? AND created_at <= ?", uuid, since, now).
			Count(metric.count).Error; err != nil {
			return d, err
		}
	}
	activeIDs := h.Challenges.DB.Model(&challenges.Challenge{}).Select("id").Where("uuid = ? AND active = ?", uuid, true)
	if err := h.Teams.DB.Model(&teams.TeamScore{}).
		Where("uuid = ? AND created_at >= ? AND created_at <= ? AND challenge_id IN (?)", uuid, d.Start, now, activeIDs).
		Distinct("challenge_id").Count(&d.SolvedChallenges).Error; err != nil {
		return d, err
	}
	if d.ActiveChallenges > 0 {
		d.Coverage = int(100 * d.SolvedChallenges / d.ActiveChallenges)
	}
	if err := h.Logs.DB.Where("uuid = ? AND visible = ? AND created_at >= ? AND created_at <= ?", uuid, true, d.Start, now).
		Order("created_at DESC, id DESC").Limit(8).Find(&d.Activity).Error; err != nil {
		return d, err
	}
	return d, nil
}

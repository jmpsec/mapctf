package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGameboardClockUsesResponsiveLayout(t *testing.T) {
	gameboard, err := os.ReadFile(filepath.Join("..", "templates", "gameboard.html"))
	require.NoError(t, err)
	css, err := os.ReadFile(filepath.Join("..", "templates", "static", "css", "mapctf.css"))
	require.NoError(t, err)

	gameboardBody := string(gameboard)
	require.Contains(t, gameboardBody, `<span class="clock-days clock-value is-empty">--</span>`)
	require.Contains(t, gameboardBody, `<span class="clock-milliseconds clock-value three-digit is-empty">---</span>`)
	require.Equal(t, 4, strings.Count(gameboardBody, `class="clock-separator"`))
	require.Contains(t, gameboardBody, `<div class="game-clock-context mctf-numbers">`)
	require.Contains(t, gameboardBody, `data-game-paused="{{ if .GamePaused }}true{{ else }}false{{ end }}"`)
	require.Contains(t, gameboardBody, `<span data-game-clock-start>--- -- --:-- UTC--:--</span>`)
	require.Contains(t, gameboardBody, `<span data-game-clock-remaining>--</span>`)
	require.Contains(t, gameboardBody, `<span data-game-clock-end>--- -- --:-- UTC--:--</span>`)

	cssBody := string(css)
	require.Contains(t, cssBody, `font-size: 3em;`)
	require.Contains(t, cssBody, `flex-wrap: wrap;`)
	require.Contains(t, cssBody, `.mctf-gameboard aside[data-module][data-module=game-clock] .game-clock .clock-value {`)
	require.Contains(t, cssBody, `flex: 0 1 2ch;`)
	require.Contains(t, cssBody, `.mctf-gameboard aside[data-module][data-module=game-clock] .game-clock .clock-milliseconds {`)
	require.Contains(t, cssBody, `flex-basis: 3ch;`)
	require.Contains(t, cssBody, `.mctf-gameboard aside[data-module][data-module=game-clock] .game-clock-context {`)
	require.Contains(t, cssBody, `justify-content: space-between;`)

	require.Contains(t, gameboardBody, `function formatClockDatestamp(date)`)
	require.Contains(t, gameboardBody, `var monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];`)
	require.Contains(t, gameboardBody, `function formatClockTimezone(date)`)
	require.Contains(t, gameboardBody, `function formatRemainingSummary(days, hours, minutes, seconds, remainingMs, durationMs)`)
	require.Contains(t, gameboardBody, `data-game-clock-remaining`)
	require.Contains(t, gameboardBody, `"/json/game-clock"`)
	require.Contains(t, gameboardBody, `GAME_CLOCK_POLL_INTERVAL_MS`)
	require.Contains(t, gameboardBody, `server_time`)
	require.Contains(t, gameboardBody, `game_paused`)
	require.Contains(t, gameboardBody, `remaining_ms`)
	require.Contains(t, gameboardBody, `duration_ms`)
	require.Contains(t, gameboardBody, `function renderStoppedGameClock()`)
	require.Contains(t, gameboardBody, `renderStoppedGameClock();`)
	require.Contains(t, gameboardBody, `cell.classList.remove("active", "current-spot");`)
	require.Contains(t, gameboardBody, `timer.textContent = "--:--:--:--";`)
	require.Contains(t, gameboardBody, `startContextEl.textContent = "--";`)
	require.Contains(t, gameboardBody, `if (gamePaused)`)
	require.Contains(t, gameboardBody, `formatRemainingPercent(remainingMs, durationMs)`)
	require.Contains(t, gameboardBody, `"% left"`)
	require.Contains(t, gameboardBody, `endContextEl.textContent = "--";`)
	require.NotContains(t, gameboardBody, `formatRemainingTime(days, hours, minutes, seconds, milliseconds)`)

	nowrapClock := regexp.MustCompile(`(?s)\.mctf-gameboard aside\[data-module\]\[data-module=game-clock\] \.game-clock \{[^}]*white-space:\s*nowrap;`)
	require.NotRegexp(t, nowrapClock, cssBody)
	require.NotContains(t, gameboardBody, `setGameClockCountdownVisible`)
	require.NotContains(t, cssBody, `.mctf-gameboard aside[data-module][data-module=game-clock] .game-clock.is-hidden`)
	require.NotContains(t, cssBody, `.mctf-gameboard aside[data-module][data-module=game-clock] .game-progress.is-hidden`)
}

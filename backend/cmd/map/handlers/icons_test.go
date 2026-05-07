package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type seedLogo struct {
	Logo string `json:"logo"`
}

var directSpriteIcons = []string{
	"close",
	"globe",
	"team-indicator",
}

func loadSeedBadgeSlugs(t *testing.T) []string {
	t.Helper()

	seedPath := filepath.Join("..", "..", "..", "pkg", "teams", "team-logos-seed.json")
	data, err := os.ReadFile(seedPath)
	require.NoError(t, err)

	var logos []seedLogo
	require.NoError(t, json.Unmarshal(data, &logos))

	slugs := make([]string, 0, len(logos))
	for _, logo := range logos {
		require.True(t, strings.HasPrefix(logo.Logo, "/static/svg/icons/badges/badge-"), "unexpected seed logo path: %s", logo.Logo)
		require.True(t, strings.HasSuffix(logo.Logo, ".svg"), "unexpected seed logo path: %s", logo.Logo)
		slug := strings.TrimPrefix(filepath.Base(logo.Logo), "badge-")
		slug = strings.TrimSuffix(slug, ".svg")
		slugs = append(slugs, slug)
	}
	return slugs
}

func TestSVGIconSourcesOnlyKeepUsedDirectIcons(t *testing.T) {
	iconsDir := filepath.Join("..", "templates", "static", "svg", "icons")
	entries, err := os.ReadDir(iconsDir)
	require.NoError(t, err)

	var actual []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".svg" || entry.Name() == "icons.svg" {
			continue
		}
		actual = append(actual, strings.TrimSuffix(entry.Name(), ".svg"))
	}
	sort.Strings(actual)

	expected := append([]string(nil), directSpriteIcons...)
	sort.Strings(expected)
	require.Equal(t, expected, actual)
}

func TestSVGIconBadgeSourcesMatchSeedLogos(t *testing.T) {
	badgesDir := filepath.Join("..", "templates", "static", "svg", "icons", "badges")
	entries, err := os.ReadDir(badgesDir)
	require.NoError(t, err)

	var actual []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".svg" {
			continue
		}
		actual = append(actual, strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "badge-"), ".svg"))
	}
	sort.Strings(actual)

	expected := loadSeedBadgeSlugs(t)
	sort.Strings(expected)
	require.Equal(t, expected, actual)
}

func TestSVGIconSpriteOnlyContainsUsedSymbols(t *testing.T) {
	spritePath := filepath.Join("..", "templates", "static", "svg", "icons", "icons.svg")
	data, err := os.ReadFile(spritePath)
	require.NoError(t, err)

	symbolIDPattern := regexp.MustCompile(`id="(icon--[^"]+)"`)
	matches := symbolIDPattern.FindAllStringSubmatch(string(data), -1)
	require.NotEmpty(t, matches)

	actual := make([]string, 0, len(matches))
	for _, match := range matches {
		actual = append(actual, match[1])
	}
	sort.Strings(actual)

	expected := make([]string, 0, len(directSpriteIcons)+len(loadSeedBadgeSlugs(t)))
	for _, icon := range directSpriteIcons {
		expected = append(expected, "icon--"+icon)
	}
	for _, slug := range loadSeedBadgeSlugs(t) {
		expected = append(expected, "icon--badge-"+slug)
	}
	sort.Strings(expected)

	require.Equal(t, expected, actual)
}

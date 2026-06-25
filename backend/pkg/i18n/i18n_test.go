package i18n

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func supportedStrings(c *Catalog) []string {
	out := make([]string, 0, len(c.Supported()))
	for _, t := range c.Supported() {
		out = append(out, t.String())
	}
	return out
}

func TestNewLoadsEmbeddedLocales(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	require.Contains(t, supportedStrings(c), language.English.String())
	require.Contains(t, supportedStrings(c), language.Spanish.String())
}

func TestResolveFallsBackToEnglish(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	require.Equal(t, "en", c.Resolve("xx").String())
	// Regional variant resolves to the base language when available.
	require.Equal(t, "es", c.Resolve("es-AR").String())
}

func TestTranslateFallsBackToEnglishThenKey(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	es := c.Resolve("es")
	require.Equal(t, "Usuario", c.Translate(es, "login.username_label"))
	// Unsupported locale falls back to English.
	fr := c.Resolve("fr")
	require.Equal(t, "Play CTF", c.Translate(fr, "nav.play_ctf"))
	// Unknown key returns the key itself.
	require.Equal(t, "no.such.key", c.Translate(es, "no.such.key"))
	// Regional variant resolves to the base catalog tag so lookup succeeds.
	require.Equal(t, "Usuario", c.Translate(c.Resolve("es-AR"), "login.username_label"))
}

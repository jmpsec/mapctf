// Package i18n provides locale-aware message translation for MapCTF.
//
// Catalogs are embedded JSON files keyed by BCP-47 language tag (e.g. "en",
// "es"). English is the required baseline and serves as the fallback when a
// message is missing for the resolved locale. Language matching and fallback
// chains are handled by golang.org/x/text/language so regional variants such
// as "es-AR" resolve to "es" before falling back to "en".
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localesFS embed.FS

// Catalog holds translated messages keyed by language tag.
type Catalog struct {
	messages  map[language.Tag]map[string]string
	supported []language.Tag
	matcher   language.Matcher
}

// New loads the embedded locale catalogs. English is always required.
func New() (*Catalog, error) {
	c := &Catalog{messages: map[language.Tag]map[string]string{}}
	entries, err := localesFS.ReadDir("locales")
	if err != nil {
		return nil, fmt.Errorf("read embedded locales: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		tag, err := language.Parse(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			return nil, fmt.Errorf("parse locale tag %q: %w", e.Name(), err)
		}
		data, err := localesFS.ReadFile("locales/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read locale %q: %w", e.Name(), err)
		}
		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			return nil, fmt.Errorf("parse locale %q: %w", e.Name(), err)
		}
		c.messages[tag] = msgs
		if tag == language.English {
			continue
		}
		c.supported = append(c.supported, tag)
	}
	if _, ok := c.messages[language.English]; !ok {
		return nil, fmt.Errorf("english locale is required but missing")
	}
	// English is always first so it is the matcher's default fallback.
	c.supported = append([]language.Tag{language.English}, c.supported...)
	c.matcher = language.NewMatcher(c.supported)
	return c, nil
}

// Supported returns the configured language tags.
func (c *Catalog) Supported() []language.Tag { return c.supported }

// Resolve picks the best supported tag for the requested language strings,
// ignoring empty values. English is returned when nothing matches.
func (c *Catalog) Resolve(preferred ...string) language.Tag {
	tags := make([]language.Tag, 0, len(preferred))
	for _, p := range preferred {
		if p == "" {
			continue
		}
		if t, err := language.Parse(p); err == nil {
			tags = append(tags, t)
		}
	}
	if len(tags) == 0 {
		return c.supported[0]
	}
	// Match returns a tag possibly carrying region/variant extensions; the
	// second return is the index into the supported list, which gives the
	// canonical catalog tag to use for message lookup.
	_, idx, _ := c.matcher.Match(tags...)
	return c.supported[idx]
}

// Translate returns the message for key in tag, falling back to English and
// finally to the key itself. Format verbs are expanded via fmt.Sprintf when
// args are provided.
func (c *Catalog) Translate(tag language.Tag, key string, args ...any) string {
	if msgs, ok := c.messages[tag]; ok {
		if s, ok := msgs[key]; ok {
			return format(s, args)
		}
	}
	if msgs, ok := c.messages[language.English]; ok {
		if s, ok := msgs[key]; ok {
			return format(s, args)
		}
	}
	return key
}

// Messages returns the message map for a tag, falling back to English. It is
// intended for serializing the active locale into client-side JavaScript.
func (c *Catalog) Messages(tag language.Tag) map[string]string {
	if msgs, ok := c.messages[tag]; ok {
		return msgs
	}
	return c.messages[language.English]
}

func format(s string, args []any) string {
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

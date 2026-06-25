package handlers

import (
	"context"
	"net/http"
	"sync"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/i18n"
	"golang.org/x/text/language"
)

const (
	// ContextKeyLocale stores the resolved language.Tag in the request context.
	ContextKeyLocale string = "locale"
	// ContextKeyT stores the per-request translation func in the request context.
	ContextKeyT string = "t"
)

var (
	defaultCatalogOnce sync.Once
	defaultCatalog     *i18n.Catalog
)

// defaultI18N returns a process-wide catalog built from the embedded locale
// files. It lets handlers render English even when no explicit catalog was
// injected (e.g. in tests), so pages never degrade to raw message keys.
func defaultI18N() *i18n.Catalog {
	defaultCatalogOnce.Do(func() {
		c, err := i18n.New()
		if err != nil {
			// The English catalog is embedded and required, so this is fatal.
			panic("i18n: failed to load default catalog: " + err.Error())
		}
		defaultCatalog = c
	})
	return defaultCatalog
}

// catalog returns the effective i18n catalog, falling back to the embedded
// default when none was explicitly injected.
func (h *HandlersMap) catalog() *i18n.Catalog {
	if h.I18N != nil {
		return h.I18N
	}
	return defaultI18N()
}

// LocaleMiddleware resolves the active language for UUID-scoped requests and
// stores the resolved language tag plus a translation func in the request
// context. Resolution order is: the per-game `language` setting (when the
// request UUID matches the configured service UUID), then Accept-Language,
// then English.
func (h *HandlersMap) LocaleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := h.catalog()
		var preferred []string
		if uuid := chi.URLParam(r, "uuid"); uuid != "" && uuid == h.Config.Map.UUID && h.Settings != nil {
			if lang, err := h.Settings.GetLanguage(uuid); err == nil && lang != "" {
				preferred = append(preferred, lang)
			}
		}
		preferred = append(preferred, r.Header.Get("Accept-Language"))
		tag := c.Resolve(preferred...)
		ctx := context.WithValue(r.Context(), ContextKeyLocale, tag)
		ctx = context.WithValue(ctx, ContextKeyT, func(key string, args ...any) string {
			return c.Translate(tag, key, args...)
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Locale returns the resolved language tag from the context, defaulting to
// English when unset (e.g. when a handler is invoked without the middleware).
func (h *HandlersMap) Locale(ctx context.Context) language.Tag {
	if v, ok := ctx.Value(ContextKeyLocale).(language.Tag); ok {
		return v
	}
	return language.English
}

// T returns the translation func bound to the resolved locale. When the
// middleware did not run it translates against the effective catalog with
// English, so direct handler calls still render readable text.
func (h *HandlersMap) T(ctx context.Context) func(string, ...any) string {
	if fn, ok := ctx.Value(ContextKeyT).(func(string, ...any) string); ok {
		return fn
	}
	c := h.catalog()
	return func(key string, args ...any) string {
		return c.Translate(language.English, key, args...)
	}
}

// LocaleMessages returns the message map for the resolved locale, for
// serializing into client-side JavaScript.
func (h *HandlersMap) LocaleMessages(ctx context.Context) map[string]string {
	return h.catalog().Messages(h.Locale(ctx))
}

// i18nFuncMap returns a template.FuncMap exposing the per-request translation
// function as "T".
func (h *HandlersMap) i18nFuncMap(r *http.Request) template.FuncMap {
	return template.FuncMap{
		"T": h.T(r.Context()),
	}
}

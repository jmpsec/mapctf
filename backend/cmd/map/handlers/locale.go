package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"golang.org/x/text/language"
)

const (
	// ContextKeyLocale stores the resolved language.Tag in the request context.
	ContextKeyLocale string = "locale"
	// ContextKeyT stores the per-request translation func in the request context.
	ContextKeyT string = "t"
)

// LocaleMiddleware resolves the active language for UUID-scoped requests and
// stores the resolved language tag plus a translation func in the request
// context. Resolution order is: the per-game `language` setting (when the
// request UUID matches the configured service UUID), then Accept-Language,
// then English.
func (h *HandlersMap) LocaleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if h.I18N != nil {
			var preferred []string
			if uuid := chi.URLParam(r, "uuid"); uuid != "" && uuid == h.Config.Map.UUID && h.Settings != nil {
				if lang, err := h.Settings.GetLanguage(uuid); err == nil && lang != "" {
					preferred = append(preferred, lang)
				}
			}
			preferred = append(preferred, r.Header.Get("Accept-Language"))
			tag := h.I18N.Resolve(preferred...)
			ctx = context.WithValue(ctx, ContextKeyLocale, tag)
			ctx = context.WithValue(ctx, ContextKeyT, func(key string, args ...any) string {
				return h.I18N.Translate(tag, key, args...)
			})
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Locale returns the resolved language tag from the context, defaulting to
// English when unset (e.g. for non-UUID routes or when i18n is disabled).
func (h *HandlersMap) Locale(ctx context.Context) language.Tag {
	if v, ok := ctx.Value(ContextKeyLocale).(language.Tag); ok {
		return v
	}
	return language.English
}

// T returns the translation func bound to the resolved locale. When i18n is
// not configured it returns a passthrough that echoes the key.
func (h *HandlersMap) T(ctx context.Context) func(string, ...any) string {
	if fn, ok := ctx.Value(ContextKeyT).(func(string, ...any) string); ok {
		return fn
	}
	if h.I18N != nil {
		return func(key string, args ...any) string {
			return h.I18N.Translate(language.English, key, args...)
		}
	}
	return func(key string, _ ...any) string { return key }
}

// LocaleMessages returns the message map for the resolved locale, for
// serializing into client-side JavaScript. Returns an empty map when i18n
// is disabled.
func (h *HandlersMap) LocaleMessages(ctx context.Context) map[string]string {
	if h.I18N == nil {
		return map[string]string{}
	}
	return h.I18N.Messages(h.Locale(ctx))
}

package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/i18n"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

// TestLocaleMiddlewareResolvesViaRealRouting guards against the regression
// where a root-level chi middleware cannot see the {uuid} URL param. The
// middleware must be mounted on the /{uuid} route group so chi has matched
// (and populated) the param before it runs.
func TestLocaleMiddlewareResolvesViaRealRouting(t *testing.T) {
	db := newJSONTestDB(t)
	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	require.NoError(t, settingsManager.Initialization(jsonTestUUID))
	require.NoError(t, settingsManager.SetLanguage("es", jsonSettingsAuthor, jsonTestUUID))

	catalog, err := i18n.New()
	require.NoError(t, err)
	sessions := scs.New()
	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{
			UUID:         jsonTestUUID,
			TemplatesDir: filepath.Join("..", "templates"),
		}}),
		WithSettings(settingsManager),
		WithSessions(sessions),
		WithI18N(catalog),
	)

	var resolved language.Tag
	rec := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resolved = handler.Locale(r.Context())
			next.ServeHTTP(w, r)
		})
	}

	mux := chi.NewRouter()
	mux.Route("/{uuid}", func(r chi.Router) {
		r.Use(handler.LocaleMiddleware)
		r.Use(rec)
		r.Get("/gameboard", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/"+jsonTestUUID+"/gameboard", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "es", resolved.String(), "middleware must resolve the per-game language setting through real routing")
}

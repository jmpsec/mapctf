package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/i18n"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
)

func newLoginI18nHandler(t *testing.T, lang string) (*HandlersMap, *scs.SessionManager) {
	t.Helper()

	db := newJSONTestDB(t)
	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	require.NoError(t, settingsManager.Initialization(jsonTestUUID))
	if lang != "" {
		require.NoError(t, settingsManager.SetLanguage(lang, jsonSettingsAuthor, jsonTestUUID))
	}

	catalog, err := i18n.New()
	require.NoError(t, err)

	sessions := scs.New()
	return CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithSettings(settingsManager),
		WithSessions(sessions),
		WithI18N(catalog),
	), sessions
}

func TestLoginHandlerRendersConfiguredLocale(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "login.html")); err != nil {
		t.Skipf("login template not available: %v", err)
	}

	cases := []struct {
		name string
		lang string
		want []string
	}{
		{
			name: "spanish",
			lang: "es",
			want: []string{`<html lang="es">`, "Jugar CTF", "Nombre de usuario", "Contraseña", "Iniciar sesión"},
		},
		{
			name: "english default",
			lang: "en",
			want: []string{`<html lang="en">`, "Play CTF", "Username", "Password", "Login"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, sessions := newLoginI18nHandler(t, tc.lang)

			req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/login", jsonTestUUID)
			ctx, err := sessions.Load(req.Context(), "")
			require.NoError(t, err)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.LocaleMiddleware(http.HandlerFunc(handler.LoginHandler)).ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
			body := rr.Body.String()
			for _, want := range tc.want {
				require.Contains(t, body, want)
			}
			// The client-side bundle must be injected and carry the locale messages.
			require.Contains(t, body, "window.MCTF_I18N")
			require.Contains(t, body, `"nav.login"`)
		})
	}
}

func TestLoginHandlerFallsBackToEnglishForUnknownLanguage(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "login.html")); err != nil {
		t.Skipf("login template not available: %v", err)
	}

	handler, sessions := newLoginI18nHandler(t, "xx") // unsupported code
	req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/login", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.LoginHandler)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "Play CTF")
}

func TestRegistrationHandlerRendersConfiguredLocale(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "registration.html")); err != nil {
		t.Skipf("registration template not available: %v", err)
	}

	handler, sessions := newLoginI18nHandler(t, "es")

	req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/registration", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.RegistrationTemplateHandler)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	body := rr.Body.String()
	for _, want := range []string{`<html lang="es">`, "Jugar CTF", "Nombre completo", "Nombre de equipo", "Regístrate"} {
		require.Contains(t, body, want)
	}
}

func TestGameboardHandlerRendersConfiguredLocale(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "gameboard.html")); err != nil {
		t.Skipf("gameboard template not available: %v", err)
	}

	handler, sessions, settingsManager, _, _ := newGameboardTemplateHandler(t)
	require.NoError(t, settingsManager.SetLanguage("es", jsonSettingsAuthor, jsonTestUUID))

	req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/gameboard", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.GameboardTemplateHandler)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	body := rr.Body.String()
	for _, want := range []string{
		`<html lang="es">`,
		"Ranking",
		"Actividad",
		"Sin equipo",
		"Posición",
		"Puntos",
		"Reloj de juego",
	} {
		require.Contains(t, body, want)
	}
	require.Contains(t, body, `window.MCTF_LANG = "es"`)
}

func TestAdminUsersHandlerRendersConfiguredLocale(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "admin", "users.html")); err != nil {
		t.Skipf("admin users template not available: %v", err)
	}

	db := newJSONTestDB(t)
	settingsManager, err := settings.CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	require.NoError(t, settingsManager.Initialization(jsonTestUUID))
	require.NoError(t, settingsManager.SetLanguage("es", jsonSettingsAuthor, jsonTestUUID))
	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)
	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{Secret: "test-secret", HoursToExpire: 24})
	require.NoError(t, err)
	catalog, err := i18n.New()
	require.NoError(t, err)
	sessions := scs.New()
	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID, TemplatesDir: filepath.Join("..", "templates")}}),
		WithSettings(settingsManager),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSessions(sessions),
		WithI18N(catalog),
	)

	req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/admin/users", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.AdminUsersTemplateHandler)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	body := rr.Body.String()
	for _, want := range []string{`<html lang="es">`, "Administración del juego", "Administrar usuarios", "Añadir usuario", `window.MCTF_LANG = "es"`} { //nolint:misspell // "Administrar" is a valid Spanish word
		require.Contains(t, body, want)
	}
}

func TestLoginHandlerRendersFrenchLocale(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "login.html")); err != nil {
		t.Skipf("login template not available: %v", err)
	}
	handler, sessions := newLoginI18nHandler(t, "fr")

	req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/login", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.LoginHandler)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	body := rr.Body.String()
	for _, want := range []string{`<html lang="fr">`, "Jouer au CTF", "Nom d'utilisateur", "Mot de passe", "Connexion", `window.MCTF_LANG = "fr"`} {
		require.Contains(t, body, want)
	}
}

func TestLoginHandlerRendersPortugueseLocale(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "templates", "login.html")); err != nil {
		t.Skipf("login template not available: %v", err)
	}
	handler, sessions := newLoginI18nHandler(t, "pt")

	req := newTemplateRequestWithUUID(http.MethodGet, "/"+jsonTestUUID+"/login", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.LocaleMiddleware(http.HandlerFunc(handler.LoginHandler)).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	body := rr.Body.String()
	for _, want := range []string{`<html lang="pt">`, "Jogar CTF", "Nome de usuário", "Senha", "Entrar", `window.MCTF_LANG = "pt"`} {
		require.Contains(t, body, want)
	}
}

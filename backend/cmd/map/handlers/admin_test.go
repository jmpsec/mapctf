package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAdminTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *chat.ChatManager, *teams.TeamManager) {
	t.Helper()

	db := newJSONTestDB(t)

	chatManager, err := chat.CreateChatManager(db, jsonTestUUID)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	userManager, err := users.CreateUserManager(db, &config.ConfigurationJWT{
		Secret:        "test-secret",
		HoursToExpire: 24,
	})
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithChat(chatManager),
		WithTeams(teamManager),
		WithUsers(userManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, chatManager, teamManager
}

func newAdminCountryActionHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *countries.CountriesManager, *challenges.ChallengeManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, countriesManager, challengesManager
}

func newAdminChallengeActivityHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *countries.CountriesManager, *challenges.ChallengeManager, *logs.LogManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithLogs(logManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, countriesManager, challengesManager, logManager
}

func newAdminActivityTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *logs.LogManager) {
	t.Helper()

	db := newJSONTestDB(t)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithLogs(logManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, logManager
}

func newAdminChallengesTemplateHandler(t *testing.T) (*HandlersMap, *scs.SessionManager, *countries.CountriesManager, *challenges.ChallengeManager, *logs.LogManager, *teams.TeamManager) {
	t.Helper()

	db := newJSONTestDB(t)

	countriesManager, err := countries.CreateCountries(db, jsonTestUUID)
	require.NoError(t, err)

	challengesManager, err := challenges.CreateChallengeManager(db)
	require.NoError(t, err)

	logManager, err := logs.CreateLogManager(db)
	require.NoError(t, err)

	teamManager, err := teams.CreateTeams(db)
	require.NoError(t, err)

	sessionManager := scs.New()

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithCountries(countriesManager),
		WithChallenges(challengesManager),
		WithLogs(logManager),
		WithTeams(teamManager),
		WithSessions(sessionManager),
	)

	return handler, sessionManager, countriesManager, challengesManager, logManager, teamManager
}

func newAdminRequestWithUUID(method, target, uuid string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", uuid)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func newAdminMultipartRequestWithUUID(t *testing.T, target, uuid string, fields map[string]string, fileField, fileName string, fileData []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		require.NoError(t, writer.WriteField(name, value))
	}
	if fileField != "" {
		part, err := writer.CreateFormFile(fileField, fileName)
		require.NoError(t, err)
		_, err = part.Write(fileData)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set(ContentType, writer.FormDataContentType())
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", uuid)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestAdminDashboardTemplateRendersOpsDashboard(t *testing.T) {
	handler, sessions, _, _ := newAdminTemplateHandler(t)

	req := newAdminRequestWithUUID(http.MethodGet, "/admin", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTemplateHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, `class="admin-dashboard"`)
	require.Contains(t, body, `class="dashboard-kpi-grid"`)
	require.Contains(t, body, `Live game metrics`)
	require.Contains(t, body, `Platform pulse`)
	require.Contains(t, body, `Leaderboard preview`)
	require.Contains(t, body, `Recent game activity`)
	require.Contains(t, body, `Infrastructure status`)
	require.Contains(t, body, `Quick actions`)
	require.Contains(t, body, `73%`)
	require.Contains(t, body, `p95 118ms`)
	require.Contains(t, body, `href="/`+jsonTestUUID+`/admin/challenges"`)
	require.Contains(t, body, `/static/css/mapctf.css?v=`)
	require.NotContains(t, body, `Dashboard Placeholders`)
	require.NotContains(t, body, `<table>`)
}

func TestAdminTeamsTemplateHandlerListsPlatformAndMapLogos(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)

	platformLogo, err := teamManager.NewLogo("Platform Bee", "/static/svg/icons/badges/badge-bee.svg", true, false, 0, teams.NoUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(platformLogo))

	mapLogo, err := teamManager.NewLogo("Map Custom", "custom-map", false, true, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(mapLogo))

	rasterLogo, err := teamManager.NewLogo("Raster Custom", "/static/img/team-logos/badge-raster-custom.png", true, true, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(rasterLogo))

	otherLogo, err := teamManager.NewLogo("Other Map", "other-map", true, true, 0, jsonOtherTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(otherLogo))

	team, err := teamManager.New("Existing Team", "bee", false, true, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.Create(team))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/teams", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamsTemplateHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, `<option value="bee" selected>Platform Bee</option>`)
	require.Contains(t, body, `<option value="custom-map">Map Custom (disabled)</option>`)
	require.Contains(t, body, `<option value="/static/img/team-logos/badge-raster-custom.png">Raster Custom</option>`)
	require.NotContains(t, body, `Other Map`)
}

func TestAdminAddTeamModalUsesImageAwareLogoPreview(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "templates", "static", "inc", "modals", "add-team.html"))
	require.NoError(t, err)
	require.Contains(t, string(data), `class="admin-add-team-logo-preview-wrap"`)

	adminJS, err := os.ReadFile(filepath.Join("..", "templates", "static", "js", "admin.js"))
	require.NoError(t, err)
	js := string(adminJS)
	require.Contains(t, js, `var logoPreviewContainer = form.querySelector(".admin-add-team-logo-preview-wrap");`)
	require.Contains(t, js, `setAdminLogoMedia(logoPreviewContainer, logo);`)
	require.NotContains(t, js, `logoPreviewUse.setAttribute("xlink:href", "#icon--badge-" + logo);`)
}

func TestAdminTeamsTemplateUsesVersionedStaticAssets(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "templates", "admin", "teams.html"))
	require.NoError(t, err)

	body := string(data)
	require.Contains(t, body, `/static/css/mapctf.css?v=`)
	require.Contains(t, body, `/static/js/admin.js?v=`)
}

func TestAdminTeamLogosTemplateHandlerSeparatesProtectedAndCustomLogos(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)

	platformLogo, err := teamManager.NewLogo("Platform Bee", "/static/svg/icons/badges/badge-bee.svg", true, false, 0, teams.NoUUID)
	require.NoError(t, err)
	platformLogo.Protected = true
	require.NoError(t, teamManager.CreateLogo(platformLogo))

	customLogo, err := teamManager.NewLogo("Map Custom", "custom-map", true, true, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(customLogo))

	otherLogo, err := teamManager.NewLogo("Other Map", "other-map", true, true, 0, jsonOtherTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(otherLogo))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/team-logos", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosTemplateHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	customHeader := strings.Index(body, `<h3>Custom logos <span class="admin-logo-section-count">1</span></h3>`)
	customItem := strings.Index(body, `data-logo-name="Map Custom"`)
	catalogHeader := strings.Index(body, `<h3>Catalog <span class="admin-logo-section-count">1</span></h3>`)
	platformItem := strings.Index(body, `data-logo-name="Platform Bee"`)
	teamActions := strings.Index(body, `<section id="team-actions"`)
	logosHeader := strings.Index(body, `class="admin-logo-list-header"`)
	searchInput := strings.Index(body, `id="admin-logos-search"`)
	addLogoButton := strings.Index(body, `data-action="add-new-logo"`)
	platformGroup := strings.Index(body, `admin-logo-group--platform`)

	require.NotEqual(t, -1, customHeader)
	require.NotEqual(t, -1, customItem)
	require.NotEqual(t, -1, catalogHeader)
	require.NotEqual(t, -1, platformItem)
	require.NotEqual(t, -1, teamActions)
	require.NotEqual(t, -1, logosHeader)
	require.NotEqual(t, -1, searchInput)
	require.NotEqual(t, -1, addLogoButton)
	require.NotEqual(t, -1, platformGroup)
	require.Less(t, logosHeader, customHeader)
	require.Less(t, customHeader, customItem)
	require.Less(t, customItem, catalogHeader)
	require.Less(t, catalogHeader, platformItem)
	require.Greater(t, searchInput, logosHeader)
	require.Greater(t, addLogoButton, logosHeader)
	require.Less(t, searchInput, customHeader)
	require.Less(t, addLogoButton, customHeader)
	require.Less(t, searchInput, catalogHeader)
	require.Less(t, addLogoButton, catalogHeader)
	require.Greater(t, searchInput, teamActions)
	require.Greater(t, addLogoButton, teamActions)
	require.Contains(t, body, `<h3>Logos <span class="admin-logo-section-count">2</span></h3>`)
	require.Equal(t, 2, strings.Count(body, `class="admin-logo-section-header"`))
	require.NotContains(t, body, `admin-box-header admin-logo-catalog-header`)
	require.NotContains(t, body, "<h3>Platform logos</h3>")
	require.NotContains(t, body, `Other Map`)
}

func TestAdminAddLogoModalAllowsRasterImageUploads(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "templates", "static", "inc", "modals", "add-logo.html"))
	require.NoError(t, err)

	body := string(data)
	require.Contains(t, body, `name="logo_file"`)
	require.Contains(t, body, `accept=".svg,.gif,.png,.jpg,.jpeg,image/svg+xml,image/gif,image/png,image/jpeg"`)
	require.Contains(t, body, `required`)
	require.Contains(t, body, `Upload an SVG, GIF, PNG, or JPG/JPEG file.`)
	require.Contains(t, body, `class="admin-logo-upload-formats"`)
	require.Contains(t, body, `Accepted formats`)
	require.Contains(t, body, `Max file size: 512 KB`)
	require.Contains(t, body, `Recommended size: 64 x 48 px`)
	require.Contains(t, body, `class="admin-logo-upload-file-size"`)
	require.Contains(t, body, `No file selected`)
	for _, format := range []string{"SVG", "GIF", "PNG", "JPG/JPEG"} {
		require.Contains(t, body, `<span class="admin-logo-format-chip">`+format+`</span>`)
	}
	require.NotContains(t, body, `Upload Custom Image (optional)`)

	adminJS, err := os.ReadFile(filepath.Join("..", "templates", "static", "js", "admin.js"))
	require.NoError(t, err)
	require.Contains(t, string(adminJS), `var ADMIN_LOGO_UPLOAD_ACCEPT = ".svg,.gif,.png,.jpg,.jpeg,image/svg+xml,image/gif,image/png,image/jpeg";`)
	require.Contains(t, string(adminJS), `var ADMIN_LOGO_UPLOAD_FORMATS = ["SVG", "GIF", "PNG", "JPG/JPEG"];`)
	require.Contains(t, string(adminJS), `var ADMIN_LOGO_UPLOAD_MAX_BYTES = 512 * 1024;`)
	require.Contains(t, string(adminJS), `var ADMIN_LOGO_UPLOAD_MAX_LABEL = "512 KB";`)
	require.Contains(t, string(adminJS), `var ADMIN_LOGO_UPLOAD_RECOMMENDED_SIZE = "64 x 48 px";`)
	require.Contains(t, string(adminJS), `function formatAdminLogoFileSize(bytes)`)
	require.Contains(t, string(adminJS), `Selected file: " + logoFile.name + " (" + formatAdminLogoFileSize(logoFile.size) + ")"`)
	require.Contains(t, string(adminJS), `logoFile.size > ADMIN_LOGO_UPLOAD_MAX_BYTES`)
	require.Contains(t, string(adminJS), `function ensureAdminLogoUploadFormats(form)`)
	require.Contains(t, string(adminJS), `ensureAdminLogoUploadFormats(form);`)
	require.Contains(t, string(adminJS), `logoFileInput.setAttribute("accept", ADMIN_LOGO_UPLOAD_ACCEPT);`)
	require.Contains(t, string(adminJS), `logoFileInput.setAttribute("required", "required");`)
	require.Contains(t, string(adminJS), `Upload a logo file before creating a custom logo`)
}

func TestAdminTeamLogosTemplateUsesVersionedStaticAssets(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "templates", "admin", "team-logos.html"))
	require.NoError(t, err)

	body := string(data)
	require.Contains(t, body, `/static/css/mapctf.css?v=`)
	require.Contains(t, body, `/static/js/admin.js?v=`)
}

func TestAdminTeamLogosPOSTHandlerRejectsMultipartLogoCreateWithoutFile(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	staticDir := t.TempDir()
	handler.Config.Map.StaticDir = staticDir

	req := newAdminMultipartRequestWithUUID(t, "/admin/team-logos/logos", jsonTestUUID, map[string]string{
		"name":      "Slug Only Badge",
		"logo":      "slug-only-badge",
		"logo_slug": "slug-only-badge",
	}, "", "", nil)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosPOSTHandler(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Error(t, teamManager.DB.Where("uuid = ? AND name = ?", jsonTestUUID, "Slug Only Badge").First(&teams.TeamLogo{}).Error)
	require.NoFileExists(t, filepath.Join(staticDir, "svg", "icons", "custom", "badge-slug-only-badge.svg"))
}

func TestAdminTeamLogosPOSTHandlerStoresRasterLogoUpload(t *testing.T) {
	tests := []struct {
		name         string
		slug         string
		fileName     string
		fileData     []byte
		expectedLogo string
	}{
		{
			name:         "PNG",
			slug:         "raster-png",
			fileName:     "raster-png.png",
			fileData:     []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
			expectedLogo: "/static/img/team-logos/badge-raster-png.png",
		},
		{
			name:         "JPG",
			slug:         "raster-jpg",
			fileName:     "raster-jpg.jpg",
			fileData:     []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x01, 0x00, 0x60},
			expectedLogo: "/static/img/team-logos/badge-raster-jpg.jpg",
		},
		{
			name:         "JPEG",
			slug:         "raster-jpeg",
			fileName:     "raster-jpeg.jpeg",
			fileData:     []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x01, 0x00, 0x60},
			expectedLogo: "/static/img/team-logos/badge-raster-jpeg.jpeg",
		},
		{
			name:         "GIF",
			slug:         "raster-gif",
			fileName:     "raster-gif.gif",
			fileData:     []byte{'G', 'I', 'F', '8', '9', 'a', 0x01, 0x00, 0x01, 0x00},
			expectedLogo: "/static/img/team-logos/badge-raster-gif.gif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, sessions, _, teamManager := newAdminTemplateHandler(t)
			staticDir := t.TempDir()
			handler.Config.Map.StaticDir = staticDir

			logoName := "Raster " + tt.name
			req := newAdminMultipartRequestWithUUID(t, "/admin/team-logos/logos", jsonTestUUID, map[string]string{
				"name":      logoName,
				"logo":      tt.slug,
				"logo_slug": tt.slug,
			}, "logo_file", tt.fileName, tt.fileData)
			ctx, err := sessions.Load(req.Context(), "")
			require.NoError(t, err)
			sessions.Put(ctx, string(ContextKeyUser), "admin")
			sessions.Put(ctx, string(ContextKeyAdmin), true)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.AdminTeamLogosPOSTHandler(rr, req)
			require.Equal(t, http.StatusOK, rr.Code)

			var row teams.TeamLogo
			require.NoError(t, teamManager.DB.Where("uuid = ? AND name = ?", jsonTestUUID, logoName).First(&row).Error)
			require.Equal(t, tt.expectedLogo, row.Logo)
			require.True(t, row.Custom)
			require.FileExists(t, filepath.Join(staticDir, strings.TrimPrefix(tt.expectedLogo, "/static/")))
		})
	}
}

func TestAdminTeamLogosPOSTHandlerStoresSVGUploadWithCustomLogoAssets(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	staticDir := t.TempDir()
	handler.Config.Map.StaticDir = staticDir

	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 48"><script>alert(1)</script><path d="M4 4h56v40H4z"/></svg>`)
	req := newAdminMultipartRequestWithUUID(t, "/admin/team-logos/logos", jsonTestUUID, map[string]string{
		"name":      "SVG Badge",
		"logo":      "svg-badge",
		"logo_slug": "svg-badge",
	}, "logo_file", "svg-badge.svg", svgData)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosPOSTHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var row teams.TeamLogo
	require.NoError(t, teamManager.DB.Where("uuid = ? AND name = ?", jsonTestUUID, "SVG Badge").First(&row).Error)
	require.Equal(t, "/static/img/team-logos/badge-svg-badge.svg", row.Logo)
	require.True(t, row.Custom)
	savedPath := filepath.Join(staticDir, "img", "team-logos", "badge-svg-badge.svg")
	require.FileExists(t, savedPath)
	require.NotContains(t, strings.ToLower(string(mustReadFile(t, savedPath))), "<script")
	require.NoFileExists(t, filepath.Join(staticDir, "svg", "icons", "custom", "badge-svg-badge.svg"))
}

func TestAdminTeamLogosPOSTHandlerRejectsUnsupportedLogoUploadType(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	staticDir := t.TempDir()
	handler.Config.Map.StaticDir = staticDir

	req := newAdminMultipartRequestWithUUID(t, "/admin/team-logos/logos", jsonTestUUID, map[string]string{
		"name":      "Text Badge",
		"logo":      "text-badge",
		"logo_slug": "text-badge",
	}, "logo_file", "text-badge.txt", []byte("not a supported logo"))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosPOSTHandler(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Error(t, teamManager.DB.Where("uuid = ? AND name = ?", jsonTestUUID, "Text Badge").First(&teams.TeamLogo{}).Error)
	require.NoFileExists(t, filepath.Join(staticDir, "img", "team-logos", "badge-text-badge.txt"))
}

func TestAdminTeamLogosPOSTHandlerRejectsDuplicateRasterLogoUpload(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	staticDir := t.TempDir()
	handler.Config.Map.StaticDir = staticDir

	firstUpload := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x01}
	req := newAdminMultipartRequestWithUUID(t, "/admin/team-logos/logos", jsonTestUUID, map[string]string{
		"name":      "Raster Badge",
		"logo":      "raster-badge",
		"logo_slug": "raster-badge",
	}, "logo_file", "raster-badge.png", firstUpload)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosPOSTHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	secondUpload := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x02}
	req = newAdminMultipartRequestWithUUID(t, "/admin/team-logos/logos", jsonTestUUID, map[string]string{
		"name":      "Duplicate Raster Badge",
		"logo":      "raster-badge",
		"logo_slug": "raster-badge",
	}, "logo_file", "raster-badge.png", secondUpload)
	ctx, err = sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()
	handler.AdminTeamLogosPOSTHandler(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Error(t, teamManager.DB.Where("uuid = ? AND name = ?", jsonTestUUID, "Duplicate Raster Badge").First(&teams.TeamLogo{}).Error)
	require.FileExists(t, filepath.Join(staticDir, "img", "team-logos", "badge-raster-badge.png"))
	require.Equal(t, firstUpload, mustReadFile(t, filepath.Join(staticDir, "img", "team-logos", "badge-raster-badge.png")))
}

func TestAdminTeamLogosTemplateHandlerRendersRasterLogoImage(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)

	customLogo, err := teamManager.NewLogo("Raster Badge", "/static/img/team-logos/badge-raster-badge.png", true, true, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(customLogo))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/team-logos", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosTemplateHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, `data-logo-symbol="/static/img/team-logos/badge-raster-badge.png"`)
	require.Contains(t, body, `data-logo-file="/static/img/team-logos/badge-raster-badge.png"`)
	require.Contains(t, body, `<img class="icon icon--badge admin-logo-img" src="/static/img/team-logos/badge-raster-badge.png" alt="" />`)
	require.NotContains(t, body, `#icon--badge-/static/img/team-logos/badge-raster-badge.png`)
}

func TestAdminChatTemplateHandlerIncludesRecentChatSection(t *testing.T) {
	handler, sessions, chatManager, teamManager := newAdminTemplateHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:   "blue-team",
		UUID:   jsonTestUUID,
		Active: true,
	}))

	allTeams, err := teamManager.GetAll(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, allTeams, 1)

	require.NoError(t, chatManager.CreateNew("alice", "hello admin chat", allTeams[0].ID, chat.DefaultMaxLen))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/chat", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "World Chat")
	require.Contains(t, body, "hello admin chat")
	require.Contains(t, body, "blue-team")
	require.Contains(t, body, "alice")
}

func TestAdminChatTemplateHandlerShowsEmptyChatState(t *testing.T) {
	handler, sessions, _, _ := newAdminTemplateHandler(t)

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/chat", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "No chat messages yet.")
}

func TestAdminActivityTemplateHandlerIncludesActivityEntries(t *testing.T) {
	handler, sessions, logManager := newAdminActivityTemplateHandler(t)

	activity, err := logManager.NewActivity(true, "Blue Team", "completed", "Captured Spain", 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, logManager.CreateActivity(activity))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/activity", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminActivityTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Activity Log")
	require.Contains(t, body, "Blue Team")
	require.Contains(t, body, "completed")
	require.Contains(t, body, "Captured Spain")
}

func TestAdminActivityPOSTHandlerCreatesCustomEntry(t *testing.T) {
	handler, sessions, logManager := newAdminActivityTemplateHandler(t)

	payload := AdminActivityCreateRequest{
		Subject: "Blue Team",
		Action:  "custom",
		Message: "Custom activity",
		Visible: true,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/activity", bytes.NewReader(body))
	req.Header.Set(ContentType, JSONApplicationUTF8)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uuid", jsonTestUUID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminActivityPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 1)
	require.True(t, activityEntries[0].Visible)
	require.Equal(t, "Blue Team", activityEntries[0].Subject)
	require.Equal(t, "custom", activityEntries[0].Action)
	require.Equal(t, "Custom activity", activityEntries[0].Message)
}

func TestAdminActivityDeletePOSTHandlerDeletesEntry(t *testing.T) {
	handler, sessions, logManager := newAdminActivityTemplateHandler(t)

	activity, err := logManager.NewActivity(true, "Blue Team", "announcement", "Delete me", 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, logManager.CreateActivity(activity))

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/activity/1/delete", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", strconv.FormatUint(uint64(activityEntries[0].ID), 10))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminActivityDeletePOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err = logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 0)
}

func TestAdminChallengesTemplateHandlerShowsChallengeRelatedActivity(t *testing.T) {
	handler, sessions, countriesManager, challengesManager, logManager, teamManager := newAdminChallengesTemplateHandler(t)

	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
	}))
	require.NoError(t, challengesManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 3},
		Name:  "Web",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Model:      gorm.Model{ID: 77},
		Title:      "Spanish Challenge",
		CategoryID: 3,
		Country:    "ES",
		Active:     true,
		Points:     100,
		Flag:       "MAP{es}",
		UUID:       jsonTestUUID,
	}))
	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Model:   gorm.Model{ID: 5},
		Name:    "Blue Team",
		UUID:    jsonTestUUID,
		Active:  true,
		Visible: true,
	}))

	activity, err := logManager.NewActivity(true, "Blue Team", "completed", "Spanish Challenge", 77, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, logManager.CreateActivity(activity))
	require.NoError(t, logManager.CreateFailuresLog(logs.FailuresLog{
		ChallengeID: 77,
		TeamID:      5,
		Flag:        "wrong-flag",
		UUID:        jsonTestUUID,
	}))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/challenges", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChallengesTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Spanish Challenge")
	require.Contains(t, body, "Blue Team")
	require.Contains(t, body, "completed")
	require.Contains(t, body, "Failure")
	require.Contains(t, body, "wrong-flag")
}

func TestAdminChallengeUpdatePOSTHandlerLogsEnableAndDisableStateChanges(t *testing.T) {
	handler, sessions, _, challengesManager, logManager := newAdminChallengeActivityHandler(t)

	require.NoError(t, challengesManager.CreateCategory(challenges.Category{
		Model: gorm.Model{ID: 1},
		Name:  "Web",
		UUID:  jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Model:       gorm.Model{ID: 50},
		Title:       "Spain",
		CategoryID:  1,
		Active:      false,
		Points:      100,
		Bonus:       0,
		BonusDecay:  0,
		HintPenalty: 0,
		HelpPenalty: 0,
		Flag:        "MAP{es}",
		Hint:        "hint",
		UUID:        jsonTestUUID,
	}))

	makeRequest := func(activeValue string) *http.Request {
		payload := AdminChallengeCreateRequest{
			Title:       "Spain",
			Description: "",
			CategoryID:  "1",
			Country:     "",
			Active:      activeValue,
			Points:      "100",
			Bonus:       "0",
			BonusDecay:  "0",
			HintPenalty: "0",
			HelpPenalty: "0",
			Flag:        "MAP{es}",
			Hint:        "hint",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/admin/challenges/50", bytes.NewReader(body))
		req.Header.Set(ContentType, JSONApplicationUTF8)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("uuid", jsonTestUUID)
		routeCtx.URLParams.Add("id", "50")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
		ctx, err := sessions.Load(req.Context(), "")
		require.NoError(t, err)
		sessions.Put(ctx, string(ContextKeyUser), "admin")
		sessions.Put(ctx, string(ContextKeyAdmin), true)
		return req.WithContext(ctx)
	}

	rr := httptest.NewRecorder()
	handler.AdminChallengeUpdatePOSTHandler(rr, makeRequest("true"))
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.AdminChallengeUpdatePOSTHandler(rr, makeRequest("false"))
	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 2)
	require.Equal(t, "admin", activityEntries[0].Subject)
	require.Equal(t, "enabled", activityEntries[0].Action)
	require.Equal(t, "Challenge Spain (Web) was enabled: 100 points", activityEntries[0].Message)
	require.Equal(t, uint(50), activityEntries[0].ChallengeID)
	require.Equal(t, "disabled", activityEntries[1].Action)
}

func TestAdminChallengesBulkStateChangeHandlersLogActivity(t *testing.T) {
	handler, sessions, _, challengesManager, logManager := newAdminChallengeActivityHandler(t)

	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:  "One",
		Active: false,
		Flag:   "MAP{one}",
		UUID:   jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:  "Two",
		Active: false,
		Flag:   "MAP{two}",
		UUID:   jsonTestUUID,
	}))

	makeRequest := func() *http.Request {
		req := newAdminRequestWithUUID(http.MethodPost, "/admin/challenges", jsonTestUUID)
		ctx, err := sessions.Load(req.Context(), "")
		require.NoError(t, err)
		sessions.Put(ctx, string(ContextKeyUser), "admin")
		sessions.Put(ctx, string(ContextKeyAdmin), true)
		return req.WithContext(ctx)
	}

	rr := httptest.NewRecorder()
	handler.AdminChallengesEnableAllPOSTHandler(rr, makeRequest())
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.AdminChallengesDisableAllPOSTHandler(rr, makeRequest())
	require.Equal(t, http.StatusOK, rr.Code)

	activityEntries, err := logManager.AllActivity(jsonTestUUID)
	require.NoError(t, err)
	require.Len(t, activityEntries, 2)
	require.Equal(t, "enabled", activityEntries[0].Action)
	require.Equal(t, "enabled all challenges (2)", activityEntries[0].Message)
	require.Equal(t, "disabled", activityEntries[1].Action)
	require.Equal(t, "disabled all challenges (2)", activityEntries[1].Message)
}

func TestAdminChatTemplateHandlerIncludesModerationControls(t *testing.T) {
	handler, sessions, chatManager, teamManager := newAdminTemplateHandler(t)

	require.NoError(t, teamManager.Create(teams.PlatformTeam{
		Name:   "blue-team",
		UUID:   jsonTestUUID,
		Active: true,
	}))
	allTeams, err := teamManager.GetAll(jsonTestUUID)
	require.NoError(t, err)

	require.NoError(t, chatManager.CreateNew("alice", "needs moderation", allTeams[0].ID, chat.DefaultMaxLen))

	req := newAdminRequestWithUUID(http.MethodGet, "/admin/chat", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatTemplateHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "/admin/chat/")
	require.Contains(t, body, "/visibility")
	require.Contains(t, body, "/delete")
	require.Contains(t, body, "admin-chat-list")
	require.Contains(t, body, "data-admin-reload-on-success=\"true\"")
	require.Contains(t, body, "Hide")
	require.Contains(t, body, "Delete")
}

func TestAdminChatSetHiddenPOSTHandlerHidesEntry(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "hide me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/visibility", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)
	req.PostForm = map[string][]string{"hidden": {"true"}}

	rr := httptest.NewRecorder()
	handler.AdminChatSetHiddenPOSTHandler(rr, req)

	require.Equal(t, http.StatusFound, rr.Code)
	updated, err := chatManager.GetByID(entries[0].ID)
	require.NoError(t, err)
	require.True(t, updated.Hidden)
	require.Contains(t, rr.Header().Get("Location"), "status=ok")
}

func TestAdminChatSetHiddenPOSTHandlerHidesEntryForAJAX(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "hide me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	reqBody := bytes.NewBufferString(`{"hidden":true}`)
	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/visibility", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	req.Header.Set(ContentType, JSONApplication)
	req.Header.Set("Accept", JSONApplication)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Body = io.NopCloser(reqBody)
	req.ContentLength = int64(reqBody.Len())

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatSetHiddenPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Chat message hidden", resp.Message)

	updated, err := chatManager.GetByID(entries[0].ID)
	require.NoError(t, err)
	require.True(t, updated.Hidden)
}

func TestAdminChatSetHiddenPOSTHandlerHidesEntryForAJAXStringValue(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "hide me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	reqBody := bytes.NewBufferString(`{"hidden":"true"}`)
	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/visibility", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	req.Header.Set(ContentType, JSONApplication)
	req.Header.Set("Accept", JSONApplication)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Body = io.NopCloser(reqBody)
	req.ContentLength = int64(reqBody.Len())

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatSetHiddenPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Chat message hidden", resp.Message)

	updated, err := chatManager.GetByID(entries[0].ID)
	require.NoError(t, err)
	require.True(t, updated.Hidden)
}

func TestAdminChatDeletePOSTHandlerDeletesScopedEntry(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "delete me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/delete", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatDeletePOSTHandler(rr, req)

	require.Equal(t, http.StatusFound, rr.Code)
	remaining, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, remaining, 0)
	require.Contains(t, rr.Header().Get("Location"), "status=ok")
}

func TestAdminChatDeletePOSTHandlerDeletesScopedEntryForAJAX(t *testing.T) {
	handler, sessions, chatManager, _ := newAdminTemplateHandler(t)

	require.NoError(t, chatManager.CreateNew("alice", "delete me", 1, chat.DefaultMaxLen))
	entries, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/chat/1/delete", jsonTestUUID)
	routeCtx := chi.RouteContext(req.Context())
	routeCtx.URLParams.Add("id", "1")
	req.Header.Set(ContentType, JSONApplication)
	req.Header.Set("Accept", JSONApplication)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Body = io.NopCloser(bytes.NewBufferString(`{}`))

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminChatDeletePOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Chat message deleted", resp.Message)

	remaining, err := chatManager.GetAll()
	require.NoError(t, err)
	require.Len(t, remaining, 0)
}

func TestJSONChatHandlerExcludesHiddenEntries(t *testing.T) {
	db := newJSONTestDB(t)

	chatManager, err := chat.CreateChatManager(db, jsonTestUUID)
	require.NoError(t, err)

	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{Map: config.ConfigurationMap{UUID: jsonTestUUID}}),
		WithChat(chatManager),
	)

	require.NoError(t, chatManager.Create(chat.ChatEntry{
		Username: "visible",
		Body:     "shown",
		TeamID:   1,
		UUID:     jsonTestUUID,
		Hidden:   false,
	}))
	require.NoError(t, chatManager.Create(chat.ChatEntry{
		Username: "hidden",
		Body:     "secret",
		TeamID:   1,
		UUID:     jsonTestUUID,
		Hidden:   true,
	}))

	req := newRequestWithUUID(http.MethodGet, "/json/chat", jsonTestUUID)
	rr := httptest.NewRecorder()

	handler.JSONChatHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []chat.ChatEntry
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	require.Equal(t, "visible", resp[0].Username)
}

func TestAdminCountriesDeleteAllPOSTHandlerDeletesScopedCountriesAndClearsChallengeCountries(t *testing.T) {
	handler, sessions, countriesManager, challengesManager := newAdminCountryActionHandler(t)

	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "Spain",
		CountryCode: "ES",
		Active:      true,
	}))
	require.NoError(t, countriesManager.Create(countries.MapCountry{
		Name:        "France",
		CountryCode: "FR",
		Active:      true,
	}))
	require.NoError(t, countriesManager.DB.Create(&countries.MapCountry{
		Name:        "Other UUID Country",
		CountryCode: "DE",
		Active:      true,
		UUID:        jsonOtherTestUUID,
	}).Error)

	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Scoped challenge",
		Country: "ES",
		Active:  true,
		UUID:    jsonTestUUID,
	}))
	require.NoError(t, challengesManager.Create(challenges.Challenge{
		Title:   "Other UUID challenge",
		Country: "DE",
		Active:  true,
		UUID:    jsonOtherTestUUID,
	}))

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/countries/delete-all", jsonTestUUID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set(ContentType, JSONApplication)
	req.Body = io.NopCloser(bytes.NewBufferString(`{}`))

	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminCountriesDeleteAllPOSTHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, JSONApplicationUTF8, rr.Header().Get(ContentType))

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, "Deleted 2 country(ies)", resp.Message)

	remainingCountries, err := countriesManager.GetAll()
	require.NoError(t, err)
	require.Empty(t, remainingCountries)

	var otherCountries []countries.MapCountry
	require.NoError(t, countriesManager.DB.Where("uuid = ?", jsonOtherTestUUID).Find(&otherCountries).Error)
	require.Len(t, otherCountries, 1)
	require.Equal(t, "DE", otherCountries[0].CountryCode)

	scopedChallenge, err := challengesManager.GetByID(1, jsonTestUUID)
	require.NoError(t, err)
	require.Empty(t, scopedChallenge.Country)

	otherChallenge, err := challengesManager.GetByID(2, jsonOtherTestUUID)
	require.NoError(t, err)
	require.Equal(t, "DE", otherChallenge.Country)
}

func TestAdminCountriesDeleteAllPOSTHandlerReturnsErrorWithoutManagers(t *testing.T) {
	sessionManager := scs.New()
	handler := CreateHandlersMap(
		WithConfig(config.MapCTFConfiguration{
			Map: config.ConfigurationMap{
				UUID:         jsonTestUUID,
				TemplatesDir: filepath.Join("..", "templates"),
			},
		}),
		WithSessions(sessionManager),
	)

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/countries/delete-all", jsonTestUUID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set(ContentType, JSONApplication)
	req.Body = io.NopCloser(bytes.NewBufferString(`{}`))

	ctx, err := sessionManager.Load(req.Context(), "")
	require.NoError(t, err)
	sessionManager.Put(ctx, string(ContextKeyUser), "admin")
	sessionManager.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminCountriesDeleteAllPOSTHandler(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp adminActionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "error", resp.Status)
	require.Equal(t, "Countries or challenges manager is not initialized", resp.Message)
}

func TestAdminTeamLogoUpdatePOSTHandler_rejectsPlatformNameChange(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	platformLogo, err := teamManager.NewLogo("SeedLogo", "invader", true, false, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(platformLogo))

	var stored teams.TeamLogo
	require.NoError(t, teamManager.DB.Where("uuid = ? AND name = ?", jsonTestUUID, "SeedLogo").First(&stored).Error)

	body, err := json.Marshal(AdminLogoUpdateRequest{
		Name:      "Changed",
		Enabled:   "true",
		Protected: "false",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/teams/logos/"+strconv.FormatUint(uint64(stored.ID), 10), bytes.NewReader(body))
	req.Header.Set(ContentType, JSONApplicationUTF8)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("uuid", jsonTestUUID)
	rc.URLParams.Add("id", strconv.FormatUint(uint64(stored.ID), 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogoUpdatePOSTHandler(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var row teams.TeamLogo
	require.NoError(t, teamManager.DB.First(&row, stored.ID).Error)
	require.Equal(t, "SeedLogo", row.Name)
}

func TestAdminTeamLogoUpdatePOSTHandler_allowsPlatformToggleWithoutRename(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	platformLogo, err := teamManager.NewLogo("PlatToggle", "invader", true, false, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(platformLogo))

	var stored teams.TeamLogo
	require.NoError(t, teamManager.DB.Where("name = ?", "PlatToggle").First(&stored).Error)

	body, err := json.Marshal(AdminLogoUpdateRequest{
		Name:      "PlatToggle",
		Enabled:   "false",
		Protected: "true",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/teams/logos/"+strconv.FormatUint(uint64(stored.ID), 10), bytes.NewReader(body))
	req.Header.Set(ContentType, JSONApplicationUTF8)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("uuid", jsonTestUUID)
	rc.URLParams.Add("id", strconv.FormatUint(uint64(stored.ID), 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogoUpdatePOSTHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var row teams.TeamLogo
	require.NoError(t, teamManager.DB.First(&row, stored.ID).Error)
	require.False(t, row.Enabled)
	require.True(t, row.Protected)
	require.Equal(t, "invader", row.Logo)
}

func TestAdminTeamLogosDeleteAllPOSTHandler_deletesOnlyCustom(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	plat, err := teamManager.NewLogo("PlatOnly", "bee", true, false, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(plat))
	cust, err := teamManager.NewLogo("CustOnly", "myslug", true, true, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(cust))

	req := newAdminRequestWithUUID(http.MethodPost, "/admin/teams/logos/delete-all", jsonTestUUID)
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamLogosDeleteAllPOSTHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var logos []teams.TeamLogo
	require.NoError(t, teamManager.DB.Where("uuid = ?", jsonTestUUID).Order("id").Find(&logos).Error)
	require.Len(t, logos, 1)
	require.False(t, logos[0].Custom)
	require.Equal(t, "PlatOnly", logos[0].Name)
}

func TestAdminTeamsImportLogosHandler_doesNotOverwritePlatformSlug(t *testing.T) {
	handler, sessions, _, teamManager := newAdminTemplateHandler(t)
	plat, err := teamManager.NewLogo("ImportPlat", "bee", true, false, 0, jsonTestUUID)
	require.NoError(t, err)
	require.NoError(t, teamManager.CreateLogo(plat))

	payload := adminTeamsTransferPayload{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Logos: []adminTeamsTransferLogo{
			{Name: "ImportPlat", Logo: "tampered-slug", Enabled: false, Custom: false, Protected: true, Used: true},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/team-logos/import", bytes.NewReader(body))
	req.Header.Set(ContentType, JSONApplicationUTF8)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("uuid", jsonTestUUID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	ctx, err := sessions.Load(req.Context(), "")
	require.NoError(t, err)
	sessions.Put(ctx, string(ContextKeyUser), "admin")
	sessions.Put(ctx, string(ContextKeyAdmin), true)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.AdminTeamsImportLogosHandler(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var row teams.TeamLogo
	require.NoError(t, teamManager.DB.Where("name = ?", "ImportPlat").First(&row).Error)
	require.Equal(t, "bee", row.Logo)
	require.False(t, row.Enabled)
	require.True(t, row.Protected)
}

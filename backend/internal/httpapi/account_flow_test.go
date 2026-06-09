package httpapi

// Implements DESIGN-006 AuthController and DESIGN-008 ProfileController composed account-flow verification.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/profile"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

type accountFlowProfileService struct {
	userID  uuid.UUID
	profile profile.UserProfile
	updated bool
}

func (s *accountFlowProfileService) GetProfile(_ context.Context, userID uuid.UUID) (profile.UserProfile, error) {
	s.userID = userID
	return s.profile, nil
}

func (s *accountFlowProfileService) UpdatePreferences(_ context.Context, userID uuid.UUID, req profile.UpdateRequest) (profile.UpdateResult, error) {
	s.userID = userID
	s.updated = true
	if req.DisplayName != nil {
		s.profile.DisplayName = *req.DisplayName
	}
	s.profile.UnitSystem = req.UnitSystem
	s.profile.ThemePreference = req.ThemePreference
	return profile.UpdateResult{Profile: s.profile, RequiresUnitRecalculation: true}, nil
}

type accountFlowUserDataService struct {
	userID         uuid.UUID
	deletedItemID  uuid.UUID
	deletedKind    repository.SavedItemKind
	historyCleared bool
}

func (s *accountFlowUserDataService) ListSaved(_ context.Context, userID uuid.UUID, kind *repository.SavedItemKind) ([]repository.SavedItem, error) {
	s.userID = userID
	itemKind := repository.SavedItemKindFavorite
	if kind != nil {
		itemKind = *kind
	}
	return []repository.SavedItem{{ID: uuid.New(), UserID: userID, ItemID: uuid.MustParse("2a4d4f9d-6cb6-4fd5-bdc6-a9685858b7d8"), Kind: itemKind}}, nil
}

func (s *accountFlowUserDataService) DeleteSaved(_ context.Context, userID uuid.UUID, itemID uuid.UUID, kind repository.SavedItemKind) error {
	s.userID = userID
	s.deletedItemID = itemID
	s.deletedKind = kind
	return nil
}

func (s *accountFlowUserDataService) ListHistory(_ context.Context, userID uuid.UUID, _ int) ([]userdata.SearchHistoryEntry, error) {
	s.userID = userID
	return []userdata.SearchHistoryEntry{{ID: uuid.New(), Query: "lentils", Mode: "catalog", FiltersHash: "filters-v1"}}, nil
}

func (s *accountFlowUserDataService) ClearHistory(_ context.Context, userID uuid.UUID) error {
	s.userID = userID
	s.historyCleared = true
	return nil
}

type accountFlowExportService struct {
	userID uuid.UUID
	format string
}

func (s *accountFlowExportService) BuildExport(_ context.Context, userID uuid.UUID, format string) (userdata.ExportPayload, error) {
	s.userID = userID
	s.format = format
	return userdata.ExportPayload{Format: format, ContentType: "application/json", Filename: "account.json", Body: []byte(`{"format":"` + format + `"}`)}, nil
}

type accountFlowDeletionService struct {
	userID uuid.UUID
}

func (s *accountFlowDeletionService) RequestDeletion(_ context.Context, userID uuid.UUID) (repository.DataDeletionRequest, error) {
	s.userID = userID
	return repository.DataDeletionRequest{ID: uuid.MustParse("b4b85b0a-4fc4-4c6d-83cf-05773d663226"), UserID: userID, Status: "pending", RequestedAt: time.Now().UTC()}, nil
}

// TestComposedAccountFlowThroughGateway verifies DESIGN-006 AuthController middleware order with Phase 03 profile, export, saved-data, and deletion routes.
func TestComposedAccountFlowThroughGateway(t *testing.T) {
	cfg := testConfig()
	cfg.Account.AccessCookieName = "__Host-test_access"
	cfg.Account.RefreshCookieName = "__Host-test_refresh"
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.MustParse("9b3458d6-8b53-4ca8-9770-b369723289ce"), nil)
	userID := authenticator.sessions.(*httpSessionRepository).byHash[auth.HashRefreshToken(authCookies[1].Value)].UserID
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	csrf := NewCSRFManager(cfg, nil)
	sessionManager := NewAuthSessionManager(cfg, csrf)
	sessionManager.now = func() time.Time { return now }
	authService := &fakeAuthService{session: auth.AuthSession{
		UserID:                 userID,
		AccessToken:            authCookies[0].Value,
		RefreshToken:           authCookies[1].Value,
		AccessExpiresAt:        now.Add(15 * time.Minute),
		RefreshExpiresAt:       now.Add(7 * 24 * time.Hour),
		HasVerifiedLoginMethod: true,
		Role:                   "user",
	}}
	profiles := &accountFlowProfileService{profile: profile.UserProfile{UserID: userID, DisplayName: "", UnitSystem: repository.UnitSystemMetric, ThemePreference: "system"}}
	userData := &accountFlowUserDataService{}
	exports := &accountFlowExportService{}
	deletions := &accountFlowDeletionService{}
	routes := []RouteDefinition{}
	for _, controllerRoutes := range [][]RouteDefinition{
		NewAuthController(authService, sessionManager).Routes(),
		NewProfileController(profiles).Routes(),
		NewUserDataController(userData).Routes(),
		NewExportController(exports).Routes(),
		NewAccountDeletionController(deletions, sessionManager).Routes(),
	} {
		routes = append(routes, controllerRoutes...)
	}
	audit := &auditSink{}
	telemetry := &observability.MemorySink{}
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: csrf, Auth: authenticator, Audit: audit, Logs: telemetry, Metrics: telemetry, Routes: routes})

	resp := accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/register", `{"email":"flow@example.test","password":"StrongerPassword1!","privacyPolicyVersion":"privacy-v1","termsVersion":"terms-v1"}`, nil, "")
	if resp.StatusCode != fiber.StatusCreated || findCookie(resp.Cookies(), cfg.Account.AccessCookieName) == nil || authService.registerCall != 1 {
		t.Fatalf("register = %d cookies=%+v calls=%d", resp.StatusCode, resp.Cookies(), authService.registerCall)
	}
	registerCookies := resp.Cookies()
	resp.Body.Close()
	if len(audit.entries) == 0 || len(telemetry.Metrics) == 0 || len(telemetry.Logs) == 0 {
		t.Fatalf("missing account-flow cross-cutting telemetry: audit=%d metrics=%d logs=%d", len(audit.entries), len(telemetry.Metrics), len(telemetry.Logs))
	}

	resp = accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/login", `{"email":"flow@example.test","password":"StrongerPassword1!"}`, nil, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || authService.loginCall != 1 {
		t.Fatalf("login = %d calls=%d", resp.StatusCode, authService.loginCall)
	}

	resp = accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/refresh", "", registerCookies, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || authService.refreshCall != 1 || authService.lastRefresh != authCookies[1].Value {
		t.Fatalf("refresh = %d calls=%d token=%q", resp.StatusCode, authService.refreshCall, authService.lastRefresh)
	}

	token, csrfCookies := fetchCSRFToken(t, app, registerCookies...)
	resp = accountFlowJSON(t, app, fiber.MethodPut, "/api/v1/profile", `{"displayName":"Ada","unitSystem":"imperial","themePreference":"dark"}`, csrfCookies, token)
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["requiresUnitRecalculation"] != true || !profiles.updated || profiles.userID != userID {
		t.Fatalf("profile update = %d body=%+v service=%+v", resp.StatusCode, body, profiles)
	}

	resp = accountFlowJSON(t, app, fiber.MethodGet, "/api/v1/profile", "", csrfCookies, "")
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["displayName"] != "Ada" {
		t.Fatalf("profile read = %d body=%+v", resp.StatusCode, body)
	}

	resp = accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/verify-email", `{"userId":"`+uuid.NewString()+`"}`, csrfCookies, token)
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["hasVerifiedLoginMethod"] != true || authService.verifiedUser != userID {
		t.Fatalf("verify = %d body=%+v verified=%s", resp.StatusCode, body, authService.verifiedUser)
	}

	resp = accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/password-reset/request", `{"email":"flow@example.test"}`, nil, "")
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["accepted"] != true || body.Data["resetToken"] != nil || authService.resetEmail != "flow@example.test" {
		t.Fatalf("reset request = %d body=%+v email=%q", resp.StatusCode, body, authService.resetEmail)
	}
	resp = accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/password-reset/consume", `{"token":"reset-token","newPassword":"NewPassword1!"}`, nil, "")
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["reset"] != true || authService.resetUseToken != "reset-token" {
		t.Fatalf("reset consume = %d body=%+v token=%q", resp.StatusCode, body, authService.resetUseToken)
	}

	resp = accountFlowJSON(t, app, fiber.MethodGet, "/api/v1/saved-items?kind=favorite", "", csrfCookies, "")
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || userData.userID != userID || len(body.Data["items"].([]any)) != 1 {
		t.Fatalf("saved items = %d body=%+v user=%s", resp.StatusCode, body, userData.userID)
	}
	resp = accountFlowJSON(t, app, fiber.MethodGet, "/api/v1/search-history", "", csrfCookies, "")
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || len(body.Data["history"].([]any)) != 1 {
		t.Fatalf("history = %d body=%+v", resp.StatusCode, body)
	}

	resp = accountFlowJSON(t, app, fiber.MethodGet, "/api/v1/account/export?format=json", "", csrfCookies, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || exports.userID != userID || exports.format != "json" || resp.Header.Get("Content-Disposition") == "" {
		t.Fatalf("export = %d user=%s format=%q disposition=%q", resp.StatusCode, exports.userID, exports.format, resp.Header.Get("Content-Disposition"))
	}

	resp = accountFlowJSON(t, app, fiber.MethodDelete, "/api/v1/search-history", "", csrfCookies, token)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || !userData.historyCleared {
		t.Fatalf("clear history = %d cleared=%t", resp.StatusCode, userData.historyCleared)
	}

	itemID := uuid.MustParse("2a4d4f9d-6cb6-4fd5-bdc6-a9685858b7d8")
	resp = accountFlowJSON(t, app, fiber.MethodDelete, "/api/v1/saved-items/favorite/"+itemID.String(), "", csrfCookies, token)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || userData.deletedItemID != itemID || userData.deletedKind != repository.SavedItemKindFavorite {
		t.Fatalf("delete saved = %d item=%s kind=%s", resp.StatusCode, userData.deletedItemID, userData.deletedKind)
	}

	resp = accountFlowJSON(t, app, fiber.MethodPost, "/api/v1/auth/logout", "", csrfCookies, token)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || authService.logoutCall != 1 {
		t.Fatalf("logout = %d calls=%d", resp.StatusCode, authService.logoutCall)
	}

	token, csrfCookies = fetchCSRFToken(t, app, registerCookies...)
	resp = accountFlowJSON(t, app, fiber.MethodDelete, "/api/v1/account", "", csrfCookies, token)
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || deletions.userID != userID || body.Data["status"] != "pending" || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "" {
		t.Fatalf("deletion = %d body=%+v user=%s cookies=%+v", resp.StatusCode, body, deletions.userID, resp.Cookies())
	}
	deletedCookies := mergeCookies(registerCookies, resp.Cookies())
	resp = accountFlowJSON(t, app, fiber.MethodGet, "/api/v1/profile", "", deletedCookies, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("profile after deletion = %d, want unauthorized", resp.StatusCode)
	}
	resp = accountFlowJSON(t, app, fiber.MethodGet, "/api/v1/account/export", "", deletedCookies, "")
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("export after deletion = %d, want unauthorized", resp.StatusCode)
	}
}

func accountFlowJSON(t *testing.T, app *fiber.App, method string, target string, body string, cookies []*http.Cookie, csrfToken string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrfToken != "" {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}
	addCookies(req, cookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

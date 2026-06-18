package httpapi

// Implements DESIGN-010 RouteHandler, CSRFValidator, RateLimiter, RequestValidator, CORSHandler and DESIGN-017 GlobalExceptionHandler verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type auditSink struct{ entries []security.AuditLogEntry }

func (s *auditSink) Audit(_ context.Context, entry security.AuditLogEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

type failingAuditSink struct{}

func (failingAuditSink) Audit(context.Context, security.AuditLogEntry) error {
	return errors.New("audit down")
}

type failingObservabilitySink struct{}

func (failingObservabilitySink) Log(context.Context, observability.LogEvent) error {
	return errors.New("log down")
}

func (failingObservabilitySink) RecordMetric(context.Context, observability.MetricPoint) error {
	return errors.New("metric down")
}

type failingStorage struct{}

func (failingStorage) Get(string) ([]byte, error)              { return nil, errors.New("storage down") }
func (failingStorage) Set(string, []byte, time.Duration) error { return errors.New("storage down") }
func (failingStorage) Delete(string) error                     { return errors.New("storage down") }
func (failingStorage) Reset() error                            { return errors.New("storage down") }
func (failingStorage) Close() error                            { return errors.New("storage down") }

type httpSigningKeys struct {
	active  string
	entries map[string][]byte
}

func (k httpSigningKeys) ActiveSigningKey(context.Context) (string, []byte, error) {
	return k.active, k.entries[k.active], nil
}

func (k httpSigningKeys) SigningKey(_ context.Context, version string) ([]byte, error) {
	key, ok := k.entries[version]
	if !ok {
		return nil, errors.New("missing key")
	}
	return key, nil
}

type httpSessionRepository struct {
	byHash map[string]repository.UserSession
}

type staticAccessTokenValidator struct {
	claims auth.AccessTokenClaims
	err    error
}

func (v staticAccessTokenValidator) ValidateAccessToken(context.Context, string) (auth.AccessTokenClaims, error) {
	return v.claims, v.err
}

func (r *httpSessionRepository) CreateSession(context.Context, repository.UserSession) (uuid.UUID, error) {
	return uuid.Nil, errors.New("unused")
}

func (r *httpSessionRepository) GetSessionByRefreshTokenHash(_ context.Context, hash string) (repository.UserSession, error) {
	session, ok := r.byHash[hash]
	if !ok {
		return repository.UserSession{}, repository.NewError(repository.ErrorKindNotFound, "session not found", nil)
	}
	return session, nil
}

func (r *httpSessionRepository) RevokeSession(context.Context, uuid.UUID) error       { return nil }
func (r *httpSessionRepository) RevokeSessionFamily(context.Context, uuid.UUID) error { return nil }
func (r *httpSessionRepository) RevokeUserSessions(context.Context, uuid.UUID) error  { return nil }

func testJWTAuth(t *testing.T, cfg config.Config, userID uuid.UUID, mutate func(*repository.UserSession)) (*JWTAuthenticator, []*http.Cookie) {
	t.Helper()
	if userID == uuid.Nil {
		userID = uuid.New()
	}
	now := time.Now().UTC()
	sessionID := uuid.New()
	familyID := uuid.New()
	refreshToken := "refresh-" + uuid.NewString()
	session := repository.UserSession{ID: sessionID, UserID: userID, RefreshTokenHash: auth.HashRefreshToken(refreshToken), RefreshFamilyID: familyID, AccessExpiresAt: now.Add(15 * time.Minute), RefreshExpiresAt: now.Add(time.Hour)}
	if mutate != nil {
		mutate(&session)
	}
	manager := auth.NewJWTManager(httpSigningKeys{active: "jwt-v1", entries: map[string][]byte{"jwt-v1": []byte("11111111111111111111111111111111")}})
	accessToken, err := manager.CreateAccessToken(context.Background(), auth.AccessTokenClaims{UserID: userID, Role: "user", HasVerifiedLoginMethod: true, SessionID: sessionID, RefreshFamilyID: familyID, ExpiresAt: session.AccessExpiresAt})
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	repo := &httpSessionRepository{byHash: map[string]repository.UserSession{session.RefreshTokenHash: session}}
	authenticator := NewJWTAuthenticator(cfg, manager, repo)
	return authenticator, []*http.Cookie{
		{Name: cfg.Account.AccessCookieName, Value: accessToken, Path: "/"},
		{Name: cfg.Account.RefreshCookieName, Value: refreshToken, Path: "/"},
	}
}

type fakeAuthService struct {
	session       auth.AuthSession
	registerErr   error
	loginErr      error
	refreshErr    error
	logoutErr     error
	verifyErr     error
	resetReqErr   error
	resetUseErr   error
	resetToken    string
	registerCall  int
	loginCall     int
	refreshCall   int
	logoutCall    int
	lastRefresh   string
	verifiedUser  uuid.UUID
	resetEmail    string
	resetUseToken string
}

func (s *fakeAuthService) Register(context.Context, string, string, auth.RegistrationConsent) (auth.AuthSession, error) {
	s.registerCall++
	return s.session, s.registerErr
}

func (s *fakeAuthService) Login(context.Context, string, string) (auth.AuthSession, error) {
	s.loginCall++
	return s.session, s.loginErr
}

func (s *fakeAuthService) Refresh(_ context.Context, refreshToken string) (auth.AuthSession, error) {
	s.refreshCall++
	s.lastRefresh = refreshToken
	return s.session, s.refreshErr
}

func (s *fakeAuthService) Logout(_ context.Context, refreshToken string) error {
	s.logoutCall++
	s.lastRefresh = refreshToken
	return s.logoutErr
}

func (s *fakeAuthService) MarkEmailVerified(_ context.Context, userID uuid.UUID) error {
	s.verifiedUser = userID
	return s.verifyErr
}

func (s *fakeAuthService) RequestPasswordReset(_ context.Context, email string) (string, error) {
	s.resetEmail = email
	return s.resetToken, s.resetReqErr
}

func (s *fakeAuthService) ConsumePasswordReset(_ context.Context, token string, _ string) error {
	s.resetUseToken = token
	return s.resetUseErr
}

func testConfig() config.Config {
	return config.Config{FrontendOrigin: "http://localhost:5173", AllowedOrigins: []string{"http://localhost:5173"}, APITimeout: time.Second, HSTSMaxAge: 60, Account: config.AccountConfig{AccessCookieName: "__Host-test_access", RefreshCookieName: "__Host-test_refresh"}}
}

func mustNewRouter(t *testing.T, deps Dependencies) *fiber.App {
	t.Helper()
	app, err := NewRouter(deps)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	return app
}

func decodeEnvelope(t *testing.T, body any) Envelope {
	t.Helper()
	var envelope Envelope
	if err := json.NewDecoder(body.(interface{ Read([]byte) (int, error) })).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return envelope
}

func fetchCSRFToken(t *testing.T, app *fiber.App, cookies ...*http.Cookie) (string, []*http.Cookie) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/csrf-token", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := decodeEnvelope(t, resp.Body)
	token, ok := body.Data["csrfToken"].(string)
	if resp.StatusCode != fiber.StatusOK || !ok || token == "" {
		t.Fatalf("csrf token response = %d, %+v", resp.StatusCode, body)
	}
	return token, mergeCookies(cookies, resp.Cookies())
}

func TestRouterServesSimilarityIndicatorAssets(t *testing.T) {
	app := mustNewRouter(t, Dependencies{Config: testConfig()})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/assets/similarity/poor.svg", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("asset status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "image/svg+xml") {
		t.Fatalf("asset content type = %q", contentType)
	}
}

func mergeCookies(existing []*http.Cookie, updates []*http.Cookie) []*http.Cookie {
	merged := map[string]*http.Cookie{}
	for _, cookie := range existing {
		merged[cookie.Name] = cookie
	}
	for _, cookie := range updates {
		merged[cookie.Name] = cookie
	}
	result := make([]*http.Cookie, 0, len(merged))
	for _, cookie := range merged {
		result = append(result, cookie)
	}
	return result
}

func addCookies(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
}

func TestHealthAndReadiness(t *testing.T) {
	metrics := &observability.MemorySink{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Metrics: metrics, PostgresPing: func(context.Context) error { return nil }, RedisPing: func(context.Context) error { return errors.New("down") }})
	for _, path := range []string{"/health", "/api/v1/health"} {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		if err != nil || resp.StatusCode != fiber.StatusOK {
			t.Fatalf("GET %s = %v, %v", path, resp.StatusCode, err)
		}
		body := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		if body.RequestID == "" || resp.Header.Get("X-Frame-Options") != "DENY" {
			t.Fatalf("GET %s body/header = %+v, %v", path, body, resp.Header)
		}
	}
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable || len(metrics.Metrics) == 0 {
		t.Fatalf("ready status/metrics = %d/%d", resp.StatusCode, len(metrics.Metrics))
	}
}

func TestObservabilitySinkFailureUsesStderrFallback(t *testing.T) {
	var fallback bytes.Buffer
	previous := observabilityFallbackWriter
	observabilityFallbackWriter = &fallback
	t.Cleanup(func() { observabilityFallbackWriter = previous })

	sink := failingObservabilitySink{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Logs: sink, Metrics: sink})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if output := fallback.String(); !strings.Contains(output, "observability log sink failure: log down") || !strings.Contains(output, "observability metric sink failure: metric down") {
		t.Fatalf("fallback output = %q", output)
	}
}

func TestCORSTLSAndNotFoundErrors(t *testing.T) {
	cfg := testConfig()
	app := mustNewRouter(t, Dependencies{Config: cfg})
	req := httptest.NewRequest(fiber.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", cfg.FrontendOrigin)
	resp, _ := app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("preflight = %d, %v", resp.StatusCode, resp.Header)
	}
	req = httptest.NewRequest(fiber.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://bad.example")
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("forbidden origin = %d", resp.StatusCode)
	}
	cfg.EnforceTLS = true
	app = mustNewRouter(t, Dependencies{Config: cfg})
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusPermanentRedirect {
		t.Fatalf("redirect = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodGet, "/missing", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusPermanentRedirect {
		t.Fatalf("spoofed forwarded scheme = %d, %v", resp.StatusCode, resp.Header)
	}
	previousRequestIsTLS := requestIsTLS
	requestIsTLS = func(*fiber.Ctx) bool { return true }
	t.Cleanup(func() { requestIsTLS = previousRequestIsTLS })
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || resp.Header.Get("Strict-Transport-Security") == "" {
		t.Fatalf("https health = %d, %v", resp.StatusCode, resp.Header)
	}
}

func TestProtectedMutationValidationRateLimitAndLogging(t *testing.T) {
	audit := &auditSink{}
	sink := &observability.MemorySink{}
	rule := RateLimitRule{Scope: "endpoint", MaxRequests: 1, WindowSeconds: 60}
	cfg := testConfig()
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063"), nil)
	app := mustNewRouter(t, Dependencies{Config: cfg, Audit: audit, Logs: sink, Metrics: sink, Auth: authenticator, Routes: []RouteDefinition{{
		Method: fiber.MethodPost, Path: "/fixture", RequiresAuth: true, RequiresCSRF: true, RateLimit: &rule,
		Validate: ValidateJSON(func(body map[string]any) error {
			if body["name"] != "ok" {
				return errors.New("bad")
			}
			return nil
		}),
		Handler: func(ctx *fiber.Ctx) error { return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx)}) },
	}}})
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("unauthorized = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "invalid")
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("missing cookies with spoofed user header = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "2d4a5f20-c55f-4ba7-9751-779e682f7063")
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden || len(audit.entries) < 1 {
		t.Fatalf("csrf = %d audit=%d", resp.StatusCode, len(audit.entries))
	}
	if audit.entries[0].Resource != "/api/v1/fixture" {
		t.Fatalf("csrf audit resource = %q", audit.entries[0].Resource)
	}
	token, cookies := fetchCSRFToken(t, app)
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "spoofed")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("validation = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "spoofed")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || len(sink.Logs) == 0 {
		t.Fatalf("valid = %d logs=%d", resp.StatusCode, len(sink.Logs))
	}
	if got := sink.Logs[len(sink.Logs)-1].Fields["userId"]; got != "2d4a5f20-c55f-4ba7-9751-779e682f7063" {
		t.Fatalf("log user id = %v", got)
	}
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("limited = %d retry=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
}

func TestJWTProtectedRoutesRejectInvalidSessionStateAndSpoofedHeaders(t *testing.T) {
	cfg := testConfig()
	userID := uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063")
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: []RouteDefinition{{
		Method: fiber.MethodGet, Path: "/protected", RequiresAuth: true, RateLimit: &RateLimitRule{Scope: "user", MaxRequests: 1, WindowSeconds: 60},
		Handler: func(ctx *fiber.Ctx) error {
			user, ok := authenticatedUser(ctx)
			if !ok || user.UserID != userID || user.Role != "user" || !user.HasVerifiedLoginMethod {
				return errors.New("authenticated context mismatch")
			}
			return ctx.SendStatus(fiber.StatusNoContent)
		},
	}}})

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("X-Test-User-ID", uuid.NewString())
	addCookies(req, authCookies)
	resp, _ := app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("authenticated protected route = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("X-Test-User-ID", uuid.NewString())
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("user-scoped rate limit = %d", resp.StatusCode)
	}

	assertUnauthorized := func(name string, cookies []*http.Cookie, mutate func(*repository.UserSession)) {
		t.Helper()
		authenticator, validCookies := testJWTAuth(t, cfg, userID, mutate)
		testApp := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: []RouteDefinition{{
			Method: fiber.MethodGet, Path: "/" + name, RequiresAuth: true,
			Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
		}}})
		if cookies == nil {
			cookies = validCookies
		}
		req := httptest.NewRequest(fiber.MethodGet, "/api/v1/"+name, nil)
		addCookies(req, cookies)
		resp, _ := testApp.Test(req)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("%s status = %d, want unauthorized", name, resp.StatusCode)
		}
	}
	assertUnauthorized("missing", []*http.Cookie{}, nil)
	malformed := append([]*http.Cookie{}, authCookies...)
	malformed[0] = &http.Cookie{Name: cfg.Account.AccessCookieName, Value: "bad", Path: "/"}
	assertUnauthorized("malformed", malformed, nil)
	assertUnauthorized("revoked", nil, func(session *repository.UserSession) {
		now := time.Now()
		session.RevokedAt = &now
	})
	assertUnauthorized("expired", nil, func(session *repository.UserSession) {
		session.RefreshExpiresAt = time.Now().Add(-time.Minute)
	})
}

func TestSessionBoundCSRFLifecycleAndSPADelivery(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
	manager := NewCSRFManager(cfg, nil)
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: manager, Auth: authenticator, Routes: []RouteDefinition{
		{
			Method: fiber.MethodPost, Path: "/fixture", RequiresAuth: true, RequiresCSRF: true,
			Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
		},
		{
			Method: fiber.MethodGet, Path: "/rotate",
			Handler: func(ctx *fiber.Ctx) error { return manager.RegenerateAuthorizationState(ctx) },
		},
		{
			Method: fiber.MethodGet, Path: "/logout",
			Handler: func(ctx *fiber.Ctx) error { return manager.InvalidateAuthorizationState(ctx) },
		},
	}})
	token, cookies := fetchCSRFToken(t, app)
	if len(cookies) != 2 {
		t.Fatalf("csrf cookies = %+v", cookies)
	}
	for _, cookie := range cookies {
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("csrf cookie attributes = %+v", cookie)
		}
	}
	assertMutation := func(token string, cookies []*http.Cookie, want int) {
		t.Helper()
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", nil)
		req.Header.Set("X-Test-User-ID", "spoofed")
		req.Header.Set("X-CSRF-Token", token)
		addCookies(req, cookies)
		addCookies(req, authCookies)
		resp, _ := app.Test(req)
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("mutation = %d, want %d", resp.StatusCode, want)
		}
	}
	assertMutation(token, cookies, fiber.StatusNoContent)

	otherToken, otherCookies := fetchCSRFToken(t, app)
	assertMutation(token, otherCookies, fiber.StatusForbidden)
	assertMutation(otherToken, cookies, fiber.StatusForbidden)

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/rotate", nil)
	addCookies(req, cookies)
	resp, _ := app.Test(req)
	resp.Body.Close()
	cookies = mergeCookies(cookies, resp.Cookies())
	assertMutation(token, cookies, fiber.StatusForbidden)
	token, cookies = fetchCSRFToken(t, app, cookies...)
	assertMutation(token, cookies, fiber.StatusNoContent)

	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/logout", nil)
	addCookies(req, cookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	cookies = mergeCookies(cookies, resp.Cookies())
	assertMutation(token, cookies, fiber.StatusForbidden)
}

func TestCSRFUsesSecureCookiesWhenTLSIsEnforced(t *testing.T) {
	cfg := testConfig()
	cfg.EnforceTLS = true
	manager := NewCSRFManager(cfg, nil)
	app := fiber.New()
	app.Use(manager.IssueToken)
	app.Get("/", func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) })
	req := httptest.NewRequest(fiber.MethodGet, "https://example.test/", nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
	for _, cookie := range resp.Cookies() {
		if !cookie.Secure {
			t.Fatalf("cookie is not secure: %+v", cookie)
		}
	}
}

func TestAuthSessionCookiesRotateClearAndPreserveCSRFIsolation(t *testing.T) {
	cfg := testConfig()
	cfg.Account.AccessCookieName = "__Host-test_access"
	cfg.Account.RefreshCookieName = "__Host-test_refresh"
	manager := NewCSRFManager(cfg, nil)
	sessionManager := NewAuthSessionManager(cfg, manager)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	sessionManager.now = func() time.Time { return now }
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: manager, Routes: []RouteDefinition{
		{Method: fiber.MethodPost, Path: "/login", ExemptCSRF: true, Handler: func(ctx *fiber.Ctx) error {
			return sessionManager.SetAuthenticatedCookies(ctx, AuthSessionTokens{AccessToken: "access-1", RefreshToken: "refresh-1", AccessExpiresAt: now.Add(15 * time.Minute), RefreshExpiresAt: now.Add(7 * 24 * time.Hour)})
		}},
		{Method: fiber.MethodPost, Path: "/logout", ExemptCSRF: true, Handler: func(ctx *fiber.Ctx) error {
			return sessionManager.ClearAuthenticatedCookies(ctx)
		}},
		{Method: fiber.MethodPost, Path: "/protected", RequiresCSRF: true, Handler: func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(fiber.StatusNoContent)
		}},
	}})

	token, cookies := fetchCSRFToken(t, app)
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/login", nil)
	addCookies(req, cookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	cookies = mergeCookies(cookies, resp.Cookies())
	access := findCookie(cookies, cfg.Account.AccessCookieName)
	refresh := findCookie(cookies, cfg.Account.RefreshCookieName)
	if access == nil || refresh == nil || access.Value != "access-1" || refresh.Value != "refresh-1" {
		t.Fatalf("auth cookies = %+v", cookies)
	}
	for _, cookie := range []*http.Cookie{access, refresh} {
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.MaxAge <= 0 {
			t.Fatalf("auth cookie attributes = %+v", cookie)
		}
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/protected", nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("old csrf token after login = %d, want forbidden", resp.StatusCode)
	}

	token, cookies = fetchCSRFToken(t, app, cookies...)
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/protected", nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("new csrf token after login = %d", resp.StatusCode)
	}
	otherToken, _ := fetchCSRFToken(t, app)
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/protected", nil)
	req.Header.Set("X-CSRF-Token", otherToken)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("cross-session csrf = %d, want forbidden", resp.StatusCode)
	}

	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/logout", nil)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	access = findCookie(resp.Cookies(), cfg.Account.AccessCookieName)
	refresh = findCookie(resp.Cookies(), cfg.Account.RefreshCookieName)
	if access == nil || refresh == nil || access.Value != "" || refresh.Value != "" || !access.Expires.Before(now) || !refresh.Expires.Before(now) {
		t.Fatalf("cleared auth cookies = %+v", resp.Cookies())
	}
}

func TestAuthSessionCookiesAreSecureWhenTLSIsEnforced(t *testing.T) {
	cfg := testConfig()
	cfg.EnforceTLS = true
	cfg.Account.AccessCookieName = "__Host-test_access"
	cfg.Account.RefreshCookieName = "__Host-test_refresh"
	manager := NewCSRFManager(cfg, nil)
	sessionManager := NewAuthSessionManager(cfg, manager)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	sessionManager.now = func() time.Time { return now }
	previousRequestIsTLS := requestIsTLS
	requestIsTLS = func(*fiber.Ctx) bool { return true }
	t.Cleanup(func() { requestIsTLS = previousRequestIsTLS })
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: manager, Routes: []RouteDefinition{{Method: fiber.MethodPost, Path: "/login", ExemptCSRF: true, Handler: func(ctx *fiber.Ctx) error {
		return sessionManager.SetAuthenticatedCookies(ctx, AuthSessionTokens{AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour)})
	}}}})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/api/v1/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, cookie := range resp.Cookies() {
		if (cookie.Name == cfg.Account.AccessCookieName || cookie.Name == cfg.Account.RefreshCookieName) && !cookie.Secure {
			t.Fatalf("auth cookie is not secure: %+v", cookie)
		}
	}
}

func TestAuthControllerRegisterLoginRefreshLogout(t *testing.T) {
	cfg := testConfig()
	cfg.Account.AccessCookieName = "__Host-test_access"
	cfg.Account.RefreshCookieName = "__Host-test_refresh"
	manager := NewCSRFManager(cfg, nil)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
	userID := authenticator.sessions.(*httpSessionRepository).byHash[auth.HashRefreshToken(authCookies[1].Value)].UserID
	service := &fakeAuthService{session: auth.AuthSession{
		UserID:                 userID,
		AccessToken:            authCookies[0].Value,
		RefreshToken:           authCookies[1].Value,
		AccessExpiresAt:        now.Add(15 * time.Minute),
		RefreshExpiresAt:       now.Add(7 * 24 * time.Hour),
		HasVerifiedLoginMethod: false,
		Role:                   "user",
	}}
	sessionManager := NewAuthSessionManager(cfg, manager)
	sessionManager.now = func() time.Time { return now }
	controller := NewAuthController(service, sessionManager)
	audit := &auditSink{}
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: manager, Auth: authenticator, Audit: audit, Routes: controller.Routes()})

	spoofedUserID := uuid.New().String()
	registerReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"new@example.test","password":"StrongerPassword1!","privacyPolicyVersion":"privacy-v1","termsVersion":"terms-v1","userId":"`+spoofedUserID+`"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(registerReq)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated || body.Data["userId"] == "" || findCookie(resp.Cookies(), cfg.Account.AccessCookieName) == nil || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName) == nil {
		t.Fatalf("register response = %d body=%+v cookies=%+v", resp.StatusCode, body, resp.Cookies())
	}
	if body.Data["userId"] == spoofedUserID {
		t.Fatal("register accepted client-supplied user ID")
	}
	if body.Data["hasVerifiedLoginMethod"] != false {
		t.Fatalf("unverified login projection = %+v", body.Data)
	}
	if service.registerCall != 1 || len(audit.entries) == 0 {
		t.Fatalf("register service/audit = %d/%d", service.registerCall, len(audit.entries))
	}

	loginReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"new@example.test","password":"StrongerPassword1!"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(loginReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.loginCall != 1 {
		t.Fatalf("login response = %d calls=%d", resp.StatusCode, service.loginCall)
	}
	cookies := resp.Cookies()
	refreshCookie := findCookie(cookies, cfg.Account.RefreshCookieName)
	if refreshCookie == nil || refreshCookie.Value != authCookies[1].Value {
		t.Fatalf("login cookies = %+v", cookies)
	}

	refreshReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/refresh", nil)
	addCookies(refreshReq, cookies)
	resp, err = app.Test(refreshReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.refreshCall != 1 || service.lastRefresh != authCookies[1].Value || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != authCookies[1].Value {
		t.Fatalf("refresh response = %d service=%+v cookies=%+v", resp.StatusCode, service, resp.Cookies())
	}

	token, cookies := fetchCSRFToken(t, app, mergeCookies(cookies, resp.Cookies())...)
	logoutReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("X-CSRF-Token", token)
	addCookies(logoutReq, cookies)
	resp, err = app.Test(logoutReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || service.logoutCall != 1 || service.lastRefresh != authCookies[1].Value {
		t.Fatalf("logout response = %d service=%+v", resp.StatusCode, service)
	}
	if findCookie(resp.Cookies(), cfg.Account.AccessCookieName).Value != "" || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "" {
		t.Fatalf("logout cookies not cleared: %+v", resp.Cookies())
	}

	unauthLogoutReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/logout", nil)
	resp, err = app.Test(unauthLogoutReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || service.logoutCall != 1 {
		t.Fatalf("unauthenticated logout = %d calls=%d", resp.StatusCode, service.logoutCall)
	}

	logoutReq = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/logout", nil)
	addCookies(logoutReq, cookies)
	resp, err = app.Test(logoutReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("logout without csrf = %d, want forbidden", resp.StatusCode)
	}

	invalidRegisterReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"new@example.test","password":"StrongerPassword1!","privacyPolicyVersion":"bad version","termsVersion":"terms-v1"}`))
	invalidRegisterReq.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(invalidRegisterReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || service.registerCall != 1 {
		t.Fatalf("invalid register validation = %d calls=%d", resp.StatusCode, service.registerCall)
	}

	token, verifyCookies := fetchCSRFToken(t, app, cookies...)
	verifyReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/verify-email", nil)
	verifyReq.Header.Set("X-CSRF-Token", token)
	addCookies(verifyReq, mergeCookies(cookies, verifyCookies))
	resp, err = app.Test(verifyReq)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["hasVerifiedLoginMethod"] != true || service.verifiedUser != service.session.UserID {
		t.Fatalf("verify response = %d body=%+v verified=%s", resp.StatusCode, body, service.verifiedUser)
	}

	spoofVerifyReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/verify-email", strings.NewReader(`{"userId":"`+uuid.NewString()+`"}`))
	spoofVerifyReq.Header.Set("Content-Type", "application/json")
	spoofVerifyReq.Header.Set("X-CSRF-Token", token)
	addCookies(spoofVerifyReq, mergeCookies(cookies, verifyCookies))
	resp, err = app.Test(spoofVerifyReq)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.verifiedUser != service.session.UserID {
		t.Fatalf("verify trusted spoofed body = %d body=%+v verified=%s", resp.StatusCode, body, service.verifiedUser)
	}

	service.resetToken = "reset-token"
	resetReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/password-reset/request", strings.NewReader(`{"email":"new@example.test"}`))
	resetReq.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(resetReq)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["accepted"] != true || body.Data["resetToken"] != nil || service.resetEmail != "new@example.test" {
		t.Fatalf("reset request = %d body=%+v email=%q", resp.StatusCode, body, service.resetEmail)
	}

	consumeReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/password-reset/consume", strings.NewReader(`{"token":"reset-token","newPassword":"NewPassword1!"}`))
	consumeReq.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(consumeReq)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["reset"] != true || service.resetUseToken != "reset-token" || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "" {
		t.Fatalf("reset consume = %d body=%+v token=%q cookies=%+v", resp.StatusCode, body, service.resetUseToken, resp.Cookies())
	}
}

func TestAuthControllerFailures(t *testing.T) {
	cfg := testConfig()
	cfg.Account.AccessCookieName = "__Host-test_access"
	cfg.Account.RefreshCookieName = "__Host-test_refresh"
	manager := NewCSRFManager(cfg, nil)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	service := &fakeAuthService{session: auth.AuthSession{UserID: uuid.New(), AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour), Role: "user"}}
	sessionManager := NewAuthSessionManager(cfg, manager)
	sessionManager.now = func() time.Time { return now }
	controller := NewAuthController(service, sessionManager)
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: manager, Audit: &auditSink{}, Routes: controller.Routes()})

	service.loginErr = auth.ErrInvalidCredentials
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@example.test","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || body.Error == nil || body.Error.Code != "invalid_credentials" || strings.Contains(fmt.Sprint(body), "a@example.test") {
		t.Fatalf("invalid login response = %d %+v", resp.StatusCode, body)
	}

	service.loginErr = &auth.AccountLocked{RetryAfter: 15 * time.Minute}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@example.test","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests || resp.Header.Get("Retry-After") != "900" {
		t.Fatalf("locked login response = %d retry=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}

	service.registerErr = repository.NewError(repository.ErrorKindConflict, "duplicate email", nil)
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"a@example.test","password":"StrongerPassword1!","privacyPolicyVersion":"privacy-v1","termsVersion":"terms-v1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("duplicate register response = %d", resp.StatusCode)
	}

	service.refreshErr = auth.ErrTokenReuseDetected
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Account.RefreshCookieName, Value: "reused"})
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || body.Error == nil || body.Error.Code != "token_reuse_detected" || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "" {
		t.Fatalf("reuse response = %d body=%+v cookies=%+v", resp.StatusCode, body, resp.Cookies())
	}

	failingCSRF := NewCSRFManager(cfg, nil)
	failingCSRF.sessionStore.Storage = failingStorage{}
	failingSessionManager := NewAuthSessionManager(cfg, failingCSRF)
	warningSink := &observability.MemorySink{}
	service.refreshErr = auth.ErrTokenReuseDetected
	warningController := NewAuthController(service, failingSessionManager).WithLogSink(warningSink)
	warningApp := mustNewRouter(t, Dependencies{Config: cfg, CSRF: failingCSRF, Routes: warningController.Routes()})
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Account.RefreshCookieName, Value: "reused"})
	resp, err = warningApp.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || body.Error == nil || body.Error.Code != "token_reuse_detected" {
		t.Fatalf("cleanup failure masked refresh error = %d body=%+v", resp.StatusCode, body)
	}
	if len(warningSink.Logs) != 1 || warningSink.Logs[0].Level != "warning" || warningSink.Logs[0].Message != "auth_refresh_cookie_clear_failed" {
		t.Fatalf("cleanup warning logs = %+v", warningSink.Logs)
	}

	service.resetReqErr = nil
	service.resetToken = ""
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/password-reset/request", strings.NewReader(`{"email":"missing@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["accepted"] != true || body.Data["resetToken"] != nil {
		t.Fatalf("generic missing reset = %d %+v", resp.StatusCode, body)
	}
	service.resetUseErr = auth.ErrPasswordResetInvalid
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/password-reset/consume", strings.NewReader(`{"token":"used","newPassword":"NewPassword1!"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "password_reset_invalid" {
		t.Fatalf("used reset = %d %+v", resp.StatusCode, body)
	}

	unaudited := mustNewRouter(t, Dependencies{Config: cfg, CSRF: manager, Routes: controller.Routes()})
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"a@example.test","password":"StrongerPassword1!","privacyPolicyVersion":"privacy-v1","termsVersion":"terms-v1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = unaudited.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("required audit unavailable register = %d", resp.StatusCode)
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestAuthControllerValidatorsAndErrorMapping(t *testing.T) {
	invalidRegisterBodies := []map[string]any{
		{},
		{"email": "a@example.test"},
		{"email": "a@example.test", "password": "password"},
		{"email": "a@example.test", "password": "password", "privacyPolicyVersion": "p"},
		{"email": "a@example.test", "password": "password", "privacyPolicyVersion": "bad version", "termsVersion": "t"},
		{"email": "a@example.test", "password": "password", "privacyPolicyVersion": "p", "termsVersion": "bad version"},
	}
	for _, body := range invalidRegisterBodies {
		if err := validateRegisterBody(body); err == nil {
			t.Fatalf("validateRegisterBody() accepted %+v", body)
		}
	}
	for _, body := range []map[string]any{{}, {"email": "a@example.test"}} {
		if err := validateLoginBody(body); err == nil {
			t.Fatalf("validateLoginBody() accepted %+v", body)
		}
	}
	for _, body := range []map[string]any{{}, {"email": " "}} {
		if err := validatePasswordResetRequestBody(body); err == nil {
			t.Fatalf("validatePasswordResetRequestBody() accepted %+v", body)
		}
	}
	for _, body := range []map[string]any{{}, {"token": "token"}} {
		if err := validatePasswordResetConsumeBody(body); err == nil {
			t.Fatalf("validatePasswordResetConsumeBody() accepted %+v", body)
		}
	}

	cases := []struct {
		err  error
		code string
	}{
		{auth.ErrSessionExpired, "session_expired"},
		{auth.ErrTokenReuseDetected, "token_reuse_detected"},
		{auth.ErrPasswordResetInvalid, "password_reset_invalid"},
		{errors.New("consent_missing"), "consent_missing"},
		{errors.New("consent_version_stale"), "consent_version_stale"},
		{errors.New("consent_version_invalid"), "consent_version_invalid"},
	}
	for _, tc := range cases {
		mapped, ok := mapAuthError(nil, tc.err).(AppError)
		if !ok || mapped.Code != tc.code {
			t.Fatalf("mapAuthError(%v) = %#v", tc.err, mapped)
		}
	}
}

func TestAuthControllerBodyParserAndAuthenticationFailures(t *testing.T) {
	cfg := testConfig()
	controller := NewAuthController(&fakeAuthService{}, NewAuthSessionManager(cfg, NewCSRFManager(cfg, nil)))
	app := fiber.New()
	app.Post("/register", controller.Register)
	app.Post("/login", controller.Login)
	app.Post("/reset-request", controller.RequestPasswordReset)
	app.Post("/reset-consume", controller.ConsumePasswordReset)
	app.Post("/verify", controller.VerifyEmail)
	for _, path := range []string{"/register", "/login", "/reset-request", "/reset-consume"} {
		req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("%s malformed body = %d", path, resp.StatusCode)
		}
	}
	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/verify", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("verify unauthenticated = %d", resp.StatusCode)
	}
}

func TestJWTAuthenticatorAndSessionManagerFailures(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig()
	wantErr := errors.New("token failed")
	authenticator := NewJWTAuthenticator(cfg, staticAccessTokenValidator{err: wantErr}, &httpSessionRepository{})
	if _, err := authenticator.Authenticate(ctx, "access", "refresh"); !errors.Is(err, wantErr) {
		t.Fatalf("Authenticate() token error = %v", err)
	}
	claims := auth.AccessTokenClaims{UserID: uuid.New(), SessionID: uuid.New(), RefreshFamilyID: uuid.New()}
	authenticator = NewJWTAuthenticator(cfg, staticAccessTokenValidator{claims: claims}, &httpSessionRepository{byHash: map[string]repository.UserSession{}})
	if _, err := authenticator.Authenticate(ctx, "access", "refresh"); err == nil {
		t.Fatal("Authenticate() accepted missing session")
	}
	session := repository.UserSession{ID: uuid.New(), UserID: claims.UserID, RefreshFamilyID: claims.RefreshFamilyID, RefreshExpiresAt: time.Now().Add(time.Hour)}
	authenticator.sessions = &httpSessionRepository{byHash: map[string]repository.UserSession{auth.HashRefreshToken("refresh"): session}}
	if _, err := authenticator.Authenticate(ctx, "access", "refresh"); err == nil {
		t.Fatal("Authenticate() accepted mismatched claims")
	}

	app := fiber.New()
	app.Get("/required", requireAuth(nil), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/required", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("requireAuth(nil) = %d", resp.StatusCode)
	}

	manager := NewAuthSessionManager(cfg, nil)
	app = fiber.New()
	app.Get("/invalid", func(c *fiber.Ctx) error { return manager.SetAuthenticatedCookies(c, AuthSessionTokens{}) })
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/invalid", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("invalid session tokens = %d", resp.StatusCode)
	}
	failingCSRF := NewCSRFManager(cfg, nil)
	failingCSRF.sessionStore.Storage = failingStorage{}
	manager = NewAuthSessionManager(cfg, failingCSRF)
	now := time.Now()
	app = fiber.New()
	app.Get("/storage", func(c *fiber.Ctx) error {
		return manager.SetAuthenticatedCookies(c, AuthSessionTokens{AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour)})
	})
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/storage", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("session storage failure = %d", resp.StatusCode)
	}
}

func TestCSRFLifecycleReportsSessionStorageFailures(t *testing.T) {
	manager := NewCSRFManager(testConfig(), nil)
	manager.sessionStore.Storage = failingStorage{}
	app := fiber.New()
	app.Get("/rotate", func(ctx *fiber.Ctx) error { return manager.RegenerateAuthorizationState(ctx) })
	app.Get("/logout", func(ctx *fiber.Ctx) error { return manager.InvalidateAuthorizationState(ctx) })
	for _, path := range []string{"/rotate", "/logout"} {
		req := httptest.NewRequest(fiber.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "existing"})
		resp, _ := app.Test(req)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("GET %s = %d", path, resp.StatusCode)
		}
	}
	app = fiber.New()
	app.Get("/", csrfToken)
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("missing token = %d", resp.StatusCode)
	}
}

func TestTimeoutAndHelpers(t *testing.T) {
	cfg := testConfig()
	cfg.APITimeout = time.Millisecond
	app := mustNewRouter(t, Dependencies{Config: cfg, Routes: []RouteDefinition{{Method: fiber.MethodGet, Path: "/slow", Handler: func(ctx *fiber.Ctx) error {
		<-ctx.UserContext().Done()
		return nil
	}}}})
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/slow", nil), 100)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusGatewayTimeout {
		t.Fatalf("timeout = %d", resp.StatusCode)
	}
	if FailedLoginRule().MaxRequests != 10 {
		t.Fatal("helper baseline mismatch")
	}
}

func TestFailedLoginLockoutHTTPBehavior(t *testing.T) {
	audit := &auditSink{}
	rule := FailedLoginRule()
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Audit: audit, Routes: []RouteDefinition{{
		Method: fiber.MethodPost, Path: "/login", ExemptCSRF: true, RateLimit: &rule,
		Handler: func(ctx *fiber.Ctx) error {
			security.RecordAuditBestEffort(ctx.UserContext(), audit, security.AuditLogEntry{
				RequestID: requestID(ctx), Action: "auth.login", Resource: "/api/v1/login", Outcome: "failure",
				IP: ctx.IP(), UserAgent: ctx.Get("User-Agent"), CreatedAt: time.Now(),
			})
			if ctx.Get("X-Locked") == "true" {
				return AccountLockedError(ctx, 15*time.Minute)
			}
			return InvalidCredentialsError()
		},
	}}})

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/login", strings.NewReader(`{"email":"secret@example.test","password":"SecretPassword1!"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || body.Error == nil || body.Error.Code != "invalid_credentials" || body.Error.Message != "invalid email or password" {
		t.Fatalf("invalid credentials response = %d %+v", resp.StatusCode, body)
	}
	if len(audit.entries) == 0 {
		t.Fatal("missing failed-login audit")
	}
	for _, entry := range audit.entries {
		if strings.Contains(entry.Resource, "secret@example.test") || strings.Contains(entry.UserAgent, "SecretPassword1!") {
			t.Fatalf("audit leaked credential material: %+v", entry)
		}
	}

	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/login", nil)
	req.Header.Set("X-Locked", "true")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests || resp.Header.Get("Retry-After") != "900" || body.Error == nil || body.Error.Message != "invalid email or password" {
		t.Fatalf("locked response = %d retry=%q body=%+v", resp.StatusCode, resp.Header.Get("Retry-After"), body)
	}

	for i := 0; i < rule.MaxRequests; i++ {
		req = httptest.NewRequest(fiber.MethodPost, "/api/v1/login", nil)
		resp, err = app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/login", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("ip limiter response = %d retry=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
}

func TestCSRFClassificationAndLimiterScopes(t *testing.T) {
	if got := (AppError{Code: "x"}).Error(); got != "x" {
		t.Fatalf("Error() = %q", got)
	}
	classified := ClassifyServerError(AppError{Code: "x"})
	if classified.HTTPStatus != fiber.StatusInternalServerError {
		t.Fatalf("default status = %d", classified.HTTPStatus)
	}
	if got := ClassifyServerError(errors.New("secret")); got.Code != "internal_error" || got.Message != "internal server error" {
		t.Fatalf("unknown classification = %+v", got)
	}

	ctxApp := fiber.New()
	ctxApp.Get("/", func(ctx *fiber.Ctx) error {
		if requestID(ctx) != "" || routeTemplate(ctx) != "/" {
			t.Fatal("helper mismatch")
		}
		if rateLimitKey(ctx, "user") != "user:anonymous" || rateLimitKey(ctx, "endpoint") != "endpoint:/" {
			t.Fatal("rate scope mismatch")
		}
		userID := uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063")
		ctx.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: userID})
		if rateLimitKey(ctx, "user") != "user:"+userID.String() {
			t.Fatal("authenticated rate scope mismatch")
		}
		if auditedUserID, err := auditUserID(ctx); err != nil || auditedUserID == nil || *auditedUserID != userID {
			t.Fatal("authenticated audit user ID rejected")
		}
		ctx.Locals(authenticatedUserLocal, nil)
		if auditedUserID, err := auditUserID(ctx); err != nil || auditedUserID != nil {
			t.Fatal("missing audit user ID should be optional")
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})
	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	resp, _ := ctxApp.Test(req)
	resp.Body.Close()
}

func TestRouteTemplateFallsBackForUnmatchedContext(t *testing.T) {
	app := fiber.New()
	app.Use(func(ctx *fiber.Ctx) error {
		err := ctx.Next()
		if got := routeTemplate(ctx); got != "unmatched" {
			t.Fatalf("routeTemplate() = %q, want unmatched", got)
		}
		return err
	})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/missing", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestInvalidJSONAndReadySuccess(t *testing.T) {
	app := mustNewRouter(t, Dependencies{Config: testConfig(), PostgresPing: func(context.Context) error { return nil }, Routes: []RouteDefinition{{
		Method: fiber.MethodPost, Path: "/json", ExemptCSRF: true, Validate: ValidateJSON(func(map[string]any) error { return nil }), Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
	}}})
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("ready = %d", resp.StatusCode)
	}
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/json", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("invalid json = %d", resp.StatusCode)
	}
}

func TestRouterRejectsInvalidConfigAndLimitsByIP(t *testing.T) {
	for _, cfg := range []config.Config{
		{AllowedOrigins: []string{"http://localhost:5173"}},
		{APITimeout: time.Second},
	} {
		if _, err := NewRouter(Dependencies{Config: cfg}); err == nil {
			t.Fatalf("NewRouter() accepted invalid config: %+v", cfg)
		}
	}
	cfg := testConfig()
	app := mustNewRouter(t, Dependencies{Config: cfg, Routes: []RouteDefinition{{
		Method: fiber.MethodGet, Path: "/limited", RateLimit: &RateLimitRule{Scope: "ip", MaxRequests: 1, WindowSeconds: 60}, Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
	}}})
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/limited", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("first = %d", resp.StatusCode)
	}
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/limited", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("second = %d", resp.StatusCode)
	}
}

func TestRequiredMutationAuditFailsClosedAndReadsEmitCompletionAudit(t *testing.T) {
	called := false
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Audit: failingAuditSink{}, Routes: []RouteDefinition{{
		Method: fiber.MethodPost, Path: "/sensitive", ExemptCSRF: true, RequiresAudit: true,
		Handler: func(ctx *fiber.Ctx) error {
			called = true
			return ctx.SendStatus(fiber.StatusNoContent)
		},
	}}})
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodPost, "/api/v1/sensitive", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable || called {
		t.Fatalf("sensitive mutation = %d, called=%v", resp.StatusCode, called)
	}

	audit := &auditSink{}
	app = mustNewRouter(t, Dependencies{Config: testConfig(), Audit: audit, Routes: []RouteDefinition{{
		Method: fiber.MethodPost, Path: "/sensitive", ExemptCSRF: true, RequiresAudit: true,
		Handler: func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(fiber.StatusNoContent)
		},
	}}})
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/sensitive", nil)
	req.Header.Set("X-Test-User-ID", "invalid")
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("spoofed audit user header = %d", resp.StatusCode)
	}
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodPost, "/api/v1/sensitive", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || len(audit.entries) < 2 {
		t.Fatalf("audited mutation = %d, %+v", resp.StatusCode, audit.entries)
	}

	audit = &auditSink{}
	app = mustNewRouter(t, Dependencies{Config: testConfig(), Audit: audit})
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || len(audit.entries) != 1 || audit.entries[0].Action != "api.request" {
		t.Fatalf("read audit = %d, %+v", resp.StatusCode, audit.entries)
	}
}

func TestPanicRecoveryRetainsHeaders(t *testing.T) {
	sink := &observability.MemorySink{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Logs: sink, Metrics: sink, Routes: []RouteDefinition{{
		Method: fiber.MethodGet, Path: "/panic", Handler: func(*fiber.Ctx) error { panic("fixture") },
	}}})
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/panic", nil))
	defer resp.Body.Close()
	body := decodeEnvelope(t, resp.Body)
	if resp.StatusCode != fiber.StatusInternalServerError || body.Error == nil || body.Error.Message != "internal server error" || resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("panic response = %d, %+v, %v", resp.StatusCode, body, resp.Header)
	}
	if len(sink.Logs) != 1 || sink.Logs[0].Level != "error" {
		t.Fatalf("panic logs = %+v", sink.Logs)
	}
}

func TestQueryPathValidationAndProtectedMutationGuard(t *testing.T) {
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{
		{
			Method: fiber.MethodGet, Path: "/query",
			Validate: ValidateQuery(func(values map[string]string) error {
				if values["q"] != "ok" {
					return errors.New("bad query")
				}
				return nil
			}),
			Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
		},
		{
			Method: fiber.MethodGet, Path: "/fixture/:id",
			Validate: ValidatePath("id", func(value string) error {
				if value != "ok" {
					return errors.New("bad path")
				}
				return nil
			}),
			Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
		},
	}})
	for path, want := range map[string]int{"/api/v1/query?q=ok": fiber.StatusNoContent, "/api/v1/query?q=bad": fiber.StatusBadRequest, "/api/v1/fixture/ok": fiber.StatusNoContent, "/api/v1/fixture/bad": fiber.StatusBadRequest} {
		resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("GET %s = %d, want %d", path, resp.StatusCode, want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("missing protected mutation guard")
		}
	}()
	mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{{Method: fiber.MethodDelete, Path: "/unsafe", RequiresAuth: true, Handler: func(ctx *fiber.Ctx) error { return nil }}}})
}

func TestMutationRoutesRequireExactlyOneCSRFPolicy(t *testing.T) {
	for _, route := range []RouteDefinition{
		{Method: fiber.MethodPost, Path: "/missing", Handler: func(ctx *fiber.Ctx) error { return nil }},
		{Method: fiber.MethodPost, Path: "/contradictory", RequiresCSRF: true, ExemptCSRF: true, Handler: func(ctx *fiber.Ctx) error { return nil }},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("accepted mutation policy: %+v", route)
				}
			}()
			mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{route}})
		}()
	}
}

func TestLimiterResetRepositoryClassificationAndLogLevels(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: writeError})
	app.Use(rateLimitHandler(RateLimitRule{Scope: "ip", MaxRequests: 1, WindowSeconds: 1}))
	app.Get("/", func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) })
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	resp.Body.Close()
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("limited = %d", resp.StatusCode)
	}
	time.Sleep(1100 * time.Millisecond)
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("reset = %d", resp.StatusCode)
	}
	for err, code := range map[error]string{
		repository.NewError(repository.ErrorKindValidation, "internal", nil): "validation_failed",
		repository.NewError(repository.ErrorKindNotFound, "internal", nil):   "not_found",
		repository.NewError(repository.ErrorKindConflict, "internal", nil):   "conflict",
		repository.NewError(repository.ErrorKindConnection, "internal", nil): "dependency_unavailable",
	} {
		if got := ClassifyServerError(err); got.Code != code {
			t.Fatalf("ClassifyServerError(%v) = %+v", err, got)
		}
	}
	if logLevel(200) != "info" || logLevel(400) != "warn" || logLevel(500) != "error" || !isMutation(fiber.MethodPatch) || isMutation(fiber.MethodGet) {
		t.Fatal("severity or mutation mapping mismatch")
	}
	if got := ClassifyServerError(context.DeadlineExceeded); got.HTTPStatus != fiber.StatusGatewayTimeout {
		t.Fatalf("deadline classification = %+v", got)
	}
	if got := ClassifyServerError(fiber.NewError(fiber.StatusTeapot, "short and stout")); got.HTTPStatus != fiber.StatusTeapot {
		t.Fatalf("fiber classification = %+v", got)
	}
}

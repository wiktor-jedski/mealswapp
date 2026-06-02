package httpapi

// Implements DESIGN-010 RouteHandler, CSRFValidator, RateLimiter, RequestValidator, CORSHandler and DESIGN-017 GlobalExceptionHandler verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
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

func testConfig() config.Config {
	return config.Config{FrontendOrigin: "http://localhost:5173", AllowedOrigins: []string{"http://localhost:5173"}, APITimeout: time.Second, HSTSMaxAge: 60}
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
	cfg.EnforceTLS, cfg.TrustedProxy = true, true
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
	if resp.StatusCode != fiber.StatusNotFound || resp.Header.Get("Strict-Transport-Security") == "" {
		t.Fatalf("https not found = %d, %v", resp.StatusCode, resp.Header)
	}
}

func TestProtectedMutationValidationRateLimitAndLogging(t *testing.T) {
	audit := &auditSink{}
	sink := &observability.MemorySink{}
	rule := RateLimitRule{Scope: "endpoint", MaxRequests: 1, WindowSeconds: 60}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Audit: audit, Logs: sink, Metrics: sink, Routes: []RouteDefinition{{
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
		t.Fatalf("malformed user ID = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "2d4a5f20-c55f-4ba7-9751-779e682f7063")
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
	req.Header.Set("X-Test-User-ID", "2d4a5f20-c55f-4ba7-9751-779e682f7063")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("validation = %d", resp.StatusCode)
	}
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/fixture", strings.NewReader(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "2d4a5f20-c55f-4ba7-9751-779e682f7063")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || len(sink.Logs) == 0 {
		t.Fatalf("valid = %d logs=%d", resp.StatusCode, len(sink.Logs))
	}
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("limited = %d retry=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
}

func TestSessionBoundCSRFLifecycleAndSPADelivery(t *testing.T) {
	manager := NewCSRFManager(testConfig(), nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), CSRF: manager, Routes: []RouteDefinition{
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
		req.Header.Set("X-Test-User-ID", "2d4a5f20-c55f-4ba7-9751-779e682f7063")
		req.Header.Set("X-CSRF-Token", token)
		addCookies(req, cookies)
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
		if rateLimitKey(ctx, "user") != "user:u" || rateLimitKey(ctx, "endpoint") != "endpoint:/" {
			t.Fatal("rate scope mismatch")
		}
		ctx.Request().Header.Set("X-Test-User-ID", "2d4a5f20-c55f-4ba7-9751-779e682f7063")
		if userID, err := auditUserID(ctx); err != nil || userID == nil {
			t.Fatal("valid audit user ID rejected")
		}
		ctx.Request().Header.Set("X-Test-User-ID", "invalid")
		if userID, err := auditUserID(ctx); err == nil || userID != nil {
			t.Fatal("invalid audit user ID accepted")
		}
		ctx.Request().Header.Del("X-Test-User-ID")
		if userID, err := auditUserID(ctx); err != nil || userID != nil {
			t.Fatal("missing audit user ID should be optional")
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})
	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set("X-Test-User-ID", "u")
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
		Method: fiber.MethodPost, Path: "/json", Validate: ValidateJSON(func(map[string]any) error { return nil }), Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
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

func TestLimiterResetRepositoryClassificationAndLogLevels(t *testing.T) {
	now := time.Now()
	limiter := NewRateLimiter()
	limiter.now = func() time.Time { return now }
	app := fiber.New(fiber.Config{ErrorHandler: writeError})
	app.Use(limiter.Handler(RateLimitRule{Scope: "ip", MaxRequests: 1, WindowSeconds: 1}))
	app.Get("/", func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) })
	resp, _ := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	resp.Body.Close()
	resp, _ = app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("limited = %d", resp.StatusCode)
	}
	now = now.Add(2 * time.Second)
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
}

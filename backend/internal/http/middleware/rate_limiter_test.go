package middleware

import (
	"encoding/json"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func TestRateLimiterReturns429EnvelopeAndRetryMetadata(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	app := newRateLimitApp(RateLimiterConfig{
		Now: func() time.Time { return now },
		Rules: []RateLimitRule{
			{Name: "search", PathPrefix: "/api/v1/search", KeyScope: RateLimitKeyIP, MaxRequests: 2, Window: time.Minute},
		},
	})

	for i := 0; i < 2; i++ {
		res := performRateLimitRequest(t, app, http.MethodGet, "/api/v1/search")
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected allowed status 200, got %d", res.StatusCode)
		}
	}

	res := performRateLimitRequest(t, app, http.MethodGet, "/api/v1/search")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", res.StatusCode)
	}
	if got := res.Header.Get(fiber.HeaderRetryAfter); got != "60" {
		t.Fatalf("expected Retry-After 60, got %q", got)
	}

	var payload responses.Envelope
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Success || payload.Error == nil || payload.Error.Code != "rate_limited" || !payload.Error.Retryable {
		t.Fatalf("expected rate_limited retryable envelope, got %#v", payload)
	}
	fields, ok := payload.Error.Fields.(map[string]any)
	if !ok {
		t.Fatalf("expected retry metadata fields, got %#v", payload.Error.Fields)
	}
	if fields["retryAfterSeconds"] != float64(60) || fields["rule"] != "search" {
		t.Fatalf("unexpected retry metadata: %#v", fields)
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	app := newRateLimitApp(RateLimiterConfig{
		Now: func() time.Time { return now },
		Rules: []RateLimitRule{
			{Name: "auth-login", PathPrefix: "/api/v1/auth/login", KeyScope: RateLimitKeyIP, MaxRequests: 1, Window: 10 * time.Minute},
		},
	})

	res := performRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected initial request allowed, got %d", res.StatusCode)
	}

	blocked := performRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
	blocked.Body.Close()
	if blocked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected second request blocked, got %d", blocked.StatusCode)
	}

	now = now.Add(10*time.Minute + time.Second)
	allowed := performRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
	allowed.Body.Close()
	if allowed.StatusCode != http.StatusOK {
		t.Fatalf("expected request allowed after window reset, got %d", allowed.StatusCode)
	}
}

func TestRateLimiterUsesMostSpecificRuleAndUserScope(t *testing.T) {
	app := newRateLimitApp(RateLimiterConfig{
		Rules: []RateLimitRule{
			{Name: "api-default", PathPrefix: "/api/v1", KeyScope: RateLimitKeyIP, MaxRequests: 10, Window: time.Minute},
			{Name: "admin", PathPrefix: "/api/v1/admin", KeyScope: RateLimitKeyUser, MaxRequests: 1, Window: time.Minute},
		},
	})

	res := performRateLimitRequest(t, app, http.MethodGet, "/api/v1/admin/users")
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected first admin request allowed, got %d", res.StatusCode)
	}

	blocked := performRateLimitRequest(t, app, http.MethodGet, "/api/v1/admin/users")
	blocked.Body.Close()
	if blocked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected most specific admin rule to block second request, got %d", blocked.StatusCode)
	}
}

func TestRateLimiterExemptsHealthEndpoints(t *testing.T) {
	app := newRateLimitApp(RateLimiterConfig{
		ExemptPaths: []string{"/health"},
		Rules: []RateLimitRule{
			{Name: "all", PathPrefix: "/", KeyScope: RateLimitKeyIP, MaxRequests: 1, Window: time.Minute},
		},
	})

	for i := 0; i < 3; i++ {
		res := performRateLimitRequest(t, app, http.MethodGet, "/health")
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected health request %d allowed, got %d", i+1, res.StatusCode)
		}
	}
}

func newRateLimitApp(config RateLimiterConfig) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: func(ctx *fiber.Ctx, err error) error {
		appErr, ok := apperrors.As(err)
		if !ok {
			appErr = apperrors.Internal(err)
		}

		envelope := responses.Failure(appErr.Code, appErr.Message, requestID(ctx))
		envelope.Error.Category = string(appErr.Category)
		envelope.Error.Retryable = appErr.Retryable
		envelope.Error.Fields = appErr.Fields

		return ctx.Status(appErr.Status).JSON(envelope)
	}})
	app.Use(requestid.New())
	app.Use(func(ctx *fiber.Ctx) error {
		ctx.Locals("userID", "user-1")
		return ctx.Next()
	})
	app.Use(RateLimiter(config))
	app.All("/*", func(ctx *fiber.Ctx) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})
	return app
}

func requestID(ctx *fiber.Ctx) string {
	if value, ok := ctx.Locals("requestid").(string); ok {
		return value
	}
	return ctx.GetRespHeader("X-Request-ID")
}

func performRateLimitRequest(t *testing.T, app *fiber.App, method string, path string) *http.Response {
	t.Helper()

	req := newRequest(t, method, path)
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

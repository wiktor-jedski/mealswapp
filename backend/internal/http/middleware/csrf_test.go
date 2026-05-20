package middleware

import (
	"encoding/json"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func TestCSRFValidatorAllowsSafeMethodsOnProtectedPaths(t *testing.T) {
	app := newCSRFApp(DefaultCSRFConfig())

	res := performCSRFRequest(t, app, http.MethodGet, "/api/v1/profile", "", "")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected safe method allowed, got %d", res.StatusCode)
	}
}

func TestCSRFValidatorRejectsProtectedMutationWithoutToken(t *testing.T) {
	app := newCSRFApp(DefaultCSRFConfig())

	res := performCSRFRequest(t, app, http.MethodPost, "/api/v1/profile", "", "")
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", res.StatusCode)
	}

	var payload responses.Envelope
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Success || payload.Error == nil || payload.Error.Code != "csrf_failed" {
		t.Fatalf("expected csrf_failed envelope, got %#v", payload)
	}
}

func TestCSRFValidatorAllowsProtectedMutationWithMatchingToken(t *testing.T) {
	app := newCSRFApp(DefaultCSRFConfig())

	res := performCSRFRequest(t, app, http.MethodPatch, "/api/v1/profile", "token-123", "token-123")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected matching token allowed, got %d", res.StatusCode)
	}
}

func TestCSRFValidatorRejectsMismatchedToken(t *testing.T) {
	app := newCSRFApp(DefaultCSRFConfig())

	res := performCSRFRequest(t, app, http.MethodDelete, "/api/v1/account", "header-token", "cookie-token")
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected mismatched token rejected, got %d", res.StatusCode)
	}
}

func TestCSRFValidatorExemptsStripeWebhookRoute(t *testing.T) {
	app := newCSRFApp(DefaultCSRFConfig())

	res := performCSRFRequest(t, app, http.MethodPost, "/api/v1/webhooks/stripe", "", "")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected webhook route exempt, got %d", res.StatusCode)
	}
}

func TestCSRFValidatorProtectsSessionCookieMutationsOutsideConfiguredPrefixes(t *testing.T) {
	app := newCSRFApp(DefaultCSRFConfig())

	req := newRequest(t, http.MethodPost, "/api/v1/custom")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-1"})

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected session-cookie mutation rejected, got %d", res.StatusCode)
	}
}

func newCSRFApp(config CSRFConfig) *fiber.App {
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
	app.Use(CSRFValidator(config))
	app.All("/*", func(ctx *fiber.Ctx) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})
	return app
}

func performCSRFRequest(t *testing.T, app *fiber.App, method string, path string, headerToken string, cookieToken string) *http.Response {
	t.Helper()

	req := newRequest(t, method, path)
	if headerToken != "" {
		req.Header.Set("X-CSRF-Token", headerToken)
	}
	if cookieToken != "" {
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookieToken})
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

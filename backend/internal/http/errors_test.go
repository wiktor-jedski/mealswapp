package http

import (
	"errors"
	"net/http"
	"testing"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func TestGlobalExceptionHandlerFormatsExpectedErrorClasses(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		statusCode int
		code       string
		category   string
		retryable  bool
	}{
		{
			name:       "validation",
			err:        validation.ValidationError{Fields: []validation.FieldError{{Field: "name", Code: "required", Message: "name is required"}}},
			statusCode: http.StatusBadRequest,
			code:       "validation_error",
			category:   "validation",
		},
		{name: "auth", err: apperrors.Unauthorized("Unauthorized"), statusCode: http.StatusUnauthorized, code: "unauthorized", category: "auth"},
		{name: "not found", err: apperrors.NotFound("Route not found"), statusCode: http.StatusNotFound, code: "not_found", category: "unknown"},
		{name: "conflict", err: apperrors.Conflict("Conflict"), statusCode: http.StatusConflict, code: "conflict", category: "validation"},
		{name: "rate limit", err: apperrors.RateLimited("Too many requests"), statusCode: http.StatusTooManyRequests, code: "rate_limited", category: "dependency", retryable: true},
		{name: "dependency", err: apperrors.DependencyUnavailable("Service unavailable"), statusCode: http.StatusServiceUnavailable, code: "dependency_unavailable", category: "dependency", retryable: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := errorTestApp(tc.err)
			res := performRequest(t, app, http.MethodGet, "/boom")
			defer res.Body.Close()

			if res.StatusCode != tc.statusCode {
				t.Fatalf("expected %d, got %d", tc.statusCode, res.StatusCode)
			}

			payload := decodeEnvelope(t, res)
			assertErrorEnvelope(t, payload, tc.code, tc.category, tc.retryable)
			if payload.Meta == nil || payload.Meta.RequestID == "" || payload.Error.RequestID == "" {
				t.Fatalf("expected request IDs, got %#v", payload)
			}
		})
	}
}

func TestGlobalExceptionHandlerHidesInternalDetails(t *testing.T) {
	app := errorTestApp(errors.New("database password leaked in raw error"))

	res := performRequest(t, app, http.MethodGet, "/boom")
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.StatusCode)
	}

	payload := decodeEnvelope(t, res)
	assertErrorEnvelope(t, payload, "internal_error", "server", false)
	if payload.Error.Message != "Internal server error" {
		t.Fatalf("expected generic internal message, got %q", payload.Error.Message)
	}
}

func TestGlobalExceptionHandlerMapsFiberErrors(t *testing.T) {
	app := errorTestApp(fiber.NewError(fiber.StatusGatewayTimeout, "raw timeout"))

	res := performRequest(t, app, http.MethodGet, "/boom")
	defer res.Body.Close()

	if res.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", res.StatusCode)
	}

	payload := decodeEnvelope(t, res)
	assertErrorEnvelope(t, payload, "timeout", "timeout", true)
	if payload.Error.Message != "Request timed out" {
		t.Fatalf("expected safe timeout message, got %q", payload.Error.Message)
	}
}

func errorTestApp(err error) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: GlobalExceptionHandler})
	app.Use(requestid.New())
	app.Get("/boom", func(ctx *fiber.Ctx) error {
		return err
	})
	return app
}

func assertErrorEnvelope(t *testing.T, payload responses.Envelope, code string, category string, retryable bool) {
	t.Helper()

	if payload.Success {
		t.Fatal("expected failure envelope")
	}
	if payload.Error == nil {
		t.Fatal("expected error payload")
	}
	if payload.Error.Code != code || payload.Error.Category != category || payload.Error.Retryable != retryable {
		t.Fatalf("unexpected error payload: %#v", payload.Error)
	}
}

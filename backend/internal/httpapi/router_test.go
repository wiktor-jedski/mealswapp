package httpapi

// Implements DESIGN-010 RouteHandler and DESIGN-017 GlobalExceptionHandler HTTP verification.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/mealswapp/mealswapp/backend/internal/config"
)

// TestHealthReturnsEnvelope verifies DESIGN-010 RouteHandler health envelope behavior.
func TestHealthReturnsEnvelope(t *testing.T) {
	app := NewRouter(Dependencies{Config: config.Config{Environment: "test"}})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" || body.RequestID == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

// TestReadyReportsUnavailableDependency verifies DESIGN-010 RouteHandler readiness failure behavior.
func TestReadyReportsUnavailableDependency(t *testing.T) {
	app := NewRouter(Dependencies{
		Config: config.Config{Environment: "test"},
		PostgresPing: func(context.Context) error {
			return errors.New("down")
		},
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}
}

// TestReadyReportsAvailableDependencies verifies DESIGN-010 RouteHandler readiness success behavior.
func TestReadyReportsAvailableDependencies(t *testing.T) {
	app := NewRouter(Dependencies{
		Config: config.Config{Environment: "test"},
		PostgresPing: func(context.Context) error {
			return nil
		},
		RedisPing: func(context.Context) error {
			return nil
		},
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

// TestReadyReportsUnavailableRedis verifies DESIGN-010 RouteHandler Redis readiness failure behavior.
func TestReadyReportsUnavailableRedis(t *testing.T) {
	app := NewRouter(Dependencies{
		Config: config.Config{Environment: "test"},
		RedisPing: func(context.Context) error {
			return errors.New("down")
		},
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}
}

// TestPanicReturnsErrorEnvelope verifies DESIGN-017 GlobalExceptionHandler panic envelopes.
func TestPanicReturnsErrorEnvelope(t *testing.T) {
	app := NewRouter(Dependencies{Config: config.Config{Environment: "test"}})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/panic-test", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
}

// TestNotFoundReturnsFiberErrorEnvelope verifies DESIGN-017 GlobalExceptionHandler Fiber error envelopes.
func TestNotFoundReturnsFiberErrorEnvelope(t *testing.T) {
	app := NewRouter(Dependencies{Config: config.Config{Environment: "test"}})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/missing", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}

// TestRequestIDReturnsEmptyWhenUnset verifies DESIGN-010 RouteHandler request ID fallback behavior.
func TestRequestIDReturnsEmptyWhenUnset(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		if got := requestID(ctx); got != "" {
			t.Fatalf("requestID() = %q, want empty", got)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}
}

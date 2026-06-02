package app

// Implements DESIGN-010 RouteHandler app constructor verification.

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
)

// TestNewBuildsRouter proves that app router is built,
// /health is reachable and returns OK health response
// TestNewBuildsRouter verifies DESIGN-010 RouteHandler app constructor behavior.
func TestNewBuildsRouter(t *testing.T) {
	server, err := New(httpapi.Dependencies{Config: config.Config{APITimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173"}}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	resp, err := server.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

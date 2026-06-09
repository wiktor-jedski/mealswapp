package app

// Implements DESIGN-010 RouteHandler app constructor verification.

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
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

type fakeProductionPostgres struct{}

func (fakeProductionPostgres) Ping(context.Context) error { return nil }
func (fakeProductionPostgres) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("transaction not available")
}
func (fakeProductionPostgres) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("exec not available")
}
func (fakeProductionPostgres) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("query not available")
}
func (fakeProductionPostgres) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeProductionRow{}
}

type fakeProductionRow struct{}

func (fakeProductionRow) Scan(...any) error { return errors.New("row not available") }

// TestNewProductionExposesPhase03Routes verifies DESIGN-010 RouteHandler production composition.
func TestNewProductionExposesPhase03Routes(t *testing.T) {
	cfg := config.Config{
		APITimeout:     time.Second,
		AllowedOrigins: []string{"http://localhost:5173"},
		Environment:    "development",
		Account: config.AccountConfig{
			AccessTokenTTL:              15 * time.Minute,
			RefreshTokenTTL:             7 * 24 * time.Hour,
			AccessCookieName:            "__Host-test_access",
			RefreshCookieName:           "__Host-test_refresh",
			CurrentPrivacyPolicyVersion: "privacy-v1",
			CurrentTermsVersion:         "terms-v1",
		},
	}
	server, err := NewProduction(cfg, fakeProductionPostgres{}, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	checks := []struct {
		method string
		path   string
		body   string
	}{
		{fiber.MethodGet, "/api/v1/auth/csrf-token", ""},
		{fiber.MethodGet, "/api/v1/disclaimers?location=login", ""},
		{fiber.MethodGet, "/api/v1/auth/oauth/google/start", ""},
		{fiber.MethodPost, "/api/v1/auth/register", `{"bad":true}`},
		{fiber.MethodGet, "/api/v1/profile", ""},
		{fiber.MethodGet, "/api/v1/account/export", ""},
		{fiber.MethodDelete, "/api/v1/account", ""},
	}
	for _, check := range checks {
		req := httptest.NewRequest(check.method, check.path, strings.NewReader(check.body))
		if check.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := server.Test(req)
		if err != nil {
			t.Fatalf("%s %s error = %v", check.method, check.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == fiber.StatusNotFound {
			t.Fatalf("%s %s returned 404; route is not composed", check.method, check.path)
		}
	}
}

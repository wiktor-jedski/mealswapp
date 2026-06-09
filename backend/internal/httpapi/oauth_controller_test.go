package httpapi

// Implements DESIGN-006 OAuthHandler verification.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
)

type fakeOAuthGateway struct {
	startProvider    string
	startState       string
	callbackProvider string
	callbackQuery    map[string]string
	profile          auth.OAuthProfile
	startErr         error
	callbackErr      error
}

func (g *fakeOAuthGateway) StartOAuth(_ context.Context, provider string, state string) (string, error) {
	g.startProvider = provider
	g.startState = state
	return "https://oauth.example/" + provider + "?state=" + state, g.startErr
}

func (g *fakeOAuthGateway) CompleteOAuth(_ context.Context, provider string, query map[string]string) (auth.OAuthProfile, error) {
	g.callbackProvider = provider
	g.callbackQuery = query
	return g.profile, g.callbackErr
}

type fakeOAuthService struct {
	result   auth.OAuthResult
	err      error
	provider string
	profile  auth.OAuthProfile
}

func (s *fakeOAuthService) CompleteOAuth(_ context.Context, provider string, profile auth.OAuthProfile) (auth.OAuthResult, error) {
	s.provider = provider
	s.profile = profile
	return s.result, s.err
}

// TestOAuthControllerStartAndCallback verifies DESIGN-006 OAuthHandler HTTP behavior.
func TestOAuthControllerStartAndCallback(t *testing.T) {
	cfg := testConfig()
	cfg.Account.AccessCookieName = "__Host-test_access"
	cfg.Account.RefreshCookieName = "__Host-test_refresh"
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	sessionManager := NewAuthSessionManager(cfg, NewCSRFManager(cfg, nil))
	sessionManager.now = func() time.Time { return now }
	gateway := &fakeOAuthGateway{profile: auth.OAuthProfile{Provider: "google", ProviderUserID: "google-user-1", Email: "oauth@example.test", EmailVerified: true}}
	service := &fakeOAuthService{result: auth.OAuthResult{Session: auth.AuthSession{UserID: uuid.New(), AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour), HasVerifiedLoginMethod: true, Role: "user"}}}
	controller := NewOAuthController(service, gateway, sessionManager)
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: sessionManager.csrf, Routes: controller.Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/Google/start", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusFound || !strings.HasPrefix(resp.Header.Get("Location"), "https://oauth.example/google") || gateway.startProvider != "google" || gateway.startState == "" {
		t.Fatalf("start response = %d location=%q gateway=%#v", resp.StatusCode, resp.Header.Get("Location"), gateway)
	}

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback?code=abc&state=xyz", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["hasVerifiedLoginMethod"] != true || service.provider != "google" || service.profile.ProviderUserID != "google-user-1" {
		t.Fatalf("callback response = %d body=%+v service=%#v", resp.StatusCode, body, service)
	}
	if gateway.callbackQuery["code"] != "abc" || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "refresh" {
		t.Fatalf("callback query/cookies = %#v %+v", gateway.callbackQuery, resp.Cookies())
	}
}

// TestOAuthControllerFailures verifies DESIGN-006 OAuthHandler safe error mapping.
func TestOAuthControllerFailures(t *testing.T) {
	cfg := testConfig()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	sessionManager := NewAuthSessionManager(cfg, NewCSRFManager(cfg, nil))
	sessionManager.now = func() time.Time { return now }
	gateway := &fakeOAuthGateway{profile: auth.OAuthProfile{Provider: "google", ProviderUserID: "google-user-1", Email: "oauth@example.test"}}
	service := &fakeOAuthService{}
	controller := NewOAuthController(service, gateway, sessionManager)
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: sessionManager.csrf, Routes: controller.Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/github/start", nil))
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "validation_failed" {
		t.Fatalf("invalid provider response = %d body=%+v", resp.StatusCode, body)
	}

	service.err = auth.ErrOAuthProviderMismatch
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback", nil))
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "oauth_provider_mismatch" {
		t.Fatalf("mismatch response = %d body=%+v", resp.StatusCode, body)
	}

	service.err = &auth.OAuthLinkRequired{UserID: uuid.New()}
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback", nil))
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusConflict || body.Error == nil || body.Error.Code != "oauth_link_required" {
		t.Fatalf("link required response = %d body=%+v", resp.StatusCode, body)
	}

	gateway.callbackErr = errors.New("provider down")
	service.err = nil
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("provider error response = %d", resp.StatusCode)
	}
}

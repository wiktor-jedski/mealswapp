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
	stateCookie := findCookie(resp.Cookies(), oauthStateCookieName)
	returnCookie := findCookie(resp.Cookies(), oauthReturnCookieName)
	if resp.StatusCode != fiber.StatusFound || !strings.HasPrefix(resp.Header.Get("Location"), "https://oauth.example/google") || gateway.startProvider != "google" || gateway.startState == "" {
		t.Fatalf("start response = %d location=%q gateway=%#v", resp.StatusCode, resp.Header.Get("Location"), gateway)
	}
	if stateCookie.Value != gateway.startState || returnCookie.Value != "/" || gateway.startState == "" {
		t.Fatalf("state cookies = state:%+v return:%+v gateway=%#v", stateCookie, returnCookie, gateway)
	}

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback?code=abc&state="+gateway.startState, nil)
	req.AddCookie(stateCookie)
	req.AddCookie(returnCookie)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "http://localhost:5173/" || service.provider != "google" || service.profile.ProviderUserID != "google-user-1" {
		t.Fatalf("callback response = %d location=%q service=%#v", resp.StatusCode, resp.Header.Get("Location"), service)
	}
	if gateway.callbackQuery["code"] != "abc" || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "refresh" {
		t.Fatalf("callback query/cookies = %#v %+v", gateway.callbackQuery, resp.Cookies())
	}
	if !isClearedCookie(findLastCookie(resp.Cookies(), oauthStateCookieName)) || !isClearedCookie(findLastCookie(resp.Cookies(), oauthReturnCookieName)) {
		t.Fatalf("callback did not clear OAuth cookies: %+v", resp.Cookies())
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
	resp, err = app.Test(validOAuthCallbackRequest(t, "google", "state-mismatch"))
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "oauth_provider_mismatch" {
		t.Fatalf("mismatch response = %d body=%+v", resp.StatusCode, body)
	}

	service.err = &auth.OAuthLinkRequired{UserID: uuid.New()}
	resp, err = app.Test(validOAuthCallbackRequest(t, "google", "state-link"))
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
	resp, err = app.Test(validOAuthCallbackRequest(t, "google", "state-provider"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("provider error response = %d", resp.StatusCode)
	}

	gateway.callbackErr = ErrOAuthProviderUnavailable
	resp, err = app.Test(validOAuthCallbackRequest(t, "google", "state-unavailable"))
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || body.Error == nil || body.Error.Code != "oauth_provider_unavailable" {
		t.Fatalf("unavailable provider response = %d body=%+v", resp.StatusCode, body)
	}
}

// TestOAuthControllerStateAndReturnValidation verifies DESIGN-006 OAuthHandler callback CSRF handling.
func TestOAuthControllerStateAndReturnValidation(t *testing.T) {
	cfg := testConfig()
	sessionManager := NewAuthSessionManager(cfg, NewCSRFManager(cfg, nil))
	controller := NewOAuthController(&fakeOAuthService{}, &fakeOAuthGateway{}, sessionManager)
	app := mustNewRouter(t, Dependencies{Config: cfg, CSRF: sessionManager.csrf, Routes: controller.Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/start?return_to=/subscription?plan=annual", nil))
	if err != nil {
		t.Fatal(err)
	}
	stateCookie := findCookie(resp.Cookies(), oauthStateCookieName)
	returnCookie := findCookie(resp.Cookies(), oauthReturnCookieName)
	resp.Body.Close()
	if returnCookie.Value != "/subscription?plan=annual" || stateCookie.HttpOnly != true || stateCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("OAuth cookies = state:%+v return:%+v", stateCookie, returnCookie)
	}

	for name, target := range map[string]string{
		"absolute": "https://evil.test",
		"network":  "//evil.test/path",
		"relative": "subscription",
	} {
		t.Run(name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/start?return_to="+urlQueryEscape(target), nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if got := findCookie(resp.Cookies(), oauthReturnCookieName).Value; got != "/" {
				t.Fatalf("return cookie = %q, want /", got)
			}
		})
	}

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback?state=wrong", nil)
	req.AddCookie(stateCookie)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "oauth_state_invalid" {
		t.Fatalf("mismatched state response = %d body=%+v", resp.StatusCode, body)
	}
	if !isClearedCookie(findLastCookie(resp.Cookies(), oauthStateCookieName)) {
		t.Fatalf("mismatched state did not clear cookie: %+v", resp.Cookies())
	}

	for name, request := range map[string]*http.Request{
		"missing_state_cookie": httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback?state="+stateCookie.Value, nil),
		"missing_state_query": func() *http.Request {
			req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/google/callback", nil)
			req.AddCookie(stateCookie)
			return req
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			resp, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			body := decodeEnvelope(t, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "oauth_state_invalid" {
				t.Fatalf("missing state response = %d body=%+v", resp.StatusCode, body)
			}
			if !isClearedCookie(findLastCookie(resp.Cookies(), oauthStateCookieName)) || !isClearedCookie(findLastCookie(resp.Cookies(), oauthReturnCookieName)) {
				t.Fatalf("missing state did not clear OAuth cookies: %+v", resp.Cookies())
			}
		})
	}
}

func urlQueryEscape(value string) string {
	return strings.NewReplacer(":", "%3A", "/", "%2F", "?", "%3F").Replace(value)
}

func validOAuthCallbackRequest(t *testing.T, provider string, state string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/oauth/"+provider+"/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: state})
	return req
}

func findLastCookie(cookies []*http.Cookie, name string) *http.Cookie {
	var match *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == name {
			match = cookie
		}
	}
	return match
}

func isClearedCookie(cookie *http.Cookie) bool {
	return cookie != nil && cookie.Value == "" && cookie.Expires.Equal(time.Unix(0, 0))
}

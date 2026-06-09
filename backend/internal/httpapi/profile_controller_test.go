package httpapi

// Implements DESIGN-008 ProfileController verification.

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/profile"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type fakeProfileService struct {
	profile profile.UserProfile
	result  profile.UpdateResult
	err     error
	gotUser uuid.UUID
	gotReq  profile.UpdateRequest
}

func (s *fakeProfileService) GetProfile(_ context.Context, userID uuid.UUID) (profile.UserProfile, error) {
	s.gotUser = userID
	return s.profile, s.err
}

func (s *fakeProfileService) UpdatePreferences(_ context.Context, userID uuid.UUID, req profile.UpdateRequest) (profile.UpdateResult, error) {
	s.gotUser = userID
	s.gotReq = req
	return s.result, s.err
}

// TestProfileControllerAuthenticatedProfile verifies DESIGN-008 ProfileController HTTP behavior.
func TestProfileControllerAuthenticatedProfile(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	csrf := NewCSRFManager(cfg, nil)
	service := &fakeProfileService{
		profile: profile.UserProfile{UserID: userID, DisplayName: "Ada", UnitSystem: repository.UnitSystemMetric, ThemePreference: "system"},
		result:  profile.UpdateResult{Profile: profile.UserProfile{UserID: userID, DisplayName: "Ada", UnitSystem: repository.UnitSystemImperial, ThemePreference: "dark"}, RequiresUnitRecalculation: true},
	}
	controller := NewProfileController(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: csrf, Routes: controller.Routes()})

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/profile", nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["displayName"] != "Ada" || service.gotUser != userID {
		t.Fatalf("profile response = %d body=%+v user=%s", resp.StatusCode, body, service.gotUser)
	}

	token, csrfCookies := fetchCSRFToken(t, app)
	req = httptest.NewRequest(fiber.MethodPut, "/api/v1/profile", strings.NewReader(`{"displayName":"Ada","unitSystem":"imperial","themePreference":"dark"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["unitSystem"] != "imperial" || body.Data["requiresUnitRecalculation"] != true || service.gotReq.UnitSystem != repository.UnitSystemImperial {
		t.Fatalf("update response = %d body=%+v req=%#v", resp.StatusCode, body, service.gotReq)
	}
}

// TestProfileControllerRejectsCSRFAndCrossUserAccess verifies DESIGN-008 ProfileController guards.
func TestProfileControllerRejectsCSRFAndUndocumentedUserRoute(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeProfileService{profile: profile.UserProfile{UserID: userID, UnitSystem: repository.UnitSystemMetric, ThemePreference: "system"}}
	controller := NewProfileController(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})

	req := httptest.NewRequest(fiber.MethodPut, "/api/v1/profile", strings.NewReader(`{"unitSystem":"metric","themePreference":"system"}`))
	req.Header.Set("Content-Type", "application/json")
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("missing csrf update = %d", resp.StatusCode)
	}

	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/users/"+uuid.NewString()+"/profile", nil)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound || body.Error == nil || body.Error.Code != "Not Found" {
		t.Fatalf("undocumented user profile route response = %d body=%+v", resp.StatusCode, body)
	}

	service.err = errors.New("bad preference")
	token, csrfCookies := fetchCSRFToken(t, app)
	req = httptest.NewRequest(fiber.MethodPut, "/api/v1/profile", strings.NewReader(`{"unitSystem":"bad","themePreference":"system"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("invalid update = %d", resp.StatusCode)
	}
}

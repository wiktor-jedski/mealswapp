package httpapi

// Implements DESIGN-006 AuthController and DESIGN-008 authenticated controller defensive verification.

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
)

func TestAuthenticatedControllersRejectMissingPrincipalDirectly(t *testing.T) {
	cfg := testConfig()
	sessions := NewAuthSessionManager(cfg, nil)
	cases := []struct {
		method  string
		path    string
		handler fiber.Handler
	}{
		{fiber.MethodDelete, "/account", NewAccountDeletionController(&fakeAccountDeletionService{}, sessions).DeleteAccount},
		{fiber.MethodGet, "/export", NewExportController(&fakeExportService{}).ExportData},
		{fiber.MethodGet, "/profile", NewProfileController(&fakeProfileService{}).GetProfile},
		{fiber.MethodPut, "/profile-update", NewProfileController(&fakeProfileService{}).UpdatePreferences},
		{fiber.MethodGet, "/saved", NewUserDataController(&fakeUserDataService{}).ListSaved},
		{fiber.MethodDelete, "/saved/:kind/:itemId", NewUserDataController(&fakeUserDataService{}).DeleteSaved},
		{fiber.MethodGet, "/history", NewUserDataController(&fakeUserDataService{}).ListHistory},
		{fiber.MethodDelete, "/history", NewUserDataController(&fakeUserDataService{}).ClearHistory},
	}
	for _, tc := range cases {
		app := fiber.New()
		app.Add(tc.method, tc.path, tc.handler)
		path := tc.path
		if strings.Contains(path, ":kind") {
			path = "/saved/favorite/" + uuid.NewString()
		}
		resp, err := app.Test(httptest.NewRequest(tc.method, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("%s %s status = %d", tc.method, path, resp.StatusCode)
		}
	}
}

func TestControllersDefendAgainstBypassedValidation(t *testing.T) {
	user := AuthenticatedUser{UserID: uuid.New()}
	withUser := func(handler fiber.Handler) fiber.Handler {
		return func(ctx *fiber.Ctx) error {
			ctx.Locals(authenticatedUserLocal, user)
			return handler(ctx)
		}
	}

	profileController := NewProfileController(&fakeProfileService{})
	app := fiber.New()
	app.Put("/profile", withUser(profileController.UpdatePreferences))
	req := httptest.NewRequest(fiber.MethodPut, "/profile", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("profile malformed body = %d", resp.StatusCode)
	}

	userdataController := NewUserDataController(&fakeUserDataService{})
	app = fiber.New()
	app.Get("/saved", withUser(userdataController.ListSaved))
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/saved?kind=bad", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("saved invalid kind = %d", resp.StatusCode)
	}
	app = fiber.New()
	app.Delete("/saved/:kind/:itemId", withUser(userdataController.DeleteSaved))
	resp, err = app.Test(httptest.NewRequest(fiber.MethodDelete, "/saved/favorite/bad", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("saved invalid id = %d", resp.StatusCode)
	}
}

func TestControllerValidationHelpers(t *testing.T) {
	if err := validateExportQuery(map[string]string{}); err != nil {
		t.Fatalf("default export format error = %v", err)
	}
	if err := validateExportQuery(map[string]string{"format": "xml"}); err == nil {
		t.Fatal("validateExportQuery() accepted xml")
	}
	for _, body := range []map[string]any{
		{"unitSystem": "metric"},
		{"unitSystem": "metric", "themePreference": "bad"},
		{"unitSystem": "metric", "themePreference": "system", "displayName": 1},
	} {
		if err := validateProfilePreferenceBody(body); err == nil {
			t.Fatalf("validateProfilePreferenceBody() accepted %+v", body)
		}
	}
	if err := validateSavedItemsQuery(map[string]string{"kind": "bad"}); err == nil {
		t.Fatal("validateSavedItemsQuery() accepted invalid kind")
	}

	app := fiber.New()
	app.Delete("/saved/:kind/:itemId", validateDeleteSavedPath(), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	for _, path := range []string{"/saved/bad/" + uuid.NewString(), "/saved/favorite/bad"} {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodDelete, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("%s status = %d", path, resp.StatusCode)
		}
	}
}

func TestOAuthControllerDirectFailures(t *testing.T) {
	cfg := testConfig()
	sessions := NewAuthSessionManager(cfg, nil)
	gateway := &fakeOAuthGateway{startErr: errors.New("provider failed")}
	service := &fakeOAuthService{result: auth.OAuthResult{}}
	controller := NewOAuthController(service, gateway, sessions)
	app := fiber.New()
	app.Get("/start/:provider", controller.StartOAuth)
	app.Get("/callback/:provider", controller.CompleteOAuth)
	for _, path := range []string{"/start/github", "/callback/github", "/start/google", "/callback/google"} {
		if strings.Contains(path, "callback/google") {
			gateway.startErr = nil
			gateway.callbackErr = nil
		} else if strings.Contains(path, "callback") {
			gateway.callbackErr = errors.New("provider failed")
		}
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("%s status = %d", path, resp.StatusCode)
		}
	}
	if mapped, ok := mapOAuthError(auth.ErrOAuthProviderMismatch).(AppError); !ok || mapped.Code != "oauth_provider_mismatch" {
		t.Fatalf("mapOAuthError() = %#v", mapped)
	}
	if _, err := savedKindQuery(""); err != nil {
		t.Fatalf("savedKindQuery(empty) = %v", err)
	}
}

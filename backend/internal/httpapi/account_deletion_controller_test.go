package httpapi

// Implements DESIGN-008 AccountDeleter verification.

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type fakeAccountDeletionService struct {
	request repository.DataDeletionRequest
	userID  uuid.UUID
}

func (s *fakeAccountDeletionService) RequestDeletion(_ context.Context, userID uuid.UUID) (repository.DataDeletionRequest, error) {
	s.userID = userID
	return s.request, nil
}

// TestAccountDeletionController verifies DESIGN-008 AccountDeleter HTTP behavior.
func TestAccountDeletionController(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	csrf := NewCSRFManager(cfg, nil)
	sessionManager := NewAuthSessionManager(cfg, csrf)
	service := &fakeAccountDeletionService{request: repository.DataDeletionRequest{ID: uuid.New(), UserID: userID, Status: "pending"}}
	controller := NewAccountDeletionController(service, sessionManager)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: csrf, Audit: &auditSink{}, Routes: controller.Routes()})

	req := httptest.NewRequest(fiber.MethodDelete, "/api/v1/account", nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("delete without csrf = %d", resp.StatusCode)
	}

	token, csrfCookies := fetchCSRFToken(t, app)
	req = httptest.NewRequest(fiber.MethodDelete, "/api/v1/account", nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["status"] != "pending" || service.userID != userID || findCookie(resp.Cookies(), cfg.Account.RefreshCookieName).Value != "" {
		t.Fatalf("delete response = %d body=%+v user=%s cookies=%+v", resp.StatusCode, body, service.userID, resp.Cookies())
	}
}

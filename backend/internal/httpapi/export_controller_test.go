package httpapi

// Implements DESIGN-008 DataExporter verification.

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

type fakeExportService struct {
	payload userdata.ExportPayload
	err     error
	userID  uuid.UUID
	format  string
}

func (s *fakeExportService) BuildExport(_ context.Context, userID uuid.UUID, format string) (userdata.ExportPayload, error) {
	s.userID = userID
	s.format = format
	return s.payload, s.err
}

// TestExportControllerRequiresAuthAndReturnsPayload verifies DESIGN-008 DataExporter HTTP behavior.
func TestExportControllerRequiresAuthAndReturnsPayload(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	sink := &observability.MemorySink{}
	service := &fakeExportService{payload: userdata.ExportPayload{Format: "json", ContentType: "application/json", Filename: "export.json", Body: []byte(`{"email":"ada@example.test"}`)}}
	controller := NewExportController(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Logs: sink, Routes: controller.Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/account/export?format=json", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("unauthenticated export = %d", resp.StatusCode)
	}

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/account/export?format=csv", nil)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.userID != userID || service.format != "csv" || string(body) != `{"email":"ada@example.test"}` {
		t.Fatalf("export response = %d body=%q user=%s format=%s", resp.StatusCode, string(body), service.userID, service.format)
	}
	for _, log := range sink.Logs {
		for _, value := range log.Fields {
			if value == "ada@example.test" {
				t.Fatal("plaintext export PII was written to log fields")
			}
		}
	}
}

// TestExportControllerRejectsUnsupportedFormat verifies DESIGN-008 DataExporter validation errors.
func TestExportControllerRejectsUnsupportedFormat(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
	service := &fakeExportService{err: errors.New("export format is unsupported")}
	controller := NewExportController(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: controller.Routes()})
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/account/export?format=xml", nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "validation_failed" {
		t.Fatalf("unsupported format response = %d body=%+v", resp.StatusCode, body)
	}
}

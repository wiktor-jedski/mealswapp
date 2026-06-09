package httpapi

// Implements DESIGN-015 DisclaimerRenderer verification.

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/compliance"
)

type fakeDisclaimerService struct {
	content  compliance.DisclaimerContent
	location string
	err      error
}

func (s *fakeDisclaimerService) GetDisclaimer(_ context.Context, location string) (compliance.DisclaimerContent, error) {
	s.location = location
	return s.content, s.err
}

// TestDisclaimerController verifies DESIGN-015 DisclaimerRenderer HTTP behavior.
func TestDisclaimerController(t *testing.T) {
	service := &fakeDisclaimerService{content: compliance.DisclaimerContent{Location: "login", Version: "fallback-v1", Markdown: "Disclaimer", Fallback: true, Alert: "configured_disclaimer_unavailable"}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewDisclaimerController(service).Routes()})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/disclaimers?location=login", nil))
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["markdown"] != "Disclaimer" || body.Data["fallback"] != true || resp.Header.Get("Cache-Control") == "" || service.location != "login" {
		t.Fatalf("disclaimer response = %d body=%+v cache=%q location=%q", resp.StatusCode, body, resp.Header.Get("Cache-Control"), service.location)
	}
}

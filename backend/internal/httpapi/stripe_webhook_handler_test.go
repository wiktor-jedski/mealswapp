package httpapi

// Implements DESIGN-007 StripeWebhookHandler HTTP verification.

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

type fakeStripeWebhookProcessor struct {
	result subscription.WebhookResult
	err    error
	req    subscription.WebhookRequest
}

func (p *fakeStripeWebhookProcessor) HandleWebhook(_ context.Context, req subscription.WebhookRequest) (subscription.WebhookResult, error) {
	p.req = req
	return p.result, p.err
}

func TestStripeWebhookHandlerReturns400AndAuditsInvalidSignature(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	cfg := testConfig()
	audit := &auditSink{}
	processor := &fakeStripeWebhookProcessor{err: subscription.ErrWebhookInvalidSignature}
	app := mustNewRouter(t, Dependencies{Config: cfg, Audit: audit, Routes: NewStripeWebhookHandler(processor, audit).Routes()})

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/stripe/webhook", bytes.NewBufferString(`{"id":"evt"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	found := false
	for _, entry := range audit.entries {
		if entry.Action == "stripe_webhook_signature_invalid" && entry.Outcome == "blocked" {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit entries = %#v, want invalid-signature security event", audit.entries)
	}
}

func TestStripeWebhookHandlerReturns200ForDuplicateAndPassesRawProviderFields(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	processor := &fakeStripeWebhookProcessor{result: subscription.WebhookResult{EventID: "evt_duplicate", EventType: "checkout.session.completed", Duplicate: true}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewStripeWebhookHandler(processor, nil).Routes()})

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/stripe/webhook", bytes.NewBufferString(`{"id":"evt_duplicate"}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=abc")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if processor.req.Signature != "t=1,v1=abc" || string(processor.req.Payload) != `{"id":"evt_duplicate"}` {
		t.Fatalf("processor request = %#v, want raw body and signature", processor.req)
	}
}

func TestStripeWebhookHandlerReturns500ForWriteFailureSoStripeRetries(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	processor := &fakeStripeWebhookProcessor{err: errors.Join(subscription.ErrWebhookStoreFailed, errors.New("db failed"))}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewStripeWebhookHandler(processor, nil).Routes()})

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/stripe/webhook", bytes.NewBufferString(`{"id":"evt_retry"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

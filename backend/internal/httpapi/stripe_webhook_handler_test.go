// Implements DESIGN-007 StripeWebhookHandler.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type mockWebhookEntitlementRepo struct {
	repository.EntitlementRepository
	appended        []repository.Entitlement
	insertedEvent   bool
	insertEventErr  error
	customerEnts    map[string]repository.Entitlement
	subEnts         map[string]repository.Entitlement
	duplicateEvents map[string]bool
}

func (m *mockWebhookEntitlementRepo) AppendEntitlement(ctx context.Context, ent repository.Entitlement) error {
	m.appended = append(m.appended, ent)
	if ent.StripeCustomerID != "" {
		m.customerEnts[ent.StripeCustomerID] = ent
	}
	if ent.StripeSubscriptionID != "" {
		m.subEnts[ent.StripeSubscriptionID] = ent
	}
	return nil
}

func (m *mockWebhookEntitlementRepo) GetLatestByStripeCustomer(ctx context.Context, customerID string) (repository.Entitlement, error) {
	if ent, ok := m.customerEnts[customerID]; ok {
		return ent, nil
	}
	return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
}

func (m *mockWebhookEntitlementRepo) GetLatestByStripeSubscription(ctx context.Context, subscriptionID string) (repository.Entitlement, error) {
	if ent, ok := m.subEnts[subscriptionID]; ok {
		return ent, nil
	}
	return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
}

func (m *mockWebhookEntitlementRepo) InsertProcessedStripeEvent(ctx context.Context, event repository.ProcessedStripeEvent) (bool, error) {
	if m.insertEventErr != nil {
		return false, m.insertEventErr
	}
	if m.duplicateEvents[event.EventID] {
		return false, nil
	}
	m.duplicateEvents[event.EventID] = true
	m.insertedEvent = true
	return true, nil
}

type mockAuditLogger struct {
	logged []security.AuditLogEntry
}

func (m *mockAuditLogger) Audit(ctx context.Context, entry security.AuditLogEntry) error {
	m.logged = append(m.logged, entry)
	return nil
}

func setupWebhookTestApp(m *mockWebhookEntitlementRepo, a *mockAuditLogger) *fiber.App {
	cfg := config.Config{
		Billing: config.BillingConfig{
			StripeWebhookSecret: "whsec_test",
		},
	}
	handler := NewStripeWebhookHandler(cfg, m, m, a)

	app := fiber.New()
	for _, route := range handler.Routes() {
		app.Add(route.Method, route.Path, route.Handler)
	}
	return app
}

func createSignedRequest(payload []byte, secret string) *http.Request {
	req := httptest.NewRequest("POST", "/billing/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	// Create stripe signature
	// format: t=<timestamp>,v1=<signature>
	now := time.Now()
	t := now.Unix()
	mac := webhook.ComputeSignature(now, payload, secret)
	sigHeader := fmt.Sprintf("t=%d,v1=%x", t, mac)

	req.Header.Set("Stripe-Signature", sigHeader)
	return req
}

func TestWebhook_InvalidSignature(t *testing.T) {
	repo := &mockWebhookEntitlementRepo{}
	audit := &mockAuditLogger{}
	app := setupWebhookTestApp(repo, audit)

	req := httptest.NewRequest("POST", "/billing/webhook", bytes.NewReader([]byte("{}")))
	req.Header.Set("Stripe-Signature", "t=1,v1=invalid")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400 for invalid signature, got %d", resp.StatusCode)
	}

	if len(audit.logged) == 0 || audit.logged[0].Outcome != "failure" {
		t.Errorf("Expected security audit event logged as failure")
	}
}

func TestWebhook_DuplicateEvent(t *testing.T) {
	repo := &mockWebhookEntitlementRepo{
		duplicateEvents: map[string]bool{"evt_123": true},
	}
	app := setupWebhookTestApp(repo, &mockAuditLogger{})

	event := stripe.Event{
		ID:   "evt_123",
		Type: "checkout.session.completed",
	}
	payload, _ := json.Marshal(event)
	req := createSignedRequest(payload, "whsec_test")

	resp, _ := app.Test(req)

	if true {
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200 for duplicate event, got %d", resp.StatusCode)
	}
	if len(repo.appended) > 0 {
		t.Errorf("Expected no entitlements appended on duplicate")
	}
}

func TestWebhook_CheckoutSessionCompleted(t *testing.T) {
	repo := &mockWebhookEntitlementRepo{
		duplicateEvents: map[string]bool{},
		customerEnts:    map[string]repository.Entitlement{},
		subEnts:         map[string]repository.Entitlement{},
	}
	app := setupWebhookTestApp(repo, &mockAuditLogger{})

	userID := uuid.New()

	session := stripe.CheckoutSession{
		ClientReferenceID: userID.String(),
		Mode:              stripe.CheckoutSessionModeSubscription,
		Customer:          &stripe.Customer{ID: "cus_123"},
		Subscription:      &stripe.Subscription{ID: "sub_123"},
	}

	rawSession, _ := json.Marshal(session)
	event := stripe.Event{
		ID:   "evt_1",
		Type: "checkout.session.completed",
		Data: &stripe.EventData{Raw: rawSession},
	}

	payload, _ := json.Marshal(event)
	req := createSignedRequest(payload, "whsec_test")

	resp, _ := app.Test(req)
	if true {
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	if len(repo.appended) != 1 {
		t.Fatalf("Expected 1 entitlement appended, got %d", len(repo.appended))
	}

	ent := repo.appended[0]
	if ent.UserID != userID || ent.Tier != "paid" || ent.Status != "active" {
		t.Errorf("Expected paid active entitlement for user, got %s %s", ent.Tier, ent.Status)
	}
	if ent.StripeCustomerID != "cus_123" || ent.StripeSubscriptionID != "sub_123" {
		t.Errorf("Expected stripe ids mapped correctly")
	}
}

func TestWebhook_InvoiceFailed(t *testing.T) {
	userID := uuid.New()
	repo := &mockWebhookEntitlementRepo{
		duplicateEvents: map[string]bool{},
		customerEnts:    map[string]repository.Entitlement{},
		subEnts: map[string]repository.Entitlement{
			"sub_456": {UserID: userID, StripeCustomerID: "cus_456", StripeSubscriptionID: "sub_456"},
		},
	}
	app := setupWebhookTestApp(repo, &mockAuditLogger{})

	invoice := stripe.Invoice{
		Customer:     &stripe.Customer{ID: "cus_456"},
		Subscription: &stripe.Subscription{ID: "sub_456"},
	}
	rawInvoice, _ := json.Marshal(invoice)

	event := stripe.Event{
		ID:   "evt_2",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{Raw: rawInvoice},
	}

	payload, _ := json.Marshal(event)
	req := createSignedRequest(payload, "whsec_test")

	resp, _ := app.Test(req)
	if true {
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	if len(repo.appended) != 1 {
		t.Fatalf("Expected 1 entitlement appended, got %d", len(repo.appended))
	}

	ent := repo.appended[0]
	if ent.Tier != "paid" || ent.Status != "past_due" {
		t.Errorf("Expected paid past_due entitlement, got %s %s", ent.Tier, ent.Status)
	}
}

func TestWebhook_SubscriptionCancelled(t *testing.T) {
	userID := uuid.New()
	repo := &mockWebhookEntitlementRepo{
		duplicateEvents: map[string]bool{},
		customerEnts:    map[string]repository.Entitlement{},
		subEnts: map[string]repository.Entitlement{
			"sub_789": {UserID: userID, StripeCustomerID: "cus_789", StripeSubscriptionID: "sub_789"},
		},
	}
	app := setupWebhookTestApp(repo, &mockAuditLogger{})

	sub := stripe.Subscription{
		ID:       "sub_789",
		Customer: &stripe.Customer{ID: "cus_789"},
	}
	rawSub, _ := json.Marshal(sub)

	event := stripe.Event{
		ID:   "evt_3",
		Type: "customer.subscription.canceled",
		Data: &stripe.EventData{Raw: rawSub},
	}

	payload, _ := json.Marshal(event)
	req := createSignedRequest(payload, "whsec_test")

	resp, _ := app.Test(req)
	if true {
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	if len(repo.appended) != 1 {
		t.Fatalf("Expected 1 entitlement appended, got %d", len(repo.appended))
	}

	ent := repo.appended[0]
	if ent.Tier != "paid" || ent.Status != "cancelled" {
		t.Errorf("Expected paid cancelled entitlement, got %s %s", ent.Tier, ent.Status)
	}
}

func TestWebhook_DatabaseFailureReturns500(t *testing.T) {
	repo := &mockWebhookEntitlementRepo{
		duplicateEvents: map[string]bool{},
		insertEventErr:  fmt.Errorf("db connection down"),
	}
	app := setupWebhookTestApp(repo, &mockAuditLogger{})

	event := stripe.Event{
		ID:   "evt_999",
		Type: "checkout.session.completed",
	}
	payload, _ := json.Marshal(event)
	req := createSignedRequest(payload, "whsec_test")

	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected 500 for database write failure, got %d", resp.StatusCode)
	}
}

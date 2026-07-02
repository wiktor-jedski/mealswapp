package subscription

// Implements DESIGN-007 StripeWebhookHandler verification.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryStripeWebhookStore struct {
	events       map[string]repository.ProcessedStripeEvent
	entitlements []repository.Entitlement
	deadLetters  []repository.StripeDeadLetter
	err          error
}

func (s *memoryStripeWebhookStore) ProcessStripeWebhookEvent(_ context.Context, event repository.ProcessedStripeEvent, entitlement *repository.Entitlement) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	if s.events == nil {
		s.events = map[string]repository.ProcessedStripeEvent{}
	}
	if _, exists := s.events[event.EventID]; exists {
		return false, nil
	}
	s.events[event.EventID] = event
	if entitlement != nil {
		s.entitlements = append(s.entitlements, *entitlement)
	}
	return true, nil
}

func (s *memoryStripeWebhookStore) InsertStripeDeadLetter(_ context.Context, entry repository.StripeDeadLetter) error {
	s.deadLetters = append(s.deadLetters, entry)
	return nil
}

func TestStripeWebhookServiceVerifiesSignatureAndAppendsPaidActiveEntitlement(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	userID := uuid.New()
	store := &memoryStripeWebhookStore{}
	service := NewStripeWebhookService("whsec_test_secret", store)
	payload := webhookPayload("evt_checkout", "checkout.session.completed", userID, "cus_123", "sub_123", "")

	result, err := service.HandleWebhook(context.Background(), WebhookRequest{Payload: []byte(payload), Signature: signWebhookPayload([]byte(payload), "whsec_test_secret")})
	if err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if result.Duplicate || result.EventID != "evt_checkout" {
		t.Fatalf("result = %+v, want first event", result)
	}
	if len(store.entitlements) != 1 {
		t.Fatalf("entitlements = %#v, want one append", store.entitlements)
	}
	entitlement := store.entitlements[0]
	if entitlement.UserID != userID || entitlement.Tier != "paid" || entitlement.Status != "active" || entitlement.StripeCustomerID != "cus_123" || entitlement.StripeSubscriptionID != "sub_123" {
		t.Fatalf("entitlement = %#v, want paid active Stripe projection", entitlement)
	}
}

func TestStripeWebhookServiceRejectsInvalidOrMissingSignatures(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	service := NewStripeWebhookService("whsec_test_secret", &memoryStripeWebhookStore{})
	payload := []byte(webhookPayload("evt_bad_sig", "checkout.session.completed", uuid.New(), "cus_123", "sub_123", ""))
	for _, signature := range []string{"", "t=123,v1=bad"} {
		if _, err := service.HandleWebhook(context.Background(), WebhookRequest{Payload: payload, Signature: signature}); !errors.Is(err, ErrWebhookInvalidSignature) {
			t.Fatalf("HandleWebhook(signature=%q) error = %v, want invalid signature", signature, err)
		}
	}
}

func TestStripeWebhookServiceDuplicatesReturnSuccessWithoutDuplicateEntitlement(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	userID := uuid.New()
	store := &memoryStripeWebhookStore{}
	service := NewStripeWebhookService("whsec_test_secret", store)
	payload := []byte(webhookPayload("evt_duplicate", "checkout.session.completed", userID, "cus_123", "sub_123", ""))
	signature := signWebhookPayload(payload, "whsec_test_secret")

	if _, err := service.HandleWebhook(context.Background(), WebhookRequest{Payload: payload, Signature: signature}); err != nil {
		t.Fatalf("first HandleWebhook() error = %v", err)
	}
	result, err := service.HandleWebhook(context.Background(), WebhookRequest{Payload: payload, Signature: signature})
	if err != nil {
		t.Fatalf("duplicate HandleWebhook() error = %v", err)
	}
	if !result.Duplicate {
		t.Fatalf("duplicate result = %+v, want duplicate=true", result)
	}
	if len(store.entitlements) != 1 {
		t.Fatalf("entitlement appends = %d, want 1", len(store.entitlements))
	}
}

func TestStripeWebhookServiceMapsFailedAndCancelledEventsWithoutDeletingHistory(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	userID := uuid.New()
	store := &memoryStripeWebhookStore{}
	service := NewStripeWebhookService("whsec_test_secret", store)
	events := []struct {
		id       string
		event    string
		status   string
		want     string
		subField string
	}{
		{id: "evt_failed", event: "invoice.payment_failed", want: "past_due", subField: "sub_123"},
		{id: "evt_cancelled", event: "customer.subscription.deleted", want: "cancelled", subField: "sub_123"},
		{id: "evt_subscription_past_due", event: "customer.subscription.updated", status: "past_due", want: "past_due", subField: ""},
	}

	for _, tt := range events {
		payload := []byte(webhookPayload(tt.id, tt.event, userID, "cus_123", tt.subField, tt.status))
		if _, err := service.HandleWebhook(context.Background(), WebhookRequest{Payload: payload, Signature: signWebhookPayload(payload, "whsec_test_secret")}); err != nil {
			t.Fatalf("HandleWebhook(%s) error = %v", tt.event, err)
		}
		got := store.entitlements[len(store.entitlements)-1]
		if got.Status != tt.want {
			t.Fatalf("HandleWebhook(%s) status = %q, want %q", tt.event, got.Status, tt.want)
		}
	}
	if len(store.entitlements) != len(events) {
		t.Fatalf("entitlement history length = %d, want %d", len(store.entitlements), len(events))
	}
}

func TestStripeWebhookServiceReturnsStoreFailureForStripeRetry(t *testing.T) {
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-013.
	// Traces SW-REQ-045 and SW-REQ-052.
	expected := errors.New("database write failed for card 4242 and payer@example.test")
	store := &memoryStripeWebhookStore{err: expected}
	service := NewStripeWebhookService("whsec_test_secret", store)
	payload := []byte(webhookPayload("evt_store_failure", "checkout.session.completed", uuid.New(), "cus_123", "sub_123", ""))

	_, err := service.HandleWebhook(context.Background(), WebhookRequest{Payload: payload, Signature: signWebhookPayload(payload, "whsec_test_secret")})
	if !errors.Is(err, ErrWebhookStoreFailed) || !errors.Is(err, expected) {
		t.Fatalf("HandleWebhook() error = %v, want wrapped store failure", err)
	}
	if len(store.deadLetters) != 1 {
		t.Fatalf("dead letters = %#v, want one sanitized entry", store.deadLetters)
	}
	deadLetter := store.deadLetters[0]
	if deadLetter.EventID != "evt_store_failure" || deadLetter.EventType != "checkout.session.completed" || deadLetter.FailureCategory != "webhook_processing_failed" {
		t.Fatalf("dead letter = %#v, want event metadata", deadLetter)
	}
	if deadLetter.PayloadSHA256 == "" || deadLetter.StripeCustomerID != "cus_123" || deadLetter.StripeSubscriptionID != "sub_123" || deadLetter.UserID == nil {
		t.Fatalf("dead letter = %#v, want sanitized provider metadata and payload hash", deadLetter)
	}
	if strings.Contains(deadLetter.ErrorMessage, "4242") || strings.Contains(deadLetter.ErrorMessage, "payer@example.test") {
		t.Fatalf("dead letter error message = %q, want sanitized message", deadLetter.ErrorMessage)
	}
}

func signWebhookPayload(payload []byte, secret string) string {
	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

func webhookPayload(eventID string, eventType string, userID uuid.UUID, customerID string, subscriptionID string, status string) string {
	objectID := "cs_test_123"
	if eventType == "customer.subscription.deleted" || eventType == "customer.subscription.updated" {
		objectID = "sub_object_123"
	}
	return fmt.Sprintf(`{"id":%q,"type":%q,"data":{"object":{"id":%q,"client_reference_id":%q,"customer":%q,"subscription":%q,"status":%q,"metadata":{"user_id":%q}}}}`,
		eventID, eventType, objectID, userID.String(), customerID, subscriptionID, status, userID.String())
}

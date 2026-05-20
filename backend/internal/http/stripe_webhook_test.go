package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/services/entitlements"
)

func TestStripeWebhookHandlerAcceptsSignedEventsAndDuplicates(t *testing.T) {
	now := fixedWebhookNow()
	store := entitlements.NewMemoryWebhookEventStore()
	processor := entitlements.NewStripeWebhookProcessorWithClock("whsec_test", store, nil, func() time.Time { return now })
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, StripeWebhookService: processor})
	payload := []byte(`{"id":"evt_handler","type":"payment_intent.succeeded"}`)
	signature := testStripeSignature("whsec_test", now, payload)

	res := performStripeWebhookRequest(t, app, payload, signature)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected webhook 200, got %d", res.StatusCode)
	}
	data := dataMap(t, decodeEnvelope(t, res).Data)
	if data["outcome"] != "success" || data["eventId"] != "evt_handler" {
		t.Fatalf("unexpected webhook response: %#v", data)
	}

	duplicate := performStripeWebhookRequest(t, app, payload, signature)
	defer duplicate.Body.Close()
	if duplicate.StatusCode != http.StatusOK {
		t.Fatalf("expected duplicate webhook 200, got %d", duplicate.StatusCode)
	}
	duplicateData := dataMap(t, decodeEnvelope(t, duplicate).Data)
	if duplicateData["outcome"] != "duplicate" {
		t.Fatalf("expected duplicate outcome, got %#v", duplicateData)
	}
}

func TestStripeWebhookHandlerRejectsBadSignaturesAndRetriesFailures(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config:               config.Config{Environment: "test"},
		StripeWebhookService: fakeStripeWebhookService{err: entitlements.ErrWebhookSignatureInvalid},
	})

	badSignature := performStripeWebhookRequest(t, app, []byte(`{"id":"evt_bad","type":"payment_intent.succeeded"}`), "bad")
	defer badSignature.Body.Close()
	if badSignature.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad signature 400, got %d", badSignature.StatusCode)
	}
	payload := decodeEnvelope(t, badSignature)
	if payload.Error == nil || payload.Error.Code != "webhook_signature_invalid" {
		t.Fatalf("expected signature error envelope, got %#v", payload)
	}

	app = NewRouter(ServiceDependencies{
		Config:               config.Config{Environment: "test"},
		StripeWebhookService: fakeStripeWebhookService{err: errors.New("entitlement write failed")},
	})
	failed := performStripeWebhookRequest(t, app, []byte(`{"id":"evt_fail","type":"customer.subscription.updated"}`), "signed")
	defer failed.Body.Close()
	if failed.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected failed dispatch 500 for Stripe retry, got %d", failed.StatusCode)
	}
}

type fakeStripeWebhookService struct {
	err error
}

func (service fakeStripeWebhookService) Handle(ctx context.Context, signature string, payload []byte) (entitlements.ProcessedEvent, error) {
	if service.err != nil {
		return entitlements.ProcessedEvent{}, service.err
	}
	return entitlements.ProcessedEvent{EventID: "evt_fake", ProcessedAt: fixedWebhookNow(), Outcome: "success"}, nil
}

func performStripeWebhookRequest(t *testing.T, app interface {
	Test(*http.Request, ...int) (*http.Response, error)
}, payload []byte, signature string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", signature)
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func fixedWebhookNow() time.Time {
	return time.Date(2026, 5, 20, 12, 30, 0, 0, time.UTC)
}

func testStripeSignature(secret string, now time.Time, payload []byte) string {
	timestamp := now.Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	return "t=" + strconv.FormatInt(timestamp, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

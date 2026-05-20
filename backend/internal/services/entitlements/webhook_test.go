package entitlements

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestStripeWebhookProcessorVerifiesDispatchesAndDeduplicates(t *testing.T) {
	now := fixedNow()
	store := NewMemoryWebhookEventStore()
	dispatcher := &fakeWebhookDispatcher{}
	processor := NewStripeWebhookProcessorWithClock("whsec_test", store, dispatcher, func() time.Time { return now })
	payload := []byte(`{"id":"evt_123","type":"payment_intent.succeeded"}`)
	signature := testStripeSignature("whsec_test", now, payload)

	result, err := processor.Handle(context.Background(), signature, payload)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "success" || result.EventID != "evt_123" || dispatcher.calls != 1 || store.Status("evt_123") != "processed" {
		t.Fatalf("unexpected successful webhook result: result=%#v calls=%d status=%q", result, dispatcher.calls, store.Status("evt_123"))
	}

	duplicate, err := processor.Handle(context.Background(), signature, payload)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Outcome != "duplicate" || dispatcher.calls != 1 {
		t.Fatalf("expected duplicate without dispatch, result=%#v calls=%d", duplicate, dispatcher.calls)
	}
}

func TestStripeWebhookProcessorRejectsBadSignatureAndPayload(t *testing.T) {
	now := fixedNow()
	processor := NewStripeWebhookProcessorWithClock("whsec_test", NewMemoryWebhookEventStore(), nil, func() time.Time { return now })
	payload := []byte(`{"id":"evt_123","type":"payment_intent.succeeded"}`)

	if _, err := processor.Handle(context.Background(), "t=1,v1=bad", payload); !errors.Is(err, ErrWebhookSignatureInvalid) {
		t.Fatalf("expected signature error, got %v", err)
	}
	badPayload := []byte(`{"type":"payment_intent.succeeded"}`)
	if _, err := processor.Handle(context.Background(), testStripeSignature("whsec_test", now, badPayload), badPayload); !errors.Is(err, ErrWebhookEventInvalid) {
		t.Fatalf("expected event payload error, got %v", err)
	}
}

func TestStripeWebhookProcessorRecordsFailedDispatchForRetry(t *testing.T) {
	now := fixedNow()
	store := NewMemoryWebhookEventStore()
	dispatchErr := errors.New("entitlement write failed")
	processor := NewStripeWebhookProcessorWithClock("whsec_test", store, &fakeWebhookDispatcher{err: dispatchErr}, func() time.Time { return now })
	payload := []byte(`{"id":"evt_failed","type":"customer.subscription.updated"}`)

	_, err := processor.Handle(context.Background(), testStripeSignature("whsec_test", now, payload), payload)
	if !errors.Is(err, dispatchErr) {
		t.Fatalf("expected dispatch error, got %v", err)
	}
	if store.Status("evt_failed") != "failed" {
		t.Fatalf("expected failed event recorded for retry, got %q", store.Status("evt_failed"))
	}
}

type fakeWebhookDispatcher struct {
	calls int
	err   error
}

func (dispatcher *fakeWebhookDispatcher) DispatchStripeEvent(ctx context.Context, event StripeWebhookEvent) error {
	dispatcher.calls++
	return dispatcher.err
}

func testStripeSignature(secret string, now time.Time, payload []byte) string {
	timestamp := now.Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	return "t=" + strconv.FormatInt(timestamp, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

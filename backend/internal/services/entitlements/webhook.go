package entitlements

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrWebhookSignatureInvalid = errors.New("stripe webhook signature invalid")
	ErrWebhookEventInvalid     = errors.New("stripe webhook event invalid")
)

type StripeWebhookProcessor struct {
	secret     string
	store      WebhookEventStore
	dispatcher WebhookDispatcher
	now        func() time.Time
	tolerance  time.Duration
}

type WebhookEventStore interface {
	BeginProcessing(ctx context.Context, event StripeWebhookEvent) (duplicate bool, err error)
	MarkProcessed(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, event StripeWebhookEvent, cause error) error
}

type WebhookDispatcher interface {
	DispatchStripeEvent(ctx context.Context, event StripeWebhookEvent) error
}

type StripeWebhookEvent struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Signature  string          `json:"signature"`
	ReceivedAt time.Time       `json:"receivedAt"`
}

type ProcessedEvent struct {
	EventID     string    `json:"eventId"`
	ProcessedAt time.Time `json:"processedAt"`
	Outcome     string    `json:"outcome"`
}

func NewStripeWebhookProcessor(secret string, store WebhookEventStore, dispatcher WebhookDispatcher) StripeWebhookProcessor {
	return NewStripeWebhookProcessorWithClock(secret, store, dispatcher, time.Now)
}

func NewStripeWebhookProcessorWithClock(secret string, store WebhookEventStore, dispatcher WebhookDispatcher, now func() time.Time) StripeWebhookProcessor {
	if store == nil {
		store = NewMemoryWebhookEventStore()
	}
	return StripeWebhookProcessor{
		secret:     secret,
		store:      store,
		dispatcher: dispatcher,
		now:        now,
		tolerance:  5 * time.Minute,
	}
}

func (processor StripeWebhookProcessor) Handle(ctx context.Context, signature string, payload []byte) (ProcessedEvent, error) {
	if err := processor.verify(signature, payload); err != nil {
		return ProcessedEvent{}, err
	}
	event, err := parseStripeEvent(payload)
	if err != nil {
		return ProcessedEvent{}, err
	}
	event.Signature = signature
	event.Payload = append(json.RawMessage(nil), payload...)
	event.ReceivedAt = processor.now().UTC()

	duplicate, err := processor.store.BeginProcessing(ctx, event)
	if err != nil {
		return ProcessedEvent{}, err
	}
	if duplicate {
		return ProcessedEvent{EventID: event.ID, ProcessedAt: processor.now().UTC(), Outcome: "duplicate"}, nil
	}

	if processor.dispatcher != nil {
		if err := processor.dispatcher.DispatchStripeEvent(ctx, event); err != nil {
			_ = processor.store.MarkFailed(ctx, event, err)
			return ProcessedEvent{}, err
		}
	}
	if err := processor.store.MarkProcessed(ctx, event.ID); err != nil {
		_ = processor.store.MarkFailed(ctx, event, err)
		return ProcessedEvent{}, err
	}
	return ProcessedEvent{EventID: event.ID, ProcessedAt: processor.now().UTC(), Outcome: "success"}, nil
}

func (processor StripeWebhookProcessor) verify(signatureHeader string, payload []byte) error {
	if processor.secret == "" || strings.TrimSpace(signatureHeader) == "" {
		return ErrWebhookSignatureInvalid
	}
	timestamp, signatures, err := parseSignatureHeader(signatureHeader)
	if err != nil || len(signatures) == 0 {
		return ErrWebhookSignatureInvalid
	}
	if processor.tolerance > 0 {
		signedAt := time.Unix(timestamp, 0)
		now := processor.now()
		if signedAt.Before(now.Add(-processor.tolerance)) || signedAt.After(now.Add(processor.tolerance)) {
			return ErrWebhookSignatureInvalid
		}
	}

	mac := hmac.New(sha256.New, []byte(processor.secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	expected := mac.Sum(nil)
	for _, signature := range signatures {
		decoded, err := hex.DecodeString(signature)
		if err == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return ErrWebhookSignatureInvalid
}

func parseSignatureHeader(header string) (int64, []string, error) {
	var timestamp int64
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, nil, err
			}
			timestamp = parsed
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == 0 {
		return 0, nil, fmt.Errorf("missing timestamp")
	}
	return timestamp, signatures, nil
}

func parseStripeEvent(payload []byte) (StripeWebhookEvent, error) {
	var envelope struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return StripeWebhookEvent{}, ErrWebhookEventInvalid
	}
	if strings.TrimSpace(envelope.ID) == "" || strings.TrimSpace(envelope.Type) == "" {
		return StripeWebhookEvent{}, ErrWebhookEventInvalid
	}
	return StripeWebhookEvent{ID: envelope.ID, Type: envelope.Type}, nil
}

type MemoryWebhookEventStore struct {
	mu     sync.Mutex
	events map[string]string
	failed map[string]StripeWebhookEvent
}

func NewMemoryWebhookEventStore() *MemoryWebhookEventStore {
	return &MemoryWebhookEventStore{
		events: map[string]string{},
		failed: map[string]StripeWebhookEvent{},
	}
}

func (store *MemoryWebhookEventStore) BeginProcessing(ctx context.Context, event StripeWebhookEvent) (bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.events[event.ID] == "processed" || store.events[event.ID] == "processing" {
		return true, nil
	}
	store.events[event.ID] = "processing"
	return false, nil
}

func (store *MemoryWebhookEventStore) MarkProcessed(ctx context.Context, eventID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.events[eventID] = "processed"
	return nil
}

func (store *MemoryWebhookEventStore) MarkFailed(ctx context.Context, event StripeWebhookEvent, cause error) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.events[event.ID] = "failed"
	store.failed[event.ID] = event
	return nil
}

func (store *MemoryWebhookEventStore) Status(eventID string) string {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.events[eventID]
}

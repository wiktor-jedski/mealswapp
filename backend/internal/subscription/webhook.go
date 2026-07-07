package subscription

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
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-007 StripeWebhookHandler replay window for signed provider messages.
const stripeSignatureTolerance = 5 * time.Minute

// Implements DESIGN-007 StripeWebhookHandler webhook processing errors.
var (
	// ErrWebhookInvalidSignature means the Stripe-Signature header failed verification.
	ErrWebhookInvalidSignature = errors.New("stripe webhook signature is invalid")
	// ErrWebhookInvalidPayload means the verified Stripe event payload cannot be applied.
	ErrWebhookInvalidPayload = errors.New("stripe webhook payload is invalid")
	// ErrWebhookStoreFailed means webhook idempotency or entitlement persistence failed.
	ErrWebhookStoreFailed = errors.New("stripe webhook persistence failed")
)

// StripeWebhookStore persists provider event idempotency and entitlement changes transactionally.
// Implements DESIGN-007 StripeWebhookHandler.
type StripeWebhookStore interface {
	ProcessStripeWebhookEvent(context.Context, repository.ProcessedStripeEvent, *repository.Entitlement) (bool, error)
	InsertStripeDeadLetter(context.Context, repository.StripeDeadLetter) error
}

// WebhookRequest carries the raw provider request fields needed for verification.
// Implements DESIGN-007 StripeWebhookHandler.
type WebhookRequest struct {
	Payload    []byte
	Signature  string
	ReceivedAt time.Time
}

// WebhookResult reports whether a verified event caused side effects.
// Implements DESIGN-007 StripeWebhookHandler.
type WebhookResult struct {
	EventID   string
	EventType string
	Duplicate bool
}

// StripeWebhookService verifies Stripe events and applies entitlement side effects.
// Implements DESIGN-007 StripeWebhookHandler.
type StripeWebhookService struct {
	signingSecret string
	store         StripeWebhookStore
	logs          observability.LogSink
	now           func() time.Time
}

// NewStripeWebhookService creates the Stripe webhook service.
// Implements DESIGN-007 StripeWebhookHandler.
func NewStripeWebhookService(signingSecret string, store StripeWebhookStore) *StripeWebhookService {
	return &StripeWebhookService{signingSecret: strings.TrimSpace(signingSecret), store: store, now: time.Now}
}

// WithLogSink attaches best-effort structured logs for recognized no-op billing events.
// Implements DESIGN-007 StripeWebhookHandler.
func (s *StripeWebhookService) WithLogSink(logs observability.LogSink) *StripeWebhookService {
	if s == nil {
		return s
	}
	s.logs = logs
	return s
}

// HandleWebhook verifies, deduplicates, and applies subscription entitlement state.
// Implements DESIGN-007 StripeWebhookHandler.
func (s *StripeWebhookService) HandleWebhook(ctx context.Context, req WebhookRequest) (WebhookResult, error) {
	if s == nil || s.store == nil {
		return WebhookResult{}, ErrWebhookStoreFailed
	}
	receivedAt := req.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = s.now()
	}
	if !verifyStripeSignature(req.Payload, req.Signature, s.signingSecret, receivedAt) {
		return WebhookResult{}, ErrWebhookInvalidSignature
	}
	event, err := parseStripeEvent(req.Payload)
	if err != nil {
		return WebhookResult{}, err
	}
	entitlement, err := entitlementFromStripeEvent(event)
	if err != nil {
		return WebhookResult{}, err
	}
	inserted, err := s.store.ProcessStripeWebhookEvent(ctx, repository.ProcessedStripeEvent{
		EventID:   event.ID,
		EventType: event.Type,
		Outcome:   "success",
		Payload:   req.Payload,
	}, entitlement)
	if err != nil {
		s.recordDeadLetter(ctx, event, req.Payload, entitlement, err)
		return WebhookResult{}, fmt.Errorf("%w: %w", ErrWebhookStoreFailed, err)
	}
	if inserted && entitlement == nil && isRecognizedNoopStripeBillingEvent(event) {
		s.logNoop(ctx, event)
	}
	return WebhookResult{EventID: event.ID, EventType: event.Type, Duplicate: !inserted}, nil
}

// recordDeadLetter persists only allow-listed failure metadata for retry triage.
// Implements DESIGN-007 StripeWebhookHandler dead-letter persistence.
func (s *StripeWebhookService) recordDeadLetter(ctx context.Context, event stripeEvent, payload []byte, entitlement *repository.Entitlement, cause error) {
	sum := sha256.Sum256(payload)
	entry := repository.StripeDeadLetter{
		EventID:              event.ID,
		EventType:            event.Type,
		FailureCategory:      "webhook_processing_failed",
		ErrorMessage:         sanitizedErrorMessage(cause),
		PayloadSHA256:        hex.EncodeToString(sum[:]),
		StripeCustomerID:     strings.TrimSpace(event.Object.Customer),
		StripeSubscriptionID: strings.TrimSpace(event.Object.Subscription),
	}
	if entry.StripeSubscriptionID == "" && strings.HasPrefix(event.Object.ID, "sub_") {
		entry.StripeSubscriptionID = event.Object.ID
	}
	if entitlement != nil {
		userID := entitlement.UserID
		entry.UserID = &userID
	}
	_ = s.store.InsertStripeDeadLetter(ctx, entry)
}

// logNoop records sanitized operator context for recognized billing events with no local side effect.
// Implements DESIGN-007 StripeWebhookHandler.
func (s *StripeWebhookService) logNoop(ctx context.Context, event stripeEvent) {
	if s == nil || s.logs == nil {
		return
	}
	_ = s.logs.Log(ctx, observability.LogEvent{
		Service: "subscription.webhook",
		Level:   "info",
		Message: "stripe webhook recognized no-op",
		Fields: map[string]any{
			"stripe_event_id":        strings.TrimSpace(event.ID),
			"stripe_event_type":      strings.TrimSpace(event.Type),
			"stripe_subscription_id": stripeSubscriptionID(event.Object),
			"stripe_customer_id":     strings.TrimSpace(event.Object.Customer),
		},
		CreatedAt: s.now().UTC(),
	})
}

// sanitizedErrorMessage keeps dead-letter diagnostics bounded and payload-free.
// Implements DESIGN-007 StripeWebhookHandler dead-letter persistence.
func sanitizedErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "context_cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	case repository.IsKind(err, repository.ErrorKindNotFound):
		return "repository_not_found"
	case repository.IsKind(err, repository.ErrorKindValidation):
		return "repository_validation"
	case repository.IsKind(err, repository.ErrorKindConflict):
		return "repository_conflict"
	case repository.IsKind(err, repository.ErrorKindConnection):
		return "repository_connection"
	case repository.IsKind(err, repository.ErrorKindRetryable):
		return "repository_retryable"
	case repository.IsKind(err, repository.ErrorKindCanceled):
		return "repository_canceled"
	case repository.IsKind(err, repository.ErrorKindInternal):
		return "repository_internal"
	}
	var repoErr *repository.Error
	if errors.As(err, &repoErr) {
		return "repository_unknown"
	}
	return "stripe webhook processing failed"
}

// stripeEvent is the minimal verified event envelope consumed by Phase 06.
// Implements DESIGN-007 StripeWebhookHandler.
type stripeEvent struct {
	ID     string       `json:"id"`
	Type   string       `json:"type"`
	Object stripeObject `json:"-"`
}

// stripeObject contains the subscription/customer/user fields used for entitlement projection.
// Implements DESIGN-007 StripeWebhookHandler.
type stripeObject struct {
	ClientReferenceID string            `json:"client_reference_id"`
	Customer          string            `json:"customer"`
	Subscription      string            `json:"subscription"`
	ID                string            `json:"id"`
	Status            string            `json:"status"`
	Metadata          map[string]string `json:"metadata"`
}

// parseStripeEvent extracts only the stable fields needed for entitlement updates.
// Implements DESIGN-007 StripeWebhookHandler.
func parseStripeEvent(payload []byte) (stripeEvent, error) {
	var envelope struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object stripeObject `json:"object"`
		} `json:"data"`
	}
	if len(payload) == 0 || !json.Valid(payload) {
		return stripeEvent{}, ErrWebhookInvalidPayload
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return stripeEvent{}, ErrWebhookInvalidPayload
	}
	event := stripeEvent{ID: strings.TrimSpace(envelope.ID), Type: strings.TrimSpace(envelope.Type), Object: envelope.Data.Object}
	if event.ID == "" || event.Type == "" {
		return stripeEvent{}, ErrWebhookInvalidPayload
	}
	return event, nil
}

// entitlementFromStripeEvent maps supported Stripe events into append-only entitlement state.
// Implements DESIGN-007 StripeWebhookHandler.
func entitlementFromStripeEvent(event stripeEvent) (*repository.Entitlement, error) {
	status, ok := entitlementStatusForStripeEvent(event)
	if !ok {
		return nil, nil
	}
	userID, err := userIDFromStripeObject(event.Object)
	if err != nil {
		return nil, err
	}
	entitlement := paidEntitlementForWebhook(userID, status, event.Object)
	return &entitlement, nil
}

// entitlementStatusForStripeEvent classifies retry-safe local entitlement transitions.
// Implements DESIGN-007 StripeWebhookHandler.
func entitlementStatusForStripeEvent(event stripeEvent) (string, bool) {
	switch event.Type {
	case "checkout.session.completed", "invoice.payment_succeeded":
		return "active", true
	case "invoice.payment_failed", "invoice.payment_action_required":
		return "past_due", true
	case "customer.subscription.deleted":
		return "cancelled", true
	case "customer.subscription.paused":
		return "past_due", true
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.resumed":
		return entitlementStatusForStripeSubscription(event.Object.Status)
	default:
		return "", false
	}
}

// isRecognizedNoopStripeBillingEvent identifies Stripe billing events that are intentionally observed only.
// Implements DESIGN-007 StripeWebhookHandler.
func isRecognizedNoopStripeBillingEvent(event stripeEvent) bool {
	switch event.Type {
	case "customer.subscription.trial_will_end":
		return true
	default:
		return false
	}
}

// userIDFromStripeObject reads the server-authored checkout/session user id from metadata first.
// Implements DESIGN-007 StripeWebhookHandler.
func userIDFromStripeObject(object stripeObject) (uuid.UUID, error) {
	candidates := []string{object.Metadata["user_id"], object.Metadata["userId"], object.ClientReferenceID}
	for _, candidate := range candidates {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			id, err := uuid.Parse(candidate)
			if err != nil {
				return uuid.Nil, ErrWebhookInvalidPayload
			}
			return id, nil
		}
	}
	return uuid.Nil, ErrWebhookInvalidPayload
}

// paidEntitlementForWebhook builds the append-only paid entitlement projection.
// Implements DESIGN-007 StripeWebhookHandler.
func paidEntitlementForWebhook(userID uuid.UUID, status string, object stripeObject) repository.Entitlement {
	return repository.Entitlement{
		UserID:               userID,
		Tier:                 "paid",
		Status:               status,
		SearchLimitPer24h:    0,
		AllowedModes:         []string{"catalog", "substitution", "daily_diet_alternative"},
		StripeCustomerID:     strings.TrimSpace(object.Customer),
		StripeSubscriptionID: stripeSubscriptionID(object),
	}
}

// stripeSubscriptionID returns the allow-listed subscription identifier from Stripe object shapes.
// Implements DESIGN-007 StripeWebhookHandler.
func stripeSubscriptionID(object stripeObject) string {
	subscriptionID := strings.TrimSpace(object.Subscription)
	if subscriptionID == "" && strings.HasPrefix(object.ID, "sub_") {
		subscriptionID = strings.TrimSpace(object.ID)
	}
	return subscriptionID
}

// verifyStripeSignature validates Stripe's timestamped v1 HMAC signature.
// Implements DESIGN-007 StripeWebhookHandler.
func verifyStripeSignature(payload []byte, header string, secret string, receivedAt time.Time) bool {
	timestamp, signatures := parseStripeSignatureHeader(header)
	if timestamp == "" || len(signatures) == 0 || strings.TrimSpace(secret) == "" {
		return false
	}
	unixTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	signedAt := time.Unix(unixTimestamp, 0)
	if receivedAt.Sub(signedAt) > stripeSignatureTolerance || signedAt.Sub(receivedAt) > stripeSignatureTolerance {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := mac.Sum(nil)
	for _, signature := range signatures {
		got, err := hex.DecodeString(signature)
		if err == nil && hmac.Equal(got, expected) {
			return true
		}
	}
	return false
}

// parseStripeSignatureHeader extracts the timestamp and v1 signatures.
// Implements DESIGN-007 StripeWebhookHandler.
func parseStripeSignatureHeader(header string) (string, []string) {
	var timestamp string
	signatures := []string{}
	for part := range strings.SplitSeq(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "t":
			if _, err := strconv.ParseInt(value, 10, 64); err == nil {
				timestamp = value
			}
		case "v1":
			if value = strings.TrimSpace(value); value != "" {
				signatures = append(signatures, value)
			}
		}
	}
	return timestamp, signatures
}

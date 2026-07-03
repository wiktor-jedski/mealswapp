package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// StripeSubscriptionGateway lists Stripe-owned subscription state for reconciliation.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type StripeSubscriptionGateway interface {
	ListSubscriptions(context.Context) ([]StripeSubscription, error)
}

// StripeSubscription is the sanitized Stripe subscription projection used locally.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type StripeSubscription struct {
	UserID         uuid.UUID
	CustomerID     string
	SubscriptionID string
	Status         string
}

// ReconciliationResult summarizes one reconciliation pass.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type ReconciliationResult struct {
	Checked  int
	Appended int
	Skipped  int
}

// ReconciliationService repairs local entitlement drift from Stripe subscription state.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type ReconciliationService struct {
	gateway StripeSubscriptionGateway
	store   repository.EntitlementRepository
	logs    observability.LogSink
}

// NewReconciliationService creates Stripe entitlement reconciliation.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func NewReconciliationService(gateway StripeSubscriptionGateway, store repository.EntitlementRepository, logs observability.LogSink) *ReconciliationService {
	return &ReconciliationService{gateway: gateway, store: store, logs: logs}
}

// ReconcileStripeEntitlements appends missing entitlement states and skips exact matches.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func (s *ReconciliationService) ReconcileStripeEntitlements(ctx context.Context) (ReconciliationResult, error) {
	if s == nil || s.gateway == nil || s.store == nil {
		return ReconciliationResult{}, ErrStripeUnavailable
	}
	subscriptions, err := s.gateway.ListSubscriptions(ctx)
	if err != nil {
		s.warn(ctx, "stripe reconciliation failed", map[string]any{"reason": "stripe_unavailable"})
		return ReconciliationResult{}, fmt.Errorf("%w: %w", ErrStripeUnavailable, err)
	}
	result := ReconciliationResult{Checked: len(subscriptions)}
	for _, sub := range subscriptions {
		entitlement, ok := entitlementFromStripeSubscription(sub)
		if !ok {
			result.Skipped++
			continue
		}
		latest, err := s.store.GetLatest(ctx, sub.UserID)
		if err == nil && sameStripeEntitlement(latest, entitlement) {
			result.Skipped++
			continue
		}
		if err != nil && !repository.IsKind(err, repository.ErrorKindNotFound) {
			return result, err
		}
		if err := s.store.AppendEntitlement(ctx, entitlement); err != nil {
			return result, err
		}
		result.Appended++
	}
	return result, nil
}

// RunHourly starts the hourly reconciliation loop until ctx cancellation.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation job.
func (s *ReconciliationService) RunHourly(ctx context.Context) error {
	if _, err := s.ReconcileStripeEntitlements(ctx); err != nil {
		s.warn(ctx, "stripe reconciliation pass failed", map[string]any{"error": err.Error()})
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := s.ReconcileStripeEntitlements(ctx); err != nil {
				s.warn(ctx, "stripe reconciliation pass failed", map[string]any{"error": err.Error()})
			}
		}
	}
}

// warn emits best-effort operator-visible reconciliation warnings.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func (s *ReconciliationService) warn(ctx context.Context, message string, fields map[string]any) {
	if s == nil || s.logs == nil {
		return
	}
	_ = s.logs.Log(ctx, observability.LogEvent{
		Service:   "subscription.reconciliation",
		Level:     "warning",
		Message:   message,
		Fields:    fields,
		CreatedAt: time.Now().UTC(),
	})
}

// entitlementFromStripeSubscription maps Stripe states into local paid entitlement states.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func entitlementFromStripeSubscription(sub StripeSubscription) (repository.Entitlement, bool) {
	status, ok := entitlementStatusForStripeSubscription(sub.Status)
	if !ok || sub.UserID == uuid.Nil || strings.TrimSpace(sub.SubscriptionID) == "" {
		// Cannot repair missing entitlement without knowing UserID; rely on checkout session persistence.
		return repository.Entitlement{}, false
	}
	return repository.Entitlement{
		UserID:               sub.UserID,
		Tier:                 "paid",
		Status:               status,
		SearchLimitPer24h:    0,
		AllowedModes:         []string{"catalog", "substitution", "daily_diet_alternative"},
		StripeCustomerID:     strings.TrimSpace(sub.CustomerID),
		StripeSubscriptionID: strings.TrimSpace(sub.SubscriptionID),
	}, true
}

// entitlementStatusForStripeSubscription normalizes Stripe subscription status.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func entitlementStatusForStripeSubscription(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "active", "trialing":
		return "active", true
	case "past_due", "unpaid", "incomplete_expired":
		return "past_due", true
	case "canceled", "cancelled":
		return "cancelled", true
	default:
		return "", false
	}
}

// sameStripeEntitlement checks whether reconciliation would append duplicate state.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func sameStripeEntitlement(latest repository.Entitlement, next repository.Entitlement) bool {
	return latest.UserID == next.UserID &&
		latest.Tier == next.Tier &&
		latest.Status == next.Status &&
		latest.StripeCustomerID == next.StripeCustomerID &&
		latest.StripeSubscriptionID == next.StripeSubscriptionID
}

// StripeSubscriptionHTTPGateway reads subscription fixtures or sandbox data from Stripe.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type StripeSubscriptionHTTPGateway struct {
	secretKey string
	client    *http.Client
	baseURL   string
}

// NewStripeSubscriptionGateway creates a Stripe subscription list gateway.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func NewStripeSubscriptionGateway(secretKey string, client *http.Client) *StripeSubscriptionHTTPGateway {
	return NewStripeSubscriptionGatewayWithBaseURL(secretKey, client, "https://api.stripe.com")
}

// NewStripeSubscriptionGatewayWithBaseURL creates an injectable Stripe gateway for tests.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func NewStripeSubscriptionGatewayWithBaseURL(secretKey string, client *http.Client, baseURL string) *StripeSubscriptionHTTPGateway {
	if client == nil {
		client = http.DefaultClient
	}
	return &StripeSubscriptionHTTPGateway{secretKey: strings.TrimSpace(secretKey), client: client, baseURL: strings.TrimRight(baseURL, "/")}
}

// ListSubscriptions fetches all Stripe subscriptions with minimal allow-listed fields.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func (g *StripeSubscriptionHTTPGateway) ListSubscriptions(ctx context.Context) ([]StripeSubscription, error) {
	if g == nil || g.client == nil || g.secretKey == "" || g.baseURL == "" {
		return nil, ErrStripeUnavailable
	}
	var subscriptions []StripeSubscription
	var startingAfter string
	for {
		page, err := g.listSubscriptionPage(ctx, startingAfter)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, page.Subscriptions...)
		if !page.HasMore || page.LastID == "" {
			return subscriptions, nil
		}
		startingAfter = page.LastID
	}
}

// stripeSubscriptionPage carries one sanitized Stripe subscription page.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type stripeSubscriptionPage struct {
	Subscriptions []StripeSubscription
	HasMore       bool
	LastID        string
}

// listSubscriptionPage fetches one Stripe subscription page.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func (g *StripeSubscriptionHTTPGateway) listSubscriptionPage(ctx context.Context, startingAfter string) (stripeSubscriptionPage, error) {
	values := url.Values{}
	values.Set("status", "all")
	values.Set("limit", "100")
	if startingAfter != "" {
		values.Set("starting_after", startingAfter)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/v1/subscriptions?"+values.Encode(), nil)
	if err != nil {
		return stripeSubscriptionPage{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return stripeSubscriptionPage{}, ErrStripeUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		return stripeSubscriptionPage{}, ErrStripeUnavailable
	}
	var payload stripeSubscriptionListPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return stripeSubscriptionPage{}, err
	}
	page := stripeSubscriptionPage{HasMore: payload.HasMore}
	for _, item := range payload.Data {
		sub, err := item.toSubscription()
		if err == nil {
			page.Subscriptions = append(page.Subscriptions, sub)
			page.LastID = item.ID
		}
	}
	return page, nil
}

// stripeSubscriptionListPayload is Stripe's subscription list envelope.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type stripeSubscriptionListPayload struct {
	Data    []stripeSubscriptionPayload `json:"data"`
	HasMore bool                        `json:"has_more"`
}

// stripeSubscriptionPayload contains the allow-listed Stripe subscription fields.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type stripeSubscriptionPayload struct {
	ID       string            `json:"id"`
	Customer stripeCustomerRef `json:"customer"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata"`
}

// toSubscription converts Stripe payload data into a sanitized local projection.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func (p stripeSubscriptionPayload) toSubscription() (StripeSubscription, error) {
	userID, err := uuid.Parse(strings.TrimSpace(p.Metadata["user_id"]))
	if err != nil {
		return StripeSubscription{}, err
	}
	return StripeSubscription{
		UserID:         userID,
		CustomerID:     p.Customer.ID,
		SubscriptionID: strings.TrimSpace(p.ID),
		Status:         strings.TrimSpace(p.Status),
	}, nil
}

// stripeCustomerRef reads Stripe customer references without retaining payment data.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
type stripeCustomerRef struct {
	ID string
}

// UnmarshalJSON accepts Stripe customer references as either ids or expanded objects.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation.
func (r *stripeCustomerRef) UnmarshalJSON(data []byte) error {
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		r.ID = strings.TrimSpace(id)
		return nil
	}
	var object struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return errors.New("stripe customer reference is invalid")
	}
	r.ID = strings.TrimSpace(object.ID)
	return nil
}

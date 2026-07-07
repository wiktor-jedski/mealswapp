// Package subscription owns billing checkout creation behavior.
package subscription

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-007 SubscriptionController Stripe gateway response safety.
const maxStripeResponseBytes = 1 << 20

// Implements DESIGN-007 SubscriptionController checkout creation errors.
var (
	// ErrMissingIdempotencyKey means checkout creation was attempted without an Idempotency-Key.
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
	// ErrIdempotencyConflict means an Idempotency-Key was reused for a different checkout body.
	ErrIdempotencyConflict = errors.New("idempotency key reused with different body")
	// ErrInvalidPlan means the requested checkout plan is not configured.
	ErrInvalidPlan = errors.New("checkout plan is invalid")
	// ErrStripeUnavailable means checkout creation could not reach or use Stripe.
	ErrStripeUnavailable = errors.New("stripe is unavailable")
	// ErrNoActiveSubscription means a billing portal was requested without a paid active customer.
	ErrNoActiveSubscription = errors.New("active paid subscription is required")
)

// CheckoutGateway creates provider-hosted checkout sessions without receiving raw card data.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
type CheckoutGateway interface {
	CreateCheckoutSession(context.Context, CheckoutSessionRequest) (CheckoutSession, error)
}

// PortalGateway creates Stripe-hosted customer billing portal sessions.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
type PortalGateway interface {
	CreatePortalSession(context.Context, PortalSessionRequest) (PortalSession, error)
}

// CheckoutSessionRequest is the gateway checkout-session input.
// Implements DESIGN-007 SubscriptionController.
type CheckoutSessionRequest struct {
	UserID                 uuid.UUID
	Plan                   string
	PriceID                string
	SuccessURL             string
	CancelURL              string
	ProviderIdempotencyKey string
}

// CheckoutSession is the provider-hosted checkout-session output.
// Implements DESIGN-007 SubscriptionController.
type CheckoutSession struct {
	ID  string
	URL string
}

// PortalSessionRequest is the gateway billing-portal input.
// Implements DESIGN-007 SubscriptionController.
type PortalSessionRequest struct {
	CustomerID string
	ReturnURL  string
}

// PortalSession is the Stripe-hosted billing-portal output.
// Implements DESIGN-007 SubscriptionController.
type PortalSession struct {
	URL string
}

// StripeCheckoutGateway creates Stripe-hosted Checkout Sessions over the Stripe API.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
type StripeCheckoutGateway struct {
	secretKey string
	client    *http.Client
	baseURL   string
}

// NewStripeCheckoutGateway creates a Stripe API checkout gateway.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
func NewStripeCheckoutGateway(secretKey string, client *http.Client) *StripeCheckoutGateway {
	return NewStripeCheckoutGatewayWithBaseURL(secretKey, client, "https://api.stripe.com")
}

// NewStripeCheckoutGatewayWithBaseURL creates a Stripe gateway with an injectable base URL for tests.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
func NewStripeCheckoutGatewayWithBaseURL(secretKey string, client *http.Client, baseURL string) *StripeCheckoutGateway {
	if client == nil {
		client = http.DefaultClient
	}
	return &StripeCheckoutGateway{secretKey: strings.TrimSpace(secretKey), client: client, baseURL: strings.TrimRight(baseURL, "/")}
}

// CreateCheckoutSession creates a Stripe-hosted subscription checkout session.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
func (g *StripeCheckoutGateway) CreateCheckoutSession(ctx context.Context, req CheckoutSessionRequest) (CheckoutSession, error) {
	if g == nil || g.client == nil || g.secretKey == "" || g.baseURL == "" || strings.TrimSpace(req.ProviderIdempotencyKey) == "" {
		return CheckoutSession{}, ErrStripeUnavailable
	}
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("success_url", req.SuccessURL)
	form.Set("cancel_url", req.CancelURL)
	form.Set("client_reference_id", req.UserID.String())
	form.Set("line_items[0][price]", req.PriceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("metadata[user_id]", req.UserID.String())
	form.Set("metadata[plan]", req.Plan)
	form.Set("subscription_data[metadata][user_id]", req.UserID.String())
	form.Set("subscription_data[metadata][plan]", req.Plan)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return CheckoutSession{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Idempotency-Key", strings.TrimSpace(req.ProviderIdempotencyKey))

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return CheckoutSession{}, ErrStripeUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		return CheckoutSession{}, ErrStripeUnavailable
	}
	var payload struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxStripeResponseBytes)).Decode(&payload); err != nil {
		return CheckoutSession{}, err
	}
	if strings.TrimSpace(payload.ID) == "" || strings.TrimSpace(payload.URL) == "" {
		return CheckoutSession{}, fmt.Errorf("stripe checkout response missing session fields")
	}
	return CheckoutSession{ID: payload.ID, URL: payload.URL}, nil
}

// CreatePortalSession creates a Stripe-hosted customer billing portal session.
// Implements DESIGN-007 SubscriptionController Stripe gateway abstraction.
func (g *StripeCheckoutGateway) CreatePortalSession(ctx context.Context, req PortalSessionRequest) (PortalSession, error) {
	if g == nil || g.client == nil || g.secretKey == "" || g.baseURL == "" || strings.TrimSpace(req.CustomerID) == "" {
		return PortalSession{}, ErrStripeUnavailable
	}
	form := url.Values{}
	form.Set("customer", strings.TrimSpace(req.CustomerID))
	form.Set("return_url", req.ReturnURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v1/billing_portal/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return PortalSession{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.secretKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return PortalSession{}, ErrStripeUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		return PortalSession{}, ErrStripeUnavailable
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxStripeResponseBytes)).Decode(&payload); err != nil {
		return PortalSession{}, err
	}
	if strings.TrimSpace(payload.URL) == "" {
		return PortalSession{}, fmt.Errorf("stripe portal response missing session URL")
	}
	return PortalSession{URL: payload.URL}, nil
}

// CheckoutRequest is the authenticated service checkout input.
// Implements DESIGN-007 SubscriptionController.
type CheckoutRequest struct {
	UserID         uuid.UUID
	IdempotencyKey string
	Method         string
	Route          string
	Plan           string
	SuccessURL     string
	CancelURL      string
}

// CheckoutResponse is the sanitized API checkout payload.
// Implements DESIGN-007 SubscriptionController.
type CheckoutResponse struct {
	CheckoutSessionID string `json:"checkoutSessionId"`
	CheckoutURL       string `json:"checkoutUrl"`
	Plan              string `json:"plan"`
	PriceID           string `json:"priceId"`
	AmountCents       int    `json:"amountCents"`
}

// CheckoutResult carries a checkout response and HTTP replay metadata.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
type CheckoutResult struct {
	Response   CheckoutResponse
	StatusCode int
	Replayed   bool
}

// PortalRequest is the authenticated service billing-portal input.
// Implements DESIGN-007 SubscriptionController.
type PortalRequest struct {
	UserID    uuid.UUID
	ReturnURL string
}

// PortalResponse is the sanitized billing-portal API payload.
// Implements DESIGN-007 SubscriptionController.
type PortalResponse struct {
	PortalURL string `json:"portalUrl"`
}

// CheckoutService coordinates checkout idempotency and Stripe session creation.
// Implements DESIGN-007 SubscriptionController.
type CheckoutService struct {
	billing config.BillingConfig
	store   repository.CheckoutIdempotencyRepository
	gateway CheckoutGateway
}

// NewCheckoutService creates the checkout creation service.
// Implements DESIGN-007 SubscriptionController.
func NewCheckoutService(billing config.BillingConfig, store repository.CheckoutIdempotencyRepository, gateway CheckoutGateway) *CheckoutService {
	return &CheckoutService{billing: billing, store: store, gateway: gateway}
}

// PortalService coordinates billing portal creation without exposing provider customer IDs.
// Implements DESIGN-007 SubscriptionController billing portal handoff.
type PortalService struct {
	entitlements repository.EntitlementRepository
	gateway      PortalGateway
}

// NewPortalService creates the billing portal service.
// Implements DESIGN-007 SubscriptionController billing portal handoff.
func NewPortalService(entitlements repository.EntitlementRepository, gateway PortalGateway) *PortalService {
	return &PortalService{entitlements: entitlements, gateway: gateway}
}

// CreateBillingPortal creates a provider-hosted billing portal for the user's active paid subscription.
// Implements DESIGN-007 SubscriptionController billing portal handoff.
func (s *PortalService) CreateBillingPortal(ctx context.Context, req PortalRequest) (PortalResponse, error) {
	if s == nil || s.entitlements == nil || s.gateway == nil {
		return PortalResponse{}, ErrStripeUnavailable
	}
	entitlement, err := s.entitlements.GetLatest(ctx, req.UserID)
	if err != nil {
		if repository.IsKind(err, repository.ErrorKindNotFound) {
			return PortalResponse{}, ErrNoActiveSubscription
		}
		return PortalResponse{}, err
	}
	if entitlement.Tier != "paid" || entitlement.Status != "active" || strings.TrimSpace(entitlement.StripeCustomerID) == "" {
		return PortalResponse{}, ErrNoActiveSubscription
	}
	session, err := s.gateway.CreatePortalSession(ctx, PortalSessionRequest{CustomerID: entitlement.StripeCustomerID, ReturnURL: req.ReturnURL})
	if err != nil {
		return PortalResponse{}, fmt.Errorf("%w: %w", ErrStripeUnavailable, err)
	}
	return PortalResponse{PortalURL: session.URL}, nil
}

// CreateCheckout creates or replays a provider-hosted checkout session.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func (s *CheckoutService) CreateCheckout(ctx context.Context, req CheckoutRequest) (CheckoutResult, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return CheckoutResult{}, ErrMissingIdempotencyKey
	}
	plan, err := s.plan(req.Plan)
	if err != nil {
		return CheckoutResult{}, err
	}
	bodyHash, err := normalizedCheckoutBodyHash(req)
	if err != nil {
		return CheckoutResult{}, err
	}
	if s.store == nil || s.gateway == nil {
		return CheckoutResult{}, ErrStripeUnavailable
	}
	record, err := s.store.GetCheckoutIdempotency(ctx, req.UserID, req.Method, req.Route, req.IdempotencyKey)
	if err == nil {
		if record.BodyHash != bodyHash {
			return CheckoutResult{}, ErrIdempotencyConflict
		}
		var response CheckoutResponse
		if err := json.Unmarshal(record.ResponseBody, &response); err != nil {
			return CheckoutResult{}, err
		}
		return CheckoutResult{Response: response, StatusCode: record.StatusCode, Replayed: true}, nil
	}
	if !repository.IsKind(err, repository.ErrorKindNotFound) {
		return CheckoutResult{}, err
	}

	session, err := s.gateway.CreateCheckoutSession(ctx, CheckoutSessionRequest{UserID: req.UserID, Plan: plan.Code, PriceID: plan.PriceID, SuccessURL: req.SuccessURL, CancelURL: req.CancelURL, ProviderIdempotencyKey: stripeCheckoutIdempotencyKey(req)})
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("%w: %w", ErrStripeUnavailable, err)
	}
	response := CheckoutResponse{CheckoutSessionID: session.ID, CheckoutURL: session.URL, Plan: plan.Code, PriceID: plan.PriceID, AmountCents: plan.AmountCents}
	payload, err := json.Marshal(response)
	if err != nil {
		return CheckoutResult{}, err
	}
	if err := s.store.StoreCheckoutIdempotency(ctx, repository.CheckoutIdempotencyRecord{UserID: req.UserID, Method: req.Method, Route: req.Route, Key: req.IdempotencyKey, BodyHash: bodyHash, StatusCode: 200, ResponseBody: payload}); err != nil {
		if repository.IsKind(err, repository.ErrorKindConflict) {
			return CheckoutResult{}, ErrIdempotencyConflict
		}
		return CheckoutResult{}, err
	}
	return CheckoutResult{Response: response, StatusCode: 200}, nil
}

// plan maps public plan choices to configured Stripe price IDs.
// Implements DESIGN-007 SubscriptionController and SW-REQ-050 pricing tiers.
func (s *CheckoutService) plan(code string) (config.BillingPlan, error) {
	switch strings.TrimSpace(code) {
	case s.billing.MonthlyPlan.Code:
		return s.billing.MonthlyPlan, nil
	case s.billing.AnnualPlan.Code:
		return s.billing.AnnualPlan, nil
	default:
		return config.BillingPlan{}, ErrInvalidPlan
	}
}

// normalizedCheckoutBodyHash hashes only server-accepted checkout fields.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func normalizedCheckoutBodyHash(req CheckoutRequest) (string, error) {
	payload, err := json.Marshal(struct {
		Plan       string `json:"plan"`
		SuccessURL string `json:"successUrl"`
		CancelURL  string `json:"cancelUrl"`
	}{
		Plan:       strings.TrimSpace(req.Plan),
		SuccessURL: strings.TrimSpace(req.SuccessURL),
		CancelURL:  strings.TrimSpace(req.CancelURL),
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// stripeCheckoutIdempotencyKey derives a Stripe-safe key from the authenticated checkout scope.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func stripeCheckoutIdempotencyKey(req CheckoutRequest) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		req.UserID.String(),
		strings.TrimSpace(strings.ToUpper(req.Method)),
		strings.TrimSpace(req.Route),
		strings.TrimSpace(req.IdempotencyKey),
	}, "|")))
	return "mealswapp_checkout_" + hex.EncodeToString(sum[:])
}

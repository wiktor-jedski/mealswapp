package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"time"
)

// Implements DESIGN-007 SubscriptionController configuration tests.

func TestSubscriptionController_MapsPlans(t *testing.T) {
	cfg, _ := config.Load()
	cfg.Billing.MonthlyPlanPriceID = "price_123"
	cfg.Billing.AnnualPlanPriceID = "price_456"

	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(&fakeEntitlementRepo{}), subscription.NewUsageLimiter(&fakeEntitlementRepo{}, 3))
	if ctrl.plans["price_123"].AmountUS != 300 || ctrl.plans["price_123"].Label != "monthly" {
		t.Errorf("monthly plan incorrectly mapped: %+v", ctrl.plans["price_123"])
	}
	if ctrl.plans["price_456"].AmountUS != 2500 || ctrl.plans["price_456"].Label != "annual" {
		t.Errorf("annual plan incorrectly mapped: %+v", ctrl.plans["price_456"])
	}
}

func TestSubscriptionController_ValidateRedirectURLs(t *testing.T) {
	cfg, _ := config.Load()
	cfg.FrontendOrigin = "https://example.com"
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(&fakeEntitlementRepo{}), subscription.NewUsageLimiter(&fakeEntitlementRepo{}, 3))

	validReq := PaymentIntentRequest{
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
	}
	if err := ctrl.ValidateRedirectURLs(validReq); err != nil {
		t.Errorf("expected valid urls to pass, got %v", err)
	}

	invalidOriginReq := PaymentIntentRequest{
		SuccessURL: "https://evil.com/success",
		CancelURL:  "https://example.com/cancel",
	}
	if err := ctrl.ValidateRedirectURLs(invalidOriginReq); err == nil {
		t.Errorf("expected invalid origin to fail")
	}

	invalidFormatReq := PaymentIntentRequest{
		SuccessURL: "/relative/success",
		CancelURL:  "https://example.com/cancel",
	}
	if err := ctrl.ValidateRedirectURLs(invalidFormatReq); err == nil {
		t.Errorf("expected relative urls to fail")
	}
}

func TestPaymentIntentRequest_NoCardFields(t *testing.T) {
	payload := []byte(`{"priceId": "price_1", "successUrl": "http://a", "cancelUrl": "http://b", "cardNumber": "12345"}`)
	var req PaymentIntentRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if req.PriceID != "price_1" {
		t.Errorf("expected price_1")
	}
}

func TestSubscriptionController_CreateCheckout(t *testing.T) {
	cfg, _ := config.Load()
	cfg.FrontendOrigin = "https://example.com"
	cfg.Billing.MonthlyPlanPriceID = "price_monthly"
	cfg.Billing.AnnualPlanPriceID = "price_annual"

	gateway := &fakeCheckoutGateway{urls: []string{"https://checkout.stripe.com/123"}}
	ctrl := NewSubscriptionController(cfg, gateway, subscription.NewEntitlementManager(&fakeEntitlementRepo{}), subscription.NewUsageLimiter(&fakeEntitlementRepo{}, 3))

	app := fiber.New()
	app.Post("/checkout", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.CreateCheckout(c)
	})

	reqBody := PaymentIntentRequest{
		PriceID:    "price_monthly",
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Test 1: Successful creation
	req := httptest.NewRequest("POST", "/checkout", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idemp_1")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Test 2: Idempotency exact retry
	req2 := httptest.NewRequest("POST", "/checkout", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "idemp_1")
	resp2, _ := app.Test(req2)

	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	// Test 3: Idempotency reused with different body
	reqBodyDiff := reqBody
	reqBodyDiff.PriceID = "price_annual"
	bodyBytesDiff, _ := json.Marshal(reqBodyDiff)

	req3 := httptest.NewRequest("POST", "/checkout", bytes.NewReader(bodyBytesDiff))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Idempotency-Key", "idemp_1")
	resp3, _ := app.Test(req3)

	if resp3.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp3.StatusCode)
	}

	// Test 4: Missing Idempotency Key
	req4 := httptest.NewRequest("POST", "/checkout", bytes.NewReader(bodyBytes))
	req4.Header.Set("Content-Type", "application/json")
	resp4, _ := app.Test(req4)

	if resp4.StatusCode != 400 {
		t.Fatalf("expected 400 for missing idempotency key, got %d", resp4.StatusCode)
	}

	// Test 5: Gateway error maps to 503
	gateway2 := &fakeCheckoutGateway{err: errors.New("stripe down")}
	ctrl2 := NewSubscriptionController(cfg, gateway2, subscription.NewEntitlementManager(&fakeEntitlementRepo{}), subscription.NewUsageLimiter(&fakeEntitlementRepo{}, 3))
	app2 := fiber.New()
	app2.Post("/checkout", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl2.CreateCheckout(c)
	})

	req5 := httptest.NewRequest("POST", "/checkout", bytes.NewReader(bodyBytes))
	req5.Header.Set("Content-Type", "application/json")
	req5.Header.Set("Idempotency-Key", "idemp_2")
	resp5, _ := app2.Test(req5)

	if resp5.StatusCode != 503 {
		t.Fatalf("expected 503, got %d", resp5.StatusCode)
	}
}


type fakeEntitlementRepo struct {
	ent repository.Entitlement
	err error
	usage repository.UsageWindow
}

func (f *fakeEntitlementRepo) AppendEntitlement(ctx context.Context, entitlement repository.Entitlement) error { return nil }
func (f *fakeEntitlementRepo) GetLatest(ctx context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	if f.err != nil { return repository.Entitlement{}, f.err }
	return f.ent, nil
}
func (f *fakeEntitlementRepo) RecordUsage(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error) { return repository.UsageWindow{}, nil }
func (f *fakeEntitlementRepo) GetUsageSince(ctx context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error) { return f.usage, nil }
func (f *fakeEntitlementRepo) ListExpiredTrials(ctx context.Context, now time.Time) ([]repository.Entitlement, error) { return nil, nil }
func (f *fakeEntitlementRepo) InsertProcessedStripeEvent(ctx context.Context, event repository.ProcessedStripeEvent) (bool, error) { return false, nil }

func TestGetEntitlement_SuccessFree(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{
		ent: repository.Entitlement{
			Tier:   "free",
			Status: "active",
		},
		usage: repository.UsageWindow{SearchCount: 1},
	}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.GetEntitlement(c)
	})

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	if data["tier"] != "free" || data["status"] != "active" {
		t.Errorf("unexpected tier/status: %v", data)
	}
	if data["usageRemaining"].(float64) != 2 { // 3 - 1
		t.Errorf("expected 2 usage remaining, got %v", data["usageRemaining"])
	}
	modes := data["allowedModes"].([]interface{})
	if len(modes) != 2 {
		t.Errorf("expected 2 allowed modes for free, got %d", len(modes))
	}
}

func TestGetEntitlement_SuccessPaid(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{
		ent: repository.Entitlement{
			Tier:   "paid",
			Status: "active",
		},
	}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.GetEntitlement(c)
	})

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	
	if data["tier"] != "paid" || data["status"] != "active" {
		t.Errorf("unexpected tier/status: %v", data)
	}
	if data["usageRemaining"].(float64) == 0 { 
		t.Errorf("expected unlimited usage remaining for paid, got %v", data["usageRemaining"])
	}
	modes := data["allowedModes"].([]interface{})
	if len(modes) < 4 {
		t.Errorf("expected all allowed modes for paid, got %d", len(modes))
	}
}

func TestGetEntitlement_Anonymous(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", ctrl.GetEntitlement)

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}


func TestGetEntitlement_Trial(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{
		ent: repository.Entitlement{
			Tier:   "trial",
			Status: "active",
		},
	}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.GetEntitlement(c)
	})

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	if data["tier"] != "trial" || data["status"] != "active" {
		t.Errorf("unexpected tier/status: %v", data)
	}
	if data["usageRemaining"].(float64) == 0 { 
		t.Errorf("expected unlimited usage remaining for trial, got %v", data["usageRemaining"])
	}
}

func TestGetEntitlement_PastDue(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{
		ent: repository.Entitlement{
			Tier:   "paid",
			Status: "past_due",
		},
		usage: repository.UsageWindow{SearchCount: 1},
	}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.GetEntitlement(c)
	})

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	if data["tier"] != "paid" || data["status"] != "past_due" {
		t.Errorf("unexpected tier/status: %v", data)
	}
	// "past_due" behaves like free for usage limit
	if data["usageRemaining"].(float64) != 2 { 
		t.Errorf("expected 2 usage remaining for past_due, got %v", data["usageRemaining"])
	}
	modes := data["allowedModes"].([]interface{})
	if len(modes) != 2 {
		t.Errorf("expected 2 allowed modes for past_due, got %d", len(modes))
	}
}

func TestGetEntitlement_Cancelled(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{
		ent: repository.Entitlement{
			Tier:   "paid",
			Status: "cancelled",
		},
		usage: repository.UsageWindow{SearchCount: 1},
	}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.GetEntitlement(c)
	})

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	if data["tier"] != "paid" || data["status"] != "cancelled" {
		t.Errorf("unexpected tier/status: %v", data)
	}
	// "cancelled" behaves like free for usage limit
	if data["usageRemaining"].(float64) != 2 { 
		t.Errorf("expected 2 usage remaining for cancelled, got %v", data["usageRemaining"])
	}
	modes := data["allowedModes"].([]interface{})
	if len(modes) != 2 {
		t.Errorf("expected 2 allowed modes for cancelled, got %d", len(modes))
	}
}

func TestGetEntitlement_NoStripeSecrets(t *testing.T) {
	cfg, _ := config.Load()
	repo := &fakeEntitlementRepo{
		ent: repository.Entitlement{
			Tier:   "free",
			Status: "active",
			StripeCustomerID: "cus_123",
			StripeSubscriptionID: "sub_123",
		},
	}
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{}, subscription.NewEntitlementManager(repo), subscription.NewUsageLimiter(repo, 3))
	app := fiber.New()
	app.Get("/entitlements", func(c *fiber.Ctx) error {
		c.Locals(authenticatedUserLocal, AuthenticatedUser{UserID: uuid.New()})
		return ctrl.GetEntitlement(c)
	})

	req := httptest.NewRequest("GET", "/entitlements", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	
	// Test the envelope
	if _, ok := body["data"]; !ok {
		t.Errorf("missing data field")
	}
	if _, ok := body["status"]; !ok {
		t.Errorf("missing status field")
	}
	
	for key := range body {
		if key != "data" && key != "status" && key != "requestId" {
			t.Errorf("unexpected envelope field: %s", key)
		}
	}

	data := body["data"].(map[string]interface{})
	
	// Validate only expected data fields are present
	expectedFields := map[string]bool{
		"tier": true,
		"status": true,
		"allowedModes": true,
		"searchLimitPer24h": true,
		"usageRemaining": true,
		"expiresAt": true,
	}

	for key := range data {
		if !expectedFields[key] {
			t.Errorf("unexpected field in data: %s - possible secret leak", key)
		}
	}
}


func (f *fakeEntitlementRepo) GetLatestByStripeCustomer(ctx context.Context, customerID string) (repository.Entitlement, error) {
	return repository.Entitlement{}, nil
}

func (f *fakeEntitlementRepo) GetLatestByStripeSubscription(ctx context.Context, subscriptionID string) (repository.Entitlement, error) {
	return repository.Entitlement{}, nil
}

package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
)

// Implements DESIGN-007 SubscriptionController configuration tests.

func TestSubscriptionController_MapsPlans(t *testing.T) {
	cfg, _ := config.Load()
	cfg.Billing.MonthlyPlanPriceID = "price_123"
	cfg.Billing.AnnualPlanPriceID = "price_456"

	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{})
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
	ctrl := NewSubscriptionController(cfg, &fakeCheckoutGateway{})

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
	ctrl := NewSubscriptionController(cfg, gateway)

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
	ctrl2 := NewSubscriptionController(cfg, gateway2)
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

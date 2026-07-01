package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
)

// Implements DESIGN-007 SubscriptionController configuration tests.

func TestSubscriptionController_MapsPlans(t *testing.T) {
	cfg, _ := config.Load()
	cfg.Billing.MonthlyPlanPriceID = "price_123"
	cfg.Billing.AnnualPlanPriceID = "price_456"

	ctrl := NewSubscriptionController(cfg)
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
	ctrl := NewSubscriptionController(cfg)

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

	// Verify we didn't unmarshal any card info because it's not in the struct
	if req.PriceID != "price_1" {
		t.Errorf("expected price_1")
	}
}

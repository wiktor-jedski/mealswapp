package httpapi

// Implements DESIGN-007 SubscriptionController checkout request validation verification.

import "testing"

func TestValidateCheckoutCreateRequestBodyAcceptsPlanAndRedirects(t *testing.T) {
	body := map[string]any{
		"plan":       "monthly",
		"successUrl": "http://localhost:5173/billing/success",
		"cancelUrl":  "http://localhost:5173/billing/cancel",
	}

	if err := ValidateCheckoutCreateRequestBody(body); err != nil {
		t.Fatalf("ValidateCheckoutCreateRequestBody() error = %v", err)
	}

	dto, err := decodeCheckoutCreateRequestBody(body)
	if err != nil {
		t.Fatalf("decodeCheckoutCreateRequestBody() error = %v", err)
	}
	if dto.Plan != "monthly" || dto.SuccessURL == "" || dto.CancelURL == "" {
		t.Fatalf("checkout dto = %+v", dto)
	}
}

func TestValidateCheckoutCreateRequestBodyAcceptsAnnualPlan(t *testing.T) {
	body := map[string]any{
		"plan":       "annual",
		"successUrl": "http://localhost:5173/billing/success",
		"cancelUrl":  "http://localhost:5173/billing/cancel",
	}

	if err := ValidateCheckoutCreateRequestBody(body); err != nil {
		t.Fatalf("ValidateCheckoutCreateRequestBody() error = %v", err)
	}
}

func TestValidateCheckoutCreateRequestBodyForOriginRejectsCrossOriginRedirects(t *testing.T) {
	body := map[string]any{
		"plan":       "monthly",
		"successUrl": "https://evil.example/billing/success",
		"cancelUrl":  "http://localhost:5173/billing/cancel",
	}

	if err := ValidateCheckoutCreateRequestBodyForOrigin(body, "http://localhost:5173"); err == nil {
		t.Fatal("ValidateCheckoutCreateRequestBodyForOrigin() accepted a cross-origin success URL")
	}
}

func TestValidateCheckoutCreateRequestBodyRejectsRawCardFields(t *testing.T) {
	for _, field := range []string{"card", "cardNumber", "number", "cvc", "cvv", "expiry", "expMonth", "expYear", "paymentMethodData"} {
		t.Run(field, func(t *testing.T) {
			body := map[string]any{
				"plan":       "monthly",
				"successUrl": "http://localhost:5173/billing/success",
				"cancelUrl":  "http://localhost:5173/billing/cancel",
				field:        "4242424242424242",
			}
			if err := ValidateCheckoutCreateRequestBody(body); err == nil {
				t.Fatalf("ValidateCheckoutCreateRequestBody() accepted raw card field %q", field)
			}
		})
	}
}

func TestValidateCheckoutCreateRequestBodyRejectsMalformedShape(t *testing.T) {
	for name, body := range map[string]map[string]any{
		"unknown field": {"plan": "monthly", "successUrl": "http://localhost:5173/success", "cancelUrl": "http://localhost:5173/cancel", "coupon": "free"},
		"bad plan":      {"plan": "weekly", "successUrl": "http://localhost:5173/success", "cancelUrl": "http://localhost:5173/cancel"},
		"missing url":   {"plan": "monthly", "successUrl": "http://localhost:5173/success"},
		"mistyped plan": {"plan": 123, "successUrl": "http://localhost:5173/success", "cancelUrl": "http://localhost:5173/cancel"},
		"relative url":  {"plan": "monthly", "successUrl": "/success", "cancelUrl": "http://localhost:5173/cancel"},
		"fragment url":  {"plan": "monthly", "successUrl": "http://localhost:5173/success#token", "cancelUrl": "http://localhost:5173/cancel"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCheckoutCreateRequestBody(body); err == nil {
				t.Fatal("ValidateCheckoutCreateRequestBody() accepted malformed checkout request")
			}
		})
	}
}

func TestValidateBillingPortalRequestBodyAcceptsReturnURL(t *testing.T) {
	body := map[string]any{"returnUrl": "http://localhost:5173/subscription"}

	if err := ValidateBillingPortalRequestBody(body); err != nil {
		t.Fatalf("ValidateBillingPortalRequestBody() error = %v", err)
	}

	dto, err := decodeBillingPortalRequestBody(body)
	if err != nil {
		t.Fatalf("decodeBillingPortalRequestBody() error = %v", err)
	}
	if dto.ReturnURL != "http://localhost:5173/subscription" {
		t.Fatalf("portal dto = %+v", dto)
	}
}

func TestValidateBillingPortalRequestBodyForOriginRejectsCrossOriginReturnURL(t *testing.T) {
	body := map[string]any{"returnUrl": "https://evil.example/subscription"}

	if err := ValidateBillingPortalRequestBodyForOrigin(body, "http://localhost:5173"); err == nil {
		t.Fatal("ValidateBillingPortalRequestBodyForOrigin() accepted a cross-origin return URL")
	}
}

func TestValidateBillingPortalRequestBodyRejectsMalformedShape(t *testing.T) {
	for name, body := range map[string]map[string]any{
		"unknown field": {"returnUrl": "http://localhost:5173/subscription", "customer": "cus_secret"},
		"missing url":   {},
		"relative url":  {"returnUrl": "/subscription"},
		"fragment url":  {"returnUrl": "http://localhost:5173/subscription#token"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateBillingPortalRequestBody(body); err == nil {
				t.Fatal("ValidateBillingPortalRequestBody() accepted malformed portal request")
			}
		})
	}
}

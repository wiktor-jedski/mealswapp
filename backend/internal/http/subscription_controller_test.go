package http

import (
	"context"
	"net/http"
	"testing"

	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/handlers"
	"mealswapp/backend/internal/services/entitlements"
	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/google/uuid"
)

func TestSubscriptionControllerRequiresAuthenticatedAccess(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		SubscriptionService: &fakeSubscriptionService{},
	})

	res := performRequest(t, app, http.MethodGet, "/api/v1/subscription/status")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated status request to fail, got %d", res.StatusCode)
	}
}

func TestSubscriptionControllerReturnsStatusAndEntitlementShapes(t *testing.T) {
	service := &fakeSubscriptionService{entitlement: testEntitlement()}
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		SubscriptionService: service,
	})

	statusRes := performJSONRequest(t, app, http.MethodGet, "/api/v1/subscription/status", "", "access-token", false)
	defer statusRes.Body.Close()
	if statusRes.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusRes.StatusCode)
	}
	status := dataMap(t, decodeEnvelope(t, statusRes).Data)
	if status["billingState"] != "active" {
		t.Fatalf("expected active billing state, got %#v", status)
	}
	entitlement := status["entitlement"].(map[string]any)
	if entitlement["tier"] != "paid" || entitlement["searchLimitPer24h"].(float64) != -1 {
		t.Fatalf("unexpected entitlement shape: %#v", entitlement)
	}
	if service.lastStatusToken != "access-token" {
		t.Fatalf("expected service token propagation, got %q", service.lastStatusToken)
	}

	entitlementRes := performJSONRequest(t, app, http.MethodGet, "/api/v1/subscription/entitlement", "", "access-token", false)
	defer entitlementRes.Body.Close()
	if entitlementRes.StatusCode != http.StatusOK {
		t.Fatalf("expected entitlement 200, got %d", entitlementRes.StatusCode)
	}
	data := dataMap(t, decodeEnvelope(t, entitlementRes).Data)
	if data["tier"] != "paid" || len(data["allowedModes"].([]any)) != 3 {
		t.Fatalf("unexpected entitlement response: %#v", data)
	}
}

func TestSubscriptionControllerCreatesCheckoutAndPortalWithStripeStub(t *testing.T) {
	service := &fakeSubscriptionService{entitlement: testEntitlement()}
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		SubscriptionService: service,
	})

	checkoutRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/subscription/checkout", `{
		"priceId":"price_monthly",
		"successUrl":"https://example.test/success",
		"cancelUrl":"https://example.test/cancel"
	}`, "access-token", true)
	defer checkoutRes.Body.Close()
	if checkoutRes.StatusCode != http.StatusCreated {
		t.Fatalf("expected checkout 201, got %d", checkoutRes.StatusCode)
	}
	checkout := dataMap(t, decodeEnvelope(t, checkoutRes).Data)
	if checkout["url"] != "https://stripe.test/checkout/session" || service.lastCheckout.PriceID != "price_monthly" {
		t.Fatalf("unexpected checkout response/service call: data=%#v service=%#v", checkout, service.lastCheckout)
	}

	portalRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/subscription/portal", `{"returnUrl":"https://example.test/account"}`, "access-token", true)
	defer portalRes.Body.Close()
	if portalRes.StatusCode != http.StatusOK {
		t.Fatalf("expected portal 200, got %d", portalRes.StatusCode)
	}
	portal := dataMap(t, decodeEnvelope(t, portalRes).Data)
	if portal["url"] != "https://stripe.test/customer-portal" || service.lastReturnURL != "https://example.test/account" {
		t.Fatalf("unexpected portal response/service call: data=%#v return=%q", portal, service.lastReturnURL)
	}
}

func TestSubscriptionControllerValidatesCheckoutPayload(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		SubscriptionService: &fakeSubscriptionService{},
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/subscription/checkout", `{"priceId":"price_monthly"}`, "access-token", true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected validation 400, got %d", res.StatusCode)
	}
	payload := decodeEnvelope(t, res)
	if payload.Error == nil || payload.Error.Code != "validation_error" {
		t.Fatalf("expected validation envelope, got %#v", payload)
	}
}

type fakeSubscriptionService struct {
	entitlement     entitlements.Entitlement
	lastStatusToken string
	lastCheckout    handlers.CheckoutRequest
	lastReturnURL   string
}

func (service *fakeSubscriptionService) GetStatus(ctx context.Context, accessToken string) (handlers.SubscriptionStatus, error) {
	service.lastStatusToken = accessToken
	return handlers.SubscriptionStatus{
		Entitlement:  service.entitlement,
		BillingState: string(service.entitlement.Status),
		Plans: []entitlements.Plan{
			{ID: "paid_monthly", Tier: entitlements.TierPaid, Interval: "monthly", PriceCents: 300},
			{ID: "paid_annual", Tier: entitlements.TierPaid, Interval: "annual", PriceCents: 2500},
		},
	}, nil
}

func (service *fakeSubscriptionService) CreateCheckout(ctx context.Context, accessToken string, request handlers.CheckoutRequest) (handlers.CheckoutSession, error) {
	service.lastCheckout = request
	return handlers.CheckoutSession{ID: "cs_test_123", URL: "https://stripe.test/checkout/session"}, nil
}

func (service *fakeSubscriptionService) CreateCustomerPortal(ctx context.Context, accessToken string, returnURL string) (handlers.CustomerPortalSession, error) {
	service.lastReturnURL = returnURL
	return handlers.CustomerPortalSession{URL: "https://stripe.test/customer-portal"}, nil
}

func (service *fakeSubscriptionService) GetEntitlement(ctx context.Context, accessToken string) (entitlements.Entitlement, error) {
	return service.entitlement, nil
}

func testEntitlement() entitlements.Entitlement {
	return entitlements.Entitlement{
		UserID:            uuid.MustParse("00000000-0000-0000-0000-000000000777"),
		Tier:              entitlements.TierPaid,
		Status:            entitlements.StatusActive,
		SearchLimitPer24h: -1,
		AllowedModes:      []searchsvc.Mode{searchsvc.ModeSingle, searchsvc.ModeReplacement, searchsvc.ModeDiet},
		AllowedFeatures:   []entitlements.Feature{entitlements.FeatureSingle, entitlements.FeatureIngredient, entitlements.FeatureMeal, entitlements.FeatureDiet},
	}
}

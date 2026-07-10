package httpapi

// Implements DESIGN-007 SubscriptionController checkout HTTP verification.

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

type fakeCheckoutCreator struct {
	result  subscription.CheckoutResult
	err     error
	gotReq  subscription.CheckoutRequest
	calls   int
	results []subscription.CheckoutResult
	errors  []error
}

type fakeBillingPortalCreator struct {
	result subscription.PortalResponse
	err    error
	gotReq subscription.PortalRequest
	calls  int
}

func (s *fakeBillingPortalCreator) CreateBillingPortal(_ context.Context, req subscription.PortalRequest) (subscription.PortalResponse, error) {
	s.calls++
	s.gotReq = req
	if s.err != nil {
		return subscription.PortalResponse{}, s.err
	}
	return s.result, nil
}

func (s *fakeCheckoutCreator) CreateCheckout(_ context.Context, req subscription.CheckoutRequest) (subscription.CheckoutResult, error) {
	s.calls++
	s.gotReq = req
	if len(s.errors) >= s.calls {
		return subscription.CheckoutResult{}, s.errors[s.calls-1]
	}
	if s.err != nil {
		return subscription.CheckoutResult{}, s.err
	}
	if len(s.results) >= s.calls {
		return s.results[s.calls-1], nil
	}
	return s.result, nil
}

type httpMemoryCheckoutStore struct {
	records map[string]repository.CheckoutIdempotencyRecord
}

func (s *httpMemoryCheckoutStore) GetCheckoutIdempotency(_ context.Context, userID uuid.UUID, method string, route string, key string) (repository.CheckoutIdempotencyRecord, error) {
	record, ok := s.records[userID.String()+"|"+method+"|"+route+"|"+key]
	if !ok {
		return repository.CheckoutIdempotencyRecord{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return record, nil
}

func (s *httpMemoryCheckoutStore) StoreCheckoutIdempotency(_ context.Context, record repository.CheckoutIdempotencyRecord) error {
	s.records[record.UserID.String()+"|"+record.Method+"|"+record.Route+"|"+record.Key] = record
	return nil
}

type httpFakeCheckoutGateway struct {
	calls int
}

func (g *httpFakeCheckoutGateway) CreateCheckoutSession(context.Context, subscription.CheckoutSessionRequest) (subscription.CheckoutSession, error) {
	g.calls++
	return subscription.CheckoutSession{ID: "cs_test_http", URL: "https://checkout.stripe.test/http"}, nil
}

type httpEntitlementStatusStore struct {
	entitlements map[uuid.UUID]repository.Entitlement
	usageCount   int
	gotUsageUser uuid.UUID
}

func (s *httpEntitlementStatusStore) AppendEntitlement(_ context.Context, entitlement repository.Entitlement) error {
	if s.entitlements == nil {
		s.entitlements = map[uuid.UUID]repository.Entitlement{}
	}
	s.entitlements[entitlement.UserID] = entitlement
	return nil
}

func (s *httpEntitlementStatusStore) GetLatest(_ context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	entitlement, ok := s.entitlements[userID]
	if !ok {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "entitlement not found", nil)
	}
	return entitlement, nil
}

func (s *httpEntitlementStatusStore) RecordUsage(_ context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error) {
	return repository.UsageWindow{UserID: userID, Feature: feature, StartedAt: occurredAt, SearchCount: s.usageCount}, nil
}

func (s *httpEntitlementStatusStore) RecordUsageWithinLimit(_ context.Context, userID uuid.UUID, feature string, occurredAt time.Time, _ time.Time, _ int) (repository.UsageWindow, bool, error) {
	return repository.UsageWindow{UserID: userID, Feature: feature, StartedAt: occurredAt, SearchCount: s.usageCount}, true, nil
}

func (s *httpEntitlementStatusStore) GetUsageSince(_ context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error) {
	s.gotUsageUser = userID
	return repository.UsageWindow{UserID: userID, Feature: feature, StartedAt: since, SearchCount: s.usageCount}, nil
}

func TestSubscriptionControllerRoutesRequireBillingRedirectOrigin(t *testing.T) {
	for name, origin := range map[string]string{"missing": "", "blank": " \t"} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("Routes() accepted a missing billing redirect origin")
				}
			}()
			NewSubscriptionController(nil).WithBillingRedirectOrigin(origin).Routes()
		})
	}
}

func TestSubscriptionControllerCreatesCheckoutFromAuthenticatedCookies(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCheckoutCreator{result: subscription.CheckoutResult{StatusCode: fiber.StatusOK, Response: subscription.CheckoutResponse{CheckoutSessionID: "cs_test_123", CheckoutURL: "https://checkout.stripe.test/session", Plan: "monthly", PriceID: "price_monthly_test", AmountCents: 300}}}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewSubscriptionController(service).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"plan":"monthly","successUrl":"http://localhost:5173/billing/success","cancelUrl":"http://localhost:5173/billing/cancel"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	req.Header.Set("Idempotency-Key", "checkout-123")
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["checkoutUrl"] != "https://checkout.stripe.test/session" {
		t.Fatalf("checkout response = %d body=%+v", resp.StatusCode, body)
	}
	if service.gotReq.UserID != userID || service.gotReq.IdempotencyKey != "checkout-123" {
		t.Fatalf("service request = %+v", service.gotReq)
	}
	if _, ok := body.Data["cardNumber"]; ok {
		t.Fatalf("checkout response leaked card field: %+v", body.Data)
	}
}

func TestSubscriptionControllerCreatesBillingPortalFromAuthenticatedCookies(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	portal := &fakeBillingPortalCreator{result: subscription.PortalResponse{PortalURL: "https://billing.stripe.test/session"}}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewSubscriptionController(nil).WithBillingPortal(portal).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/portal", strings.NewReader(`{"returnUrl":"http://localhost:5173/subscription"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Data["portalUrl"] != "https://billing.stripe.test/session" {
		t.Fatalf("portal response = %d body=%+v", resp.StatusCode, body)
	}
	if portal.gotReq.UserID != userID || portal.gotReq.ReturnURL != "http://localhost:5173/subscription" {
		t.Fatalf("portal request = %+v", portal.gotReq)
	}
}

func TestSubscriptionControllerRejectsBillingPortalWithoutActiveSubscription(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
	portal := &fakeBillingPortalCreator{err: subscription.ErrNoActiveSubscription}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewSubscriptionController(nil).WithBillingPortal(portal).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/portal", strings.NewReader(`{"returnUrl":"http://localhost:5173/subscription"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusConflict || body.Error == nil || body.Error.Code != "billing_portal_unavailable" {
		t.Fatalf("portal conflict response = %d body=%+v", resp.StatusCode, body)
	}
}

func TestSubscriptionControllerReadsAuthenticatedEntitlementStatus(t *testing.T) {
	fixedNow := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		entitlement   func(uuid.UUID) repository.Entitlement
		usageCount    int
		wantTier      string
		wantStatus    string
		wantModes     []string
		wantRemaining any
		wantRecovery  string
		wantTrial     bool
	}{
		{
			name:          "missing entitlement falls back to free",
			usageCount:    2,
			wantTier:      "free",
			wantStatus:    "active",
			wantModes:     []string{"catalog", "substitution"},
			wantRemaining: float64(1),
			wantRecovery:  "none",
		},
		{
			name: "active trial exposes paid modes and expiry",
			entitlement: func(userID uuid.UUID) repository.Entitlement {
				expiresAt := fixedNow.Add(48 * time.Hour)
				return repository.Entitlement{UserID: userID, Tier: "trial", Status: "active", SearchLimitPer24h: 0, AllowedModes: []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"}, ExpiresAt: &expiresAt, StripeCustomerID: "cus_secret", StripeSubscriptionID: "sub_secret"}
			},
			wantTier:      "trial",
			wantStatus:    "active",
			wantModes:     []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"},
			wantRemaining: nil,
			wantRecovery:  "none",
			wantTrial:     true,
		},
		{
			name: "active paid exposes paid modes",
			entitlement: func(userID uuid.UUID) repository.Entitlement {
				return repository.Entitlement{UserID: userID, Tier: "paid", Status: "active", SearchLimitPer24h: 0, AllowedModes: []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"}, StripeCustomerID: "cus_secret", StripeSubscriptionID: "sub_secret"}
			},
			wantTier:      "paid",
			wantStatus:    "active",
			wantModes:     []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"},
			wantRemaining: nil,
			wantRecovery:  "none",
		},
		{
			name: "past due keeps free visible modes and recovery state",
			entitlement: func(userID uuid.UUID) repository.Entitlement {
				return repository.Entitlement{UserID: userID, Tier: "paid", Status: "past_due", SearchLimitPer24h: 0, AllowedModes: []string{"catalog", "substitution"}, StripeCustomerID: "cus_secret", StripeSubscriptionID: "sub_secret"}
			},
			wantTier:      "paid",
			wantStatus:    "past_due",
			wantModes:     []string{"catalog", "substitution"},
			wantRemaining: nil,
			wantRecovery:  "action_required",
		},
		{
			name: "cancelled keeps free visible modes and cancellation state",
			entitlement: func(userID uuid.UUID) repository.Entitlement {
				return repository.Entitlement{UserID: userID, Tier: "paid", Status: "cancelled", SearchLimitPer24h: 0, AllowedModes: []string{"catalog", "substitution"}, StripeCustomerID: "cus_secret", StripeSubscriptionID: "sub_secret"}
			},
			wantTier:      "paid",
			wantStatus:    "cancelled",
			wantModes:     []string{"catalog", "substitution"},
			wantRemaining: nil,
			wantRecovery:  "cancelled",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			userID := uuid.New()
			authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
			store := &httpEntitlementStatusStore{entitlements: map[uuid.UUID]repository.Entitlement{}, usageCount: tc.usageCount}
			if tc.entitlement != nil {
				store.entitlements[userID] = tc.entitlement(userID)
			}
			statusReader := entitlement.NewStatusServiceWithClock(store, store, func() time.Time { return fixedNow })
			app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSubscriptionController(nil, statusReader).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})

			req := httptest.NewRequest(fiber.MethodGet, "/api/v1/billing/entitlement", nil)
			addCookies(req, authCookies)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			body := decodeEnvelope(t, resp.Body)
			resp.Body.Close()

			if resp.StatusCode != fiber.StatusOK || body.Status != "ok" {
				t.Fatalf("entitlement response = %d body=%+v", resp.StatusCode, body)
			}
			assertEntitlementStatusEnvelope(t, body.Data, userID, tc.wantTier, tc.wantStatus, tc.wantModes, tc.wantRemaining, tc.wantRecovery, tc.wantTrial)
			if _, ok := body.Data["stripeCustomerId"]; ok {
				t.Fatalf("entitlement leaked stripe customer id: %+v", body.Data)
			}
			if _, ok := body.Data["stripeSubscriptionId"]; ok {
				t.Fatalf("entitlement leaked stripe subscription id: %+v", body.Data)
			}
			if tc.wantTier == "free" && store.gotUsageUser != userID {
				t.Fatalf("usage lookup user = %s, want %s", store.gotUsageUser, userID)
			}
		})
	}
}

func TestSubscriptionControllerRejectsAnonymousEntitlementStatus(t *testing.T) {
	store := &httpEntitlementStatusStore{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSubscriptionController(nil, entitlement.NewStatusService(store, store)).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/billing/entitlement", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || body.Error == nil || body.Error.Code != "unauthorized" {
		t.Fatalf("anonymous entitlement response = %d body=%+v", resp.StatusCode, body)
	}
}

func TestSubscriptionControllerEntitlementStatusEnvelopeMatchesGeneratedContract(t *testing.T) {
	// Implements DESIGN-007 SubscriptionController and DESIGN-017 ErrorMessageMapper generated contract verification.
	openapi := readRepoFile(t, "api/openapi.yaml")
	generated := readRepoFile(t, "frontend/src/lib/api/generated.ts")
	requiredFields := openAPIRequiredFields(t, openapi, "EntitlementStatusData")
	assertOpenAPIEntitlementEndpoint(t, openapi)
	assertGeneratedEntitlementContract(t, generated, requiredFields)

	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	expiresAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	store := &httpEntitlementStatusStore{
		entitlements: map[uuid.UUID]repository.Entitlement{
			userID: {
				UserID:            userID,
				Tier:              "trial",
				Status:            "active",
				SearchLimitPer24h: 0,
				AllowedModes:      []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"},
				ExpiresAt:         &expiresAt,
			},
		},
	}
	statusReader := entitlement.NewStatusServiceWithClock(store, store, func() time.Time { return expiresAt.Add(-24 * time.Hour) })
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSubscriptionController(nil, statusReader).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/billing/entitlement", nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || body.Status != "ok" || body.RequestID == "" {
		t.Fatalf("entitlement contract envelope = %d body=%+v", resp.StatusCode, body)
	}
	assertEntitlementDataMatchesContracts(t, body.Data, requiredFields, openapi, generated)
}

func TestSubscriptionControllerMapsIdempotencyAndStripeErrors(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "missing key", err: subscription.ErrMissingIdempotencyKey, wantStatus: fiber.StatusBadRequest, wantCode: "idempotency_key_required"},
		{name: "conflict", err: subscription.ErrIdempotencyConflict, wantStatus: fiber.StatusConflict, wantCode: "idempotency_key_conflict"},
		{name: "stripe unavailable", err: subscription.ErrStripeUnavailable, wantStatus: fiber.StatusServiceUnavailable, wantCode: "stripe_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
			app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewSubscriptionController(&fakeCheckoutCreator{err: tc.err}).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
			token, csrfCookies := fetchCSRFToken(t, app)
			req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"plan":"annual","successUrl":"http://localhost:5173/billing/success","cancelUrl":"http://localhost:5173/billing/cancel"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", token)
			req.Header.Set("Idempotency-Key", "checkout-123")
			addCookies(req, csrfCookies)
			addCookies(req, authCookies)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			body := decodeEnvelope(t, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != tc.wantStatus || body.Error == nil || body.Error.Code != tc.wantCode {
				t.Fatalf("response = %d body=%+v", resp.StatusCode, body)
			}
		})
	}
}

func TestSubscriptionControllerReplaysExactCheckoutAndRejectsBodyConflict(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	gateway := &httpFakeCheckoutGateway{}
	service := subscription.NewCheckoutService(httpTestBillingConfig(), &httpMemoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}, gateway)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewSubscriptionController(service).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	postCheckout := func(body string, key string) (int, Envelope) {
		t.Helper()
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", token)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		addCookies(req, csrfCookies)
		addCookies(req, authCookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		envelope := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		return resp.StatusCode, envelope
	}

	body := `{"plan":"monthly","successUrl":"http://localhost:5173/billing/success","cancelUrl":"http://localhost:5173/billing/cancel"}`
	status, first := postCheckout(body, "checkout-replay")
	if status != fiber.StatusOK || first.Data["checkoutSessionId"] != "cs_test_http" {
		t.Fatalf("first checkout = %d body=%+v", status, first)
	}
	status, second := postCheckout(body, "checkout-replay")
	if status != fiber.StatusOK || second.Data["checkoutSessionId"] != "cs_test_http" || gateway.calls != 1 {
		t.Fatalf("replay checkout = %d body=%+v gateway calls=%d", status, second, gateway.calls)
	}
	status, conflict := postCheckout(`{"plan":"monthly","successUrl":"http://localhost:5173/billing/other","cancelUrl":"http://localhost:5173/billing/cancel"}`, "checkout-replay")
	if status != fiber.StatusConflict || conflict.Error == nil || conflict.Error.Code != "idempotency_key_conflict" {
		t.Fatalf("conflict checkout = %d body=%+v", status, conflict)
	}
	status, missing := postCheckout(body, "")
	if status != fiber.StatusBadRequest || missing.Error == nil || missing.Error.Code != "idempotency_key_required" {
		t.Fatalf("missing key checkout = %d body=%+v", status, missing)
	}
}

func TestSubscriptionControllerRejectsRawCardFieldsBeforeService(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	cfg := testConfig()
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
	service := &fakeCheckoutCreator{}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewSubscriptionController(service).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"plan":"monthly","successUrl":"http://localhost:5173/billing/success","cancelUrl":"http://localhost:5173/billing/cancel","cardNumber":"4242424242424242","cvc":"123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	req.Header.Set("Idempotency-Key", "checkout-123")
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || service.calls != 0 {
		t.Fatalf("raw card response = %d service calls=%d", resp.StatusCode, service.calls)
	}
}

func httpTestBillingConfig() config.BillingConfig {
	return config.BillingConfig{
		MonthlyPlan: config.BillingPlan{Code: "monthly", Label: "Monthly", AmountCents: 300, PriceID: "price_monthly_test"},
		AnnualPlan:  config.BillingPlan{Code: "annual", Label: "Annual", AmountCents: 2500, PriceID: "price_annual_test"},
	}
}

func assertEntitlementStatusEnvelope(t *testing.T, data map[string]any, userID uuid.UUID, tier string, status string, modes []string, remaining any, recovery string, wantTrial bool) {
	t.Helper()
	required := []string{"userId", "tier", "status", "allowedModes", "searchLimitPer24h", "usageUsed", "usageRemaining", "usageWindowStartedAt", "trialExpiresAt", "billingRecoveryState"}
	for _, field := range required {
		if _, ok := data[field]; !ok {
			t.Fatalf("entitlement envelope missing %s: %+v", field, data)
		}
	}
	if data["userId"] != userID.String() || data["tier"] != tier || data["status"] != status || data["billingRecoveryState"] != recovery {
		t.Fatalf("entitlement identity/state data = %+v", data)
	}
	gotModes, ok := data["allowedModes"].([]any)
	if !ok || len(gotModes) != len(modes) {
		t.Fatalf("allowedModes = %#v, want %v", data["allowedModes"], modes)
	}
	for i, mode := range modes {
		if gotModes[i] != mode {
			t.Fatalf("allowedModes[%d] = %v, want %s", i, gotModes[i], mode)
		}
	}
	if data["usageRemaining"] != remaining {
		t.Fatalf("usageRemaining = %#v, want %#v", data["usageRemaining"], remaining)
	}
	if wantTrial {
		if data["trialExpiresAt"] == nil {
			t.Fatalf("trialExpiresAt missing for trial response: %+v", data)
		}
		return
	}
	if data["trialExpiresAt"] != nil {
		t.Fatalf("trialExpiresAt = %#v, want nil", data["trialExpiresAt"])
	}
}

func TestSubscriptionControllerRejectsUnauthenticatedCheckout(t *testing.T) {
	service := &fakeCheckoutCreator{err: errors.New("should not be called")}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), CSRF: NewCSRFManager(testConfig(), nil), Routes: NewSubscriptionController(service).WithBillingRedirectOrigin(testBillingRedirectOrigin).Routes()})
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"plan":"monthly","successUrl":"http://localhost:5173/billing/success","cancelUrl":"http://localhost:5173/billing/cancel"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("unauthenticated response = %d service calls=%d", resp.StatusCode, service.calls)
	}
}

func readRepoFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile("../../../" + path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertOpenAPIEntitlementEndpoint(t *testing.T, openapi string) {
	t.Helper()
	if !strings.Contains(openapi, "/api/v1/billing/entitlement:") ||
		!strings.Contains(openapi, "$ref: \"#/components/responses/EntitlementStatus\"") ||
		!strings.Contains(openapi, "$ref: \"#/components/schemas/EntitlementStatusEnvelope\"") {
		t.Fatal("OpenAPI entitlement endpoint or envelope reference is missing")
	}
}

func assertGeneratedEntitlementContract(t *testing.T, generated string, requiredFields []string) {
	t.Helper()
	if !strings.Contains(generated, "export type EntitlementStatusEnvelope = Envelope<EntitlementStatusData>;") {
		t.Fatal("generated EntitlementStatusEnvelope alias is missing")
	}
	interfaceBlock := namedBlock(t, generated, "export interface EntitlementStatusData")
	for _, field := range requiredFields {
		if !strings.Contains(interfaceBlock, "\t"+field+":") {
			t.Fatalf("generated EntitlementStatusData is missing required OpenAPI field %s", field)
		}
	}
}

func assertEntitlementDataMatchesContracts(t *testing.T, data map[string]any, requiredFields []string, openapi string, generated string) {
	t.Helper()
	if len(data) != len(requiredFields) {
		t.Fatalf("entitlement data has %d fields, want %d from OpenAPI required list: %+v", len(data), len(requiredFields), data)
	}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Fatalf("entitlement data missing OpenAPI required field %s: %+v", field, data)
		}
	}

	assertStringInContracts(t, data["tier"], openAPIEnum(t, openapi, "SubscriptionTier"), generatedUnion(t, generated, "SubscriptionTier"), "tier")
	assertStringInContracts(t, data["status"], openAPIEnum(t, openapi, "EntitlementState"), generatedUnion(t, generated, "EntitlementState"), "status")
	assertStringInContracts(t, data["billingRecoveryState"], openAPIEnum(t, openapi, "BillingRecoveryState"), generatedUnion(t, generated, "BillingRecoveryState"), "billingRecoveryState")
	for _, mode := range data["allowedModes"].([]any) {
		assertStringInContracts(t, mode, openAPIEnum(t, openapi, "SearchMode"), generatedUnion(t, generated, "SearchMode"), "allowedModes")
	}

	if _, ok := data["userId"].(string); !ok {
		t.Fatalf("userId = %#v, want generated string field", data["userId"])
	}
	if _, ok := data["searchLimitPer24h"].(float64); !ok {
		t.Fatalf("searchLimitPer24h = %#v, want generated number field", data["searchLimitPer24h"])
	}
	if _, ok := data["usageUsed"].(float64); !ok {
		t.Fatalf("usageUsed = %#v, want generated number field", data["usageUsed"])
	}
	if data["usageRemaining"] != nil {
		t.Fatalf("usageRemaining = %#v, want null for uncapped trial contract sample", data["usageRemaining"])
	}
	if data["usageWindowStartedAt"] != nil {
		t.Fatalf("usageWindowStartedAt = %#v, want null for uncapped trial contract sample", data["usageWindowStartedAt"])
	}
	if _, ok := data["trialExpiresAt"].(string); !ok {
		t.Fatalf("trialExpiresAt = %#v, want generated string|null field with string sample", data["trialExpiresAt"])
	}
}

func assertStringInContracts(t *testing.T, value any, openapiValues []string, generatedValues []string, field string) {
	t.Helper()
	stringValue, ok := value.(string)
	if !ok {
		t.Fatalf("%s = %#v, want string", field, value)
	}
	if !containsString(openapiValues, stringValue) {
		t.Fatalf("%s = %s is not allowed by OpenAPI enum %v", field, stringValue, openapiValues)
	}
	if !containsString(generatedValues, stringValue) {
		t.Fatalf("%s = %s is not allowed by generated union %v", field, stringValue, generatedValues)
	}
}

func openAPIRequiredFields(t *testing.T, openapi string, schemaName string) []string {
	t.Helper()
	block := namedBlock(t, openapi, "    "+schemaName+":")
	requiredBlock := between(t, block, "      required:\n", "      properties:")
	fields := []string{}
	for _, line := range strings.Split(requiredBlock, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line != "" {
			fields = append(fields, line)
		}
	}
	if len(fields) == 0 {
		t.Fatalf("OpenAPI schema %s has no required fields", schemaName)
	}
	return fields
}

func openAPIEnum(t *testing.T, openapi string, schemaName string) []string {
	t.Helper()
	block := namedBlock(t, openapi, "    "+schemaName+":")
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "enum: [") {
			return csvValues(strings.TrimSuffix(strings.TrimPrefix(line, "enum: ["), "]"))
		}
	}
	t.Fatalf("OpenAPI schema %s enum is missing", schemaName)
	return nil
}

func generatedUnion(t *testing.T, generated string, typeName string) []string {
	t.Helper()
	prefix := "export type " + typeName + " = "
	start := strings.Index(generated, prefix)
	if start < 0 {
		t.Fatalf("generated type %s is missing", typeName)
	}
	rest := generated[start+len(prefix):]
	end := strings.Index(rest, ";")
	if end < 0 {
		t.Fatalf("generated type %s is unterminated", typeName)
	}
	raw := strings.ReplaceAll(rest[:end], "|", ",")
	raw = strings.ReplaceAll(raw, "\n", ",")
	return csvValues(raw)
}

func namedBlock(t *testing.T, content string, marker string) string {
	t.Helper()
	start := strings.Index(content, marker)
	if start < 0 {
		t.Fatalf("marker %q is missing", marker)
	}
	rest := content[start:]
	if strings.HasPrefix(marker, "    ") {
		lines := strings.Split(rest, "\n")
		block := []string{lines[0]}
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") && strings.TrimSpace(line) != "" {
				break
			}
			block = append(block, line)
		}
		return strings.Join(block, "\n")
	}
	next := strings.Index(rest[len(marker):], "\n    ")
	if next < 0 {
		return rest
	}
	return rest[:len(marker)+next]
}

func between(t *testing.T, content string, startMarker string, endMarker string) string {
	t.Helper()
	start := strings.Index(content, startMarker)
	if start < 0 {
		t.Fatalf("start marker %q is missing", startMarker)
	}
	rest := content[start+len(startMarker):]
	end := strings.Index(rest, endMarker)
	if end < 0 {
		t.Fatalf("end marker %q is missing", endMarker)
	}
	return rest[:end]
}

func csvValues(raw string) []string {
	values := []string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

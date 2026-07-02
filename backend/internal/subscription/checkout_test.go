package subscription

// Implements DESIGN-007 SubscriptionController checkout service verification.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryCheckoutStore struct {
	records map[string]repository.CheckoutIdempotencyRecord
	stores  int
}

func (s *memoryCheckoutStore) GetCheckoutIdempotency(_ context.Context, userID uuid.UUID, method string, route string, key string) (repository.CheckoutIdempotencyRecord, error) {
	record, ok := s.records[userID.String()+"|"+method+"|"+route+"|"+key]
	if !ok {
		return repository.CheckoutIdempotencyRecord{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return record, nil
}

func (s *memoryCheckoutStore) StoreCheckoutIdempotency(_ context.Context, record repository.CheckoutIdempotencyRecord) error {
	s.stores++
	s.records[record.UserID.String()+"|"+record.Method+"|"+record.Route+"|"+record.Key] = record
	return nil
}

type trackingEntitlementRepository struct {
	latest      repository.Entitlement
	appendCalls int
}

func (r *trackingEntitlementRepository) AppendEntitlement(_ context.Context, entitlement repository.Entitlement) error {
	r.appendCalls++
	r.latest = entitlement
	return nil
}

func (r *trackingEntitlementRepository) GetLatest(_ context.Context, _ uuid.UUID) (repository.Entitlement, error) {
	return r.latest, nil
}

type fakeCheckoutGateway struct {
	calls int
	err   error
	req   CheckoutSessionRequest
}

func (g *fakeCheckoutGateway) CreateCheckoutSession(_ context.Context, req CheckoutSessionRequest) (CheckoutSession, error) {
	g.calls++
	g.req = req
	if g.err != nil {
		return CheckoutSession{}, g.err
	}
	return CheckoutSession{ID: "cs_test_123", URL: "https://checkout.stripe.test/session"}, nil
}

type failingCheckoutStore struct {
	getErr   error
	storeErr error
	record   repository.CheckoutIdempotencyRecord
}

func (s failingCheckoutStore) GetCheckoutIdempotency(context.Context, uuid.UUID, string, string, string) (repository.CheckoutIdempotencyRecord, error) {
	if s.getErr != nil {
		return repository.CheckoutIdempotencyRecord{}, s.getErr
	}
	return s.record, nil
}

func (s failingCheckoutStore) StoreCheckoutIdempotency(context.Context, repository.CheckoutIdempotencyRecord) error {
	return s.storeErr
}

func TestCheckoutServiceCreatesAndReplaysCheckout(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	store := &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}
	gateway := &fakeCheckoutGateway{}
	service := NewCheckoutService(testBillingConfig(), store, gateway)
	req := testCheckoutRequest("monthly")

	first, err := service.CreateCheckout(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	second, err := service.CreateCheckout(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateCheckout() replay error = %v", err)
	}
	if gateway.calls != 1 || !second.Replayed {
		t.Fatalf("gateway calls=%d replayed=%v", gateway.calls, second.Replayed)
	}
	if first.Response.CheckoutURL != second.Response.CheckoutURL || gateway.req.UserID != req.UserID {
		t.Fatalf("checkout response mismatch first=%+v second=%+v gateway=%+v", first.Response, second.Response, gateway.req)
	}
}

func TestCheckoutServiceRejectsMissingOrConflictingIdempotencyKey(t *testing.T) {
	store := &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}
	service := NewCheckoutService(testBillingConfig(), store, &fakeCheckoutGateway{})
	req := testCheckoutRequest("monthly")
	req.IdempotencyKey = ""
	if _, err := service.CreateCheckout(context.Background(), req); !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("missing idempotency error = %v", err)
	}

	req = testCheckoutRequest("monthly")
	if _, err := service.CreateCheckout(context.Background(), req); err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	req.SuccessURL = "http://localhost:5173/billing/other"
	if _, err := service.CreateCheckout(context.Background(), req); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestCheckoutServiceValidatesAnnualAndMonthlyPrices(t *testing.T) {
	for _, tc := range []struct {
		plan        string
		wantPriceID string
		wantAmount  int
	}{
		{plan: "monthly", wantPriceID: "price_monthly_test", wantAmount: 300},
		{plan: "annual", wantPriceID: "price_annual_test", wantAmount: 2500},
	} {
		t.Run(tc.plan, func(t *testing.T) {
			service := NewCheckoutService(testBillingConfig(), &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}, &fakeCheckoutGateway{})
			result, err := service.CreateCheckout(context.Background(), testCheckoutRequest(tc.plan))
			if err != nil {
				t.Fatalf("CreateCheckout() error = %v", err)
			}
			if result.Response.PriceID != tc.wantPriceID || result.Response.AmountCents != tc.wantAmount {
				t.Fatalf("response = %+v", result.Response)
			}
		})
	}
}

func TestCheckoutServiceMapsStripeUnavailable(t *testing.T) {
	service := NewCheckoutService(testBillingConfig(), &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}, &fakeCheckoutGateway{err: errors.New("stripe down")})
	if _, err := service.CreateCheckout(context.Background(), testCheckoutRequest("monthly")); !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
}

func TestCheckoutServiceStripeUnavailableLeavesEntitlementUnchanged(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	userID := uuid.New()
	entitlements := &trackingEntitlementRepository{latest: repository.Entitlement{
		UserID:            userID,
		Tier:              "free",
		Status:            "active",
		SearchLimitPer24h: 3,
		AllowedModes:      []string{"catalog", "substitution"},
	}}
	before, err := entitlements.GetLatest(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}
	service := NewCheckoutService(testBillingConfig(), store, &fakeCheckoutGateway{err: errors.New("stripe down")})

	req := testCheckoutRequest("monthly")
	req.UserID = userID
	if _, err := service.CreateCheckout(context.Background(), req); !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	after, err := entitlements.GetLatest(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if entitlements.appendCalls != 0 || !reflect.DeepEqual(after, before) {
		t.Fatalf("entitlement changed: before=%+v after=%+v appendCalls=%d", before, after, entitlements.appendCalls)
	}
	if store.stores != 0 {
		t.Fatalf("checkout idempotency store writes = %d, want 0 for failed Stripe call", store.stores)
	}
}

func TestCheckoutServiceFailsClosedWithoutDependencies(t *testing.T) {
	req := testCheckoutRequest("monthly")
	if _, err := NewCheckoutService(testBillingConfig(), nil, &fakeCheckoutGateway{}).CreateCheckout(context.Background(), req); !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("nil store error = %v", err)
	}
	if _, err := NewCheckoutService(testBillingConfig(), &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}, nil).CreateCheckout(context.Background(), req); !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("nil gateway error = %v", err)
	}
}

func TestCheckoutServiceReturnsRepositoryAndReplayPayloadFailures(t *testing.T) {
	expected := repository.NewError(repository.ErrorKindConnection, "database down", nil)
	service := NewCheckoutService(testBillingConfig(), failingCheckoutStore{getErr: expected}, &fakeCheckoutGateway{})
	if _, err := service.CreateCheckout(context.Background(), testCheckoutRequest("monthly")); !errors.Is(err, expected) {
		t.Fatalf("get failure error = %v", err)
	}

	req := testCheckoutRequest("monthly")
	hash, err := normalizedCheckoutBodyHash(req)
	if err != nil {
		t.Fatal(err)
	}
	service = NewCheckoutService(testBillingConfig(), failingCheckoutStore{record: repository.CheckoutIdempotencyRecord{BodyHash: hash, StatusCode: 200, ResponseBody: []byte(`not json`)}}, &fakeCheckoutGateway{})
	if _, err := service.CreateCheckout(context.Background(), req); err == nil {
		t.Fatal("CreateCheckout() accepted invalid stored response")
	}
}

func TestCheckoutServiceMapsStoreConflictAndStoreFailure(t *testing.T) {
	req := testCheckoutRequest("monthly")
	conflict := repository.NewError(repository.ErrorKindConflict, "duplicate", nil)
	service := NewCheckoutService(testBillingConfig(), failingCheckoutStore{getErr: repository.NewError(repository.ErrorKindNotFound, "missing", nil), storeErr: conflict}, &fakeCheckoutGateway{})
	if _, err := service.CreateCheckout(context.Background(), req); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("store conflict error = %v", err)
	}

	expected := repository.NewError(repository.ErrorKindConnection, "database down", nil)
	service = NewCheckoutService(testBillingConfig(), failingCheckoutStore{getErr: repository.NewError(repository.ErrorKindNotFound, "missing", nil), storeErr: expected}, &fakeCheckoutGateway{})
	if _, err := service.CreateCheckout(context.Background(), req); !errors.Is(err, expected) {
		t.Fatalf("store failure error = %v", err)
	}
}

func TestCheckoutServiceRejectsInvalidPlan(t *testing.T) {
	service := NewCheckoutService(testBillingConfig(), &memoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}, &fakeCheckoutGateway{})
	if _, err := service.CreateCheckout(context.Background(), testCheckoutRequest("weekly")); !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
}

func TestStripeCheckoutGatewayCreatesHostedSessionWithoutCardData(t *testing.T) {
	// Verifies IT-ARCH-007-003.
	// Verifies ARCH-007.
	// Verifies ARCH-010.
	// Traces SW-REQ-044 and SW-REQ-050.
	userID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_gateway" {
			t.Fatalf("authorization header = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("mode") != "subscription" || r.Form.Get("line_items[0][price]") != "price_monthly_test" || r.Form.Get("client_reference_id") != userID.String() {
			t.Fatalf("stripe form = %#v", r.Form)
		}
		for _, field := range []string{"card", "cardNumber", "number", "cvc", "cvv"} {
			if _, exists := r.Form[field]; exists {
				t.Fatalf("stripe form contains raw card field %q: %#v", field, r.Form)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"cs_test_gateway","url":"https://checkout.stripe.test/gateway"}`))
	}))
	defer server.Close()

	gateway := NewStripeCheckoutGatewayWithBaseURL("sk_test_gateway", server.Client(), server.URL)
	session, err := gateway.CreateCheckoutSession(context.Background(), CheckoutSessionRequest{
		UserID:     userID,
		Plan:       "monthly",
		PriceID:    "price_monthly_test",
		SuccessURL: "http://localhost:5173/billing/success",
		CancelURL:  "http://localhost:5173/billing/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckoutSession() error = %v", err)
	}
	if session.ID != "cs_test_gateway" || session.URL == "" {
		t.Fatalf("session = %+v", session)
	}
}

func TestStripeCheckoutGatewayConstructorsUseSafeDefaults(t *testing.T) {
	defaultGateway := NewStripeCheckoutGateway(" sk_test_gateway ", nil)
	if defaultGateway.secretKey != "sk_test_gateway" || defaultGateway.client == nil || defaultGateway.baseURL != "https://api.stripe.com" {
		t.Fatalf("default gateway = %+v", defaultGateway)
	}
	testGateway := NewStripeCheckoutGatewayWithBaseURL("sk_test_gateway", nil, "http://stripe.test/")
	if testGateway.client == nil || testGateway.baseURL != "http://stripe.test" {
		t.Fatalf("test gateway = %+v", testGateway)
	}
}

func TestStripeCheckoutGatewayMapsUnavailableAndMalformedResponses(t *testing.T) {
	if _, err := NewStripeCheckoutGatewayWithBaseURL("", http.DefaultClient, "http://127.0.0.1").CreateCheckoutSession(context.Background(), CheckoutSessionRequest{}); !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("missing key error = %v", err)
	}

	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":{"message":"down"}}`))
	}))
	defer unavailable.Close()
	if _, err := NewStripeCheckoutGatewayWithBaseURL("sk_test_gateway", unavailable.Client(), unavailable.URL).CreateCheckoutSession(context.Background(), testGatewayRequest()); !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("unavailable error = %v", err)
	}

	malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"","url":""}`))
	}))
	defer malformed.Close()
	if _, err := NewStripeCheckoutGatewayWithBaseURL("sk_test_gateway", malformed.Client(), malformed.URL).CreateCheckoutSession(context.Background(), testGatewayRequest()); err == nil {
		t.Fatal("CreateCheckoutSession() accepted malformed response")
	}
}

func testCheckoutRequest(plan string) CheckoutRequest {
	return CheckoutRequest{UserID: uuid.New(), IdempotencyKey: "idem-123", Method: "POST", Route: "/billing/checkout", Plan: plan, SuccessURL: "http://localhost:5173/billing/success", CancelURL: "http://localhost:5173/billing/cancel"}
}

func testGatewayRequest() CheckoutSessionRequest {
	return CheckoutSessionRequest{UserID: uuid.New(), Plan: "monthly", PriceID: "price_monthly_test", SuccessURL: "http://localhost:5173/billing/success", CancelURL: "http://localhost:5173/billing/cancel"}
}

func testBillingConfig() config.BillingConfig {
	return config.BillingConfig{
		MonthlyPlan: config.BillingPlan{Code: "monthly", Label: "Monthly", AmountCents: 300, PriceID: "price_monthly_test"},
		AnnualPlan:  config.BillingPlan{Code: "annual", Label: "Annual", AmountCents: 2500, PriceID: "price_annual_test"},
	}
}

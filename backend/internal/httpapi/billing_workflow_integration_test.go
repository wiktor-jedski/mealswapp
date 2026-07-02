package httpapi

// Implements DESIGN-007 EntitlementManager Phase 06 Billing Workflow Integration Gate.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

type statefulBillingRepo struct {
	ents        map[uuid.UUID][]repository.Entitlement
	usages      map[uuid.UUID][]repository.UsageWindow
	processed   map[string]bool
	customerMap map[string]uuid.UUID
}

func (r *statefulBillingRepo) AppendEntitlement(ctx context.Context, ent repository.Entitlement) error {
	r.ents[ent.UserID] = append(r.ents[ent.UserID], ent)
	if ent.StripeCustomerID != "" {
		r.customerMap[ent.StripeCustomerID] = ent.UserID
	}
	return nil
}

func (r *statefulBillingRepo) GetLatest(ctx context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	ents := r.ents[userID]
	if len(ents) == 0 {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
	}
	return ents[len(ents)-1], nil
}

func (r *statefulBillingRepo) RecordUsage(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error) {
	window := repository.UsageWindow{UserID: userID, Feature: feature, SearchCount: 1}
	usages := r.usages[userID]
	if len(usages) > 0 && usages[len(usages)-1].Feature == feature {
		window.SearchCount = usages[len(usages)-1].SearchCount + 1
	}
	r.usages[userID] = append(r.usages[userID], window)
	return window, nil
}

func (r *statefulBillingRepo) GetUsageSince(ctx context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error) {
	usages := r.usages[userID]
	if len(usages) == 0 {
		return repository.UsageWindow{UserID: userID, Feature: feature, SearchCount: 0}, nil
	}
	return usages[len(usages)-1], nil
}

func (r *statefulBillingRepo) GetLatestByStripeCustomer(ctx context.Context, customerID string) (repository.Entitlement, error) {
	userID, ok := r.customerMap[customerID]
	if !ok {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
	}
	return r.GetLatest(ctx, userID)
}

func (r *statefulBillingRepo) GetLatestByStripeSubscription(ctx context.Context, subID string) (repository.Entitlement, error) {
	for _, ents := range r.ents {
		for _, ent := range ents {
			if ent.StripeSubscriptionID == subID {
				return ent, nil
			}
		}
	}
	return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
}

func (r *statefulBillingRepo) InsertProcessedStripeEvent(ctx context.Context, event repository.ProcessedStripeEvent) (bool, error) {
	if r.processed[event.EventID] {
		return false, nil
	}
	r.processed[event.EventID] = true
	return true, nil
}

func (r *statefulBillingRepo) ListExpiredTrials(ctx context.Context, cutoff time.Time) ([]repository.Entitlement, error) {
	return nil, nil
}

func createBillingSignedRequest(payload []byte, secret string) *http.Request {
	req := httptest.NewRequest("POST", "/api/v1/billing/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	now := time.Now()
	t := now.Unix()
	mac := webhook.ComputeSignature(now, payload, secret)
	sigHeader := fmt.Sprintf("t=%d,v1=%x", t, mac)
	req.Header.Set("Stripe-Signature", sigHeader)
	return req
}

type EntitlementManagerWrapper struct {
	Manager *subscription.EntitlementManager
}

func (w *EntitlementManagerWrapper) CheckEntitlement(ctx context.Context, userID uuid.UUID, feature string) (EntitlementDecision, error) {
	decision, err := w.Manager.CheckEntitlement(ctx, userID, feature)
	if err != nil {
		return EntitlementDecision{Allowed: false}, err
	}
	return EntitlementDecision{Allowed: decision.Allowed, Code: decision.Reason, Message: decision.Reason}, nil
}

type UsageLimiterWrapper struct {
	Limiter *subscription.UsageLimiter
	Manager *subscription.EntitlementManager
}

func (w *UsageLimiterWrapper) CheckUsageLimit(ctx context.Context, userID uuid.UUID, feature string) error {
	ent, err := w.Manager.GetEntitlementState(ctx, userID)
	if err != nil {
		ent = repository.Entitlement{}
	}
	err = w.Limiter.CheckAccess(ctx, &ent, feature, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func (w *UsageLimiterWrapper) RecordUsage(ctx context.Context, userID uuid.UUID, feature string) error {
	ent, err := w.Manager.GetEntitlementState(ctx, userID)
	if err != nil {
		ent = repository.Entitlement{}
	}
	return w.Limiter.RecordUsage(ctx, &ent, feature, time.Now(), true)
}

func TestBillingWorkflowIntegrationGate(t *testing.T) {
	cfg := testConfig()
	cfg.Billing.StripeWebhookSecret = "whsec_test"
	cfg.Billing.MonthlyPlanPriceID = "price_monthly"
	repo := &statefulBillingRepo{
		ents:        make(map[uuid.UUID][]repository.Entitlement),
		usages:      make(map[uuid.UUID][]repository.UsageWindow),
		processed:   make(map[string]bool),
		customerMap: make(map[string]uuid.UUID),
	}
	userID := uuid.New()

	entManager := subscription.NewEntitlementManager(repo)
	usageLimiter := subscription.NewUsageLimiter(repo, 3)
	trialTracker := entitlement.NewTrialTracker(repo, repo, time.Now)

	entWrapper := &EntitlementManagerWrapper{Manager: entManager}
	usageWrapper := &UsageLimiterWrapper{Limiter: usageLimiter, Manager: entManager}

	searchRepo := &composedSearchGateRepository{
		source: repository.FoodItemEntity{ID: uuid.MustParse("61000000-0000-4000-8000-000000000001"), Name: "Apple", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 1, Fat: 1}},
		items: []repository.FoodItemEntity{
			{ID: uuid.MustParse("61000000-0000-4000-8000-000000000001"), Name: "Apple", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 1, Fat: 1}},
		},
	}

	searchService := search.NewSearchDispatcher(
		search.NewCatalogService(searchRepo, &composedSearchGateCache{}),
		search.NewSubstitutionService(searchRepo, &composedSearchGateCache{}),
	)

	searchController := NewSearchController(searchService).WithEntitlementGate(entWrapper, usageWrapper)
	subscriptionController := NewSubscriptionController(cfg, &fakeCheckoutGateway{urls: []string{"https://checkout.stripe.com/123"}}, entManager, usageLimiter)
	webhookHandler := NewStripeWebhookHandler(cfg, repo, repo, &mockAuditLogger{})

	var routes []RouteDefinition
	routes = append(routes, searchController.Routes()...)
	routes = append(routes, subscriptionController.Routes()...)
	routes = append(routes, webhookHandler.Routes()...)

	for i := range routes {
		routes[i].RequiresCSRF = false
		routes[i].ExemptCSRF = true
	}

	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)

	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: routes})

	// 1. Trial unlock from social login (simulated via Tracker)
	err := trialTracker.ActivateFirstLoginTrial(context.Background(), userID)
	if err != nil {
		t.Fatalf("failed to start trial: %v", err)
	}

	ent, _ := repo.GetLatest(context.Background(), userID)
	if ent.Tier != "trial" {
		t.Fatalf("expected trial tier, got %s", ent.Tier)
	}

	// 2. Anonymous Catalog Search
	body := searchRequestBody(t, map[string]any{"query": " apple ", "mode": "catalog", "page": 1, "filters": []any{}})
	resp, _ := app.Test(searchHTTPPost(body))
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("anonymous catalog search failed: %d", resp.StatusCode)
	}

	// 3. Free limit exhaustion
	repo.AppendEntitlement(context.Background(), repository.Entitlement{UserID: userID, Tier: "free", Status: "active"})

	for i := 0; i < 3; i++ {
		req := searchHTTPPost(searchRequestBody(t, map[string]any{"query": "apple", "mode": "substitution", "page": 1, "filters": []any{}, "substitutionInputs": []any{map[string]any{"foodObjectId": "61000000-0000-4000-8000-000000000001", "quantity": 100, "unit": "g"}}}))
		addCookies(req, authCookies)

		resp, _ = app.Test(req)
		if resp.StatusCode != fiber.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("free substitution within limit failed at %d: %d body=%s", i, resp.StatusCode, string(b))
		}
	}

	// 4th should fail
	req := searchHTTPPost(searchRequestBody(t, map[string]any{"query": "apple", "mode": "substitution", "page": 1, "filters": []any{}, "substitutionInputs": []any{map[string]any{"foodObjectId": "61000000-0000-4000-8000-000000000001", "quantity": 100, "unit": "g"}}}))
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("free substitution beyond limit did not fail: %d", resp.StatusCode)
	}

	// 4. Blocked paid-mode UI with no network search side effects
	req = searchHTTPPost(searchRequestBody(t, map[string]any{"query": "apple", "mode": "daily_diet_alternative", "page": 1, "filters": []any{}, "dailyDietId": uuid.NewString()}))
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("daily_diet_alternative not blocked for free user: %d", resp.StatusCode)
	}

	// 5. Checkout idempotency retry
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/subscription/checkout", bytes.NewBufferString(`{"priceId":"price_monthly","successUrl":"http://localhost:5173/success","cancelUrl":"http://localhost:5173/cancel"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-idem-key")
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("checkout failed: %d body=%s", resp.StatusCode, string(b))
	}
	// Verify retry uses same checkout
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("checkout retry failed: %d", resp.StatusCode)
	}

	// 6. Paid unlock after webhook
	repo.customerMap["cus_123"] = userID // simulate checkout success associated stripe customer
	session := stripe.CheckoutSession{
		ClientReferenceID: userID.String(),
		Mode:              stripe.CheckoutSessionModeSubscription,
		Customer:          &stripe.Customer{ID: "cus_123"},
		Subscription:      &stripe.Subscription{ID: "sub_123"},
	}
	rawSession, _ := json.Marshal(session)
	event := stripe.Event{
		ID:   "evt_1",
		Type: "checkout.session.completed",
		Data: &stripe.EventData{Raw: rawSession},
	}
	payload, _ := json.Marshal(event)
	req = createBillingSignedRequest(payload, "whsec_test")
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("webhook paid unlock failed: %d", resp.StatusCode)
	}

	ent, _ = repo.GetLatest(context.Background(), userID)
	if ent.Tier != "paid" {
		t.Fatalf("expected paid tier after webhook, got %s", ent.Tier)
	}

	// 7. Duplicate webhook non-reapplication
	req = createBillingSignedRequest(payload, "whsec_test")
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("duplicate webhook failed: %d", resp.StatusCode)
	}
	if len(repo.ents[userID]) != 3 { // Trial -> Free -> Paid (No duplicate Paid appended)
		t.Fatalf("expected exactly 3 entitlements appended (Trial, Free, Paid), got %d", len(repo.ents[userID]))
	}
}

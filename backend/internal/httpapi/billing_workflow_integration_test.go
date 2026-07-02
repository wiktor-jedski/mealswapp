package httpapi

// Implements DESIGN-007 EntitlementManager Phase 06 billing workflow integration gate.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

type billingWorkflowStore struct {
	entitlements map[uuid.UUID]repository.Entitlement
	history      map[uuid.UUID][]repository.Entitlement
	usageCount   int
	events       map[string]repository.ProcessedStripeEvent
	deadLetters  []repository.StripeDeadLetter
}

func (s *billingWorkflowStore) AppendEntitlement(_ context.Context, entitlement repository.Entitlement) error {
	if s.entitlements == nil {
		s.entitlements = map[uuid.UUID]repository.Entitlement{}
	}
	if s.history == nil {
		s.history = map[uuid.UUID][]repository.Entitlement{}
	}
	s.entitlements[entitlement.UserID] = entitlement
	s.history[entitlement.UserID] = append(s.history[entitlement.UserID], entitlement)
	return nil
}

func (s *billingWorkflowStore) GetLatest(_ context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	entitlement, ok := s.entitlements[userID]
	if !ok {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "entitlement not found", nil)
	}
	return entitlement, nil
}

func (s *billingWorkflowStore) RecordUsage(_ context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error) {
	s.usageCount++
	return s.usageWindow(userID, feature, occurredAt), nil
}

func (s *billingWorkflowStore) RecordUsageWithinLimit(_ context.Context, userID uuid.UUID, feature string, occurredAt time.Time, since time.Time, limit int) (repository.UsageWindow, bool, error) {
	if s.usageCount >= limit {
		return s.usageWindow(userID, feature, since), false, nil
	}
	s.usageCount++
	return s.usageWindow(userID, feature, occurredAt), true, nil
}

func (s *billingWorkflowStore) GetUsageSince(_ context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error) {
	return s.usageWindow(userID, feature, since), nil
}

func (s *billingWorkflowStore) ProcessStripeWebhookEvent(_ context.Context, event repository.ProcessedStripeEvent, entitlement *repository.Entitlement) (bool, error) {
	if s.events == nil {
		s.events = map[string]repository.ProcessedStripeEvent{}
	}
	if _, ok := s.events[event.EventID]; ok {
		return false, nil
	}
	s.events[event.EventID] = event
	if entitlement != nil {
		return true, s.AppendEntitlement(context.Background(), *entitlement)
	}
	return true, nil
}

func (s *billingWorkflowStore) InsertStripeDeadLetter(_ context.Context, entry repository.StripeDeadLetter) error {
	s.deadLetters = append(s.deadLetters, entry)
	return nil
}

func (s *billingWorkflowStore) usageWindow(userID uuid.UUID, feature string, startedAt time.Time) repository.UsageWindow {
	return repository.UsageWindow{UserID: userID, Feature: feature, StartedAt: startedAt, SearchCount: s.usageCount, CreatedAt: startedAt, UpdatedAt: startedAt}
}

func TestPhase06BillingWorkflowIntegrationGate(t *testing.T) {
	// Verifies IT-ARCH-007-001.
	// Verifies IT-ARCH-007-003.
	// Verifies IT-ARCH-007-004.
	// Verifies ARCH-007.
	// Verifies ARCH-002.
	// Verifies ARCH-010.
	// Verifies ARCH-013.
	// Traces SW-REQ-042, SW-REQ-044, SW-REQ-045, SW-REQ-050, SW-REQ-052, and SW-REQ-053.
	// Implements DESIGN-007 EntitlementManager, UsageLimiter, SubscriptionController, and StripeWebhookHandler workflow gate.
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	store := &billingWorkflowStore{entitlements: map[uuid.UUID]repository.Entitlement{}, history: map[uuid.UUID][]repository.Entitlement{}, usageCount: 3}
	checkoutGateway := &httpFakeCheckoutGateway{}
	checkoutStore := &httpMemoryCheckoutStore{records: map[string]repository.CheckoutIdempotencyRecord{}}
	checkoutService := subscription.NewCheckoutService(httpTestBillingConfig(), checkoutStore, checkoutGateway)
	usageGate := entitlement.NewUsageLimiter(entitlement.NewEntitlementManager(store), store)
	statusReader := entitlement.NewStatusService(store, store)
	searchService := &fakeSearchService{response: search.SearchResponse{
		Items:            []repository.FoodItemEntity{{ID: uuid.New(), Name: "Apple", PhysicalState: repository.PhysicalStateSolid}},
		TotalCount:       1,
		Page:             1,
		SimilarityScores: []float64{1},
		Warnings:         []string{},
	}}
	webhookService := subscription.NewStripeWebhookService("whsec_phase06_gate", store)
	routes := []RouteDefinition{}
	routes = append(routes, NewSearchController(searchService).WithSearchUsageGate(usageGate).Routes()...)
	routes = append(routes, NewSubscriptionController(checkoutService, statusReader).Routes()...)
	routes = append(routes, NewStripeWebhookHandler(webhookService, nil).Routes()...)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: routes})

	anonymousCatalog := searchHTTPPost(searchRequestBody(t, map[string]any{"query": "apple", "mode": "catalog", "page": 1, "filters": []any{}}))
	resp, err := app.Test(anonymousCatalog)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || searchService.calls != 1 || store.usageCount != 3 {
		t.Fatalf("anonymous catalog status=%d searchCalls=%d usage=%d", resp.StatusCode, searchService.calls, store.usageCount)
	}

	freeLimited := searchHTTPPost(singleSubstitutionBody(t))
	addCookies(freeLimited, authCookies)
	resp, err = app.Test(freeLimited)
	if err != nil {
		t.Fatal(err)
	}
	limitedEnvelope := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusTooManyRequests || limitedEnvelope.Error == nil || limitedEnvelope.Error.Code != "free_usage_limit_reached" || searchService.calls != 1 {
		t.Fatalf("free limit status=%d envelope=%+v searchCalls=%d", resp.StatusCode, limitedEnvelope, searchService.calls)
	}

	status, entitlementBody := getBillingEntitlement(t, app, authCookies)
	if status != fiber.StatusOK || entitlementBody.Data["tier"] != "free" || entitlementBody.Data["usageRemaining"] != float64(0) {
		t.Fatalf("free entitlement status=%d body=%+v", status, entitlementBody)
	}

	postCheckout := func() (int, Envelope) {
		t.Helper()
		token, csrfCookies := fetchCSRFToken(t, app)
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"plan":"monthly","successUrl":"http://localhost:5173/billing/success","cancelUrl":"http://localhost:5173/billing/cancel"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", token)
		req.Header.Set("Idempotency-Key", "phase06-gate-checkout")
		addCookies(req, csrfCookies)
		addCookies(req, authCookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		body := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		return resp.StatusCode, body
	}
	firstCheckoutStatus, firstCheckout := postCheckout()
	secondCheckoutStatus, secondCheckout := postCheckout()
	if firstCheckoutStatus != fiber.StatusOK || secondCheckoutStatus != fiber.StatusOK || firstCheckout.Data["checkoutSessionId"] != secondCheckout.Data["checkoutSessionId"] || checkoutGateway.calls != 1 {
		t.Fatalf("checkout replay first=%d/%+v second=%d/%+v gatewayCalls=%d", firstCheckoutStatus, firstCheckout, secondCheckoutStatus, secondCheckout, checkoutGateway.calls)
	}

	payload := []byte(phase06WebhookPayload("evt_phase06_paid", "checkout.session.completed", userID, "cus_phase06", "sub_phase06", ""))
	signature := signPhase06WebhookPayload(payload, "whsec_phase06_gate")
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/billing/stripe/webhook", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", signature)
		resp, err = app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("webhook delivery %d status=%d", i+1, resp.StatusCode)
		}
	}
	if len(store.history[userID]) != 1 || store.history[userID][0].Tier != "paid" || store.history[userID][0].Status != "active" {
		t.Fatalf("paid entitlement history after duplicate webhook = %#v", store.history[userID])
	}

	paidDailyDiet := searchHTTPPost(searchRequestBody(t, map[string]any{"query": "lentil", "mode": "daily_diet", "page": 1, "filters": []any{}, "dailyDietId": "61e0cae4-0f45-4854-8ac5-b228214cdd1d"}))
	addCookies(paidDailyDiet, authCookies)
	resp, err = app.Test(paidDailyDiet)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || searchService.calls != 2 || store.usageCount != 3 {
		t.Fatalf("paid daily diet status=%d searchCalls=%d usage=%d", resp.StatusCode, searchService.calls, store.usageCount)
	}

	status, entitlementBody = getBillingEntitlement(t, app, authCookies)
	if status != fiber.StatusOK || entitlementBody.Data["tier"] != "paid" || entitlementBody.Data["status"] != "active" || entitlementBody.Data["usageRemaining"] != nil {
		t.Fatalf("paid entitlement status=%d body=%+v", status, entitlementBody)
	}
}

func getBillingEntitlement(t *testing.T, app *fiber.App, authCookies []*http.Cookie) (int, Envelope) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/billing/entitlement", nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, body
}

func signPhase06WebhookPayload(payload []byte, secret string) string {
	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

func phase06WebhookPayload(eventID string, eventType string, userID uuid.UUID, customerID string, subscriptionID string, status string) string {
	return fmt.Sprintf(`{"id":%q,"type":%q,"data":{"object":{"id":"cs_phase06","client_reference_id":%q,"customer":%q,"subscription":%q,"status":%q,"metadata":{"user_id":%q}}}}`,
		eventID, eventType, userID.String(), customerID, subscriptionID, status, userID.String())
}

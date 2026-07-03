package subscription

// Implements DESIGN-007 StripeWebhookHandler reconciliation verification.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryReconciliationGateway struct {
	subscriptions []StripeSubscription
	err           error
}

func (g memoryReconciliationGateway) ListSubscriptions(context.Context) ([]StripeSubscription, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.subscriptions, nil
}

type memoryEntitlementStore struct {
	latest  map[uuid.UUID]repository.Entitlement
	appends []repository.Entitlement
}

func (s *memoryEntitlementStore) AppendEntitlement(_ context.Context, entitlement repository.Entitlement) error {
	if s.latest == nil {
		s.latest = map[uuid.UUID]repository.Entitlement{}
	}
	s.latest[entitlement.UserID] = entitlement
	s.appends = append(s.appends, entitlement)
	return nil
}

func (s *memoryEntitlementStore) GetLatest(_ context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	if s.latest != nil {
		if entitlement, ok := s.latest[userID]; ok {
			return entitlement, nil
		}
	}
	return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
}

func TestReconcileStripeEntitlementsAppendsMissingPaidCancelledAndPastDueState(t *testing.T) {
	// Verifies IT-ARCH-007-005.
	// Verifies ARCH-007.
	// Traces SW-REQ-045 and SW-REQ-052.
	activeUser := uuid.New()
	cancelledUser := uuid.New()
	pastDueUser := uuid.New()
	store := &memoryEntitlementStore{}
	service := NewReconciliationService(memoryReconciliationGateway{subscriptions: []StripeSubscription{
		{UserID: activeUser, CustomerID: "cus_active", SubscriptionID: "sub_active", Status: "active"},
		{UserID: cancelledUser, CustomerID: "cus_cancelled", SubscriptionID: "sub_cancelled", Status: "canceled"},
		{UserID: pastDueUser, CustomerID: "cus_past_due", SubscriptionID: "sub_past_due", Status: "past_due"},
	}}, store, nil)

	result, err := service.ReconcileStripeEntitlements(context.Background())
	if err != nil {
		t.Fatalf("ReconcileStripeEntitlements() error = %v", err)
	}
	if result.Checked != 3 || result.Appended != 3 || result.Skipped != 0 {
		t.Fatalf("result = %+v, want three appends", result)
	}
	got := map[uuid.UUID]string{}
	for _, entitlement := range store.appends {
		got[entitlement.UserID] = entitlement.Status
		if entitlement.Tier != "paid" || entitlement.SearchLimitPer24h != 0 || len(entitlement.AllowedModes) == 0 {
			t.Fatalf("entitlement = %#v, want paid entitlement projection", entitlement)
		}
	}
	if got[activeUser] != "active" || got[cancelledUser] != "cancelled" || got[pastDueUser] != "past_due" {
		t.Fatalf("statuses = %#v, want active/cancelled/past_due", got)
	}
}

func TestReconcileStripeEntitlementsIsIdempotentAcrossDuplicateRuns(t *testing.T) {
	// Verifies IT-ARCH-007-005.
	// Verifies ARCH-007.
	// Traces SW-REQ-045 and SW-REQ-052.
	userID := uuid.New()
	store := &memoryEntitlementStore{}
	service := NewReconciliationService(memoryReconciliationGateway{subscriptions: []StripeSubscription{
		{UserID: userID, CustomerID: "cus_123", SubscriptionID: "sub_123", Status: "active"},
	}}, store, nil)

	if _, err := service.ReconcileStripeEntitlements(context.Background()); err != nil {
		t.Fatalf("first ReconcileStripeEntitlements() error = %v", err)
	}
	result, err := service.ReconcileStripeEntitlements(context.Background())
	if err != nil {
		t.Fatalf("second ReconcileStripeEntitlements() error = %v", err)
	}
	if result.Appended != 0 || result.Skipped != 1 || len(store.appends) != 1 {
		t.Fatalf("second result = %+v appends=%d, want idempotent skip", result, len(store.appends))
	}
}

func TestReconcileStripeEntitlementsSkipsSubscriptionsWithoutLocalUserIdentity(t *testing.T) {
	// Verifies IT-ARCH-007-005.
	// Verifies ARCH-007.
	// Traces SW-REQ-045 and SW-REQ-052.
	store := &memoryEntitlementStore{}
	service := NewReconciliationService(memoryReconciliationGateway{subscriptions: []StripeSubscription{
		{CustomerID: "cus_missing_user", SubscriptionID: "sub_missing_user", Status: "active"},
	}}, store, nil)

	result, err := service.ReconcileStripeEntitlements(context.Background())
	if err != nil {
		t.Fatalf("ReconcileStripeEntitlements() error = %v", err)
	}
	if result.Checked != 1 || result.Appended != 0 || result.Skipped != 1 || len(store.appends) != 0 {
		t.Fatalf("result = %+v appends=%d, want defensive skip", result, len(store.appends))
	}
}

func TestReconcileStripeEntitlementsFailureLeavesLocalStateAndWarns(t *testing.T) {
	// Verifies IT-ARCH-007-005.
	// Verifies ARCH-007.
	// Traces SW-REQ-045 and SW-REQ-052.
	userID := uuid.New()
	original := repository.Entitlement{UserID: userID, Tier: "paid", Status: "active", SearchLimitPer24h: 0, AllowedModes: []string{"catalog"}, StripeCustomerID: "cus_123", StripeSubscriptionID: "sub_123"}
	store := &memoryEntitlementStore{latest: map[uuid.UUID]repository.Entitlement{userID: original}}
	logs := &observability.MemorySink{}
	service := NewReconciliationService(memoryReconciliationGateway{err: errors.New("stripe timeout")}, store, logs)

	_, err := service.ReconcileStripeEntitlements(context.Background())
	if !errors.Is(err, ErrStripeUnavailable) {
		t.Fatalf("ReconcileStripeEntitlements() error = %v, want Stripe unavailable", err)
	}
	if len(store.appends) != 0 || store.latest[userID].Status != "active" {
		t.Fatalf("store changed: appends=%#v latest=%#v", store.appends, store.latest[userID])
	}
	if len(logs.Logs) != 1 || logs.Logs[0].Level != "warning" {
		t.Fatalf("logs = %#v, want observable warning", logs.Logs)
	}
}

func TestStripeSubscriptionHTTPGatewayListsSanitizedSubscriptionFixtures(t *testing.T) {
	userID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_fixture" {
			t.Fatalf("Authorization = %q, want bearer secret", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/v1/subscriptions" || r.URL.Query().Get("status") != "all" {
			t.Fatalf("request URL = %s, want subscription list", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"has_more":false,"data":[{"id":"sub_123","customer":{"id":"cus_123","email":"payer@example.test"},"status":"active","metadata":{"user_id":"` + userID.String() + `"},"latest_invoice":{"card":{"last4":"4242"}}}]}`))
	}))
	defer server.Close()

	gateway := NewStripeSubscriptionGatewayWithBaseURL("sk_test_fixture", server.Client(), server.URL)
	subscriptions, err := gateway.ListSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("subscriptions = %#v, want one", subscriptions)
	}
	got := subscriptions[0]
	if got.UserID != userID || got.CustomerID != "cus_123" || got.SubscriptionID != "sub_123" || got.Status != "active" {
		t.Fatalf("subscription = %#v, want sanitized projection", got)
	}
	if strings.Contains(strings.Join([]string{got.CustomerID, got.SubscriptionID, got.Status}, " "), "4242") {
		t.Fatalf("subscription = %#v, raw payment fixture leaked", got)
	}
}

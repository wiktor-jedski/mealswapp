// Implements DESIGN-007 EntitlementManager.
package subscription_test

import (
	"context"
	"errors"
	"testing"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

type mockReconciliationEntitlementRepo struct {
	repository.EntitlementRepository
	appended     []repository.Entitlement
	subEnts      map[string]repository.Entitlement
}

func (m *mockReconciliationEntitlementRepo) AppendEntitlement(ctx context.Context, ent repository.Entitlement) error {
	m.appended = append(m.appended, ent)
	if ent.StripeSubscriptionID != "" {
		m.subEnts[ent.StripeSubscriptionID] = ent
	}
	return nil
}

func (m *mockReconciliationEntitlementRepo) GetLatestByStripeSubscription(ctx context.Context, subscriptionID string) (repository.Entitlement, error) {
	if ent, ok := m.subEnts[subscriptionID]; ok {
		return ent, nil
	}
	return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
}

type mockStripeSubscriptionGateway struct {
	subs []subscription.StripeSubscriptionStatus
	err  error
}

func (m *mockStripeSubscriptionGateway) ListSubscriptions(ctx context.Context) ([]subscription.StripeSubscriptionStatus, error) {
	return m.subs, m.err
}

func TestReconcileStripeEntitlements_AppendsDrift(t *testing.T) {
	userID := uuid.New()
	repo := &mockReconciliationEntitlementRepo{
		subEnts: map[string]repository.Entitlement{
			"sub_1": {UserID: userID, Tier: "paid", Status: "active", StripeSubscriptionID: "sub_1", StripeCustomerID: "cus_1"},
			"sub_2": {UserID: userID, Tier: "paid", Status: "active", StripeSubscriptionID: "sub_2", StripeCustomerID: "cus_2"},
			"sub_3": {UserID: userID, Tier: "paid", Status: "active", StripeSubscriptionID: "sub_3", StripeCustomerID: "cus_3"},
		},
	}
	manager := subscription.NewEntitlementManager(repo)
	
	gateway := &mockStripeSubscriptionGateway{
		subs: []subscription.StripeSubscriptionStatus{
			{SubscriptionID: "sub_1", CustomerID: "cus_1", Status: "active"},             // no drift
			{SubscriptionID: "sub_2", CustomerID: "cus_2", Status: "past_due"},           // drift: active -> past_due
			{SubscriptionID: "sub_3", CustomerID: "cus_3", Status: "canceled"},           // drift: active -> cancelled
			{SubscriptionID: "sub_unknown", CustomerID: "cus_4", Status: "active"},       // not found locally, ignored
		},
	}
	
	err := manager.ReconcileStripeEntitlements(context.Background(), gateway)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(repo.appended) != 2 {
		t.Fatalf("expected 2 entitlements appended for drift, got %d", len(repo.appended))
	}
	
	var foundPastDue, foundCancelled bool
	for _, ent := range repo.appended {
		if ent.StripeSubscriptionID == "sub_2" && ent.Status == "past_due" {
			foundPastDue = true
		}
		if ent.StripeSubscriptionID == "sub_3" && ent.Status == "cancelled" {
			foundCancelled = true
		}
	}
	if !foundPastDue || !foundCancelled {
		t.Errorf("expected to append past_due and cancelled entitlements")
	}
}

func TestReconcileStripeEntitlements_Idempotent(t *testing.T) {
	userID := uuid.New()
	repo := &mockReconciliationEntitlementRepo{
		subEnts: map[string]repository.Entitlement{
			"sub_1": {UserID: userID, Tier: "paid", Status: "past_due", StripeSubscriptionID: "sub_1", StripeCustomerID: "cus_1"},
		},
	}
	manager := subscription.NewEntitlementManager(repo)
	
	gateway := &mockStripeSubscriptionGateway{
		subs: []subscription.StripeSubscriptionStatus{
			{SubscriptionID: "sub_1", CustomerID: "cus_1", Status: "past_due"}, // already past_due
		},
	}
	
	err := manager.ReconcileStripeEntitlements(context.Background(), gateway)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(repo.appended) != 0 {
		t.Fatalf("expected 0 entitlements appended for identical state, got %d", len(repo.appended))
	}
}

func TestReconcileStripeEntitlements_StripeFailure(t *testing.T) {
	repo := &mockReconciliationEntitlementRepo{}
	manager := subscription.NewEntitlementManager(repo)
	
	gateway := &mockStripeSubscriptionGateway{
		err: errors.New("stripe API error"),
	}
	
	err := manager.ReconcileStripeEntitlements(context.Background(), gateway)
	if err == nil || err.Error() != "stripe API error" {
		t.Fatalf("expected stripe API error, got: %v", err)
	}
	
	if len(repo.appended) != 0 {
		t.Fatalf("expected no changes on API failure")
	}
}

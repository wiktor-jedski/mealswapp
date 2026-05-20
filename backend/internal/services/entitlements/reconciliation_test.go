package entitlements

import (
	"context"
	"testing"
	"time"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestReconcilerRepairsPastDueDriftAndWritesAudit(t *testing.T) {
	userID := uuid.New()
	periodEnd := fixedNow().Add(30 * 24 * time.Hour)
	store := &fakeReconciliationStore{
		local: []LocalSubscription{{
			UserID:               userID,
			Tier:                 TierPaid,
			Status:               StatusActive,
			StripeSubscriptionID: "sub_123",
		}},
	}
	stripe := fakeStripeSubscriptions{items: map[string]StripeSubscription{
		"sub_123": {ID: "sub_123", Status: "past_due", CurrentPeriodEnd: &periodEnd},
	}}
	reconciler := NewReconcilerWithClock(store, stripe, fixedNow)

	result, err := reconciler.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Checked != 1 || result.Repaired != 1 || result.Changes[0].To != StatusPastDue {
		t.Fatalf("unexpected reconciliation result: %#v", result)
	}
	if store.upserted[0].Status != "past_due" || store.upserted[0].Plan != "paid" {
		t.Fatalf("expected repaired past_due entitlement, got %#v", store.upserted)
	}
	if len(store.audit) != 1 || store.audit[0].Action != "entitlement.reconciled" {
		t.Fatalf("expected reconciliation audit, got %#v", store.audit)
	}
}

func TestReconcilerRepairsCanceledSubscription(t *testing.T) {
	userID := uuid.New()
	store := &fakeReconciliationStore{
		local: []LocalSubscription{{
			UserID:               userID,
			Tier:                 TierPaid,
			Status:               StatusActive,
			StripeSubscriptionID: "sub_cancelled",
		}},
	}
	stripe := fakeStripeSubscriptions{items: map[string]StripeSubscription{
		"sub_cancelled": {ID: "sub_cancelled", Status: "canceled"},
	}}

	result, err := NewReconcilerWithClock(store, stripe, fixedNow).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Repaired != 1 || store.upserted[0].Status != "cancelled" {
		t.Fatalf("expected cancelled repair, result=%#v upserted=%#v", result, store.upserted)
	}
}

func TestReconcilerSkipsRecordsWithoutStripeSubscriptionID(t *testing.T) {
	store := &fakeReconciliationStore{
		local: []LocalSubscription{{UserID: uuid.New(), Tier: TierFree, Status: StatusActive}},
	}

	result, err := NewReconcilerWithClock(store, fakeStripeSubscriptions{}, fixedNow).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Checked != 1 || result.Skipped != 1 || result.Repaired != 0 {
		t.Fatalf("expected skipped local-only entitlement, got %#v", result)
	}
}

func TestReconcilerLeavesMatchingActiveSubscriptionUnchanged(t *testing.T) {
	userID := uuid.New()
	periodEnd := fixedNow().Add(30 * 24 * time.Hour)
	store := &fakeReconciliationStore{
		local: []LocalSubscription{{
			UserID:               userID,
			Tier:                 TierPaid,
			Status:               StatusActive,
			ExpiresAt:            &periodEnd,
			StripeSubscriptionID: "sub_active",
		}},
	}
	stripe := fakeStripeSubscriptions{items: map[string]StripeSubscription{
		"sub_active": {ID: "sub_active", Status: "active", CurrentPeriodEnd: &periodEnd},
	}}

	result, err := NewReconcilerWithClock(store, stripe, fixedNow).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Repaired != 0 || len(store.upserted) != 0 || len(store.audit) != 0 {
		t.Fatalf("expected no repair, result=%#v upserted=%#v audit=%#v", result, store.upserted, store.audit)
	}
}

func TestReconcilerMapsStripeTrialingToTrialEntitlement(t *testing.T) {
	userID := uuid.New()
	periodEnd := fixedNow().Add(7 * 24 * time.Hour)
	store := &fakeReconciliationStore{
		local: []LocalSubscription{{
			UserID:               userID,
			Tier:                 TierPaid,
			Status:               StatusActive,
			StripeSubscriptionID: "sub_trialing",
		}},
	}
	stripe := fakeStripeSubscriptions{items: map[string]StripeSubscription{
		"sub_trialing": {ID: "sub_trialing", Status: "trialing", CurrentPeriodEnd: &periodEnd},
	}}

	result, err := NewReconcilerWithClock(store, stripe, fixedNow).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Repaired != 1 || store.upserted[0].Plan != "trial" || store.upserted[0].Status != "active" {
		t.Fatalf("expected trialing repair, result=%#v upserted=%#v", result, store.upserted)
	}
}

type fakeReconciliationStore struct {
	local    []LocalSubscription
	upserted []repositories.EntitlementEntity
	audit    []ReconciliationAuditEvent
}

func (store *fakeReconciliationStore) ListLocalSubscriptions(ctx context.Context) ([]LocalSubscription, error) {
	return store.local, nil
}

func (store *fakeReconciliationStore) UpsertEntitlement(ctx context.Context, entitlement repositories.EntitlementEntity) error {
	store.upserted = append(store.upserted, entitlement)
	return nil
}

func (store *fakeReconciliationStore) WriteReconciliationAudit(ctx context.Context, event ReconciliationAuditEvent) error {
	store.audit = append(store.audit, event)
	return nil
}

type fakeStripeSubscriptions struct {
	items map[string]StripeSubscription
}

func (stripe fakeStripeSubscriptions) GetSubscription(ctx context.Context, stripeSubscriptionID string) (StripeSubscription, error) {
	return stripe.items[stripeSubscriptionID], nil
}

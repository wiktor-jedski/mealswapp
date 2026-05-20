package entitlements

import (
	"context"
	"encoding/json"
	"time"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

type ReconciliationStore interface {
	ListLocalSubscriptions(ctx context.Context) ([]LocalSubscription, error)
	UpsertEntitlement(ctx context.Context, entitlement repositories.EntitlementEntity) error
	WriteReconciliationAudit(ctx context.Context, event ReconciliationAuditEvent) error
}

type StripeSubscriptionClient interface {
	GetSubscription(ctx context.Context, stripeSubscriptionID string) (StripeSubscription, error)
}

type LocalSubscription struct {
	UserID               uuid.UUID
	Tier                 Tier
	Status               Status
	ExpiresAt            *time.Time
	StripeCustomerID     string
	StripeSubscriptionID string
}

type StripeSubscription struct {
	ID                string
	CustomerID        string
	Status            string
	CurrentPeriodEnd  *time.Time
	CancelAtPeriodEnd bool
}

type ReconciliationAuditEvent struct {
	ActorID   *uuid.UUID
	Action    string
	Target    string
	Metadata  []byte
	CreatedAt time.Time
}

type ReconciliationResult struct {
	Checked  int      `json:"checked"`
	Repaired int      `json:"repaired"`
	Skipped  int      `json:"skipped"`
	Changes  []Repair `json:"changes"`
}

type Repair struct {
	UserID uuid.UUID `json:"userId"`
	From   Status    `json:"from"`
	To     Status    `json:"to"`
	Reason string    `json:"reason"`
}

type Reconciler struct {
	store  ReconciliationStore
	stripe StripeSubscriptionClient
	now    func() time.Time
}

func NewReconciler(store ReconciliationStore, stripe StripeSubscriptionClient) Reconciler {
	return NewReconcilerWithClock(store, stripe, time.Now)
}

func NewReconcilerWithClock(store ReconciliationStore, stripe StripeSubscriptionClient, now func() time.Time) Reconciler {
	return Reconciler{store: store, stripe: stripe, now: now}
}

func (reconciler Reconciler) Run(ctx context.Context) (ReconciliationResult, error) {
	locals, err := reconciler.store.ListLocalSubscriptions(ctx)
	if err != nil {
		return ReconciliationResult{}, err
	}

	result := ReconciliationResult{Checked: len(locals)}
	for _, local := range locals {
		if local.StripeSubscriptionID == "" {
			result.Skipped++
			continue
		}
		stripeSubscription, err := reconciler.stripe.GetSubscription(ctx, local.StripeSubscriptionID)
		if err != nil {
			return result, err
		}
		desired := entitlementFromStripe(local, stripeSubscription)
		if local.Status == Status(desired.Status) && local.Tier == Tier(desired.Plan) && sameExpiry(local.ExpiresAt, desired.ExpiresAt) {
			continue
		}
		repair := Repair{
			UserID: local.UserID,
			From:   local.Status,
			To:     Status(desired.Status),
			Reason: "stripe_drift",
		}
		if err := reconciler.store.UpsertEntitlement(ctx, desired); err != nil {
			return result, err
		}
		if err := reconciler.store.WriteReconciliationAudit(ctx, ReconciliationAuditEvent{
			Action:    "entitlement.reconciled",
			Target:    "user:" + local.UserID.String(),
			Metadata:  repairMetadata(local, stripeSubscription, repair),
			CreatedAt: reconciler.now().UTC(),
		}); err != nil {
			return result, err
		}
		result.Repaired++
		result.Changes = append(result.Changes, repair)
	}
	return result, nil
}

func entitlementFromStripe(local LocalSubscription, stripe StripeSubscription) repositories.EntitlementEntity {
	status := StatusActive
	tier := TierPaid
	if stripe.CancelAtPeriodEnd {
		status = StatusCancelled
	}
	switch stripe.Status {
	case "active", "trialing":
		if stripe.Status == "trialing" {
			tier = TierTrial
		}
	case "past_due", "unpaid":
		status = StatusPastDue
	case "canceled", "cancelled":
		status = StatusCancelled
	default:
		status = StatusExpired
		tier = TierFree
	}
	return repositories.EntitlementEntity{
		UserID:    local.UserID,
		Plan:      string(tier),
		Status:    string(status),
		ExpiresAt: stripe.CurrentPeriodEnd,
	}
}

func sameExpiry(left *time.Time, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func repairMetadata(local LocalSubscription, stripe StripeSubscription, repair Repair) []byte {
	payload := map[string]any{
		"stripeSubscriptionId": local.StripeSubscriptionID,
		"stripeStatus":         stripe.Status,
		"from":                 repair.From,
		"to":                   repair.To,
		"reason":               repair.Reason,
	}
	metadata, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{}`)
	}
	return metadata
}

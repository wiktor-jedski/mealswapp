// Package subscription provides entitlement and billing services.
// Implements DESIGN-007 EntitlementManager.
package subscription

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Feature flags for entitlement checks.
// Implements DESIGN-007 EntitlementManager.
const (
	FeatureCatalog              = "catalog"
	FeatureSubstitutionSingle   = "substitution:single"
	FeatureSubstitutionMulti    = "substitution:multi"
	FeatureDailyDiet            = "daily_diet"
	FeatureDailyDietAlternative = "daily_diet_alternative"
)

// Decision represents an access control decision for a requested feature.
// Implements DESIGN-007 EntitlementManager.
type Decision struct {
	Allowed bool
	Reason  string
}

// EntitlementManager provides feature access control based on user subscription state.
// Implements DESIGN-007 EntitlementManager.
type EntitlementManager struct {
	repo repository.EntitlementRepository
}

// NewEntitlementManager creates a new EntitlementManager.
// Implements DESIGN-007 EntitlementManager.
func NewEntitlementManager(repo repository.EntitlementRepository) *EntitlementManager {
	return &EntitlementManager{repo: repo}
}

// CheckEntitlement resolves the user's entitlement and makes a feature access decision.
// Implements DESIGN-007 EntitlementManager.
func (m *EntitlementManager) CheckEntitlement(ctx context.Context, userID uuid.UUID, feature string) (Decision, error) {
	// decisions never trust client-supplied user IDs
	if userID == uuid.Nil {
		return Decision{Allowed: false, Reason: "invalid user identity"}, errors.New("decisions never trust client-supplied user IDs")
	}

	ent, err := m.repo.GetLatest(ctx, userID)
	if err != nil {
		// Missing entitlement falls back to free behavior.
		ent = repository.Entitlement{
			Tier:   "free",
			Status: "active",
		}
	}

	// Trial and paid active users allow all Phase 06-visible paid modes.
	if (ent.Tier == "trial" || ent.Tier == "paid") && ent.Status == "active" {
		return Decision{Allowed: true, Reason: "active subscription"}, nil
	}

	// Free active users allow Catalog and single-input Substitution only.
	// Expired/past_due/cancelled users block paid-only modes.
	switch feature {
	case FeatureCatalog, FeatureSubstitutionSingle:
		return Decision{Allowed: true, Reason: "free feature"}, nil
	default:
		return Decision{Allowed: false, Reason: "requires active subscription"}, nil
	}
}

// GetEntitlementState returns the user's raw entitlement state from the repository.
// Missing entitlements default to free active.
// Implements DESIGN-007 EntitlementManager.
func (m *EntitlementManager) GetEntitlementState(ctx context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	if userID == uuid.Nil {
		return repository.Entitlement{}, errors.New("invalid user identity")
	}
	ent, err := m.repo.GetLatest(ctx, userID)
	if err != nil {
		// Missing entitlement falls back to free behavior.
		ent = repository.Entitlement{
			Tier:   "free",
			Status: "active",
		}
	}
	return ent, nil
}

// ReconcileStripeEntitlements queries Stripe sandbox subscriptions and repairs local entitlement drift.
// Implements DESIGN-007 EntitlementManager.
func (m *EntitlementManager) ReconcileStripeEntitlements(ctx context.Context, gateway StripeSubscriptionGateway) error {
	subs, err := gateway.ListSubscriptions(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list subscriptions from stripe during reconciliation", "error", err)
		return err
	}

	for _, s := range subs {
		ent, err := m.repo.GetLatestByStripeSubscription(ctx, s.SubscriptionID)
		if err != nil {
			if repository.IsKind(err, repository.ErrorKindNotFound) {
				continue // Cannot repair missing entitlement without knowing UserID, relies on checkout session
			}
			slog.ErrorContext(ctx, "failed to get latest entitlement by subscription during reconciliation", "subscription_id", s.SubscriptionID, "error", err)
			return err
		}

		var newStatus string
		switch s.Status {
		case "active", "trialing":
			newStatus = "active"
		case "past_due", "unpaid":
			newStatus = "past_due"
		case "canceled", "incomplete_expired":
			newStatus = "cancelled"
		default:
			continue
		}

		if ent.Status != newStatus || ent.Tier != "paid" {
			slog.InfoContext(ctx, "repairing entitlement drift", "user_id", ent.UserID, "old_status", ent.Status, "new_status", newStatus)
			newEnt := repository.Entitlement{
				UserID:               ent.UserID,
				Tier:                 "paid",
				Status:               newStatus,
				StripeCustomerID:     ent.StripeCustomerID,
				StripeSubscriptionID: ent.StripeSubscriptionID,
			}

			allowedModes := []string{"catalog", "substitution:single", "substitution:multi", "daily_diet", "daily_diet_alternative"}
			if newStatus != "active" {
				allowedModes = []string{"catalog", "substitution:single"}
			}
			newEnt.AllowedModes = allowedModes

			if err := m.repo.AppendEntitlement(ctx, newEnt); err != nil {
				slog.ErrorContext(ctx, "failed to append repaired entitlement", "user_id", ent.UserID, "error", err)
				return err
			}
		}
	}
	return nil
}

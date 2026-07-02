// Package entitlement resolves subscription-backed feature access.
package entitlement

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Feature identifies one entitlement-protected backend capability.
// Implements DESIGN-007 EntitlementManager.
type Feature string

// Implements DESIGN-007 EntitlementManager feature decisions.
const (
	FeatureCatalog              Feature = "catalog"
	FeatureSingleSubstitution   Feature = "single_substitution"
	FeatureMultiSubstitution    Feature = "multi_substitution"
	FeatureDailyDiet            Feature = "daily_diet"
	FeatureDailyDietAlternative Feature = "daily_diet_alternative"
)

// Implements DESIGN-007 EntitlementManager and UsageLimiter free-tier search cap.
const freeSearchLimitPer24h = 3

// DenyReason identifies why a feature decision was blocked.
// Implements DESIGN-007 EntitlementManager.
type DenyReason string

// Implements DESIGN-007 EntitlementManager decision states.
const (
	DenyReasonNone              DenyReason = ""
	DenyReasonInvalidFeature    DenyReason = "invalid_feature"
	DenyReasonFreeTierScope     DenyReason = "free_tier_scope"
	DenyReasonInactivePaidState DenyReason = "inactive_paid_state"
)

// Decision reports the server-side entitlement decision for one feature.
// Implements DESIGN-007 EntitlementManager.
type Decision struct {
	UserID     uuid.UUID
	Feature    Feature
	Allowed    bool
	Tier       string
	Status     string
	DenyReason DenyReason
}

// DecisionRequest carries the authenticated identity and requested feature.
// ClientSuppliedUserID is deliberately ignored; handlers must pass the
// authenticated server-side identity in AuthenticatedUserID.
// Implements DESIGN-007 EntitlementManager.
type DecisionRequest struct {
	AuthenticatedUserID  uuid.UUID
	ClientSuppliedUserID *uuid.UUID
	Feature              Feature
}

// EntitlementManager resolves repository entitlement state into feature access.
// Implements DESIGN-007 EntitlementManager.
type EntitlementManager struct {
	repo repository.EntitlementRepository
}

// NewEntitlementManager creates an entitlement decision service.
// Implements DESIGN-007 EntitlementManager.
func NewEntitlementManager(repo repository.EntitlementRepository) *EntitlementManager {
	return &EntitlementManager{repo: repo}
}

// CheckEntitlement resolves access for one authenticated user and feature.
// Implements DESIGN-007 EntitlementManager.
func (m *EntitlementManager) CheckEntitlement(ctx context.Context, userID uuid.UUID, feature Feature) (Decision, error) {
	return m.Decide(ctx, DecisionRequest{AuthenticatedUserID: userID, Feature: feature})
}

// Decide resolves access using only the authenticated server-side user ID.
// Implements DESIGN-007 EntitlementManager.
func (m *EntitlementManager) Decide(ctx context.Context, req DecisionRequest) (Decision, error) {
	if req.AuthenticatedUserID == uuid.Nil {
		return Decision{}, repository.NewError(repository.ErrorKindValidation, "authenticated user id is required", nil)
	}
	if !validFeature(req.Feature) {
		return Decision{UserID: req.AuthenticatedUserID, Feature: req.Feature, Tier: "free", Status: "active", DenyReason: DenyReasonInvalidFeature}, nil
	}

	entitlement, err := m.repo.GetLatest(ctx, req.AuthenticatedUserID)
	if err != nil {
		if !repository.IsKind(err, repository.ErrorKindNotFound) {
			return Decision{}, err
		}
		entitlement = freeFallbackEntitlement(req.AuthenticatedUserID)
	}
	return decideFromEntitlement(entitlement, req.Feature), nil
}

// decideFromEntitlement maps persisted tier/status to free and paid feature scopes.
// Implements DESIGN-007 EntitlementManager.
func decideFromEntitlement(entitlement repository.Entitlement, feature Feature) Decision {
	decision := Decision{
		UserID:  entitlement.UserID,
		Feature: feature,
		Tier:    entitlement.Tier,
		Status:  entitlement.Status,
	}
	if !validFeature(feature) {
		decision.DenyReason = DenyReasonInvalidFeature
		return decision
	}
	if freeFeature(feature) {
		decision.Allowed = true
		return decision
	}
	if entitlement.Status != "active" {
		decision.DenyReason = DenyReasonInactivePaidState
		return decision
	}
	if entitlement.Tier == "trial" || entitlement.Tier == "paid" {
		decision.Allowed = true
		return decision
	}
	decision.DenyReason = DenyReasonFreeTierScope
	return decision
}

// freeFallbackEntitlement applies free behavior when no entitlement row exists.
// Implements DESIGN-007 EntitlementManager.
func freeFallbackEntitlement(userID uuid.UUID) repository.Entitlement {
	return repository.Entitlement{
		UserID:            userID,
		Tier:              "free",
		Status:            "active",
		SearchLimitPer24h: freeSearchLimitPer24h,
		AllowedModes:      []string{"catalog", "substitution"},
	}
}

// validFeature reports whether a feature is known to Phase 06 entitlement checks.
// Implements DESIGN-007 EntitlementManager.
func validFeature(feature Feature) bool {
	switch feature {
	case FeatureCatalog, FeatureSingleSubstitution, FeatureMultiSubstitution, FeatureDailyDiet, FeatureDailyDietAlternative:
		return true
	default:
		return false
	}
}

// freeFeature reports whether a free-scope decision can allow the feature.
// Implements DESIGN-007 EntitlementManager.
func freeFeature(feature Feature) bool {
	return feature == FeatureCatalog || feature == FeatureSingleSubstitution
}

// IsEntitlementValidationError reports validation failures from this service boundary.
// Implements DESIGN-007 EntitlementManager.
func IsEntitlementValidationError(err error) bool {
	var repoErr *repository.Error
	return errors.As(err, &repoErr) && repoErr.Kind == repository.ErrorKindValidation
}

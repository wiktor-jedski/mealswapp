package entitlement

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Status reports sanitized entitlement and billing state for frontend reads.
// Implements DESIGN-007 SubscriptionController.
type Status struct {
	UserID               uuid.UUID
	Tier                 string
	EntitlementStatus    string
	AllowedModes         []string
	SearchLimitPer24h    int
	UsageUsed            int
	UsageRemaining       *int
	UsageWindowStartedAt *time.Time
	TrialExpiresAt       *time.Time
	BillingRecoveryState string
}

// StatusService resolves frontend-safe entitlement and usage state.
// Implements DESIGN-007 SubscriptionController.
type StatusService struct {
	entitlements repository.EntitlementRepository
	usage        repository.UsageRepository
	now          func() time.Time
}

// NewStatusService creates a read-only entitlement status service.
// Implements DESIGN-007 SubscriptionController.
func NewStatusService(entitlements repository.EntitlementRepository, usage repository.UsageRepository) *StatusService {
	return NewStatusServiceWithClock(entitlements, usage, time.Now)
}

// NewStatusServiceWithClock creates a status service with an injectable clock.
// Implements DESIGN-007 SubscriptionController.
func NewStatusServiceWithClock(entitlements repository.EntitlementRepository, usage repository.UsageRepository, now func() time.Time) *StatusService {
	if now == nil {
		now = time.Now
	}
	return &StatusService{entitlements: entitlements, usage: usage, now: now}
}

// GetEntitlementStatus returns sanitized billing and access state for one user.
// Implements DESIGN-007 SubscriptionController.
func (s *StatusService) GetEntitlementStatus(ctx context.Context, userID uuid.UUID) (Status, error) {
	if s == nil || s.entitlements == nil || s.usage == nil {
		return Status{}, repository.NewError(repository.ErrorKindValidation, "entitlement status dependencies are required", nil)
	}
	if userID == uuid.Nil {
		return Status{}, repository.NewError(repository.ErrorKindValidation, "authenticated user id is required", nil)
	}

	entitlementState, err := s.entitlements.GetLatest(ctx, userID)
	if err != nil {
		if !repository.IsKind(err, repository.ErrorKindNotFound) {
			return Status{}, err
		}
		entitlementState = freeFallbackEntitlement(userID)
	}

	status := Status{
		UserID:               userID,
		Tier:                 entitlementState.Tier,
		EntitlementStatus:    entitlementState.Status,
		AllowedModes:         allowedSearchModes(entitlementState),
		SearchLimitPer24h:    entitlementState.SearchLimitPer24h,
		TrialExpiresAt:       entitlementState.ExpiresAt,
		BillingRecoveryState: billingRecoveryState(entitlementState),
	}
	if entitlementState.Tier == "free" && entitlementState.Status == "active" && entitlementState.SearchLimitPer24h > 0 {
		since := s.now().UTC().Add(-freeUsageWindowDuration)
		window, err := s.usage.GetUsageSince(ctx, userID, UsageFeatureSearch, since)
		if err != nil {
			return Status{}, err
		}
		remaining := max(entitlementState.SearchLimitPer24h-window.SearchCount, 0)
		status.UsageUsed = window.SearchCount
		status.UsageRemaining = &remaining
		status.UsageWindowStartedAt = &since
	}
	return status, nil
}

// allowedSearchModes maps server decisions to frontend-visible search modes.
// Implements DESIGN-007 SubscriptionController.
func allowedSearchModes(entitlementState repository.Entitlement) []string {
	modeSet := map[string]struct{}{}
	for _, feature := range []Feature{FeatureCatalog, FeatureSingleSubstitution, FeatureMultiSubstitution, FeatureDailyDiet, FeatureDailyDietAlternative} {
		decision := decideFromEntitlement(entitlementState, feature)
		if decision.Allowed {
			modeSet[modeForFeature(feature)] = struct{}{}
		}
	}

	modes := []string{}
	for _, mode := range []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"} {
		if _, ok := modeSet[mode]; ok {
			modes = append(modes, mode)
		}
	}
	return modes
}

// modeForFeature returns the public search mode controlled by one entitlement feature.
// Implements DESIGN-007 SubscriptionController.
func modeForFeature(feature Feature) string {
	switch feature {
	case FeatureDailyDiet:
		return "daily_diet"
	case FeatureDailyDietAlternative:
		return "daily_diet_alternative"
	case FeatureSingleSubstitution, FeatureMultiSubstitution:
		return "substitution"
	default:
		return "catalog"
	}
}

// billingRecoveryState maps provider states to frontend-safe recovery hints.
// Implements DESIGN-007 SubscriptionController.
func billingRecoveryState(entitlementState repository.Entitlement) string {
	switch entitlementState.Status {
	case "past_due":
		return "action_required"
	case "cancelled":
		return "cancelled"
	case "expired":
		return "expired"
	default:
		return "none"
	}
}

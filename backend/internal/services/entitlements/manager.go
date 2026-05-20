package entitlements

import (
	"context"
	"slices"
	"time"

	"mealswapp/backend/internal/repositories"
	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Tier string
type Status string
type Feature string

const (
	TierFree  Tier = "free"
	TierTrial Tier = "trial"
	TierPaid  Tier = "paid"

	StatusActive    Status = "active"
	StatusExpired   Status = "expired"
	StatusPastDue   Status = "past_due"
	StatusCancelled Status = "cancelled"

	FeatureSingle     Feature = "single"
	FeatureIngredient Feature = "ingredient"
	FeatureMeal       Feature = "meal"
	FeatureDiet       Feature = "diet"
)

const FreeSearchLimitPer24h = 3

var (
	freeModes = []searchsvc.Mode{searchsvc.ModeSingle}
	paidModes = []searchsvc.Mode{
		searchsvc.ModeSingle,
		searchsvc.ModeReplacement,
		searchsvc.ModeDiet,
	}
	freeFeatures = []Feature{FeatureSingle}
	paidFeatures = []Feature{FeatureSingle, FeatureIngredient, FeatureMeal, FeatureDiet}
)

type Repository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error)
}

type Manager struct {
	repository Repository
	now        func() time.Time
}

type Entitlement struct {
	UserID               uuid.UUID        `json:"userId"`
	Tier                 Tier             `json:"tier"`
	Status               Status           `json:"status"`
	SearchLimitPer24h    int              `json:"searchLimitPer24h"`
	AllowedModes         []searchsvc.Mode `json:"allowedModes"`
	AllowedFeatures      []Feature        `json:"allowedFeatures"`
	ExpiresAt            *time.Time       `json:"expiresAt,omitempty"`
	StripeCustomerID     string           `json:"stripeCustomerId,omitempty"`
	StripeSubscriptionID string           `json:"stripeSubscriptionId,omitempty"`
}

type Decision struct {
	Allowed     bool        `json:"allowed"`
	Code        string      `json:"code"`
	Reason      string      `json:"reason"`
	Entitlement Entitlement `json:"entitlement"`
}

type Plan struct {
	ID                string           `json:"id"`
	Tier              Tier             `json:"tier"`
	Interval          string           `json:"interval"`
	PriceCents        int              `json:"priceCents"`
	SearchLimitPer24h int              `json:"searchLimitPer24h"`
	AllowedModes      []searchsvc.Mode `json:"allowedModes"`
	AllowedFeatures   []Feature        `json:"allowedFeatures"`
}

func NewManager(repository Repository) Manager {
	return Manager{repository: repository, now: time.Now}
}

func NewManagerWithClock(repository Repository, now func() time.Time) Manager {
	return Manager{repository: repository, now: now}
}

func (manager Manager) Get(ctx context.Context, userID *uuid.UUID) (Entitlement, error) {
	if userID == nil || *userID == uuid.Nil {
		return manager.freeEntitlement(uuid.Nil), nil
	}
	entity, err := manager.repository.GetByUserID(ctx, *userID)
	if err == nil {
		return manager.normalize(entity), nil
	}
	if err == pgx.ErrNoRows {
		return manager.freeEntitlement(*userID), nil
	}
	return Entitlement{}, err
}

func (manager Manager) CheckMode(ctx context.Context, userID *uuid.UUID, mode searchsvc.Mode, searchesUsed int) (Decision, error) {
	entitlement, err := manager.Get(ctx, userID)
	if err != nil {
		return Decision{}, err
	}
	if !slices.Contains(entitlement.AllowedModes, mode) {
		return Decision{
			Allowed:     false,
			Code:        "mode_not_allowed",
			Reason:      "Current plan does not allow this search mode.",
			Entitlement: entitlement,
		}, nil
	}
	if entitlement.SearchLimitPer24h >= 0 && searchesUsed >= entitlement.SearchLimitPer24h {
		return Decision{
			Allowed:     false,
			Code:        "search_limit_reached",
			Reason:      "Free plan search limit reached for the current 24-hour window.",
			Entitlement: entitlement,
		}, nil
	}
	return Decision{Allowed: true, Code: "allowed", Entitlement: entitlement}, nil
}

func (manager Manager) CheckFeature(ctx context.Context, userID *uuid.UUID, feature Feature, searchesUsed int) (Decision, error) {
	entitlement, err := manager.Get(ctx, userID)
	if err != nil {
		return Decision{}, err
	}
	if !slices.Contains(entitlement.AllowedFeatures, feature) {
		return Decision{
			Allowed:     false,
			Code:        "feature_not_allowed",
			Reason:      "Current plan does not allow this feature.",
			Entitlement: entitlement,
		}, nil
	}
	if feature == FeatureSingle && entitlement.SearchLimitPer24h >= 0 && searchesUsed >= entitlement.SearchLimitPer24h {
		return Decision{
			Allowed:     false,
			Code:        "search_limit_reached",
			Reason:      "Free plan search limit reached for the current 24-hour window.",
			Entitlement: entitlement,
		}, nil
	}
	return Decision{Allowed: true, Code: "allowed", Entitlement: entitlement}, nil
}

func LookupPlan(planID string) (Plan, bool) {
	plans := map[string]Plan{
		"free": {
			ID:                "free",
			Tier:              TierFree,
			Interval:          "none",
			PriceCents:        0,
			SearchLimitPer24h: FreeSearchLimitPer24h,
			AllowedModes:      append([]searchsvc.Mode(nil), freeModes...),
			AllowedFeatures:   append([]Feature(nil), freeFeatures...),
		},
		"paid_monthly": {
			ID:                "paid_monthly",
			Tier:              TierPaid,
			Interval:          "monthly",
			PriceCents:        300,
			SearchLimitPer24h: -1,
			AllowedModes:      append([]searchsvc.Mode(nil), paidModes...),
			AllowedFeatures:   append([]Feature(nil), paidFeatures...),
		},
		"paid_annual": {
			ID:                "paid_annual",
			Tier:              TierPaid,
			Interval:          "annual",
			PriceCents:        2500,
			SearchLimitPer24h: -1,
			AllowedModes:      append([]searchsvc.Mode(nil), paidModes...),
			AllowedFeatures:   append([]Feature(nil), paidFeatures...),
		},
	}
	plan, ok := plans[planID]
	return plan, ok
}

func (manager Manager) normalize(entity repositories.EntitlementEntity) Entitlement {
	tier := Tier(entity.Plan)
	status := Status(entity.Status)
	if status == "" {
		status = StatusActive
	}
	if !slices.Contains([]Tier{TierFree, TierTrial, TierPaid}, tier) {
		tier = TierFree
	}
	if !slices.Contains([]Status{StatusActive, StatusExpired, StatusPastDue, StatusCancelled}, status) {
		status = StatusExpired
	}

	if status != StatusActive || (entity.ExpiresAt != nil && !entity.ExpiresAt.After(manager.now().UTC())) {
		return Entitlement{
			UserID:            entity.UserID,
			Tier:              TierFree,
			Status:            expiredStatus(status),
			SearchLimitPer24h: FreeSearchLimitPer24h,
			AllowedModes:      append([]searchsvc.Mode(nil), freeModes...),
			AllowedFeatures:   append([]Feature(nil), freeFeatures...),
			ExpiresAt:         entity.ExpiresAt,
		}
	}
	if tier == TierTrial || tier == TierPaid {
		return Entitlement{
			UserID:            entity.UserID,
			Tier:              tier,
			Status:            StatusActive,
			SearchLimitPer24h: -1,
			AllowedModes:      append([]searchsvc.Mode(nil), paidModes...),
			AllowedFeatures:   append([]Feature(nil), paidFeatures...),
			ExpiresAt:         entity.ExpiresAt,
		}
	}
	return manager.freeEntitlement(entity.UserID)
}

func (manager Manager) freeEntitlement(userID uuid.UUID) Entitlement {
	return Entitlement{
		UserID:            userID,
		Tier:              TierFree,
		Status:            StatusActive,
		SearchLimitPer24h: FreeSearchLimitPer24h,
		AllowedModes:      append([]searchsvc.Mode(nil), freeModes...),
		AllowedFeatures:   append([]Feature(nil), freeFeatures...),
	}
}

func expiredStatus(status Status) Status {
	if status == StatusCancelled || status == StatusPastDue {
		return status
	}
	return StatusExpired
}

package entitlements

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"mealswapp/backend/internal/repositories"
	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestAnonymousGetsFreeSingleModeEntitlement(t *testing.T) {
	manager := NewManagerWithClock(fakeRepository{}, fixedNow)

	decision, err := manager.CheckMode(context.Background(), nil, searchsvc.ModeSingle, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.Entitlement.Tier != TierFree || decision.Entitlement.SearchLimitPer24h != 3 {
		t.Fatalf("expected anonymous free decision, got %#v", decision)
	}
}

func TestFreeEntitlementBlocksPaidModesAndSearchLimit(t *testing.T) {
	userID := uuid.New()
	manager := NewManagerWithClock(fakeRepository{err: pgx.ErrNoRows}, fixedNow)

	replacement, err := manager.CheckMode(context.Background(), &userID, searchsvc.ModeReplacement, 0)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Allowed || replacement.Code != "mode_not_allowed" {
		t.Fatalf("expected replacement mode blocked for free user, got %#v", replacement)
	}

	limited, err := manager.CheckMode(context.Background(), &userID, searchsvc.ModeSingle, 3)
	if err != nil {
		t.Fatal(err)
	}
	if limited.Allowed || limited.Code != "search_limit_reached" {
		t.Fatalf("expected free limit block, got %#v", limited)
	}
}

func TestActiveTrialAllowsPaidFeaturesUntilExpiry(t *testing.T) {
	userID := uuid.New()
	expiresAt := fixedNow().Add(7 * 24 * time.Hour)
	manager := NewManagerWithClock(fakeRepository{entity: repositories.EntitlementEntity{
		UserID:    userID,
		Plan:      "trial",
		Status:    "active",
		ExpiresAt: &expiresAt,
	}}, fixedNow)

	decision, err := manager.CheckFeature(context.Background(), &userID, FeatureDiet, 99)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.Entitlement.SearchLimitPer24h != -1 || decision.Entitlement.Tier != TierTrial {
		t.Fatalf("expected active trial to allow diet feature, got %#v", decision)
	}
}

func TestPaidBypassesFreeLimitAndAllowsAllSearchModes(t *testing.T) {
	userID := uuid.New()
	manager := NewManagerWithClock(fakeRepository{entity: repositories.EntitlementEntity{
		UserID: userID,
		Plan:   "paid",
		Status: "active",
	}}, fixedNow)

	decision, err := manager.CheckMode(context.Background(), &userID, searchsvc.ModeDiet, 300)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.Entitlement.SearchLimitPer24h != -1 {
		t.Fatalf("expected paid user to bypass limit, got %#v", decision)
	}
}

func TestExpiredSubscriptionFallsBackToFreeScope(t *testing.T) {
	userID := uuid.New()
	expiresAt := fixedNow().Add(-time.Hour)
	manager := NewManagerWithClock(fakeRepository{entity: repositories.EntitlementEntity{
		UserID:    userID,
		Plan:      "paid",
		Status:    "active",
		ExpiresAt: &expiresAt,
	}}, fixedNow)

	decision, err := manager.CheckMode(context.Background(), &userID, searchsvc.ModeReplacement, 0)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Code != "mode_not_allowed" || decision.Entitlement.Tier != TierFree || decision.Entitlement.Status != StatusExpired {
		t.Fatalf("expected expired paid entitlement to fall back to free, got %#v", decision)
	}
}

func TestCancelledSubscriptionKeepsFreeSingleModeAccess(t *testing.T) {
	userID := uuid.New()
	manager := NewManagerWithClock(fakeRepository{entity: repositories.EntitlementEntity{
		UserID: userID,
		Plan:   "paid",
		Status: "cancelled",
	}}, fixedNow)

	decision, err := manager.CheckMode(context.Background(), &userID, searchsvc.ModeSingle, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.Entitlement.Tier != TierFree || decision.Entitlement.Status != StatusCancelled {
		t.Fatalf("expected cancelled subscription to retain free single access, got %#v", decision)
	}
}

func TestExpiredTrialFallsBackToFreeScope(t *testing.T) {
	userID := uuid.New()
	expiresAt := fixedNow().Add(-time.Minute)
	manager := NewManagerWithClock(fakeRepository{entity: repositories.EntitlementEntity{
		UserID:    userID,
		Plan:      "trial",
		Status:    "active",
		ExpiresAt: &expiresAt,
	}}, fixedNow)

	entitlement, err := manager.Get(context.Background(), &userID)
	if err != nil {
		t.Fatal(err)
	}
	if entitlement.Tier != TierFree || entitlement.Status != StatusExpired || len(entitlement.AllowedModes) != 1 || entitlement.AllowedModes[0] != searchsvc.ModeSingle {
		t.Fatalf("expected expired trial fallback, got %#v", entitlement)
	}
}

func TestPlanLookupAndEntitlementJSONShape(t *testing.T) {
	plan, ok := LookupPlan("paid_annual")
	if !ok {
		t.Fatal("expected annual paid plan")
	}
	if plan.PriceCents != 2500 || plan.Interval != "annual" || !containsMode(plan.AllowedModes, searchsvc.ModeDiet) {
		t.Fatalf("unexpected annual plan: %#v", plan)
	}

	payload, err := json.Marshal(Entitlement{
		UserID:            uuid.MustParse("00000000-0000-0000-0000-000000000123"),
		Tier:              TierFree,
		Status:            StatusActive,
		SearchLimitPer24h: FreeSearchLimitPer24h,
		AllowedModes:      []searchsvc.Mode{searchsvc.ModeSingle},
		AllowedFeatures:   []Feature{FeatureSingle},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"userId":"00000000-0000-0000-0000-000000000123","tier":"free","status":"active","searchLimitPer24h":3,"allowedModes":["single"],"allowedFeatures":["single"]}` {
		t.Fatalf("unexpected entitlement JSON: %s", payload)
	}
}

type fakeRepository struct {
	entity repositories.EntitlementEntity
	err    error
}

func (repo fakeRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error) {
	if repo.err != nil {
		return repositories.EntitlementEntity{}, repo.err
	}
	return repo.entity, nil
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
}

func containsMode(modes []searchsvc.Mode, mode searchsvc.Mode) bool {
	for _, candidate := range modes {
		if candidate == mode {
			return true
		}
	}
	return false
}

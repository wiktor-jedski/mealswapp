package entitlements

import (
	"context"
	"testing"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestTrialTrackerCreatesSevenDaySocialLoginTrial(t *testing.T) {
	userID := uuid.New()
	now := fixedNow()
	repo := newFakeTrialRepository()
	tracker := NewTrialTrackerWithClock(repo, func() time.Time { return now })

	entitlement, err := tracker.StartTrialForSource(context.Background(), userID, TrialStartSourceSocialLogin)
	if err != nil {
		t.Fatal(err)
	}
	if entitlement.Tier != TierTrial || entitlement.Status != StatusActive || entitlement.SearchLimitPer24h != -1 {
		t.Fatalf("expected active trial entitlement, got %#v", entitlement)
	}
	if entitlement.ExpiresAt == nil || !entitlement.ExpiresAt.Equal(now.Add(TrialDuration)) {
		t.Fatalf("expected seven-day expiry, got %#v", entitlement.ExpiresAt)
	}
	stored := repo.items[userID]
	if stored.Plan != "trial" || stored.Status != "active" {
		t.Fatalf("expected persisted trial, got %#v", stored)
	}
}

func TestTrialTrackerRejectsNonSocialTrialStarts(t *testing.T) {
	tracker := NewTrialTrackerWithClock(newFakeTrialRepository(), fixedNow)

	_, err := tracker.StartTrialForSource(context.Background(), uuid.New(), TrialStartSource("password_signup"))
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "validation_error" {
		t.Fatalf("expected validation error for non-social trial, got %v", err)
	}
}

func TestTrialTrackerPreventsDuplicateTrials(t *testing.T) {
	userID := uuid.New()
	expiresAt := fixedNow().Add(-time.Hour)
	repo := newFakeTrialRepository()
	repo.items[userID] = repositories.EntitlementEntity{
		UserID:    userID,
		Plan:      "trial",
		Status:    "expired",
		ExpiresAt: &expiresAt,
	}
	tracker := NewTrialTrackerWithClock(repo, fixedNow)

	err := tracker.StartTrial(context.Background(), userID)
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "conflict" {
		t.Fatalf("expected duplicate trial conflict, got %v", err)
	}
}

func TestTrialTrackerKeepsActivePaidEntitlement(t *testing.T) {
	userID := uuid.New()
	repo := newFakeTrialRepository()
	repo.items[userID] = repositories.EntitlementEntity{UserID: userID, Plan: "paid", Status: "active"}
	tracker := NewTrialTrackerWithClock(repo, fixedNow)

	entitlement, err := tracker.StartTrialForSource(context.Background(), userID, TrialStartSourceSocialLogin)
	if err != nil {
		t.Fatal(err)
	}
	if entitlement.Tier != TierPaid || repo.items[userID].Plan != "paid" {
		t.Fatalf("expected paid entitlement to remain unchanged, got entitlement=%#v stored=%#v", entitlement, repo.items[userID])
	}
}

func TestTrialTrackerExpiresElapsedTrials(t *testing.T) {
	userID := uuid.New()
	expiredAt := fixedNow().Add(-time.Second)
	repo := newFakeTrialRepository()
	repo.items[userID] = repositories.EntitlementEntity{
		UserID:    userID,
		Plan:      "trial",
		Status:    "active",
		ExpiresAt: &expiredAt,
	}
	tracker := NewTrialTrackerWithClock(repo, fixedNow)

	entitlement, changed, err := tracker.ExpireTrial(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || entitlement.Tier != TierFree || entitlement.Status != StatusExpired {
		t.Fatalf("expected expired trial downgraded to free scope, changed=%v entitlement=%#v", changed, entitlement)
	}
	if repo.items[userID].Status != "expired" {
		t.Fatalf("expected persisted expired status, got %#v", repo.items[userID])
	}
}

func TestTrialTrackerDoesNotExpireActiveOrNonTrialEntitlements(t *testing.T) {
	activeTrialUserID := uuid.New()
	paidUserID := uuid.New()
	expiresAt := fixedNow().Add(time.Hour)
	repo := newFakeTrialRepository()
	repo.items[activeTrialUserID] = repositories.EntitlementEntity{
		UserID:    activeTrialUserID,
		Plan:      "trial",
		Status:    "active",
		ExpiresAt: &expiresAt,
	}
	repo.items[paidUserID] = repositories.EntitlementEntity{UserID: paidUserID, Plan: "paid", Status: "active"}
	tracker := NewTrialTrackerWithClock(repo, fixedNow)

	count, err := tracker.ExpireTrials(context.Background(), []uuid.UUID{activeTrialUserID, paidUserID})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || repo.items[activeTrialUserID].Status != "active" || repo.items[paidUserID].Status != "active" {
		t.Fatalf("expected no expirations, count=%d items=%#v", count, repo.items)
	}
}

type fakeTrialRepository struct {
	items map[uuid.UUID]repositories.EntitlementEntity
}

func newFakeTrialRepository() *fakeTrialRepository {
	return &fakeTrialRepository{items: map[uuid.UUID]repositories.EntitlementEntity{}}
}

func (repo *fakeTrialRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error) {
	item, ok := repo.items[userID]
	if !ok {
		return repositories.EntitlementEntity{}, pgx.ErrNoRows
	}
	return item, nil
}

func (repo *fakeTrialRepository) Upsert(ctx context.Context, entitlement repositories.EntitlementEntity) error {
	repo.items[entitlement.UserID] = entitlement
	return nil
}

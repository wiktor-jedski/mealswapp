// Implements DESIGN-007 TrialTracker verification.
package entitlement

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type trialRepositoryStub struct {
	entitlements map[uuid.UUID]repository.Entitlement
	appended     []repository.Entitlement
	expired      []repository.Entitlement
	getErr       error
	appendErr    error
	listErr      error
}

func (r *trialRepositoryStub) AppendEntitlement(_ context.Context, entitlement repository.Entitlement) error {
	if r.appendErr != nil {
		return r.appendErr
	}
	if r.entitlements == nil {
		r.entitlements = map[uuid.UUID]repository.Entitlement{}
	}
	r.entitlements[entitlement.UserID] = entitlement
	r.appended = append(r.appended, entitlement)
	return nil
}

func (r *trialRepositoryStub) GetLatest(_ context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	if r.getErr != nil {
		return repository.Entitlement{}, r.getErr
	}
	entitlement, ok := r.entitlements[userID]
	if !ok {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "missing entitlement", nil)
	}
	return entitlement, nil
}

func (r *trialRepositoryStub) ListExpiredTrials(context.Context, time.Time) ([]repository.Entitlement, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.expired, nil
}

func TestTrialTrackerStartTrialCreatesOneSevenDayTrial(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 1, 8, 30, 0, 0, time.UTC)
	userID := uuid.New()
	repo := &trialRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{}}
	tracker := NewTrialTrackerWithClock(repo, repo, func() time.Time { return now })

	trial, err := tracker.StartTrial(ctx, userID)
	if err != nil {
		t.Fatalf("StartTrial() error = %v", err)
	}
	if trial.Tier != "trial" || trial.Status != "active" || trial.SearchLimitPer24h != 0 {
		t.Fatalf("trial entitlement = %#v, want active trial", trial)
	}
	if trial.ExpiresAt == nil || !trial.ExpiresAt.Equal(now.Add(7*24*time.Hour)) {
		t.Fatalf("trial expiry = %v, want %v", trial.ExpiresAt, now.Add(7*24*time.Hour))
	}
	if len(repo.appended) != 1 {
		t.Fatalf("appended entitlements = %d, want 1", len(repo.appended))
	}

	second, err := tracker.StartTrial(ctx, userID)
	if err != nil {
		t.Fatalf("second StartTrial() error = %v", err)
	}
	if second.ExpiresAt == nil || !second.ExpiresAt.Equal(*trial.ExpiresAt) || len(repo.appended) != 1 {
		t.Fatalf("second trial = %#v appended=%d, want existing trial without extension", second, len(repo.appended))
	}
}

func TestTrialTrackerConstructorDefaultsNilClockToNow(t *testing.T) {
	repo := &trialRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{}}
	tracker := NewTrialTrackerWithClock(repo, repo, nil)
	if tracker.now == nil {
		t.Fatal("NewTrialTrackerWithClock(nil clock) left now unset")
	}
}

func TestTrialTrackerDoesNotCreateTrialWhenAnyEntitlementExists(t *testing.T) {
	userID := uuid.New()
	existing := repository.Entitlement{UserID: userID, Tier: "paid", Status: "active", SearchLimitPer24h: 0, AllowedModes: paidModes()}
	repo := &trialRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{userID: existing}}
	tracker := NewTrialTracker(repo, repo)

	got, err := tracker.StartTrial(context.Background(), userID)
	if err != nil {
		t.Fatalf("StartTrial() error = %v", err)
	}
	if got.Tier != "paid" || len(repo.appended) != 0 {
		t.Fatalf("StartTrial() got=%#v appended=%d, want existing paid untouched", got, len(repo.appended))
	}
}

func TestTrialTrackerExpireTrialsDowngradesExpiredTrialsToFreeWithoutHistoryDeletion(t *testing.T) {
	// Verifies IT-ARCH-007-002.
	// Verifies ARCH-007.
	// Verifies ARCH-006.
	// Traces SW-REQ-046, SW-REQ-051, and SW-REQ-052.
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Minute)
	userID := uuid.New()
	trial := repository.Entitlement{UserID: userID, Tier: "trial", Status: "active", SearchLimitPer24h: 0, AllowedModes: paidModes(), ExpiresAt: &expiredAt}
	repo := &trialRepositoryStub{
		entitlements: map[uuid.UUID]repository.Entitlement{userID: trial},
		expired:      []repository.Entitlement{trial},
	}
	tracker := NewTrialTracker(repo, repo)

	if err := tracker.ExpireTrials(context.Background(), now); err != nil {
		t.Fatalf("ExpireTrials() error = %v", err)
	}
	if len(repo.appended) != 1 {
		t.Fatalf("appended entitlements = %d, want free downgrade", len(repo.appended))
	}
	downgrade := repo.appended[0]
	if downgrade.Tier != "free" || downgrade.Status != "active" || downgrade.SearchLimitPer24h != 3 || downgrade.ExpiresAt != nil {
		t.Fatalf("downgrade entitlement = %#v, want free active", downgrade)
	}

	if err := tracker.ExpireTrials(context.Background(), now); err != nil {
		t.Fatalf("second ExpireTrials() error = %v", err)
	}
	if len(repo.appended) != 1 {
		t.Fatalf("second expiry appended=%d, want idempotent no-op", len(repo.appended))
	}
}

func TestTrialTrackerExpireTrialsDoesNotDowngradePaidUsersOrUnexpiredTrials(t *testing.T) {
	// Verifies IT-ARCH-007-002.
	// Verifies ARCH-007.
	// Verifies ARCH-006.
	// Traces SW-REQ-046, SW-REQ-051, and SW-REQ-052.
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	futureAt := now.Add(time.Hour)
	paidUserID := uuid.New()
	unexpiredUserID := uuid.New()
	expiredPaidTrial := repository.Entitlement{UserID: paidUserID, Tier: "trial", Status: "active", SearchLimitPer24h: 0, AllowedModes: paidModes(), ExpiresAt: &expiredAt}
	unexpiredTrial := repository.Entitlement{UserID: unexpiredUserID, Tier: "trial", Status: "active", SearchLimitPer24h: 0, AllowedModes: paidModes(), ExpiresAt: &futureAt}
	repo := &trialRepositoryStub{
		entitlements: map[uuid.UUID]repository.Entitlement{
			paidUserID:      {UserID: paidUserID, Tier: "paid", Status: "active", SearchLimitPer24h: 0, AllowedModes: paidModes()},
			unexpiredUserID: unexpiredTrial,
		},
		expired: []repository.Entitlement{expiredPaidTrial, unexpiredTrial},
	}
	tracker := NewTrialTracker(repo, repo)

	if err := tracker.ExpireTrials(context.Background(), now); err != nil {
		t.Fatalf("ExpireTrials() error = %v", err)
	}
	if len(repo.appended) != 0 {
		t.Fatalf("appended entitlements = %#v, want no downgrade", repo.appended)
	}
}

func TestTrialTrackerValidationAndRepositoryErrors(t *testing.T) {
	tracker := NewTrialTracker(&trialRepositoryStub{}, &trialRepositoryStub{})
	if _, err := tracker.StartTrial(context.Background(), uuid.Nil); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("StartTrial(nil) error = %v, want validation", err)
	}
	if err := tracker.ExpireTrials(context.Background(), time.Time{}); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("ExpireTrials(zero) error = %v, want validation", err)
	}

	wantErr := errors.New("repository failed")
	repo := &trialRepositoryStub{getErr: wantErr}
	tracker = NewTrialTracker(repo, repo)
	if _, err := tracker.StartTrial(context.Background(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("StartTrial() error = %v, want %v", err, wantErr)
	}

	repo = &trialRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{}, appendErr: wantErr}
	tracker = NewTrialTracker(repo, repo)
	if _, err := tracker.StartTrial(context.Background(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("StartTrial() append error = %v, want %v", err, wantErr)
	}

	repo = &trialRepositoryStub{listErr: wantErr}
	tracker = NewTrialTracker(repo, repo)
	if err := tracker.ExpireTrials(context.Background(), time.Now()); !errors.Is(err, wantErr) {
		t.Fatalf("ExpireTrials() list error = %v, want %v", err, wantErr)
	}
}

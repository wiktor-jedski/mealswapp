package auth

// Implements DESIGN-006 AccountLockoutTracker verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryLockoutRepository struct {
	state      repository.AccountLockoutState
	err        error
	getErr     error
	failureErr error
	resetErr   error
}

func (r *memoryLockoutRepository) GetLockoutState(context.Context, uuid.UUID) (repository.AccountLockoutState, error) {
	if r.getErr != nil {
		return repository.AccountLockoutState{}, r.getErr
	}
	return r.state, r.err
}

func (r *memoryLockoutRepository) RecordFailedLogin(_ context.Context, userID uuid.UUID, threshold int, lockedUntil time.Time, now time.Time) (repository.AccountLockoutState, error) {
	if r.failureErr != nil {
		return repository.AccountLockoutState{}, r.failureErr
	}
	if r.err != nil {
		return repository.AccountLockoutState{}, r.err
	}
	if r.state.LockedUntil != nil && !r.state.LockedUntil.After(now) {
		r.state.FailedLoginCount = 0
		r.state.LockedUntil = nil
	}
	r.state.UserID = userID
	r.state.FailedLoginCount++
	if r.state.FailedLoginCount >= threshold {
		r.state.LockedUntil = &lockedUntil
	}
	return r.state, nil
}

func (r *memoryLockoutRepository) ResetFailedLogins(_ context.Context, userID uuid.UUID) (repository.AccountLockoutState, error) {
	if r.resetErr != nil {
		return repository.AccountLockoutState{}, r.resetErr
	}
	if r.err != nil {
		return repository.AccountLockoutState{}, r.err
	}
	r.state = repository.AccountLockoutState{UserID: userID}
	return r.state, nil
}

// TestAccountLockoutTracker verifies DESIGN-006 AccountLockoutTracker account policy.
func TestAccountLockoutTracker(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	repo := &memoryLockoutRepository{state: repository.AccountLockoutState{UserID: userID}}
	tracker := NewAccountLockoutTracker(repo)
	tracker.now = func() time.Time { return now }

	for i := 1; i <= 4; i++ {
		state, err := tracker.RecordFailure(ctx, userID)
		if err != nil {
			t.Fatalf("RecordFailure(%d) error = %v", i, err)
		}
		if state.Locked() || state.AccountFailures != i {
			t.Fatalf("failure %d state = %#v", i, state)
		}
	}
	state, err := tracker.RecordFailure(ctx, userID)
	if err != nil {
		t.Fatalf("RecordFailure(lock) error = %v", err)
	}
	if !state.Locked() || state.AccountFailures != 5 || state.RetryAfter != 15*time.Minute {
		t.Fatalf("locked state = %#v", state)
	}
	checked, err := tracker.Check(ctx, userID)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !checked.Locked() || checked.RetryAfter != 15*time.Minute {
		t.Fatalf("checked state = %#v", checked)
	}
	if err := tracker.RecordSuccess(ctx, userID); err != nil {
		t.Fatalf("RecordSuccess() error = %v", err)
	}
	checked, err = tracker.Check(ctx, userID)
	if err != nil {
		t.Fatalf("Check() after reset error = %v", err)
	}
	if checked.Locked() || checked.AccountFailures != 0 {
		t.Fatalf("reset state = %#v", checked)
	}
}

// TestAccountLockoutTrackerExpiredLocks verifies DESIGN-006 AccountLockoutTracker expiry behavior.
func TestAccountLockoutTrackerExpiredLocks(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Second)
	repo := &memoryLockoutRepository{state: repository.AccountLockoutState{UserID: userID, FailedLoginCount: 5, LockedUntil: &expired}}
	tracker := NewAccountLockoutTracker(repo)
	tracker.now = func() time.Time { return now }

	state, err := tracker.Check(ctx, userID)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if state.Locked() {
		t.Fatalf("expired state reported locked: %#v", state)
	}
	state, err = tracker.RecordFailure(ctx, userID)
	if err != nil {
		t.Fatalf("RecordFailure() error = %v", err)
	}
	if state.AccountFailures != 1 || state.Locked() {
		t.Fatalf("expired lock did not restart count: %#v", state)
	}
	if GenericInvalidCredentialMessage != "invalid email or password" {
		t.Fatal("generic invalid credential message changed")
	}
}

func TestAccountLockoutTrackerPropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("repository failed")
	tracker := NewAccountLockoutTracker(&memoryLockoutRepository{err: wantErr})
	if _, err := tracker.Check(context.Background(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("Check() error = %v", err)
	}
	if _, err := tracker.RecordFailure(context.Background(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("RecordFailure() error = %v", err)
	}
	if err := tracker.RecordSuccess(context.Background(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("RecordSuccess() error = %v", err)
	}
}

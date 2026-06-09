package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// LockoutState reports credential-attempt throttling state.
// Implements DESIGN-006 AccountLockoutTracker.
type LockoutState struct {
	AccountFailures int
	LockedUntil     *time.Time
	RetryAfter      time.Duration
}

// AccountLockoutTracker enforces account failed-login lockouts.
// Implements DESIGN-006 AccountLockoutTracker.
type AccountLockoutTracker struct {
	repo         repository.AccountLockoutRepository
	threshold    int
	lockDuration time.Duration
	now          func() time.Time
}

// NewAccountLockoutTracker creates a lockout tracker with a 5-failure, 15-minute policy.
// Implements DESIGN-006 AccountLockoutTracker.
func NewAccountLockoutTracker(repo repository.AccountLockoutRepository) *AccountLockoutTracker {
	return &AccountLockoutTracker{repo: repo, threshold: 5, lockDuration: 15 * time.Minute, now: time.Now}
}

// Check returns current lockout state before password verification.
// Implements DESIGN-006 AccountLockoutTracker.
func (t *AccountLockoutTracker) Check(ctx context.Context, userID uuid.UUID) (LockoutState, error) {
	state, err := t.repo.GetLockoutState(ctx, userID)
	if err != nil {
		return LockoutState{}, err
	}
	return t.toLockoutState(state), nil
}

// RecordFailure increments counters and returns generic lockout state.
// Implements DESIGN-006 AccountLockoutTracker.
func (t *AccountLockoutTracker) RecordFailure(ctx context.Context, userID uuid.UUID) (LockoutState, error) {
	now := t.now()
	state, err := t.repo.RecordFailedLogin(ctx, userID, t.threshold, now.Add(t.lockDuration), now)
	if err != nil {
		return LockoutState{}, err
	}
	return t.toLockoutState(state), nil
}

// RecordSuccess clears counters after a successful login.
// Implements DESIGN-006 AccountLockoutTracker.
func (t *AccountLockoutTracker) RecordSuccess(ctx context.Context, userID uuid.UUID) error {
	_, err := t.repo.ResetFailedLogins(ctx, userID)
	return err
}

// GenericInvalidCredentialMessage returns the only safe failed-login message.
// Implements DESIGN-006 AccountLockoutTracker.
func GenericInvalidCredentialMessage() string {
	return "invalid email or password"
}

// toLockoutState maps persisted state to runtime retry metadata.
// Implements DESIGN-006 AccountLockoutTracker.
func (t *AccountLockoutTracker) toLockoutState(state repository.AccountLockoutState) LockoutState {
	result := LockoutState{AccountFailures: state.FailedLoginCount, LockedUntil: state.LockedUntil}
	now := t.now()
	if state.LockedUntil != nil && state.LockedUntil.After(now) {
		result.RetryAfter = state.LockedUntil.Sub(now)
	}
	return result
}

// Locked reports whether an account lock is currently active.
// Implements DESIGN-006 AccountLockoutTracker.
func (s LockoutState) Locked() bool {
	return s.RetryAfter > 0
}

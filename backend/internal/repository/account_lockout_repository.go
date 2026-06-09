package repository

import (
	"context"
	_ "embed"
	"time"

	"github.com/google/uuid"
)

// Implements DESIGN-006 AccountLockoutTracker lookup query.
//
//go:embed sql/account_lockout_get.sql
var accountLockoutGetSQL string

// Implements DESIGN-006 AccountLockoutTracker failure counter query.
//
//go:embed sql/account_lockout_record_failure.sql
var accountLockoutRecordFailureSQL string

// Implements DESIGN-006 AccountLockoutTracker successful-login reset query.
//
//go:embed sql/account_lockout_reset.sql
var accountLockoutResetSQL string

// PostgresAccountLockoutRepository persists failed-login lockout counters.
// Implements DESIGN-006 AccountLockoutTracker.
type PostgresAccountLockoutRepository struct {
	db sqlExecutor
}

// NewPostgresAccountLockoutRepository creates a PostgreSQL lockout repository.
// Implements DESIGN-006 AccountLockoutTracker.
func NewPostgresAccountLockoutRepository(db sqlExecutor) *PostgresAccountLockoutRepository {
	return &PostgresAccountLockoutRepository{db: db}
}

// GetLockoutState loads persisted lockout metadata.
// Implements DESIGN-006 AccountLockoutTracker.
func (r *PostgresAccountLockoutRepository) GetLockoutState(ctx context.Context, userID uuid.UUID) (AccountLockoutState, error) {
	if userID == uuid.Nil {
		return AccountLockoutState{}, validationError("user id is required")
	}
	state := AccountLockoutState{UserID: userID}
	if err := r.db.QueryRow(ctx, accountLockoutGetSQL, userID).Scan(&state.FailedLoginCount, &state.LockedUntil); err != nil {
		return AccountLockoutState{}, mapPostgresError(err, "load account lockout")
	}
	return state, nil
}

// RecordFailedLogin increments failed-login state and locks accounts at threshold.
// Implements DESIGN-006 AccountLockoutTracker.
func (r *PostgresAccountLockoutRepository) RecordFailedLogin(ctx context.Context, userID uuid.UUID, threshold int, lockedUntil time.Time, now time.Time) (AccountLockoutState, error) {
	if userID == uuid.Nil {
		return AccountLockoutState{}, validationError("user id is required")
	}
	if threshold <= 0 || lockedUntil.IsZero() || now.IsZero() || !lockedUntil.After(now) {
		return AccountLockoutState{}, validationError("lockout parameters are invalid")
	}
	state := AccountLockoutState{UserID: userID}
	if err := r.db.QueryRow(ctx, accountLockoutRecordFailureSQL, userID, threshold, lockedUntil, now).Scan(&state.FailedLoginCount, &state.LockedUntil); err != nil {
		return AccountLockoutState{}, mapPostgresError(err, "record failed login")
	}
	return state, nil
}

// ResetFailedLogins clears failed-login state after successful login.
// Implements DESIGN-006 AccountLockoutTracker.
func (r *PostgresAccountLockoutRepository) ResetFailedLogins(ctx context.Context, userID uuid.UUID) (AccountLockoutState, error) {
	if userID == uuid.Nil {
		return AccountLockoutState{}, validationError("user id is required")
	}
	state := AccountLockoutState{UserID: userID}
	if err := r.db.QueryRow(ctx, accountLockoutResetSQL, userID).Scan(&state.FailedLoginCount, &state.LockedUntil); err != nil {
		return AccountLockoutState{}, mapPostgresError(err, "reset failed login")
	}
	return state, nil
}

// Implements DESIGN-006 AccountLockoutTracker compile-time repository contract.
var _ AccountLockoutRepository = (*PostgresAccountLockoutRepository)(nil)

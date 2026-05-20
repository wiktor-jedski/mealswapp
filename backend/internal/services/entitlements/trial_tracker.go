package entitlements

import (
	"context"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const TrialDuration = 7 * 24 * time.Hour

type TrialRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error)
	Upsert(ctx context.Context, entitlement repositories.EntitlementEntity) error
}

type TrialTracker struct {
	repository TrialRepository
	manager    Manager
	now        func() time.Time
}

type TrialStartSource string

const (
	TrialStartSourceSocialLogin TrialStartSource = "social_login"
)

func NewTrialTracker(repository TrialRepository) TrialTracker {
	return NewTrialTrackerWithClock(repository, time.Now)
}

func NewTrialTrackerWithClock(repository TrialRepository, now func() time.Time) TrialTracker {
	return TrialTracker{
		repository: repository,
		manager:    NewManagerWithClock(repository, now),
		now:        now,
	}
}

func (tracker TrialTracker) StartTrial(ctx context.Context, userID uuid.UUID) error {
	_, err := tracker.StartTrialForSource(ctx, userID, TrialStartSourceSocialLogin)
	return err
}

func (tracker TrialTracker) StartTrialForSource(ctx context.Context, userID uuid.UUID, source TrialStartSource) (Entitlement, error) {
	if userID == uuid.Nil {
		return Entitlement{}, validationFailure("userId", "required")
	}
	if source != TrialStartSourceSocialLogin {
		return Entitlement{}, validationFailure("source", "unsupported")
	}

	existing, err := tracker.repository.GetByUserID(ctx, userID)
	if err != nil && err != pgx.ErrNoRows {
		return Entitlement{}, err
	}
	if err == nil {
		if existing.Plan == string(TierTrial) {
			return Entitlement{}, apperrors.Conflict("Trial has already been used")
		}
		if existing.Plan == string(TierPaid) && existing.Status == string(StatusActive) {
			return tracker.manager.normalize(existing), nil
		}
	}

	expiresAt := tracker.now().UTC().Add(TrialDuration)
	entity := repositories.EntitlementEntity{
		UserID:    userID,
		Plan:      string(TierTrial),
		Status:    string(StatusActive),
		ExpiresAt: &expiresAt,
	}
	if err := tracker.repository.Upsert(ctx, entity); err != nil {
		return Entitlement{}, err
	}
	return tracker.manager.normalize(entity), nil
}

func (tracker TrialTracker) ExpireTrial(ctx context.Context, userID uuid.UUID) (Entitlement, bool, error) {
	if userID == uuid.Nil {
		return Entitlement{}, false, validationFailure("userId", "required")
	}
	existing, err := tracker.repository.GetByUserID(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return tracker.manager.freeEntitlement(userID), false, nil
		}
		return Entitlement{}, false, err
	}
	if existing.Plan != string(TierTrial) || existing.Status != string(StatusActive) || existing.ExpiresAt == nil || existing.ExpiresAt.After(tracker.now().UTC()) {
		return tracker.manager.normalize(existing), false, nil
	}

	existing.Status = string(StatusExpired)
	if err := tracker.repository.Upsert(ctx, existing); err != nil {
		return Entitlement{}, false, err
	}
	return tracker.manager.normalize(existing), true, nil
}

func (tracker TrialTracker) ExpireTrials(ctx context.Context, userIDs []uuid.UUID) (int, error) {
	expired := 0
	for _, userID := range userIDs {
		_, changed, err := tracker.ExpireTrial(ctx, userID)
		if err != nil {
			return expired, err
		}
		if changed {
			expired++
		}
	}
	return expired, nil
}

func validationFailure(field string, code string) error {
	return apperrors.Validation("Trial validation failed", []map[string]string{{"field": field, "code": code}})
}

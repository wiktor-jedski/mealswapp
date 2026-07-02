package entitlement

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TrialTracker owns first-social-login trial creation and expiry downgrade.
// Implements DESIGN-007 TrialTracker.
type TrialTracker struct {
	entitlements repository.EntitlementRepository
	trials       repository.TrialRepository
	now          func() time.Time
}

// NewTrialTracker creates the Phase 06 trial activation and expiry service.
// Implements DESIGN-007 TrialTracker.
func NewTrialTracker(entitlements repository.EntitlementRepository, trials repository.TrialRepository) *TrialTracker {
	return &TrialTracker{entitlements: entitlements, trials: trials, now: time.Now}
}

// ActivateFirstLoginTrial creates one seven-day trial when no entitlement history exists.
// Implements DESIGN-007 TrialTracker StartTrial and DESIGN-006 OAuthAuthenticator hook.
func (t *TrialTracker) ActivateFirstLoginTrial(ctx context.Context, userID uuid.UUID) error {
	_, err := t.StartTrial(ctx, userID)
	return err
}

// StartTrial creates one active seven-day trial for a newly social-authenticated user.
// Implements DESIGN-007 TrialTracker StartTrial.
func (t *TrialTracker) StartTrial(ctx context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	if userID == uuid.Nil {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindValidation, "user id is required", nil)
	}
	existing, err := t.entitlements.GetLatest(ctx, userID)
	if err == nil {
		return existing, nil
	}
	if !repository.IsKind(err, repository.ErrorKindNotFound) {
		return repository.Entitlement{}, err
	}

	expiresAt := t.now().UTC().Add(7 * 24 * time.Hour)
	trial := repository.Entitlement{
		UserID:            userID,
		Tier:              "trial",
		Status:            "active",
		SearchLimitPer24h: 0,
		AllowedModes:      paidModes(),
		ExpiresAt:         &expiresAt,
	}
	if err := t.entitlements.AppendEntitlement(ctx, trial); err != nil {
		return repository.Entitlement{}, err
	}
	return trial, nil
}

// ExpireTrials appends free active entitlement rows for expired active trials.
// Implements DESIGN-007 TrialTracker ExpireTrials.
func (t *TrialTracker) ExpireTrials(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		return repository.NewError(repository.ErrorKindValidation, "now is required", nil)
	}
	expired, err := t.trials.ListExpiredTrials(ctx, now.UTC())
	if err != nil {
		return err
	}
	for _, trial := range expired {
		if trial.UserID == uuid.Nil {
			return repository.NewError(repository.ErrorKindValidation, "expired trial user id is required", nil)
		}
		latest, err := t.entitlements.GetLatest(ctx, trial.UserID)
		if err != nil {
			return err
		}
		if latest.Tier != "trial" || latest.Status != "active" || latest.ExpiresAt == nil || latest.ExpiresAt.After(now.UTC()) {
			continue
		}
		if err := t.entitlements.AppendEntitlement(ctx, freeActiveEntitlement(trial.UserID)); err != nil {
			return err
		}
	}
	return nil
}

// freeActiveEntitlement maps expired trial access back to the free tier.
// Implements DESIGN-007 TrialTracker.
func freeActiveEntitlement(userID uuid.UUID) repository.Entitlement {
	return repository.Entitlement{
		UserID:            userID,
		Tier:              "free",
		Status:            "active",
		SearchLimitPer24h: 3,
		AllowedModes:      []string{"catalog", "substitution"},
	}
}

// paidModes returns Phase 06-visible paid/trial feature modes.
// Implements DESIGN-007 TrialTracker.
func paidModes() []string {
	return []string{"catalog", "substitution", "daily_diet_alternative"}
}

package entitlement

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TrialTracker creates one-time 7-day trials and manages expiry downgrades.
// Implements DESIGN-007 TrialTracker.
type TrialTracker struct {
	repo   repository.EntitlementRepository
	trials repository.TrialRepository
	now    func() time.Time
}

// NewTrialTracker creates a new TrialTracker.
// Implements DESIGN-007 TrialTracker.
func NewTrialTracker(repo repository.EntitlementRepository, trials repository.TrialRepository, now func() time.Time) *TrialTracker {
	return &TrialTracker{repo: repo, trials: trials, now: now}
}

// ActivateFirstLoginTrial creates a one-time 7-day trial for a new user.
// Implements DESIGN-007 TrialTracker.
func (t *TrialTracker) ActivateFirstLoginTrial(ctx context.Context, userID uuid.UUID) error {
	latest, err := t.repo.GetLatest(ctx, userID)
	if err == nil && latest.Tier != "" {
		return nil
	}
	if err != nil && !repository.IsKind(err, repository.ErrorKindNotFound) {
		return err
	}
	now := t.now()
	expiresAt := now.Add(7 * 24 * time.Hour)
	entitlement := repository.Entitlement{
		UserID:            userID,
		Tier:              "trial",
		Status:            "active",
		SearchLimitPer24h: 0,
		AllowedModes:      []string{"catalog", "substitution", "daily_diet_alternative"},
		ExpiresAt:         &expiresAt,
	}
	return t.repo.AppendEntitlement(ctx, entitlement)
}

// ExpireTrials finds all expired trials and downgrades them to free.
// Implements DESIGN-007 TrialTracker.
func (t *TrialTracker) ExpireTrials(ctx context.Context) error {
	now := t.now()
	expired, err := t.trials.ListExpiredTrials(ctx, now)
	if err != nil {
		return err
	}
	for _, ent := range expired {
		newEnt := repository.Entitlement{
			UserID:               ent.UserID,
			Tier:                 "free",
			Status:               "active",
			SearchLimitPer24h:    3,
			AllowedModes:         []string{"catalog", "substitution"},
			StripeCustomerID:     ent.StripeCustomerID,
			StripeSubscriptionID: ent.StripeSubscriptionID,
		}
		if err := t.repo.AppendEntitlement(ctx, newEnt); err != nil {
			return err
		}
	}
	return nil
}

// Implements DESIGN-007 TrialTracker expiry command verification.
package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type expiryCommandRepository struct {
	latest  map[uuid.UUID]repository.Entitlement
	history map[uuid.UUID][]repository.Entitlement
	expired []repository.Entitlement
}

func (r *expiryCommandRepository) AppendEntitlement(_ context.Context, entitlement repository.Entitlement) error {
	if r.latest == nil {
		r.latest = map[uuid.UUID]repository.Entitlement{}
	}
	if r.history == nil {
		r.history = map[uuid.UUID][]repository.Entitlement{}
	}
	r.latest[entitlement.UserID] = entitlement
	r.history[entitlement.UserID] = append(r.history[entitlement.UserID], entitlement)
	return nil
}

func (r *expiryCommandRepository) GetLatest(_ context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	entitlement, ok := r.latest[userID]
	if !ok {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "missing entitlement", nil)
	}
	return entitlement, nil
}

func (r *expiryCommandRepository) ListExpiredTrials(context.Context, time.Time) ([]repository.Entitlement, error) {
	return r.expired, nil
}

func TestRunExpireTrialsIsIdempotentAndPreservesHistory(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Minute)
	userID := uuid.New()
	trial := repository.Entitlement{
		UserID:            userID,
		Tier:              "trial",
		Status:            "active",
		SearchLimitPer24h: 0,
		AllowedModes:      []string{"catalog", "substitution", "daily_diet_alternative"},
		ExpiresAt:         &expiredAt,
	}
	repo := &expiryCommandRepository{
		latest:  map[uuid.UUID]repository.Entitlement{userID: trial},
		history: map[uuid.UUID][]repository.Entitlement{userID: []repository.Entitlement{trial}},
		expired: []repository.Entitlement{trial},
	}

	if err := runExpireTrials(context.Background(), repo, now); err != nil {
		t.Fatalf("runExpireTrials() first error = %v", err)
	}
	if err := runExpireTrials(context.Background(), repo, now); err != nil {
		t.Fatalf("runExpireTrials() second error = %v", err)
	}

	history := repo.history[userID]
	if len(history) != 2 {
		t.Fatalf("history count = %d, want original trial plus one free downgrade", len(history))
	}
	if history[0].Tier != "trial" || history[1].Tier != "free" || history[1].Status != "active" {
		t.Fatalf("history = %#v, want preserved trial and one free active downgrade", history)
	}
}

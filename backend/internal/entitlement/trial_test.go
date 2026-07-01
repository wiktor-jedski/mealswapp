// Package entitlement_test verifies the TrialTracker behavior.
// Implements DESIGN-007 TrialTracker.
package entitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// mockEntitlementRepo simulates entitlement storage.
// Implements DESIGN-007 TrialTracker.
type mockEntitlementRepo struct {
	ents []repository.Entitlement
}

// AppendEntitlement stores a new entitlement.
// Implements DESIGN-007 TrialTracker.
func (m *mockEntitlementRepo) AppendEntitlement(ctx context.Context, ent repository.Entitlement) error {
	m.ents = append(m.ents, ent)
	return nil
}

// GetLatest returns the most recent entitlement for a user.
// Implements DESIGN-007 TrialTracker.
func (m *mockEntitlementRepo) GetLatest(ctx context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	for i := len(m.ents) - 1; i >= 0; i-- {
		if m.ents[i].UserID == userID {
			return m.ents[i], nil
		}
	}
	return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
}

// ListExpiredTrials returns active trials past their expiration.
// Implements DESIGN-007 TrialTracker.
func (m *mockEntitlementRepo) ListExpiredTrials(ctx context.Context, now time.Time) ([]repository.Entitlement, error) {
	var expired []repository.Entitlement
	latestMap := make(map[uuid.UUID]repository.Entitlement)
	for _, ent := range m.ents {
		latestMap[ent.UserID] = ent
	}
	for _, ent := range latestMap {
		if ent.Tier == "trial" && ent.Status == "active" && ent.ExpiresAt != nil && ent.ExpiresAt.Before(now) {
			expired = append(expired, ent)
		}
	}
	return expired, nil
}

// TestTrialTracker_ActivateFirstLoginTrial verifies first login trial logic.
// Implements DESIGN-007 TrialTracker.
func TestTrialTracker_ActivateFirstLoginTrial(t *testing.T) {
	repo := &mockEntitlementRepo{}
	nowTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tracker := entitlement.NewTrialTracker(repo, repo, func() time.Time { return nowTime })

	userID := uuid.New()

	err := tracker.ActivateFirstLoginTrial(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.ents) != 1 {
		t.Fatalf("expected 1 entitlement, got %d", len(repo.ents))
	}
	ent := repo.ents[0]
	if ent.Tier != "trial" || ent.Status != "active" {
		t.Errorf("expected active trial, got %v %v", ent.Tier, ent.Status)
	}
	if !ent.ExpiresAt.Equal(nowTime.Add(7 * 24 * time.Hour)) {
		t.Errorf("expected expiry in 7 days, got %v", ent.ExpiresAt)
	}

	nowTime2 := nowTime.Add(24 * time.Hour)
	tracker2 := entitlement.NewTrialTracker(repo, repo, func() time.Time { return nowTime2 })
	err = tracker2.ActivateFirstLoginTrial(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error on second login: %v", err)
	}
	if len(repo.ents) != 1 {
		t.Fatalf("expected no new entitlement, still 1, got %d", len(repo.ents))
	}
}

// TestTrialTracker_ExpireTrials verifies trial expiration.
// Implements DESIGN-007 TrialTracker.
func TestTrialTracker_ExpireTrials(t *testing.T) {
	repo := &mockEntitlementRepo{}

	userExpired := uuid.New()
	userActive := uuid.New()
	userPaid := uuid.New()

	nowTime := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	repo.ents = []repository.Entitlement{
		{
			UserID:    userExpired,
			Tier:      "trial",
			Status:    "active",
			ExpiresAt: func(timeTime time.Time) *time.Time { return &timeTime }(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)),
		},
		{
			UserID:    userActive,
			Tier:      "trial",
			Status:    "active",
			ExpiresAt: func(timeTime time.Time) *time.Time { return &timeTime }(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			UserID:    userPaid,
			Tier:      "paid",
			Status:    "active",
			ExpiresAt: nil,
		},
	}

	tracker := entitlement.NewTrialTracker(repo, repo, func() time.Time { return nowTime })
	err := tracker.ExpireTrials(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	latestExpired, _ := repo.GetLatest(context.Background(), userExpired)
	if latestExpired.Tier != "free" {
		t.Errorf("expected userExpired to be downgraded to free, got %v", latestExpired.Tier)
	}

	latestActive, _ := repo.GetLatest(context.Background(), userActive)
	if latestActive.Tier != "trial" {
		t.Errorf("expected userActive to remain trial, got %v", latestActive.Tier)
	}

	latestPaid, _ := repo.GetLatest(context.Background(), userPaid)
	if latestPaid.Tier != "paid" {
		t.Errorf("expected userPaid to remain paid, got %v", latestPaid.Tier)
	}
}

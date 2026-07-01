// Package subscription provides entitlement and billing services.
// Implements DESIGN-007 EntitlementManager.
package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// fakeEntitlementRepository implements repository.EntitlementRepository for tests.
// Implements DESIGN-007 EntitlementManager.
type fakeEntitlementRepository struct {
	entitlement *repository.Entitlement
	err         error
}

// Implements DESIGN-007 EntitlementManager.
func (r *fakeEntitlementRepository) AppendEntitlement(_ context.Context, _ repository.Entitlement) error {
	return errors.New("unimplemented")
}

// Implements DESIGN-007 EntitlementManager.
func (r *fakeEntitlementRepository) GetLatest(_ context.Context, _ uuid.UUID) (repository.Entitlement, error) {
	if r.err != nil {
		return repository.Entitlement{}, r.err
	}
	if r.entitlement != nil {
		return *r.entitlement, nil
	}
	return repository.Entitlement{}, errors.New("not found")
}

// TestCheckEntitlement verifies decision logic.
// Implements DESIGN-007 EntitlementManager.
func TestCheckEntitlement(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name        string
		userID      uuid.UUID
		repoEnt     *repository.Entitlement
		repoErr     error
		feature     string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name:        "missing entitlement falls back to free behavior (allows catalog)",
			userID:      validUserID,
			repoErr:     errors.New("not found"),
			feature:     FeatureCatalog,
			wantAllowed: true,
		},
		{
			name:        "missing entitlement falls back to free behavior (blocks multi)",
			userID:      validUserID,
			repoErr:     errors.New("not found"),
			feature:     FeatureSubstitutionMulti,
			wantAllowed: false,
		},
		{
			name:   "free active user allows single-input substitution",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "free",
				Status: "active",
			},
			feature:     FeatureSubstitutionSingle,
			wantAllowed: true,
		},
		{
			name:   "free active user blocks paid mode",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "free",
				Status: "active",
			},
			feature:     FeatureDailyDiet,
			wantAllowed: false,
		},
		{
			name:   "trial active user allows paid mode",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "trial",
				Status: "active",
			},
			feature:     FeatureDailyDietAlternative,
			wantAllowed: true,
		},
		{
			name:   "paid active user allows paid mode",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "paid",
				Status: "active",
			},
			feature:     FeatureSubstitutionMulti,
			wantAllowed: true,
		},
		{
			name:   "expired user blocks paid mode",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "paid",
				Status: "expired",
			},
			feature:     FeatureSubstitutionMulti,
			wantAllowed: false,
		},
		{
			name:   "past_due user blocks paid mode",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "paid",
				Status: "past_due",
			},
			feature:     FeatureDailyDiet,
			wantAllowed: false,
		},
		{
			name:   "cancelled user blocks paid mode",
			userID: validUserID,
			repoEnt: &repository.Entitlement{
				Tier:   "paid",
				Status: "cancelled",
			},
			feature:     FeatureDailyDietAlternative,
			wantAllowed: false,
		},
		{
			name:        "nil user ID is rejected",
			userID:      uuid.Nil,
			feature:     FeatureCatalog,
			wantAllowed: false,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeEntitlementRepository{
				entitlement: tc.repoEnt,
				err:         tc.repoErr,
			}
			manager := NewEntitlementManager(repo)

			decision, err := manager.CheckEntitlement(context.Background(), tc.userID, tc.feature)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if decision.Allowed != tc.wantAllowed {
				t.Errorf("expected allowed %v, got %v", tc.wantAllowed, decision.Allowed)
			}
		})
	}
}

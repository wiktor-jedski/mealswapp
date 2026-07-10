// Implements DESIGN-007 EntitlementManager verification.
package entitlement

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// entitlementRepositoryStub supports DESIGN-007 EntitlementManager service verification.
type entitlementRepositoryStub struct {
	entitlements map[uuid.UUID]repository.Entitlement
	err          error
	lookups      []uuid.UUID
}

func (r *entitlementRepositoryStub) AppendEntitlement(context.Context, repository.Entitlement) error {
	return nil
}

func (r *entitlementRepositoryStub) GetLatest(_ context.Context, userID uuid.UUID) (repository.Entitlement, error) {
	r.lookups = append(r.lookups, userID)
	if r.err != nil {
		return repository.Entitlement{}, r.err
	}
	entitlement, ok := r.entitlements[userID]
	if !ok {
		return repository.Entitlement{}, repository.NewError(repository.ErrorKindNotFound, "missing entitlement", nil)
	}
	return entitlement, nil
}

func TestEntitlementManagerAllowsFreeScopeOnlyForFreeActiveUsers(t *testing.T) {
	userID := uuid.New()
	manager := NewEntitlementManager(&entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{
		userID: freeEntitlement(userID),
	}})

	cases := map[Feature]bool{
		FeatureCatalog:              true,
		FeatureSingleSubstitution:   true,
		FeatureMultiSubstitution:    false,
		FeatureDailyDiet:            false,
		FeatureDailyDietAlternative: false,
	}
	for feature, wantAllowed := range cases {
		decision, err := manager.CheckEntitlement(context.Background(), userID, feature)
		if err != nil {
			t.Fatalf("CheckEntitlement(%s) error = %v", feature, err)
		}
		if decision.Allowed != wantAllowed {
			t.Fatalf("CheckEntitlement(%s) allowed = %v, want %v", feature, decision.Allowed, wantAllowed)
		}
		if !wantAllowed && decision.DenyReason != DenyReasonFreeTierScope {
			t.Fatalf("CheckEntitlement(%s) deny reason = %q, want %q", feature, decision.DenyReason, DenyReasonFreeTierScope)
		}
	}
}

func TestEntitlementManagerAllowsAllPhase06PaidModesForActiveTrialAndPaidUsers(t *testing.T) {
	for _, tier := range []string{"trial", "paid"} {
		t.Run(tier, func(t *testing.T) {
			userID := uuid.New()
			manager := NewEntitlementManager(&entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{
				userID: paidEntitlement(userID, tier, "active"),
			}})

			for _, feature := range allFeatures() {
				decision, err := manager.CheckEntitlement(context.Background(), userID, feature)
				if err != nil {
					t.Fatalf("CheckEntitlement(%s) error = %v", feature, err)
				}
				if !decision.Allowed {
					t.Fatalf("CheckEntitlement(%s) allowed = false for %s active", feature, tier)
				}
			}
		})
	}
}

func TestEntitlementManagerBlocksPaidOnlyModesForInactiveStates(t *testing.T) {
	for _, status := range []string{"expired", "past_due", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			userID := uuid.New()
			manager := NewEntitlementManager(&entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{
				userID: paidEntitlement(userID, "paid", status),
			}})

			freeDecision, err := manager.CheckEntitlement(context.Background(), userID, FeatureSingleSubstitution)
			if err != nil || !freeDecision.Allowed {
				t.Fatalf("free-scope decision = %+v err=%v, want allowed", freeDecision, err)
			}

			for _, feature := range []Feature{FeatureMultiSubstitution, FeatureDailyDiet, FeatureDailyDietAlternative} {
				decision, err := manager.CheckEntitlement(context.Background(), userID, feature)
				if err != nil {
					t.Fatalf("CheckEntitlement(%s) error = %v", feature, err)
				}
				if decision.Allowed || decision.DenyReason != DenyReasonInactivePaidState {
					t.Fatalf("CheckEntitlement(%s) = %+v, want inactive paid block", feature, decision)
				}
			}
		})
	}
}

func TestEntitlementManagerMissingEntitlementFallsBackToFreeBehavior(t *testing.T) {
	userID := uuid.New()
	manager := NewEntitlementManager(&entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{}})

	catalog, err := manager.CheckEntitlement(context.Background(), userID, FeatureCatalog)
	if err != nil || !catalog.Allowed || catalog.Tier != "free" || catalog.Status != "active" {
		t.Fatalf("catalog fallback = %+v err=%v, want free active allow", catalog, err)
	}
	dailyDiet, err := manager.CheckEntitlement(context.Background(), userID, FeatureDailyDiet)
	if err != nil || dailyDiet.Allowed || dailyDiet.DenyReason != DenyReasonFreeTierScope {
		t.Fatalf("daily diet fallback = %+v err=%v, want free-scope block", dailyDiet, err)
	}
}

func TestEntitlementManagerUsesAuthenticatedUserIDAndIgnoresClientSuppliedUserID(t *testing.T) {
	authenticatedUserID := uuid.New()
	clientUserID := uuid.New()
	repo := &entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{
		authenticatedUserID: freeEntitlement(authenticatedUserID),
		clientUserID:        paidEntitlement(clientUserID, "paid", "active"),
	}}
	manager := NewEntitlementManager(repo)

	decision, err := manager.Decide(context.Background(), DecisionRequest{
		AuthenticatedUserID:  authenticatedUserID,
		ClientSuppliedUserID: &clientUserID,
		Feature:              FeatureDailyDiet,
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.Allowed {
		t.Fatalf("Decide() allowed with client user id entitlement: %+v", decision)
	}
	if len(repo.lookups) != 1 || repo.lookups[0] != authenticatedUserID {
		t.Fatalf("repository lookups = %v, want only authenticated user %s", repo.lookups, authenticatedUserID)
	}
}

func TestEntitlementManagerValidationAndRepositoryErrors(t *testing.T) {
	manager := NewEntitlementManager(&entitlementRepositoryStub{})
	if _, err := manager.CheckEntitlement(context.Background(), uuid.Nil, FeatureCatalog); !IsEntitlementValidationError(err) {
		t.Fatalf("nil user error = %v, want validation", err)
	}

	userID := uuid.New()
	decision, err := manager.CheckEntitlement(context.Background(), userID, Feature("unknown"))
	if err != nil || decision.Allowed || decision.DenyReason != DenyReasonInvalidFeature {
		t.Fatalf("invalid feature decision = %+v err=%v, want invalid-feature block", decision, err)
	}

	wantErr := errors.New("database unavailable")
	manager = NewEntitlementManager(&entitlementRepositoryStub{err: wantErr})
	if _, err := manager.CheckEntitlement(context.Background(), userID, FeatureCatalog); !errors.Is(err, wantErr) {
		t.Fatalf("repository error = %v, want %v", err, wantErr)
	}
}

func TestEntitlementDecisionMappingCoversInvalidAndInactiveFreeStates(t *testing.T) {
	userID := uuid.New()
	invalid := decideFromEntitlement(freeEntitlement(userID), Feature("unknown"))
	if invalid.Allowed || invalid.DenyReason != DenyReasonInvalidFeature {
		t.Fatalf("invalid mapped decision = %+v, want invalid feature block", invalid)
	}

	inactiveFree := freeEntitlement(userID)
	inactiveFree.Status = "expired"
	decision := decideFromEntitlement(inactiveFree, FeatureDailyDiet)
	if decision.Allowed || decision.DenyReason != DenyReasonInactivePaidState {
		t.Fatalf("inactive free paid-only decision = %+v, want inactive state block", decision)
	}
}

func freeEntitlement(userID uuid.UUID) repository.Entitlement {
	return repository.Entitlement{
		UserID:            userID,
		Tier:              "free",
		Status:            "active",
		SearchLimitPer24h: 3,
		AllowedModes:      []string{"catalog", "substitution"},
	}
}

func paidEntitlement(userID uuid.UUID, tier string, status string) repository.Entitlement {
	return repository.Entitlement{
		UserID:            userID,
		Tier:              tier,
		Status:            status,
		SearchLimitPer24h: 0,
		AllowedModes:      []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"},
	}
}

func allFeatures() []Feature {
	return []Feature{
		FeatureCatalog,
		FeatureSingleSubstitution,
		FeatureMultiSubstitution,
		FeatureDailyDiet,
		FeatureDailyDietAlternative,
	}
}

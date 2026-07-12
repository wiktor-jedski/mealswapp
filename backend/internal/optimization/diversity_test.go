// Implements DESIGN-004 DiversityPenalizer verification for SW-REQ-023 and
// SW-REQ-030.
package optimization

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

var (
	diversityMealA = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	diversityMealB = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	diversityMealC = uuid.MustParse("00000000-0000-0000-0000-000000000023")
	diversityMealD = uuid.MustParse("00000000-0000-0000-0000-000000000024")
)

func TestBuildConstraintsPenalizesOriginalMealsWithoutForbiddingThem(t *testing.T) {
	model, err := BuildConstraints(DietOptimizationRequest{
		OriginalMeals:    []MealQuantity{{MealID: diversityMealA, Quantity: 100, Unit: "g"}},
		TolerancePercent: 0,
		MaxQuantity:      100,
	}, diversityMeals(diversityMealA, diversityMealB))
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}

	original := findVariable(t, model, diversityMealA.String())
	if original.UpperBound == 0 {
		t.Fatal("original meal was absolutely forbidden instead of softly penalized")
	}
	if original.DiversityPenalty != DefaultDiversityPenalty {
		t.Fatalf("original meal penalty = %v, want %v", original.DiversityPenalty, DefaultDiversityPenalty)
	}
	other := findVariable(t, model, diversityMealB.String())
	if other.DiversityPenalty != 0 {
		t.Fatalf("non-original meal penalty = %v, want 0", other.DiversityPenalty)
	}

	objective, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}
	if objective.Coefficients[original.ItemID] <= objective.Coefficients[other.ItemID] {
		t.Fatalf("original objective coefficient = %v, other = %v; want original higher", objective.Coefficients[original.ItemID], objective.Coefficients[other.ItemID])
	}
	if got := objective.DiversityPenalties[original.ItemID]; got != DefaultDiversityPenalty {
		t.Fatalf("objective diversity penalty = %v, want %v", got, DefaultDiversityPenalty)
	}
}

func TestGenerateAlternativesReturnsDeterministicOneOrTwoResults(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		selectedIDs []uuid.UUID
	}{
		{name: "one alternative", limit: 1, selectedIDs: []uuid.UUID{diversityMealA}},
		{name: "two alternatives", limit: 2, selectedIDs: []uuid.UUID{diversityMealA, diversityMealB}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := 0
			solve := func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
				if call >= len(tt.selectedIDs) {
					t.Fatalf("solver called %d times, want at most %d", call+1, len(tt.selectedIDs))
				}
				mealID := tt.selectedIDs[call].String()
				call++
				return LPSolution{mealID: 100}, nil
			}
			results, err := GenerateAlternatives(context.Background(), DietOptimizationRequest{
				OriginalMeals:    []MealQuantity{{MealID: diversityMealA, Quantity: 100, Unit: "g"}},
				TolerancePercent: 0,
				MaxQuantity:      100,
			}, diversityMeals(diversityMealA, diversityMealB, diversityMealC), tt.limit, solve)
			if err != nil {
				t.Fatalf("GenerateAlternatives() error = %v", err)
			}
			if got := len(results); got != tt.limit {
				t.Fatalf("alternative count = %d, want %d", got, tt.limit)
			}
			if got := call; got != tt.limit {
				t.Fatalf("solver calls = %d, want %d", got, tt.limit)
			}
			for index, result := range results {
				want := LPSolution{tt.selectedIDs[index].String(): 100}
				if !reflect.DeepEqual(result, want) {
					t.Fatalf("alternative %d = %#v, want %#v", index+1, result, want)
				}
			}
		})
	}
}

func TestGenerateAlternativesDeduplicatesAndCapsResults(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB, diversityMealC, diversityMealD)
	req := DietOptimizationRequest{
		OriginalMeals:    []MealQuantity{{MealID: diversityMealA, Quantity: 100, Unit: "g"}},
		TolerancePercent: 0,
		MaxQuantity:      100,
	}
	call := 0
	solve := func(_ context.Context, model LPModel, objective ObjectiveFunction) (LPSolution, error) {
		call++
		if got := objective.DiversityPenalties[diversityMealA.String()]; got != DefaultDiversityPenalty {
			t.Fatalf("solve %d original penalty = %v, want %v", call, got, DefaultDiversityPenalty)
		}
		if call > 1 {
			constraint := findConstraint(t, model, "alternative_1")
			if got := constraintValue(constraint, map[string]float64{diversityMealA.String(): 100}); got <= constraint.UpperBound {
				t.Fatalf("solve %d did not constrain repeated high-weight selection: value %v <= %v", call, got, constraint.UpperBound)
			}
		}
		if call > 2 {
			constraint := findConstraint(t, model, "alternative_2")
			if got := constraintValue(constraint, map[string]float64{diversityMealB.String(): 100}); got > constraint.UpperBound {
				t.Fatalf("solve %d unexpectedly rejects a new meal set: value %v > %v", call, got, constraint.UpperBound)
			}
		}
		switch call {
		case 1, 2:
			return LPSolution{diversityMealA.String(): 100}, nil
		case 3:
			return LPSolution{diversityMealB.String(): 100}, nil
		default:
			return LPSolution{diversityMealC.String(): 100}, nil
		}
	}

	results, err := GenerateAlternatives(context.Background(), req, meals, 10, solve)
	if err != nil {
		t.Fatalf("GenerateAlternatives() error = %v", err)
	}
	if got, want := call, 4; got != want {
		t.Fatalf("solver calls = %d, want duplicate retry then three accepted results (%d)", got, want)
	}
	if got, want := len(results), 3; got != want {
		t.Fatalf("alternative count = %d, want %d", got, want)
	}
	if !reflect.DeepEqual(results, []LPSolution{
		{diversityMealA.String(): 100},
		{diversityMealB.String(): 100},
		{diversityMealC.String(): 100},
	}) {
		t.Fatalf("alternatives = %#v, want deterministic distinct meal sets", results)
	}
}

func TestGenerateAlternativesRejectsSolverOutputThatViolatesHardExclusion(t *testing.T) {
	req := DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1},
		TolerancePercent: 0,
		MaxQuantity:      100,
		ExcludedMealIDs:  []uuid.UUID{diversityMealB},
	}
	results, err := GenerateAlternatives(context.Background(), req, diversityMeals(diversityMealA, diversityMealB), 1, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		return LPSolution{diversityMealB.String(): 100}, nil
	})
	if err == nil {
		t.Fatal("GenerateAlternatives() accepted an excluded meal")
	}
	if len(results) != 0 {
		t.Fatalf("partial alternatives = %v, want none", results)
	}
}

func diversityMeals(ids ...uuid.UUID) []repository.MealEntity {
	meals := make([]repository.MealEntity, len(ids))
	for index, id := range ids {
		meals[index] = repository.MealEntity{
			ID:           id,
			MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1},
		}
	}
	return meals
}

func findVariable(t *testing.T, model LPModel, itemID string) LPVariable {
	t.Helper()
	for _, variable := range model.Variables {
		if variable.ItemID == itemID {
			return variable
		}
	}
	t.Fatalf("variable %q not found in %+v", itemID, model.Variables)
	return LPVariable{}
}

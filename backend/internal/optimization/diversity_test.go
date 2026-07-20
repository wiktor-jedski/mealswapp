// Implements DESIGN-004 DiversityPenalizer verification for SW-REQ-023 and
// SW-REQ-030.
package optimization

import (
	"context"
	"math"
	"os"
	"os/exec"
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
	meals := diversityMeals(diversityMealA, diversityMealB)
	model, err := BuildConstraints(diversityRequest(meals[0], nil), meals, nil)
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

	policy, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}
	if policy.Primary.Coefficients[original.ItemID] != policy.Primary.Coefficients[other.ItemID] {
		t.Fatalf("primary calories differ: original %v, other %v", policy.Primary.Coefficients[original.ItemID], policy.Primary.Coefficients[other.ItemID])
	}
	if got := policy.Secondary.Coefficients[original.ItemID]; got != DefaultDiversityPenalty {
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
				attempt := call / 2
				if attempt >= len(tt.selectedIDs) {
					t.Fatalf("solver called %d times, want at most %d", call+1, len(tt.selectedIDs)*2)
				}
				mealID := tt.selectedIDs[attempt].String()
				call++
				return LPSolution{mealID: 100}, nil
			}
			meals := diversityMeals(diversityMealA, diversityMealB, diversityMealC)
			results, err := GenerateAlternatives(context.Background(), diversityRequest(meals[0], nil), meals, tt.limit, solve)
			if err != nil {
				t.Fatalf("GenerateAlternatives() error = %v", err)
			}
			if got := len(results); got != tt.limit {
				t.Fatalf("alternative count = %d, want %d", got, tt.limit)
			}
			if got := call; got != tt.limit*2 {
				t.Fatalf("solver calls = %d, want %d lexicographic passes", got, tt.limit*2)
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

func TestGenerateAlternativesUsesLexicographicPassesAndCapsResults(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB, diversityMealC, diversityMealD)
	req := diversityRequest(meals[0], nil)
	call := 0
	solve := func(_ context.Context, model LPModel, objective ObjectiveFunction) (LPSolution, error) {
		call++
		attempt := (call - 1) / 2
		secondary := call%2 == 0
		if secondary {
			if got := objective.Coefficients[diversityMealA.String()]; got != DefaultDiversityPenalty {
				t.Fatalf("solve %d original diversity coefficient = %v, want %v", call, got, DefaultDiversityPenalty)
			}
			_ = findConstraint(t, model, "primary_calorie_optimum")
		} else if got := objective.Coefficients[diversityMealA.String()]; got == DefaultDiversityPenalty {
			t.Fatalf("solve %d used diversity as the primary objective", call)
		}
		if attempt > 0 {
			constraint := findConstraint(t, model, "alternative_1")
			if got := constraintValue(constraint, map[string]float64{diversityMealA.String(): 100}); got <= constraint.UpperBound {
				t.Fatalf("solve %d did not constrain repeated high-weight selection: value %v <= %v", call, got, constraint.UpperBound)
			}
		}
		if attempt > 1 {
			constraint := findConstraint(t, model, "alternative_2")
			if got := constraintValue(constraint, map[string]float64{diversityMealB.String(): 100}); got <= constraint.UpperBound {
				t.Fatalf("solve %d did not constrain the second accepted meal: value %v <= %v", call, got, constraint.UpperBound)
			}
		}
		switch attempt {
		case 0:
			return LPSolution{diversityMealA.String(): 100}, nil
		case 1:
			return LPSolution{diversityMealB.String(): 100}, nil
		default:
			return LPSolution{diversityMealC.String(): 100}, nil
		}
	}

	results, err := GenerateAlternatives(context.Background(), req, meals, 10, solve)
	if err != nil {
		t.Fatalf("GenerateAlternatives() error = %v", err)
	}
	if got, want := call, 6; got != want {
		t.Fatalf("solver calls = %d, want two passes for three results (%d)", got, want)
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
	meals := diversityMeals(diversityMealA, diversityMealB)
	req := diversityRequest(meals[0], []uuid.UUID{diversityMealB})
	results, err := GenerateAlternatives(context.Background(), req, meals, 1, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		return LPSolution{diversityMealB.String(): 100}, nil
	})
	if err == nil {
		t.Fatal("GenerateAlternatives() accepted an excluded meal")
	}
	if len(results) != 0 {
		t.Fatalf("partial alternatives = %v, want none", results)
	}
}

func TestSolveObjectivePolicyRejectsDiversityThatOverturnsCalorieOrdering(t *testing.T) {
	model := lexicographicFixtureModel(1.000001)
	policy, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}
	call := 0
	_, err = solveObjectivePolicy(context.Background(), model, policy, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		call++
		if call == 1 {
			return LPSolution{diversityMealA.String(): 1}, nil
		}
		return LPSolution{diversityMealB.String(): 1}, nil
	})
	if err == nil {
		t.Fatal("lexicographic policy accepted a more diverse but higher-calorie secondary solution")
	}
}

func TestTask219PackagedCLPLexicographicObjective(t *testing.T) {
	executable := os.Getenv("MEALSWAPP_CLP_PATH")
	if executable == "" {
		executable, _ = exec.LookPath(DefaultCLPExecutable)
	}
	if executable == "" {
		t.Skip("native CLP executable is not installed; packaged worker CI supplies it")
	}
	solver := NewLPSolverWrapper(CLPConfig{Executable: executable})
	for _, tt := range []struct {
		name            string
		diverseCalories float64
		want            uuid.UUID
	}{
		{name: "calories remain primary", diverseCalories: 1.000001, want: diversityMealA},
		{name: "diversity breaks calorie tie", diverseCalories: 1, want: diversityMealB},
	} {
		t.Run(tt.name, func(t *testing.T) {
			model := lexicographicFixtureModel(tt.diverseCalories)
			policy, err := BuildObjective(model.Variables)
			if err != nil {
				t.Fatalf("BuildObjective() error = %v", err)
			}
			solution, err := solveObjectivePolicy(context.Background(), model, policy, solver.Solve)
			if err != nil {
				t.Fatalf("packaged CLP lexicographic solve: %v", err)
			}
			if math.Abs(solution[tt.want.String()]-1) > quantityTolerance(1) || len(solution) != 1 {
				t.Fatalf("solution = %#v, want only %s", solution, tt.want)
			}
		})
	}
}

// Implements DESIGN-004 SolutionValidator CLP text-precision boundary verification.
func TestSolutionSatisfiesModelAcceptsRoundedCLPBoundaryQuantity(t *testing.T) {
	mealID := uuid.MustParse("00000000-0000-4000-8000-000000000099")
	model := LPModel{
		Variables: []LPVariable{{ItemID: mealID.String(), LowerBound: 0, UpperBound: MaximumMealQuantity}},
		Constraints: []LPConstraint{{
			Name: "carbohydrate", LowerBound: 144, UpperBound: 176,
			Coefficients: map[string]float64{mealID.String(): 1},
		}},
	}
	solution := LPSolution{mealID.String(): 143.999997}
	if err := solutionSatisfiesModel(solution, model); err != nil {
		t.Fatalf("rounded CLP boundary solution rejected: %v", err)
	}
}

func lexicographicFixtureModel(diverseCalories float64) LPModel {
	return LPModel{
		Variables: []LPVariable{
			{ItemID: diversityMealA.String(), LowerBound: 0, UpperBound: 1, CaloriesPerUnit: 1, DiversityPenalty: 1},
			{ItemID: diversityMealB.String(), LowerBound: 0, UpperBound: 1, CaloriesPerUnit: diverseCalories},
		},
		Constraints: []LPConstraint{{Name: "quantity", LowerBound: 1, UpperBound: 1, Coefficients: map[string]float64{diversityMealA.String(): 1, diversityMealB.String(): 1}}},
	}
}

func diversityMeals(ids ...uuid.UUID) []repository.MealEntity {
	meals := make([]repository.MealEntity, len(ids))
	for index, id := range ids {
		meals[index] = repository.MealEntity{
			ID: id, Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100:              repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1},
			NormalizedMacrosAvailable: true,
		}
	}
	return meals
}

func diversityRequest(original repository.MealEntity, excluded []uuid.UUID) DietOptimizationRequest {
	return DietOptimizationRequest{
		OriginalDiet: repository.SavedDiet{
			ID:      uuid.MustParse("00000000-0000-4000-8000-000000000020"),
			UserID:  uuid.MustParse("00000000-0000-4000-8000-000000000025"),
			Entries: []repository.SavedDietMealEntry{{MealID: original.ID, Quantity: 100, Unit: "g"}},
		},
		ExcludedMealIDs: excluded,
	}
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

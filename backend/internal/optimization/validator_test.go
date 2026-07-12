// Implements DESIGN-004 SolutionValidator verification for SW-REQ-021,
// SW-REQ-022, and SW-REQ-030.
package optimization

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

var (
	validatorMealA = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	validatorMealB = uuid.MustParse("00000000-0000-0000-0000-000000000032")
	validatorMealC = uuid.MustParse("00000000-0000-0000-0000-000000000033")
)

func TestValidateSolutionRecomputesEveryAcceptedAlternativeFromRepositoryMeals(t *testing.T) {
	meals := []repository.MealEntity{
		{ID: validatorMealB, PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 4, Carbohydrates: 8, Fat: 1}},
		{ID: validatorMealA, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 10, Fat: 2}},
	}
	req := DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 12, Carbohydrates: 9, Fat: 1.5},
		TolerancePercent: 0,
		MaxQuantity:      100,
	}

	alternative, err := ValidateSolution(LPSolution{
		validatorMealB.String(): 50,
		validatorMealA.String(): 50,
	}, req, meals)
	if err != nil {
		t.Fatalf("ValidateSolution() error = %v", err)
	}
	if got, want := alternative.Macros.Protein, 12.0; got != want {
		t.Fatalf("protein = %v, want %v", got, want)
	}
	if got, want := alternative.Macros.Carbohydrates, 9.0; got != want {
		t.Fatalf("carbohydrates = %v, want %v", got, want)
	}
	if got, want := alternative.Macros.Fat, 1.5; got != want {
		t.Fatalf("fat = %v, want %v", got, want)
	}
	if got, want := alternative.Calories, 97.5; got != want {
		t.Fatalf("calories = %v, want %v", got, want)
	}
	if len(alternative.Meals) != 2 || alternative.Meals[0].MealID != validatorMealA || alternative.Meals[1].MealID != validatorMealB {
		t.Fatalf("meals = %+v, want deterministic repository order", alternative.Meals)
	}
	if alternative.Meals[0].Unit != "g" || alternative.Meals[1].Unit != "ml" {
		t.Fatalf("meal units = %+v, want g then ml", alternative.Meals)
	}
}

func TestValidateSolutionAcceptsToleranceBoundariesAndFloatingPointEpsilon(t *testing.T) {
	meal := repository.MealEntity{ID: validatorMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}}
	baseRequest := DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5},
		TolerancePercent: 10,
		MaxQuantity:      200,
	}
	tests := []struct {
		name     string
		quantity float64
		wantErr  bool
	}{
		{name: "lower boundary", quantity: 90, wantErr: false},
		{name: "upper boundary", quantity: 110, wantErr: false},
		{name: "rounding epsilon inside upper boundary", quantity: 110 + 5e-10, wantErr: false},
		{name: "materially outside upper boundary", quantity: 110.001, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateSolution(LPSolution{validatorMealA.String(): tt.quantity}, baseRequest, []repository.MealEntity{meal})
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSolution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSolutionRejectsMalformedQuantitiesIDsAndAlternatives(t *testing.T) {
	meal := repository.MealEntity{ID: validatorMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1}}
	req := DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1},
		TolerancePercent: 0,
		MaxQuantity:      100,
		ExcludedMealIDs:  []uuid.UUID{validatorMealB},
	}
	tests := []struct {
		name     string
		solution LPSolution
	}{
		{name: "empty alternative", solution: nil},
		{name: "zero alternative", solution: LPSolution{validatorMealA.String(): 0}},
		{name: "negative quantity", solution: LPSolution{validatorMealA.String(): -1}},
		{name: "nan quantity", solution: LPSolution{validatorMealA.String(): math.NaN()}},
		{name: "infinite quantity", solution: LPSolution{validatorMealA.String(): math.Inf(1)}},
		{name: "unknown meal", solution: LPSolution{validatorMealC.String(): 100}},
		{name: "excluded meal", solution: LPSolution{validatorMealB.String(): 100}},
		{name: "excluded zero meal", solution: LPSolution{validatorMealA.String(): 100, validatorMealB.String(): 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateSolution(tt.solution, req, []repository.MealEntity{meal})
			if err == nil || FailureCodeOf(err) != FailureCodeValidation {
				t.Fatalf("ValidateSolution() error = %v, code = %q, want failed_validation", err, FailureCodeOf(err))
			}
			if strings.Contains(err.Error(), validatorMealC.String()) || strings.Contains(err.Error(), "CLP") {
				t.Fatalf("error leaked internal validation details: %q", err.Error())
			}
		})
	}
}

func TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails(t *testing.T) {
	meal := repository.MealEntity{ID: validatorMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1}}
	call := 0
	results, err := GenerateValidatedAlternatives(context.Background(), DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1},
		TolerancePercent: 0,
		MaxQuantity:      100,
	}, []repository.MealEntity{meal}, 2, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		call++
		if call == 1 {
			return LPSolution{validatorMealA.String(): 100}, nil
		}
		return nil, errors.New("private solver diagnostic must not escape")
	})
	if len(results) != 1 {
		t.Fatalf("partial alternatives = %d, want 1", len(results))
	}
	if err == nil || FailureCodeOf(err) != FailureCodeWorkerCrash {
		t.Fatalf("error = %v, code = %q, want worker_crash", err, FailureCodeOf(err))
	}
	if got := err.Error(); got != string(FailureCodeWorkerCrash) {
		t.Fatalf("error = %q, want only safe failure code", got)
	}
	if got := results[0].Calories; got != 89 {
		t.Fatalf("partial result calories = %v, want 89", got)
	}
}

func TestSafeOptimizationFailureMapsSolverErrorsToUserSafeCodes(t *testing.T) {
	tests := []struct {
		name string
		kind SolverErrorKind
		want OptimizationFailureCode
	}{
		{name: "timeout", kind: SolverErrorTimeout, want: FailureCodeSolverTimeout},
		{name: "infeasible", kind: SolverErrorInfeasible, want: FailureCodeSolverInfeasible},
		{name: "malformed", kind: SolverErrorMalformed, want: FailureCodeWorkerCrash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := safeOptimizationFailure(&SolverError{Kind: tt.kind, Diagnostic: "secret process output"})
			if FailureCodeOf(err) != tt.want || err.Error() != string(tt.want) {
				t.Fatalf("safe error = %q (%q), want code %q", err, FailureCodeOf(err), tt.want)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("safe error leaked solver diagnostic")
			}
		})
	}
}

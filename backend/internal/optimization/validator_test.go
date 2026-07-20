// Implements DESIGN-004 SolutionValidator verification for SW-REQ-021,
// SW-REQ-022, and SW-REQ-030.
package optimization

import (
	"context"
	"encoding/json"
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
		validatorMeal(validatorMealB, repository.PhysicalStateLiquid, MacroTarget{Protein: 4, Carbohydrates: 8, Fat: 1}),
		validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 20, Carbohydrates: 10, Fat: 2}),
	}
	req := validatorRequest([]repository.SavedDietMealEntry{
		{MealID: validatorMealB, Quantity: 50, Unit: "ml"},
		{MealID: validatorMealA, Quantity: 50, Unit: "g"},
	}, 0, nil)

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

// Implements DESIGN-004 SolutionValidator public quantity precision verification.
func TestValidateSolutionQuantizesPublishedQuantitiesAndRecalculatesMacros(t *testing.T) {
	meal := validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5})
	req := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 10, nil)
	alternative, err := ValidateSolution(LPSolution{validatorMealA.String(): 90.0004}, req, []repository.MealEntity{meal})
	if err != nil {
		t.Fatalf("ValidateSolution() error = %v", err)
	}
	if alternative.Meals[0].Quantity != 90 {
		t.Fatalf("published quantity = %v, want 90", alternative.Meals[0].Quantity)
	}
	if alternative.Macros != (MacroTarget{Protein: 9, Carbohydrates: 18, Fat: 4.5}) || alternative.Calories != 148.5 {
		t.Fatalf("recalculated projection = macros %+v calories %v", alternative.Macros, alternative.Calories)
	}
}

// Implements DESIGN-003 CosineSimilarityCalculator and DESIGN-004 SolutionValidator.
func TestValidateSolutionCalculatesBoundedMacroSimilarity(t *testing.T) {
	meals := []repository.MealEntity{
		validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1}),
		validatorMeal(validatorMealB, repository.PhysicalStateSolid, MacroTarget{Protein: 1, Carbohydrates: 10, Fat: 10}),
	}
	req := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 100, nil)
	tests := []struct {
		name     string
		solution LPSolution
		want     float64
	}{
		{name: "identical quantities", solution: LPSolution{validatorMealA.String(): 100}, want: 1},
		{name: "mixed macro profile rounded", solution: LPSolution{validatorMealA.String(): 50, validatorMealB.String(): 10}, want: 0.9899},
		{name: "disjoint meal set still compares macros", solution: LPSolution{validatorMealB.String(): 20}, want: 0.597},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alternative, err := ValidateSolution(tt.solution, req, meals)
			if err != nil {
				t.Fatalf("ValidateSolution() error = %v", err)
			}
			if alternative.SimilarityScore != tt.want {
				t.Fatalf("similarityScore = %v, want %v", alternative.SimilarityScore, tt.want)
			}
		})
	}
}

// Implements DESIGN-004 SolutionValidator authoritative publication boundary.
func TestValidateDietAlternativeRejectsMalformedResultShape(t *testing.T) {
	valid := DietAlternative{
		Meals:           []MealQuantity{{MealID: validatorMealA, Name: "Meal A", Quantity: 100, Unit: "g", Position: 0}},
		Macros:          MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10},
		Calories:        290,
		SimilarityScore: 0.1234,
	}
	tests := []struct {
		name   string
		mutate func(*DietAlternative)
	}{
		{name: "nonfinite macro", mutate: func(alternative *DietAlternative) { alternative.Macros.Protein = math.Inf(1) }},
		{name: "negative calories", mutate: func(alternative *DietAlternative) { alternative.Calories = -1 }},
		{name: "invalid quantity", mutate: func(alternative *DietAlternative) { alternative.Meals[0].Quantity = 0 }},
		{name: "missing name", mutate: func(alternative *DietAlternative) { alternative.Meals[0].Name = "" }},
		{name: "unquantized quantity", mutate: func(alternative *DietAlternative) { alternative.Meals[0].Quantity = 100.0004 }},
		{name: "invalid position", mutate: func(alternative *DietAlternative) { alternative.Meals[0].Position = 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alternative := valid
			alternative.Meals = append([]MealQuantity(nil), valid.Meals...)
			tt.mutate(&alternative)
			if err := ValidateDietAlternative(alternative); err == nil {
				t.Fatalf("ValidateDietAlternative(%+v) succeeded", alternative)
			}
		})
	}
	if err := ValidateDietAlternative(valid); err != nil {
		t.Fatalf("ValidateDietAlternative(valid) error = %v", err)
	}
}

// Implements DESIGN-004 SolutionValidator authoritative result publication.
func TestValidateDietAlternativeRejectsInvalidSimilarityScores(t *testing.T) {
	valid := DietAlternative{
		Meals:  []MealQuantity{{MealID: validatorMealA, Name: "Meal A", Quantity: 100, Unit: "g", Position: 0}},
		Macros: MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5}, Calories: 165, SimilarityScore: 0.1234,
	}
	if err := ValidateDietAlternative(valid); err != nil {
		t.Fatalf("ValidateDietAlternative(valid) error = %v", err)
	}
	for _, score := range []float64{-0.0001, 1.0001, 0.12345, math.NaN(), math.Inf(1)} {
		alternative := valid
		alternative.SimilarityScore = score
		if err := ValidateDietAlternative(alternative); err == nil {
			t.Fatalf("ValidateDietAlternative(similarityScore=%v) succeeded", score)
		}
	}
}

// Implements DESIGN-004 SolutionValidator typed-nil error boundary.
func TestOptimizationFailureClassificationHandlesTypedNilErrors(t *testing.T) {
	var failure *OptimizationFailure
	var failureErr error = failure
	if code := FailureCodeOf(failureErr); code.Valid() {
		t.Fatalf("FailureCodeOf(typed nil) = %q, want invalid code", code)
	}
	if err := safeOptimizationFailure(failureErr); FailureCodeOf(err) != FailureCodeWorkerCrash {
		t.Fatalf("safeOptimizationFailure(typed nil) = %v, code %q, want worker_crash", err, FailureCodeOf(err))
	}

	meal := validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1})
	req := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 0, nil)
	var solverErr *SolverError
	solveCalls := 0
	results, err := GenerateValidatedAlternatives(context.Background(), req, []repository.MealEntity{meal}, 1, func(context.Context, LPModel, ObjectiveFunction) (LPSolution, error) {
		solveCalls++
		return nil, solverErr
	})
	if solveCalls != 1 || len(results) != 0 || FailureCodeOf(err) != FailureCodeWorkerCrash {
		t.Fatalf("typed-nil solve calls=%d results=%v error=%v code=%q, want one call and retryable worker_crash classification", solveCalls, results, err, FailureCodeOf(err))
	}
}

// Implements DESIGN-004 SolutionValidator bounded failure classification.
func TestSafeOptimizationFailureNormalizesInvalidExistingFailure(t *testing.T) {
	err := safeOptimizationFailure(&OptimizationFailure{})
	if code := FailureCodeOf(err); code != FailureCodeWorkerCrash {
		t.Fatalf("safeOptimizationFailure(invalid failure) code = %q, want worker_crash", code)
	}
	if got := err.Error(); got != FailureCodeWorkerCrash.String() {
		t.Fatalf("safeOptimizationFailure(invalid failure) = %q, want %q", got, FailureCodeWorkerCrash)
	}
}

// Implements DESIGN-004 JobStatusTracker bounded persisted failure vocabulary.
func TestOptimizationFailureCodeJSONAcceptsLegacyValuesAndRejectsUnknownValues(t *testing.T) {
	for _, code := range []OptimizationFailureCode{FailureCodeValidation, FailureCodeSolverTimeout, FailureCodeSolverInfeasible, FailureCodeWorkerCrash} {
		encoded, err := json.Marshal(code)
		if err != nil {
			t.Fatalf("json.Marshal(%s) error = %v", code, err)
		}
		var decoded OptimizationFailureCode
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != code {
			t.Fatalf("json.Unmarshal(%s) = %s, %v", encoded, decoded, err)
		}
	}
	for _, encoded := range []string{`""`, `"queue_unavailable"`, `"result_expired"`, `"arbitrary"`, `null`, `42`} {
		var decoded OptimizationFailureCode
		if err := json.Unmarshal([]byte(encoded), &decoded); err == nil {
			t.Fatalf("json.Unmarshal(%s) succeeded with %s", encoded, decoded)
		}
	}
	if _, err := json.Marshal(OptimizationFailureCode{}); err == nil {
		t.Fatal("zero failure code marshalled successfully")
	}
}

func TestValidateSolutionAcceptsToleranceBoundariesAndFloatingPointEpsilon(t *testing.T) {
	meal := validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5})
	baseRequest := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 10, nil)
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

func TestSolutionValidatorRequiresSavedDietIdentity(t *testing.T) {
	meal := validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1})
	validRequest := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 0, nil)
	validator := NewSolutionValidator([]repository.MealEntity{meal})
	tests := []struct {
		name    string
		mutate  func(*DietOptimizationRequest)
		wantErr bool
	}{
		{name: "valid identity", mutate: func(*DietOptimizationRequest) {}, wantErr: false},
		{name: "missing saved diet id", mutate: func(req *DietOptimizationRequest) { req.OriginalDiet.ID = uuid.Nil }, wantErr: true},
		{name: "missing saved diet owner", mutate: func(req *DietOptimizationRequest) { req.OriginalDiet.UserID = uuid.Nil }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest
			tt.mutate(&req)
			_, err := validator.Validate(LPSolution{validatorMealA.String(): 100}, req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SolutionValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && FailureCodeOf(err) != FailureCodeValidation {
				t.Fatalf("SolutionValidator.Validate() error code = %q, want %q", FailureCodeOf(err), FailureCodeValidation)
			}
		})
	}
}

func TestValidateSolutionRejectsMalformedQuantitiesIDsAndAlternatives(t *testing.T) {
	meal := validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1})
	req := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 0, []uuid.UUID{validatorMealB})
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
	meal := validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1})
	call := 0
	results, err := GenerateValidatedAlternatives(context.Background(), validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealA, Quantity: 100, Unit: "g"}}, 0, nil), []repository.MealEntity{meal}, 2, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		call++
		if call <= 2 {
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
	if got := err.Error(); got != FailureCodeWorkerCrash.String() {
		t.Fatalf("error = %q, want only safe failure code", got)
	}
	if got := results[0].Calories; got != 89 {
		t.Fatalf("partial result calories = %v, want 89", got)
	}
}

func validatorMeal(id uuid.UUID, state repository.PhysicalState, macros MacroTarget) repository.MealEntity {
	return repository.MealEntity{
		ID: id, Name: "Test meal", Type: repository.MealTypeSingle, PhysicalState: state,
		MacrosPer100:              repository.MacroValues{Protein: macros.Protein, Carbohydrates: macros.Carbohydrates, Fat: macros.Fat},
		NormalizedMacrosAvailable: true,
	}
}

func validatorRequest(entries []repository.SavedDietMealEntry, tolerance float64, excluded []uuid.UUID) DietOptimizationRequest {
	return DietOptimizationRequest{
		OriginalDiet: repository.SavedDiet{
			ID: uuid.MustParse("00000000-0000-4000-8000-000000000034"), UserID: uuid.MustParse("00000000-0000-4000-8000-000000000035"), Entries: entries,
		},
		TolerancePercent: tolerance,
		ExcludedMealIDs:  excluded,
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
			if FailureCodeOf(err) != tt.want || err.Error() != tt.want.String() {
				t.Fatalf("safe error = %q (%q), want code %q", err, FailureCodeOf(err), tt.want)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("safe error leaked solver diagnostic")
			}
		})
	}
}

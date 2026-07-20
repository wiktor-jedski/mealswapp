// Implements DESIGN-004 DiversityPenalizer and SolutionValidator verification.
package optimization

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce verifies
// IT-ARCH-004-002, ARCH-004, DESIGN-004 DiversityPenalizer/SolutionValidator,
// and SW-REQ-021/SW-REQ-022/SW-REQ-023/SW-REQ-030 across the real generation,
// canonical validation, projection, and solver-boundary collaboration.
func TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB)
	req := diversityRequest(meals[0], nil)
	for _, generate := range []struct {
		name string
		run  func(context.Context, AlternativeSolveFunc) (int, error)
	}{
		{name: "raw", run: func(ctx context.Context, solve AlternativeSolveFunc) (int, error) {
			results, err := GenerateAlternatives(ctx, req, meals, 2, solve)
			return len(results), err
		}},
		{name: "validated", run: func(ctx context.Context, solve AlternativeSolveFunc) (int, error) {
			results, err := GenerateValidatedAlternatives(ctx, req, meals, 2, solve)
			return len(results), err
		}},
	} {
		t.Run(generate.name, func(t *testing.T) {
			builds, projections, calls := 0, 0, 0
			ctx := context.WithValue(context.Background(), generationInstrumentationContextKey{}, &generationInstrumentation{
				indexBuilt:          func() { builds++ },
				validationProjected: func() { projections++ },
			})
			resultCount, err := generate.run(ctx, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
				selected := []uuid.UUID{diversityMealA, diversityMealB}[calls/2]
				calls++
				return LPSolution{selected.String(): 100}, nil
			})
			if err != nil {
				t.Fatalf("generator error = %v", err)
			}
			if builds != 1 || projections != resultCount || resultCount != 2 {
				t.Fatalf("builds = %d, projections = %d, results = %d; want 1, 2, 2", builds, projections, resultCount)
			}
		})
	}
}

func TestPublicAlternativeGeneratorsCapAttemptBudgetBeforeMultiplication(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB, diversityMealC)
	req := diversityRequest(meals[0], nil)
	for _, limit := range []int{4, int(^uint(0) >> 1)} {
		for _, generate := range []struct {
			name string
			run  func(AlternativeSolveFunc) (int, error)
		}{
			{name: "raw", run: func(solve AlternativeSolveFunc) (int, error) {
				results, err := GenerateAlternatives(context.Background(), req, meals, limit, solve)
				return len(results), err
			}},
			{name: "validated", run: func(solve AlternativeSolveFunc) (int, error) {
				results, err := GenerateValidatedAlternatives(context.Background(), req, meals, limit, solve)
				return len(results), err
			}},
		} {
			t.Run(fmt.Sprintf("%s/limit_%d", generate.name, limit), func(t *testing.T) {
				calls := 0
				resultCount, err := generate.run(func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
					selected := []uuid.UUID{diversityMealA, diversityMealB, diversityMealC}[calls/2]
					calls++
					return LPSolution{selected.String(): 100}, nil
				})
				if err != nil || resultCount != MaxAlternativeCount || calls != MaxAlternativeCount*2 {
					t.Fatalf("results = %d, calls = %d, error = %v; want 3, 6, nil", resultCount, calls, err)
				}
			})
		}
	}
}

func TestPublicAlternativeGeneratorsDoNotSolveNonPositiveLimits(t *testing.T) {
	meal := diversityMeals(diversityMealA)[0]
	req := diversityRequest(meal, nil)
	for _, limit := range []int{0, -1} {
		for _, generate := range []struct {
			name string
			run  func(AlternativeSolveFunc) (int, error)
		}{
			{name: "raw", run: func(solve AlternativeSolveFunc) (int, error) {
				results, err := GenerateAlternatives(context.Background(), req, []repository.MealEntity{meal}, limit, solve)
				return len(results), err
			}},
			{name: "validated", run: func(solve AlternativeSolveFunc) (int, error) {
				results, err := GenerateValidatedAlternatives(context.Background(), req, []repository.MealEntity{meal}, limit, solve)
				return len(results), err
			}},
		} {
			t.Run(fmt.Sprintf("%s/limit_%d", generate.name, limit), func(t *testing.T) {
				resultCount, err := generate.run(func(context.Context, LPModel, ObjectiveFunction) (LPSolution, error) {
					t.Fatal("solver called for non-positive limit")
					return nil, nil
				})
				if err != nil || resultCount != 0 {
					t.Fatalf("results = %d, error = %v; want 0, nil", resultCount, err)
				}
			})
		}
	}
}

// TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation verifies
// IT-ARCH-004-002, ARCH-004, DESIGN-004 SolutionValidator/DiversityPenalizer,
// and SW-REQ-021/SW-REQ-023/SW-REQ-030 malformed solver-output handling.
func TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB)
	validator := NewSolutionValidator(meals)
	var projections int
	validator.validationObserver = func() { projections++ }
	call := 0
	validated, err := generateAlternativePipeline(context.Background(), diversityRequest(meals[0], nil), validator, 2, 6, func(_ context.Context, model LPModel, _ ObjectiveFunction) (LPSolution, error) {
		call++
		if call == 3 && len(model.Constraints) != 4 {
			t.Fatalf("second-attempt constraints = %d, want three macros plus one accepted alternative", len(model.Constraints))
		}
		return LPSolution{diversityMealA.String(): 100}, nil
	})
	if err == nil || FailureCodeOf(err) != FailureCodeValidation {
		t.Fatalf("error = %v (%q), want failed_validation", err, FailureCodeOf(err))
	}
	if len(validated) != 1 || projections != 1 || call != 3 {
		t.Fatalf("results = %d, projections = %d, solver calls = %d; want 1, 1, 3", len(validated), projections, call)
	}
}

func TestAlternativePipelineCanonicalizesResidueForCurrentAndPreviousSolutions(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB)
	call := 0
	results, err := GenerateAlternatives(context.Background(), diversityRequest(meals[0], nil), meals, 2, func(_ context.Context, model LPModel, _ ObjectiveFunction) (LPSolution, error) {
		attempt := call / 2
		call++
		if attempt == 0 {
			return LPSolution{diversityMealA.String(): 100, diversityMealB.String(): 5e-10}, nil
		}
		if got := findConstraint(t, model, "alternative_1").Coefficients; !reflect.DeepEqual(got, map[string]float64{diversityMealA.String(): 1}) {
			t.Fatalf("canonical previous-solution exclusion = %#v", got)
		}
		return LPSolution{diversityMealA.String(): -5e-10, diversityMealB.String(): 100}, nil
	})
	if err != nil {
		t.Fatalf("GenerateAlternatives() error = %v", err)
	}
	want := []LPSolution{{diversityMealA.String(): 100}, {diversityMealB.String(): 100}}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("canonical alternatives = %#v, want %#v", results, want)
	}
}

// TestAlternativePipelineAttemptExhaustionReturnsValidPartialResults verifies
// IT-ARCH-004-002, ARCH-004, DESIGN-004 DiversityPenalizer/SolutionValidator,
// and SW-REQ-021/SW-REQ-030 bounded distinct-alternative partial completion.
func TestAlternativePipelineAttemptExhaustionReturnsValidPartialResults(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB)
	validated, err := generateAlternativePipeline(context.Background(), diversityRequest(meals[0], nil), NewSolutionValidator(meals), 2, 1, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		return LPSolution{diversityMealA.String(): 100}, nil
	})
	if err != nil || len(validated) != 1 {
		t.Fatalf("attempt exhaustion returned %d results and error %v, want one valid partial and nil", len(validated), err)
	}
}

func TestSolutionValidatorSelectedMealCountMatchesOpenAPIBoundary(t *testing.T) {
	meals := make([]repository.MealEntity, 101)
	solution := make(LPSolution, len(meals))
	for index := range meals {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", index+1))
		meals[index] = validatorMeal(id, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1})
		solution[id.String()] = 1
	}
	req := validatorRequest([]repository.SavedDietMealEntry{{MealID: meals[0].ID, Quantity: 100, Unit: "g"}}, 100, nil)
	validator := NewSolutionValidator(meals)
	withOneResidue := make(LPSolution, len(solution))
	for id, quantity := range solution {
		withOneResidue[id] = quantity
	}
	withOneResidue[meals[100].ID.String()] = 5e-10
	if _, err := validator.Validate(withOneResidue, req); err != nil {
		t.Fatalf("100 selected meals plus residue: %v", err)
	}
	if _, err := validator.Validate(solution, req); err == nil || FailureCodeOf(err) != FailureCodeValidation {
		t.Fatalf("101 selected meals error = %v (%q), want failed_validation", err, FailureCodeOf(err))
	}
}

func TestAlternativePipelineRejectsNilContextAndMalformedSnapshots(t *testing.T) {
	meal := diversityMeals(diversityMealA)[0]
	req := diversityRequest(meal, nil)
	duplicate := []repository.MealEntity{meal, meal}
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		meals []repository.MealEntity
	}{
		{name: "nil context", ctx: nil, meals: []repository.MealEntity{meal}},
		{name: "nil snapshot", ctx: context.Background(), meals: nil},
		{name: "duplicate snapshot IDs", ctx: context.Background(), meals: duplicate},
		{name: "nil snapshot ID", ctx: context.Background(), meals: []repository.MealEntity{{}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			results, err := GenerateValidatedAlternatives(tt.ctx, req, tt.meals, 1, func(context.Context, LPModel, ObjectiveFunction) (LPSolution, error) {
				t.Fatal("solver called for invalid pipeline input")
				return nil, nil
			})
			if err == nil || len(results) != 0 || FailureCodeOf(err) != FailureCodeValidation {
				t.Fatalf("results = %v, error = %v (%q), want no results and failed_validation", results, err, FailureCodeOf(err))
			}
		})
	}
}

// TestAlternativePipelineSnapshotIgnoresLaterCallerMutation verifies
// IT-ARCH-004-002, ARCH-004, DESIGN-004 SolutionValidator, and
// SW-REQ-021/SW-REQ-023 immutable repository-to-solver data flow.
func TestAlternativePipelineSnapshotIgnoresLaterCallerMutation(t *testing.T) {
	meals := diversityMeals(diversityMealA, diversityMealB)
	req := diversityRequest(meals[0], nil)
	mutated := false
	validated, err := GenerateValidatedAlternatives(context.Background(), req, meals, 1, func(_ context.Context, _ LPModel, _ ObjectiveFunction) (LPSolution, error) {
		if !mutated {
			mutated = true
			meals[0].ID = uuid.Nil
			meals[0].MacrosPer100 = repository.MacroValues{}
			req.OriginalDiet.Entries[0].MealID = uuid.Nil
			req.ExcludedMealIDs = append(req.ExcludedMealIDs, diversityMealA)
		}
		return LPSolution{diversityMealA.String(): 100}, nil
	})
	if err != nil || len(validated) != 1 {
		t.Fatalf("immutable snapshot result count = %d, error = %v", len(validated), err)
	}
}

func TestDeterministicObjectiveAndConstraintEvaluation(t *testing.T) {
	ids := []string{diversityMealA.String(), diversityMealB.String(), diversityMealC.String()}
	coefficients := map[string]float64{ids[0]: 1e16, ids[1]: 1, ids[2]: -1e16}
	solution := LPSolution{ids[0]: 1, ids[1]: 1, ids[2]: 1}
	for iteration := 0; iteration < 100; iteration++ {
		value, err := objectiveValueForSolution(ObjectiveFunction{Coefficients: coefficients}, solution)
		if err != nil || value != 0 {
			t.Fatalf("iteration %d deterministic objective = %v, error = %v; want 0", iteration, value, err)
		}
		model := LPModel{
			Variables:   []LPVariable{{ItemID: ids[2], UpperBound: 1}, {ItemID: ids[0], UpperBound: 1}, {ItemID: ids[1], UpperBound: 1}},
			Constraints: []LPConstraint{{Name: "stable", LowerBound: 0, UpperBound: 0, Coefficients: coefficients}},
		}
		if err := solutionSatisfiesModel(solution, model); err != nil {
			t.Fatalf("iteration %d deterministic model evaluation: %v", iteration, err)
		}
	}
}

// TestSolutionValidatorConcurrentMetricAndLiquidProjection verifies
// IT-ARCH-004-002, ARCH-004, DESIGN-004 SolutionValidator, and
// SW-REQ-021/SW-REQ-023 under concurrent canonical projection.
func TestSolutionValidatorConcurrentMetricAndLiquidProjection(t *testing.T) {
	callerMeals := []repository.MealEntity{
		validatorMeal(validatorMealB, repository.PhysicalStateLiquid, MacroTarget{Protein: 4, Carbohydrates: 8, Fat: 1}),
		validatorMeal(validatorMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 20, Carbohydrates: 10, Fat: 2}),
	}
	validator := NewSolutionValidator(callerMeals)
	req := validatorRequest([]repository.SavedDietMealEntry{{MealID: validatorMealB, Quantity: 50, Unit: "ml"}, {MealID: validatorMealA, Quantity: 50, Unit: "g"}}, 0, nil)
	solution := LPSolution{validatorMealB.String(): 50, validatorMealA.String(): 50}
	var failures atomic.Int64
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for iteration := 0; iteration < 100; iteration++ {
				alternative, err := validator.Validate(solution, req)
				if err != nil || len(alternative.Meals) != 2 || alternative.Meals[0].Unit != "g" || alternative.Meals[1].Unit != "ml" {
					failures.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if failures.Load() != 0 {
		t.Fatalf("concurrent validation failures = %d", failures.Load())
	}
}

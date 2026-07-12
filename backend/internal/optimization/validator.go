package optimization

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-004 SolutionValidator.
const (
	// SolutionValidationEpsilon absorbs arithmetic noise at a hard tolerance
	// boundary without making a materially invalid solution acceptable.
	SolutionValidationEpsilon = 1e-9
	maxAlternativeMealCount   = 100
)

// OptimizationFailureCode is the stable, user-safe failure vocabulary for an
// optimization job. Internal validation and solver diagnostics are never
// included in Error's returned string.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
type OptimizationFailureCode string

// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
const (
	FailureCodeValidation       OptimizationFailureCode = "failed_validation"
	FailureCodeSolverTimeout    OptimizationFailureCode = "solver_timeout"
	FailureCodeSolverInfeasible OptimizationFailureCode = "solver_infeasible"
	FailureCodeQueueUnavailable OptimizationFailureCode = "queue_unavailable"
	FailureCodeWorkerCrash      OptimizationFailureCode = "worker_crash"
	FailureCodeResultExpired    OptimizationFailureCode = "result_expired"
)

// OptimizationFailure is a safe terminal optimization error. Cause is kept
// for server-side classification and logging, while Error exposes only Code.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
type OptimizationFailure struct {
	Code  OptimizationFailureCode
	cause error
}

// Error returns only the stable public failure code.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
func (e *OptimizationFailure) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code)
}

// Unwrap returns the internal cause for server-side errors.Is/errors.As use.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
func (e *OptimizationFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// FailureCodeOf returns the public optimization failure code carried by err.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
func FailureCodeOf(err error) OptimizationFailureCode {
	var failure *OptimizationFailure
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

// DietAlternative is a solver assignment after server-side recalculation.
// Macros and Calories are derived from repository meal data; no solver totals
// are accepted. SimilarityScore remains a later best-effort projection.
// Implements DESIGN-004 SolutionValidator.
type DietAlternative struct {
	Meals           []MealQuantity `json:"meals"`
	Macros          MacroTarget    `json:"macros"`
	Calories        float64        `json:"calories"`
	SimilarityScore float64        `json:"similarityScore"`
}

// SolutionValidator validates sparse solver assignments against the same
// repository meal data used to build the LP model.
// Implements DESIGN-004 SolutionValidator.
type SolutionValidator struct {
	meals []repository.MealEntity
}

// NewSolutionValidator creates a repository-backed solution validator.
// Implements DESIGN-004 SolutionValidator.
func NewSolutionValidator(meals []repository.MealEntity) *SolutionValidator {
	return &SolutionValidator{meals: append([]repository.MealEntity(nil), meals...)}
}

// ValidateSolution validates one solver assignment and independently derives
// its meal quantities, macros, and calories from repository data. The optional
// meal argument keeps the package-level API convenient while the validator
// type provides a reusable dependency-injected boundary.
// Implements DESIGN-004 SolutionValidator and SW-REQ-021/SW-REQ-022.
func ValidateSolution(solution LPSolution, req DietOptimizationRequest, repositoryMeals ...[]repository.MealEntity) (DietAlternative, error) {
	var meals []repository.MealEntity
	if len(repositoryMeals) == 1 {
		meals = repositoryMeals[0]
	} else if len(repositoryMeals) == 0 {
		meals = req.RepositoryMeals
	} else {
		return DietAlternative{}, solutionValidationError("repository meal data was supplied more than once")
	}
	return NewSolutionValidator(meals).Validate(solution, req)
}

// Validate validates one solver assignment using the validator's repository
// meal snapshot.
// Implements DESIGN-004 SolutionValidator.
func (v *SolutionValidator) Validate(solution LPSolution, req DietOptimizationRequest) (DietAlternative, error) {
	if v == nil {
		return DietAlternative{}, solutionValidationError("solution validator is required")
	}
	if len(solution) == 0 || len(solution) > maxAlternativeMealCount {
		return DietAlternative{}, solutionValidationError("alternative meal count is invalid")
	}

	meals, err := mealIndex(v.meals)
	if err != nil {
		return DietAlternative{}, solutionValidationError("repository meal data is invalid")
	}
	if len(meals) == 0 {
		return DietAlternative{}, solutionValidationError("repository meal data is required")
	}
	if !finite(req.TolerancePercent) || req.TolerancePercent < 0 || req.TolerancePercent > 100 {
		return DietAlternative{}, solutionValidationError("tolerance is invalid")
	}

	excluded := excludedMealIDs(req)
	ids := make([]string, 0, len(solution))
	for itemID, quantity := range solution {
		meal, ok := meals[itemID]
		if !ok || meal.ID == uuid.Nil {
			return DietAlternative{}, solutionValidationError("alternative contains an unknown meal")
		}
		if excluded[itemID] {
			return DietAlternative{}, solutionValidationError("alternative contains an excluded meal")
		}
		if !finite(quantity) || quantity < 0 {
			return DietAlternative{}, solutionValidationError("alternative quantities are invalid")
		}
		if quantity > solutionMaxQuantity(req)+quantityEpsilon(quantity) {
			return DietAlternative{}, solutionValidationError("alternative quantity exceeds the allowed maximum")
		}
		if quantity > 0 {
			if err := validateMeal(meal); err != nil {
				return DietAlternative{}, solutionValidationError("alternative meal data is invalid")
			}
			ids = append(ids, itemID)
		}
	}
	if len(ids) == 0 {
		return DietAlternative{}, solutionValidationError("alternative must contain a positive meal quantity")
	}

	target, err := targetForRequest(req, meals)
	if err != nil {
		return DietAlternative{}, solutionValidationError("optimization target is invalid")
	}

	sort.Strings(ids)
	alternative := DietAlternative{Meals: make([]MealQuantity, 0, len(ids))}
	for position, itemID := range ids {
		meal := meals[itemID]
		quantity := solution[itemID]
		macros := scaleMealMacros(meal.MacrosPer100, quantity)
		alternative.Macros.Protein += macros.Protein
		alternative.Macros.Carbohydrates += macros.Carbohydrates
		alternative.Macros.Fat += macros.Fat
		alternative.Meals = append(alternative.Meals, MealQuantity{
			MealID:   meal.ID,
			Quantity: quantity,
			Unit:     mealBaseUnit(meal),
			Position: position,
		})
	}
	alternative.Macros.Carbs = alternative.Macros.Carbohydrates
	alternative.Calories = alternative.Macros.Protein*4 + alternative.Macros.Carbohydrates*4 + alternative.Macros.Fat*9
	if !finite(alternative.Macros.Protein) || !finite(alternative.Macros.Carbohydrates) || !finite(alternative.Macros.Fat) || !finite(alternative.Calories) {
		return DietAlternative{}, solutionValidationError("alternative totals are not finite")
	}
	if alternative.Macros.Protein < 0 || alternative.Macros.Carbohydrates < 0 || alternative.Macros.Fat < 0 || alternative.Calories < 0 {
		return DietAlternative{}, solutionValidationError("alternative totals are negative")
	}
	if !macroWithinTolerance(alternative.Macros.Protein, target.Protein, req.TolerancePercent) ||
		!macroWithinTolerance(alternative.Macros.Carbohydrates, target.Carbohydrates, req.TolerancePercent) ||
		!macroWithinTolerance(alternative.Macros.Fat, target.Fat, req.TolerancePercent) {
		return DietAlternative{}, solutionValidationError("alternative macros are outside the requested tolerance")
	}
	return alternative, nil
}

// GenerateValidatedAlternatives adapts the raw diversity generator to the
// publication-safe alternative shape while retaining valid partial results if
// a later solve fails.
// Implements DESIGN-004 SolutionValidator and SW-REQ-030.
func GenerateValidatedAlternatives(ctx context.Context, req DietOptimizationRequest, meals []repository.MealEntity, limit int, solve AlternativeSolveFunc) ([]DietAlternative, error) {
	solutions, solveErr := GenerateAlternatives(ctx, req, meals, limit, solve)
	result := make([]DietAlternative, 0, len(solutions))
	for _, solution := range solutions {
		alternative, err := ValidateSolution(solution, req, meals)
		if err != nil {
			return result, safeOptimizationFailure(err)
		}
		result = append(result, alternative)
	}
	if solveErr != nil {
		return result, safeOptimizationFailure(solveErr)
	}
	return result, nil
}

// safeOptimizationFailure maps internal validation/solver failures into the
// public optimization failure vocabulary without exposing diagnostics.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
func safeOptimizationFailure(err error) error {
	if err == nil {
		return nil
	}
	var existing *OptimizationFailure
	if errors.As(err, &existing) {
		return err
	}
	code := FailureCodeWorkerCrash
	var solverErr *SolverError
	switch {
	case errors.As(err, &solverErr) && solverErr.Kind == SolverErrorInfeasible:
		code = FailureCodeSolverInfeasible
	case errors.As(err, &solverErr) && solverErr.Kind == SolverErrorTimeout:
		code = FailureCodeSolverTimeout
	case errors.Is(err, context.DeadlineExceeded):
		code = FailureCodeSolverTimeout
	case repository.IsKind(err, repository.ErrorKindValidation):
		code = FailureCodeValidation
	}
	return &OptimizationFailure{Code: code, cause: err}
}

// mealIndex creates a unique repository meal lookup by solver item ID.
// Implements DESIGN-004 SolutionValidator.
func mealIndex(meals []repository.MealEntity) (map[string]repository.MealEntity, error) {
	result := make(map[string]repository.MealEntity, len(meals))
	for _, meal := range meals {
		if meal.ID == uuid.Nil {
			return nil, errors.New("meal id is required")
		}
		id := meal.ID.String()
		if _, exists := result[id]; exists {
			return nil, errors.New("duplicate meal id")
		}
		result[id] = meal
	}
	return result, nil
}

// solutionMaxQuantity resolves the request's effective solver quantity bound.
// Implements DESIGN-004 SolutionValidator.
func solutionMaxQuantity(req DietOptimizationRequest) float64 {
	if req.MaxQuantity == 0 {
		return DefaultMaxQuantity
	}
	if !finite(req.MaxQuantity) || req.MaxQuantity < 0 {
		return -1
	}
	return req.MaxQuantity
}

// quantityEpsilon returns a scale-aware comparison tolerance.
// Implements DESIGN-004 SolutionValidator.
func quantityEpsilon(value float64) float64 {
	return SolutionValidationEpsilon * math.Max(1, math.Abs(value))
}

// macroWithinTolerance checks a recomputed macro against its requested band.
// Implements DESIGN-004 SolutionValidator.
func macroWithinTolerance(value, target, tolerance float64) bool {
	if !finite(value) || !finite(target) || !finite(tolerance) || target < 0 || tolerance < 0 {
		return false
	}
	margin := target * tolerance / 100
	if !finite(margin) {
		return false
	}
	return value >= target-margin-quantityEpsilon(target-margin) && value <= target+margin+quantityEpsilon(target+margin)
}

// scaleMealMacros independently scales repository per-100 macros by quantity.
// Implements DESIGN-004 SolutionValidator.
func scaleMealMacros(base repository.MacroValues, quantity float64) repository.MacroValues {
	factor := quantity / 100
	return repository.MacroValues{
		Protein:       base.Protein * factor,
		Carbohydrates: base.Carbohydrates * factor,
		Fat:           base.Fat * factor,
	}
}

// mealBaseUnit maps repository physical state to the solver's canonical unit.
// Implements DESIGN-004 SolutionValidator.
func mealBaseUnit(meal repository.MealEntity) string {
	if meal.PhysicalState == repository.PhysicalStateLiquid {
		return "ml"
	}
	return "g"
}

// solutionValidationError hides internal validation details behind a safe code.
// Implements DESIGN-004 SolutionValidator.
func solutionValidationError(message string) error {
	return &OptimizationFailure{
		Code:  FailureCodeValidation,
		cause: repository.NewError(repository.ErrorKindValidation, fmt.Sprintf("solution validation failed: %s", message), nil),
	}
}

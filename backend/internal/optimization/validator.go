package optimization

import (
	"context"
	"encoding/json"
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
type OptimizationFailureCode struct {
	value string
}

// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
var (
	// FailureCodeValidation identifies rejected or invalid solver output.
	FailureCodeValidation = OptimizationFailureCode{value: "failed_validation"}
	// FailureCodeSolverTimeout identifies a job that exceeded its solver deadline.
	FailureCodeSolverTimeout = OptimizationFailureCode{value: "solver_timeout"}
	// FailureCodeSolverInfeasible identifies a model with no feasible solution.
	FailureCodeSolverInfeasible = OptimizationFailureCode{value: "solver_infeasible"}
	// FailureCodeWorkerCrash identifies exhausted recovery after worker failure.
	FailureCodeWorkerCrash = OptimizationFailureCode{value: "worker_crash"}
)

// ParseOptimizationFailureCode accepts only the persisted terminal vocabulary.
// Implements DESIGN-004 JobStatusTracker persisted compatibility boundary.
func ParseOptimizationFailureCode(value string) (OptimizationFailureCode, bool) {
	switch value {
	case FailureCodeValidation.value:
		return FailureCodeValidation, true
	case FailureCodeSolverTimeout.value:
		return FailureCodeSolverTimeout, true
	case FailureCodeSolverInfeasible.value:
		return FailureCodeSolverInfeasible, true
	case FailureCodeWorkerCrash.value:
		return FailureCodeWorkerCrash, true
	default:
		return OptimizationFailureCode{}, false
	}
}

// String returns the stable wire value without exposing a construction seam.
// Implements DESIGN-004 JobStatusTracker.
func (c OptimizationFailureCode) String() string { return c.value }

// Valid reports whether the code belongs to the bounded terminal vocabulary.
// Implements DESIGN-004 JobStatusTracker.
func (c OptimizationFailureCode) Valid() bool {
	parsed, ok := ParseOptimizationFailureCode(c.value)
	return ok && parsed == c
}

// MarshalJSON writes the legacy-compatible string representation and rejects zero values.
// Implements DESIGN-004 JobStatusTracker persisted compatibility boundary.
func (c OptimizationFailureCode) MarshalJSON() ([]byte, error) {
	if !c.Valid() {
		return nil, errors.New("optimization failure code is invalid")
	}
	return json.Marshal(c.value)
}

// UnmarshalJSON accepts retained legacy string values and rejects unknown or empty codes.
// Implements DESIGN-004 JobStatusTracker persisted compatibility boundary.
func (c *OptimizationFailureCode) UnmarshalJSON(data []byte) error {
	if c == nil {
		return errors.New("optimization failure code target is required")
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return errors.New("optimization failure code must be a string")
	}
	parsed, ok := ParseOptimizationFailureCode(value)
	if !ok {
		return errors.New("optimization failure code is invalid")
	}
	*c = parsed
	return nil
}

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
	return e.Code.String()
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
	if errors.As(err, &failure) && failure != nil {
		return failure.Code
	}
	return OptimizationFailureCode{}
}

// DietAlternative is a solver assignment after server-side recalculation.
// Macros and Calories are derived from repository meal data; no solver totals
// are accepted. SimilarityScore is the rounded quantity-weighted Jaccard
// similarity between original and alternative canonical meal quantities.
// Implements DESIGN-004 SolutionValidator.
type DietAlternative struct {
	Meals           []MealQuantity `json:"meals"`
	Macros          MacroTarget    `json:"macros"`
	Calories        float64        `json:"calories"`
	SimilarityScore float64        `json:"similarityScore"`
}

// UnmarshalJSON preserves the required numeric similarityScore distinction
// between an explicit zero and missing or null persisted data.
// Implements DESIGN-004 SolutionValidator persisted compatibility boundary.
func (a *DietAlternative) UnmarshalJSON(data []byte) error {
	if a == nil {
		return errors.New("optimization alternative target is required")
	}
	var decoded struct {
		Meals           []MealQuantity  `json:"meals"`
		Macros          MacroTarget     `json:"macros"`
		Calories        float64         `json:"calories"`
		SimilarityScore json.RawMessage `json:"similarityScore"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if len(decoded.SimilarityScore) == 0 {
		return errors.New("optimization alternative similarity score is required")
	}
	var score *float64
	if err := json.Unmarshal(decoded.SimilarityScore, &score); err != nil {
		return errors.New("optimization alternative similarity score must be a number")
	}
	if score == nil {
		return errors.New("optimization alternative similarity score is required")
	}
	*a = DietAlternative{
		Meals: decoded.Meals, Macros: decoded.Macros, Calories: decoded.Calories, SimilarityScore: *score,
	}
	return nil
}

// ValidateDietAlternative checks the authoritative result shape before it
// crosses persistence or HTTP projection boundaries.
// Implements DESIGN-004 SolutionValidator authoritative result publication.
func ValidateDietAlternative(alternative DietAlternative) error {
	if len(alternative.Meals) == 0 || len(alternative.Meals) > maxAlternativeMealCount {
		return errors.New("optimization alternative meal count is invalid")
	}
	for position, meal := range alternative.Meals {
		if meal.MealID == uuid.Nil || !finite(meal.Quantity) || meal.Quantity <= 0 || meal.Quantity > MaximumMealQuantity ||
			(meal.Unit != "g" && meal.Unit != "ml") || meal.Position != position {
			return errors.New("optimization alternative meal projection is invalid")
		}
	}
	if !boundedProjectionNumber(alternative.Macros.Protein) ||
		!boundedProjectionNumber(alternative.Macros.Carbohydrates) ||
		!boundedProjectionNumber(alternative.Macros.Fat) ||
		!boundedProjectionNumber(alternative.Calories) {
		return errors.New("optimization alternative macro projection is invalid")
	}
	if !finite(alternative.SimilarityScore) || alternative.SimilarityScore < 0 || alternative.SimilarityScore > 1 ||
		math.Abs(alternative.SimilarityScore*10_000-math.Round(alternative.SimilarityScore*10_000)) > SolutionValidationEpsilon {
		return errors.New("optimization alternative similarity score is invalid")
	}
	return nil
}

// boundedProjectionNumber enforces the public macro/calorie projection range.
// Implements DESIGN-004 SolutionValidator authoritative result publication.
func boundedProjectionNumber(value float64) bool {
	return finite(value) && value >= 0 && value <= 1_000_000_000
}

// SolutionValidator validates sparse solver assignments against the same
// repository meal data used to build the LP model.
// Implements DESIGN-004 SolutionValidator.
type SolutionValidator struct {
	meals              map[string]repository.MealEntity
	orderedMealIDs     []string
	snapshotErr        error
	validationObserver func()
}

// generationInstrumentation observes the actual snapshot-index and
// validation/projection boundaries in focused package tests.
// Implements DESIGN-004 SolutionValidator verification.
type generationInstrumentation struct {
	indexBuilt          func()
	validationProjected func()
}

// generationInstrumentationContextKey isolates package-test instrumentation.
// Implements DESIGN-004 SolutionValidator verification.
type generationInstrumentationContextKey struct{}

// generationInstrumentationFromContext returns optional package-test probes.
// Implements DESIGN-004 SolutionValidator verification.
func generationInstrumentationFromContext(ctx context.Context) *generationInstrumentation {
	if ctx == nil {
		return nil
	}
	instrumentation, _ := ctx.Value(generationInstrumentationContextKey{}).(*generationInstrumentation)
	return instrumentation
}

// NewSolutionValidator creates a repository-backed solution validator.
// Implements DESIGN-004 SolutionValidator.
func NewSolutionValidator(meals []repository.MealEntity) *SolutionValidator {
	return newSolutionValidator(meals, nil)
}

// newSolutionValidator wires one immutable index to model and projection use.
// Implements DESIGN-004 SolutionValidator.
func newSolutionValidator(meals []repository.MealEntity, instrumentation *generationInstrumentation) *SolutionValidator {
	indexed, orderedIDs, err := immutableMealSnapshot(meals, instrumentation)
	validator := &SolutionValidator{meals: indexed, orderedMealIDs: orderedIDs, snapshotErr: err}
	if instrumentation != nil {
		validator.validationObserver = instrumentation.validationProjected
	}
	return validator
}

// ValidateSolution validates one solver assignment and independently derives
// its meal quantities, macros, and calories from explicit repository data.
// Implements DESIGN-004 SolutionValidator and SW-REQ-021/SW-REQ-022.
func ValidateSolution(solution LPSolution, req DietOptimizationRequest, meals []repository.MealEntity) (DietAlternative, error) {
	return NewSolutionValidator(meals).Validate(solution, req)
}

// Validate validates one solver assignment using the validator's repository
// meal snapshot.
// Implements DESIGN-004 SolutionValidator.
func (v *SolutionValidator) Validate(solution LPSolution, req DietOptimizationRequest) (DietAlternative, error) {
	if v == nil {
		return DietAlternative{}, solutionValidationError("solution validator is required")
	}
	if v.snapshotErr != nil || len(v.meals) == 0 {
		return DietAlternative{}, solutionValidationError("repository meal data is invalid")
	}
	if v.validationObserver != nil {
		v.validationObserver()
	}
	if err := validateRequest(req); err != nil {
		return DietAlternative{}, solutionValidationError("optimization request is invalid")
	}
	canonical, _, err := canonicalQuantities(solution, v.meals)
	if err != nil || len(canonical) > maxAlternativeMealCount {
		return DietAlternative{}, solutionValidationError("alternative meal count is invalid")
	}
	excluded, err := excludedMealIDs(req)
	if err != nil {
		return DietAlternative{}, solutionValidationError("excluded meal data is invalid")
	}
	ids := make([]string, 0, len(canonical))
	for itemID, quantity := range canonical {
		meal, ok := v.meals[itemID]
		if !ok || meal.ID == uuid.Nil {
			return DietAlternative{}, solutionValidationError("alternative contains an unknown meal")
		}
		if _, isExcluded := excluded[meal.ID]; isExcluded {
			return DietAlternative{}, solutionValidationError("alternative contains an excluded meal")
		}
		if quantity > MaximumMealQuantity+quantityTolerance(quantity, MaximumMealQuantity) {
			return DietAlternative{}, solutionValidationError("alternative quantity exceeds the allowed maximum")
		}
		if err := validateMeal(meal); err != nil {
			return DietAlternative{}, solutionValidationError("alternative meal data is invalid")
		}
		ids = append(ids, itemID)
	}
	if len(ids) == 0 {
		return DietAlternative{}, solutionValidationError("alternative must contain a positive meal quantity")
	}

	target, err := targetForRequest(req, v.meals)
	if err != nil {
		return DietAlternative{}, solutionValidationError("optimization target is invalid")
	}

	sort.Strings(ids)
	alternative := DietAlternative{Meals: make([]MealQuantity, 0, len(ids))}
	for position, itemID := range ids {
		meal := v.meals[itemID]
		quantity := canonical[itemID]
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
	alternative.SimilarityScore, err = quantityWeightedSimilarity(req, canonical, v.meals)
	if err != nil {
		return DietAlternative{}, solutionValidationError("alternative similarity could not be calculated")
	}
	return alternative, nil
}

// quantityWeightedSimilarity calculates intersection-over-union over canonical
// g/ml quantities and rounds the public ratio to four decimal places.
// Implements DESIGN-004 SolutionValidator authoritative similarity projection.
func quantityWeightedSimilarity(req DietOptimizationRequest, alternative map[string]float64, meals map[string]repository.MealEntity) (float64, error) {
	original := make(map[string]float64, len(req.OriginalDiet.Entries))
	for _, entry := range req.OriginalDiet.Entries {
		meal, ok := meals[entry.MealID.String()]
		if !ok {
			return 0, errors.New("original meal is unavailable")
		}
		quantity, err := quantityInNutritionBasis(entry, meal)
		if err != nil {
			return 0, err
		}
		original[entry.MealID.String()] += quantity
	}
	intersection, union := 0.0, 0.0
	ids := make(map[string]struct{}, len(original)+len(alternative))
	for id := range original {
		ids[id] = struct{}{}
	}
	for id := range alternative {
		ids[id] = struct{}{}
	}
	for id := range ids {
		intersection += math.Min(original[id], alternative[id])
		union += math.Max(original[id], alternative[id])
	}
	if !finite(intersection) || !finite(union) || union <= 0 {
		return 0, errors.New("similarity quantities are invalid")
	}
	score := math.Round((intersection/union)*10_000) / 10_000
	if !finite(score) || score < 0 || score > 1 {
		return 0, errors.New("similarity score is invalid")
	}
	return score, nil
}

// GenerateValidatedAlternatives adapts the raw diversity generator to the
// publication-safe alternative shape while retaining valid partial results if
// a later solve fails.
// Implements DESIGN-004 SolutionValidator and SW-REQ-030.
func GenerateValidatedAlternatives(ctx context.Context, req DietOptimizationRequest, meals []repository.MealEntity, limit int, solve AlternativeSolveFunc) ([]DietAlternative, error) {
	limit, attemptBudget := alternativeGenerationLimits(limit)
	validator := newSolutionValidator(meals, generationInstrumentationFromContext(ctx))
	validated, err := generateAlternativePipeline(ctx, cloneOptimizationRequest(req), validator, limit, attemptBudget, solve)
	result := make([]DietAlternative, len(validated))
	for index := range validated {
		result[index] = validated[index].alternative
	}
	return result, err
}

// safeOptimizationFailure maps internal validation/solver failures into the
// public optimization failure vocabulary without exposing diagnostics.
// Implements DESIGN-004 SolutionValidator and JobStatusTracker.
func safeOptimizationFailure(err error) error {
	if err == nil {
		return nil
	}
	var existing *OptimizationFailure
	if errors.As(err, &existing) && existing != nil && existing.Code.Valid() {
		return err
	}
	code := FailureCodeWorkerCrash
	var solverErr *SolverError
	var repositoryErr *repository.Error
	switch {
	case errors.As(err, &solverErr) && solverErr != nil && solverErr.Kind == SolverErrorInfeasible:
		code = FailureCodeSolverInfeasible
	case solverErr != nil && solverErr.Kind == SolverErrorTimeout:
		code = FailureCodeSolverTimeout
	case errors.Is(err, context.DeadlineExceeded):
		code = FailureCodeSolverTimeout
	case errors.As(err, &repositoryErr) && repositoryErr != nil && repositoryErr.Kind == repository.ErrorKindValidation:
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

// immutableMealSnapshot builds the generator's sole index and a detached,
// deterministic slice used by every model build.
// Implements DESIGN-004 SolutionValidator.
func immutableMealSnapshot(meals []repository.MealEntity, instrumentation *generationInstrumentation) (map[string]repository.MealEntity, []string, error) {
	detached := make([]repository.MealEntity, len(meals))
	for index, meal := range meals {
		meal.RecipeItems = append([]repository.RecipeIngredientEntity(nil), meal.RecipeItems...)
		meal.Classifications = append([]repository.ClassificationEntity(nil), meal.Classifications...)
		detached[index] = meal
	}
	indexed, err := mealIndex(detached)
	if err != nil {
		return nil, nil, err
	}
	if instrumentation != nil && instrumentation.indexBuilt != nil {
		instrumentation.indexBuilt()
	}
	orderedIDs := make([]string, 0, len(indexed))
	for itemID := range indexed {
		orderedIDs = append(orderedIDs, itemID)
	}
	sort.Strings(orderedIDs)
	return indexed, orderedIDs, nil
}

// quantityTolerance is the sole scale-aware numeric comparison tolerance.
// Implements DESIGN-004 SolutionValidator and DiversityPenalizer.
func quantityTolerance(values ...float64) float64 {
	scale := 1.0
	for _, value := range values {
		if magnitude := math.Abs(value); magnitude > scale {
			scale = magnitude
		}
	}
	return SolutionValidationEpsilon * scale
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
	return value >= target-margin-quantityTolerance(value, target-margin) && value <= target+margin+quantityTolerance(value, target+margin)
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

package optimization

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-004 DiversityPenalizer.
const (
	// MaxAlternativeCount is the public result ceiling required by SW-REQ-030.
	MaxAlternativeCount = 3
	// DefaultDiversityPenalty counts one base quantity unit (g or ml) of an
	// original meal in the lexicographic secondary objective.
	DefaultDiversityPenalty = 1
)

// DiversityPenalizer adds a soft objective penalty to meal IDs from the
// original diet. Original meals remain eligible unless explicitly excluded.
// Implements DESIGN-004 DiversityPenalizer for SW-REQ-023.
type DiversityPenalizer struct {
	OriginalMealIDs map[string]struct{}
	Penalty         float64
}

// NewDiversityPenalizer creates the default penalty policy for a request.
// Implements DESIGN-004 DiversityPenalizer.
func NewDiversityPenalizer(req DietOptimizationRequest) DiversityPenalizer {
	ids := make(map[string]struct{})
	for _, entry := range req.OriginalDiet.Entries {
		ids[entry.MealID.String()] = struct{}{}
	}
	return DiversityPenalizer{OriginalMealIDs: ids, Penalty: DefaultDiversityPenalty}
}

// Apply returns a copy of variables with the soft penalty attached to
// original-diet meal IDs. It never changes eligibility bounds.
// Implements DESIGN-004 DiversityPenalizer.
func (p DiversityPenalizer) Apply(variables []LPVariable) ([]LPVariable, error) {
	if !finite(p.Penalty) || p.Penalty < 0 {
		return nil, validationError("diversity penalty must be finite and non-negative")
	}
	result := make([]LPVariable, len(variables))
	copy(result, variables)
	for index := range result {
		if result[index].ItemID == "" {
			return nil, validationError("LP variable item ID is required")
		}
		result[index].DiversityPenalty = 0
		if _, ok := p.OriginalMealIDs[result[index].ItemID]; ok {
			result[index].DiversityPenalty = p.Penalty
		}
	}
	return result, nil
}

// AlternativeSolveFunc solves one already-built LP model. Keeping this
// boundary injectable lets the worker add the concrete solver independently.
// Implements DESIGN-004 DiversityPenalizer.
type AlternativeSolveFunc func(context.Context, LPModel, ObjectiveFunction) (LPSolution, error)

// LPSolution is the sparse meal-quantity assignment returned by one solve.
// Implements DESIGN-004 DiversityPenalizer.
type LPSolution = map[string]float64

// GenerateAlternatives repeatedly builds and solves a model, excluding one
// deterministic selected meal from each accepted solution. Duplicate meal
// sets are discarded and the caller-provided limit can never exceed three.
// Implements DESIGN-004 DiversityPenalizer for SW-REQ-023 and SW-REQ-030.
func GenerateAlternatives(ctx context.Context, req DietOptimizationRequest, meals []repository.MealEntity, limit int, solve AlternativeSolveFunc) ([]LPSolution, error) {
	limit, attemptBudget := alternativeGenerationLimits(limit)
	validator := newSolutionValidator(meals, generationInstrumentationFromContext(ctx))
	validated, err := generateAlternativePipeline(ctx, cloneOptimizationRequest(req), validator, limit, attemptBudget, solve)
	results := make([]LPSolution, len(validated))
	for index := range validated {
		results[index] = validated[index].solution
	}
	return results, err
}

// alternativeGenerationLimits caps the result count before deriving its
// attempt budget, making the multiplication bounded and overflow-safe.
// Implements DESIGN-004 DiversityPenalizer attempt policy.
func alternativeGenerationLimits(limit int) (int, int) {
	if limit <= 0 {
		return limit, 0
	}
	limit = min(limit, MaxAlternativeCount)
	return limit, limit * 3
}

// validatedAlternative couples one canonical assignment to its sole
// repository-derived publication projection.
// Implements DESIGN-004 DiversityPenalizer and SolutionValidator.
type validatedAlternative struct {
	solution    LPSolution
	alternative DietAlternative
}

// generateAlternativePipeline owns canonical validation and commits iteration
// state only after one repository projection has accepted the solver result.
// Attempt exhaustion returns already accepted alternatives without an error;
// any later solve, model, duplicate, or projection failure returns them with a
// safe terminal error.
// Implements DESIGN-004 DiversityPenalizer and SolutionValidator.
func generateAlternativePipeline(ctx context.Context, req DietOptimizationRequest, validator *SolutionValidator, limit, attemptBudget int, solve AlternativeSolveFunc) ([]validatedAlternative, error) {
	if limit <= 0 {
		return []validatedAlternative{}, nil
	}
	if limit > MaxAlternativeCount {
		limit = MaxAlternativeCount
	}
	if ctx == nil {
		return nil, safeOptimizationFailure(validationError("alternative context is required"))
	}
	if solve == nil {
		return nil, safeOptimizationFailure(validationError("alternative solver is required"))
	}
	if validator == nil || validator.snapshotErr != nil || len(validator.meals) == 0 {
		return nil, safeOptimizationFailure(validationError("repository meal snapshot is invalid"))
	}

	previous := []LPSolution{}
	seen := make(map[string]struct{})
	results := make([]validatedAlternative, 0, limit)
	for attempts := 0; len(results) < limit && attempts < attemptBudget; attempts++ {
		if err := ctx.Err(); err != nil {
			return results, safeOptimizationFailure(err)
		}
		model, err := buildConstraintsFromIndex(req, validator.meals, validator.orderedMealIDs, cloneSolutions(previous))
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		policy, err := BuildObjective(model.Variables)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		canonical, err := solveObjectivePolicy(ctx, model, policy, solve)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		alternative, err := validator.Validate(canonical, req)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		key := selectedMealSetKey(canonical)
		if _, duplicate := seen[key]; duplicate {
			return results, safeOptimizationFailure(validationError("solver returned a duplicate alternative"))
		}

		seen[key] = struct{}{}
		previous = append(previous, canonical)
		results = append(results, validatedAlternative{solution: canonical, alternative: alternative})
	}
	return results, nil
}

// solveObjectivePolicy performs a true lexicographic solve. The second solve
// cannot trade any primary calories for diversity because the first optimum is
// added as an equality constraint rather than a weighted sum.
// Implements DESIGN-004 ObjectiveFunction and DiversityPenalizer.
func solveObjectivePolicy(ctx context.Context, model LPModel, policy ObjectivePolicy, solve AlternativeSolveFunc) (LPSolution, error) {
	primary, err := solve(ctx, model, policy.Primary)
	if err != nil {
		return nil, err
	}
	primary, _, err = canonicalSolution(primary, model)
	if err != nil {
		return nil, err
	}
	if err := solutionSatisfiesModel(primary, model); err != nil {
		return nil, err
	}
	if !hasPositiveCoefficient(policy.Secondary) {
		return primary, nil
	}

	optimum, err := objectiveValueForSolution(policy.Primary, primary)
	if err != nil {
		return nil, err
	}
	secondaryModel := model
	secondaryModel.Constraints = append(append([]LPConstraint(nil), model.Constraints...), LPConstraint{
		Name:         "primary_calorie_optimum",
		LowerBound:   optimum,
		UpperBound:   optimum,
		Coefficients: policy.Primary.Coefficients,
	})
	secondary, err := solve(ctx, secondaryModel, policy.Secondary)
	if err != nil {
		return nil, err
	}
	secondary, _, err = canonicalSolution(secondary, secondaryModel)
	if err != nil {
		return nil, err
	}
	if err := solutionSatisfiesModel(secondary, secondaryModel); err != nil {
		return nil, err
	}
	return secondary, nil
}

// hasPositiveCoefficient reports whether an objective can affect a solve.
// Implements DESIGN-004 ObjectiveFunction validation.
func hasPositiveCoefficient(objective ObjectiveFunction) bool {
	for _, coefficient := range objective.Coefficients {
		if coefficient > 0 {
			return true
		}
	}
	return false
}

// objectiveValueForSolution evaluates a validated sparse assignment.
// Implements DESIGN-004 ObjectiveFunction validation.
func objectiveValueForSolution(objective ObjectiveFunction, solution LPSolution) (float64, error) {
	value := 0.0
	for _, itemID := range sortedCoefficientIDs(objective.Coefficients) {
		coefficient := objective.Coefficients[itemID]
		quantity := solution[itemID]
		if !finite(coefficient) || !finite(quantity) {
			return 0, validationError("objective value inputs must be finite")
		}
		value += coefficient * quantity
	}
	if !finite(value) {
		return 0, validationError("objective value must be finite")
	}
	return value, nil
}

// canonicalSolution normalizes one sparse solver assignment and identifies its
// selected meal set.
// Implements DESIGN-004 DiversityPenalizer.
func canonicalSolution(solution LPSolution, model LPModel) (LPSolution, string, error) {
	known := make(map[string]struct{}, len(model.Variables))
	for _, variable := range model.Variables {
		if variable.ItemID == "" {
			return nil, "", validationError("model variable item ID is required")
		}
		if _, duplicate := known[variable.ItemID]; duplicate {
			return nil, "", validationError("model contains duplicate variable: " + variable.ItemID)
		}
		known[variable.ItemID] = struct{}{}
	}
	return canonicalQuantities(solution, known)
}

// canonicalQuantities removes signed solver residue using the shared
// scale-aware tolerance and returns a deterministic selected-meal-set key.
// Implements DESIGN-004 DiversityPenalizer and SolutionValidator.
func canonicalQuantities[T any](solution LPSolution, known map[string]T) (LPSolution, string, error) {
	canonical := make(LPSolution, len(solution))
	for itemID, quantity := range solution {
		if _, ok := known[itemID]; !ok {
			return nil, "", validationError("solution contains unknown meal: " + itemID)
		}
		if !finite(quantity) || quantity < -quantityTolerance(quantity) {
			return nil, "", validationError("solution quantities must be finite and non-negative")
		}
		if math.Abs(quantity) > quantityTolerance(quantity) {
			canonical[itemID] = quantity
		}
	}
	if len(canonical) == 0 {
		return nil, "", validationError("solution must select a positive finite quantity")
	}
	return canonical, selectedMealSetKey(canonical), nil
}

// solutionSatisfiesModel rejects assignments outside any hard LP bound.
// Implements DESIGN-004 DiversityPenalizer.
func solutionSatisfiesModel(solution LPSolution, model LPModel) error {
	quantities := make(map[string]float64, len(model.Variables))
	for _, variable := range model.Variables {
		quantity := solution[variable.ItemID]
		if !finite(variable.LowerBound) || !finite(variable.UpperBound) || variable.LowerBound > variable.UpperBound {
			return validationError("model variable bounds are invalid")
		}
		tolerance := quantityTolerance(quantity, variable.LowerBound, variable.UpperBound)
		if quantity < variable.LowerBound-tolerance || quantity > variable.UpperBound+tolerance {
			return validationError("solution violates meal quantity bounds")
		}
		quantities[variable.ItemID] = quantity
	}
	for _, constraint := range model.Constraints {
		value := 0.0
		for _, itemID := range sortedCoefficientIDs(constraint.Coefficients) {
			coefficient := constraint.Coefficients[itemID]
			if _, ok := quantities[itemID]; !ok || !finite(coefficient) {
				return validationError("model constraint coefficients are invalid")
			}
			value += coefficient * quantities[itemID]
		}
		tolerance := quantityTolerance(value, constraint.LowerBound, constraint.UpperBound)
		if !finite(constraint.LowerBound) || !finite(constraint.UpperBound) || constraint.LowerBound > constraint.UpperBound || !finite(value) || value < constraint.LowerBound-tolerance || value > constraint.UpperBound+tolerance {
			return validationError("solution violates " + constraint.Name)
		}
	}
	return nil
}

// sortedCoefficientIDs returns deterministic objective/constraint order.
// Implements DESIGN-004 DiversityPenalizer and SolutionValidator.
func sortedCoefficientIDs(coefficients map[string]float64) []string {
	ids := make([]string, 0, len(coefficients))
	for itemID := range coefficients {
		ids = append(ids, itemID)
	}
	sort.Strings(ids)
	return ids
}

// selectedMealSetKey gives equivalent sparse assignments a deterministic key.
// Implements DESIGN-004 DiversityPenalizer.
func selectedMealSetKey(solution LPSolution) string {
	ids := make([]string, 0, len(solution))
	for itemID := range solution {
		ids = append(ids, itemID)
	}
	sort.Strings(ids)
	return strings.Join(ids, "\x00")
}

// cloneSolutions prevents iterative generation from mutating request state.
// Implements DESIGN-004 DiversityPenalizer.
func cloneSolutions(solutions []map[string]float64) []map[string]float64 {
	result := make([]map[string]float64, len(solutions))
	for index, solution := range solutions {
		result[index] = make(map[string]float64, len(solution))
		for itemID, quantity := range solution {
			result[index][itemID] = quantity
		}
	}
	return result
}

// cloneOptimizationRequest detaches caller-owned slices for one generation.
// Implements DESIGN-004 DiversityPenalizer.
func cloneOptimizationRequest(req DietOptimizationRequest) DietOptimizationRequest {
	req.OriginalDiet.Entries = append([]repository.SavedDietMealEntry(nil), req.OriginalDiet.Entries...)
	req.ExcludedMealIDs = append([]uuid.UUID(nil), req.ExcludedMealIDs...)
	return req
}

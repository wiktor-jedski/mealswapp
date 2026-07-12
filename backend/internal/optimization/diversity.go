package optimization

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-004 DiversityPenalizer.
const (
	// MaxAlternativeCount is the public result ceiling required by SW-REQ-030.
	MaxAlternativeCount = 3
	// DefaultDiversityPenalty is an additive calorie-equivalent objective weight.
	// It is deliberately finite and non-zero, while remaining a best-effort
	// preference rather than a hard exclusion of an original meal.
	DefaultDiversityPenalty    = 0.1
	diversityValidationEpsilon = 1e-8
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
	if len(req.OriginalDiet.Entries) > 0 {
		for _, entry := range req.OriginalDiet.Entries {
			if entry.MealID != uuid.Nil {
				ids[entry.MealID.String()] = struct{}{}
			}
		}
	} else {
		for _, meal := range req.OriginalMeals {
			if meal.MealID != uuid.Nil {
				ids[meal.MealID.String()] = struct{}{}
			}
		}
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

// GenerateAlternatives repeatedly builds and solves a model, adding a
// normalized overlap constraint for each accepted solution. Duplicate meal
// sets are discarded and the caller-provided limit can never exceed three.
// Implements DESIGN-004 DiversityPenalizer for SW-REQ-023 and SW-REQ-030.
func GenerateAlternatives(ctx context.Context, req DietOptimizationRequest, meals []repository.MealEntity, limit int, solve AlternativeSolveFunc) ([]LPSolution, error) {
	if limit <= 0 {
		return []LPSolution{}, nil
	}
	if limit > MaxAlternativeCount {
		limit = MaxAlternativeCount
	}
	if solve == nil {
		return nil, safeOptimizationFailure(validationError("alternative solver is required"))
	}

	previous := cloneSolutions(req.PreviousSolutions)
	seen := make(map[string]struct{}, len(previous))
	for _, solution := range previous {
		if len(solution) > 0 {
			seen[selectedMealSetKey(solution)] = struct{}{}
		}
	}
	results := make([]LPSolution, 0, limit)
	maxAttempts := limit * 3
	for attempts := 0; len(results) < limit && attempts < maxAttempts; attempts++ {
		if err := ctx.Err(); err != nil {
			return results, safeOptimizationFailure(err)
		}
		nextRequest := req
		nextRequest.PreviousSolutions = cloneSolutions(previous)
		model, err := BuildConstraints(nextRequest, meals)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		objective, err := BuildObjective(model.Variables)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		solution, err := solve(ctx, model, objective)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		if _, err := ValidateSolution(solution, req, meals); err != nil {
			return results, safeOptimizationFailure(err)
		}
		canonical, key, err := canonicalSolution(solution, model)
		if err != nil {
			return results, safeOptimizationFailure(err)
		}
		previous = append(previous, canonical)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		if err := solutionSatisfiesModel(canonical, model); err != nil {
			return results, safeOptimizationFailure(err)
		}
		seen[key] = struct{}{}
		results = append(results, canonical)
	}
	return results, nil
}

// canonicalSolution normalizes one sparse solver assignment and identifies its
// selected meal set.
// Implements DESIGN-004 DiversityPenalizer.
func canonicalSolution(solution LPSolution, model LPModel) (LPSolution, string, error) {
	known := make(map[string]struct{}, len(model.Variables))
	for _, variable := range model.Variables {
		known[variable.ItemID] = struct{}{}
	}
	canonical := make(LPSolution, len(solution))
	for itemID, quantity := range solution {
		if _, ok := known[itemID]; !ok {
			return nil, "", validationError("solution contains unknown meal: " + itemID)
		}
		if !finite(quantity) || quantity < 0 {
			return nil, "", validationError("solution quantities must be finite and non-negative")
		}
		if quantity > 0 {
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
		if quantity < variable.LowerBound-diversityValidationEpsilon || quantity > variable.UpperBound+diversityValidationEpsilon {
			return validationError("solution violates meal quantity bounds")
		}
		quantities[variable.ItemID] = quantity
	}
	for _, constraint := range model.Constraints {
		value := 0.0
		for itemID, coefficient := range constraint.Coefficients {
			value += coefficient * quantities[itemID]
		}
		if !finite(value) || value < constraint.LowerBound-diversityValidationEpsilon || value > constraint.UpperBound+diversityValidationEpsilon {
			return validationError("solution violates " + constraint.Name)
		}
	}
	return nil
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

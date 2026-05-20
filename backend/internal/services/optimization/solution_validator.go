package optimization

import (
	"errors"
	"math"
	"sort"
	"strings"
)

var ErrInvalidSolution = errors.New("optimization solution is invalid")

type MealQuantity struct {
	ItemID   string  `json:"itemId"`
	Quantity float64 `json:"quantity"`
}

type DietAlternative struct {
	Meals           []MealQuantity `json:"meals"`
	Macros          MacroTarget    `json:"macros"`
	Calories        float64        `json:"calories"`
	SimilarityScore float64        `json:"similarityScore"`
}

func ValidateSolution(solution LPSolution, request DietOptimizationRequest, variables []LPVariable) (DietAlternative, error) {
	variableByID := map[string]LPVariable{}
	for _, variable := range variables {
		variableByID[variable.ItemID] = variable
	}

	alternative := DietAlternative{}
	for itemID, quantity := range solution.Quantities {
		if quantity == 0 {
			continue
		}
		if !finiteNonNegative(quantity) {
			return DietAlternative{}, ErrInvalidSolution
		}
		if containsString(request.ExcludedIDs, itemID) {
			return DietAlternative{}, ErrInvalidSolution
		}
		variable, ok := variableByID[itemID]
		if !ok {
			return DietAlternative{}, ErrInvalidSolution
		}
		alternative.Meals = append(alternative.Meals, MealQuantity{ItemID: itemID, Quantity: quantity})
		alternative.Macros.Protein += variable.ProteinPerUnit * quantity
		alternative.Macros.Carbs += variable.CarbsPerUnit * quantity
		alternative.Macros.Fat += variable.FatPerUnit * quantity
		alternative.Calories += variable.CaloriesPerUnit * quantity
	}
	if len(alternative.Meals) == 0 || !finiteNonNegative(alternative.Calories) {
		return DietAlternative{}, ErrInvalidSolution
	}
	if !withinTolerance(alternative.Macros.Protein, request.TargetMacros.Protein, request.TolerancePercent) ||
		!withinTolerance(alternative.Macros.Carbs, request.TargetMacros.Carbs, request.TolerancePercent) ||
		!withinTolerance(alternative.Macros.Fat, request.TargetMacros.Fat, request.TolerancePercent) {
		return DietAlternative{}, ErrInvalidSolution
	}
	sort.Slice(alternative.Meals, func(i int, j int) bool {
		return alternative.Meals[i].ItemID < alternative.Meals[j].ItemID
	})
	alternative.SimilarityScore = 1
	return alternative, nil
}

func ValidateAlternatives(alternatives []DietAlternative, limit int) error {
	if limit <= 0 {
		limit = 3
	}
	if len(alternatives) > limit {
		return ErrInvalidSolution
	}
	seen := map[string]bool{}
	for _, alternative := range alternatives {
		key := alternativeKey(alternative)
		if seen[key] {
			return ErrInvalidSolution
		}
		seen[key] = true
	}
	return nil
}

func withinTolerance(value float64, target float64, tolerancePercent float64) bool {
	if target <= 0 || tolerancePercent < 0 {
		return false
	}
	tolerance := tolerancePercent / 100
	return value >= target*(1-tolerance)-0.000001 && value <= target*(1+tolerance)+0.000001
}

func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func alternativeKey(alternative DietAlternative) string {
	parts := make([]string, len(alternative.Meals))
	for i, meal := range alternative.Meals {
		parts[i] = meal.ItemID
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

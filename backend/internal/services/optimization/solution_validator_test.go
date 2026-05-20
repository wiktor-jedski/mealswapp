package optimization

import (
	"errors"
	"math"
	"testing"
)

func TestValidateSolutionBuildsDietAlternative(t *testing.T) {
	request := validatorRequest()
	solution := LPSolution{Status: "optimal", Quantities: map[string]float64{"beans": 1, "tofu": 1}}

	alternative, err := ValidateSolution(solution, request, validatorVariables())
	if err != nil {
		t.Fatal(err)
	}
	if len(alternative.Meals) != 2 || alternative.Macros.Protein != 35 || alternative.Macros.Carbs != 35 || alternative.Macros.Fat != 15 {
		t.Fatalf("unexpected alternative: %#v", alternative)
	}
	if alternative.Calories != 320 || alternative.SimilarityScore != 1 {
		t.Fatalf("unexpected calories/similarity: %#v", alternative)
	}
}

func TestValidateSolutionRejectsInvalidQuantitiesAndExcludedItems(t *testing.T) {
	request := validatorRequest()
	for _, solution := range []LPSolution{
		{Quantities: map[string]float64{"beans": math.NaN()}},
		{Quantities: map[string]float64{"beans": -1}},
		{Quantities: map[string]float64{"excluded": 1}},
	} {
		if _, err := ValidateSolution(solution, request, validatorVariables()); !errors.Is(err, ErrInvalidSolution) {
			t.Fatalf("expected invalid solution for %#v, got %v", solution, err)
		}
	}
}

func TestValidateSolutionRejectsMacroToleranceMiss(t *testing.T) {
	request := validatorRequest()
	solution := LPSolution{Quantities: map[string]float64{"beans": 1}}

	if _, err := ValidateSolution(solution, request, validatorVariables()); !errors.Is(err, ErrInvalidSolution) {
		t.Fatalf("expected macro tolerance miss, got %v", err)
	}
}

func TestValidateAlternativesRejectsDuplicatesAndOverLimit(t *testing.T) {
	alt := DietAlternative{Meals: []MealQuantity{{ItemID: "beans", Quantity: 1}, {ItemID: "tofu", Quantity: 1}}}
	duplicate := DietAlternative{Meals: []MealQuantity{{ItemID: "tofu", Quantity: 1}, {ItemID: "beans", Quantity: 1}}}
	if err := ValidateAlternatives([]DietAlternative{alt, duplicate}, 3); !errors.Is(err, ErrInvalidSolution) {
		t.Fatalf("expected duplicate alternatives rejected, got %v", err)
	}

	alternatives := []DietAlternative{
		{Meals: []MealQuantity{{ItemID: "a", Quantity: 1}}},
		{Meals: []MealQuantity{{ItemID: "b", Quantity: 1}}},
		{Meals: []MealQuantity{{ItemID: "c", Quantity: 1}}},
		{Meals: []MealQuantity{{ItemID: "d", Quantity: 1}}},
	}
	if err := ValidateAlternatives(alternatives, 3); !errors.Is(err, ErrInvalidSolution) {
		t.Fatalf("expected over-limit alternatives rejected, got %v", err)
	}
}

func validatorRequest() DietOptimizationRequest {
	return DietOptimizationRequest{
		OriginalMeals:    []MealInput{{ID: "original", Quantity: 1}},
		TargetMacros:     MacroTarget{Protein: 35, Carbs: 35, Fat: 15},
		ExcludedIDs:      []string{"excluded"},
		TolerancePercent: 1,
	}
}

func validatorVariables() []LPVariable {
	return []LPVariable{
		{ItemID: "beans", CaloriesPerUnit: 200, ProteinPerUnit: 20, CarbsPerUnit: 30, FatPerUnit: 5},
		{ItemID: "tofu", CaloriesPerUnit: 120, ProteinPerUnit: 15, CarbsPerUnit: 5, FatPerUnit: 10},
		{ItemID: "excluded", CaloriesPerUnit: 100, ProteinPerUnit: 35, CarbsPerUnit: 35, FatPerUnit: 15},
	}
}

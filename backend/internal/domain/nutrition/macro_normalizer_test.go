package nutrition

import (
	"errors"
	"testing"

	"mealswapp/backend/internal/domain/food"
)

func TestNormalizePerAmount(t *testing.T) {
	got, err := Normalize(MacroInput{
		Macros: food.MacroValues{
			ProteinGrams: 20,
			CarbsGrams:   40,
			FatGrams:     10,
		},
		Calories: 330,
		Basis:    BasisPerAmount,
		Amount:   200,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.MacrosPer100.ProteinGrams != 10 || got.MacrosPer100.CarbsGrams != 20 || got.MacrosPer100.FatGrams != 5 {
		t.Fatalf("unexpected normalized macros: %#v", got.MacrosPer100)
	}
	if got.CaloriesPer100 != 165 {
		t.Fatalf("expected 165 calories per 100, got %.3f", got.CaloriesPer100)
	}
}

func TestNormalizePerServing(t *testing.T) {
	got, err := Normalize(MacroInput{
		Macros: food.MacroValues{
			ProteinGrams: 8,
			CarbsGrams:   16,
			FatGrams:     4,
		},
		Calories:    132,
		Basis:       BasisPerServing,
		ServingSize: 80,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.MacrosPer100.ProteinGrams != 10 || got.CaloriesPer100 != 165 {
		t.Fatalf("unexpected serving normalization: %#v", got)
	}
}

func TestCheckCaloriesAllowsZeroValues(t *testing.T) {
	if err := CheckCalories(food.MacroValues{}, 0); err != nil {
		t.Fatalf("expected zero macros and zero calories to be valid, got %v", err)
	}
}

func TestCheckCaloriesRejectsMismatchesOutsideTolerance(t *testing.T) {
	err := CheckCalories(food.MacroValues{ProteinGrams: 10, CarbsGrams: 10, FatGrams: 10}, 400)
	if !errors.Is(err, ErrCalorieMismatch) {
		t.Fatalf("expected calorie mismatch, got %v", err)
	}
}

func TestNormalizeRejectsInvalidInput(t *testing.T) {
	if _, err := Normalize(MacroInput{Basis: BasisPerAmount, Amount: 0}); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected invalid amount, got %v", err)
	}

	if _, err := Normalize(MacroInput{Basis: "raw"}); !errors.Is(err, ErrInvalidBasis) {
		t.Fatalf("expected invalid basis, got %v", err)
	}

	if _, err := Normalize(MacroInput{
		Macros:   food.MacroValues{ProteinGrams: -1},
		Calories: 0,
		Basis:    BasisPer100,
	}); !errors.Is(err, food.ErrInvalidMacros) {
		t.Fatalf("expected invalid macros, got %v", err)
	}
}

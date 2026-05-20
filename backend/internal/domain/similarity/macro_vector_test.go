package similarity

import (
	"errors"
	"math"
	"testing"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/meal"
	"mealswapp/backend/internal/domain/recipe"

	"github.com/google/uuid"
)

func TestBuildFoodMacroVector(t *testing.T) {
	vector := BuildFoodMacroVector(food.FoodItemEntity{MacrosPer100: food.MacroValues{ProteinGrams: 10, CarbsGrams: 20, FatGrams: 5}})

	if vector != (MacroVector{Protein: 10, Carbs: 20, Fat: 5}) {
		t.Fatalf("unexpected food vector: %#v", vector)
	}
}

func TestBuildRecipeMacroVector(t *testing.T) {
	vector := BuildRecipeMacroVector(recipe.RecipeEntity{MacrosTotal: food.MacroValues{ProteinGrams: 30, CarbsGrams: 40, FatGrams: 10}})

	if vector != (MacroVector{Protein: 30, Carbs: 40, Fat: 10}) {
		t.Fatalf("unexpected recipe vector: %#v", vector)
	}
}

func TestBuildMealMacroVector(t *testing.T) {
	foodID := uuid.New()
	entity := meal.MealEntity{
		Items: []meal.MealItemEntity{{FoodItemID: foodID, Quantity: 150, Unit: meal.IngredientUnitGram}},
	}

	vector, err := BuildMealMacroVector(entity, map[uuid.UUID]MacroVector{
		foodID: {Protein: 10, Carbs: 20, Fat: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	if vector != (MacroVector{Protein: 15, Carbs: 30, Fat: 3}) {
		t.Fatalf("unexpected meal vector: %#v", vector)
	}
}

func TestNormalizeMacroVector(t *testing.T) {
	normalized, err := NormalizeMacroVector(MacroVector{Protein: 3, Carbs: 4, Fat: 0})
	if err != nil {
		t.Fatal(err)
	}

	if normalized.Magnitude != 5 || math.Abs(normalized.Protein-0.6) > 0.0001 || math.Abs(normalized.Carbs-0.8) > 0.0001 {
		t.Fatalf("unexpected normalized vector: %#v", normalized)
	}
}

func TestNormalizeRejectsZeroAndInvalidVectors(t *testing.T) {
	_, err := NormalizeMacroVector(MacroVector{})
	if !errors.Is(err, ErrZeroMacroVector) {
		t.Fatalf("expected zero vector error, got %v", err)
	}

	_, err = NormalizeMacroVector(MacroVector{Protein: -1, Carbs: 1, Fat: 1})
	if !errors.Is(err, ErrInvalidMacroVector) {
		t.Fatalf("expected invalid vector error, got %v", err)
	}
}

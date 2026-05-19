package recipe

import (
	"errors"
	"testing"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/meal"

	"github.com/google/uuid"
)

func TestRecipeValidationAcceptsValidRecipe(t *testing.T) {
	recipe := validRecipe()

	if err := recipe.Validate(); err != nil {
		t.Fatalf("expected valid recipe, got %v", err)
	}
}

func TestRecipeValidationRejectsMissingIngredients(t *testing.T) {
	recipe := validRecipe()
	recipe.Ingredients = nil

	if err := recipe.Validate(); !errors.Is(err, ErrMissingIngredients) {
		t.Fatalf("expected missing ingredients error, got %v", err)
	}
}

func TestRecipeValidationRejectsInvalidAggregate(t *testing.T) {
	recipe := validRecipe()
	recipe.MacrosTotal = food.MacroValues{ProteinGrams: -1}

	if err := recipe.Validate(); !errors.Is(err, ErrInvalidAggregate) {
		t.Fatalf("expected invalid aggregate error, got %v", err)
	}
}

func validRecipe() RecipeEntity {
	return RecipeEntity{
		UserID: uuid.New(),
		Name:   "Porridge",
		Ingredients: []RecipeIngredientEntity{
			{FoodItemID: uuid.New(), Quantity: 80, Unit: meal.IngredientUnitGram, Position: 0},
			{FoodItemID: uuid.New(), Quantity: 200, Unit: meal.IngredientUnitMilliliter, Position: 1},
		},
		CaloriesTotal: 395,
		MacrosTotal: food.MacroValues{
			ProteinGrams: 20.3,
			CarbsGrams:   63.0,
			FatGrams:     7.5,
		},
	}
}

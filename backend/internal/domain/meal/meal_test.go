package meal

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestMealValidationAcceptsRecipeWithMultipleItems(t *testing.T) {
	meal := validMeal()

	if err := meal.Validate(); err != nil {
		t.Fatalf("expected valid meal, got %v", err)
	}
}

func TestMealValidationRejectsMissingItems(t *testing.T) {
	meal := validMeal()
	meal.Items = nil

	if err := meal.Validate(); !errors.Is(err, ErrMissingItems) {
		t.Fatalf("expected missing items error, got %v", err)
	}
}

func TestMealValidationRejectsUnsupportedUnit(t *testing.T) {
	meal := validMeal()
	meal.Items[0].Unit = "cup"

	if err := meal.Validate(); !errors.Is(err, ErrUnsupportedItemUnit) {
		t.Fatalf("expected unsupported unit error, got %v", err)
	}
}

func TestMealValidationRejectsSingleMealWithMultipleItems(t *testing.T) {
	meal := validMeal()
	meal.Type = MealTypeSingle

	if err := meal.Validate(); !errors.Is(err, ErrSingleMealItemMismatch) {
		t.Fatalf("expected single meal mismatch error, got %v", err)
	}
}

func validMeal() MealEntity {
	return MealEntity{
		UserID: uuid.New(),
		Name:   "Breakfast bowl",
		Type:   MealTypeRecipe,
		Items: []MealItemEntity{
			{FoodItemID: uuid.New(), Quantity: 100, Unit: IngredientUnitGram, Position: 0},
			{FoodItemID: uuid.New(), Quantity: 50, Unit: IngredientUnitGram, Position: 1},
		},
	}
}

package similarity

import (
	"errors"
	"math"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/meal"
	"mealswapp/backend/internal/domain/recipe"

	"github.com/google/uuid"
)

var (
	ErrInvalidMacroVector = errors.New("macro vector values must be finite and non-negative")
	ErrZeroMacroVector    = errors.New("macro vector magnitude must be greater than zero")
	ErrMissingMealMacro   = errors.New("meal item macro vector is missing")
)

type MacroVector struct {
	Protein float64
	Carbs   float64
	Fat     float64
}

type NormalizedMacroVector struct {
	Protein   float64
	Carbs     float64
	Fat       float64
	Magnitude float64
}

func BuildFoodMacroVector(item food.FoodItemEntity) MacroVector {
	return fromMacroValues(item.MacrosPer100)
}

func BuildRecipeMacroVector(recipe recipe.RecipeEntity) MacroVector {
	return fromMacroValues(recipe.MacrosTotal)
}

func BuildMealMacroVector(entity meal.MealEntity, itemMacros map[uuid.UUID]MacroVector) (MacroVector, error) {
	var vector MacroVector
	for _, item := range entity.Items {
		itemVector, ok := itemMacros[item.FoodItemID]
		if !ok {
			return MacroVector{}, ErrMissingMealMacro
		}
		scale := item.Quantity
		if item.Unit == meal.IngredientUnitGram || item.Unit == meal.IngredientUnitMilliliter {
			scale = item.Quantity / 100
		}
		vector.Protein += itemVector.Protein * scale
		vector.Carbs += itemVector.Carbs * scale
		vector.Fat += itemVector.Fat * scale
	}
	return vector, nil
}

func NormalizeMacroVector(vector MacroVector) (NormalizedMacroVector, error) {
	if !valid(vector.Protein) || !valid(vector.Carbs) || !valid(vector.Fat) {
		return NormalizedMacroVector{}, ErrInvalidMacroVector
	}

	magnitude := math.Sqrt(vector.Protein*vector.Protein + vector.Carbs*vector.Carbs + vector.Fat*vector.Fat)
	if magnitude == 0 {
		return NormalizedMacroVector{}, ErrZeroMacroVector
	}

	return NormalizedMacroVector{
		Protein:   vector.Protein / magnitude,
		Carbs:     vector.Carbs / magnitude,
		Fat:       vector.Fat / magnitude,
		Magnitude: magnitude,
	}, nil
}

func fromMacroValues(macros food.MacroValues) MacroVector {
	return MacroVector{
		Protein: macros.ProteinGrams,
		Carbs:   macros.CarbsGrams,
		Fat:     macros.FatGrams,
	}
}

func valid(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

package nutrition

import (
	"errors"
	"math"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/units"
)

const CalorieTolerance = 0.20

type InputBasis string

const (
	BasisPer100     InputBasis = "per_100"
	BasisPerAmount  InputBasis = "per_amount"
	BasisPerServing InputBasis = "per_serving"
)

type MacroInput struct {
	Macros      food.MacroValues
	Calories    float64
	Basis       InputBasis
	Amount      float64
	ServingSize float64
}

type NormalizedMacros struct {
	MacrosPer100   food.MacroValues
	CaloriesPer100 float64
}

var (
	ErrInvalidBasis       = errors.New("invalid macro input basis")
	ErrInvalidAmount      = errors.New("amount must be greater than zero")
	ErrInvalidServingSize = errors.New("serving size must be greater than zero")
	ErrInvalidCalories    = errors.New("calories must be zero or greater")
	ErrCalorieMismatch    = errors.New("calories are inconsistent with macro energy")
)

func Normalize(input MacroInput) (NormalizedMacros, error) {
	if err := input.Macros.Validate(); err != nil {
		return NormalizedMacros{}, err
	}
	if invalidNumber(input.Calories) {
		return NormalizedMacros{}, ErrInvalidCalories
	}

	divisor, err := divisorFor(input)
	if err != nil {
		return NormalizedMacros{}, err
	}

	normalized := NormalizedMacros{
		MacrosPer100: food.MacroValues{
			ProteinGrams: units.Round(input.Macros.ProteinGrams / divisor),
			CarbsGrams:   units.Round(input.Macros.CarbsGrams / divisor),
			FatGrams:     units.Round(input.Macros.FatGrams / divisor),
		},
		CaloriesPer100: units.Round(input.Calories / divisor),
	}

	if err := CheckCalories(normalized.MacrosPer100, normalized.CaloriesPer100); err != nil {
		return NormalizedMacros{}, err
	}

	return normalized, nil
}

func CheckCalories(macros food.MacroValues, calories float64) error {
	if err := macros.Validate(); err != nil {
		return err
	}
	if invalidNumber(calories) {
		return ErrInvalidCalories
	}

	expected := macros.ProteinGrams*4 + macros.CarbsGrams*4 + macros.FatGrams*9
	if expected == 0 && calories == 0 {
		return nil
	}
	if expected == 0 && calories > 0 {
		return ErrCalorieMismatch
	}

	diffRatio := math.Abs(calories-expected) / expected
	if diffRatio > CalorieTolerance {
		return ErrCalorieMismatch
	}

	return nil
}

func divisorFor(input MacroInput) (float64, error) {
	switch input.Basis {
	case BasisPer100:
		return 1, nil
	case BasisPerAmount:
		if input.Amount <= 0 || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
			return 0, ErrInvalidAmount
		}
		return input.Amount / 100, nil
	case BasisPerServing:
		if input.ServingSize <= 0 || math.IsNaN(input.ServingSize) || math.IsInf(input.ServingSize, 0) {
			return 0, ErrInvalidServingSize
		}
		return input.ServingSize / 100, nil
	default:
		return 0, ErrInvalidBasis
	}
}

func invalidNumber(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0) || value < 0
}

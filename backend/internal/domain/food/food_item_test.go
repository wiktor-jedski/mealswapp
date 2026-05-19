package food

import (
	"errors"
	"math"
	"testing"
)

func TestFoodItemValidationAcceptsValidItem(t *testing.T) {
	item := validFoodItem()

	if err := item.Validate(); err != nil {
		t.Fatalf("expected valid food item, got %v", err)
	}
}

func TestFoodItemValidationRejectsMissingName(t *testing.T) {
	item := validFoodItem()
	item.Name = "   "

	if err := item.Validate(); !errors.Is(err, ErrMissingName) {
		t.Fatalf("expected missing name error, got %v", err)
	}
}

func TestFoodItemValidationRejectsInvalidMacros(t *testing.T) {
	cases := map[string]MacroValues{
		"negative protein": {ProteinGrams: -1, CarbsGrams: 10, FatGrams: 2},
		"nan carbs":        {ProteinGrams: 1, CarbsGrams: math.NaN(), FatGrams: 2},
		"infinite fat":     {ProteinGrams: 1, CarbsGrams: 10, FatGrams: math.Inf(1)},
	}

	for name, macros := range cases {
		t.Run(name, func(t *testing.T) {
			item := validFoodItem()
			item.MacrosPer100 = macros

			if err := item.Validate(); !errors.Is(err, ErrInvalidMacros) {
				t.Fatalf("expected invalid macros error, got %v", err)
			}
		})
	}
}

func TestFoodItemValidationRejectsUnsupportedServingUnit(t *testing.T) {
	item := validFoodItem()
	item.ServingUnit = "cup"

	if err := item.Validate(); !errors.Is(err, ErrUnsupportedServingUnit) {
		t.Fatalf("expected unsupported serving unit error, got %v", err)
	}
}

func TestFoodItemValidationRejectsInvalidMicronutrients(t *testing.T) {
	item := validFoodItem()
	item.Micros = map[string]float64{"Sodium": -1}

	if err := item.Validate(); !errors.Is(err, ErrInvalidMicronutrients) {
		t.Fatalf("expected invalid micronutrients error, got %v", err)
	}
}

func validFoodItem() FoodItemEntity {
	return FoodItemEntity{
		Name:           "Greek yogurt",
		PhysicalState:  PhysicalStateSolid,
		ServingUnit:    ServingUnitGram,
		ServingSize:    100,
		CaloriesPer100: 59,
		MacrosPer100: MacroValues{
			ProteinGrams: 10,
			CarbsGrams:   3.6,
			FatGrams:     0.4,
		},
		Micros: map[string]float64{
			"Calcium": 110,
		},
		PrepTimeMinutes:        0,
		AverageUnitWeightGrams: 100,
	}
}

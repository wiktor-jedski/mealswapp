package externaldata

import (
	"errors"
	"testing"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/micronutrient"
)

func TestNormalizeExternalRecordUSDAFixture(t *testing.T) {
	record := ExternalFoodRecord{
		Provider:    ProviderUSDA,
		ExternalID:  "1101",
		Name:        " Cheddar Cheese ",
		ServingSize: ptrFloat(28),
		ServingUnit: "g",
		Nutrients: map[string]float64{
			"Protein":                     24.9,
			"Carbohydrate, by difference": 1.3,
			"Total lipid (fat)":           33.1,
			"Calories":                    403,
			"Calcium":                     710,
			"Unsupported vitamin x":       4,
		},
		ImageURL: "https://example.test/cheddar.jpg",
	}

	candidate, err := NormalizeExternalRecord(record, testVocabulary())
	if err != nil {
		t.Fatalf("unexpected normalization error: %v", err)
	}

	if candidate.Provider != ProviderUSDA || candidate.ExternalID != "1101" || candidate.Name != "Cheddar Cheese" {
		t.Fatalf("unexpected identity: %#v", candidate)
	}
	if candidate.PhysicalState != food.PhysicalStateSolid || candidate.ServingUnit != "gram" {
		t.Fatalf("unexpected physical state or serving unit: %#v", candidate)
	}
	if candidate.MacrosPer100.ProteinGrams != 24.9 || candidate.MacrosPer100.CarbsGrams != 1.3 || candidate.MacrosPer100.FatGrams != 33.1 {
		t.Fatalf("unexpected macros: %#v", candidate.MacrosPer100)
	}
	if candidate.Micros["Calcium"] != 710 {
		t.Fatalf("expected calcium to normalize, got %#v", candidate.Micros)
	}
	if !hasWarning(candidate.Warnings, "rejected_nutrient") {
		t.Fatalf("expected unsupported micronutrient warning, got %#v", candidate.Warnings)
	}
}

func TestNormalizeExternalRecordOpenFoodFactsFixture(t *testing.T) {
	record := ExternalFoodRecord{
		Provider:    ProviderOpenFoodFacts,
		ExternalID:  "737628064502",
		Name:        "Organic Tofu",
		ServingSize: ptrFloat(85),
		ServingUnit: "g",
		Nutrients: map[string]float64{
			"proteins_100g":      12.3,
			"carbohydrates_100g": 1.7,
			"fat_100g":           6.1,
			"energy-kcal_100g":   111,
			"iron_100g":          2.4,
		},
	}

	candidate, err := NormalizeExternalRecord(record, testVocabulary())
	if err != nil {
		t.Fatalf("unexpected normalization error: %v", err)
	}

	if candidate.MacrosPer100.ProteinGrams != 12.3 || candidate.CaloriesPer100 != 111 {
		t.Fatalf("unexpected normalized macros: %#v", candidate)
	}
	if candidate.Micros["Iron"] != 2.4 {
		t.Fatalf("expected iron to normalize, got %#v", candidate.Micros)
	}
	if !hasWarning(candidate.Warnings, "missing_image") {
		t.Fatalf("expected missing image warning, got %#v", candidate.Warnings)
	}
}

func TestNormalizeExternalRecordConvertsServingBasedValues(t *testing.T) {
	record := ExternalFoodRecord{
		Provider:    ProviderOpenFoodFacts,
		ExternalID:  "serving-1",
		Name:        "Serving Based Bar",
		ServingSize: ptrFloat(50),
		ServingUnit: "g",
		Nutrients: map[string]float64{
			"protein":       10,
			"carbohydrates": 20,
			"fat":           5,
			"calories":      165,
			"sodium":        120,
		},
	}

	candidate, err := NormalizeExternalRecord(record, testVocabulary())
	if err != nil {
		t.Fatalf("unexpected normalization error: %v", err)
	}

	if candidate.MacrosPer100.ProteinGrams != 20 || candidate.MacrosPer100.CarbsGrams != 40 || candidate.MacrosPer100.FatGrams != 10 {
		t.Fatalf("expected serving values scaled to per 100g, got %#v", candidate.MacrosPer100)
	}
	if candidate.Micros["Sodium"] != 240 {
		t.Fatalf("expected sodium scaled to per 100g, got %#v", candidate.Micros)
	}
}

func TestNormalizeExternalRecordWarnsOnCalorieMismatch(t *testing.T) {
	record := ExternalFoodRecord{
		Provider:   ProviderUSDA,
		ExternalID: "mismatch-1",
		Name:       "Mismatch",
		Nutrients: map[string]float64{
			"protein":       10,
			"carbohydrates": 10,
			"fat":           10,
			"calories":      400,
		},
	}

	candidate, err := NormalizeExternalRecord(record, testVocabulary())
	if err != nil {
		t.Fatalf("expected mismatch to return candidate with warning, got %v", err)
	}
	if !hasWarning(candidate.Warnings, "calorie_mismatch") {
		t.Fatalf("expected calorie mismatch warning, got %#v", candidate.Warnings)
	}
}

func TestNormalizeExternalRecordRejectsMissingRequiredFields(t *testing.T) {
	_, err := NormalizeExternalRecord(ExternalFoodRecord{Provider: ProviderUSDA, ExternalID: "1", Name: "No Macros"}, testVocabulary())
	if !errors.Is(err, ErrMissingRequiredMacros) {
		t.Fatalf("expected missing macros error, got %v", err)
	}

	_, err = NormalizeExternalRecord(ExternalFoodRecord{Provider: ProviderUSDA, Name: "No ID"}, testVocabulary())
	if !errors.Is(err, ErrMissingExternalIdentity) {
		t.Fatalf("expected missing identity error, got %v", err)
	}
}

func testVocabulary() []micronutrient.Entry {
	return []micronutrient.Entry{
		{Key: "Calcium", DisplayName: "Calcium", Unit: micronutrient.UnitMilligram, Active: true},
		{Key: "Iron", DisplayName: "Iron", Unit: micronutrient.UnitMilligram, Active: true},
		{Key: "Sodium", DisplayName: "Sodium", Unit: micronutrient.UnitMilligram, Active: true},
	}
}

func ptrFloat(value float64) *float64 {
	return &value
}

func hasWarning(warnings []ExternalDataWarning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

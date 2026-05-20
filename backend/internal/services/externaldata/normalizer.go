package externaldata

import (
	"errors"
	"fmt"
	"strings"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/micronutrient"
	"mealswapp/backend/internal/domain/nutrition"
	"mealswapp/backend/internal/domain/units"
)

var (
	ErrMissingExternalIdentity = errors.New("external record identity is required")
	ErrMissingExternalName     = errors.New("external record name is required")
	ErrMissingRequiredMacros   = errors.New("external record is missing required macros")
)

func NormalizeExternalRecord(record ExternalFoodRecord, vocabulary []micronutrient.Entry) (NormalizedFoodCandidate, error) {
	provider := record.Provider
	externalID := strings.TrimSpace(record.ExternalID)
	if provider == "" || externalID == "" {
		return NormalizedFoodCandidate{}, ErrMissingExternalIdentity
	}
	name := strings.TrimSpace(record.Name)
	if name == "" {
		return NormalizedFoodCandidate{}, ErrMissingExternalName
	}

	macros, calories, warnings, err := ConvertNutrientsToPer100(record, vocabulary)
	if err != nil {
		return NormalizedFoodCandidate{}, err
	}
	if record.ImageURL == "" {
		warnings = append(warnings, warning(record, "missing_image", "External record does not include an image URL"))
	}

	servingSize := record.ServingSize
	if servingSize != nil && *servingSize <= 0 {
		warnings = append(warnings, warning(record, "invalid_serving_size", "External serving size was ignored because it is not positive"))
		servingSize = nil
	}

	return NormalizedFoodCandidate{
		Provider:       provider,
		ExternalID:     externalID,
		Name:           name,
		PhysicalState:  inferPhysicalState(record.ServingUnit),
		MacrosPer100:   macros.MacrosPer100,
		CaloriesPer100: macros.CaloriesPer100,
		Micros:         calories.Micros,
		ServingSize:    servingSize,
		ServingUnit:    normalizeServingUnit(record.ServingUnit),
		ImageURL:       strings.TrimSpace(record.ImageURL),
		Warnings:       warnings,
	}, nil
}

type convertedNutrients struct {
	MacrosPer100   food.MacroValues
	CaloriesPer100 float64
	Micros         map[string]float64
}

func ConvertNutrientsToPer100(record ExternalFoodRecord, vocabulary []micronutrient.Entry) (nutrition.NormalizedMacros, convertedNutrients, []ExternalDataWarning, error) {
	warnings := []ExternalDataWarning{}
	macros, calories, missing := extractMacros(record.Nutrients)
	for _, key := range missing {
		warnings = append(warnings, warning(record, "missing_macro", fmt.Sprintf("Missing %s value", key)))
	}
	if len(missing) > 0 {
		return nutrition.NormalizedMacros{}, convertedNutrients{}, warnings, ErrMissingRequiredMacros
	}

	basis, amount := nutrientBasis(record)
	normalized, err := nutrition.Normalize(nutrition.MacroInput{
		Macros:   macros,
		Calories: calories,
		Basis:    basis,
		Amount:   amount,
	})
	if err != nil {
		if errors.Is(err, nutrition.ErrCalorieMismatch) {
			warnings = append(warnings, warning(record, "calorie_mismatch", "Calories do not match macro-derived energy within tolerance"))
			normalized = normalizeWithoutCalorieCheck(macros, calories, amount)
		} else {
			return nutrition.NormalizedMacros{}, convertedNutrients{}, warnings, err
		}
	}

	micros, microWarnings := extractMicros(record, vocabulary, amount)
	warnings = append(warnings, microWarnings...)

	return normalized, convertedNutrients{
		MacrosPer100:   normalized.MacrosPer100,
		CaloriesPer100: normalized.CaloriesPer100,
		Micros:         micros,
	}, warnings, nil
}

func extractMacros(values map[string]float64) (food.MacroValues, float64, []string) {
	protein, okProtein := firstNutrient(values, "protein", "proteins", "protein_100g", "proteins_100g")
	carbs, okCarbs := firstNutrient(values, "carbohydrate, by difference", "carbohydrate", "carbohydrates", "carbohydrates_100g")
	fat, okFat := firstNutrient(values, "total lipid (fat)", "fat", "fat_100g")
	calories, _ := firstNutrient(values, "energy-kcal_100g", "energy-kcal", "energy kcal", "calories", "energy")

	missing := []string{}
	if !okProtein {
		missing = append(missing, "protein")
	}
	if !okCarbs {
		missing = append(missing, "carbohydrates")
	}
	if !okFat {
		missing = append(missing, "fat")
	}
	if calories == 0 && len(missing) == 0 {
		calories = protein*4 + carbs*4 + fat*9
	}
	return food.MacroValues{ProteinGrams: protein, CarbsGrams: carbs, FatGrams: fat}, calories, missing
}

func extractMicros(record ExternalFoodRecord, vocabulary []micronutrient.Entry, amount float64) (map[string]float64, []ExternalDataWarning) {
	micros := map[string]float64{}
	warnings := []ExternalDataWarning{}
	acceptedAliases := map[string]struct{}{}
	for _, entry := range vocabulary {
		if !entry.Active {
			continue
		}
		aliases := nutrientAliases(entry.Key)
		for _, alias := range aliases {
			acceptedAliases[strings.ToLower(strings.TrimSpace(alias))] = struct{}{}
		}
		if value, ok := firstNutrient(record.Nutrients, aliases...); ok {
			micros[entry.Key] = units.Round(value / (amount / 100))
		}
	}
	for key := range record.Nutrients {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if _, accepted := acceptedAliases[normalized]; !accepted && looksLikeKnownRejectedNutrient(key) {
			warnings = append(warnings, warning(record, "rejected_nutrient", fmt.Sprintf("Nutrient %q is not in the active vocabulary", key)))
		}
	}
	return micros, warnings
}

func firstNutrient(values map[string]float64, names ...string) (float64, bool) {
	normalized := make(map[string]float64, len(values))
	for key, value := range values {
		normalized[strings.ToLower(strings.TrimSpace(key))] = value
	}
	for _, name := range names {
		if value, ok := normalized[strings.ToLower(strings.TrimSpace(name))]; ok {
			return value, true
		}
	}
	return 0, false
}

func nutrientBasis(record ExternalFoodRecord) (nutrition.InputBasis, float64) {
	if record.Provider == ProviderUSDA {
		return nutrition.BasisPer100, 100
	}
	if hasPer100Nutrients(record.Nutrients) {
		return nutrition.BasisPer100, 100
	}
	if record.ServingSize != nil && *record.ServingSize > 0 {
		return nutrition.BasisPerAmount, *record.ServingSize
	}
	return nutrition.BasisPer100, 100
}

func hasPer100Nutrients(values map[string]float64) bool {
	for key := range values {
		if strings.Contains(strings.ToLower(key), "_100g") || strings.Contains(strings.ToLower(key), "_100ml") {
			return true
		}
	}
	return false
}

func normalizeWithoutCalorieCheck(macros food.MacroValues, calories float64, amount float64) nutrition.NormalizedMacros {
	divisor := amount / 100
	if divisor <= 0 {
		divisor = 1
	}
	return nutrition.NormalizedMacros{
		MacrosPer100: food.MacroValues{
			ProteinGrams: units.Round(macros.ProteinGrams / divisor),
			CarbsGrams:   units.Round(macros.CarbsGrams / divisor),
			FatGrams:     units.Round(macros.FatGrams / divisor),
		},
		CaloriesPer100: units.Round(calories / divisor),
	}
}

func normalizeServingUnit(unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "g", "gram", "grams":
		return string(food.ServingUnitGram)
	case "ml", "milliliter", "milliliters":
		return string(food.ServingUnitMilliliter)
	case "serving", "servings":
		return string(food.ServingUnitServing)
	case "piece", "pieces":
		return string(food.ServingUnitPiece)
	default:
		return ""
	}
}

func inferPhysicalState(servingUnit string) food.PhysicalState {
	if normalizeServingUnit(servingUnit) == string(food.ServingUnitMilliliter) {
		return food.PhysicalStateLiquid
	}
	return food.PhysicalStateSolid
}

func nutrientAliases(key string) []string {
	switch key {
	case "Calcium":
		return []string{"calcium", "calcium_100g"}
	case "Iron":
		return []string{"iron", "iron_100g"}
	case "Potassium":
		return []string{"potassium", "potassium_100g"}
	case "Sodium":
		return []string{"sodium", "sodium_100g", "salt_100g"}
	case "VitaminA":
		return []string{"vitamin a", "vitamin-a_100g", "vitamin_a"}
	case "VitaminC":
		return []string{"vitamin c", "vitamin-c_100g", "vitamin_c"}
	case "VitaminD":
		return []string{"vitamin d", "vitamin-d_100g", "vitamin_d"}
	default:
		return []string{key, strings.ToLower(key)}
	}
}

func looksLikeKnownRejectedNutrient(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(normalized, "vitamin") || strings.Contains(normalized, "calcium") || strings.Contains(normalized, "iron") || strings.Contains(normalized, "sodium") || strings.Contains(normalized, "potassium")
}

func warning(record ExternalFoodRecord, code string, message string) ExternalDataWarning {
	return ExternalDataWarning{Provider: record.Provider, ExternalID: strings.TrimSpace(record.ExternalID), Code: code, Message: message}
}

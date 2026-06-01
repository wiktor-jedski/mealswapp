package repository

import "math"

// NormalizeMacros converts macro values for a quantity into the per-100 storage basis.
// Implements DESIGN-005 MacroNormalizer.
func NormalizeMacros(value MacroValues, quantity float64, state PhysicalState) (MacroValues, error) {
	if err := ValidatePhysicalState(state); err != nil {
		return MacroValues{}, err
	}
	if quantity <= 0 {
		return MacroValues{}, validationError("quantity must be positive")
	}
	if err := ValidateMacros(value); err != nil {
		return MacroValues{}, err
	}
	normalized := ScaleMacros(value, 100, quantity)
	if err := ValidateMacrosPer100(normalized, state); err != nil {
		return MacroValues{}, err
	}
	return normalized, nil
}

// ValidatePhysicalState checks supported storage bases.
// Implements DESIGN-005 MacroNormalizer.
func ValidatePhysicalState(state PhysicalState) error {
	switch state {
	case PhysicalStateSolid, PhysicalStateLiquid:
		return nil
	default:
		return validationError("physical state must be solid or liquid")
	}
}

// ValidateMacros rejects negative or non-finite macro values.
// Implements DESIGN-005 MacroNormalizer.
func ValidateMacros(value MacroValues) error {
	if invalidNumber(value.Protein) || invalidNumber(value.Carbohydrates) || invalidNumber(value.Fat) {
		return validationError("macro values must be finite")
	}
	if value.Protein < 0 || value.Carbohydrates < 0 || value.Fat < 0 {
		return validationError("macro values cannot be negative")
	}
	return nil
}

// ValidateMacrosPer100 checks macro invariants for the persisted per-100 storage basis.
// Implements DESIGN-005 MacroNormalizer.
func ValidateMacrosPer100(value MacroValues, state PhysicalState) error {
	if err := ValidatePhysicalState(state); err != nil {
		return err
	}
	if err := ValidateMacros(value); err != nil {
		return err
	}
	if state == PhysicalStateSolid && value.Protein+value.Carbohydrates+value.Fat > 100 {
		return validationError("solid macro values per 100 g cannot exceed 100 g")
	}
	return nil
}

// ValidateMicronutrientKeys ensures all provided keys are active canonical vocabulary entries.
// Implements DESIGN-005 MicronutrientVocabulary.
func ValidateMicronutrientKeys(values MicroValues, vocabulary []MicronutrientVocabularyEntry) error {
	active := make(map[string]struct{}, len(vocabulary))
	for _, entry := range vocabulary {
		if entry.Active {
			active[entry.Key] = struct{}{}
		}
	}
	for key := range values {
		if _, ok := active[key]; !ok {
			return NewError(ErrorKindInvalidMicronutrientKey, "micronutrient key is not active: "+key, nil)
		}
	}
	return nil
}

// ScaleMacros scales macro values by quantity over basis.
// Implements DESIGN-005 MacroNormalizer.
func ScaleMacros(base MacroValues, quantity float64, basis float64) MacroValues {
	if basis == 0 {
		return MacroValues{}
	}
	factor := quantity / basis
	return MacroValues{
		Protein:       round4(base.Protein * factor),
		Carbohydrates: round4(base.Carbohydrates * factor),
		Fat:           round4(base.Fat * factor),
	}
}

// invalidNumber reports whether a numeric value is not finite.
// Implements DESIGN-005 MacroNormalizer.
func invalidNumber(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0)
}

// round4 rounds a numeric value to four decimal places.
// Implements DESIGN-005 MacroNormalizer.
func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

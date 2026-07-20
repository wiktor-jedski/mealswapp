package repository

// Implements DESIGN-005 UnitConverter.
const (
	gramsPerOunce            = 28.349523125
	millilitersPerFluidOunce = 29.5735295625
)

// ConvertUnit converts between supported metric and imperial units.
// Implements DESIGN-005 UnitConverter.
func ConvertUnit(value float64, fromUnit string, toUnit string) (float64, error) {
	if invalidNumber(value) {
		return 0, unitConversionError("value must be finite")
	}
	if value < 0 {
		return 0, unitConversionError("value cannot be negative")
	}
	if ValidateQuantityUnit(fromUnit) != nil || ValidateQuantityUnit(toUnit) != nil {
		return 0, unitConversionError("unsupported unit conversion from %q to %q", fromUnit, toUnit)
	}
	if fromUnit == toUnit {
		return value, nil
	}

	var converted float64
	switch {
	case fromUnit == "g" && toUnit == "oz":
		converted = round4(value / gramsPerOunce)
	case fromUnit == "oz" && toUnit == "g":
		converted = round4(value * gramsPerOunce)
	case fromUnit == "ml" && toUnit == "fl_oz":
		converted = round4(value / millilitersPerFluidOunce)
	case fromUnit == "fl_oz" && toUnit == "ml":
		converted = round4(value * millilitersPerFluidOunce)
	default:
		return 0, unitConversionError("unsupported unit conversion from %q to %q", fromUnit, toUnit)
	}
	if invalidNumber(converted) {
		return 0, unitConversionError("converted value must be finite")
	}
	return converted, nil
}

// ConvertRecipeServingToBase converts recipe serving counts using the ingredient's matching serving measure.
// Implements DESIGN-005 UnitConverter recipe/per-unit boundary and SW-REQ-036.
func ConvertRecipeServingToBase(servings float64, averageUnitWeightGrams float64, averageServingVolumeMilliliters float64, state PhysicalState) (float64, string, error) {
	if invalidNumber(servings) {
		return 0, "", unitConversionError("servings must be finite")
	}
	if invalidNumber(averageUnitWeightGrams) || invalidNumber(averageServingVolumeMilliliters) {
		return 0, "", unitConversionError("serving measures must be finite")
	}
	if servings < 0 {
		return 0, "", unitConversionError("servings cannot be negative")
	}
	if err := ValidatePhysicalState(state); err != nil {
		return 0, "", err
	}
	if state == PhysicalStateLiquid {
		if averageServingVolumeMilliliters <= 0 {
			return 0, "", unitConversionError("average serving volume must be positive for liquid serving conversion")
		}
		quantity := round4(servings * averageServingVolumeMilliliters)
		if invalidNumber(quantity) {
			return 0, "", unitConversionError("serving conversion result must be finite")
		}
		return quantity, "ml", nil
	}
	if averageUnitWeightGrams <= 0 {
		return 0, "", unitConversionError("average unit weight must be positive for solid serving conversion")
	}
	quantity := round4(servings * averageUnitWeightGrams)
	if invalidNumber(quantity) {
		return 0, "", unitConversionError("serving conversion result must be finite")
	}
	return quantity, "g", nil
}

// ValidateQuantityUnit accepts the canonical mass and volume quantity vocabulary.
// Implements DESIGN-005 UnitConverter.
func ValidateQuantityUnit(unit string) error {
	switch unit {
	case "g", "ml", "oz", "fl_oz":
		return nil
	default:
		return unitConversionError("unsupported quantity unit %q", unit)
	}
}

// ValidateRecipeIngredientUnit accepts physical units plus serving at the recipe/per-unit boundary.
// Implements DESIGN-005 UnitConverter recipe ingredient rules and SW-REQ-036.
func ValidateRecipeIngredientUnit(unit string, state PhysicalState) error {
	if err := ValidatePhysicalState(state); err != nil {
		return err
	}
	if unit == "serving" {
		return nil
	}
	if err := ValidateQuantityUnit(unit); err != nil {
		return err
	}
	if state == PhysicalStateSolid && (unit == "ml" || unit == "fl_oz") {
		return unitConversionError("unit %q requires a liquid ingredient", unit)
	}
	if state == PhysicalStateLiquid && (unit == "g" || unit == "oz") {
		return unitConversionError("unit %q requires a solid ingredient", unit)
	}
	return nil
}

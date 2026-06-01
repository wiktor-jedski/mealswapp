package repository

// Implements DESIGN-005 UnitConverter.
const (
	gramsPerOunce            = 28.349523125
	millilitersPerFluidOunce = 29.5735295625
)

// ConvertUnit converts between supported metric and imperial units.
// Implements DESIGN-005 UnitConverter.
func ConvertUnit(value float64, fromUnit string, toUnit string) (float64, error) {
	if value < 0 {
		return 0, unitConversionError("value cannot be negative")
	}
	if !validUnit(fromUnit) || !validUnit(toUnit) {
		return 0, unitConversionError("unsupported unit conversion from %q to %q", fromUnit, toUnit)
	}
	if fromUnit == toUnit {
		return value, nil
	}

	switch {
	case fromUnit == "g" && toUnit == "oz":
		return round4(value / gramsPerOunce), nil
	case fromUnit == "oz" && toUnit == "g":
		return round4(value * gramsPerOunce), nil
	case fromUnit == "ml" && toUnit == "fl_oz":
		return round4(value / millilitersPerFluidOunce), nil
	case fromUnit == "fl_oz" && toUnit == "ml":
		return round4(value * millilitersPerFluidOunce), nil
	default:
		return 0, unitConversionError("unsupported unit conversion from %q to %q", fromUnit, toUnit)
	}
}

// ConvertServingToBase converts serving counts to grams or milliliters with the matching serving measure.
// Implements DESIGN-005 UnitConverter.
func ConvertServingToBase(servings float64, averageUnitWeightGrams float64, averageServingVolumeMilliliters float64, state PhysicalState) (float64, string, error) {
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
		return round4(servings * averageServingVolumeMilliliters), "ml", nil
	}
	if averageUnitWeightGrams <= 0 {
		return 0, "", unitConversionError("average unit weight must be positive for solid serving conversion")
	}
	return round4(servings * averageUnitWeightGrams), "g", nil
}

// validUnit reports whether unit is a canonical repository unit.
// Implements DESIGN-005 UnitConverter.
func validUnit(unit string) bool {
	return unit == "g" || unit == "ml" || unit == "oz" || unit == "fl_oz" || unit == "serving"
}

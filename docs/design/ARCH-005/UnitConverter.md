# UnitConverter

**Traceability:** ARCH-005

## 1. Data Structures & Types

```go
package repository

type PhysicalState string

const (
	PhysicalStateSolid  PhysicalState = "solid"
	PhysicalStateLiquid PhysicalState = "liquid"
)

type UnitSystem string

const (
	UnitSystemMetric    UnitSystem = "metric"
	UnitSystemImperial  UnitSystem = "imperial"
)

type UnitType string

const (
	UnitTypeWeight    UnitType = "weight"
	UnitTypeVolume    UnitType = "volume"
)

type ConversionResult struct {
	Value     float64
	Unit      string
	UnitType  UnitType
	Precision int
}

type MacroValues struct {
	Protein float64
	Carbs   float64
	Fat     float64
}

type NormalizedMacros struct {
	Per100g  MacroValues
	Per100ml MacroValues
}

type UnitConverter struct {
	precision int
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 ConvertWeight

**Purpose:** Convert weight values between metric (grams) and imperial (ounces)

**Algorithm:**
```
1. Input: value (float64), fromUnit (string), toUnit (string)
2. Validate that both units are weight units
3. Normalize input value to grams:
   IF fromUnit == "g": normalizedValue = value
   IF fromUnit == "kg": normalizedValue = value * 1000
   IF fromUnit == "oz": normalizedValue = value * 28.3495
   IF fromUnit == "lb": normalizedValue = value * 453.592
4. Convert from grams to target unit:
   IF toUnit == "g": result = normalizedValue
   IF toUnit == "kg": result = normalizedValue / 1000
   IF toUnit == "oz": result = normalizedValue / 28.3495
   IF toUnit == "lb": result = normalizedValue / 453.592
5. Round result to configured precision
6. Return ConversionResult with Value, Unit, UnitType=Weight, Precision
```

### 2.2 ConvertVolume

**Purpose:** Convert volume values between metric (ml) and imperial (fl oz)

**Algorithm:**
```
1. Input: value (float64), fromUnit (string), toUnit (string)
2. Validate that both units are volume units
3. Normalize input value to milliliters:
   IF fromUnit == "ml": normalizedValue = value
   IF fromUnit == "l": normalizedValue = value * 1000
   IF fromUnit == "fl oz": normalizedValue = value * 29.5735
4. Convert from ml to target unit:
   IF toUnit == "ml": result = normalizedValue
   IF toUnit == "l": result = normalizedValue / 1000
   IF toUnit == "fl oz": result = normalizedValue / 29.5735
5. Round result to configured precision
6. Return ConversionResult with Value, Unit, UnitType=Volume, Precision
```

### 2.3 ConvertMacros

**Purpose:** Convert macro values from per 100g/100ml to requested unit system

**Algorithm:**
```
1. Input: macros (MacroValues), physicalState (PhysicalState), targetUnitSystem (UnitSystem), quantity (float64)
2. Initialize result as copy of input macros (per 100g baseline)
3. IF targetUnitSystem == "imperial":
   a. Determine conversion factor based on physicalState:
      IF physicalState == "solid":
         factor = gramsToOunces(100)  // 3.5274
      IF physicalState == "liquid":
         factor = mlToFlOz(100)       // 3.3814
   b. Scale macros: result = macros / factor * quantity
4. IF targetUnitSystem == "metric":
   result = macros (baseline is already per 100g/100ml)
5. Return scaled MacroValues
```

### 2.4 ConvertFoodItem

**Purpose:** Convert entire FoodItem entity macros to requested unit system

**Algorithm:**
```
1. Input: foodItem (FoodItem), targetUnitSystem (UnitSystem), quantity (float64)
2. Validate quantity > 0
3. Call ConvertMacros(foodItem.macros, foodItem.physicalState, targetUnitSystem, quantity)
4. Return new FoodItem with converted macros
```

### 2.5 ConvertMeal

**Purpose:** Convert entire Meal entity macros to requested unit system

**Algorithm:**
```
1. Input: meal (Meal), targetUnitSystem (UnitSystem), quantity (float64)
2. Validate quantity > 0
3. IF meal.type == "single":
   a. Call ConvertFoodItem(meal.items, targetUnitSystem, quantity)
   b. Set meal.items to converted FoodItem
4. IF meal.type == "recipe":
   a. FOR EACH ingredient in meal.recipe:
      i. Call ConvertFoodItem(ingredient.item, targetUnitSystem, ingredient.qty * quantity)
      ii. Update ingredient.item with converted macros
   b. Recalculate total macros: sum of all converted ingredient macros
5. Return meal with all macros converted
```

### 2.6 NormalizeMacrosTo100g

**Purpose:** Convert any input macros to standard per-100g or per-100ml baseline

**Algorithm:**
```
1. Input: macros (MacroValues), sourceQuantity (float64), sourceUnit (string), physicalState (PhysicalState)
2. Validate sourceQuantity > 0
3. Normalize sourceQuantity to grams or ml:
   IF sourceUnit is weight unit:
      IF physicalState == "solid": baseGrams = 100
      baseMl = baseGrams / foodItem.averageUnitWeight * 100
   IF sourceUnit is volume unit:
      baseMl = 100
      baseGrams = baseMl * foodItem.averageUnitWeight / 100
4. Calculate normalized macros:
   normalizedProtein = (macros.protein / sourceQuantity) * baseValue
   normalizedCarbs = (macros.carbs / sourceQuantity) * baseValue
   normalizedFat = (macros.fat / sourceQuantity) * baseValue
5. Return NormalizedMacros with Per100g and Per100ml values
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Error Type | Handling Strategy |
| :--- | :--- | :--- |
| Invalid unit type | `ErrInvalidUnit` | Return error with valid unit types |
| Unsupported unit | `ErrUnsupportedUnit` | Return error listing supported units |
| Negative quantity | `ErrNegativeQuantity` | Return error; macros cannot be negative |
| Zero quantity | `ErrZeroQuantity` | Return error; division by zero prevention |
| Nil macros input | `ErrNilMacros` | Return error; input validation |
| Invalid physical state | `ErrInvalidPhysicalState` | Return error; validate against enum |
| Target unit system unknown | `ErrUnknownUnitSystem` | Return error; validate against enum |

### 3.2 State Transitions

```
INITIAL: UnitConverter created with default precision (2)

STATE: Ready
  - Can receive conversion requests
  - On error: transition to Error state with error message

STATE: Error
  - Cannot process conversions
  - On Reset(): transition to Ready with same configuration
```

### 3.3 Precision Configuration

```
Precision Range: 0 to 6 decimal places
Default: 2 decimal places
Validation: If precision < 0, set to 0; if > 6, set to 6
```

### 3.4 Immutable FoodItem/Meal

```
Rule: Original FoodItem and Meal entities are never modified
Implementation: All conversion methods return NEW entity instances
Benefits:
  - Thread-safe concurrent access
  - Preserves source of truth (database values)
  - Enables undo/redo functionality
```

## 4. Component Interfaces

```go
package repository

type UnitConverterInterface interface {
	ConvertWeight(value float64, fromUnit, toUnit string) (*ConversionResult, error)
	ConvertVolume(value float64, fromUnit, toUnit string) (*ConversionResult, error)
	ConvertMacros(macros MacroValues, physicalState PhysicalState, targetUnitSystem UnitSystem, quantity float64) (MacroValues, error)
	ConvertFoodItem(foodItem *FoodItem, targetUnitSystem UnitSystem, quantity float64) (*FoodItem, error)
	ConvertMeal(meal *Meal, targetUnitSystem UnitSystem, quantity float64) (*Meal, error)
	NormalizeMacrosTo100g(macros MacroValues, sourceQuantity float64, sourceUnit string, physicalState PhysicalState) (*NormalizedMacros, error)
	GetSupportedUnits(unitType UnitType) []string
	SetPrecision(precision int)
	GetPrecision() int
}

func NewUnitConverter(precision int) *UnitConverter {
	if precision < 0 {
		precision = 0
	}
	if precision > 6 {
		precision = 6
	}
	return &UnitConverter{precision: precision}
}

func (uc *UnitConverter) ConvertWeight(value float64, fromUnit, toUnit string) (*ConversionResult, error) {
	weightUnits := map[string]float64{
		"g":  1,
		"kg": 1000,
		"oz": 28.3495,
		"lb": 453.592,
	}

	fromFactor, fromExists := weightUnits[fromUnit]
	if !fromExists {
		return nil, ErrUnsupportedUnit{Unit: fromUnit, UnitType: UnitTypeWeight}
	}

	toFactor, toExists := weightUnits[toUnit]
	if !toExists {
		return nil, ErrUnsupportedUnit{Unit: toUnit, UnitType: UnitTypeWeight}
	}

	normalizedGrams := value * fromFactor
	resultValue := normalizedGrams / toFactor
	roundedResult := uc.round(resultValue)

	return &ConversionResult{
		Value:     roundedResult,
		Unit:      toUnit,
		UnitType:  UnitTypeWeight,
		Precision: uc.precision,
	}, nil
}

func (uc *UnitConverter) ConvertVolume(value float64, fromUnit, toUnit string) (*ConversionResult, error) {
	volumeUnits := map[string]float64{
		"ml":    1,
		"l":     1000,
		"fl oz": 29.5735,
	}

	fromFactor, fromExists := volumeUnits[fromUnit]
	if !fromExists {
		return nil, ErrUnsupportedUnit{Unit: fromUnit, UnitType: UnitTypeVolume}
	}

	toFactor, toExists := volumeUnits[toUnit]
	if !toExists {
		return nil, ErrUnsupportedUnit{Unit: toUnit, UnitType: UnitTypeVolume}
	}

	normalizedMl := value * fromFactor
	resultValue := normalizedMl / toFactor
	roundedResult := uc.round(resultValue)

	return &ConversionResult{
		Value:     roundedResult,
		Unit:      toUnit,
		UnitType:  UnitTypeVolume,
		Precision: uc.precision,
	}, nil
}

func (uc *UnitConverter) ConvertMacros(macros MacroValues, physicalState PhysicalState, targetUnitSystem UnitSystem, quantity float64) (MacroValues, error) {
	if quantity <= 0 {
		return MacroValues{}, ErrNegativeQuantity{Value: quantity}
	}

	result := MacroValues{
		Protein: macros.Protein,
		Carbs:   macros.Carbs,
		Fat:     macros.Fat,
	}

	if targetUnitSystem == UnitSystemImperial {
		var factor float64
		if physicalState == PhysicalStateSolid {
			factor = 28.3495 * 3.5274
		} else if physicalState == PhysicalStateLiquid {
			factor = 29.5735 * 3.3814
		} else {
			return MacroValues{}, ErrInvalidPhysicalState{State: physicalState}
		}

		result.Protein = uc.round((result.Protein / factor) * quantity)
		result.Carbs = uc.round((result.Carbs / factor) * quantity)
		result.Fat = uc.round((result.Fat / factor) * quantity)
	}

	return result, nil
}

func (uc *UnitConverter) ConvertFoodItem(foodItem *FoodItem, targetUnitSystem UnitSystem, quantity float64) (*FoodItem, error) {
	if quantity <= 0 {
		return nil, ErrNegativeQuantity{Value: quantity}
	}

	convertedMacros, err := uc.ConvertMacros(foodItem.macros, foodItem.physicalState, targetUnitSystem, quantity)
	if err != nil {
		return nil, err
	}

	newFoodItem := *foodItem
	newFoodItem.macros = convertedMacros
	return &newFoodItem, nil
}

func (uc *UnitConverter) ConvertMeal(meal *Meal, targetUnitSystem UnitSystem, quantity float64) (*Meal, error) {
	if quantity <= 0 {
		return nil, ErrNegativeQuantity{Value: quantity}
	}

	newMeal := *meal

	if meal.type == MealTypeSingle {
		convertedItem, err := uc.ConvertFoodItem(meal.items, targetUnitSystem, quantity)
		if err != nil {
			return nil, err
		}
		newMeal.items = convertedItem
	} else if meal.type == MealTypeRecipe {
		var totalMacros MacroValues
		for i := range meal.recipe {
			ingredientQty := meal.recipe[i].qty * quantity
			convertedItem, err := uc.ConvertFoodItem(&meal.recipe[i].item, targetUnitSystem, ingredientQty)
			if err != nil {
				return nil, err
			}
			newMeal.recipe[i].item = *convertedItem
			totalMacros.Protein += convertedItem.macros.Protein
			totalMacros.Carbs += convertedItem.macros.Carbs
			totalMacros.Fat += convertedItem.macros.Fat
		}
	}

	return &newMeal, nil
}

func (uc *UnitConverter) NormalizeMacrosTo100g(macros MacroValues, sourceQuantity float64, sourceUnit string, physicalState PhysicalState) (*NormalizedMacros, error) {
	if sourceQuantity <= 0 {
		return nil, ErrZeroQuantity{}
	}

	normalized := &NormalizedMacros{
		Per100g:  MacroValues{},
		Per100ml: MacroValues{},
	}

	per100g, err := uc.normalizeToPer100g(macros, sourceQuantity, sourceUnit)
	if err != nil {
		return nil, err
	}
	normalized.Per100g = per100g
	normalized.Per100ml = uc.convertWeightToVolume(per100g, physicalState)

	return normalized, nil
}

func (uc *UnitConverter) GetSupportedUnits(unitType UnitType) []string {
	switch unitType {
	case UnitTypeWeight:
		return []string{"g", "kg", "oz", "lb"}
	case UnitTypeVolume:
		return []string{"ml", "l", "fl oz"}
	default:
		return []string{}
	}
}

func (uc *UnitConverter) SetPrecision(precision int) {
	if precision < 0 {
		uc.precision = 0
	} else if precision > 6 {
		uc.precision = 6
	} else {
		uc.precision = precision
	}
}

func (uc *UnitConverter) GetPrecision() int {
	return uc.precision
}

func (uc *UnitConverter) round(value float64) float64 {
	factor := math.Pow(10, float64(uc.precision))
	return math.Round(value*factor) / factor
}

func (uc *UnitConverter) normalizeToPer100g(macros MacroValues, sourceQuantity float64, sourceUnit string) (MacroValues, error) {
	weightUnits := map[string]bool{"g": true, "kg": true, "oz": true, "lb": true}
	volumeUnits := map[string]bool{"ml": true, "l": true, "fl oz": true}

	if weightUnits[sourceUnit] {
		grams := sourceQuantity
		if sourceUnit == "kg" {
			grams *= 1000
		} else if sourceUnit == "oz" {
			grams *= 28.3495
		} else if sourceUnit == "lb" {
			grams *= 453.592
		}
		return MacroValues{
			Protein: (macros.Protein / grams) * 100,
			Carbs:   (macros.Carbs / grams) * 100,
			Fat:     (macros.Fat / grams) * 100,
		}, nil
	}

	if volumeUnits[sourceUnit] {
		ml := sourceQuantity
		if sourceUnit == "l" {
			ml *= 1000
		} else if sourceUnit == "fl oz" {
			ml *= 29.5735
		}
		return MacroValues{
			Protein: (macros.Protein / ml) * 100,
			Carbs:   (macros.Carbs / ml) * 100,
			Fat:     (macros.Fat / ml) * 100,
		}, nil
	}

	return MacroValues{}, ErrUnsupportedUnit{Unit: sourceUnit, UnitType: UnitTypeWeight}
}

func (uc *UnitConverter) convertWeightToVolume(macros MacroValues, physicalState PhysicalState) MacroValues {
	return MacroValues{
		Protein: macros.Protein,
		Carbs:   macros.Carbs,
		Fat:     macros.Fat,
	}
}
```

### Error Types

```go
package repository

type ErrUnsupportedUnit struct {
	Unit     string
	UnitType UnitType
}

func (e ErrUnsupportedUnit) Error() string {
	return fmt.Sprintf("unsupported unit: %s for type: %s", e.Unit, e.UnitType)
}

type ErrNegativeQuantity struct {
	Value float64
}

func (e ErrNegativeQuantity) Error() string {
	return fmt.Sprintf("quantity cannot be zero or negative: %f", e.Value)
}

type ErrZeroQuantity struct{}

func (e ErrZeroQuantity) Error() string {
	return "quantity cannot be zero"
}

type ErrNilMacros struct{}

func (e ErrNilMacros) Error() string {
	return "macros input cannot be nil"
}

type ErrInvalidPhysicalState struct {
	State PhysicalState
}

func (e ErrInvalidPhysicalState) Error() string {
	return fmt.Sprintf("invalid physical state: %s", e.State)
}

type ErrInvalidUnit struct {
	Unit    string
	Message string
}

func (e ErrInvalidUnit) Error() string {
	return fmt.Sprintf("invalid unit %s: %s", e.Unit, e.Message)
}
```

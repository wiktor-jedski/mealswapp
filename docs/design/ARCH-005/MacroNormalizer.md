# MacroNormalizer

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

type UnitPreference struct {
	System     UnitSystem
	VolumeUnit string // "ml", "fl oz", "l"
	WeightUnit string // "g", "oz", "kg"
}

type Macronutrients struct {
	Protein float64 // grams per 100g or 100ml
	Carbs   float64 // grams per 100g or 100ml
	Fat     float64 // grams per 100g or 100ml
}

type Micronutrients struct {
	Sodium   float64
	Fiber    float64
	Sugar    float64
	Cholesterol float64
	Potassium float64
	Iron     float64
	Calcium  float64
	VitaminA float64
	VitaminC float64
	VitaminD float64
	// Additional micronutrients stored in JSONB column
	Additional map[string]float64
}

type FoodItem struct {
	ID                 string
	Name               string
	PhysicalState      PhysicalState
	PrepTime           int // minutes
	AverageUnitWeight  float64 // grams
	Macros             Macronutrients
	Micros             Micronutrients
	CategoryTags       []string
	FunctionalityTags  []string
	ImageURL           *string
}

type RecipeItem struct {
	FoodItem *FoodItem
	Quantity float64 // in grams
}

type Meal struct {
	ID                  string
	Type                string // "single" | "recipe"
	Items               []*FoodItem // for single dish
	Recipe              []*RecipeItem // for recipe composition
	PhysicalState       PhysicalState
	PrepTime            int // minutes
	AverageUnitWeight   float64 // grams
	CategoryTags        []string
	FunctionalityTags   []string
}

type NormalizedFoodItem struct {
	ID                 string
	Name               string
	PhysicalState      PhysicalState
	PrepTime           int
	AverageUnitWeight  float64
	NormalizedMacros   Macronutrients
	NormalizedMicros   Micronutrients
	CategoryTags       []string
	FunctionalityTags  []string
	ImageURL           *string
}

type NormalizedMeal struct {
	ID                  string
	Type                string
	AggregatedMacros    Macronutrients
	AggregatedMicros    Micronutrients
	PhysicalState       PhysicalState
	PrepTime            int
	AverageUnitWeight   float64
	CategoryTags        []string
	FunctionalityTags   []string
}

type ConversionFactors struct {
	GramsToOunces    float64 = 0.035274
	OuncesToGrams    float64 = 28.3495
	MillilitersToFlOz float64 = 0.033814
	FlOzToMilliliters float64 = 29.5735
}
```

## 2. Logic & Algorithms

### 2.1 NormalizeMacrosTo100g

```go
// NormalizeMacrosTo100g converts any macro values to per 100g/100ml standard
// Returns error if foodItem is nil or has invalid macro values
func (n *MacroNormalizer) NormalizeMacrosTo100g(foodItem *FoodItem, servingSizeGrams float64) (*NormalizedFoodItem, error) {
	// Step 1: Validate input parameters
	if foodItem == nil {
		return nil, ErrNilFoodItem
	}
	if servingSizeGrams <= 0 {
		return nil, ErrInvalidServingSize
	}

	// Step 2: Extract raw macros from source
	rawProtein := foodItem.Macros.Protein
	rawCarbs := foodItem.Macros.Carbs
	rawFat := foodItem.Macros.Fat

	// Step 3: Calculate normalization factor
	// If macros are already per 100g, factor is 1.0
	// Otherwise, scale proportionally
	normalizationFactor := 100.0 / servingSizeGrams

	// Step 4: Apply normalization factor
	normalizedProtein := rawProtein * normalizationFactor
	normalizedCarbs := rawCarbs * normalizationFactor
	normalizedFat := rawFat * normalizationFactor

	// Step 5: Construct normalized food item
	normalized := &NormalizedFoodItem{
		ID:                foodItem.ID,
		Name:              foodItem.Name,
		PhysicalState:     foodItem.PhysicalState,
		PrepTime:          foodItem.PrepTime,
		AverageUnitWeight: foodItem.AverageUnitWeight,
		NormalizedMacros: Macronutrients{
			Protein: roundToTwoDecimal(normalizedProtein),
			Carbs:   roundToTwoDecimal(normalizedCarbs),
			Fat:     roundToTwoDecimal(normalizedFat),
		},
		NormalizedMicros:  foodItem.Micros,
		CategoryTags:      foodItem.CategoryTags,
		FunctionalityTags: foodItem.FunctionalityTags,
		ImageURL:          foodItem.ImageURL,
	}

	return normalized, nil
}
```

### 2.2 ConvertUnitSystem

```go
// ConvertUnitSystem transforms normalized values to requested unit system
// Handles both weight (g/oz/kg) and volume (ml/fl oz/l) conversions
func (n *MacroNormalizer) ConvertUnitSystem(
	normalizedItem *NormalizedFoodItem,
	preference UnitPreference,
) (*NormalizedFoodItem, error) {
	// Step 1: Validate preference
	if preference.System != UnitSystemMetric && preference.System != UnitSystemImperial {
		return nil, ErrUnknownUnitSystem
	}

	// Step 2: Create a copy to avoid mutation of original
	result := &NormalizedFoodItem{
		ID:                normalizedItem.ID,
		Name:              normalizedItem.Name,
		PhysicalState:     normalizedItem.PhysicalState,
		PrepTime:          normalizedItem.PrepTime,
		NormalizedMacros:  normalizedItem.NormalizedMacros,
		NormalizedMicros:  normalizedItem.NormalizedMicros,
		CategoryTags:      normalizedItem.CategoryTags,
		FunctionalityTags: normalizedItem.FunctionalityTags,
		ImageURL:          normalizedItem.ImageURL,
	}

	// Step 3: Handle weight unit conversion if needed
	switch preference.WeightUnit {
	case "oz":
		result.AverageUnitWeight = normalizedItem.AverageUnitWeight * ConversionFactors.GramsToOunces
		result.AverageUnitWeight = roundToTwoDecimal(result.AverageUnitWeight)
	case "kg":
		result.AverageUnitWeight = normalizedItem.AverageUnitWeight / 1000.0
		result.AverageUnitWeight = roundToTwoDecimal(result.AverageUnitWeight)
	case "g":
		// No conversion needed
	default:
		// Use default metric if unknown unit
	}

	// Step 4: Handle volume unit conversion for liquids
	if normalizedItem.PhysicalState == PhysicalStateLiquid {
		switch preference.VolumeUnit {
		case "fl oz":
			result.AverageUnitWeight = normalizedItem.AverageUnitWeight * ConversionFactors.MillilitersToFlOz
			result.AverageUnitWeight = roundToTwoDecimal(result.AverageUnitWeight)
		case "l":
			result.AverageUnitWeight = normalizedItem.AverageUnitWeight / 1000.0
			result.AverageUnitWeight = roundToTwoDecimal(result.AverageUnitWeight)
		case "ml":
			// No conversion needed
		default:
			// Use default metric if unknown unit
		}
	}

	return result, nil
}
```

### 2.3 AggregateMealMacros

```go
// AggregateMealMacros calculates total macros for recipe-based meals
// Sums constituent ingredients with proper scaling
func (n *MacroNormalizer) AggregateMealMacros(meal *Meal) (*NormalizedMeal, error) {
	// Step 1: Validate meal type
	if meal.Type != "single" && meal.Type != "recipe" {
		return nil, ErrInvalidMealType
	}

	// Step 2: Initialize accumulators
	totalProtein := 0.0
	totalCarbs := 0.0
	totalFat := 0.0
	totalMicros := Micronutrients{
		Additional: make(map[string]float64),
	}

	var weightedPrepTime float64
	var totalWeight float64
	var categorySet = make(map[string]bool)
	var functionalitySet = make(map[string]bool)

	// Step 3: Process based on meal type
	if meal.Type == "single" {
		for _, item := range meal.Items {
			if item == nil {
				continue
			}
			// Use average unit weight as default quantity
			quantity := item.AverageUnitWeight
			totalWeight += quantity

			// Scale macros by quantity
			factor := quantity / 100.0
			totalProtein += item.Macros.Protein * factor
			totalCarbs += item.Macros.Carbs * factor
			totalFat += item.Macros.Fat * factor

			// Accumulate prep time weighted by ingredient contribution
			weightedPrepTime += float64(item.PrepTime) * factor

			// Merge tags
			for _, tag := range item.CategoryTags {
				categorySet[tag] = true
			}
			for _, tag := range item.FunctionalityTags {
				functionalitySet[tag] = true
			}
		}
	} else if meal.Type == "recipe" {
		for _, recipeItem := range meal.Recipe {
			if recipeItem == nil || recipeItem.FoodItem == nil {
				continue
			}
			quantity := recipeItem.Quantity
			totalWeight += quantity

			// Scale macros by quantity
			factor := quantity / 100.0
			totalProtein += recipeItem.FoodItem.Macros.Protein * factor
			totalCarbs += recipeItem.FoodItem.Macros.Carbs * factor
			totalFat += recipeItem.FoodItem.Macros.Fat * factor

			// Accumulate prep time weighted by ingredient contribution
			weightedPrepTime += float64(recipeItem.FoodItem.PrepTime) * factor

			// Merge tags
			for _, tag := range recipeItem.FoodItem.CategoryTags {
				categorySet[tag] = true
			}
			for _, tag := range recipeItem.FoodItem.FunctionalityTags {
				functionalitySet[tag] = true
			}
		}
	}

	// Step 4: Calculate average prep time
	avgPrepTime := 0
	if totalWeight > 0 {
		avgPrepTime = int(weightedPrepTime / (totalWeight / 100.0))
	}

	// Step 5: Convert category and functionality sets to slices
	categories := make([]string, 0, len(categorySet))
	for tag := range categorySet {
		categories = append(categories, tag)
	}

	functionalities := make([]string, 0, len(functionalitySet))
	for tag := range functionalitySet {
		functionalities = append(functionalities, tag)
	}

	// Step 6: Construct aggregated meal
	aggregated := &NormalizedMeal{
		ID:              meal.ID,
		Type:            meal.Type,
		AggregatedMacros: Macronutrients{
			Protein: roundToTwoDecimal(totalProtein),
			Carbs:   roundToTwoDecimal(totalCarbs),
			Fat:     roundToTwoDecimal(totalFat),
		},
		AggregatedMicros:  totalMicros,
		PhysicalState:     meal.PhysicalState,
		PrepTime:          avgPrepTime,
		AverageUnitWeight: totalWeight,
		CategoryTags:      categories,
		FunctionalityTags: functionalities,
	}

	return aggregated, nil
}
```

### 2.4 ScaleMacrosByQuantity

```go
// ScaleMacrosByQuantity calculates macros for a specific serving size
// Uses normalized per-100g values and scales proportionally
func (n *MacroNormalizer) ScaleMacrosByQuantity(
	normalizedItem *NormalizedFoodItem,
	quantityGrams float64,
) (Macronutrients, error) {
	// Step 1: Validate inputs
	if normalizedItem == nil {
		return Macronutrients{}, ErrNilFoodItem
	}
	if quantityGrams <= 0 {
		return Macronutrients{}, ErrInvalidQuantity
	}

	// Step 2: Calculate scaling factor
	// Normalized values are per 100g
	factor := quantityGrams / 100.0

	// Step 3: Scale each macro nutrient
	scaledProtein := normalizedItem.NormalizedMacros.Protein * factor
	scaledCarbs := normalizedItem.NormalizedMacros.Carbs * factor
	scaledFat := normalizedItem.NormalizedMacros.Fat * factor

	// Step 4: Return scaled macros
	return Macronutrients{
		Protein: roundToTwoDecimal(scaledProtein),
		Carbs:   roundToTwoDecimal(scaledCarbs),
		Fat:     roundToTwoDecimal(scaledFat),
	}, nil
}
```

### 2.5 ConvertPhysicalState

```go
// ConvertPhysicalState handles macro adjustments for physical state changes
// For example, raw vs cooked weight differences
func (n *MacroNormalizer) ConvertPhysicalState(
	foodItem *FoodItem,
	conversionFactor float64, // e.g., 0.8 for 20% weight loss during cooking
) (*NormalizedFoodItem, error) {
	// Step 1: Validate conversion factor
	if conversionFactor <= 0 {
		return nil, ErrInvalidConversionFactor
	}

	// Step 2: Normalize to 100g first
	normalized, err := n.NormalizeMacrosTo100g(foodItem, foodItem.AverageUnitWeight)
	if err != nil {
		return nil, err
	}

	// Step 3: Apply physical state conversion
	adjustedMacros := Macronutrients{
		Protein: normalized.NormalizedMacros.Protein / conversionFactor,
		Carbs:   normalized.NormalizedMacros.Carbs / conversionFactor,
		Fat:     normalized.NormalizedMacros.Fat / conversionFactor,
	}

	// Step 4: Return with adjusted macros
	return &NormalizedFoodItem{
		ID:                normalized.ID,
		Name:              normalized.Name,
		PhysicalState:     normalized.PhysicalState,
		PrepTime:          normalized.PrepTime,
		AverageUnitWeight: normalized.AverageUnitWeight * conversionFactor,
		NormalizedMacros:  adjustedMacros,
		NormalizedMicros:  normalized.NormalizedMicros,
		CategoryTags:      normalized.CategoryTags,
		FunctionalityTags: normalized.FunctionalityTags,
		ImageURL:          normalized.ImageURL,
	}, nil
}
```

## 3. State Management & Error Handling

### 3.1 Error Definitions

```go
package repository

import "errors"

var (
	ErrNilFoodItem           = errors.New("food item cannot be nil")
	ErrInvalidServingSize    = errors.New("serving size must be positive")
	ErrUnknownUnitSystem     = errors.New("unknown unit system")
	ErrInvalidMealType       = errors.New("meal type must be 'single' or 'recipe'")
	ErrInvalidQuantity       = errors.New("quantity must be positive")
	ErrInvalidConversionFactor = errors.New("conversion factor must be positive")
	ErrMacrosNotAvailable    = errors.New("macronutrient data not available")
	ErrMicrosNotAvailable    = errors.New("micronutrient data not available")
)
```

### 3.2 State Transitions

| Current State | Trigger | Next State | Action |
|---------------|---------|------------|--------|
| RawFoodItem | NormalizeMacrosTo100g | NormalizedFoodItem | Macro values converted to per 100g standard |
| NormalizedFoodItem | ConvertUnitSystem | ConvertedFoodItem | Values transformed to imperial/metric units |
| MultipleItems | AggregateMealMacros | NormalizedMeal | Macros summed, prep time averaged, tags merged |
| NormalizedFoodItem | ScaleMacrosByQuantity | ScaledMacros | Macros proportionally adjusted by quantity |
| FoodItem | ConvertPhysicalState | AdjustedFoodItem | Macros adjusted for physical state change |

### 3.3 Error Scenarios

| Error Condition | Handling Strategy | User Impact |
|-----------------|-------------------|-------------|
| Nil food item passed | Return ErrNilFoodItem immediately | Prevents null pointer dereference |
| Zero or negative serving size | Return ErrInvalidServingSize | Input validation failure |
| Unknown unit system | Return ErrUnknownUnitSystem | Default to metric system |
| Invalid meal type | Return ErrInvalidMealType | Prevents invalid aggregation |
| Negative conversion factor | Return ErrInvalidConversionFactor | Input validation failure |
| Missing macro data | Return ErrMacrosNotAvailable | Display "data unavailable" to user |
| Missing micro data | Return ErrMacrosNotAvailable | Display partial data with note |

## 4. Component Interfaces

### 4.1 Public Interface

```go
type MacroNormalizer interface {
	// NormalizeMacrosTo100g converts food item macros to per 100g/100ml standard
	NormalizeMacrosTo100g(foodItem *FoodItem, servingSizeGrams float64) (*NormalizedFoodItem, error)

	// ConvertUnitSystem transforms normalized values to requested unit system
	ConvertUnitSystem(normalizedItem *NormalizedFoodItem, preference UnitPreference) (*NormalizedFoodItem, error)

	// AggregateMealMacros calculates total macros for recipe-based meals
	AggregateMealMacros(meal *Meal) (*NormalizedMeal, error)

	// ScaleMacrosByQuantity calculates macros for a specific serving size
	ScaleMacrosByQuantity(normalizedItem *NormalizedFoodItem, quantityGrams float64) (Macronutrients, error)

	// ConvertPhysicalState handles macro adjustments for physical state changes
	ConvertPhysicalState(foodItem *FoodItem, conversionFactor float64) (*NormalizedFoodItem, error)
}
```

### 4.2 Internal Helper Functions

```go
// roundToTwoDecimal rounds a float64 to two decimal places
func roundToTwoDecimal(value float64) float64 {
	return math.Round(value*100) / 100
}

// gramsToOunces converts grams to ounces
func gramsToOunces(grams float64) float64 {
	return grams * ConversionFactors.GramsToOunces
}

// ouncesToGrams converts ounces to grams
func ouncesToGrams(ounces float64) float64 {
	return ounces * ConversionFactors.OuncesToGrams
}

// millilitersToFlOz converts milliliters to fluid ounces
func millilitersToFlOz(ml float64) float64 {
	return ml * ConversionFactors.MillilitersToFlOz
}
```

### 4.3 Repository Integration

The MacroNormalizer integrates with the Data Repository Module at the boundary between database queries and domain entity returns. All read operations flow through normalization logic before returning data to callers.

```go
type Repository interface {
	GetFoodItemByID(ctx context.Context, id string, unitPref UnitPreference) (*NormalizedFoodItem, error)
	GetMealByID(ctx context.Context, id string, unitPref UnitPreference) (*NormalizedMeal, error)
	SearchFoodItems(ctx context.Context, query string, unitPref UnitPreference) ([]*NormalizedFoodItem, error)
}
```

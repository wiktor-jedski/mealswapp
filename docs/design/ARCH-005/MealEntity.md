# MealEntity Detailed Design

**Traceability:** ARCH-005

## 1. Data Structures & Types

```go
package entity

import (
	"time"
	"github.com/google/uuid"
)

type MealType string

const (
	MealTypeSingle   MealType = "single"
	MealTypeRecipe   MealType = "recipe"
)

type PhysicalState string

const (
	PhysicalStateSolid  PhysicalState = "solid"
	PhysicalStateLiquid PhysicalState = "liquid"
)

type Macros struct {
	Protein float64 `json:"protein"` // grams per 100g/100ml
	Carbs   float64 `json:"carbs"`   // grams per 100g/100ml
	Fat     float64 `json:"fat"`     // grams per 100g/100ml
}

type Micros struct {
	Sodium   float64 `json:"sodium,omitempty"`
	Fiber    float64 `json:"fiber,omitempty"`
	Sugar    float64 `json:"sugar,omitempty"`
	Cholesterol float64 `json:"cholesterol,omitempty"`
	Potassium float64 `json:"potassium,omitempty"`
	VitaminA float64 `json:"vitamin_a,omitempty"`
	VitaminC float64 `json:"vitamin_c,omitempty"`
	Calcium  float64 `json:"calcium,omitempty"`
	Iron     float64 `json:"iron,omitempty"`
}

type RecipeIngredient struct {
	FoodItemID uuid.UUID `json:"food_item_id"`
	Quantity   float64   `json:"quantity"` // grams
}

type RecipeComposition struct {
	Ingredients []RecipeIngredient `json:"ingredients"`
}

type CategoryTag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type FunctionalityTag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type Meal struct {
	ID                     uuid.UUID          `json:"id" db:"id"`
	Type                   MealType           `json:"type" db:"type"`
	PhysicalState          PhysicalState      `json:"physical_state" db:"physical_state"`
	PrepTime               int                `json:"prep_time" db:"prep_time"` // minutes
	AverageUnitWeight      float64            `json:"average_unit_weight" db:"average_unit_weight"` // grams
	CategoryTags           []CategoryTag      `json:"category_tags"`
	FunctionalityTags      []FunctionalityTag `json:"functionality_tags"`
	SingleFoodItemID       *uuid.UUID         `json:"single_food_item_id,omitempty" db:"single_food_item_id"`
	RecipeComposition      *RecipeComposition `json:"recipe_composition,omitempty"`
	CalculatedMacros       Macros             `json:"calculated_macros"`
	CalculatedMicros       Micros             `json:"calculated_micros"`
	ScaledMacros           *Macros            `json:"scaled_macros,omitempty"`
	ScaledMicros           *Micros            `json:"scaled_micros,omitempty"`
	CreatedAt              time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at" db:"updated_at"`
}

type CreateMealInput struct {
	Type              MealType
	PhysicalState     PhysicalState
	PrepTime          int
	AverageUnitWeight float64
	CategoryTagIDs    []uuid.UUID
	FunctionalityTagIDs []uuid.UUID
	SingleFoodItemID  *uuid.UUID
	RecipeIngredients []RecipeIngredient
}

type UpdateMealInput struct {
	ID                 uuid.UUID
	Type               *MealType
	PhysicalState      *PhysicalState
	PrepTime           *int
	AverageUnitWeight  *float64
	CategoryTagIDs     []uuid.UUID
	FunctionalityTagIDs []uuid.UUID
	SingleFoodItemID   *uuid.UUID
	RecipeIngredients  []RecipeIngredient
}

type MealQueryOptions struct {
	MealTypeFilter      *MealType
	PhysicalStateFilter *PhysicalState
	PrepTimeMax         *int
	CategoryTagIDs      []uuid.UUID
	FunctionalityTagIDs []uuid.UUID
	IncludeMacros       bool
	UnitPreference      string // "metric" or "imperial"
}

type UnitSystem string

const (
	UnitSystemMetric   UnitSystem = "metric"
	UnitSystemImperial UnitSystem = "imperial"
)

type UnitConversionFactors struct {
	Mass float64 // 28.3495 for g to oz
	Volume float64 // 29.5735 for ml to fl oz
}
```

## 2. Logic & Algorithms

### 2.1 Meal Creation Algorithm

```
FUNCTION CreateMeal(input CreateMealInput) -> (Meal, error)
	BEGIN
		mealID := uuid.New()
		meal := Meal{
			ID: mealID,
			Type: input.Type,
			PhysicalState: input.PhysicalState,
			PrepTime: input.PrepTime,
			AverageUnitWeight: input.AverageUnitWeight,
			CategoryTags: [],
			FunctionalityTags: [],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		SWITCH input.Type
			CASE "single":
				IF input.SingleFoodItemID == nil THEN
					RETURN error("Single meal requires SingleFoodItemID")
				END IF
				meal.SingleFoodItemID = input.SingleFoodItemID
				
				foodItem, err := foodItemRepository.GetByID(input.SingleFoodItemID)
				IF err != nil THEN
					RETURN error("Failed to fetch food item: " + err.Error())
				END IF
				
				meal.CalculatedMacros = CalculateMacrosFromFoodItem(foodItem, 100.0)
				meal.CalculatedMicros = CalculateMicrosFromFoodItem(foodItem, 100.0)
				meal.PhysicalState = foodItem.PhysicalState
				meal.AverageUnitWeight = foodItem.AverageUnitWeight

			CASE "recipe":
				IF len(input.RecipeIngredients) == 0 THEN
					RETURN error("Recipe meal requires at least one ingredient")
				END IF
				
				meal.RecipeComposition = &RecipeComposition{
					Ingredients: input.RecipeIngredients,
				}
				
				meal.CalculatedMacros, meal.CalculatedMicros = 
					CalculateAggregatedMacros(input.RecipeIngredients)
			DEFAULT:
				RETURN error("Invalid meal type: " + input.Type)
		END SWITCH

		meal.CategoryTags = fetchTagsByIDs(input.CategoryTagIDs)
		meal.FunctionalityTags = fetchTagsByIDs(input.FunctionalityTagIDs)

		err := mealRepository.Insert(meal)
		IF err != nil THEN
			RETURN error("Failed to insert meal: " + err.Error())
		END IF

		RETURN meal, nil
	END
```

### 2.2 Macro Aggregation Algorithm (Recipe Meals)

```
FUNCTION CalculateAggregatedMacros(ingredients []RecipeIngredient) -> (Macros, Micros)
	BEGIN
		totalMacros := Macros{Protein: 0, Carbs: 0, Fat: 0}
		totalMicros := Micros{}
		totalWeight := 0.0

		FOR each ingredient IN ingredients
			foodItem, err := foodItemRepository.GetByID(ingredient.FoodItemID)
			IF err != nil THEN
				CONTINUE // Skip problematic ingredients
			END IF

			scaleFactor := ingredient.Quantity / 100.0

			totalMacros.Protein += foodItem.Macros.Protein * scaleFactor
			totalMacros.Carbs += foodItem.Macros.Carbs * scaleFactor
			totalMacros.Fat += foodItem.Macros.Fat * scaleFactor

			totalMicros.Sodium += foodItem.Micros.Sodium * scaleFactor
			totalMicros.Fiber += foodItem.Micros.Fiber * scaleFactor
			totalMicros.Sugar += foodItem.Micros.Sugar * scaleFactor
			totalMicros.Cholesterol += foodItem.Micros.Cholesterol * scaleFactor
			totalMicros.Potassium += foodItem.Micros.Potassium * scaleFactor
			totalMicros.VitaminA += foodItem.Micros.VitaminA * scaleFactor
			totalMicros.VitaminC += foodItem.Micros.VitaminC * scaleFactor
			totalMicros.Calcium += foodItem.Micros.Calcium * scaleFactor
			totalMicros.Iron += foodItem.Micros.Iron * scaleFactor

			totalWeight += ingredient.Quantity
		END FOR

		mealPhysicalState := determineDominantPhysicalState(ingredients)
		mealAverageUnitWeight := calculateAverageUnitWeight(ingredients, totalWeight)

		RETURN totalMacros, totalMicros
	END
```

### 2.3 Unit Conversion Algorithm

```
FUNCTION ConvertMealToUnitSystem(meal Meal, system UnitSystem) -> Meal
	BEGIN
		IF system == UnitSystemMetric THEN
			RETURN meal // Already stored in metric
		END IF

		IF system == UnitSystemImperial THEN
			factors := UnitConversionFactors{
				Mass: 0.035274, // g to oz
				Volume: 0.033814, // ml to fl oz
			}

			meal.ScaledMacros = &Macros{
				Protein: meal.CalculatedMacros.Protein * factors.Mass,
				Carbs: meal.CalculatedMacros.Carbs * factors.Mass,
				Fat: meal.CalculatedMacros.Fat * factors.Mass,
			}

			meal.ScaledMicros = &Micros{
				Sodium: meal.CalculatedMicros.Sodium * factors.Mass,
				Fiber: meal.CalculatedMicros.Fiber * factors.Mass,
				Sugar: meal.CalculatedMicros.Sugar * factors.Mass,
			}
		END IF

		RETURN meal
	END
```

### 2.4 Quantity Scaling Algorithm

```
FUNCTION ScaleMealToQuantity(meal Meal, targetGrams float64) -> Meal
	BEGIN
		baseQuantity := 100.0
		scaleFactor := targetGrams / baseQuantity

		scaled := meal
		scaled.ScaledMacros = &Macros{
			Protein: meal.CalculatedMacros.Protein * scaleFactor,
			Carbs: meal.CalculatedMacros.Carbs * scaleFactor,
			Fat: meal.CalculatedMacros.Fat * scaleFactor,
		}

		scaled.ScaledMicros = &Micros{
			Sodium: meal.CalculatedMicros.Sodium * scaleFactor,
			Fiber: meal.CalculatedMicros.Fiber * scaleFactor,
			Sugar: meal.CalculatedMicros.Sugar * scaleFactor,
			Cholesterol: meal.CalculatedMicros.Cholesterol * scaleFactor,
			Potassium: meal.CalculatedMicros.Potassium * scaleFactor,
			VitaminA: meal.CalculatedMicros.VitaminA * scaleFactor,
			VitaminC: meal.CalculatedMicros.VitaminC * scaleFactor,
			Calcium: meal.CalculatedMicros.Calcium * scaleFactor,
			Iron: meal.CalculatedMicros.Iron * scaleFactor,
		}

		RETURN scaled
	END
```

### 2.5 Meal Query Algorithm

```
FUNCTION QueryMeals(options MealQueryOptions) -> ([]Meal, error)
	BEGIN
		query := "SELECT * FROM meals WHERE 1=1"
		args := []interface{}{}
		argIndex := 1

		IF options.MealTypeFilter != nil THEN
			query += " AND type = $" + argIndex
			args = append(args, *options.MealTypeFilter)
			argIndex++
		END IF

		IF options.PhysicalStateFilter != nil THEN
			query += " AND physical_state = $" + argIndex
			args = append(args, *options.PhysicalStateFilter)
			argIndex++
		END IF

		IF options.PrepTimeMax != nil THEN
			query += " AND prep_time <= $" + argIndex
			args = append(args, *options.PrepTimeMax)
			argIndex++
		END IF

		rows, err := db.Query(query, args...)
		IF err != nil THEN
			RETURN nil, error("Query failed: " + err.Error())
		END IF
		DEFERRows(rows)

		meals := []Meal{}
		FOR rows.Next()
			meal := Meal{}
			err := rows.Scan(&meal.ID, &meal.Type, ...)
			IF err != nil THEN
				RETURN nil, error("Scan failed: " + err.Error())
			END IF

			meal.CategoryTags = fetchMealCategoryTags(meal.ID)
			meal.FunctionalityTags = fetchMealFunctionalityTags(meal.ID)

			IF meal.Type == MealTypeRecipe THEN
				meal.RecipeComposition = fetchRecipeComposition(meal.ID)
			END IF

			IF options.IncludeMacros THEN
				meal.CalculatedMacros = fetchCalculatedMacros(meal.ID)
				meal.CalculatedMicros = fetchCalculatedMicros(meal.ID)
			END IF

			IF options.UnitPreference == "imperial" THEN
				meal = ConvertMealToUnitSystem(meal, UnitSystemImperial)
			END IF

			meals = append(meals, meal)
		END FOR

		RETURN meals, nil
	END
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | Condition | Severity | Recovery Action |
|------------|-----------|----------|-----------------|
| ERR_MEAL_NOT_FOUND | Meal ID does not exist | Low | Validate ID before operations |
| ERR_INVALID_MEAL_TYPE | Type is not "single" or "recipe" | High | Validate input before creation |
| ERR_SINGLE_MEAL_NO_FOOD_ITEM | Single meal missing FoodItemID | High | Require FoodItemID for single meals |
| ERR_RECIPE_EMPTY | Recipe meal has no ingredients | High | Require at least one ingredient |
| ERR_RECIPE_INGREDIENT_NOT_FOUND | Referenced FoodItemID missing | Medium | Validate all ingredients exist |
| ERR_MACRO_CALCULATION_FAILED | Macro aggregation error | High | Log and return error |
| ERR_UNIT_CONVERSION_FAILED | Invalid unit system requested | Low | Default to metric |
| ERR_DATABASE_CONNECTION | DB connectivity issue | Critical | Retry with exponential backoff |
| ERR_CONCURRENT_MODIFICATION | Version conflict on update | Medium | Reload and retry |

### 3.2 State Transitions

```
STATE: Created
  -> VALIDATED: All required fields present
  -> ERROR: Missing required fields

STATE: VALIDATED
  -> STORED: Successful database insert
  -> ERROR: Database constraint violation

STATE: STORED
  -> QUERIED: Retrieved by client
  -> UPDATED: Fields modified
  -> DELETED: Removed from database
  -> ERROR: Cache invalidation failed

STATE: UPDATED
  -> RECALCULATED: Macros recomputed
  -> STORED: Changes persisted
  -> ERROR: Optimistic lock failure
```

### 3.3 Retry Logic for Database Operations

```go
func (r *MealRepository) executeWithRetry(ctx context.Context, operation func() error) error {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := operation(); err != nil {
			lastErr = err

			if isTransientError(err) {
				delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
				time.Sleep(delay)
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, lastErr)
}
```

## 4. Component Interfaces

### 4.1 MealRepository Interface

```go
type MealRepository interface {
	Create(ctx context.Context, input CreateMealInput) (*Meal, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Meal, error)
	Update(ctx context.Context, input UpdateMealInput) (*Meal, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Query(ctx context.Context, options MealQueryOptions) ([]Meal, error)
	Count(ctx context.Context, filter MealQueryOptions) (int, error)
}
```

### 4.2 MealService Interface

```go
type MealService interface {
	CreateMeal(ctx context.Context, input CreateMealInput) (*Meal, error)
	GetMeal(ctx context.Context, id uuid.UUID, unitPreference string) (*Meal, error)
	UpdateMeal(ctx context.Context, input UpdateMealInput) (*Meal, error)
	DeleteMeal(ctx context.Context, id uuid.UUID) error
	ListMeals(ctx context.Context, options MealQueryOptions, unitPreference string) ([]Meal, error)
	ScaleMeal(ctx context.Context, id uuid.UUID, targetGrams float64) (*Meal, error)
	CalculateMacrosForQuantity(ctx context.Context, id uuid.UUID, grams float64) (*Macros, error)
}
```

### 4.3 Internal Helper Functions

```go
func calculateMacrosFromFoodItem(item *FoodItem, quantity float64) Macros {
	factor := quantity / 100.0
	return Macros{
		Protein: item.Macros.Protein * factor,
		Carbs:   item.Macros.Carbs * factor,
		Fat:     item.Macros.Fat * factor,
	}
}

func calculateMicrosFromFoodItem(item *FoodItem, quantity float64) Micros {
	factor := quantity / 100.0
	return Micros{
		Sodium:     item.Micros.Sodium * factor,
		Fiber:      item.Micros.Fiber * factor,
		Sugar:      item.Micros.Sugar * factor,
		Cholesterol: item.Micros.Cholesterol * factor,
		Potassium:  item.Micros.Potassium * factor,
		VitaminA:   item.Micros.VitaminA * factor,
		VitaminC:   item.Micros.VitaminC * factor,
		Calcium:    item.Micros.Calcium * factor,
		Iron:       item.Micros.Iron * factor,
	}
}

func determineDominantPhysicalState(ingredients []RecipeIngredient) PhysicalState {
	solidCount := 0
	liquidCount := 0

	for _, ing := range ingredients {
		item, _ := foodItemRepository.GetByID(ing.FoodItemID)
		if item.PhysicalState == PhysicalStateSolid {
			solidCount++
		} else {
			liquidCount++
		}
	}

	if solidCount >= liquidCount {
		return PhysicalStateSolid
	}
	return PhysicalStateLiquid
}

func calculateAverageUnitWeight(ingredients []RecipeIngredient, totalWeight float64) float64 {
	if totalWeight == 0 {
		return 0
	}

	weightedSum := 0.0
	for _, ing := range ingredients {
		item, _ := foodItemRepository.GetByID(ing.FoodItemID)
		weightedSum += item.AverageUnitWeight * ing.Quantity
	}

	return weightedSum / totalWeight
}

func fetchTagsByIDs(ids []uuid.UUID) []Tag {
	if len(ids) == 0 {
		return []Tag{}
	}
	tags, _ := tagRepository.GetByIDs(ids)
	return tags
}
```

### 4.4 Database Schema Integration

```sql
-- meals table
CREATE TABLE meals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('single', 'recipe')),
    physical_state VARCHAR(20) NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    prep_time INTEGER NOT NULL DEFAULT 0,
    average_unit_weight DECIMAL(10, 2) NOT NULL DEFAULT 0,
    single_food_item_id UUID REFERENCES food_items(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- meal_category_tags junction table
CREATE TABLE meal_category_tags (
    meal_id UUID REFERENCES meals(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES category_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (meal_id, tag_id)
);

-- meal_functionality_tags junction table
CREATE TABLE meal_functionality_tags (
    meal_id UUID REFERENCES meals(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES functionality_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (meal_id, tag_id)
);

-- recipe_ingredients table
CREATE TABLE recipe_ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_id UUID REFERENCES meals(id) ON DELETE CASCADE,
    food_item_id UUID REFERENCES food_items(id) NOT NULL,
    quantity DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Calculated macros view for efficient querying
CREATE VIEW meal_macros_view AS
SELECT
    m.id,
    m.type,
    COALESCE(
        SUM(fi.macros_protein * ri.quantity / 100),
        fi.macros_protein
    ) as protein,
    COALESCE(
        SUM(fi.macros_carbs * ri.quantity / 100),
        fi.macros_carbs
    ) as carbs,
    COALESCE(
        SUM(fi.macros_fat * ri.quantity / 100),
        fi.macros_fat
    ) as fat
FROM meals m
LEFT JOIN recipe_ingredients ri ON m.id = ri.meal_id
LEFT JOIN food_items fi ON ri.food_item_id = fi.id OR m.single_food_item_id = fi.id
GROUP BY m.id, m.type, fi.macros_protein, fi.macros_carbs, fi.macros_fat;
```

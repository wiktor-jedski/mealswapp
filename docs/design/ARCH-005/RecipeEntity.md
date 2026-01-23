# RecipeEntity

**Traceability:** ARCH-005

## 1. Data Structures & Types

```go
package entity

import (
	"time"
	"github.com/google/uuid"
)

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

type MacroValues struct {
	Protein float64 `json:"protein"` // grams per 100g
	Carbs   float64 `json:"carbs"`   // grams per 100g
	Fat     float64 `json:"fat"`     // grams per 100g
}

type MicroValues struct {
	Sodium  float64 `json:"sodium"`  // mg per 100g
	Fiber   float64 `json:"fiber"`   // g per 100g
	// Additional micronutrients stored as JSONB in PostgreSQL
	Others  map[string]float64 `json:"others,omitempty"`
}

type RecipeIngredient struct {
	FoodItemID   uuid.UUID  `json:"foodItemId"`
	Quantity     float64    `json:"quantity"`     // in grams
	FoodItem     *FoodItem  `json:"foodItem,omitempty"` // populated on read
}

type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	TagType   string    `json:"tagType"` // "category" or "functionality"
	CreatedAt time.Time `json:"createdAt"`
}

type RecipeEntity struct {
	ID                 uuid.UUID         `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description,omitempty"`
	Ingredients        []RecipeIngredient `json:"ingredients"`
	PhysicalState      PhysicalState     `json:"physicalState"`
	PrepTime           int               `json:"prepTime"` // in minutes
	AverageUnitWeight  float64           `json:"averageUnitWeight"` // grams
	Instructions       []string          `json:"instructions,omitempty"`
	Servings           int               `json:"servings"`
	CategoryTags       []Tag             `json:"categoryTags"`
	FunctionalityTags  []Tag             `json:"functionalityTags"`
	ImageURL           string            `json:"imageUrl,omitempty"`
	TotalMacros        *MacroValues      `json:"totalMacros"` // calculated, per 100g
	TotalMicros        *MicroValues      `json:"totalMicros"` // calculated, per 100g
	MacrosPerServing   *MacroValues      `json:"macrosPerServing"` // calculated
	MicrosPerServing   *MicroValues      `json:"microsPerServing"` // calculated
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type RecipeCreateInput struct {
	Name              string                    `json:"name" validate:"required,min=1,max=200"`
	Description       string                    `json:"description,omitempty" validate:"max=2000"`
	Ingredients       []RecipeIngredientInput   `json:"ingredients" validate:"required,min=1,dive"`
	PhysicalState     PhysicalState             `json:"physicalState" validate:"required,oneof=solid liquid"`
	PrepTime          int                      `json:"prepTime" validate:"required,min=0"`
	Instructions      []string                  `json:"instructions,omitempty"`
	Servings          int                      `json:"servings" validate:"required,min=1"`
	CategoryTagIDs    []uuid.UUID               `json:"categoryTagIds,omitempty"`
	FunctionalityTagIDs []uuid.UUID             `json:"functionalityTagIds,omitempty"`
	ImageURL          string                    `json:"imageUrl,omitempty" validate:"omitempty,url"`
}

type RecipeIngredientInput struct {
	FoodItemID uuid.UUID `json:"foodItemId" validate:"required"`
	Quantity   float64   `json:"quantity" validate:"required,gt=0"`
}

type RecipeUpdateInput struct {
	Name              *string                   `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description       *string                   `json:"description,omitempty" validate:"omitempty,max=2000"`
	Ingredients       []RecipeIngredientInput   `json:"ingredients,omitempty" validate:"omitempty,dive"`
	PhysicalState     *PhysicalState            `json:"physicalState,omitempty" validate:"omitempty,oneof=solid liquid"`
	PrepTime          *int                      `json:"prepTime,omitempty" validate:"omitempty,min=0"`
	Instructions      []string                  `json:"instructions,omitempty"`
	Servings          *int                      `json:"servings,omitempty" validate:"omitempty,min=1"`
	CategoryTagIDs    []uuid.UUID               `json:"categoryTagIds,omitempty"`
	FunctionalityTagIDs []uuid.UUID             `json:"functionalityTagIds,omitempty"`
	ImageURL          *string                   `json:"imageUrl,omitempty" validate:"omitempty,url"`
}

type RecipeQueryFilter struct {
	CategoryTagIDs     []uuid.UUID `query:"categoryTagIds"`
	FunctionalityTagIDs []uuid.UUID `query:"functionalityTagIds"`
	PhysicalState      *PhysicalState `query:"physicalState"`
	MinPrepTime        *int `query:"minPrepTime"`
	MaxPrepTime        *int `query:"maxPrepTime"`
	MinProtein         *float64 `query:"minProtein"`
	MaxProtein         *float64 `query:"maxProtein"`
	MinCarbs           *float64 `query:"minCarbs"`
	MaxCarbs           *float64 `query:"maxCarbs"`
	MinFat             *float64 `query:"minFat"`
	MaxFat             *float64 `query:"maxFat"`
	SearchName         string `query:"search"`
}

type RecipeRepository interface {
	Create(ctx context.Context, input *RecipeCreateInput) (*RecipeEntity, error)
	Update(ctx context.Context, id uuid.UUID, input *RecipeUpdateInput) (*RecipeEntity, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*RecipeEntity, error)
	List(ctx context.Context, filter *RecipeQueryFilter, offset, limit int) ([]*RecipeEntity, error)
	Count(ctx context.Context, filter *RecipeQueryFilter) (int, error)
	GetIngredients(ctx context.Context, recipeID uuid.UUID) ([]RecipeIngredient, error)
	CalculateMacros(ctx context.Context, recipeID uuid.UUID) (*MacroValues, *MicroValues, error)
}

type UnitConverter interface {
	GramsToOunces(grams float64) float64
	OuncesToGrams(ounces float64) float64
	MillilitersToFluidOunces(ml float64) float64
	FluidOuncesToMilliliters(flOz float64) float64
	ConvertMacros(macros *MacroValues, fromState PhysicalState, toSystem UnitSystem) *MacroValues
}
```

## 2. Logic & Algorithms

### 2.1 Recipe Creation Flow

```
FUNCTION Create(ctx context.Context, input *RecipeCreateInput) (*RecipeEntity, error)
	1. Validate input fields
	   - name: non-empty, <= 200 chars
	   - ingredients: non-empty array, each has valid FoodItemID and quantity > 0
	   - physicalState: must be "solid" or "liquid"
	   - prepTime: >= 0
	   - servings: >= 1

	2. Verify all FoodItemIDs exist in database
	   FOR EACH ingredient IN input.Ingredients
		   IF NOT EXISTS FoodItem WHERE id = ingredient.FoodItemID
			   RETURN Error(ErrFoodItemNotFound, "ingredient food item not found")
	   END FOR

	3. Fetch category tags if provided
	   categoryTags = FETCH Tags WHERE id IN input.CategoryTagIDs AND tagType = "category"
	   IF LENGTH(categoryTags) != LENGTH(input.CategoryTagIDs)
		   RETURN Error(ErrTagNotFound, "one or more category tags not found")

	4. Fetch functionality tags if provided
	   functionalityTags = FETCH Tags WHERE id IN input.FunctionalityTagIDs AND tagType = "functionality"
	   IF LENGTH(functionalityTags) != LENGTH(input.FunctionalityTagIDs)
		   RETURN Error(ErrTagNotFound, "one or more functionality tags not found")

	5. Calculate average unit weight
	   totalWeight = 0
	   FOR EACH ingredient IN input.Ingredients
		   foodItem = FETCH FoodItem WHERE id = ingredient.FoodItemID
		   totalWeight += foodItem.averageUnitWeight * ingredient.Quantity / 100
	   END FOR
	   averageUnitWeight = totalWeight / LENGTH(input.Ingredients)

	6. Determine physical state from majority of ingredients
	   solidCount = COUNT ingredients WHERE foodItem.physicalState = "solid"
	   liquidCount = COUNT ingredients WHERE foodItem.physicalState = "liquid"
	   physicalState = IF solidCount >= liquidCount THEN "solid" ELSE "liquid"

	7. Create RecipeEntity with calculated macros
	   macros, micros = CalculateTotalMacros(input.Ingredients)

	8. INSERT INTO recipes table
	   recipe = EXECUTE SQL INSERT ...

	9. INSERT INTO recipe_ingredients junction table
	   FOR EACH ingredient IN input.Ingredients
		   EXECUTE SQL INSERT INTO recipe_ingredients ...
	   END FOR

	10. INSERT INTO recipe_tags junction table
	   FOR EACH tag IN categoryTags
		   EXECUTE SQL INSERT INTO recipe_tags ...
	   END FOR
	   FOR EACH tag IN functionalityTags
		   EXECUTE SQL INSERT INTO recipe_tags ...
	   END FOR

	11. RETURN populated RecipeEntity
END FUNCTION
```

### 2.2 Macro Calculation Algorithm

```
FUNCTION CalculateTotalMacros(ingredients []RecipeIngredientInput) (*MacroValues, *MicroValues)
	macros = NEW MacroValues with all zeros
	micros = NEW MicroValues with all zeros
	totalWeight = 0

	FOR EACH ingredient IN ingredients
		foodItem = FETCH FoodItem WHERE id = ingredient.FoodItemID

		// Scale macros by quantity (ingredient.Quantity is in grams)
		scaleFactor = ingredient.Quantity / 100.0

		macros.protein += foodItem.macros.protein * scaleFactor
		macros.carbs += foodItem.macros.carbs * scaleFactor
		macros.fat += foodItem.macros.fat * scaleFactor

		micros.sodium += foodItem.micros.sodium * scaleFactor
		micros.fiber += foodItem.micros.fiber * scaleFactor

		FOR EACH (key, value) IN foodItem.micros.others
			micros.others[key] += value * scaleFactor
		END FOR

		totalWeight += ingredient.Quantity
	END FOR

	// Normalize to per 100g
	IF totalWeight > 0
		macros.protein = (macros.protein / totalWeight) * 100
		macros.carbs = (macros.carbs / totalWeight) * 100
		macros.fat = (macros.fat / totalWeight) * 100

		micros.sodium = (micros.sodium / totalWeight) * 100
		micros.fiber = (micros.fiber / totalWeight) * 100

		FOR EACH key IN micros.others
			micros.others[key] = (micros.others[key] / totalWeight) * 100
		END FOR
	END IF

	RETURN macros, micros
END FUNCTION
```

### 2.3 Unit Conversion Algorithm

```
FUNCTION ConvertMacros(macros *MacroValues, fromState PhysicalState, toSystem UnitSystem) *MacroValues
	IF toSystem == UnitSystemMetric
		RETURN macros // Already in metric (per 100g)
	END IF

	// Convert from per 100g to per 100oz
	gramsPerOunce = 28.3495
	gramsIn100oz = 100 * gramsPerOunce

	converted = NEW MacroValues
	converted.protein = macros.protein * gramsIn100oz / 100
	converted.carbs = macros.carbs * gramsIn100oz / 100
	converted.fat = macros.fat * gramsIn100oz / 100

	RETURN converted
END FUNCTION
```

### 2.4 Recipe Scaling Algorithm

```
FUNCTION ScaleRecipe(recipe *RecipeEntity, targetServings int) (*MacroValues, *MacroValues, error)
	IF targetServings <= 0
		RETURN nil, nil, Error(ErrInvalidServings, "servings must be positive")
	END IF

	scaleFactor = FLOAT(targetServings) / FLOAT(recipe.Servings)

	// Scale macros per serving
	macrosPerServing = NEW MacroValues
	macrosPerServing.protein = recipe.TotalMacros.protein * scaleFactor
	macrosPerServing.carbs = recipe.TotalMacros.carbs * scaleFactor
	macrosPerServing.fat = recipe.TotalMacros.fat * scaleFactor

	// Calculate new total macros for scaled recipe
	totalMacros = NEW MacroValues
	totalMacros.protein = macrosPerServing.protein * targetServings
	totalMacros.carbs = macrosPerServing.carbs * targetServings
	totalMacros.fat = macrosPerServing.fat * targetServings

	RETURN totalMacros, macrosPerServing, nil
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Type | Condition | HTTP Status | Recovery Action |
|------------|-----------|-------------|-----------------|
| ErrFoodItemNotFound | Ingredient references non-existent FoodItem | 400/404 | Validate FoodItemIDs before creation |
| ErrTagNotFound | Tag ID doesn't exist or wrong type | 400/404 | Validate tag IDs before update |
| ErrRecipeNotFound | GetByID/Update/Delete on missing recipe | 404 | Check recipe exists before operations |
| ErrInvalidPhysicalState | Invalid physical state value | 400 | Validate input enum values |
| ErrInvalidServings | Servings <= 0 or non-integer | 400 | Validate servings in input |
| ErrCircularDependency | Recipe contains itself (future feature) | 400 | Validate ingredient tree |
| ErrIngredientConflict | Duplicate FoodItemID in ingredients | 400 | Deduplicate before processing |
| ErrDatabaseConnection | PostgreSQL unavailable | 503 | Retry with exponential backoff |
| ErrTransactionRollback | Create/Update transaction failed | 500 | Log error, suggest retry |
| ErrMacrosCalculation | Division by zero in macro calc | 500 | Validate total weight > 0 |

### 3.2 State Transitions

```
State Machine: Recipe Lifecycle

[NEW] -> [VALIDATING] -> [CREATING] -> [ACTIVE]
   |          |              |
   |          |              v
   |          |          [FAILED] -> [ROLLBACK] -> [NEW]
   |          |
   v          v
[FAILED] -> [CLEANUP]

CREATE:
  NEW -> VALIDATING: Input validation begins
  VALIDATING -> CREATING: All references validated
  CREATING -> ACTIVE: Database inserts complete
  CREATING -> FAILED: Database error or constraint violation
  FAILED -> ROLLBACK: Reverse partial changes
  ROLLBACK -> NEW: Ready for retry

UPDATE:
  ACTIVE -> VALIDATING: Update validation begins
  VALIDATING -> CREATING: Update validated
  CREATING -> ACTIVE: Update complete
  CREATING -> FAILED: Update failed
  FAILED -> ACTIVE: Rollback to previous state

DELETE:
  ACTIVE -> DELETING: Soft/hard delete initiated
  DELETING -> DELETED: Record removed
```

### 3.3 Retry Logic

```go
func (r *recipeRepository) withRetry(ctx context.Context, maxRetries int, fn func() error) error {
	backoff := time.Millisecond * 100
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := fn(); err != nil {
			if isRetryable(err) && attempt < maxRetries {
				time.Sleep(backoff)
				backoff *= 2 // Exponential backoff
				continue
			}
			return err
		}
		return nil
	}
	return nil
}
```

## 4. Component Interfaces

### 4.1 Repository Interface Methods

```go
type RecipeRepository interface {
	// Create inserts a new recipe with ingredients and tags
	Create(ctx context.Context, input *RecipeCreateInput) (*RecipeEntity, error)

	// Update modifies an existing recipe's fields
	Update(ctx context.Context, id uuid.UUID, input *RecipeUpdateInput) (*RecipeEntity, error)

	// Delete removes a recipe by ID (hard delete)
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a recipe with all relationships populated
	GetByID(ctx context.Context, id uuid.UUID) (*RecipeEntity, error)

	// List returns recipes matching filter with pagination
	List(ctx context.Context, filter *RecipeQueryFilter, offset, limit int) ([]*RecipeEntity, error)

	// Count returns total number of recipes matching filter
	Count(ctx context.Context, filter *RecipeQueryFilter) (int, error)

	// GetIngredients returns all ingredients for a recipe
	GetIngredients(ctx context.Context, recipeID uuid.UUID) ([]RecipeIngredient, error)

	// CalculateMacros computes total macros for a recipe on demand
	CalculateMacros(ctx context.Context, recipeID uuid.UUID) (*MacroValues, *MicroValues, error)
}
```

### 4.2 SQL Queries

```sql
-- Create Recipe
INSERT INTO recipes (
	id, name, description, physical_state, prep_time, 
	average_unit_weight, servings, instructions, image_url,
	total_macros_protein, total_macros_carbs, total_macros_fat,
	total_micros_sodium, total_micros_fiber, total_micros_others,
	created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- Create Recipe Ingredient
INSERT INTO recipe_ingredients (
	recipe_id, food_item_id, quantity, created_at
) VALUES ($1, $2, $3, $4);

-- Get Recipe by ID with relations
SELECT r.*,
       json_agg(DISTINCT jsonb_build_object('id', t.id, 'name', t.name, 'tagType', t.tag_type)) FILTER (WHERE t.id IS NOT NULL) as tags,
       json_agg(DISTINCT jsonb_build_object('foodItemId', ri.food_item_id, 'quantity', ri.quantity)) FILTER (WHERE ri.food_item_id IS NOT NULL) as ingredients
FROM recipes r
LEFT JOIN recipe_tags rt ON r.id = rt.recipe_id
LEFT JOIN tags t ON rt.tag_id = t.id
LEFT JOIN recipe_ingredients ri ON r.id = ri.recipe_id
WHERE r.id = $1
GROUP BY r.id;

-- List with filters
SELECT r.*,
       json_agg(DISTINCT t.id) FILTER (WHERE t.id IS NOT NULL) as category_tag_ids,
       json_agg(DISTINCT t.id) FILTER (WHERE t.id IS NOT NULL) as functionality_tag_ids
FROM recipes r
LEFT JOIN recipe_tags rt ON r.id = rt.recipe_id
LEFT JOIN tags t ON rt.tag_id = t.id
WHERE 
  ($1::uuid[] IS NULL OR t.id = ANY($1))
  AND ($2::text IS NULL OR r.name ILIKE '%' || $2 || '%')
  AND ($3::text IS NULL OR r.physical_state = $3)
GROUP BY r.id
ORDER BY r.created_at DESC
LIMIT $4 OFFSET $5;

-- Calculate macros for recipe
SELECT 
  SUM(f.macros_protein * ri.quantity / 100.0) as protein,
  SUM(f.macros_carbs * ri.quantity / 100.0) as carbs,
  SUM(f.macros_fat * ri.quantity / 100.0) as fat,
  SUM(f.micros_sodium * ri.quantity / 100.0) as sodium,
  SUM(f.micros_fiber * ri.quantity / 100.0) as fiber
FROM recipe_ingredients ri
JOIN food_items f ON ri.food_item_id = f.id
WHERE ri.recipe_id = $1;
```

### 4.3 Service Layer Interface

```go
type RecipeService interface {
	CreateRecipe(ctx context.Context, input *RecipeCreateInput) (*RecipeEntity, error)
	UpdateRecipe(ctx context.Context, id uuid.UUID, input *RecipeUpdateInput) (*RecipeEntity, error)
	DeleteRecipe(ctx context.Context, id uuid.UUID) error
	GetRecipe(ctx context.Context, id uuid.UUID) (*RecipeEntity, error)
	ListRecipes(ctx context.Context, filter *RecipeQueryFilter, page, pageSize int) ([]*RecipeEntity, int, error)
	GetRecipeWithUnitConversion(ctx context.Context, id uuid.UUID, unitSystem UnitSystem) (*RecipeEntity, error)
	ScaleRecipe(ctx context.Context, id uuid.UUID, targetServings int) (*RecipeEntity, error)
}
```

### 4.4 Unit Converter Interface

```go
type UnitConverter interface {
	GramsToOunces(grams float64) float64
	OuncesToGrams(ounces float64) float64
	MillilitersToFluidOunces(ml float64) float64
	FluidOuncesToMilliliters(flOz float64) float64
	ConvertMacros(macros *MacroValues, fromState PhysicalState, toSystem UnitSystem) *MacroValues
	ConvertMicros(micros *MicroValues, fromState PhysicalState, toSystem UnitSystem) *MicroValues
}
```

# FILE: FoodItemEntity.md
**Traceability:** ARCH-005

## 1. Data Structures & Types

```go
package entity

import (
	"time"
	"github.com/google/uuid"
)

// PhysicalState represents the physical state of a food item
type PhysicalState string

const (
	PhysicalStateSolid  PhysicalState = "solid"
	PhysicalStateLiquid PhysicalState = "liquid"
)

// Macros represents macronutrient values per 100g or 100ml
type Macros struct {
	Protein float64 `json:"protein"` // grams
	Carbs   float64 `json:"carbs"`   // grams
	Fat     float64 `json:"fat"`     // grams
}

// Micros represents micronutrient values per 100g or 100ml
type Micros struct {
	Sodium   float64            `json:"sodium"`   // mg
	Fiber    float64            `json:"fiber"`    // g
	Sugar    float64            `json:"sugar"`    // g
	Cholesterol float64         `json:"cholesterol"` // mg
	Potassium float64           `json:"potassium"`   // mg
	VitaminA  float64           `json:"vitamin_a"`   // mcg
	VitaminC  float64           `json:"vitamin_c"`   // mg
	Calcium   float64           `json:"calcium"`     // mg
	Iron      float64           `json:"iron"`        // mg
	Additional map[string]float64 `json:"additional,omitempty"` // extensibility for additional micros
}

// Tag represents a categorization or functionality tag
type Tag struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TagType     TagType   `json:"tag_type"`
	ColorHex    string    `json:"color_hex,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TagType distinguishes between category and functionality tags
type TagType string

const (
	TagTypeCategory       TagType = "category"
	TagTypeFunctionality  TagType = "functionality"
)

// FoodItem represents a single food item in the domain model
type FoodItem struct {
	ID                   uuid.UUID    `json:"id"`
	Name                 string       `json:"name"`
	PhysicalState        PhysicalState `json:"physical_state"`
	PrepTime             int          `json:"prep_time"` // minutes
	AverageUnitWeight    float64      `json:"average_unit_weight"` // grams
	Macros               Macros       `json:"macros"`
	Micros               Micros       `json:"micros"`
	CategoryTags         []Tag        `json:"category_tags"`
	FunctionalityTags    []Tag        `json:"functionality_tags"`
	ImageURL             *string      `json:"image_url,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// UnitPreference defines the user's preferred unit system
type UnitPreference string

const (
	UnitPreferenceMetric    UnitPreference = "metric"
	UnitPreferenceImperial  UnitPreference = "imperial"
)

// FoodItemQuery represents filtering criteria for food item queries
type FoodItemQuery struct {
	IDs               []uuid.UUID    `query:"ids,omitempty"`
	Name              *string        `query:"name,omitempty"`
	PhysicalState     *PhysicalState `query:"physical_state,omitempty"`
	CategoryTagIDs    []uuid.UUID    `query:"category_tag_ids,omitempty"`
	FunctionalityTagIDs []uuid.UUID   `query:"functionality_tag_ids,omitempty"`
	MinProtein        *float64       `query:"min_protein,omitempty"`
	MaxProtein        *float64       `query:"max_protein,omitempty"`
	MinCarbs          *float64       `query:"min_carbs,omitempty"`
	MaxCarbs          *float64       `query:"max_carbs,omitempty"`
	MinFat            *float64       `query:"min_fat,omitempty"`
	MaxFat            *float64       `query:"max_fat,omitempty"`
	Page              int            `query:"page"`
	PageSize          int            `query:"page_size"`
	SortBy            string         `query:"sort_by"`
	SortOrder         string         `query:"sort_order"`
}

// FoodItemCreate represents the input for creating a new food item
type FoodItemCreate struct {
	Name                 string       `json:"name" validate:"required,min=1,max=255"`
	PhysicalState        PhysicalState `json:"physical_state" validate:"required,oneof=solid liquid"`
	PrepTime             int          `json:"prep_time" validate:"required,min=0"`
	AverageUnitWeight    float64      `json:"average_unit_weight" validate:"required,min=0"`
	Macros               Macros       `json:"macros" validate:"required"`
	Micros               Micros       `json:"micros"`
	CategoryTagIDs       []uuid.UUID  `json:"category_tag_ids"`
	FunctionalityTagIDs  []uuid.UUID  `json:"functionality_tag_ids"`
	ImageURL             *string      `json:"image_url,omitempty"`
}

// FoodItemUpdate represents the input for updating an existing food item
type FoodItemUpdate struct {
	Name                *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	PhysicalState       *PhysicalState `json:"physical_state,omitempty" validate:"omitempty,oneof=solid liquid"`
	PrepTime            *int           `json:"prep_time,omitempty" validate:"omitempty,min=0"`
	AverageUnitWeight   *float64       `json:"average_unit_weight,omitempty" validate:"omitempty,min=0"`
	Macros              *Macros        `json:"macros,omitempty"`
	Micros              *Micros        `json:"micros,omitempty"`
	CategoryTagIDs      []uuid.UUID    `json:"category_tag_ids,omitempty"`
	FunctionalityTagIDs []uuid.UUID    `json:"functionality_tag_ids,omitempty"`
	ImageURL            *string        `json:"image_url,omitempty"`
}

// FoodItemResponse represents the API response for a food item
type FoodItemResponse struct {
	ID                uuid.UUID      `json:"id"`
	Name              string         `json:"name"`
	PhysicalState     PhysicalState  `json:"physical_state"`
	PrepTime          int            `json:"prep_time"`
	AverageUnitWeight float64        `json:"average_unit_weight"`
	Macros            Macros         `json:"macros"`
	Micros            Micros         `json:"micros"`
	CategoryTags      []Tag          `json:"category_tags"`
	FunctionalityTags []Tag          `json:"functionality_tags"`
	ImageURL          *string        `json:"image_url,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// ConvertedFoodItem represents a food item with macros in user-preferred units
type ConvertedFoodItem struct {
	*FoodItem
	ConvertedMacros Macros `json:"converted_macros"`
	UnitPreference  UnitPreference `json:"unit_preference"`
	DisplayWeight   float64 `json:"display_weight"` // in preferred units
}

// ScaledFoodItem represents a food item scaled to a specific quantity
type ScaledFoodItem struct {
	*ConvertedFoodItem
	OriginalQuantity float64 `json:"original_quantity"` // grams/ml
	ScaledQuantity   float64 `json:"scaled_quantity"`   // in user units
	ScaledMacros     Macros  `json:"scaled_macros"`
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Create FoodItem

```
FUNCTION CreateFoodItem(input FoodItemCreate) RETURNS (*FoodItem, error)
	BEGIN
		1. Validate input fields
		   - Name: not empty, max 255 characters
		   - PhysicalState: must be "solid" or "liquid"
		   - PrepTime: >= 0
		   - AverageUnitWeight: >= 0
		   - Macros: all values >= 0

		2. Generate new UUID for the food item

		3. Fetch category tags by IDs
		   - IF any ID not found, RETURN error "invalid_category_tag_id"

		4. Fetch functionality tags by IDs
		   - IF any ID not found, RETURN error "invalid_functionality_tag_id"

		5. Normalize macros to per 100g/100ml
		   - IF input values already normalized, use as-is
		   - ELSE divide by input quantity and multiply by 100

		6. Validate micros structure
		   - Ensure all values >= 0
		   - Apply default values for missing required fields

		7. Create FoodItem struct with normalized data

		8. Insert into database
		   - Execute INSERT INTO food_items ...
		   - Execute INSERT INTO food_item_category_tags ...
		   - Execute INSERT INTO food_item_functionality_tags ...

		9. Return created FoodItem with timestamps
	END
```

### 2.2 GetFoodItemByID

```
FUNCTION GetFoodItemByID(id uuid.UUID, unitPref UnitPreference) RETURNS (*ConvertedFoodItem, error)
	BEGIN
		1. Query food_items table by ID
		   - IF not found, RETURN error "food_item_not_found"

		2. Load associated category tags
		   - JOIN with tags table WHERE tag_type = "category"

		3. Load associated functionality tags
		   - JOIN with tags table WHERE tag_type = "functionality"

		4. Build FoodItem from database row

		5. Convert macros based on unit preference
		   - IF unitPref = imperial AND physicalState = solid
			 - Convert grams to ounces (g / 28.3495)
		   - IF unitPref = imperial AND physicalState = liquid
			 - Convert ml to fl oz (ml / 29.5735)
		   - ELSE keep as metric

		6. Calculate display weight
		   - IF unitPref = imperial AND physicalState = solid
			 - displayWeight = averageUnitWeight / 28.3495 (oz)
		   - IF unitPref = imperial AND physicalState = liquid
			 - displayWeight = averageUnitWeight / 29.5735 (fl oz)
		   - ELSE displayWeight = averageUnitWeight (g/ml)

		7. Return ConvertedFoodItem
	END
```

### 2.3 ListFoodItems

```
FUNCTION ListFoodItems(query FoodItemQuery) RETURNS ([]ConvertedFoodItem, int64, error)
	BEGIN
		1. Build dynamic WHERE clause based on query parameters
		   - IF IDs provided: id = ANY($1)
		   - IF Name provided: name ILIKE $2
		   - IF PhysicalState provided: physical_state = $3
		   - IF CategoryTagIDs provided: EXISTS subquery matching tags
		   - IF FunctionalityTagIDs provided: EXISTS subquery matching tags
		   - IF MinProtein provided: macros->>'protein' >= $n
		   - IF MaxProtein provided: macros->>'protein' <= $n
		   - Similar for Carbs and Fat

		2. Build ORDER BY clause
		   - sortBy: name, prep_time, created_at, macros->>'protein', etc.
		   - sortOrder: ASC or DESC

		3. Execute paginated query with COUNT
		   - SELECT COUNT(*) FROM food_items WHERE ...
		   - SELECT * FROM food_items WHERE ... LIMIT $n OFFSET $n

		4. For each row:
		   - Build FoodItem struct
		   - Load associated tags (two separate queries or JOIN)
		   - Convert units based on default unit preference
		   - Append to results

		5. Return results, total count, and pagination info
	END
```

### 2.4 UpdateFoodItem

```
FUNCTION UpdateFoodItem(id uuid.UUID, input FoodItemUpdate) RETURNS (*FoodItem, error)
	BEGIN
		1. Fetch existing food item
		   - IF not found, RETURN error "food_item_not_found"

		2. Apply updates (only non-nil fields)
		   - IF Name provided: item.Name = *input.Name
		   - IF PhysicalState provided: item.PhysicalState = *input.PhysicalState
		   - IF PrepTime provided: item.PrepTime = *input.PrepTime
		   - IF AverageUnitWeight provided: item.AverageUnitWeight = *input.AverageUnitWeight
		   - IF Macros provided: item.Macros = *input.Macros
		   - IF Micros provided: item.Micros = *input.Micros
		   - IF ImageURL provided: item.ImageURL = input.ImageURL

		3. Validate updated macros/micros (all >= 0)

		4. IF CategoryTagIDs provided:
		   - DELETE FROM food_item_category_tags WHERE food_item_id = $1
		   - INSERT INTO food_item_category_tags ... (new IDs)

		5. IF FunctionalityTagIDs provided:
		   - DELETE FROM food_item_functionality_tags WHERE food_item_id = $1
		   - INSERT INTO food_item_functionality_tags ... (new IDs)

		6. Update food_items table
		   - UPDATE food_items SET name=$1, physical_state=$2, ... WHERE id=$n

		7. Update UpdatedAt timestamp

		8. Return updated FoodItem
	END
```

### 2.5 DeleteFoodItem

```
FUNCTION DeleteFoodItem(id uuid.UUID) RETURNS error
	BEGIN
		1. Check if food item exists
		   - IF not found, RETURN error "food_item_not_found"

		2. Check if item is used in any recipes
		   - SELECT COUNT(*) FROM recipe_ingredients WHERE food_item_id = $1
		   - IF count > 0, RETURN error "food_item_in_use"

		3. Delete associated tag mappings
		   - DELETE FROM food_item_category_tags WHERE food_item_id = $1
		   - DELETE FROM food_item_functionality_tags WHERE food_item_id = $1

		4. Delete the food item
		   - DELETE FROM food_items WHERE id = $1

		5. RETURN nil on success
	END
```

### 2.6 ScaleFoodItemMacros

```
FUNCTION ScaleFoodItemMacros(item *FoodItem, quantity float64, unitPref UnitPreference) RETURNS (*ScaledFoodItem, error)
	BEGIN
		1. Validate quantity (must be > 0)

		2. Get converted food item
		   - converted = GetConvertedFoodItem(item, unitPref)

		3. Calculate scaling factor
		   - scaleFactor = quantity / 100.0

		4. Scale macros
		   - scaledMacros.Protein = convertedMacros.Protein * scaleFactor
		   - scaledMacros.Carbs = convertedMacros.Carbs * scaleFactor
		   - scaledMacros.Fat = convertedMacros.Fat * scaleFactor

		5. Calculate scaled quantity in display units
		   - IF unitPref = imperial AND physicalState = solid
			 scaledQuantity = quantity / 28.3495 (oz)
		   - ELSE scaledQuantity = quantity (g/ml)

		6. Return ScaledFoodItem with all calculated values
	END
```

### 2.7 NormalizeMacros

```
FUNCTION NormalizeMacros(value float64, originalUnit string) RETURNS float64
	BEGIN
		SWITCH originalUnit
			CASE "per_100g", "per_100ml":
				RETURN value
			CASE "per_serving":
				RETURN value * (100.0 / servingSizeGrams)
			CASE "per_oz":
				RETURN value * 3.5274 // oz to g
			CASE "per_lb":
				RETURN value * 0.22046 // lb to kg, then to per 100g
			CASE "per_cup":
				RETURN value * (100.0 / cupsToGrams[foodType])
			DEFAULT:
				RETURN value // assume already normalized
		END SWITCH
	END
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Error Code | HTTP Status | Recovery Action |
|----------------|------------|-------------|-----------------|
| Food item not found | `err_food_item_not_found` | 404 | Verify ID, check if deleted |
| Invalid category tag ID | `err_invalid_category_tag_id` | 400 | Provide valid tag IDs |
| Invalid functionality tag ID | `err_invalid_functionality_tag_id` | 400 | Provide valid tag IDs |
| Food item in use | `err_food_item_in_use` | 409 | Remove from recipes first |
| Invalid unit preference | `err_invalid_unit_preference` | 400 | Use "metric" or "imperial" |
| Invalid physical state | `err_invalid_physical_state` | 400 | Use "solid" or "liquid" |
| Negative macro value | `err_negative_macro_value` | 400 | Ensure all macros >= 0 |
| Quantity out of range | `err_quantity_out_of_range` | 400 | Quantity must be > 0 |
| Database connection failure | `err_db_connection` | 503 | Retry with backoff |
| Unique constraint violation | `err_unique_violation` | 409 | Use different name |

### 3.2 State Transitions

```
Initial State: NEW

NEW → CREATED: CreateFoodItem succeeds
NEW → VALIDATION_ERROR: Input validation fails
NEW → DATABASE_ERROR: Database operation fails

CREATED → ACTIVE: Item available for queries
ACTIVE → UPDATED: UpdateFoodItem succeeds
ACTIVE → DELETED: DeleteFoodItem succeeds
ACTIVE → ERROR: Tag reference becomes invalid

DELETED → NOT_FOUND: Subsequent lookups return 404
```

### 3.3 Validation States

```go
// ValidationError represents validation failures
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// State enum for food item lifecycle
type FoodItemState string

const (
	FoodItemStateNew       FoodItemState = "new"
	FoodItemStateActive    FoodItemState = "active"
	FoodItemStateDeprecated FoodItemState = "deprecated"
	FoodItemStateDeleted   FoodItemState = "deleted"
)
```

## 4. Component Interfaces

### 4.1 FoodItemRepository Interface

```go
package repository

import (
	"context"
	"github.com/google/uuid"
	"mealswapp/internal/entity"
)

// FoodItemRepository defines the contract for food item data access
type FoodItemRepository interface {
	Create(ctx context.Context, item *entity.FoodItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.FoodItem, error)
	List(ctx context.Context, query entity.FoodItemQuery) ([]*entity.FoodItem, int64, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.FoodItem, error)
	Count(ctx context.Context, query entity.FoodItemQuery) (int64, error)
}
```

### 4.2 FoodItemService Interface

```go
package service

import (
	"context"
	"github.com/google/uuid"
	"mealswapp/internal/entity"
)

// FoodItemService defines the business logic interface
type FoodItemService interface {
	CreateFoodItem(ctx context.Context, input entity.FoodItemCreate) (*entity.FoodItem, error)
	GetFoodItem(ctx context.Context, id uuid.UUID, unitPref entity.UnitPreference) (*entity.ConvertedFoodItem, error)
	ListFoodItems(ctx context.Context, query entity.FoodItemQuery) ([]entity.ConvertedFoodItem, int64, error)
	UpdateFoodItem(ctx context.Context, id uuid.UUID, input entity.FoodItemUpdate) (*entity.FoodItem, error)
	DeleteFoodItem(ctx context.Context, id uuid.UUID) error
	ScaleFoodItem(ctx context.Context, id uuid.UUID, quantity float64, unitPref entity.UnitPreference) (*entity.ScaledFoodItem, error)
	ValidateFoodItem(ctx context.Context, input entity.FoodItemCreate) []entity.ValidationError
}
```

### 4.3 TagRepository Interface

```go
package repository

import (
	"context"
	"github.com/google/uuid"
	"mealswapp/internal/entity"
)

// TagRepository defines tag data access operations
type TagRepository interface {
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.Tag, error)
	GetByType(ctx context.Context, tagType entity.TagType) ([]entity.Tag, error)
	GetCategoryTags(ctx context.Context) ([]entity.Tag, error)
	GetFunctionalityTags(ctx context.Context) ([]entity.Tag, error)
	Create(ctx context.Context, tag *entity.Tag) error
}
```

### 4.4 Handler Functions

```go
package handler

import (
	"github.com/gofiber/fiber"
	"github.com/google/uuid"
	"mealswapp/internal/entity"
)

// FoodItemHandler handles HTTP requests for food items
type FoodItemHandler interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	Scale(c *fiber.Ctx) error
}

// Create handles POST /api/v1/food-items
// @Summary Create a new food item
// @Description Creates a new food item with the provided data
// @Tags food-items
// @Accept json
// @Produce json
// @Param food_item body entity.FoodItemCreate true "Food item data"
// @Success 201 {object} entity.FoodItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/food-items [post]
func (h *foodItemHandler) Create(c *fiber.Ctx) error {
	var input entity.FoodItemCreate
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_request_body",
			Message: "Failed to parse request body",
		})
	}

	item, err := h.service.CreateFoodItem(c.Context(), input)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(201).JSON(entity.FoodItemResponse{
		ID:                item.ID,
		Name:              item.Name,
		PhysicalState:     item.PhysicalState,
		PrepTime:          item.PrepTime,
		AverageUnitWeight: item.AverageUnitWeight,
		Macros:            item.Macros,
		Micros:            item.Micros,
		CategoryTags:      item.CategoryTags,
		FunctionalityTags: item.FunctionalityTags,
		ImageURL:          item.ImageURL,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	})
}

// GetByID handles GET /api/v1/food-items/:id
func (h *foodItemHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	unitPref := entity.UnitPreference(c.Query("units", "metric"))
	item, err := h.service.GetFoodItem(c.Context(), id, unitPref)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(item)
}

// List handles GET /api/v1/food-items
func (h *foodItemHandler) List(c *fiber.Ctx) error {
	query := entity.FoodItemQuery{
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
		SortBy:   c.Query("sort_by", "name"),
		SortOrder: c.Query("sort_order", "asc"),
	}

	items, total, err := h.service.ListFoodItems(c.Context(), query)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(ListResponse{
		Data:       items,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: (int(total) + query.PageSize - 1) / query.PageSize,
	})
}

// Update handles PUT /api/v1/food-items/:id
func (h *foodItemHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	var input entity.FoodItemUpdate
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_request_body",
			Message: "Failed to parse request body",
		})
	}

	item, err := h.service.UpdateFoodItem(c.Context(), id, input)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(item)
}

// Delete handles DELETE /api/v1/food-items/:id
func (h *foodItemHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	if err := h.service.DeleteFoodItem(c.Context(), id); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(204).JSON(nil)
}

// Scale handles GET /api/v1/food-items/:id/scale
func (h *foodItemHandler) Scale(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid food item ID format",
		})
	}

	quantity := c.QueryFloat("quantity", 100)
	unitPref := entity.UnitPreference(c.Query("units", "metric"))

	scaled, err := h.service.ScaleFoodItem(c.Context(), id, quantity, unitPref)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(scaled)
}
```

### 4.5 Error Response Types

```go
package handler

// ErrorResponse represents a standard API error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}
```

### 4.6 Database Schema (PostgreSQL)

```sql
-- Food items table
CREATE TABLE food_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    physical_state VARCHAR(20) NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    prep_time INTEGER NOT NULL DEFAULT 0,
    average_unit_weight DECIMAL(10, 2) NOT NULL DEFAULT 0,
    macros JSONB NOT NULL DEFAULT '{"protein": 0, "carbs": 0, "fat": 0}',
    micros JSONB NOT NULL DEFAULT '{}',
    image_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_food_items_name ON food_items(name);
CREATE INDEX idx_food_items_physical_state ON food_items(physical_state);
CREATE INDEX idx_food_items_macros ON food_items USING GIN (macros);
CREATE INDEX idx_food_items_created_at ON food_items(created_at DESC);

-- Category tags junction table
CREATE TABLE food_item_category_tags (
    food_item_id UUID REFERENCES food_items(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (food_item_id, tag_id)
);

-- Functionality tags junction table
CREATE TABLE food_item_functionality_tags (
    food_item_id UUID REFERENCES food_items(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (food_item_id, tag_id)
);

-- Unique constraint for food item name
CREATE UNIQUE INDEX idx_food_items_name_unique ON food_items(name);
```

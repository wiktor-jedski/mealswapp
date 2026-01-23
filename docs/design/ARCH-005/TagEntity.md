# FILE: TagEntity.md

**Traceability:** ARCH-005

## 1. Data Structures & Types

```go
package entity

import (
	"time"
)

// TagType represents the category of a tag
type TagType string

const (
	TagTypeCategory      TagType = "category"       // Dietary category tags (e.g., "vegetarian", "gluten-free")
	TagTypeFunctionality TagType = "functionality"  // Functional tags (e.g., "high-protein", "quick-meal")
)

// Tag represents a tag entity for categorizing and describing food items and meals
type Tag struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Type        TagType   `json:"type" db:"type"`
	Description string    `json:"description" db:"description"`
	ColorHex    string    `json:"color_hex" db:"color_hex"` // UI display color
	IconURL     string    `json:"icon_url" db:"icon_url"`   // Optional icon for UI
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// TagCreateInput represents the input structure for creating a new tag
type TagCreateInput struct {
	Name        string   `json:"name" validate:"required,min=1,max=100"`
	Type        TagType  `json:"type" validate:"required,oneof=category functionality"`
	Description string   `json:"description" validate:"max=500"`
	ColorHex    string   `json:"color_hex" validate:"omitempty,hexcolor|len=7"`
	IconURL     string   `json:"icon_url" validate:"omitempty,url"`
}

// TagUpdateInput represents the input structure for updating an existing tag
type TagUpdateInput struct {
	Name        *string  `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=500"`
	ColorHex    *string  `json:"color_hex,omitempty" validate:"omitempty,hexcolor|len=7"`
	IconURL     *string  `json:"icon_url,omitempty" validate:"omitempty,url"`
}

// TagFilter represents filter options for querying tags
type TagFilter struct {
	Types     []TagType `json:"types,omitempty"`
	Search    string    `json:"search,omitempty"`     // Search in name and description
	Slug      string    `json:"slug,omitempty"`
	Limit     int       `json:"limit,omitempty"`      // Default: 50
	Offset    int       `json:"offset,omitempty"`     // Default: 0
	OrderBy   string    `json:"order_by,omitempty"`   // Default: "name"
	OrderDir  string    `json:"order_dir,omitempty"`  // Default: "asc"
}

// TagListResult represents a paginated list of tags
type TagListResult struct {
	Tags      []Tag  `json:"tags"`
	Total     int    `json:"total"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	HasMore   bool   `json:"has_more"`
}

// TagValidationError represents validation errors for tag operations
type TagValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// TagRepository defines the interface for tag data access operations
type TagRepository interface {
	Create(ctx context.Context, input TagCreateInput) (*Tag, error)
	Update(ctx context.Context, id string, input TagUpdateInput) (*Tag, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetBySlug(ctx context.Context, slug string) (*Tag, error)
	List(ctx context.Context, filter TagFilter) (*TagListResult, error)
	Exists(ctx context.Context, id string) (bool, error)
	ExistsBySlug(ctx context.Context, slug string, excludeID string) (bool, error)
	CountByType(ctx context.Context, tagType TagType) (int, error)
	GetTagsForFoodItem(ctx context.Context, foodItemID string) ([]Tag, error)
	GetTagsForMeal(ctx context.Context, mealID string) ([]Tag, error)
	AssignTagsToFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error
	RemoveTagsFromFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error
	AssignTagsToMeal(ctx context.Context, mealID string, tagIDs []string) error
	RemoveTagsFromMeal(ctx context.Context, mealID string, tagIDs []string) error
}

// TagService defines the interface for tag business logic
type TagService interface {
	CreateTag(ctx context.Context, input TagCreateInput) (*Tag, error)
	UpdateTag(ctx context.Context, id string, input TagUpdateInput) (*Tag, error)
	DeleteTag(ctx context.Context, id string) error
	GetTag(ctx context.Context, id string) (*Tag, error)
	GetTagBySlug(ctx context.Context, slug string) (*Tag, error)
	ListTags(ctx context.Context, filter TagFilter) (*TagListResult, error)
	ValidateTagInput(input TagCreateInput) []TagValidationError
	ValidateTagUpdate(input TagUpdateInput) []TagValidationError
	GenerateSlug(name string) string
	GetCategoryTags(ctx context.Context) ([]Tag, error)
	GetFunctionalityTags(ctx context.Context) ([]Tag, error)
}

// TagServiceError represents errors that can occur during tag service operations
type TagServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details error  `json:"details,omitempty"`
}

func (e *TagServiceError) Error() string {
	return e.Message
}

// Error codes for tag service
const (
	ErrCodeTagNotFound       = "TAG_NOT_FOUND"
	ErrCodeTagAlreadyExists  = "TAG_ALREADY_EXISTS"
	ErrCodeTagInvalidInput   = "TAG_INVALID_INPUT"
	ErrCodeTagInUse          = "TAG_IN_USE"
	ErrCodeTagDatabaseError  = "TAG_DATABASE_ERROR"
	ErrCodeTagValidation     = "TAG_VALIDATION_ERROR"
)
```

## 2. Logic & Algorithms

### 2.1 Tag Creation Flow

```
INPUT: TagCreateInput { name, type, description?, color_hex?, icon_url? }

1. VALIDATE INPUT
   a. Check if name is provided and within length limits (1-100 chars)
   b. Check if type is valid (category or functionality)
   c. Check if description length is within limit (0-500 chars)
   d. Check if color_hex is valid hex format (if provided)
   e. Check if icon_url is valid URL format (if provided)
   f. IF any validation fails:
      RETURN TagValidationError list

2. GENERATE SLUG
   a. Convert name to lowercase
   b. Replace spaces with hyphens
   c. Remove special characters except hyphens
   d. Truncate to 100 characters
   e. Append random suffix (4 chars) if slug already exists
   f. RETURN generated slug

3. CHECK DUPLICATE
   a. Call TagRepository.ExistsBySlug(ctx, slug, excludeID="")
   b. IF exists:
      RETURN error with code ErrCodeTagAlreadyExists

4. CREATE TAG IN DATABASE
   a. Call TagRepository.Create(ctx, input)
   b. Construct Tag with:
      - ID: UUID v4
      - Name: from input
      - Slug: from step 2
      - Type: from input
      - Description: from input (or empty string)
      - ColorHex: from input (or default color based on type)
      - IconURL: from input (or empty string)
      - CreatedAt: current UTC timestamp
      - UpdatedAt: current UTC timestamp
   c. INSERT into tags table
   d. RETURN created Tag

OUTPUT: (*Tag, error)
```

### 2.2 Tag Update Flow

```
INPUT: tagID string, TagUpdateInput

1. FETCH EXISTING TAG
   a. Call TagRepository.GetByID(ctx, tagID)
   b. IF not found:
      RETURN error with code ErrCodeTagNotFound

2. VALIDATE INPUT
   a. For each non-nil field in input:
      - Name: check length 1-100 chars
      - Description: check length 0-500 chars
      - ColorHex: validate hex format
      - IconURL: validate URL format
   b. IF validation fails:
      RETURN TagValidationError list

3. HANDLE SLUG GENERATION (if name changed)
   a. IF input.Name is provided and differs from current:
      i. Generate new slug from input.Name
      ii. Call TagRepository.ExistsBySlug(ctx, newSlug, excludeID=tagID)
      iii. IF slug exists:
           RETURN error with code ErrCodeTagAlreadyExists

4. UPDATE IN DATABASE
   a. Construct update fields map from non-nil input values
   b. Add UpdatedAt = current UTC timestamp
   c. Call TagRepository.Update(ctx, tagID, input)
   d. RETURN updated Tag

OUTPUT: (*Tag, error)
```

### 2.3 Tag List Query Flow

```
INPUT: TagFilter { types?, search?, limit?, offset?, order_by?, order_dir? }

1. APPLY DEFAULT VALUES
   a. IF limit not set or > 100: set to 50
   b. IF offset not set: set to 0
   c. IF order_by not set: set to "name"
   d. IF order_dir not set: set to "asc"
   e. Validate order_by is in allowed fields ["name", "created_at", "updated_at"]
   f. Validate order_dir is "asc" or "desc"

2. BUILD QUERY CONDITIONS
   a. IF types is provided:
      - Add condition: type IN (provided types)
   b. IF search is provided:
      - Add condition: (name ILIKE search OR description ILIKE search)
   c. IF slug is provided:
      - Add condition: slug = slug

3. EXECUTE QUERY
   a. Call TagRepository.List(ctx, filter)
   b. Get total count with same filters (excluding pagination)
   c. Calculate has_more = (offset + len(tags)) < total

4. RETURN RESULT
   a. Construct TagListResult with:
      - Tags: retrieved tags
      - Total: total count
      - Limit: applied limit
      - Offset: applied offset
      - HasMore: calculated has_more

OUTPUT: (*TagListResult, error)
```

### 2.4 Tag Assignment to FoodItem/Meal Flow

```
INPUT: entityID string, tagIDs []string

1. VALIDATE TAG IDS
   a. FOR each tagID in tagIDs:
      i. Call TagRepository.Exists(ctx, tagID)
      ii. IF any tag not found:
          RETURN error with code ErrCodeTagNotFound

2. VALIDATE ENTITY EXISTS
   a. Call appropriate repository to check entity exists
   b. IF entity not found:
      RETURN error with appropriate not found code

3. GET CURRENT ASSIGNED TAGS
   a. Call TagRepository.GetTagsForFoodItem/Meal(ctx, entityID)
   b. Build set of current tag IDs

4. COMPUTE TAG CHANGES
   a. tags_to_add = tagIDs - current_tag_ids
   b. tags_to_remove = current_tag_ids - tagIDs

5. EXECUTE CHANGES (in transaction)
   a. IF tags_to_add not empty:
      Call TagRepository.AssignTagsToFoodItem/Meal(ctx, entityID, tags_to_add)
   b. IF tags_to_remove not empty:
      Call TagRepository.RemoveTagsFromFoodItem/Meal(ctx, entityID, tags_to_remove)

6. RETURN SUCCESS
   a. No return value on success

OUTPUT: error
```

### 2.5 Slug Generation Algorithm

```
INPUT: name string

1. NORMALIZE
   a. Convert to lowercase
   b. Remove diacritics/accents (e.g., "ñ" -> "n", "é" -> "e")
   c. Replace spaces with hyphens
   d. Remove all characters except alphanumeric and hyphens
   e. Collapse multiple hyphens to single hyphen
   f. Trim hyphens from start and end

2. TRUNCATE
   a. IF length > 100: truncate to 100 characters
   b. Trim trailing hyphens after truncation

3. RETURN normalized slug
```

### 2.6 Default Color Assignment Algorithm

```
INPUT: tagType TagType

1. SWITCH tagType:
   CASE category:
      RETURN "#3B82F6"  // Blue
   CASE functionality:
      RETURN "#10B981"  // Emerald green
   DEFAULT:
      RETURN "#6B7280"  // Gray

OUTPUT: colorHex string
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | Condition | HTTP Status | User Message |
|------------|-----------|-------------|--------------|
| TAG_NOT_FOUND | Tag with given ID/slug does not exist | 404 | "Tag not found" |
| TAG_ALREADY_EXISTS | Tag slug already in use | 409 | "A tag with this name already exists" |
| TAG_INVALID_INPUT | Invalid input parameters | 400 | "Invalid input provided" |
| TAG_IN_USE | Tag is assigned to food items or meals | 409 | "Cannot delete tag as it is in use" |
| TAG_DATABASE_ERROR | Database operation failed | 500 | "An internal error occurred" |
| TAG_VALIDATION_ERROR | Input validation failed | 400 | Validation errors in response body |

### 3.2 State Transitions

```
Tag Creation State Machine:

[NEW] --validation pass--> [VALIDATING]
[VALIDATING] --check duplicate--> [CHECKING_DUPLICATE]
[CHECKING_DUPLICATE] --not exists--> [CREATING]
[CREATING] --success--> [CREATED]
[CREATING] --failure--> [ERROR]
[ERROR] --retry--> [VALIDATING]

[VALIDATING] --validation fail--> [VALIDATION_FAILED]
[CHECKING_DUPLICATE] --exists--> [DUPLICATE_FOUND]
[CREATED] --deletion--> [DELETING]
[DELETING] --in use--> [DELETE_BLOCKED]
[DELETING] --success--> [DELETED]
```

### 3.3 Tag Lifecycle States

| State | Description | Allowed Operations |
|-------|-------------|-------------------|
| ACTIVE | Tag is active and can be used | Read, Update, Delete (if not in use) |
| IN_USE | Tag is assigned to entities | Read, Update |
| DELETED | Soft-deleted tag | Restore (optional) |

### 3.4 Concurrent Access Handling

1. **Optimistic Locking**: Use `UpdatedAt` timestamp for concurrent update detection
2. **Race Condition Prevention**:
   - Slug uniqueness check with retry mechanism (max 3 attempts)
   - Tag assignment using database transactions with proper isolation
3. **Deadlock Prevention**: Always acquire locks in consistent order (by tag ID)

### 3.5 Error Recovery Strategies

| Error Type | Recovery Strategy |
|------------|-------------------|
| Database connection lost | Retry with exponential backoff (max 5 attempts) |
| Unique constraint violation | Regenerate slug with new suffix, retry once |
| Validation error | Return detailed error list to client |
| Tag in use on delete | Return error with list of dependent entities |

### 3.6 Cache Invalidation Strategy

```
Cache Keys:
- tag:{id} - Individual tag by ID
- tag:slug:{slug} - Individual tag by slug
- tag:list:{filter_hash} - List query results
- tag:types:{type} - Tags by type (category/functionality)

Invalidation Rules:
1. On Tag Create:
   - Invalidate tag:list:* (full list)
   - Invalidate tag:types:{type}

2. On Tag Update:
   - Invalidate tag:{id}
   - Invalidate tag:slug:{old_slug}
   - Invalidate tag:slug:{new_slug}
   - Invalidate tag:list:*
   - Invalidate tag:types:{type}

3. On Tag Delete:
   - Invalidate tag:{id}
   - Invalidate tag:slug:{slug}
   - Invalidate tag:list:*
   - Invalidate tag:types:{type}

4. On Tag Assignment:
   - Invalidate affected food item cache
   - Invalidate affected meal cache
```

## 4. Component Interfaces

### 4.1 TagRepository Interface

```go
package repository

import (
	"context"
	"mealswapp/entity"
)

// TagRepository defines the interface for tag data access operations
type TagRepository interface {
	// Create inserts a new tag into the database
	Create(ctx context.Context, input entity.TagCreateInput) (*entity.Tag, error)

	// Update modifies an existing tag
	Update(ctx context.Context, id string, input entity.TagUpdateInput) (*entity.Tag, error)

	// Delete removes a tag (hard delete)
	Delete(ctx context.Context, id string) error

	// GetByID retrieves a tag by its UUID
	GetByID(ctx context.Context, id string) (*entity.Tag, error)

	// GetBySlug retrieves a tag by its slug
	GetBySlug(ctx context.Context, slug string) (*entity.Tag, error)

	// List retrieves tags matching the filter criteria
	List(ctx context.Context, filter entity.TagFilter) (*entity.TagListResult, error)

	// Exists checks if a tag exists by ID
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsBySlug checks if a slug is already in use (excluding specified ID)
	ExistsBySlug(ctx context.Context, slug string, excludeID string) (bool, error)

	// CountByType returns the count of tags of a specific type
	CountByType(ctx context.Context, tagType entity.TagType) (int, error)

	// GetTagsForFoodItem retrieves all tags assigned to a food item
	GetTagsForFoodItem(ctx context.Context, foodItemID string) ([]entity.Tag, error)

	// AssignTagsToFoodItem assigns tags to a food item
	AssignTagsToFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error

	// RemoveTagsFromFoodItem removes tags from a food item
	RemoveTagsFromFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error

	// GetTagsForMeal retrieves all tags assigned to a meal
	GetTagsForMeal(ctx context.Context, mealID string) ([]entity.Tag, error)

	// AssignTagsToMeal assigns tags to a meal
	AssignTagsToMeal(ctx context.Context, mealID string, tagIDs []string) error

	// RemoveTagsFromMeal removes tags from a meal
	RemoveTagsFromMeal(ctx context.Context, mealID string, tagIDs []string) error
}
```

### 4.2 TagService Interface

```go
package service

import (
	"context"
	"mealswapp/entity"
)

// TagService defines the interface for tag business logic
type TagService interface {
	// CreateTag creates a new tag with validation
	CreateTag(ctx context.Context, input entity.TagCreateInput) (*entity.Tag, error)

	// UpdateTag updates an existing tag
	UpdateTag(ctx context.Context, id string, input entity.TagUpdateInput) (*entity.Tag, error)

	// DeleteTag deletes a tag (fails if tag is in use)
	DeleteTag(ctx context.Context, id string) error

	// GetTag retrieves a tag by ID
	GetTag(ctx context.Context, id string) (*entity.Tag, error)

	// GetTagBySlug retrieves a tag by slug
	GetTagBySlug(ctx context.Context, slug string) (*entity.Tag, error)

	// ListTags retrieves tags matching filter criteria
	ListTags(ctx context.Context, filter entity.TagFilter) (*entity.TagListResult, error)

	// ValidateTagInput validates tag creation input
	ValidateTagInput(input entity.TagCreateInput) []entity.TagValidationError

	// ValidateTagUpdate validates tag update input
	ValidateTagUpdate(input entity.TagUpdateInput) []entity.TagValidationError

	// GenerateSlug generates a URL-safe slug from a name
	GenerateSlug(name string) string

	// GetCategoryTags retrieves all category tags
	GetCategoryTags(ctx context.Context) ([]entity.Tag, error)

	// GetFunctionalityTags retrieves all functionality tags
	GetFunctionalityTags(ctx context.Context) ([]entity.Tag, error)

	// AssignTagsToFoodItem assigns tags to a food item
	AssignTagsToFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error

	// ReplaceTagsForFoodItem replaces all tags for a food item
	ReplaceTagsForFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error

	// AssignTagsToMeal assigns tags to a meal
	AssignTagsToMeal(ctx context.Context, mealID string, tagIDs []string) error

	// ReplaceTagsForMeal replaces all tags for a meal
	ReplaceTagsForMeal(ctx context.Context, mealID string, tagIDs []string) error
}
```

### 4.3 Tag Entity SQL Schema

```sql
-- tags table
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('category', 'functionality')),
    description VARCHAR(500),
    color_hex VARCHAR(7) DEFAULT '#6B7280',
    icon_url VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_tags_slug ON tags(slug);
CREATE INDEX idx_tags_type ON tags(type);
CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_tags_type_name ON tags(type, name);

-- food_item_tags junction table
CREATE TABLE food_item_tags (
    food_item_id UUID NOT NULL REFERENCES food_items(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (food_item_id, tag_id)
);

CREATE INDEX idx_food_item_tags_tag_id ON food_item_tags(tag_id);

-- meal_tags junction table
CREATE TABLE meal_tags (
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (meal_id, tag_id)
);

CREATE INDEX idx_meal_tags_tag_id ON meal_tags(tag_id);
```

### 4.4 Public API Endpoints (Fiber)

```go
// TagHandler handles HTTP requests for tag operations
type TagHandler interface {
	// GET /api/v1/tags
	// Query params: types, search, limit, offset, order_by, order_dir
	ListTags(c *fiber.Ctx) error

	// GET /api/v1/tags/:id
	GetTag(c *fiber.Ctx) error

	// GET /api/v1/tags/slug/:slug
	GetTagBySlug(c *fiber.Ctx) error

	// GET /api/v1/tags/categories
	GetCategoryTags(c *fiber.Ctx) error

	// GET /api/v1/tags/functionalities
	GetFunctionalityTags(c *fiber.Ctx) error

	// POST /api/v1/tags
	CreateTag(c *fiber.Ctx) error

	// PATCH /api/v1/tags/:id
	UpdateTag(c *fiber.Ctx) error

	// DELETE /api/v1/tags/:id
	DeleteTag(c *fiber.Ctx) error

	// POST /api/v1/food-items/:foodItemId/tags
	AssignTagsToFoodItem(c *fiber.Ctx) error

	// PUT /api/v1/food-items/:foodItemId/tags
	ReplaceTagsForFoodItem(c *fiber.Ctx) error

	// DELETE /api/v1/food-items/:foodItemId/tags
	RemoveTagsFromFoodItem(c *fiber.Ctx) error

	// POST /api/v1/meals/:mealId/tags
	AssignTagsToMeal(c *fiber.Ctx) error

	// PUT /api/v1/meals/:mealId/tags
	ReplaceTagsForMeal(c *fiber.Ctx) error

	// DELETE /api/v1/meals/:mealId/tags
	RemoveTagsFromMeal(c *fiber.Ctx) error
}
```

### 4.5 Request/Response DTOs

```go
// CreateTagRequest represents the HTTP request body for creating a tag
type CreateTagRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	ColorHex    string `json:"color_hex,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
}

// UpdateTagRequest represents the HTTP request body for updating a tag
type UpdateTagRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	ColorHex    *string `json:"color_hex,omitempty"`
	IconURL     *string `json:"icon_url,omitempty"`
}

// TagResponse represents the HTTP response for a single tag
type TagResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	ColorHex    string    `json:"color_hex"`
	IconURL     string    `json:"icon_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TagListResponse represents the HTTP response for tag list queries
type TagListResponse struct {
	Tags    []TagResponse `json:"tags"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
	HasMore bool          `json:"has_more"`
}

// AssignTagsRequest represents the HTTP request body for assigning tags
type AssignTagsRequest struct {
	TagIDs []string `json:"tag_ids" validate:"required,min=1,max=50,dive,uuid"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code    string                   `json:"code"`
	Message string                   `json:"message"`
	Errors  []entity.TagValidationError `json:"errors,omitempty"`
}
```

### 4.6 Configuration

```go
// TagConfig holds configuration for tag-related operations
type TagConfig struct {
	// DefaultColors maps tag types to default hex colors
	DefaultColors map[string]string

	// MaxNameLength maximum allowed length for tag names
	MaxNameLength int

	// MaxDescriptionLength maximum allowed length for tag descriptions
	MaxDescriptionLength int

	// DefaultLimit default limit for list queries
	DefaultLimit int

	// MaxLimit maximum allowed limit for list queries
	MaxLimit int

	// SlugLength maximum length for generated slugs
	SlugLength int
}

// DefaultTagConfig returns the default configuration
func DefaultTagConfig() TagConfig {
	return TagConfig{
		DefaultColors: map[string]string{
			"category":      "#3B82F6",
			"functionality": "#10B981",
		},
		MaxNameLength:        100,
		MaxDescriptionLength: 500,
		DefaultLimit:         50,
		MaxLimit:             100,
		SlugLength:           100,
	}
}
```

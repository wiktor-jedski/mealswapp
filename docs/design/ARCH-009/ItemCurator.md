# ItemCurator

**Traceability:** ARCH-009

## 1. Data Structures & Types

```go
package itemcurator

import (
	"time"
)

// FoodItem represents a curated food item in the database
type FoodItem struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Brand       string    `json:"brand" db:"brand"`
	ImageURL    string    `json:"image_url" db:"image_url"`
	ImagePath   string    `json:"image_path" db:"image_path"`
	Barcode     string    `json:"barcode" db:"barcode"`
	ServingSize float64   `json:"serving_size" db:"serving_size"`
	ServingUnit string    `json:"serving_unit" db:"serving_unit"`
	Calories    float64   `json:"calories" db:"calories"`
	Protein     float64   `json:"protein" db:"protein"`
	Carbs       float64   `json:"carbs" db:"carbs"`
	Fat         float64   `json:"fat" db:"fat"`
	Fiber       float64   `json:"fiber" db:"fiber"`
	Sodium      float64   `json:"sodium" db:"sodium"`
	Sugar       float64   `json:"sugar" db:"sugar"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
	UpdatedBy   string    `json:"updated_by" db:"updated_by"`
}

// ItemTag represents the many-to-many relationship between items and tags
type ItemTag struct {
	ItemID    string    `db:"item_id"`
	TagID     string    `db:"tag_id"`
	CreatedAt time.Time `db:"created_at"`
}

// Tag represents a tag that can be applied to items
type Tag struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	TagType     TagType   `json:"tag_type" db:"tag_type"`
	ParentID    *string   `json:"parent_id" db:"parent_id"`
	Description string    `json:"description" db:"description"`
	Color       string    `json:"color" db:"color"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// TagType represents the type of tag
type TagType string

const (
	TagTypeCategory       TagType = "category"
	TagTypeFunctionality  TagType = "functionality"
	TagTypeDietary        TagType = "dietary"
	TagTypeAllergen       TagType = "allergen"
	TagTypeMacroBreakdown TagType = "macro_breakdown"
)

// CreateItemRequest represents the request body for creating a new item
type CreateItemRequest struct {
	Name         string             `json:"name" validate:"required,min=1,max=255"`
	Description  string             `json:"description" validate:"max=1000"`
	Brand        string             `json:"brand" validate:"max=100"`
	ImageBase64  string             `json:"image_base64,omitempty"`
	Barcode      string             `json:"barcode" validate:"max=50"`
	ServingSize  float64            `json:"serving_size" validate:"required,gt=0"`
	ServingUnit  string             `json:"serving_unit" validate:"required,max=20"`
	Macros       MacroInput         `json:"macros" validate:"required"`
	TagIDs       []string           `json:"tag_ids" validate:"max=20"`
	ExternalData *ExternalItemData  `json:"external_data,omitempty"`
}

// MacroInput represents the macro nutrients for an item
type MacroInput struct {
	Calories float64 `json:"calories" validate:"required,gte=0"`
	Protein  float64 `json:"protein" validate:"required,gte=0"`
	Carbs    float64 `json:"carbs" validate:"required,gte=0"`
	Fat      float64 `json:"fat" validate:"required,gte=0"`
	Fiber    float64 `json:"fiber" validate:"required,gte=0"`
	Sodium   float64 `json:"sodium" validate:"required,gte=0"`
	Sugar    float64 `json:"sugar" validate:"required,gte=0"`
}

// UpdateItemRequest represents the request body for updating an item
type UpdateItemRequest struct {
	Name         *string            `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description  *string            `json:"description,omitempty" validate:"omitempty,max=1000"`
	Brand        *string            `json:"brand,omitempty" validate:"omitempty,max=100"`
	ImageBase64  *string            `json:"image_base64,omitempty"`
	Barcode      *string            `json:"barcode,omitempty" validate:"omitempty,max=50"`
	ServingSize  *float64           `json:"serving_size,omitempty" validate:"omitempty,gt=0"`
	ServingUnit  *string            `json:"serving_unit,omitempty" validate:"omitempty,max=20"`
	Macros       *MacroInput        `json:"macros,omitempty"`
	TagIDs       *[]string          `json:"tag_ids,omitempty" validate:"omitempty,max=20"`
	IsActive     *bool              `json:"is_active,omitempty"`
	ExternalData *ExternalItemData  `json:"external_data,omitempty"`
}

// ExternalItemData represents data imported from external sources
type ExternalItemData struct {
	Source      string `json:"source" db:"source"`
	ExternalID  string `json:"external_id" db:"external_id"`
	RawJSON     string `json:"raw_json" db:"raw_json"`
	ImportedAt  time.Time `json:"imported_at" db:"imported_at"`
}

// ItemListFilter represents filter parameters for listing items
type ItemListFilter struct {
	Search     string    `query:"search"`
	TagIDs     []string  `query:"tag_ids"`
	TagType    TagType   `query:"tag_type"`
	MinProtein *float64  `query:"min_protein"`
	MaxProtein *float64  `query:"max_protein"`
	MinCarbs   *float64  `query:"min_carbs"`
	MaxCarbs   *float64  `query:"max_carbs"`
	MinFat     *float64  `query:"min_fat"`
	MaxFat     *float64  `query:"max_fat"`
	IsActive   *bool     `query:"is_active"`
	Page       int       `query:"page"`
	PageSize   int       `query:"page_size"`
	SortBy     string    `query:"sort_by"`
	SortOrder  string    `query:"sort_order"`
}

// ItemListResponse represents the paginated response for item listing
type ItemListResponse struct {
	Items      []FoodItem `json:"items"`
	TotalCount int        `json:"total_count"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// CuratorError represents an error from the ItemCurator
type CuratorError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *CuratorError) Error() string {
	return e.Message
}

// Error codes
const (
	ErrCodeItemNotFound       = "ITEM_NOT_FOUND"
	ErrCodeInvalidInput       = "INVALID_INPUT"
	ErrCodeDuplicateBarcode   = "DUPLICATE_BARCODE"
	ErrCodeTagNotFound        = "TAG_NOT_FOUND"
	ErrCodeImageUploadFailed  = "IMAGE_UPLOAD_FAILED"
	ErrCodeDatabaseError      = "DATABASE_ERROR"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
)
```

## 2. Logic & Algorithms

### 2.1 CreateItem

```
ALGORITHM: CreateItem
INPUT: ctx (Fiber context), req (CreateItemRequest), adminID (string)
OUTPUT: FoodItem, error

1. Validate request
   1.1 Validate required fields using validator
   1.2 If validation fails, return ErrCodeInvalidInput error

2. Check for duplicate barcode
   2.1 Query database: SELECT id FROM food_items WHERE barcode = $1 AND is_active = true
   2.2 If row exists, return ErrCodeDuplicateBarcode error

3. Handle image upload if base64 provided
   3.1 Call UploadImage(ctx, req.ImageBase64, req.Name)
   3.2 If upload fails, return ErrCodeImageUploadFailed error
   3.3 Set imageURL and imagePath from result

4. Generate unique ID for item
   4.1 Use UUID v4 generation

5. Set timestamps
   5.1 createdAt = current UTC time
   5.2 updatedAt = current UTC time

6. Insert item into database
   6.1 BEGIN transaction
   6.2 INSERT into food_items with all fields
   6.3 If error, ROLLBACK and return ErrCodeDatabaseError

7. Handle tag associations
   7.1 If req.TagIDs is not empty
       7.1.1 Validate all tag IDs exist
       7.1.2 INSERT into item_tags for each tag_id

8. Handle external data if present
   8.1 INSERT into external_item_data table

9. COMMIT transaction

10. Fetch and return created item
    10.1 SELECT * FROM food_items WHERE id = itemID
    10.2 Return item
```

### 2.2 UpdateItem

```
ALGORITHM: UpdateItem
INPUT: ctx (Fiber context), itemID (string), req (UpdateItemRequest), adminID (string)
OUTPUT: FoodItem, error

1. Fetch existing item
   1.1 SELECT * FROM food_items WHERE id = $1 AND is_active = true
   1.2 If no row found, return ErrCodeItemNotFound error

2. Validate request if present
   2.1 Validate provided fields using validator
   2.2 If validation fails, return ErrCodeInvalidInput error

3. Handle barcode change
   3.1 If req.Barcode is present and different from current
       3.1.1 Check for duplicate barcode
       3.1.2 If duplicate, return ErrCodeDuplicateBarcode error

4. Handle image upload if base64 provided
   4.1 Call UploadImage(ctx, req.ImageBase64, currentItem.Name)
   4.2 If upload fails, return ErrCodeImageUploadFailed error
   4.3 Update imageURL and imagePath

5. Build update query dynamically
   5.1 Start with empty SET clause
   5.2 Add fields that are present in request
   5.3 Add updated_at = current UTC time
   5.4 Add updated_by = adminID

6. Execute update
   6.1 BEGIN transaction
   6.2 UPDATE food_items SET ...
   6.3 If error, ROLLBACK and return ErrCodeDatabaseError

7. Handle tag updates if req.TagIDs is present
   7.1 DELETE FROM item_tags WHERE item_id = itemID
   7.2 INSERT new tag associations
   7.3 If tag ID invalid, return ErrCodeTagNotFound error

8. COMMIT transaction

9. Fetch and return updated item
    9.1 SELECT * FROM food_items WHERE id = itemID
    9.2 Return item
```

### 2.3 DeleteItem (Soft Delete)

```
ALGORITHM: DeleteItem
INPUT: ctx (Fiber context), itemID (string), adminID (string)
OUTPUT: error

1. Fetch existing item
   1.1 SELECT * FROM food_items WHERE id = $1 AND is_active = true
   1.2 If no row found, return ErrCodeItemNotFound error

2. Perform soft delete
   2.1 BEGIN transaction
   2.2 UPDATE food_items SET is_active = false, updated_at = NOW(), updated_by = adminID
       WHERE id = itemID
   2.3 If error, ROLLBACK and return ErrCodeDatabaseError

3. COMMIT transaction

4. Return nil error (success)
```

### 2.4 GetItem

```
ALGORITHM: GetItem
INPUT: ctx (Fiber context), itemID (string)
OUTPUT: FoodItemWithTags, error

1. Fetch item
   1.1 SELECT * FROM food_items WHERE id = $1 AND is_active = true
   1.2 If no row found, return ErrCodeItemNotFound error

2. Fetch associated tags
   2.1 SELECT t.* FROM tags t
       INNER JOIN item_tags it ON t.id = it.tag_id
       WHERE it.item_id = itemID

3. Build response with tags
   3.1 Create FoodItemWithTags struct
   3.2 Populate item fields
   3.3 Populate tags slice

4. Return response
```

### 2.5 ListItems

```
ALGORITHM: ListItems
INPUT: ctx (Fiber context), filter (ItemListFilter)
OUTPUT: ItemListResponse, error

1. Set default pagination if not provided
   1.1 page = 1
   1.2 page_size = 20
   1.3 max_page_size = 100

2. Build WHERE clause dynamically
   2.1 Start with WHERE is_active = true
   2.2 Add search condition if provided (name ILIKE OR description ILIKE)
   2.3 Add tag filter if provided (EXISTS subquery)
   2.4 Add macro range filters if provided
   2.5 Add is_active filter if provided

3. Build ORDER BY clause
   3.1 Default: updated_at DESC
   3.2 Validate sort_by against allowed fields
   3.3 Validate sort_order (ASC/DESC)

4. Execute count query
   4.1 SELECT COUNT(*) FROM food_items WHERE ...
   4.2 Store totalCount

5. Execute data query with pagination
   5.1 SELECT * FROM food_items WHERE ...
   5.2 ORDER BY sort_by sort_order
   5.3 LIMIT page_size OFFSET (page-1) * page_size

6. Fetch tags for all items
   6.1 Batch fetch tags for returned items
   6.2 Map tags to items

7. Calculate totalPages
   7.1 totalPages = (totalCount + pageSize - 1) / pageSize

8. Build and return response
```

### 2.6 ImportFromExternal

```
ALGORITHM: ImportFromExternal
INPUT: ctx (Fiber context), externalItem (ExternalSearchResult), adminID (string)
OUTPUT: FoodItem, error

1. Map external data to CreateItemRequest
   1.1 Extract name, brand, macros
   1.2 Set serving_size and serving_unit with defaults
   1.3 Store raw JSON in external_data field

2. Generate item name if not present
   2.1 Use brand + product_name format

3. Set default macros for missing values
   3.1 Use 0 for any missing macro fields

4. Call CreateItem with mapped request
   4.1 Pass adminID for audit trail

5. Return created item
```

## 3. State Management & Error Handling

### 3.1 State Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           ItemCurator States                            │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   IDLE       │───>│  VALIDATING  │───>│  PROCESSING  │
│              │    │              │    │              │
│  Ready for   │    │  Checking    │    │  Uploading   │
│  requests    │    │  input,      │    │  images,     │
│              │    │  barcode     │    │  inserting   │
└──────────────┘    │  duplicates  │    │  to DB       │
       ▲            └──────────────┘    └──────────────┘
       │                    │                    │
       │                    │ Error              │ Success
       │                    ▼                    ▼
       │            ┌──────────────┐    ┌──────────────┐
       │            │   ERROR      │    │  COMPLETED   │
       │            │              │    │              │
       └────────────│  Error state │    │  Return item │
                    │  with error  │    │  to caller   │
                    │  code        │    │              │
                    └──────────────┘    └──────────────┘
```

### 3.2 Error States and Transitions

| Error State | Trigger | HTTP Status | Transition To |
|-------------|---------|-------------|---------------|
| ErrCodeItemNotFound | GET/PUT/DELETE itemID not found | 404 | IDLE after response |
| ErrCodeInvalidInput | Validation fails | 400 | IDLE after response |
| ErrCodeDuplicateBarcode | Barcode already exists | 409 | VALIDATING after response |
| ErrCodeTagNotFound | Tag ID doesn't exist | 400 | PROCESSING after response |
| ErrCodeImageUploadFailed | Image upload to GCS fails | 500 | IDLE after response |
| ErrCodeDatabaseError | DB operation fails | 500 | IDLE after response |
| ErrCodeUnauthorized | User lacks admin role | 403 | N/A (reject request) |

### 3.3 Recovery Strategies

1. **Database Errors**
   - Retry transaction up to 3 times with exponential backoff
   - Log error with context for debugging
   - Return user-friendly error message

2. **Image Upload Failures**
   - Retry up to 2 times
   - If still failing, fallback to placeholder image
   - Log GCS error for monitoring

3. **Validation Errors**
   - Return detailed field errors to client
   - Include validation rules in response

4. **Tag Association Errors**
   - Rollback item creation if tag insert fails
   - Return specific tag ID that failed

### 3.4 Logging and Monitoring

- Log all CRUD operations with admin ID and item ID
- Track operation duration for performance monitoring
- Log error details with stack traces
- Emit metrics for:
  - Items created/updated/deleted per hour
  - Average operation latency
  - Error rate by type
  - External import success rate

## 4. Component Interfaces

### 4.1 Public Functions

```go
package itemcurator

import (
	"github.com/gofiber/fiber"
)

// Interface defines the contract for ItemCurator
type Interface interface {
	CreateItem(ctx *fiber.Ctx, req *CreateItemRequest, adminID string) (*FoodItem, error)
	UpdateItem(ctx *fiber.Ctx, itemID string, req *UpdateItemRequest, adminID string) (*FoodItem, error)
	DeleteItem(ctx *fiber.Ctx, itemID string, adminID string) error
	GetItem(ctx *fiber.Ctx, itemID string) (*FoodItemWithTags, error)
	ListItems(ctx *fiber.Ctx, filter *ItemListFilter) (*ItemListResponse, error)
	ImportFromExternal(ctx *fiber.Ctx, externalItem *ExternalSearchResult, adminID string) (*FoodItem, error)
	GetItemTags(ctx *fiber.Ctx, itemID string) ([]*Tag, error)
	UpdateItemTags(ctx *fiber.Ctx, itemID string, tagIDs []string, adminID string) error
}

// ItemCurator implements the Interface
type ItemCurator struct {
	db          *sql.DB
	redis       *redis.Client
	gcsClient   *storage.Client
	tagManager  tagmanager.Interface
	auditLogger *AuditLogger
}

// New creates a new ItemCurator instance
func New(db *sql.DB, redis *redis.Client, gcsClient *storage.Client, tagManager tagmanager.Interface) *ItemCurator {
	return &ItemCurator{
		db:          db,
		redis:       redis,
		gcsClient:   gcsClient,
		tagManager:  tagManager,
		auditLogger: NewAuditLogger(db),
	}
}
```

### 4.2 CreateItem Signature and Implementation

```go
func (c *ItemCurator) CreateItem(ctx *fiber.Ctx, req *CreateItemRequest, adminID string) (*FoodItem, error) {
	// Validate request
	if err := validateCreateRequest(req); err != nil {
		return nil, &CuratorError{Code: ErrCodeInvalidInput, Message: err.Error()}
	}

	// Check duplicate barcode
	exists, err := c.checkBarcodeExists(ctx.Context(), req.Barcode)
	if err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to check barcode", Details: err.Error()}
	}
	if exists {
		return nil, &CuratorError{Code: ErrCodeDuplicateBarcode, Message: "Barcode already exists"}
	}

	// Handle image upload
	var imageURL, imagePath string
	if req.ImageBase64 != "" {
		imageURL, imagePath, err = c.uploadImage(ctx.Context(), req.ImageBase64, req.Name)
		if err != nil {
			return nil, &CuratorError{Code: ErrCodeImageUploadFailed, Message: "Failed to upload image", Details: err.Error()}
		}
	}

	// Validate tags
	if len(req.TagIDs) > 0 {
		valid, err := c.tagManager.ValidateTagIDs(ctx.Context(), req.TagIDs)
		if err != nil {
			return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to validate tags", Details: err.Error()}
		}
		if !valid {
			return nil, &CuratorError{Code: ErrCodeTagNotFound, Message: "One or more tags not found"}
		}
	}

	// Generate ID and timestamps
	itemID := uuid.New().String()
	now := time.Now().UTC()

	// Execute in transaction
	var item *FoodItem
	err = c.db.BeginTx(ctx.Context(), &sql.TxOptions{})
	if err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to start transaction", Details: err.Error()}
	}

	item, err = c.insertItem(ctx.Context(), tx, itemID, req, imageURL, imagePath, adminID, now)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(req.TagIDs) > 0 {
		if err := c.insertItemTags(ctx.Context(), tx, itemID, req.TagIDs, now); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to commit transaction", Details: err.Error()}
	}

	// Log audit
	c.auditLogger.Log(ctx.Context(), "ITEM_CREATE", itemID, adminID, nil)

	// Cache item
	c.cacheItem(ctx.Context(), item)

	return item, nil
}
```

### 4.3 UpdateItem Signature and Implementation

```go
func (c *ItemCurator) UpdateItem(ctx *fiber.Ctx, itemID string, req *UpdateItemRequest, adminID string) (*FoodItem, error) {
	// Fetch existing item
	existing, err := c.getItemByID(ctx.Context(), itemID)
	if err != nil {
		return nil, err
	}

	// Validate request
	if err := validateUpdateRequest(req); err != nil {
		return nil, &CuratorError{Code: ErrCodeInvalidInput, Message: err.Error()}
	}

	// Handle image upload if needed
	if req.ImageBase64 != nil && *req.ImageBase64 != "" {
		imageURL, imagePath, err := c.uploadImage(ctx.Context(), *req.ImageBase64, existing.Name)
		if err != nil {
			return nil, &CuratorError{Code: ErrCodeImageUploadFailed, Message: "Failed to upload image", Details: err.Error()}
		}
		req.ImageURL = &imageURL
		req.ImagePath = &imagePath
	}

	// Check barcode duplicate if changing
	if req.Barcode != nil && *req.Barcode != existing.Barcode {
		exists, err := c.checkBarcodeExists(ctx.Context(), *req.Barcode)
		if err != nil {
			return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to check barcode", Details: err.Error()}
		}
		if exists {
			return nil, &CuratorError{Code: ErrCodeDuplicateBarcode, Message: "Barcode already exists"}
		}
	}

	// Execute update in transaction
	tx, err := c.db.BeginTx(ctx.Context(), &sql.TxOptions{})
	if err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to start transaction", Details: err.Error()}
	}

	item, err := c.performUpdate(ctx.Context(), tx, existing, req, adminID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Handle tag updates if provided
	if req.TagIDs != nil {
		if err := c.updateItemTags(ctx.Context(), tx, itemID, *req.TagIDs); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to commit transaction", Details: err.Error()}
	}

	// Log audit
	c.auditLogger.Log(ctx.Context(), "ITEM_UPDATE", itemID, adminID, map[string]interface{}{
		"changes": c.calculateChanges(existing, item),
	})

	// Invalidate cache
	c.invalidateItemCache(ctx.Context(), itemID)

	return item, nil
}
```

### 4.4 DeleteItem Signature

```go
func (c *ItemCurator) DeleteItem(ctx *fiber.Ctx, itemID string, adminID string) error {
	// Fetch existing item
	existing, err := c.getItemByID(ctx.Context(), itemID)
	if err != nil {
		return err
	}

	// Soft delete
	result, err := c.db.ExecContext(
		ctx.Context(),
		"UPDATE food_items SET is_active = false, updated_at = $1, updated_by = $2 WHERE id = $3",
		time.Now().UTC(), adminID, itemID,
	)
	if err != nil {
		return &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to delete item", Details: err.Error()}
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &CuratorError{Code: ErrCodeItemNotFound, Message: "Item not found"}
	}

	// Log audit
	c.auditLogger.Log(ctx.Context(), "ITEM_DELETE", itemID, adminID, nil)

	// Invalidate cache
	c.invalidateItemCache(ctx.Context(), itemID)

	return nil
}
```

### 4.5 ListItems Signature

```go
func (c *ItemCurator) ListItems(ctx *fiber.Ctx, filter *ItemListFilter) (*ItemListResponse, error) {
	// Apply defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	// Build query
	query := c.buildListQuery(filter)
	args := c.buildListArgs(filter)

	// Execute count
	totalCount, err := c.countItems(ctx.Context(), query, args...)
	if err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to count items", Details: err.Error()}
	}

	// Execute query with pagination
	items, err := c.queryItems(ctx.Context(), query, args...)
	if err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to fetch items", Details: err.Error()}
	}

	// Fetch tags for all items
	itemIDs := make([]string, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}
	tagsByItemID, err := c.batchFetchTags(ctx.Context(), itemIDs)
	if err != nil {
		return nil, &CuratorError{Code: ErrCodeDatabaseError, Message: "Failed to fetch tags", Details: err.Error()}
	}

	// Build response
	response := &ItemListResponse{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: (totalCount + filter.PageSize - 1) / filter.PageSize,
	}

	// Attach tags
	for i := range response.Items {
		response.Items[i].Tags = tagsByItemID[response.Items[i].ID]
	}

	return response, nil
}
```

### 4.6 Helper Functions

```go
// validateCreateRequest validates the CreateItemRequest
func validateCreateRequest(req *CreateItemRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.ServingSize <= 0 {
		return errors.New("serving_size must be greater than 0")
	}
	if req.Macros.Calories < 0 {
		return errors.New("calories cannot be negative")
	}
	// Additional validations...
	return nil
}

// checkBarcodeExists checks if a barcode already exists
func (c *ItemCurator) checkBarcodeExists(ctx context.Context, barcode string) (bool, error) {
	var count int
	err := c.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM food_items WHERE barcode = $1 AND is_active = true",
		barcode,
	).Scan(&count)
	return count > 0, err
}

// uploadImage uploads an image to GCS and returns the URL and path
func (c *ItemCurator) uploadImage(ctx context.Context, base64Image, itemName string) (string, string, error) {
	// Decode base64
	data, err := base64.DecodeString(base64Image)
	if err != nil {
		return "", "", err
	}

	// Detect content type
	contentType := http.DetectContentType(data[:512])
	if !strings.HasPrefix(contentType, "image/") {
		return "", "", errors.New("invalid image format")
	}

	// Generate unique filename
	filename := fmt.Sprintf("items/%s/%s.%s",
		time.Now().Format("2006/01/02"),
		uuid.New().String(),
		strings.TrimPrefix(contentType, "image/"),
	)

	// Upload to GCS
	wc := c.gcsClient.Bucket(GCSBucket).Object(filename).NewWriter(ctx)
	if _, err := wc.Write(data); err != nil {
		return "", "", err
	}
	if err := wc.Close(); err != nil {
		return "", "", err
	}

	// Return public URL
	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", GCSBucket, filename)
	return url, filename, nil
}

// buildListQuery builds the SQL query for listing items
func (c *ItemCurator) buildListQuery(filter *ItemListFilter) string {
	var conditions []string
	conditions = append(conditions, "WHERE is_active = true")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"AND (name ILIKE $%d OR description ILIKE $%d)",
			len(conditions)+1, len(conditions)+2,
		))
	}

	if len(filter.TagIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf(
			"AND id IN (SELECT item_id FROM item_tags WHERE tag_id = ANY($%d))",
			len(conditions)+1,
		))
	}

	if filter.MinProtein != nil {
		conditions = append(conditions, fmt.Sprintf("AND protein >= $%d", len(conditions)+1))
	}

	// Additional filters...

	return "SELECT * FROM food_items " + strings.Join(conditions, " ")
}
```

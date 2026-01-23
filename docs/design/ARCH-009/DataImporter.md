# DataImporter

**Traceability:** ARCH-009

## 1. Data Structures & Types

```go
package dataimporter

import (
	"time"
)

type CuratedItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Brand       string    `json:"brand"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Macros      Macros    `json:"macros"`
	ImageURL    string    `json:"image_url"`
	Source      string    `json:"source"`        // "USDA" or "OpenFoodFacts"
	SourceID    string    `json:"source_id"`     // Original ID from external source
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Macros struct {
	Calories   float64 `json:"calories"`
	Protein    float64 `json:"protein"`
	Carbs      float64 `json:"carbs"`
	Fat        float64 `json:"fat"`
	Fiber      float64 `json:"fiber"`
	Sugar      float64 `json:"sugar"`
	Sodium     float64 `json:"sodium"`
	ServingSize string  `json:"serving_size"`
	ServingUnit string  `json:"serving_unit"`
}

type ExternalItem struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Brand       string       `json:"brand"`
	Category    string       `json:"category"`
	Macros      Macros       `json:"macros"`
	ImageURL    string       `json:"image_url"`
	Source      string       `json:"source"`
	SourceID    string       `json:"source_id"`
	RawData     interface{}  `json:"raw_data"` // Preserves original API response
}

type ImportResult struct {
	Success     bool      `json:"success"`
	ItemID      string    `json:"item_id,omitempty"`
	Error       string    `json:"error,omitempty"`
	ImportedAt  time.Time `json:"imported_at"`
}

type ImportRequest struct {
	ExternalItem ExternalItem `json:"external_item"`
	Edits        ItemEdits    `json:"edits"`
}

type ItemEdits struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Brand       *string  `json:"brand,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Macros      *Macros  `json:"macros,omitempty"`
	ImageURL    *string  `json:"image_url,omitempty"`
}

type DataImporter struct {
	repo     Repository
	tagMgr   TagManager
	imageMgr ImageManager
}

type Repository interface {
	CreateItem(item *CuratedItem) error
	UpdateItem(id string, item *CuratedItem) error
	GetItemBySource(source, sourceID string) (*CuratedItem, error)
}

type TagManager interface {
	EnsureTagsExist(tags []string) error
	NormalizeTags(tags []string) []string
}

type ImageManager interface {
	DownloadAndUploadImage(url string) (string, error)
}
```

## 2. Logic & Algorithms

### 2.1 ImportItem

```
FUNCTION ImportItem(request ImportRequest) ImportResult:
1. VALIDATE request:
   a. IF request.ExternalItem.Name is empty:
      RETURN ImportResult{Success: false, Error: "item name is required"}
   b. IF request.ExternalItem.Macros.Calories is empty AND request.Edits.Macros is empty:
      RETURN ImportResult{Success: false, Error: "macros are required"}

2. CHECK for duplicates:
   a. existing ← repo.GetItemBySource(request.ExternalItem.Source, request.ExternalItem.SourceID)
   b. IF existing is not null:
      i. IF request.ExternalItem.Name == existing.Name AND macros match:
         - RETURN ImportResult{Success: true, ItemID: existing.ID, Error: ""}
      ii. ELSE:
         - Continue to create new item (allow duplicates from external sources)

3. BUILD item from request.ExternalItem with request.Edits applied:
   a. item ← NEW CuratedItem
   b. item.Name ← COALESCE(request.Edits.Name, request.ExternalItem.Name)
   c. item.Description ← COALESCE(request.Edits.Description, request.ExternalItem.Description)
   d. item.Brand ← COALESCE(request.Edits.Brand, request.ExternalItem.Brand)
   e. item.Category ← COALESCE(request.Edits.Category, request.ExternalItem.Category)
   f. item.Macros ← COALESCE(request.Edits.Macros, request.ExternalItem.Macros)
   g. item.ImageURL ← COALESCE(request.Edits.ImageURL, request.ExternalItem.ImageURL)
   h. item.Source ← request.ExternalItem.Source
   i. item.SourceID ← request.ExternalItem.SourceID
   j. item.Tags ← request.Edits.Tags (if provided) ELSE request.ExternalItem.Category normalized

4. NORMALIZE tags:
   a. item.Tags ← tagMgr.NormalizeTags(item.Tags)

5. PROCESS image if ImageURL is provided:
   a. IF item.ImageURL is not empty AND item.ImageURL is external URL:
      i. localURL ← imageMgr.DownloadAndUploadImage(item.ImageURL)
      ii. IF localURL is not null:
          - item.ImageURL ← localURL

6. ENSURE tags exist in database:
   a. tagMgr.EnsureTagsExist(item.Tags)

7. SAVE item to database:
   a. item.ID ← generate UUID
   b. item.CreatedAt ← NOW()
   c. item.UpdatedAt ← NOW()
   d. err ← repo.CreateItem(item)
   e. IF err is not null:
      - RETURN ImportResult{Success: false, Error: err.Message}

8. RETURN ImportResult{Success: true, ItemID: item.ID}
```

### 2.2 ProcessBatchImport

```
FUNCTION ProcessBatchImport(requests []ImportRequest) []ImportResult:
1. results ← EMPTY list
2. FOR each request in requests:
   a. result ← ImportItem(request)
   b. APPEND result to results

3. RETURN results
```

### 2.3 ValidateAndSanitizeMacros

```
FUNCTION ValidateAndSanitizeMacros(macros Macros) Macros:
1. VALIDATE numeric fields:
   a. IF macros.Calories < 0: macros.Calories ← 0
   b. IF macros.Protein < 0: macros.Protein ← 0
   c. IF macros.Carbs < 0: macros.Carbs ← 0
   d. IF macros.Fat < 0: macros.Fat ← 0
   e. IF macros.Fiber < 0: macros.Fiber ← 0
   f. IF macros.Sugar < 0: macros.Sugar ← 0
   g. IF macros.Sodium < 0: macros.Sodium ← 0

2. SANITIZE serving information:
   a. IF macros.ServingSize is empty:
      - macros.ServingSize ← "100"
   b. IF macros.ServingUnit is empty:
      - macros.ServingUnit ← "g"

3. ROUND numeric values to 2 decimal places

4. RETURN macros
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Transition | Resolution |
| :--- | :--- | :--- |
| `InvalidRequest` | Request validation fails | Return error with specific field missing |
| `DuplicateItem` | Source+SourceID already exists | Skip or update based on config |
| `ImageDownloadFailed` | External image URL unreachable | Continue without image, log warning |
| `TagNormalizationFailed` | Tag contains invalid characters | Reject with cleaned tags suggestion |
| `DatabaseConnectionLost` | DB connection timeout | Retry 3x with exponential backoff |
| `DatabaseConstraintViolation` | Unique constraint violation | Skip duplicate, return existing ID |

### 3.2 State Transitions

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Initial   │────>│ Validating  │────>│ Processing  │────>│   Saving    │
│             │     │  Request    │     │   Image     │     │   to DB     │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │
       │                   │                   │                   │
       ▼                   ▼                   ▼                   ▼
  [Invalid]          [Validating]        [Processing]         [Complete]
                                         [Error: Image]
                                         [Continue]
```

### 3.3 Retry Strategy

- Database operations: 3 retries with exponential backoff (100ms, 500ms, 2s)
- Image download: 2 retries with 1s delay, then skip with warning
- Tag operations: No retry, fail fast on invalid tags

## 4. Component Interfaces

### 4.1 Public Functions

```go
// ImportItem imports a single curated item from external source
func (di *DataImporter) ImportItem(ctx context.Context, req ImportRequest) (*ImportResult, error)

// ProcessBatchImport imports multiple items in a single transaction
func (di *DataImporter) ProcessBatchImport(ctx context.Context, reqs []ImportRequest) ([]ImportResult, error)

// ValidateAndSanitizeMacros validates and normalizes macro values
func (di *DataImporter) ValidateAndSanitizeMacros(macros Macros) Macros
```

### 4.2 Repository Interface

```go
// CreateItem inserts a new curated item into the database
func (r *postgresRepository) CreateItem(ctx context.Context, item *CuratedItem) error

// UpdateItem updates an existing curated item by ID
func (r *postgresRepository) UpdateItem(ctx context.Context, id string, item *CuratedItem) error

// GetItemBySource retrieves an item by its external source and ID
func (r *postgresRepository) GetItemBySource(ctx context.Context, source, sourceID string) (*CuratedItem, error)
```

### 4.3 Handler Integration

```go
// DataImporterHandler handles HTTP requests for data import
type DataImporterHandler struct {
	importer *DataImporter
}

// POST /api/admin/import - Import single item
func (h *DataImporterHandler) ImportItem(c *fiber.Ctx) error

// POST /api/admin/import/batch - Import multiple items
func (h *DataImporterHandler) ProcessBatchImport(c *fiber.Ctx) error

// POST /api/admin/import/validate - Validate import request without saving
func (h *DataImporterHandler) ValidateImport(c *fiber.Ctx) error
```

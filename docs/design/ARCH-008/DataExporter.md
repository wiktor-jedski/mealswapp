# DataExporter

**Traceability:** ARCH-008

## 1. Data Structures & Types

```go
package exporter

import (
	"time"
)

type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
)

type ExportRequest struct {
	UserID    string       `json:"user_id"`
	Format    ExportFormat `json:"format"`
	Include   []string     `json:"include"` // e.g., ["profile", "saved_items", "diets", "history"]
	RequestID string       `json:"request_id"`
	CreatedAt time.Time    `json:"created_at"`
}

type ExportResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"` // "completed", "failed", "in_progress"
	DownloadURL string `json:"download_url,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"
	Error     string `json:"error,omitempty"`
}

type UserProfileData struct {
	UserID        string    `json:"user_id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Preferences   UserPreferences `json:"preferences"`
}

type UserPreferences struct {
	DefaultUnit     string   `json:"default_unit"` // "metric" or "imperial"
	DietaryRestrictions []string `json:"dietary_restrictions"`
	Allergies       []string `json:"allergies"`
	PreferredCuisines []string `json:"preferred_cuisines"`
}

type SavedItem struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	SavedAt     time.Time `json:"saved_at"`
}

type Diet struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Goals       []DietGoal `json:"goals"`
	CreatedAt   time.Time `json:"created_at"`
}

type DietNutrient  Goal struct {
	 string `json:"nutrient"`
	TargetValue float64 `json:"target_value"`
	Unit       string `json:"unit"`
}

type SearchHistory struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Query     string    `json:"query"`
	Filters   string    `json:"filters"` // JSON encoded
	ResultsCount int    `json:"results_count"`
	ClickedItems []string `json:"clicked_items"` // IDs of items clicked
	Timestamp time.Time `json:"timestamp"`
}

type ExportData struct {
	GeneratedAt   time.Time       `json:"generated_at"`
	Format        ExportFormat    `json:"format"`
	UserProfile   *UserProfileData `json:"user_profile,omitempty"`
	SavedItems    []SavedItem     `json:"saved_items,omitempty"`
	Diets         []Diet          `json:"diets,omitempty"`
	SearchHistory []SearchHistory `json:"search_history,omitempty"`
}
```

## 2. Logic & Algorithms

### 2.1 Export Data Flow

```
1. RECEIVE ExportRequest
   ├─ Validate user authentication (ARCH-006)
   ├─ Validate user has access to requested data
   └─ Generate or retrieve RequestID

2. GATHER user data from repository (ARCH-005)
   ├─ Fetch UserProfileData
   ├─ Fetch SavedItems (user-scoped query)
   ├─ Fetch Diets (user-scoped query)
   └─ Fetch SearchHistory (user-scoped query)

3. BUILD ExportData structure
   ├─ Populate UserProfileData if "profile" in Include
   ├─ Populate SavedItems if "saved_items" in Include
   ├─ Populate Diets if "diets" in Include
   └─ Populate SearchHistory if "history" in Include

4. SERIALIZE data to requested format
   ├─ IF Format == JSON:
   │   └─ Use json.Marshal with indentation
   └─ IF Format == CSV:
       ├─ Flatten nested structures
       ├─ Create headers from field names
       └─ Convert arrays to semicolon-separated values

5. WRITE to temporary storage
   ├─ Generate unique filename: export_{user_id}_{timestamp}.{ext}
   ├─ Upload to GCP Cloud Storage
   └─ Set expiry to 24 hours

6. RETURN ExportResponse with download URL
```

### 2.2 JSON Serialization Algorithm

```
FUNCTION ExportToJSON(data ExportData) ([]byte, error):
    1. Create marshaled, err := json.MarshalIndent(data, "", "  ")
    2. IF err != nil:
        RETURN nil, wrapError(ErrSerializationFailed, err)
    3. RETURN marshaled, nil
```

### 2.3 CSV Serialization Algorithm

```
FUNCTION ExportToCSV(data ExportData) ([]byte, error):
    1. Initialize buffer := bytes.NewBuffer(nil)
    2. Initialize writer := csv.NewWriter(buffer)
    3. WRITE profile section IF data.UserProfile != nil:
       ├─ WriteRow(["Section", "Profile"])
       ├─ WriteRow(["UserID", data.UserProfile.UserID])
       ├─ WriteRow(["Email", data.UserProfile.Email])
       ├─ WriteRow(["Name", data.UserProfile.Name])
       ├─ WriteRow(["CreatedAt", data.UserProfile.CreatedAt.Format(time.RFC3339)])
       ├─ WriteRow(["UpdatedAt", data.UserProfile.UpdatedAt.Format(time.RFC3339)])
       ├─ WriteRow(["DefaultUnit", data.UserProfile.Preferences.DefaultUnit])
       ├─ WriteRow(["DietaryRestrictions", join(data.UserProfile.Preferences.DietaryRestrictions, ";")])
       ├─ WriteRow(["Allergies", join(data.UserProfile.Preferences.Allergies, ";")])
       ├─ WriteRow(["PreferredCuisines", join(data.UserProfile.Preferences.PreferredCuisines, ";")])
       └─ Write empty row
    4. WRITE saved items section IF data.SavedItems not empty:
       ├─ WriteRow(["Section", "SavedItems"])
       ├─ WriteHeader(["ID", "Name", "Description", "Category", "CreatedAt", "SavedAt"])
       └─ FOR each item IN data.SavedItems:
           ├─ WriteRow([item.ID, item.Name, item.Description, item.Category, item.CreatedAt.Format(time.RFC3339), item.SavedAt.Format(time.RFC3339)])
           └─ Write empty row
    5. WRITE diets section IF data.Diets not empty:
       ├─ WriteRow(["Section", "Diets"])
       ├─ WriteHeader(["ID", "Name", "Description", "StartDate", "EndDate"])
       └─ FOR each diet IN data.Diets:
           ├─ endDate := ""
           ├─ IF diet.EndDate != nil:
           │   └─ endDate = diet.EndDate.Format(time.RFC3339)
           ├─ WriteRow([diet.ID, diet.Name, diet.Description, diet.StartDate.Format(time.RFC3339), endDate])
           ├─ WriteRow(["Goals"])
           ├─ WriteHeader(["Nutrient", "TargetValue", "Unit"])
           └─ FOR each goal IN diet.Goals:
               ├─ WriteRow([goal.Nutrient, strconv.FormatFloat(goal.TargetValue, 'f', 2, 64), goal.Unit])
               └─ Write empty row
    6. WRITE search history section IF data.SearchHistory not empty:
       ├─ WriteRow(["Section", "SearchHistory"])
       ├─ WriteHeader(["ID", "Query", "ResultsCount", "ClickedItems", "Timestamp"])
       └─ FOR each history IN data.SearchHistory:
           ├─ WriteRow([history.ID, history.Query, strconv.Itoa(history.ResultsCount), join(history.ClickedItems, ";"), history.Timestamp.Format(time.RFC3339)])
           └─ Write empty row
    7. Flush writer
    8. RETURN buffer.Bytes(), nil
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | Condition | HTTP Status | User Message |
| :--- | :--- | :--- | :--- |
| `ErrUnauthorized` | Missing or invalid auth token | 401 | Authentication required |
| `ErrForbidden` | User cannot access requested data | 403 | Access denied |
| `ErrInvalidFormat` | Unsupported export format | 400 | Invalid export format |
| `ErrInvalidInclude` | Unknown include section | 400 | Invalid data section requested |
| `ErrRepositoryFailure` | Database query failed | 500 | Unable to retrieve data |
| `ErrStorageFailure` | Cloud storage upload failed | 500 | Export file creation failed |
| `ErrSerializationFailed` | Data marshaling failed | 500 | Data processing error |
| `ErrExportTimeout` | Export took too long (>30s) | 504 | Export timed out |
| `ErrRateLimitExceeded` | Too many export requests | 429 | Too many requests |

### 3.2 State Transitions

```
IDLE → VALIDATING → GATHERING → SERIALIZING → STORING → COMPLETED
          ↓
        ERROR

GATHERING → SERIALIZING → STORING → COMPLETED
                ↓
              ERROR

COMPLETED → EXPIRED (after 24h)
```

### 3.3 Error Handling Strategy

```go
func (e *DataExporter) ExportData(ctx context.Context, req ExportRequest) (*ExportResponse, error) {
	// 1. Validate authentication
	userID, err := e.authService.ValidateSession(ctx)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if userID != req.UserID {
		return nil, ErrForbidden
	}

	// 2. Validate format
	if req.Format != FormatJSON && req.Format != FormatCSV {
		return nil, ErrInvalidFormat
	}

	// 3. Validate include sections
	validSections := map[string]bool{
		"profile":      true,
		"saved_items":  true,
		"diets":        true,
		"history":      true,
	}
	for _, section := range req.Include {
		if !validSections[section] {
			return nil, ErrInvalidInclude
		}
	}

	// 4. Gather data with timeout
	dataCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	data, err := e.gatherExportData(dataCtx, req.UserID, req.Include)
	if err != nil {
		return nil, err
	}

	// 5. Serialize with timeout
	var exportBytes []byte
	serialized := make(chan struct {
		bytes []byte
		err   error
	})
	go func() {
		bytes, err := e.serializeData(data, req.Format)
		serialized <- struct {
			bytes []byte
			err   error
		}{bytes, err}
	}()

	select {
	case result := <-serialized:
		if result.err != nil {
			return nil, ErrSerializationFailed
		}
		exportBytes = result.bytes
	case <-dataCtx.Done():
		return nil, ErrExportTimeout
	}

	// 6. Upload to storage
	downloadURL, fileSize, err := e.uploadToStorage(ctx, req.UserID, req.RequestID, req.Format, exportBytes)
	if err != nil {
		return nil, ErrStorageFailure
	}

	return &ExportResponse{
		RequestID:   req.RequestID,
		Status:      "completed",
		DownloadURL: downloadURL,
		FileSize:    fileSize,
	}, nil
}
```

## 4. Component Interfaces

### 4.1 DataExporter Interface

```go
type DataExporter interface {
	ExportData(ctx context.Context, req ExportRequest) (*ExportResponse, error)
	GetExportStatus(ctx context.Context, requestID string) (*ExportResponse, error)
	DeleteExport(ctx context.Context, requestID string) error
}
```

### 4.2 Repository Interface

```go
type UserDataRepository interface {
	GetUserProfile(ctx context.Context, userID string) (*UserProfileData, error)
	GetSavedItems(ctx context.Context, userID string) ([]SavedItem, error)
	GetDiets(ctx context.Context, userID string) ([]Diet, error)
	GetSearchHistory(ctx context.Context, userID string, limit int) ([]SearchHistory, error)
}
```

### 4.3 Storage Interface

```go
type Storage interface {
	UploadExport(ctx context.Context, userID string, requestID string, format ExportFormat, data []byte) (string, int64, error)
	GetExportURL(ctx context.Context, objectPath string) (string, error)
	DeleteExport(ctx context.Context, objectPath string) error
}
```

### 4.4 Handler Signatures

```go
func NewDataExporter(
	repo UserDataRepository,
	storage Storage,
	authService AuthService,
) *DataExporter

func (e *DataExporter) ExportData(ctx context.Context, req ExportRequest) (*ExportResponse, error)

func (e *DataExporter) serializeData(data ExportData, format ExportFormat) ([]byte, error)

func (e *DataExporter) gatherExportData(ctx context.Context, userID string, include []string) (*ExportData, error)

func (e *DataExporter) uploadToStorage(ctx context.Context, userID string, requestID string, format ExportFormat, data []byte) (string, int64, error)
```

### 4.5 Fiber Route Handler

```go
func (h *DataExporterHandler) RegisterRoutes(app *fiber.App) {
	exports := app.Group("/api/v1/exports")
	exports.Post("/", h.CreateExport)
	exports.Get("/:requestID", h.GetExportStatus)
	exports.Delete("/:requestID", h.DeleteExport)
}

func (h *DataExporterHandler) CreateExport(c *fiber.Ctx) error {
	var req ExportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	req.UserID = c.Locals("user_id").(string)
	req.RequestID = generateUUID()
	req.CreatedAt = time.Now()

	response, err := h.exporter.ExportData(c.Context(), req)
	if err != nil {
		return mapErrorToHTTP(err)
	}
	return c.Status(200).JSON(response)
}
```

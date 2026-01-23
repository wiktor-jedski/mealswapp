# FILE: SavedDataRepository.md

**Traceability:** ARCH-008

## 1. Data Structures & Types

```go
package repository

import (
    "context"
    "time"

    "github.com/google/uuid"
)

type SavedItemType string

const (
    SavedItemTypeRecipe    SavedItemType = "recipe"
    SavedItemTypeMealPlan  SavedItemType = "meal_plan"
    SavedItemTypeShoppingList SavedItemType = "shopping_list"
)

type SavedItem struct {
    ID          uuid.UUID    `json:"id" db:"id"`
    UserID      uuid.UUID    `json:"user_id" db:"user_id"`
    ItemType    SavedItemType `json:"item_type" db:"item_type"`
    ExternalID  *uuid.UUID   `json:"external_id,omitempty" db:"external_id"`
    Name        string       `json:"name" db:"name"`
    Metadata    JSONB        `json:"metadata,omitempty" db:"metadata"`
    CreatedAt   time.Time    `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type SearchHistory struct {
    ID          uuid.UUID  `json:"id" db:"id"`
    UserID      uuid.UUID  `json:"user_id" db:"user_id"`
    Query       string     `json:"query" db:"query"`
    Filters     JSONB      `json:"filters,omitempty" db:"filters"`
    ResultCount int        `json:"result_count" db:"result_count"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type Favorite struct {
    ID          uuid.UUID  `json:"id" db:"id"`
    UserID      uuid.UUID  `json:"user_id" db:"user_id"`
    ItemType    SavedItemType `json:"item_type" db:"item_type"`
    ExternalID  uuid.UUID  `json:"external_id" db:"external_id"`
    Name        string     `json:"name" db:"name"`
    Metadata    JSONB      `json:"metadata,omitempty" db:"metadata"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type UserPreference struct {
    ID            uuid.UUID  `json:"id" db:"id"`
    UserID        uuid.UUID  `json:"user_id" db:"user_id"`
    PreferenceKey string     `json:"preference_key" db:"preference_key"`
    PreferenceValue string   `json:"preference_value" db:"preference_value"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type SavedDataFilter struct {
    ItemType    *SavedItemType `json:"item_type,omitempty"`
    ExternalID  *uuid.UUID    `json:"external_id,omitempty"`
    CreatedAfter *time.Time   `json:"created_after,omitempty"`
    CreatedBefore *time.Time  `json:"created_before,omitempty"`
    Limit       int           `json:"limit"`
    Offset      int           `json:"offset"`
}

type PaginationResult struct {
    Items      interface{} `json:"items"`
    TotalCount int64       `json:"total_count"`
    Limit      int         `json:"limit"`
    Offset     int         `json:"offset"`
}

type JSONB map[string]interface{}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 SaveItem Procedure

```
PROCEDURE: SaveItem(ctx context.Context, item *SavedItem) error
STEPS:
1. Validate item.UserID is not empty
2. Validate item.ItemType is a valid SavedItemType
3. Validate item.Name is not empty and <= 255 characters
4. Generate UUID for item.ID if not set
5. Set item.CreatedAt to current time if not set
6. Set item.UpdatedAt to current time
7. Execute INSERT INTO saved_items VALUES (...)
8. If conflict on (user_id, item_type, external_id), UPDATE instead
9. Return error if database operation fails
```

### 2.2 GetSavedItems Procedure

```
PROCEDURE: GetSavedItems(ctx context.Context, userID uuid.UUID, filter SavedDataFilter) ([]SavedItem, error)
STEPS:
1. Validate userID is not empty
2. Build dynamic SQL query with WHERE clauses:
   - user_id = $1 (mandatory)
   - item_type = $2 (if filter.ItemType set)
   - external_id = $3 (if filter.ExternalID set)
   - created_at >= $4 (if filter.CreatedAfter set)
   - created_at <= $5 (if filter.CreatedBefore set)
3. Append ORDER BY created_at DESC
4. Append LIMIT and OFFSET for pagination
5. Execute query with constructed parameters
6. Scan results into []SavedItem slice
7. Return results and error if any
```

### 2.3 AddToFavorites Procedure

```
PROCEDURE: AddToFavorites(ctx context.Context, favorite *Favorite) error
STEPS:
1. Validate favorite.UserID is not empty
2. Validate favorite.ExternalID is not empty
3. Check if favorite already exists:
   SELECT id FROM favorites WHERE user_id = $1 AND external_id = $2
4. If exists, return nil (idempotent operation)
5. Generate UUID for favorite.ID if not set
6. Set favorite.CreatedAt to current time
7. Execute INSERT INTO favorites VALUES (...)
8. Return error if database operation fails
```

### 2.4 RecordSearchHistory Procedure

```
PROCEDURE: RecordSearchHistory(ctx context.Context, history *SearchHistory) error
STEPS:
1. Validate history.UserID is not empty
2. Validate history.Query is not empty
3. Generate UUID for history.ID if not set
4. Set history.CreatedAt to current time
5. Execute INSERT INTO search_history VALUES (...)
6. If search history count exceeds limit (e.g., 100 per user):
   DELETE FROM search_history
   WHERE user_id = $1
   AND id NOT IN (
       SELECT id FROM search_history
       WHERE user_id = $1
       ORDER BY created_at DESC
       LIMIT 100
   )
7. Return error if any
```

### 2.5 UpdateUserPreference Procedure

```
PROCEDURE: UpdateUserPreference(ctx context.Context, pref *UserPreference) error
STEPS:
1. Validate pref.UserID is not empty
2. Validate pref.PreferenceKey is not empty
3. Set pref.UpdatedAt to current time
4. Execute INSERT INTO user_preferences VALUES (...)
   ON CONFLICT (user_id, preference_key) DO UPDATE
   SET preference_value = EXCLUDED.preference_value,
       updated_at = EXCLUDED.updated_at
5. Return error if database operation fails
```

### 2.6 GetAllUserData Procedure (GDPR Export)

```
PROCEDURE: GetAllUserData(ctx context.Context, userID uuid.UUID) (*UserDataExport, error)
STEPS:
1. Validate userID is not empty
2. Query all saved items for user
3. Query all search history for user
4. Query all favorites for user
5. Query all user preferences for user
6. Query user profile data
7. Compile all data into UserDataExport struct
8. Return export data and error if any
```

### 2.7 DeleteAllUserData Procedure (GDPR Deletion)

```
PROCEDURE: DeleteAllUserData(ctx context.Context, userID uuid.UUID) error
STEPS:
1. Validate userID is not empty
2. Begin database transaction
3. DELETE FROM saved_items WHERE user_id = $1
4. DELETE FROM search_history WHERE user_id = $1
5. DELETE FROM favorites WHERE user_id = $1
6. DELETE FROM user_preferences WHERE user_id = $1
7. Commit transaction
8. If any error, rollback and return error
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Error Type | Handling Strategy |
| :--- | :--- | :--- |
| Database connection timeout | ErrDatabaseTimeout | Retry with exponential backoff (max 3 retries) |
| Constraint violation | ErrConstraintViolation | Return user-facing error with details |
| User not found | ErrUserNotFound | Return nil for read operations, error for writes |
| Invalid UUID format | ErrInvalidUUID | Return validation error to caller |
| Concurrency conflict | ErrOptimisticLock | Reload and retry operation (max 2 retries) |
| Query timeout | ErrQueryTimeout | Return timeout error, suggest retry |

### 3.2 State Transitions

```
State: Idle → Validating → Querying → Returning
         ↓         ↓           ↓
         └─────────┴───────────┴─ Error → ErrorState

ErrorState transitions:
- ErrDatabaseTimeout → Retry (max 3) → Idle or Failed
- ErrConstraintViolation → Failed (return to caller)
- ErrUserNotFound → Failed (return to caller)
- ErrInvalidUUID → Failed (return to caller)
- ErrOptimisticLock → Retry (max 2) → Idle or Failed
- ErrQueryTimeout → Failed (return to caller)
```

### 3.3 Isolation Level

- **Default Isolation:** Read Committed
- **For Deletion/Export:** Serializable (prevents phantom reads during bulk operations)
- **For Updates:** Repeatable Read (prevents non-repeatable reads)

## 4. Component Interfaces

### 4.1 SavedDataRepository Interface

```go
type SavedDataRepository interface {
    SaveItem(ctx context.Context, item *SavedItem) error
    GetSavedItems(ctx context.Context, userID uuid.UUID, filter SavedDataFilter) ([]SavedItem, error)
    DeleteSavedItem(ctx context.Context, itemID, userID uuid.UUID) error
    
    AddToFavorites(ctx context.Context, favorite *Favorite) error
    RemoveFromFavorites(ctx context.Context, favoriteID, userID uuid.UUID) error
    GetFavorites(ctx context.Context, userID uuid.UUID, itemType *SavedItemType) ([]Favorite, error)
    
    RecordSearchHistory(ctx context.Context, history *SearchHistory) error
    GetSearchHistory(ctx context.Context, userID uuid.UUID, limit int) ([]SearchHistory, error)
    ClearSearchHistory(ctx context.Context, userID uuid.UUID) error
    
    UpdateUserPreference(ctx context.Context, pref *UserPreference) error
    GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]UserPreference, error)
    DeleteUserPreference(ctx context.Context, userID uuid.UUID, key string) error
    
    GetAllUserData(ctx context.Context, userID uuid.UUID) (*UserDataExport, error)
    DeleteAllUserData(ctx context.Context, userID uuid.UUID) error
    
    WithTransaction(tx *sql.Tx) SavedDataRepository
}
```

### 4.2 Function Signatures

```go
func (r *savedDataRepository) SaveItem(ctx context.Context, item *SavedItem) error
func (r *savedDataRepository) GetSavedItems(ctx context.Context, userID uuid.UUID, filter SavedDataFilter) ([]SavedItem, error)
func (r *savedDataRepository) DeleteSavedItem(ctx context.Context, itemID, userID uuid.UUID) error
func (r *savedDataRepository) AddToFavorites(ctx context.Context, favorite *Favorite) error
func (r *savedDataRepository) RemoveFromFavorites(ctx context.Context, favoriteID, userID uuid.UUID) error
func (r *savedDataRepository) GetFavorites(ctx context.Context, userID uuid.UUID, itemType *SavedItemType) ([]Favorite, error)
func (r *savedDataRepository) RecordSearchHistory(ctx context.Context, history *SearchHistory) error
func (r *savedDataRepository) GetSearchHistory(ctx context.Context, userID uuid.UUID, limit int) ([]SearchHistory, error)
func (r *savedDataRepository) ClearSearchHistory(ctx context.Context, userID uuid.UUID) error
func (r *savedDataRepository) UpdateUserPreference(ctx context.Context, pref *UserPreference) error
func (r *savedDataRepository) GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]UserPreference, error)
func (r *savedDataRepository) DeleteUserPreference(ctx context.Context, userID uuid.UUID, key string) error
func (r *savedDataRepository) GetAllUserData(ctx context.Context, userID uuid.UUID) (*UserDataExport, error)
func (r *savedDataRepository) DeleteAllUserData(ctx context.Context, userID uuid.UUID) error
func (r *savedDataRepository) WithTransaction(tx *sql.Tx) SavedDataRepository
```

### 4.3 UserDataExport Struct

```go
type UserDataExport struct {
    UserID          uuid.UUID          `json:"user_id"`
    ExportedAt      time.Time          `json:"exported_at"`
    SavedItems      []SavedItem        `json:"saved_items"`
    SearchHistory   []SearchHistory    `json:"search_history"`
    Favorites       []Favorite         `json:"favorites"`
    Preferences     []UserPreference   `json:"preferences"`
    Profile         *UserProfile       `json:"profile,omitempty"`
}
```

### 4.4 Repository Factory

```go
func NewSavedDataRepository(db *sql.DB) SavedDataRepository {
    return &savedDataRepository{
        db:             db,
        tablePrefix:    "app_",
        queryTimeout:   30 * time.Second,
        maxRetries:     3,
    }
}
```

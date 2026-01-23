# FILE: ProfileController.md

**Traceability:** ARCH-008

## 1. Data Structures & Types

### 1.1 User Profile Domain Types

```go
type UserProfile struct {
    ID          string    `json:"id"`
    Email       string    `json:"email"`
    DisplayName string    `json:"display_name"`
    AvatarURL   string    `json:"avatar_url,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type UserPreferences struct {
    UserID           string        `json:"user_id"`
    DefaultUnit      UnitSystem    `json:"default_unit"` // METRIC or IMPERIAL
    DefaultDiet      DietType      `json:"default_diet"`
    Allergies        []Allergen    `json:"allergies"`
    CalorieGoal      *int          `json:"calorie_goal,omitempty"`
    MacroSplit       *MacroSplit   `json:"macro_split,omitempty"`
    Theme            ThemeMode     `json:"theme"` // LIGHT, DARK, SYSTEM
    Language         string        `json:"language"`
    Notifications    NotificationSettings `json:"notifications"`
    CreatedAt        time.Time     `json:"created_at"`
    UpdatedAt        time.Time     `json:"updated_at"`
}

type UnitSystem string

const (
    UnitSystemMetric   UnitSystem = "METRIC"
    UnitSystemImperial UnitSystem = "IMPERIAL"
)

type DietType string

const (
    DietTypeNone           DietType = "NONE"
    DietTypeVegetarian     DietType = "VEGETARIAN"
    DietTypeVegan          DietType = "VEGAN"
    DietTypeKeto           DietType = "KETO"
    DietTypePaleo          DietType = "PALEO"
    DietTypeLowFODMAP      DietType = "LOW_FODMAP"
    DietTypeHalal          DietType = "HALAL"
    DietTypeKosher         DietType = "KOSHER"
)

type Allergen string

const (
    AllergenGluten  Allergen = "GLUTEN"
    AllergenDairy   Allergen = "DAIRY"
    AllergenNuts    Allergen = "NUTS"
    AllergenSoy     Allergen = "SOY"
    AllergenEggs    Allergen = "EGGS"
    AllergenFish    Allergen = "FISH"
    AllergenShellfish Allergen = "SHELLFISH"
    AllergenSesame  Allergen = "SESAME"
)

type MacroSplit struct {
    ProteinPercent float64 `json:"protein_percent"`
    CarbsPercent   float64 `json:"carbs_percent"`
    FatPercent     float64 `json:"fat_percent"`
}

type ThemeMode string

const (
    ThemeModeLight  ThemeMode = "LIGHT"
    ThemeModeDark   ThemeMode = "DARK"
    ThemeModeSystem ThemeMode = "SYSTEM"
)

type NotificationSettings struct {
    EmailEnabled    bool `json:"email_enabled"`
    PushEnabled     bool `json:"push_enabled"`
    MealReminders   bool `json:"meal_reminders"`
    WeeklyReport    bool `json:"weekly_report"`
    PromoEmails     bool `json:"promo_emails"`
}
```

### 1.2 Saved Data Types

```go
type SavedMeal struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    RecipeIDs   []string  `json:"recipe_ids"`
    MealType    MealType  `json:"meal_type"`
    Tags        []string  `json:"tags"`
    IsFavorite  bool      `json:"is_favorite"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type MealType string

const (
    MealTypeBreakfast MealType = "BREAKFAST"
    MealTypeLunch     MealType = "LUNCH"
    MealTypeDinner    MealType = "DINNER"
    MealTypeSnack     MealType = "SNACK"
)

type SearchHistory struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    Query       string    `json:"query"`
    Filters     string    `json:"filters"` // JSON-encoded filter state
    ResultCount int       `json:"result_count"`
    CreatedAt   time.Time `json:"created_at"`
}

type FavoriteRecipe struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    RecipeID    string    `json:"recipe_id"`
    Notes       string    `json:"notes,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### 1.3 Export/Deletion Types

```go
type ExportFormat string

const (
    ExportFormatJSON ExportFormat = "JSON"
    ExportFormatCSV  ExportFormat = "CSV"
)

type ExportRequest struct {
    UserID  string       `json:"user_id"`
    Format  ExportFormat `json:"format"`
    Sections []string    `json:"sections"` // e.g., ["profile", "preferences", "history", "favorites"]
}

type ExportResult struct {
    DownloadURL string    `json:"download_url"`
    ExpiresAt   time.Time `json:"expires_at"`
    FileSize    int64     `json:"file_size"`
}

type DeletionRequest struct {
    UserID          string `json:"user_id"`
    ConfirmEmail    string `json:"confirm_email"`
    Password        string `json:"password"`
    DeleteBackups   bool   `json:"delete_backups"`
    Reason          string `json:"reason,omitempty"`
}

type DeletionStatus struct {
    UserID             string    `json:"user_id"`
    Status             DeletionStatusType `json:"status"`
    DeletedAt          *time.Time `json:"deleted_at,omitempty"`
    RecordsDeleted     int       `json:"records_deleted"`
    ScheduledPurgeAt   *time.Time `json:"scheduled_purge_at,omitempty"`
}

type DeletionStatusType string

const (
    DeletionStatusPending    DeletionStatusType = "PENDING"
    DeletionStatusProcessing DeletionStatusType = "PROCESSING"
    DeletionStatusCompleted  DeletionStatusType = "COMPLETED"
    DeletionStatusFailed     DeletionStatusType = "FAILED"
)
```

### 1.4 Controller State Types

```go
type ProfileController struct {
    repo            *repository.UserRepository
    prefMgr         *PreferenceManager
    dataExporter    *DataExporter
    accountDeleter  *AccountDeleter
    cache           *redis.Client
    logger          *log.Logger
    exportBucket    *storage.Bucket
}

type ControllerState string

const (
    StateIdle        ControllerState = "IDLE"
    StateProcessing  ControllerState = "PROCESSING"
    StateError       ControllerState = "ERROR"
)

type ControllerError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

## 2. Logic & Algorithms

### 2.1 GetUserProfile

```
FUNCTION GetUserProfile(userID string) -> (UserProfile, ControllerError)
    
    // Step 1: Validate userID is not empty
    IF userID IS EMPTY THEN
        RETURN empty UserProfile, ControllerError{
            Code: "INVALID_USER_ID",
            Message: "User ID is required"
        }
    END IF

    // Step 2: Validate user exists via Authentication service (ARCH-006)
    authStatus, authErr := authentication.ValidateSession(userID)
    IF authErr IS NOT NULL THEN
        RETURN empty UserProfile, ControllerError{
            Code: "AUTH_ERROR",
            Message: "Failed to validate user session",
            Details: authErr.Error()
        }
    END IF
    IF authStatus.IsAuthenticated == FALSE THEN
        RETURN empty UserProfile, ControllerError{
            Code: "UNAUTHORIZED",
            Message: "User is not authenticated"
        }
    END IF

    // Step 3: Query profile from repository with user-scoped isolation
    profile, repoErr := repo.GetUserProfile(userID)
    IF repoErr IS NOT NULL THEN
        RETURN empty UserProfile, ControllerError{
            Code: "REPOSITORY_ERROR",
            Message: "Failed to retrieve user profile",
            Details: repoErr.Error()
        }
    END IF

    // Step 4: Return profile
    RETURN profile, nil
END FUNCTION
```

### 2.2 UpdatePreferences

```
FUNCTION UpdatePreferences(userID string, prefs UserPreferences) -> (UserPreferences, ControllerError)
    
    // Step 1: Validate userID and preferences
    IF userID IS EMPTY THEN
        RETURN empty UserPreferences, ControllerError{
            Code: "INVALID_USER_ID",
            Message: "User ID is required"
        }
    END IF
    IF prefs.UserID != userID THEN
        RETURN empty UserPreferences, ControllerError{
            Code: "USER_ID_MISMATCH",
            Message: "User ID in preferences must match request"
        }
    END IF

    // Step 2: Validate preference values
    validationErr := validatePreferences(prefs)
    IF validationErr IS NOT NULL THEN
        RETURN empty UserPreferences, ControllerError{
            Code: "INVALID_PREFERENCES",
            Message: validationErr.Message
        }
    END IF

    // Step 3: Authenticate user
    authStatus, authErr := authentication.ValidateSession(userID)
    IF authErr IS NOT NULL OR authStatus.IsAuthenticated == FALSE THEN
        RETURN empty UserPreferences, ControllerError{
            Code: "UNAUTHORIZED",
            Message: "User is not authenticated"
        }
    END IF

    // Step 4: Check if unit system changed (triggers real-time recalculation)
    currentPrefs, getErr := prefMgr.GetPreferences(userID)
    IF getErr IS NULL AND currentPrefs.DefaultUnit != prefs.DefaultUnit THEN
        // Async recalculation - non-blocking
        GO func()
            recalcErr := prefMgr.RecalculateDisplayUnits(userID, prefs.DefaultUnit)
            IF recalcErr IS NOT NULL THEN
                logger.Error("Failed to recalculate display units", "error", recalcErr)
            END IF
        END
    END IF

    // Step 5: Update preferences in repository
    prefs.UpdatedAt = time.Now()
    updatedPrefs, updateErr := prefMgr.UpdatePreferences(userID, prefs)
    IF updateErr IS NOT NULL THEN
        RETURN empty UserPreferences, ControllerError{
            Code: "UPDATE_ERROR",
            Message: "Failed to update preferences",
            Details: updateErr.Error()
        }
    END IF

    // Step 6: Invalidate cached preferences
    cache.Del(context.Background(), fmt.Sprintf("prefs:%s", userID))

    RETURN updatedPrefs, nil
END FUNCTION
```

### 2.3 ExportUserData

```
FUNCTION ExportUserData(request ExportRequest) -> (ExportResult, ControllerError)
    
    // Step 1: Validate request
    IF request.UserID IS EMPTY THEN
        RETURN empty ExportResult, ControllerError{
            Code: "INVALID_USER_ID",
            Message: "User ID is required"
        }
    END IF
    IF request.Format NOT IN [JSON, CSV] THEN
        RETURN empty ExportResult, ControllerError{
            Code: "INVALID_FORMAT",
            Message: "Export format must be JSON or CSV"
        }
    END IF

    // Step 2: Authenticate user
    authStatus, authErr := authentication.ValidateSession(request.UserID)
    IF authErr IS NOT NULL OR authStatus.IsAuthenticated == FALSE THEN
        RETURN empty ExportResult, ControllerError{
            Code: "UNAUTHORIZED",
            Message: "User is not authenticated"
        }
    END IF

    // Step 3: Verify password confirmation (GDPR requirement)
    passwordValid, passErr := authentication.VerifyPassword(request.UserID, request.Password)
    IF passErr IS NOT NULL OR passwordValid == FALSE THEN
        RETURN empty ExportResult, ControllerError{
            Code: "INVALID_PASSWORD",
            Message: "Password verification failed"
        }
    END IF

    // Step 4: Collect data sections
    exportData := make(map[string]interface{})
    
    FOR EACH section IN request.Sections DO
        SWITCH section DO
            CASE "profile":
                exportData["profile"] = collectProfileData(request.UserID)
            CASE "preferences":
                exportData["preferences"] = collectPreferencesData(request.UserID)
            CASE "history":
                exportData["history"] = collectSearchHistory(request.UserID)
            CASE "favorites":
                exportData["favorites"] = collectFavorites(request.UserID)
            CASE "saved_meals":
                exportData["saved_meals"] = collectSavedMeals(request.UserID)
            DEFAULT:
                // Skip unknown sections
        END SWITCH
    END FOR

    // Step 5: Generate export file
    exportFile, genErr := dataExporter.Generate(exportData, request.Format)
    IF genErr IS NOT NULL THEN
        RETURN empty ExportResult, ControllerError{
            Code: "EXPORT_ERROR",
            Message: "Failed to generate export file",
            Details: genErr.Error()
        }
    END IF

    // Step 6: Upload to cloud storage with expiry
    uploadResult, uploadErr := dataExporter.UploadToCloud(exportFile, request.UserID)
    IF uploadErr IS NOT NULL THEN
        RETURN empty ExportResult, ControllerError{
            Code: "UPLOAD_ERROR",
            Message: "Failed to upload export file",
            Details: uploadErr.Error()
        }
    END IF

    // Step 7: Log export event for audit trail
    logger.Info("User data exported", "user_id", request.UserID, "format", request.Format, "sections", request.Sections)

    RETURN uploadResult, nil
END FUNCTION
```

### 2.4 RequestAccountDeletion

```
FUNCTION RequestAccountDeletion(request DeletionRequest) -> (DeletionStatus, ControllerError)
    
    // Step 1: Validate request
    IF request.UserID IS EMPTY THEN
        RETURN empty DeletionStatus, ControllerError{
            Code: "INVALID_USER_ID",
            Message: "User ID is required"
        }
    END IF
    IF request.ConfirmEmail IS EMPTY THEN
        RETURN empty DeletionStatus, ControllerError{
            Code: "INVALID_CONFIRMATION",
            Message: "Email confirmation is required"
        }
    END IF

    // Step 2: Get user profile to verify email
    profile, profileErr := repo.GetUserProfile(request.UserID)
    IF profileErr IS NOT NULL THEN
        RETURN empty DeletionStatus, ControllerError{
            Code: "PROFILE_ERROR",
            Message: "Could not retrieve user profile"
        }
    END IF
    IF profile.Email != request.ConfirmEmail THEN
        RETURN empty DeletionStatus, ControllerError{
            Code: "EMAIL_MISMATCH",
            Message: "Email confirmation does not match"
        }
    END IF

    // Step 3: Verify password
    passwordValid, passErr := authentication.VerifyPassword(request.UserID, request.Password)
    IF passErr IS NOT NULL OR passwordValid == FALSE THEN
        RETURN empty DeletionStatus, ControllerError{
            Code: "INVALID_PASSWORD",
            Message: "Password verification failed"
        }
    END IF

    // Step 4: Initiate deletion process
    deletionStatus, delErr := accountDeleter.InitiateDeletion(request.UserID, request.DeleteBackups)
    IF delErr IS NOT NULL THEN
        RETURN empty DeletionStatus, ControllerError{
            Code: "DELETION_ERROR",
            Message: "Failed to initiate account deletion",
            Details: delErr.Error()
        }
    END IF

    // Step 5: Send confirmation email
    GO func()
        emailErr := sendDeletionConfirmationEmail(request.UserID)
        IF emailErr IS NOT NULL THEN
            logger.Error("Failed to send deletion confirmation email", "error", emailErr)
        END IF
    END

    // Step 6: Log deletion request
    logger.Info("Account deletion requested", "user_id", request.UserID, "reason", request.Reason)

    RETURN deletionStatus, nil
END FUNCTION
```

### 2.5 GetSavedMeals

```
FUNCTION GetSavedMeals(userID string, pagination PaginationParams) -> ([]SavedMeal, PaginationResult, ControllerError)
    
    // Step 1: Validate inputs
    IF userID IS EMPTY THEN
        RETURN empty []SavedMeal, empty PaginationResult, ControllerError{
            Code: "INVALID_USER_ID",
            Message: "User ID is required"
        }
    END IF

    // Step 2: Authenticate
    authStatus, authErr := authentication.ValidateSession(userID)
    IF authErr IS NOT NULL OR authStatus.IsAuthenticated == FALSE THEN
        RETURN empty []SavedMeal, empty PaginationResult, ControllerError{
            Code: "UNAUTHORIZED",
            Message: "User is not authenticated"
        }
    END IF

    // Step 3: Query with user-scoped isolation (mandatory per ARCH-008)
    meals, count, queryErr := repo.GetSavedMealsForUser(userID, pagination)
    IF queryErr IS NOT NULL THEN
        RETURN empty []SavedMeal, empty PaginationResult, ControllerError{
            Code: "REPOSITORY_ERROR",
            Message: "Failed to retrieve saved meals",
            Details: queryErr.Error()
        }
    END IF

    // Step 4: Build pagination result
    result := PaginationResult{
        Items:      meals,
        TotalCount: count,
        Page:       pagination.Page,
        PageSize:   pagination.PageSize,
        HasMore:    (pagination.Page * pagination.PageSize) < count,
    }

    RETURN meals, result, nil
END FUNCTION
```

### 2.6 AddToFavorites

```
FUNCTION AddToFavorites(userID string, recipeID string) -> (FavoriteRecipe, ControllerError)
    
    // Step 1: Validate inputs
    IF userID IS EMPTY OR recipeID IS EMPTY THEN
        RETURN empty FavoriteRecipe, ControllerError{
            Code: "INVALID_INPUT",
            Message: "User ID and recipe ID are required"
        }
    END IF

    // Step 2: Authenticate
    authStatus, authErr := authentication.ValidateSession(userID)
    IF authErr IS NOT NULL OR authStatus.IsAuthenticated == FALSE THEN
        RETURN empty FavoriteRecipe, ControllerError{
            Code: "UNAUTHORIZED",
            Message: "User is not authenticated"
        }
    END IF

    // Step 3: Check if already favorited
    existing, getErr := repo.GetFavorite(userID, recipeID)
    IF getErr IS NULL AND existing.ID != "" THEN
        RETURN existing, nil // Already exists, return current
    END IF

    // Step 4: Create favorite
    favorite := FavoriteRecipe{
        ID:        generateUUID(),
        UserID:    userID,
        RecipeID:  recipeID,
        CreatedAt: time.Now(),
    }

    createErr := repo.CreateFavorite(favorite)
    IF createErr IS NOT NULL THEN
        RETURN empty FavoriteRecipe, ControllerError{
            Code: "CREATE_ERROR",
            Message: "Failed to add recipe to favorites",
            Details: createErr.Error()
        }
    END IF

    RETURN favorite, nil
END FUNCTION
```

### 2.7 RecordSearchHistory

```
FUNCTION RecordSearchHistory(userID string, query string, filters string, resultCount int) -> (SearchHistory, ControllerError)
    
    // Step 1: Validate inputs
    IF userID IS EMPTY OR query IS EMPTY THEN
        RETURN empty SearchHistory, ControllerError{
            Code: "INVALID_INPUT",
            Message: "User ID and query are required"
        }
    END IF

    // Step 2: Authenticate
    authStatus, authErr := authentication.ValidateSession(userID)
    IF authErr IS NOT NULL OR authStatus.IsAuthenticated == FALSE THEN
        RETURN empty SearchHistory, ControllerError{
            Code: "UNAUTHORIZED",
            Message: "User is not authenticated"
        }
    END IF

    // Step 3: Create history entry
    history := SearchHistory{
        ID:          generateUUID(),
        UserID:      userID,
        Query:       query,
        Filters:     filters,
        ResultCount: resultCount,
        CreatedAt:   time.Now(),
    }

    // Step 4: Save to repository
    createErr := repo.CreateSearchHistory(history)
    IF createErr IS NOT NULL THEN
        RETURN empty SearchHistory, ControllerError{
            Code: "CREATE_ERROR",
            Message: "Failed to record search history",
            Details: createErr.Error()
        }
    END IF

    // Step 5: Update localStorage cache for recent history (client-side caching per ARCH-008)
    GO func()
        updateLocalHistoryCache(userID, history)
    END

    RETURN history, nil
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | Condition | HTTP Status | User Message | Recovery Action |
|------------|-----------|-------------|--------------|-----------------|
| INVALID_USER_ID | userID is empty/null | 400 | Invalid request | Check input parameters |
| INVALID_INPUT | Missing required fields | 400 | Invalid request | Check input parameters |
| INVALID_FORMAT | Unsupported export format | 400 | Invalid format | Use JSON or CSV |
| INVALID_PREFERENCES | Validation failed | 400 | Invalid preferences | Review and correct preferences |
| USER_ID_MISMATCH | IDs don't match | 400 | Request mismatch | Verify request data |
| EMAIL_MISMATCH | Email confirmation failed | 400 | Email mismatch | Confirm correct email |
| UNAUTHORIZED | Not authenticated | 401 | Please log in | Re-authenticate |
| INVALID_PASSWORD | Password verification failed | 401 | Invalid password | Enter correct password |
| PROFILE_ERROR | Profile retrieval failed | 404 | Profile not found | Contact support |
| EXPORT_ERROR | Export generation failed | 500 | Export failed | Try again later |
| UPLOAD_ERROR | Cloud upload failed | 500 | Export failed | Try again later |
| CREATE_ERROR | Repository create failed | 500 | Operation failed | Try again later |
| UPDATE_ERROR | Repository update failed | 500 | Update failed | Try again later |
| REPOSITORY_ERROR | Generic repository error | 500 | Service error | Contact support |
| DELETION_ERROR | Deletion process failed | 500 | Deletion failed | Contact support |
| AUTH_ERROR | Authentication service error | 503 | Service unavailable | Try again later |

### 3.2 State Transitions

```
IDLE --(valid request)--> PROCESSING
PROCESSING --(success)--> IDLE
PROCESSING --(error)--> ERROR
ERROR --(new request)--> PROCESSING
ERROR --(recovery)--> IDLE
```

### 3.3 Retry Logic

For transient errors (repository timeouts, cloud storage errors):
- Maximum 3 retry attempts
- Exponential backoff: 100ms, 500ms, 2.5s
- Log each retry attempt with context

### 3.4 Data Isolation Enforcement

All repository queries MUST include userID filter:
```go
// WRONG - susceptible to cross-user access
repo.GetSavedMeals(pagination)

// CORRECT - enforces user-scoped isolation
repo.GetSavedMealsForUser(userID, pagination)
```

### 3.5 GDPR Compliance States

| State | Description | Action |
|-------|-------------|--------|
| EXPORT_PENDING | Export being generated | Async processing, poll for completion |
| EXPORT_READY | Export file available | Provide download URL (expires in 24h) |
| DELETION_SCHEDULED | Deletion queued | 30-day grace period before purge |
| DELETION_PROCESSING | Actual deletion in progress | Cancel not possible |
| DELETION_COMPLETED | All data purged | Account fully removed |

## 4. Component Interfaces

### 4.1 Public Methods

```go
type ProfileControllerInterface interface {
    GetUserProfile(userID string) (UserProfile, ControllerError)
    UpdateUserProfile(userID string, profile UpdateProfileRequest) (UserProfile, ControllerError)
    GetPreferences(userID string) (UserPreferences, ControllerError)
    UpdatePreferences(userID string, prefs UserPreferences) (UserPreferences, ControllerError)
    GetSavedMeals(userID string, pagination PaginationParams) ([]SavedMeal, PaginationResult, ControllerError)
    CreateSavedMeal(userID string, meal SavedMeal) (SavedMeal, ControllerError)
    UpdateSavedMeal(userID string, mealID string, meal SavedMeal) (SavedMeal, ControllerError)
    DeleteSavedMeal(userID string, mealID string) ControllerError
    GetSearchHistory(userID string, limit int) ([]SearchHistory, ControllerError)
    ClearSearchHistory(userID string) ControllerError
    RecordSearchHistory(userID string, query string, filters string, resultCount int) (SearchHistory, ControllerError)
    GetFavorites(userID string, pagination PaginationParams) ([]FavoriteRecipe, PaginationResult, ControllerError)
    AddToFavorites(userID string, recipeID string) (FavoriteRecipe, ControllerError)
    RemoveFromFavorites(userID string, recipeID string) ControllerError
    ExportUserData(request ExportRequest, password string) (ExportResult, ControllerError)
    RequestAccountDeletion(request DeletionRequest) (DeletionStatus, ControllerError)
    CancelDeletionRequest(userID string, password string) ControllerError
    GetDeletionStatus(userID string) (DeletionStatus, ControllerError)
}
```

### 4.2 Pagination Parameters

```go
type PaginationParams struct {
    Page     int `query:"page"`
    PageSize int `query:"page_size"`
}

func (p PaginationParams) Offset() int {
    return (p.Page - 1) * p.PageSize
}

func (p PaginationParams) Validate() ControllerError {
    if p.Page < 1 {
        p.Page = 1
    }
    if p.PageSize < 1 || p.PageSize > 100 {
        p.PageSize = 20
    }
    return nil
}
```

### 4.3 Update Profile Request

```go
type UpdateProfileRequest struct {
    DisplayName *string `json:"display_name"`
    AvatarURL   *string `json:"avatar_url"`
}
```

### 4.4 Repository Interface (ARCH-005 Dependency)

```go
type UserRepositoryInterface interface {
    GetUserProfile(userID string) (UserProfile, error)
    UpdateUserProfile(userID string, profile UserProfile) (UserProfile, error)
    GetPreferences(userID string) (UserPreferences, error)
    UpdatePreferences(userID string, prefs UserPreferences) (UserPreferences, error)
    GetSavedMealsForUser(userID string, pagination PaginationParams) ([]SavedMeal, int, error)
    CreateSavedMeal(meal SavedMeal) error
    UpdateSavedMeal(userID string, mealID string, meal SavedMeal) error
    DeleteSavedMeal(userID string, mealID string) error
    GetSearchHistoryForUser(userID string, limit int) ([]SearchHistory, error)
    CreateSearchHistory(history SearchHistory) error
    ClearSearchHistoryForUser(userID string) error
    GetFavoritesForUser(userID string, pagination PaginationParams) ([]FavoriteRecipe, int, error)
    CreateFavorite(favorite FavoriteRecipe) error
    DeleteFavorite(userID string, recipeID string) error
    GetFavorite(userID string, recipeID string) (FavoriteRecipe, error)
}
```

### 4.5 Authentication Interface (ARCH-006 Dependency)

```go
type AuthenticationServiceInterface interface {
    ValidateSession(userID string) (AuthStatus, error)
    VerifyPassword(userID string, password string) (bool, error)
    GetUserEmail(userID string) (string, error)
}
```

### 4.6 Data Exporter Interface

```go
type DataExporterInterface interface {
    Generate(data map[string]interface{}, format ExportFormat) ([]byte, error)
    UploadToCloud(data []byte, userID string) (ExportResult, error)
}
```

### 4.7 Account Deleter Interface

```go
type AccountDeleterInterface interface {
    InitiateDeletion(userID string, deleteBackups bool) (DeletionStatus, error)
    CancelDeletion(userID string) error
    ProcessDeletion(userID string) error
    GetStatus(userID string) (DeletionStatus, error)
}
```

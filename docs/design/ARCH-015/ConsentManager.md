# ConsentManager

**Traceability:** ARCH-015

## 1. Data Structures & Types

### ConsentRecord
```go
type ConsentRecord struct {
    ID              string    `json:"id" db:"id"`
    UserID          string    `json:"user_id" db:"user_id"`
    ConsentType     string    `json:"consent_type" db:"consent_type"` // "privacy_policy", "terms_of_service", "marketing"
    ConsentGiven    bool      `json:"consent_given" db:"consent_given"`
    ConsentVersion  string    `json:"consent_version" db:"consent_version"`
    IPAddress       string    `json:"ip_address" db:"ip_address"`
    UserAgent       string    `json:"user_agent" db:"user_agent"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
```

### ConsentRequest
```go
type ConsentRequest struct {
    UserID          string   `json:"user_id" validate:"required,uuid"`
    PrivacyPolicy   bool     `json:"privacy_policy" validate:"required,eq=true"`
    TermsOfService  bool     `json:"terms_of_service" validate:"required,eq=true"`
    Marketing       *bool    `json:"marketing,omitempty"` // Optional, nil means no change
}
```

### ConsentResponse
```go
type ConsentResponse struct {
    Success         bool     `json:"success"`
    ConsentStatus   map[string]bool `json:"consent_status"`
    MissingConsents []string `json:"missing_consents,omitempty"`
    Message         string   `json:"message,omitempty"`
}
```

### DeletionRequest
```go
type DeletionRequest struct {
    UserID          string    `json:"user_id" validate:"required,uuid"`
    RequestedAt     time.Time `json:"requested_at"`
    ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
    Status          string    `json:"status"` // "pending", "processing", "completed", "failed"
    ErrorMessage    string    `json:"error_message,omitempty"`
}
```

### DeletionStatus
```go
type DeletionStatus struct {
    RequestID       string    `json:"request_id"`
    UserID          string    `json:"user_id"`
    Status          string    `json:"status"`
    Progress        int       `json:"progress"` // 0-100
    CompletedSteps  []string  `json:"completed_steps"`
    RemainingSteps  []string  `json:"remaining_steps"`
    EstimatedTime   string    `json:"estimated_time,omitempty"`
    ErrorMessage    string    `json:"error_message,omitempty"`
}
```

### DisclaimerContent
```go
type DisclaimerContent struct {
    ID              string   `json:"id" db:"id"`
    Type            string   `json:"type"` // "medical", "legal", "general"
    Title           string   `json:"title" db:"title"`
    Content         string   `json:"content" db:"content"`
    Version         string   `json:"version" db:"version"`
    EffectiveDate   time.Time `json:"effective_date" db:"effective_date"`
    IsActive        bool     `json:"is_active" db:"is_active"`
}
```

### BackupMetadata
```go
type BackupMetadata struct {
    ID              string    `json:"id" db:"id"`
    BackupType      string    `json:"backup_type"` // "full", "incremental"
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
    ExpiresAt       time.Time `json:"expires_at" db:"expires_at"`
    SizeBytes       int64     `json:"size_bytes" db:"size_bytes"`
    Status          string    `json:"status"` // "creating", "ready", "expired", "deleted"
    Location        string    `json:"location" db:"location"`
}
```

## 2. Logic & Algorithms

### 2.1 CaptureConsent Algorithm

```
FUNCTION CaptureConsent(request ConsentRequest) -> ConsentResponse:
    1. Validate request.UserID exists and is valid UUID
    2. Check if request.PrivacyPolicy == true
       IF false:
           RETURN ConsentResponse with missing_consents = ["privacy_policy"]
    3. Check if request.TermsOfService == true
       IF false:
           RETURN ConsentResponse with missing_consents = ["terms_of_service"]
    4. Extract IP address from context
    5. Extract User-Agent from context
    6. Get current consent version from configuration
    7. FOR EACH consent_type IN ["privacy_policy", "terms_of_service", "marketing"]:
        a. IF consent_type == "marketing" AND request.Marketing == nil:
            CONTINUE to next iteration
        b. value = true for required types, or request.Marketing for optional
        c. UPSERT into consent_records table:
            - user_id = request.UserID
            - consent_type = consent_type
            - consent_given = value
            - consent_version = current_version
            - ip_address = extracted_ip
            - user_agent = extracted_user_agent
            - updated_at = NOW()
    8. Update user profile consent_status = "completed"
    9. RETURN ConsentResponse with success = true
```

### 2.2 ValidateRegistrationConsent Algorithm

```
FUNCTION ValidateRegistrationConsent(userID string) -> (bool, []string):
    1. Query consent_records WHERE user_id = userID
    2. consent_status = {}
    3. FOR EACH record IN query_results:
        consent_status[record.consent_type] = record.consent_given
    4. Check required consents:
        a. IF NOT consent_status["privacy_policy"]:
            missing[] = "privacy_policy"
        b. IF NOT consent_status["terms_of_service"]:
            missing[] = "terms_of_service"
    5. IF missing is not empty:
        RETURN false, missing
    6. RETURN true, []
```

### 2.3 ProcessDeletionRequest Algorithm

```
FUNCTION ProcessDeletionRequest(userID string) -> DeletionStatus:
    1. Create DeletionRequest record with status = "pending"
    2. Update status = "processing"
    3. Get all data repositories from ARCH-005:
        a. user_profile_repo
        b. meals_repo
        c. preferences_repo
        d. activity_log_repo
    4. FOR EACH repository IN repositories:
        a. Execute soft delete (set deleted_at = NOW(), deleted = true)
        b. Record completion in deletion_steps
    5. Schedule backup purge job:
        a. Add to job queue with delay = 30 days
        b. Action = "purge_user_backup"
    6. Update deletion_request status = "completed"
    7. RETURN DeletionStatus with completed_steps
```

### 2.4 EnforceBackupRetention Algorithm

```
FUNCTION EnforceBackupRetention() -> int:
    1. Query backup_metadata WHERE status = "ready" AND expires_at < NOW()
    2. expired_count = 0
    3. FOR EACH backup IN query_results:
        a. Delete backup file from storage
        b. Update backup.status = "deleted"
        c. expired_count++
    4. RETURN expired_count
```

### 2.5 RenderDisclaimer Algorithm

```
FUNCTION RenderDisclaimer(disclaimerType string) -> DisclaimerContent:
    1. Query disclaimer_content WHERE type = disclaimerType AND is_active = true
    2. ORDER BY effective_date DESC LIMIT 1
    3. IF no results:
        RETURN default disclaimer for type
    4. RETURN first result
```

## 3. State Management & Error Handling

### 3.1 Consent States

| State | Condition | Transition |
|-------|-----------|------------|
| `pending` | User started registration but not completed | → `completed` when both required consents given |
| `completed` | All required consents captured | → `withdrawn` if user requests deletion |
| `withdrawn` | User revoked consent via deletion request | None (triggers deletion workflow) |

### 3.2 Deletion Request States

| State | Condition | Transition |
|-------|-----------|------------|
| `pending` | Request received, not yet processed | → `processing` when worker picks up |
| `processing` | Deletion in progress | → `completed` or `failed` |
| `completed` | All data deleted from primary database | Final state |
| `failed` | Error occurred during deletion | → `processing` (retry) or manual intervention |

### 3.3 Error States

| Error | Condition | Handler |
|-------|-----------|---------|
| `MissingRequiredConsent` | Privacy Policy or ToS not accepted | Return validation error with missing_consents list |
| `UserNotFound` | UserID does not exist | Return 404 with error message |
| `DeletionInProgress` | User already has pending deletion request | Return 409 with request ID |
| `BackupPurgeFailed` | Cannot delete expired backup | Log error, alert operations team |
| `DatabaseConnectionFailed` | Cannot connect to PostgreSQL | Return 503, retry with exponential backoff |
| `RedisConnectionFailed` | Cannot connect to Redis for job queue | Return 503, queue consent for later processing |

### 3.4 Retry Logic

```
FUNCTION processDeletionWithRetry(userID string, maxRetries int) -> DeletionStatus:
    FOR attempt FROM 1 TO maxRetries:
        TRY:
            ProcessDeletionRequest(userID)
            RETURN success
        CATCH error:
            IF attempt < maxRetries:
                sleep(2^attempt seconds)  // exponential backoff
                CONTINUE
            ELSE:
                UPDATE deletion_request status = "failed"
                RETURN failure
```

## 4. Component Interfaces

### 4.1 Internal Functions

```go
package compliance

// Consent capture and validation
func CaptureConsent(ctx fiber.Ctx) error
func ValidateRegistrationConsent(userID string) (bool, []string, error)
func GetUserConsentStatus(userID string) (map[string]bool, error)
func RevokeConsent(userID string, consentType string) error

// Deletion request handling
func CreateDeletionRequest(userID string) (*DeletionRequest, error)
func ProcessDeletionRequest(requestID string) error
func GetDeletionStatus(requestID string) (*DeletionStatus, error)

// Disclaimer management
func GetDisclaimer(disclaimerType string) (*DisclaimerContent, error)
func RenderDisclaimerHTML(disclaimerType string) (string, error)

// Backup and retention
func CreateBackup(backupType string) error
func EnforceBackupRetention() (int, error)
func RestoreFromBackup(backupID string, userID string) error
```

### 4.2 Repository Interfaces

```go
type ConsentRepository interface {
    Upsert(record *ConsentRecord) error
    GetByUserID(userID string) ([]ConsentRecord, error)
    GetByType(userID, consentType string) (*ConsentRecord, error)
    DeleteByUserID(userID string) error
}

type DeletionRequestRepository interface {
    Create(request *DeletionRequest) error
    GetByID(id string) (*DeletionRequest, error)
    GetPending() ([]DeletionRequest, error)
    UpdateStatus(id string, status string) error
    UpdateProgress(id string, progress int) error
}

type DisclaimerRepository interface {
    GetActiveByType(disclaimerType string) (*DisclaimerContent, error)
    GetAllActive() ([]DisclaimerContent, error)
    Create(content *DisclaimerContent) error
    Deactivate(id string) error
}

type BackupRepository interface {
    Create(metadata *BackupMetadata) error
    GetByID(id string) (*BackupMetadata, error)
    GetExpired() ([]BackupMetadata, error)
    MarkDeleted(id string) error
    DeletePhysical(id string) error
}
```

### 4.3 External Dependencies

| Dependency | Type | Interface |
|------------|------|-----------|
| ARCH-005 | Data Repository | UserRepository, MealRepository, PreferenceRepository |
| ARCH-008 | User Profile | UserProfileService |
| Redis | Job Queue | go-redis/queue for backup purge scheduling |
| PostgreSQL | Primary Database | lib/pq driver |

### 4.4 API Endpoints

```
POST   /api/v1/consent              - Capture consent from user
GET    /api/v1/consent/:userId      - Get user's consent status
POST   /api/v1/deletion/request     - Request account deletion
GET    /api/v1/deletion/:requestId  - Get deletion request status
GET    /api/v1/disclaimer/:type     - Get disclaimer content
GET    /api/v1/disclaimer/:type/html - Get disclaimer rendered as HTML
```

### 4.5 Job Queue Tasks

```go
type BackupPurgeTask struct {
    BackupID   string    `json:"backup_id"`
    UserID     string    `json:"user_id"`
    ScheduledAt time.Time `json:"scheduled_at"`
}

type DeletionTask struct {
    RequestID  string    `json:"request_id"`
    UserID     string    `json:"user_id"`
    Retries    int       `json:"retries"`
}
```

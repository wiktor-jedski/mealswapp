# AccountDeleter - Detailed Design

**Traceability:** ARCH-008

---

## 1. Data Structures & Types

```go
package user

import "time"

// AccountDeleter handles GDPR-compliant account deletion
type AccountDeleter struct {
    repo       *SavedDataRepository
    auth       *AuthService
    cache      *RedisClient
    logger     *Logger
    queue      *RedisQueue
}

// DeletionRequest represents a user account deletion request
type DeletionRequest struct {
    UserID        string    `json:"user_id"`
    RequestedAt   time.Time `json:"requested_at"`
    ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
    Status        DeletionStatus `json:"status"`
    CascadeTables []string  `json:"cascade_tables"`
}

// DeletionStatus represents the state of a deletion request
type DeletionStatus string

const (
    DeletionStatusPending    DeletionStatus = "pending"
    DeletionStatusConfirmed  DeletionStatus = "confirmed"
    DeletionStatusProcessing DeletionStatus = "processing"
    DeletionStatusCompleted  DeletionStatus = "completed"
    DeletionStatusFailed     DeletionStatus = "failed"
)

// DeletionResult contains the outcome of account deletion
type DeletionResult struct {
    UserID           string            `json:"user_id"`
    DeletedRecords   map[string]int64  `json:"deleted_records"`
    TotalRowsRemoved int64             `json:"total_rows_removed"`
    Duration         time.Duration     `json:"duration"`
    Error            *DeletionError    `json:"error,omitempty"`
}

// DeletionError contains error details for failed deletions
type DeletionError struct {
    Code        string `json:"code"`
    Message     string `json:"message"`
    TableName   string `json:"table_name,omitempty"`
    Retryable   bool   `json:"retryable"`
}

// TablesToDelete defines the order and tables for cascading deletion
var TablesToDelete = []TableInfo{
    {Name: "user_sessions", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_search_history", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_favorites", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_diets", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_allergies", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_meal_plans", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_shopping_lists", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "data_export_requests", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_notifications", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_preferences", ParentColumn: "user_id", DeleteColumn: "user_id"},
    {Name: "user_profiles", ParentColumn: "id", DeleteColumn: "id"},
}

// TableInfo holds metadata about tables for deletion
type TableInfo struct {
    Name          string
    ParentColumn  string
    DeleteColumn  string
    SoftDelete    bool
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Request Account Deletion Flow

```
FUNCTION RequestDeletion(userID: string) -> DeletionRequest
    1. VALIDATE userID is not empty
    2. FETCH current user from database to verify existence
    3. IF user does not exist
           RETURN Error: "User not found"
    4. CHECK if deletion request already exists
    5. IF pending request exists
           RETURN existing request
    6. CREATE DeletionRequest with status = "pending"
    7. STORE request in deletion_requests table
    8. SEND confirmation email to user's email address
    9. RETURN DeletionRequest
```

### 2.2 Confirm and Process Deletion Flow

```
FUNCTION ConfirmDeletion(requestID: string, confirmationToken: string) -> DeletionResult
    1. FETCH DeletionRequest by requestID
    2. IF request not found
           RETURN Error: "Request not found"
    3. VALIDATE confirmationToken matches stored token
    4. IF token mismatch
           RETURN Error: "Invalid confirmation token"
    5. UPDATE request status to "processing"
    6. INITIATE background job for deletion
    7. RETURN success acknowledgment
```

### 2.3 Execute Deletion Flow

```
FUNCTION ExecuteDeletion(userID: string) -> DeletionResult
    1. INITIALIZE DeletionResult with UserID
    2. START transaction on database
    3. SET totalRowsRemoved = 0

    4. FOR EACH table IN TablesToDelete
        a. EXECUTE delete query: DELETE FROM table WHERE deleteColumn = userID
        b. GET affectedRows from query result
        c. IF error occurs
               ROLLBACK transaction
               RETURN DeletionResult with error
        d. INCREMENT totalRowsRemoved by affectedRows
        e. STORE count in DeletedRecords[table.Name]

    6. DELETE user session from Redis
       EXECUTE: DEL session:user:{userID}
       EXECUTE: DEL token:{userID}:*

    7. INVALIDATE all cached data for user
       EXECUTE: DEL cache:user:{userID}:*

    8. COMMIT transaction

    9. SEND deletion confirmation email

    10. LOG deletion audit record
        INSERT INTO audit_log (user_id, action, timestamp, details)
        VALUES (userID, 'ACCOUNT_DELETED', NOW(), json result)

    11. RETURN DeletionResult
```

### 2.4 Cascading Deletion Order

The deletion follows a specific order to maintain referential integrity:

1. **Session Data** - Remove active sessions first (Redis)
2. **User-Generated Content** - Search history, favorites, meal plans
3. **User Preferences** - Diets, allergies, preferences
4. **User Profile** - Final deletion of core profile record

---

## 3. State Management & Error Handling

### 3.1 Deletion Request States

| State | Trigger | Next State | Valid Transitions |
|-------|---------|------------|-------------------|
| pending | Initial request | confirmed, expired | User confirms via email |
| confirmed | Email confirmation | processing | Background job picks up |
| processing | Job started | completed, failed | Deletion execution |
| completed | All data removed | terminal | None |
| failed | Deletion error | pending (retry) | Retry with backoff |

### 3.2 Error States and Recovery

| Error Code | Condition | Retryable | Recovery Action |
|------------|-----------|-----------|-----------------|
| ERR_USER_NOT_FOUND | User ID does not exist | No | Return 404 to client |
| ERR_REQUEST_NOT_FOUND | Invalid request ID | No | Return 404 to client |
| ERR_INVALID_TOKEN | Wrong confirmation token | No | Prompt for new confirmation |
| ERR_DB_CONSTRAINT | Foreign key violation | No | Manual intervention required |
| ERR_DB_CONNECTION | Database unavailable | Yes | Retry with exponential backoff |
| ERR_REDIS_UNAVAILABLE | Redis connection failed | Yes | Retry; may skip cache cleanup |
| ERR_PARTIAL_DELETE | Some tables deleted, others failed | Yes | Retry from failed table |
| ERR_EMAIL_FAILED | Confirmation email failed | Yes | Queue retry |

### 3.3 State Transition Diagram

```
                    [pending]
                        |
              (email sent to user)
                        |
          +-------------+-------------+
          |                           |
    (user confirms)             (user cancels)
          |                           |
          v                           v
    [confirmed]                  [cancelled]
          |                           |
    (job queued)                   (end)
          |
          v
    [processing]
          |
    +-----+-----+
    |           |
(success)   (failure)
    |           |
    v           v
[completed]   [failed]
                  |
            (retry after delay)
                  |
                  v
            [pending]  (new request)
```

### 3.4 Concurrent Deletion Handling

- Use `SELECT FOR UPDATE` on user profile to prevent concurrent deletions
- Set Redis distributed lock during deletion execution
- Idempotent deletion: Running deletion twice produces same result
- Background job uses idempotency key to prevent duplicate processing

---

## 4. Component Interfaces

### 4.1 Public Methods

```go
// NewAccountDeleter creates a new AccountDeleter instance
func NewAccountDeleter(
    repo *SavedDataRepository,
    auth *AuthService,
    cache *RedisClient,
    queue *RedisQueue,
    logger *Logger,
) *AccountDeleter

// RequestDeletion initiates an account deletion request
// Returns DeletionRequest or error
func (ad *AccountDeleter) RequestDeletion(ctx context.Context, userID string) (*DeletionRequest, error)

// ConfirmDeletion processes the user's confirmation and starts deletion
// Returns DeletionResult or error
func (ad *AccountDeleter) ConfirmDeletion(ctx context.Context, requestID string, token string) (*DeletionResult, error)

// CancelDeletion allows user to cancel a pending deletion request
func (ad *AccountDeleter) CancelDeletion(ctx context.Context, requestID string, userID string) error

// GetDeletionStatus returns the current status of a deletion request
func (ad *AccountDeleter) GetDeletionStatus(ctx context.Context, requestID string) (*DeletionRequest, error)

// ExecuteImmediateDeletion performs immediate deletion (admin use only)
// Requires elevated privileges
func (ad *AccountDeleter) ExecuteImmediateDeletion(ctx context.Context, userID string, adminID string) (*DeletionResult, error)
```

### 4.2 Internal Methods

```go
// deleteFromTable removes records from a specific table
// Returns number of deleted rows or error
func (ad *AccountDeleter) deleteFromTable(ctx context.Context, tx *sql.Tx, table TableInfo, userID string) (int64, error)

// cleanupRedisUserData removes all Redis keys associated with user
func (ad *AccountDeleter) cleanupRedisUserData(ctx context.Context, userID string) error

// sendDeletionConfirmationEmail sends email to confirm deletion
func (ad *AccountDeleter) sendDeletionConfirmationEmail(ctx context.Context, userID string, requestID string) error

// sendDeletionCompleteEmail notifies user of successful deletion
func (ad *AccountDeleter) sendDeletionCompleteEmail(ctx context.Context, email string, result *DeletionResult) error

// logAuditRecord stores deletion audit trail
func (ad *AccountDeleter) logAuditRecord(ctx context.Context, userID string, action string, details []byte) error

// generateConfirmationToken creates secure random token
func (ad *AccountDeleter) generateConfirmationToken() (string, error)
```

### 4.3 Database Queries

```go
// Delete from table with user scope
const deleteFromTable = `
    DELETE FROM %s
    WHERE %s = $1
    RETURNING id
`

// Soft delete variant for audit compliance
const softDeleteFromTable = `
    UPDATE %s
    SET deleted_at = NOW(), deleted_by = $2
    WHERE %s = $1 AND deleted_at IS NULL
`

// Verify user exists before deletion
const verifyUserExists = `
    SELECT id, email, deleted_at FROM user_profiles
    WHERE id = $1 AND deleted_at IS NULL
`

// Insert deletion request
const insertDeletionRequest = `
    INSERT INTO deletion_requests (user_id, status, confirmation_token, requested_at)
    VALUES ($1, 'pending', $2, NOW())
    RETURNING id, requested_at
`

// Update deletion request status
const updateDeletionStatus = `
    UPDATE deletion_requests
    SET status = $2, confirmed_at = NOW()
    WHERE id = $1 AND status = 'pending'
`
```

### 4.4 Redis Operations

```go
// Cache key patterns
const (
    cacheKeyUserSession     = "session:user:%s"
    cacheKeyUserTokens      = "token:%s:*"
    cacheKeyUserData        = "cache:user:%s:*"
    cacheKeyDeletionLock    = "lock:deletion:%s"
)

// Redis command patterns
const (
    redisDel     = "DEL %s"
    redisPattern = "KEYS %s"
    redisSetex   = "SETEX %s %d %s"
)
```

### 4.5 Background Job Queue Messages

```go
type DeletionJob struct {
    RequestID     string    `json:"request_id"`
    UserID        string    `json:"user_id"`
    RequestedAt   time.Time `json:"requested_at"`
    RetryCount    int       `json:"retry_count"`
    IdempotencyKey string   `json:"idempotency_key"`
}
```

---

## 5. GDPR Compliance Notes

### 5.1 Data Categories Handled

| Category | Examples | Deletion Method |
|----------|----------|-----------------|
| Identity | Name, email, avatar | Permanent delete |
| Authentication | Password hash, sessions | Permanent delete + cache flush |
| Preferences | Unit preferences, dietary rules | Permanent delete |
| Activity | Search history, favorites | Permanent delete |
| Generated Content | Meal plans, shopping lists | Permanent delete |
| Export History | Past export requests | Permanent delete |
| Communications | Notifications, emails | Permanent delete |

### 5.2 Audit Trail Requirements

- Log all deletion requests with timestamps
- Record which admin triggered admin deletions
- Store deletion results for 90 days for compliance
- Maintain checksum of deleted data for verification

# DataRetentionPolicy

**Traceability:** ARCH-015

## 1. Data Structures & Types

```go
package compliance

import (
	"time"
)

type RetentionPolicy struct {
	ID              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	RetentionDays   int       `json:"retention_days" db:"retention_days"`
	BackupType      string    `json:"backup_type" db:"backup_type"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type BackupRecord struct {
	ID            string    `json:"id" db:"id"`
	BackupType    string    `json:"backup_type" db:"backup_type"`
	StoragePath   string    `json:"storage_path" db:"storage_path"`
	SizeBytes     int64     `json:"size_bytes" db:"size_bytes"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	ExpiresAt     time.Time `json:"expires_at" db:"expires_at"`
	Status        string    `json:"status" db:"status"`
	Checksum      string    `json:"checksum" db:"checksum"`
}

type PointInTimeRecoveryRequest struct {
	TargetTimestamp time.Time `json:"target_timestamp"`
	TargetUserID    string    `json:"target_user_id,omitempty"`
	RestoreTables   []string  `json:"restore_tables"`
}

type PointInTimeRecoveryStatus struct {
	RequestID        string    `json:"request_id"`
	Status           string    `json:"status"`
	Progress         int       `json:"progress"`
	StartedAt        time.Time `json:"started_at"`
	EstimatedEndTime time.Time `json:"estimated_end_time"`
	ErrorMessage     string    `json:"error_message,omitempty"`
}

type RetentionStats struct {
	TotalBackups      int            `json:"total_backups"`
	ExpiredBackups    int            `json:"expired_backups"`
	PendingDeletion   int            `json:"pending_deletion"`
	StorageUsedBytes  int64          `json:"storage_used_bytes"`
	OldestBackup      time.Time      `json:"oldest_backup"`
	NewestBackup      time.Time      `json:"newest_backup"`
}

type DataRetentionError struct {
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	RequestID   string    `json:"request_id"`
	Recoverable bool      `json:"recoverable"`
}

const (
	BackupStatusPending    = "pending"
	BackupStatusInProgress = "in_progress"
	BackupStatusCompleted  = "completed"
	BackupStatusFailed     = "failed"
	BackupStatusExpired    = "expired"
	BackupStatusDeleted    = "deleted"

	BackupTypeFull    = "full"
	BackupTypeIncremental = "incremental"
	BackupTypePointInTime = "point_in_time"

	RecoveryStatusPending    = "pending"
	RecoveryStatusProcessing = "processing"
	RecoveryStatusCompleted  = "completed"
	RecoveryStatusFailed     = "failed"

	RetentionPolicyName = "30-day-backup-retention"
	DefaultRetentionDays = 30
)

type BackupRepository interface {
	CreateBackupRecord(record *BackupRecord) error
	UpdateBackupStatus(id string, status string) error
	GetExpiredBackups() ([]BackupRecord, error)
	DeleteBackupRecord(id string) error
	GetBackupRecordsByDateRange(start, end time.Time) ([]BackupRecord, error)
	GetLatestBackupBefore(timestamp time.Time, backupType string) (*BackupRecord, error)
}

type StorageProvider interface {
	StoreBackup(path string, data []byte) error
	DeleteBackup(path string) error
	BackupExists(path string) (bool, error)
	GetBackupSize(path string) (int64, error)
	CopyBackup(srcPath, destPath string) error
	ListBackups(prefix string) ([]string, error)
}

type JobQueue interface {
	Enqueue(jobName string, payload map[string]interface{}, delay time.Duration) error
	GetJobStatus(jobID string) (string, error)
	ScheduleRecurringJob(spec string, jobName string, payload map[string]interface{}) error
}

type DataRetentionPolicyService struct {
	backupRepo    BackupRepository
	storage       StorageProvider
	jobQueue      JobQueue
	retentionDays int
	logger        *log.Logger
}

func NewDataRetentionPolicyService(
	backupRepo BackupRepository,
	storage StorageProvider,
	jobQueue JobQueue,
	retentionDays int,
	logger *log.Logger,
) *DataRetentionPolicyService {
	if retentionDays <= 0 {
		retentionDays = DefaultRetentionDays
	}
	return &DataRetentionPolicyService{
		backupRepo:    backupRepo,
		storage:       storage,
		jobQueue:      jobQueue,
		retentionDays: retentionDays,
		logger:        logger,
	}
}

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Backup Creation Workflow

```
FUNCTION CreateBackup(backupType string, data map[string]interface{}) -> (BackupRecord, error):
    1. Generate unique backup ID using UUID v4
    2. Generate storage path: /backups/{backupType}/{YYYY-MM-DD}/{backupID}.sql
    3. CREATE BackupRecord with:
       - ID: generated backup ID
       - BackupType: backupType parameter
       - StoragePath: generated path
       - Status: BackupStatusPending
       - CreatedAt: current UTC timestamp
       - ExpiresAt: current UTC timestamp + retentionDays
    4. CALL backupRepo.CreateBackupRecord(record)
    5. IF error:
       RETURN error with Code: "BACKUP_RECORD_CREATION_FAILED"
    6. ENQUEUE background job "execute-backup" with payload:
       - backup_id: record.ID
       - storage_path: record.StoragePath
       - backup_type: backupType
    7. RETURN record, nil
```

### 2.2 Scheduled Backup Cleanup Workflow

```
FUNCTION RunScheduledCleanup() -> error:
    1. CALL backupRepo.GetExpiredBackups()
    2. IF error:
       RETURN error with Code: "EXPIRED_BACKUPS_QUERY_FAILED"
    3. FOR EACH backup IN expiredBackups:
       a. IF backup.Status == BackupStatusInProgress:
          CONTINUE to next backup
       b. ENQUEUE background job "delete-expired-backup" with payload:
          - backup_id: backup.ID
          - storage_path: backup.StoragePath
    4. RETURN nil
```

### 2.3 Delete Expired Backup Job

```
FUNCTION DeleteExpiredBackupJob(payload map[string]interface{}) -> error:
    1. EXTRACT backup_id, storage_path FROM payload
    2. CALL storage.DeleteBackup(storage_path)
    3. IF error AND error is not "not found":
       LOG error with backup_id
       RETURN error with Code: "STORAGE_DELETE_FAILED"
    4. CALL backupRepo.UpdateBackupStatus(backup_id, BackupStatusDeleted)
    5. IF error:
       LOG error but continue processing
    6. CALL backupRepo.DeleteBackupRecord(backup_id)
    7. IF error:
       LOG error with Code: "RECORD_DELETE_FAILED"
    8. RETURN nil
```

### 2.4 Point-in-Time Recovery Workflow

```
FUNCTION InitiatePointInTimeRecovery(request PointInTimeRecoveryRequest) -> (PointInTimeRecoveryStatus, error):
    1. VALIDATE request:
       a. IF request.TargetTimestamp is in future:
          RETURN error with Code: "INVALID_RECOVERY_TIMESTAMP"
       b. IF request.TargetUserID is empty AND request.RestoreTables is empty:
          RETURN error with Code: "INVALID_RECOVERY_REQUEST"
    2. IDENTIFY backup for recovery:
       a. CALL backupRepo.GetLatestBackupBefore(request.TargetTimestamp, BackupTypeFull)
       b. IF no full backup found:
          CALL backupRepo.GetLatestBackupBefore(request.TargetTimestamp, BackupTypeIncremental)
       c. IF no incremental backup found:
          RETURN error with Code: "NO_BACKUP_FOUND"
    3. CREATE recovery status record:
       - RequestID: generate UUID
       - Status: RecoveryStatusPending
       - Progress: 0
       - StartedAt: current timestamp
    4. ENQUEUE background job "execute-point-in-time-recovery" with payload:
       - request_id: status.RequestID
       - target_timestamp: request.TargetTimestamp
       - backup_id: identifiedBackup.ID
       - target_user_id: request.TargetUserID
       - restore_tables: request.RestoreTables
    5. RETURN status, nil
```

### 2.5 Execute Point-in-Time Recovery Job

```
FUNCTION ExecutePointInTimeRecoveryJob(payload map[string]interface{}) -> error:
    1. EXTRACT all fields FROM payload
    2. UPDATE recovery status to RecoveryStatusProcessing
    3. DETERMINE recovery strategy:
       a. IF target_user_id is specified:
          - RESTORE data for specific user only
          - Use table-specific restore with user_id filter
       b. ELSE:
          - RESTORE full database state
    4. EXECUTE recovery sequence:
       a. FETCH base backup from storage using backup_id
       b. IF recovery to timestamp after backup creation:
          - APPLY subsequent incremental backups
          - STOP at target_timestamp
       c. IF restore_tables specified:
          - FILTER restore to only those tables
    5. VALIDATE restored data integrity:
       - CHECK critical tables have expected record counts
       - VERIFY foreign key relationships
    6. IF validation fails:
       UPDATE status with error
       RETURN error with Code: "RECOVERY_VALIDATION_FAILED"
    7. UPDATE recovery status to RecoveryStatusCompleted
    8. UPDATE progress to 100
    9. RETURN nil
```

### 2.6 Retention Policy Enforcement Algorithm

```
FUNCTION EnforceRetentionPolicy() -> (RetentionStats, error):
    1. DEFINE cutoffDate = current UTC timestamp - retentionDays
    2. QUERY for backups with CreatedAt < cutoffDate AND Status != BackupStatusDeleted
    3. INITIALIZE stats with zero values
    4. FOR EACH backup IN results:
       a. INCREMENT stats.TotalBackups
       b. IF backup.Status == BackupStatusExpired:
          INCREMENT stats.ExpiredBackups
       c. IF backup.Status == BackupStatusPending:
          INCREMENT stats.PendingDeletion
       d. ADD backup.SizeBytes to stats.StorageUsedBytes
       e. UPDATE stats.OldestBackup if backup.CreatedAt < stats.OldestBackup
       f. UPDATE stats.NewestBackup if backup.CreatedAt > stats.NewestBackup
    5. FOR EACH backup WHERE CreatedAt < cutoffDate:
       a. CALL storage.DeleteBackup(backup.StoragePath)
       b. CALL backupRepo.UpdateBackupStatus(backup.ID, BackupStatusDeleted)
    6. RETURN stats, nil
```

### 2.7 Backup Verification Algorithm

```
FUNCTION VerifyBackupIntegrity(backupID string) -> (bool, error):
    1. FETCH backup record from backupRepo by backupID
    2. IF not found:
       RETURN false, error with Code: "BACKUP_NOT_FOUND"
    3. CALCULATE actual checksum of storage file:
       READ backup file from storage.StoragePath
       COMPUTE SHA-256 checksum of file contents
    4. COMPARE computed checksum with backup.Checksum
    5. IF mismatch:
       UPDATE backup status to BackupStatusFailed
       RETURN false, nil
    6. UPDATE backup status to BackupStatusCompleted
    7. RETURN true, nil
```

## 3. State Management & Error Handling

### 3.1 Backup States

| State | Description | Transitions To |
|-------|-------------|----------------|
| `pending` | Backup job created, waiting to start | `in_progress`, `failed` |
| `in_progress` | Backup execution in progress | `completed`, `failed` |
| `completed` | Backup successfully created and verified | `expired` (after retention period) |
| `failed` | Backup creation or verification failed | None (manual intervention required) |
| `expired` | Backup past retention period, queued for deletion | `deleted` |
| `deleted` | Backup removed from storage and records | None (terminal state) |

### 3.2 Recovery States

| State | Description | Transitions To |
|-------|-------------|----------------|
| `pending` | Recovery request received, job queued | `processing` |
| `processing` | Recovery execution in progress | `completed`, `failed` |
| `completed` | Recovery successfully finished | None (terminal state) |
| `failed` | Recovery failed due to error | None (terminal state, requires new request) |

### 3.3 Error States and Handling

| Error Code | HTTP Status | Description | Handling Strategy |
|------------|-------------|-------------|-------------------|
| `BACKUP_RECORD_CREATION_FAILED` | 500 | Failed to create backup record | Retry once, alert ops team |
| `EXPIRED_BACKUPS_QUERY_FAILED` | 500 | Database error querying expired backups | Retry with exponential backoff |
| `STORAGE_DELETE_FAILED` | 500 | Storage deletion failed | Retry 3 times, mark for manual cleanup |
| `RECORD_DELETE_FAILED` | 500 | Database record deletion failed | Retry, alert ops team |
| `INVALID_RECOVERY_TIMESTAMP` | 400 | Timestamp is in the future | Return client error, no retry |
| `INVALID_RECOVERY_REQUEST` | 400 | Missing required recovery parameters | Return client error, no retry |
| `NO_BACKUP_FOUND` | 404 | No backup exists before requested timestamp | Return client error, suggest earlier timestamp |
| `BACKUP_NOT_FOUND` | 404 | Specified backup ID not found | Return client error, no retry |
| `RECOVERY_VALIDATION_FAILED` | 500 | Restored data failed integrity checks | Alert ops team, preserve recovery state |
| `STORAGE_READ_FAILED` | 500 | Unable to read backup from storage | Retry, check storage connectivity |
| `DATABASE_RESTORE_FAILED` | 500 | Database restore operation failed | Restore from previous backup, alert ops team |

### 3.4 Retry Policy

```
RETRY CONFIGURATION:
  - MaxRetries: 3
  - InitialDelay: 1 second
  - MaxDelay: 30 seconds
  - Multiplier: 2x exponential backoff

RETRYABLE ERROR CODES:
  - BACKUP_RECORD_CREATION_FAILED
  - EXPIRED_BACKUPS_QUERY_FAILED
  - STORAGE_DELETE_FAILED
  - RECORD_DELETE_FAILED
  - STORAGE_READ_FAILED
  - DATABASE_RESTORE_FAILED

NON-RETRYABLE ERROR CODES:
  - INVALID_RECOVERY_TIMESTAMP
  - INVALID_RECOVERY_REQUEST
  - NO_BACKUP_FOUND
  - BACKUP_NOT_FOUND
```

### 3.5 Alerting Rules

```
CRITICAL ALERTS (Paging required):
  - 3 consecutive backup failures
  - Recovery validation failure
  - Storage delete failures exceed 10% of attempts
  - Retention policy enforcement fails

WARNING ALERTS (No paging):
  - Single backup failure (auto-retry scheduled)
  - Backup verification checksum mismatch
  - Storage usage exceeds 80% of allocated quota
```

## 4. Component Interfaces

### 4.1 Public API Functions

```go
package compliance

type DataRetentionPolicyService interface {
	CreateBackup(backupType string, data map[string]interface{}) (*BackupRecord, error)
	RunScheduledCleanup() error
	InitiatePointInTimeRecovery(request PointInTimeRecoveryRequest) (*PointInTimeRecoveryStatus, error)
	GetRecoveryStatus(requestID string) (*PointInTimeRecoveryStatus, error)
	EnforceRetentionPolicy() (*RetentionStats, error)
	VerifyBackupIntegrity(backupID string) (bool, error)
	GetRetentionStats() (*RetentionStats, error)
	GetBackupHistory(limit int, offset int) ([]BackupRecord, error)
}
```

### 4.2 Function Signatures

```go
// CreateBackup initiates a new backup creation process
func (s *DataRetentionPolicyService) CreateBackup(backupType string, data map[string]interface{}) (*BackupRecord, error)

// RunScheduledCleanup executes the scheduled cleanup of expired backups
// Called by cron job or scheduler every 24 hours
func (s *DataRetentionPolicyService) RunScheduledCleanup() error

// InitiatePointInTimeRecovery starts a point-in-time recovery process
// request: Contains target timestamp and recovery scope
// Returns: Recovery status with request ID for tracking
func (s *DataRetentionPolicyService) InitiatePointInTimeRecovery(request PointInTimeRecoveryRequest) (*PointInTimeRecoveryStatus, error)

// GetRecoveryStatus retrieves the current status of a recovery request
// requestID: The ID returned from InitiatePointInTimeRecovery
func (s *DataRetentionPolicyService) GetRecoveryStatus(requestID string) (*PointInTimeRecoveryStatus, error)

// EnforceRetentionPolicy manually triggers retention policy enforcement
// Use for on-demand cleanup or scheduled maintenance window
func (s *DataRetentionPolicyService) EnforceRetentionPolicy() (*RetentionStats, error)

// VerifyBackupIntegrity validates checksum of a backup file
// backupID: ID of backup to verify
// Returns: true if integrity check passes, false if checksum mismatch
func (s *DataRetentionPolicyService) VerifyBackupIntegrity(backupID string) (bool, error)

// GetRetentionStats returns statistics about current backup state
// Use for monitoring dashboards and alerting thresholds
func (s *DataRetentionPolicyService) GetRetentionStats() (*RetentionStats, error)

// GetBackupHistory retrieves paginated list of backup records
// limit: Maximum number of records to return (default 50, max 100)
// offset: Number of records to skip for pagination
func (s *DataRetentionPolicyService) GetBackupHistory(limit int, offset int) ([]BackupRecord, error)
```

### 4.3 Background Job Handlers

```go
package compliance

type BackupJobHandler struct {
	service *DataRetentionPolicyService
}

func NewBackupJobHandler(service *DataRetentionPolicyService) *BackupJobHandler {
	return &BackupJobHandler{service: service}
}

// ExecuteBackupJob handles the "execute-backup" job from the queue
func (h *BackupJobHandler) ExecuteBackupJob(ctx context.Context, payload map[string]interface{}) error

// DeleteExpiredBackupJob handles the "delete-expired-backup" job from the queue
func (h *BackupJobHandler) DeleteExpiredBackupJob(ctx context.Context, payload map[string]interface{}) error

// ExecutePointInTimeRecoveryJob handles the "execute-point-in-time-recovery" job
func (h *BackupJobHandler) ExecutePointInTimeRecoveryJob(ctx context.Context, payload map[string]interface{}) error

// VerifyAllBackupsJob periodically verifies integrity of all completed backups
func (h *BackupJobHandler) VerifyAllBackupsJob(ctx context.Context, payload map[string]interface{}) error
```

### 4.4 HTTP Handlers (Fiber Routes)

```go
package compliance

type DataRetentionPolicyHandler struct {
	service DataRetentionPolicyService
}

func NewDataRetentionPolicyHandler(service DataRetentionPolicyService) *DataRetentionPolicyHandler {
	return &DataRetentionPolicyHandler{service: service}
}

func (h *DataRetentionPolicyHandler) RegisterRoutes(app *fiber.App) {
	backups := app.Group("/api/v1/backups")
	backups.Post("/", h.CreateBackup)
	backups.Get("/", h.ListBackups)
	backups.Get("/:id", h.GetBackup)
	backups.Get("/:id/verify", h.VerifyBackup)
	backups.Delete("/:id", h.DeleteBackup)

	recovery := app.Group("/api/v1/recovery")
	recovery.Post("/point-in-time", h.InitiatePointInTimeRecovery)
	recovery.Get("/status/:requestId", h.GetRecoveryStatus)

	policy := app.Group("/api/v1/policy")
	policy.Get("/retention", h.GetRetentionStats)
	policy.Post("/enforce", h.EnforceRetentionPolicy)
	policy.Get("/cleanup/schedule", h.GetCleanupSchedule)
	policy.Put("/cleanup/schedule", h.UpdateCleanupSchedule)
}

// CreateBackup handles POST /api/v1/backups
func (h *DataRetentionPolicyHandler) CreateBackup(c *fiber.Ctx) error

// ListBackups handles GET /api/v1/backups
func (h *DataRetentionPolicyHandler) ListBackups(c *fiber.Ctx) error

// GetBackup handles GET /api/v1/backups/:id
func (h *DataRetentionPolicyHandler) GetBackup(c *fiber.Ctx) error

// VerifyBackup handles GET /api/v1/backups/:id/verify
func (h *DataRetentionPolicyHandler) VerifyBackup(c *fiber.Ctx) error

// DeleteBackup handles DELETE /api/v1/backups/:id
func (h *DataRetentionPolicyHandler) DeleteBackup(c *fiber.Ctx) error

// InitiatePointInTimeRecovery handles POST /api/v1/recovery/point-in-time
func (h *DataRetentionPolicyHandler) InitiatePointInTimeRecovery(c *fiber.Ctx) error

// GetRecoveryStatus handles GET /api/v1/recovery/status/:requestId
func (h *DataRetentionPolicyHandler) GetRecoveryStatus(c *fiber.Ctx) error

// GetRetentionStats handles GET /api/v1/policy/retention
func (h *DataRetentionPolicyHandler) GetRetentionStats(c *fiber.Ctx) error

// EnforceRetentionPolicy handles POST /api/v1/policy/enforce
func (h *DataRetentionPolicyHandler) EnforceRetentionPolicy(c *fiber.Ctx) error

// GetCleanupSchedule handles GET /api/v1/policy/cleanup/schedule
func (h *DataRetentionPolicyHandler) GetCleanupSchedule(c *fiber.Ctx) error

// UpdateCleanupSchedule handles PUT /api/v1/policy/cleanup/schedule
func (h *DataRetentionPolicyHandler) UpdateCleanupSchedule(c *fiber.Ctx) error
```

### 4.5 Configuration

```go
package compliance

type DataRetentionConfig struct {
	// RetentionDays defines how long backups are kept (default: 30)
	RetentionDays int `env:"DATA_RETENTION_DAYS" envDefault:"30"`

	// CleanupSchedule cron expression (default: "0 2 * * *" - 2 AM daily)
	CleanupSchedule string `env:"CLEANUP_SCHEDULE" envDefault:"0 2 * * *"`

	// BackupSchedule cron expression for full backups (default: "0 3 * * 0" - 3 AM Sunday)
	BackupSchedule string `env:"BACKUP_SCHEDULE" envDefault:"0 3 * * 0"`

	// IncrementalBackupSchedule cron expression (default: "0 3 * * *" - 3 AM daily)
	IncrementalBackupSchedule string `env:"INCREMENTAL_BACKUP_SCHEDULE" envDefault:"0 3 * * *"`

	// StoragePath base path for backup storage
	StoragePath string `env:"BACKUP_STORAGE_PATH" envDefault:"/backups"`

	// VerificationSchedule cron expression (default: "0 4 * * 1" - 4 AM Monday)
	VerificationSchedule string `env:"VERIFICATION_SCHEDULE" envDefault:"0 4 * * 1"`

	// MaxConcurrentBackups maximum parallel backup operations
	MaxConcurrentBackups int `env:"MAX_CONCURRENT_BACKUPS" envDefault:"2"`

	// PointInTimeRecoveryEnabled enables point-in-time recovery feature
	PointInTimeRecoveryEnabled bool `env:"PITR_ENABLED" envDefault:"true"`
}
```

### 4.6 Database Schema (PostgreSQL)

```sql
-- Backup records table
CREATE TABLE compliance.backup_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_type VARCHAR(50) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    checksum VARCHAR(128),
    error_message TEXT,
    metadata JSONB,
    created_by UUID REFERENCES auth.users(id)
);

CREATE INDEX idx_backup_records_created_at ON compliance.backup_records(created_at);
CREATE INDEX idx_backup_records_status ON compliance.backup_records(status);
CREATE INDEX idx_backup_records_expires_at ON compliance.backup_records(expires_at);
CREATE INDEX idx_backup_records_backup_type ON compliance.backup_records(backup_type);

-- Recovery requests table
CREATE TABLE compliance.recovery_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_type VARCHAR(50) NOT NULL,
    target_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    target_user_id UUID REFERENCES auth.users(id),
    restore_tables TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    requested_by UUID REFERENCES auth.users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovery_requests_status ON compliance.recovery_requests(status);
CREATE INDEX idx_recovery_requests_created_at ON compliance.recovery_requests(created_at);

-- Retention policy configuration table
CREATE TABLE compliance.retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    retention_days INTEGER NOT NULL,
    backup_types TEXT[] NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Insert default retention policy
INSERT INTO compliance.retention_policies (name, description, retention_days, backup_types)
VALUES (
    '30-day-backretention',
    'Default backup retention policy enforcing 30-day retention for all backup types',
    30,
    ARRAY['full', 'incremental', 'point_in_time']
);
```

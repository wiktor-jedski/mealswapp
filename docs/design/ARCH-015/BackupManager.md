# BackupManager

**Traceability:** ARCH-015

## 1. Data Structures & Types

### 1.1 BackupRecord
```go
type BackupRecord struct {
    ID            string    `json:"id" db:"id"`
    BackupType    string    `json:"backup_type" db:"backup_type"` // "full", "incremental"
    Status        string    `json:"status" db:"status"` // "pending", "in_progress", "completed", "failed"
    StoragePath   string    `json:"storage_path" db:"storage_path"`
    SizeBytes     int64     `json:"size_bytes" db:"size_bytes"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
    ExpiresAt     time.Time `json:"expires_at" db:"expires_at"`
    Checksum      string    `json:"checksum" db:"checksum"`
    Metadata      string    `json:"metadata" db:"metadata"` // JSON string
}
```

### 1.2 BackupConfig
```go
type BackupConfig struct {
    RetentionDays        int           `json:"retention_days"`
    FullBackupInterval   time.Duration `json:"full_backup_interval"` // e.g., 24 * time.Hour
    IncrementalInterval  time.Duration `json:"incremental_interval"` // e.g., 1 * time.Hour
    StorageBucket        string        `json:"storage_bucket"`
    EncryptionKeyID      string        `json:"encryption_key_id"`
}
```

### 1.3 PointInTimeRecoveryRequest
```go
type PointInTimeRecoveryRequest struct {
    TargetTime   time.Time `json:"target_time"`
    UserID       string    `json:"user_id"`
    RestoreTables []string  `json:"restore_tables"`
}
```

### 1.4 PointInTimeRecoveryResult
```go
type PointInTimeRecoveryResult struct {
    Success       bool      `json:"success"`
    RestoredCount int       `json:"restored_count"`
    BackupID      string    `json:"backup_id"`
    Message       string    `json:"message"`
}
```

### 1.5 BackupStatus
```go
type BackupStatus struct {
    LastBackupTime   time.Time  `json:"last_backup_time"`
    NextBackupTime   time.Time  `json:"next_backup_time"`
    ActiveBackups    int        `json:"active_backups"`
    StorageUsedBytes int64      `json:"storage_used_bytes"`
    HealthStatus     string     `json:"health_status"` // "healthy", "warning", "critical"
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Backup Creation Flow

**Step 1: Check schedule eligibility**
```
IF current_time >= next_scheduled_backup_time:
    Proceed to Step 2
ELSE:
    Exit - backup not yet due
```

**Step 2: Determine backup type**
```
IF current_time - last_full_backup_time >= full_backup_interval:
    backup_type = "full"
ELSE:
    backup_type = "incremental"
```

**Step 3: Lock backup mutex**
```
Acquire mutex with timeout of 5 seconds
IF acquisition fails:
    Log warning and exit (backup already in progress)
```

**Step 4: Create database backup**
```
1. Execute pg_dump with:
   - --format=custom
   - --compress=9
   - --blobs
   - --verbose
   
2. Stream output to temporary file
3. Calculate SHA-256 checksum of file
4. Encrypt file using AES-256-GCM with key from GCP Secret Manager
5. Upload encrypted file to GCP Cloud Storage
```

**Step 5: Record backup metadata**
```
1. Insert BackupRecord into PostgreSQL backup_records table
2. Set status = "completed"
3. Set expires_at = created_at + retention_days
```

**Step 6: Update scheduling**
```
1. Update next_scheduled_backup_time
2. If full backup: reset incremental counter
3. Release mutex
```

### 2.2 Backup Retention Enforcement Flow

**Step 1: Query expired backups**
```
SELECT id, storage_path FROM backup_records
WHERE expires_at < current_timestamp
AND status = 'completed'
```

**Step 2: Delete expired backups from storage**
```
FOR each expired_backup IN expired_backups:
    1. Delete object from GCP Cloud Storage bucket
    2. If deletion succeeds:
        Update BackupRecord set status = 'purged'
    3. If deletion fails:
        Log error
        Update BackupRecord set status = 'purge_failed'
```

**Step 3: Clean up database records**
```
DELETE FROM backup_records WHERE status = 'purged'
AND created_at < current_timestamp - 90 days
```

### 2.3 Point-in-Time Recovery Flow

**Step 1: Validate request parameters**
```
IF target_time > current_timestamp:
    RETURN error: "Cannot restore to future time"
    
IF target_time < earliest_backup_time:
    RETURN error: "Target time before earliest backup"
```

**Step 2: Identify relevant backups**
```
1. Find full backup with created_at <= target_time (most recent)
2. Find all incremental backups where:
   - created_at >= full_backup_time
   - created_at <= target_time
3. Order incremental backups chronologically
```

**Step 3: Create recovery environment**
```
1. Create temporary database with name: "restore_<user_id>_<timestamp>"
2. Restore full backup to temporary database
3. Apply each incremental backup in sequence to temporary database
```

**Step 4: Extract user data**
```
1. Connect to temporary database
2. For each table in restore_tables:
    Export data where user_id = target_user_id
3. Import data to primary database
4. Log recovery operation with user_id, target_time, restored_count
```

**Step 5: Cleanup**
```
1. Drop temporary database
2. Delete temporary backup files
```

### 2.4 Erasure Coordination Flow (GDPR)

**Step 1: Receive erasure request**
```
1. Parse request containing user_id
2. Create erasure_work record
3. Set status = "pending"
```

**Step 2: Schedule backup purge**
```
1. Create purge_schedule record with:
   - user_id = target_user_id
   - scheduled_time = current_timestamp + 24 hours
   - status = "scheduled"
2. This ensures backup data is retained for regulatory requirements
   but purged after mandatory period
```

**Step 3: Acknowledge request**
```
RETURN erasure_confirmation with scheduled_purge_date
```

## 3. State Management & Error Handling

### 3.1 Backup States
| State | Description | Transitions To |
|-------|-------------|----------------|
| `pending` | Backup job created, waiting to start | `in_progress`, `failed` |
| `in_progress` | Backup actively being created | `completed`, `failed` |
| `completed` | Backup successfully created and stored | `purged`, `purge_failed` |
| `failed` | Backup creation encountered error | `pending` (retry), `failed` (manual) |
| `purged` | Backup expired and deleted from storage | - |
| `purge_failed` | Backup expired but deletion from storage failed | `purged` (retry), `failed` (manual) |

### 3.2 Error States

**Mutex Acquisition Timeout**
- Error Code: `ERR_BACKUP_MUTEX_TIMEOUT`
- Message: "Unable to acquire backup mutex within timeout period"
- Action: Skip this backup cycle, retry at next scheduled time
- Logging: Warning level

**Database Backup Failure**
- Error Code: `ERR_BACKUP_DATABASE_FAILED`
- Message: "pg_dump execution failed: [details]"
- Action: Set status = "failed", increment retry counter
- Logging: Error level, trigger alert if retries exceed 3

**Storage Upload Failure**
- Error Code: `ERR_BACKUP_UPLOAD_FAILED`
- Message: "Failed to upload backup to Cloud Storage: [details]"
- Action: Set status = "failed", delete partial local file
- Logging: Error level, trigger alert if retries exceed 2

**Encryption Failure**
- Error Code: `ERR_BACKUP_ENCRYPTION_FAILED`
- Message: "AES-256-GCM encryption failed: [details]"
- Action: Delete unencrypted backup file, set status = "failed"
- Logging: Error level

**Storage Deletion Failure**
- Error Code: `ERR_BACKUP_PURGE_FAILED`
- Message: "Failed to delete backup from Cloud Storage: [details]"
- Action: Set status = "purge_failed", retry in 1 hour
- Logging: Warning level

**Point-in-Time Recovery Error**
- Error Code: `ERR_PITR_INVALID_TIME`
- Message: "Target time outside valid recovery window"
- Action: Return error to caller, no state change

**Point-in-Time Recovery Error**
- Error Code: `ERR_PITR_RESTORE_FAILED`
- Message: "Database restore operation failed: [details]"
- Action: Set status = "failed", cleanup temporary database
- Logging: Error level, trigger alert

**Erasure Coordination Error**
- Error Code: `ERR_ERASURE_SCHEDULE_FAILED`
- Message: "Failed to schedule backup purge: [details]"
- Action: Return error, erasure request remains pending
- Logging: Error level

### 3.3 Health Status Determination
```go
func (bm *BackupManager) calculateHealthStatus() string {
    lastBackupAge := time.Since(bm.lastBackupTime)
    failedBackups := bm.getFailedBackupCount(24 * time.Hour)
    
    switch {
    case lastBackupAge > 48*time.Hour:
        return "critical"
    case failedBackups >= 3:
        return "critical"
    case lastBackupAge > 36*time.Hour:
        return "warning"
    case failedBackups >= 2:
        return "warning"
    default:
        return "healthy"
    }
}
```

## 4. Component Interfaces

### 4.1 CreateBackup
```go
func (bm *BackupManager) CreateBackup(ctx context.Context) (*BackupRecord, error)
```

**Parameters:**
- `ctx` - Context with cancellation support

**Returns:**
- `*BackupRecord` - Created backup record with status and metadata
- `error` - Error if backup creation failed

**Behavior:**
- Checks if backup is due based on schedule
- Determines if full or incremental backup
- Acquires backup mutex
- Creates database dump
- Encrypts and uploads to Cloud Storage
- Records metadata in PostgreSQL
- Updates scheduling for next backup

### 4.2 EnforceRetention
```go
func (bm *BackupManager) EnforceRetention(ctx context.Context) (int, error)
```

**Parameters:**
- `ctx` - Context with cancellation support

**Returns:**
- `int` - Number of backups purged
- `error` - Error if retention enforcement failed

**Behavior:**
- Queries database for expired backups
- Deletes from Cloud Storage
- Updates backup records
- Cleans up old purge records

### 4.3 PointInTimeRecovery
```go
func (bm *BackupManager) PointInTimeRecovery(ctx context.Context, req *PointInTimeRecoveryRequest) (*PointInTimeRecoveryResult, error)
```

**Parameters:**
- `ctx` - Context with cancellation support
- `req` - Recovery request with target time, user ID, tables

**Returns:**
- `*PointInTimeRecoveryResult` - Result of recovery operation
- `error` - Error if recovery failed

**Behavior:**
- Validates target time is within recovery window
- Identifies relevant backups
- Creates temporary database
- Restores full backup and applies incrementals
- Exports user data from temporary database
- Imports to primary database
- Cleans up temporary resources

### 4.4 ScheduleErasurePurge
```go
func (bm *BackupManager) ScheduleErasurePurge(ctx context.Context, userID string) (time.Time, error)
```

**Parameters:**
- `ctx` - Context with cancellation support
- `userID` - User ID whose backups should be purged

**Returns:**
- `time.Time` - Scheduled purge time (24 hours from now)
- `error` - Error if scheduling failed

**Behavior:**
- Creates purge schedule record
- Sets purge time to 24 hours from now (GDPR compliance)
- Returns scheduled purge time for user notification

### 4.5 GetBackupStatus
```go
func (bm *BackupManager) GetBackupStatus(ctx context.Context) (*BackupStatus, error)
```

**Parameters:**
- `ctx` - Context with cancellation support

**Returns:**
- `*BackupStatus` - Current backup system status
- `error` - Error if status retrieval failed

**Behavior:**
- Calculates last and next backup times
- Counts active backups
- Calculates storage usage
- Determines health status

### 4.6 GetBackupRecords
```go
func (bm *BackupManager) GetBackupRecords(ctx context.Context, limit, offset int) ([]BackupRecord, error)
```

**Parameters:**
- `ctx` - Context with cancellation support
- `limit` - Maximum number of records to return
- `offset` - Number of records to skip

**Returns:**
- `[]BackupRecord` - List of backup records
- `error` - Error if retrieval failed

**Behavior:**
- Queries backup_records table with pagination
- Returns records ordered by created_at descending

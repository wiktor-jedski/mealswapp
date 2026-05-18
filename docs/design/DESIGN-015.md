## FILE: DESIGN-015.md
**Traceability:** ARCH-015

**Static aspects covered:** ConsentManager, DisclaimerRenderer, DataRetentionPolicy, BackupManager.

### 0. Static Aspect Responsibilities
- `ConsentManager`: owns Privacy Policy and ToS consent capture, versioning, and enforcement.
- `DisclaimerRenderer`: owns medical disclaimer content retrieval and display locations.
- `DataRetentionPolicy`: owns production erasure and 30-day backup retention rules.
- `BackupManager`: owns backup status, point-in-time recovery checks, and retention enforcement coordination.

### 1. Data Structures & Types
- `interface ConsentRecord { userId: UUID; privacyPolicyVersion: string; termsVersion: string; acceptedAt: time.Time; ip: string }`
- `interface DisclaimerContent { version: string; bodyMarkdown: string; effectiveAt: time.Time; locations: string[] }`
- `interface RetentionPolicy { backupRetentionDays: 30; productionDeletionMode: "immediate"; backupPurgeMode: "scheduled" }`
- `interface ErasureRequest { userId: UUID; requestedAt: time.Time; status: "pending" | "processing" | "completed" | "failed"; completedAt?: time.Time }`
- `interface BackupStatus { backupId: string; createdAt: time.Time; expiresAt: time.Time; pointInTimeRecovery: boolean }`

### 2. Logic & Algorithms (Step-by-Step)
1. Registration cannot complete until Privacy Policy and ToS checkboxes are explicitly accepted.
2. Persist consent version, timestamp, IP, and user ID through ARCH-005.
3. Serve medical disclaimer content to the login screen and About section through a stable content API.
4. On account erasure request, call ARCH-008 deletion workflow for production data.
5. Record an erasure request and status transitions for auditability without retaining unnecessary PII.
6. Enforce 30-day backup retention by marking backups for expiration and verifying purge completion.
7. Coordinate point-in-time recovery configuration with database backup status monitoring in ARCH-014.
8. Fail closed for missing consent on protected registration or account creation paths.

### 3. State Management & Error Handling
- `consent_missing`: block registration completion.
- `consent_recorded`: registration can proceed.
- `disclaimer_unavailable`: render bundled fallback content and alert maintainers.
- `erasure_pending`: request accepted but deletion not started.
- `erasure_processing`: production deletion is underway.
- `erasure_completed`: production data deleted and backup purge scheduled according to retention.
- `backup_retention_breach`: critical monitoring event.
- `deletion_failed`: account remains locked from normal use until retry or investigation.

### 4. Component Interfaces
- `func RecordConsent(ctx context.Context, record ConsentRecord) error`
- `func HasRequiredConsent(ctx context.Context, userID UUID, privacyVersion string, termsVersion string) (bool, error)`
- `func GetDisclaimer(ctx context.Context, location string) (DisclaimerContent, error)`
- `func RequestErasure(ctx context.Context, userID UUID) (ErasureRequest, error)`
- `func ProcessErasure(ctx context.Context, requestID UUID) error`
- `func EnforceBackupRetention(ctx context.Context, policy RetentionPolicy) error`
- `func GetBackupStatus(ctx context.Context) ([]BackupStatus, error)`

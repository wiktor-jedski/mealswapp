# TrialTracker

**Traceability:** ARCH-007

## 1. Data Structures & Types

```go
package subscription

import "time"

type TrialStatus string

const (
    TrialStatusActive   TrialStatus = "active"
    TrialStatusExpired  TrialStatus = "expired"
    TrialStatusConverted TrialStatus = "converted"
    TrialStatusNever    TrialStatus = "never_enrolled"
)

type Trial struct {
    ID              string      `json:"id" db:"id"`
    UserID          string      `json:"user_id" db:"user_id"`
    Status          TrialStatus `json:"status" db:"status"`
    StartDate       time.Time   `json:"start_date" db:"start_date"`
    EndDate         time.Time   `json:"end_date" db:"end_date"`
    ConvertedDate   *time.Time  `json:"converted_date,omitempty" db:"converted_date"`
    CreatedAt       time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

type TrialEntitlement struct {
    UserID          string    `json:"user_id"`
    HasActiveTrial  bool      `json:"has_active_trial"`
    TrialEndDate    time.Time `json:"trial_end_date,omitempty"`
    DaysRemaining   int       `json:"days_remaining,omitempty"`
    Tier            string    `json:"tier"`
    SearchesRemaining int     `json:"searches_remaining"`
    MaxItemsPerSearch int     `json:"max_items_per_search"`
}

type TrialConfig struct {
    DurationDays        int   `json:"duration_days"`
    DailySearchLimit    int   `json:"daily_search_limit"`
    MaxItemsPerSearch   int   `json:"max_items_per_search"`
    FeaturesEnabled     []string `json:"features_enabled"`
}

type TrialEligibilityCheck struct {
    UserID          string    `json:"user_id"`
    Eligible        bool      `json:"eligible"`
    Reason          string    `json:"reason,omitempty"`
    ExistingTrial   *Trial    `json:"existing_trial,omitempty"`
}

type CreateTrialRequest struct {
    UserID      string `json:"user_id" validate:"required,uuid"`
    Provider    string `json:"provider" validate:"required"`
}

type CreateTrialResponse struct {
    Success     bool      `json:"success"`
    Trial       *Trial    `json:"trial,omitempty"`
    Message     string    `json:"message"`
}
```

## 2. Logic & Algorithms

### 2.1 Initialize Trial on Social Login

```
FUNCTION InitializeTrialForSocialLogin(userID string, provider string) Result<Trial>
    eligibilityCheck = CheckTrialEligibility(userID)
    
    IF NOT eligibilityCheck.Eligible THEN
        RETURN Result<Trial>.Failure(eligibilityCheck.Reason)
    END IF
    
    trial = Trial{
        ID: GenerateUUID(),
        UserID: userID,
        Status: TrialStatusActive,
        StartDate: NOW_UTC(),
        EndDate: NOW_UTC().AddDays(TRIAL_DURATION_DAYS),
        CreatedAt: NOW_UTC(),
        UpdatedAt: NOW_UTC()
    }
    
    tx = BEGIN_TRANSACTION()
    
    TRY
        INSERT INTO trials (id, user_id, status, start_date, end_date, created_at, updated_at)
        VALUES (trial.ID, trial.UserID, trial.Status, trial.StartDate, trial.EndDate, trial.CreatedAt, trial.UpdatedAt)
        
        UPDATE user_entitlements
        SET tier = 'trial', trial_end_date = trial.EndDate, updated_at = NOW_UTC()
        WHERE user_id = userID
        
        COMMIT_TRANSACTION(tx)
        
        PUBLISH_EVENT("trial.activated", {userID: userID, endDate: trial.EndDate})
        
        RETURN Result<Trial>.Success(trial)
        
    CATCH error
        ROLLBACK_TRANSACTION(tx)
        RETURN Result<Trial>.Failure("Failed to initialize trial: " + error.Message)
    END TRY
END FUNCTION
```

### 2.2 Check Trial Eligibility

```
FUNCTION CheckTrialEligibility(userID string) TrialEligibilityCheck
    existingTrial = QUERY "SELECT * FROM trials WHERE user_id = ? ORDER BY created_at DESC LIMIT 1" (userID)
    
    IF existingTrial EXISTS THEN
        IF existingTrial.Status == TrialStatusActive THEN
            RETURN TrialEligibilityCheck{
                UserID: userID,
                Eligible: false,
                Reason: "Active trial already exists",
                ExistingTrial: existingTrial
            }
        END IF
        
        IF existingTrial.Status == TrialStatusExpired THEN
            daysSinceExpiry = DAYS_BETWEEN(existingTrial.EndDate, NOW_UTC())
            IF daysSinceExpiry < TRIAL_COOLDOWN_DAYS THEN
                RETURN TrialEligibilityCheck{
                    UserID: userID,
                    Eligible: false,
                    Reason: "Trial recently expired. Cooldown period active.",
                    ExistingTrial: existingTrial
                }
            END IF
        END IF
    END IF
    
    hasPaidSubscription = QUERY "SELECT COUNT(*) FROM subscriptions WHERE user_id = ? AND status = 'active'" (userID)
    IF hasPaidSubscription > 0 THEN
        RETURN TrialEligibilityCheck{
            UserID: userID,
            Eligible: false,
            Reason: "User already has a paid subscription"
        }
    END IF
    
    RETURN TrialEligibilityCheck{
        UserID: userID,
        Eligible: true,
        Reason: ""
    }
END FUNCTION
```

### 2.3 Check Trial Status and Entitlement

```
FUNCTION GetTrialEntitlement(userID string) TrialEntitlement
    trial = QUERY "SELECT * FROM trials WHERE user_id = ? AND status = 'active' ORDER BY created_at DESC LIMIT 1" (userID)
    
    IF trial NOT FOUND OR trial.Status != TrialStatusActive THEN
        RETURN TrialEntitlement{
            UserID: userID,
            HasActiveTrial: false,
            Tier: "free",
            SearchesRemaining: FREE_TIER_DAILY_LIMIT,
            MaxItemsPerSearch: FREE_TIER_MAX_ITEMS
        }
    END IF
    
    now = NOW_UTC()
    daysRemaining = CEIL(HOURS_BETWEEN(trial.EndDate, now) / 24)
    
    IF now.After(trial.EndDate) THEN
        ExpireTrial(trial.ID)
        RETURN GetTrialEntitlement(userID)
    END IF
    
    searchesRemaining = GetRemainingSearches(userID, now)
    
    RETURN TrialEntitlement{
        UserID: userID,
        HasActiveTrial: true,
        TrialEndDate: trial.EndDate,
        DaysRemaining: daysRemaining,
        Tier: "trial",
        SearchesRemaining: searchesRemaining,
        MaxItemsPerSearch: TRIAL_MAX_ITEMS
    }
END FUNCTION
```

### 2.4 Expire Trial and Downgrade to Free

```
FUNCTION ExpireTrial(trialID string) error
    trial = QUERY "SELECT * FROM trials WHERE id = ?" (trialID)
    
    IF trial.Status == TrialStatusExpired THEN
        RETURN nil
    END IF
    
    tx = BEGIN_TRANSACTION()
    
    TRY
        UPDATE trials
        SET status = ?, updated_at = ?
        WHERE id = ? (TrialStatusExpired, NOW_UTC(), trialID)
        
        UPDATE user_entitlements
        SET tier = 'free', trial_end_date = NULL, updated_at = NOW_UTC()
        WHERE user_id = trial.UserID
        
        DELETE FROM daily_search_usage WHERE user_id = trial.UserID
        
        COMMIT_TRANSACTION(tx)
        
        PUBLISH_EVENT("trial.expired", {userID: trial.UserID, trialID: trialID})
        
        RETURN nil
        
    CATCH error
        ROLLBACK_TRANSACTION(tx)
        RETURN error
    END TRY
END FUNCTION
```

### 2.5 Batch Expiration Job

```
FUNCTION RunTrialExpirationJob() JobResult
    expiredTrials = QUERY "
        SELECT id, user_id FROM trials
        WHERE status = 'active' AND end_date < NOW_UTC()
        LIMIT BATCH_SIZE
    " ()
    
    successCount = 0
    failureCount = 0
    
    FOR EACH trial IN expiredTrials DO
        error = ExpireTrial(trial.id)
        IF error THEN
            LOG_ERROR("Failed to expire trial", trial.id, error)
            failureCount++
        ELSE
            successCount++
        END IF
    END FOR
    
    RETURN JobResult{
        Processed: len(expiredTrials),
        Succeeded: successCount,
        Failed: failureCount
    }
END FUNCTION
```

### 2.6 Mark Trial as Converted

```
FUNCTION MarkTrialConverted(userID string, subscriptionID string) error
    trial = QUERY "SELECT * FROM trials WHERE user_id = ? AND status = 'active' ORDER BY created_at DESC LIMIT 1" (userID)
    
    IF trial NOT FOUND THEN
        RETURN nil
    END IF
    
    convertedDate = NOW_UTC()
    
    tx = BEGIN_TRANSACTION()
    
    TRY
        UPDATE trials
        SET status = ?, converted_date = ?, updated_at = ?
        WHERE id = ? (TrialStatusConverted, convertedDate, NOW_UTC(), trial.ID)
        
        UPDATE trial_conversions
        SET trial_id = ?, subscription_id = ?, converted_at = ?
        VALUES (?, ?, ?) ON CONFLICT DO NOTHING (trial.ID, subscriptionID, convertedDate, trial.ID, subscriptionID, convertedDate)
        
        COMMIT_TRANSACTION(tx)
        
        PUBLISH_EVENT("trial.converted", {userID: userID, trialID: trial.ID, subscriptionID: subscriptionID})
        
        RETURN nil
        
    CATCH error
        ROLLBACK_TRANSACTION(tx)
        RETURN error
    END TRY
END FUNCTION
```

### 2.7 Get Remaining Searches

```
FUNCTION GetRemainingSearches(userID string, date time.Time) int
    dailyLimit = GetDailySearchLimit(userID)
    
    usageToday = QUERY "
        SELECT COALESCE(SUM(search_count), 0) FROM daily_search_usage
        WHERE user_id = ? AND usage_date = DATE(date)
    " (userID)
    
    remaining = dailyLimit - usageToday
    IF remaining < 0 THEN
        remaining = 0
    END IF
    
    RETURN remaining
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Trial States

```
┌─────────────┐
│   NONE      │───► InitializeTrialForSocialLogin() ───►┐
│ (No trial)  │                                            │
└─────────────┘                                           │
                                                           │
                   ◄─────────────────────────────── ExpireTrial()
                         Auto-downgrade                  │
                                                           ▼
┌─────────────┐                                    ┌─────────────┐
│   ACTIVE    │───► Convert to paid ──────────────►│ CONVERTED   │
│  (7 days)   │     MarkTrialConverted()          │ (Paid cust.)│
└─────────────┘                                    └─────────────┘
      │
      │ ExpireTrial()
      │ (7 days elapses)
      ▼
┌─────────────┐
│  EXPIRED    │───► Cooldown period ───►┐
│ (Free tier) │                         │ CheckTrialEligibility()
└─────────────┘                         │ (if eligible)
                                        ▼
                                   ┌─────────────┐
                                   │   ACTIVE    │
                                   │ (7 days)    │
                                   └─────────────┘
```

### 3.2 Error States and Handling

| Error State | Description | Handling Strategy |
|-------------|-------------|-------------------|
| `E001: TrialAlreadyActive` | User already has active trial | Return existing trial info, do not create duplicate |
| `E002: TrialRecentlyExpired` | Trial expired within cooldown period | Block new trial, suggest paid subscription |
| `E003: TrialInitFailed` | Database error during trial creation | Rollback transaction, log error, return user-friendly message |
| `E004: TrialExpirationFailed` | Failed to expire trial on schedule | Retry with exponential backoff, alert operations |
| `E005: TrialConversionFailed` | Failed to mark trial as converted | Rollback transaction, prevent subscription creation |
| `E006: EntitlementQueryFailed` | Failed to fetch entitlement data | Return default free tier, log error for investigation |
| `E007: WebhookProcessingFailed` | Stripe webhook processing error | Return error to trigger retry, do not mark trial converted |

### 3.3 Retry Policy for Expiration Job

- Maximum 3 retry attempts
- Exponential backoff: 1s, 5s, 30s
- After final failure, move to dead-letter queue
- Reconciliation job checks for stuck expired trials hourly

### 3.4 Idempotency Guarantees

- `InitializeTrialForSocialLogin()`: Check `existing_trial` before insert
- `ExpireTrial()`: Early return if `status == TrialStatusExpired`
- `MarkTrialConverted()`: Early return if no active trial found
- All webhook handlers check `processed_events` table

## 4. Component Interfaces

### 4.1 Public Functions

```go
type TrialTracker interface {
    // Initialize a new trial for a user who just completed social login
    InitializeTrialForSocialLogin(ctx context.Context, req CreateTrialRequest) (*CreateTrialResponse, error)
    
    // Check if user is eligible for a trial
    CheckTrialEligibility(ctx context.Context, userID string) (*TrialEligibilityCheck, error)
    
    // Get current trial status and entitlements
    GetTrialEntitlement(ctx context.Context, userID string) (*TrialEntitlement, error)
    
    // Force expire a trial (for admin use)
    ExpireTrial(ctx context.Context, trialID string) error
    
    // Mark a trial as converted to paid subscription
    MarkTrialConverted(ctx context.Context, userID string, subscriptionID string) error
    
    // Get remaining searches for today
    GetRemainingSearches(ctx context.Context, userID string) (int, error)
    
    // Record a search usage
    RecordSearchUsage(ctx context.Context, userID string) error
    
    // Run batch job to expire all expired trials
    RunTrialExpirationJob(ctx context.Context) (*JobResult, error)
    
    // Get trial details by ID
    GetTrialByID(ctx context.Context, trialID string) (*Trial, error)
    
    // Get trial history for a user
    GetTrialHistory(ctx context.Context, userID string) ([]*Trial, error)
}
```

### 4.2 Database Operations

```go
type TrialRepository interface {
    Create(ctx context.Context, trial *Trial) error
    GetByID(ctx context.Context, trialID string) (*Trial, error)
    GetActiveByUserID(ctx context.Context, userID string) (*Trial, error)
    GetLatestByUserID(ctx context.Context, userID string) (*Trial, error)
    GetAllByUserID(ctx context.Context, userID string) ([]*Trial, error)
    UpdateStatus(ctx context.Context, trialID string, status TrialStatus) error
    UpdateConverted(ctx context.Context, trialID string, convertedDate time.Time) error
    ListExpired(ctx context.Context, limit int) ([]*Trial, error)
    Delete(ctx context.Context, trialID string) error
}
```

### 4.3 Event Publisher

```go
type TrialEventPublisher interface {
    PublishTrialActivated(ctx context.Context, userID string, endDate time.Time) error
    PublishTrialExpired(ctx context.Context, userID string, trialID string) error
    PublishTrialConverted(ctx context.Context, userID string, trialID string, subscriptionID string) error
}
```

### 4.4 Redis Keys

```
trial:{userID}:entitlement  TTL: 1 hour  Stores serialized TrialEntitlement
trial:{userID}:searches:{date}  TTL: 25 hours  Daily search counter
```

### 4.5 HTTP Handlers

```go
func RegisterTrialRoutes(app *fiber.App, tracker TrialTracker) {
    trials := app.Group("/api/v1/trials")
    
    trials.Get("/eligibility/:userID", func(c *fiber.Ctx) error {
        userID := c.Params("userID")
        check, err := tracker.CheckTrialEligibility(c.Context(), userID)
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
        }
        return c.JSON(check)
    })
    
    trials.Get("/status/:userID", func(c *fiber.Ctx) error {
        userID := c.Params("userID")
        entitlement, err := tracker.GetTrialEntitlement(c.Context(), userID)
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
        }
        return c.JSON(entitlement)
    })
    
    trials.Get("/history/:userID", func(c *fiber.Ctx) error {
        userID := c.Params("userID")
        history, err := tracker.GetTrialHistory(c.Context(), userID)
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
        }
        return c.JSON(history)
    })
}
```

### 4.6 Scheduled Jobs

```go
func StartTrialScheduler(tracker TrialTracker, queue *redis.Queue) {
    // Run expiration check every hour
    cron.Schedule("0 * * * *", func() {
        result, err := tracker.RunTrialExpirationJob(context.Background())
        if err != nil {
            log.Errorf("Trial expiration job failed: %v", err)
        } else {
            log.Infof("Trial expiration job completed: %d processed, %d succeeded, %d failed",
                result.Processed, result.Succeeded, result.Failed)
        }
    })
}
```

### 4.3 Constants

```go
const (
    TRIAL_DURATION_DAYS          = 7
    TRIAL_COOLDOWN_DAYS          = 30
    TRIAL_DAILY_SEARCH_LIMIT     = -1  // -1 means unlimited
    TRIAL_MAX_ITEMS_PER_SEARCH   = -1  // -1 means unlimited
    FREE_TIER_DAILY_LIMIT        = 3
    FREE_TIER_MAX_ITEMS          = 1
    TRIAL_EXPIRATION_BATCH_SIZE  = 100
    TRIAL_ENTITLEMENT_CACHE_TTL  = time.Hour
    DAILY_SEARCH_COUNTER_TTL     = 25 * time.Hour
)
```

# AccountLockoutTracker

**Traceability:** ARCH-006

## 1. Data Structures & Types

```go
package auth

import (
    "time"
)

const (
    MaxFailedAttemptsPerAccount = 5
    MaxFailedAttemptsPerIP      = 10
    AccountLockoutDuration      = 15 * time.Minute
    IPLockoutDuration           = 10 * time.Minute
    FailedAttemptWindow         = 15 * time.Minute
    IPFailedAttemptWindow       = 10 * time.Minute
)

type LockoutReason string

const (
    LockoutReasonAccountExceeded    LockoutReason = "account_exceeded"
    LockoutReasonIPExceeded         LockoutReason = "ip_exceeded"
    LockoutReasonManual             LockoutReason = "manual_admin"
)

type LockoutStatus string

const (
    LockoutStatusActive   LockoutStatus = "active"
    LockoutStatusReleased LockoutStatus = "released"
    LockoutStatusExpired  LockoutStatus = "expired"
)

type FailedAttempt struct {
    UserID    string    `json:"user_id"`
    IPAddress string    `json:"ip_address"`
    Timestamp time.Time `json:"timestamp"`
    Reason    string    `json:"reason"`
}

type AccountLockout struct {
    ID        string         `json:"id"`
    UserID    string         `json:"user_id"`
    IPAddress string         `json:"ip_address"`
    Reason    LockoutReason  `json:"reason"`
    Status    LockoutStatus  `json:"status"`
    CreatedAt time.Time      `json:"created_at"`
    ExpiresAt time.Time      `json:"expires_at"`
    ReleasedAt *time.Time    `json:"released_at,omitempty"`
}

type LockoutCheckResult struct {
    IsLocked       bool          `json:"is_locked"`
    LockoutType    LockoutReason `json:"lockout_type,omitempty"`
    RemainingTime  time.Duration `json:"remaining_time,omitempty"`
    FailedAttempts int           `json:"failed_attempts"`
    RetryAfter     time.Duration `json:"retry_after,omitempty"`
}

type LockoutConfig struct {
    MaxAccountAttempts int
    MaxIPAttempts      int
    AccountLockoutTime time.Duration
    IPLockoutTime      time.Duration
    AttemptWindow      time.Duration
    IPAttemptWindow    time.Duration
}

type AccountLockoutTracker struct {
    redisClient *redis.Client
    config      LockoutConfig
    repo        DataRepository
}

type LockoutRepository interface {
    CreateLockout(ctx context.Context, lockout *AccountLockout) error
    GetActiveLockout(ctx context.Context, userID, ipAddress string) (*AccountLockout, error)
    ReleaseLockout(ctx context.Context, lockoutID string) error
    GetLockoutHistory(ctx context.Context, userID string, limit, offset int) ([]AccountLockout, error)
}

type DataRepository interface {
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    GetUserByID(ctx context.Context, id string) (*User, error)
}

type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 CheckAndRecordFailedAttempt

```
FUNCTION CheckAndRecordFailedAttempt(ctx context.Context, userID, ipAddress string) -> LockoutCheckResult:

1. CONCURRENTLY CHECK:
   a. GetCurrentFailedAttempts(ctx, userID, "account")
   b. GetCurrentFailedAttempts(ctx, ipAddress, "ip")

2. IF account attempts >= MaxAccountAttempts:
   a. GetActiveLockout(ctx, userID, "")
   b. IF lockout exists AND not expired:
      RETURN LockoutCheckResult:
        - IsLocked: true
        - LockoutType: LockoutReasonAccountExceeded
        - RemainingTime: lockout.ExpiresAt - Now()
        - FailedAttempts: account attempts
        - RetryAfter: RemainingTime
   c. ELSE:
      CreateAccountLockout(ctx, userID, ipAddress)
      RETURN LockoutCheckResult:
        - IsLocked: true
        - LockoutType: LockoutReasonAccountExceeded
        - RemainingTime: AccountLockoutDuration
        - FailedAttempts: account attempts

3. IF IP attempts >= MaxIPAttempts:
   a. GetActiveLockout(ctx, "", ipAddress)
   b. IF lockout exists AND not expired:
      RETURN LockoutCheckResult:
        - IsLocked: true
        - LockoutType: LockoutReasonIPExceeded
        - RemainingTime: lockout.ExpiresAt - Now()
        - FailedAttempts: IP attempts
        - RetryAfter: RemainingTime
   c. ELSE:
      CreateIPLockout(ctx, userID, ipAddress)
      RETURN LockoutCheckResult:
        - IsLocked: true
        - LockoutType: LockoutReasonIPExceeded
        - RemainingTime: IPLockoutDuration
        - FailedAttempts: IP attempts

4. RecordFailedAttempt(ctx, userID, ipAddress, "login_failure")

5. RETURN LockoutCheckResult:
   - IsLocked: false
   - FailedAttempts: account attempts + 1
```

### 2.2 GetCurrentFailedAttempts

```
FUNCTION GetCurrentFailedAttempts(ctx context.Context, identifier, attemptType string) -> int:

1. SET key = BuildRedisKey(identifier, attemptType)
   - If attemptType == "account": key = "auth:failed:user:{userID}"
   - If attemptType == "ip": key = "auth:failed:ip:{ipAddress}"

2. EXECUTE redisClient.ZCARD(key) to count attempts within window

3. IF attemptType == "account":
   DELETE attempts older than AttemptWindow using ZREMRANGEBYSCORE(key, 0, Now() - AttemptWindow)

4. RETURN count
```

### 2.3 RecordFailedAttempt

```
FUNCTION RecordFailedAttempt(ctx context.Context, userID, ipAddress, reason string):

1. SET accountKey = "auth:failed:user:{userID}"
2. SET ipKey = "auth:failed:ip:{ipAddress}"
3. SET score = Unix timestamp of Now()

4. EXECUTE redisClient.ZADD(accountKey, score, GenerateUUID())
5. EXECUTE redisClient.ZADD(ipKey, score, GenerateUUID())

6. SET accountTTL = AttemptWindow
7. SET ipTTL = IPAttemptWindow

8. EXECUTE redisClient.EXPIRE(accountKey, accountTTL)
9. EXECUTE redisClient.EXPIRE(ipKey, ipTTL)
```

### 2.4 CreateAccountLockout

```
FUNCTION CreateAccountLockout(ctx context.Context, userID, ipAddress string) -> error:

1. SET lockout = AccountLockout:
   - ID: GenerateUUID()
   - UserID: userID
   - IPAddress: ipAddress
   - Reason: LockoutReasonAccountExceeded
   - Status: LockoutStatusActive
   - CreatedAt: Now()
   - ExpiresAt: Now() + AccountLockoutDuration

2. EXECUTE repo.CreateLockout(ctx, lockout)

3. SET redisKey = "auth:lockout:user:{userID}"
4. SET redisValue = lockout.ID + ":" + string(LockoutReasonAccountExceeded)

5. EXECUTE redisClient.SET(redisKey, redisValue, AccountLockoutDuration)

6. LOG.Info("Account lockout created", "userID", userID, "ipAddress", ipAddress)

7. RETURN nil
```

### 2.5 CreateIPLockout

```
FUNCTION CreateIPLockout(ctx context.Context, userID, ipAddress string) -> error:

1. SET lockout = AccountLockout:
   - ID: GenerateUUID()
   - UserID: userID
   - IPAddress: ipAddress
   - Reason: LockoutReasonIPExceeded
   - Status: LockoutStatusActive
   - CreatedAt: Now()
   - ExpiresAt: Now() + IPLockoutDuration

2. EXECUTE repo.CreateLockout(ctx, lockout)

3. SET redisKey = "auth:lockout:ip:{ipAddress}"
4. SET redisValue = lockout.ID + ":" + string(LockoutReasonIPExceeded)

5. EXECUTE redisClient.SET(redisKey, redisValue, IPLockoutDuration)

6. LOG.Info("IP lockout created", "userID", userID, "ipAddress", ipAddress)

7. RETURN nil
```

### 2.6 IsLockedOut

```
FUNCTION IsLockedOut(ctx context.Context, userID, ipAddress string) -> LockoutCheckResult:

1. CHECK account lockout first:
   SET accountKey = "auth:lockout:user:{userID}"
   EXECUTE redisClient.GET(accountKey)

   IF result exists:
     PARSE result as "lockoutID:reason"
     GET TTL using redisClient.TTL(accountKey)
     RETURN LockoutCheckResult:
       - IsLocked: true
       - LockoutType: LockoutReason(reason)
       - RemainingTime: Duration(TTL)
       - RetryAfter: Duration(TTL)

2. CHECK IP lockout:
   SET ipKey = "auth:lockout:ip:{ipAddress}"
   EXECUTE redisClient.GET(ipKey)

   IF result exists:
     PARSE result as "lockoutID:reason"
     GET TTL using redisClient.TTL(ipKey)
     RETURN LockoutCheckResult:
       - IsLocked: true
       - LockoutType: LockoutReason(reason)
       - RemainingTime: Duration(TTL)
       - RetryAfter: Duration(TTL)

3. RETURN LockoutCheckResult:
   - IsLocked: false
```

### 2.7 ClearFailedAttempts

```
FUNCTION ClearFailedAttempts(ctx context.Context, userID, ipAddress string) -> error:

1. SET accountKey = "auth:failed:user:{userID}"
2. SET ipKey = "auth:failed:ip:{ipAddress}"

3. EXECUTE redisClient.DEL(accountKey)
4. EXECUTE redisClient.DEL(ipKey)

5. LOG.Info("Failed attempts cleared", "userID", userID, "ipAddress", ipAddress)

6. RETURN nil
```

### 2.8 ReleaseLockout

```
FUNCTION ReleaseLockout(ctx context.Context, lockoutID string, reason string) -> error:

1. GET lockout from repo.GetActiveLockout("", "") - search by ID
   IF error:
     RETURN ErrLockoutNotFound

2. SET now = Now()
3. SET lockout.Status = LockoutStatusReleased
4. SET lockout.ReleasedAt = &now

5. EXECUTE repo.ReleaseLockout(ctx, lockoutID)

6. IF lockout.UserID != "":
   SET redisKey = "auth:lockout:user:{lockout.UserID}"
   EXECUTE redisClient.DEL(redisKey)

7. IF lockout.IPAddress != "":
   SET redisKey = "auth:lockout:ip:{lockout.IPAddress}"
   EXECUTE redisClient.DEL(redisKey)

8. CLEAR failed attempts for user/IP
   EXECUTE ClearFailedAttempts(ctx, lockout.UserID, lockout.IPAddress)

9. LOG.Info("Lockout released", "lockoutID", lockoutID, "reason", reason)

10. RETURN nil
```

### 2.9 GetLockoutStatus

```
FUNCTION GetLockoutStatus(ctx context.Context, userID string) -> []AccountLockout:

1. EXECUTE repo.GetLockoutHistory(ctx, userID, 10, 0)

2. FOR EACH lockout in results:
   IF lockout.Status == LockoutStatusActive AND lockout.ExpiresAt < Now():
     SET lockout.Status = LockoutStatusExpired

3. RETURN processed lockouts
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error Code | Condition | Handling Strategy |
| :--- | :--- | :--- |
| `ErrRedisUnavailable` | Redis connection failure | Fail closed - treat as locked out, log critical error |
| `ErrLockoutNotFound` | Attempt to release non-existent lockout | Return error, log warning |
| `ErrDatabaseUnavailable` | PostgreSQL connection failure | Retry with exponential backoff, fail after max retries |
| `ErrRateLimitExceeded` | Too many lockout check requests | Apply rate limiting middleware |
| `ErrInvalidLockoutID` | Malformed lockout ID format | Return validation error |

### 3.2 State Transitions

```
State Diagram:

[Normal] --> [Account Lockout] --> [Released/Expired] --> [Normal]
    |                |                     |
    |                v                     v
    +-----------> [IP Lockout] ------------+
                       |
                       v
                 [Released/Expired]

Lockout States:
- LockoutStatusActive: Lockout is in effect, attempts blocked
- LockoutStatusReleased: Manually released by admin or successful login
- LockoutStatusExpired: Lockout duration has passed, automatically cleared

Failed Attempts States:
- Active: Within the attempt window, counted toward lockout threshold
- Expired: Outside the attempt window, automatically removed
```

### 3.3 Concurrency Handling

1. **Redis Atomic Operations**: Use Redis transactions (MULTI/EXEC) for critical operations
2. **Lock Acquisition**: Use Redis SETNX for distributed locking during lockout creation
3. **Race Condition Prevention**:
   - Check-then-act pattern avoided for lockout creation
   - Use INCR/DECR for atomic counter operations
   - Idempotent lockout creation via Redis SET with NX flag

### 3.4 Fail-Safe Behavior

```
On Redis Failure:
  1. Log critical error with full context
  2. Default to LOCKED OUT state for all requests
  3. Attempt Redis reconnection in background
  4. Fall back to database-only mode if reconnection fails
  5. Alert on-call personnel via configured channels

On Database Failure:
  1. Use cached lockout status from Redis
  2. Queue lockout creation for retry
  3. Log error and continue with Redis-only validation
  4. Attempt database reconnection with exponential backoff
```

## 4. Component Interfaces

### 4.1 Public Methods

```go
type AccountLockoutTracker interface {
    CheckAndRecordFailedAttempt(ctx context.Context, userID, ipAddress string) (*LockoutCheckResult, error)
    IsLockedOut(ctx context.Context, userID, ipAddress string) (*LockoutCheckResult, error)
    ClearFailedAttempts(ctx context.Context, userID, ipAddress string) error
    ReleaseLockout(ctx context.Context, lockoutID, reason string) error
    GetLockoutStatus(ctx context.Context, userID string) ([]AccountLockout, error)
    GetConfig() LockoutConfig
}
```

### 4.2 Method Signatures

```go
func (t *AccountLockoutTracker) CheckAndRecordFailedAttempt(
    ctx context.Context,
    userID string,
    ipAddress string,
) (*LockoutCheckResult, error)

func (t *AccountLockoutTracker) IsLockedOut(
    ctx context.Context,
    userID string,
    ipAddress string,
) (*LockoutCheckResult, error)

func (t *AccountLockoutTracker) ClearFailedAttempts(
    ctx context.Context,
    userID string,
    ipAddress string,
) error

func (t *AccountLockoutTracker) ReleaseLockout(
    ctx context.Context,
    lockoutID string,
    reason string,
) error

func (t *AccountLockoutTracker) GetLockoutStatus(
    ctx context.Context,
    userID string,
) ([]AccountLockout, error)

func (t *AccountLockoutTracker) GetConfig() LockoutConfig
```

### 4.3 Integration with AuthController

```go
type AuthController struct {
    lockoutTracker AccountLockoutTracker
    passwordHasher PasswordHasher
    jwtManager     JWTManager
    sessionManager *SessionManager
    oauthHandler   OAuthHandler
    repo           DataRepository
}

func (c *AuthController) Login(ctx *fiber.Ctx) error {
    email := ctx.FormValue("email")
    password := ctx.FormValue("password")
    ipAddress := ctx.IP()

    result, err := c.lockoutTracker.IsLockedOut(ctx.Context(), email, ipAddress)
    if err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Lockout check failed")
    }

    if result.IsLocked {
        return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
            "error":           "account_locked",
            "message":         "Too many failed attempts. Please try again later.",
            "retry_after":     result.RetryAfter.Seconds(),
            "lockout_type":    result.LockoutType,
        })
    }

    user, err := c.repo.GetUserByEmail(ctx.Context(), email)
    if err != nil {
        return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
    }

    if !c.passwordHasher.Verify(user.Password, password) {
        lockoutResult, _ := c.lockoutTracker.CheckAndRecordFailedAttempt(
            ctx.Context(), user.ID, ipAddress,
        )
        return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error":           "invalid_credentials",
            "message":         "Invalid email or password",
            "failed_attempts": lockoutResult.FailedAttempts,
        })
    }

    c.lockoutTracker.ClearFailedAttempts(ctx.Context(), user.ID, ipAddress)

    return c.issueTokens(ctx, user)
}
```

### 4.4 Redis Key Patterns

```
Key Patterns:
- auth:failed:user:{userID}     - Sorted set of failed attempts for account
- auth:failed:ip:{ipAddress}    - Sorted set of failed attempts for IP
- auth:lockout:user:{userID}    - Active account lockout
- auth:lockout:ip:{ipAddress}   - Active IP lockout
- auth:lock:{key}               - Distributed lock for critical sections
```

### 4.5 Metrics and Monitoring

```go
type LockoutMetrics struct {
    FailedAttemptsTotal    *prometheus.CounterVec
    LockoutsCreatedTotal   *prometheus.CounterVec
    LockoutsReleasedTotal  *prometheus.CounterVec
    ActiveLockoutsGauge    *prometheus.GaugeVec
    LockoutDurationHistogram *prometheus.HistogramVec
}

func RecordFailedAttempt(userID, ipAddress string, lockoutType LockoutReason) {
    metrics.FailedAttemptsTotal.WithLabelValues(
        string(lockoutType),
        userID,
        ipAddress,
    ).Inc()
}

func RecordLockoutCreated(lockout *AccountLockout) {
    metrics.LockoutsCreatedTotal.WithLabelValues(
        string(lockout.Reason),
        lockout.UserID,
        lockout.IPAddress,
    ).Inc()
}


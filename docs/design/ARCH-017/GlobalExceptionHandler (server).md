# GlobalExceptionHandler (Server)

**Traceability:** ARCH-017

---

## 1. Data Structures & Types

### 1.1 Core Error Types

```go
package errors

type ErrorCategory string

const (
    ErrorCategoryValidation     ErrorCategory = "validation"
    ErrorCategoryAuthentication ErrorCategory = "authentication"
    ErrorCategoryAuthorization  ErrorCategory = "authorization"
    ErrorCategoryNotFound       ErrorCategory = "not_found"
    ErrorCategoryConflict       ErrorCategory = "conflict"
    ErrorCategoryRateLimit      ErrorCategory = "rate_limit"
    ErrorCategoryTimeout        ErrorCategory = "timeout"
    ErrorCategoryInternal       ErrorCategory = "internal"
    ErrorCategoryExternal       ErrorCategory = "external"
    ErrorCategoryNetwork        ErrorCategory = "network"
)

type ErrorSeverity string

const (
    ErrorSeverityLow      ErrorSeverity = "low"
    ErrorSeverityMedium   ErrorSeverity = "medium"
    ErrorSeverityHigh     ErrorSeverity = "high"
    ErrorSeverityCritical ErrorSeverity = "critical"
)

type AppError struct {
    Category       ErrorCategory    `json:"category"`
    Severity       ErrorSeverity    `json:"severity"`
    Code           string           `json:"code"`
    Message        string           `json:"message"`
    UserMessage    string           `json:"userMessage"`
    Details        map[string]any   `json:"details,omitempty"`
    InternalLog    string           `json:"-"`
    Retryable      bool             `json:"retryable"`
    FeatureFlag    string           `json:"featureFlag,omitempty"`
    Timestamp      time.Time        `json:"timestamp"`
    RequestID      string           `json:"requestId"`
    StackTrace     string           `json:"-"`
    WrappedError   error            `json:"-"`
}
```

### 1.2 Error Response Types

```go
package errors

type ErrorResponse struct {
    Success    bool              `json:"success"`
    Error      ErrorInfo         `json:"error"`
    RequestID  string            `json:"requestId"`
    RetryAfter *int              `json:"retryAfter,omitempty"`
}

type ErrorInfo struct {
    Category    ErrorCategory    `json:"category"`
    Code        string           `json:"code"`
    Message     string           `json:"message"`
    Details     map[string]any   `json:"details,omitempty"`
}
```

### 1.3 Feature Degradation Types

```go
package errors

type FeatureStatus string

const (
    FeatureStatusActive      FeatureStatus = "active"
    FeatureStatusDegraded    FeatureStatus = "degraded"
    FeatureStatusDisabled    FeatureStatus = "disabled"
)

type DegradationState struct {
    Feature       string          `json:"feature"`
    Status        FeatureStatus   `json:"status"`
    Error         *AppError       `json:"error,omitempty"`
    LastUpdated   time.Time       `json:"lastUpdated"`
    RetryCount    int             `json:"retryCount"`
    MaxRetries    int             `json:"maxRetries"`
}
```

### 1.4 Retry Configuration Types

```go
package errors

type RetryConfig struct {
    MaxAttempts       int           `json:"maxAttempts"`
    InitialDelay      time.Duration `json:"initialDelay"`
    MaxDelay          time.Duration `json:"maxDelay"`
    Multiplier        float64       `json:"multiplier"`
    Jitter            bool          `json:"jitter"`
    RetryableCategories map[ErrorCategory]bool `json:"retryableCategories"`
}
```

### 1.5 Error Mapper Configuration

```go
package errors

type MessageMapping struct {
    Category       ErrorCategory    `json:"category"`
    Code           string           `json:"code"`
    UserMessage    string           `json:"userMessage"`
    HttpStatusCode int              `json:"httpStatusCode"`
    Severity       ErrorSeverity    `json:"severity"`
    Retryable      bool             `json:"retryable"`
}
```

### 1.6 Global Handler Context

```go
package errors

type HandlerContext struct {
    Config              *HandlerConfig
    Logger              *log.Logger
    ErrorMapper         *ErrorMapper
    RetryManager        *RetryManager
    FeatureManager      *FeatureManager
    MetricsClient       metrics.Client
    ErrorChannel        chan *AppError
   Wg                  sync.WaitGroup
    ShutdownCtx         context.Context
    CancelShutdown      context.CancelFunc
}

type HandlerConfig struct {
    Env                    string            `json:"env"`
    EnableStackTrace       bool              `json:"enableStackTrace"`
    EnableInternalLog      bool              `json:"enableInternalLog"`
    DefaultRetryConfig     RetryConfig       `json:"defaultRetryConfig"`
    FeatureTimeout         time.Duration     `json:"featureTimeout"`
    RecoveryTimeout        time.Duration     `json:"recoveryTimeout"`
    MessageMappings        []MessageMapping  `json:"messageMappings"`
    CriticalCategories     []ErrorCategory   `json:"criticalCategories"`
    DegradedFeatures       map[string]string `json:"degradedFeatures"`
    NotificationWebhookURL string            `json:"notificationWebhookURL"`
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 GlobalExceptionHandler Initialization

**Algorithm: InitializeHandlerContext**

```
1. Load HandlerConfig from environment/config file
2. Initialize logger with configured level and output
3. Create ErrorMapper with message mappings
4. Initialize RetryManager with default retry configuration
5. Initialize FeatureManager with degraded features configuration
6. Set up metrics client for error tracking
7. Create buffered error channel (size: 1000)
8. Start background workers:
   a. ErrorProcessorWorker (consumes error channel)
   b. RetryWorker (handles retry logic)
   c. RecoveryWorker (monitors feature recovery)
9. Register Fiber middleware handler
10. Return initialized HandlerContext
```

### 2.2 Error Interception Middleware Flow

**Algorithm: HandleFiberError**

```
1. Receive fiber.Ctx and error from Fiber router
2. Extract RequestID from context (or generate new one)
3. IF error is nil:
   a. Call next handler
   b. RETURN

4. WRAP error with RequestID and timestamp
5. CLASSIFY error using ClassifyError()
6. IF error is critical:
   a. TRIGGER critical error handler
   b. Log error with full stack trace
   c. Send notification to webhook if configured
   d. RETURN 500 Internal Server Error

7. MAP error to user-friendly message using ErrorMapper
8. DETERMINE HTTP status code from error category
9. CHECK if feature should be degraded:
   a. IF degraded feature identified:
      b. UPDATE FeatureManager state to degraded
      c. Log degradation event

10. BUILD ErrorResponse with:
    a. success: false
    b. error: mapped error info
    c. requestId: RequestID
    d. retryAfter: if rate limited

11. LOG error (internal only, based on config)
12. SEND response with appropriate status code
13. PUBLISH error to error channel for background processing
```

### 2.3 Error Classification Algorithm

**Algorithm: ClassifyError**

```
1. INPUT: error, request context

2. IF error implements AppError interface:
   a. RETURN error.Category

3. CHECK error type using type assertion:
   a. IF *ValidationError:          RETURN ErrorCategoryValidation
   b. IF *AuthenticationError:      RETURN ErrorCategoryAuthentication
   c. IF *AuthorizationError:       RETURN ErrorCategoryAuthorization
   d. IF *NotFoundError:            RETURN ErrorCategoryNotFound
   e. IF *ConflictError:            RETURN ErrorCategoryConflict
   f. IF *RateLimitError:           RETURN ErrorCategoryRateLimit
   g. IF *TimeoutError:             RETURN ErrorCategoryTimeout

4. IF error is from external service:
   a. EXTRACT service name from error
   b. RETURN ErrorCategoryExternal

5. IF error is network-related (timeout, connection refused):
   a. RETURN ErrorCategoryNetwork

6. IF error contains "sql" or "database":
   a. RETURN ErrorCategoryInternal

7. DEFAULT:
   a. RETURN ErrorCategoryInternal
```

### 2.4 Error Mapping Algorithm

**Algorithm: MapErrorToUserMessage**

```
1. INPUT: AppError

2. LOOKUP message mapping by Category and Code
3. IF mapping found:
   a. RETURN mapping.UserMessage

4. IF no mapping found:
   a. GENERATE generic message based on category:
      - Validation: "The information you provided is invalid"
      - Authentication: "Please sign in to continue"
      - Authorization: "You don't have permission to do this"
      - NotFound: "The requested resource was not found"
      - Conflict: "This action conflicts with existing data"
      - RateLimit: "Too many requests. Please try again later"
      - Timeout: "The request took too long. Please try again"
      - Internal: "Something went wrong on our end"
      - External: "An external service is unavailable"
      - Network: "Unable to connect. Please check your connection"

5. IF in development mode:
   a. INCLUDE error code in message

6. RETURN generated message
```

### 2.5 Retry Logic Algorithm

**Algorithm: ShouldRetry**

```
1. INPUT: AppError, currentAttempt int, retryConfig RetryConfig

2. IF currentAttempt >= retryConfig.MaxAttempts:
   a. RETURN false

3. IF error.Retryable is true:
   a. RETURN true

4. IF error.Category is in retryableCategories:
   a. RETURN true

5. IF error.Category is Timeout or Network:
   a. RETURN true

6. RETURN false
```

### 2.6 Retry Delay Calculation Algorithm

**Algorithm: CalculateRetryDelay**

```
1. INPUT: currentAttempt int, retryConfig RetryConfig

2. CALCULATE base delay:
   delay = retryConfig.InitialDelay * (retryConfig.Multiplier ^ currentAttempt)

3. IF delay > retryConfig.MaxDelay:
   a. delay = retryConfig.MaxDelay

4. IF retryConfig.Jitter is true:
   a. ADD random jitter: delay = delay * (0.5 + rand.Float64() * 0.5)

5. RETURN delay
```

### 2.7 Feature Degradation Algorithm

**Algorithm: UpdateFeatureStatus**

```
1. INPUT: featureName string, error *AppError

2. ACQUIRE lock on FeatureManager

3. IF featureName not in degradedFeatures:
   a. Log warning: "Unknown feature requested for degradation"
   b. RETURN

4. IF error is nil:
   a. SET feature status to FeatureStatusActive
   b. CLEAR error from degradation state
   c. Log: featureName + " recovered"
   d. PUBLISH recovery event

5. ELSE:
   a. INCREMENT retry count in DegradationState
   b. IF retry count > max retries:
      i. SET feature status to FeatureStatusDisabled
   c. ELSE:
      i. SET feature status to FeatureStatusDegraded
      ii. SET error in DegradationState
      iii. PUBLISH degradation event

6. UPDATE lastUpdated timestamp
7. RELEASE lock
```

### 2.8 Background Error Processing Algorithm

**Algorithm: ErrorProcessorWorker**

```
1. LOOP until shutdown:
   a. SELECT from error channel:
      i. CASE error := <-errorChannel:
         - LOG error with appropriate level (based on severity)
         - IF severity is High or Critical:
           * SEND alert to notification webhook
           * INCREMENT error counter metric
           * UPDATE error rate metric
         - IF error is from external service:
           * LOG service name and error details
           * INCREMENT external error counter
         - IF error category is in criticalCategories:
           * TRIGGER circuit breaker check
         - CONTINUE to next iteration

      ii. CASE <-shutdownCtx.Done():
         a. BREAK loop

2. LOG: "Error processor worker shutting down"
3. WAIT for all in-flight error processing to complete
```

### 2.9 Panic Recovery Algorithm

**Algorithm: RecoverFromPanic**

```
1. RECEIVE panic value from recover()

2. IF panic value is nil:
   a. RETURN

3. EXTRACT RequestID from context
4. CREATE AppError from panic:
   a. Category: ErrorCategoryInternal
   b. Severity: ErrorSeverityCritical
   c. Code: "PANIC_RECOVERY"
   d. Message: "Internal server error"
   e. InternalLog: panic value as string
   f. StackTrace: capture from runtime.Stack()

5. LOG error with stack trace at FATAL level
6. SEND notification if webhook configured
7. SEND generic 500 response to client
8. PUBLISH error to error channel
```

---

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error State | Trigger | Transition |
|-------------|---------|------------|
| **Normal Operation** | No errors | Transition to Degraded on non-critical error |
| **Degraded Mode** | Non-critical feature failure | Transition to Normal on recovery, Disabled on max retries |
| **Disabled Feature** | Feature exceeds max retry attempts | Transition to Degraded on manual reset or timeout |
| **Critical Failure** | Critical error (database, auth) | Halt affected operations, alert on-call |
| **Rate Limited** | Too many requests | Transition to Normal after retry-after period |
| **Timeout** | Request exceeds 10 seconds | Offer manual retry, auto-retry on connectivity |
| **Network Failure** | Connectivity loss | Preserve state, auto-retry on restoration |

### 3.2 State Transitions

```
State Machine: Feature Status

     [ACTIVE] ----non-critical error----> [DEGRADED]
        ^                                   |
        |                                   | recovery
        |                                   v
        +----<-----------<----------- [RETRYING] (background)
        |                                   |
        |                  max retries      v
        +----<-----------<----------- [DISABLED]
        |                                   |
        |              manual reset/timeout v
        +----<-----------<----------- [ACTIVE]
```

### 3.3 Error Handling Strategies

**Database Errors:**
- Log with full context including query and parameters
- Check connection pool health
- If primary DB fails, attempt read from replica if operation is read-only
- Set feature status to degraded for affected data operations
- Retry with exponential backoff if error is connection-related

**External Service Errors:**
- Classify by service (Stripe, Resend, USDA API)
- Map to user-friendly messages specific to service
- If service is unavailable, degrade related features
- Do NOT retry if error indicates invalid input (4xx from external)
- Retry with backoff for 5xx errors from external services

**Authentication/Authorization Errors:**
- Log security events at WARNING level
- Include user ID and IP address in logs
- Do NOT expose internal error details
- Return generic auth failure message
- No retry for auth errors (require user action)

**Validation Errors:**
- Collect all validation errors in Details map
- Return 400 status with field-level error messages
- No retry necessary
- Include field names and validation rules in response

**Rate Limit Errors:**
- Extract Retry-After header from rate limit response
- Set RetryAfter in ErrorResponse
- Do NOT retry until after retry-after period
- Log rate limit hit for monitoring

### 3.4 Error Logging Strategy

**Log Levels by Error Category:**

| Category | Log Level | Include Stack | Include Request |
|----------|-----------|---------------|-----------------|
| Validation | DEBUG | No | Yes |
| Authentication | WARN | Yes | Yes |
| Authorization | WARN | Yes | Yes |
| NotFound | INFO | No | Yes |
| Conflict | WARN | Yes | Yes |
| RateLimit | INFO | No | Yes |
| Timeout | WARN | Yes | Yes |
| Internal | ERROR | Yes | Yes |
| External | ERROR | Yes | Yes |
| Network | ERROR | Yes | Yes |

### 3.5 Metrics and Monitoring

**Metrics to Track:**

```go
package metrics

type ErrorMetrics struct {
    TotalErrors           counter.Counter
    ErrorsByCategory      *map[ErrorCategory]counter.Counter
    ErrorsByFeature       *map[string]counter.Counter
    ErrorRate             gauge.Gauge
    RetryAttempts         counter.Counter
    RetrySuccesses        counter.Counter
    RetryFailures         counter.Counter
    FeatureDegradations   counter.Counter
    FeatureRecoveries     counter.Counter
    PanicCount            counter.Counter
    MeanTimeToRecovery    histogram.Histogram
}
```

---

## 4. Component Interfaces

### 4.1 GlobalExceptionHandler Public Interface

```go
package fibererror

import (
    "github.com/gofiber/fiber/v2"
)

type GlobalExceptionHandler interface {
    // Middleware returns a Fiber handler for error interception
    Middleware() fiber.Handler

    // HandleError processes an error and returns appropriate response
    HandleError(ctx *fiber.Ctx, err error) error

    // HandlePanic recovers from panic and processes the error
    HandlePanic(ctx *fiber.Ctx, recovered interface{}, stack []byte)

    // IsFeatureDegraded checks if a feature is in degraded state
    IsFeatureDegraded(featureName string) bool

    // GetFeatureStatus returns the current status of a feature
    GetFeatureStatus(featureName string) *DegradationState

    // ForceFeatureRecovery manually resets a feature to active
    ForceFeatureRecovery(featureName string)

    // Shutdown gracefully shuts down the handler
    Shutdown(ctx context.Context) error
}
```

### 4.2 ErrorMapper Interface

```go
package fibererror

type ErrorMapper interface {
    // ClassifyError determines the error category
    ClassifyError(err error) ErrorCategory

    // MapToUserMessage converts error to user-friendly message
    MapToUserMessage(err error) string

    // MapToResponse creates an ErrorResponse from an error
    MapToResponse(err error, requestID string) *ErrorResponse

    // RegisterMapping adds a new message mapping
    RegisterMapping(mapping MessageMapping)

    // GetMappings returns all registered mappings
    GetMappings() []MessageMapping
}
```

### 4.3 RetryManager Interface

```go
package fibererror

type RetryManager interface {
    // ShouldRetry determines if an operation should be retried
    ShouldRetry(err error, attempt int) bool

    // CalculateDelay returns the delay before next retry attempt
    CalculateDelay(attempt int) time.Duration

    // ExecuteWithRetry runs a function with retry logic
    ExecuteWithRetry(ctx context.Context, fn func() error, config *RetryConfig) error

    // GetRetryConfig returns the current retry configuration
    GetRetryConfig() *RetryConfig

    // UpdateRetryConfig updates the retry configuration
    UpdateRetryConfig(config RetryConfig)
}
```

### 4.4 FeatureManager Interface

```go
package fibererror

type FeatureManager interface {
    // SetDegraded marks a feature as degraded with associated error
    SetDegraded(featureName string, err error) error

    // SetActive marks a feature as active (recovered)
    SetActive(featureName string) error

    // GetStatus returns the current status of a feature
    GetStatus(featureName string) (*DegradationState, error)

    // GetAllStatuses returns status of all tracked features
    GetAllStatuses() map[string]*DegradationState

    // IsActive checks if a feature is fully operational
    IsActive(featureName string) bool

    // IsDegraded checks if a feature is in degraded mode
    IsDegraded(featureName string) bool

    // Shutdown gracefully shuts down the manager
    Shutdown(ctx context.Context) error
}
```

### 4.5 Internal Function Signatures

```go
package fibererror

func (h *globalExceptionHandler) buildErrorResponse(err error, requestID string) *ErrorResponse

func (h *globalExceptionHandler) determineHttpStatus(category ErrorCategory) int

func (h *globalExceptionHandler) extractRequestID(ctx *fiber.Ctx) string

func (h *globalExceptionHandler) logError(err *AppError)

func (h *globalExceptionHandler) sendAlert(err *AppError)

func (h *globalExceptionHandler) startBackgroundWorkers(ctx context.Context)

func (h *globalExceptionHandler) stopBackgroundWorkers(timeout time.Duration)

func (em *errorMapper) classifyError(err error) ErrorCategory

func (em *errorMapper) mapError(err error) *MessageMapping

func (rm *retryManager) executeWithRetry(ctx context.Context, fn func() error, config RetryConfig) error

func (fm *featureManager) updateStatus(featureName string, updateFunc func(*DegradationState))
```

---

## 5. Fiber Middleware Integration

```go
package fibererror

func NewGlobalExceptionHandler(config HandlerConfig, logger *log.Logger) GlobalExceptionHandler {
    ctx := &HandlerContext{
        Config:       &config,
        Logger:       logger,
        ErrorChannel: make(chan *AppError, 1000),
    }

    handler := &globalExceptionHandler{
        context: ctx,
    }

    ctx.ErrorMapper = NewErrorMapper(config.MessageMappings)
    ctx.RetryManager = NewRetryManager(config.DefaultRetryConfig)
    ctx.FeatureManager = NewFeatureManager(config.DegradedFeatures)

    return handler
}

func (h *globalExceptionHandler) Middleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        defer func() {
            if r := recover(); r != nil {
                h.HandlePanic(c, r, nil)
            }
        }()

        err := c.Next()

        if err != nil {
            return h.HandleError(c, err)
        }

        return nil
    }
}

func main() {
    app := fiber.New(fiber.Config{
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            handler := NewGlobalExceptionHandler(config, logger)
            return handler.HandleError(c, err)
        },
    })

    app.Use(NewGlobalExceptionHandler(config, logger).Middleware())
}
```

---

## 6. Configuration

### 6.1 Environment Variables

```bash
ERROR_HANDLER_ENV=production
ERROR_HANDLER_ENABLE_STACK_TRACE=false
ERROR_HANDLER_ENABLE_INTERNAL_LOG=true
ERROR_HANDLER_FEATURE_TIMEOUT=5m
ERROR_HANDLER_RECOVERY_TIMEOUT=30s
ERROR_HANDLER_NOTIFICATION_WEBHOOK_URL=https://hooks.example.com/alerts
```

### 6.2 YAML Configuration

```yaml
errorHandler:
  env: production
  enableStackTrace: false
  enableInternalLog: true
  defaultRetryConfig:
    maxAttempts: 3
    initialDelay: 1s
    maxDelay: 30s
    multiplier: 2.0
    jitter: true
    retryableCategories:
      - timeout
      - network
      - external
  featureTimeout: 5m
  recoveryTimeout: 30s
  criticalCategories:
    - authentication
    - internal
  degradedFeatures:
    recommendations: "recommendations-service"
    historySync: "history-sync-service"
  messageMappings:
    - category: validation
      code: VALIDATION_ERROR
      userMessage: "Please check your input and try again"
      httpStatusCode: 400
      severity: low
      retryable: false
    - category: authentication
      code: AUTH_REQUIRED
      userMessage: "Please sign in to continue"
      httpStatusCode: 401
      severity: medium
      retryable: false
    - category: rate_limit
      code: RATE_LIMIT_EXCEEDED
      userMessage: "Too many requests. Please try again later"
      httpStatusCode: 429
      severity: medium
      retryable: true
```

---

## 7. Dependencies

- **Fiber** (`github.com/gofiber/fiber/v2`) - Web framework, middleware support
- **Context** (`context`) - Standard library for cancellation and timeouts
- **Sync** (`sync`) - WaitGroup for graceful shutdown
- **Log** (`log`) - Standard library logging
- **Runtime** (`runtime`) - Stack trace capture
- **Redis Client** - Optional, for distributed feature state
- **Metrics Client** - Optional, for error tracking

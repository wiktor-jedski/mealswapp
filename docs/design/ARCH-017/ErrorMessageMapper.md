# ErrorMessageMapper

**Traceability:** ARCH-017

## 1. Data Structures & Types

```go
// ErrorCategory represents the classification of an error for user messaging
type ErrorCategory string

const (
    ErrorCategoryNetwork         ErrorCategory = "network"
    ErrorCategoryTimeout         ErrorCategory = "timeout"
    ErrorCategoryValidation      ErrorCategory = "validation"
    ErrorCategoryAuthentication  ErrorCategory = "authentication"
    ErrorCategoryAuthorization   ErrorCategory = "authorization"
    ErrorCategoryNotFound        ErrorCategory = "not_found"
    ErrorCategoryConflict        ErrorCategory = "conflict"
    ErrorCategoryRateLimit       ErrorCategory = "rate_limit"
    ErrorCategoryInternal        ErrorCategory = "internal"
    ErrorCategoryExternalService ErrorCategory = "external_service"
    ErrorCategoryGracefulDegradation ErrorCategory = "graceful_degradation"
)

// ErrorSeverity indicates how the error affects the user experience
type ErrorSeverity string

const (
    ErrorSeverityLow      ErrorSeverity = "low"
    ErrorSeverityMedium   ErrorSeverity = "medium"
    ErrorSeverityHigh     ErrorSeverity = "high"
    ErrorSeverityCritical ErrorSeverity = "critical"
)

// ErrorContext provides additional information for error mapping
type ErrorContext struct {
    UserID       string                 `json:"user_id,omitempty"`
    RequestPath  string                 `json:"request_path,omitempty"`
    RequestMethod string                `json:"request_method,omitempty"`
    FeatureName  string                 `json:"feature_name,omitempty"`
    RetryCount   int                    `json:"retry_count,omitempty"`
    Timestamp    time.Time              `json:"timestamp"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UserFacingError represents the final error message shown to users
type UserFacingError struct {
    Message       string         `json:"message"`
    Category      ErrorCategory  `json:"category"`
    Severity      ErrorSeverity  `json:"severity"`
    Retryable     bool           `json:"retryable"`
    RetryAfter    *time.Duration `json:"retry_after,omitempty"`
    ActionLabel   string         `json:"action_label,omitempty"`
    FeatureFlag   string         `json:"feature_flag,omitempty"`
    TechnicalRef  string         `json:"technical_ref,omitempty"`
}

// ErrorMappingRule defines how to map an error to a user-facing error
type ErrorMappingRule struct {
    MatchCondition func(error, ErrorContext) bool
    Category       ErrorCategory
    Severity       ErrorSeverity
    MessageKey     string
    Retryable      bool
    DefaultMessage string
}

// ErrorMappingConfig holds the configuration for error mapping
type ErrorMappingConfig struct {
    EnableDetailedLogging bool
    DefaultLanguage       string
    SupportedLanguages    []string
    MaxContextDepth       int
}

// InternalError represents the structure of internal/system errors
type InternalError struct {
    Code        string                 `json:"code"`
    Message     string                 `json:"message"`
    OriginalErr error                  `json:"original_error,omitempty"`
    StackTrace  string                 `json:"stack_trace,omitempty"`
    Context     ErrorContext           `json:"context"`
}
```

## 2. Logic & Algorithms

### 2.1 Error Mapping Flow

```
FUNCTION MapError(internalError error, context ErrorContext) UserFacingError:
    IF internalError IS NIL:
        RETURN UserFacingError{
            Message:   "An unexpected error occurred",
            Category:  ErrorCategoryInternal,
            Severity:  ErrorSeverityMedium,
            Retryable: false,
        }

    internalErr := TOInternalError(internalError)

    FOR EACH rule IN mappingRules:
        IF rule.MatchCondition(internalErr, context):
            userErr := CREATEUserFacingError(rule, internalErr, context)
            LOGErrorMapping(internalErr, userErr)
            RETURN userErr

    RETURN CREATEDefaultError(internalErr, context)
```

### 2.2 Match Condition Evaluation

```
FUNCTION EvaluateMatchCondition(rule ErrorMappingRule, err InternalError, ctx ErrorContext) bool:
    IF rule.CodeMatch != "" AND rule.CodeMatch != err.Code:
        RETURN false

    IF rule.CategoryMatch != "" AND NOT matchesCategory(err, rule.CategoryMatch):
        RETURN false

    IF rule.ContextRequired != "":
        IF ctx.FeatureName != rule.ContextRequired:
            RETURN false

    IF rule.RetryCountMatch != nil:
        IF ctx.RetryCount != rule.RetryCountMatch:
            RETURN false

    RETURN true
```

### 2.3 Message Resolution

```
FUNCTION ResolveMessage(rule ErrorMappingRule, err InternalError, ctx ErrorContext) string:
    messageKey := rule.MessageKey

    localizedMessage := LOOKUPLocalizedMessage(messageKey, ctx.Language)

    IF localizedMessage != "":
        RETURN INTERPOLATE(localizedMessage, err, ctx)

    RETURN rule.DefaultMessage
```

### 2.4 Graceful Degradation Check

```
FUNCTION ShouldDegradeGracefully(category ErrorCategory, context ErrorContext) bool:
    degradableFeatures := {"history_sync", "recommendations", "analytics"}

    IF category IS ErrorCategoryExternalService:
        IF context.FeatureName IN degradableFeatures:
            RETURN true

    IF category IS ErrorCategoryTimeout:
        IF context.FeatureName IN degradableFeatures AND context.RetryCount >= 2:
            RETURN true

    RETURN false
```

### 2.5 Retry After Calculation

```
FUNCTION CalculateRetryAfter(err InternalError, context ErrorContext) *time.Duration:
    baseDelay := 1 * time.Second
    maxDelay := 30 * time.Second
    exponentialBase := 2.0

    IF context.RetryCount == 0:
        RETURN &baseDelay

    delay := baseDelay * POW(exponentialBase, float64(context.RetryCount-1))

    IF delay > maxDelay:
        delay = maxDelay

    jitter := RANDOM(-delay*0.1, delay*0.1)
    delayWithJitter := delay + jitter

    RETURN &delayWithJitter
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| State | Trigger | User Experience | Retry Behavior |
|-------|---------|-----------------|----------------|
| `NetworkFailure` | Connection refused, DNS failure, TLS error | "Unable to connect. Please check your internet connection." | Auto-retry on connectivity restoration |
| `Timeout` | Request exceeds 10s threshold | "This is taking longer than expected. Would you like to wait or try again?" | Manual retry only after timeout |
| `ValidationError` | Invalid input, missing required fields | "Please check your input and try again." | Manual retry with corrected input |
| `AuthenticationRequired` | 401 Unauthorized | "Please sign in to continue." | Redirect to login |
| `ForbiddenAccess` | 403 Forbidden | "You don't have permission to access this resource." | Contact support or admin |
| `NotFound` | 404 Not Found | "The requested resource could not be found." | Manual retry with valid input |
| `RateLimited` | 429 Too Many Requests | "You've made too many requests. Please wait a moment." | Auto-retry after delay |
| `ServiceUnavailable` | 503 Service Unavailable | "Service temporarily unavailable. Retrying..." | Auto-retry with exponential backoff |
| `ExternalServiceError` | Stripe, Resend, USDA API failure | Feature-specific message with graceful degradation | Conditional retry |
| `InternalError` | 500 Internal Server Error | "Something went wrong on our end. Please try again later." | Manual retry, auto-report |

### 3.2 State Transitions

```
STATE: Ready → Processing
    TRIGGER: MapError called
    ACTION: Start error analysis

STATE: Processing → Mapped
    TRIGGER: Matching rule found
    ACTION: Return user-facing error

STATE: Processing → DefaultFallback
    TRIGGER: No matching rule found
    ACTION: Return generic error message

STATE: Mapped → GracefulDegradation
    TRIGGER: ShouldDegradeGracefully returns true
    ACTION: Set feature_flag, return degraded experience message

STATE: GracefulDegradation → Recovered
    TRIGGER: Error resolved on subsequent call
    ACTION: Clear feature_flag, restore full functionality
```

### 3.3 Error Logging

```go
func (m *ErrorMessageMapper) logErrorMapping(internalErr InternalError, userErr UserFacingError) {
    if !m.config.EnableDetailedLogging {
        return
    }

    logEntry := map[string]interface{}{
        "timestamp":      time.Now().UTC(),
        "internal_code":  internalErr.Code,
        "user_category":  userErr.Category,
        "user_severity":  userErr.Severity,
        "retryable":      userErr.Retryable,
        "feature":        internalErr.Context.FeatureName,
        "user_id":        internalErr.Context.UserID,
        "request_path":   internalErr.Context.RequestPath,
        "mapped_success": true,
    }

    if userErr.TechnicalRef != "" {
        logEntry["technical_ref"] = userErr.TechnicalRef
    }

    m.logger.Log("error_mapped", logEntry)
}
```

## 4. Component Interfaces

### 4.1 ErrorMessageMapper Interface

```go
type ErrorMessageMapper interface {
    MapError(err error, context ErrorContext) UserFacingError
    MapErrorCode(code string, context ErrorContext) UserFacingError
    RegisterRule(rule ErrorMappingRule) error
    SetLanguage(lang string) error
    GetSupportedLanguages() []string
    ShouldGracefullyDegrade(category ErrorCategory, context ErrorContext) bool
    CalculateRetryDelay(retryCount int) time.Duration
    Close()
}
```

### 4.2 Constructor

```go
func NewErrorMessageMapper(config ErrorMappingConfig, logger Logger) (*ErrorMessageMapper, error) {
    if config.DefaultLanguage == "" {
        config.DefaultLanguage = "en"
    }

    if len(config.SupportedLanguages) == 0 {
        config.SupportedLanguages = []string{"en", "es", "fr"}
    }

    mapper := &ErrorMessageMapper{
        config:       config,
        logger:       logger,
        language:     config.DefaultLanguage,
        mappingRules: initializeDefaultRules(),
        messageStore: NewMessageStore(config.SupportedLanguages),
    }

    return mapper, nil
}
```

### 4.3 Public Methods

```go
func (m *ErrorMessageMapper) MapError(err error, context ErrorContext) UserFacingError {
    startTime := time.Now()

    internalErr := m.extractInternalError(err)
    context.Timestamp = startTime

    userErr := m.findAndApplyRule(internalErr, context)

    if userErr.Category == "" {
        userErr = m.createFallbackError(internalErr, context)
    }

    m.logErrorMapping(internalErr, userErr)
    return userErr
}

func (m *ErrorMessageMapper) MapErrorCode(code string, context ErrorContext) UserFacingError {
    mockErr := InternalError{
        Code:    code,
        Message: code,
        Context: context,
    }

    return m.findAndApplyRule(mockErr, context)
}

func (m *ErrorMessageMapper) RegisterRule(rule ErrorMappingRule) error {
    if rule.MatchCondition == nil {
        return errors.New("rule must have a MatchCondition function")
    }

    if rule.MessageKey == "" && rule.DefaultMessage == "" {
        return errors.New("rule must have either MessageKey or DefaultMessage")
    }

    m.mappingRules = append(m.mappingRules, rule)
    return nil
}

func (m *ErrorMessageMapper) SetLanguage(lang string) error {
    for _, supported := range m.config.SupportedLanguages {
        if supported == lang {
            m.language = lang
            return nil
        }
    }

    return fmt.Errorf("language %s not supported. Supported: %v", lang, m.config.SupportedLanguages)
}

func (m *ErrorMessageMapper) GetSupportedLanguages() []string {
    return m.config.SupportedLanguages
}

func (m *ErrorMessageMapper) ShouldGracefullyDegrade(category ErrorCategory, context ErrorContext) bool {
    return ShouldDegradeGracefully(category, context)
}

func (m *ErrorMessageMapper) CalculateRetryDelay(retryCount int) time.Duration {
    delay := CalculateRetryAfter(InternalError{}, ErrorContext{RetryCount: retryCount})
    return *delay
}
```

### 4.4 Internal Methods

```go
func (m *ErrorMessageMapper) extractInternalError(err error) InternalError {
    if err == nil {
        return InternalError{
            Code:    "UNKNOWN",
            Message: "Unknown error",
            Context: ErrorContext{},
        }
    }

    if internalErr, ok := err.(InternalError); ok {
        return internalErr
    }

    if fiberErr, ok := err.(*fiber.Error); ok {
        return InternalError{
            Code:    fmt.Sprintf("FIBER_%d", fiberErr.Code),
            Message: fiberErr.Message,
            Context: ErrorContext{},
        }
    }

    if errors.Is(err, context.DeadlineExceeded) {
        return InternalError{
            Code:    "TIMEOUT",
            Message: "Request timeout",
            Context: ErrorContext{},
        }
    }

    if errors.Is(err, sql.ErrNoRows) {
        return InternalError{
            Code:    "NOT_FOUND",
            Message: "Resource not found",
            Context: ErrorContext{},
        }
    }

    return InternalError{
        Code:    "INTERNAL_ERROR",
        Message: err.Error(),
        Context: ErrorContext{},
    }
}

func (m *ErrorMessageMapper) findAndApplyRule(err InternalError, context ErrorContext) UserFacingError {
    for _, rule := range m.mappingRules {
        if EvaluateMatchCondition(rule, err, context) {
            return m.applyRule(rule, err, context)
        }
    }

    return UserFacingError{}
}

func (m *ErrorMessageMapper) applyRule(rule ErrorMappingRule, err InternalError, context ErrorContext) UserFacingError {
    message := ResolveMessage(rule, err, context)

    retryable := rule.Retryable
    if rule.Category == ErrorCategoryInternal || rule.Category == ErrorCategoryNetwork {
        retryable = true
    }

    var retryAfter *time.Duration
    if retryable {
        delay := CalculateRetryAfter(err, context)
        retryAfter = &delay
    }

    return UserFacingError{
        Message:      message,
        Category:     rule.Category,
        Severity:     rule.Severity,
        Retryable:    retryable,
        RetryAfter:   retryAfter,
        ActionLabel:  m.getActionLabel(rule.Category),
        TechnicalRef: err.Code,
    }
}

func (m *ErrorMessageMapper) createFallbackError(err InternalError, context ErrorContext) UserFacingError {
    message := "An unexpected error occurred. Please try again later."

    if context.FeatureName != "" {
        message = fmt.Sprintf("Something went wrong with %s. Please try again.", context.FeatureName)
    }

    return UserFacingError{
        Message:    message,
        Category:   ErrorCategoryInternal,
        Severity:   ErrorSeverityMedium,
        Retryable:  true,
        ActionLabel: "Try Again",
    }
}

func (m *ErrorMessageMapper) getActionLabel(category ErrorCategory) string {
    actionLabels := map[ErrorCategory]string{
        ErrorCategoryNetwork:         "Retry Connection",
        ErrorCategoryTimeout:         "Try Again",
        ErrorCategoryValidation:      "Check Input",
        ErrorCategoryAuthentication:  "Sign In",
        ErrorCategoryAuthorization:   "Request Access",
        ErrorCategoryNotFound:        "Search Again",
        ErrorCategoryConflict:        "Review Changes",
        ErrorCategoryRateLimit:       "Wait",
        ErrorCategoryInternal:        "Try Again",
        ErrorCategoryExternalService: "Retry",
    }

    if label, ok := actionLabels[category]; ok {
        return label
    }

    return "Retry"
}
```

### 4.5 Default Mapping Rules

```go
func initializeDefaultRules() []ErrorMappingRule {
    return []ErrorMappingRule{
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && strings.Contains(internalErr.Code, "ECONNREFUSED")
            },
            Category:       ErrorCategoryNetwork,
            Severity:       ErrorSeverityHigh,
            MessageKey:     "error.network.connection_refused",
            Retryable:      true,
            DefaultMessage: "Unable to connect. Please check your internet connection.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && internalErr.Code == "TIMEOUT"
            },
            Category:       ErrorCategoryTimeout,
            Severity:       ErrorSeverityMedium,
            MessageKey:     "error.timeout",
            Retryable:      true,
            DefaultMessage: "This is taking longer than expected. Would you like to wait or try again?",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && internalErr.Code == "FIBER_401"
            },
            Category:       ErrorCategoryAuthentication,
            Severity:       ErrorSeverityHigh,
            MessageKey:     "error.auth.required",
            Retryable:      false,
            DefaultMessage: "Please sign in to continue.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && internalErr.Code == "FIBER_403"
            },
            Category:       ErrorCategoryAuthorization,
            Severity:       ErrorSeverityHigh,
            MessageKey:     "error.forbidden",
            Retryable:      false,
            DefaultMessage: "You don't have permission to access this resource.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && internalErr.Code == "FIBER_404"
            },
            Category:       ErrorCategoryNotFound,
            Severity:       ErrorSeverityMedium,
            MessageKey:     "error.not_found",
            Retryable:      true,
            DefaultMessage: "The requested resource could not be found.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && internalErr.Code == "FIBER_429"
            },
            Category:       ErrorCategoryRateLimit,
            Severity:       ErrorSeverityMedium,
            MessageKey:     "error.rate_limit",
            Retryable:      true,
            DefaultMessage: "You've made too many requests. Please wait a moment.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && internalErr.Code == "FIBER_500"
            },
            Category:       ErrorCategoryInternal,
            Severity:       ErrorSeverityHigh,
            MessageKey:     "error.internal",
            Retryable:      true,
            DefaultMessage: "Something went wrong on our end. Please try again later.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && strings.Contains(internalErr.Code, "STRIPE")
            },
            Category:       ErrorCategoryExternalService,
            Severity:       ErrorSeverityHigh,
            MessageKey:     "error.payment.service_error",
            Retryable:      true,
            DefaultMessage: "Payment service is temporarily unavailable. Please try again.",
        },
        {
            MatchCondition: func(err error, ctx ErrorContext) bool {
                internalErr, ok := err.(InternalError)
                return ok && strings.Contains(internalErr.Code, "USDA")
            },
            Category:       ErrorCategoryExternalService,
            Severity:       ErrorSeverityLow,
            MessageKey:     "error.food_data.unavailable",
            Retryable:      true,
            DefaultMessage: "Food data service is temporarily unavailable. Some results may be incomplete.",
        },
    }
}
```

### 4.6 Message Store Interface

```go
type MessageStore interface {
    GetMessage(key string, language string) string
    LoadMessages(language string, reader io.Reader) error
    HasMessage(key string, language string) bool
    GetAllKeys(language string) []string
}
```

### 4.7 Integration Points

```go
// Fiber middleware integration
func ErrorMapperMiddleware(mapper *ErrorMessageMapper) fiber.Handler {
    return func(c *fiber.Ctx) error {
        err := c.Next()

        if err != nil {
            context := ErrorContext{
                UserID:        c.Locals("user_id").(string),
                RequestPath:   c.Path(),
                RequestMethod: c.Method(),
                FeatureName:   c.Locals("feature_name").(string),
                Timestamp:     time.Now(),
            }

            userErr := mapper.MapError(err, context)

            c.Status(mapErrorCodeToHTTPStatus(userErr.Category))
            return c.JSON(userErr)
        }

        return nil
    }
}

// Svelte store integration
type ErrorStore struct {
    currentError UserFacingError
    subscribers  []chan UserFacingError
}

func NewErrorStore() *ErrorStore {
    store := &ErrorStore{
        currentError: UserFacingError{},
        subscribers:  make([]chan UserFacingError, 0),
    }

    go store.broadcastErrors()
    return store
}

func (s *ErrorStore) SetError(err UserFacingError) {
    s.currentError = err
    for _, ch := range s.subscribers {
        ch <- err
    }
}

func (s *ErrorStore) ClearError() {
    s.currentError = UserFacingError{}
}
```

# AuditLogger

**Traceability:** ARCH-013

## 1. Data Structures & Types

```go
package audit

type EventType string

const (
    EventTypeAuthentication EventType = "authentication"
    EventTypeAPIRequest     EventType = "api_request"
    EventTypeError          EventType = "error"
    EventTypeAdminAction    EventType = "admin_action"
)

type EventSeverity string

const (
    SeverityInfo     EventSeverity = "info"
    SeverityWarning  EventSeverity = "warning"
    SeverityError    EventSeverity = "error"
    SeverityCritical EventSeverity = "critical"
)

type AuditEvent struct {
    ID          string         `json:"id"`
    Timestamp   time.Time      `json:"timestamp"`
    EventType   EventType      `json:"event_type"`
    Severity    EventSeverity  `json:"severity"`
    UserID      string         `json:"user_id,omitempty"`
    IPAddress   string         `json:"ip_address"`
    UserAgent   string         `json:"user_agent,omitempty"`
    RequestPath string         `json:"request_path,omitempty"`
    RequestMethod string       `json:"request_method,omitempty"`
    Action      string         `json:"action"`
    Resource    string         `json:"resource,omitempty"`
    Details     map[string]any `json:"details,omitempty"`
    StatusCode  int            `json:"status_code,omitempty"`
    ErrorMsg    string         `json:"error_msg,omitempty"`
}

type Logger interface {
    Log(event AuditEvent) error
    LogAuthentication(userID string, success bool, reason string, c *fiber.Ctx) error
    LogAPIRequest(c *fiber.Ctx, statusCode int, duration time.Duration) error
    LogError(err error, c *fiber.Ctx, context string) error
    LogAdminAction(userID string, action string, resource string, details map[string]any) error
}

type FileLogger struct {
    file    *os.File
    mu      sync.Mutex
    encoder *json.Encoder
}

type DatabaseLogger struct {
    db          *gorm.DB
    tableName   string
    buffer      chan AuditEvent
    flushTicker *time.Ticker
    batchSize   int
    maxBuffer   int
}

type CompositeLogger struct {
    loggers []Logger
}

type LoggerConfig struct {
    OutputPath       string
    DatabaseDSN      string
    EnableConsole    bool
    EnableFile       bool
    EnableDatabase   bool
    LogLevel         EventSeverity
    BufferSize       int
    FlushInterval    time.Duration
    RotateInterval   time.Duration
    MaxFileSizeMB    int
    RetentionDays    int
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Log Event Flow

```
1. Receive AuditEvent with all required fields populated
2. Set Timestamp to current time if not already set
3. Generate unique ID (UUID v4) if not already set
4. Validate required fields:
   - Timestamp must be set
   - EventType must be valid
   - Severity must be valid
   - IPAddress must be valid format
   - Action must be non-empty
5. Enrich event with additional context from Fiber context if available
6. Dispatch to all configured logger backends
7. Handle success/failure for each backend independently
8. Return combined error if any backend fails
```

### 2.2 Log Authentication Event

```
1. Create AuditEvent with:
   - EventType = "authentication"
   - Severity = "error" if !success, otherwise "info"
   - UserID = provided userID
   - Action = "login_attempt"
   - Details = {"success": success, "reason": reason}
2. Extract IPAddress and UserAgent from Fiber context
3. Call Log(event)
```

### 2.3 Log API Request Event

```
1. Create AuditEvent with:
   - EventType = "api_request"
   - Severity = "error" if statusCode >= 400, otherwise "info"
   - RequestPath = c.Path()
   - RequestMethod = c.Method()
   - StatusCode = statusCode
   - Duration = duration
2. Extract UserID from JWT claims if authenticated
3. Extract IPAddress and UserAgent from Fiber context
4. Call Log(event)
```

### 2.4 Log Error Event

```
1. Create AuditEvent with:
   - EventType = "error"
   - Severity = "error" or "critical" based on error type
   - ErrorMsg = err.Error()
   - Action = context string
   - Details = stack trace if available
2. Extract UserID and IPAddress from Fiber context if available
3. Call Log(event)
```

### 2.5 Log Admin Action Event

```
1. Create AuditEvent with:
   - EventType = "admin_action"
   - Severity = "warning"
   - UserID = provided userID
   - Action = provided action
   - Resource = provided resource
   - Details = provided details map
2. Set timestamp to current time
3. Call Log(event)
```

### 2.6 File Rotation Algorithm

```
1. Check file size on each write
2. If size > MaxFileSizeMB:
   - Close current file
   - Rename current file to include timestamp
   - Create new file with original path
3. Also rotate on schedule based on RotateInterval
4. Delete files older than RetentionDays
```

### 2.7 Database Buffer Flush

```
1. Events are buffered in channel up to BufferSize
2. Background goroutine flushes every FlushInterval
3. On flush:
   - Drain buffer up to batchSize
   - Batch insert events to database
   - Clear buffer
4. If buffer full, block or drop based on configuration
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error State | Cause | Recovery |
|------------|-------|----------|
| FilePermissionDenied | Insufficient permissions to write file | Log to console, attempt retry, alert admin |
| FileRotationFailed | Disk full or I/O error during rotation | Continue logging to existing file, alert admin |
| DatabaseConnectionLost | Network issue or DB restart | Retry with exponential backoff, buffer events |
| DatabaseWriteFailed | Constraint violation or timeout | Retry once, if fails log to alternative backend |
| BufferOverflow | Events arrive faster than flush rate | Block new events or drop oldest (configurable) |
| JSONEncodeFailed | Invalid event structure | Log error to console, skip event |
| IPAddressInvalid | Malformed IP in request | Use placeholder, log warning |

### 3.2 State Transitions

```
INITIALIZING -> READY:
    - All configured backends initialized successfully

READY -> ERROR:
    - Any write operation fails

ERROR -> READY:
    - Retry successful after transient failure
    - Database: connection restored
    - File: permission issue resolved

READY -> SHUTDOWN:
    - Flush all buffered events
    - Close all file handles
    - Graceful shutdown with timeout
```

### 3.3 Retry Logic

```
For transient errors (network, timeout):
1. Wait 100ms
2. Retry up to 3 times
3. Exponential backoff: 100ms, 500ms, 1s
4. If all retries fail:
   - Log to console as fallback
   - Return error with context
```

## 4. Component Interfaces

### 4.1 Logger Interface

```go
type Logger interface {
    Log(event AuditEvent) error
    LogAuthentication(userID string, success bool, reason string, c *fiber.Ctx) error
    LogAPIRequest(c *fiber.Ctx, statusCode int, duration time.Duration) error
    LogError(err error, c *fiber.Ctx, context string) error
    LogAdminAction(userID string, action string, resource string, details map[string]any) error
}
```

### 4.2 FileLogger Methods

```go
func NewFileLogger(path string, config LoggerConfig) (*FileLogger, error)
func (l *FileLogger) Log(event AuditEvent) error
func (l *FileLogger) Close() error
func (l *FileLogger) Rotate() error
```

### 4.3 DatabaseLogger Methods

```go
func NewDatabaseLogger(dsn string, config LoggerConfig) (*DatabaseLogger, error)
func (l *DatabaseLogger) Log(event AuditEvent) error
func (l *DatabaseLogger) Close() error
func (l *DatabaseLogger) Flush() error
```

### 4.4 CompositeLogger Methods

```go
func NewCompositeLogger(loggers []Logger) *CompositeLogger
func (l *CompositeLogger) Log(event AuditEvent) error
func (l *CompositeLogger) AddLogger(logger Logger)
func (l *CompositeLogger) RemoveLogger(logger Logger)
```

### 4.5 Fiber Middleware

```go
func AuditLoggerMiddleware(logger Logger, skipPaths []string) fiber.Handler
```

Implementation:
```go
func AuditLoggerMiddleware(logger Logger, skipPaths []string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()
        duration := time.Since(start)
        
        skip := false
        for _, path := range skipPaths {
            if c.Path() == path {
                skip = true
                break
            }
        }
        
        if !skip {
            logger.LogAPIRequest(c, c.Response.StatusCode(), duration)
        }
        
        return err
    }
}
```

### 4.6 Initialization Helper

```go
func NewAuditLogger(config LoggerConfig) (Logger, error)
```

Implementation:
```go
func NewAuditLogger(config LoggerConfig) (Logger, error) {
    var loggers []Logger
    
    if config.EnableConsole {
        loggers = append(loggers, NewConsoleLogger())
    }
    
    if config.EnableFile {
        fileLogger, err := NewFileLogger(config.OutputPath, config)
        if err != nil {
            return nil, fmt.Errorf("failed to create file logger: %w", err)
        }
        loggers = append(loggers, fileLogger)
    }
    
    if config.EnableDatabase {
        dbLogger, err := NewDatabaseLogger(config.DatabaseDSN, config)
        if err != nil {
            return nil, fmt.Errorf("failed to create database logger: %w", err)
        }
        loggers = append(loggers, dbLogger)
    }
    
    if len(loggers) == 0 {
        return nil, fmt.Errorf("no logger backends enabled")
    }
    
    return NewCompositeLogger(loggers), nil
}
```

# FILE: FiberLogger (Fiber Logger Middleware)

**Traceability:** ARCH-014

## 1. Data Structures & Types

### Configuration Struct

```go
type LoggerConfig struct {
    // Output configuration
    Output io.Writer

    // Format configuration
    Format string // "json" or "text"

    // Time format for timestamp (Go's time format string)
    TimeFormat string

    // Custom time zone for timestamps
    TimeZone *time.Location

    // Enable colors for console output (text format only)
    EnableColors bool

    // Enable UTC timestamps
    EnableUTC bool

    // GCP Cloud Monitoring configuration
    GCPConfig *GCPConfig

    // Skip successful health check logs
    SkipHealthCheckLogs bool

    // Skip successful uptime check logs
    SkipUptimeCheckLogs bool
}
```

### GCP Configuration Struct

```go
type GCPConfig struct {
    // GCP project ID
    ProjectID string

    // Log name prefix for log entries
    LogNamePrefix string

    // Enable structured logging for GCP
    EnableStructuredLogging bool

    // Batch size for log upload
    BatchSize int

    // Flush interval for batched logs
    FlushInterval time.Duration

    // Context timeout for GCP operations
    Timeout time.Duration
}
```

### Log Entry Structure

```go
type LogEntry struct {
    // Timestamp in RFC3339 format
    Timestamp string `json:"timestamp"`

    // Log level: debug, info, warning, error, critical
    Level string `json:"level"`

    // Request ID for correlation
    RequestID string `json:"request_id,omitempty"`

    // HTTP method
    Method string `json:"method,omitempty"`

    // Request path
    Path string `json:"path,omitempty"`

    // Response status code
    StatusCode int `json:"status_code,omitempty"`

    // Response latency in milliseconds
    Latency int `json:"latency_ms,omitempty"`

    // Client IP address
    ClientIP string `json:"client_ip,omitempty"`

    // User agent
    UserAgent string `json:"user_agent,omitempty"`

    // Error message if applicable
    Error string `json:"error,omitempty"`

    // Custom fields for structured logging
    Fields map[string]interface{} `json:"fields,omitempty"`

    // Message content
    Message string `json:"message"`
}
```

### Log Level Constants

```go
const (
    LogLevelDebug   = "debug"
    LogLevelInfo    = "info"
    LogLevelWarning = "warning"
    LogLevelError   = "error"
    LogLevelCritical = "critical"
)
```

## 2. Logic & Algorithms

### Initialization Algorithm

```
1. Load LoggerConfig with defaults
   - Set Output to os.Stdout
   - Set Format to "json"
   - Set TimeFormat to "2006-01-02 15:04:05"
   - Set EnableColors to true
   - Set BatchSize to 100
   - Set FlushInterval to 5 seconds

2. If GCPConfig is provided:
   - Initialize GCP Cloud Logging client
   - Validate ProjectID is not empty
   - Create log writer buffer for batch processing
   - Start background goroutine for batch flushing

3. Register middleware with Fiber app
   - Call fiberApp.Use(Logger())
```

### Middleware Execution Flow

```
1. Receive Fiber context (c *fiber.Ctx)

2. Capture request start time
   startTime := time.Now()

3. Generate or extract request ID
   requestID := c.Get("X-Request-ID")
   if requestID is empty:
       requestID = generateUUID()

4. Store request ID in context
   c.Locals("request_id", requestID)

5. Set request ID in response header
   c.Set("X-Request-ID", requestID)

6. Execute next handler in chain
   err := c.Next()

7. Capture response details after handler execution
   - statusCode := c.Response.StatusCode()
   - latency := time.Since(startTime)
   - method := c.Method()
   - path := c.Path()
   - clientIP := c.IP()
   - userAgent := c.Get("User-Agent")

8. Determine log level based on status code
   if statusCode >= 500:
       level = LogLevelCritical
   else if statusCode >= 400:
       level = LogLevelError
   else if statusCode >= 300:
       level = LogLevelWarning
   else:
       level = LogLevelInfo

9. Build LogEntry with captured data

10. Apply skip filters
    if SkipHealthCheckLogs AND path == "/health":
        return nil
    if SkipUptimeCheckLogs AND path == "/uptime":
        return nil

11. Format log entry
    if Format == "json":
        entryJSON, _ := json.Marshal(logEntry)
        write to output
    else:
        format as: "[timestamp] level: message | method: value | path: value | status: value | latency: valuems"
        write to output

12. If GCPConfig is enabled:
    - Add entry to batch buffer
    - If batch reaches BatchSize:
        flushBatch()
    - Start timer for periodic flush

13. Return to Fiber handler chain
```

### GCP Batch Flush Algorithm

```
flushBatch():
1. Lock batch buffer mutex
2. Take all entries from buffer (up to BatchSize)
3. Unlock mutex

4. For each entry:
   - Convert to GCP LogEntry format
   - Add resource labels (service name, environment)
   - Add trace span ID for correlation

5. Write batch to GCP Cloud Logging
   - Use WriteLogEntries API
   - Set log name: projects/{ProjectID}/logs/{LogNamePrefix}
   - Set resource type: "gae_app" or "k8s_container"
   - Include request ID for cross-service correlation

6. If write succeeds:
   - Clear flushed entries
   - Reset flush timer

7. If write fails:
   - Log error to stderr
   - Keep entries in buffer for retry
   - Trigger retry after backoff delay
```

### Error Message Extraction Algorithm

```
extractErrorMessage(err error) string:
1. If err is nil:
   return ""

2. If err implements Unwrap() []error:
   recursively unwrap and find deepest error

3. For Fiber errors:
   if err is *fiber.Error:
       return err.Message

4. For standard errors:
   return err.Error()

5. Sanitize message:
   - Remove newlines
   - Escape quotes
   - Limit to 1000 characters
```

## 3. State Management & Error Handling

### Possible Error States

| Error State | Cause | Transition |
|------------|-------|------------|
| GCP Connection Timeout | Network partition, invalid credentials | Retry with exponential backoff; fallback to local logging |
| GCP Write Quota Exceeded | Rate limiting, too many logs | Buffer entries; apply sampling if buffer full |
| Invalid Log Format | Malformed structured data | Skip invalid field; log warning with field name |
| Buffer Overflow | Batched logs exceed memory limit | Drop oldest entries; log warning |
| Context Deadline Exceeded | Slow GCP API response | Return cached entries for this request; log error |
| JSON Marshal Failure | Invalid log entry structure | Write raw entry to stderr; continue |
| Color Output Error | Non-TTY terminal | Disable colors; continue with text format |

### State Transitions

```
State: Initial
  -> Config validation passed -> State: Ready
  -> Config validation failed -> State: Error (log to stderr, use defaults)

State: Ready
  -> Middleware registered -> State: Active
  -> GCP client initialized -> State: GCPEnabled

State: Active
  -> Log entry created -> State: Processing
  -> Batch buffer full -> State: Flushing
  -> Handler error -> State: LoggingError (log error, continue)

State: Flushing
  -> Flush successful -> State: Active
  -> Flush failed -> State: RetryPending (schedule retry)

State: RetryPending
  -> Retry successful -> State: Active
  -> Max retries exceeded -> State: DropEntries (log warning, drop entries)
```

### Error Handling Strategy

```
For each log operation:
1. Attempt primary logging method
2. If error occurs:
   - Log error to stderr with context
   - Attempt fallback logging (local file)
   - If fallback also fails:
     - Drop current entry
     - Increment error counter
     - Emit metrics for monitoring

For GCP operations:
1. Use context with timeout
2. Implement retry with backoff (3 attempts)
3. Backoff delays: 100ms, 500ms, 2s
4. On final failure:
   - Write to local buffer file
   - Trigger alert for operator

For panic recovery:
1. Use defer recover() in goroutine
2. If panic during logging:
   - Write to os.Stderr
   - Prevent crash of main application
   - Increment panic counter
```

## 4. Component Interfaces

### Public Functions

```go
// New creates a new Logger middleware with the provided configuration
func New(config LoggerConfig) fiber.Handler

// NewWithGCP creates a Logger middleware with GCP Cloud Monitoring integration
func NewWithGCP(gcpConfig GCPConfig) fiber.Handler

// Default returns a Logger middleware with sensible default configuration
func Default() fiber.Handler
```

### LoggerConfig Methods

```go
// WithOutput sets the output writer for log entries
func (c *LoggerConfig) WithOutput(output io.Writer) *LoggerConfig

// WithFormat sets the log format (json or text)
func (c *LoggerConfig) WithFormat(format string) *LoggerConfig

// WithTimeFormat sets the timestamp format
func (c *LoggerConfig) WithTimeFormat(format string) *LoggerConfig

// WithTimeZone sets the timezone for timestamps
func (c *LoggerConfig) WithTimeZone(tz *time.Location) *LoggerConfig

// WithColors enables or disables colored console output
func (c *LoggerConfig) WithColors(enable bool) *LoggerConfig

// WithUTC enables UTC timestamps
func (c *LoggerConfig) WithUTC(enable bool) *LoggerConfig

// WithSkipHealthCheckLogs skips logging for health check endpoints
func (c *LoggerConfig) WithSkipHealthCheckLogs(skip bool) *LoggerConfig

// WithSkipUptimeCheckLogs skips logging for uptime check endpoints
func (c *LoggerConfig) WithSkipUptimeCheckLogs(skip bool) *LoggerConfig

// WithGCPConfig sets the GCP Cloud Monitoring configuration
func (c *LoggerConfig) WithGCPConfig(config GCPConfig) *LoggerConfig
```

### GCP Logging Interface

```go
// GCPWriter handles writing logs to Google Cloud Platform Cloud Monitoring
type GCPWriter interface {
    // WriteLogEntry sends a single log entry to GCP
    WriteLogEntry(ctx context.Context, entry LogEntry) error

    // WriteLogBatch sends multiple log entries to GCP
    WriteLogBatch(ctx context.Context, entries []LogEntry) error

    // Flush ensures all buffered logs are sent
    Flush(ctx context.Context) error

    // Close releases resources and flushes remaining logs
    Close() error

    // Health checks if the GCP connection is healthy
    Health(ctx context.Context) error
}
```

### Log Entry Builder

```go
// LogEntryBuilder provides a fluent interface for constructing LogEntry
type LogEntryBuilder struct {
    entry LogEntry
}

// NewLogEntryBuilder creates a new builder with required fields
func NewLogEntryBuilder(level string, message string) *LogEntryBuilder

// WithRequestID sets the request correlation ID
func (b *LogEntryBuilder) WithRequestID(id string) *LogEntryBuilder

// WithHTTPDetails adds HTTP request/response details
func (b *LogEntryBuilder) WithHTTPDetails(method string, path string, statusCode int) *LogEntryBuilder

// WithLatency sets the response latency in milliseconds
func (b *LogEntryBuilder) WithLatency(ms int) *LogEntryBuilder

// WithClientIP sets the client IP address
func (b *LogEntryBuilder) WithClientIP(ip string) *LogEntryBuilder

// WithUserAgent sets the user agent string
func (b *LogEntryBuilder) WithUserAgent(ua string) *LogEntryBuilder

// WithError adds error information
func (b *LogEntryBuilder) WithError(err error) *LogEntryBuilder

// WithField adds a custom field for structured logging
func (b *LogEntryBuilder) WithField(key string, value interface{}) *LogEntryBuilder

// WithFields adds multiple custom fields
func (b *LogEntryBuilder) WithFields(fields map[string]interface{}) *LogEntryBuilder

// Build returns the final LogEntry
func (b *LogEntryBuilder) Build() LogEntry
```

### Utility Functions

```go
// ExtractRequestID extracts request ID from Fiber context
// Returns the ID from header, locals, or generates a new one
func ExtractRequestID(c *fiber.Ctx) string

// LogLevelFromStatus returns the appropriate log level for HTTP status code
func LogLevelFromStatus(statusCode int) string

// FormatLatency formats duration as milliseconds string
func FormatLatency(d time.Duration) string

// SanitizeLogMessage removes sensitive data and normalizes the message
func SanitizeLogMessage(msg string) string
```

### Metrics Methods

```go
// Metrics provides access to logging metrics for monitoring
type Metrics struct {
    // TotalLogEntries returns the count of all logged entries
    TotalLogEntries() int64

    // LogEntriesByLevel returns a map of level to count
    LogEntriesByLevel() map[string]int64

    // FailedGCPWrites returns the count of failed GCP write operations
    FailedGCPWrites() int64

    // AverageFlushDuration returns the average time to flush logs to GCP
    AverageFlushDuration() time.Duration

    // Reset clears all metrics counters
    Reset()
}

// GetMetrics returns the global Metrics instance
func GetMetrics() Metrics
```

### Configuration Default Values

```go
var DefaultLoggerConfig = LoggerConfig{
    Output:           os.Stdout,
    Format:           "json",
    TimeFormat:       "2006-01-02 15:04:05",
    TimeZone:         time.Local,
    EnableColors:     true,
    EnableUTC:        false,
    GCPConfig:        nil,
    SkipHealthCheckLogs: true,
    SkipUptimeCheckLogs: true,
}

var DefaultGCPConfig = GCPConfig{
    ProjectID:              "",
    LogNamePrefix:          "mealswapp",
    EnableStructuredLogging: true,
    BatchSize:              100,
    FlushInterval:          5 * time.Second,
    Timeout:                30 * time.Second,
}
```

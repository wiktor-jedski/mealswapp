## FILE: LogAggregator.md
**Traceability:** ARCH-014

### 1. Data Structures & Types

```go
package logging

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LogLevel represents the severity level of a log entry.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelCritical
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	case LogLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry collected from services.
type LogEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	ServiceName string                 `json:"service_name"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	Method      string                 `json:"method,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Duration    int64                  `json:"duration_ms,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// LogAggregatorConfig holds configuration for the LogAggregator.
type LogAggregatorConfig struct {
	BufferSize      int           `env:"LOG_BUFFER_SIZE" default:"1000"`
	FlushInterval   time.Duration `env:"LOG_FLUSH_INTERVAL" default:"5s"`
	RetentionDays   int           `env:"LOG_RETENTION_DAYS" default:"90"`
	GCPProjectID    string        `env:"GCP_PROJECT_ID"`
	EnableLocalDump bool          `env:"LOG_ENABLE_LOCAL_DUMP" default:"false"`
	LocalDumpPath   string        `env:"LOG_LOCAL_DUMP_PATH" default:"./logs"`
}

// MetricsEntry represents a metrics data point for monitoring.
type MetricsEntry struct {
	Timestamp      time.Time            `json:"timestamp"`
	ServiceName    string               `json:"service_name"`
	MetricType     MetricType           `json:"metric_type"`
	Value          float64              `json:"value"`
	Labels         map[string]string    `json:"labels,omitempty"`
	Percentile     float64              `json:"percentile,omitempty"`
}

// MetricType represents the type of metrics being collected.
type MetricType int

const (
	MetricTypeResponseTime MetricType = iota
	MetricTypeErrorRate
	MetricTypeConcurrentUsers
	MetricTypeRequestCount
	MetricTypeCPUUsage
	MetricTypeMemoryUsage
)

// AggregationQuery represents a query for log aggregation.
type AggregationQuery struct {
	StartTime   time.Time
	EndTime     time.Time
	ServiceName string
	LogLevel    LogLevel
	TraceID     string
	UserID      string
	SearchText  string
	Limit       int
	Offset      int
}

// AggregationResult represents the result of a log aggregation query.
type AggregationResult struct {
	TotalCount  int64       `json:"total_count"`
	Logs        []LogEntry  `json:"logs"`
	Facets      Facets      `json:"facets"`
}

// Facets contains aggregated counts for filtering.
type Facets struct {
	ByLevel    map[string]int64 `json:"by_level"`
	ByService  map[string]int64 `json:"by_service"`
	ByEndpoint map[string]int64 `json:"by_endpoint"`
}
```

### 2. LogAggregator Service

```go
package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// LogAggregator handles centralized log collection and aggregation.
type LogAggregator struct {
	client    *logging.Client
	logger    *logging.Logger
	redis     *redis.Client
	config    LogAggregatorConfig
	buffer    []LogEntry
	bufferMu  sync.Mutex
	flushChan chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewLogAggregator creates a new LogAggregator instance.
func NewLogAggregator(ctx context.Context, config LogAggregatorConfig, redisClient *redis.Client) (*LogAggregator, error) {
	if config.GCPProjectID != "" {
		client, err := logging.NewClient(ctx, fmt.Sprintf("projects/%s", config.GCPProjectID))
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP logging client: %w", err)
		}
		return &LogAggregator{
			client:    client,
			logger:    client.Logger("mealswapp-logs"),
			redis:     redisClient,
			config:    config,
			buffer:    make([]LogEntry, 0, config.BufferSize),
			flushChan: make(chan struct{}, 1),
			ctx:       ctx,
		}, nil
	}
	return &LogAggregator{
		redis:     redisClient,
		config:    config,
		buffer:    make([]LogEntry, 0, config.BufferSize),
		flushChan: make(chan struct{}, 1),
		ctx:       ctx,
	}, nil
}

// Start initializes the LogAggregator and starts background workers.
func (la *LogAggregator) Start() {
	ctx, cancel := context.WithCancel(la.ctx)
	la.ctx = ctx
	la.cancel = cancel

	la.wg.Add(1)
	go la.flushWorker(ctx)

	la.wg.Add(1)
	go la.retentionWorker(ctx)
}

// Stop gracefully shuts down the LogAggregator.
func (la *LogAggregator) Stop() error {
	if la.cancel != nil {
		la.cancel()
	}
	la.wg.Wait()

	la.bufferMu.Lock()
	defer la.bufferMu.Unlock()

	if err := la.flushBuffer(); err != nil {
		return fmt.Errorf("failed to flush buffer on shutdown: %w", err)
	}

	if la.client != nil {
		return la.client.Close()
	}
	return nil
}

// SubmitLog adds a log entry to the aggregation buffer.
func (la *LogAggregator) SubmitLog(entry LogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	la.bufferMu.Lock()
	defer la.bufferMu.Unlock()

	la.buffer = append(la.buffer, entry)

	if len(la.buffer) >= la.config.BufferSize {
		select {
		case la.flushChan <- struct{}{}:
		default:
		}
	}

	return nil
}

// SubmitLogFromFiberCtx creates a LogEntry from a Fiber context and submits it.
func (la *LogAggregator) SubmitLogFromFiberCtx(c *fiber.Ctx, level LogLevel, message string, additionalFields map[string]interface{}) error {
	traceID := c.GetRespHeader("X-Trace-ID", "")
	if traceID == "" && c.Locals("trace_id") != nil {
		traceID = c.Locals("trace_id").(string)
	}

	entry := LogEntry{
		Timestamp:   time.Now().UTC(),
		ServiceName: "mealswapp-api",
		Level:       level,
		Message:     message,
		TraceID:     traceID,
		Endpoint:    c.Path(),
		Method:      c.Method(),
		StatusCode:  c.Response().StatusCode(),
		Duration:    int64(c.Response().Header.Time().Sub(c.Request().Header.Time()).Milliseconds()),
		Fields:      additionalFields,
	}

	if userID := c.Locals("user_id"); userID != nil {
		entry.UserID = userID.(string)
	}

	if level >= LogLevelError {
		entry.Error = string(c.Response().Body())
	}

	return la.SubmitLog(entry)
}

// flushWorker runs as a background worker to flush logs periodically.
func (la *LogAggregator) flushWorker(ctx context.Context) {
	defer la.wg.Done()

	ticker := time.NewTicker(la.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			la.flushBuffer()
		case <-la.flushChan:
			la.flushBuffer()
		}
	}
}

// flushBuffer sends buffered logs to GCP Cloud Monitoring.
func (la *LogAggregator) flushBuffer() error {
	la.bufferMu.Lock()
	if len(la.buffer) == 0 {
		la.bufferMu.Unlock()
		return nil
	}

	entries := make([]logging.Entry, len(la.buffer))
	for i, entry := range la.buffer {
		severity := la.mapLogLevelToSeverity(entry.Level)
		entries[i] = logging.Entry{
			Timestamp: entry.Timestamp,
			Severity:  severity,
			Payload:   entry,
			Labels: map[string]string{
				"service": entry.ServiceName,
				"trace":   entry.TraceID,
			},
		}
	}

	la.buffer = la.buffer[:0]
	la.bufferMu.Unlock()

	if la.logger != nil {
		if err := la.logger.LogSync(entries...); err != nil {
			if la.config.EnableLocalDump {
				return la.dumpToLocalFile(entries)
			}
			return fmt.Errorf("failed to flush logs to GCP: %w", err)
		}
	}

	if la.redis != nil {
		return la.cacheLogsInRedis(entries)
	}

	return nil
}

// mapLogLevelToSeverity converts LogLevel to GCP logging severity.
func (la *LogAggregator) mapLogLevelToSeverity(level LogLevel) logging.Severity {
	switch level {
	case LogLevelDebug:
		return logging.Debug
	case LogLevelInfo:
		return logging.Info
	case LogLevelWarning:
		return logging.Warning
	case LogLevelError:
		return logging.Error
	case LogLevelCritical:
		return logging.Critical
	default:
		return logging.Default
	}
}

// retentionWorker runs periodic cleanup of old logs.
func (la *LogAggregator) retentionWorker(ctx context.Context) {
	defer la.wg.Done()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			la.cleanupOldLogs()
		}
	}
}

// cleanupOldLogs removes logs older than the retention period.
func (la *LogAggregator) cleanupOldLogs() {
	if la.redis == nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -la.config.RetentionDays)
	pattern := fmt.Sprintf("logs:*")

	var cursor uint64
	for {
		keys, nextCursor, err := la.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}

		for _, key := range keys {
			ts, err := la.redis.HGet(ctx, key, "timestamp").Result()
			if err != nil {
				continue
			}

			tsTime, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				continue
			}

			if tsTime.Before(cutoff) {
				la.redis.Del(ctx, key)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

// dumpToLocalFile writes logs to a local file when GCP is unavailable.
func (la *LogAggregator) dumpToLocalFile(entries []logging.Entry) error {
	if err := os.MkdirAll(la.config.LocalDumpPath, 0755); err != nil {
		return err
	}

	filename := filepath.Join(la.config.LocalDumpPath, fmt.Sprintf("logs_%s.json", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry.Payload); err != nil {
			return err
		}
	}

	return nil
}

// cacheLogsInRedis stores recent logs in Redis for quick access.
func (la *LogAggregator) cacheLogsInRedis(entries []logging.Entry) error {
	ctx := context.Background()

	for _, entry := range entries {
		logData, err := json.Marshal(entry.Payload)
		if err != nil {
			continue
		}

		key := fmt.Sprintf("logs:%s:%s", entry.Timestamp.Format("2006-01-02"), uuid.New().String())
		la.redis.HSet(ctx, key, map[string]interface{}{
			"timestamp": entry.Timestamp.Format(time.RFC3339),
			"data":      string(logData),
		})
		la.redis.Expire(ctx, key, time.Duration(la.config.RetentionDays)*24*time.Hour)
	}

	return nil
}

// QueryLogs retrieves logs based on the provided query.
func (la *LogAggregator) QueryLogs(ctx context.Context, query AggregationQuery) (*AggregationResult, error) {
	result := &AggregationResult{
		Facets: Facets{
			ByLevel:    make(map[string]int64),
			ByService:  make(map[string]int64),
			ByEndpoint: make(map[string]int64),
		},
	}

	if la.redis == nil {
		return result, nil
	}

	pattern := fmt.Sprintf("logs:%s", query.StartTime.Format("2006-01-02"))
	if query.StartTime.Format("2006-01-02") != query.EndTime.Format("2006-01-02") {
		pattern = "logs:*"
	}

	var cursor uint64
	var allLogs []LogEntry
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := query.Offset

	for {
		keys, nextCursor, err := la.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan failed: %w", err)
		}

		for _, key := range keys {
			data, err := la.redis.HGet(ctx, key, "data").Result()
			if err != nil {
				continue
			}

			var entry LogEntry
			if err := json.Unmarshal([]byte(data), &entry); err != nil {
				continue
			}

			if la.matchQuery(&entry, &query) {
				allLogs = append(allLogs, entry)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	totalCount := int64(len(allLogs))

	if offset >= len(allLogs) {
		result.Logs = []LogEntry{}
		result.TotalCount = totalCount
		return result, nil
	}

	endIdx := offset + limit
	if endIdx > len(allLogs) {
		endIdx = len(allLogs)
	}

	result.Logs = allLogs[offset:endIdx]
	result.TotalCount = totalCount

	for _, log := range allLogs {
		result.Facets.ByLevel[log.Level.String()]++
		result.Facets.ByService[log.ServiceName]++
		if log.Endpoint != "" {
			result.Facets.ByEndpoint[log.Endpoint]++
		}
	}

	return result, nil
}

// matchQuery checks if a log entry matches the query criteria.
func (la *LogAggregator) matchQuery(entry *LogEntry, query *AggregationQuery) bool {
	if !query.StartTime.IsZero() && entry.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && entry.Timestamp.After(query.EndTime) {
		return false
	}
	if query.ServiceName != "" && entry.ServiceName != query.ServiceName {
		return false
	}
	if query.LogLevel != 0 && entry.Level != query.LogLevel {
		return false
	}
	if query.TraceID != "" && entry.TraceID != query.TraceID {
		return false
	}
	if query.UserID != "" && entry.UserID != query.UserID {
		return false
	}
	if query.SearchText != "" {
		found := false
		if contains(entry.Message, query.SearchText) {
			found = true
		}
		if !found && entry.Error != "" && contains(entry.Error, query.SearchText) {
			found = true
		}
		if !found {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

### 3. FiberLogger Middleware

```go
package logging

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// FiberLogger returns a Fiber middleware for automatic log collection.
func FiberLogger(aggregator *LogAggregator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		err := c.Next()

		duration := time.Since(startTime)
		c.Response().Header.Set("X-Response-Time", duration.String())

		traceID := c.GetRespHeader("X-Trace-ID", "")
		if traceID == "" {
			traceID = generateTraceID()
			c.Set("X-Trace-ID", traceID)
		}
		c.Locals("trace_id", traceID)

		logLevel := LogLevelInfo
		if c.Response().StatusCode() >= 500 {
			logLevel = LogLevelError
		} else if c.Response().StatusCode() >= 400 {
			logLevel = LogLevelWarning
		} else if duration > 5*time.Second {
			logLevel = LogLevelWarning
		}

		additionalFields := fiber.Map{
			"request_id":     c.GetRespHeader("X-Request-ID", ""),
			"content_length": c.Response().Header.ContentLength(),
			"user_agent":     c.Get("User-Agent"),
			"referer":        c.Get("Referer"),
		}

		if err != nil {
			additionalFields["error"] = err.Error()
			if logLevel < LogLevelError {
				logLevel = LogLevelError
			}
		}

		if aggregator != nil {
			_ = aggregator.SubmitLogFromFiberCtx(c, logLevel, formatLogMessage(c, err), additionalFields)
		}

		return err
	}
}

func generateTraceID() string {
	return strings.ReplaceAll(time.Now().Format(time.RFC3339Nano), ".", "")[:32]
}

func formatLogMessage(c *fiber.Ctx, err error) string {
	if err != nil {
		return err.Error()
	}
	return "Request processed"
}
```

### 4. MetricsCollector

```go
package logging

import (
	"context"
	"sync"
	"time"
)

// MetricsCollector collects and reports system metrics.
type MetricsCollector struct {
	aggregator *LogAggregator
	interval   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewMetricsCollector creates a new MetricsCollector.
func NewMetricsCollector(aggregator *LogAggregator, interval time.Duration) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &MetricsCollector{
		aggregator: aggregator,
		interval:   interval,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins collecting metrics.
func (mc *MetricsCollector) Start() {
	mc.wg.Add(1)
	go mc.collectWorker()
}

// Stop stops the metrics collector.
func (mc *MetricsCollector) Stop() {
	mc.cancel()
	mc.wg.Wait()
}

func (mc *MetricsCollector) collectWorker() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.collectAndReportMetrics()
		}
	}
}

func (mc *MetricsCollector) collectAndReportMetrics() {
	now := time.Now().UTC()
	serviceName := "mealswapp-api"

	mc.aggregator.SubmitLog(LogEntry{
		Timestamp:   now,
		ServiceName: serviceName,
		Level:       LogLevelInfo,
		Message:     "metrics_ping",
		Fields: fiber.Map{
			"metric_type": "response_time_p95",
			"value":       150.5,
		},
	})
}

// RecordMetric records a single metric data point.
func (mc *MetricsCollector) RecordMetric(serviceName string, metricType MetricType, value float64, labels map[string]string) {
	entry := LogEntry{
		Timestamp:   time.Now().UTC(),
		ServiceName: serviceName,
		Level:       LogLevelInfo,
		Message:     "metric_record",
		Fields: fiber.Map{
			"metric_type": metricType.String(),
			"value":       value,
			"labels":      labels,
		},
	}
	mc.aggregator.SubmitLog(entry)
}

// RecordLatency records response latency for a request.
func (mc *MetricsCollector) RecordLatency(serviceName string, duration time.Duration, endpoint string) {
	entry := LogEntry{
		Timestamp:   time.Now().UTC(),
		ServiceName: serviceName,
		Level:       LogLevelInfo,
		Message:     "latency_record",
		Endpoint:    endpoint,
		Duration:    duration.Milliseconds(),
		Fields: fiber.Map{
			"metric_type": MetricTypeResponseTime.String(),
			"value":       float64(duration.Milliseconds()),
		},
	}
	mc.aggregator.SubmitLog(entry)
}

func (m MetricType) String() string {
	switch m {
	case MetricTypeResponseTime:
		return "response_time"
	case MetricTypeErrorRate:
		return "error_rate"
	case MetricTypeConcurrentUsers:
		return "concurrent_users"
	case MetricTypeRequestCount:
		return "request_count"
	case MetricTypeCPUUsage:
		return "cpu_usage"
	case MetricTypeMemoryUsage:
		return "memory_usage"
	default:
		return "unknown"
	}
}
```

### 5. State Management & Error Handling

#### Error States

| Error Condition | Handling Strategy |
| :--- | :--- |
| **GCP Logging Unavailable** | Buffer logs locally, retry connection every 30s, enable local dump fallback |
| **Redis Connection Lost** | Continue sending to GCP, cache locally in memory buffer, retry Redis connection |
| **Buffer Overflow** | Immediate flush, drop oldest logs if buffer exceeds 2x capacity |
| **Invalid Log Entry** | Skip invalid entry, log error to stderr, continue processing |
| **Query Timeout** | Return partial results with error indicator, implement pagination |
| **Retention Cleanup Failure** | Log error, skip cleanup cycle, retry on next cycle |

#### State Transitions

```
IDLE → COLLECTING (on log submission)
COLLECTING → FLUSHING (buffer full or interval reached)
FLUSHING → COLLECTING (flush complete)
FLUSHING → FALLBACK (GCP unavailable)
FALLBACK → COLLECTING (GCP restored)
ANY → STOPPED (on shutdown)
```

### 6. Component Interfaces

```go
type LogAggregatorInterface interface {
	Start()
	Stop() error
	SubmitLog(entry LogEntry) error
	SubmitLogFromFiberCtx(c *fiber.Ctx, level LogLevel, message string, additionalFields map[string]interface{}) error
	QueryLogs(ctx context.Context, query AggregationQuery) (*AggregationResult, error)
}

type MetricsCollectorInterface interface {
	Start()
	Stop()
	RecordMetric(serviceName string, metricType MetricType, value float64, labels map[string]string)
	RecordLatency(serviceName string, duration time.Duration, endpoint string)
}

type FiberLoggerMiddleware interface {
	Handler() fiber.Handler
}
```

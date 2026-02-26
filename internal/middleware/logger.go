// Phase: phase-01 | Task: 13 | Architecture: ARCH-014 | Design: FiberLogger

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	LogLevelDebug    = "debug"
	LogLevelInfo     = "info"
	LogLevelWarning  = "warning"
	LogLevelError    = "error"
	LogLevelCritical = "critical"
)

type LoggerConfig struct {
	Output              io.Writer
	Format              string
	TimeFormat          string
	TimeZone            *time.Location
	EnableColors        bool
	EnableUTC           bool
	GCPConfig           *GCPConfig
	SkipHealthCheckLogs bool
	SkipUptimeCheckLogs bool
}

type GCPConfig struct {
	ProjectID               string
	LogNamePrefix           string
	EnableStructuredLogging bool
	BatchSize               int
	FlushInterval           time.Duration
	Timeout                 time.Duration
}

type LogEntry struct {
	Timestamp  string                 `json:"timestamp"`
	Level      string                 `json:"level"`
	RequestID  string                 `json:"request_id,omitempty"`
	Method     string                 `json:"method,omitempty"`
	Path       string                 `json:"path,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	Latency    int                    `json:"latency_ms,omitempty"`
	ClientIP   string                 `json:"client_ip,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	Message    string                 `json:"message"`
}

var DefaultLoggerConfig = LoggerConfig{
	Output:              os.Stdout,
	Format:              "json",
	TimeFormat:          "2006-01-02 15:04:05",
	TimeZone:            time.Local,
	EnableColors:        true,
	EnableUTC:           false,
	GCPConfig:           nil,
	SkipHealthCheckLogs: true,
	SkipUptimeCheckLogs: true,
}

var DefaultGCPConfig = GCPConfig{
	ProjectID:               "",
	LogNamePrefix:           "mealswapp",
	EnableStructuredLogging: true,
	BatchSize:               100,
	FlushInterval:           5 * time.Second,
	Timeout:                 30 * time.Second,
}

func (c *LoggerConfig) WithOutput(output io.Writer) *LoggerConfig {
	c.Output = output
	return c
}

func (c *LoggerConfig) WithFormat(format string) *LoggerConfig {
	c.Format = format
	return c
}

func (c *LoggerConfig) WithTimeFormat(format string) *LoggerConfig {
	c.TimeFormat = format
	return c
}

func (c *LoggerConfig) WithTimeZone(tz *time.Location) *LoggerConfig {
	c.TimeZone = tz
	return c
}

func (c *LoggerConfig) WithColors(enable bool) *LoggerConfig {
	c.EnableColors = enable
	return c
}

func (c *LoggerConfig) WithUTC(enable bool) *LoggerConfig {
	c.EnableUTC = enable
	return c
}

func (c *LoggerConfig) WithSkipHealthCheckLogs(skip bool) *LoggerConfig {
	c.SkipHealthCheckLogs = skip
	return c
}

func (c *LoggerConfig) WithSkipUptimeCheckLogs(skip bool) *LoggerConfig {
	c.SkipUptimeCheckLogs = skip
	return c
}

func (c *LoggerConfig) WithGCPConfig(config GCPConfig) *LoggerConfig {
	c.GCPConfig = &config
	return c
}

func New(config LoggerConfig) fiber.Handler {
	return newLogger(config)
}

func NewWithGCP(gcpConfig GCPConfig) fiber.Handler {
	config := DefaultLoggerConfig
	config.GCPConfig = &gcpConfig
	return newLogger(config)
}

func Default() fiber.Handler {
	return newLogger(DefaultLoggerConfig)
}

type logger struct {
	config    LoggerConfig
	gcpWriter *gcpWriter
	metrics   *metrics
}

func newLogger(config LoggerConfig) fiber.Handler {
	l := &logger{
		config:  config,
		metrics: newMetrics(),
	}

	if config.GCPConfig != nil && config.GCPConfig.ProjectID != "" {
		l.gcpWriter = newGCPWriter(*config.GCPConfig)
	}

	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Locals("request_id", requestID)
		c.Set("X-Request-ID", requestID)

		err := c.Next()

		latency := time.Since(startTime)
		statusCode := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		clientIP := c.IP()
		userAgent := c.Get("User-Agent")

		level := LogLevelFromStatus(statusCode)

		entry := LogEntry{
			Timestamp:  time.Now().Format(time.RFC3339),
			Level:      level,
			RequestID:  requestID,
			Method:     method,
			Path:       path,
			StatusCode: statusCode,
			Latency:    int(latency.Milliseconds()),
			ClientIP:   clientIP,
			UserAgent:  userAgent,
		}

		if c.Locals("error") != nil {
			entry.Error = extractErrorMessage(c.Locals("error"))
		}

		if c.Locals("log_message") != nil {
			entry.Message = c.Locals("log_message").(string)
		} else {
			entry.Message = fmt.Sprintf("%s %s completed", method, path)
		}

		if config.SkipHealthCheckLogs && path == "/health" {
			return err
		}
		if config.SkipUptimeCheckLogs && path == "/uptime" {
			return err
		}

		l.writeLog(entry)

		return err
	}
}

func (l *logger) writeLog(entry LogEntry) {
	l.metrics.increment(entry.Level)

	if l.config.Format == "json" {
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal log entry: %v\n", err)
			return
		}
		fmt.Fprintln(l.config.Output, string(entryJSON))
	} else {
		timestamp := entry.Timestamp
		if l.config.EnableUTC {
			timestamp = time.Now().UTC().Format(l.config.TimeFormat)
		} else if l.config.TimeZone != nil {
			timestamp = time.Now().In(l.config.TimeZone).Format(l.config.TimeFormat)
		}

		logLine := fmt.Sprintf("[%s] %s: %s | method: %s | path: %s | status: %d | latency: %dms",
			timestamp,
			entry.Level,
			entry.Message,
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.Latency,
		)

		if entry.Error != "" {
			logLine += fmt.Sprintf(" | error: %s", entry.Error)
		}

		fmt.Fprintln(l.config.Output, logLine)
	}

	if l.gcpWriter != nil {
		l.gcpWriter.write(entry)
	}
}

type GCPWriter interface {
	WriteLogEntry(ctx context.Context, entry LogEntry) error
	WriteLogBatch(ctx context.Context, entries []LogEntry) error
	Flush(ctx context.Context) error
	Close() error
	Health(ctx context.Context) error
}

type gcpWriter struct {
	config     GCPConfig
	buffer     []LogEntry
	bufferMu   sync.Mutex
	flushTimer *time.Timer
	closed     bool
	metrics    *metrics
}

func newGCPWriter(config GCPConfig) *gcpWriter {
	if config.BatchSize == 0 {
		config.BatchSize = DefaultGCPConfig.BatchSize
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = DefaultGCPConfig.FlushInterval
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultGCPConfig.Timeout
	}

	gw := &gcpWriter{
		config:  config,
		buffer:  make([]LogEntry, 0, config.BatchSize),
		metrics: newMetrics(),
	}

	gw.flushTimer = time.AfterFunc(config.FlushInterval, func() {
		gw.flush()
	})

	return gw
}

func (gw *gcpWriter) write(entry LogEntry) {
	gw.bufferMu.Lock()
	defer gw.bufferMu.Unlock()

	if gw.closed {
		return
	}

	gw.buffer = append(gw.buffer, entry)

	if len(gw.buffer) >= gw.config.BatchSize {
		gw.flushTimer.Stop()
		go gw.flush()
	}
}

func (gw *gcpWriter) flush() {
	gw.bufferMu.Lock()
	if len(gw.buffer) == 0 {
		gw.bufferMu.Unlock()
		gw.resetTimer()
		return
	}

	entries := make([]LogEntry, len(gw.buffer))
	copy(entries, gw.buffer)
	gw.buffer = gw.buffer[:0]
	gw.bufferMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), gw.config.Timeout)
	defer cancel()

	err := gw.WriteLogBatch(ctx, entries)
	if err != nil {
		gw.metrics.incrementFailed()
		fmt.Fprintf(os.Stderr, "failed to flush logs to GCP: %v\n", err)
		gw.bufferMu.Lock()
		gw.buffer = append(gw.buffer, entries...)
		gw.bufferMu.Unlock()
	}

	gw.resetTimer()
}

func (gw *gcpWriter) resetTimer() {
	gw.bufferMu.Lock()
	defer gw.bufferMu.Unlock()
	if !gw.closed {
		gw.flushTimer.Reset(gw.config.FlushInterval)
	}
}

func (gw *gcpWriter) WriteLogEntry(ctx context.Context, entry LogEntry) error {
	return gw.WriteLogBatch(ctx, []LogEntry{entry})
}

func (gw *gcpWriter) WriteLogBatch(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	logName := fmt.Sprintf("projects/%s/logs/%s", gw.config.ProjectID, gw.config.LogNamePrefix)
	fmt.Fprintf(os.Stderr, "GCP WriteLogBatch: %s (%d entries)\n", logName, len(entries))

	return nil
}

func (gw *gcpWriter) Flush(ctx context.Context) error {
	gw.bufferMu.Lock()
	defer gw.bufferMu.Unlock()

	if len(gw.buffer) == 0 {
		return nil
	}

	entries := make([]LogEntry, len(gw.buffer))
	copy(entries, gw.buffer)
	gw.buffer = gw.buffer[:0]

	return gw.WriteLogBatch(ctx, entries)
}

func (gw *gcpWriter) Close() error {
	gw.bufferMu.Lock()
	defer gw.bufferMu.Unlock()

	gw.closed = true
	gw.flushTimer.Stop()

	if len(gw.buffer) > 0 {
		entries := make([]LogEntry, len(gw.buffer))
		copy(entries, gw.buffer)
		gw.buffer = gw.buffer[:0]

		ctx, cancel := context.WithTimeout(context.Background(), gw.config.Timeout)
		defer cancel()
		return gw.WriteLogBatch(ctx, entries)
	}

	return nil
}

func (gw *gcpWriter) Health(ctx context.Context) error {
	return nil
}

type LogEntryBuilder struct {
	entry LogEntry
}

func NewLogEntryBuilder(level string, message string) *LogEntryBuilder {
	return &LogEntryBuilder{
		entry: LogEntry{
			Level:   level,
			Message: message,
			Fields:  make(map[string]interface{}),
		},
	}
}

func (b *LogEntryBuilder) WithRequestID(id string) *LogEntryBuilder {
	b.entry.RequestID = id
	return b
}

func (b *LogEntryBuilder) WithHTTPDetails(method string, path string, statusCode int) *LogEntryBuilder {
	b.entry.Method = method
	b.entry.Path = path
	b.entry.StatusCode = statusCode
	return b
}

func (b *LogEntryBuilder) WithLatency(ms int) *LogEntryBuilder {
	b.entry.Latency = ms
	return b
}

func (b *LogEntryBuilder) WithClientIP(ip string) *LogEntryBuilder {
	b.entry.ClientIP = ip
	return b
}

func (b *LogEntryBuilder) WithUserAgent(ua string) *LogEntryBuilder {
	b.entry.UserAgent = ua
	return b
}

func (b *LogEntryBuilder) WithError(err error) *LogEntryBuilder {
	b.entry.Error = extractErrorMessage(err)
	return b
}

func (b *LogEntryBuilder) WithField(key string, value interface{}) *LogEntryBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogEntryBuilder) WithFields(fields map[string]interface{}) *LogEntryBuilder {
	for k, v := range fields {
		b.entry.Fields[k] = v
	}
	return b
}

func (b *LogEntryBuilder) Build() LogEntry {
	if b.entry.Timestamp == "" {
		b.entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	return b.entry
}

func ExtractRequestID(c *fiber.Ctx) string {
	if id := c.Get("X-Request-ID"); id != "" {
		return id
	}
	if id, ok := c.Locals("request_id").(string); ok && id != "" {
		return id
	}
	return uuid.New().String()
}

func LogLevelFromStatus(statusCode int) string {
	switch {
	case statusCode >= 500:
		return LogLevelCritical
	case statusCode >= 400:
		return LogLevelError
	case statusCode >= 300:
		return LogLevelWarning
	default:
		return LogLevelInfo
	}
}

func FormatLatency(d time.Duration) string {
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func SanitizeLogMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", "")
	msg = strings.ReplaceAll(msg, "\"", "'")

	if len(msg) > 1000 {
		msg = msg[:1000]
	}

	return msg
}

func extractErrorMessage(err interface{}) string {
	if err == nil {
		return ""
	}

	switch e := err.(type) {
	case error:
		msg := e.Error()
		msg = strings.ReplaceAll(msg, "\n", " ")
		msg = strings.ReplaceAll(msg, "\"", "'")
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		return msg
	case string:
		return SanitizeLogMessage(e)
	default:
		return fmt.Sprintf("%v", err)
	}
}

type metrics struct {
	mu                 sync.RWMutex
	totalLogEntries    int64
	logEntriesByLevel  map[string]int64
	failedGCPWrites    int64
	totalFlushDuration time.Duration
	flushCount         int
}

func newMetrics() *metrics {
	return &metrics{
		logEntriesByLevel: make(map[string]int64),
	}
}

func (m *metrics) increment(level string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLogEntries++
	m.logEntriesByLevel[level]++
}

func (m *metrics) incrementFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedGCPWrites++
}

func (m *metrics) recordFlush(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalFlushDuration += d
	m.flushCount++
}

func (m *metrics) TotalLogEntries() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalLogEntries
}

func (m *metrics) LogEntriesByLevel() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]int64, len(m.logEntriesByLevel))
	for k, v := range m.logEntriesByLevel {
		result[k] = v
	}
	return result
}

func (m *metrics) FailedGCPWrites() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failedGCPWrites
}

func (m *metrics) AverageFlushDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.flushCount == 0 {
		return 0
	}
	return m.totalFlushDuration / time.Duration(m.flushCount)
}

func (m *metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLogEntries = 0
	m.logEntriesByLevel = make(map[string]int64)
	m.failedGCPWrites = 0
	m.totalFlushDuration = 0
	m.flushCount = 0
}

var globalMetrics = newMetrics()

func GetMetrics() *metrics {
	return globalMetrics
}

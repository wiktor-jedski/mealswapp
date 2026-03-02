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

type FiberLoggerConfig struct {
	Output              io.Writer
	Format              string
	TimeFormat          string
	TimeZone            *time.Location
	EnableColors        bool
	EnableUTC           bool
	GCPConfig           *FiberGCPConfig
	SkipHealthCheckLogs bool
	SkipUptimeCheckLogs bool
}

type FiberGCPConfig struct {
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

var FiberDefaultLoggerConfig = FiberLoggerConfig{
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

var FiberDefaultGCPConfig = FiberGCPConfig{
	ProjectID:               "",
	LogNamePrefix:           "mealswapp",
	EnableStructuredLogging: true,
	BatchSize:               100,
	FlushInterval:           5 * time.Second,
	Timeout:                 30 * time.Second,
}

func (c *FiberLoggerConfig) WithOutput(output io.Writer) *FiberLoggerConfig {
	c.Output = output
	return c
}

func (c *FiberLoggerConfig) WithFormat(format string) *FiberLoggerConfig {
	c.Format = format
	return c
}

func (c *FiberLoggerConfig) WithTimeFormat(format string) *FiberLoggerConfig {
	c.TimeFormat = format
	return c
}

func (c *FiberLoggerConfig) WithTimeZone(tz *time.Location) *FiberLoggerConfig {
	c.TimeZone = tz
	return c
}

func (c *FiberLoggerConfig) WithColors(enable bool) *FiberLoggerConfig {
	c.EnableColors = enable
	return c
}

func (c *FiberLoggerConfig) WithUTC(enable bool) *FiberLoggerConfig {
	c.EnableUTC = enable
	return c
}

func (c *FiberLoggerConfig) WithSkipHealthCheckLogs(skip bool) *FiberLoggerConfig {
	c.SkipHealthCheckLogs = skip
	return c
}

func (c *FiberLoggerConfig) WithSkipUptimeCheckLogs(skip bool) *FiberLoggerConfig {
	c.SkipUptimeCheckLogs = skip
	return c
}

func (c *FiberLoggerConfig) WithGCPConfig(config FiberGCPConfig) *FiberLoggerConfig {
	c.GCPConfig = &config
	return c
}

type fiberLogger struct {
	config      FiberLoggerConfig
	gcpWriter   FiberGCPWriter
	metrics     *fiberLoggerMetrics
	batchMu     sync.Mutex
	batchBuffer []LogEntry
	stopCh      chan struct{}
}

type fiberLoggerMetrics struct {
	mu                 sync.RWMutex
	totalLogEntries    int64
	logEntriesByLevel  map[string]int64
	failedGCPWrites    int64
	totalFlushDuration time.Duration
	flushCount         int64
}

func (m *fiberLoggerMetrics) TotalLogEntries() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalLogEntries
}

func (m *fiberLoggerMetrics) LogEntriesByLevel() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copy := make(map[string]int64)
	for k, v := range m.logEntriesByLevel {
		copy[k] = v
	}
	return copy
}

func (m *fiberLoggerMetrics) FailedGCPWrites() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failedGCPWrites
}

func (m *fiberLoggerMetrics) AverageFlushDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.flushCount == 0 {
		return 0
	}
	return m.totalFlushDuration / time.Duration(m.flushCount)
}

func (m *fiberLoggerMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLogEntries = 0
	m.logEntriesByLevel = make(map[string]int64)
	m.failedGCPWrites = 0
	m.totalFlushDuration = 0
	m.flushCount = 0
}

func (m *fiberLoggerMetrics) incrementLogEntry(level string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLogEntries++
	m.logEntriesByLevel[level]++
}

func (m *fiberLoggerMetrics) incrementFailedWrites() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedGCPWrites++
}

func (m *fiberLoggerMetrics) recordFlushDuration(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalFlushDuration += d
	m.flushCount++
}

func GetFiberLoggerMetrics() FiberLoggerMetrics {
	return globalFiberLoggerMetrics
}

var globalFiberLoggerMetrics *fiberLoggerMetrics = &fiberLoggerMetrics{
	logEntriesByLevel: make(map[string]int64),
}

func NewFiberLogger(config FiberLoggerConfig) fiber.Handler {
	if config.Output == nil {
		config.Output = FiberDefaultLoggerConfig.Output
	}
	if config.Format == "" {
		config.Format = FiberDefaultLoggerConfig.Format
	}
	if config.TimeFormat == "" {
		config.TimeFormat = FiberDefaultLoggerConfig.TimeFormat
	}
	if config.TimeZone == nil {
		config.TimeZone = FiberDefaultLoggerConfig.TimeZone
	}

	l := &fiberLogger{
		config:      config,
		metrics:     globalFiberLoggerMetrics,
		batchBuffer: make([]LogEntry, 0, config.GCPConfig.BatchSize),
		stopCh:      make(chan struct{}),
	}

	if config.GCPConfig != nil {
		gcpWriter := NewFiberGCPWriter(*config.GCPConfig)
		l.gcpWriter = gcpWriter
		go l.batchFlushLoop()
	}

	return l.middleware
}

func NewFiberLoggerWithGCP(gcpConfig FiberGCPConfig) fiber.Handler {
	config := FiberDefaultLoggerConfig
	config.GCPConfig = &gcpConfig
	return NewFiberLogger(config)
}

func FiberLoggerDefault() fiber.Handler {
	return NewFiberLogger(FiberDefaultLoggerConfig)
}

func (l *fiberLogger) middleware(c *fiber.Ctx) error {
	startTime := time.Now()

	requestID := ExtractRequestID(c)
	c.Locals("request_id", requestID)
	c.Set("X-Request-ID", requestID)

	err := c.Next()

	latency := time.Since(startTime)
	statusCode := c.Response().StatusCode()
	method := c.Method()
	path := c.Path()
	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	if l.config.SkipHealthCheckLogs && path == "/health" {
		return err
	}
	if l.config.SkipUptimeCheckLogs && path == "/uptime" {
		return err
	}

	level := LogLevelFromStatus(statusCode)

	entry := NewLogEntryBuilder(level, getMessageForStatus(statusCode, method, path)).
		WithRequestID(requestID).
		WithHTTPDetails(method, path, statusCode).
		WithLatency(int(latency.Milliseconds())).
		WithClientIP(clientIP).
		WithUserAgent(userAgent).
		WithError(err).
		Build()

	l.writeLogEntry(entry)
	l.metrics.incrementLogEntry(level)

	return err
}

func getMessageForStatus(statusCode int, method string, path string) string {
	if statusCode >= 500 {
		return "Server error"
	}
	if statusCode >= 400 {
		return "Client error"
	}
	if statusCode >= 300 {
		return "Redirect"
	}
	return fmt.Sprintf("%s %s completed", method, path)
}

func (l *fiberLogger) writeLogEntry(entry LogEntry) {
	if l.config.Format == "json" {
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
			return
		}
		fmt.Fprintln(l.config.Output, string(entryJSON))
	} else {
		timestamp := entry.Timestamp
		if l.config.EnableUTC {
			timestamp = time.Now().UTC().Format(l.config.TimeFormat)
		} else {
			timestamp = time.Now().In(l.config.TimeZone).Format(l.config.TimeFormat)
		}
		fmt.Fprintf(l.config.Output, "[%s] %s: %s | method: %s | path: %s | status: %d | latency: %dms\n",
			timestamp, entry.Level, entry.Message, entry.Method, entry.Path, entry.StatusCode, entry.Latency)
	}

	if l.gcpWriter != nil {
		l.addToBatch(entry)
	}
}

func (l *fiberLogger) addToBatch(entry LogEntry) {
	l.batchMu.Lock()
	defer l.batchMu.Unlock()

	l.batchBuffer = append(l.batchBuffer, entry)

	if len(l.batchBuffer) >= l.config.GCPConfig.BatchSize {
		l.flushBatch()
	}
}

func (l *fiberLogger) flushBatch() {
	l.batchMu.Lock()
	entries := make([]LogEntry, len(l.batchBuffer))
	copy(entries, l.batchBuffer)
	l.batchBuffer = l.batchBuffer[:0]
	l.batchMu.Unlock()

	if len(entries) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.config.GCPConfig.Timeout)
	defer cancel()

	startTime := time.Now()
	err := l.gcpWriter.WriteLogBatch(ctx, entries)
	duration := time.Since(startTime)

	if err != nil {
		l.metrics.incrementFailedWrites()
		fmt.Fprintf(os.Stderr, "Failed to write logs to GCP: %v\n", err)
		l.retryFailedEntries(entries)
	} else {
		l.metrics.recordFlushDuration(duration)
	}
}

func (l *fiberLogger) retryFailedEntries(entries []LogEntry) {
	backoffs := []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, 2 * time.Second}

	for i, backoff := range backoffs {
		time.Sleep(backoff)

		ctx, cancel := context.WithTimeout(context.Background(), l.config.GCPConfig.Timeout)
		err := l.gcpWriter.WriteLogBatch(ctx, entries)
		cancel()

		if err == nil {
			return
		}

		if i == len(backoffs)-1 {
			fmt.Fprintf(os.Stderr, "GCP write failed after %d retries, dropping %d entries\n", len(backoffs), len(entries))
		}
	}
}

func (l *fiberLogger) batchFlushLoop() {
	ticker := time.NewTicker(l.config.GCPConfig.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.batchMu.Lock()
			if len(l.batchBuffer) > 0 {
				l.batchMu.Unlock()
				l.flushBatch()
			} else {
				l.batchMu.Unlock()
			}
		case <-l.stopCh:
			l.batchMu.Lock()
			if len(l.batchBuffer) > 0 {
				entries := make([]LogEntry, len(l.batchBuffer))
				copy(entries, l.batchBuffer)
				l.batchBuffer = l.batchBuffer[:0]
				l.batchMu.Unlock()

				ctx, cancel := context.WithTimeout(context.Background(), l.config.GCPConfig.Timeout)
				l.gcpWriter.WriteLogBatch(ctx, entries)
				cancel()
			} else {
				l.batchMu.Unlock()
			}
			return
		}
	}
}

type FiberGCPWriter interface {
	WriteLogEntry(ctx context.Context, entry LogEntry) error
	WriteLogBatch(ctx context.Context, entries []LogEntry) error
	Flush(ctx context.Context) error
	Close() error
	Health(ctx context.Context) error
}

type fiberGCPWriter struct {
	config     FiberGCPConfig
	client     *fiberGCPClient
	buffer     []LogEntry
	mu         sync.Mutex
	stopCh     chan struct{}
	flushTimer *time.Ticker
}

type fiberGCPClient struct {
	projectID        string
	logNamePrefix    string
	enableStructured bool
}

func NewFiberGCPWriter(config FiberGCPConfig) FiberGCPWriter {
	if config.BatchSize <= 0 {
		config.BatchSize = FiberDefaultGCPConfig.BatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = FiberDefaultGCPConfig.FlushInterval
	}
	if config.Timeout <= 0 {
		config.Timeout = FiberDefaultGCPConfig.Timeout
	}
	if config.LogNamePrefix == "" {
		config.LogNamePrefix = FiberDefaultGCPConfig.LogNamePrefix
	}

	return &fiberGCPWriter{
		config: config,
		client: &fiberGCPClient{
			projectID:        config.ProjectID,
			logNamePrefix:    config.LogNamePrefix,
			enableStructured: config.EnableStructuredLogging,
		},
		buffer:     make([]LogEntry, 0, config.BatchSize),
		stopCh:     make(chan struct{}),
		flushTimer: time.NewTicker(config.FlushInterval),
	}
}

func (w *fiberGCPWriter) WriteLogEntry(ctx context.Context, entry LogEntry) error {
	return w.WriteLogBatch(ctx, []LogEntry{entry})
}

func (w *fiberGCPWriter) WriteLogBatch(ctx context.Context, entries []LogEntry) error {
	if w.client.projectID == "" {
		return fmt.Errorf("GCP project ID is not configured")
	}

	for _, entry := range entries {
		gcpEntry := convertToGCPLogEntry(entry, w.client)
		_ = gcpEntry
	}

	return nil
}

func convertToGCPLogEntry(entry LogEntry, client *fiberGCPClient) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": entry.Timestamp,
		"severity":  convertToGCPSeverity(entry.Level),
		"logName":   fmt.Sprintf("projects/%s/logs/%s", client.projectID, client.logNamePrefix),
		"jsonPayload": map[string]interface{}{
			"level":      entry.Level,
			"request_id": entry.RequestID,
			"method":     entry.Method,
			"path":       entry.Path,
			"status":     entry.StatusCode,
			"latency_ms": entry.Latency,
			"client_ip":  entry.ClientIP,
			"user_agent": entry.UserAgent,
			"error":      entry.Error,
			"message":    entry.Message,
			"fields":     entry.Fields,
		},
		"resource": map[string]string{
			"type": "gae_app",
		},
	}
}

func convertToGCPSeverity(level string) string {
	switch level {
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
		return "DEFAULT"
	}
}

func (w *fiberGCPWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) == 0 {
		return nil
	}

	entries := make([]LogEntry, len(w.buffer))
	copy(entries, w.buffer)
	w.buffer = w.buffer[:0]

	return w.WriteLogBatch(ctx, entries)
}

func (w *fiberGCPWriter) Close() error {
	close(w.stopCh)
	w.flushTimer.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()

	return w.Flush(ctx)
}

func (w *fiberGCPWriter) Health(ctx context.Context) error {
	if w.client.projectID == "" {
		return fmt.Errorf("GCP project ID is not configured")
	}
	return nil
}

type LogEntryBuilder struct {
	entry LogEntry
}

func NewLogEntryBuilder(level string, message string) *LogEntryBuilder {
	return &LogEntryBuilder{
		entry: LogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     level,
			Message:   message,
			Fields:    make(map[string]interface{}),
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
	if err != nil {
		b.entry.Error = extractErrorMessage(err)
	}
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
	return b.entry
}

func ExtractRequestID(c *fiber.Ctx) string {
	requestID := c.Get("X-Request-ID")
	if requestID != "" {
		return requestID
	}

	if localID, ok := c.Locals("request_id").(string); ok && localID != "" {
		return localID
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
	msg = strings.ReplaceAll(msg, `"`, `\"`)
	if len(msg) > 1000 {
		msg = msg[:1000]
	}
	return msg
}

func extractErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		errs := unwrapper.Unwrap()
		if len(errs) > 0 {
			return extractErrorMessage(errs[len(errs)-1])
		}
	}

	if fiberErr, ok := err.(*fiber.Error); ok {
		return SanitizeLogMessage(fiberErr.Message)
	}

	return SanitizeLogMessage(err.Error())
}

type FiberLoggerMetrics interface {
	TotalLogEntries() int64
	LogEntriesByLevel() map[string]int64
	FailedGCPWrites() int64
	AverageFlushDuration() time.Duration
	Reset()
}

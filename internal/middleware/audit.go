// Phase: phase-01 | Task: 14 | Architecture: ARCH-014 | Design: AuditLogger

package middleware

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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
	ID            string         `json:"id"`
	Timestamp     time.Time      `json:"timestamp"`
	EventType     EventType      `json:"event_type"`
	Severity      EventSeverity  `json:"severity"`
	UserID        string         `json:"user_id,omitempty"`
	IPAddress     string         `json:"ip_address"`
	UserAgent     string         `json:"user_agent,omitempty"`
	RequestPath   string         `json:"request_path,omitempty"`
	RequestMethod string         `json:"request_method,omitempty"`
	Action        string         `json:"action"`
	Resource      string         `json:"resource,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	StatusCode    int            `json:"status_code,omitempty"`
	Duration      time.Duration  `json:"duration,omitempty"`
	ErrorMsg      string         `json:"error_msg,omitempty"`
}

type Logger interface {
	Log(event AuditEvent) error
	LogAuthentication(userID string, success bool, reason string, c *fiber.Ctx) error
	LogAPIRequest(c *fiber.Ctx, statusCode int, duration time.Duration) error
	LogError(err error, c *fiber.Ctx, context string) error
	LogAdminAction(userID string, action string, resource string, details map[string]any) error
	Close() error
}

type FileLogger struct {
	file        *os.File
	mu          sync.Mutex
	encoder     *json.Encoder
	config      LoggerConfig
	currentSize int64
	lastRotate  time.Time
}

type DatabaseLogger struct {
	buffer      chan AuditEvent
	flushTicker *time.Ticker
	stopCh      chan struct{}
	config      LoggerConfig
	mu          sync.Mutex
	closed      bool
}

type CompositeLogger struct {
	loggers []Logger
	mu      sync.Mutex
}

type ConsoleLogger struct {
	config LoggerConfig
}

type LoggerConfig struct {
	OutputPath     string
	DatabaseDSN    string
	EnableConsole  bool
	EnableFile     bool
	EnableDatabase bool
	LogLevel       EventSeverity
	BufferSize     int
	FlushInterval  time.Duration
	RotateInterval time.Duration
	MaxFileSizeMB  int
	RetentionDays  int
}

func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		EnableConsole:  true,
		EnableFile:     false,
		EnableDatabase: false,
		LogLevel:       SeverityInfo,
		BufferSize:     1000,
		FlushInterval:  time.Second * 30,
		RotateInterval: time.Hour * 24,
		MaxFileSizeMB:  100,
		RetentionDays:  90,
	}
}

func NewFileLogger(path string, config LoggerConfig) (*FileLogger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	logger := &FileLogger{
		file:        file,
		encoder:     json.NewEncoder(file),
		config:      config,
		currentSize: stat.Size(),
		lastRotate:  time.Now(),
	}

	go logger.rotateWorker()

	return logger, nil
}

func (l *FileLogger) rotateWorker() {
	ticker := time.NewTicker(l.config.RotateInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := l.Rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "audit: file rotation failed: %v\n", err)
		}
	}
}

func (l *FileLogger) Log(event AuditEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to encode audit event: %w", err)
	}

	l.currentSize += int64(len(event.Action) + 100)

	maxSize := int64(l.config.MaxFileSizeMB * 1024 * 1024)
	if l.currentSize > maxSize {
		if err := l.Rotate(); err != nil {
			return fmt.Errorf("failed to rotate file: %w", err)
		}
	}

	return nil
}

func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *FileLogger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}

	l.file.Sync()

	ts := l.lastRotate.Format("20060102-150405")
	newPath := fmt.Sprintf("%s.%s.log", l.config.OutputPath, ts)

	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close old file: %w", err)
	}

	if err := os.Rename(l.config.OutputPath, newPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	file, err := os.OpenFile(l.config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.file = file
	l.encoder = json.NewEncoder(file)
	l.currentSize = 0
	l.lastRotate = time.Now()

	go l.cleanOldFiles()

	return nil
}

func (l *FileLogger) cleanOldFiles() {
	dir := filepath.Dir(l.config.OutputPath)
	base := filepath.Base(l.config.OutputPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -l.config.RetentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > len(base)+1 && name[:len(base)+1] == base+"." {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(dir, name))
			}
		}
	}
}

func NewDatabaseLogger(dsn string, config LoggerConfig) (*DatabaseLogger, error) {
	logger := &DatabaseLogger{
		buffer:      make(chan AuditEvent, config.BufferSize),
		flushTicker: time.NewTicker(config.FlushInterval),
		stopCh:      make(chan struct{}),
		config:      config,
	}

	go logger.flushWorker()

	return logger, nil
}

func (l *DatabaseLogger) flushWorker() {
	for {
		select {
		case <-l.flushTicker.C:
			l.Flush()
		case <-l.stopCh:
			l.Flush()
			return
		}
	}
}

func (l *DatabaseLogger) Log(event AuditEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case l.buffer <- event:
		return nil
	default:
		return fmt.Errorf("audit: buffer full, cannot log event")
	}
}

func (l *DatabaseLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	l.flushTicker.Stop()
	close(l.stopCh)

	return nil
}

func (l *DatabaseLogger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	count := 0
	for len(l.buffer) > 0 {
		select {
		case event := <-l.buffer:
			if err := l.insertEvent(event); err != nil {
				fmt.Fprintf(os.Stderr, "audit: failed to insert event: %v\n", err)
				continue
			}
			count++
		default:
			break
		}

		if count >= l.config.BufferSize {
			break
		}
	}

	return nil
}

func (l *DatabaseLogger) insertEvent(event AuditEvent) error {
	return nil
}

func NewCompositeLogger(loggers []Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

func (l *CompositeLogger) Log(event AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for _, logger := range l.loggers {
		if err := logger.Log(event); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple logger errors: %v", errs)
	}
	return nil
}

func (l *CompositeLogger) AddLogger(logger Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loggers = append(l.loggers, logger)
}

func (l *CompositeLogger) RemoveLogger(logger Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, l := range l.loggers {
		if l == logger {
			l.loggers = append(l.loggers[:i], l.loggers[i+1:]...)
			return
		}
	}
}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{config: DefaultLoggerConfig()}
}

func (l *ConsoleLogger) Log(event AuditEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	output, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func (l *ConsoleLogger) Close() error {
	return nil
}

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
			_ = logger.LogAPIRequest(c, c.Response().StatusCode(), duration)
		}

		return err
	}
}

func NewAuditLogger(config LoggerConfig) (Logger, error) {
	var loggers []Logger

	if config.EnableConsole {
		loggers = append(loggers, NewConsoleLogger())
	}

	if config.EnableFile {
		if config.OutputPath == "" {
			config.OutputPath = "logs/audit.log"
		}
		fileLogger, err := NewFileLogger(config.OutputPath, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create file logger: %w", err)
		}
		loggers = append(loggers, fileLogger)
	}

	if config.EnableDatabase {
		if config.DatabaseDSN == "" {
			return nil, fmt.Errorf("database DSN required when EnableDatabase is true")
		}
		dbLogger, err := NewDatabaseLogger(config.DatabaseDSN, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create database logger: %w", err)
		}
		loggers = append(loggers, dbLogger)
	}

	if len(loggers) == 0 {
		loggers = append(loggers, NewConsoleLogger())
	}

	return NewCompositeLogger(loggers), nil
}

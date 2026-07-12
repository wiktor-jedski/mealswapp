// Package queue contains the Redis Streams boundary for asynchronous
// optimization jobs.
package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// Implements DESIGN-004 JobQueueManager.
const (
	DefaultStream             = "mealswapp:optimization:jobs:v1"
	DefaultGroup              = "optimization-workers"
	DefaultVisibilityTimeout  = 45 * time.Second
	DefaultReadBlock          = time.Second
	DefaultBatchSize          = int64(1)
	DefaultMaxAttempts        = 3
	DefaultCompletedTTL       = time.Hour
	DefaultAttemptTTL         = 24 * time.Hour
	minimumVisibilityTimeout  = 30 * time.Second
	jobIDField                = "job_id"
	enqueuedAtField           = "enqueued_at"
	completedValue            = "completed"
	failedValue               = "failed"
	processingLockValuePrefix = "consumer"
)

// ErrNoJob means that a bounded reservation wait returned no message.
// Implements DESIGN-004 JobQueueManager.
var ErrNoJob = errors.New("no optimization job available")

// ErrJobInProgress means another consumer currently owns the same logical job.
// Implements DESIGN-004 JobQueueManager.
var ErrJobInProgress = errors.New("optimization job is already being processed")

// ErrInvalidJob means that a stream entry did not contain a valid job ID.
// Implements DESIGN-004 JobQueueManager.
var ErrInvalidJob = errors.New("invalid optimization job stream entry")

// ErrQueueUnavailable identifies Redis connection and command failures. The
// API must map this error to an unavailable queue response and must not invoke
// a synchronous solver fallback.
// Implements DESIGN-004 JobQueueManager.
var ErrQueueUnavailable = errors.New("optimization queue unavailable")

// Processor is the worker-only alternative-generation boundary. It receives
// only a server-created job ID; request data and authoritative publication
// remain outside the Redis stream payload.
// Implements DESIGN-004 JobQueueManager and LPSolverWrapper worker boundary.
type Processor func(context.Context, Job) error

// TerminalHandler records a terminal failure or cancellation before the queue
// acknowledges the delivery, so the handler must make its status update
// idempotent for a crash between publication and XACK.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
type TerminalHandler func(context.Context, Job, error) error

// Config controls one Redis Streams queue and consumer group.
// Implements DESIGN-004 JobQueueManager.
type Config struct {
	Stream            string
	Group             string
	Consumer          string
	VisibilityTimeout time.Duration
	ReadBlock         time.Duration
	BatchSize         int64
	MaxAttempts       int
	CompletedTTL      time.Duration
	AttemptTTL        time.Duration
	TerminalHandler   TerminalHandler
}

// DefaultConfig returns production-safe queue settings. The visibility
// timeout is longer than the solver's hard 30-second deadline.
// Implements DESIGN-004 JobQueueManager.
func DefaultConfig() Config {
	consumer := "optimization-worker"
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		consumer = hostname
	}
	consumer += "-" + strconv.Itoa(os.Getpid()) + "-" + uuid.NewString()
	return Config{
		Stream:            DefaultStream,
		Group:             DefaultGroup,
		Consumer:          consumer,
		VisibilityTimeout: DefaultVisibilityTimeout,
		ReadBlock:         DefaultReadBlock,
		BatchSize:         DefaultBatchSize,
		MaxAttempts:       DefaultMaxAttempts,
		CompletedTTL:      DefaultCompletedTTL,
		AttemptTTL:        DefaultAttemptTTL,
	}
}

// Job is the server-owned stream delivery passed to a worker processor.
// EntryID identifies the Redis delivery; ID identifies the logical job and is
// the idempotency boundary for duplicate stream entries.
// Implements DESIGN-004 JobQueueManager.
type Job struct {
	ID            string
	EntryID       string
	EnqueuedAt    time.Time
	Attempt       int
	DeliveryCount int64
}

// QueueStats exposes stream lag, pending deliveries, and age information for
// capacity/readiness telemetry without exposing diet contents or user data.
// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector.
type QueueStats struct {
	StreamLength     int64
	QueueDepth       int64
	PendingDepth     int64
	OldestPendingAge time.Duration
	OldestQueuedAge  time.Duration
}

// JobQueueManager owns one Redis stream and one consumer group.
// Implements DESIGN-004 JobQueueManager.
type JobQueueManager struct {
	client    redis.UniversalClient
	config    Config
	telemetry *observability.OptimizationTelemetry

	bootstrapMu  sync.Mutex
	bootstrapped bool
}

// NewJobQueueManager constructs a Redis Streams queue manager. Configuration
// is validated when Bootstrap is called so construction remains non-blocking.
// Implements DESIGN-004 JobQueueManager.
func NewJobQueueManager(client redis.UniversalClient, config Config) *JobQueueManager {
	defaults := DefaultConfig()
	if config.Stream == "" {
		config.Stream = defaults.Stream
	}
	if config.Group == "" {
		config.Group = defaults.Group
	}
	if config.Consumer == "" {
		config.Consumer = defaults.Consumer
	}
	if config.VisibilityTimeout == 0 {
		config.VisibilityTimeout = defaults.VisibilityTimeout
	}
	if config.ReadBlock == 0 {
		config.ReadBlock = defaults.ReadBlock
	}
	if config.BatchSize == 0 {
		config.BatchSize = defaults.BatchSize
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = defaults.MaxAttempts
	}
	if config.CompletedTTL == 0 {
		config.CompletedTTL = defaults.CompletedTTL
	}
	if config.AttemptTTL == 0 {
		config.AttemptTTL = defaults.AttemptTTL
	}
	return &JobQueueManager{client: client, config: config}
}

// Config returns the normalized queue configuration.
// Implements DESIGN-004 JobQueueManager.
func (m *JobQueueManager) Config() Config {
	if m == nil {
		return Config{}
	}
	return m.config
}

// WithTelemetry attaches bounded queue and retry telemetry without changing
// queue request or delivery payloads.
// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector.
func (m *JobQueueManager) WithTelemetry(telemetry *observability.OptimizationTelemetry) *JobQueueManager {
	if m != nil {
		m.telemetry = telemetry
	}
	return m
}

// Bootstrap creates the stream and its sole consumer group idempotently. A
// BUSYGROUP response is success because another API or worker raced this call.
// Implements DESIGN-004 JobQueueManager stream/group bootstrap.
func (m *JobQueueManager) Bootstrap(ctx context.Context) error {
	if err := m.validate(); err != nil {
		return err
	}
	if err := contextError(ctx); err != nil {
		return err
	}

	m.bootstrapMu.Lock()
	if m.bootstrapped {
		m.bootstrapMu.Unlock()
		return nil
	}
	m.bootstrapMu.Unlock()

	err := m.client.XGroupCreateMkStream(ctx, m.config.Stream, m.config.Group, "0").Err()
	if err != nil && !isBusyGroup(err) {
		return unavailable("bootstrap Redis stream group", err)
	}

	m.bootstrapMu.Lock()
	m.bootstrapped = true
	m.bootstrapMu.Unlock()
	return nil
}

// Enqueue appends one server-created logical job ID with XADD. The stream
// payload deliberately contains no request body, user ID, or meal data.
// Implements DESIGN-004 JobQueueManager enqueue.
func (m *JobQueueManager) Enqueue(ctx context.Context, jobID string) (string, error) {
	if err := m.validate(); err != nil {
		return "", err
	}
	if err := contextError(ctx); err != nil {
		return "", err
	}
	if strings.TrimSpace(jobID) == "" || strings.ContainsAny(jobID, "\x00\r\n") {
		return "", fmt.Errorf("%w: job ID is invalid", ErrInvalidJob)
	}
	if err := m.Bootstrap(ctx); err != nil {
		return "", err
	}
	entryID, err := m.client.Eval(ctx, enqueueScript, []string{m.enqueueKey(jobID), m.config.Stream}, jobID, strconv.FormatInt(time.Now().UnixMilli(), 10), strconv.FormatInt(int64(m.config.AttemptTTL/time.Second), 10)).Text()
	if err != nil {
		return "", unavailable("enqueue optimization job", err)
	}
	if entryID == "" || entryID == "__pending__" {
		return "", unavailable("enqueue optimization job", errors.New("Redis returned an invalid queue entry"))
	}
	return entryID, nil
}

// Reserve obtains one abandoned delivery through XAUTOCLAIM before waiting
// for a new delivery through XREADGROUP. Attempts are counted per logical job
// so duplicate stream entries share the same retry budget.
// Implements DESIGN-004 JobQueueManager reserve and XAUTOCLAIM recovery.
func (m *JobQueueManager) Reserve(ctx context.Context) (Job, error) {
	if err := m.validate(); err != nil {
		return Job{}, err
	}
	if err := contextError(ctx); err != nil {
		return Job{}, err
	}
	if err := m.Bootstrap(ctx); err != nil {
		return Job{}, err
	}

	messages, err := m.claim(ctx, m.config.VisibilityTimeout)
	if err != nil {
		return Job{}, err
	}
	if len(messages) == 0 {
		result, readErr := m.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    m.config.Group,
			Consumer: m.config.Consumer,
			Streams:  []string{m.config.Stream, ">"},
			Count:    m.config.BatchSize,
			Block:    m.config.ReadBlock,
		}).Result()
		if errors.Is(readErr, redis.Nil) {
			return Job{}, ErrNoJob
		}
		if readErr != nil {
			if ctx.Err() != nil {
				return Job{}, ctx.Err()
			}
			return Job{}, unavailable("reserve optimization job", readErr)
		}
		if len(result) == 0 || len(result[0].Messages) == 0 {
			return Job{}, ErrNoJob
		}
		messages = result[0].Messages
	}
	return m.prepareDelivery(ctx, messages[0])
}

// Reclaim explicitly reclaims deliveries older than minIdle. Passing a
// non-positive duration uses the configured visibility timeout. This method
// is useful to workers with a scheduler and to deterministic integration
// tests; normal workers use Reserve's configured timeout.
// Implements DESIGN-004 JobQueueManager XAUTOCLAIM recovery.
func (m *JobQueueManager) Reclaim(ctx context.Context, minIdle time.Duration) ([]Job, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if err := m.Bootstrap(ctx); err != nil {
		return nil, err
	}
	if minIdle <= 0 {
		minIdle = m.config.VisibilityTimeout
	}
	messages, err := m.claim(ctx, minIdle)
	if err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(messages))
	for _, message := range messages {
		job, prepareErr := m.prepareDelivery(ctx, message)
		if prepareErr != nil {
			return nil, prepareErr
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// ProcessNext reserves and processes one job. It returns false with ErrNoJob
// when a bounded wait has no work. Processor failures remain pending until
// the next XAUTOCLAIM, except for the third failure which is terminally acked.
// Implements DESIGN-004 JobQueueManager worker execution.
func (m *JobQueueManager) ProcessNext(ctx context.Context, processor Processor) (bool, error) {
	job, err := m.Reserve(ctx)
	if errors.Is(err, ErrNoJob) {
		return false, ErrNoJob
	}
	if err != nil {
		return false, err
	}
	return true, m.Process(ctx, job, processor)
}

// Process executes one reserved job under a Redis lock. Successful and
// terminal deliveries use a Redis script to set the logical completion marker
// and XACK atomically, preventing concurrent duplicate authoritative work.
// Implements DESIGN-004 JobQueueManager at-least-once execution.
func (m *JobQueueManager) Process(ctx context.Context, job Job, processor Processor) error {
	if err := m.validate(); err != nil {
		return err
	}
	if processor == nil {
		return errors.New("optimization job processor is required")
	}
	if job.ID == "" || job.EntryID == "" {
		return ErrInvalidJob
	}
	if err := contextError(ctx); err != nil {
		return err
	}

	done, err := m.isDone(ctx, job.ID)
	if err != nil {
		return err
	}
	if done {
		return m.ack(ctx, job)
	}

	lockValue := processingLockValuePrefix + ":" + m.config.Consumer + ":" + job.EntryID
	locked, err := m.client.SetNX(ctx, m.lockKey(job.ID), lockValue, lockTTL(m.config.VisibilityTimeout)).Result()
	if err != nil {
		return unavailable("lock optimization job", err)
	}
	if !locked {
		// A duplicate stream entry is safe to acknowledge while the original
		// delivery holds the logical-job lock. The original remains pending and
		// can still be reclaimed if its consumer crashes.
		if err := m.ackDelivery(ctx, job.EntryID); err != nil {
			return unavailable("ack duplicate optimization delivery", err)
		}
		return nil
	}
	defer m.releaseLock(context.Background(), job.ID, lockValue)

	done, err = m.isDone(ctx, job.ID)
	if err != nil {
		return err
	}
	if done {
		return m.ack(ctx, job)
	}

	processErr := processor(ctx, job)
	if processErr == nil {
		return m.finalize(ctx, job, completedValue)
	}
	if ctx.Err() != nil {
		// A shutting-down worker leaves the delivery pending for another worker.
		// The processor still observed the caller's cancellation.
		return ctx.Err()
	}
	if job.Attempt < m.config.MaxAttempts && !errors.Is(processErr, context.Canceled) {
		return nil
	}
	if m.telemetry != nil {
		m.telemetry.Retry(ctx, "exhausted")
	}

	if m.config.TerminalHandler != nil {
		if err := m.config.TerminalHandler(ctx, job, processErr); err != nil {
			return err
		}
	}
	return m.finalize(ctx, job, failedValue)
}

// Ack terminally acknowledges a reserved delivery and records logical
// completion. It is intended for non-worker callers that have already applied
// an authoritative terminal update.
// Implements DESIGN-004 JobQueueManager XACK terminal handling.
func (m *JobQueueManager) Ack(ctx context.Context, job Job) error {
	if err := m.validate(); err != nil {
		return err
	}
	if job.ID == "" || job.EntryID == "" {
		return ErrInvalidJob
	}
	if err := m.Bootstrap(ctx); err != nil {
		return err
	}
	return m.finalize(ctx, job, completedValue)
}

// Run keeps one worker process consuming until its context is canceled. It
// never invokes the processor when Redis is unavailable.
// Implements DESIGN-004 JobQueueManager dedicated worker loop.
func (m *JobQueueManager) Run(ctx context.Context, processor Processor) error {
	if processor == nil {
		return errors.New("optimization job processor is required")
	}
	if err := m.Bootstrap(ctx); err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return nil
		}
		_, err := m.ProcessNext(ctx, processor)
		if errors.Is(err, ErrNoJob) {
			continue
		}
		if errors.Is(err, ErrJobInProgress) {
			continue
		}
		if errors.Is(err, ErrInvalidJob) {
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if ctx.Err() != nil {
				return nil
			}
		}
		if err != nil {
			return err
		}
	}
}

// Stats reports stream length, group lag, pending depth, and the age of the
// oldest queued/pending entries.
// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector.
func (m *JobQueueManager) Stats(ctx context.Context) (QueueStats, error) {
	if err := m.validate(); err != nil {
		return QueueStats{}, err
	}
	if err := m.Bootstrap(ctx); err != nil {
		return QueueStats{}, err
	}
	stream, err := m.client.XInfoStream(ctx, m.config.Stream).Result()
	if err != nil {
		return QueueStats{}, unavailable("read optimization stream stats", err)
	}
	pending, err := m.client.XPending(ctx, m.config.Stream, m.config.Group).Result()
	if err != nil {
		return QueueStats{}, unavailable("read optimization pending stats", err)
	}
	groups, err := m.client.XInfoGroups(ctx, m.config.Stream).Result()
	if err != nil {
		return QueueStats{}, unavailable("read optimization group stats", err)
	}
	var lag int64
	for _, group := range groups {
		if group.Name == m.config.Group {
			lag = group.Lag
			break
		}
	}
	if lag < 0 {
		lag = 0
	}
	stats := QueueStats{
		StreamLength: stream.Length,
		QueueDepth:   pending.Count + lag,
		PendingDepth: pending.Count,
	}
	if pending.Count > 0 {
		oldest, pendingErr := m.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: m.config.Stream,
			Group:  m.config.Group,
			Start:  "-",
			End:    "+",
			Count:  1,
		}).Result()
		if pendingErr != nil {
			return QueueStats{}, unavailable("read oldest optimization delivery", pendingErr)
		}
		if len(oldest) > 0 {
			stats.OldestPendingAge = streamEntryAge(oldest[0].ID)
		}
	}
	if stream.FirstEntry.ID != "" {
		stats.OldestQueuedAge = streamEntryAge(stream.FirstEntry.ID)
	}
	if m.telemetry != nil {
		m.telemetry.QueueStats(ctx, stats.QueueDepth, stats.OldestQueuedAge, stats.OldestPendingAge)
	}
	return stats, nil
}

// claim reclaims pending deliveries older than minIdle for this consumer.
// Implements DESIGN-004 JobQueueManager XAUTOCLAIM recovery.
func (m *JobQueueManager) claim(ctx context.Context, minIdle time.Duration) ([]redis.XMessage, error) {
	messages, _, err := m.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   m.config.Stream,
		Group:    m.config.Group,
		Consumer: m.config.Consumer,
		MinIdle:  minIdle,
		Start:    "0-0",
		Count:    m.config.BatchSize,
	}).Result()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, unavailable("reclaim optimization delivery", err)
	}
	return messages, nil
}

// prepareDelivery decodes a stream entry and increments its logical retry count.
// Implements DESIGN-004 JobQueueManager attempt counting.
func (m *JobQueueManager) prepareDelivery(ctx context.Context, message redis.XMessage) (Job, error) {
	job, err := decodeJob(message)
	if err != nil {
		ackErr := m.client.XAck(ctx, m.config.Stream, m.config.Group, message.ID).Err()
		if ackErr != nil {
			return Job{}, unavailable("ack malformed optimization delivery", ackErr)
		}
		return Job{}, err
	}
	attempt, err := m.client.Incr(ctx, m.attemptKey(job.ID)).Result()
	if err != nil {
		return Job{}, unavailable("count optimization attempt", err)
	}
	if expireErr := m.client.Expire(ctx, m.attemptKey(job.ID), m.config.AttemptTTL).Err(); expireErr != nil {
		return Job{}, unavailable("expire optimization attempt", expireErr)
	}
	job.Attempt = int(attempt)
	if attempt > 1 && m.telemetry != nil {
		m.telemetry.Retry(ctx, "retry")
	}
	return job, nil
}

// decodeJob maps the untrusted stream fields into a server-owned Job.
// Implements DESIGN-004 JobQueueManager stream payload handling.
func decodeJob(message redis.XMessage) (Job, error) {
	jobID, ok := streamValueString(message.Values[jobIDField])
	if !ok || strings.TrimSpace(jobID) == "" || strings.ContainsAny(jobID, "\x00\r\n") {
		return Job{}, fmt.Errorf("%w: missing job ID", ErrInvalidJob)
	}
	enqueuedAt := streamEntryTime(message.ID)
	if value, ok := streamValueString(message.Values[enqueuedAtField]); ok {
		if millis, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
			enqueuedAt = time.UnixMilli(millis)
		}
	}
	return Job{
		ID:            jobID,
		EntryID:       message.ID,
		EnqueuedAt:    enqueuedAt,
		DeliveryCount: message.DeliveredCount,
	}, nil
}

// streamValueString accepts the string representations returned by go-redis.
// Implements DESIGN-004 JobQueueManager stream payload handling.
func streamValueString(value interface{}) (string, bool) {
	switch value := value.(type) {
	case string:
		return value, true
	case []byte:
		return string(value), true
	default:
		return "", false
	}
}

// finalize atomically records terminal state, acknowledges, and removes a delivery.
// Implements DESIGN-004 JobQueueManager XACK terminal handling.
func (m *JobQueueManager) finalize(ctx context.Context, job Job, value string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	result, err := m.client.Eval(ctx, finalizeScript, []string{m.doneKey(job.ID), m.config.Stream}, value, strconv.FormatInt(int64(m.config.CompletedTTL/time.Second), 10), m.config.Group, job.EntryID).Int64()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return unavailable("ack optimization delivery", err)
	}
	if result < 0 {
		return unavailable("ack optimization delivery", errors.New("Redis did not acknowledge delivery"))
	}
	return nil
}

// ack records successful logical completion and acknowledges the delivery.
// Implements DESIGN-004 JobQueueManager XACK terminal handling.
func (m *JobQueueManager) ack(ctx context.Context, job Job) error {
	return m.finalize(ctx, job, completedValue)
}

// ackDelivery acknowledges and removes a duplicate delivery that is covered by
// another consumer's logical-job lock.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func (m *JobQueueManager) ackDelivery(ctx context.Context, entryID string) error {
	if err := m.client.XAck(ctx, m.config.Stream, m.config.Group, entryID).Err(); err != nil {
		return err
	}
	return m.client.XDel(ctx, m.config.Stream, entryID).Err()
}

// isDone checks the logical completion marker used for duplicate safety.
// Implements DESIGN-004 JobQueueManager at-least-once execution.
func (m *JobQueueManager) isDone(ctx context.Context, jobID string) (bool, error) {
	result, err := m.client.Exists(ctx, m.doneKey(jobID)).Result()
	if err != nil {
		return false, unavailable("read optimization completion marker", err)
	}
	return result == 1, nil
}

// releaseLock deletes a lock only when its owner token still matches.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func (m *JobQueueManager) releaseLock(ctx context.Context, jobID, value string) {
	_, _ = m.client.Eval(ctx, releaseLockScript, []string{m.lockKey(jobID)}, value).Result()
}

// attemptKey returns the hashed Redis key for logical retry state.
// Implements DESIGN-004 JobQueueManager attempt counting.
func (m *JobQueueManager) attemptKey(jobID string) string {
	return m.key("attempt", jobID)
}

// doneKey returns the hashed Redis key for logical terminal state.
// Implements DESIGN-004 JobQueueManager terminal handling.
func (m *JobQueueManager) doneKey(jobID string) string {
	return m.key("done", jobID)
}

// lockKey returns the hashed Redis key for logical processing ownership.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func (m *JobQueueManager) lockKey(jobID string) string {
	return m.key("lock", jobID)
}

// enqueueKey returns the cross-process logical publication marker for one job.
// Implements DESIGN-004 JobQueueManager idempotent enqueue.
func (m *JobQueueManager) enqueueKey(jobID string) string {
	return m.key("enqueue", jobID)
}

// key derives a bounded, non-PII Redis key from the server job ID.
// Implements DESIGN-004 JobQueueManager Redis state isolation.
func (m *JobQueueManager) key(kind, jobID string) string {
	digest := sha256.Sum256([]byte(jobID))
	return "mealswapp:optimization:queue:v1:" + kind + ":" + hex.EncodeToString(digest[:])
}

// validate checks queue configuration before any Redis command is issued.
// Implements DESIGN-004 JobQueueManager configuration validation.
func (m *JobQueueManager) validate() error {
	if m == nil || m.client == nil {
		return unavailable("use optimization queue", errors.New("Redis client is required"))
	}
	if strings.TrimSpace(m.config.Stream) == "" || strings.TrimSpace(m.config.Group) == "" || strings.TrimSpace(m.config.Consumer) == "" {
		return errors.New("optimization queue stream, group, and consumer are required")
	}
	if m.config.VisibilityTimeout <= minimumVisibilityTimeout {
		return fmt.Errorf("optimization queue visibility timeout must be greater than %s", minimumVisibilityTimeout)
	}
	if m.config.ReadBlock < 0 || m.config.BatchSize <= 0 || m.config.MaxAttempts <= 0 || m.config.CompletedTTL <= 0 || m.config.AttemptTTL <= 0 {
		return errors.New("optimization queue timing and retry settings are invalid")
	}
	return nil
}

// contextError returns cancellation without converting it into queue outage.
// Implements DESIGN-004 JobQueueManager cancellation propagation.
func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("optimization queue context is required")
	}
	return ctx.Err()
}

// unavailable wraps a Redis failure in the API-visible queue outage sentinel.
// Implements DESIGN-004 JobQueueManager queue-unavailable behavior.
func unavailable(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrQueueUnavailable, operation, err)
}

// isBusyGroup recognizes idempotent consumer-group bootstrap races.
// Implements DESIGN-004 JobQueueManager stream/group bootstrap.
func isBusyGroup(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "BUSYGROUP")
}

// lockTTL expires logical ownership just before the visibility timeout.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func lockTTL(visibility time.Duration) time.Duration {
	ttl := visibility - time.Second
	if ttl <= 0 {
		return time.Second
	}
	return ttl
}

// streamEntryTime parses the millisecond timestamp portion of a stream ID.
// Implements DESIGN-004 JobQueueManager queue-age observability.
func streamEntryTime(id string) time.Time {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		return time.Time{}
	}
	millis, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || millis <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(millis)
}

// streamEntryAge returns a non-negative age for a Redis stream entry.
// Implements DESIGN-004 JobQueueManager queue-age observability.
func streamEntryAge(id string) time.Duration {
	created := streamEntryTime(id)
	if created.IsZero() {
		return 0
	}
	age := time.Since(created)
	if age < 0 {
		return 0
	}
	return age
}

// finalizeScript atomically marks logical state, XACKs, and removes a delivery.
// Implements DESIGN-004 JobQueueManager XACK terminal handling.
const finalizeScript = `
if redis.call('exists', KEYS[1]) == 0 then
  redis.call('set', KEYS[1], ARGV[1], 'ex', ARGV[2])
end
local acknowledged = redis.call('xack', KEYS[2], ARGV[3], ARGV[4])
redis.call('xdel', KEYS[2], ARGV[4])
return acknowledged
`

// enqueueScript publishes one stream entry per logical job ID across all API processes.
// Implements DESIGN-004 JobQueueManager idempotent enqueue.
const enqueueScript = `
local existing = redis.call('get', KEYS[1])
if existing then
  return existing
end
local entry = redis.pcall('xadd', KEYS[2], '*', 'job_id', ARGV[1], 'enqueued_at', ARGV[2])
if type(entry) == 'table' and entry.err then
  return redis.error_reply(entry.err)
end
redis.call('set', KEYS[1], entry, 'ex', ARGV[3])
return entry
`

// releaseLockScript prevents one worker from deleting another worker's lock.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
const releaseLockScript = `
if redis.call('get', KEYS[1]) == ARGV[1] then
  return redis.call('del', KEYS[1])
end
return 0
`

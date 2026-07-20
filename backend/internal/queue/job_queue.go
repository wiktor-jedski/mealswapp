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
	// DefaultStream is the Redis Stream used for optimization job delivery.
	DefaultStream = "mealswapp:optimization:{queue-v1}:jobs"
	// DefaultGroup is the Redis consumer group shared by optimization workers.
	DefaultGroup = "optimization-workers"
	// DefaultVisibilityTimeout bounds one reserved delivery before reclamation.
	DefaultVisibilityTimeout = 45 * time.Second
	// DefaultReadBlock bounds one blocking Redis Stream read.
	DefaultReadBlock = time.Second
	// DefaultBatchSize limits each reservation to one optimization job.
	DefaultBatchSize = int64(1)
	// DefaultMaxAttempts limits terminal processing attempts per job.
	DefaultMaxAttempts = 3
	// DefaultCompletedTTL retains terminal job completion markers.
	DefaultCompletedTTL = time.Hour
	// DefaultAttemptTTL retains bounded delivery-attempt counters.
	DefaultAttemptTTL          = 24 * time.Hour
	optimizationWorkTimeout    = 30 * time.Second
	optimizationFinalizeBudget = 5 * time.Second
	processingLockMargin       = time.Second
	lockCleanupTimeout         = 100 * time.Millisecond
	jobIDField                 = "job_id"
	enqueuedAtField            = "enqueued_at"
	completedValue             = "completed"
	failedValue                = "failed"
	processingLockValuePrefix  = "consumer"
)

// ErrNoJob means that a bounded reservation wait returned no message.
// Implements DESIGN-004 JobQueueManager.
var ErrNoJob = errors.New("no optimization job available")

// ErrInvalidJob means that a stream entry did not contain a valid job ID.
// Implements DESIGN-004 JobQueueManager.
var ErrInvalidJob = errors.New("invalid optimization job stream entry")

// ErrQueueUnavailable identifies Redis connection and command failures. The
// API must map this error to an unavailable queue response and must not invoke
// a synchronous solver fallback.
// Implements DESIGN-004 JobQueueManager.
var ErrQueueUnavailable = errors.New("optimization queue unavailable")

// ErrTerminalPublicationRequired leaves a delivery pending when worker code
// has not explicitly confirmed an authoritative completed or failed record.
// Implements DESIGN-004 JobQueueManager terminal publication contract.
var ErrTerminalPublicationRequired = errors.New("optimization terminal publication is required")

// TerminalPublication identifies the authoritative status written before ACK.
// Implements DESIGN-004 JobQueueManager terminal publication contract.
type TerminalPublication string

// Terminal publication values distinguish completed and failed acknowledgement.
// Implements DESIGN-004 JobQueueManager terminal publication contract.
const (
	// PublishedCompleted confirms an authoritative completed record exists.
	PublishedCompleted TerminalPublication = completedValue
	// PublishedFailed confirms an authoritative failed record exists.
	PublishedFailed TerminalPublication = failedValue
)

// Processor is the worker-only alternative-generation boundary. It receives
// only a server-created job ID; request data and authoritative publication
// remain outside the Redis stream payload.
// Implements DESIGN-004 JobQueueManager and LPSolverWrapper worker boundary.
type Processor func(context.Context, Job) (TerminalPublication, error)

// TerminalHandler records a terminal failure or cancellation before the queue
// acknowledges the delivery, so the handler must make its status update
// idempotent for a crash between publication and XACK.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
type TerminalHandler func(context.Context, Job, error) (TerminalPublication, error)

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
	defer m.bootstrapMu.Unlock()
	if m.bootstrapped {
		return nil
	}

	err := m.client.XGroupCreateMkStream(ctx, m.config.Stream, m.config.Group, "0").Err()
	if err != nil && !isBusyGroup(err) {
		return unavailable("bootstrap Redis stream group", err)
	}

	m.bootstrapped = true
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
	if !canonicalJobID(jobID) {
		return "", fmt.Errorf("%w: job ID is invalid", ErrInvalidJob)
	}
	if err := m.Bootstrap(ctx); err != nil {
		return "", err
	}
	entryID, err := enqueueScript.Run(ctx, m.client, []string{m.enqueueKey(jobID), m.config.Stream}, jobID, strconv.FormatInt(time.Now().UnixMilli(), 10), strconv.FormatInt(m.config.AttemptTTL.Milliseconds(), 10)).Text()
	if err != nil {
		return "", unavailable("enqueue optimization job", err)
	}
	if entryID == "" {
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
		result, readErr := m.readNew(ctx)
		if isNoGroup(readErr) {
			if recoverErr := m.recoverGroup(ctx); recoverErr != nil {
				return Job{}, recoverErr
			}
			result, readErr = m.readNew(ctx)
		}
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
// tests; normal workers use Reserve's configured timeout. If preparation ever
// receives multiple deliveries, it returns the valid prefix with the error.
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
	return m.prepareDeliveries(ctx, messages)
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
	if !canonicalJobID(job.ID) || job.EntryID == "" {
		return ErrInvalidJob
	}
	if err := contextError(ctx); err != nil {
		return err
	}

	done, err := m.doneState(ctx, job.ID)
	if err != nil {
		return err
	}
	if done.valid() {
		return m.finalize(ctx, job, done)
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
		if err := m.removeDelivery(ctx, job.EntryID); err != nil {
			return unavailable("ack duplicate optimization delivery", err)
		}
		return nil
	}
	defer m.releaseLock(ctx, job.ID, lockValue)

	done, err = m.doneState(ctx, job.ID)
	if err != nil {
		return err
	}
	if done.valid() {
		return m.finalize(ctx, job, done)
	}

	attempt, err := m.countAttempt(ctx, job.ID)
	if err != nil {
		return err
	}
	job.Attempt = attempt
	if attempt > 1 && m.telemetry != nil {
		m.telemetry.Retry(ctx, "retry")
	}

	publication, processErr := processor(ctx, job)
	if processErr == nil {
		if !publication.valid() {
			return ErrTerminalPublicationRequired
		}
		return m.finalize(ctx, job, publication)
	}
	if publication.valid() {
		return errors.New("optimization processor returned publication with an error")
	}
	if ctx.Err() != nil {
		// A shutting-down worker leaves the delivery pending for another worker.
		// The processor still observed the caller's cancellation.
		return ctx.Err()
	}
	if job.Attempt < m.config.MaxAttempts {
		return nil
	}
	if m.telemetry != nil {
		m.telemetry.Retry(ctx, "exhausted")
	}

	if m.config.TerminalHandler == nil {
		return ErrTerminalPublicationRequired
	}
	publication, err = m.config.TerminalHandler(ctx, job, processErr)
	if err != nil {
		return err
	}
	if publication != PublishedFailed {
		return ErrTerminalPublicationRequired
	}
	return m.finalize(ctx, job, publication)
}

// AckCompleted acknowledges only after authoritative completed publication.
// Implements DESIGN-004 JobQueueManager XACK terminal handling.
func (m *JobQueueManager) AckCompleted(ctx context.Context, job Job) error {
	return m.ackPublished(ctx, job, PublishedCompleted)
}

// AckFailed acknowledges only after authoritative failed publication.
// Implements DESIGN-004 JobQueueManager XACK terminal handling.
func (m *JobQueueManager) AckFailed(ctx context.Context, job Job) error {
	return m.ackPublished(ctx, job, PublishedFailed)
}

// ackPublished validates direct acknowledgement before atomic finalization.
// Implements DESIGN-004 JobQueueManager terminal publication contract.
func (m *JobQueueManager) ackPublished(ctx context.Context, job Job, publication TerminalPublication) error {
	if err := m.validate(); err != nil {
		return err
	}
	if !canonicalJobID(job.ID) || job.EntryID == "" {
		return ErrInvalidJob
	}
	if err := m.Bootstrap(ctx); err != nil {
		return err
	}
	return m.finalize(ctx, job, publication)
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
	if isNoGroup(err) {
		if recoverErr := m.recoverGroup(ctx); recoverErr != nil {
			return QueueStats{}, recoverErr
		}
		pending, err = m.client.XPending(ctx, m.config.Stream, m.config.Group).Result()
	}
	if err != nil {
		return QueueStats{}, unavailable("read optimization pending stats", err)
	}
	groups, err := m.client.XInfoGroups(ctx, m.config.Stream).Result()
	if err != nil {
		return QueueStats{}, unavailable("read optimization group stats", err)
	}
	var group redis.XInfoGroup
	foundGroup := false
	for _, candidate := range groups {
		if candidate.Name == m.config.Group {
			foundGroup = true
			group = candidate
			break
		}
	}
	if !foundGroup || group.Lag < 0 || group.LastDeliveredID == "" {
		return QueueStats{}, unavailable("read optimization group stats", errors.New("Redis returned incomplete consumer-group metadata"))
	}
	stats := QueueStats{
		StreamLength: stream.Length,
		QueueDepth:   pending.Count + group.Lag,
		PendingDepth: pending.Count,
	}
	if pending.Count > 0 {
		stats.OldestPendingAge, err = m.oldestPendingIdle(ctx)
		if err != nil {
			return QueueStats{}, err
		}
	}
	if group.Lag > 0 {
		waiting, waitingErr := m.client.XRangeN(ctx, m.config.Stream, "("+group.LastDeliveredID, "+", 1).Result()
		if waitingErr != nil {
			return QueueStats{}, unavailable("read oldest waiting optimization delivery", waitingErr)
		}
		if len(waiting) != 1 {
			return QueueStats{}, unavailable("read oldest waiting optimization delivery", errors.New("Redis lag did not identify a waiting entry"))
		}
		stats.OldestQueuedAge = streamEntryAge(waiting[0].ID)
	}
	if m.telemetry != nil {
		m.telemetry.QueueStats(ctx, stats.QueueDepth, stats.OldestQueuedAge, stats.OldestPendingAge)
	}
	return stats, nil
}

// oldestPendingIdle scans Redis pending metadata for the longest authoritative
// delivery idle duration. Stream creation time is not pending age.
// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector.
func (m *JobQueueManager) oldestPendingIdle(ctx context.Context) (time.Duration, error) {
	const pageSize = int64(100)
	start := "-"
	var oldest time.Duration
	for {
		entries, err := m.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: m.config.Stream, Group: m.config.Group,
			Start: start, End: "+", Count: pageSize,
		}).Result()
		if err != nil {
			return 0, unavailable("read oldest pending optimization delivery", err)
		}
		for _, entry := range entries {
			if entry.Idle > oldest {
				oldest = entry.Idle
			}
		}
		if len(entries) < int(pageSize) {
			return oldest, nil
		}
		start = "(" + entries[len(entries)-1].ID
	}
}

// claim reclaims pending deliveries older than minIdle for this consumer.
// Implements DESIGN-004 JobQueueManager XAUTOCLAIM recovery.
func (m *JobQueueManager) claim(ctx context.Context, minIdle time.Duration) ([]redis.XMessage, error) {
	messages, _, err := m.autoClaim(ctx, minIdle)
	if isNoGroup(err) {
		if recoverErr := m.recoverGroup(ctx); recoverErr != nil {
			return nil, recoverErr
		}
		messages, _, err = m.autoClaim(ctx, minIdle)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, unavailable("reclaim optimization delivery", err)
	}
	return messages, nil
}

// autoClaim performs one bounded Redis pending-delivery reclaim command.
// Implements DESIGN-004 JobQueueManager XAUTOCLAIM recovery.
func (m *JobQueueManager) autoClaim(ctx context.Context, minIdle time.Duration) ([]redis.XMessage, string, error) {
	return m.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   m.config.Stream,
		Group:    m.config.Group,
		Consumer: m.config.Consumer,
		MinIdle:  minIdle,
		Start:    "0-0",
		Count:    m.config.BatchSize,
	}).Result()
}

// readNew performs one bounded Redis consumer-group read for new deliveries.
// Implements DESIGN-004 JobQueueManager stream/group recovery.
func (m *JobQueueManager) readNew(ctx context.Context) ([]redis.XStream, error) {
	return m.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: m.config.Group, Consumer: m.config.Consumer,
		Streams: []string{m.config.Stream, ">"}, Count: m.config.BatchSize, Block: m.config.ReadBlock,
	}).Result()
}

// recoverGroup invalidates only local bootstrap state after a Redis NOGROUP
// response, then recreates the stream/group once. Other errors are never used
// as recovery signals and therefore fail closed.
// Implements DESIGN-004 JobQueueManager live stream/group recovery.
func (m *JobQueueManager) recoverGroup(ctx context.Context) error {
	m.bootstrapMu.Lock()
	m.bootstrapped = false
	m.bootstrapMu.Unlock()
	return m.Bootstrap(ctx)
}

// prepareDeliveries returns the successfully decoded prefix with any later
// error so callers never lose deliveries already reclaimed from Redis.
// Implements DESIGN-004 JobQueueManager reclaim partial-failure handling.
func (m *JobQueueManager) prepareDeliveries(ctx context.Context, messages []redis.XMessage) ([]Job, error) {
	jobs := make([]Job, 0, len(messages))
	for _, message := range messages {
		job, err := m.prepareDelivery(ctx, message)
		if err != nil {
			return jobs, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// prepareDelivery decodes one untrusted stream entry. Processing attempts are
// counted later, only after Process acquires logical-job ownership.
// Implements DESIGN-004 JobQueueManager delivery validation.
func (m *JobQueueManager) prepareDelivery(ctx context.Context, message redis.XMessage) (Job, error) {
	job, err := decodeJob(message)
	if err != nil {
		ackErr := m.removeDelivery(ctx, message.ID)
		if ackErr != nil {
			return Job{}, unavailable("remove malformed optimization delivery", ackErr)
		}
		return Job{}, err
	}
	return job, nil
}

// decodeJob maps the untrusted stream fields into a server-owned Job.
// Implements DESIGN-004 JobQueueManager stream payload handling.
func decodeJob(message redis.XMessage) (Job, error) {
	jobID, ok := streamValueString(message.Values[jobIDField])
	if !ok || !canonicalJobID(jobID) {
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

// canonicalJobID accepts only the lowercase hyphenated non-nil UUID form
// produced by server-side job creation.
// Implements DESIGN-004 JobQueueManager UUID validation.
func canonicalJobID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}

// countAttempt atomically advances and expires retry state after ownership.
// Implements DESIGN-004 JobQueueManager ownership-based attempt counting.
func (m *JobQueueManager) countAttempt(ctx context.Context, jobID string) (int, error) {
	attempt, err := countAttemptScript.Run(ctx, m.client, []string{m.attemptKey(jobID)}, strconv.FormatInt(m.config.AttemptTTL.Milliseconds(), 10)).Int64()
	if err != nil {
		return 0, unavailable("count optimization attempt", err)
	}
	return int(attempt), nil
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
func (m *JobQueueManager) finalize(ctx context.Context, job Job, publication TerminalPublication) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if !publication.valid() {
		return ErrTerminalPublicationRequired
	}
	result, err := finalizeScript.Run(ctx, m.client, []string{m.doneKey(job.ID), m.config.Stream}, string(publication), strconv.FormatInt(m.config.CompletedTTL.Milliseconds(), 10), m.config.Group, job.EntryID).Int64()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return unavailable("ack optimization delivery", err)
	}
	if result == -1 {
		return unavailable("ack optimization delivery", errors.New("delivery was not pending and had no terminal marker"))
	}
	if result == -2 {
		return unavailable("ack optimization delivery", errors.New("terminal publication conflicts with existing marker"))
	}
	return nil
}

// removeDelivery atomically acknowledges and removes duplicate or malformed
// input. A zero/zero result is successful idempotent replay.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func (m *JobQueueManager) removeDelivery(ctx context.Context, entryID string) error {
	_, err := removeDeliveryScript.Run(ctx, m.client, []string{m.config.Stream}, m.config.Group, entryID).Result()
	return err
}

// doneState reads the logical terminal marker used for duplicate safety.
// Implements DESIGN-004 JobQueueManager at-least-once execution.
func (m *JobQueueManager) doneState(ctx context.Context, jobID string) (TerminalPublication, error) {
	result, err := m.client.Get(ctx, m.doneKey(jobID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", unavailable("read optimization completion marker", err)
	}
	publication := TerminalPublication(result)
	if !publication.valid() {
		return "", unavailable("read optimization completion marker", errors.New("invalid terminal marker"))
	}
	return publication, nil
}

// releaseLock deletes a lock only when its owner token still matches.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func (m *JobQueueManager) releaseLock(ctx context.Context, jobID, value string) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), lockCleanupTimeout)
	defer cancel()
	if _, err := releaseLockScript.Run(cleanupCtx, m.client, []string{m.lockKey(jobID)}, value).Result(); err != nil && m.telemetry != nil {
		m.telemetry.QueueCleanupFailed(ctx)
	}
}

// valid reports whether the processor confirmed a supported terminal state.
// Implements DESIGN-004 JobQueueManager terminal publication contract.
func (p TerminalPublication) valid() bool {
	return p == PublishedCompleted || p == PublishedFailed
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
	return "mealswapp:optimization:{" + streamHashTag(m.config.Stream) + "}:" + kind + ":" + hex.EncodeToString(digest[:])
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
	if streamHashTag(m.config.Stream) == "" {
		return errors.New("optimization queue stream requires a Redis Cluster hash tag")
	}
	if lockTTL(m.config.VisibilityTimeout).Truncate(time.Millisecond) <= optimizationWorkTimeout+optimizationFinalizeBudget {
		return fmt.Errorf("optimization queue visibility timeout must preserve the %s processing and finalization window plus the %s lock margin", optimizationWorkTimeout+optimizationFinalizeBudget, processingLockMargin)
	}
	if m.config.ReadBlock < 0 || m.config.BatchSize != DefaultBatchSize || m.config.MaxAttempts <= 0 || m.config.CompletedTTL < time.Millisecond || m.config.AttemptTTL < time.Millisecond {
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
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(err.Error()), "BUSYGROUP")
}

// isNoGroup recognizes only Redis's missing consumer-group response.
// Implements DESIGN-004 JobQueueManager live stream/group recovery.
func isNoGroup(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "NOGROUP")
}

// streamHashTag extracts the non-empty Redis Cluster hash tag from a key.
// Implements DESIGN-004 JobQueueManager Redis key topology.
func streamHashTag(key string) string {
	start := strings.IndexByte(key, '{')
	if start < 0 {
		return ""
	}
	remainder := key[start+1:]
	end := strings.IndexByte(remainder, '}')
	if end <= 0 {
		return ""
	}
	return remainder[:end]
}

// lockTTL expires logical ownership just before the visibility timeout.
// Implements DESIGN-004 JobQueueManager duplicate-delivery safety.
func lockTTL(visibility time.Duration) time.Duration {
	ttl := visibility - processingLockMargin
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

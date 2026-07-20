package queue

// Implements DESIGN-004 JobQueueManager real-Redis integration verification.

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

func openQueueIntegrationRedis(t *testing.T) *redis.Client {
	t.Helper()
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Skipf("Redis integration URL is invalid: %v", err)
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Skipf("Redis integration service unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func newIntegrationQueue(t *testing.T, client redis.UniversalClient) *JobQueueManager {
	t.Helper()
	stream := "mealswapp:test:optimization:{" + uuid.NewString() + "}"
	manager := NewJobQueueManager(client, Config{
		Stream:            stream,
		Group:             "optimization-workers",
		Consumer:          "test-" + uuid.NewString(),
		VisibilityTimeout: DefaultVisibilityTimeout,
		ReadBlock:         10 * time.Millisecond,
		BatchSize:         1,
		MaxAttempts:       3,
		CompletedTTL:      time.Minute,
		AttemptTTL:        time.Hour,
	})
	t.Cleanup(func() { _ = client.Del(context.Background(), stream).Err() })
	return manager
}

// TestJobQueueBootstrapIsIdempotentAndUsesOneConsumerGroup verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004, and SW-REQ-080/SW-REQ-082.
func TestJobQueueBootstrapIsIdempotentAndUsesOneConsumerGroup(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap() first error = %v", err)
	}
	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap() second error = %v", err)
	}
	other := NewJobQueueManager(client, Config{
		Stream:            manager.Config().Stream,
		Group:             manager.Config().Group,
		Consumer:          "test-" + uuid.NewString(),
		VisibilityTimeout: manager.Config().VisibilityTimeout,
		ReadBlock:         manager.Config().ReadBlock,
	})
	if err := other.Bootstrap(context.Background()); err != nil {
		t.Fatalf("racing Bootstrap() error = %v", err)
	}
	groups, err := client.XInfoGroups(context.Background(), manager.Config().Stream).Result()
	if err != nil {
		t.Fatalf("XInfoGroups() error = %v", err)
	}
	if len(groups) != 1 || groups[0].Name != manager.Config().Group {
		t.Fatalf("groups = %#v, want one %q group", groups, manager.Config().Group)
	}
}

// TestJobQueueEnqueueReserveAndAckUseRedisStreams verifies IT-ARCH-004-003,
// ARCH-004, DESIGN-004, and SW-REQ-021.
func TestJobQueueEnqueueReserveAndAckUseRedisStreams(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	jobID := uuid.NewString()
	entryID, err := manager.Enqueue(ctx, jobID)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if job.ID != jobID || job.EntryID != entryID || job.Attempt != 0 || job.EnqueuedAt.IsZero() {
		t.Fatalf("reserved job = %#v, want ID=%q entry=%q uncounted attempt with timestamp", job, jobID, entryID)
	}
	if err := manager.AckCompleted(ctx, job); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
	pending, err := client.XPending(ctx, manager.Config().Stream, manager.Config().Group).Result()
	if err != nil {
		t.Fatalf("XPending() error = %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending count = %d, want 0 after XACK", pending.Count)
	}
}

// TestJobQueueEnqueueIsIdempotentPerLogicalJob verifies IT-ARCH-004-003 cross-process publication deduplication.
func TestJobQueueEnqueueIsIdempotentPerLogicalJob(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	jobID := uuid.NewString()
	first, err := manager.Enqueue(context.Background(), jobID)
	if err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	second, err := manager.Enqueue(context.Background(), jobID)
	if err != nil {
		t.Fatalf("second Enqueue() error = %v", err)
	}
	if first != second {
		t.Fatalf("entry IDs = %q, %q, want one logical publication", first, second)
	}
	entries, err := client.XRange(context.Background(), manager.Config().Stream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("stream entries = %d, want one", len(entries))
	}
}

// TestJobQueueConcurrentConsumersDoNotProcessDuplicateLogicalJob verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004, and SW-REQ-021/SW-REQ-030.
func TestJobQueueConcurrentConsumersDoNotProcessDuplicateLogicalJob(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	first := newIntegrationQueue(t, client)
	second := NewJobQueueManager(client, Config{
		Stream:            first.Config().Stream,
		Group:             first.Config().Group,
		Consumer:          "test-" + uuid.NewString(),
		VisibilityTimeout: first.Config().VisibilityTimeout,
		ReadBlock:         first.Config().ReadBlock,
	})
	if err := second.Bootstrap(context.Background()); err != nil {
		t.Fatalf("second Bootstrap() error = %v", err)
	}
	jobID := uuid.NewString()
	if _, err := first.Enqueue(context.Background(), jobID); err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	if _, err := first.Enqueue(context.Background(), jobID); err != nil {
		t.Fatalf("duplicate Enqueue() error = %v", err)
	}

	var processed atomic.Int32
	processor := func(context.Context, Job) (TerminalPublication, error) {
		processed.Add(1)
		time.Sleep(50 * time.Millisecond)
		return PublishedCompleted, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	var wg sync.WaitGroup
	for _, manager := range []*JobQueueManager{first, second} {
		wg.Add(1)
		go func(manager *JobQueueManager) {
			defer wg.Done()
			if err := manager.Run(ctx, processor); err != nil {
				t.Errorf("Run() error = %v", err)
			}
		}(manager)
	}
	wg.Wait()
	if got := processed.Load(); got != 1 {
		t.Fatalf("processor calls = %d, want one authoritative processing", got)
	}
	pending, err := client.XPending(context.Background(), first.Config().Stream, first.Config().Group).Result()
	if err != nil {
		t.Fatalf("XPending() error = %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending count = %d, want duplicate delivery acknowledged", pending.Count)
	}
}

// TestJobQueueReclaimsAbandonedDeliveryWithXAUTOCLAIM verifies IT-ARCH-004-003,
// ARCH-004, DESIGN-004, and SW-REQ-080/SW-REQ-082.
func TestJobQueueReclaimsAbandonedDeliveryWithXAUTOCLAIM(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	first := newIntegrationQueue(t, client)
	second := NewJobQueueManager(client, Config{
		Stream:            first.Config().Stream,
		Group:             first.Config().Group,
		Consumer:          "test-" + uuid.NewString(),
		VisibilityTimeout: first.Config().VisibilityTimeout,
		ReadBlock:         first.Config().ReadBlock,
	})
	jobID := uuid.NewString()
	if _, err := first.Enqueue(context.Background(), jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	abandoned, err := first.Reserve(context.Background())
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	reclaimed, err := second.Reclaim(context.Background(), time.Millisecond)
	if err != nil {
		t.Fatalf("Reclaim() error = %v", err)
	}
	if len(reclaimed) != 1 || reclaimed[0].ID != abandoned.ID || reclaimed[0].Attempt != 0 {
		t.Fatalf("reclaimed = %#v, want same job with attempt uncounted before ownership", reclaimed)
	}
	if err := second.Process(context.Background(), reclaimed[0], func(context.Context, Job) (TerminalPublication, error) { return PublishedCompleted, nil }); err != nil {
		t.Fatalf("Process() after reclaim error = %v", err)
	}
}

// TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004, and SW-REQ-080.
func TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	var terminalCalls atomic.Int32
	manager := newIntegrationQueue(t, client)
	telemetrySink := &observability.MemorySink{}
	manager.WithTelemetry(observability.NewOptimizationTelemetry(telemetrySink, telemetrySink, 1))
	manager.config.TerminalHandler = func(_ context.Context, _ Job, err error) (TerminalPublication, error) {
		terminalCalls.Add(1)
		if !errors.Is(err, errProcessingFixture) {
			return "", errors.New("terminal handler received wrong cause")
		}
		return PublishedFailed, nil
	}
	if _, err := manager.Enqueue(context.Background(), uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("initial Reserve() error = %v", err)
	}
	var processingAttempts []int
	for attempt := 1; attempt <= 3; attempt++ {
		if err := manager.Process(context.Background(), job, func(_ context.Context, owned Job) (TerminalPublication, error) {
			processingAttempts = append(processingAttempts, owned.Attempt)
			return "", errProcessingFixture
		}); err != nil {
			t.Fatalf("Process() attempt %d error = %v", attempt, err)
		}
		if attempt == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
		reclaimed, reclaimErr := manager.Reclaim(context.Background(), time.Millisecond)
		if reclaimErr != nil {
			t.Fatalf("Reclaim() attempt %d error = %v", attempt+1, reclaimErr)
		}
		if len(reclaimed) != 1 {
			t.Fatalf("reclaimed attempt %d = %#v, want one delivery", attempt+1, reclaimed)
		}
		job = reclaimed[0]
	}
	if got := processingAttempts; len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("processor attempts = %v, want [1 2 3]", got)
	}
	if terminalCalls.Load() != 1 {
		t.Fatalf("terminal handler calls = %d, want 1", terminalCalls.Load())
	}
	var retryOutcomes []string
	for _, point := range telemetrySink.Metrics {
		if point.Name == observability.MetricOptimizationRetryTotal {
			retryOutcomes = append(retryOutcomes, point.Labels["outcome"])
		}
	}
	if len(retryOutcomes) != 3 || retryOutcomes[0] != "retry" || retryOutcomes[1] != "retry" || retryOutcomes[2] != "exhausted" {
		t.Fatalf("retry telemetry = %v, want [retry retry exhausted]", retryOutcomes)
	}
	pending, err := client.XPending(context.Background(), manager.Config().Stream, manager.Config().Group).Result()
	if err != nil {
		t.Fatalf("XPending() error = %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending count = %d, want terminal XACK", pending.Count)
	}
}

var errProcessingFixture = errors.New("fixture processor failed")

// TestJobQueueStatsExposeDepthAndAge verifies IT-ARCH-004-007, ARCH-004,
// DESIGN-004/DESIGN-014, and SW-REQ-080/SW-REQ-082.
func TestJobQueueStatsExposeDepthAndAge(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	if _, err := manager.Enqueue(ctx, uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	queued, err := manager.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() queued error = %v", err)
	}
	if queued.QueueDepth < 1 || queued.StreamLength < 1 {
		t.Fatalf("queued stats = %#v, want queue and stream depth", queued)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	pending, err := manager.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() pending error = %v", err)
	}
	if pending.PendingDepth != 1 || pending.OldestPendingAge < 0 || pending.OldestQueuedAge < 0 {
		t.Fatalf("pending stats = %#v, want pending depth and non-negative ages", pending)
	}
	if err := manager.AckCompleted(ctx, job); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
}

// TestJobQueueCancellationReachesProcessorAndLeavesDeliveryRecoverable verifies
// IT-ARCH-004-003 and IT-ARCH-004-005, ARCH-004,
// DESIGN-004 JobQueueManager, and
// SW-REQ-021/SW-REQ-080 cancellation without terminal loss in real Redis.
func TestJobQueueCancellationReachesProcessorAndLeavesDeliveryRecoverable(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	if _, err := manager.Enqueue(context.Background(), uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- manager.Process(ctx, job, func(processCtx context.Context, _ Job) (TerminalPublication, error) {
			close(started)
			<-processCtx.Done()
			return "", processCtx.Err()
		})
	}()
	<-started
	cancel()
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("Process() cancellation error = %v, want context canceled", err)
	}
	pending, err := client.XPending(context.Background(), manager.Config().Stream, manager.Config().Group).Result()
	if err != nil {
		t.Fatalf("XPending() error = %v", err)
	}
	if pending.Count != 1 {
		t.Fatalf("pending count = %d, want recoverable delivery", pending.Count)
	}
	if err := manager.AckCompleted(context.Background(), job); err != nil {
		t.Fatalf("cleanup Ack() error = %v", err)
	}
}

// TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004 JobQueueManager, and
// SW-REQ-021/SW-REQ-080 malformed-contract cleanup against real Redis Streams.
func TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	validID := uuid.NewString()
	for _, jobID := range []string{"", uuid.Nil.String(), strings.ToUpper(validID), "{" + validID + "}", "not-a-uuid", validID + "\n"} {
		if _, err := manager.Enqueue(ctx, jobID); !errors.Is(err, ErrInvalidJob) {
			t.Errorf("Enqueue(%q) error = %v, want ErrInvalidJob", jobID, err)
		}
	}

	if err := manager.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	entryID, err := client.XAdd(ctx, &redis.XAddArgs{Stream: manager.Config().Stream, Values: map[string]any{jobIDField: uuid.Nil.String()}}).Result()
	if err != nil {
		t.Fatalf("XAdd() malformed error = %v", err)
	}
	if _, err := manager.Reserve(ctx); !errors.Is(err, ErrInvalidJob) {
		t.Fatalf("Reserve() malformed error = %v, want ErrInvalidJob", err)
	}
	entries, err := client.XRange(ctx, manager.Config().Stream, entryID, entryID).Result()
	if err != nil {
		t.Fatalf("XRange() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("malformed stream entry remains: %#v", entries)
	}
	pending, err := client.XPending(ctx, manager.Config().Stream, manager.Config().Group).Result()
	if err != nil {
		t.Fatalf("XPending() error = %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending malformed deliveries = %d, want 0", pending.Count)
	}
}

// TestTask224ReclaimPreparationReturnsValidPrefixOnLaterFailure verifies that
// already claimed work remains visible when a later real-Redis entry is bad.
func TestTask224ReclaimPreparationReturnsValidPrefixOnLaterFailure(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	if err := manager.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	validID := uuid.NewString()
	if err := client.XAdd(ctx, &redis.XAddArgs{Stream: manager.Config().Stream, Values: map[string]any{jobIDField: validID}}).Err(); err != nil {
		t.Fatalf("XAdd() valid error = %v", err)
	}
	malformedEntry, err := client.XAdd(ctx, &redis.XAddArgs{Stream: manager.Config().Stream, Values: map[string]any{jobIDField: "malformed"}}).Result()
	if err != nil {
		t.Fatalf("XAdd() malformed error = %v", err)
	}
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: manager.Config().Group, Consumer: "abandoned", Streams: []string{manager.Config().Stream, ">"}, Count: 2,
	}).Result()
	if err != nil || len(streams) != 1 || len(streams[0].Messages) != 2 {
		t.Fatalf("XReadGroup() streams = %#v, error = %v", streams, err)
	}
	jobs, prepareErr := manager.prepareDeliveries(ctx, streams[0].Messages)
	if !errors.Is(prepareErr, ErrInvalidJob) {
		t.Fatalf("prepareDeliveries() error = %v, want ErrInvalidJob", prepareErr)
	}
	if len(jobs) != 1 || jobs[0].ID != validID || jobs[0].Attempt != 0 {
		t.Fatalf("prepared prefix = %#v, want one uncounted valid job", jobs)
	}
	if exists, err := client.XRange(ctx, manager.Config().Stream, malformedEntry, malformedEntry).Result(); err != nil || len(exists) != 0 {
		t.Fatalf("malformed entry after partial failure = %#v, error = %v", exists, err)
	}
	if err := manager.AckCompleted(ctx, jobs[0]); err != nil {
		t.Fatalf("Ack() valid prefix error = %v", err)
	}
}

// TestTask224LockMissAndCompletedDeliveryDoNotConsumeAttempts verifies logical
// ownership precedes retry accounting for duplicate and completed deliveries.
func TestTask224LockMissAndCompletedDeliveryDoNotConsumeAttempts(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	jobID := uuid.NewString()
	if _, err := manager.Enqueue(ctx, jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if err := client.XAdd(ctx, &redis.XAddArgs{Stream: manager.Config().Stream, Values: map[string]any{jobIDField: jobID}}).Err(); err != nil {
		t.Fatalf("XAdd() duplicate error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := client.Set(ctx, manager.lockKey(jobID), "another-owner", time.Minute).Err(); err != nil {
		t.Fatalf("Set() lock error = %v", err)
	}
	called := false
	if err := manager.Process(ctx, job, func(context.Context, Job) (TerminalPublication, error) { called = true; return PublishedCompleted, nil }); err != nil {
		t.Fatalf("Process() lock miss error = %v", err)
	}
	if called {
		t.Fatal("processor called without logical ownership")
	}
	if exists := client.Exists(ctx, manager.attemptKey(jobID)).Val(); exists != 0 {
		t.Fatalf("attempt key exists after lock miss: %d", exists)
	}
	duplicate, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() duplicate error = %v", err)
	}
	if duplicate.ID != jobID {
		t.Fatalf("duplicate job ID = %q, want %q", duplicate.ID, jobID)
	}
	if err := manager.Process(ctx, duplicate, func(context.Context, Job) (TerminalPublication, error) { called = true; return PublishedCompleted, nil }); err != nil {
		t.Fatalf("Process() duplicate lock miss error = %v", err)
	}
	if exists := client.Exists(ctx, manager.attemptKey(jobID)).Val(); exists != 0 {
		t.Fatalf("attempt key exists after duplicate lock miss: %d", exists)
	}

	completedID := uuid.NewString()
	entryID, err := manager.Enqueue(ctx, completedID)
	if err != nil {
		t.Fatalf("Enqueue() completed fixture error = %v", err)
	}
	completedJob, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() completed fixture error = %v", err)
	}
	if completedJob.EntryID != entryID {
		t.Fatalf("completed fixture entry = %q, want %q", completedJob.EntryID, entryID)
	}
	if err := client.Set(ctx, manager.doneKey(completedID), completedValue, time.Minute).Err(); err != nil {
		t.Fatalf("Set() completion marker error = %v", err)
	}
	if err := manager.Process(ctx, completedJob, func(context.Context, Job) (TerminalPublication, error) { called = true; return PublishedCompleted, nil }); err != nil {
		t.Fatalf("Process() completed delivery error = %v", err)
	}
	if exists := client.Exists(ctx, manager.attemptKey(completedID)).Val(); exists != 0 {
		t.Fatalf("attempt key exists after completed delivery: %d", exists)
	}
}

// TestTask224AtomicAttemptAndMillisecondTTLs verifies sub-second Redis TTLs do
// not truncate and a failed atomic update cannot leave a partial counter.
func TestTask224AtomicAttemptAndMillisecondTTLs(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := NewJobQueueManager(client, Config{
		Stream: "mealswapp:test:optimization:{" + uuid.NewString() + "}", Group: "workers", Consumer: "task-224",
		VisibilityTimeout: DefaultVisibilityTimeout, ReadBlock: 10 * time.Millisecond, BatchSize: 1,
		MaxAttempts: 3, CompletedTTL: 500 * time.Millisecond, AttemptTTL: 500 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Del(context.Background(), manager.Config().Stream).Err() })
	ctx := context.Background()
	jobID := uuid.NewString()
	if _, err := manager.Enqueue(ctx, jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := manager.Process(ctx, job, func(_ context.Context, owned Job) (TerminalPublication, error) {
		if owned.Attempt != 1 {
			t.Fatalf("owned attempt = %d, want 1", owned.Attempt)
		}
		return PublishedCompleted, nil
	}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	for name, key := range map[string]string{"attempt": manager.attemptKey(jobID), "completed": manager.doneKey(jobID), "enqueue": manager.enqueueKey(jobID)} {
		if ttl := client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > 500*time.Millisecond {
			t.Errorf("%s TTL = %s, want (0, 500ms]", name, ttl)
		}
	}

	failedID := uuid.NewString()
	failedKey := manager.attemptKey(failedID)
	if _, err := countAttemptScript.Run(ctx, client, []string{failedKey}, "invalid-ttl").Result(); err == nil {
		t.Fatal("countAttemptScript invalid TTL error = nil")
	}
	if exists := client.Exists(ctx, failedKey).Val(); exists != 0 {
		t.Fatalf("attempt key exists after failed atomic update: %d", exists)
	}
}

func TestTask224ConfigurationBoundaries(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	base := Config{Stream: "task-224:{queue}", Group: "workers", Consumer: "worker", VisibilityTimeout: DefaultVisibilityTimeout, BatchSize: 1, CompletedTTL: time.Second, AttemptTTL: time.Second}
	tests := []struct {
		name   string
		mutate func(*Config)
		valid  bool
	}{
		{name: "default visibility", valid: true},
		{name: "millisecond-safe visibility", mutate: func(c *Config) { c.VisibilityTimeout = 36*time.Second + time.Millisecond }, valid: true},
		{name: "nanosecond-only visibility margin", mutate: func(c *Config) { c.VisibilityTimeout = 36*time.Second + time.Nanosecond }},
		{name: "sub-millisecond visibility margin", mutate: func(c *Config) { c.VisibilityTimeout = 36*time.Second + time.Millisecond - time.Nanosecond }},
		{name: "overlapping visibility boundary", mutate: func(c *Config) { c.VisibilityTimeout = 36 * time.Second }},
		{name: "multiple reservation", mutate: func(c *Config) { c.BatchSize = 2 }},
		{name: "millisecond completed TTL", mutate: func(c *Config) { c.CompletedTTL = time.Millisecond }, valid: true},
		{name: "sub-millisecond completed TTL", mutate: func(c *Config) { c.CompletedTTL = time.Millisecond - time.Nanosecond }},
		{name: "millisecond attempt TTL", mutate: func(c *Config) { c.AttemptTTL = time.Millisecond }, valid: true},
		{name: "sub-millisecond attempt TTL", mutate: func(c *Config) { c.AttemptTTL = time.Millisecond - time.Nanosecond }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := base
			config.Stream += ":" + uuid.NewString()
			if tt.mutate != nil {
				tt.mutate(&config)
			}
			err := NewJobQueueManager(client, config).validate()
			if tt.valid && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("validate() error = nil")
			}
		})
	}
}

// TestTask224RedisEffectiveLockTTLExceedsProcessingBoundary verifies the live
// Redis lock retains a measurable millisecond margin beyond solve/finalization.
func TestTask224RedisEffectiveLockTTLExceedsProcessingBoundary(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	manager.config.VisibilityTimeout = 36*time.Second + 100*time.Millisecond
	ctx := context.Background()
	jobID := uuid.NewString()
	if _, err := manager.Enqueue(ctx, jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := manager.Process(ctx, job, func(context.Context, Job) (TerminalPublication, error) {
		ttl, err := client.PTTL(ctx, manager.lockKey(jobID)).Result()
		if err != nil {
			t.Fatalf("PTTL() lock error = %v", err)
		}
		if boundary := optimizationWorkTimeout + optimizationFinalizeBudget; ttl <= boundary {
			t.Fatalf("live lock PTTL = %s, want greater than %s", ttl, boundary)
		}
		return PublishedCompleted, nil
	}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
}

// TestJobQueueUnavailableDoesNotInvokeProcessor verifies IT-ARCH-004-004,
// ARCH-004, DESIGN-004, and SW-REQ-080.
func TestJobQueueUnavailableDoesNotInvokeProcessor(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 25 * time.Millisecond, ReadTimeout: 25 * time.Millisecond, WriteTimeout: 25 * time.Millisecond})
	defer client.Close()
	manager := NewJobQueueManager(client, Config{Stream: "unavailable:{queue}", Group: "workers", Consumer: "consumer", VisibilityTimeout: DefaultVisibilityTimeout})
	var called atomic.Int32
	if _, err := manager.Enqueue(context.Background(), uuid.NewString()); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("Enqueue() error = %v, want ErrQueueUnavailable", err)
	}
	if _, err := manager.ProcessNext(context.Background(), func(context.Context, Job) (TerminalPublication, error) {
		called.Add(1)
		return PublishedCompleted, nil
	}); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("ProcessNext() error = %v, want ErrQueueUnavailable", err)
	}
	if called.Load() != 0 {
		t.Fatal("processor was invoked while Redis was unavailable")
	}
}

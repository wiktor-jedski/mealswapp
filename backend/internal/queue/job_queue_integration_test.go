package queue

// Implements DESIGN-004 JobQueueManager real-Redis integration verification.

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	stream := "mealswapp:test:optimization:" + uuid.NewString()
	manager := NewJobQueueManager(client, Config{
		Stream:            stream,
		Group:             "optimization-workers",
		Consumer:          "test-" + uuid.NewString(),
		VisibilityTimeout: 31 * time.Second,
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
	if job.ID != jobID || job.EntryID != entryID || job.Attempt != 1 || job.EnqueuedAt.IsZero() {
		t.Fatalf("reserved job = %#v, want ID=%q entry=%q attempt=1 with timestamp", job, jobID, entryID)
	}
	if err := manager.Ack(ctx, job); err != nil {
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
	processor := func(context.Context, Job) error {
		processed.Add(1)
		time.Sleep(50 * time.Millisecond)
		return nil
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
	if len(reclaimed) != 1 || reclaimed[0].ID != abandoned.ID || reclaimed[0].Attempt != 2 {
		t.Fatalf("reclaimed = %#v, want same job with attempt 2", reclaimed)
	}
	if err := second.Process(context.Background(), reclaimed[0], func(context.Context, Job) error { return nil }); err != nil {
		t.Fatalf("Process() after reclaim error = %v", err)
	}
}

// TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004, and SW-REQ-080.
func TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	var terminalCalls atomic.Int32
	manager := newIntegrationQueue(t, client)
	manager.config.TerminalHandler = func(_ context.Context, _ Job, err error) error {
		terminalCalls.Add(1)
		if !errors.Is(err, errProcessingFixture) {
			return errors.New("terminal handler received wrong cause")
		}
		return nil
	}
	if _, err := manager.Enqueue(context.Background(), uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("initial Reserve() error = %v", err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if err := manager.Process(context.Background(), job, func(context.Context, Job) error { return errProcessingFixture }); err != nil {
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
	if job.Attempt != 3 {
		t.Fatalf("final attempt = %d, want 3", job.Attempt)
	}
	if terminalCalls.Load() != 1 {
		t.Fatalf("terminal handler calls = %d, want 1", terminalCalls.Load())
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
	if err := manager.Ack(ctx, job); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
}

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
		finished <- manager.Process(ctx, job, func(processCtx context.Context, _ Job) error {
			close(started)
			<-processCtx.Done()
			return processCtx.Err()
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
	if err := manager.Ack(context.Background(), job); err != nil {
		t.Fatalf("cleanup Ack() error = %v", err)
	}
}

// TestJobQueueUnavailableDoesNotInvokeProcessor verifies IT-ARCH-004-004,
// ARCH-004, DESIGN-004, and SW-REQ-080.
func TestJobQueueUnavailableDoesNotInvokeProcessor(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 25 * time.Millisecond, ReadTimeout: 25 * time.Millisecond, WriteTimeout: 25 * time.Millisecond})
	defer client.Close()
	manager := NewJobQueueManager(client, Config{Stream: "unavailable", Group: "workers", Consumer: "consumer", VisibilityTimeout: 31 * time.Second})
	var called atomic.Int32
	if _, err := manager.Enqueue(context.Background(), uuid.NewString()); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("Enqueue() error = %v, want ErrQueueUnavailable", err)
	}
	if _, err := manager.ProcessNext(context.Background(), func(context.Context, Job) error {
		called.Add(1)
		return nil
	}); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("ProcessNext() error = %v, want ErrQueueUnavailable", err)
	}
	if called.Load() != 0 {
		t.Fatal("processor was invoked while Redis was unavailable")
	}
}

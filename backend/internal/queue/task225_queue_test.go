package queue

// Implements DESIGN-004 JobQueueManager Task 225 atomic finalization and recovery verification.

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// TestTask225RequiresExplicitTerminalPublication verifies IT-ARCH-004-003,
// ARCH-004, DESIGN-004 JobQueueManager, and SW-REQ-021/SW-REQ-080 failure
// publication before acknowledgement across the real Redis boundary.
func TestTask225RequiresExplicitTerminalPublication(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	manager.config.MaxAttempts = 1
	ctx := context.Background()
	if _, err := manager.Enqueue(ctx, uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	err = manager.Process(ctx, job, func(context.Context, Job) (TerminalPublication, error) {
		return "", errors.New("retry budget exhausted")
	})
	if !errors.Is(err, ErrTerminalPublicationRequired) {
		t.Fatalf("Process() error = %v, want ErrTerminalPublicationRequired", err)
	}
	if pending := client.XPending(ctx, manager.config.Stream, manager.config.Group).Val().Count; pending != 1 {
		t.Fatalf("pending count = %d, want delivery retained", pending)
	}
	if exists := client.Exists(ctx, manager.doneKey(job.ID)).Val(); exists != 0 {
		t.Fatalf("terminal marker exists without publication: %d", exists)
	}

	manager.config.TerminalHandler = func(context.Context, Job, error) (TerminalPublication, error) {
		return PublishedFailed, nil
	}
	if err := manager.Process(ctx, job, func(context.Context, Job) (TerminalPublication, error) {
		return "", errors.New("retry budget exhausted")
	}); err != nil {
		t.Fatalf("Process() with failed publication error = %v", err)
	}
	if got := client.Get(ctx, manager.doneKey(job.ID)).Val(); got != failedValue {
		t.Fatalf("terminal marker = %q, want %q", got, failedValue)
	}
}

// TestTask225DistinctFinalizationAndZeroAckSemantics verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004 JobQueueManager, and
// SW-REQ-021/SW-REQ-080 atomic terminal state and acknowledgement data flow.
func TestTask225DistinctFinalizationAndZeroAckSemantics(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	jobID := uuid.NewString()
	if _, err := manager.Enqueue(ctx, jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := manager.AckFailed(ctx, job); err != nil {
		t.Fatalf("AckFailed() error = %v", err)
	}
	if err := manager.AckFailed(ctx, job); err != nil {
		t.Fatalf("idempotent AckFailed() error = %v", err)
	}
	if err := manager.AckCompleted(ctx, job); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("conflicting AckCompleted() error = %v, want ErrQueueUnavailable", err)
	}
	if got := client.Get(ctx, manager.doneKey(jobID)).Val(); got != failedValue {
		t.Fatalf("terminal marker = %q after conflict, want failed", got)
	}

	forgedID := uuid.NewString()
	entryID, err := manager.Enqueue(ctx, forgedID)
	if err != nil {
		t.Fatalf("Enqueue() forged fixture error = %v", err)
	}
	forged := Job{ID: forgedID, EntryID: entryID}
	if err := manager.AckCompleted(ctx, forged); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("AckCompleted(non-pending) error = %v, want ErrQueueUnavailable", err)
	}
	if exists := client.Exists(ctx, manager.doneKey(forgedID)).Val(); exists != 0 {
		t.Fatalf("non-pending ACK created terminal marker: %d", exists)
	}
	if entries := client.XRange(ctx, manager.config.Stream, entryID, entryID).Val(); len(entries) != 1 {
		t.Fatalf("non-pending entry count = %d, want retained", len(entries))
	}
}

func TestTask225EmbeddedScriptsUseCacheFallbackAndClusterSlot(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	for name, source := range map[string]string{
		"enqueue": enqueueLua, "finalize": finalizeLua, "attempt": countAttemptLua,
		"remove": removeDeliveryLua, "release": releaseLockLua,
	} {
		if strings.TrimSpace(source) == "" || !strings.Contains(source, "DESIGN-004") {
			t.Errorf("embedded %s script is empty or untraced", name)
		}
	}
	jobID := uuid.NewString()
	keys := []string{manager.config.Stream, manager.enqueueKey(jobID), manager.doneKey(jobID), manager.attemptKey(jobID), manager.lockKey(jobID)}
	for _, key := range keys {
		if got := streamHashTag(key); got != streamHashTag(manager.config.Stream) {
			t.Fatalf("key %q hash tag = %q, want %q", key, got, streamHashTag(manager.config.Stream))
		}
	}
	if err := enqueueScript.Load(ctx, client).Err(); err != nil {
		t.Fatalf("Load(enqueue) error = %v", err)
	}
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatalf("ScriptFlush() error = %v", err)
	}
	if _, err := manager.Enqueue(ctx, jobID); err != nil {
		t.Fatalf("Enqueue() after NOSCRIPT cache flush error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := manager.Process(ctx, job, func(context.Context, Job) (TerminalPublication, error) {
		return PublishedCompleted, nil
	}); err != nil {
		t.Fatalf("Process() embedded scripts error = %v", err)
	}
}

// TestTask225AtomicDuplicateCleanupUnderRace verifies IT-ARCH-004-003,
// ARCH-004, DESIGN-004 JobQueueManager, and SW-REQ-080/SW-REQ-082 across
// concurrent real-Redis pending/stream cleanup.
func TestTask225AtomicDuplicateCleanupUnderRace(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	if _, err := manager.Enqueue(ctx, uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- manager.removeDelivery(ctx, job.EntryID)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("removeDelivery() race error = %v", err)
		}
	}
	if pending := client.XPending(ctx, manager.config.Stream, manager.config.Group).Val().Count; pending != 0 {
		t.Fatalf("pending count = %d, want 0", pending)
	}
	if entries := client.XRange(ctx, manager.config.Stream, job.EntryID, job.EntryID).Val(); len(entries) != 0 {
		t.Fatalf("stream entry survived duplicate cleanup: %#v", entries)
	}
}

// TestTask225LiveManagerRecoversGroupAndDataLoss verifies IT-ARCH-004-003,
// ARCH-004, DESIGN-004 JobQueueManager, and SW-REQ-021/SW-REQ-080/SW-REQ-082
// queue group loss, data loss, and concurrent bootstrap recovery in real Redis.
func TestTask225LiveManagerRecoversGroupAndDataLoss(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	jobID := uuid.NewString()
	if _, err := manager.Enqueue(ctx, jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if destroyed := client.XGroupDestroy(ctx, manager.config.Stream, manager.config.Group).Val(); destroyed != 1 {
		t.Fatal("XGroupDestroy() did not remove group")
	}
	job, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() after group deletion error = %v", err)
	}
	if job.ID != jobID {
		t.Fatalf("recovered job = %q, want %q", job.ID, jobID)
	}
	if err := manager.AckCompleted(ctx, job); err != nil {
		t.Fatalf("AckCompleted() recovery cleanup error = %v", err)
	}

	if err := client.Del(ctx, manager.config.Stream, manager.enqueueKey(jobID), manager.doneKey(jobID), manager.attemptKey(jobID), manager.lockKey(jobID)).Err(); err != nil {
		t.Fatalf("delete restart-loss fixture state error = %v", err)
	}
	postRestartID := uuid.NewString()
	if _, err := manager.Enqueue(ctx, postRestartID); err != nil {
		t.Fatalf("Enqueue() after restart-loss error = %v", err)
	}
	postRestart, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() after restart-loss error = %v", err)
	}
	if postRestart.ID != postRestartID {
		t.Fatalf("post-restart job = %q, want %q", postRestart.ID, postRestartID)
	}

	if destroyed := client.XGroupDestroy(ctx, manager.config.Stream, manager.config.Group).Val(); destroyed != 1 {
		t.Fatal("second XGroupDestroy() did not remove group")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- manager.recoverGroup(ctx)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent recoverGroup() error = %v", err)
		}
	}
	groups, err := client.XInfoGroups(ctx, manager.config.Stream).Result()
	if err != nil || len(groups) != 1 || groups[0].Name != manager.config.Group {
		t.Fatalf("groups after concurrent recovery = %#v, error = %v", groups, err)
	}
}

// TestTask225LiveManagerRecoversAfterRedisRestart verifies IT-ARCH-004-003,
// ARCH-004, DESIGN-004 JobQueueManager, and SW-REQ-021/SW-REQ-080 recovery
// across a real Redis process loss and restart.
func TestTask225LiveManagerRecoversAfterRedisRestart(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve Redis test port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release Redis test port: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:" + strconv.Itoa(port), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	redisArgs := []string{
		"--bind", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--protected-mode", "no",
		"--save", "",
		"--appendonly", "no",
	}
	var serverCommand func() *exec.Cmd
	if redisServer, lookErr := exec.LookPath("redis-server"); lookErr == nil {
		serverCommand = func() *exec.Cmd {
			return exec.Command(redisServer, append(redisArgs, "--dir", t.TempDir())...)
		}
	} else if docker, dockerErr := exec.LookPath("docker"); dockerErr == nil && exec.Command(docker, "image", "inspect", "redis:7-alpine").Run() == nil {
		serverCommand = func() *exec.Cmd {
			return exec.Command(docker, append([]string{"run", "--rm", "--network", "host", "redis:7-alpine", "redis-server"}, redisArgs...)...)
		}
	} else {
		t.Skip("neither redis-server nor an available redis:7-alpine Docker image can run the restart fixture")
	}
	start := func() (*exec.Cmd, <-chan error) {
		cmd := serverCommand()
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if startErr := cmd.Start(); startErr != nil {
			t.Fatalf("start isolated redis-server: %v", startErr)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			pingCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			pingErr := client.Ping(pingCtx).Err()
			cancel()
			if pingErr == nil {
				return cmd, done
			}
			time.Sleep(10 * time.Millisecond)
		}
		_ = cmd.Process.Signal(os.Interrupt)
		<-done
		t.Fatal("isolated redis-server did not become ready")
		return nil, nil
	}
	stop := func(cmd *exec.Cmd, done <-chan error) {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		shutdownErr := client.Shutdown(shutdownCtx).Err()
		cancel()
		if shutdownErr != nil && !errors.Is(shutdownErr, redis.Nil) {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
			t.Fatal("isolated redis-server did not stop")
		}
	}

	firstServer, firstDone := start()
	manager := NewJobQueueManager(client, Config{
		Stream: "task225:{restart}", Group: "workers", Consumer: "worker",
		ReadBlock: 10 * time.Millisecond,
	})
	if _, err := manager.Enqueue(context.Background(), uuid.NewString()); err != nil {
		t.Fatalf("Enqueue() before restart error = %v", err)
	}
	stop(firstServer, firstDone)

	secondServer, secondDone := start()
	t.Cleanup(func() { stop(secondServer, secondDone) })
	jobID := uuid.NewString()
	if _, err := manager.Enqueue(context.Background(), jobID); err != nil {
		t.Fatalf("Enqueue() after Redis restart error = %v", err)
	}
	job, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("Reserve() after Redis restart error = %v", err)
	}
	if job.ID != jobID {
		t.Fatalf("job after Redis restart = %q, want %q", job.ID, jobID)
	}
	if err := manager.AckCompleted(context.Background(), job); err != nil {
		t.Fatalf("AckCompleted() after Redis restart error = %v", err)
	}
}

// TestTask225AuthorizationAndConnectivityErrorsFailClosed verifies
// IT-ARCH-004-003, ARCH-004, DESIGN-004 JobQueueManager, and
// SW-REQ-080 degraded Redis collaboration without permissive recovery.
func TestTask225AuthorizationAndConnectivityErrorsFailClosed(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	options := *client.Options()
	options.Username = "task225-denied"
	options.Password = uuid.NewString()
	denied := redis.NewClient(&options)
	t.Cleanup(func() { _ = denied.Close() })
	manager := NewJobQueueManager(denied, Config{Stream: "task225:{denied}", Group: "workers", Consumer: "worker"})
	if err := manager.Bootstrap(context.Background()); !errors.Is(err, ErrQueueUnavailable) {
		t.Fatalf("Bootstrap() authorization error = %v, want ErrQueueUnavailable", err)
	}
	if manager.bootstrapped {
		t.Fatal("authorization failure marked manager bootstrapped")
	}
}

func TestTask225LockCleanupIsBoundedAndObservable(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: time.Second})
	t.Cleanup(func() { _ = client.Close() })
	sink := &observability.MemorySink{}
	manager := NewJobQueueManager(client, Config{Stream: "task225:{cleanup}", Group: "workers", Consumer: "worker"}).
		WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
	started := time.Now()
	manager.releaseLock(context.Background(), uuid.NewString(), "owner")
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("releaseLock() elapsed = %s, want bounded cleanup", elapsed)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		metrics, _ := sink.Snapshot()
		for _, point := range metrics {
			if point.Name == observability.MetricOptimizationQueueCleanup && point.Labels["outcome"] == "failed" {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("lock cleanup failure telemetry not observed")
}

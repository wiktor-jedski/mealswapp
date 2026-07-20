package worker

// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector Task 234 timeout/release gate.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestTask234QueuedDurableJobRemainsPendingWhenProcessingAndFinalizationFail(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	ctx := context.Background()
	jobID := uuid.New()
	store := NewRedisOptimizationJobStore(client)
	if err := store.Save(ctx, OptimizationJob{JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	manager := queue.NewJobQueueManager(client, queue.Config{
		Stream: "mealswapp:test:task234:{" + uuid.NewString() + "}", Group: "workers", Consumer: "worker", ReadBlock: 10 * time.Millisecond,
	})
	if _, err := manager.Enqueue(ctx, jobID.String()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	delivery, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	processor := NewOptimizationProcessor(&task234MarkProcessingTimeoutStore{RedisOptimizationJobStore: store}, &deadlineInputLoader{}, &deadlineSolver{})
	processor.jobDeadline = 10 * time.Millisecond
	processor.finalizationTimeout = time.Second
	if err := manager.Process(ctx, delivery, processor.ProcessOptimizationJob); err != nil {
		t.Fatalf("Process() retryable failure error = %v", err)
	}

	durable, err := store.Load(ctx, jobID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if durable.Status != OptimizationJobQueued {
		t.Fatalf("durable status = %q, want queued", durable.Status)
	}
	if pending := client.XPending(ctx, manager.Config().Stream, manager.Config().Group).Val().Count; pending != 1 {
		t.Fatalf("pending deliveries = %d, want 1", pending)
	}
	if entries := client.XLen(ctx, manager.Config().Stream).Val(); entries != 1 {
		t.Fatalf("stream entries = %d, want 1", entries)
	}
}

func TestTask234TerminalPublicationRequiresProcessingAndMatchesDurableState(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	ctx := context.Background()
	failure := OptimizationJobFailure{Code: optimization.FailureCodeSolverTimeout, Message: safeFailureMessage(optimization.FailureCodeSolverTimeout)}

	for _, publish := range []struct {
		name string
		run  func(*RedisOptimizationJobStore, uuid.UUID) error
	}{
		{name: "completed", run: func(store *RedisOptimizationJobStore, jobID uuid.UUID) error {
			return store.PublishCompleted(ctx, jobID, []optimization.DietAlternative{task221Alternative(0.5)}, time.Now())
		}},
		{name: "failed", run: func(store *RedisOptimizationJobStore, jobID uuid.UUID) error {
			return store.PublishFailed(ctx, jobID, nil, failure, time.Now())
		}},
	} {
		t.Run("queued_"+publish.name+"_is_rejected", func(t *testing.T) {
			jobID := uuid.New()
			store := NewRedisOptimizationJobStore(client)
			if err := store.Save(ctx, OptimizationJob{JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued}); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			if err := publish.run(store, jobID); err == nil {
				t.Fatalf("Publish%s() error = nil, want rejected queued transition", publish.name)
			}
			job, err := store.Load(ctx, jobID)
			if err != nil || job.Status != OptimizationJobQueued {
				t.Fatalf("Load() = status %q, %v, want queued", job.Status, err)
			}
		})
	}

	completedID := task234ProcessingJob(t, ctx, client)
	completedStore := NewRedisOptimizationJobStore(client)
	if err := completedStore.PublishCompleted(ctx, completedID, []optimization.DietAlternative{task221Alternative(0.5)}, time.Now()); err != nil {
		t.Fatalf("PublishCompleted(processing) error = %v", err)
	}
	if job, err := completedStore.Load(ctx, completedID); err != nil || job.Status != OptimizationJobCompleted {
		t.Fatalf("completed durable state = %q, %v", job.Status, err)
	}
	if err := completedStore.PublishFailed(ctx, completedID, nil, failure, time.Now()); err == nil {
		t.Fatal("PublishFailed(completed) error = nil, want conflicting terminal publication rejection")
	}

	failedID := task234ProcessingJob(t, ctx, client)
	failedStore := NewRedisOptimizationJobStore(client)
	if err := failedStore.PublishFailed(ctx, failedID, nil, failure, time.Now()); err != nil {
		t.Fatalf("PublishFailed(processing) error = %v", err)
	}
	if job, err := failedStore.Load(ctx, failedID); err != nil || job.Status != OptimizationJobFailed {
		t.Fatalf("failed durable state = %q, %v", job.Status, err)
	}
	if err := failedStore.PublishCompleted(ctx, failedID, []optimization.DietAlternative{task221Alternative(0.5)}, time.Now()); err == nil {
		t.Fatal("PublishCompleted(failed) error = nil, want conflicting terminal publication rejection")
	}
}

// TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe verifies
// IT-ARCH-004-007, ARCH-004, DESIGN-004 JobQueueManager,
// DESIGN-014 MetricsCollector/LogAggregator, and SW-REQ-080/SW-REQ-082.
func TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	client.AddHook(task234ReleaseFailureHook{err: errors.New("private-job-id private-user@example.com")})
	jobID := uuid.NewString()
	manager := queue.NewJobQueueManager(client, queue.Config{})
	if _, err := manager.Enqueue(context.Background(), jobID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	clpPath := filepath.Join(t.TempDir(), "clp")
	if err := os.WriteFile(clpPath, []byte("#!/bin/sh\nprintf 'Clp version 1.17.11\\n'\n"), 0o700); err != nil {
		t.Fatalf("write CLP fixture: %v", err)
	}
	sink := &observability.MemorySink{}
	telemetry := observability.NewOptimizationTelemetry(sink, sink, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunWithProcessorAndTelemetry(ctx, config.Config{Environment: "test", CLPExecutable: clpPath, CLPVersion: "1.17.11"}, client, func(context.Context, queue.Job) (queue.TerminalPublication, error) {
			return queue.PublishedCompleted, nil
		}, telemetry)
	}()
	t.Cleanup(func() {
		cancel()
		if done != nil {
			<-done
		}
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		metrics, logs := sink.Snapshot()
		if len(metrics) == 1 && len(logs) == 1 {
			point, event := metrics[0], logs[0]
			if point.Name != observability.MetricOptimizationQueueCleanup || point.Value != 1 || point.Unit != "cleanups" || len(point.Labels) != 1 || point.Labels["outcome"] != "failed" {
				t.Fatalf("queue cleanup metric = %+v", point)
			}
			if event.Message != "optimization_queue_cleanup" || len(event.Fields) != 1 || event.Fields["outcome"] != "failed" {
				t.Fatalf("queue cleanup event = %+v", event)
			}
			payload, err := json.Marshal(struct{ Metrics, Logs any }{metrics, logs})
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{jobID, "private-job-id", "private-user@example.com", "job_id", "entry_id"} {
				if bytes.Contains(payload, []byte(forbidden)) {
					t.Fatalf("production queue telemetry leaked %q: %s", forbidden, payload)
				}
			}
			cancel()
			if err := <-done; err != nil {
				t.Fatalf("RunWithProcessorAndTelemetry() error = %v", err)
			}
			done = nil
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("production worker did not emit queue cleanup telemetry")
}

// TestTask234SolverTimeoutAndAdmissionReleaseFailureKeepSafeFinalTelemetry
// verifies IT-ARCH-004-005 and IT-ARCH-004-007, ARCH-004,
// DESIGN-004 JobStatusTracker,
// DESIGN-014 MetricsCollector, and SW-REQ-021/SW-REQ-080 degraded finalization.
func TestTask234SolverTimeoutAndAdmissionReleaseFailureKeepSafeFinalTelemetry(t *testing.T) {
	jobID, userID, dietID, mealID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store := &deadlineJobStore{job: OptimizationJob{JobID: jobID, UserID: userID, DailyDietID: dietID, Status: OptimizationJobQueued, CreatedAt: time.Now()}}
	inputs := &deadlineInputLoader{result: optimization.SavedDietOptimizationInputs{
		Request: optimization.DietOptimizationRequest{OriginalDiet: repository.SavedDiet{ID: dietID, UserID: userID, Entries: []repository.SavedDietMealEntry{{MealID: mealID, Quantity: 100, Unit: "g"}}}},
		Meals:   []repository.MealEntity{{ID: mealID, Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 10}, NormalizedMacrosAvailable: true}},
	}}
	sink := &observability.MemorySink{}
	releaseErr := errors.New("private admission key and solver diagnostic")
	processor := NewOptimizationProcessor(store, inputs, &task234DeadlineSolver{mealID: mealID}).
		WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1)).
		WithAdmissionGate(task234FailingAdmission{err: releaseErr})
	processor.jobDeadline = 20 * time.Millisecond
	processor.finalizationTimeout = time.Second

	publication, err := processor.ProcessOptimizationJob(context.Background(), queue.Job{ID: jobID.String(), EntryID: "private-stream-entry", Attempt: 2})
	if !errors.Is(err, releaseErr) || publication != "" {
		t.Fatalf("processor result = %q, %v, want published-state release failure pending for retry", publication, err)
	}
	if store.failure.Code != optimization.FailureCodeSolverTimeout {
		t.Fatalf("final failure = %+v, want solver_timeout", store.failure)
	}

	metrics, logs := sink.Snapshot()
	statuses := map[string]int{}
	workerValues := []float64{}
	for _, point := range metrics {
		if len(point.Labels) > 1 {
			t.Fatalf("unbounded worker metric: %+v", point)
		}
		switch point.Name {
		case observability.MetricOptimizationSolveTotal, observability.MetricOptimizationJobTotal:
			statuses[point.Labels["status"]]++
		case observability.MetricOptimizationWorkerActive:
			workerValues = append(workerValues, point.Value)
		}
	}
	if statuses["completed"] != 2 || statuses["timeout"] != 2 || len(statuses) != 2 {
		t.Fatalf("solve/job statuses = %#v, want two completed solves plus timeout solve and job", statuses)
	}
	if len(workerValues) != 2 || workerValues[0] != 1 || workerValues[1] != 0 {
		t.Fatalf("worker active values = %v, want [1 0]", workerValues)
	}
	payload, marshalErr := json.Marshal(struct{ Metrics, Logs any }{metrics, logs})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, forbidden := range []string{jobID.String(), userID.String(), dietID.String(), mealID.String(), releaseErr.Error(), "private-stream-entry"} {
		if bytes.Contains(payload, []byte(forbidden)) {
			t.Fatalf("worker telemetry leaked %q: %s", forbidden, payload)
		}
	}
}

type task234MarkProcessingTimeoutStore struct {
	*RedisOptimizationJobStore
}

func (s *task234MarkProcessingTimeoutStore) MarkProcessing(ctx context.Context, _ uuid.UUID, _ time.Time) (OptimizationJob, error) {
	<-ctx.Done()
	return OptimizationJob{}, ctx.Err()
}

func task234ProcessingJob(t *testing.T, ctx context.Context, client redis.UniversalClient) uuid.UUID {
	t.Helper()
	jobID := uuid.New()
	store := NewRedisOptimizationJobStore(client)
	if err := store.Save(ctx, OptimizationJob{JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := store.MarkProcessing(ctx, jobID, time.Now()); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}
	return jobID
}

type task234ReleaseFailureHook struct{ err error }

func (task234ReleaseFailureHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (task234ReleaseFailureHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h task234ReleaseFailureHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		name := strings.ToLower(cmd.Name())
		isRelease := (name == "eval" || name == "evalsha") && task234CommandContains(cmd, ":lock:")
		err := next(ctx, cmd)
		if isRelease && (err == nil || !strings.Contains(strings.ToUpper(err.Error()), "NOSCRIPT")) {
			return h.err
		}
		return err
	}
}

func task234CommandContains(cmd redis.Cmder, fragment string) bool {
	for _, argument := range cmd.Args() {
		if strings.Contains(fmt.Sprint(argument), fragment) {
			return true
		}
	}
	return false
}

type task234DeadlineSolver struct {
	mealID uuid.UUID
	calls  int
}

func (s *task234DeadlineSolver) Solve(ctx context.Context, _ optimization.LPModel, _ optimization.ObjectiveFunction) (optimization.LPSolution, error) {
	s.calls++
	if s.calls <= 2 {
		return optimization.LPSolution{s.mealID.String(): 100}, nil
	}
	<-ctx.Done()
	return nil, &optimization.SolverError{Kind: optimization.SolverErrorTimeout, Diagnostic: "private solver body diagnostic"}
}

type task234FailingAdmission struct{ err error }

func (task234FailingAdmission) Acquire(context.Context, OptimizationAdmissionRequest) (OptimizationAdmissionDecision, error) {
	return OptimizationAdmissionDecision{}, nil
}

func (a task234FailingAdmission) Release(context.Context, uuid.UUID, uuid.UUID) error { return a.err }

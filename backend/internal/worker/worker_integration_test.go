package worker

// Implements DESIGN-004 JobQueueManager worker bootstrap integration verification.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestRunPublishesAlternativeBeforeAcknowledgingQueuedJob(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	manager := queue.NewJobQueueManager(client, queue.Config{})
	jobID := uuid.NewString()
	userID := uuid.New()
	dietID := uuid.New()
	mealIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	jobStore := NewRedisOptimizationJobStore(client)
	if err := jobStore.Save(context.Background(), OptimizationJob{
		JobID: jobUUID(t, jobID), UserID: userID, DailyDietID: dietID,
		TolerancePercent: 0, Status: OptimizationJobQueued,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	entryID, err := manager.Enqueue(context.Background(), jobID)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	clpPath := filepath.Join(t.TempDir(), "clp")
	if err := os.WriteFile(clpPath, []byte("#!/bin/sh\nprintf 'Clp version 1.17.11\\n'\n"), 0o700); err != nil {
		t.Fatalf("write CLP fixture: %v", err)
	}
	cfg := config.Config{Environment: "test", CLPExecutable: clpPath, CLPVersion: "1.17.11"}
	inputs := &integrationInputLoader{inputs: optimization.SavedDietOptimizationInputs{
		Request: optimization.DietOptimizationRequest{
			OriginalDiet:     repository.SavedDiet{ID: dietID, UserID: userID, Entries: []repository.SavedDietMealEntry{{MealID: mealIDs[0], Quantity: 100, Unit: "g", Position: 0}}},
			TolerancePercent: 0,
		},
		Meals: []repository.MealEntity{
			{ID: mealIDs[0], Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10}, NormalizedMacrosAvailable: true},
			{ID: mealIDs[1], Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10}, NormalizedMacrosAvailable: true},
			{ID: mealIDs[2], Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10}, NormalizedMacrosAvailable: true},
		},
	}}
	processor := NewOptimizationProcessor(jobStore, inputs, &integrationSolver{mealIDs: mealIDs})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- RunWithProcessor(ctx, cfg, client, processor.ProcessOptimizationJob, processor.Terminal)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, loadErr := jobStore.Load(context.Background(), jobUUID(t, jobID))
		if loadErr != nil {
			t.Fatalf("Load() error = %v", loadErr)
		}
		if job.Status == OptimizationJobCompleted {
			if len(job.Alternatives) != 3 || len(job.Alternatives[0].Meals) == 0 {
				t.Fatalf("published job = %+v, want three alternatives", job)
			}
			seen := make(map[uuid.UUID]struct{}, len(job.Alternatives))
			for _, alternative := range job.Alternatives {
				seen[alternative.Meals[0].MealID] = struct{}{}
			}
			if len(seen) != 3 {
				t.Fatalf("published alternatives are not distinct: %+v", job.Alternatives)
			}
			entries, rangeErr := client.XRange(context.Background(), queue.DefaultStream, entryID, entryID).Result()
			if rangeErr != nil {
				t.Fatalf("XRange() error = %v", rangeErr)
			}
			if len(entries) != 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			cancel()
			if runErr := <-workerDone; runErr != nil {
				t.Fatalf("RunWithProcessor() error after processing = %v", runErr)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-workerDone
	t.Fatalf("worker did not publish a completed alternative and acknowledge entry %s", entryID)
}

// TestRedisOptimizationJobStoreTerminalTransitionsAreAtomic verifies IT-ARCH-004-002 and DESIGN-004 monotonic state publication.
func TestRedisOptimizationJobStoreTerminalTransitionsAreAtomic(t *testing.T) {
	base := openWorkerIntegrationRedis(t)
	jobID := uuid.New()
	userID := uuid.New()
	dietID := uuid.New()
	if err := NewRedisOptimizationJobStore(base).Save(context.Background(), OptimizationJob{JobID: jobID, UserID: userID, DailyDietID: dietID, Status: OptimizationJobQueued}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := NewRedisOptimizationJobStore(base).MarkProcessing(context.Background(), jobID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}

	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parse Redis URL: %v", err)
	}
	firstClient := redis.NewClient(options)
	secondClient := redis.NewClient(options)
	t.Cleanup(func() { _ = firstClient.Close(); _ = secondClient.Close() })
	jobKey := optimizationJobKey(jobID)
	getReady := make(chan struct{})
	releaseGets := make(chan struct{})
	firstPublished := make(chan struct{})
	state := &transitionHookState{jobKey: jobKey, getReady: getReady, releaseGets: releaseGets, firstPublished: firstPublished}
	firstClient.AddHook(&optimizationTransitionHook{state: state, role: "first"})
	secondClient.AddHook(&optimizationTransitionHook{state: state, role: "second"})

	completed := optimization.DietAlternative{Meals: []optimization.MealQuantity{{MealID: uuid.New(), Quantity: 100, Unit: "g", Position: 0}}, Macros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, Calories: 200}
	results := make(chan error, 2)
	go func() {
		results <- NewRedisOptimizationJobStore(firstClient).PublishCompleted(context.Background(), jobID, []optimization.DietAlternative{completed}, time.Now().UTC())
	}()
	go func() {
		results <- NewRedisOptimizationJobStore(secondClient).PublishFailed(context.Background(), jobID, nil, OptimizationJobFailure{Code: optimization.FailureCodeSolverTimeout, Message: "Optimization took too long. Please try again."}, time.Now().UTC())
	}()
	select {
	case <-getReady:
		close(releaseGets)
	case <-time.After(time.Second):
		t.Fatal("terminal transition loads did not reach barrier")
	}
	publicationErrors := []error{<-results, <-results}
	var successful, rejected int
	for _, err := range publicationErrors {
		if err == nil {
			successful++
		} else {
			rejected++
		}
	}
	if successful != 1 || rejected != 1 {
		t.Fatalf("PublishCompleted/PublishFailed errors = %v, want one durable winner and one conflict", publicationErrors)
	}
	final, err := NewRedisOptimizationJobStore(base).Load(context.Background(), jobID)
	if err != nil {
		t.Fatalf("Load() final error = %v", err)
	}
	if final.Status != OptimizationJobCompleted {
		t.Fatalf("final status = %q, want completed after guarded terminal overwrite", final.Status)
	}
}

// TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop verifies
// IT-ARCH-004-007, ARCH-004, DESIGN-004 JobStatusTracker,
// DESIGN-014 MetricsCollector, and SW-REQ-080/SW-REQ-082.
func TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	consumer := "task-208-heartbeat-" + uuid.NewString()
	stop, err := startWorkerHeartbeat(context.Background(), client, consumer)
	if err != nil {
		t.Fatalf("startWorkerHeartbeat() error = %v", err)
	}
	if _, err := client.ZScore(context.Background(), WorkerHeartbeatKey, consumer).Result(); err != nil {
		t.Fatalf("worker heartbeat score error = %v", err)
	}
	stop()
	if _, err := client.ZScore(context.Background(), WorkerHeartbeatKey, consumer).Result(); !errors.Is(err, redis.Nil) {
		t.Fatalf("worker heartbeat after stop = %v, want redis.Nil", err)
	}
}

type transitionHookState struct {
	mu             sync.Mutex
	jobKey         string
	getCount       int
	getReady       chan struct{}
	releaseGets    <-chan struct{}
	firstPublished chan struct{}
}

// Implements DESIGN-004 JobStatusTracker Redis interleaving fixture.
type optimizationTransitionHook struct {
	state *transitionHookState
	role  string
}

func (h *optimizationTransitionHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *optimizationTransitionHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *optimizationTransitionHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		name := strings.ToLower(cmd.Name())
		transitionCommand := name == "set" || name == "eval"
		if name == "get" && commandContains(cmd, h.state.jobKey) {
			h.state.mu.Lock()
			h.state.getCount++
			if h.state.getCount == 2 {
				close(h.state.getReady)
			}
			h.state.mu.Unlock()
			<-h.state.releaseGets
		}
		if transitionCommand && commandContains(cmd, h.state.jobKey) && h.role == "second" {
			<-h.state.firstPublished
		}
		err := next(ctx, cmd)
		if transitionCommand && commandContains(cmd, h.state.jobKey) && h.role == "first" {
			select {
			case <-h.state.firstPublished:
			default:
				close(h.state.firstPublished)
			}
		}
		return err
	}
}

func commandContains(cmd redis.Cmder, value string) bool {
	for _, argument := range cmd.Args() {
		if fmt.Sprint(argument) == value {
			return true
		}
	}
	return false
}

type integrationInputLoader struct {
	inputs optimization.SavedDietOptimizationInputs
}

// Implements DESIGN-004 JobQueueManager worker integration fixture.
func (l *integrationInputLoader) Load(context.Context, OptimizationJob) (optimization.SavedDietOptimizationInputs, error) {
	return l.inputs, nil
}

type integrationSolver struct {
	mealIDs []uuid.UUID
	calls   atomic.Int32
}

// Implements DESIGN-004 LPSolverWrapper worker integration fixture.
func (s *integrationSolver) Solve(_ context.Context, _ optimization.LPModel, _ optimization.ObjectiveFunction) (optimization.LPSolution, error) {
	index := int(s.calls.Add(1)-1) / 2 % len(s.mealIDs)
	return optimization.LPSolution{s.mealIDs[index].String(): 100}, nil
}

func jobUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	jobID, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("parse job ID: %v", err)
	}
	return jobID
}

func openWorkerIntegrationRedis(t *testing.T) *redis.Client {
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
		_ = client.Close()
		t.Skipf("Redis integration service unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

package worker

// Implements DESIGN-004 JobStatusTracker Task 210 SWE.5 integration verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure verifies
// IT-ARCH-004-002, ARCH-004, DESIGN-004, and SW-REQ-021/SW-REQ-022/SW-REQ-030.
func TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	jobID := uuid.New()
	userID := uuid.New()
	dietID := uuid.New()
	mealID := uuid.New()
	store := NewRedisOptimizationJobStore(client)
	if err := store.Save(context.Background(), OptimizationJob{
		JobID: jobID, UserID: userID, DailyDietID: dietID,
		TargetMacros:     optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10},
		TolerancePercent: 0, Status: OptimizationJobQueued,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stream := "mealswapp:test:task210:partial:" + uuid.NewString()
	manager := queue.NewJobQueueManager(client, queue.Config{
		Stream: stream, Group: "task210-workers", Consumer: "task210-" + uuid.NewString(),
		VisibilityTimeout: 31 * time.Second, ReadBlock: 10 * time.Millisecond,
		BatchSize: 1, MaxAttempts: 3, CompletedTTL: time.Minute, AttemptTTL: time.Hour,
	})
	t.Cleanup(func() { _ = client.Del(context.Background(), stream).Err() })
	if _, err := manager.Enqueue(context.Background(), jobID.String()); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	delivery, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	inputs := &integrationInputLoader{inputs: optimization.SavedDietOptimizationInputs{
		Request: optimization.DietOptimizationRequest{
			OriginalDiet: repository.SavedDiet{
				ID: dietID, UserID: userID,
				Entries: []repository.SavedDietMealEntry{{MealID: mealID, Quantity: 100, Unit: "g", Position: 0}},
			},
			TargetMacros:     optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10},
			TolerancePercent: 0,
		},
		Meals: []repository.MealEntity{{
			ID: mealID, Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100:              repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10},
			NormalizedMacrosAvailable: true,
		}},
	}}
	solver := &task210PartialSolver{}
	processor := NewOptimizationProcessor(store, inputs, solver)
	if err := manager.Process(context.Background(), delivery, processor.ProcessOptimizationJob); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	job, err := store.Load(context.Background(), jobID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if job.Status != OptimizationJobFailed || job.Failure == nil || job.Failure.Code != optimization.FailureCodeSolverTimeout {
		t.Fatalf("terminal job = %+v, want solver_timeout failure", job)
	}
	if len(job.Alternatives) != 1 || len(job.Alternatives[0].Meals) != 1 || job.Alternatives[0].Meals[0].MealID != mealID {
		t.Fatalf("partial alternatives = %+v, want one validated meal alternative", job.Alternatives)
	}
	if solver.calls != 2 {
		t.Fatalf("solver calls = %d, want first result plus later failure", solver.calls)
	}
	pending, err := client.XPending(context.Background(), stream, "task210-workers").Result()
	if err != nil {
		t.Fatalf("XPending() error = %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending deliveries = %d, want terminal publication acknowledged", pending.Count)
	}
}

// TestTask210RedisJobStoreExpiresResultsWithOwnerMarker verifies
// IT-ARCH-004-008, ARCH-004, DESIGN-004, and SW-REQ-006/SW-REQ-043.
func TestTask210RedisJobStoreExpiresResultsWithOwnerMarker(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	jobID := uuid.New()
	ownerID := uuid.New()
	store := NewRedisOptimizationJobStoreWithTTL(client, 25*time.Millisecond)
	if err := store.Save(context.Background(), OptimizationJob{
		JobID: jobID, UserID: ownerID, DailyDietID: uuid.New(),
		TargetMacros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10},
		Status:       OptimizationJobQueued,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := store.MarkProcessing(context.Background(), jobID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}
	alternative := optimization.DietAlternative{
		Meals:  []optimization.MealQuantity{{MealID: uuid.New(), Quantity: 100, Unit: "g", Position: 0}},
		Macros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, Calories: 290,
	}
	if err := store.PublishCompleted(context.Background(), jobID, []optimization.DietAlternative{alternative}, time.Now().UTC()); err != nil {
		t.Fatalf("PublishCompleted() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if exists, err := client.Exists(context.Background(), optimizationJobKey(jobID)).Result(); err != nil {
			t.Fatalf("job key existence error = %v", err)
		} else if exists == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if exists, err := client.Exists(context.Background(), optimizationJobKey(jobID)).Result(); err != nil {
		t.Fatalf("job key final existence error = %v", err)
	} else if exists != 0 {
		t.Fatal("completed result did not expire")
	}

	_, err := store.Load(context.Background(), jobID)
	var expired OptimizationJobExpiredError
	if !errors.As(err, &expired) || expired.UserID != ownerID {
		t.Fatalf("expired Load() error = %v, owner = %s, want owner marker %s", err, expired.UserID, ownerID)
	}
	if !errors.Is(err, ErrOptimizationJobNotFound) {
		t.Fatalf("expired Load() error = %v, want not-found classification", err)
	}
}

type task210PartialSolver struct {
	calls int
}

// Solve returns one valid alternative, then simulates the bounded solver
// timeout that must preserve the already validated partial result.
func (s *task210PartialSolver) Solve(_ context.Context, model optimization.LPModel, _ optimization.ObjectiveFunction) (optimization.LPSolution, error) {
	s.calls++
	if s.calls == 1 {
		return optimization.LPSolution{model.Variables[0].ItemID: 100}, nil
	}
	return nil, &optimization.SolverError{Kind: optimization.SolverErrorTimeout}
}

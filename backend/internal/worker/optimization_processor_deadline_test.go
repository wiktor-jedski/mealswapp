package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-004 JobQueueManager whole-job deadline verification.
func TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves(t *testing.T) {
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000071")
	mealID := uuid.MustParse("00000000-0000-0000-0000-000000000072")
	store := &deadlineJobStore{job: OptimizationJob{
		JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued, CreatedAt: time.Now(),
	}}
	inputs := &deadlineInputLoader{result: optimization.SavedDietOptimizationInputs{
		Request: optimization.DietOptimizationRequest{
			TargetMacros: optimization.MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 10},
		},
		Meals: []repository.MealEntity{{
			ID: mealID, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 10},
		}},
	}}
	processor := NewOptimizationProcessor(store, inputs, &deadlineSolver{mealID: mealID})
	admission := &deadlineAdmission{}
	processor.WithAdmissionGate(admission)
	processor.jobDeadline = 20 * time.Millisecond
	processor.finalizationTimeout = time.Second

	err := processor.ProcessOptimizationJob(context.Background(), queue.Job{ID: jobID.String(), EntryID: "1-0", Attempt: 1})
	if err != nil {
		t.Fatalf("ProcessOptimizationJob() error = %v", err)
	}
	if store.failure.Code != optimization.FailureCodeSolverTimeout {
		t.Fatalf("failure code = %q, want %q", store.failure.Code, optimization.FailureCodeSolverTimeout)
	}
	if len(store.alternatives) != 1 {
		t.Fatalf("partial alternatives = %d, want 1", len(store.alternatives))
	}
	if store.publicationContextErr != nil {
		t.Fatalf("publication context error = %v, want live finalization context", store.publicationContextErr)
	}
	if admission.releases != 1 {
		t.Fatalf("admission releases = %d, want 1", admission.releases)
	}
}

// Implements DESIGN-004 JobQueueManager ownership-window verification.
func TestOptimizationDeadlineFitsWithinQueueVisibility(t *testing.T) {
	if OptimizationJobDeadline+OptimizationFinalizationTimeout >= queue.DefaultVisibilityTimeout {
		t.Fatalf(
			"processing plus finalization = %s, want less than queue visibility %s",
			OptimizationJobDeadline+OptimizationFinalizationTimeout,
			queue.DefaultVisibilityTimeout,
		)
	}
}

type deadlineJobStore struct {
	job                   OptimizationJob
	failure               OptimizationJobFailure
	alternatives          []optimization.DietAlternative
	publicationContextErr error
}

func (s *deadlineJobStore) Load(context.Context, uuid.UUID) (OptimizationJob, error) {
	return s.job, nil
}

func (s *deadlineJobStore) MarkProcessing(_ context.Context, _ uuid.UUID, startedAt time.Time) (OptimizationJob, error) {
	s.job.Status = OptimizationJobProcessing
	s.job.StartedAt = &startedAt
	return s.job, nil
}

func (s *deadlineJobStore) PublishCompleted(context.Context, uuid.UUID, []optimization.DietAlternative, time.Time) error {
	return nil
}

func (s *deadlineJobStore) PublishFailed(ctx context.Context, _ uuid.UUID, alternatives []optimization.DietAlternative, failure OptimizationJobFailure, _ time.Time) error {
	s.publicationContextErr = ctx.Err()
	s.alternatives = append([]optimization.DietAlternative(nil), alternatives...)
	s.failure = failure
	return nil
}

type deadlineInputLoader struct {
	result optimization.SavedDietOptimizationInputs
}

func (l *deadlineInputLoader) Load(context.Context, OptimizationJob) (optimization.SavedDietOptimizationInputs, error) {
	return l.result, nil
}

type deadlineSolver struct {
	mealID uuid.UUID
	calls  int
}

type deadlineAdmission struct{ releases int }

func (a *deadlineAdmission) Acquire(context.Context, OptimizationAdmissionRequest) (OptimizationAdmissionDecision, error) {
	return OptimizationAdmissionDecision{}, nil
}

func (a *deadlineAdmission) Release(context.Context, uuid.UUID, uuid.UUID) error {
	a.releases++
	return nil
}

func (s *deadlineSolver) Solve(ctx context.Context, _ optimization.LPModel, _ optimization.ObjectiveFunction) (optimization.LPSolution, error) {
	s.calls++
	if s.calls == 1 {
		return optimization.LPSolution{s.mealID.String(): 100}, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

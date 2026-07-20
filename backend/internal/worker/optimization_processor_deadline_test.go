package worker

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

// Implements DESIGN-004 JobQueueManager whole-job deadline verification.
// TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves verifies
// IT-ARCH-004-005, ARCH-004, DESIGN-004 LPSolverWrapper/JobStatusTracker, and
// SW-REQ-021/SW-REQ-022/SW-REQ-080 whole-job timeout publication.
func TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves(t *testing.T) {
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000071")
	mealID := uuid.MustParse("00000000-0000-0000-0000-000000000072")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000073")
	dietID := uuid.MustParse("00000000-0000-4000-8000-000000000074")
	store := &deadlineJobStore{job: OptimizationJob{
		JobID: jobID, UserID: userID, DailyDietID: dietID, Status: OptimizationJobQueued, CreatedAt: time.Now(),
	}}
	inputs := &deadlineInputLoader{result: optimization.SavedDietOptimizationInputs{
		Request: optimization.DietOptimizationRequest{
			OriginalDiet: repository.SavedDiet{ID: dietID, UserID: userID, Entries: []repository.SavedDietMealEntry{{MealID: mealID, Quantity: 100, Unit: "g"}}},
		},
		Meals: []repository.MealEntity{{
			ID: mealID, Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 10}, NormalizedMacrosAvailable: true,
		}},
	}}
	processor := NewOptimizationProcessor(store, inputs, &deadlineSolver{mealID: mealID})
	admission := &deadlineAdmission{}
	processor.WithAdmissionGate(admission)
	processor.jobDeadline = 20 * time.Millisecond
	processor.finalizationTimeout = time.Second

	publication, err := processor.ProcessOptimizationJob(context.Background(), queue.Job{ID: jobID.String(), EntryID: "1-0", Attempt: 1})
	if err != nil {
		t.Fatalf("ProcessOptimizationJob() error = %v", err)
	}
	if publication != queue.PublishedFailed {
		t.Fatalf("ProcessOptimizationJob() publication = %q, want failed", publication)
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

// Implements DESIGN-004 JobQueueManager shutdown cancellation retry semantics.
// TestOptimizationProcessorLeavesShutdownCancellationPendingForRetry verifies
// IT-ARCH-004-005, ARCH-004, DESIGN-004 JobQueueManager/JobStatusTracker, and
// SW-REQ-021/SW-REQ-080 recoverable worker-shutdown cancellation.
func TestOptimizationProcessorLeavesShutdownCancellationPendingForRetry(t *testing.T) {
	jobID := uuid.New()
	store := &deadlineJobStore{job: OptimizationJob{JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued}}
	processor := NewOptimizationProcessor(store, &deadlineInputLoader{}, &deadlineSolver{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	publication, err := processor.ProcessOptimizationJob(ctx, queue.Job{ID: jobID.String(), Attempt: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ProcessOptimizationJob() error = %v, want context cancellation", err)
	}
	if publication != "" {
		t.Fatalf("ProcessOptimizationJob() publication = %q, want none", publication)
	}
	if store.failure.Code.Valid() || len(store.alternatives) != 0 {
		t.Fatalf("shutdown cancellation published terminal state: %+v", store.failure)
	}
}

// Implements DESIGN-004 JobStatusTracker persisted safe-message validation.
func TestValidateOptimizationJobRejectsUnboundedTerminalFailureShapes(t *testing.T) {
	base := OptimizationJob{JobID: uuid.New(), UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobFailed}
	tests := []struct {
		name    string
		failure *OptimizationJobFailure
	}{
		{name: "missing failure"},
		{name: "empty code", failure: &OptimizationJobFailure{Message: "Optimization could not be completed. Please try again."}},
		{name: "diagnostic message", failure: &OptimizationJobFailure{Code: optimization.FailureCodeWorkerCrash, Message: "dial redis.internal:6379"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := base
			job.Failure = tt.failure
			if err := validateOptimizationJob(job); err == nil {
				t.Fatalf("validateOptimizationJob(%+v) succeeded", tt.failure)
			}
		})
	}
	base.Failure = &OptimizationJobFailure{Code: optimization.FailureCodeWorkerCrash, Message: "Optimization could not be completed. Please try again."}
	if err := validateOptimizationJob(base); err != nil {
		t.Fatalf("validateOptimizationJob(canonical failure) error = %v", err)
	}
}

// Implements DESIGN-004 JobStatusTracker bounded producer, telemetry, and message vocabulary.
func TestTerminalFailureVocabularyHasCanonicalConsumers(t *testing.T) {
	tests := []struct {
		code      optimization.OptimizationFailureCode
		message   string
		telemetry string
	}{
		{optimization.FailureCodeValidation, "The optimization request could not be validated.", "validation"},
		{optimization.FailureCodeSolverTimeout, "Optimization took too long. Please try again.", "timeout"},
		{optimization.FailureCodeSolverInfeasible, "No meal combination matches the requested targets.", "infeasible"},
		{optimization.FailureCodeWorkerCrash, "Optimization could not be completed. Please try again.", "worker_crash"},
	}
	for _, tt := range tests {
		if got := safeFailureMessage(tt.code); got != tt.message {
			t.Errorf("safeFailureMessage(%s) = %q, want %q", tt.code, got, tt.message)
		}
		if got := telemetryStatusForFailure(tt.code); got != tt.telemetry {
			t.Errorf("telemetryStatusForFailure(%s) = %q, want %q", tt.code, got, tt.telemetry)
		}
	}
}

// Implements DESIGN-004 JobQueueManager typed-nil retry and telemetry policy.
func TestOptimizationProcessorTreatsTypedNilFailureAsRetryableUnknown(t *testing.T) {
	var typedFailure *optimization.OptimizationFailure
	var typedRepositoryError *repository.Error
	for _, err := range []error{typedFailure, typedRepositoryError} {
		store := &deadlineJobStore{}
		processor := NewOptimizationProcessor(store, &deadlineInputLoader{}, &deadlineSolver{})
		publication, got := processor.handleProcessingError(context.Background(), context.Background(), uuid.New(), uuid.New(), nil, err)
		if got != err {
			t.Fatalf("handleProcessingError(%T typed nil) = %v, want original retryable error", err, got)
		}
		if publication != "" {
			t.Fatalf("handleProcessingError(%T typed nil) publication = %q, want none", err, publication)
		}
		if store.failure.Code.Valid() {
			t.Fatalf("handleProcessingError(%T typed nil) published %+v", err, store.failure)
		}
		if status := solverTelemetryStatus(err); status != "failed" {
			t.Fatalf("solverTelemetryStatus(%T typed nil) = %q, want failed", err, status)
		}
	}
}

// Implements DESIGN-004 JobQueueManager invalid failure normalization and retry policy.
func TestOptimizationProcessorTreatsInvalidFailureAsRetryableWorkerCrash(t *testing.T) {
	mealID := uuid.New()
	userID := uuid.New()
	dietID := uuid.New()
	jobID := uuid.New()
	store := &deadlineJobStore{job: OptimizationJob{JobID: jobID, UserID: userID, DailyDietID: dietID, Status: OptimizationJobQueued}}
	inputs := &deadlineInputLoader{result: optimization.SavedDietOptimizationInputs{
		Request: optimization.DietOptimizationRequest{OriginalDiet: repository.SavedDiet{
			ID: dietID, UserID: userID, Entries: []repository.SavedDietMealEntry{{MealID: mealID, Quantity: 100, Unit: "g"}},
		}},
		Meals: []repository.MealEntity{{
			ID: mealID, Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 10}, NormalizedMacrosAvailable: true,
		}},
	}}
	processor := NewOptimizationProcessor(store, inputs, &invalidFailureSolver{})
	publication, err := processor.ProcessOptimizationJob(context.Background(), queue.Job{ID: jobID.String(), Attempt: 1})
	if code := optimization.FailureCodeOf(err); code != optimization.FailureCodeWorkerCrash {
		t.Fatalf("ProcessOptimizationJob() error = %v, code %q, want retryable worker_crash", err, code)
	}
	if publication != "" {
		t.Fatalf("ProcessOptimizationJob() publication = %q, want none", publication)
	}
	if store.failure.Code.Valid() {
		t.Fatalf("ProcessOptimizationJob() published terminal failure %+v", store.failure)
	}
	if status := solverTelemetryStatus(err); status != "failed" {
		t.Fatalf("solverTelemetryStatus() = %q, want failed", status)
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

type invalidFailureSolver struct{}

func (*invalidFailureSolver) Solve(context.Context, optimization.LPModel, optimization.ObjectiveFunction) (optimization.LPSolution, error) {
	return nil, &optimization.OptimizationFailure{}
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
	if s.calls <= 2 {
		return optimization.LPSolution{s.mealID.String(): 100}, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-004 JobQueueManager worker-owned job state.
const (
	optimizationJobKeyPrefix     = "mealswapp:optimization:job:v1:"
	optimizationExpiredKeyPrefix = "mealswapp:optimization:expired:v1:"
	optimizationJobTTL           = time.Hour
	// OptimizationJobDeadline bounds repository loading and all alternative
	// solver attempts under one ownership-safe processing window.
	OptimizationJobDeadline = 30 * time.Second
	// OptimizationFinalizationTimeout bounds terminal timeout publication after
	// the processing context has expired.
	OptimizationFinalizationTimeout = 5 * time.Second
)

// OptimizationJobStatus is the worker-visible terminal state vocabulary.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationJobStatus string

// Implements DESIGN-004 JobStatusTracker.
const (
	// OptimizationJobQueued identifies a job awaiting worker reservation.
	OptimizationJobQueued OptimizationJobStatus = "queued"
	// OptimizationJobProcessing identifies a job owned by an active worker.
	OptimizationJobProcessing OptimizationJobStatus = "processing"
	// OptimizationJobCompleted identifies a job with validated alternatives.
	OptimizationJobCompleted OptimizationJobStatus = "completed"
	// OptimizationJobFailed identifies a job with a safe terminal failure.
	OptimizationJobFailed OptimizationJobStatus = "failed"
	// OptimizationJobCancelled identifies a job cancelled before completion.
	OptimizationJobCancelled OptimizationJobStatus = "cancelled"
)

// OptimizationJobFailure contains only a stable user-safe failure code and
// message. Solver, database, and Redis diagnostics never cross this boundary.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationJobFailure struct {
	Code    optimization.OptimizationFailureCode `json:"code"`
	Message string                               `json:"message"`
}

// Valid reports whether a failure contains one retained code and its canonical safe message.
// Implements DESIGN-004 JobStatusTracker producer/consumer boundary.
func (f OptimizationJobFailure) Valid() bool {
	return f.Code.Valid() && f.Message == safeFailureMessage(f.Code)
}

// OptimizationJob is the Redis-backed worker envelope. The stream carries
// only JobID; this record carries the server-owned diet and constraint inputs.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
type OptimizationJob struct {
	JobID            uuid.UUID                      `json:"jobId"`
	UserID           uuid.UUID                      `json:"userId"`
	DailyDietID      uuid.UUID                      `json:"dailyDietId"`
	TolerancePercent float64                        `json:"tolerancePercent"`
	ExcludedMealIDs  []uuid.UUID                    `json:"excludedMealIds"`
	Status           OptimizationJobStatus          `json:"status"`
	CreatedAt        time.Time                      `json:"createdAt"`
	StartedAt        *time.Time                     `json:"startedAt,omitempty"`
	FinishedAt       *time.Time                     `json:"finishedAt,omitempty"`
	Alternatives     []optimization.DietAlternative `json:"alternatives,omitempty"`
	Failure          *OptimizationJobFailure        `json:"failure,omitempty"`
}

// ErrOptimizationJobNotFound identifies an expired or unknown worker job.
// Implements DESIGN-004 JobStatusTracker.
var ErrOptimizationJobNotFound = errors.New("optimization job not found")

// OptimizationJobExpiredError identifies a result that expired while retaining
// the owner needed to keep cross-user polling indistinguishable from missing.
// Implements DESIGN-004 JobStatusTracker result TTL.
type OptimizationJobExpiredError struct {
	UserID uuid.UUID
}

// Error returns the stable internal expiration classification.
// Implements DESIGN-004 JobStatusTracker.
func (e OptimizationJobExpiredError) Error() string { return "optimization job result expired" }

// Is lets callers classify an expired result without exposing owner metadata.
// Implements DESIGN-004 JobStatusTracker.
func (e OptimizationJobExpiredError) Is(target error) bool {
	return target == ErrOptimizationJobNotFound
}

// OptimizationJobStore persists the worker envelope and authoritative result.
// Implementations must make terminal publication idempotent.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationJobStore interface {
	Load(context.Context, uuid.UUID) (OptimizationJob, error)
	MarkProcessing(context.Context, uuid.UUID, time.Time) (OptimizationJob, error)
	PublishCompleted(context.Context, uuid.UUID, []optimization.DietAlternative, time.Time) error
	PublishFailed(context.Context, uuid.UUID, []optimization.DietAlternative, OptimizationJobFailure, time.Time) error
}

// RedisOptimizationJobStore stores job input and terminal state as one bounded
// Redis JSON value. The stream remains an ID-only delivery mechanism.
// Implements DESIGN-004 JobStatusTracker.
type RedisOptimizationJobStore struct {
	client    redis.UniversalClient
	ttl       time.Duration
	telemetry *observability.OptimizationTelemetry
}

// NewRedisOptimizationJobStore constructs the worker's Redis job-state store.
// Implements DESIGN-004 JobStatusTracker.
func NewRedisOptimizationJobStore(client redis.UniversalClient) *RedisOptimizationJobStore {
	return &RedisOptimizationJobStore{client: client, ttl: optimizationJobTTL}
}

// NewRedisOptimizationJobStoreWithTTL constructs a job store with a testable
// result lifetime while preserving the one-hour production default.
// Implements DESIGN-004 JobStatusTracker result TTL.
func NewRedisOptimizationJobStoreWithTTL(client redis.UniversalClient, ttl time.Duration) *RedisOptimizationJobStore {
	if ttl <= 0 {
		ttl = optimizationJobTTL
	}
	return &RedisOptimizationJobStore{client: client, ttl: ttl}
}

// WithTelemetry attaches result-expiry telemetry without adding identifiers
// to metric labels or logs.
// Implements DESIGN-004 JobStatusTracker and DESIGN-014 MetricsCollector.
func (s *RedisOptimizationJobStore) WithTelemetry(telemetry *observability.OptimizationTelemetry) *RedisOptimizationJobStore {
	if s != nil {
		s.telemetry = telemetry
	}
	return s
}

// Save writes one queued server-owned optimization job. API submission can
// use this method before calling JobQueueManager.Enqueue.
// Implements DESIGN-004 JobStatusTracker.
func (s *RedisOptimizationJobStore) Save(ctx context.Context, job OptimizationJob) error {
	if job.Status == "" {
		job.Status = OptimizationJobQueued
	}
	if err := validateOptimizationJob(job); err != nil {
		return err
	}
	if job.Status != OptimizationJobQueued {
		return errors.New("new optimization job must be queued")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	_, err := s.transition(ctx, job, "save")
	return err
}

// Load reads one queued, processing, or terminal optimization job.
// Implements DESIGN-004 JobStatusTracker.
func (s *RedisOptimizationJobStore) Load(ctx context.Context, jobID uuid.UUID) (OptimizationJob, error) {
	if err := validateJobID(jobID); err != nil {
		return OptimizationJob{}, err
	}
	if s == nil || s.client == nil {
		return OptimizationJob{}, queueUnavailable("load optimization job", errors.New("Redis client is required"))
	}
	payload, err := s.client.Get(ctx, optimizationJobKey(jobID)).Bytes()
	if errors.Is(err, redis.Nil) {
		owner, markerErr := s.client.Get(ctx, optimizationExpiredKey(jobID)).Result()
		if markerErr == nil {
			if s.telemetry != nil {
				s.telemetry.ResultExpired(ctx)
			}
			ownerID, parseErr := uuid.Parse(owner)
			if parseErr == nil {
				return OptimizationJob{}, OptimizationJobExpiredError{UserID: ownerID}
			}
			return OptimizationJob{}, OptimizationJobExpiredError{}
		}
		if !errors.Is(markerErr, redis.Nil) {
			return OptimizationJob{}, queueUnavailable("load expired optimization job", markerErr)
		}
		return OptimizationJob{}, ErrOptimizationJobNotFound
	}
	if err != nil {
		return OptimizationJob{}, queueUnavailable("load optimization job", err)
	}
	var job OptimizationJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return OptimizationJob{}, fmt.Errorf("optimization job state is malformed: %w", err)
	}
	if err := validateOptimizationJob(job); err != nil {
		return OptimizationJob{}, err
	}
	return job, nil
}

// MarkProcessing records the worker start time while preserving terminal
// states so duplicate delivery cannot regress an authoritative result.
// Implements DESIGN-004 JobStatusTracker.
func (s *RedisOptimizationJobStore) MarkProcessing(ctx context.Context, jobID uuid.UUID, startedAt time.Time) (OptimizationJob, error) {
	job, err := s.Load(ctx, jobID)
	if err != nil {
		return OptimizationJob{}, err
	}
	if terminalJobStatus(job.Status) {
		return job, nil
	}
	job.Status = OptimizationJobProcessing
	if job.StartedAt == nil {
		startedAt = startedAt.UTC()
		job.StartedAt = &startedAt
	}
	changed, err := s.transition(ctx, job, "processing")
	if err != nil {
		return OptimizationJob{}, err
	}
	if !changed {
		return s.Load(ctx, jobID)
	}
	return job, nil
}

// PublishCompleted atomically replaces the worker envelope with validated
// alternatives and a completed terminal state.
// Implements DESIGN-004 JobStatusTracker authoritative result publication.
func (s *RedisOptimizationJobStore) PublishCompleted(ctx context.Context, jobID uuid.UUID, alternatives []optimization.DietAlternative, finishedAt time.Time) error {
	job, err := s.Load(ctx, jobID)
	if err != nil {
		return err
	}
	if terminalJobStatus(job.Status) {
		return requireDurableTerminalStatus(job.Status, OptimizationJobCompleted)
	}
	if len(alternatives) == 0 || len(alternatives) > optimization.MaxAlternativeCount {
		return errors.New("completed optimization job requires one to three alternatives")
	}
	if err := validateOptimizationAlternatives(alternatives); err != nil {
		return err
	}
	job.Status = OptimizationJobCompleted
	job.Alternatives = append([]optimization.DietAlternative(nil), alternatives...)
	job.Failure = nil
	finishedAt = finishedAt.UTC()
	job.FinishedAt = &finishedAt
	changed, err := s.transition(ctx, job, "completed")
	if err != nil || changed {
		return err
	}
	current, err := s.Load(ctx, jobID)
	if err != nil {
		return err
	}
	return requireDurableTerminalStatus(current.Status, OptimizationJobCompleted)
}

// PublishFailed records a safe terminal failure and any valid partial
// alternatives before the queue acknowledges the delivery.
// Implements DESIGN-004 JobStatusTracker authoritative failure publication.
func (s *RedisOptimizationJobStore) PublishFailed(ctx context.Context, jobID uuid.UUID, alternatives []optimization.DietAlternative, failure OptimizationJobFailure, finishedAt time.Time) error {
	job, err := s.Load(ctx, jobID)
	if err != nil {
		return err
	}
	if terminalJobStatus(job.Status) {
		return requireDurableTerminalStatus(job.Status, OptimizationJobFailed)
	}
	if !failure.Valid() {
		return errors.New("failed optimization job requires a safe failure")
	}
	if len(alternatives) > optimization.MaxAlternativeCount {
		return errors.New("failed optimization job has too many alternatives")
	}
	if err := validateOptimizationAlternatives(alternatives); err != nil {
		return err
	}
	job.Status = OptimizationJobFailed
	job.Alternatives = append([]optimization.DietAlternative(nil), alternatives...)
	job.Failure = &failure
	finishedAt = finishedAt.UTC()
	job.FinishedAt = &finishedAt
	changed, err := s.transition(ctx, job, "failed")
	if err != nil || changed {
		return err
	}
	current, err := s.Load(ctx, jobID)
	if err != nil {
		return err
	}
	return requireDurableTerminalStatus(current.Status, OptimizationJobFailed)
}

// requireDurableTerminalStatus prevents queue publication when Redis rejected
// the requested terminal transition or already retained the opposite result.
// Implements DESIGN-004 JobStatusTracker publication-before-ACK ordering.
func requireDurableTerminalStatus(current, expected OptimizationJobStatus) error {
	if current != expected {
		return errors.New("optimization terminal publication conflicts with durable state")
	}
	return nil
}

// Delete removes a saved job that could not be published to the queue.
// Implements DESIGN-004 JobStatusTracker queue failure rollback.
func (s *RedisOptimizationJobStore) Delete(ctx context.Context, jobID uuid.UUID) error {
	if err := validateJobID(jobID); err != nil {
		return err
	}
	if s == nil || s.client == nil {
		return queueUnavailable("delete optimization job", errors.New("Redis client is required"))
	}
	if err := s.client.Del(ctx, optimizationJobKey(jobID), optimizationExpiredKey(jobID)).Err(); err != nil {
		return queueUnavailable("delete optimization job", err)
	}
	return nil
}

// write serializes and stores one queued worker job without regressing state.
// Implements DESIGN-004 JobStatusTracker.
func (s *RedisOptimizationJobStore) write(ctx context.Context, job OptimizationJob) error {
	_, err := s.transition(ctx, job, "save")
	return err
}

// transition performs one compare-and-set state transition with Redis-side terminal guards.
// Implements DESIGN-004 JobStatusTracker atomic monotonic publication.
func (s *RedisOptimizationJobStore) transition(ctx context.Context, job OptimizationJob, operation string) (bool, error) {
	if s == nil || s.client == nil {
		return false, queueUnavailable("transition optimization job", errors.New("Redis client is required"))
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return false, fmt.Errorf("marshal optimization job: %w", err)
	}
	markerTTL := s.ttl * 2
	if markerTTL < time.Hour {
		markerTTL = time.Hour
	}
	result, err := s.client.Eval(ctx, optimizationStateTransitionScript, []string{optimizationJobKey(job.JobID), optimizationExpiredKey(job.JobID)}, payload, operation, durationMilliseconds(s.ttl), job.UserID.String(), durationMilliseconds(markerTTL)).Int64()
	if err != nil {
		return false, queueUnavailable("transition optimization job", err)
	}
	if result < 0 {
		return false, ErrOptimizationJobNotFound
	}
	return result == 1, nil
}

// durationMilliseconds keeps Redis PX arguments valid for short integration-test TTLs.
// Implements DESIGN-004 JobStatusTracker result TTL.
func durationMilliseconds(value time.Duration) string {
	milliseconds := value.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	return strconv.FormatInt(milliseconds, 10)
}

// OptimizationInputLoader reloads the owned diet and all eligible meals from
// repository state before model construction.
// Implements DESIGN-004 ConstraintBuilder and JobQueueManager.
type OptimizationInputLoader interface {
	Load(context.Context, OptimizationJob) (optimization.SavedDietOptimizationInputs, error)
}

// RepositoryOptimizationInputLoader adapts ConstraintBuilder to the worker
// processor without allowing queue payloads to supply meal data.
// Implements DESIGN-004 ConstraintBuilder and JobQueueManager.
type RepositoryOptimizationInputLoader struct {
	builder *optimization.ConstraintBuilder
}

// NewRepositoryOptimizationInputLoader creates a repository-backed input loader.
// Implements DESIGN-004 ConstraintBuilder and JobQueueManager.
func NewRepositoryOptimizationInputLoader(builder *optimization.ConstraintBuilder) *RepositoryOptimizationInputLoader {
	return &RepositoryOptimizationInputLoader{builder: builder}
}

// Load reloads the saved diet and eligible repository meals under the job owner.
// Implements DESIGN-004 ConstraintBuilder and JobQueueManager.
func (l *RepositoryOptimizationInputLoader) Load(ctx context.Context, job OptimizationJob) (optimization.SavedDietOptimizationInputs, error) {
	if l == nil || l.builder == nil {
		return optimization.SavedDietOptimizationInputs{}, errors.New("optimization input loader is required")
	}
	return l.builder.LoadFromSavedDiet(ctx, job.UserID, job.DailyDietID, optimization.DietOptimizationRequest{
		TolerancePercent: job.TolerancePercent,
		ExcludedMealIDs:  append([]uuid.UUID(nil), job.ExcludedMealIDs...),
	})
}

// OptimizationSolver is the injectable solver seam used by the worker.
// Implements DESIGN-004 LPSolverWrapper and JobQueueManager.
type OptimizationSolver interface {
	Solve(context.Context, optimization.LPModel, optimization.ObjectiveFunction) (optimization.LPSolution, error)
}

// OptimizationProcessor owns the complete worker-side orchestration from job
// state through repository reload, model generation, solving, validation, and
// authoritative terminal publication.
// Implements DESIGN-004 JobQueueManager, ConstraintBuilder, ObjectiveFunction,
// DiversityPenalizer, SolutionValidator, and JobStatusTracker.
// Implements DESIGN-004 JobQueueManager.
type OptimizationProcessor struct {
	jobs                OptimizationJobStore
	inputs              OptimizationInputLoader
	solver              OptimizationSolver
	telemetry           *observability.OptimizationTelemetry
	jobDeadline         time.Duration
	finalizationTimeout time.Duration
	admission           OptimizationAdmissionGate
}

// NewOptimizationProcessor constructs a dedicated worker processor.
// Implements DESIGN-004 JobQueueManager.
func NewOptimizationProcessor(jobs OptimizationJobStore, inputs OptimizationInputLoader, solver OptimizationSolver) *OptimizationProcessor {
	return &OptimizationProcessor{
		jobs:                jobs,
		inputs:              inputs,
		solver:              solver,
		jobDeadline:         OptimizationJobDeadline,
		finalizationTimeout: OptimizationFinalizationTimeout,
	}
}

// WithTelemetry attaches worker and solver telemetry to the dedicated worker
// orchestration boundary.
// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector.
func (p *OptimizationProcessor) WithTelemetry(telemetry *observability.OptimizationTelemetry) *OptimizationProcessor {
	if p != nil {
		p.telemetry = telemetry
	}
	return p
}

// WithAdmissionGate releases per-user capacity after authoritative terminal publication.
// Implements DESIGN-004 JobStatusTracker.
func (p *OptimizationProcessor) WithAdmissionGate(admission OptimizationAdmissionGate) *OptimizationProcessor {
	if p != nil {
		p.admission = admission
	}
	return p
}

// ProcessOptimizationJob executes one queued optimization job and returns nil only after a
// completed or safe terminal failure has been published. Queue ACK therefore
// occurs only after authoritative job handling.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
func (p *OptimizationProcessor) ProcessOptimizationJob(ctx context.Context, delivery queue.Job) (queue.TerminalPublication, error) {
	if p == nil || p.jobs == nil || p.inputs == nil || p.solver == nil {
		return "", errors.New("optimization processor dependencies are required")
	}
	if delivery.Attempt > 1 && p.telemetry != nil {
		p.telemetry.Retry(ctx, "retry")
	}
	jobID, err := uuid.Parse(delivery.ID)
	if err != nil {
		return "", fmt.Errorf("optimization job ID is invalid: %w", err)
	}
	jobDeadline := p.jobDeadline
	if jobDeadline <= 0 {
		jobDeadline = OptimizationJobDeadline
	}
	processingCtx, cancelProcessing := context.WithTimeout(ctx, jobDeadline)
	defer cancelProcessing()

	job, err := p.jobs.Load(processingCtx, jobID)
	if err != nil {
		return "", err
	}
	if terminalJobStatus(job.Status) {
		return p.confirmTerminal(processingCtx, job)
	}
	if err := processingCtx.Err(); err != nil {
		return "", err
	}
	job, err = p.jobs.MarkProcessing(processingCtx, jobID, time.Now())
	if err != nil {
		return p.handleProcessingError(ctx, processingCtx, job.UserID, jobID, nil, err)
	}
	if terminalJobStatus(job.Status) {
		return p.confirmTerminal(processingCtx, job)
	}
	if p.telemetry != nil {
		p.telemetry.WorkerStarted(ctx)
		defer p.telemetry.WorkerFinished(ctx)
	}

	inputs, err := p.inputs.Load(processingCtx, job)
	if err != nil {
		return p.handleProcessingError(ctx, processingCtx, job.UserID, jobID, nil, err)
	}
	model, err := optimization.BuildConstraints(inputs.Request, inputs.Meals, nil)
	if err != nil {
		return p.handleProcessingError(ctx, processingCtx, job.UserID, jobID, nil, err)
	}
	if _, err := optimization.BuildObjective(model.Variables); err != nil {
		return p.handleProcessingError(ctx, processingCtx, job.UserID, jobID, nil, err)
	}
	alternatives, err := optimization.GenerateValidatedAlternatives(processingCtx, inputs.Request, inputs.Meals, optimization.MaxAlternativeCount, func(solveCtx context.Context, nextModel optimization.LPModel, objective optimization.ObjectiveFunction) (optimization.LPSolution, error) {
		started := time.Now()
		solution, solveErr := p.solver.Solve(solveCtx, nextModel, objective)
		if p.telemetry != nil {
			p.telemetry.Solve(solveCtx, time.Since(started), solverTelemetryStatus(solveErr))
		}
		return solution, solveErr
	})
	if err != nil {
		return p.handleProcessingError(ctx, processingCtx, job.UserID, jobID, alternatives, err)
	}
	if len(alternatives) == 0 {
		return p.publishFailure(processingCtx, job.UserID, jobID, alternatives, optimization.FailureCodeValidation)
	}
	if err := processingCtx.Err(); err != nil {
		return p.handleProcessingError(ctx, processingCtx, job.UserID, jobID, alternatives, err)
	}
	err = p.jobs.PublishCompleted(processingCtx, jobID, alternatives, time.Now())
	if err == nil && p.telemetry != nil {
		p.telemetry.JobOutcome(ctx, "completed")
	}
	if err != nil {
		return "", err
	}
	if err := p.releaseAdmission(processingCtx, job.UserID, jobID); err != nil {
		return "", err
	}
	return queue.PublishedCompleted, nil
}

// Process is a concise queue.Processor adapter for the orchestration method.
// Implements DESIGN-004 JobQueueManager.
func (p *OptimizationProcessor) Process(ctx context.Context, delivery queue.Job) (queue.TerminalPublication, error) {
	return p.ProcessOptimizationJob(ctx, delivery)
}

// Terminal handles queue exhaustion after retryable worker failures.
// Implements DESIGN-004 JobQueueManager retry terminal handling.
func (p *OptimizationProcessor) Terminal(ctx context.Context, delivery queue.Job, cause error) (queue.TerminalPublication, error) {
	if p == nil || p.jobs == nil {
		return "", errors.New("optimization processor job store is required")
	}
	if p.telemetry != nil {
		p.telemetry.Retry(ctx, "exhausted")
	}
	jobID, err := uuid.Parse(delivery.ID)
	if err != nil {
		return "", err
	}
	job, err := p.jobs.Load(ctx, jobID)
	if err != nil {
		return "", err
	}
	err = p.jobs.PublishFailed(ctx, jobID, nil, OptimizationJobFailure{
		Code:    optimization.FailureCodeWorkerCrash,
		Message: safeFailureMessage(optimization.FailureCodeWorkerCrash),
	}, time.Now())
	if err == nil && p.telemetry != nil {
		p.telemetry.JobOutcome(ctx, "worker_crash")
	}
	if err != nil {
		return "", err
	}
	if err := p.releaseAdmission(ctx, job.UserID, jobID); err != nil {
		return "", err
	}
	return queue.PublishedFailed, nil
}

// handleProcessingError maps terminal solver/validation failures to safe
// publication while leaving infrastructure failures pending for retry.
// Implements DESIGN-004 JobQueueManager retry and JobStatusTracker.
func (p *OptimizationProcessor) handleProcessingError(parentCtx, processingCtx context.Context, userID, jobID uuid.UUID, alternatives []optimization.DietAlternative, err error) (queue.TerminalPublication, error) {
	if errors.Is(processingCtx.Err(), context.DeadlineExceeded) && parentCtx.Err() == nil {
		finalizationTimeout := p.finalizationTimeout
		if finalizationTimeout <= 0 {
			finalizationTimeout = OptimizationFinalizationTimeout
		}
		finalizationCtx, cancelFinalization := context.WithTimeout(context.WithoutCancel(parentCtx), finalizationTimeout)
		defer cancelFinalization()
		return p.publishFailure(finalizationCtx, userID, jobID, alternatives, optimization.FailureCodeSolverTimeout)
	}
	if parentCtx.Err() != nil || errors.Is(err, queue.ErrQueueUnavailable) {
		return "", err
	}
	code := optimization.FailureCodeOf(err)
	if !code.Valid() {
		var repositoryErr *repository.Error
		if errors.As(err, &repositoryErr) && repositoryErr != nil && repositoryErr.Kind == repository.ErrorKindValidation {
			code = optimization.FailureCodeValidation
		}
	}
	if code != optimization.FailureCodeValidation && code != optimization.FailureCodeSolverTimeout && code != optimization.FailureCodeSolverInfeasible {
		return "", err
	}
	return p.publishFailure(processingCtx, userID, jobID, alternatives, code)
}

// publishFailure writes a safe terminal failure before queue acknowledgement.
// Implements DESIGN-004 JobStatusTracker.
func (p *OptimizationProcessor) publishFailure(ctx context.Context, userID, jobID uuid.UUID, alternatives []optimization.DietAlternative, code optimization.OptimizationFailureCode) (queue.TerminalPublication, error) {
	err := p.jobs.PublishFailed(ctx, jobID, alternatives, OptimizationJobFailure{
		Code:    code,
		Message: safeFailureMessage(code),
	}, time.Now())
	if err == nil && p.telemetry != nil {
		p.telemetry.JobOutcome(ctx, telemetryStatusForFailure(code))
	}
	if err != nil {
		return "", err
	}
	if err := p.releaseAdmission(ctx, userID, jobID); err != nil {
		return "", err
	}
	return queue.PublishedFailed, nil
}

// confirmTerminal deliberately maps persisted terminal state to queue finalization.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
func (p *OptimizationProcessor) confirmTerminal(ctx context.Context, job OptimizationJob) (queue.TerminalPublication, error) {
	if err := p.releaseAdmission(ctx, job.UserID, job.JobID); err != nil {
		return "", err
	}
	if job.Status == OptimizationJobCompleted {
		return queue.PublishedCompleted, nil
	}
	return queue.PublishedFailed, nil
}

// releaseAdmission returns terminal per-user capacity when a gate is configured.
// Implements DESIGN-004 JobStatusTracker.
func (p *OptimizationProcessor) releaseAdmission(ctx context.Context, userID, jobID uuid.UUID) error {
	if p == nil || p.admission == nil {
		return nil
	}
	return p.admission.Release(ctx, userID, jobID)
}

// solverTelemetryStatus maps solver errors to a fixed metric status.
// Implements DESIGN-014 MetricsCollector.
func solverTelemetryStatus(err error) string {
	if err == nil {
		return "completed"
	}
	if errors.Is(err, optimization.ErrSolverTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, optimization.ErrSolverInfeasible) {
		return "infeasible"
	}
	switch optimization.FailureCodeOf(err) {
	case optimization.FailureCodeSolverTimeout:
		return "timeout"
	case optimization.FailureCodeSolverInfeasible:
		return "infeasible"
	case optimization.FailureCodeValidation:
		return "validation"
	default:
		return "failed"
	}
}

// telemetryStatusForFailure maps public failures to a fixed metric status.
// Implements DESIGN-014 MetricsCollector.
func telemetryStatusForFailure(code optimization.OptimizationFailureCode) string {
	switch code {
	case optimization.FailureCodeSolverTimeout:
		return "timeout"
	case optimization.FailureCodeSolverInfeasible:
		return "infeasible"
	case optimization.FailureCodeValidation:
		return "validation"
	case optimization.FailureCodeWorkerCrash:
		return "worker_crash"
	default:
		return "failed"
	}
}

// validateOptimizationJob checks the persisted worker envelope shape.
// Implements DESIGN-004 JobStatusTracker.
func validateOptimizationJob(job OptimizationJob) error {
	if err := validateJobID(job.JobID); err != nil {
		return err
	}
	if job.UserID == uuid.Nil || job.DailyDietID == uuid.Nil {
		return errors.New("optimization job owner and daily diet are required")
	}
	if job.Status == "" {
		return errors.New("optimization job status is required")
	}
	switch job.Status {
	case OptimizationJobQueued, OptimizationJobProcessing, OptimizationJobCompleted, OptimizationJobFailed, OptimizationJobCancelled:
	default:
		return errors.New("optimization job status is invalid")
	}
	if job.Status == OptimizationJobFailed {
		if job.Failure == nil || !job.Failure.Valid() {
			return errors.New("failed optimization job requires a bounded safe failure")
		}
	} else if job.Failure != nil {
		return errors.New("non-failed optimization job cannot contain a failure")
	}
	switch job.Status {
	case OptimizationJobCompleted:
		if len(job.Alternatives) == 0 || len(job.Alternatives) > optimization.MaxAlternativeCount {
			return errors.New("completed optimization job requires one to three alternatives")
		}
	case OptimizationJobFailed:
		if len(job.Alternatives) > optimization.MaxAlternativeCount {
			return errors.New("failed optimization job has too many alternatives")
		}
	default:
		if len(job.Alternatives) != 0 {
			return errors.New("non-result optimization job cannot contain alternatives")
		}
	}
	if err := validateOptimizationAlternatives(job.Alternatives); err != nil {
		return err
	}
	return nil
}

// validateOptimizationAlternatives applies the shared authoritative result
// validator to every persisted alternative.
// Implements DESIGN-004 JobStatusTracker authoritative result publication.
func validateOptimizationAlternatives(alternatives []optimization.DietAlternative) error {
	for _, alternative := range alternatives {
		if err := optimization.ValidateDietAlternative(alternative); err != nil {
			return err
		}
	}
	return nil
}

// validateJobID rejects missing logical job identity before Redis access.
// Implements DESIGN-004 JobQueueManager.
func validateJobID(jobID uuid.UUID) error {
	if jobID == uuid.Nil {
		return errors.New("optimization job ID is required")
	}
	return nil
}

// terminalJobStatus identifies states that duplicate deliveries must not regress.
// Implements DESIGN-004 JobStatusTracker.
func terminalJobStatus(status OptimizationJobStatus) bool {
	return status == OptimizationJobCompleted || status == OptimizationJobFailed || status == OptimizationJobCancelled
}

// optimizationJobKey derives the bounded Redis key for one job envelope.
// Implements DESIGN-004 JobStatusTracker.
func optimizationJobKey(jobID uuid.UUID) string {
	return optimizationJobKeyPrefix + jobID.String()
}

// optimizationExpiredKey derives the bounded Redis key retained after result expiry.
// Implements DESIGN-004 JobStatusTracker result TTL and ownership isolation.
func optimizationExpiredKey(jobID uuid.UUID) string {
	return optimizationExpiredKeyPrefix + jobID.String()
}

// queueUnavailable preserves queue outage identity for retry handling.
// Implements DESIGN-004 JobQueueManager outage semantics.
func queueUnavailable(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", queue.ErrQueueUnavailable, operation, err)
}

// safeFailureMessage maps internal failure classes to user-safe text.
// Implements DESIGN-004 JobStatusTracker.
func safeFailureMessage(code optimization.OptimizationFailureCode) string {
	switch code {
	case optimization.FailureCodeValidation:
		return "The optimization request could not be validated."
	case optimization.FailureCodeSolverTimeout:
		return "Optimization took too long. Please try again."
	case optimization.FailureCodeSolverInfeasible:
		return "No meal combination matches the requested targets."
	default:
		return "Optimization could not be completed. Please try again."
	}
}

// optimizationStateTransitionScript atomically guards every job-state write.
// Implements DESIGN-004 JobStatusTracker atomic monotonic publication.
const optimizationStateTransitionScript = `
local currentPayload = redis.call('get', KEYS[1])
if not currentPayload then
  if redis.call('exists', KEYS[2]) == 1 then
    return -1
  end
  if ARGV[2] ~= 'save' then
    return -1
  end
  redis.call('set', KEYS[1], ARGV[1], 'px', ARGV[3])
  return 1
end

local current = cjson.decode(currentPayload)
local status = current.status
local operation = ARGV[2]
local allowed = false
if operation == 'save' then
  allowed = status == 'queued'
elseif operation == 'processing' then
  allowed = status == 'queued' or status == 'processing'
elseif operation == 'completed' or operation == 'failed' then
  allowed = status == 'processing'
end
if not allowed then
  return 0
end

redis.call('set', KEYS[1], ARGV[1], 'px', ARGV[3])
if operation == 'completed' or operation == 'failed' then
  redis.call('set', KEYS[2], current.userId, 'px', ARGV[5])
end
return 1
`

// Compile-time checks keep the concrete worker seams explicit.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
var _ OptimizationJobStore = (*RedisOptimizationJobStore)(nil)

// Compile-time checks keep the repository-backed input loader explicit.
// Implements DESIGN-004 ConstraintBuilder.
var _ OptimizationInputLoader = (*RepositoryOptimizationInputLoader)(nil)

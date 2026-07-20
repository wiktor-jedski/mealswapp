package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// Implements DESIGN-004 JobStatusTracker submission and polling boundary.
const (
	optimizationJobsRoute               = "/optimization/jobs"
	optimizationPollPath                = "/api/v1/optimization/jobs/"
	optimizationMethod                  = "POST"
	minimumIdempotencyKey               = 8
	maximumIdempotencyKey               = 255
	maximumExcludedMeals                = 100
	optimizationAdmissionCleanupTimeout = 100 * time.Millisecond
	optimizationAdmissionCleanupLimit   = 1
)

// Implements DESIGN-004 JobStatusTracker persisted acknowledgement validation.
var errInvalidOptimizationAcknowledgement = errors.New("persisted optimization acknowledgement is invalid")

// OptimizationJobStateStore persists server-owned optimization jobs.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationJobStateStore interface {
	Save(context.Context, worker.OptimizationJob) error
	Load(context.Context, uuid.UUID) (worker.OptimizationJob, error)
}

// OptimizationJobEnqueuer publishes only server-created optimization IDs.
// Implements DESIGN-004 JobQueueManager and JobStatusTracker.
type OptimizationJobEnqueuer interface {
	Enqueue(context.Context, string) (string, error)
}

// OptimizationEntitlementChecker resolves authenticated feature access.
// Implements DESIGN-007 EntitlementManager and DESIGN-004 JobStatusTracker.
type OptimizationEntitlementChecker interface {
	CheckEntitlement(context.Context, uuid.UUID, entitlement.Feature) (entitlement.Decision, error)
}

// OptimizationIdempotencyRepository persists and completes one typed
// optimization acknowledgement for each user-scoped request key.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationIdempotencyRepository interface {
	repository.CheckoutIdempotencyRepository
	UpdateCheckoutIdempotencyResponse(context.Context, repository.CheckoutIdempotencyRecord) error
}

// OptimizationController owns authenticated submission and user-scoped polling.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RouteHandler.
type OptimizationController struct {
	jobs             OptimizationJobStateStore
	queue            OptimizationJobEnqueuer
	diets            repository.DailyDietRepository
	entitlements     OptimizationEntitlementChecker
	idempotency      OptimizationIdempotencyRepository
	admission        worker.OptimizationAdmissionGate
	telemetry        *observability.OptimizationTelemetry
	admissionCleanup chan struct{}
}

// Implements DESIGN-004 JobStatusTracker compile-time route controller contract.
var _ Controller = (*OptimizationController)(nil)

// NewOptimizationController constructs the asynchronous optimization API boundary.
// Implements DESIGN-004 JobStatusTracker.
func NewOptimizationController(jobs OptimizationJobStateStore, enqueuer OptimizationJobEnqueuer, diets repository.DailyDietRepository, entitlements OptimizationEntitlementChecker, idempotency OptimizationIdempotencyRepository, admission worker.OptimizationAdmissionGate) *OptimizationController {
	return &OptimizationController{jobs: jobs, queue: enqueuer, diets: diets, entitlements: entitlements, idempotency: idempotency, admission: admission, admissionCleanup: make(chan struct{}, optimizationAdmissionCleanupLimit)}
}

// WithTelemetry attaches bounded submission metrics and logs to the API
// submission boundary; request bodies and authenticated identifiers are never
// forwarded to the telemetry adapter.
// Implements DESIGN-004 JobStatusTracker and DESIGN-014 MetricsCollector.
func (c *OptimizationController) WithTelemetry(telemetry *observability.OptimizationTelemetry) *OptimizationController {
	if c != nil {
		c.telemetry = telemetry
	}
	return c
}

// Routes returns the authenticated optimization submission and polling routes.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodPost, Path: optimizationJobsRoute, RequiresAuth: true, RequiresCSRF: true, Validate: ValidateJSON(validateOptimizationSubmissionBody), Handler: c.Submit},
		{Method: fiber.MethodGet, Path: optimizationJobsRoute + "/:jobId", RequiresAuth: true, Validate: ValidatePath("jobId", validateUUIDValue), Handler: c.GetJob},
	}
}

// Submit accepts a validated request, reloads the saved diet under session ownership,
// stores a queued job, and enqueues only its server-created ID. It never invokes a solver.
// Implements DESIGN-004 JobStatusTracker and SW-REQ-006.
func (c *OptimizationController) Submit(ctx *fiber.Ctx) (resultErr error) {
	outcome := observability.OptimizationSubmissionError
	defer func() {
		if resultErr != nil {
			outcome = optimizationSubmissionFailure(resultErr)
		}
		if c != nil && c.telemetry != nil {
			c.telemetry.Submission(ctx.UserContext(), outcome)
		}
	}()
	user, ok := authenticatedUser(ctx)
	if !ok {
		outcome = observability.OptimizationSubmissionRejected
		return unauthorizedError()
	}
	req, err := parseOptimizationSubmission(ctx)
	if err != nil {
		outcome = observability.OptimizationSubmissionRejected
		return err
	}
	key, err := validateOptimizationIdempotencyKey(ctx.Get("Idempotency-Key"))
	if err != nil {
		outcome = observability.OptimizationSubmissionRejected
		return err
	}
	bodyHash, err := optimizationRequestHash(req)
	if err != nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}

	if c.idempotency == nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}
	existing, acknowledgement, found, err := c.lookupOptimizationIdempotency(ctx.UserContext(), user.UserID, key, bodyHash)
	if err != nil {
		resultErr = optimizationError(err)
		outcome = optimizationSubmissionFailure(resultErr)
		return resultErr
	}
	if found && acknowledgement.PublicationState == optimizationPublicationPublished {
		resultErr = writeOptimizationAcknowledgement(ctx, existing, acknowledgement)
		if resultErr == nil {
			outcome = observability.OptimizationSubmissionReplayed
		}
		return resultErr
	}
	if found {
		outcome, resultErr = c.repairOptimizationPublication(ctx, user.UserID, req, key, bodyHash, existing, acknowledgement)
		return resultErr
	}

	if c.entitlements == nil || c.admission == nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}
	decision, err := c.entitlements.CheckEntitlement(ctx.UserContext(), user.UserID, entitlement.FeatureDailyDietAlternative)
	if err != nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}
	if !decision.Allowed {
		outcome = observability.OptimizationSubmissionRejected
		return optimizationEntitlementDenied()
	}

	if c.jobs == nil || c.queue == nil || c.diets == nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}
	if _, err := c.diets.Get(ctx.UserContext(), user.UserID, req.DailyDietID); err != nil {
		resultErr = optimizationError(err)
		outcome = optimizationSubmissionFailure(resultErr)
		return resultErr
	}
	jobID := uuid.New()
	acquiredJobID := jobID
	admissionDecision, err := c.admission.Acquire(ctx.UserContext(), worker.OptimizationAdmissionRequest{
		UserID: user.UserID, JobID: jobID, IdempotencyKey: key, BodyHash: bodyHash, CountRate: true,
	})
	if err != nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationError(err)
	}
	ownedSlot := admissionDecision.Status == worker.OptimizationAdmissionAcquired
	if ownedSlot {
		defer func() {
			if ownedSlot {
				c.releaseAdmission(ctx.UserContext(), user.UserID, acquiredJobID)
			}
		}()
	}
	switch admissionDecision.Status {
	case worker.OptimizationAdmissionAcquired:
	case worker.OptimizationAdmissionReplay:
		jobID = admissionDecision.JobID
	case worker.OptimizationAdmissionConflict:
		outcome = observability.OptimizationSubmissionRejected
		return optimizationIdempotencyConflict()
	case worker.OptimizationAdmissionActive:
		outcome = observability.OptimizationSubmissionRejected
		return retryableTooManyRequests(ctx, admissionDecision.RetryAfter, "rate_limit", "optimization_in_progress", "An optimization is already in progress.")
	case worker.OptimizationAdmissionRateLimited:
		outcome = observability.OptimizationSubmissionRejected
		return retryableTooManyRequests(ctx, admissionDecision.RetryAfter, "rate_limit", "optimization_rate_limited", "Too many optimization jobs were requested. Please try again later.")
	default:
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}
	job := worker.OptimizationJob{
		JobID:            jobID,
		UserID:           user.UserID,
		DailyDietID:      req.DailyDietID,
		TolerancePercent: req.TolerancePercent,
		ExcludedMealIDs:  append([]uuid.UUID(nil), req.ExcludedMealIDs...),
		Status:           worker.OptimizationJobQueued,
		CreatedAt:        time.Now().UTC(),
	}
	acknowledgement = optimizationPersistedAcknowledgement{
		JobID: jobID, Status: worker.OptimizationJobQueued,
		PollURL: optimizationPollPath + jobID.String(), PublicationState: optimizationPublicationPending,
	}
	payload, err := json.Marshal(acknowledgement)
	if err != nil {
		outcome = observability.OptimizationSubmissionDependencyError
		return optimizationDependencyError()
	}
	record := repository.CheckoutIdempotencyRecord{
		UserID: user.UserID, Method: optimizationMethod, Route: optimizationJobsRoute,
		Key: key, BodyHash: bodyHash, StatusCode: fiber.StatusAccepted, ResponseBody: payload,
	}
	if err := c.idempotency.StoreCheckoutIdempotency(ctx.UserContext(), record); err != nil {
		if !repository.IsKind(err, repository.ErrorKindConflict) {
			resultErr = optimizationError(err)
			outcome = optimizationSubmissionFailure(resultErr)
			return resultErr
		}
		original, originalAcknowledgement, found, lookupErr := c.lookupOptimizationIdempotency(ctx.UserContext(), user.UserID, key, bodyHash)
		if lookupErr != nil {
			resultErr = optimizationError(lookupErr)
			outcome = optimizationSubmissionFailure(resultErr)
			return resultErr
		}
		if !found {
			outcome = observability.OptimizationSubmissionRejected
			return optimizationError(err)
		}
		if ownedSlot {
			c.releaseAdmission(ctx.UserContext(), user.UserID, acquiredJobID)
			ownedSlot = false
		}
		if originalAcknowledgement.PublicationState == optimizationPublicationPublished {
			resultErr = writeOptimizationAcknowledgement(ctx, original, originalAcknowledgement)
			if resultErr == nil {
				outcome = observability.OptimizationSubmissionReplayed
			}
			return resultErr
		}
		outcome, resultErr = c.repairOptimizationPublication(ctx, user.UserID, req, key, bodyHash, original, originalAcknowledgement)
		return resultErr
	}
	// The durable claim is made before publication. Replays repair a crash or
	// queue outage by repeating this idempotent save/enqueue pair for the same
	// server-created job ID.
	if err := c.jobs.Save(ctx.UserContext(), job); err != nil {
		resultErr = optimizationError(err)
		outcome = optimizationSubmissionFailure(resultErr)
		return resultErr
	}
	if _, err := c.queue.Enqueue(ctx.UserContext(), jobID.String()); err != nil {
		outcome = observability.OptimizationSubmissionQueueError
		return optimizationQueueError()
	}
	// Queue publication transfers the active slot to the worker. A later
	// acknowledgement-update failure remains repairable and must not admit a
	// second job while this published job is queued or processing.
	ownedSlot = false
	record, acknowledgement, err = c.completeOptimizationPublication(ctx.UserContext(), record, acknowledgement)
	if err != nil {
		resultErr = optimizationError(err)
		outcome = optimizationSubmissionFailure(resultErr)
		return resultErr
	}
	resultErr = writeOptimizationAcknowledgement(ctx, record, acknowledgement)
	if resultErr == nil {
		outcome = observability.OptimizationSubmissionAccepted
	}
	return resultErr
}

// repairOptimizationPublication revalidates an unpublished durable claim and
// reacquires capacity without recounting an exact retry.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) repairOptimizationPublication(ctx *fiber.Ctx, userID uuid.UUID, req optimizationSubmissionRequest, key, bodyHash string, record repository.CheckoutIdempotencyRecord, acknowledgement optimizationPersistedAcknowledgement) (outcome observability.OptimizationSubmissionOutcome, resultErr error) {
	outcome = observability.OptimizationSubmissionError
	if acknowledgement.PublicationState != optimizationPublicationPending || c.entitlements == nil || c.admission == nil || c.jobs == nil || c.queue == nil || c.diets == nil {
		return observability.OptimizationSubmissionDependencyError, optimizationDependencyError()
	}
	decision, err := c.entitlements.CheckEntitlement(ctx.UserContext(), userID, entitlement.FeatureDailyDietAlternative)
	if err != nil {
		return observability.OptimizationSubmissionDependencyError, optimizationDependencyError()
	}
	if !decision.Allowed {
		return observability.OptimizationSubmissionRejected, optimizationEntitlementDenied()
	}
	if _, err := c.diets.Get(ctx.UserContext(), userID, req.DailyDietID); err != nil {
		resultErr = optimizationError(err)
		return optimizationSubmissionFailure(resultErr), resultErr
	}
	jobID := acknowledgement.JobID
	admissionDecision, err := c.admission.Acquire(ctx.UserContext(), worker.OptimizationAdmissionRequest{
		UserID: userID, JobID: jobID, IdempotencyKey: key, BodyHash: bodyHash, CountRate: false,
	})
	if err != nil {
		resultErr = optimizationError(err)
		return optimizationSubmissionFailure(resultErr), resultErr
	}
	ownedSlot := admissionDecision.Status == worker.OptimizationAdmissionAcquired
	if ownedSlot {
		defer func() {
			if ownedSlot {
				c.releaseAdmission(ctx.UserContext(), userID, jobID)
			}
		}()
	}
	switch admissionDecision.Status {
	case worker.OptimizationAdmissionAcquired, worker.OptimizationAdmissionReplay:
	case worker.OptimizationAdmissionConflict:
		return observability.OptimizationSubmissionRejected, optimizationIdempotencyConflict()
	case worker.OptimizationAdmissionActive:
		return observability.OptimizationSubmissionRejected, retryableTooManyRequests(ctx, admissionDecision.RetryAfter, "rate_limit", "optimization_in_progress", "An optimization is already in progress.")
	case worker.OptimizationAdmissionRateLimited:
		return observability.OptimizationSubmissionRejected, retryableTooManyRequests(ctx, admissionDecision.RetryAfter, "rate_limit", "optimization_rate_limited", "Too many optimization jobs were requested. Please try again later.")
	default:
		return observability.OptimizationSubmissionDependencyError, optimizationDependencyError()
	}
	if ownedSlot {
		// A pending acknowledgement read before admission can become stale while
		// another controller publishes and its worker releases this same job. Re-read
		// the durable authority before a fresh reservation performs repair effects.
		// Implements DESIGN-004 JobStatusTracker pending-repair linearization.
		currentRecord, currentAcknowledgement, found, err := c.lookupOptimizationIdempotency(ctx.UserContext(), userID, key, bodyHash)
		if err != nil {
			resultErr = optimizationError(err)
			return optimizationSubmissionFailure(resultErr), resultErr
		}
		if !found || currentAcknowledgement.JobID != jobID {
			return observability.OptimizationSubmissionDependencyError, optimizationDependencyError()
		}
		record, acknowledgement = currentRecord, currentAcknowledgement
		if acknowledgement.PublicationState == optimizationPublicationPublished {
			resultErr = writeOptimizationAcknowledgement(ctx, record, acknowledgement)
			if resultErr != nil {
				return observability.OptimizationSubmissionError, resultErr
			}
			return observability.OptimizationSubmissionReplayed, nil
		}
	}
	job := worker.OptimizationJob{
		JobID: jobID, UserID: userID, DailyDietID: req.DailyDietID,
		TolerancePercent: req.TolerancePercent, ExcludedMealIDs: append([]uuid.UUID(nil), req.ExcludedMealIDs...),
		Status: worker.OptimizationJobQueued, CreatedAt: time.Now().UTC(),
	}
	if err := c.jobs.Save(ctx.UserContext(), job); err != nil {
		resultErr = optimizationError(err)
		return optimizationSubmissionFailure(resultErr), resultErr
	}
	if _, err := c.queue.Enqueue(ctx.UserContext(), jobID.String()); err != nil {
		return observability.OptimizationSubmissionQueueError, optimizationQueueError()
	}
	if ownedSlot {
		// An idempotent queue marker does not prove that a worker still owns this
		// newly acquired reservation. Redis job transitions preserve terminal state,
		// so only a queued/processing job can accept the handoff. If it turns terminal
		// after this read, the worker's owner-scoped release removes the same slot.
		// Implements DESIGN-004 JobStatusTracker worker-delivery ownership.
		currentJob, err := c.jobs.Load(ctx.UserContext(), jobID)
		if err != nil {
			resultErr = optimizationError(err)
			return optimizationSubmissionFailure(resultErr), resultErr
		}
		switch currentJob.Status {
		case worker.OptimizationJobQueued, worker.OptimizationJobProcessing:
			ownedSlot = false
		case worker.OptimizationJobCompleted, worker.OptimizationJobFailed, worker.OptimizationJobCancelled:
		default:
			return observability.OptimizationSubmissionDependencyError, optimizationDependencyError()
		}
	}
	record, acknowledgement, err = c.completeOptimizationPublication(ctx.UserContext(), record, acknowledgement)
	if err != nil {
		resultErr = optimizationError(err)
		return optimizationSubmissionFailure(resultErr), resultErr
	}
	resultErr = writeOptimizationAcknowledgement(ctx, record, acknowledgement)
	if resultErr != nil {
		return observability.OptimizationSubmissionError, resultErr
	}
	return observability.OptimizationSubmissionReplayed, nil
}

// optimizationSubmissionFailure maps the final public response to one bounded outcome.
// Implements DESIGN-014 MetricsCollector.
func optimizationSubmissionFailure(err error) observability.OptimizationSubmissionOutcome {
	classified := ClassifyServerError(err)
	switch {
	case classified.Code == "queue_unavailable":
		return observability.OptimizationSubmissionQueueError
	case classified.Category == "dependency" || classified.Category == "timeout":
		return observability.OptimizationSubmissionDependencyError
	case classified.HTTPStatus >= fiber.StatusBadRequest && classified.HTTPStatus < fiber.StatusInternalServerError:
		return observability.OptimizationSubmissionRejected
	default:
		return observability.OptimizationSubmissionError
	}
}

// releaseAdmission bounds best-effort controller cleanup and reports only a fixed failure event.
// A release failure never replaces the submission's authoritative HTTP result.
// Implements DESIGN-004 JobStatusTracker and DESIGN-014 MetricsCollector.
func (c *OptimizationController) releaseAdmission(ctx context.Context, userID, jobID uuid.UUID) {
	if c == nil || c.admission == nil {
		return
	}
	failed := false
	select {
	case c.admissionCleanup <- struct{}{}:
		detachedCtx := context.WithoutCancel(ctx)
		cleanupCtx, cancel := context.WithTimeout(detachedCtx, optimizationAdmissionCleanupTimeout)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			defer func() { <-c.admissionCleanup }()
			done <- c.admission.Release(cleanupCtx, userID, jobID)
		}()
		select {
		case err := <-done:
			failed = err != nil
		case <-cleanupCtx.Done():
			failed = true
		}
	default:
		// A noncooperative release may retain the sole cleanup lane, but cannot
		// create unbounded goroutines or replace the authoritative response.
		failed = true
	}
	if failed && c.telemetry != nil {
		c.telemetry.AdmissionCleanupFailed(ctx)
	}
}

// GetJob returns a job only when its owner matches the authenticated session.
// Implements DESIGN-004 JobStatusTracker and DESIGN-006 JWTManager.
func (c *OptimizationController) GetJob(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	jobID, err := uuid.Parse(ctx.Params("jobId"))
	if err != nil || jobID == uuid.Nil {
		return optimizationValidationError()
	}
	if c.jobs == nil {
		return optimizationDependencyError()
	}
	job, err := c.jobs.Load(ctx.UserContext(), jobID)
	if err != nil {
		var expired worker.OptimizationJobExpiredError
		if errors.As(err, &expired) {
			if expired.UserID != uuid.Nil && expired.UserID != user.UserID {
				return optimizationNotFoundError()
			}
			return optimizationExpiredError()
		}
		mapped := optimizationError(err)
		var appErr AppError
		if errors.As(mapped, &appErr) {
			return mapped
		}
		return optimizationDependencyError()
	}
	if job.UserID != user.UserID {
		return optimizationNotFoundError()
	}
	if (job.Status == worker.OptimizationJobFailed && (job.Failure == nil || !job.Failure.Valid())) || (job.Status != worker.OptimizationJobFailed && job.Failure != nil) {
		return optimizationDependencyError()
	}
	if err := validateOptimizationJobAlternatives(job); err != nil {
		return optimizationDependencyError()
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: optimizationJobData(job)})
}

// validateOptimizationJobAlternatives prevents alternate stores from
// bypassing the authoritative persistence validator before HTTP projection.
// Implements DESIGN-004 JobStatusTracker authoritative result projection.
func validateOptimizationJobAlternatives(job worker.OptimizationJob) error {
	switch job.Status {
	case worker.OptimizationJobCompleted:
		if len(job.Alternatives) == 0 || len(job.Alternatives) > optimization.MaxAlternativeCount {
			return errors.New("completed optimization job alternatives are invalid")
		}
	case worker.OptimizationJobFailed:
		if len(job.Alternatives) > optimization.MaxAlternativeCount {
			return errors.New("failed optimization job alternatives are invalid")
		}
	default:
		if len(job.Alternatives) != 0 {
			return errors.New("non-result optimization job alternatives are invalid")
		}
	}
	for _, alternative := range job.Alternatives {
		if err := optimization.ValidateDietAlternative(alternative); err != nil {
			return err
		}
	}
	return nil
}

// validateOptimizationSubmissionBody validates the JSON shape before any service dispatch.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validateOptimizationSubmissionBody(body map[string]any) error {
	if len(body) != 3 {
		return errors.New("optimization request contains unsupported fields")
	}
	dailyDietID, ok := body["dailyDietId"].(string)
	parsedDailyDietID, parseErr := uuid.Parse(dailyDietID)
	if !ok || parseErr != nil || parsedDailyDietID == uuid.Nil {
		return errors.New("daily diet id is invalid")
	}
	tolerance, ok := body["tolerancePercent"].(float64)
	if !ok || !finiteOptimizationNumber(tolerance) || tolerance < 0 || tolerance > 100 || math.Abs(tolerance*10-math.Round(tolerance*10)) > 1e-9 {
		return errors.New("optimization tolerance is invalid")
	}
	excluded, ok := body["excludedMealIds"].([]any)
	if !ok || len(excluded) > maximumExcludedMeals {
		return errors.New("excluded meal IDs are invalid")
	}
	seen := make(map[uuid.UUID]struct{}, len(excluded))
	for _, raw := range excluded {
		value, ok := raw.(string)
		id, parseErr := uuid.Parse(value)
		if !ok || parseErr != nil || id == uuid.Nil {
			return errors.New("excluded meal IDs are invalid")
		}
		if _, exists := seen[id]; exists {
			return errors.New("excluded meal IDs must be unique")
		}
		seen[id] = struct{}{}
	}
	return nil
}

// optimizationSubmissionRequest is the server-normalized optimization input.
// Implements DESIGN-004 JobStatusTracker.
type optimizationSubmissionRequest struct {
	DailyDietID      uuid.UUID   `json:"dailyDietId"`
	TolerancePercent float64     `json:"tolerancePercent"`
	ExcludedMealIDs  []uuid.UUID `json:"excludedMealIds"`
}

// parseOptimizationSubmission parses the validated optimization request.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func parseOptimizationSubmission(ctx *fiber.Ctx) (optimizationSubmissionRequest, error) {
	var raw struct {
		DailyDietID      string   `json:"dailyDietId"`
		TolerancePercent float64  `json:"tolerancePercent"`
		ExcludedMealIDs  []string `json:"excludedMealIds"`
	}
	if err := ctx.BodyParser(&raw); err != nil {
		return optimizationSubmissionRequest{}, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	dailyDietID, err := uuid.Parse(raw.DailyDietID)
	if err != nil || dailyDietID == uuid.Nil {
		return optimizationSubmissionRequest{}, optimizationValidationError()
	}
	if !finiteOptimizationNumber(raw.TolerancePercent) || raw.TolerancePercent < 0 || raw.TolerancePercent > 100 || math.Abs(raw.TolerancePercent*10-math.Round(raw.TolerancePercent*10)) > 1e-9 || len(raw.ExcludedMealIDs) > maximumExcludedMeals {
		return optimizationSubmissionRequest{}, optimizationValidationError()
	}
	excluded := make([]uuid.UUID, 0, len(raw.ExcludedMealIDs))
	seen := make(map[uuid.UUID]struct{}, len(raw.ExcludedMealIDs))
	for _, value := range raw.ExcludedMealIDs {
		id, parseErr := uuid.Parse(value)
		if parseErr != nil || id == uuid.Nil {
			return optimizationSubmissionRequest{}, optimizationValidationError()
		}
		if _, exists := seen[id]; exists {
			return optimizationSubmissionRequest{}, optimizationValidationError()
		}
		seen[id] = struct{}{}
		excluded = append(excluded, id)
	}
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].String() < excluded[j].String() })
	tolerance := math.Round(raw.TolerancePercent*10) / 10
	if tolerance == 0 {
		tolerance = 0 // Normalize negative zero before hashing and persistence.
	}
	return optimizationSubmissionRequest{DailyDietID: dailyDietID, TolerancePercent: tolerance, ExcludedMealIDs: excluded}, nil
}

// optimizationRequestHash creates the digest of canonical parsed values. UUID
// exclusions are an explicitly sorted, duplicate-free set.
// Implements DESIGN-004 JobStatusTracker.
func optimizationRequestHash(request optimizationSubmissionRequest) (string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// lookupOptimizationIdempotency returns an exact replay or a body conflict.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) lookupOptimizationIdempotency(ctx context.Context, userID uuid.UUID, key, bodyHash string) (repository.CheckoutIdempotencyRecord, optimizationPersistedAcknowledgement, bool, error) {
	record, err := c.idempotency.GetCheckoutIdempotency(ctx, userID, optimizationMethod, optimizationJobsRoute, key)
	if err != nil {
		if repository.IsKind(err, repository.ErrorKindNotFound) {
			return repository.CheckoutIdempotencyRecord{}, optimizationPersistedAcknowledgement{}, false, nil
		}
		return repository.CheckoutIdempotencyRecord{}, optimizationPersistedAcknowledgement{}, false, err
	}
	if record.BodyHash != bodyHash {
		return repository.CheckoutIdempotencyRecord{}, optimizationPersistedAcknowledgement{}, false, repository.NewError(repository.ErrorKindConflict, "idempotency key reused with different body", nil)
	}
	acknowledgement, err := decodeOptimizationAcknowledgement(record)
	if err != nil {
		return repository.CheckoutIdempotencyRecord{}, optimizationPersistedAcknowledgement{}, false, err
	}
	return record, acknowledgement, true, nil
}

// optimizationPublicationState distinguishes side-effect-free replay from
// repair of a durable claim whose queue publication was not confirmed.
// Implements DESIGN-004 JobStatusTracker.
type optimizationPublicationState string

// Implements DESIGN-004 JobStatusTracker.
const (
	optimizationPublicationPending   optimizationPublicationState = "pending"
	optimizationPublicationPublished optimizationPublicationState = "published"
)

// optimizationPersistedAcknowledgement is the sole durable acknowledgement
// shape. PublicationState is internal and is never projected into API data.
// Implements DESIGN-004 JobStatusTracker.
type optimizationPersistedAcknowledgement struct {
	JobID            uuid.UUID                    `json:"jobId"`
	Status           worker.OptimizationJobStatus `json:"status"`
	PollURL          string                       `json:"pollUrl"`
	PublicationState optimizationPublicationState `json:"publicationState"`
}

// decodeOptimizationAcknowledgement rejects malformed or aliased persisted responses.
// Implements DESIGN-004 JobStatusTracker.
func decodeOptimizationAcknowledgement(record repository.CheckoutIdempotencyRecord) (optimizationPersistedAcknowledgement, error) {
	var acknowledgement optimizationPersistedAcknowledgement
	decoder := json.NewDecoder(bytes.NewReader(record.ResponseBody))
	decoder.DisallowUnknownFields()
	if record.StatusCode != fiber.StatusAccepted || decoder.Decode(&acknowledgement) != nil || decoder.Decode(&struct{}{}) != io.EOF || acknowledgement.JobID == uuid.Nil || acknowledgement.Status != worker.OptimizationJobQueued || acknowledgement.PollURL != optimizationPollPath+acknowledgement.JobID.String() || (acknowledgement.PublicationState != optimizationPublicationPending && acknowledgement.PublicationState != optimizationPublicationPublished) {
		return optimizationPersistedAcknowledgement{}, errInvalidOptimizationAcknowledgement
	}
	return acknowledgement, nil
}

// completeOptimizationPublication persists confirmation after idempotent queue publication.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) completeOptimizationPublication(ctx context.Context, record repository.CheckoutIdempotencyRecord, acknowledgement optimizationPersistedAcknowledgement) (repository.CheckoutIdempotencyRecord, optimizationPersistedAcknowledgement, error) {
	acknowledgement.PublicationState = optimizationPublicationPublished
	payload, err := json.Marshal(acknowledgement)
	if err != nil {
		return record, acknowledgement, err
	}
	record.ResponseBody = payload
	if err := c.idempotency.UpdateCheckoutIdempotencyResponse(ctx, record); err != nil {
		return record, acknowledgement, err
	}
	return record, acknowledgement, nil
}

// writeOptimizationAcknowledgement projects only the exact public 202 fields.
// Implements DESIGN-004 JobStatusTracker.
func writeOptimizationAcknowledgement(ctx *fiber.Ctx, record repository.CheckoutIdempotencyRecord, acknowledgement optimizationPersistedAcknowledgement) error {
	ctx.Set(fiber.HeaderLocation, acknowledgement.PollURL)
	return ctx.Status(record.StatusCode).JSON(Envelope{Status: "accepted", RequestID: requestID(ctx), Data: map[string]any{
		"jobId": acknowledgement.JobID.String(), "status": string(acknowledgement.Status), "pollUrl": acknowledgement.PollURL,
	}})
}

// optimizationJobData builds the user-scoped polling response.
// Implements DESIGN-004 JobStatusTracker.
func optimizationJobData(job worker.OptimizationJob) map[string]any {
	data := map[string]any{
		"jobId": job.JobID.String(), "dailyDietId": job.DailyDietID.String(), "status": string(job.Status),
		"pollUrl": optimizationPollPath + job.JobID.String(), "createdAt": job.CreatedAt,
	}
	if job.StartedAt != nil {
		data["startedAt"] = *job.StartedAt
	}
	if job.FinishedAt != nil {
		data["finishedAt"] = *job.FinishedAt
	}
	if len(job.Alternatives) > 0 {
		alternatives := make([]map[string]any, 0, len(job.Alternatives))
		for _, alternative := range job.Alternatives {
			meals := make([]map[string]any, 0, len(alternative.Meals))
			for _, meal := range alternative.Meals {
				meals = append(meals, map[string]any{"mealId": meal.MealID.String(), "quantity": meal.Quantity, "unit": meal.Unit, "position": meal.Position})
			}
			alternatives = append(alternatives, map[string]any{
				"meals": meals,
				"macros": map[string]any{
					"protein": alternative.Macros.Protein, "carbohydrates": alternative.Macros.Carbohydrates,
					"fat": alternative.Macros.Fat, "calories": alternative.Calories,
				},
				"similarityScore": alternative.SimilarityScore,
			})
		}
		data["alternatives"] = alternatives
	}
	if job.Failure != nil {
		data["failure"] = map[string]any{"code": job.Failure.Code.String(), "message": job.Failure.Message}
	}
	return data
}

// validateOptimizationIdempotencyKey validates the replay key boundary.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validateOptimizationIdempotencyKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < minimumIdempotencyKey || len(value) > maximumIdempotencyKey {
		return "", AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	}
	// Fiber header strings may alias request buffers that are reused after the
	// handler returns; durable identity must own its bytes.
	return strings.Clone(value), nil
}

// validateUUIDValue validates a non-nil UUID path value.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validateUUIDValue(value string) error {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return errors.New("uuid is invalid")
	}
	return nil
}

// finiteOptimizationNumber rejects NaN and infinite optimization values.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func finiteOptimizationNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// optimizationValidationError returns the stable request validation error.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func optimizationValidationError() AppError {
	return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
}

// optimizationEntitlementDenied returns the stable feature-gate error.
// Implements DESIGN-004 JobStatusTracker and DESIGN-007 EntitlementManager.
func optimizationEntitlementDenied() AppError {
	return AppError{HTTPStatus: fiber.StatusForbidden, Category: "entitlement", Code: "entitlement_denied", Message: "an active trial or paid subscription is required for optimization"}
}

// optimizationIdempotencyConflict returns the stable changed-body replay error.
// Implements DESIGN-004 JobStatusTracker.
func optimizationIdempotencyConflict() AppError {
	return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "idempotency_key_conflict", Message: "Idempotency-Key was already used with a different request body"}
}

// optimizationDependencyError returns the generic unavailable response.
// Implements DESIGN-004 JobStatusTracker.
func optimizationDependencyError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "optimization_unavailable", Message: "optimization service is unavailable", Retryable: true}
}

// optimizationQueueError returns the queue outage response.
// Implements DESIGN-004 JobStatusTracker and JobQueueManager.
func optimizationQueueError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "queue_unavailable", Message: "optimization queue is unavailable", Retryable: true}
}

// optimizationNotFoundError hides unknown or cross-user jobs.
// Implements DESIGN-004 JobStatusTracker.
func optimizationNotFoundError() AppError {
	return AppError{HTTPStatus: fiber.StatusNotFound, Category: "validation", Code: "not_found", Message: "optimization job not found"}
}

// optimizationExpiredError reports an expired owned result.
// Implements DESIGN-004 JobStatusTracker.
func optimizationExpiredError() AppError {
	return AppError{HTTPStatus: fiber.StatusGone, Category: "validation", Code: "result_expired", Message: "optimization result has expired"}
}

// optimizationError maps internal job failures to safe HTTP errors.
// Implements DESIGN-004 JobStatusTracker.
func optimizationError(err error) error {
	var expired worker.OptimizationJobExpiredError
	switch {
	case errors.As(err, &expired):
		return optimizationExpiredError()
	case errors.Is(err, worker.ErrOptimizationJobNotFound), repository.IsKind(err, repository.ErrorKindNotFound):
		return optimizationNotFoundError()
	case errors.Is(err, queue.ErrQueueUnavailable):
		return optimizationQueueError()
	case errors.Is(err, errInvalidOptimizationAcknowledgement):
		return optimizationDependencyError()
	case repository.IsKind(err, repository.ErrorKindValidation):
		return optimizationValidationError()
	case repository.IsKind(err, repository.ErrorKindConflict):
		return optimizationIdempotencyConflict()
	default:
		return err
	}
}

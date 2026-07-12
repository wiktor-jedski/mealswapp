package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// Implements DESIGN-004 JobStatusTracker submission and polling boundary.
const (
	optimizationJobsRoute = "/optimization/jobs"
	optimizationPollPath  = "/api/v1/optimization/jobs/"
	optimizationMethod    = "POST"
	minimumIdempotencyKey = 8
	maximumIdempotencyKey = 255
	maximumExcludedMeals  = 100
)

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

// OptimizationController owns authenticated submission and user-scoped polling.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RouteHandler.
type OptimizationController struct {
	jobs         OptimizationJobStateStore
	queue        OptimizationJobEnqueuer
	diets        repository.DailyDietRepository
	entitlements OptimizationEntitlementChecker
	idempotency  repository.CheckoutIdempotencyRepository
	admission    worker.OptimizationAdmissionGate
	telemetry    *observability.OptimizationTelemetry
	mu           sync.Mutex
}

// Implements DESIGN-004 JobStatusTracker compile-time route controller contract.
var _ Controller = (*OptimizationController)(nil)

// NewOptimizationController constructs the asynchronous optimization API boundary.
// Implements DESIGN-004 JobStatusTracker.
func NewOptimizationController(jobs OptimizationJobStateStore, enqueuer OptimizationJobEnqueuer, diets repository.DailyDietRepository, entitlements OptimizationEntitlementChecker, idempotency repository.CheckoutIdempotencyRepository, admission worker.OptimizationAdmissionGate) *OptimizationController {
	return &OptimizationController{jobs: jobs, queue: enqueuer, diets: diets, entitlements: entitlements, idempotency: idempotency, admission: admission}
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
		{Method: fiber.MethodGet, Path: optimizationJobsRoute + "/:jobId", RequiresAuth: true, Validate: ValidatePath("jobId", validateOptimizationJobID), Handler: c.GetJob},
	}
}

// Submit accepts a validated request, reloads the saved diet under session ownership,
// stores a queued job, and enqueues only its server-created ID. It never invokes a solver.
// Implements DESIGN-004 JobStatusTracker and SW-REQ-006.
func (c *OptimizationController) Submit(ctx *fiber.Ctx) error {
	outcome := "error"
	defer func() {
		if c != nil && c.telemetry != nil {
			c.telemetry.Submission(ctx.UserContext(), outcome)
		}
	}()
	user, ok := authenticatedUser(ctx)
	if !ok {
		outcome = "rejected"
		return unauthorizedError()
	}
	req, err := parseOptimizationSubmission(ctx)
	if err != nil {
		outcome = "rejected"
		return err
	}
	key, err := validateOptimizationIdempotencyKey(ctx.Get("Idempotency-Key"))
	if err != nil {
		outcome = "rejected"
		return err
	}
	bodyHash := optimizationRequestHash(ctx.Body())

	// Serialize same-controller retries so a lost response cannot create two jobs.
	// The durable idempotency row remains the cross-process authority.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entitlements == nil || c.idempotency == nil || c.admission == nil {
		outcome = "dependency_error"
		return optimizationDependencyError()
	}
	decision, err := c.entitlements.CheckEntitlement(ctx.UserContext(), user.UserID, entitlement.FeatureDailyDietAlternative)
	if err != nil {
		outcome = "dependency_error"
		return optimizationDependencyError()
	}
	if !decision.Allowed {
		outcome = "rejected"
		return optimizationEntitlementDenied()
	}

	if c.jobs == nil || c.queue == nil || c.diets == nil {
		outcome = "dependency_error"
		return optimizationDependencyError()
	}
	if _, err := c.diets.Get(ctx.UserContext(), user.UserID, req.DailyDietID); err != nil {
		return optimizationError(err)
	}
	if existing, found, lookupErr := c.lookupOptimizationIdempotency(ctx.UserContext(), user.UserID, key, bodyHash); lookupErr != nil {
		return optimizationError(lookupErr)
	} else if found {
		return c.repairOptimizationPublication(ctx, user.UserID, req, key, bodyHash, existing)
	}

	jobID := uuid.New()
	acquiredJobID := jobID
	admissionDecision, err := c.admission.Acquire(ctx.UserContext(), worker.OptimizationAdmissionRequest{
		UserID: user.UserID, JobID: jobID, IdempotencyKey: key, BodyHash: bodyHash, CountRate: true,
	})
	if err != nil {
		outcome = "dependency_error"
		return optimizationError(err)
	}
	ownedSlot := admissionDecision.Status == worker.OptimizationAdmissionAcquired
	if ownedSlot {
		defer func() {
			if ownedSlot {
				_ = c.admission.Release(context.WithoutCancel(ctx.UserContext()), user.UserID, acquiredJobID)
			}
		}()
	}
	switch admissionDecision.Status {
	case worker.OptimizationAdmissionAcquired:
	case worker.OptimizationAdmissionReplay:
		jobID = admissionDecision.JobID
	case worker.OptimizationAdmissionConflict:
		outcome = "rejected"
		return optimizationIdempotencyConflict()
	case worker.OptimizationAdmissionActive:
		outcome = "rejected"
		setOptimizationRetryAfter(ctx, admissionDecision.RetryAfter)
		return optimizationAdmissionError("optimization_in_progress", "An optimization is already in progress.")
	case worker.OptimizationAdmissionRateLimited:
		outcome = "rejected"
		setOptimizationRetryAfter(ctx, admissionDecision.RetryAfter)
		return optimizationAdmissionError("optimization_rate_limited", "Too many optimization jobs were requested. Please try again later.")
	default:
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
	ack := optimizationAcknowledgementData(jobID)
	payload, err := json.Marshal(ack)
	if err != nil {
		return optimizationDependencyError()
	}
	record := repository.CheckoutIdempotencyRecord{
		UserID: user.UserID, Method: optimizationMethod, Route: optimizationJobsRoute,
		Key: key, BodyHash: bodyHash, StatusCode: fiber.StatusAccepted, ResponseBody: payload,
	}
	if err := c.idempotency.StoreCheckoutIdempotency(ctx.UserContext(), record); err != nil {
		if !repository.IsKind(err, repository.ErrorKindConflict) {
			return optimizationError(err)
		}
		original, found, lookupErr := c.lookupOptimizationIdempotency(ctx.UserContext(), user.UserID, key, bodyHash)
		if lookupErr != nil {
			return optimizationError(lookupErr)
		}
		if !found {
			return optimizationError(err)
		}
		if ownedSlot {
			if err := c.admission.Release(ctx.UserContext(), user.UserID, acquiredJobID); err != nil {
				return optimizationError(err)
			}
			ownedSlot = false
		}
		outcome = "replayed"
		return c.repairOptimizationPublication(ctx, user.UserID, req, key, bodyHash, original)
	}
	// The durable claim is made before publication. Replays repair a crash or
	// queue outage by repeating this idempotent save/enqueue pair for the same
	// server-created job ID.
	if err := c.jobs.Save(ctx.UserContext(), job); err != nil {
		return optimizationError(err)
	}
	if _, err := c.queue.Enqueue(ctx.UserContext(), jobID.String()); err != nil {
		outcome = "queue_error"
		return optimizationQueueError()
	}
	ownedSlot = false
	outcome = "accepted"
	return c.writeAcknowledgement(ctx, record, false)
}

// repairOptimizationPublication reacquires capacity without recounting an exact retry.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) repairOptimizationPublication(ctx *fiber.Ctx, userID uuid.UUID, req optimizationSubmissionRequest, key, bodyHash string, record repository.CheckoutIdempotencyRecord) error {
	jobID, err := optimizationJobIDFromAcknowledgement(record)
	if err != nil {
		return optimizationDependencyError()
	}
	decision, err := c.admission.Acquire(ctx.UserContext(), worker.OptimizationAdmissionRequest{
		UserID: userID, JobID: jobID, IdempotencyKey: key, BodyHash: bodyHash, CountRate: false,
	})
	if err != nil {
		return optimizationError(err)
	}
	ownedSlot := decision.Status == worker.OptimizationAdmissionAcquired
	if ownedSlot {
		defer func() {
			if ownedSlot {
				_ = c.admission.Release(context.WithoutCancel(ctx.UserContext()), userID, jobID)
			}
		}()
	}
	switch decision.Status {
	case worker.OptimizationAdmissionAcquired, worker.OptimizationAdmissionReplay:
	case worker.OptimizationAdmissionConflict:
		return optimizationIdempotencyConflict()
	case worker.OptimizationAdmissionActive:
		setOptimizationRetryAfter(ctx, decision.RetryAfter)
		return optimizationAdmissionError("optimization_in_progress", "An optimization is already in progress.")
	default:
		return optimizationDependencyError()
	}
	job := worker.OptimizationJob{
		JobID: jobID, UserID: userID, DailyDietID: req.DailyDietID,
		TolerancePercent: req.TolerancePercent, ExcludedMealIDs: append([]uuid.UUID(nil), req.ExcludedMealIDs...),
		Status: worker.OptimizationJobQueued, CreatedAt: time.Now().UTC(),
	}
	if err := c.jobs.Save(ctx.UserContext(), job); err != nil {
		if errors.Is(err, worker.ErrOptimizationJobNotFound) {
			return c.writeAcknowledgement(ctx, record, true)
		}
		return optimizationError(err)
	}
	if _, err := c.queue.Enqueue(ctx.UserContext(), jobID.String()); err != nil {
		return optimizationQueueError()
	}
	ownedSlot = false
	return c.writeAcknowledgement(ctx, record, true)
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
		return optimizationError(err)
	}
	if job.UserID != user.UserID {
		return optimizationNotFoundError()
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: optimizationJobData(job)})
}

// validateOptimizationSubmissionBody validates the JSON shape before any service dispatch.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validateOptimizationSubmissionBody(body map[string]any) error {
	if len(body) != 3 {
		return errors.New("optimization request contains unsupported fields")
	}
	dailyDietID, ok := body["dailyDietId"].(string)
	if !ok || !validUUIDString(dailyDietID) {
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
	DailyDietID      uuid.UUID
	TolerancePercent float64
	ExcludedMealIDs  []uuid.UUID
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
	return optimizationSubmissionRequest{DailyDietID: dailyDietID, TolerancePercent: raw.TolerancePercent, ExcludedMealIDs: excluded}, nil
}

// optimizationRequestHash creates the digest of the validated request bytes.
// Implements DESIGN-004 JobStatusTracker.
func optimizationRequestHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// lookupOptimizationIdempotency returns an exact replay or a body conflict.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) lookupOptimizationIdempotency(ctx context.Context, userID uuid.UUID, key, bodyHash string) (repository.CheckoutIdempotencyRecord, bool, error) {
	record, err := c.idempotency.GetCheckoutIdempotency(ctx, userID, optimizationMethod, optimizationJobsRoute, key)
	if err != nil {
		if repository.IsKind(err, repository.ErrorKindNotFound) {
			return repository.CheckoutIdempotencyRecord{}, false, nil
		}
		return repository.CheckoutIdempotencyRecord{}, false, err
	}
	if record.BodyHash != bodyHash {
		return repository.CheckoutIdempotencyRecord{}, false, repository.NewError(repository.ErrorKindConflict, "idempotency key reused with different body", nil)
	}
	return record, true, nil
}

// writeAcknowledgement returns the persisted 202 response for a submission.
// Implements DESIGN-004 JobStatusTracker.
func (c *OptimizationController) writeAcknowledgement(ctx *fiber.Ctx, record repository.CheckoutIdempotencyRecord, replayed bool) error {
	var ack map[string]any
	if err := json.Unmarshal(record.ResponseBody, &ack); err != nil {
		return optimizationDependencyError()
	}
	ctx.Set(fiber.HeaderLocation, stringValue(ack["pollUrl"]))
	return ctx.Status(record.StatusCode).JSON(Envelope{Status: "accepted", RequestID: requestID(ctx), Data: ack})
}

// optimizationJobIDFromAcknowledgement extracts the durable job identity from a claim.
// Implements DESIGN-004 JobStatusTracker.
func optimizationJobIDFromAcknowledgement(record repository.CheckoutIdempotencyRecord) (uuid.UUID, error) {
	var acknowledgement struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(record.ResponseBody, &acknowledgement); err != nil {
		return uuid.Nil, err
	}
	jobID, err := uuid.Parse(acknowledgement.JobID)
	if err != nil || jobID == uuid.Nil {
		return uuid.Nil, errors.New("optimization acknowledgement job ID is invalid")
	}
	return jobID, nil
}

// optimizationAcknowledgementData builds the public queued response.
// Implements DESIGN-004 JobStatusTracker.
func optimizationAcknowledgementData(jobID uuid.UUID) map[string]any {
	return map[string]any{"jobId": jobID.String(), "status": string(worker.OptimizationJobQueued), "pollUrl": optimizationPollPath + jobID.String()}
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
		data["failure"] = map[string]any{"code": string(job.Failure.Code), "message": job.Failure.Message}
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
	return value, nil
}

// validateOptimizationJobID validates the polling path identifier.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validateOptimizationJobID(value string) error {
	return validateUUIDValue(value, "optimization job id")
}

// validUUIDString reports whether a request UUID is valid and non-zero.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validUUIDString(value string) bool {
	return validateUUIDValue(value, "uuid") == nil
}

// validateUUIDValue validates a non-zero UUID string.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func validateUUIDValue(value, _ string) error {
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

// stringValue reads a string field from an acknowledgement map.
// Implements DESIGN-004 JobStatusTracker.
func stringValue(value any) string {
	valueString, _ := value.(string)
	return valueString
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

// optimizationAdmissionError returns a safe retryable per-user capacity error.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
func optimizationAdmissionError(code, message string) AppError {
	return AppError{HTTPStatus: fiber.StatusTooManyRequests, Category: "rate_limit", Code: code, Message: message, Retryable: true}
}

// setOptimizationRetryAfter writes a positive whole-second retry delay.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
func setOptimizationRetryAfter(ctx *fiber.Ctx, retryAfter time.Duration) {
	seconds := int64(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	ctx.Set(fiber.HeaderRetryAfter, strconv.FormatInt(seconds, 10))
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
	case repository.IsKind(err, repository.ErrorKindValidation):
		return optimizationValidationError()
	case repository.IsKind(err, repository.ErrorKindConflict):
		return optimizationIdempotencyConflict()
	default:
		return err
	}
}

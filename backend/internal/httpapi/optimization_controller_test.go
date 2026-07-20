package httpapi

// Implements DESIGN-004 JobStatusTracker HTTP integration verification.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// TestOptimizationHTTPSubmissionAndPolling verifies IT-ARCH-004-001 and SW-REQ-006.
func TestOptimizationHTTPSubmissionAndPolling(t *testing.T) {
	userID := uuid.New()
	dietID := uuid.New()
	store := newOptimizationHTTPJobStore()
	queue := &optimizationHTTPQueue{}
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}
	telemetrySink := &observability.MemorySink{}
	controller := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{}).WithTelemetry(observability.NewOptimizationTelemetry(telemetrySink, telemetrySink, 1))
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)
	body := optimizationHTTPBody(dietID, 20)

	response := optimizationHTTPSubmit(t, app, body, cookies, csrf, "optimization-key-1")
	if response.StatusCode != fiber.StatusAccepted {
		t.Fatalf("submit status = %d, want 202", response.StatusCode)
	}
	jobID := optimizationHTTPJobID(t, response)
	if queue.calls != 1 || len(store.jobs) != 1 || diets.getCalls != 1 {
		t.Fatalf("submit side effects queue=%d jobs=%d dietLoads=%d, want 1, 1, 1", queue.calls, len(store.jobs), diets.getCalls)
	}
	if !optimizationMetricOutcome(telemetrySink.Metrics, "accepted") {
		t.Fatalf("missing accepted optimization metric: %+v", telemetrySink.Metrics)
	}
	job := store.jobs[jobID]
	if job.UserID != userID || job.DailyDietID != dietID || job.TolerancePercent != 20 || job.Status != worker.OptimizationJobQueued {
		t.Fatalf("saved job = %+v, want server-owned queued request", job)
	}
	if got := response.Header.Get(fiber.HeaderLocation); got != optimizationPollPath+jobID.String() {
		t.Fatalf("Location = %q, want poll URL", got)
	}

	poll := optimizationHTTPPoll(t, app, jobID, cookies)
	if poll.StatusCode != fiber.StatusOK || poll.Data["status"] != string(worker.OptimizationJobQueued) {
		t.Fatalf("queued poll = %d %+v", poll.StatusCode, poll.Data)
	}
	started := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	job.Status, job.StartedAt = worker.OptimizationJobProcessing, &started
	store.setJob(job)
	poll = optimizationHTTPPoll(t, app, jobID, cookies)
	if poll.StatusCode != fiber.StatusOK || poll.Data["status"] != string(worker.OptimizationJobProcessing) || poll.Data["startedAt"] == nil {
		t.Fatalf("processing poll = %d %+v", poll.StatusCode, poll.Data)
	}
	finished := started.Add(time.Minute)
	job.Status, job.FinishedAt = worker.OptimizationJobCompleted, &finished
	job.Alternatives = []optimization.DietAlternative{{Meals: []optimization.MealQuantity{{MealID: uuid.New(), Quantity: 100, Unit: "g", Position: 0}}, Macros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, Calories: 290}}
	store.setJob(job)
	poll = optimizationHTTPPoll(t, app, jobID, cookies)
	if poll.StatusCode != fiber.StatusOK || poll.Data["status"] != string(worker.OptimizationJobCompleted) || len(poll.Data["alternatives"].([]any)) != 1 {
		t.Fatalf("completed poll = %d %+v", poll.StatusCode, poll.Data)
	}
	alternative := poll.Data["alternatives"].([]any)[0].(map[string]any)
	macros := alternative["macros"].(map[string]any)
	if macros["calories"] != float64(290) {
		t.Fatalf("completed alternative macros = %+v, want nested calories 290", macros)
	}
	if _, exists := alternative["calories"]; exists {
		t.Fatalf("completed alternative = %+v, must not expose legacy top-level calories", alternative)
	}

	job.Status = worker.OptimizationJobFailed
	job.Failure = &worker.OptimizationJobFailure{Code: optimization.FailureCodeSolverInfeasible, Message: "No meal combination matches the requested targets."}
	store.setJob(job)
	poll = optimizationHTTPPoll(t, app, jobID, cookies)
	if poll.StatusCode != fiber.StatusOK || poll.Data["status"] != string(worker.OptimizationJobCompleted) {
		t.Fatalf("terminal transition regressed = %d %+v", poll.StatusCode, poll.Data)
	}
}

func optimizationMetricOutcome(metrics []observability.MetricPoint, outcome string) bool {
	for _, metric := range metrics {
		if metric.Name == observability.MetricOptimizationSubmissionTotal && metric.Labels["outcome"] == outcome {
			return true
		}
	}
	return false
}

// TestOptimizationHTTPEntitlementAndOwnershipGuards verifies IT-ARCH-004-001,
// ARCH-004, DESIGN-004, and SW-REQ-006/SW-REQ-042/SW-REQ-043: denial before
// queue side effects and server-scoped diet reads.
func TestOptimizationHTTPEntitlementAndOwnershipGuards(t *testing.T) {
	dietID := uuid.New()
	queue := &optimizationHTTPQueue{}
	store := newOptimizationHTTPJobStore()
	diets := &optimizationHTTPDiets{dietID: dietID}
	controller := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{})
	freeUser := uuid.New()
	authenticator, cookies := testJWTAuth(t, testConfig(), freeUser, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)
	response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "free-user-key")
	if response.StatusCode != fiber.StatusForbidden || queue.calls != 0 || len(store.jobs) != 0 || diets.getCalls != 0 {
		t.Fatalf("free submit status=%d queue=%d jobs=%d dietLoads=%d, want denied before side effects", response.StatusCode, queue.calls, len(store.jobs), diets.getCalls)
	}

	otherUser := uuid.New()
	otherAuth, otherCookies := testJWTAuth(t, testConfig(), otherUser, nil)
	otherApp := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: otherAuth, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	job := worker.OptimizationJob{JobID: uuid.New(), UserID: freeUser, DailyDietID: dietID, Status: worker.OptimizationJobQueued, CreatedAt: time.Now().UTC()}
	store.jobs[job.JobID] = job
	poll := optimizationHTTPPoll(t, otherApp, job.JobID, otherCookies)
	if poll.StatusCode != fiber.StatusNotFound || poll.Data != nil {
		t.Fatalf("cross-user poll = %d %+v, want indistinguishable not-found", poll.StatusCode, poll.Data)
	}
}

// TestOptimizationHTTPAnonymousSubmissionIsDeniedBeforeSideEffects verifies IT-ARCH-004-001 protected submission behavior.
func TestOptimizationHTTPAnonymousSubmissionIsDeniedBeforeSideEffects(t *testing.T) {
	queue := &optimizationHTTPQueue{}
	store := newOptimizationHTTPJobStore()
	dietID := uuid.New()
	controller := NewOptimizationController(store, queue, &optimizationHTTPDiets{dietID: dietID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{})
	app := mustNewRouter(t, Dependencies{Config: testConfig(), CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})

	response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), nil, "", "anonymous-key")
	if response.StatusCode != fiber.StatusUnauthorized || queue.calls != 0 || len(store.jobs) != 0 {
		t.Fatalf("anonymous submit status=%d queue=%d jobs=%d, want 401 before side effects", response.StatusCode, queue.calls, len(store.jobs))
	}
}

func TestOptimizationHTTPAdmissionRejectsBeforeDurableSideEffects(t *testing.T) {
	tests := []struct {
		name, code string
		status     worker.OptimizationAdmissionStatus
	}{
		{name: "active job", code: "optimization_in_progress", status: worker.OptimizationAdmissionActive},
		{name: "hourly rate", code: "optimization_rate_limited", status: worker.OptimizationAdmissionRateLimited},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, dietID := uuid.New(), uuid.New()
			store := newOptimizationHTTPJobStore()
			queue := &optimizationHTTPQueue{}
			idempotency := newOptimizationHTTPIdempotencyStore()
			admission := &optimizationHTTPAdmission{decision: worker.OptimizationAdmissionDecision{Status: tt.status, RetryAfter: time.Minute}}
			controller := NewOptimizationController(store, queue, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, idempotency, admission)
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
			csrf, cookies := fetchCSRFToken(t, app, cookies...)

			response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "admission-key-1")
			if response.StatusCode != fiber.StatusTooManyRequests || response.Error == nil || response.Error.Code != tt.code {
				t.Fatalf("response = %d %+v, want 429 %s", response.StatusCode, response.Error, tt.code)
			}
			if response.Header.Get(fiber.HeaderRetryAfter) == "" || queue.calls != 0 || len(store.jobs) != 0 || len(idempotency.records) != 0 {
				t.Fatalf("rejected side effects retry=%q queue=%d jobs=%d idempotency=%d", response.Header.Get(fiber.HeaderRetryAfter), queue.calls, len(store.jobs), len(idempotency.records))
			}
		})
	}
}

func TestOptimizationHTTPRejectsClientAuthoredTargetMacros(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store := newOptimizationHTTPJobStore()
	queue := &optimizationHTTPQueue{}
	controller := NewOptimizationController(store, queue, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{})
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)
	body := `{"dailyDietId":"` + dietID.String() + `","targetMacros":{"protein":40,"carbohydrates":80,"fat":20},"tolerancePercent":10,"excludedMealIds":[]}`

	response := optimizationHTTPSubmit(t, app, body, cookies, csrf, "legacy-target-key")
	if response.StatusCode != fiber.StatusBadRequest || queue.calls != 0 || len(store.jobs) != 0 {
		t.Fatalf("legacy target response=%d queue=%d jobs=%d, want side-effect-free 400", response.StatusCode, queue.calls, len(store.jobs))
	}
}

// TestOptimizationHTTPIdempotencyAndQueueFailure verifies IT-ARCH-004-003 and
// IT-ARCH-004-004, ARCH-004, DESIGN-004, and SW-REQ-080: exact replay,
// changed-body conflict, and asynchronous outage handling.
func TestOptimizationHTTPIdempotencyAndQueueFailure(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store := newOptimizationHTTPJobStore()
	queue := &optimizationHTTPQueue{}
	idempotency := newOptimizationHTTPIdempotencyStore()
	controller := NewOptimizationController(store, queue, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, idempotency, &optimizationHTTPAdmission{})
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)
	body := optimizationHTTPBody(dietID, 20)
	first := optimizationHTTPSubmit(t, app, body, cookies, csrf, "same-key-1")
	firstID := optimizationHTTPJobID(t, first)
	replay := optimizationHTTPSubmit(t, app, body, cookies, csrf, "same-key-1")
	if replay.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, replay) != firstID || queue.calls != 1 {
		t.Fatalf("replay status=%d job=%s queueCalls=%d, want original acknowledgement and one enqueue", replay.StatusCode, optimizationHTTPJobID(t, replay), queue.calls)
	}
	conflict := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 21), cookies, csrf, "same-key-1")
	if conflict.StatusCode != fiber.StatusConflict || queue.calls != 1 {
		t.Fatalf("changed-body status=%d queueCalls=%d, want 409 and no second enqueue", conflict.StatusCode, queue.calls)
	}

	firstExcluded, secondExcluded := uuid.New(), uuid.New()
	ordered := optimizationHTTPSubmit(t, app, optimizationHTTPBodyWithExcluded(dietID, 22, []uuid.UUID{firstExcluded, secondExcluded}), cookies, csrf, "ordered-key-1")
	if ordered.StatusCode != fiber.StatusAccepted {
		t.Fatalf("ordered submit status = %d, want 202", ordered.StatusCode)
	}
	reordered := optimizationHTTPSubmit(t, app, optimizationHTTPBodyWithExcluded(dietID, 22, []uuid.UUID{secondExcluded, firstExcluded}), cookies, csrf, "ordered-key-1")
	if reordered.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, reordered) != optimizationHTTPJobID(t, ordered) || queue.calls != 2 {
		t.Fatalf("reordered-body status=%d queueCalls=%d, want exact replay and no publication", reordered.StatusCode, queue.calls)
	}
	syntacticallyChanged := `{"excludedMealIds":["` + firstExcluded.String() + `","` + secondExcluded.String() + `"],"dailyDietId":"` + dietID.String() + `","tolerancePercent":22}`
	if changed := optimizationHTTPSubmit(t, app, syntacticallyChanged, cookies, csrf, "ordered-key-1"); changed.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, changed) != optimizationHTTPJobID(t, ordered) || queue.calls != 2 {
		t.Fatalf("syntactically changed body status=%d queueCalls=%d, want canonical replay and no publication", changed.StatusCode, queue.calls)
	}
	uppercaseUUIDs := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(syntacticallyChanged, dietID.String(), strings.ToUpper(dietID.String())), firstExcluded.String(), strings.ToUpper(firstExcluded.String())), secondExcluded.String(), strings.ToUpper(secondExcluded.String()))
	if changed := optimizationHTTPSubmit(t, app, uppercaseUUIDs, cookies, csrf, "ordered-key-1"); changed.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, changed) != optimizationHTTPJobID(t, ordered) || queue.calls != 2 {
		t.Fatalf("UUID-case changed body status=%d queueCalls=%d, want canonical replay and no publication", changed.StatusCode, queue.calls)
	}
	numericSpelling := strings.Replace(syntacticallyChanged, `"tolerancePercent":22`, `"tolerancePercent":2.2e1`, 1)
	if changed := optimizationHTTPSubmit(t, app, numericSpelling, cookies, csrf, "ordered-key-1"); changed.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, changed) != optimizationHTTPJobID(t, ordered) || queue.calls != 2 {
		t.Fatalf("numeric-spelling changed body status=%d queueCalls=%d, want canonical replay and no publication", changed.StatusCode, queue.calls)
	}

	zero := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 0), cookies, csrf, "zero-tolerance-key")
	negativeZero := strings.Replace(optimizationHTTPBody(dietID, 0), `"tolerancePercent":0`, `"tolerancePercent":-0.0`, 1)
	if changed := optimizationHTTPSubmit(t, app, negativeZero, cookies, csrf, "zero-tolerance-key"); changed.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, changed) != optimizationHTTPJobID(t, zero) || queue.calls != 3 {
		t.Fatalf("negative-zero body status=%d queueCalls=%d, want canonical replay and no publication", changed.StatusCode, queue.calls)
	}

	queue.err = errors.New("redis down")
	failed := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 30), cookies, csrf, "queue-failure-key")
	if failed.StatusCode != fiber.StatusServiceUnavailable || queue.calls != 4 || len(store.jobs) != 4 {
		t.Fatalf("queue failure status=%d queueCalls=%d jobs=%d, want 503 and recoverable queued state", failed.StatusCode, queue.calls, len(store.jobs))
	}
	queue.err = nil
	recovered := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 30), cookies, csrf, "queue-failure-key")
	if recovered.StatusCode != fiber.StatusAccepted || queue.calls != 5 || len(store.jobs) != 4 {
		t.Fatalf("queue recovery status=%d error=%+v queueCalls=%d jobs=%d, want replayed 202 and one repaired publication", recovered.StatusCode, recovered.Error, queue.calls, len(store.jobs))
	}
}

// TestOptimizationHTTPConcurrentControllersClaimOneJob verifies IT-ARCH-004-001 cross-process idempotency publication.
func TestOptimizationHTTPConcurrentControllersClaimOneJob(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store := newOptimizationHTTPJobStore()
	queue := &optimizationHTTPQueue{}
	idempotency := newOptimizationHTTPIdempotencyStore()
	idempotency.storeBarrier = make(chan struct{})
	idempotency.storeReady = make(chan struct{})
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}

	buildApp := func() (*fiber.App, []*http.Cookie, string) {
		authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
		app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, idempotency, &optimizationHTTPAdmission{}).Routes()})
		csrf, cookies := fetchCSRFToken(t, app, cookies...)
		return app, cookies, csrf
	}
	firstApp, firstCookies, firstCSRF := buildApp()
	secondApp, secondCookies, secondCSRF := buildApp()
	body := optimizationHTTPBody(dietID, 20)
	responses := make(chan *optimizationHTTPEnvelopeResponse, 2)
	go func() {
		responses <- optimizationHTTPSubmit(t, firstApp, body, firstCookies, firstCSRF, "cross-process-key")
	}()
	go func() {
		responses <- optimizationHTTPSubmit(t, secondApp, body, secondCookies, secondCSRF, "cross-process-key")
	}()
	select {
	case <-idempotency.storeReady:
		close(idempotency.storeBarrier)
	case <-time.After(time.Second):
		t.Fatal("concurrent idempotency stores did not reach barrier")
	}
	firstResponse, secondResponse := <-responses, <-responses
	if firstResponse.StatusCode != fiber.StatusAccepted || secondResponse.StatusCode != fiber.StatusAccepted {
		t.Fatalf("concurrent statuses = %d, %d, want two 202 responses", firstResponse.StatusCode, secondResponse.StatusCode)
	}
	if firstID, secondID := optimizationHTTPJobID(t, firstResponse), optimizationHTTPJobID(t, secondResponse); firstID != secondID {
		t.Fatalf("concurrent job IDs = %s, %s, want one durable job", firstID, secondID)
	}
	if queue.calls != 1 || len(store.jobs) != 1 || len(idempotency.records) != 1 {
		t.Fatalf("concurrent publication queue=%d jobs=%d idempotency=%d, want 1, 1, 1", queue.calls, len(store.jobs), len(idempotency.records))
	}
}

// TestOptimizationHTTPExpiryKeepsOwnerIsolation verifies IT-ARCH-004-008,
// ARCH-004, DESIGN-004, and SW-REQ-006/SW-REQ-043: stable expired responses
// without cross-user disclosure.
func TestOptimizationHTTPExpiryKeepsOwnerIsolation(t *testing.T) {
	owner, other, jobID := uuid.New(), uuid.New(), uuid.New()
	store := newOptimizationHTTPJobStore()
	store.expired[jobID] = worker.OptimizationJobExpiredError{UserID: owner}
	controller := NewOptimizationController(store, nil, nil, nil, nil, nil)
	ownerAuth, ownerCookies := testJWTAuth(t, testConfig(), owner, nil)
	ownerApp := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: ownerAuth, Routes: controller.Routes()})
	ownerPoll := optimizationHTTPPoll(t, ownerApp, jobID, ownerCookies)
	if ownerPoll.StatusCode != fiber.StatusGone || ownerPoll.Error == nil || ownerPoll.Error.Code != "result_expired" {
		t.Fatalf("owner expired poll = %d %+v", ownerPoll.StatusCode, ownerPoll.Error)
	}
	otherAuth, otherCookies := testJWTAuth(t, testConfig(), other, nil)
	otherApp := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: otherAuth, Routes: controller.Routes()})
	otherPoll := optimizationHTTPPoll(t, otherApp, jobID, otherCookies)
	if otherPoll.StatusCode != fiber.StatusNotFound || otherPoll.Data != nil {
		t.Fatalf("other expired poll = %d %+v, want not-found", otherPoll.StatusCode, otherPoll.Data)
	}
}

// TestOptimizationHTTPFailedPollingUsesSafeSolverMessages verifies
// IT-ARCH-004-005, ARCH-004, DESIGN-004, and SW-REQ-021/SW-REQ-022 safe
// infeasible and timeout errors.
func TestOptimizationHTTPFailedPollingUsesSafeSolverMessages(t *testing.T) {
	tests := []struct {
		name    string
		code    optimization.OptimizationFailureCode
		message string
	}{
		{name: "infeasible", code: optimization.FailureCodeSolverInfeasible, message: "No meal combination matches the requested targets."},
		{name: "timeout", code: optimization.FailureCodeSolverTimeout, message: "Optimization took too long. Please try again."},
		{name: "validation", code: optimization.FailureCodeValidation, message: "The optimization request could not be validated."},
		{name: "worker crash", code: optimization.FailureCodeWorkerCrash, message: "Optimization could not be completed. Please try again."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, jobID := uuid.New(), uuid.New()
			store := newOptimizationHTTPJobStore()
			store.setJob(worker.OptimizationJob{
				JobID: jobID, UserID: userID, DailyDietID: uuid.New(), Status: worker.OptimizationJobFailed,
				CreatedAt: time.Now().UTC(), Failure: &worker.OptimizationJobFailure{Code: tt.code, Message: tt.message},
			})
			controller := NewOptimizationController(store, nil, nil, nil, nil, nil)
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, Routes: controller.Routes()})

			poll := optimizationHTTPPoll(t, app, jobID, cookies)
			failure, ok := poll.Data["failure"].(map[string]any)
			if poll.StatusCode != fiber.StatusOK || poll.Data["status"] != string(worker.OptimizationJobFailed) || !ok || failure["code"] != tt.code.String() || failure["message"] != tt.message {
				t.Fatalf("safe failure poll = %d %+v, want code=%q message=%q", poll.StatusCode, poll.Data, tt.code, tt.message)
			}
		})
	}
}

func TestOptimizationHTTPRejectsInvalidPersistedFailureBeforeProjection(t *testing.T) {
	userID, jobID := uuid.New(), uuid.New()
	store := newOptimizationHTTPJobStore()
	store.setJob(worker.OptimizationJob{
		JobID: jobID, UserID: userID, DailyDietID: uuid.New(), Status: worker.OptimizationJobFailed,
		CreatedAt: time.Now().UTC(), Failure: &worker.OptimizationJobFailure{Message: "postgres://internal"},
	})
	controller := NewOptimizationController(store, nil, nil, nil, nil, nil)
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, Routes: controller.Routes()})

	poll := optimizationHTTPPoll(t, app, jobID, cookies)
	if poll.StatusCode != fiber.StatusServiceUnavailable || poll.Data != nil {
		t.Fatalf("invalid persisted failure poll = %d %+v, want bounded dependency error", poll.StatusCode, poll.Data)
	}
}

// Implements DESIGN-004 JobStatusTracker authoritative similarity projection boundary.
func TestOptimizationHTTPRejectsInvalidPersistedSimilarityBeforeProjection(t *testing.T) {
	tests := []struct {
		name       string
		score      float64
		wantStatus int
	}{
		{name: "negative", score: -0.0001, wantStatus: fiber.StatusServiceUnavailable},
		{name: "above one", score: 1.0001, wantStatus: fiber.StatusServiceUnavailable},
		{name: "unrounded", score: 0.12345, wantStatus: fiber.StatusServiceUnavailable},
		{name: "rounded", score: 0.1234, wantStatus: fiber.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, jobID := uuid.New(), uuid.New()
			store := newOptimizationHTTPJobStore()
			store.setJob(worker.OptimizationJob{
				JobID: jobID, UserID: userID, DailyDietID: uuid.New(), Status: worker.OptimizationJobCompleted,
				CreatedAt: time.Now().UTC(), Alternatives: []optimization.DietAlternative{{
					Meals:  []optimization.MealQuantity{{MealID: uuid.New(), Quantity: 100, Unit: "g", Position: 0}},
					Macros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, Calories: 290, SimilarityScore: tt.score,
				}},
			})
			controller := NewOptimizationController(store, nil, nil, nil, nil, nil)
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, Routes: controller.Routes()})

			poll := optimizationHTTPPoll(t, app, jobID, cookies)
			if poll.StatusCode != tt.wantStatus {
				t.Fatalf("poll status = %d, want %d; data=%+v", poll.StatusCode, tt.wantStatus, poll.Data)
			}
			if tt.wantStatus != fiber.StatusOK && poll.Data != nil {
				t.Fatalf("invalid persisted result projected: %+v", poll.Data)
			}
		})
	}
}

// Implements DESIGN-004 JobStatusTracker raw persisted result decoding before HTTP projection.
func TestOptimizationHTTPRejectsMalformedRawSimilarityScore(t *testing.T) {
	userID, jobID := uuid.New(), uuid.New()
	tests := []struct {
		name       string
		score      any
		omitScore  bool
		wantStatus int
	}{
		{name: "omitted", omitScore: true, wantStatus: fiber.StatusServiceUnavailable},
		{name: "null", score: nil, wantStatus: fiber.StatusServiceUnavailable},
		{name: "string", score: "0", wantStatus: fiber.StatusServiceUnavailable},
		{name: "zero", score: float64(0), wantStatus: fiber.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := optimizationHTTPRawCompletedJob(t, jobID, userID, tt.score, tt.omitScore)
			controller := NewOptimizationController(&optimizationHTTPRawJobStore{payload: payload}, nil, nil, nil, nil, nil)
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, Routes: controller.Routes()})

			poll := optimizationHTTPPoll(t, app, jobID, cookies)
			if poll.StatusCode != tt.wantStatus {
				t.Fatalf("poll status = %d, want %d; data=%+v", poll.StatusCode, tt.wantStatus, poll.Data)
			}
			if tt.wantStatus != fiber.StatusOK && poll.Data != nil {
				t.Fatalf("malformed persisted result projected: %+v", poll.Data)
			}
		})
	}
}

type optimizationHTTPRawJobStore struct{ payload []byte }

func (s *optimizationHTTPRawJobStore) Load(context.Context, uuid.UUID) (worker.OptimizationJob, error) {
	var job worker.OptimizationJob
	return job, json.Unmarshal(s.payload, &job)
}

func (*optimizationHTTPRawJobStore) Save(context.Context, worker.OptimizationJob) error { return nil }
func (*optimizationHTTPRawJobStore) Delete(context.Context, uuid.UUID) error            { return nil }

func optimizationHTTPRawCompletedJob(t *testing.T, jobID, userID uuid.UUID, score any, omitScore bool) []byte {
	t.Helper()
	payload, err := json.Marshal(worker.OptimizationJob{
		JobID: jobID, UserID: userID, DailyDietID: uuid.New(), Status: worker.OptimizationJobCompleted,
		CreatedAt: time.Now().UTC(), Alternatives: []optimization.DietAlternative{{
			Meals:  []optimization.MealQuantity{{MealID: uuid.New(), Quantity: 100, Unit: "g", Position: 0}},
			Macros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, Calories: 290,
		}},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("decode raw fixture: %v", err)
	}
	alternative := raw["alternatives"].([]any)[0].(map[string]any)
	if omitScore {
		delete(alternative, "similarityScore")
	} else {
		alternative["similarityScore"] = score
	}
	payload, err = json.Marshal(raw)
	if err != nil {
		t.Fatalf("encode raw fixture: %v", err)
	}
	return payload
}

type optimizationHTTPEnvelope struct {
	Status    string         `json:"status"`
	RequestID string         `json:"requestId"`
	Data      map[string]any `json:"data"`
	Error     *AppError      `json:"error"`
}

func optimizationHTTPBody(dietID uuid.UUID, protein float64) string {
	return optimizationHTTPBodyWithExcluded(dietID, protein, nil)
}

func optimizationHTTPBodyWithExcluded(dietID uuid.UUID, protein float64, excluded []uuid.UUID) string {
	excludedValues := make([]string, 0, len(excluded))
	for _, id := range excluded {
		excludedValues = append(excludedValues, id.String())
	}
	payload := map[string]any{"dailyDietId": dietID.String(), "tolerancePercent": protein, "excludedMealIds": excludedValues}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func optimizationHTTPSubmit(t *testing.T, app *fiber.App, body string, cookies []*http.Cookie, csrf, key string) *optimizationHTTPEnvelopeResponse {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/optimization/jobs", nil)
	req.Body = http.NoBody
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/optimization/jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	req.Header.Set("X-CSRF-Token", csrf)
	addCookies(req, cookies)
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return readOptimizationHTTPResponse(t, response)
}

func optimizationHTTPPoll(t *testing.T, app *fiber.App, jobID uuid.UUID, cookies []*http.Cookie) *optimizationHTTPEnvelopeResponse {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, optimizationPollPath+jobID.String(), nil)
	addCookies(req, cookies)
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return readOptimizationHTTPResponse(t, response)
}

type optimizationHTTPEnvelopeResponse struct {
	Status     int
	StatusCode int
	Header     http.Header
	Data       map[string]any
	Error      *AppError
	Response   *http.Response
}

func readOptimizationHTTPResponse(t *testing.T, response *http.Response) *optimizationHTTPEnvelopeResponse {
	t.Helper()
	defer response.Body.Close()
	var envelope optimizationHTTPEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode optimization response: %v", err)
	}
	return &optimizationHTTPEnvelopeResponse{Status: response.StatusCode, StatusCode: response.StatusCode, Header: response.Header, Data: envelope.Data, Error: envelope.Error, Response: response}
}

func optimizationHTTPJobID(t *testing.T, response *optimizationHTTPEnvelopeResponse) uuid.UUID {
	t.Helper()
	value, ok := response.Data["jobId"].(string)
	if !ok {
		t.Fatalf("acknowledgement data = %+v", response.Data)
	}
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("acknowledgement job ID = %q: %v", value, err)
	}
	return id
}

type optimizationHTTPDiets struct {
	mu              sync.Mutex
	dietID, ownerID uuid.UUID
	err             error
	getCalls        int
	blockUser       uuid.UUID
	blockOnce       bool
	blockedCalls    int
	entered         chan struct{}
	release         chan struct{}
}

func (d *optimizationHTTPDiets) Create(context.Context, uuid.UUID, repository.SavedDiet) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (d *optimizationHTTPDiets) Get(ctx context.Context, userID, dietID uuid.UUID) (repository.SavedDiet, error) {
	d.mu.Lock()
	d.getCalls++
	block := d.blockUser == userID && d.release != nil && (!d.blockOnce || d.blockedCalls == 0)
	if block {
		d.blockedCalls++
	}
	entered, release, dependencyErr := d.entered, d.release, d.err
	configuredDietID, ownerID := d.dietID, d.ownerID
	d.mu.Unlock()
	if block {
		if entered != nil {
			select {
			case entered <- struct{}{}:
			default:
			}
		}
		select {
		case <-release:
		case <-ctx.Done():
			return repository.SavedDiet{}, ctx.Err()
		}
	}
	if dependencyErr != nil {
		return repository.SavedDiet{}, dependencyErr
	}
	if configuredDietID != dietID || (ownerID != uuid.Nil && ownerID != userID) {
		return repository.SavedDiet{}, repository.NewError(repository.ErrorKindNotFound, "diet not found", nil)
	}
	return repository.SavedDiet{ID: dietID, UserID: userID}, nil
}
func (d *optimizationHTTPDiets) List(context.Context, uuid.UUID) ([]repository.SavedDiet, error) {
	return nil, nil
}
func (d *optimizationHTTPDiets) Replace(context.Context, uuid.UUID, repository.SavedDiet) error {
	return nil
}
func (d *optimizationHTTPDiets) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type optimizationHTTPEntitlements struct {
	mu      sync.Mutex
	allowed bool
	err     error
	calls   int
}

func (e *optimizationHTTPEntitlements) CheckEntitlement(context.Context, uuid.UUID, entitlement.Feature) (entitlement.Decision, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	return entitlement.Decision{Allowed: e.allowed, Feature: entitlement.FeatureDailyDietAlternative}, e.err
}

type optimizationHTTPQueue struct {
	mu      sync.Mutex
	calls   int
	err     error
	entries map[string]string
}

type optimizationHTTPAdmission struct {
	mu       sync.Mutex
	decision worker.OptimizationAdmissionDecision
	err      error
	acquires int
	releases int
}

func (a *optimizationHTTPAdmission) Acquire(_ context.Context, req worker.OptimizationAdmissionRequest) (worker.OptimizationAdmissionDecision, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.acquires++
	if a.err != nil {
		return worker.OptimizationAdmissionDecision{}, a.err
	}
	if a.decision.Status == "" {
		return worker.OptimizationAdmissionDecision{Status: worker.OptimizationAdmissionAcquired, JobID: req.JobID}, nil
	}
	return a.decision, nil
}

func (a *optimizationHTTPAdmission) Release(context.Context, uuid.UUID, uuid.UUID) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.releases++
	return nil
}

func (q *optimizationHTTPQueue) Enqueue(_ context.Context, jobID string) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.entries == nil {
		q.entries = map[string]string{}
	}
	if entry, ok := q.entries[jobID]; ok {
		return entry, nil
	}
	q.calls++
	if q.err != nil {
		return "", q.err
	}
	q.entries[jobID] = "1-0"
	return q.entries[jobID], nil
}

type optimizationHTTPIdempotencyStore struct {
	mu           sync.Mutex
	records      map[string]repository.CheckoutIdempotencyRecord
	updateErr    error
	storeBarrier chan struct{}
	storeReady   chan struct{}
	storeCount   int
}

func newOptimizationHTTPIdempotencyStore() *optimizationHTTPIdempotencyStore {
	return &optimizationHTTPIdempotencyStore{records: map[string]repository.CheckoutIdempotencyRecord{}}
}

func (s *optimizationHTTPIdempotencyStore) GetCheckoutIdempotency(_ context.Context, userID uuid.UUID, method, route, key string) (repository.CheckoutIdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[userID.String()+method+route+key]
	if !ok {
		return repository.CheckoutIdempotencyRecord{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return record, nil
}
func (s *optimizationHTTPIdempotencyStore) StoreCheckoutIdempotency(_ context.Context, record repository.CheckoutIdempotencyRecord) error {
	s.mu.Lock()
	if s.storeBarrier != nil {
		s.storeCount++
		if s.storeCount == 2 {
			close(s.storeReady)
		}
		barrier := s.storeBarrier
		s.mu.Unlock()
		<-barrier
	} else {
		s.mu.Unlock()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := record.UserID.String() + record.Method + record.Route + record.Key
	if _, exists := s.records[key]; exists {
		return repository.NewError(repository.ErrorKindConflict, "duplicate", nil)
	}
	s.records[key] = record
	return nil
}

func (s *optimizationHTTPIdempotencyStore) UpdateCheckoutIdempotencyResponse(_ context.Context, record repository.CheckoutIdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateErr != nil {
		return s.updateErr
	}
	key := record.UserID.String() + record.Method + record.Route + record.Key
	current, exists := s.records[key]
	if !exists {
		return repository.NewError(repository.ErrorKindConflict, "claim changed", nil)
	}
	current.StatusCode = record.StatusCode
	current.ResponseBody = append([]byte(nil), record.ResponseBody...)
	s.records[key] = current
	return nil
}

type optimizationHTTPJobStore struct {
	mu      sync.Mutex
	jobs    map[uuid.UUID]worker.OptimizationJob
	expired map[uuid.UUID]worker.OptimizationJobExpiredError
}

func newOptimizationHTTPJobStore() *optimizationHTTPJobStore {
	return &optimizationHTTPJobStore{jobs: map[uuid.UUID]worker.OptimizationJob{}, expired: map[uuid.UUID]worker.OptimizationJobExpiredError{}}
}
func (s *optimizationHTTPJobStore) Save(_ context.Context, job worker.OptimizationJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, exists := s.jobs[job.JobID]; exists && (current.Status == worker.OptimizationJobCompleted || current.Status == worker.OptimizationJobFailed || current.Status == worker.OptimizationJobCancelled) {
		return nil
	}
	s.jobs[job.JobID] = job
	return nil
}
func (s *optimizationHTTPJobStore) Load(_ context.Context, jobID uuid.UUID) (worker.OptimizationJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expired, ok := s.expired[jobID]; ok {
		return worker.OptimizationJob{}, expired
	}
	job, ok := s.jobs[jobID]
	if !ok {
		return worker.OptimizationJob{}, worker.ErrOptimizationJobNotFound
	}
	return job, nil
}
func (s *optimizationHTTPJobStore) Delete(_ context.Context, jobID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, jobID)
	return nil
}

func (s *optimizationHTTPJobStore) setJob(job worker.OptimizationJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, exists := s.jobs[job.JobID]; exists && (current.Status == worker.OptimizationJobCompleted || current.Status == worker.OptimizationJobFailed || current.Status == worker.OptimizationJobCancelled) {
		return
	}
	s.jobs[job.JobID] = job
}

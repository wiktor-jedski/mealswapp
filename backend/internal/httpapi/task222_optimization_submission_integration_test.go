package httpapi

// Implements DESIGN-004 JobStatusTracker Task 222 HTTP integration coverage.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// TestOptimizationHTTPDifferentUsersProceedIndependentlyAndSameKeyIsUserScoped
// verifies IT-ARCH-004-001 and IT-ARCH-004-004, ARCH-004,
// DESIGN-004 JobStatusTracker, and
// SW-REQ-006/SW-REQ-021/SW-REQ-080/SW-REQ-082 concurrent identity isolation.
func TestOptimizationHTTPDifferentUsersProceedIndependentlyAndSameKeyIsUserScoped(t *testing.T) {
	firstUser, secondUser, dietID := uuid.New(), uuid.New(), uuid.New()
	diets := &optimizationHTTPDiets{dietID: dietID, blockUser: firstUser, entered: make(chan struct{}, 1), release: make(chan struct{})}
	store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{}
	controller := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{})
	firstApp, firstCookies, firstCSRF := optimizationHTTPTestApp(t, controller, firstUser)
	secondApp, secondCookies, secondCSRF := optimizationHTTPTestApp(t, controller, secondUser)

	firstResponse := make(chan *optimizationHTTPEnvelopeResponse, 1)
	go func() {
		firstResponse <- optimizationHTTPSubmit(t, firstApp, optimizationHTTPBody(dietID, 20), firstCookies, firstCSRF, "shared-user-key")
	}()
	select {
	case <-diets.entered:
	case <-time.After(time.Second):
		t.Fatal("first user did not reach blocked repository load")
	}
	second := optimizationHTTPSubmit(t, secondApp, optimizationHTTPBody(dietID, 20), secondCookies, secondCSRF, "shared-user-key")
	if second.StatusCode != fiber.StatusAccepted {
		t.Fatalf("second user status = %d, want independent 202", second.StatusCode)
	}
	close(diets.release)
	first := <-firstResponse
	if first.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, first) == optimizationHTTPJobID(t, second) {
		t.Fatalf("user-scoped acknowledgements = %d/%d, jobs=%s/%s", first.StatusCode, second.StatusCode, optimizationHTTPJobID(t, first), optimizationHTTPJobID(t, second))
	}
}

// TestOptimizationHTTPSubmissionHonorsRequestCancellation verifies
// IT-ARCH-004-001 and IT-ARCH-004-005, ARCH-004,
// DESIGN-004 JobStatusTracker, and
// SW-REQ-021/SW-REQ-080 cancellation before durable or queue side effects.
func TestOptimizationHTTPSubmissionHonorsRequestCancellation(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	diets := &optimizationHTTPDiets{dietID: dietID, blockUser: userID, release: make(chan struct{})}
	store, queue, admission := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{}, &optimizationHTTPAdmission{}
	idempotency := newOptimizationHTTPIdempotencyStore()
	controller := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, idempotency, admission)
	cfg := testConfig()
	cfg.APITimeout = 20 * time.Millisecond
	authenticator, cookies := testJWTAuth(t, cfg, userID, nil)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)

	response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "cancelled-key")
	if response.StatusCode != fiber.StatusGatewayTimeout || queue.calls != 0 || len(store.jobs) != 0 || len(idempotency.records) != 0 || admission.acquires != 0 {
		t.Fatalf("cancelled submit status=%d queue=%d jobs=%d claims=%d admission=%d", response.StatusCode, queue.calls, len(store.jobs), len(idempotency.records), admission.acquires)
	}
}

// TestOptimizationHTTPPublishedAcknowledgementReplayHasNoCurrentStateSideEffects
// verifies IT-ARCH-004-001 and IT-ARCH-004-004, ARCH-004,
// DESIGN-004 JobStatusTracker, and
// SW-REQ-006/SW-REQ-021 exact replay after current-state replacement or loss.
func TestOptimizationHTTPPublishedAcknowledgementReplayHasNoCurrentStateSideEffects(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{}
	entitlements := &optimizationHTTPEntitlements{allowed: true}
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}
	admission := &optimizationHTTPAdmission{}
	controller := NewOptimizationController(store, queue, diets, entitlements, newOptimizationHTTPIdempotencyStore(), admission)
	app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
	body := optimizationHTTPBody(dietID, 20)
	first := optimizationHTTPSubmit(t, app, body, cookies, csrf, "stable-replay-key")
	jobID := optimizationHTTPJobID(t, first)
	store.expired[jobID] = worker.OptimizationJobExpiredError{UserID: userID}
	entitlements.mu.Lock()
	entitlements.allowed, entitlements.err = false, errors.New("transient entitlement dependency failure")
	entitlements.mu.Unlock()
	diets.mu.Lock()
	diets.dietID, diets.ownerID = uuid.New(), uuid.New()
	diets.err = errors.New("transient diet dependency failure")
	diets.mu.Unlock()
	queue.mu.Lock()
	queue.err = errors.New("transient queue dependency failure")
	queue.mu.Unlock()
	admission.mu.Lock()
	admission.err = errors.New("transient admission dependency failure")
	admission.mu.Unlock()

	replay := optimizationHTTPSubmit(t, app, body, cookies, csrf, "stable-replay-key")
	wantPollURL := optimizationPollPath + jobID.String()
	if replay.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, replay) != jobID || len(replay.Data) != 3 || replay.Data["status"] != string(worker.OptimizationJobQueued) || replay.Data["pollUrl"] != wantPollURL || replay.Header.Get(fiber.HeaderLocation) != wantPollURL || replay.Header.Get(fiber.HeaderLocation) != first.Header.Get(fiber.HeaderLocation) || queue.calls != 1 || diets.getCalls != 1 || entitlements.calls != 1 || admission.acquires != 1 {
		t.Fatalf("exact replay status=%d job=%s location=%q side-effects queue=%d diet=%d entitlement=%d admission=%d", replay.StatusCode, optimizationHTTPJobID(t, replay), replay.Header.Get(fiber.HeaderLocation), queue.calls, diets.getCalls, entitlements.calls, admission.acquires)
	}
}

// TestOptimizationHTTPUnpublishedRepairRevalidatesAndPublishesOnce verifies
// IT-ARCH-004-001, ARCH-004, DESIGN-004 JobStatusTracker, and
// SW-REQ-006/SW-REQ-021/SW-REQ-080 across durable repair and queue publication.
func TestOptimizationHTTPUnpublishedRepairRevalidatesAndPublishesOnce(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("queue unavailable")}
	entitlements := &optimizationHTTPEntitlements{allowed: true}
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}
	admission := &optimizationHTTPAdmission{}
	controller := NewOptimizationController(store, queue, diets, entitlements, newOptimizationHTTPIdempotencyStore(), admission)
	app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
	body := optimizationHTTPBody(dietID, 20)
	if failed := optimizationHTTPSubmit(t, app, body, cookies, csrf, "repair-state-key"); failed.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("initial publication status = %d, want 503", failed.StatusCode)
	}

	entitlements.mu.Lock()
	entitlements.allowed = false
	entitlements.mu.Unlock()
	if denied := optimizationHTTPSubmit(t, app, body, cookies, csrf, "repair-state-key"); denied.StatusCode != fiber.StatusForbidden || queue.calls != 1 {
		t.Fatalf("entitlement repair status=%d queue=%d, want 403/no publication", denied.StatusCode, queue.calls)
	}
	entitlements.mu.Lock()
	entitlements.allowed = true
	entitlements.mu.Unlock()
	diets.ownerID = uuid.New()
	if missing := optimizationHTTPSubmit(t, app, body, cookies, csrf, "repair-state-key"); missing.StatusCode != fiber.StatusNotFound || queue.calls != 1 {
		t.Fatalf("ownership repair status=%d queue=%d, want 404/no publication", missing.StatusCode, queue.calls)
	}
	diets.ownerID = userID
	queue.mu.Lock()
	queue.err = nil
	queue.mu.Unlock()
	repaired := optimizationHTTPSubmit(t, app, body, cookies, csrf, "repair-state-key")
	if repaired.StatusCode != fiber.StatusAccepted || queue.calls != 2 || len(store.jobs) != 1 {
		t.Fatalf("repaired status=%d queue=%d jobs=%d, want one durable publication", repaired.StatusCode, queue.calls, len(store.jobs))
	}
	if replay := optimizationHTTPSubmit(t, app, body, cookies, csrf, "repair-state-key"); replay.StatusCode != fiber.StatusAccepted || queue.calls != 2 {
		t.Fatalf("post-repair replay status=%d queue=%d, want side-effect-free 202", replay.StatusCode, queue.calls)
	}
}

// TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission verifies
// IT-ARCH-004-001, ARCH-004, DESIGN-004 JobStatusTracker, and
// SW-REQ-021/SW-REQ-080/SW-REQ-082 concurrent repair and ownership transfer.
func TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("queue unavailable")}
	idempotency := newOptimizationHTTPIdempotencyStore()
	admission := &optimizationOwnershipAdmission{}
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}
	controllerA := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, idempotency, admission)
	controllerB := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, idempotency, admission)
	appA, cookiesA, csrfA := optimizationHTTPTestApp(t, controllerA, userID)
	appB, cookiesB, csrfB := optimizationHTTPTestApp(t, controllerB, userID)
	body, key := optimizationHTTPBody(dietID, 20), "stale-pending-repair-key"

	failed := optimizationHTTPSubmit(t, appA, body, cookiesA, csrfA, key)
	if failed.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("initial publication status = %d, want 503", failed.StatusCode)
	}
	queue.mu.Lock()
	queue.err = nil
	queue.mu.Unlock()
	diets.mu.Lock()
	diets.blockUser = userID
	diets.blockOnce = true
	diets.entered = make(chan struct{}, 1)
	diets.release = make(chan struct{})
	diets.mu.Unlock()

	responseB := make(chan *optimizationHTTPEnvelopeResponse, 1)
	go func() {
		responseB <- optimizationHTTPSubmit(t, appB, body, cookiesB, csrfB, key)
	}()
	select {
	case <-diets.entered:
	case <-time.After(time.Second):
		t.Fatal("controller B did not pause after reading pending acknowledgement")
	}

	responseA := optimizationHTTPSubmit(t, appA, body, cookiesA, csrfA, key)
	jobID := optimizationHTTPJobID(t, responseA)
	if responseA.StatusCode != fiber.StatusAccepted {
		t.Fatalf("controller A repair status = %d, want 202", responseA.StatusCode)
	}
	store.setJob(worker.OptimizationJob{JobID: jobID, UserID: userID, DailyDietID: dietID, Status: worker.OptimizationJobCompleted, CreatedAt: time.Now().UTC()})
	if err := admission.Release(context.Background(), userID, jobID); err != nil {
		t.Fatalf("worker terminal release: %v", err)
	}
	close(diets.release)
	response := <-responseB

	if response.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, response) != jobID {
		t.Fatalf("controller B replay status=%d job=%s, want 202/%s", response.StatusCode, optimizationHTTPJobID(t, response), jobID)
	}
	if admission.hasActiveSlot() {
		t.Fatal("stale pending repair stranded a newly acquired admission slot")
	}
	queue.mu.Lock()
	publications := len(queue.entries)
	queue.mu.Unlock()
	if publications != 1 {
		t.Fatalf("logical queue publications = %d, want 1", publications)
	}
}

func TestOptimizationHTTPPublishedQueueRetainsAdmissionOwnershipWhenAcknowledgementUpdateFails(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	tests := []struct {
		name           string
		initialQueue   error
		wantReleases   int
		wantQueueCalls int
	}{
		{name: "initial publication", wantQueueCalls: 1},
		{name: "repair publication", initialQueue: errors.New("queue unavailable"), wantReleases: 1, wantQueueCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: tt.initialQueue}
			idempotency := newOptimizationHTTPIdempotencyStore()
			admission := &optimizationHTTPAdmission{}
			controller := NewOptimizationController(store, queue, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, idempotency, admission)
			app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
			body := optimizationHTTPBody(dietID, 20)

			if tt.initialQueue != nil {
				failed := optimizationHTTPSubmit(t, app, body, cookies, csrf, "ack-update-failure-key")
				if failed.StatusCode != fiber.StatusServiceUnavailable || admission.releases != 1 {
					t.Fatalf("initial queue failure status=%d releases=%d", failed.StatusCode, admission.releases)
				}
				queue.mu.Lock()
				queue.err = nil
				queue.mu.Unlock()
			}
			idempotency.mu.Lock()
			idempotency.updateErr = repository.NewError(repository.ErrorKindInternal, "acknowledgement update failed", nil)
			idempotency.mu.Unlock()

			response := optimizationHTTPSubmit(t, app, body, cookies, csrf, "ack-update-failure-key")
			if response.StatusCode == fiber.StatusAccepted {
				t.Fatalf("acknowledgement update status=%d, want failure", response.StatusCode)
			}
			if admission.releases != tt.wantReleases || queue.calls != tt.wantQueueCalls {
				t.Fatalf("published handoff releases=%d queue=%d, want %d/%d", admission.releases, queue.calls, tt.wantReleases, tt.wantQueueCalls)
			}
		})
	}
}

func TestOptimizationHTTPUnpublishedRepairHonorsCancellationBeforePublication(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("queue unavailable")}
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}
	admission := &optimizationHTTPAdmission{}
	idempotency := newOptimizationHTTPIdempotencyStore()
	controller := NewOptimizationController(store, queue, diets, &optimizationHTTPEntitlements{allowed: true}, idempotency, admission)
	app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
	body := optimizationHTTPBody(dietID, 20)
	if failed := optimizationHTTPSubmit(t, app, body, cookies, csrf, "cancelled-repair-key"); failed.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("initial publication status = %d, want 503", failed.StatusCode)
	}

	diets.blockUser, diets.release = userID, make(chan struct{})
	cfg := testConfig()
	cfg.APITimeout = 20 * time.Millisecond
	authenticator, retryCookies := testJWTAuth(t, cfg, userID, nil)
	retryApp := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})
	retryCSRF, retryCookies := fetchCSRFToken(t, retryApp, retryCookies...)
	cancelled := optimizationHTTPSubmit(t, retryApp, body, retryCookies, retryCSRF, "cancelled-repair-key")
	if cancelled.StatusCode != fiber.StatusGatewayTimeout || queue.calls != 1 || len(store.jobs) != 1 || len(idempotency.records) != 1 || admission.acquires != 1 {
		t.Fatalf("cancelled repair status=%d queue=%d jobs=%d claims=%d admission=%d", cancelled.StatusCode, queue.calls, len(store.jobs), len(idempotency.records), admission.acquires)
	}
}

func TestOptimizationHTTPAdmission429UsesSharedRetryContract(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	controller := NewOptimizationController(newOptimizationHTTPJobStore(), &optimizationHTTPQueue{}, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{decision: worker.OptimizationAdmissionDecision{Status: worker.OptimizationAdmissionActive, RetryAfter: 1500 * time.Millisecond}})
	app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
	response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "admission-contract-key")
	if response.StatusCode != fiber.StatusTooManyRequests || response.Header.Get(fiber.HeaderRetryAfter) != "2" || response.Error == nil || response.Error.Category != "rate_limit" || !response.Error.Retryable {
		t.Fatalf("429 contract status=%d retry=%q error=%+v", response.StatusCode, response.Header.Get(fiber.HeaderRetryAfter), response.Error)
	}
}

func TestOptimizationHTTPRetryAfterRoundsUpAndStaysPositive(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter time.Duration
		want       string
	}{
		{name: "negative", retryAfter: -time.Second, want: "1"},
		{name: "zero", want: "1"},
		{name: "sub-second", retryAfter: time.Millisecond, want: "1"},
		{name: "fractional second", retryAfter: 1500 * time.Millisecond, want: "2"},
		{name: "whole second", retryAfter: 2 * time.Second, want: "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(ctx *fiber.Ctx) error {
				return retryableTooManyRequests(ctx, tt.retryAfter, "rate_limit", "limited", "limited")
			})
			response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			got := response.Header.Get(fiber.HeaderRetryAfter)
			seconds, parseErr := strconv.ParseUint(got, 10, 64)
			if got != tt.want || parseErr != nil || seconds == 0 {
				t.Fatalf("Retry-After = %q, parse error=%v, want %q positive base-10 seconds", got, parseErr, tt.want)
			}
		})
	}
}

func TestOptimizationHTTPAdmission429MatchesOpenAPIContract(t *testing.T) {
	source, err := os.ReadFile("../../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	contract := string(source)
	optimizationStart := strings.Index(contract, "  /api/v1/optimization/jobs:")
	if optimizationStart < 0 {
		t.Fatal("optimization submission path is missing from OpenAPI")
	}
	optimizationEnd := strings.Index(contract[optimizationStart+1:], "  /api/v1/optimization/jobs/{jobId}:")
	if optimizationEnd < 0 || !strings.Contains(contract[optimizationStart:optimizationStart+1+optimizationEnd], "\"429\":\n          $ref: \"#/components/responses/TooManyRequests\"") || !strings.Contains(contract, "    TooManyRequests:\n") || !strings.Contains(contract, "          required: true\n") || !strings.Contains(contract, "            minimum: 1\n") {
		t.Fatal("optimization 429 does not use the required positive Retry-After OpenAPI component")
	}
}

// TestOptimizationHTTPRejectsMalformedPersistedAcknowledgementWithoutLocationFallback
// verifies IT-ARCH-004-001, ARCH-004, DESIGN-004 JobStatusTracker, and
// SW-REQ-021/SW-REQ-080 fail-closed malformed durable-contract handling.
func TestOptimizationHTTPRejectsMalformedPersistedAcknowledgementWithoutLocationFallback(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	idempotency := newOptimizationHTTPIdempotencyStore()
	request := optimizationSubmissionRequest{DailyDietID: dietID, TolerancePercent: 20, ExcludedMealIDs: []uuid.UUID{}}
	bodyHash, err := optimizationRequestHash(request)
	if err != nil {
		t.Fatal(err)
	}
	idempotency.records[userID.String()+optimizationMethod+optimizationJobsRoute+"malformed-ack-key"] = repository.CheckoutIdempotencyRecord{
		UserID: userID, Method: optimizationMethod, Route: optimizationJobsRoute, Key: "malformed-ack-key", BodyHash: bodyHash, StatusCode: fiber.StatusAccepted,
		ResponseBody: []byte(`{"jobId":"` + uuid.NewString() + `","status":"queued","pollUrl":"","publicationState":"published"}`),
	}
	controller := NewOptimizationController(nil, nil, nil, nil, idempotency, nil)
	app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
	response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "malformed-ack-key")
	if response.StatusCode != fiber.StatusServiceUnavailable || response.Header.Get(fiber.HeaderLocation) != "" {
		t.Fatalf("malformed acknowledgement status=%d location=%q", response.StatusCode, response.Header.Get(fiber.HeaderLocation))
	}
}

func optimizationHTTPTestApp(t *testing.T, controller *OptimizationController, userID uuid.UUID) (*fiber.App, []*http.Cookie, string) {
	t.Helper()
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)
	return app, cookies, csrf
}

type optimizationOwnershipAdmission struct {
	mu     sync.Mutex
	active *worker.OptimizationAdmissionRequest
}

func (a *optimizationOwnershipAdmission) Acquire(_ context.Context, req worker.OptimizationAdmissionRequest) (worker.OptimizationAdmissionDecision, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.active == nil {
		copy := req
		a.active = &copy
		return worker.OptimizationAdmissionDecision{Status: worker.OptimizationAdmissionAcquired, JobID: req.JobID}, nil
	}
	if a.active.IdempotencyKey == req.IdempotencyKey && a.active.BodyHash == req.BodyHash {
		return worker.OptimizationAdmissionDecision{Status: worker.OptimizationAdmissionReplay, JobID: a.active.JobID}, nil
	}
	return worker.OptimizationAdmissionDecision{Status: worker.OptimizationAdmissionActive, JobID: a.active.JobID, RetryAfter: time.Minute}, nil
}

func (a *optimizationOwnershipAdmission) Release(_ context.Context, _ uuid.UUID, jobID uuid.UUID) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.active != nil && a.active.JobID == jobID {
		a.active = nil
	}
	return nil
}

func (a *optimizationOwnershipAdmission) hasActiveSlot() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.active != nil
}

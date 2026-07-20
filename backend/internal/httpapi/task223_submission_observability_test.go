package httpapi

// Implements DESIGN-014 MetricsCollector submission and cleanup verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

func TestTask223SubmissionHTTPOutcomesMatchFinalResponse(t *testing.T) {
	tests := []struct {
		name       string
		outcome    observability.OptimizationSubmissionOutcome
		statusCode int
		configure  func(*optimizationHTTPQueue, *optimizationHTTPEntitlements, *optimizationHTTPAdmission, *OptimizationIdempotencyRepository, *OptimizationJobStateStore)
		replay     bool
	}{
		{name: "accepted", outcome: observability.OptimizationSubmissionAccepted, statusCode: fiber.StatusAccepted},
		{name: "successful replay", outcome: observability.OptimizationSubmissionReplayed, statusCode: fiber.StatusAccepted, replay: true},
		{name: "rejected", outcome: observability.OptimizationSubmissionRejected, statusCode: fiber.StatusForbidden, configure: func(_ *optimizationHTTPQueue, entitlement *optimizationHTTPEntitlements, _ *optimizationHTTPAdmission, _ *OptimizationIdempotencyRepository, _ *OptimizationJobStateStore) {
			entitlement.allowed = false
		}},
		{name: "dependency error", outcome: observability.OptimizationSubmissionDependencyError, statusCode: fiber.StatusServiceUnavailable, configure: func(_ *optimizationHTTPQueue, _ *optimizationHTTPEntitlements, _ *optimizationHTTPAdmission, idempotency *OptimizationIdempotencyRepository, _ *OptimizationJobStateStore) {
			*idempotency = nil
		}},
		{name: "queue error", outcome: observability.OptimizationSubmissionQueueError, statusCode: fiber.StatusServiceUnavailable, configure: func(queue *optimizationHTTPQueue, _ *optimizationHTTPEntitlements, _ *optimizationHTTPAdmission, _ *OptimizationIdempotencyRepository, _ *OptimizationJobStateStore) {
			queue.err = errors.New("queue unavailable")
		}},
		{name: "unexpected error", outcome: observability.OptimizationSubmissionError, statusCode: fiber.StatusInternalServerError, configure: func(_ *optimizationHTTPQueue, _ *optimizationHTTPEntitlements, _ *optimizationHTTPAdmission, _ *OptimizationIdempotencyRepository, jobs *OptimizationJobStateStore) {
			*jobs = task223FailingJobStore{OptimizationJobStateStore: *jobs}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, dietID := uuid.New(), uuid.New()
			queue := &optimizationHTTPQueue{}
			entitlements := &optimizationHTTPEntitlements{allowed: true}
			admission := &optimizationHTTPAdmission{}
			var idempotency OptimizationIdempotencyRepository = newOptimizationHTTPIdempotencyStore()
			var jobs OptimizationJobStateStore = newOptimizationHTTPJobStore()
			if tt.configure != nil {
				tt.configure(queue, entitlements, admission, &idempotency, &jobs)
			}
			sink := &observability.MemorySink{}
			controller := NewOptimizationController(jobs, queue, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, entitlements, idempotency, admission).WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
			csrf, cookies := fetchCSRFToken(t, app, cookies...)
			body, key := optimizationHTTPBody(dietID, 20), "task-223-idempotency-key"
			if tt.replay {
				if first := optimizationHTTPSubmit(t, app, body, cookies, csrf, key); first.StatusCode != fiber.StatusAccepted {
					t.Fatalf("initial status = %d, want 202", first.StatusCode)
				}
			}

			response := optimizationHTTPSubmit(t, app, body, cookies, csrf, key)
			if response.StatusCode != tt.statusCode {
				t.Fatalf("status = %d, want %d", response.StatusCode, tt.statusCode)
			}
			if got := lastTask223SubmissionOutcome(t, sink.Metrics); got != string(tt.outcome) {
				t.Fatalf("submission outcome = %q, want %q", got, tt.outcome)
			}
			assertTask223TelemetryIsBounded(t, sink, userID, dietID, key, body)
		})
	}
}

func TestTask223FailedRepairIsNotReplayed(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	queue := &optimizationHTTPQueue{err: errors.New("queue unavailable")}
	diets := &optimizationHTTPDiets{dietID: dietID, ownerID: userID}
	sink := &observability.MemorySink{}
	controller := NewOptimizationController(newOptimizationHTTPJobStore(), queue, diets, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{}).WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)
	body, key := optimizationHTTPBody(dietID, 20), "task-223-repair-key"
	if first := optimizationHTTPSubmit(t, app, body, cookies, csrf, key); first.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("initial status = %d, want 503", first.StatusCode)
	}
	queue.err = nil
	diets.err = repository.NewError(repository.ErrorKindConnection, "diet unavailable", nil)
	if repair := optimizationHTTPSubmit(t, app, body, cookies, csrf, key); repair.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("repair status = %d, want 503", repair.StatusCode)
	}
	if got := lastTask223SubmissionOutcome(t, sink.Metrics); got != string(observability.OptimizationSubmissionDependencyError) {
		t.Fatalf("failed repair outcome = %q, want dependency_error", got)
	}
}

func TestTask223AdmissionCleanupIsBoundedObservableAndBestEffort(t *testing.T) {
	tests := []struct {
		name       string
		releaseErr error
		block      bool
	}{
		{name: "release error", releaseErr: errors.New("sensitive release diagnostic")},
		{name: "non-cooperative release", block: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, dietID := uuid.New(), uuid.New()
			admission := &task223Admission{releaseErr: tt.releaseErr}
			if tt.block {
				admission.release = make(chan struct{})
				defer close(admission.release)
			}
			sink := &observability.MemorySink{}
			controller := NewOptimizationController(newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("primary queue failure")}, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), admission).WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
			csrf, cookies := fetchCSRFToken(t, app, cookies...)
			body, key := optimizationHTTPBody(dietID, 20), "task-223-cleanup-key"

			started := time.Now()
			response := optimizationHTTPSubmit(t, app, body, cookies, csrf, key)
			if response.StatusCode != fiber.StatusServiceUnavailable || response.Error == nil || response.Error.Code != "queue_unavailable" {
				t.Fatalf("cleanup replaced primary response: status=%d error=%+v", response.StatusCode, response.Error)
			}
			if elapsed := time.Since(started); elapsed > 5*optimizationAdmissionCleanupTimeout {
				t.Fatalf("cleanup blocked for %v, deadline is %v", elapsed, optimizationAdmissionCleanupTimeout)
			}
			waitForTask223CleanupTelemetry(t, sink)
			assertTask223TelemetryIsBounded(t, sink, userID, dietID, key, body)
		})
	}
}

func TestTask223RepeatedPermanentlyBlockedReleaseHasBoundedOutstandingWork(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	admission := &task223Admission{release: make(chan struct{})}
	defer close(admission.release)
	controller := NewOptimizationController(newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("primary queue failure")}, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), admission)
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)

	for attempt := 0; attempt < 8; attempt++ {
		response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "task-223-blocked-release-"+string(rune('a'+attempt)))
		if response.StatusCode != fiber.StatusServiceUnavailable || response.Error == nil || response.Error.Code != "queue_unavailable" {
			t.Fatalf("attempt %d replaced primary response: status=%d error=%+v", attempt, response.StatusCode, response.Error)
		}
	}
	if calls := admission.releaseCalls.Load(); calls > 1 {
		t.Fatalf("permanently blocked release calls = %d, want at most 1 outstanding call", calls)
	}
}

func TestTask223BlockingCleanupSinksDoNotBlockPrimaryResponse(t *testing.T) {
	for _, blocked := range []string{"metric", "log"} {
		t.Run(blocked, func(t *testing.T) {
			userID, dietID := uuid.New(), uuid.New()
			sink := newTask223BlockingCleanupSink(blocked)
			defer sink.unblock()
			controller := NewOptimizationController(newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("primary queue failure")}, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &task223Admission{releaseErr: errors.New("sensitive release diagnostic")}).WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
			authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
			app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
			csrf, cookies := fetchCSRFToken(t, app, cookies...)

			started := time.Now()
			response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "task-223-blocking-sink-a")
			assertTask223PrimaryQueueResponse(t, blocked, response, time.Since(started))
			metric, event := sink.waitForCleanup(t)
			if metric.Name != observability.MetricOptimizationAdmissionCleanup || len(metric.Labels) != 1 || metric.Labels["outcome"] != "failed" {
				t.Fatalf("unsafe cleanup metric: %+v", metric)
			}
			if event.Message != "optimization_admission_cleanup" || len(event.Fields) != 1 || event.Fields["outcome"] != "failed" {
				t.Fatalf("unsafe cleanup event: %+v", event)
			}
			for attempt := 1; attempt < 8; attempt++ {
				started = time.Now()
				response = optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "task-223-blocking-sink-"+string(rune('a'+attempt)))
				assertTask223PrimaryQueueResponse(t, blocked, response, time.Since(started))
			}
			if calls := sink.blockedCalls(); calls != 1 {
				t.Fatalf("blocking %s sink calls = %d, want one capped outstanding call", blocked, calls)
			}
		})
	}
}

func TestTask223UniversallyBlockingSinkDoesNotBlockPrimaryResponse(t *testing.T) {
	userID, dietID := uuid.New(), uuid.New()
	writer := &task223UniversallyBlockingWriter{block: make(chan struct{})}
	defer writer.unblock()
	sink := observability.JSONSink{Writer: writer}
	controller := NewOptimizationController(newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("primary queue failure")}, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &task223Admission{releaseErr: errors.New("sensitive release diagnostic")}).WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
	authenticator, cookies := testJWTAuth(t, testConfig(), userID, nil)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Auth: authenticator, CSRF: NewCSRFManager(testConfig(), nil), Routes: controller.Routes()})
	csrf, cookies := fetchCSRFToken(t, app, cookies...)

	for attempt := 0; attempt < 8; attempt++ {
		started := time.Now()
		response := optimizationHTTPSubmit(t, app, optimizationHTTPBody(dietID, 20), cookies, csrf, "task-223-universal-blocking-writer-"+string(rune('a'+attempt)))
		assertTask223PrimaryQueueResponse(t, "universal writer", response, time.Since(started))
	}
	writer.assertCappedDeliveries(t)
}

type task223Admission struct {
	releaseErr   error
	release      chan struct{}
	releaseCalls atomic.Int64
}

type task223BlockingCleanupSink struct {
	blocked       string
	block         chan struct{}
	metricCleanup chan observability.MetricPoint
	logCleanup    chan observability.LogEvent
	metricCalls   atomic.Int64
	logCalls      atomic.Int64
	once          sync.Once
}

type task223UniversallyBlockingWriter struct {
	block             chan struct{}
	once              sync.Once
	cleanupMetrics    atomic.Int64
	cleanupLogs       atomic.Int64
	submissionMetrics atomic.Int64
	submissionLogs    atomic.Int64
	unknown           atomic.Int64
}

type task223FailingJobStore struct{ OptimizationJobStateStore }

func (task223FailingJobStore) Save(context.Context, worker.OptimizationJob) error {
	return errors.New("unexpected job-store failure")
}

func (a *task223Admission) Acquire(_ context.Context, request worker.OptimizationAdmissionRequest) (worker.OptimizationAdmissionDecision, error) {
	return worker.OptimizationAdmissionDecision{Status: worker.OptimizationAdmissionAcquired, JobID: request.JobID}, nil
}

func (a *task223Admission) Release(context.Context, uuid.UUID, uuid.UUID) error {
	a.releaseCalls.Add(1)
	if a.release != nil {
		<-a.release
	}
	return a.releaseErr
}

func newTask223BlockingCleanupSink(blocked string) *task223BlockingCleanupSink {
	return &task223BlockingCleanupSink{blocked: blocked, block: make(chan struct{}), metricCleanup: make(chan observability.MetricPoint, 1), logCleanup: make(chan observability.LogEvent, 1)}
}

func (s *task223BlockingCleanupSink) RecordMetric(_ context.Context, point observability.MetricPoint) error {
	if point.Name == observability.MetricOptimizationAdmissionCleanup {
		s.metricCalls.Add(1)
		select {
		case s.metricCleanup <- point:
		default:
		}
		if s.blocked == "metric" {
			<-s.block
		}
	}
	return nil
}

func (s *task223BlockingCleanupSink) Log(_ context.Context, event observability.LogEvent) error {
	if event.Message == "optimization_admission_cleanup" {
		s.logCalls.Add(1)
		select {
		case s.logCleanup <- event:
		default:
		}
		if s.blocked == "log" {
			<-s.block
		}
	}
	return nil
}

func (s *task223BlockingCleanupSink) waitForCleanup(t *testing.T) (observability.MetricPoint, observability.LogEvent) {
	t.Helper()
	var metric observability.MetricPoint
	var event observability.LogEvent
	for received := 0; received < 2; received++ {
		select {
		case metric = <-s.metricCleanup:
		case event = <-s.logCleanup:
		case <-time.After(5 * optimizationAdmissionCleanupTimeout):
			t.Fatal("cleanup telemetry was not delivered to both bounded lanes")
		}
	}
	return metric, event
}

func (s *task223BlockingCleanupSink) unblock() {
	s.once.Do(func() { close(s.block) })
}

func (s *task223BlockingCleanupSink) blockedCalls() int64 {
	if s.blocked == "metric" {
		return s.metricCalls.Load()
	}
	return s.logCalls.Load()
}

func (w *task223UniversallyBlockingWriter) Write(payload []byte) (int, error) {
	switch {
	case bytes.Contains(payload, []byte(`"name":"optimization_admission_cleanup_total"`)):
		w.cleanupMetrics.Add(1)
	case bytes.Contains(payload, []byte(`"message":"optimization_admission_cleanup"`)):
		w.cleanupLogs.Add(1)
	case bytes.Contains(payload, []byte(`"name":"optimization_submission_total"`)):
		w.submissionMetrics.Add(1)
	case bytes.Contains(payload, []byte(`"message":"optimization_submission"`)):
		w.submissionLogs.Add(1)
	default:
		w.unknown.Add(1)
	}
	<-w.block
	return len(payload), nil
}

func (w *task223UniversallyBlockingWriter) unblock() {
	w.once.Do(func() { close(w.block) })
}

func (w *task223UniversallyBlockingWriter) assertCappedDeliveries(t *testing.T) {
	t.Helper()
	if cleanupMetrics, cleanupLogs, submissionMetrics, submissionLogs, unknown := w.cleanupMetrics.Load(), w.cleanupLogs.Load(), w.submissionMetrics.Load(), w.submissionLogs.Load(), w.unknown.Load(); cleanupMetrics != 1 || cleanupLogs != 1 || submissionMetrics != 1 || submissionLogs != 1 || unknown != 0 {
		t.Fatalf("universally blocked writer calls cleanup_metric=%d cleanup_log=%d submission_metric=%d submission_log=%d unknown=%d, want one per bounded lane", cleanupMetrics, cleanupLogs, submissionMetrics, submissionLogs, unknown)
	}
}

func assertTask223PrimaryQueueResponse(t *testing.T, blocked string, response *optimizationHTTPEnvelopeResponse, elapsed time.Duration) {
	t.Helper()
	if response.StatusCode != fiber.StatusServiceUnavailable || response.Error == nil || response.Error.Code != "queue_unavailable" {
		t.Fatalf("blocking %s sink replaced primary response: status=%d error=%+v", blocked, response.StatusCode, response.Error)
	}
	if elapsed > 5*optimizationAdmissionCleanupTimeout {
		t.Fatalf("blocking %s sink held response for %v", blocked, elapsed)
	}
}

func lastTask223SubmissionOutcome(t *testing.T, metrics []observability.MetricPoint) string {
	t.Helper()
	for index := len(metrics) - 1; index >= 0; index-- {
		if metrics[index].Name == observability.MetricOptimizationSubmissionTotal {
			return metrics[index].Labels["outcome"]
		}
	}
	t.Fatal("submission metric not found")
	return ""
}

func task223CleanupMetric(metrics []observability.MetricPoint) bool {
	for _, metric := range metrics {
		if metric.Name == observability.MetricOptimizationAdmissionCleanup && len(metric.Labels) == 1 && metric.Labels["outcome"] == "failed" {
			return true
		}
	}
	return false
}

func assertTask223TelemetryIsBounded(t *testing.T, sink *observability.MemorySink, userID, dietID uuid.UUID, key, body string) {
	t.Helper()
	metrics, logs := sink.Snapshot()
	for _, metric := range metrics {
		if len(metric.Labels) > 1 {
			t.Fatalf("metric labels are unbounded: %+v", metric)
		}
		for label := range metric.Labels {
			if label != "outcome" {
				t.Fatalf("unexpected submission/cleanup label %q: %+v", label, metric)
			}
		}
	}
	payload, err := json.Marshal(struct {
		Metrics []observability.MetricPoint
		Logs    []observability.LogEvent
	}{metrics, logs})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{userID.String(), dietID.String(), key, body, "sensitive release diagnostic", "primary queue failure"} {
		if bytes.Contains(payload, []byte(forbidden)) {
			t.Fatalf("telemetry leaked sensitive value %q: %s", forbidden, payload)
		}
	}
}

func waitForTask223CleanupTelemetry(t *testing.T, sink *observability.MemorySink) {
	t.Helper()
	deadline := time.Now().Add(5 * optimizationAdmissionCleanupTimeout)
	for time.Now().Before(deadline) {
		metrics, logs := sink.Snapshot()
		if task223CleanupMetric(metrics) {
			for _, event := range logs {
				if event.Message == "optimization_admission_cleanup" {
					return
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("sanitized cleanup metric and event were not delivered")
}

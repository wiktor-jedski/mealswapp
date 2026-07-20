package observability

import (
	"context"
	"io"
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

// Implements DESIGN-014 MetricsCollector for Phase 07 optimization capacity evidence.
const (
	MetricOptimizationSubmissionTotal   = "optimization_submission_total"
	MetricOptimizationAdmissionCleanup  = "optimization_admission_cleanup_total"
	MetricOptimizationQueueCleanup      = "optimization_queue_cleanup_total"
	MetricOptimizationQueueDepth        = "optimization_queue_depth"
	MetricOptimizationQueueAgeSeconds   = "optimization_queue_age_seconds"
	MetricOptimizationWorkerActive      = "optimization_worker_active"
	MetricOptimizationWorkerUtilization = "optimization_worker_utilization"
	MetricOptimizationSolveDuration     = "optimization_solve_duration_seconds"
	MetricOptimizationSolveTotal        = "optimization_solve_total"
	MetricOptimizationJobTotal          = "optimization_job_total"
	MetricOptimizationRetryTotal        = "optimization_retry_total"
	MetricOptimizationResultExpiryTotal = "optimization_result_expiry_total"
	optimizationWorkerPool              = "optimization"
	optimizationTelemetryTimeout        = 100 * time.Millisecond
)

// OptimizationSubmissionOutcome is the complete bounded submission-result vocabulary.
// Implements DESIGN-014 MetricsCollector.
type OptimizationSubmissionOutcome string

// Implements DESIGN-014 MetricsCollector submission-result labels.
const (
	OptimizationSubmissionAccepted        OptimizationSubmissionOutcome = "accepted"
	OptimizationSubmissionReplayed        OptimizationSubmissionOutcome = "replayed"
	OptimizationSubmissionRejected        OptimizationSubmissionOutcome = "rejected"
	OptimizationSubmissionDependencyError OptimizationSubmissionOutcome = "dependency_error"
	OptimizationSubmissionQueueError      OptimizationSubmissionOutcome = "queue_error"
	OptimizationSubmissionError           OptimizationSubmissionOutcome = "error"
)

// QueueSnapshot is the safe readiness projection of Redis queue state.
// Implements DESIGN-014 UptimeMonitor and MetricsCollector.
type QueueSnapshot struct {
	Depth                   int64   `json:"depth"`
	OldestQueuedAgeSeconds  float64 `json:"oldestQueuedAgeSeconds"`
	OldestPendingAgeSeconds float64 `json:"oldestPendingAgeSeconds"`
}

// OptimizationTelemetry records only fixed-vocabulary optimization metrics and
// logs. It deliberately has no method accepting a user, diet, or job ID.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
type OptimizationTelemetry struct {
	metrics          MetricsCollector
	logs             LogSink
	workerCapacity   int64
	activeWorkers    atomic.Int64
	submissionMetric chan struct{}
	submissionLog    chan struct{}
	cleanupMetric    atomic.Bool
	cleanupLog       atomic.Bool
	queueCleanup     atomic.Bool
}

// NewOptimizationTelemetry constructs the bounded optimization telemetry
// adapter. A zero capacity means one worker, the local development default.
// Implements DESIGN-014 MetricsCollector.
func NewOptimizationTelemetry(metrics MetricsCollector, logs LogSink, workerCapacity int) *OptimizationTelemetry {
	if workerCapacity <= 0 {
		workerCapacity = 1
	}
	return &OptimizationTelemetry{
		metrics: metrics, logs: logs, workerCapacity: int64(workerCapacity),
		submissionMetric: make(chan struct{}, 1), submissionLog: make(chan struct{}, 1),
	}
}

// Submission records an accepted, replayed, rejected, or failed submission.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) Submission(ctx context.Context, outcome OptimizationSubmissionOutcome) {
	if t == nil {
		return
	}
	if _, ok := optimizationSubmissionOutcomes()[string(outcome)]; !ok {
		return
	}
	deliveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), optimizationTelemetryTimeout)
	defer cancel()
	metricDone := t.deliverSubmission(deliveryCtx, t.submissionMetric, func() {
		t.record(deliveryCtx, MetricOptimizationSubmissionTotal, 1, "submissions", map[string]string{"outcome": string(outcome)})
	})
	logDone := t.deliverSubmission(deliveryCtx, t.submissionLog, func() {
		t.event(deliveryCtx, "optimization_submission", map[string]any{"outcome": string(outcome)})
	})
	waitForOptimizationDelivery(deliveryCtx, metricDone)
	waitForOptimizationDelivery(deliveryCtx, logDone)
}

// deliverSubmission preserves synchronous delivery for cooperative sinks while
// capping noncooperative sink work at one metric and one log call.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func (t *OptimizationTelemetry) deliverSubmission(ctx context.Context, lane chan struct{}, deliver func()) <-chan struct{} {
	select {
	case lane <- struct{}{}:
	case <-ctx.Done():
		return nil
	}
	done := make(chan struct{})
	go func() {
		defer func() {
			<-lane
			close(done)
		}()
		deliver()
	}()
	return done
}

// waitForOptimizationDelivery bounds the caller lifecycle even when a sink
// ignores cancellation.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func waitForOptimizationDelivery(ctx context.Context, done <-chan struct{}) {
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// AdmissionCleanupFailed records a failed controller-owned slot release without identifiers.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func (t *OptimizationTelemetry) AdmissionCleanupFailed(ctx context.Context) {
	if t == nil {
		return
	}
	t.deliverCleanup(&t.cleanupMetric, func(deliveryCtx context.Context) {
		t.record(deliveryCtx, MetricOptimizationAdmissionCleanup, 1, "cleanups", map[string]string{"outcome": "failed"})
	})
	t.deliverCleanup(&t.cleanupLog, func(deliveryCtx context.Context) {
		t.event(deliveryCtx, "optimization_admission_cleanup", map[string]any{"outcome": "failed"})
	})
}

// QueueCleanupFailed records bounded lock-release failure without identifiers.
// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) QueueCleanupFailed(ctx context.Context) {
	if t == nil {
		return
	}
	t.deliverCleanup(&t.queueCleanup, func(deliveryCtx context.Context) {
		t.record(deliveryCtx, MetricOptimizationQueueCleanup, 1, "cleanups", map[string]string{"outcome": "failed"})
		t.event(deliveryCtx, "optimization_queue_cleanup", map[string]any{"outcome": "failed"})
	})
}

// deliverCleanup isolates each fixed cleanup sink behind one capped lane. A
// sink that ignores cancellation can retain that lane, but cannot block the
// HTTP lifecycle or create unbounded goroutines.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func (t *OptimizationTelemetry) deliverCleanup(active *atomic.Bool, deliver func(context.Context)) {
	if !active.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer active.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), optimizationTelemetryTimeout)
		defer cancel()
		deliver(ctx)
	}()
}

// QueueStats records queue size and age without queue payload metadata.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) QueueStats(ctx context.Context, depth int64, queuedAge, pendingAge time.Duration) {
	queuedAge = max(queuedAge, 0)
	pendingAge = max(pendingAge, 0)
	t.record(ctx, MetricOptimizationQueueDepth, float64(maxInt64(depth, 0)), "jobs", nil)
	t.record(ctx, MetricOptimizationQueueAgeSeconds, queuedAge.Seconds(), "seconds", map[string]string{"kind": "oldest_queued"})
	t.record(ctx, MetricOptimizationQueueAgeSeconds, pendingAge.Seconds(), "seconds", map[string]string{"kind": "oldest_pending"})
	t.event(ctx, "optimization_queue", map[string]any{
		"queueDepth":         maxInt64(depth, 0),
		"oldestQueuedAgeMs":  maxInt64(queuedAge.Milliseconds(), 0),
		"oldestPendingAgeMs": maxInt64(pendingAge.Milliseconds(), 0),
	})
}

// WorkerStarted records the bounded active-worker gauge and utilization.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) WorkerStarted(ctx context.Context) {
	if t == nil {
		return
	}
	active := t.activeWorkers.Add(1)
	t.workerGauge(ctx, active)
}

// WorkerFinished records the bounded active-worker gauge and utilization.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) WorkerFinished(ctx context.Context) {
	if t == nil {
		return
	}
	active := t.activeWorkers.Add(-1)
	if active < 0 {
		t.activeWorkers.Store(0)
		active = 0
	}
	t.workerGauge(ctx, active)
}

// Solve records one solver attempt with a bounded terminal status.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) Solve(ctx context.Context, duration time.Duration, status string) {
	t.record(ctx, MetricOptimizationSolveDuration, duration.Seconds(), "seconds", map[string]string{"status": status})
	t.record(ctx, MetricOptimizationSolveTotal, 1, "solves", map[string]string{"status": status})
	t.event(ctx, "optimization_solve", map[string]any{"status": status, "durationMs": maxInt64(duration.Milliseconds(), 0)})
}

// JobOutcome records a bounded optimization terminal status.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) JobOutcome(ctx context.Context, status string) {
	t.record(ctx, MetricOptimizationJobTotal, 1, "jobs", map[string]string{"status": status})
}

// Retry records a retry or retry exhaustion without exposing delivery IDs.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) Retry(ctx context.Context, outcome string) {
	t.record(ctx, MetricOptimizationRetryTotal, 1, "retries", map[string]string{"outcome": outcome})
	t.event(ctx, "optimization_retry", map[string]any{"outcome": outcome})
}

// ResultExpired records an owner-independent result TTL expiration.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) ResultExpired(ctx context.Context) {
	t.record(ctx, MetricOptimizationResultExpiryTotal, 1, "results", nil)
	t.event(ctx, "optimization_result_expired", nil)
}

// Record exposes the allow-listed metric boundary for deterministic tests and
// future adapters. Unknown metrics, mismatched queue units, or labels are dropped by design.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) Record(ctx context.Context, name string, value float64, unit string, labels map[string]string) {
	t.record(ctx, name, value, unit, labels)
}

// workerGauge records active workers and normalized utilization.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) workerGauge(ctx context.Context, active int64) {
	t.record(ctx, MetricOptimizationWorkerActive, float64(active), "workers", map[string]string{"pool": optimizationWorkerPool})
	utilization := float64(active) / float64(t.workerCapacity)
	if utilization > 1 {
		utilization = 1
	}
	t.record(ctx, MetricOptimizationWorkerUtilization, utilization, "ratio", map[string]string{"pool": optimizationWorkerPool})
	t.event(ctx, "optimization_worker", map[string]any{"activeWorkers": active, "workerCapacity": t.workerCapacity})
}

// record emits one validated metric point.
// Implements DESIGN-014 MetricsCollector.
func (t *OptimizationTelemetry) record(ctx context.Context, name string, value float64, unit string, labels map[string]string) {
	if t == nil || t.metrics == nil || !validOptimizationMetric(name, unit, labels) {
		return
	}
	if err := t.metrics.RecordMetric(ctx, MetricPoint{Name: name, Value: value, Unit: unit, Labels: cloneLabels(labels), ObservedAt: time.Now().UTC()}); err != nil {
		reportSinkFailure("metric", err)
	}
}

// event emits one filtered optimization log event.
// Implements DESIGN-014 LogAggregator.
func (t *OptimizationTelemetry) event(ctx context.Context, message string, fields map[string]any) {
	if t == nil || t.logs == nil || !validOptimizationEvent(message, fields) {
		return
	}
	if err := t.logs.Log(ctx, LogEvent{Service: "optimization", Level: "info", Message: message, Fields: cloneOptimizationFields(fields), CreatedAt: time.Now().UTC()}); err != nil {
		reportSinkFailure("log", err)
	}
}

// validOptimizationMetric rejects unknown names, mismatched queue units, keys, and values.
// Implements DESIGN-014 MetricsCollector.
func validOptimizationMetric(name, unit string, labels map[string]string) bool {
	if name == MetricOptimizationQueueDepth && unit != "jobs" || name == MetricOptimizationQueueAgeSeconds && unit != "seconds" {
		return false
	}
	allowed := map[string]map[string]map[string]struct{}{
		MetricOptimizationSubmissionTotal:   {"outcome": optimizationSubmissionOutcomes()},
		MetricOptimizationAdmissionCleanup:  {"outcome": {"failed": {}}},
		MetricOptimizationQueueCleanup:      {"outcome": {"failed": {}}},
		MetricOptimizationQueueDepth:        {},
		MetricOptimizationQueueAgeSeconds:   {"kind": {"oldest_queued": {}, "oldest_pending": {}}},
		MetricOptimizationWorkerActive:      {"pool": {optimizationWorkerPool: {}}},
		MetricOptimizationWorkerUtilization: {"pool": {optimizationWorkerPool: {}}},
		MetricOptimizationSolveDuration:     {"status": optimizationStatuses()},
		MetricOptimizationSolveTotal:        {"status": optimizationStatuses()},
		MetricOptimizationJobTotal:          {"status": optimizationStatuses()},
		MetricOptimizationRetryTotal:        {"outcome": {"retry": {}, "exhausted": {}}},
		MetricOptimizationResultExpiryTotal: {},
	}
	allowedLabels, ok := allowed[name]
	if !ok || len(labels) != len(allowedLabels) {
		return false
	}
	for key, value := range labels {
		values, ok := allowedLabels[key]
		if !ok {
			return false
		}
		if _, ok := values[value]; !ok {
			return false
		}
	}
	return true
}

// optimizationSubmissionOutcomes derives the metric allowlist from the typed vocabulary.
// Implements DESIGN-014 MetricsCollector.
func optimizationSubmissionOutcomes() map[string]struct{} {
	return map[string]struct{}{
		string(OptimizationSubmissionAccepted): {}, string(OptimizationSubmissionReplayed): {},
		string(OptimizationSubmissionRejected): {}, string(OptimizationSubmissionDependencyError): {},
		string(OptimizationSubmissionQueueError): {}, string(OptimizationSubmissionError): {},
	}
}

// optimizationStatuses returns the fixed solver/job status vocabulary.
// Implements DESIGN-014 MetricsCollector.
func optimizationStatuses() map[string]struct{} {
	return map[string]struct{}{"completed": {}, "failed": {}, "timeout": {}, "infeasible": {}, "validation": {}, "worker_crash": {}}
}

// validOptimizationMessage returns the fixed optimization log vocabulary.
// Implements DESIGN-014 LogAggregator.
func validOptimizationMessage(message string) bool {
	switch message {
	case "optimization_submission", "optimization_admission_cleanup", "optimization_queue_cleanup", "optimization_queue", "optimization_solve", "optimization_retry", "optimization_result_expired", "optimization_worker":
		return true
	default:
		return false
	}
}

// validOptimizationEvent rejects unbounded fields even for known log messages.
// Implements DESIGN-014 LogAggregator.
func validOptimizationEvent(message string, fields map[string]any) bool {
	if !validOptimizationMessage(message) {
		return false
	}
	stringField := func(key string, allowed map[string]struct{}) bool {
		value, ok := fields[key].(string)
		if !ok {
			return false
		}
		_, ok = allowed[value]
		return ok
	}
	int64Field := func(key string) bool {
		_, ok := fields[key].(int64)
		return ok
	}
	switch message {
	case "optimization_submission":
		return len(fields) == 1 && stringField("outcome", optimizationSubmissionOutcomes())
	case "optimization_admission_cleanup", "optimization_queue_cleanup":
		return len(fields) == 1 && stringField("outcome", map[string]struct{}{"failed": {}})
	case "optimization_queue":
		return len(fields) == 3 && int64Field("queueDepth") && int64Field("oldestQueuedAgeMs") && int64Field("oldestPendingAgeMs")
	case "optimization_solve":
		return len(fields) == 2 && stringField("status", optimizationStatuses()) && int64Field("durationMs")
	case "optimization_retry":
		return len(fields) == 1 && stringField("outcome", map[string]struct{}{"retry": {}, "exhausted": {}})
	case "optimization_result_expired":
		return len(fields) == 0
	case "optimization_worker":
		return len(fields) == 2 && int64Field("activeWorkers") && int64Field("workerCapacity")
	default:
		return false
	}
}

// cloneLabels copies validated metric labels before sink delivery.
// Implements DESIGN-014 MetricsCollector.
func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	result := make(map[string]string, len(labels))
	maps.Copy(result, labels)
	return result
}

// cloneOptimizationFields keeps only non-sensitive log fields.
// Implements DESIGN-014 LogAggregator.
func cloneOptimizationFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	allowed := map[string]struct{}{
		"outcome": {}, "queueDepth": {}, "oldestQueuedAgeMs": {}, "oldestPendingAgeMs": {},
		"status": {}, "durationMs": {}, "activeWorkers": {}, "workerCapacity": {},
	}
	result := make(map[string]any, len(fields))
	for key, value := range fields {
		if _, ok := allowed[key]; ok {
			result[key] = value
		}
	}
	return result
}

// maxInt64 clamps telemetry values at a non-negative lower bound.
// Implements DESIGN-014 MetricsCollector.
func maxInt64(value, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}

// reportSinkFailure is intentionally non-recursive and keeps telemetry best effort.
// Implements DESIGN-014 LogAggregator.
func reportSinkFailure(kind string, _ error) {
	optimizationFallbackMu.Lock()
	defer optimizationFallbackMu.Unlock()
	if fallback := optimizationFallbackWriter; fallback != nil {
		_, _ = fallback.Write([]byte("optimization observability " + kind + " sink failure\n"))
	}
}

// optimizationFallbackWriter prevents telemetry sink failures from recursing.
// Implements DESIGN-014 LogAggregator.
var optimizationFallbackWriter io.Writer = io.Discard

// Implements DESIGN-014 LogAggregator concurrent fallback serialization.
var optimizationFallbackMu sync.Mutex

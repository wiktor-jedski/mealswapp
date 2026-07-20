package observability

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"
)

// LogEvent is a structured application log record.
// Implements DESIGN-014 LogAggregator.
type LogEvent struct {
	RequestID string         `json:"requestId"`
	Service   string         `json:"service"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields"`
	CreatedAt time.Time      `json:"createdAt"`
}

// LogSink accepts structured application logs.
// Implements DESIGN-014 LogAggregator.
type LogSink interface {
	Log(context.Context, LogEvent) error
}

// MetricPoint is a low-cardinality metric observation.
// Implements DESIGN-014 MetricsCollector.
type MetricPoint struct {
	Name       string            `json:"name"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
	Labels     map[string]string `json:"labels"`
	ObservedAt time.Time         `json:"observedAt"`
}

// MetricsCollector accepts application metrics.
// Implements DESIGN-014 MetricsCollector.
type MetricsCollector interface {
	RecordMetric(context.Context, MetricPoint) error
}

// MemorySink captures deterministic local logs and metrics.
// Implements DESIGN-014 MetricsCollector.
type MemorySink struct {
	mu      sync.Mutex
	Logs    []LogEvent
	Metrics []MetricPoint
}

// JSONSink emits structured logs to an operator-provided writer.
// Implements DESIGN-014 LogAggregator.
type JSONSink struct {
	Writer io.Writer
}

// Implements DESIGN-014 LogAggregator compile-time JSON log sink contract.
var _ LogSink = JSONSink{}

// Implements DESIGN-014 MetricsCollector compile-time JSON metrics contract.
var _ MetricsCollector = JSONSink{}

// Implements DESIGN-014 LogAggregator compile-time memory log sink contract.
var _ LogSink = (*MemorySink)(nil)

// Implements DESIGN-014 MetricsCollector compile-time memory metrics contract.
var _ MetricsCollector = (*MemorySink)(nil)

// Log writes one JSON log line.
// Implements DESIGN-014 LogAggregator.
func (s JSONSink) Log(_ context.Context, event LogEvent) error {
	return json.NewEncoder(s.Writer).Encode(event)
}

// RecordMetric writes one JSON metric line.
// Implements DESIGN-014 MetricsCollector.
func (s JSONSink) RecordMetric(_ context.Context, point MetricPoint) error {
	return json.NewEncoder(s.Writer).Encode(point)
}

// Log stores one structured log record.
// Implements DESIGN-014 LogAggregator.
func (s *MemorySink) Log(_ context.Context, event LogEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Logs = append(s.Logs, event)
	return nil
}

// RecordMetric stores one metric point.
// Implements DESIGN-014 MetricsCollector.
func (s *MemorySink) RecordMetric(_ context.Context, point MetricPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Metrics = append(s.Metrics, point)
	return nil
}

// Snapshot returns race-safe copies of the locally captured telemetry.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func (s *MemorySink) Snapshot() ([]MetricPoint, []LogEvent) {
	if s == nil {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]MetricPoint(nil), s.Metrics...), append([]LogEvent(nil), s.Logs...)
}

// AlertRule configures local and deployed monitoring thresholds.
// Implements DESIGN-014 AlertManager.
type AlertRule struct {
	Name            string
	Metric          string
	Threshold       float64
	Comparison      string
	DurationSeconds int
	Severity        string
}

// DefaultAlertRules returns the Phase 02 latency alert baseline.
// Implements DESIGN-014 AlertManager.
func DefaultAlertRules() []AlertRule {
	rules := []AlertRule{
		{Name: "api_latency_warning", Metric: "http_request_latency_seconds_p95", Threshold: 1.5, Comparison: ">", DurationSeconds: 60, Severity: "warning"},
		{Name: "api_latency_critical", Metric: "http_request_latency_seconds_p95", Threshold: 2, Comparison: ">", DurationSeconds: 60, Severity: "critical"},
	}
	return append(rules, OptimizationAlertRules()...)
}

// OptimizationAlertRules returns the Phase 07 queue and worker capacity
// thresholds. Deployments may tune notification routing, but not the bounded
// metric vocabulary or the solver hard deadline.
// Implements DESIGN-014 AlertManager and SW-REQ-080/SW-REQ-082.
func OptimizationAlertRules() []AlertRule {
	return []AlertRule{
		{Name: "optimization_queue_depth_warning", Metric: MetricOptimizationQueueDepth, Threshold: 20, Comparison: ">", DurationSeconds: 60, Severity: "warning"},
		{Name: "optimization_queue_age_warning", Metric: MetricOptimizationQueueAgeSeconds, Threshold: 5, Comparison: ">", DurationSeconds: 60, Severity: "warning"},
		{Name: "optimization_queue_age_critical", Metric: MetricOptimizationQueueAgeSeconds, Threshold: 15, Comparison: ">", DurationSeconds: 60, Severity: "critical"},
		{Name: "optimization_worker_utilization_warning", Metric: MetricOptimizationWorkerUtilization, Threshold: 0.7, Comparison: ">", DurationSeconds: 300, Severity: "warning"},
		{Name: "optimization_worker_utilization_critical", Metric: MetricOptimizationWorkerUtilization, Threshold: 0.9, Comparison: ">", DurationSeconds: 300, Severity: "critical"},
		{Name: "optimization_solve_duration_critical", Metric: MetricOptimizationSolveDuration, Threshold: 30, Comparison: ">=", DurationSeconds: 1, Severity: "critical"},
	}
}

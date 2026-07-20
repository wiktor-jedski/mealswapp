package observability

// Implements DESIGN-014 MetricsCollector Task 226 queue-age metric verification.

import (
	"context"
	"testing"
	"time"
)

func TestTask226QueueAgeMetricsUseExactUnitsAndBoundedLabels(t *testing.T) {
	sink := &MemorySink{}
	telemetry := NewOptimizationTelemetry(sink, sink, 1)
	telemetry.QueueStats(context.Background(), -1, -time.Second, -2*time.Second)

	if len(sink.Metrics) != 3 {
		t.Fatalf("metric count = %d, want depth plus two ages", len(sink.Metrics))
	}
	depth, queued, pending := sink.Metrics[0], sink.Metrics[1], sink.Metrics[2]
	if depth.Name != MetricOptimizationQueueDepth || depth.Unit != "jobs" || depth.Value != 0 || len(depth.Labels) != 0 {
		t.Fatalf("depth metric = %#v", depth)
	}
	for _, metric := range []struct {
		point MetricPoint
		kind  string
	}{{queued, "oldest_queued"}, {pending, "oldest_pending"}} {
		if metric.point.Name != MetricOptimizationQueueAgeSeconds || metric.point.Unit != "seconds" || metric.point.Value != 0 {
			t.Fatalf("age metric = %#v", metric.point)
		}
		if len(metric.point.Labels) != 1 || metric.point.Labels["kind"] != metric.kind {
			t.Fatalf("age labels = %#v, want kind=%q only", metric.point.Labels, metric.kind)
		}
	}

	before := len(sink.Metrics)
	telemetry.Record(context.Background(), MetricOptimizationQueueAgeSeconds, 1, "seconds", map[string]string{"kind": "oldest_stream"})
	telemetry.Record(context.Background(), MetricOptimizationQueueAgeSeconds, 1, "seconds", map[string]string{"kind": "oldest_queued", "job_id": "forbidden"})
	if len(sink.Metrics) != before {
		t.Fatalf("unknown or unbounded queue-age labels emitted %d metrics", len(sink.Metrics)-before)
	}
}

func TestTask226RecordRejectsMismatchedQueueMetricUnits(t *testing.T) {
	sink := &MemorySink{}
	telemetry := NewOptimizationTelemetry(sink, sink, 1)
	telemetry.Record(context.Background(), MetricOptimizationQueueDepth, 1, "jobs", nil)
	telemetry.Record(context.Background(), MetricOptimizationQueueAgeSeconds, 1, "seconds", map[string]string{"kind": "oldest_queued"})
	telemetry.Record(context.Background(), MetricOptimizationQueueAgeSeconds, 1, "seconds", map[string]string{"kind": "oldest_pending"})
	if len(sink.Metrics) != 3 {
		t.Fatalf("valid queue metric count = %d, want 3", len(sink.Metrics))
	}

	tests := []struct {
		name   string
		metric string
		unit   string
		labels map[string]string
	}{
		{name: "depth as seconds", metric: MetricOptimizationQueueDepth, unit: "seconds"},
		{name: "depth without a unit", metric: MetricOptimizationQueueDepth},
		{name: "queued age as milliseconds", metric: MetricOptimizationQueueAgeSeconds, unit: "milliseconds", labels: map[string]string{"kind": "oldest_queued"}},
		{name: "pending age as jobs", metric: MetricOptimizationQueueAgeSeconds, unit: "jobs", labels: map[string]string{"kind": "oldest_pending"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(sink.Metrics)
			telemetry.Record(context.Background(), test.metric, 1, test.unit, test.labels)

			if len(sink.Metrics) != before {
				t.Fatalf("mismatched queue unit emitted metric %#v", sink.Metrics[before])
			}
		})
	}
}

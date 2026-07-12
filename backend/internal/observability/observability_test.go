package observability

// Implements DESIGN-014 LogAggregator, MetricsCollector, and AlertManager verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMemorySinkAndAlertRules(t *testing.T) {
	sink := &MemorySink{}
	if err := sink.Log(context.Background(), LogEvent{Message: "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := sink.RecordMetric(context.Background(), MetricPoint{Name: "metric"}); err != nil {
		t.Fatal(err)
	}
	rules := DefaultAlertRules()
	if len(sink.Logs) != 1 || len(sink.Metrics) != 1 || len(rules) != 8 || rules[0].Threshold != 1.5 || rules[1].Threshold != 2 {
		t.Fatalf("unexpected sink/rules: %+v %+v %+v", sink.Logs, sink.Metrics, rules)
	}
}

// TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData verifies
// IT-ARCH-004-007, ARCH-004, DESIGN-004/DESIGN-014, and SW-REQ-080/SW-REQ-082.
func TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData(t *testing.T) {
	sink := &MemorySink{}
	telemetry := NewOptimizationTelemetry(sink, sink, 2)
	telemetry.Submission(context.Background(), "accepted")
	telemetry.QueueStats(context.Background(), 3, 2*time.Second, 4*time.Second)
	telemetry.WorkerStarted(context.Background())
	telemetry.Solve(context.Background(), 250*time.Millisecond, "timeout")
	telemetry.JobOutcome(context.Background(), "infeasible")
	telemetry.Retry(context.Background(), "retry")
	telemetry.ResultExpired(context.Background())
	telemetry.WorkerFinished(context.Background())
	telemetry.Record(context.Background(), "optimization_submission_total", 1, "submissions", map[string]string{"outcome": "user-email@example.test"})
	telemetry.Record(context.Background(), "untrusted_metric", 1, "count", map[string]string{"id": "diet-contents"})

	metrics, err := json.Marshal(sink.Metrics)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := json.Marshal(sink.Logs)
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range [][]byte{metrics, logs} {
		for _, forbidden := range []string{"user-email@example.test", "diet-contents", "dailyDietId", "jobId", "userId"} {
			if bytes.Contains(payload, []byte(forbidden)) {
				t.Fatalf("telemetry leaked %q: %s", forbidden, payload)
			}
		}
	}
	if len(sink.Metrics) == 0 || len(sink.Logs) == 0 {
		t.Fatal("optimization telemetry did not emit evidence")
	}
	for _, point := range sink.Metrics {
		if len(point.Labels) > 1 {
			t.Fatalf("metric has too many labels: %+v", point)
		}
	}
}

func TestJSONSink(t *testing.T) {
	var output bytes.Buffer
	if err := (JSONSink{Writer: &output}).Log(context.Background(), LogEvent{Message: "ok"}); err != nil || output.String() == "" {
		t.Fatalf("Log() output=%q err=%v", output.String(), err)
	}
	if err := (JSONSink{Writer: &output}).RecordMetric(context.Background(), MetricPoint{Name: "metric"}); err != nil {
		t.Fatal(err)
	}
	if encoded := output.String(); !strings.Contains(encoded, `"requestId"`) || !strings.Contains(encoded, `"observedAt"`) || strings.Contains(encoded, `"ObservedAt"`) {
		t.Fatalf("JSONSink output=%q", encoded)
	}
}

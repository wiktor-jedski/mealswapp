package observability

// Implements DESIGN-014 LogAggregator, MetricsCollector, and AlertManager verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCloneLabelsPreservesNilSemanticsAndMutationIndependence(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   map[string]string
	}{
		{name: "nil", labels: nil, want: nil},
		{name: "empty", labels: map[string]string{}, want: nil},
		{name: "populated", labels: map[string]string{"outcome": "accepted"}, want: map[string]string{"outcome": "accepted"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cloneLabels(tt.labels)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("cloneLabels(%v) = %#v, want %#v", tt.labels, got, tt.want)
			}
			if len(tt.labels) == 0 {
				return
			}
			tt.labels["outcome"] = "rejected"
			if got["outcome"] != "accepted" {
				t.Fatalf("destination changed with source: %#v", got)
			}
			got["outcome"] = "error"
			if tt.labels["outcome"] != "rejected" {
				t.Fatalf("source changed with destination: %#v", tt.labels)
			}
		})
	}
}

func TestOptimizationSubmissionOutcomesAreBoundedAndRaceSafe(t *testing.T) {
	outcomes := []OptimizationSubmissionOutcome{
		OptimizationSubmissionAccepted,
		OptimizationSubmissionReplayed,
		OptimizationSubmissionRejected,
		OptimizationSubmissionDependencyError,
		OptimizationSubmissionQueueError,
		OptimizationSubmissionError,
	}
	sink := &MemorySink{}
	telemetry := NewOptimizationTelemetry(sink, sink, 1)
	var workers sync.WaitGroup
	for _, outcome := range outcomes {
		outcome := outcome
		workers.Add(1)
		go func() {
			defer workers.Done()
			telemetry.Submission(context.Background(), outcome)
		}()
	}
	workers.Wait()
	telemetry.Submission(context.Background(), OptimizationSubmissionOutcome("user@example.test"))

	if len(sink.Metrics) != len(outcomes) || len(sink.Logs) != len(outcomes) {
		t.Fatalf("emissions metrics=%d logs=%d, want %d each", len(sink.Metrics), len(sink.Logs), len(outcomes))
	}
	allowed := optimizationSubmissionOutcomes()
	for _, point := range sink.Metrics {
		if point.Name != MetricOptimizationSubmissionTotal || len(point.Labels) != 1 {
			t.Fatalf("unbounded submission metric: %+v", point)
		}
		if _, ok := allowed[point.Labels["outcome"]]; !ok {
			t.Fatalf("outcome %q is outside fixed allowlist", point.Labels["outcome"])
		}
	}
}

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
	telemetry.Submission(context.Background(), OptimizationSubmissionAccepted)
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

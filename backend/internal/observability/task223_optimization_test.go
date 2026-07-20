package observability

// Implements DESIGN-014 MetricsCollector Task 223 bounded-label verification.

import (
	"context"
	"maps"
	"testing"
)

func TestTask223SubmissionVocabularyIsBounded(t *testing.T) {
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
	for _, outcome := range outcomes {
		telemetry.Submission(context.Background(), outcome)
	}
	telemetry.Submission(context.Background(), OptimizationSubmissionOutcome("user-or-job-controlled"))

	if len(sink.Metrics) != len(outcomes) {
		t.Fatalf("submission metrics = %d, want %d allow-listed outcomes", len(sink.Metrics), len(outcomes))
	}
	for i, point := range sink.Metrics {
		if point.Name != MetricOptimizationSubmissionTotal || len(point.Labels) != 1 || point.Labels["outcome"] != string(outcomes[i]) {
			t.Fatalf("submission metric %d = %+v, want outcome %q", i, point, outcomes[i])
		}
	}
}

func TestTask223CloneLabelsPreservesSemanticsAndOwnership(t *testing.T) {
	if cloned := cloneLabels(nil); cloned != nil {
		t.Fatalf("cloneLabels(nil) = %#v, want nil", cloned)
	}
	if cloned := cloneLabels(map[string]string{}); cloned != nil {
		t.Fatalf("cloneLabels(empty) = %#v, want legacy nil semantics", cloned)
	}

	original := map[string]string{"outcome": "accepted"}
	cloned := cloneLabels(original)
	if !maps.Equal(cloned, original) {
		t.Fatalf("cloneLabels(populated) = %#v, want %#v", cloned, original)
	}
	original["outcome"] = "rejected"
	if cloned["outcome"] != "accepted" {
		t.Fatalf("source mutation changed clone: %#v", cloned)
	}
	cloned["outcome"] = "error"
	if original["outcome"] != "rejected" {
		t.Fatalf("clone mutation changed source: %#v", original)
	}
}

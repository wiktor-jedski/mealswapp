package observability

// Implements DESIGN-014 MetricsCollector and LogAggregator Task 234 privacy regression gate.

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestTask234TelemetryRejectsIdentifiersAndSanitizesSinkFailures(t *testing.T) {
	const secret = "person@example.test idempotency-key request-body solver-diagnostic"
	var fallback bytes.Buffer
	previousFallback := optimizationFallbackWriter
	optimizationFallbackWriter = &fallback
	t.Cleanup(func() { optimizationFallbackWriter = previousFallback })

	sink := task234FailingSink{err: errors.New(secret)}
	telemetry := NewOptimizationTelemetry(sink, sink, 1)
	telemetry.Submission(context.Background(), OptimizationSubmissionAccepted)
	telemetry.Submission(context.Background(), OptimizationSubmissionOutcome(secret))
	telemetry.Record(context.Background(), MetricOptimizationQueueAgeSeconds, 1, "seconds", map[string]string{"kind": "oldest_queued", "job_id": secret})
	telemetry.Solve(context.Background(), 0, secret)
	telemetry.Retry(context.Background(), secret)

	output := fallback.String()
	if strings.Contains(output, secret) {
		t.Fatalf("fallback telemetry leaked sink diagnostics: %q", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("fallback telemetry = %q, want two generic bounded records", output)
	}
	counts := map[string]int{}
	for _, line := range lines {
		if line != "optimization observability metric sink failure" && line != "optimization observability log sink failure" {
			t.Fatalf("unexpected fallback record %q", line)
		}
		counts[line]++
	}
	if counts["optimization observability metric sink failure"] != 1 || counts["optimization observability log sink failure"] != 1 {
		t.Fatalf("fallback telemetry = %q, want one metric and one log failure", output)
	}
}

type task234FailingSink struct{ err error }

func (s task234FailingSink) RecordMetric(context.Context, MetricPoint) error { return s.err }
func (s task234FailingSink) Log(context.Context, LogEvent) error             { return s.err }

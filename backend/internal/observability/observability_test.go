package observability

// Implements DESIGN-014 LogAggregator, MetricsCollector, and AlertManager verification.

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
	if len(sink.Logs) != 1 || len(sink.Metrics) != 1 || len(rules) != 2 || rules[0].Threshold != 1.5 || rules[1].Threshold != 2 {
		t.Fatalf("unexpected sink/rules: %+v %+v %+v", sink.Logs, sink.Metrics, rules)
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

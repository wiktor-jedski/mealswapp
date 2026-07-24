package observability

// Implements DESIGN-014 MetricsCollector/LogAggregator Task 260 privacy and load gate.

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type task260LoadFixture struct {
	Workers    int      `json:"workers"`
	Iterations int      `json:"iterations"`
	Forbidden  []string `json:"forbidden"`
}

type task260BlockingSink struct {
	metricCalls atomic.Int32
	logCalls    atomic.Int32
	entered     chan string
	release     chan struct{}
}

func (s *task260BlockingSink) RecordMetric(context.Context, MetricPoint) error {
	s.metricCalls.Add(1)
	s.entered <- "metric"
	<-s.release
	return nil
}

func (s *task260BlockingSink) Log(context.Context, LogEvent) error {
	s.logCalls.Add(1)
	s.entered <- "log"
	<-s.release
	return nil
}

func TestAdminExternalTelemetryLoadIsBoundedAndPrivacySafe(t *testing.T) {
	payload, err := os.ReadFile("testdata/task260_load.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture task260LoadFixture
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	sink := &MemorySink{}
	telemetry := NewAdminExternalTelemetry(sink, sink)
	ctx := context.Background()

	var workers sync.WaitGroup
	for range fixture.Workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range fixture.Iterations {
				telemetry.ProviderCall(ctx, "usda", "provider_unavailable", 125*time.Millisecond)
				telemetry.ProviderRetry(ctx, "usda", "scheduled")
				telemetry.ProviderQuota(ctx, "openfoodfacts", "exhausted")
				telemetry.NormalizationWarning(ctx, "external", "invalid_external_payload")
				telemetry.ImportOutcome(ctx, "openfoodfacts", "replayed")
				telemetry.AdminMutation(ctx, "manual_update", "audit_failed")
				telemetry.CustomItemLifecycle(ctx, "delete", "not_found")
			}
		}()
	}
	workers.Wait()

	for _, forbidden := range fixture.Forbidden {
		telemetry.ProviderCall(ctx, forbidden, forbidden, time.Second)
		telemetry.ProviderRetry(ctx, forbidden, forbidden)
		telemetry.ProviderQuota(ctx, forbidden, forbidden)
		telemetry.NormalizationWarning(ctx, forbidden, forbidden)
		telemetry.ImportOutcome(ctx, forbidden, forbidden)
		telemetry.AdminMutation(ctx, forbidden, forbidden)
		telemetry.CustomItemLifecycle(ctx, forbidden, forbidden)
	}

	metrics, logs := sink.Snapshot()
	wantMetrics := fixture.Workers * fixture.Iterations * 8
	wantLogs := fixture.Workers * fixture.Iterations * 7
	if len(metrics) != wantMetrics || len(logs) != wantLogs {
		t.Fatalf("emissions metrics=%d/%d logs=%d/%d", len(metrics), wantMetrics, len(logs), wantLogs)
	}
	encoded, err := json.Marshal(struct{ Metrics, Logs any }{metrics, logs})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range fixture.Forbidden {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("telemetry contains forbidden canary %q", forbidden)
		}
	}

	auditFailures := 0
	for _, point := range metrics {
		assertTask260MetricLabels(t, point)
		if point.Name == MetricAdminMutationOutcomes && point.Labels["outcome"] == "audit_failed" {
			auditFailures++
		}
	}
	if auditFailures != fixture.Workers*fixture.Iterations {
		t.Fatalf("audit failures=%d, want %d", auditFailures, fixture.Workers*fixture.Iterations)
	}
}

func assertTask260MetricLabels(t *testing.T, point MetricPoint) {
	t.Helper()
	allowedKeys := map[string][]string{
		MetricExternalProviderCalls: {"outcome", "provider"}, MetricExternalProviderLatency: {"outcome", "provider"},
		MetricExternalProviderRetries: {"outcome", "provider"}, MetricExternalProviderQuota: {"provider", "state"},
		MetricExternalNormalization: {"provider", "warning"}, MetricAdminImportOutcomes: {"outcome", "provider"},
		MetricAdminMutationOutcomes: {"operation", "outcome"}, MetricCustomItemLifecycleOutcomes: {"operation", "outcome"},
	}
	keys, ok := allowedKeys[point.Name]
	if !ok || len(point.Labels) != len(keys) {
		t.Fatalf("unbounded metric %+v", point)
	}
	for _, key := range keys {
		value := point.Labels[key]
		allowedValues := task260AllowedLabelValues(key)
		if value == "" || !allowed(value, allowedValues...) {
			t.Fatalf("label %s=%q is outside fixed allowlist in %+v", key, value, point)
		}
	}
}

func task260AllowedLabelValues(key string) []string {
	switch key {
	case "provider":
		return []string{"usda", "openfoodfacts", "external", "manual"}
	case "outcome":
		return []string{"success", "invalid_input", "not_configured", "provider_rejected", "provider_rate_limited", "provider_unavailable", "invalid_external_payload", "provider_response_too_large", "timeout", "canceled", "error", "scheduled", "exhausted", "created", "replayed", "merged", "validation_failed", "idempotency_conflict", "provider_conflict", "name_conflict", "dependency_failed", "succeeded", "failed", "audit_failed", "conflict", "not_found"}
	case "state":
		return []string{"available", "exhausted", "blocked", "unknown"}
	case "warning":
		return []string{"missing_image", "missing_macros", "missing_micronutrients", "missing_liquid_density", "uncertain_unit_conversion", "suspicious_liquid_macros", "invalid_external_payload"}
	case "operation":
		return []string{"import_food", "manual_create", "manual_update", "manual_delete", "classification.create", "classification.update", "classification.delete", "retry_deletion", "create", "get", "update", "delete", "list", "other"}
	default:
		return nil
	}
}

func TestAdminMutationUnknownOperationIsCollapsedAndAuditFailureDistinct(t *testing.T) {
	sink := &MemorySink{}
	telemetry := NewAdminExternalTelemetry(sink, sink)
	telemetry.AdminMutation(context.Background(), "attacker-controlled-operation", "failed")
	telemetry.AdminMutation(context.Background(), "manual_create", "audit_failed")
	metrics, logs := sink.Snapshot()
	if len(metrics) != 2 || metrics[0].Labels["operation"] != "other" || metrics[0].Labels["outcome"] != "failed" || metrics[1].Labels["outcome"] != "audit_failed" {
		t.Fatalf("admin metrics=%+v", metrics)
	}
	if len(logs) != 2 || logs[0].Fields["operation"] != "other" || logs[1].Fields["outcome"] != "audit_failed" {
		t.Fatalf("admin logs=%+v", logs)
	}
}

func TestAdminExternalTelemetryLogSinkAllowsOnlyBoundedProviderAndCurationMetadata(t *testing.T) {
	sink := &MemorySink{}
	telemetry := NewAdminExternalTelemetry(nil, sink)
	valid := []LogEvent{
		{Message: "external_provider_failure", Fields: map[string]any{"provider": "usda", "code": "provider_unavailable", "status": 503, "retryable": true}},
		{Message: "external_provider_payload_dropped", Fields: map[string]any{"provider": "openfoodfacts", "count": 2}},
		{Message: "curation_input_validation", Fields: map[string]any{"field": "external_search_query", "outcome": "rejected", "changed": false, "violationCount": 0}},
	}
	for _, event := range valid {
		if err := telemetry.Log(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	_ = telemetry.Log(context.Background(), LogEvent{Message: "external_provider_failure", Fields: map[string]any{"provider": "usda", "code": "private-code", "status": 503, "retryable": true, "query": "private query"}})
	_ = telemetry.Log(context.Background(), LogEvent{Message: "curation_input_validation", Fields: map[string]any{"field": "private-field", "outcome": "rejected", "changed": false, "violationCount": 0}})

	_, logs := sink.Snapshot()
	if len(logs) != len(valid) {
		t.Fatalf("bounded logs = %+v", logs)
	}
	encoded, err := json.Marshal(logs)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-code", "private query", "private-field"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("bounded adapter leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestAdminExternalTelemetryBlockingSinksHaveBoundedDispatch(t *testing.T) {
	sink := &task260BlockingSink{entered: make(chan string, 2), release: make(chan struct{})}
	defer close(sink.release)
	telemetry := NewAdminExternalTelemetry(sink, sink)
	done := make(chan struct{})
	go func() {
		telemetry.AdminMutation(context.Background(), "manual_create", "succeeded")
		close(done)
	}()

	seen := map[string]bool{}
	for range 2 {
		select {
		case kind := <-sink.entered:
			seen[kind] = true
		case <-time.After(250 * time.Millisecond):
			t.Fatal("telemetry sink was not invoked")
		}
	}
	if !seen["metric"] || !seen["log"] {
		t.Fatalf("sink calls=%v", seen)
	}
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("blocking telemetry sink held the request path")
	}

	var callers sync.WaitGroup
	for range 32 {
		callers.Add(1)
		go func() {
			defer callers.Done()
			telemetry.AdminMutation(context.Background(), "manual_update", "failed")
		}()
	}
	callersDone := make(chan struct{})
	go func() {
		callers.Wait()
		close(callersDone)
	}()
	select {
	case <-callersDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("backpressured telemetry callers did not return")
	}
	if sink.metricCalls.Load() != 1 || sink.logCalls.Load() != 1 {
		t.Fatalf("blocked sink fan-out metrics=%d logs=%d", sink.metricCalls.Load(), sink.logCalls.Load())
	}
}

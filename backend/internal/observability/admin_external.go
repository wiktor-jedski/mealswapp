package observability

import (
	"context"
	"time"
)

// Implements DESIGN-014 MetricsCollector names for Phase 08 administration and external data.
const (
	MetricExternalProviderCalls       = "external_provider_calls_total"
	MetricExternalProviderLatency     = "external_provider_latency_seconds"
	MetricExternalProviderRetries     = "external_provider_retries_total"
	MetricExternalProviderQuota       = "external_provider_quota_total"
	MetricExternalNormalization       = "external_normalization_warnings_total"
	MetricAdminImportOutcomes         = "admin_import_outcomes_total"
	MetricAdminMutationOutcomes       = "admin_mutation_outcomes_total"
	MetricCustomItemLifecycleOutcomes = "custom_item_lifecycle_outcomes_total"
	adminExternalTelemetryTimeout     = 100 * time.Millisecond
)

// adminExternalMetric is one already-allowlisted point in a bounded sink batch.
// Implements DESIGN-014 MetricsCollector backpressure.
type adminExternalMetric struct {
	name   string
	value  float64
	unit   string
	labels map[string]string
}

// adminExternalLog is one already-filtered event in a bounded sink batch.
// Implements DESIGN-014 LogAggregator backpressure.
type adminExternalLog struct {
	level   string
	message string
	fields  map[string]any
}

// AdminExternalTelemetry emits only allowlisted categories and numeric values.
// It deliberately has no API for query text, identities, URLs, payloads, keys, or snapshots.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
type AdminExternalTelemetry struct {
	metrics    MetricsCollector
	logs       LogSink
	metricLane chan struct{}
	logLane    chan struct{}
}

// Implements DESIGN-014 LogAggregator compile-time bounded log sink contract.
var _ LogSink = (*AdminExternalTelemetry)(nil)

// NewAdminExternalTelemetry creates the Phase 08 privacy boundary.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func NewAdminExternalTelemetry(metrics MetricsCollector, logs LogSink) *AdminExternalTelemetry {
	return &AdminExternalTelemetry{metrics: metrics, logs: logs, metricLane: make(chan struct{}, 1), logLane: make(chan struct{}, 1)}
}

// ProviderCall records one bounded external-provider attempt and latency.
// Implements DESIGN-014 MetricsCollector provider dependency metrics.
func (t *AdminExternalTelemetry) ProviderCall(ctx context.Context, provider, outcome string, latency time.Duration) {
	if !allowed(provider, "usda", "openfoodfacts") || !allowed(outcome, "success", "invalid_input", "not_configured", "provider_rejected", "provider_rate_limited", "provider_unavailable", "invalid_external_payload", "provider_response_too_large", "timeout", "canceled", "error") {
		return
	}
	if latency < 0 {
		latency = 0
	}
	labels := map[string]string{"provider": provider, "outcome": outcome}
	t.emit(ctx, []adminExternalMetric{
		{name: MetricExternalProviderCalls, value: 1, unit: "calls", labels: labels},
		{name: MetricExternalProviderLatency, value: latency.Seconds(), unit: "seconds", labels: labels},
	}, &adminExternalLog{level: "info", message: "external_provider_call", fields: map[string]any{"provider": provider, "outcome": outcome, "latencyMilliseconds": latency.Milliseconds()}})
}

// ProviderRetry records a retry state without attempt numbers or request data.
// Implements DESIGN-014 MetricsCollector provider retry metrics.
func (t *AdminExternalTelemetry) ProviderRetry(ctx context.Context, provider, outcome string) {
	if !allowed(provider, "usda", "openfoodfacts") || !allowed(outcome, "scheduled", "exhausted", "canceled") {
		return
	}
	labels := map[string]string{"provider": provider, "outcome": outcome}
	t.emit(ctx, []adminExternalMetric{{name: MetricExternalProviderRetries, value: 1, unit: "retries", labels: labels}}, &adminExternalLog{level: "info", message: "external_provider_retry", fields: stringFields(labels)})
}

// ProviderQuota records only a closed quota state, never raw response headers.
// Implements DESIGN-014 MetricsCollector provider rate-limit metrics.
func (t *AdminExternalTelemetry) ProviderQuota(ctx context.Context, provider, state string) {
	if !allowed(provider, "usda", "openfoodfacts") || !allowed(state, "available", "exhausted", "blocked", "unknown") {
		return
	}
	labels := map[string]string{"provider": provider, "state": state}
	t.emit(ctx, []adminExternalMetric{{name: MetricExternalProviderQuota, value: 1, unit: "observations", labels: labels}}, &adminExternalLog{level: "info", message: "external_provider_quota", fields: stringFields(labels)})
}

// NormalizationWarning records canonical warning categories only.
// Implements DESIGN-014 MetricsCollector normalization warning metrics.
func (t *AdminExternalTelemetry) NormalizationWarning(ctx context.Context, provider, warning string) {
	if !allowed(provider, "usda", "openfoodfacts", "external") || !allowed(warning, "missing_image", "missing_macros", "missing_micronutrients", "missing_liquid_density", "uncertain_unit_conversion", "suspicious_liquid_macros", "invalid_external_payload") {
		return
	}
	labels := map[string]string{"provider": provider, "warning": warning}
	t.emit(ctx, []adminExternalMetric{{name: MetricExternalNormalization, value: 1, unit: "warnings", labels: labels}}, &adminExternalLog{level: "warn", message: "external_normalization_warning", fields: stringFields(labels)})
}

// ImportOutcome records the final curated-import class.
// Implements DESIGN-014 MetricsCollector and DESIGN-009 DataImporter.
func (t *AdminExternalTelemetry) ImportOutcome(ctx context.Context, provider, outcome string) {
	if !allowed(provider, "usda", "openfoodfacts", "manual") || !allowed(outcome, "created", "replayed", "merged", "validation_failed", "idempotency_conflict", "provider_conflict", "name_conflict", "dependency_failed", "error") {
		return
	}
	labels := map[string]string{"provider": provider, "outcome": outcome}
	t.emit(ctx, []adminExternalMetric{{name: MetricAdminImportOutcomes, value: 1, unit: "imports", labels: labels}}, &adminExternalLog{level: "info", message: "admin_import_outcome", fields: stringFields(labels)})
}

// AdminMutation records bounded operation results; audit failure is a distinct outcome.
// Implements DESIGN-014 MetricsCollector and DESIGN-009 AdminController.
func (t *AdminExternalTelemetry) AdminMutation(ctx context.Context, operation, outcome string) {
	operation = boundedOperation(operation)
	if !allowed(outcome, "succeeded", "failed", "audit_failed") {
		return
	}
	labels := map[string]string{"operation": operation, "outcome": outcome}
	level := "info"
	if outcome != "succeeded" {
		level = "warn"
	}
	t.emit(ctx, []adminExternalMetric{{name: MetricAdminMutationOutcomes, value: 1, unit: "mutations", labels: labels}}, &adminExternalLog{level: level, message: "admin_mutation_outcome", fields: stringFields(labels)})
}

// CustomItemLifecycle records owner-scoped behavior without owner or item identity.
// Implements DESIGN-014 MetricsCollector and DESIGN-008 ProfileController.
func (t *AdminExternalTelemetry) CustomItemLifecycle(ctx context.Context, operation, outcome string) {
	if !allowed(operation, "create", "get", "update", "delete", "list") || !allowed(outcome, "succeeded", "replayed", "validation_failed", "conflict", "not_found", "dependency_failed", "error") {
		return
	}
	labels := map[string]string{"operation": operation, "outcome": outcome}
	t.emit(ctx, []adminExternalMetric{{name: MetricCustomItemLifecycleOutcomes, value: 1, unit: "operations", labels: labels}}, &adminExternalLog{level: "info", message: "custom_item_lifecycle", fields: stringFields(labels)})
}

// Log accepts only the provider and curation events owned by this privacy boundary.
// Implements DESIGN-014 LogAggregator and DESIGN-013 InputNormalizer metadata policy.
func (t *AdminExternalTelemetry) Log(ctx context.Context, event LogEvent) error {
	bounded, ok := boundedAdminExternalLog(event)
	if ok {
		t.emit(ctx, nil, &bounded)
	}
	return nil
}

// emit preserves cooperative delivery while bounding a noncooperative sink to
// one metric batch and one log call outside the provider or admin request path.
// Implements DESIGN-014 MetricsCollector and LogAggregator backpressure.
func (t *AdminExternalTelemetry) emit(ctx context.Context, metrics []adminExternalMetric, event *adminExternalLog) {
	if t == nil {
		return
	}
	deliveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), adminExternalTelemetryTimeout)
	defer cancel()
	metricDone := t.dispatch(deliveryCtx, t.metricLane, t.metrics != nil, func() {
		for _, emission := range metrics {
			t.metric(deliveryCtx, emission.name, emission.value, emission.unit, emission.labels)
		}
	})
	logDone := t.dispatch(deliveryCtx, t.logLane, t.logs != nil && event != nil, func() {
		t.deliverLog(deliveryCtx, event.level, event.message, event.fields)
	})
	waitForAdminExternalDelivery(deliveryCtx, metricDone)
	waitForAdminExternalDelivery(deliveryCtx, logDone)
}

// dispatch caps each sink at one active batch even when it ignores cancellation.
// Implements DESIGN-014 MetricsCollector and LogAggregator backpressure.
func (t *AdminExternalTelemetry) dispatch(ctx context.Context, lane chan struct{}, enabled bool, deliver func()) <-chan struct{} {
	if !enabled {
		return nil
	}
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

// waitForAdminExternalDelivery bounds the caller when a sink ignores context.
// Implements DESIGN-014 MetricsCollector and LogAggregator backpressure.
func waitForAdminExternalDelivery(ctx context.Context, done <-chan struct{}) {
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// metric emits one already-allowlisted point and ignores sink failure recursively.
// Implements DESIGN-014 MetricsCollector.
func (t *AdminExternalTelemetry) metric(ctx context.Context, name string, value float64, unit string, labels map[string]string) {
	if t != nil && t.metrics != nil {
		_ = t.metrics.RecordMetric(ctx, MetricPoint{Name: name, Value: value, Unit: unit, Labels: labels, ObservedAt: time.Now().UTC()})
	}
}

// deliverLog emits one already-allowlisted structured event.
// Implements DESIGN-014 LogAggregator.
func (t *AdminExternalTelemetry) deliverLog(ctx context.Context, level, message string, fields map[string]any) {
	if t != nil && t.logs != nil {
		_ = t.logs.Log(ctx, LogEvent{Service: "api", Level: level, Message: message, Fields: fields, CreatedAt: time.Now().UTC()})
	}
}

// boundedAdminExternalLog copies only fixed provider and curation metadata.
// Implements DESIGN-014 LogAggregator low-cardinality privacy policy.
func boundedAdminExternalLog(event LogEvent) (adminExternalLog, bool) {
	switch event.Message {
	case "external_provider_failure":
		provider, providerOK := event.Fields["provider"].(string)
		code, codeOK := event.Fields["code"].(string)
		status, statusOK := event.Fields["status"].(int)
		retryable, retryableOK := event.Fields["retryable"].(bool)
		if len(event.Fields) != 4 || !providerOK || !allowed(provider, "usda", "openfoodfacts") || !codeOK || !allowedProviderFailureCode(code) || !statusOK || status < 0 || status > 599 || !retryableOK {
			return adminExternalLog{}, false
		}
		return adminExternalLog{level: "warn", message: event.Message, fields: map[string]any{"provider": provider, "code": code, "status": status, "retryable": retryable}}, true
	case "external_provider_payload_dropped":
		provider, providerOK := event.Fields["provider"].(string)
		count, countOK := event.Fields["count"].(int)
		if len(event.Fields) != 2 || !providerOK || provider != "openfoodfacts" || !countOK || count < 1 || count > 100 {
			return adminExternalLog{}, false
		}
		return adminExternalLog{level: "warn", message: event.Message, fields: map[string]any{"provider": provider, "count": count}}, true
	case "curation_input_validation":
		field, fieldOK := event.Fields["field"].(string)
		outcome, outcomeOK := event.Fields["outcome"].(string)
		changed, changedOK := event.Fields["changed"].(bool)
		violationCount, countOK := event.Fields["violationCount"].(int)
		if len(event.Fields) != 4 || !fieldOK || !allowedCurationLogField(field) || !outcomeOK || !allowed(outcome, "normalized", "rejected") || !changedOK || !countOK || violationCount < 0 || violationCount > 64 {
			return adminExternalLog{}, false
		}
		return adminExternalLog{level: "info", message: event.Message, fields: map[string]any{"field": field, "outcome": outcome, "changed": changed, "violationCount": violationCount}}, true
	default:
		return adminExternalLog{}, false
	}
}

// allowedProviderFailureCode closes concrete-client diagnostics over provider-safe categories.
// Implements DESIGN-012 USDAClient and OpenFoodFactsClient safe provider diagnostics.
func allowedProviderFailureCode(code string) bool {
	return allowed(code, "invalid_input", "not_configured", "provider_rejected", "provider_rate_limited", "provider_unavailable", "invalid_external_payload", "provider_response_too_large", "timeout", "canceled")
}

// allowedCurationLogField closes validation diagnostics over metadata-only categories.
// Implements DESIGN-013 InputNormalizer metadata policy.
func allowedCurationLogField(field string) bool {
	return allowed(field, "curation_item_name", "curation_classification_name", "external_query", "external_provider", "curation_provider", "provider_identifier", "image_url", "serving_unit", "provider_text", "pagination", "provider_identity", "macros_per_100", "serving_quantity", "serving", "micronutrients", "physical_measures", "external_search_query", "curation_item_body", "curation_classification_body")
}

// allowed checks exact membership in a fixed local vocabulary.
// Implements DESIGN-014 MetricsCollector low-cardinality policy.
func allowed(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

// boundedOperation collapses unknown admin actions instead of emitting them.
// Implements DESIGN-014 MetricsCollector low-cardinality policy.
func boundedOperation(operation string) string {
	if allowed(operation, "import_food", "manual_create", "manual_update", "manual_delete", "classification.create", "classification.update", "classification.delete", "retry_deletion") {
		return operation
	}
	return "other"
}

// stringFields copies only allowlisted metric dimensions into structured logs.
// Implements DESIGN-014 LogAggregator.
func stringFields(labels map[string]string) map[string]any {
	fields := make(map[string]any, len(labels))
	for key, value := range labels {
		fields[key] = value
	}
	return fields
}

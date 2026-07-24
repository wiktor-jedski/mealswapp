package externaldata

// Implements DESIGN-012 provider orchestration and DESIGN-014 MetricsCollector Task 260 gate.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type task260Provider struct{ calls int }

func (p *task260Provider) SearchResult(context.Context, ExternalSearchQuery) (ProviderResult, error) {
	p.calls++
	if p.calls == 1 {
		return ProviderResult{Headers: http.Header{"X-Ratelimit-Remaining": []string{"2"}}}, &ProviderError{Code: ProviderErrorUnavailable, Retryable: true}
	}
	return ProviderResult{Headers: http.Header{"X-Ratelimit-Remaining": []string{"1"}}, Records: []ExternalFoodRecord{{Provider: "usda", ExternalID: "safe", Name: "Safe", Nutrients: map[string]float64{"Protein (G)": 1, "Carbohydrate, by difference (G)": 2, "Total lipid (fat) (G)": 3}}}}, nil
}

type task260AdversarialProvider struct {
	result ProviderResult
	err    error
}

type task260BlockingTelemetrySink struct{ release chan struct{} }

type task260BlockedJSONWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *task260BlockedJSONWriter) Write(payload []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	return len(payload), nil
}

func (s task260BlockingTelemetrySink) RecordMetric(context.Context, observability.MetricPoint) error {
	<-s.release
	return nil
}

func (s task260BlockingTelemetrySink) Log(context.Context, observability.LogEvent) error {
	<-s.release
	return nil
}

func (p task260AdversarialProvider) SearchResult(context.Context, ExternalSearchQuery) (ProviderResult, error) {
	return p.result, p.err
}

type task260Vocabulary struct{}

func (task260Vocabulary) ListActive(context.Context) ([]repository.MicronutrientVocabularyEntry, error) {
	return []repository.MicronutrientVocabularyEntry{}, nil
}
func (task260Vocabulary) IsAllowed(context.Context, string) (bool, error) { return false, nil }
func (task260Vocabulary) Upsert(context.Context, repository.MicronutrientVocabularyEntry) error {
	return nil
}

func TestTask260ProviderAndNormalizationTelemetryMatchesBehavior(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clock := func() time.Time { now = now.Add(10 * time.Millisecond); return now }
	sink := &observability.MemorySink{}
	telemetry := observability.NewAdminExternalTelemetry(sink, sink)
	limits := NewRateLimitHandler(clock, func(time.Duration) time.Duration { return 0 }).WithTelemetry(telemetry)
	if err := limits.Configure(time.Second, func(context.Context, time.Duration) error { return nil }); err != nil {
		t.Fatal(err)
	}
	provider := &task260Provider{}
	records, warnings, err := searchExternalRecords(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "private query never emitted", Page: 1, PageSize: 1}, ProviderSet{USDA: provider}, limits)
	if err != nil || provider.calls != 2 || len(records) != 1 || len(warnings) != 0 {
		t.Fatalf("calls=%d records=%d warnings=%v err=%v", provider.calls, len(records), warnings, err)
	}
	normalizer := NewDataNormalizer(task260Vocabulary{}).WithTelemetry(telemetry)
	if _, _, err := normalizer.NormalizeRecordsWithWarnings(context.Background(), append(records, ExternalFoodRecord{Provider: "attacker", Name: "private name"})); err != nil {
		t.Fatal(err)
	}
	metrics, _ := sink.Snapshot()
	want := map[string]int{observability.MetricExternalProviderCalls: 2, observability.MetricExternalProviderLatency: 2, observability.MetricExternalProviderRetries: 1, observability.MetricExternalProviderQuota: 2, observability.MetricExternalNormalization: 3}
	for _, point := range metrics {
		if _, ok := want[point.Name]; ok {
			want[point.Name]--
		}
	}
	for name, remaining := range want {
		if remaining != 0 {
			t.Fatalf("metric %s remaining=%d metrics=%+v", name, remaining, metrics)
		}
	}
	if providerTelemetryOutcome(errors.New("raw provider payload")) != "error" {
		t.Fatal("unknown provider error was not collapsed")
	}
}

func TestTask260UnknownProviderCodePreservesBoundedCallAndLatencyTelemetry(t *testing.T) {
	sink := &observability.MemorySink{}
	limits := NewRateLimitHandler(nil, nil).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))
	provider := task260AdversarialProvider{err: &ProviderError{Code: ProviderErrorCode("private-unknown-code"), cause: errors.New("private provider payload")}}

	_, warnings, err := searchExternalRecords(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "private query", Page: 1, PageSize: 1}, ProviderSet{USDA: provider}, limits)
	if err != nil || len(warnings) != 1 || warnings[0].Code != WarningUnavailable {
		t.Fatalf("warnings=%v err=%v", warnings, err)
	}
	metrics, logs := sink.Snapshot()
	for _, name := range []string{observability.MetricExternalProviderCalls, observability.MetricExternalProviderLatency} {
		if !task260HasProviderMetric(metrics, name, "error") {
			t.Fatalf("missing bounded %s metric: %+v", name, metrics)
		}
	}
	encoded, marshalErr := json.Marshal(struct{ Metrics, Logs any }{metrics, logs})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, forbidden := range []string{"private-unknown-code", "private provider payload", "private query"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("telemetry leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestTask260MalformedQuotaHeaderIsUnknownNotExhausted(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
	}{
		{name: "non-numeric remaining", headers: http.Header{"X-Ratelimit-Remaining": []string{"not-a-number"}}},
		{name: "negative remaining", headers: http.Header{"X-Ratelimit-Remaining": []string{"-1"}}},
		{name: "malformed reset", headers: http.Header{"X-Ratelimit-Remaining": []string{"0"}, "X-Ratelimit-Reset": []string{"not-a-timestamp"}}},
		{name: "reset without remaining", headers: http.Header{"X-Ratelimit-Reset": []string{"200"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &observability.MemorySink{}
			limits := NewRateLimitHandler(nil, nil).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))
			provider := task260AdversarialProvider{result: ProviderResult{Headers: tt.headers}}
			_, warnings, err := searchExternalRecords(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: provider}, limits)
			if err != nil || len(warnings) != 0 {
				t.Fatalf("warnings=%v err=%v", warnings, err)
			}
			metrics, _ := sink.Snapshot()
			if !task260HasQuotaMetric(metrics, "unknown") || task260HasQuotaMetric(metrics, "exhausted") {
				t.Fatalf("malformed quota classification metrics=%+v", metrics)
			}
		})
	}
}

func TestTask260BlockingTelemetrySinkCannotHoldProviderRequest(t *testing.T) {
	sink := task260BlockingTelemetrySink{release: make(chan struct{})}
	defer close(sink.release)
	limits := NewRateLimitHandler(nil, nil).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))
	provider := task260AdversarialProvider{result: ProviderResult{Headers: http.Header{"X-Ratelimit-Remaining": []string{"1"}}}}
	done := make(chan error, 1)
	go func() {
		_, _, err := searchExternalRecords(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: provider}, limits)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(750 * time.Millisecond):
		t.Fatal("blocking telemetry sink held the provider request")
	}
}

func TestTask260ConcreteProviderFailureCannotBlockRequestOnJSONWriter(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer provider.Close()
	writer := &task260BlockedJSONWriter{started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { close(writer.release) })
	telemetry := observability.NewAdminExternalTelemetry(nil, observability.JSONSink{Writer: writer})
	client, err := NewUSDAClient(USDAConfig{APIKey: "private-provider-key", Endpoint: provider.URL, HTTPClient: provider.Client(), Logs: telemetry})
	if err != nil {
		t.Fatal(err)
	}

	completed := make(chan error, 1)
	go func() {
		_, searchErr := client.Search(context.Background(), ExternalSearchQuery{Query: "private provider query", Provider: "usda", Page: 1, PageSize: 1})
		completed <- searchErr
	}()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("concrete provider failure did not reach JSON writer")
	}
	select {
	case searchErr := <-completed:
		var providerErr *ProviderError
		if !errors.As(searchErr, &providerErr) || providerErr.Code != ProviderErrorUnavailable {
			t.Fatalf("provider error = %v, want unavailable", searchErr)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked JSON writer held concrete provider request")
	}
}

func task260HasProviderMetric(metrics []observability.MetricPoint, name, outcome string) bool {
	for _, point := range metrics {
		if point.Name == name && point.Labels["provider"] == "usda" && point.Labels["outcome"] == outcome {
			return true
		}
	}
	return false
}

func task260HasQuotaMetric(metrics []observability.MetricPoint, state string) bool {
	for _, point := range metrics {
		if point.Name == observability.MetricExternalProviderQuota && point.Labels["provider"] == "usda" && point.Labels["state"] == state {
			return true
		}
	}
	return false
}

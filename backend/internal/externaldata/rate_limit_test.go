// Implements DESIGN-012 RateLimitHandler deterministic verification.
package externaldata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeProvider struct {
	mu      sync.Mutex
	results []ProviderResult
	errs    []error
	calls   int
	headers http.Header
}

func (f *fakeProvider) SearchResult(ctx context.Context, _ ExternalSearchQuery) (ProviderResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(f.errs) > 0 {
		e := f.errs[0]
		f.errs = f.errs[1:]
		return ProviderResult{}, e
	}
	if len(f.results) > 0 {
		r := f.results[0]
		f.results = f.results[1:]
		return r, nil
	}
	return ProviderResult{}, nil
}
func (f *fakeProvider) count() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }

type blockingProvider struct{}

func (blockingProvider) SearchResult(ctx context.Context, _ ExternalSearchQuery) (ProviderResult, error) {
	<-ctx.Done()
	return ProviderResult{}, &ProviderError{Code: ProviderErrorTimeout, Retryable: true, cause: ctx.Err(), provider: "usda"}
}

type resultErrorProvider struct {
	result ProviderResult
	err    error
	calls  int
}

func (p *resultErrorProvider) SearchResult(context.Context, ExternalSearchQuery) (ProviderResult, error) {
	p.calls++
	return p.result, p.err
}

type cancelingProvider struct {
	started chan struct{}
	calls   int
}

func (p *cancelingProvider) SearchResult(ctx context.Context, _ ExternalSearchQuery) (ProviderResult, error) {
	p.calls++
	close(p.started)
	<-ctx.Done()
	return ProviderResult{}, &ProviderError{Code: ProviderErrorCanceled, cause: ctx.Err(), provider: "usda"}
}

func TestSearchExternalFoodsRecordsErrorHeadersAndSkipsUntilReset(t *testing.T) {
	now := time.Unix(100, 0)
	h := NewRateLimitHandler(func() time.Time { return now }, func(time.Duration) time.Duration { return 0 })
	headers := make(http.Header)
	headers.Set("X-RateLimit-Remaining", "0")
	headers.Set("X-RateLimit-Reset", "200")
	provider := &resultErrorProvider{result: ProviderResult{Headers: headers}, err: &ProviderError{Code: ProviderErrorRateLimited, Retryable: false, provider: "usda"}}
	query := ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}

	_, warnings, err := SearchExternalFoods(context.Background(), query, ProviderSet{USDA: provider}, h)
	if err != nil || provider.calls != 1 || len(warnings) != 1 || warnings[0].Code != WarningRateLimited {
		t.Fatalf("first call calls=%d warnings=%v err=%v", provider.calls, warnings, err)
	}
	_, warnings, err = SearchExternalFoods(context.Background(), query, ProviderSet{USDA: provider}, h)
	if err != nil || provider.calls != 1 || len(warnings) != 1 || warnings[0].Code != WarningRateLimited {
		t.Fatalf("blocked call calls=%d warnings=%v err=%v", provider.calls, warnings, err)
	}
}

func TestSearchExternalFoodsPropagatesRealProviderHeadersOnErrorAndSuccess(t *testing.T) {
	now := time.Unix(100, 0)
	usdaCalls, openFoodFactsCalls := 0, 0
	usdaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		usdaCalls++
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "200")
		w.Header().Set("X-Provider-Secret", "discard-me")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer usdaServer.Close()
	openFoodFactsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		openFoodFactsCalls++
		w.Header().Set("X-RateLimit-Remaining", "7")
		w.Header().Set("X-Provider-Secret", "discard-me")
		_, _ = w.Write([]byte(validOpenFoodFactsPayload))
	}))
	defer openFoodFactsServer.Close()

	providers := ProviderSet{
		USDA:          newTestUSDAClient(t, usdaServer.URL, nil, 0, 0),
		OpenFoodFacts: newTestOpenFoodFactsClient(t, openFoodFactsServer.URL, nil, 0, 0),
	}
	h := NewRateLimitHandler(func() time.Time { return now }, func(time.Duration) time.Duration { return 0 })
	query := ExternalSearchQuery{Provider: "all", Query: "apple", Page: 1, PageSize: 1}
	items, warnings, err := SearchExternalFoods(context.Background(), query, providers, h)
	if err != nil || len(items) != 1 || len(warnings) != 1 || warnings[0].Code != WarningRateLimited || usdaCalls != 1 || openFoodFactsCalls != 1 {
		t.Fatalf("first items=%v warnings=%v calls=%d/%d err=%v", items, warnings, usdaCalls, openFoodFactsCalls, err)
	}
	if usda := h.CheckRateLimit("usda"); usda.Remaining != 0 || !usda.ResetAt.Equal(time.Unix(200, 0)) {
		t.Fatalf("USDA state=%+v", usda)
	}
	if off := h.CheckRateLimit("openfoodfacts"); off.Remaining != 7 || !off.ResetAt.IsZero() {
		t.Fatalf("OpenFoodFacts state=%+v", off)
	}
	_, _, err = SearchExternalFoods(context.Background(), query, providers, h)
	if err != nil || usdaCalls != 1 || openFoodFactsCalls != 2 {
		t.Fatalf("second calls=%d/%d err=%v", usdaCalls, openFoodFactsCalls, err)
	}
	result, err := providers.USDA.SearchResult(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1})
	if err == nil || len(result.Headers) != 2 || result.Headers.Get("X-Provider-Secret") != "" {
		t.Fatalf("bounded error headers=%v err=%v", result.Headers, err)
	}
}

func TestSearchExternalFoodsPreservesInFlightCallerCancellation(t *testing.T) {
	provider := &cancelingProvider{started: make(chan struct{})}
	h := NewRateLimitHandler(nil, nil)
	var sleeps int
	if err := h.Configure(time.Second, func(context.Context, time.Duration) error { sleeps++; return nil }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, err := SearchExternalFoods(ctx, ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: provider}, h)
		done <- err
	}()
	<-provider.started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) || provider.calls != 1 || sleeps != 0 {
		t.Fatalf("err=%v calls=%d sleeps=%d", err, provider.calls, sleeps)
	}
}

func TestSearchExternalFoodsPreservesInFlightCallerDeadline(t *testing.T) {
	provider := &cancelingProvider{started: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := SearchExternalFoods(ctx, ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: provider}, nil)
	if !errors.Is(err, context.DeadlineExceeded) || provider.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, provider.calls)
	}
}

func TestSearchExternalFoodsRejectsInvalidInputWithoutProviderCalls(t *testing.T) {
	provider := &fakeProvider{}
	tests := []ExternalSearchQuery{
		{Provider: "unknown", Query: "apple", Page: 1, PageSize: 1},
		{Provider: "usda", Query: "", Page: 1, PageSize: 1},
		{Provider: "usda", Query: "apple", Page: 0, PageSize: 1},
		{Provider: "usda", Query: "apple", Page: 1, PageSize: 201},
		{Provider: "all", Query: "apple", Page: 1, PageSize: 101},
	}
	for _, query := range tests {
		_, _, err := SearchExternalFoods(context.Background(), query, ProviderSet{USDA: provider, OpenFoodFacts: provider}, nil)
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) || providerErr.Code != ProviderErrorInvalidInput {
			t.Fatalf("query=%+v err=%v", query, err)
		}
	}
	if provider.count() != 0 {
		t.Fatalf("invalid input made %d provider calls", provider.count())
	}
}

func TestSearchExternalFoodsReportsMissingSelectedProviders(t *testing.T) {
	ok := &fakeProvider{results: []ProviderResult{{Records: []ExternalFoodRecord{{Provider: "usda", ExternalID: "1"}}}}}
	query := ExternalSearchQuery{Provider: "all", Query: "apple", Page: 1, PageSize: 1}
	items, warnings, err := SearchExternalFoods(context.Background(), query, ProviderSet{USDA: ok}, nil)
	if err != nil || len(items) != 1 || len(warnings) != 1 || warnings[0].Provider != "openfoodfacts" || warnings[0].Code != WarningUnavailable {
		t.Fatalf("partial items=%v warnings=%v err=%v", items, warnings, err)
	}
	items, warnings, err = SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{}, nil)
	if err != nil || len(items) != 0 || len(warnings) != 1 || warnings[0].Provider != "usda" || warnings[0].Code != WarningUnavailable {
		t.Fatalf("missing items=%v warnings=%v err=%v", items, warnings, err)
	}
	items, warnings, err = SearchExternalFoods(context.Background(), query, ProviderSet{}, nil)
	if err != nil || len(items) != 0 || len(warnings) != 2 || warnings[0].Code != WarningUnavailable || warnings[1].Code != WarningUnavailable {
		t.Fatalf("both missing items=%v warnings=%v err=%v", items, warnings, err)
	}
}

func TestRateLimitHandlerAdversarialBranches(t *testing.T) {
	h := NewRateLimitHandler(nil, nil)
	if jitter := h.jitter(time.Nanosecond); jitter < 0 || jitter > time.Nanosecond {
		t.Fatalf("default jitter=%v", jitter)
	}
	for _, deadline := range []time.Duration{0, -time.Second, time.Minute + time.Nanosecond} {
		if err := h.Configure(deadline, nil); err == nil {
			t.Fatalf("deadline %v accepted", deadline)
		}
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := contextSleep(canceled, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleep cancellation=%v", err)
	}
	if err := h.RecordRateLimit(" ", nil); err == nil {
		t.Fatal("blank provider accepted")
	}

	off := &fakeProvider{results: []ProviderResult{{Records: []ExternalFoodRecord{{Provider: "openfoodfacts", ExternalID: "1"}}}}}
	items, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "openfoodfacts", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{OpenFoodFacts: off}, h)
	if err != nil || len(items) != 1 || len(warnings) != 0 {
		t.Fatalf("OpenFoodFacts selection items=%v warnings=%v err=%v", items, warnings, err)
	}

	sleepErr := errors.New("sleep interrupted")
	retrying := &fakeProvider{errs: []error{&ProviderError{Code: ProviderErrorUnavailable, Retryable: true, provider: "usda"}}}
	if err := h.Configure(time.Second, func(context.Context, time.Duration) error { return sleepErr }); err != nil {
		t.Fatal(err)
	}
	_, _, err = SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: retrying}, h)
	if !errors.Is(err, sleepErr) || retrying.count() != 1 {
		t.Fatalf("sleep err=%v calls=%d", err, retrying.count())
	}

	providerCanceled := &resultErrorProvider{err: context.Canceled}
	_, _, err = SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: providerCanceled}, NewRateLimitHandler(nil, nil))
	if !errors.Is(err, context.Canceled) || providerCanceled.calls != 1 {
		t.Fatalf("provider cancellation=%v calls=%d", err, providerCanceled.calls)
	}
}

func TestSearchExternalFoodsUsesConfiguredDeadlineAndConcurrentProviderIsolation(t *testing.T) {
	h := NewRateLimitHandler(time.Now, func(time.Duration) time.Duration { return 0 })
	if err := h.Configure(2*time.Millisecond, nil); err != nil {
		t.Fatal(err)
	}
	_, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: blockingProvider{}}, h)
	if err != nil || len(warnings) != 1 || warnings[0].Code != WarningTimeout {
		t.Fatalf("deadline warnings=%v err=%v", warnings, err)
	}
	var wg sync.WaitGroup
	for _, provider := range []string{"usda", "openfoodfacts"} {
		provider := provider
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = h.RecordRateLimit(provider, http.Header{"X-Ratelimit-Remaining": []string{"7"}})
				_ = h.CheckRateLimit(provider)
			}
		}()
	}
	wg.Wait()
	if h.CheckRateLimit("usda").Provider != "usda" || h.CheckRateLimit("openfoodfacts").Provider != "openfoodfacts" {
		t.Fatal("provider state crossed boundaries")
	}
}

// TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset verifies
// IT-ARCH-012-003, ARCH-012, DESIGN-012 RateLimitHandler, and SW-REQ-055.
func TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset(t *testing.T) {
	now := time.Unix(100, 0)
	h := NewRateLimitHandler(func() time.Time { return now }, func(time.Duration) time.Duration { return 0 })
	reset := make(http.Header)
	reset.Set("X-RateLimit-Remaining", "0")
	reset.Set("X-RateLimit-Reset", "200")
	if err := h.RecordRateLimit("usda", reset); err != nil {
		t.Fatal(err)
	}
	fake := &fakeProvider{results: []ProviderResult{{Records: []ExternalFoodRecord{{Provider: "usda", ExternalID: "1"}}}}}
	_, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: fake}, h)
	if err != nil || fake.count() != 0 || len(warnings) != 1 || warnings[0].Code != WarningRateLimited {
		t.Fatalf("before reset calls=%d warnings=%v err=%v", fake.count(), warnings, err)
	}
	now = time.Unix(200, 0)
	_, warnings, err = SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: fake}, h)
	if err != nil || fake.count() != 1 || len(warnings) != 0 {
		t.Fatalf("after reset calls=%d warnings=%v err=%v", fake.count(), warnings, err)
	}
}

func TestSearchExternalFoodsNonRetryableUnavailableIsWarningAndSingleCall(t *testing.T) {
	fake := &fakeProvider{errs: []error{&ProviderError{Code: ProviderErrorUnavailable, Retryable: false, provider: "usda", cause: errors.New("secret provider payload")}}}
	items, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: fake}, NewRateLimitHandler(nil, nil))
	if err != nil || len(items) != 0 || fake.count() != 1 || len(warnings) != 1 || warnings[0].Code != WarningUnavailable {
		t.Fatalf("items=%v calls=%d warnings=%v err=%v", items, fake.count(), warnings, err)
	}
}

func TestSearchExternalFoodsEmitsNoTelemetryAndDoesNotLeakPayloadOrSecrets(t *testing.T) {
	secret := "provider-api-secret"
	payload := `{"name":"raw-provider-payload"}`
	fake := &fakeProvider{errs: []error{&ProviderError{Code: ProviderErrorUnavailable, Retryable: false, provider: "usda", cause: errors.New(secret + " " + payload)}}}
	items, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: fake}, NewRateLimitHandler(nil, nil))
	if err != nil || len(items) != 0 || len(warnings) != 1 {
		t.Fatalf("items=%v warnings=%v err=%v", items, warnings, err)
	}
	for _, value := range []string{secret, payload} {
		if warnings[0].Message == value || strings.Contains(warnings[0].Message, value) {
			t.Fatalf("sensitive value leaked: %q", value)
		}
	}
	if warnings[0].Message != WarningUnavailable {
		t.Fatalf("unbounded warning message=%q", warnings[0].Message)
	}
}

func TestRateLimitHandlerDeterministicRetryDeadlineAndHeaderIsolation(t *testing.T) {
	now := time.Unix(100, 0)
	var sleeps []time.Duration
	h := NewRateLimitHandler(func() time.Time { return now }, func(d time.Duration) time.Duration { return d / 2 })
	if err := h.Configure(time.Second, func(ctx context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		now = now.Add(d)
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	fail := &fakeProvider{errs: []error{
		&ProviderError{Code: ProviderErrorUnavailable, Retryable: true, provider: "usda"},
		&ProviderError{Code: ProviderErrorUnavailable, Retryable: true, provider: "usda"},
		&ProviderError{Code: ProviderErrorUnavailable, Retryable: true, provider: "usda"},
		&ProviderError{Code: ProviderErrorUnavailable, Retryable: true, provider: "usda"},
	}}
	_, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: fail}, h)
	if err != nil || fail.count() != 4 || len(sleeps) != 3 || warnings[0].Code != WarningRetryExhausted {
		t.Fatalf("calls=%d sleeps=%v warnings=%v err=%v", fail.count(), sleeps, warnings, err)
	}
	if sleeps[0] != 150*time.Millisecond || sleeps[1] != 300*time.Millisecond || sleeps[2] != 600*time.Millisecond {
		t.Fatalf("backoff=%v", sleeps)
	}
	if err := h.RecordRateLimit("usda", http.Header{"X-RateLimit-Remaining": []string{"0"}, "X-RateLimit-Reset": []string{"200"}}); err != nil {
		t.Fatal(err)
	}
	if h.CheckRateLimit("usda").Remaining != 0 || h.CheckRateLimit("openfoodfacts").Remaining != 0 {
		t.Fatalf("provider state leaked: %#v", h.CheckRateLimit("openfoodfacts"))
	}
	now = time.Unix(201, 0)
	if h.CheckRateLimit("usda").Remaining != 0 {
		t.Fatal("state unexpectedly reset")
	}
}

// TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings verifies
// IT-ARCH-012-002 and IT-ARCH-012-003, ARCH-012, DESIGN-012
// RateLimitHandler, and SW-REQ-055.
func TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings(t *testing.T) {
	clock := time.Unix(100, 0)
	h := NewRateLimitHandler(func() time.Time { return clock }, func(time.Duration) time.Duration { return 0 })
	_ = h.Configure(time.Millisecond, func(context.Context, time.Duration) error { return nil })
	headers := make(http.Header)
	headers.Set("X-RateLimit-Remaining", "7")
	ok := &fakeProvider{results: []ProviderResult{{Records: []ExternalFoodRecord{{Provider: "usda", ExternalID: "1", Name: "Apple", RawPayload: []byte("SECRET")}}, Headers: headers}}}
	limited := &fakeProvider{errs: []error{&ProviderError{Code: ProviderErrorRateLimited, Retryable: true, provider: "openfoodfacts"}, &ProviderError{Code: ProviderErrorRateLimited, Retryable: true, provider: "openfoodfacts"}, &ProviderError{Code: ProviderErrorRateLimited, Retryable: true, provider: "openfoodfacts"}, &ProviderError{Code: ProviderErrorRateLimited, Retryable: true, provider: "openfoodfacts"}}}
	items, warnings, err := SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "all", Query: "apple", Page: 1, PageSize: 1}, ProviderSet{USDA: ok, OpenFoodFacts: limited}, h)
	if err != nil || len(items) != 1 || len(warnings) != 1 || warnings[0].Code != WarningRateLimited {
		t.Fatalf("partial result items=%v warnings=%v err=%v", items, warnings, err)
	}
	for _, w := range warnings {
		if w.Code != WarningRateLimited && w.Code != WarningUnavailable && w.Code != WarningTimeout && w.Code != WarningRetryExhausted {
			t.Fatal("unbounded warning")
		}
		if w.Message == "SECRET" {
			t.Fatal("secret leaked")
		}
	}
	if h.CheckRateLimit("usda").Remaining != 7 {
		t.Fatal("response headers not recorded")
	}
	permanent := &fakeProvider{errs: []error{&ProviderError{Code: ProviderErrorRejected, Retryable: false, provider: "usda"}}}
	_, _, _ = SearchExternalFoods(context.Background(), ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: permanent}, h)
	if permanent.count() != 1 {
		t.Fatal("permanent failure retried")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = SearchExternalFoods(ctx, ExternalSearchQuery{Provider: "usda", Query: "x", Page: 1, PageSize: 1}, ProviderSet{USDA: ok}, h)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel err=%v", err)
	}
}

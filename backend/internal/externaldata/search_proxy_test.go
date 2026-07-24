package externaldata

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 ExternalSearchProxy provider selection, normalization, ordering, outage, cancellation, and bounds verification.

type proxyVocabulary struct {
	calls         int
	mutationCalls int
	err           error
}

func (v *proxyVocabulary) ListActive(context.Context) ([]repository.MicronutrientVocabularyEntry, error) {
	v.calls++
	return nil, v.err
}

func (*proxyVocabulary) IsAllowed(context.Context, string) (bool, error) { return false, nil }
func (v *proxyVocabulary) Upsert(context.Context, repository.MicronutrientVocabularyEntry) error {
	v.mutationCalls++
	return errors.New("read-only test vocabulary")
}

type proxyProvider struct {
	mu      sync.Mutex
	queries []ExternalSearchQuery
	result  ProviderResult
	err     error
	search  func(context.Context) error
}

func (p *proxyProvider) SearchResult(ctx context.Context, query ExternalSearchQuery) (ProviderResult, error) {
	p.mu.Lock()
	p.queries = append(p.queries, query)
	p.mu.Unlock()
	if p.search != nil {
		return ProviderResult{}, p.search(ctx)
	}
	return p.result, p.err
}

// TestExternalSearchProxySelectsProviderAndPaginationAndMergesDeterministically
// verifies IT-ARCH-012-001, ARCH-012, DESIGN-012 USDAClient,
// OpenFoodFactsClient/DataNormalizer/RateLimitHandler, and SW-REQ-055/SW-REQ-090.
func TestExternalSearchProxySelectsProviderAndPaginationAndMergesDeterministically(t *testing.T) {
	vocabulary := &proxyVocabulary{}
	usda := &proxyProvider{result: ProviderResult{Records: []ExternalFoodRecord{
		proxyRecord("usda", "2", "banana"), proxyRecord("usda", "1", "Apple"), proxyRecord("usda", "0", "apple"),
	}}}
	off := &proxyProvider{result: ProviderResult{Records: []ExternalFoodRecord{
		proxyRecord("openfoodfacts", "3", "apple"),
	}}}
	proxy := NewExternalSearchProxy(ProviderSet{USDA: usda, OpenFoodFacts: off}, NewRateLimitHandler(nil, func(time.Duration) time.Duration { return 0 }), NewDataNormalizer(vocabulary))

	response, err := proxy.Search(context.Background(), ExternalSearchQuery{Query: "fruit", Provider: "all", Page: 3})
	if err != nil {
		t.Fatal(err)
	}
	if vocabulary.calls != 1 || len(usda.queries) != 1 || len(off.queries) != 1 {
		t.Fatalf("workflow calls vocabulary=%d usda=%d off=%d", vocabulary.calls, len(usda.queries), len(off.queries))
	}
	for _, query := range []ExternalSearchQuery{usda.queries[0], off.queries[0]} {
		if query.Page != 3 || query.PageSize != DefaultExternalSearchPageSize {
			t.Fatalf("provider query = %#v", query)
		}
	}
	got := []string{response.Candidates[0].Provider + ":" + response.Candidates[0].ExternalID, response.Candidates[1].Provider + ":" + response.Candidates[1].ExternalID, response.Candidates[2].Provider + ":" + response.Candidates[2].ExternalID, response.Candidates[3].Provider + ":" + response.Candidates[3].ExternalID}
	want := []string{"openfoodfacts:3", "usda:0", "usda:1", "usda:2"}
	if !reflect.DeepEqual(got, want) || response.Page != 3 {
		t.Fatalf("ordered response = %#v", response)
	}

	offOnly, err := proxy.Search(context.Background(), ExternalSearchQuery{Query: "fruit", Provider: "openfoodfacts", Page: 2})
	if err != nil || len(offOnly.Candidates) != 1 || len(usda.queries) != 1 || len(off.queries) != 2 {
		t.Fatalf("provider selection response=%#v err=%v calls=%d/%d", offOnly, err, len(usda.queries), len(off.queries))
	}
}

// TestExternalSearchProxyPartialAndCompleteOutageWarnings verifies
// IT-ARCH-012-002 and IT-ARCH-012-003, ARCH-012, DESIGN-012
// RateLimitHandler/DataNormalizer, and SW-REQ-055.
func TestExternalSearchProxyPartialAndCompleteOutageWarnings(t *testing.T) {
	ok := &proxyProvider{result: ProviderResult{Records: []ExternalFoodRecord{proxyRecord("usda", "1", "Apple")}}}
	down := &proxyProvider{err: &ProviderError{Code: ProviderErrorUnavailable}}
	proxy := NewExternalSearchProxy(ProviderSet{USDA: ok, OpenFoodFacts: down}, NewRateLimitHandler(nil, nil), NewDataNormalizer(&proxyVocabulary{}))
	response, err := proxy.Search(context.Background(), ExternalSearchQuery{Query: "apple", Provider: "all", Page: 1})
	if err != nil || len(response.Candidates) != 1 || len(response.Warnings) != 1 || response.Warnings[0].Provider != "openfoodfacts" || response.Warnings[0].Code != WarningUnavailable {
		t.Fatalf("partial response=%#v err=%v", response, err)
	}

	proxy = NewExternalSearchProxy(ProviderSet{}, NewRateLimitHandler(nil, nil), NewDataNormalizer(&proxyVocabulary{}))
	response, err = proxy.Search(context.Background(), ExternalSearchQuery{Query: "apple", Provider: "all", Page: 1})
	if err != nil || len(response.Candidates) != 0 || len(response.Warnings) != 2 {
		t.Fatalf("outage response=%#v err=%v", response, err)
	}
}

// TestExternalSearchProxyPropagatesCancellation verifies IT-ARCH-012-003,
// ARCH-012, DESIGN-012 RateLimitHandler, and SW-REQ-055.
func TestExternalSearchProxyPropagatesCancellation(t *testing.T) {
	started := make(chan struct{})
	provider := &proxyProvider{search: func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}
	proxy := NewExternalSearchProxy(ProviderSet{USDA: provider}, NewRateLimitHandler(nil, nil), NewDataNormalizer(&proxyVocabulary{}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := proxy.Search(ctx, ExternalSearchQuery{Query: "apple", Provider: "usda", Page: 1})
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestExternalSearchProxyDropsUnsafeFieldsAndExposesNoRawPayload(t *testing.T) {
	vocabulary := &proxyVocabulary{}
	record := proxyRecord("usda", "safe", strings.Repeat("x", 1001))
	record.RawPayload = []byte(`{"secret":"must-not-escape"}`)
	provider := &proxyProvider{result: ProviderResult{Records: []ExternalFoodRecord{record}}}
	proxy := NewExternalSearchProxy(ProviderSet{USDA: provider}, NewRateLimitHandler(nil, nil), NewDataNormalizer(vocabulary))
	response, err := proxy.Search(context.Background(), ExternalSearchQuery{Query: "apple", Provider: "usda", Page: 1})
	if err != nil || len(response.Candidates) != 0 || len(response.Warnings) != 1 || response.Warnings[0].Code != string(ProviderErrorInvalidPayload) {
		t.Fatalf("bounded response=%#v err=%v", response, err)
	}
	encoded := strings.ToLower(strings.TrimSpace(response.Warnings[0].Message + response.Warnings[0].Provider))
	if strings.Contains(encoded, "secret") || vocabulary.mutationCalls != 0 {
		t.Fatalf("unsafe or mutating proxy response = %#v", response)
	}
}

func TestExternalSearchProxyBoundsProviderCardinalityAndDeduplicatesWarnings(t *testing.T) {
	records := make([]ExternalFoodRecord, 0, DefaultExternalSearchPageSize+5)
	for index := 0; index < DefaultExternalSearchPageSize+5; index++ {
		records = append(records, proxyRecord("usda", string(rune('a'+index)), strings.Repeat("x", 1001)))
	}
	provider := &proxyProvider{result: ProviderResult{Records: records}}
	proxy := NewExternalSearchProxy(ProviderSet{USDA: provider}, NewRateLimitHandler(nil, nil), NewDataNormalizer(&proxyVocabulary{}))
	response, err := proxy.Search(context.Background(), ExternalSearchQuery{Query: "apple", Provider: "usda", Page: 1})
	if err != nil || len(response.Candidates) != 0 || len(response.Warnings) != 1 || response.Warnings[0].Code != string(ProviderErrorInvalidPayload) {
		t.Fatalf("bounded response=%#v err=%v", response, err)
	}
	if got := sortedUniqueStrings([]string{"b", "a", "b"}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("candidate warning canonicalization = %#v", got)
	}
	if got := sortedUniqueStrings(nil); len(got) != 0 || got == nil {
		t.Fatalf("empty candidate warnings = %#v", got)
	}
}

func TestNormalizeRecordsWithWarningsRejectsCompositionAndBoundsUnknownProvider(t *testing.T) {
	if _, _, err := (*DataNormalizer)(nil).NormalizeRecordsWithWarnings(context.Background(), nil); err == nil {
		t.Fatal("nil normalizer accepted")
	}
	normalizer := NewDataNormalizer(&proxyVocabulary{})
	if _, _, err := normalizer.NormalizeRecordsWithWarnings(nil, nil); err == nil {
		t.Fatal("nil context accepted")
	}
	candidates, warnings, err := normalizer.NormalizeRecordsWithWarnings(context.Background(), []ExternalFoodRecord{{Provider: "attacker", Name: strings.Repeat("x", 1001), Nutrients: map[string]float64{}}})
	if err != nil || len(candidates) != 0 || len(warnings) != 1 || warnings[0].Provider != "external" {
		t.Fatalf("unknown provider response candidates=%#v warnings=%#v err=%v", candidates, warnings, err)
	}
}

func TestExternalSearchProxyRejectsInvalidCompositionAndPropagatesWorkflowErrors(t *testing.T) {
	query := ExternalSearchQuery{Query: "apple", Provider: "usda", Page: 1}
	if _, err := (*ExternalSearchProxy)(nil).Search(context.Background(), query); err == nil {
		t.Fatal("nil proxy accepted")
	}
	proxy := NewExternalSearchProxy(ProviderSet{}, nil, NewDataNormalizer(&proxyVocabulary{}))
	if _, err := proxy.Search(nil, query); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := proxy.Search(context.Background(), ExternalSearchQuery{Query: "", Provider: "usda", Page: 1}); err == nil {
		t.Fatal("invalid query accepted")
	}
	want := errors.New("vocabulary unavailable")
	proxy = NewExternalSearchProxy(ProviderSet{USDA: &proxyProvider{}}, nil, NewDataNormalizer(&proxyVocabulary{err: want}))
	if _, err := proxy.Search(context.Background(), query); !errors.Is(err, want) {
		t.Fatalf("vocabulary error = %v", err)
	}
}

func proxyRecord(provider, id, name string) ExternalFoodRecord {
	nutrients := map[string]float64{"proteins_100g": 1, "carbohydrates_100g": 2, "fat_100g": 3}
	if provider == "usda" {
		nutrients = map[string]float64{"Protein (G)": 1, "Carbohydrate, by difference (G)": 2, "Total lipid (fat) (G)": 3}
	}
	return ExternalFoodRecord{Provider: provider, ExternalID: id, Name: name, Nutrients: nutrients}
}

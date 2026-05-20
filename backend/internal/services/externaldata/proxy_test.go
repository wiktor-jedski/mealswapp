package externaldata

import (
	"context"
	"errors"
	"testing"
)

func TestExternalSearchProxySearchesSelectedProviderAndNormalizes(t *testing.T) {
	usda := &fakeExternalClient{records: []ExternalFoodRecord{{
		Provider:   ProviderUSDA,
		ExternalID: "1101",
		Name:       "Cheddar",
		Nutrients: map[string]float64{
			"Protein":                     24.9,
			"Carbohydrate, by difference": 1.3,
			"Total lipid (fat)":           33.1,
			"Calories":                    403,
		},
	}}}
	proxy := NewExternalSearchProxy(usda, nil, testVocabulary())

	result, err := proxy.SearchExternalFoods(context.Background(), ExternalSearchQuery{Query: "cheddar", Provider: ProviderUSDA, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected proxy error: %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Provider != ProviderUSDA || result.Candidates[0].Name != "Cheddar" {
		t.Fatalf("unexpected candidates: %#v", result.Candidates)
	}
	if usda.query.Provider != ProviderUSDA || usda.query.Query != "cheddar" {
		t.Fatalf("unexpected provider query: %#v", usda.query)
	}
}

func TestExternalSearchProxyAllReturnsPartialSuccessWarnings(t *testing.T) {
	usda := &fakeExternalClient{err: ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorUnavailable, Message: "USDA unavailable", Retryable: true}}
	off := &fakeExternalClient{records: []ExternalFoodRecord{{
		Provider:   ProviderOpenFoodFacts,
		ExternalID: "737628064502",
		Name:       "Tofu",
		Nutrients: map[string]float64{
			"proteins_100g":      12,
			"carbohydrates_100g": 2,
			"fat_100g":           6,
		},
	}}}
	proxy := NewExternalSearchProxy(usda, off, testVocabulary())

	result, err := proxy.SearchExternalFoods(context.Background(), ExternalSearchQuery{Query: "tofu", Provider: ProviderAll, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("expected partial success, got %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Provider != ProviderOpenFoodFacts {
		t.Fatalf("unexpected candidates: %#v", result.Candidates)
	}
	if !hasWarning(result.Warnings, string(ProviderErrorUnavailable)) {
		t.Fatalf("expected unavailable warning, got %#v", result.Warnings)
	}
}

func TestExternalSearchProxyPropagatesSingleProviderRateLimit(t *testing.T) {
	proxy := NewExternalSearchProxy(&fakeExternalClient{err: ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorRateLimited, Message: "USDA rate limited", Retryable: true}}, nil, testVocabulary())

	_, err := proxy.SearchExternalFoods(context.Background(), ExternalSearchQuery{Query: "tofu", Provider: ProviderUSDA, Page: 1, PageSize: 10})

	var providerErr ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != ProviderErrorRateLimited {
		t.Fatalf("expected rate limit provider error, got %v", err)
	}
}

func TestExternalSearchProxyRejectsBadProvider(t *testing.T) {
	proxy := NewExternalSearchProxy(nil, nil, testVocabulary())

	_, err := proxy.SearchExternalFoods(context.Background(), ExternalSearchQuery{Query: "tofu", Provider: "bad", Page: 1, PageSize: 10})

	var providerErr ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != ProviderErrorInvalidQuery {
		t.Fatalf("expected invalid query provider error, got %v", err)
	}
}

type fakeExternalClient struct {
	query   ExternalSearchQuery
	records []ExternalFoodRecord
	err     error
}

func (client *fakeExternalClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error) {
	client.query = query
	if client.err != nil {
		return nil, client.err
	}
	return client.records, nil
}

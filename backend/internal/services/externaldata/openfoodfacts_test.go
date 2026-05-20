package externaldata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenFoodFactsClientBuildsQueryAndParsesProducts(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		if r.URL.Path != "/cgi/search.pl" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("expected json accept header")
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "MealSwapp") {
			t.Fatalf("expected MealSwapp user agent")
		}
		w.Header().Set("X-RateLimit-Remaining", "17")
		w.Header().Set("X-RateLimit-Reset", "1770000100")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{{
				"code":             "737628064502",
				"product_name":     "Organic Tofu",
				"serving_quantity": 85,
				"serving_size":     "85 g",
				"image_url":        "https://images.openfoodfacts.org/tofu.jpg",
				"nutriments": map[string]any{
					"proteins_100g":      12.3,
					"carbohydrates_100g": 1.7,
					"fat_100g":           6.1,
				},
			}},
		})
	}))
	defer server.Close()
	client := NewOpenFoodFactsClient(WithOpenFoodFactsBaseURL(server.URL))

	records, err := client.Search(context.Background(), ExternalSearchQuery{Query: " tofu ", Page: 3, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	for _, expected := range []string{"search_terms=tofu", "page=3", "page_size=20", "search_simple=1", "action=process", "json=1"} {
		if !strings.Contains(rawQuery, expected) {
			t.Fatalf("expected query param %s in %s", expected, rawQuery)
		}
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	record := records[0]
	if record.Provider != ProviderOpenFoodFacts || record.ExternalID != "737628064502" || record.Name != "Organic Tofu" {
		t.Fatalf("unexpected record identity: %#v", record)
	}
	if record.ServingSize == nil || *record.ServingSize != 85 || record.ServingUnit != "g" {
		t.Fatalf("unexpected serving data: %#v", record)
	}
	if record.Nutrients["proteins_100g"] != 12.3 || record.Nutrients["fat_100g"] != 6.1 {
		t.Fatalf("unexpected nutrients: %#v", record.Nutrients)
	}
	if record.ImageURL != "https://images.openfoodfacts.org/tofu.jpg" || len(record.RawPayload) == 0 {
		t.Fatalf("expected image and raw payload: %#v", record)
	}
	if client.RateLimit().Remaining != 17 || !client.RateLimit().ResetAt.Equal(time.Unix(1770000100, 0).UTC()) {
		t.Fatalf("unexpected rate limit: %#v", client.RateLimit())
	}
}

func TestOpenFoodFactsClientValidatesQuery(t *testing.T) {
	client := NewOpenFoodFactsClient()
	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "", Page: 1, PageSize: 10})
	assertProviderError(t, err, ProviderErrorInvalidQuery)
	_, err = client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 0, PageSize: 10})
	assertProviderError(t, err, ProviderErrorInvalidQuery)
	_, err = client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 51})
	assertProviderError(t, err, ProviderErrorInvalidQuery)
}

func TestOpenFoodFactsClientMapsRateLimitAndServerErrors(t *testing.T) {
	rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "45")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer rateLimited.Close()
	client := NewOpenFoodFactsClient(WithOpenFoodFactsBaseURL(rateLimited.URL))
	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})
	assertProviderError(t, err, ProviderErrorRateLimited)
	if client.RateLimit().BackoffUntil == nil {
		t.Fatalf("expected retry-after backoff")
	}

	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer unavailable.Close()
	client = NewOpenFoodFactsClient(WithOpenFoodFactsBaseURL(unavailable.URL))
	_, err = client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})
	assertProviderError(t, err, ProviderErrorUnavailable)
}

func TestOpenFoodFactsClientMapsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"products":`))
	}))
	defer server.Close()
	client := NewOpenFoodFactsClient(WithOpenFoodFactsBaseURL(server.URL))

	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})

	assertProviderError(t, err, ProviderErrorBadPayload)
}

func TestOpenFoodFactsClientMapsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
	}))
	defer server.Close()
	client := NewOpenFoodFactsClient(WithOpenFoodFactsBaseURL(server.URL), WithOpenFoodFactsTimeout(1*time.Millisecond))

	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})

	assertProviderError(t, err, ProviderErrorTimeout)
}

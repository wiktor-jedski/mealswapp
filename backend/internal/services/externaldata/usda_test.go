package externaldata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUSDAClientBuildsQueryAndParsesFoods(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		if r.URL.Path != "/foods/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("expected json accept header")
		}
		w.Header().Set("X-RateLimit-Remaining", "42")
		w.Header().Set("X-RateLimit-Reset", "1770000000")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"foods": []map[string]any{{
				"fdcId":           1101,
				"description":     "Cheddar Cheese",
				"servingSize":     28,
				"servingSizeUnit": "g",
				"foodNutrients": []map[string]any{
					{"nutrientName": "Protein", "value": 24.9, "unitName": "G"},
					{"nutrientName": "Carbohydrate, by difference", "value": 1.3, "unitName": "G"},
					{"nutrientName": "Total lipid (fat)", "value": 33.1, "unitName": "G"},
				},
			}},
		})
	}))
	defer server.Close()
	client := NewUSDAClient("demo-key", WithUSDABaseURL(server.URL))

	records, err := client.Search(context.Background(), ExternalSearchQuery{Query: " cheddar ", Page: 2, PageSize: 25})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if !strings.Contains(rawQuery, "query=cheddar") || !strings.Contains(rawQuery, "pageNumber=2") || !strings.Contains(rawQuery, "pageSize=25") || !strings.Contains(rawQuery, "api_key=demo-key") {
		t.Fatalf("unexpected query params: %s", rawQuery)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	record := records[0]
	if record.Provider != ProviderUSDA || record.ExternalID != "1101" || record.Name != "Cheddar Cheese" {
		t.Fatalf("unexpected record identity: %#v", record)
	}
	if record.ServingSize == nil || *record.ServingSize != 28 || record.ServingUnit != "g" {
		t.Fatalf("unexpected serving data: %#v", record)
	}
	if record.Nutrients["protein"] != 24.9 || record.Nutrients["total lipid (fat)"] != 33.1 {
		t.Fatalf("unexpected nutrients: %#v", record.Nutrients)
	}
	if len(record.RawPayload) == 0 {
		t.Fatalf("expected raw payload")
	}
	if client.RateLimit().Remaining != 42 || !client.RateLimit().ResetAt.Equal(time.Unix(1770000000, 0).UTC()) {
		t.Fatalf("unexpected rate limit: %#v", client.RateLimit())
	}
}

func TestUSDAClientValidatesQuery(t *testing.T) {
	client := NewUSDAClient("")
	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "", Page: 1, PageSize: 10})
	assertProviderError(t, err, ProviderErrorInvalidQuery)
	_, err = client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 0, PageSize: 10})
	assertProviderError(t, err, ProviderErrorInvalidQuery)
	_, err = client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 51})
	assertProviderError(t, err, ProviderErrorInvalidQuery)
}

func TestUSDAClientMapsRateLimitAndServerErrors(t *testing.T) {
	rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer rateLimited.Close()
	client := NewUSDAClient("", WithUSDABaseURL(rateLimited.URL))
	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})
	assertProviderError(t, err, ProviderErrorRateLimited)
	if client.RateLimit().BackoffUntil == nil {
		t.Fatalf("expected retry-after backoff")
	}

	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unavailable.Close()
	client = NewUSDAClient("", WithUSDABaseURL(unavailable.URL))
	_, err = client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})
	assertProviderError(t, err, ProviderErrorUnavailable)
}

func TestUSDAClientMapsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"foods":`))
	}))
	defer server.Close()
	client := NewUSDAClient("", WithUSDABaseURL(server.URL))

	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})

	assertProviderError(t, err, ProviderErrorBadPayload)
}

func TestUSDAClientMapsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
	}))
	defer server.Close()
	client := NewUSDAClient("", WithUSDABaseURL(server.URL), WithUSDATimeout(1*time.Millisecond))

	_, err := client.Search(context.Background(), ExternalSearchQuery{Query: "tofu", Page: 1, PageSize: 10})

	assertProviderError(t, err, ProviderErrorTimeout)
}

func assertProviderError(t *testing.T, err error, kind ProviderErrorKind) {
	t.Helper()
	var providerErr ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected provider error, got %T %v", err, err)
	}
	if providerErr.Kind != kind {
		t.Fatalf("expected provider error %s, got %#v", kind, providerErr)
	}
}

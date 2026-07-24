package externaldata

// Implements DESIGN-012 USDAClient fake-provider verification.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

const testUSDAKey = "SECRET-usda-key"

func TestLoadUSDAAPIKey(t *testing.T) {
	t.Setenv(USDAAPIKeyEnvironment, "  "+testUSDAKey+"  ")
	key, err := LoadUSDAAPIKey()
	if err != nil || key != testUSDAKey {
		t.Fatalf("key loaded=%q err=%v", key, err)
	}
	for _, value := range []string{"", "  ", "key\nleak"} {
		t.Run(fmt.Sprintf("invalid_%q", value), func(t *testing.T) {
			t.Setenv(USDAAPIKeyEnvironment, value)
			_, err := LoadUSDAAPIKey()
			assertProviderError(t, err, ProviderErrorNotConfigured, 0, false)
			if value != "" && strings.Contains(err.Error(), value) || strings.Contains(err.Error(), testUSDAKey) {
				t.Fatalf("credential leaked: %v", err)
			}
		})
	}
}

// TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically verifies
// IT-ARCH-012-001, ARCH-012, DESIGN-012 USDAClient, and SW-REQ-055.
func TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically(t *testing.T) {
	logs := &observability.MemorySink{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/fdc/v1/foods/search" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("query") != "crème & apple" || query.Get("pageNumber") != "10000" || query.Get("pageSize") != "200" || query.Get("api_key") != testUSDAKey {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		if strings.Contains(r.RequestURI, " ") {
			t.Errorf("unencoded URI = %q", r.RequestURI)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validUSDAPayload))
	}))
	defer server.Close()

	client := newTestUSDAClient(t, server.URL+"/fdc/v1/foods/search", logs, 0, 0)
	records, err := client.Search(context.Background(), ExternalSearchQuery{Query: "  crème & apple ", Provider: " USDA ", Page: 10000, PageSize: 200})
	if err != nil {
		t.Fatal(err)
	}
	serving := 240.0
	want := []ExternalFoodRecord{{
		Provider: "usda", ExternalID: "123", Name: "Apple juice", ServingSize: &serving, ServingUnit: "ml",
		Nutrients: map[string]float64{"Energy (KCAL)": 46, "Protein (G)": 0.1},
		Portions:  []ExternalFoodPortion{{Amount: 1, Unit: "cup", GramWeight: 248}, {Amount: 1, Unit: "tbsp", GramWeight: 15.5}},
	}}
	if len(records) != 1 {
		t.Fatalf("records = %+v", records)
	}
	want[0].RawPayload = records[0].RawPayload
	if !reflect.DeepEqual(records, want) || !json.Valid(records[0].RawPayload) {
		t.Fatalf("records = %#v", records)
	}
	_, captured := logs.Snapshot()
	if len(captured) != 0 {
		t.Fatalf("successful request logged diagnostics: %+v", captured)
	}
}

func TestUSDASearchRejectsInvalidInputBeforeOutboundRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	client := newTestUSDAClient(t, server.URL, nil, 0, 0)
	tests := []ExternalSearchQuery{
		{Query: "", Provider: "usda", Page: 1, PageSize: 1},
		{Query: strings.Repeat("a", 201), Provider: "usda", Page: 1, PageSize: 1},
		{Query: "apple\x00", Provider: "usda", Page: 1, PageSize: 1},
		{Query: "apple", Provider: "all", Page: 1, PageSize: 1},
		{Query: "apple", Provider: "usda", Page: 0, PageSize: 1},
		{Query: "apple", Provider: "usda", Page: 10001, PageSize: 1},
		{Query: "apple", Provider: "usda", Page: 1, PageSize: 0},
		{Query: "apple", Provider: "usda", Page: 1, PageSize: 201},
	}
	for i, query := range tests {
		_, err := client.Search(context.Background(), query)
		assertProviderError(t, err, ProviderErrorInvalidInput, 0, false)
		if calls.Load() != 0 {
			t.Fatalf("case %d made %d requests", i, calls.Load())
		}
	}
}

func TestUSDASearchHonorsDeadlineAndCallerCancellation(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()
	client := newTestUSDAClient(t, server.URL, nil, 20*time.Millisecond, 0)

	_, err := client.Search(context.Background(), validUSDAQuery())
	assertProviderError(t, err, ProviderErrorTimeout, 0, true)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline not preserved: %v", err)
	}
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := client.Search(ctx, validUSDAQuery()); done <- err }()
	<-started
	cancel()
	err = <-done
	assertProviderError(t, err, ProviderErrorCanceled, 0, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not preserved: %v", err)
	}
}

func TestUSDASearchPreservesContextErrorsWhileReadingBody(t *testing.T) {
	tests := []struct {
		name     string
		deadline time.Duration
		cancel   bool
		code     ProviderErrorCode
		cause    error
	}{
		{name: "deadline", deadline: 20 * time.Millisecond, code: ProviderErrorTimeout, cause: context.DeadlineExceeded},
		{name: "caller cancellation", cancel: true, code: ProviderErrorCanceled, cause: context.Canceled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			readStarted := make(chan struct{})
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				resp, err := http.DefaultTransport.RoundTrip(req)
				if err == nil {
					resp.Body = &signalingBody{ReadCloser: resp.Body, started: readStarted}
				}
				return resp, err
			})
			client, err := NewUSDAClient(USDAConfig{APIKey: testUSDAKey, Endpoint: server.URL, Deadline: tc.deadline, HTTPClient: &http.Client{Transport: transport}})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := client.Search(ctx, validUSDAQuery()); done <- err }()
			<-started
			if tc.cancel {
				<-readStarted
				cancel()
			}
			err = <-done
			assertProviderError(t, err, tc.code, http.StatusOK, tc.code == ProviderErrorTimeout)
			if !errors.Is(err, tc.cause) {
				t.Fatalf("context cause not preserved: %v", err)
			}
		})
	}
}

func TestUSDASearchDoesNotFollowCredentialBearingRedirects(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destinationCalls.Add(1)
		if strings.Contains(r.Referer(), testUSDAKey) {
			t.Errorf("credential leaked in Referer: %q", r.Referer())
		}
		_, _ = w.Write([]byte(validUSDAPayload))
	}))
	defer destination.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", destination.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	client, err := NewUSDAClient(USDAConfig{APIKey: testUSDAKey, Endpoint: source.URL, HTTPClient: source.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), validUSDAQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, http.StatusTemporaryRedirect, true)
	if destinationCalls.Load() != 0 {
		t.Fatalf("cross-host redirect made %d destination requests", destinationCalls.Load())
	}
}

func TestUSDASearchBoundsResponseAndRejectsMalformedOrPartialPayloads(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		limit    int64
		wantCode ProviderErrorCode
	}{
		{name: "too large", body: validUSDAPayload, limit: 16, wantCode: ProviderErrorResponseTooLarge},
		{name: "malformed JSON", body: `{`, wantCode: ProviderErrorInvalidPayload},
		{name: "malformed food", body: `{"totalHits":1,"currentPage":1,"totalPages":1,"foods":["bad"]}`, wantCode: ProviderErrorInvalidPayload},
		{name: "missing envelope", body: `{"foods":[]}`, wantCode: ProviderErrorInvalidPayload},
		{name: "missing foods", body: `{"totalHits":0,"currentPage":0,"totalPages":0}`, wantCode: ProviderErrorInvalidPayload},
		{name: "missing identity", body: searchPayload(`{"description":"Apple","foodNutrients":[]}`), wantCode: ProviderErrorInvalidPayload},
		{name: "missing nutrients", body: searchPayload(`{"fdcId":1,"description":"Apple"}`), wantCode: ProviderErrorInvalidPayload},
		{name: "partial serving", body: searchPayload(`{"fdcId":1,"description":"Apple","servingSize":10,"foodNutrients":[]}`), wantCode: ProviderErrorInvalidPayload},
		{name: "bad nutrient", body: searchPayload(`{"fdcId":1,"description":"Apple","foodNutrients":[{"nutrientName":"Protein","unitName":"G","value":-1}]}`), wantCode: ProviderErrorInvalidPayload},
		{name: "duplicate nutrient", body: searchPayload(`{"fdcId":1,"description":"Apple","foodNutrients":[{"nutrientName":"Protein","unitName":"G","value":1},{"nutrientName":"Protein","unitName":"G","value":2}]}`), wantCode: ProviderErrorInvalidPayload},
		{name: "bad portion", body: searchPayload(`{"fdcId":1,"description":"Apple","foodNutrients":[],"foodMeasures":[{"amount":1,"gramWeight":0,"measureUnit":{"name":"cup"}}]}`), wantCode: ProviderErrorInvalidPayload},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			client := newTestUSDAClient(t, server.URL, nil, 0, tc.limit)
			_, err := client.Search(context.Background(), validUSDAQuery())
			assertProviderError(t, err, tc.wantCode, http.StatusOK, false)
		})
	}
}

func TestUSDASearchHandlesRequestTransportAndBodyReadFailures(t *testing.T) {
	client := newTestUSDAClient(t, "https://example.com/search", nil, 0, 0)
	client.endpoint = &url.URL{Scheme: ":"}
	_, err := client.Search(context.Background(), validUSDAQuery())
	assertProviderError(t, err, ProviderErrorInvalidInput, 0, false)

	client = newTestUSDAClient(t, "https://example.com/search", nil, 0, 0)
	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("socket failed with provider detail")
	})}
	_, err = client.Search(context.Background(), validUSDAQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, 0, true)
	if errors.Unwrap(err) != nil || strings.Contains(fmt.Sprintf("%+v", err), testUSDAKey) {
		t.Fatalf("transport cause leaked through provider error: %+v", err)
	}

	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: failingBody{}}, nil
	})}
	_, err = client.Search(context.Background(), validUSDAQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, http.StatusOK, true)

	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Body: io.NopCloser(strings.NewReader("redirect"))}, nil
	})}
	_, err = client.Search(context.Background(), validUSDAQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, http.StatusTemporaryRedirect, true)
}

func TestDecodeUSDASearchAcceptsEmptyResultsAndOrdersPortionTies(t *testing.T) {
	empty, err := decodeUSDASearch([]byte(`{"totalHits":0,"currentPage":0,"totalPages":0,"foods":[]}`))
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty = %#v, %v", empty, err)
	}
	body := searchPayload(`{
		"fdcId":1,"description":"Water","foodNutrients":[],
		"foodMeasures":[
			{"amount":2,"gramWeight":20,"measureUnit":{"name":"cup"}},
			{"amount":1,"gramWeight":15,"measureUnit":{"name":"cup"}},
			{"amount":1,"gramWeight":10,"measureUnit":{"name":"cup"}},
			{"amount":1,"gramWeight":5,"disseminationText":"tablespoon","measureUnit":{}}
		]}`)
	records, err := decodeUSDASearch([]byte(body))
	want := []ExternalFoodPortion{{Amount: 1, Unit: "cup", GramWeight: 10}, {Amount: 1, Unit: "cup", GramWeight: 15}, {Amount: 2, Unit: "cup", GramWeight: 20}, {Amount: 1, Unit: "tablespoon", GramWeight: 5}}
	if err != nil || len(records) != 1 || !reflect.DeepEqual(records[0].Portions, want) {
		t.Fatalf("portions = %+v, %v", records, err)
	}
}

func TestUSDASearchMapsProviderStatusesAndLogsOnlyBoundedMetadata(t *testing.T) {
	tests := []struct {
		status    int
		code      ProviderErrorCode
		retryable bool
	}{
		{http.StatusBadRequest, ProviderErrorRejected, false},
		{http.StatusUnauthorized, ProviderErrorRejected, false},
		{http.StatusNotFound, ProviderErrorRejected, false},
		{http.StatusRequestTimeout, ProviderErrorUnavailable, true},
		{http.StatusTooManyRequests, ProviderErrorRateLimited, true},
		{http.StatusInternalServerError, ProviderErrorUnavailable, true},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			logs := &observability.MemorySink{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"provider detail SECRET-usda-key"}`))
			}))
			defer server.Close()
			client := newTestUSDAClient(t, server.URL, logs, 0, 0)
			_, err := client.Search(context.Background(), validUSDAQuery())
			assertProviderError(t, err, tc.code, tc.status, tc.retryable)
			encoded, marshalErr := json.Marshal(logs.Logs)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			text := string(encoded)
			if strings.Contains(text, testUSDAKey) || strings.Contains(text, "provider detail") || strings.Contains(text, server.URL) {
				t.Fatalf("unsafe diagnostics: %s", text)
			}
			if len(logs.Logs) != 1 || logs.Logs[0].Fields["code"] != string(tc.code) || logs.Logs[0].Fields["status"] != tc.status {
				t.Fatalf("diagnostics = %+v", logs.Logs)
			}
		})
	}
}

func TestUSDASearchResultProjectsBoundedHeadersOnSuccessAndFailure(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "success", status: http.StatusOK, body: validUSDAPayload},
		{name: "failure", status: http.StatusTooManyRequests, body: `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-RateLimit-Remaining", "3")
				w.Header().Set("X-RateLimit-Reset", "200")
				w.Header().Set("X-Provider-Secret", "discard-me")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			result, err := newTestUSDAClient(t, server.URL, nil, 0, 0).SearchResult(context.Background(), validUSDAQuery())
			if (tc.status == http.StatusOK) != (err == nil) || len(result.Headers) != 2 || result.Headers.Get("X-RateLimit-Remaining") != "3" || result.Headers.Get("X-RateLimit-Reset") != "200" || result.Headers.Get("X-Provider-Secret") != "" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestNewUSDAClientRejectsUnsafeConfiguration(t *testing.T) {
	for _, cfg := range []USDAConfig{
		{},
		{APIKey: "bad\nkey"},
		{APIKey: testUSDAKey, Endpoint: "://bad"},
		{APIKey: testUSDAKey, Endpoint: "file:///tmp/search"},
		{APIKey: testUSDAKey, Endpoint: "https://user@example.com/search"},
		{APIKey: testUSDAKey, Endpoint: "https://example.com/search?old=value"},
		{APIKey: testUSDAKey, Deadline: -time.Second},
		{APIKey: testUSDAKey, MaxBodyBytes: -1},
		{APIKey: testUSDAKey, MaxBodyBytes: maxUSDABodyLimit + 1},
		{APIKey: testUSDAKey, MaxBodyBytes: math.MaxInt64},
	} {
		if _, err := NewUSDAClient(cfg); err == nil || strings.Contains(err.Error(), testUSDAKey) {
			t.Fatalf("config accepted or leaked: %+v %v", cfg, err)
		}
	}
	client, err := NewUSDAClient(USDAConfig{APIKey: testUSDAKey, MaxBodyBytes: maxUSDABodyLimit})
	if err != nil || client.endpoint.String() != DefaultUSDAEndpoint || client.httpClient == http.DefaultClient || client.httpClient.CheckRedirect == nil {
		t.Fatalf("defaults = %+v, %v", client, err)
	}
}

func newTestUSDAClient(t *testing.T, endpoint string, logs observability.LogSink, deadline time.Duration, maxBody int64) *USDAClient {
	t.Helper()
	client, err := NewUSDAClient(USDAConfig{APIKey: testUSDAKey, Endpoint: endpoint, Deadline: deadline, MaxBodyBytes: maxBody, Logs: logs})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func validUSDAQuery() ExternalSearchQuery {
	return ExternalSearchQuery{Query: "apple", Provider: "usda", Page: 1, PageSize: 25}
}

func searchPayload(food string) string {
	return `{"totalHits":1,"currentPage":1,"totalPages":1,"foods":[` + food + `]}`
}

func assertProviderError(t *testing.T, err error, code ProviderErrorCode, status int, retryable bool) {
	t.Helper()
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != code || providerErr.HTTPStatus != status || providerErr.Retryable != retryable {
		t.Fatalf("error = %#v, want code=%s status=%d retryable=%t", err, code, status, retryable)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) {
	return 0, errors.New("body failed with provider detail")
}
func (failingBody) Close() error { return nil }

type signalingBody struct {
	io.ReadCloser
	started chan struct{}
}

func (b *signalingBody) Read(p []byte) (int, error) {
	select {
	case <-b.started:
	default:
		close(b.started)
	}
	return b.ReadCloser.Read(p)
}

const validUSDAPayload = `{
  "totalHits": 1,
  "currentPage": 1,
  "totalPages": 1,
  "foods": [{
    "fdcId": 123,
    "description": " Apple juice ",
    "servingSize": 240,
    "servingSizeUnit": " ml ",
    "foodNutrients": [
      {"nutrientName":"Protein","unitName":"G","value":0.1},
      {"nutrientName":"Energy","unitName":"KCAL","value":46}
    ],
    "foodMeasures": [
      {"amount":1,"gramWeight":15.5,"measureUnit":{"name":"tablespoon","abbreviation":"tbsp"}},
      {"amount":1,"gramWeight":248,"measureUnit":{"name":"cup","abbreviation":"cup"}}
    ]
  }]
}`

package externaldata

// Implements DESIGN-012 OpenFoodFactsClient fake-provider verification.

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
	"unicode/utf8"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

const testOpenFoodFactsCallerID = "Mealswapp/0.1 (ops@example.com)"

// TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically
// verifies IT-ARCH-012-001, ARCH-012, DESIGN-012 OpenFoodFactsClient, and SW-REQ-055.
func TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/cgi/search.pl" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		want := url.Values{
			"action":        {"process"},
			"fields":        {openFoodFactsFields},
			"json":          {"1"},
			"page":          {"10000"},
			"page_size":     {"100"},
			"search_simple": {"1"},
			"search_terms":  {"crème & apple"},
		}
		if !reflect.DeepEqual(query, want) || strings.Contains(r.RequestURI, " ") {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		if r.Header.Get("User-Agent") != testOpenFoodFactsCallerID || r.Header.Get("Accept") != "application/json" {
			t.Errorf("headers = %+v", r.Header)
		}
		_, _ = w.Write([]byte(validOpenFoodFactsPayload))
	}))
	defer server.Close()

	client := newTestOpenFoodFactsClient(t, server.URL+"/cgi/search.pl", nil, 0, 0)
	records, err := client.Search(context.Background(), ExternalSearchQuery{Query: "  crème & apple ", Provider: " Open Food Facts ", Page: 10000, PageSize: 100})
	serving := 250.0
	packageSize := 1.5
	want := []ExternalFoodRecord{{
		Provider: "openfoodfacts", ExternalID: "3017620422003", Name: "Apple drink", ServingSize: &serving, ServingUnit: "ml",
		PackageSize: &packageSize, PackageUnit: "oz",
		Nutrients: map[string]float64{"carbohydrates_100g": 5.2, "energy-kcal_100g": 46, "fat_100g": 0, "proteins_100g": 0.1},
		ImageURL:  "https://images.openfoodfacts.org/apple.jpg",
	}}
	if err != nil || !reflect.DeepEqual(records, want) || records[0].RawPayload != nil {
		t.Fatalf("records = %#v, err = %v", records, err)
	}
}

func TestOpenFoodFactsSearchRejectsInvalidInputBeforeOutboundRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	client := newTestOpenFoodFactsClient(t, server.URL, nil, 0, 0)
	tests := []ExternalSearchQuery{
		{Query: "", Provider: "openfoodfacts", Page: 1, PageSize: 1},
		{Query: strings.Repeat("a", 201), Provider: "openfoodfacts", Page: 1, PageSize: 1},
		{Query: "apple\x00", Provider: "openfoodfacts", Page: 1, PageSize: 1},
		{Query: "apple", Provider: "all", Page: 1, PageSize: 1},
		{Query: "apple", Provider: "openfoodfacts", Page: 0, PageSize: 1},
		{Query: "apple", Provider: "openfoodfacts", Page: 10001, PageSize: 1},
		{Query: "apple", Provider: "openfoodfacts", Page: 1, PageSize: 0},
		{Query: "apple", Provider: "openfoodfacts", Page: 1, PageSize: 101},
	}
	for i, query := range tests {
		_, err := client.Search(context.Background(), query)
		assertProviderError(t, err, ProviderErrorInvalidInput, 0, false)
		if calls.Load() != 0 {
			t.Fatalf("case %d made %d requests", i, calls.Load())
		}
	}
}

func TestOpenFoodFactsSearchHonorsDeadlineAndCallerCancellation(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()
	client := newTestOpenFoodFactsClient(t, server.URL, nil, 20*time.Millisecond, 0)

	_, err := client.Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorTimeout, 0, true)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline not preserved: %v", err)
	}
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := client.Search(ctx, validOpenFoodFactsQuery()); done <- err }()
	<-started
	cancel()
	err = <-done
	assertProviderError(t, err, ProviderErrorCanceled, 0, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not preserved: %v", err)
	}
}

func TestOpenFoodFactsSearchPreservesContextErrorsWhileReadingBody(t *testing.T) {
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
			client := newTestOpenFoodFactsClient(t, "https://example.com/search", nil, tc.deadline, 0)
			client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: &contextBlockingBody{ctx: req.Context(), started: started}}, nil
			})}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := client.Search(ctx, validOpenFoodFactsQuery()); done <- err }()
			<-started
			if tc.cancel {
				cancel()
			}
			err := <-done
			assertProviderError(t, err, tc.code, http.StatusOK, tc.code == ProviderErrorTimeout)
			if !errors.Is(err, tc.cause) {
				t.Fatalf("context cause not preserved: %v", err)
			}
		})
	}
}

func TestOpenFoodFactsSearchPreservesTransportContextSentinels(t *testing.T) {
	tests := []struct {
		name      string
		transport error
		code      ProviderErrorCode
		retryable bool
		cause     error
	}{
		{name: "deadline", transport: context.DeadlineExceeded, code: ProviderErrorTimeout, retryable: true, cause: context.DeadlineExceeded},
		{name: "wrapped deadline", transport: fmt.Errorf("transport: %w", context.DeadlineExceeded), code: ProviderErrorTimeout, retryable: true, cause: context.DeadlineExceeded},
		{name: "canceled", transport: context.Canceled, code: ProviderErrorCanceled, cause: context.Canceled},
		{name: "wrapped canceled", transport: fmt.Errorf("transport: %w", context.Canceled), code: ProviderErrorCanceled, cause: context.Canceled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestOpenFoodFactsClient(t, "https://example.com/search", nil, 0, 0)
			client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, tc.transport })}
			_, err := client.Search(context.Background(), validOpenFoodFactsQuery())
			assertProviderError(t, err, tc.code, 0, tc.retryable)
			if !errors.Is(err, tc.cause) || strings.Contains(fmt.Sprintf("%+v", err), "transport:") {
				t.Fatalf("transport sentinel not safely preserved: %+v", err)
			}
		})
	}
}

func TestOpenFoodFactsSearchBoundsBodiesAndHandlesMalformedOrPartialPayloads(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int64
	}{
		{name: "too large", body: validOpenFoodFactsPayload, limit: 16},
		{name: "malformed JSON", body: `{`},
		{name: "missing envelope", body: `{"products":[]}`},
		{name: "missing products", body: `{"count":0,"page":1,"page_count":0,"page_size":20}`},
		{name: "negative count", body: `{"count":-1,"page":1,"page_count":0,"page_size":20,"products":[]}`},
		{name: "zero page", body: `{"count":0,"page":0,"page_count":0,"page_size":20,"products":[]}`},
		{name: "negative page count", body: `{"count":0,"page":1,"page_count":-1,"page_size":20,"products":[]}`},
		{name: "zero page size", body: `{"count":0,"page":1,"page_count":0,"page_size":0,"products":[]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			client := newTestOpenFoodFactsClient(t, server.URL, nil, 0, tc.limit)
			_, err := client.Search(context.Background(), validOpenFoodFactsQuery())
			wantCode := ProviderErrorInvalidPayload
			if tc.limit > 0 {
				wantCode = ProviderErrorResponseTooLarge
			}
			assertProviderError(t, err, wantCode, http.StatusOK, false)
		})
	}

	logs := &observability.MemorySink{}
	body := `{"count":6,"page":1,"page_count":6,"page_size":20,"products":[` +
		`{"code":"1","product_name":"Valid","nutriments":{}},` +
		`{"product_name":"Missing code","nutriments":{}},` +
		`{"code":"2","product_name":"Missing nutrients"},` +
		`{"code":"3","product_name":"Negative nutrient","nutriments":{"fat_100g":-1}},` +
		`{"code":"4","product_name":"Null nutrient","nutriments":{"fat_100g":null}},` +
		`{"code":"5","product_name":"Overflowed nutrient","nutriments":{"fat_100g":1e999}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
	defer server.Close()
	records, err := newTestOpenFoodFactsClient(t, server.URL, logs, 0, 0).Search(context.Background(), validOpenFoodFactsQuery())
	if err != nil || len(records) != 1 || records[0].ExternalID != "1" || records[0].RawPayload != nil {
		t.Fatalf("partial records = %#v, %v", records, err)
	}
	if len(logs.Logs) != 1 || logs.Logs[0].Message != "external_provider_payload_dropped" || logs.Logs[0].Fields["count"] != 5 {
		t.Fatalf("partial diagnostics = %+v", logs.Logs)
	}
}

func TestOpenFoodFactsSearchEnforcesFiniteAllocationBound(t *testing.T) {
	exactBody := paddedOpenFoodFactsPayload(t, int(defaultOpenFoodFactsBodyLimit))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(exactBody) }))
	defer server.Close()
	records, err := newTestOpenFoodFactsClient(t, server.URL, nil, 0, defaultOpenFoodFactsBodyLimit).Search(context.Background(), validOpenFoodFactsQuery())
	if err != nil || len(records) != 1 {
		t.Fatalf("exact policy limit rejected: records=%d err=%v", len(records), err)
	}

	body := &countingInfiniteBody{}
	client := newTestOpenFoodFactsClient(t, "https://example.com/search", nil, 0, defaultOpenFoodFactsBodyLimit)
	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})}
	_, err = client.Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorResponseTooLarge, http.StatusOK, false)
	if body.read != defaultOpenFoodFactsBodyLimit+1 {
		t.Fatalf("provider body bytes read = %d, want %d", body.read, defaultOpenFoodFactsBodyLimit+1)
	}
}

func TestProjectOpenFoodFactsProductHandlesOptionalAndMalformedFields(t *testing.T) {
	products := []struct {
		name string
		json string
		ok   bool
	}{
		{name: "optional fields omitted", json: `{"code":"1","product_name":"Apple","nutriments":{"label":"per 100g","fat_100g":0}}`, ok: true},
		{name: "partial serving", json: `{"code":"1","product_name":"Apple","serving_quantity":20,"nutriments":{}}`, ok: true},
		{name: "unsupported serving unit", json: `{"code":"1","product_name":"Apple","serving_quantity":20,"serving_quantity_unit":"slice","nutriments":{}}`, ok: true},
		{name: "unsafe image", json: `{"code":"1","product_name":"Apple","image_front_url":"http://localhost/a","nutriments":{}}`, ok: true},
		{name: "numeric code", json: `{"code":1,"product_name":"Apple","nutriments":{}}`},
		{name: "blank name", json: `{"code":"1","product_name":" ","nutriments":{}}`},
		{name: "bad identifier", json: `{"code":"bad id","product_name":"Apple","nutriments":{}}`},
		{name: "control nutrient key", json: `{"code":"1","product_name":"Apple","nutriments":{"bad\u0000key":1}}`},
		{name: "long nutrient key", json: `{"code":"1","product_name":"Apple","nutriments":{"` + strings.Repeat("x", 129) + `":1}}`},
	}
	for _, tc := range products {
		t.Run(tc.name, func(t *testing.T) {
			var product openFoodFactsProduct
			if err := json.Unmarshal([]byte(tc.json), &product); err != nil {
				t.Fatal(err)
			}
			record, ok := projectOpenFoodFactsProduct(product)
			if ok != tc.ok {
				t.Fatalf("record = %#v, ok = %t", record, ok)
			}
			if ok && (record.RawPayload != nil || record.ServingSize != nil || record.ImageURL != "") {
				t.Fatalf("unsafe optional projection = %#v", record)
			}
		})
	}
	invalidUTF8 := string([]byte{utf8.RuneSelf})
	if !containsUnsafeProviderText(invalidUTF8) || !containsUnsafeProviderText("bad\u202ename") || containsUnsafeProviderText("energy-kcal_100g") {
		t.Fatal("provider-text safety mismatch")
	}
}

func TestProjectOpenFoodFactsProductRejectsMalformedNumericNutriments(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		nutriment string
		ok        bool
	}{
		{name: "null", key: "fat_100g", nutriment: `null`},
		{name: "overflow", key: "fat_100g", nutriment: `1e999`},
		{name: "boolean", key: "fat_100g", nutriment: `true`},
		{name: "object", key: "fat_100g", nutriment: `{}`},
		{name: "array", key: "fat_100g", nutriment: `[]`},
		{name: "numeric field text", key: "fat_100g", nutriment: `"unknown"`},
		{name: "unit metadata", key: "energy-kcal_unit", nutriment: `"kcal"`, ok: true},
		{name: "label metadata", key: "label", nutriment: `"per 100g"`, ok: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var product openFoodFactsProduct
			payload := `{"code":"1","product_name":"Apple","nutriments":{"` + tc.key + `":` + tc.nutriment + `,"proteins_100g":0}}`
			if err := json.Unmarshal([]byte(payload), &product); err != nil {
				t.Fatal(err)
			}
			record, ok := projectOpenFoodFactsProduct(product)
			if ok != tc.ok {
				t.Fatalf("record = %#v, ok = %t", record, ok)
			}
			if ok && !reflect.DeepEqual(record.Nutrients, map[string]float64{"proteins_100g": 0}) {
				t.Fatalf("nutrients = %#v", record.Nutrients)
			}
		})
	}
	if record, ok := projectOpenFoodFactsProduct(openFoodFactsProduct{
		Code:      json.RawMessage(`"1"`),
		Name:      json.RawMessage(`"Apple"`),
		Nutrients: map[string]json.RawMessage{"fat_100g": nil},
	}); ok {
		t.Fatalf("empty nutrient token accepted: %#v", record)
	}
}

func TestOpenFoodFactsSearchMapsStatusesAndLogsOnlyBoundedMetadata(t *testing.T) {
	tests := []struct {
		status    int
		code      ProviderErrorCode
		retryable bool
	}{
		{http.StatusPermanentRedirect, ProviderErrorUnavailable, true},
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
				_, _ = w.Write([]byte(`{"error":"provider detail SECRET-off"}`))
			}))
			defer server.Close()
			_, err := newTestOpenFoodFactsClient(t, server.URL, logs, 0, 0).Search(context.Background(), validOpenFoodFactsQuery())
			assertProviderError(t, err, tc.code, tc.status, tc.retryable)
			encoded, marshalErr := json.Marshal(logs.Logs)
			if marshalErr != nil || strings.Contains(string(encoded), "provider detail") || strings.Contains(string(encoded), server.URL) || strings.Contains(err.Error(), "usda") {
				t.Fatalf("unsafe diagnostics: %s, %v", encoded, err)
			}
			if len(logs.Logs) != 1 || logs.Logs[0].Fields["provider"] != "openfoodfacts" || logs.Logs[0].Fields["code"] != string(tc.code) {
				t.Fatalf("diagnostics = %+v", logs.Logs)
			}
		})
	}
}

func TestOpenFoodFactsSearchResultProjectsBoundedHeadersOnSuccessAndFailure(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "success", status: http.StatusOK, body: validOpenFoodFactsPayload},
		{name: "failure", status: http.StatusTooManyRequests, body: `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-RateLimit-Remaining", "4")
				w.Header().Set("X-RateLimit-Reset", "300")
				w.Header().Set("X-Provider-Secret", "discard-me")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			result, err := newTestOpenFoodFactsClient(t, server.URL, nil, 0, 0).SearchResult(context.Background(), validOpenFoodFactsQuery())
			if (tc.status == http.StatusOK) != (err == nil) || len(result.Headers) != 2 || result.Headers.Get("X-RateLimit-Remaining") != "4" || result.Headers.Get("X-RateLimit-Reset") != "300" || result.Headers.Get("X-Provider-Secret") != "" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestOpenFoodFactsSearchHandlesRequestTransportAndBodyReadFailures(t *testing.T) {
	client := newTestOpenFoodFactsClient(t, "https://example.com/search", nil, 0, 0)
	client.endpoint = &url.URL{Scheme: ":"}
	_, err := client.Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorInvalidInput, 0, false)

	client = newTestOpenFoodFactsClient(t, "https://example.com/search", nil, 0, 0)
	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("socket failed with query detail")
	})}
	_, err = client.Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, 0, true)
	if errors.Unwrap(err) != nil || strings.Contains(fmt.Sprintf("%+v", err), "query detail") {
		t.Fatalf("transport cause leaked: %+v", err)
	}

	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: failingBody{}}, nil
	})}
	_, err = client.Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, http.StatusOK, true)

	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Body: io.NopCloser(strings.NewReader("redirect"))}, nil
	})}
	_, err = client.Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, http.StatusTemporaryRedirect, true)

	var redirectedCalls atomic.Int32
	redirected := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirectedCalls.Add(1) }))
	defer redirected.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, redirected.URL, http.StatusFound) }))
	defer redirector.Close()
	_, err = newTestOpenFoodFactsClient(t, redirector.URL, nil, 0, 0).Search(context.Background(), validOpenFoodFactsQuery())
	assertProviderError(t, err, ProviderErrorUnavailable, http.StatusFound, true)
	if redirectedCalls.Load() != 0 {
		t.Fatalf("followed provider redirect %d times", redirectedCalls.Load())
	}
}

func TestNewOpenFoodFactsClientRejectsUnsafeConfiguration(t *testing.T) {
	for _, cfg := range []OpenFoodFactsConfig{
		{},
		{CallerID: "bad\ncaller"},
		{CallerID: strings.Repeat("a", 257)},
		{CallerID: testOpenFoodFactsCallerID, Endpoint: "://bad"},
		{CallerID: testOpenFoodFactsCallerID, Endpoint: "file:///tmp/search"},
		{CallerID: testOpenFoodFactsCallerID, Endpoint: "https://user@example.com/search"},
		{CallerID: testOpenFoodFactsCallerID, Endpoint: "https://example.com/search?old=value"},
		{CallerID: testOpenFoodFactsCallerID, Deadline: -time.Second},
		{CallerID: testOpenFoodFactsCallerID, MaxBodyBytes: -1},
		{CallerID: testOpenFoodFactsCallerID, MaxBodyBytes: defaultOpenFoodFactsBodyLimit + 1},
		{CallerID: testOpenFoodFactsCallerID, MaxBodyBytes: math.MaxInt64},
	} {
		if _, err := NewOpenFoodFactsClient(cfg); err == nil || cfg.CallerID != "" && strings.Contains(err.Error(), cfg.CallerID) {
			t.Fatalf("config accepted or leaked: %+v %v", cfg, err)
		}
	}
	client, err := NewOpenFoodFactsClient(OpenFoodFactsConfig{CallerID: "  " + testOpenFoodFactsCallerID + "  "})
	if err != nil || client.endpoint.String() != DefaultOpenFoodFactsEndpoint || client.httpClient.Transport != http.DefaultClient.Transport || client.callerID != testOpenFoodFactsCallerID {
		t.Fatalf("defaults = %+v, %v", client, err)
	}
}

func newTestOpenFoodFactsClient(t *testing.T, endpoint string, logs observability.LogSink, deadline time.Duration, maxBody int64) *OpenFoodFactsClient {
	t.Helper()
	client, err := NewOpenFoodFactsClient(OpenFoodFactsConfig{CallerID: testOpenFoodFactsCallerID, Endpoint: endpoint, Deadline: deadline, MaxBodyBytes: maxBody, Logs: logs})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func validOpenFoodFactsQuery() ExternalSearchQuery {
	return ExternalSearchQuery{Query: "apple", Provider: "openfoodfacts", Page: 1, PageSize: 20}
}

type contextBlockingBody struct {
	ctx     context.Context
	started chan struct{}
}

func (b *contextBlockingBody) Read([]byte) (int, error) {
	select {
	case <-b.started:
	default:
		close(b.started)
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (*contextBlockingBody) Close() error { return nil }

type countingInfiniteBody struct{ read int64 }

func (b *countingInfiniteBody) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	b.read += int64(len(p))
	return len(p), nil
}

func (*countingInfiniteBody) Close() error { return nil }

func paddedOpenFoodFactsPayload(t *testing.T, size int) []byte {
	t.Helper()
	body := []byte(validOpenFoodFactsPayload)
	if len(body) > size {
		t.Fatalf("payload length %d exceeds requested size %d", len(body), size)
	}
	padding := make([]byte, size-len(body))
	for i := range padding {
		padding[i] = ' '
	}
	return append(body, padding...)
}

const validOpenFoodFactsPayload = `{
  "count": 1,
  "page": 1,
  "page_count": 1,
  "page_size": 100,
  "products": [{
    "code": "3017620422003",
    "product_name": " Apple drink ",
    "serving_quantity": 250,
    "serving_quantity_unit": " millilitres ",
    "product_quantity": 1.5,
    "product_quantity_unit": " ounces ",
    "image_front_url": "https://images.openfoodfacts.org/apple.jpg",
    "nutriments": {
      "energy-kcal_100g": 46,
      "proteins_100g": 0.1,
      "carbohydrates_100g": 5.2,
      "fat_100g": 0,
      "energy-kcal_unit": "kcal"
    }
  }]
}`

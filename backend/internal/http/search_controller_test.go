package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/handlers"
	"mealswapp/backend/internal/services/entitlements"
	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/google/uuid"
)

func TestSearchControllerHandlesSingleSearchWithPaginationAndFilters(t *testing.T) {
	service := &fakeSearchService{
		searchResponse: handlers.SearchResponse{
			Items:            []any{map[string]any{"id": "food-1", "name": "Tofu"}},
			TotalCount:       12,
			SimilarityScores: []float64{},
		},
	}
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, SearchService: service})
	tagID := uuid.MustParse("90000000-0000-0000-0000-000000000001")

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{
		"query":" tofu ",
		"mode":"single",
		"page":2,
		"filters":[{"tagId":"`+tagID.String()+`","kind":"functionality","include":true}],
		"enabledMacros":{"protein":true,"carbs":true,"fat":false},
		"sourceProviders":["USDA"]
	}`, "", true)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected search 200, got %d", res.StatusCode)
	}
	if service.lastSearch.Query != "tofu" || service.lastSearch.Mode != searchsvc.ModeSingle || service.lastSearch.Page != 2 {
		t.Fatalf("unexpected parsed search request: %#v", service.lastSearch)
	}
	if service.lastSearch.FilterQuery.Limit != 10 || service.lastSearch.FilterQuery.Offset != 10 {
		t.Fatalf("unexpected pagination query: %#v", service.lastSearch.FilterQuery.FoodItemQuery)
	}
	if len(service.lastSearch.FilterQuery.IncludeTagIDs) != 1 || service.lastSearch.FilterQuery.IncludeTagIDs[0] != tagID {
		t.Fatalf("expected include tag filter, got %#v", service.lastSearch.FilterQuery.IncludeTagIDs)
	}
	if service.lastSearch.FilterQuery.EnabledMacros["fat"] {
		t.Fatalf("expected disabled fat macro toggle, got %#v", service.lastSearch.FilterQuery.EnabledMacros)
	}

	payload := decodeEnvelope(t, res)
	data := dataMap(t, payload.Data)
	if data["totalCount"].(float64) != 12 || data["page"].(float64) != 2 || data["pageSize"].(float64) != 10 {
		t.Fatalf("unexpected search response metadata: %#v", data)
	}
}

func TestSearchControllerP95LatencyGate(t *testing.T) {
	service := &fakeSearchService{
		searchResponse: handlers.SearchResponse{
			Items:            []any{map[string]any{"id": "food-1", "name": "Tofu"}},
			TotalCount:       1,
			SimilarityScores: []float64{},
		},
	}
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, SearchService: service})
	latencies := make([]time.Duration, 0, 50)

	for i := 0; i < 50; i++ {
		startedAt := time.Now()
		res := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"tofu","mode":"single"}`, "", true)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected search 200, got %d", res.StatusCode)
		}
		res.Body.Close()
		latencies = append(latencies, time.Since(startedAt))
	}

	if p95(latencies) > 2*time.Second {
		t.Fatalf("search handler P95 exceeded 2s target: %s", p95(latencies))
	}
}

func TestSearchControllerHandlesReplacementAndDietSearch(t *testing.T) {
	service := &fakeSearchService{searchResponse: handlers.SearchResponse{}}
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, SearchService: service})

	replacement := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"olive oil","mode":"replacement","sourceItemId":"butter"}`, "", true)
	defer replacement.Body.Close()
	if replacement.StatusCode != http.StatusOK {
		t.Fatalf("expected replacement search 200, got %d", replacement.StatusCode)
	}
	if service.lastSearch.Mode != searchsvc.ModeReplacement || service.lastSearch.SourceItem != "butter" {
		t.Fatalf("unexpected replacement request: %#v", service.lastSearch)
	}

	diet := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"mode":"diet","ingredients":[{"name":"tofu","quantity":100,"unit":"gram"}]}`, "", true)
	defer diet.Body.Close()
	if diet.StatusCode != http.StatusOK {
		t.Fatalf("expected diet search 200, got %d", diet.StatusCode)
	}
	if service.lastSearch.Mode != searchsvc.ModeDiet || len(service.lastSearch.Ingredients) != 1 {
		t.Fatalf("unexpected diet request: %#v", service.lastSearch)
	}
}

func TestSearchControllerReturnsGracefulEmptyResults(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config:        config.Config{Environment: "test"},
		SearchService: &fakeSearchService{searchResponse: handlers.SearchResponse{}},
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"missing","mode":"single"}`, "", true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected empty search 200, got %d", res.StatusCode)
	}
	data := dataMap(t, decodeEnvelope(t, res).Data)
	if len(data["items"].([]any)) != 0 || len(data["warnings"].([]any)) != 0 {
		t.Fatalf("expected graceful empty arrays, got %#v", data)
	}
}

func TestAutocompleteControllerReturnsRankedSuggestions(t *testing.T) {
	service := &fakeSearchService{
		autocompleteResponse: []searchsvc.RankedAutocomplete{
			{ItemID: "food-1", Label: "Tofu", ExactMatch: true, Rank: 1},
		},
	}
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, SearchService: service})

	res := performRequest(t, app, http.MethodGet, "/api/v1/autocomplete?query=tofu&limit=5")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected autocomplete 200, got %d", res.StatusCode)
	}
	if service.lastAutocomplete.Query != "tofu" || service.lastAutocomplete.Limit != 5 {
		t.Fatalf("unexpected autocomplete request: %#v", service.lastAutocomplete)
	}
	items := decodeEnvelope(t, res).Data.([]any)
	if len(items) != 1 || items[0].(map[string]any)["label"] != "Tofu" {
		t.Fatalf("unexpected autocomplete payload: %#v", items)
	}
}

func TestAutocompleteControllerHandlesEmptyQueryWithoutServiceCall(t *testing.T) {
	service := &fakeSearchService{}
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, SearchService: service})

	res := performRequest(t, app, http.MethodGet, "/api/v1/autocomplete?query=")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected empty autocomplete 200, got %d", res.StatusCode)
	}
	if service.autocompleteCalls != 0 {
		t.Fatalf("expected empty query to skip service, got %d calls", service.autocompleteCalls)
	}
	if len(decodeEnvelope(t, res).Data.([]any)) != 0 {
		t.Fatal("expected empty autocomplete suggestions")
	}
}

func TestSearchControllerRejectsInvalidPageAndAutocompleteLimit(t *testing.T) {
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, SearchService: &fakeSearchService{}})

	searchRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"tofu","page":0}`, "", true)
	defer searchRes.Body.Close()
	if searchRes.StatusCode != http.StatusOK {
		t.Fatalf("page zero should default to page one, got %d", searchRes.StatusCode)
	}

	invalidPage := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"tofu","page":-1}`, "", true)
	defer invalidPage.Body.Close()
	if invalidPage.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid page 400, got %d", invalidPage.StatusCode)
	}

	limitRes := performRequest(t, app, http.MethodGet, "/api/v1/autocomplete?query=tofu&limit=11")
	defer limitRes.Body.Close()
	if limitRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid autocomplete limit 400, got %d", limitRes.StatusCode)
	}
}

func TestSearchControllerUsageLimiterBlocksFreeModeAndLimit(t *testing.T) {
	service := &fakeSearchService{searchResponse: handlers.SearchResponse{}}
	limiter := &fakeSearchUsageLimiter{
		decisions: []entitlements.Decision{
			{Allowed: false, Code: "mode_not_allowed", Reason: "Current plan does not allow this search mode."},
		},
	}
	app := NewRouter(ServiceDependencies{
		Config:             config.Config{Environment: "test"},
		SearchService:      service,
		SearchUsageLimiter: limiter,
	})

	blockedMode := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"olive oil","mode":"replacement","sourceItemId":"butter"}`, "", true)
	defer blockedMode.Body.Close()
	if blockedMode.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("expected free replacement search to be payment required, got %d", blockedMode.StatusCode)
	}
	payload := decodeEnvelope(t, blockedMode)
	if payload.Error == nil || payload.Error.Code != "mode_not_allowed" || payload.Error.Category != "entitlement" {
		t.Fatalf("expected entitlement mode error, got %#v", payload)
	}
	if service.lastSearch.Mode != "" {
		t.Fatalf("blocked search should not call service, got %#v", service.lastSearch)
	}

	limiter.decisions = []entitlements.Decision{
		{Allowed: false, Code: "search_limit_reached", Reason: "Free plan search limit reached for the current 24-hour window."},
	}
	blockedLimit := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"tofu","mode":"single"}`, "", true)
	defer blockedLimit.Body.Close()
	if blockedLimit.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("expected free limit to be payment required, got %d", blockedLimit.StatusCode)
	}
	payload = decodeEnvelope(t, blockedLimit)
	if payload.Error == nil || payload.Error.Code != "search_limit_reached" || payload.Error.Category != "entitlement" {
		t.Fatalf("expected entitlement limit error, got %#v", payload)
	}
}

func TestSearchControllerUsageLimiterRecordsAllowedSearchBeforeService(t *testing.T) {
	service := &fakeSearchService{searchResponse: handlers.SearchResponse{}}
	limiter := &fakeSearchUsageLimiter{
		decisions: []entitlements.Decision{{Allowed: true, Code: "allowed"}},
	}
	app := NewRouter(ServiceDependencies{
		Config:             config.Config{Environment: "test"},
		SearchService:      service,
		SearchUsageLimiter: limiter,
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/search", `{"query":"tofu","mode":"single"}`, "access-token", true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected allowed search 200, got %d", res.StatusCode)
	}
	if limiter.calls != 1 || limiter.lastAccessToken != "access-token" || limiter.lastMode != searchsvc.ModeSingle {
		t.Fatalf("expected limiter call with token and mode, got %#v", limiter)
	}
	if service.lastSearch.Mode != searchsvc.ModeSingle || service.lastSearch.Query != "tofu" {
		t.Fatalf("expected allowed search to call service, got %#v", service.lastSearch)
	}
}

type fakeSearchService struct {
	searchResponse       handlers.SearchResponse
	autocompleteResponse []searchsvc.RankedAutocomplete
	lastSearch           handlers.SearchRequest
	lastAutocomplete     handlers.AutocompleteRequest
	autocompleteCalls    int
}

func (service *fakeSearchService) Search(ctx context.Context, request handlers.SearchRequest) (handlers.SearchResponse, error) {
	service.lastSearch = request
	return service.searchResponse, nil
}

func (service *fakeSearchService) Autocomplete(ctx context.Context, request handlers.AutocompleteRequest) ([]searchsvc.RankedAutocomplete, error) {
	service.autocompleteCalls++
	service.lastAutocomplete = request
	return service.autocompleteResponse, nil
}

type fakeSearchUsageLimiter struct {
	decisions       []entitlements.Decision
	calls           int
	lastAccessToken string
	lastMode        searchsvc.Mode
	err             error
}

func (limiter *fakeSearchUsageLimiter) CheckAndRecord(ctx context.Context, accessToken string, mode searchsvc.Mode) (entitlements.Decision, error) {
	limiter.calls++
	limiter.lastAccessToken = accessToken
	limiter.lastMode = mode
	if limiter.err != nil {
		return entitlements.Decision{}, limiter.err
	}
	if len(limiter.decisions) == 0 {
		return entitlements.Decision{Allowed: true, Code: "allowed"}, nil
	}
	decision := limiter.decisions[0]
	limiter.decisions = limiter.decisions[1:]
	return decision, nil
}

func dataMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", value)
	}
	return data
}

func p95(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	for i := 1; i < len(sorted); i++ {
		value := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > value {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = value
	}
	index := (95*len(sorted) + 99) / 100
	if index < 1 {
		index = 1
	}
	return sorted[index-1]
}

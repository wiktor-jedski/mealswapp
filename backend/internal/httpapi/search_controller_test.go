package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// Implements DESIGN-002 SearchController HTTP verification.

type fakeSearchService struct {
	response search.SearchResponse
	err      error
	request  search.SearchRequest
	calls    int
}

type fakeAutocompleteService struct {
	response search.AutocompleteResponse
	err      error
	query    string
	context  repository.RepositoryContext
	calls    int
}

func (s *fakeAutocompleteService) Autocomplete(_ context.Context, query string, rc repository.RepositoryContext) (search.AutocompleteResponse, error) {
	s.calls++
	s.query = query
	s.context = rc
	return s.response, s.err
}

func (s *fakeSearchService) Search(_ context.Context, req search.SearchRequest) (search.SearchResponse, error) {
	s.calls++
	s.request = req
	return s.response, s.err
}

type fakeSearchHistoryAppender struct {
	calls       int
	userID      uuid.UUID
	query       string
	mode        string
	filtersHash string
	err         error
}

func (h *fakeSearchHistoryAppender) AddHistory(_ context.Context, userID uuid.UUID, query string, mode string, filtersHash string) (uuid.UUID, error) {
	h.calls++
	h.userID = userID
	h.query = query
	h.mode = mode
	h.filtersHash = filtersHash
	return uuid.New(), h.err
}

type countingSearchRepository struct {
	calls int
}

func (r *countingSearchRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	r.calls++
	return []repository.FoodItemEntity{}, 0, nil
}

type substitutionSearchRepository struct {
	sourceID uuid.UUID
	targetID uuid.UUID
}

func (r substitutionSearchRepository) GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.FoodItemEntity, error) {
	return repository.FoodItemEntity{
		ID:            r.sourceID,
		Name:          "Milk",
		PhysicalState: repository.PhysicalStateLiquid,
		MacrosPer100:  repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1},
	}, nil
}

func (r substitutionSearchRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	items := []repository.FoodItemEntity{{
		ID:            r.targetID,
		Name:          "Soy Milk",
		PhysicalState: repository.PhysicalStateLiquid,
		MacrosPer100:  repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1},
	}}
	return items, len(items), nil
}

type countingSearchCache struct {
	gets int
	sets int
}

func (c *countingSearchCache) GetSearchResponse(context.Context, search.SearchRequest) (search.SearchResponse, bool, error) {
	c.gets++
	return search.SearchResponse{}, false, nil
}

func (c *countingSearchCache) SetSearchResponse(context.Context, search.SearchRequest, search.SearchResponse) error {
	c.sets++
	return nil
}

type composedSearchGateRepository struct {
	searches int
	items    []repository.FoodItemEntity
	source   repository.FoodItemEntity
}

func (r *composedSearchGateRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	r.searches++
	return append([]repository.FoodItemEntity(nil), r.items...), len(r.items), nil
}

func (r *composedSearchGateRepository) GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.FoodItemEntity, error) {
	return r.source, nil
}

type composedSearchGateCache struct {
	gets  int
	sets  int
	store map[string]search.SearchResponse
}

func (c *composedSearchGateCache) GetSearchResponse(_ context.Context, req search.SearchRequest) (search.SearchResponse, bool, error) {
	c.gets++
	if c.store == nil {
		c.store = map[string]search.SearchResponse{}
	}
	cached, ok := c.store[composedSearchGateCacheKey(req)]
	if !ok {
		return search.SearchResponse{}, false, nil
	}
	cached.Cache = &search.CacheMetadata{Status: search.CacheStatusHit, Namespace: "search", SchemaVersion: "search-response-v2", TTLSeconds: 300}
	return cached, true, nil
}

func (c *composedSearchGateCache) SetSearchResponse(_ context.Context, req search.SearchRequest, response search.SearchResponse) error {
	c.sets++
	if c.store == nil {
		c.store = map[string]search.SearchResponse{}
	}
	c.store[composedSearchGateCacheKey(req)] = response
	return nil
}

func (c *composedSearchGateCache) SearchResponseCacheMetadata(search.SearchRequest, search.CacheStatus) *search.CacheMetadata {
	return &search.CacheMetadata{Status: search.CacheStatusMiss, Namespace: "search", SchemaVersion: "search-response-v2", TTLSeconds: 300}
}

func composedSearchGateCacheKey(req search.SearchRequest) string {
	return string(req.Mode) + "|" + req.Query + "|" + string(rune(req.Page))
}

func TestSearchControllerRemainingFailurePaths(t *testing.T) {
	validBody := []byte(`{"mode":"catalog","query":"apple","page":1}`)

	service := &fakeSearchService{err: search.ErrDailyDietIDRequired}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	resp, err := app.Test(searchHTTPPost(validBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("daily diet id failure = %d", resp.StatusCode)
	}

	autocomplete := &fakeAutocompleteService{err: errors.New("repository failed")}
	app = mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(&fakeSearchService{}).WithAutocompleteService(autocomplete).Routes()})
	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/search/autocomplete?query=apple", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("autocomplete failure = %d", resp.StatusCode)
	}

	direct := fiber.New()
	controller := NewSearchController(&fakeSearchService{})
	direct.Post("/search", controller.Search)
	resp, err = direct.Test(httptest.NewRequest(fiber.MethodPost, "/search", bytes.NewBufferString("{")))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("direct parse failure = %d", resp.StatusCode)
	}
	direct.Get("/autocomplete", controller.Autocomplete)
	resp, err = direct.Test(httptest.NewRequest(fiber.MethodGet, "/autocomplete", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("disabled autocomplete = %d", resp.StatusCode)
	}

	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	history := &fakeSearchHistoryAppender{err: errors.New("history failed")}
	app = mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(&fakeSearchService{response: search.SearchResponse{Page: 1}}).WithSearchHistoryAppender(history).Routes()})
	req := searchHTTPPost(validBody)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError || history.calls != 1 {
		t.Fatalf("history failure = %d calls=%d", resp.StatusCode, history.calls)
	}
}

func TestSearchControllerReturnsCatalogResultsEnvelope(t *testing.T) {
	service := &fakeSearchService{response: search.SearchResponse{
		Items:            []repository.FoodItemEntity{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "Apple", PhysicalState: repository.PhysicalStateSolid}},
		TotalCount:       11,
		Page:             2,
		SimilarityScores: []float64{0},
		SimilarityMetadata: []search.SimilarityMetadata{{
			ItemID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Score:            0.91,
			Tier:             search.SimilarityTierExcellent,
			ImageURL:         "/assets/similarity/excellent.svg",
			MatchingQuantity: 42,
		}},
		Warnings: []string{"excluded allergen dairy"},
	}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{"query": " apple ", "mode": "catalog", "page": 2, "filters": []any{map[string]any{"filterId": "dairy", "kind": "allergen", "include": false}}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusOK || service.calls != 1 || service.request.Page != 2 || len(service.request.Filters) != 1 {
		t.Fatalf("response=%d calls=%d request=%+v envelope=%+v", resp.StatusCode, service.calls, service.request, envelope)
	}
	if envelope.Data["totalCount"].(float64) != 11 || envelope.Data["page"].(float64) != 2 || len(envelope.Data["items"].([]any)) != 1 || len(envelope.Data["warnings"].([]any)) != 1 {
		t.Fatalf("envelope data = %+v", envelope.Data)
	}
	metadata := envelope.Data["similarityMetadata"].([]any)
	if len(metadata) != 1 || metadata[0].(map[string]any)["tier"] != "excellent" || metadata[0].(map[string]any)["matchingQuantity"].(float64) != 42 {
		t.Fatalf("similarity metadata envelope = %+v", envelope.Data["similarityMetadata"])
	}
}

func TestSearchControllerFoodObjectDTOExposesClassificationMacrosAndCalories(t *testing.T) {
	// Implements DESIGN-002 SearchController food-object result contract verification.
	categoryID := uuid.MustParse("70000000-0000-4000-8000-000000000001")
	roleID := uuid.MustParse("70000000-0000-4000-8000-000000000002")
	solidID := uuid.MustParse("70000000-0000-4000-8000-000000000003")
	liquidID := uuid.MustParse("70000000-0000-4000-8000-000000000004")
	service := &fakeSearchService{response: search.SearchResponse{
		Items: []repository.FoodItemEntity{
			{
				ID:             solidID,
				Name:           "Apple",
				PhysicalState:  repository.PhysicalStateSolid,
				MacrosPer100:   repository.MacroValues{Protein: 0.5, Carbohydrates: 14, Fat: 0.3},
				FoodCategories: []repository.ClassificationEntity{{ID: categoryID, Name: "Fruits", Kind: repository.ClassificationKindFoodCategory}},
				CulinaryRoles:  []repository.ClassificationEntity{{ID: roleID, Name: "Snack", Kind: repository.ClassificationKindCulinaryRole}},
			},
			{
				ID:            liquidID,
				Name:          "Almond Milk",
				PhysicalState: repository.PhysicalStateLiquid,
				MacrosPer100:  repository.MacroValues{Protein: 1, Carbohydrates: 2, Fat: 3},
			},
		},
		TotalCount:       2,
		Page:             1,
		SimilarityScores: []float64{0, 0},
		Warnings:         []string{},
	}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "catalog", "page": 1, "filters": []any{}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("response = %d envelope=%+v", resp.StatusCode, envelope)
	}
	items := envelope.Data["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}

	solid := items[0].(map[string]any)
	if solid["id"] != solidID.String() {
		t.Fatalf("solid id = %v", solid["id"])
	}
	if solid["macroBasis"] != "100g" {
		t.Fatalf("solid macroBasis = %v", solid["macroBasis"])
	}
	macros := solid["macros"].(map[string]any)
	protein := macros["protein"].(float64)
	carbs := macros["carbohydrates"].(float64)
	fat := macros["fat"].(float64)
	if protein < 0 || carbs < 0 || fat < 0 {
		t.Fatalf("solid macros negative = %+v", macros)
	}
	if protein != 0.5 || carbs != 14 || fat != 0.3 {
		t.Fatalf("solid macros = %+v", macros)
	}
	calories := solid["calories"].(float64)
	if calories < 0 {
		t.Fatalf("solid calories negative = %v", calories)
	}
	wantSolidCalories := 0.5*4 + 14*4 + 0.3*9
	if calories != wantSolidCalories {
		t.Fatalf("solid calories = %v want %v", calories, wantSolidCalories)
	}
	classifications := solid["classifications"].([]any)
	if len(classifications) != 2 {
		t.Fatalf("solid classifications = %+v", classifications)
	}
	category := classifications[0].(map[string]any)
	if category["id"] != categoryID.String() || category["name"] != "Fruits" || category["kind"] != "food_category" {
		t.Fatalf("solid category = %+v", category)
	}
	role := classifications[1].(map[string]any)
	if role["id"] != roleID.String() || role["name"] != "Snack" || role["kind"] != "culinary_role" {
		t.Fatalf("solid role = %+v", role)
	}
	primary := solid["primaryFoodCategory"].(map[string]any)
	if primary["id"] != categoryID.String() || primary["name"] != "Fruits" || primary["kind"] != "food_category" {
		t.Fatalf("solid primaryFoodCategory = %+v", primary)
	}

	liquid := items[1].(map[string]any)
	if liquid["id"] != liquidID.String() {
		t.Fatalf("liquid id = %v", liquid["id"])
	}
	if liquid["macroBasis"] != "100ml" {
		t.Fatalf("liquid macroBasis = %v", liquid["macroBasis"])
	}
	liquidMacros := liquid["macros"].(map[string]any)
	if liquidMacros["protein"].(float64) != 1 || liquidMacros["carbohydrates"].(float64) != 2 || liquidMacros["fat"].(float64) != 3 {
		t.Fatalf("liquid macros = %+v", liquidMacros)
	}
	if liquidMacros["protein"].(float64) < 0 || liquidMacros["carbohydrates"].(float64) < 0 || liquidMacros["fat"].(float64) < 0 {
		t.Fatalf("liquid macros negative = %+v", liquidMacros)
	}
	liquidCalories := liquid["calories"].(float64)
	if liquidCalories < 0 {
		t.Fatalf("liquid calories negative = %v", liquidCalories)
	}
	if liquidCalories != 1*4+2*4+3*9 {
		t.Fatalf("liquid calories = %v want 39", liquidCalories)
	}
	liquidClassifications := liquid["classifications"].([]any)
	if len(liquidClassifications) != 0 {
		t.Fatalf("liquid classifications = %+v", liquidClassifications)
	}
	if liquid["primaryFoodCategory"] != nil {
		t.Fatalf("liquid primaryFoodCategory = %v want nil", liquid["primaryFoodCategory"])
	}
}

func TestSearchControllerAuthenticatedSuccessAppendsHistoryWithServerUser(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	spoofedUserID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	history := &fakeSearchHistoryAppender{}
	service := &fakeSearchService{response: search.SearchResponse{
		Items:            []repository.FoodItemEntity{{ID: uuid.New(), Name: "Apple", PhysicalState: repository.PhysicalStateSolid}},
		TotalCount:       1,
		Page:             1,
		SimilarityScores: []float64{0},
		Warnings:         []string{},
	}}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(service).WithSearchHistoryAppender(history).Routes()})
	body := searchRequestBody(t, map[string]any{"query": " apple ", "mode": "catalog", "page": 1, "filters": []any{map[string]any{"filterId": "dairy", "kind": "allergen", "include": false}}})
	req := searchHTTPPost(body)
	req.URL.RawQuery = "userId=" + spoofedUserID.String()
	addCookies(req, authCookies)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK || history.calls != 1 || history.userID != userID || history.query != " apple " || history.mode != "catalog" || history.filtersHash == "" {
		t.Fatalf("response=%d history=%+v serverUser=%s spoofed=%s", resp.StatusCode, history, userID, spoofedUserID)
	}
}

func TestSearchControllerRoutesExposeSearchPolicyMetadataAndCSRFExemption(t *testing.T) {
	controller := NewSearchController(&fakeSearchService{}).WithAutocompleteService(&fakeAutocompleteService{}).WithSearchHistoryAppender(&fakeSearchHistoryAppender{})
	routes := controller.Routes()
	if len(routes) != 2 {
		t.Fatalf("routes = %+v", routes)
	}
	searchRoute := routes[0]
	if searchRoute.Method != fiber.MethodPost || searchRoute.Path != "/search" || !searchRoute.OptionalAuth || searchRoute.RequiresCSRF || !searchRoute.ExemptCSRF || searchRoute.RateLimit == nil {
		t.Fatalf("search route policy = %+v", searchRoute)
	}
	autocompleteRoute := routes[1]
	if autocompleteRoute.Method != fiber.MethodGet || autocompleteRoute.Path != "/search/autocomplete" || !autocompleteRoute.OptionalAuth || autocompleteRoute.RequiresCSRF || autocompleteRoute.ExemptCSRF || autocompleteRoute.RateLimit == nil {
		t.Fatalf("autocomplete route policy = %+v", autocompleteRoute)
	}
}

func TestSearchControllerAutocompleteAllowsAnonymousAndReturnsEnvelopeWithCacheMetadata(t *testing.T) {
	autocomplete := &fakeAutocompleteService{response: search.AutocompleteResponse{
		Items: []search.RankedAutocomplete{{
			ItemID:              "00000000-0000-0000-0000-000000000001",
			Label:               "Apple",
			ExactMatch:          true,
			LevenshteinDistance: 0,
			Length:              5,
			Rank:                1,
		}},
		Cache: &search.CacheMetadata{Status: search.CacheStatusMiss, Namespace: "autocomplete", SchemaVersion: "autocomplete-response-v1", TTLSeconds: 120},
	}}
	telemetry := &observability.MemorySink{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Metrics: telemetry, Routes: NewSearchController(&fakeSearchService{}).WithAutocompleteService(autocomplete).Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/search/autocomplete?query=Apple", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusOK || envelope.Status != "ok" || envelope.RequestID == "" || autocomplete.calls != 1 || autocomplete.query != "Apple" || autocomplete.context.UserID != nil {
		t.Fatalf("response=%d envelope=%+v autocomplete=%+v", resp.StatusCode, envelope, autocomplete)
	}
	items := envelope.Data["items"].([]any)
	cacheData := envelope.Data["cache"].(map[string]any)
	if len(items) != 1 || items[0].(map[string]any)["label"] != "Apple" || cacheData["status"] != "miss" || cacheData["schemaVersion"] != "autocomplete-response-v1" {
		t.Fatalf("autocomplete envelope data = %+v", envelope.Data)
	}
	if !hasMetric(telemetry.Metrics, "http_response_total", "/api/v1/search/autocomplete", "200") {
		t.Fatalf("missing route metric: %+v", telemetry.Metrics)
	}
}

func TestSearchControllerAutocompleteUsesServerAuthContextAndIgnoresClientUserID(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	spoofedUserID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	autocomplete := &fakeAutocompleteService{response: search.AutocompleteResponse{Items: []search.RankedAutocomplete{}}}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(&fakeSearchService{}).WithAutocompleteService(autocomplete).Routes()})
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/search/autocomplete?query=Apple&userId="+spoofedUserID.String(), nil)
	addCookies(req, authCookies)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK || autocomplete.context.UserID == nil || *autocomplete.context.UserID != userID {
		t.Fatalf("response=%d context user=%v server=%s spoofed=%s", resp.StatusCode, autocomplete.context.UserID, userID, spoofedUserID)
	}
}

func TestSearchControllerDoesNotPersistAnonymousRejectedOrFailedSearches(t *testing.T) {
	tests := []struct {
		name    string
		service *fakeSearchService
		want    int
		auth    bool
		status  int
	}{
		{
			name: "anonymous success",
			service: &fakeSearchService{response: search.SearchResponse{
				Items:            []repository.FoodItemEntity{{ID: uuid.New(), Name: "Apple", PhysicalState: repository.PhysicalStateSolid}},
				TotalCount:       1,
				Page:             1,
				SimilarityScores: []float64{0},
				Warnings:         []string{},
			}},
			status: fiber.StatusOK,
		},
		{
			name:    "authenticated rejection",
			service: &fakeSearchService{response: search.SearchResponse{Rejection: &search.SearchRejection{Code: "rejected_search", Message: "filters conflict", Field: "filters"}}},
			auth:    true,
			status:  fiber.StatusUnprocessableEntity,
		},
		{
			name:    "authenticated failure",
			service: &fakeSearchService{err: repository.NewError(repository.ErrorKindConnection, "search food items", errors.New("down"))},
			auth:    true,
			status:  fiber.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			userID := uuid.New()
			authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
			history := &fakeSearchHistoryAppender{}
			app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(tt.service).WithSearchHistoryAppender(history).Routes()})
			req := searchHTTPPost(searchRequestBody(t, map[string]any{"query": "apple", "mode": "catalog", "page": 1, "filters": []any{}}))
			if tt.auth {
				addCookies(req, authCookies)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			if resp.StatusCode != tt.status || history.calls != tt.want {
				t.Fatalf("response=%d history calls=%d", resp.StatusCode, history.calls)
			}
		})
	}
}

func TestSearchControllerRealRouteSubstitutionDispatchesToSubstitutionService(t *testing.T) {
	sourceID := uuid.MustParse("60000000-0000-4000-8000-000000000001")
	targetID := uuid.MustParse("60000000-0000-4000-8000-000000000002")
	catalogRepo := &countingSearchRepository{}
	substitutionRepo := substitutionSearchRepository{sourceID: sourceID, targetID: targetID}
	service := search.NewSearchDispatcher(
		search.NewCatalogService(catalogRepo, nil),
		search.NewSubstitutionService(substitutionRepo, nil),
	)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{
		"query":   "",
		"mode":    "substitution",
		"page":    1,
		"filters": []any{},
		"substitutionInputs": []any{map[string]any{
			"foodObjectId": sourceID.String(),
			"quantity":     100,
			"unit":         "ml",
		}},
	})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusOK || envelope.Status != "ok" {
		t.Fatalf("response = %d envelope=%+v", resp.StatusCode, envelope)
	}
	if catalogRepo.calls != 0 {
		t.Fatalf("catalog service handled substitution route calls=%d", catalogRepo.calls)
	}
	metadata := envelope.Data["similarityMetadata"].([]any)
	if len(metadata) != 1 || metadata[0].(map[string]any)["itemId"] != targetID.String() || metadata[0].(map[string]any)["tier"] != "excellent" {
		t.Fatalf("similarity metadata = %+v", metadata)
	}
	summary := envelope.Data["sourceSummary"].(map[string]any)
	if summary["totalGrams"].(float64) != 0 || summary["totalMilliliters"].(float64) != 100 {
		t.Fatalf("source summary amounts = %+v", summary)
	}
	macros := summary["macros"].(map[string]any)
	if macros["protein"].(float64) != 3 || macros["carbohydrates"].(float64) != 5 || macros["fat"].(float64) != 1 {
		t.Fatalf("source summary macros = %+v", summary)
	}
}

func TestSearchControllerProductionPathDailyDietUnavailableReturns422WithoutSideEffects(t *testing.T) {
	repo := &countingSearchRepository{}
	cache := &countingSearchCache{}
	service := search.NewCatalogService(repo, cache)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{
		"query":       "lentil",
		"mode":        "daily_diet_alternative",
		"page":        1,
		"filters":     []any{},
		"dailyDietId": "61e0cae4-0f45-4854-8ac5-b228214cdd1d",
	})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity || envelope.Status != "error" {
		t.Fatalf("response = %d envelope=%+v", resp.StatusCode, envelope)
	}
	rejection := envelope.Data["rejection"].(map[string]any)
	if rejection["code"] != "phase_07_saved_diet_unavailable" || rejection["field"] != "dailyDietId" {
		t.Fatalf("rejection = %+v", rejection)
	}
	if repo.calls != 0 || cache.gets != 0 || cache.sets != 0 {
		t.Fatalf("side effects repo=%d cache gets=%d sets=%d", repo.calls, cache.gets, cache.sets)
	}
}

func TestSearchControllerProductionPathDailyDietMissingIDReturns400WithoutSideEffects(t *testing.T) {
	repo := &countingSearchRepository{}
	cache := &countingSearchCache{}
	service := search.NewCatalogService(repo, cache)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "lentil", "mode": "daily_diet_alternative", "page": 1, "filters": []any{}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || envelope.Error.Code != "validation_failed" {
		t.Fatalf("response = %d envelope=%+v", resp.StatusCode, envelope)
	}
	if repo.calls != 0 || cache.gets != 0 || cache.sets != 0 {
		t.Fatalf("side effects repo=%d cache gets=%d sets=%d", repo.calls, cache.gets, cache.sets)
	}
}

func TestSearchWorkflowIntegrationGateCatalogCacheHistoryAndDailyDiet(t *testing.T) {
	// Implements DESIGN-002 SearchController composed Phase 04 integration gate.
	// Verifies IT-ARCH-002-001.
	// Verifies IT-ARCH-002-003.
	// Verifies ARCH-002.
	// Traces SW-REQ-004, SW-REQ-010, SW-REQ-019, SW-REQ-024, SW-REQ-029.
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	repo := &composedSearchGateRepository{items: []repository.FoodItemEntity{
		{ID: uuid.MustParse("61000000-0000-4000-8000-000000000002"), Name: "Banana", PhysicalState: repository.PhysicalStateSolid},
		{ID: uuid.MustParse("61000000-0000-4000-8000-000000000001"), Name: "Apple", PhysicalState: repository.PhysicalStateSolid},
	}}
	cache := &composedSearchGateCache{}
	history := &fakeSearchHistoryAppender{}
	service := search.NewSearchDispatcher(search.NewCatalogService(repo, cache), search.NewSubstitutionService(repo, cache))
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(service).WithSearchHistoryAppender(history).Routes()})
	body := searchRequestBody(t, map[string]any{"query": " apple ", "mode": "catalog", "page": 1, "filters": []any{}})

	authenticatedReq := searchHTTPPost(body)
	addCookies(authenticatedReq, authCookies)
	resp, err := app.Test(authenticatedReq)
	if err != nil {
		t.Fatal(err)
	}
	envelope := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || repo.searches != 1 || cache.gets != 1 || cache.sets != 1 || history.calls != 1 || history.userID != userID {
		t.Fatalf("miss response=%d repo=%d cache=%+v history=%+v envelope=%+v", resp.StatusCode, repo.searches, cache, history, envelope)
	}
	items := envelope.Data["items"].([]any)
	if items[0].(map[string]any)["name"] != "Apple" {
		t.Fatalf("catalog results were not repository-to-route sorted: %+v", items)
	}
	cacheData := envelope.Data["cache"].(map[string]any)
	if cacheData["status"] != "miss" || cacheData["namespace"] != "search" || cacheData["schemaVersion"] != "search-response-v2" || cacheData["ttlSeconds"] != float64(300) {
		t.Fatalf("cache miss metadata = %+v", cacheData)
	}

	anonymousReq := searchHTTPPost(body)
	resp, err = app.Test(anonymousReq)
	if err != nil {
		t.Fatal(err)
	}
	envelope = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || repo.searches != 1 || cache.gets != 2 || cache.sets != 1 || history.calls != 1 {
		t.Fatalf("hit response=%d repo=%d cache=%+v history calls=%d envelope=%+v", resp.StatusCode, repo.searches, cache, history.calls, envelope)
	}
	cacheData = envelope.Data["cache"].(map[string]any)
	if cacheData["status"] != "hit" || cacheData["schemaVersion"] != "search-response-v2" {
		t.Fatalf("cache hit metadata = %+v", cacheData)
	}

	dailyDietBody := searchRequestBody(t, map[string]any{
		"query":       "lentil",
		"mode":        "daily_diet_alternative",
		"page":        1,
		"filters":     []any{},
		"dailyDietId": "61e0cae4-0f45-4854-8ac5-b228214cdd1d",
	})
	resp, err = app.Test(searchHTTPPost(dailyDietBody))
	if err != nil {
		t.Fatal(err)
	}
	envelope = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	rejection := envelope.Data["rejection"].(map[string]any)
	if resp.StatusCode != fiber.StatusUnprocessableEntity || rejection["code"] != "phase_07_saved_diet_unavailable" || rejection["field"] != "dailyDietId" || history.calls != 1 {
		t.Fatalf("daily diet response=%d rejection=%+v history=%+v", resp.StatusCode, rejection, history)
	}
}

func TestSearchWorkflowIntegrationGateSubstitutionSortsBySimilarity(t *testing.T) {
	// Implements DESIGN-002 SearchController composed substitution integration gate.
	// Verifies IT-ARCH-002-002.
	// Verifies ARCH-002.
	// Verifies ARCH-003.
	// Traces SW-REQ-017, SW-REQ-026, SW-REQ-031.
	sourceID := uuid.MustParse("62000000-0000-4000-8000-000000000001")
	nearID := uuid.MustParse("62000000-0000-4000-8000-000000000002")
	farID := uuid.MustParse("62000000-0000-4000-8000-000000000003")
	repo := &composedSearchGateRepository{
		source: repository.FoodItemEntity{ID: sourceID, Name: "Milk", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1}},
		items: []repository.FoodItemEntity{
			{ID: farID, Name: "Thin Milk", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 2, Carbohydrates: 6, Fat: 1}},
			{ID: nearID, Name: "Soy Milk", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1}},
		},
	}
	service := search.NewSearchDispatcher(search.NewCatalogService(repo, nil), search.NewSubstitutionService(repo, nil))
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{
		"query":   "milk",
		"mode":    "substitution",
		"page":    1,
		"filters": []any{},
		"substitutionInputs": []any{map[string]any{
			"foodObjectId": sourceID.String(),
			"quantity":     100,
			"unit":         "ml",
		}},
	})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)
	items := envelope.Data["items"].([]any)
	scores := envelope.Data["similarityScores"].([]any)
	if resp.StatusCode != fiber.StatusOK || len(items) != 2 || items[0].(map[string]any)["id"] != nearID.String() || scores[0].(float64) < scores[1].(float64) {
		t.Fatalf("substitution response=%d items=%+v scores=%+v", resp.StatusCode, items, scores)
	}
}

func TestSearchControllerReturns422ForRejectedSearch(t *testing.T) {
	service := &fakeSearchService{response: search.SearchResponse{Rejection: &search.SearchRejection{Code: "rejected_search", Message: "filters conflict", Field: "filters"}}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "milk", "mode": "catalog", "page": 1, "filters": []any{}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity || envelope.Status != "error" || envelope.Error == nil || envelope.Error.Code != "rejected_search" || envelope.Error.Category != "validation" {
		t.Fatalf("response = %d envelope=%+v", resp.StatusCode, envelope)
	}
	rejection := envelope.Data["rejection"].(map[string]any)
	if rejection["code"] != "rejected_search" || rejection["field"] != "filters" {
		t.Fatalf("rejection = %+v", rejection)
	}
}

func TestSearchControllerValidationAndRepositoryFailureEnvelopes(t *testing.T) {
	service := &fakeSearchService{err: repository.NewError(repository.ErrorKindConnection, "search food items", errors.New("down"))}
	logs := &observability.MemorySink{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Logs: logs, Routes: NewSearchController(service).Routes()})

	invalidBody := []byte(`{"query":`)
	resp, err := app.Test(searchHTTPPost(invalidBody))
	if err != nil {
		t.Fatal(err)
	}
	envelope := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || envelope.Error.Code != "invalid_json" || envelope.Error.RequestID == "" || envelope.RequestID == "" || service.calls != 0 {
		t.Fatalf("validation response=%d envelope=%+v calls=%d", resp.StatusCode, envelope, service.calls)
	}

	validBody := searchRequestBody(t, map[string]any{"query": "secret raw apple query", "mode": "catalog", "page": 1, "filters": []any{}})
	resp, err = app.Test(searchHTTPPost(validBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope = decodeEnvelope(t, resp.Body)
	if resp.StatusCode != fiber.StatusServiceUnavailable || envelope.Error == nil || envelope.Error.Category != "dependency" || envelope.Error.Code != "dependency_unavailable" || envelope.Error.RequestID == "" {
		t.Fatalf("repository response=%d envelope=%+v", resp.StatusCode, envelope)
	}
	assertLogsDoNotContain(t, logs.Logs, "secret raw apple query")
}

func TestSearchControllerMapsSimilarityUnavailableToFrontendError(t *testing.T) {
	service := &fakeSearchService{err: search.SimilarityUnavailableError{Cause: errors.New("macro engine down")}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{
		"query":   "milk",
		"mode":    "substitution",
		"page":    1,
		"filters": []any{},
		"substitutionInputs": []any{map[string]any{
			"foodObjectId": "60000000-0000-4000-8000-000000000001",
			"quantity":     100,
			"unit":         "g",
		}},
	})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusServiceUnavailable || envelope.Error == nil || envelope.Error.Category != "dependency" || envelope.Error.Code != "similarity_unavailable" || !envelope.Error.Retryable {
		t.Fatalf("similarity response=%d envelope=%+v", resp.StatusCode, envelope)
	}
}

func TestSearchControllerReturnsCacheWarningOnCatalogFallback(t *testing.T) {
	service := &fakeSearchService{response: search.SearchResponse{
		Items:            []repository.FoodItemEntity{{ID: uuid.New(), Name: "Apple", PhysicalState: repository.PhysicalStateSolid}},
		TotalCount:       1,
		Page:             1,
		SimilarityScores: []float64{0},
		Warnings:         []string{search.WarningCacheUnavailable},
	}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "catalog", "page": 1, "filters": []any{}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	warnings := envelope.Data["warnings"].([]any)
	if resp.StatusCode != fiber.StatusOK || len(warnings) != 1 || warnings[0] != search.WarningCacheUnavailable {
		t.Fatalf("cache warning response=%d envelope=%+v", resp.StatusCode, envelope)
	}
}

func assertLogsDoNotContain(t *testing.T, logs []observability.LogEvent, forbidden string) {
	t.Helper()
	payload, err := json.Marshal(logs)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte(forbidden)) {
		t.Fatalf("logs leaked forbidden query %q: %s", forbidden, payload)
	}
}

func searchRequestBody(t *testing.T, value map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func searchHTTPPost(body []byte) *http.Request {
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func hasMetric(metrics []observability.MetricPoint, name string, route string, status string) bool {
	for _, metric := range metrics {
		if metric.Name == name && metric.Labels["route"] == route && metric.Labels["status"] == status {
			return true
		}
	}
	return false
}

type fakeEntitlementManager struct {
	decision EntitlementDecision
	err      error
	feature  string
}

func (m *fakeEntitlementManager) CheckEntitlement(ctx context.Context, userID uuid.UUID, feature string) (EntitlementDecision, error) {
	m.feature = feature
	return m.decision, m.err
}

type fakeUsageLimiter struct {
	checkErr  error
	recordErr error
	checked   int
	recorded  int
}

func (l *fakeUsageLimiter) CheckUsageLimit(ctx context.Context, userID uuid.UUID, feature string) error {
	l.checked++
	return l.checkErr
}

func (l *fakeUsageLimiter) RecordUsage(ctx context.Context, userID uuid.UUID, feature string) error {
	l.recorded++
	return l.recordErr
}

func TestSearchControllerAnonymousCatalogSearchWorksWithoutEntitlement(t *testing.T) {
	// Implements DESIGN-002 SearchController phase 06 entitlement gate.
	service := &fakeSearchService{response: search.SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: 1}}
	history := &fakeSearchHistoryAppender{}
	entitlements := &fakeEntitlementManager{decision: EntitlementDecision{Allowed: true}}
	usage := &fakeUsageLimiter{}

	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).WithSearchHistoryAppender(history).WithEntitlementGate(entitlements, usage).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "catalog", "page": 1, "filters": []any{}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK || service.calls != 1 || history.calls != 0 || usage.checked != 0 || usage.recorded != 0 {
		t.Fatalf("anonymous catalog should bypass entitlement side effects: response=%d service=%d history=%d check=%d record=%d", resp.StatusCode, service.calls, history.calls, usage.checked, usage.recorded)
	}
}

func TestSearchControllerAnonymousNonCatalogBlocked(t *testing.T) {
	service := &fakeSearchService{}
	entitlements := &fakeEntitlementManager{}
	usage := &fakeUsageLimiter{}

	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewSearchController(service).WithEntitlementGate(entitlements, usage).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "substitution", "page": 1, "filters": []any{}, "substitutionInputs": []any{map[string]any{"foodObjectId": "61000000-0000-4000-8000-000000000002", "quantity": 100, "unit": "g"}}})

	resp, err := app.Test(searchHTTPPost(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("anonymous non-catalog should block: response=%d service=%d", resp.StatusCode, service.calls)
	}
}

func TestSearchControllerFreeSingleSubstitutionWithinUsageLimits(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)

	service := &fakeSearchService{response: search.SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: 1}}
	history := &fakeSearchHistoryAppender{}
	entitlements := &fakeEntitlementManager{decision: EntitlementDecision{Allowed: true}}
	usage := &fakeUsageLimiter{}

	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(service).WithSearchHistoryAppender(history).WithEntitlementGate(entitlements, usage).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "substitution", "page": 1, "filters": []any{}, "substitutionInputs": []any{map[string]any{"foodObjectId": uuid.New().String(), "quantity": 100, "unit": "g"}}})

	req := searchHTTPPost(body)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK || service.calls != 1 || history.calls != 1 || entitlements.feature != "substitution_single" || usage.checked != 1 || usage.recorded != 1 {
		t.Fatalf("allowed single substitution failure: response=%d service=%d feature=%s check=%d record=%d", resp.StatusCode, service.calls, entitlements.feature, usage.checked, usage.recorded)
	}
}

func TestSearchControllerFreeMultiSubstitutionReturnsStableError(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)

	service := &fakeSearchService{}
	history := &fakeSearchHistoryAppender{}
	entitlements := &fakeEntitlementManager{decision: EntitlementDecision{Allowed: false, Code: "entitlement_required", Message: "upgrade required"}}
	usage := &fakeUsageLimiter{}

	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(service).WithSearchHistoryAppender(history).WithEntitlementGate(entitlements, usage).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "substitution", "page": 1, "filters": []any{}, "substitutionInputs": []any{
		map[string]any{"foodObjectId": uuid.New().String(), "quantity": 100, "unit": "g"},
		map[string]any{"foodObjectId": uuid.New().String(), "quantity": 50, "unit": "ml"},
	}})

	req := searchHTTPPost(body)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusForbidden || service.calls != 0 || history.calls != 0 || entitlements.feature != "substitution_multi" || usage.checked != 0 || usage.recorded != 0 {
		t.Fatalf("blocked multi substitution failure: response=%d feature=%s", resp.StatusCode, entitlements.feature)
	}
	if envelope.Error == nil || envelope.Error.Code != "entitlement_required" {
		t.Fatalf("expected entitlement error code: %+v", envelope)
	}
}

func TestSearchControllerDailyDietBlockedWithoutEntitlement(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)

	service := &fakeSearchService{}
	history := &fakeSearchHistoryAppender{}
	entitlements := &fakeEntitlementManager{decision: EntitlementDecision{Allowed: false, Code: "entitlement_required", Message: "upgrade required"}}
	usage := &fakeUsageLimiter{}

	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(service).WithSearchHistoryAppender(history).WithEntitlementGate(entitlements, usage).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "lentil", "mode": "daily_diet_alternative", "page": 1, "filters": []any{}, "dailyDietId": uuid.New().String()})

	req := searchHTTPPost(body)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden || service.calls != 0 || history.calls != 0 || usage.checked != 0 || usage.recorded != 0 {
		t.Fatalf("blocked daily diet alternative failure: response=%d service=%d", resp.StatusCode, service.calls)
	}
}

func TestSearchControllerUsageLimitExceeded(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)

	service := &fakeSearchService{}
	history := &fakeSearchHistoryAppender{}
	entitlements := &fakeEntitlementManager{decision: EntitlementDecision{Allowed: true}}
	usage := &fakeUsageLimiter{checkErr: errors.New("usage limit exceeded")}

	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: NewSearchController(service).WithSearchHistoryAppender(history).WithEntitlementGate(entitlements, usage).Routes()})
	body := searchRequestBody(t, map[string]any{"query": "apple", "mode": "substitution", "page": 1, "filters": []any{}, "substitutionInputs": []any{map[string]any{"foodObjectId": uuid.New().String(), "quantity": 100, "unit": "g"}}})

	req := searchHTTPPost(body)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusForbidden || service.calls != 0 || history.calls != 0 || usage.checked != 1 || usage.recorded != 0 {
		t.Fatalf("usage limit exceeded failure: response=%d service=%d checked=%d", resp.StatusCode, service.calls, usage.checked)
	}
	if envelope.Error == nil || envelope.Error.Code != "usage_limit_exceeded" {
		t.Fatalf("expected usage limit error code: %+v", envelope)
	}
}

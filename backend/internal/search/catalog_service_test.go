package search

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 SearchController catalog orchestration verification.

type catalogRepositoryStub struct {
	items []repository.FoodItemEntity
	total int
	query repository.RepositoryQuery
	calls int
	err   error
}

func (r *catalogRepositoryStub) Search(_ context.Context, q repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	r.calls++
	r.query = q
	return r.items, r.total, r.err
}

type searchCacheStub struct {
	response searchCacheEntry
	hit      bool
	getErr   error
	setErr   error
	gets     int
	sets     int
	setReq   SearchRequest
}

type searchCacheEntry struct {
	value SearchResponse
}

func (c *searchCacheStub) GetSearchResponse(context.Context, SearchRequest) (SearchResponse, bool, error) {
	c.gets++
	return c.response.value, c.hit, c.getErr
}

func (c *searchCacheStub) SetSearchResponse(_ context.Context, req SearchRequest, response SearchResponse) error {
	c.sets++
	c.setReq = req
	c.response.value = response
	return c.setErr
}

func (c *searchCacheStub) SearchResponseCacheMetadata(SearchRequest, CacheStatus) *CacheMetadata {
	return &CacheMetadata{Status: CacheStatusMiss, Namespace: "search", SchemaVersion: "search-response-v1", TTLSeconds: 300}
}

func TestCatalogServiceSearchFiltersPaginationSortingWarningsAndCacheMiss(t *testing.T) {
	categoryID := uuid.New()
	excludedRoleID := uuid.New()
	repo := &catalogRepositoryStub{
		total: 3,
		items: []repository.FoodItemEntity{
			{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Name: "Zucchini", PhysicalState: repository.PhysicalStateSolid},
			{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "Apple", PhysicalState: repository.PhysicalStateSolid},
		},
	}
	cache := &searchCacheStub{}
	service := NewCatalogService(repo, cache)

	response, err := service.Search(context.Background(), SearchRequest{
		Query: "  Apple   Bowl ",
		Mode:  SearchModeCatalog,
		Page:  2,
		Filters: []SearchFilter{
			{FilterID: categoryID.String(), Kind: SearchFilterKindFoodCategory, Include: true},
			{FilterID: excludedRoleID.String(), Kind: SearchFilterKindCulinaryRole, Include: false},
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if repo.calls != 1 || cache.gets != 1 || cache.sets != 1 {
		t.Fatalf("calls repo=%d cache gets=%d sets=%d", repo.calls, cache.gets, cache.sets)
	}
	if repo.query.Name != "apple bowl" || repo.query.Limit != PageSize || repo.query.Offset != 10 || len(repo.query.FoodCategoryIDs) != 1 || len(repo.query.ExcludedCulinaryRoleIDs) != 1 {
		t.Fatalf("repository query = %+v", repo.query)
	}
	if response.TotalCount != 3 || response.Page != 2 || len(response.Items) != 2 || response.Items[0].Name != "Apple" || len(response.SimilarityScores) != 2 {
		t.Fatalf("response = %+v", response)
	}
	if len(response.Warnings) != 1 || response.Warnings[0] == "" {
		t.Fatalf("warnings = %#v", response.Warnings)
	}
	if response.Cache == nil || response.Cache.Status != CacheStatusMiss || response.Cache.Namespace != "search" || response.Cache.SchemaVersion != "search-response-v1" || response.Cache.TTLSeconds != 300 {
		t.Fatalf("cache miss metadata = %+v", response.Cache)
	}
	if cache.setReq.Query != "apple bowl" || cache.setReq.Page != 2 {
		t.Fatalf("cache request = %+v", cache.setReq)
	}
}

func TestCatalogServiceSearchCacheHitBypassesRepository(t *testing.T) {
	repo := &catalogRepositoryStub{}
	cache := &searchCacheStub{hit: true, response: searchCacheEntry{value: SearchResponse{Items: []repository.FoodItemEntity{{ID: uuid.New(), Name: "Cached"}}, TotalCount: 1, Page: 1, SimilarityScores: []float64{0}, Warnings: []string{}}}}
	service := NewCatalogService(repo, cache)

	response, err := service.Search(context.Background(), SearchRequest{Query: "cached", Mode: SearchModeCatalog, Page: 1})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if repo.calls != 0 || cache.gets != 1 || cache.sets != 0 || response.Items[0].Name != "Cached" {
		t.Fatalf("cache hit repo=%d gets=%d sets=%d response=%+v", repo.calls, cache.gets, cache.sets, response)
	}
}

func TestCatalogServiceSearchDailyDietUnavailableBypassesRepositoryAndCache(t *testing.T) {
	dailyDietID := uuid.MustParse("61e0cae4-0f45-4854-8ac5-b228214cdd1d")
	repo := &catalogRepositoryStub{}
	cache := &searchCacheStub{}
	service := NewCatalogService(repo, cache)

	response, err := service.Search(context.Background(), SearchRequest{
		Query:       "lentil",
		Mode:        SearchModeDailyDietAlternative,
		Page:        1,
		DailyDietID: &dailyDietID,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if response.Rejection == nil || response.Rejection.Code != "phase_07_saved_diet_unavailable" || response.Rejection.Field != "dailyDietId" {
		t.Fatalf("daily diet rejection = %+v", response.Rejection)
	}
	if repo.calls != 0 || cache.gets != 0 || cache.sets != 0 {
		t.Fatalf("side effects repo=%d cache gets=%d sets=%d", repo.calls, cache.gets, cache.sets)
	}
}

func TestCatalogServiceSearchDailyDietMissingIDBypassesRepositoryAndCache(t *testing.T) {
	repo := &catalogRepositoryStub{}
	cache := &searchCacheStub{}
	service := NewCatalogService(repo, cache)

	if _, err := service.Search(context.Background(), SearchRequest{Query: "lentil", Mode: SearchModeDailyDietAlternative, Page: 1}); err == nil {
		t.Fatal("Search() accepted daily diet alternative without dailyDietId")
	}
	if repo.calls != 0 || cache.gets != 0 || cache.sets != 0 {
		t.Fatalf("side effects repo=%d cache gets=%d sets=%d", repo.calls, cache.gets, cache.sets)
	}
}

func TestCatalogServiceSearchEmptyResultsAndPageBoundary(t *testing.T) {
	repo := &catalogRepositoryStub{items: []repository.FoodItemEntity{}, total: 0}
	service := NewCatalogService(repo, nil)

	response, err := service.Search(context.Background(), SearchRequest{Query: "missing", Mode: SearchModeCatalog, Page: 1})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if response.TotalCount != 0 || len(response.Items) != 0 || response.Page != 1 || repo.query.Offset != 0 {
		t.Fatalf("empty response = %+v query=%+v", response, repo.query)
	}
}

func TestCatalogServiceSearchRejectionAndRepositoryError(t *testing.T) {
	service := NewCatalogService(&catalogRepositoryStub{}, nil)
	response, err := service.Search(context.Background(), SearchRequest{Query: "milk", Mode: SearchModeCatalog, Page: 1, Filters: []SearchFilter{
		{FilterID: "dairy_free", Kind: SearchFilterKindDietaryPreset, Include: false},
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Include: true},
	}})
	if err != nil || response.Rejection == nil || response.Rejection.Code != "rejected_search" {
		t.Fatalf("rejection response=%+v err=%v", response, err)
	}

	wantErr := errors.New("database down")
	repo := &catalogRepositoryStub{err: wantErr}
	service = NewCatalogService(repo, nil)
	if _, err := service.Search(context.Background(), SearchRequest{Query: "apple", Mode: SearchModeCatalog, Page: 1}); !errors.Is(err, wantErr) {
		t.Fatalf("repository error = %v, want %v", err, wantErr)
	}
}

func TestCatalogServiceSearchReturnsCacheUnavailableWarningOnFallback(t *testing.T) {
	repo := &catalogRepositoryStub{items: []repository.FoodItemEntity{{ID: uuid.New(), Name: "Apple", PhysicalState: repository.PhysicalStateSolid}}, total: 1}
	cache := &searchCacheStub{getErr: errors.New("redis down"), setErr: errors.New("redis still down")}

	response, err := NewCatalogService(repo, cache).Search(context.Background(), SearchRequest{Query: "apple", Mode: SearchModeCatalog, Page: 1})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if repo.calls != 1 || cache.gets != 1 || cache.sets != 1 || len(response.Items) != 1 {
		t.Fatalf("fallback failed response=%+v repo=%d cache gets=%d sets=%d", response, repo.calls, cache.gets, cache.sets)
	}
	if got := countWarnings(response.Warnings, WarningCacheUnavailable); got != 1 {
		t.Fatalf("cache warning count = %d warnings=%#v", got, response.Warnings)
	}
	if response.Cache != nil {
		t.Fatalf("cache write failure advertised metadata = %+v", response.Cache)
	}
}

func countWarnings(warnings []string, want string) int {
	count := 0
	for _, warning := range warnings {
		if warning == want {
			count++
		}
	}
	return count
}

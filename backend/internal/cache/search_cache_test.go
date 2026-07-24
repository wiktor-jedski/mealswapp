package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// Implements DESIGN-011 RedisCache search-cache verification.

func TestBuildSearchCacheKeyIsDeterministicForReorderedFilters(t *testing.T) {
	left := BuildSearchCacheKey(searchRequest([]search.SearchFilter{
		{FilterID: "Vegetable", Kind: search.SearchFilterKindFoodCategory, Include: true},
		{FilterID: "peanut", Kind: search.SearchFilterKindAllergen, Include: false},
	}))
	right := BuildSearchCacheKey(searchRequest([]search.SearchFilter{
		{FilterID: "peanut", Kind: search.SearchFilterKindAllergen, Include: false},
		{FilterID: " vegetable ", Kind: search.SearchFilterKindFoodCategory, Include: true},
	}))

	if left != right {
		t.Fatalf("reordered filter keys differ: %q != %q", left.String(), right.String())
	}
}

func TestBuildSearchCacheKeySeparatesChangedInputs(t *testing.T) {
	base := searchRequest(nil)
	changedQuery := base
	changedQuery.Query = "lentil"
	changedMode := base
	changedMode.Mode = search.SearchModeSubstitution
	changedPage := base
	changedPage.Page = 3
	changedSubstitution := base
	changedSubstitution.SubstitutionInputs = []search.SubstitutionInput{{
		FoodObjectID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Quantity:     1,
		Unit:         "gram",
	}}

	baseKey := BuildSearchCacheKey(base)
	for name, req := range map[string]search.SearchRequest{
		"query":        changedQuery,
		"mode":         changedMode,
		"page":         changedPage,
		"substitution": changedSubstitution,
	} {
		if key := BuildSearchCacheKey(req); key == baseKey {
			t.Fatalf("%s change reused key %q", name, key.String())
		}
	}
}

func TestCacheKeysIncludeSchemaVersionAndHideRawPII(t *testing.T) {
	req := searchRequest(nil)
	req.Query = "Alice private dinner"
	key := BuildSearchCacheKey(req)
	raw := key.String()

	if !strings.Contains(raw, SearchSchemaVersion) {
		t.Fatalf("key %q does not include schema version %q", raw, SearchSchemaVersion)
	}
	for _, forbidden := range []string{"Alice", "alice", "private", "dinner"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("raw key %q exposes %q", raw, forbidden)
		}
	}
	if len(key.ID) != 64 {
		t.Fatalf("hash id length = %d, want sha256 hex", len(key.ID))
	}
}

func TestAutocompleteAndSimilarityKeysAreDeterministic(t *testing.T) {
	autoLeft := BuildAutocompleteCacheKey("  Fresh   TOMATO ")
	autoRight := BuildAutocompleteCacheKey("fresh tomato")
	if autoLeft != autoRight {
		t.Fatalf("normalized autocomplete keys differ: %q != %q", autoLeft.String(), autoRight.String())
	}

	inputA := search.SubstitutionInput{FoodObjectID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Quantity: 2, Unit: "g"}
	inputB := search.SubstitutionInput{FoodObjectID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Quantity: 1, Unit: "g"}
	simLeft := BuildSimilarityCacheKey([]search.SubstitutionInput{inputA, inputB})
	simRight := BuildSimilarityCacheKey([]search.SubstitutionInput{inputB, inputA})
	if simLeft != simRight {
		t.Fatalf("reordered similarity keys differ: %q != %q", simLeft.String(), simRight.String())
	}
	changed := BuildSimilarityCacheKey([]search.SubstitutionInput{{FoodObjectID: inputB.FoodObjectID, Quantity: 3, Unit: inputB.Unit}, inputA})
	if changed == simLeft {
		t.Fatal("changed similarity quantity reused key")
	}
	mealInput := inputA
	mealInput.FoodObjectType = repository.FoodObjectTypeMeal
	if BuildSimilarityCacheKey([]search.SubstitutionInput{mealInput}) == BuildSimilarityCacheKey([]search.SubstitutionInput{inputA}) {
		t.Fatal("meal and food-item inputs reused a similarity key")
	}
}

func TestSetRedisAppliesTTLAndGetRedisReportsHitAndMiss(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	key := BuildSearchCacheKey(searchRequest(nil))
	value := search.SearchResponse{TotalCount: 7, Page: 1}

	if _, hit, err := GetRedis[search.SearchResponse](ctx, store, key); err != nil || hit {
		t.Fatalf("initial get hit=%v err=%v, want miss", hit, err)
	}
	if err := SetRedis(ctx, store, key, value, 90*time.Second); err != nil {
		t.Fatalf("SetRedis() error = %v", err)
	}
	if store.ttls[key.String()] != 90*time.Second {
		t.Fatalf("ttl = %v, want 90s", store.ttls[key.String()])
	}
	got, hit, err := GetRedis[search.SearchResponse](ctx, store, key)
	if err != nil || !hit {
		t.Fatalf("cached get hit=%v err=%v", hit, err)
	}
	if got.TotalCount != value.TotalCount || got.Page != value.Page {
		t.Fatalf("cached value = %+v", got)
	}
}

func TestRedisHelpersHandleNilStoreBadJSONAndZeroTTL(t *testing.T) {
	ctx := context.Background()
	key := BuildSearchCacheKey(searchRequest(nil))
	if _, hit, err := GetRedis[search.SearchResponse](ctx, nil, key); err != nil || hit {
		t.Fatalf("nil store get hit=%v err=%v, want miss", hit, err)
	}

	store := &memoryStore{values: map[string]string{key.String(): "not json"}, ttls: map[string]time.Duration{}}
	if _, hit, err := GetRedis[search.SearchResponse](ctx, store, key); err == nil || hit {
		t.Fatalf("bad json get hit=%v err=%v, want decode error", hit, err)
	}
	if err := SetRedis(ctx, store, key, search.SearchResponse{TotalCount: 1}, 0); err != nil {
		t.Fatalf("zero ttl SetRedis() error = %v", err)
	}
	if store.ttls[key.String()] != 0 {
		t.Fatalf("zero ttl set wrote ttl %v", store.ttls[key.String()])
	}
	if err := SetRedis(ctx, store, key, make(chan int), time.Minute); err == nil {
		t.Fatal("SetRedis accepted non-json value")
	}
}

func TestGetRedisReturnsRedisFailureForCallerFallback(t *testing.T) {
	want := errors.New("redis down")
	_, hit, err := GetRedis[search.SearchResponse](context.Background(), failingStore{err: want}, BuildSearchCacheKey(searchRequest(nil)))
	if !errors.Is(err, want) || hit {
		t.Fatalf("hit=%v err=%v, want redis failure for fallback", hit, err)
	}
}

func TestGoRedisStoreReturnsClientFailures(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	defer client.Close()
	store := GoRedisStore{Client: client}

	if _, err := store.Get(ctx, "missing"); err == nil {
		t.Fatal("GoRedisStore.Get() error = nil, want connection error")
	}
	if err := store.Set(ctx, "key", "value", time.Second); err == nil {
		t.Fatal("GoRedisStore.Set() error = nil, want connection error")
	}
}

func TestSearchCacheMetadataDoesNotExposeRawKey(t *testing.T) {
	key := BuildSearchCacheKey(searchRequest(nil))
	metadata := SearchCacheMetadata(key, search.CacheStatusHit, 2*time.Minute)
	if metadata.Status != search.CacheStatusHit || metadata.Namespace != string(RedisNamespaceSearch) || metadata.SchemaVersion != SearchSchemaVersion || metadata.TTLSeconds != 120 {
		t.Fatalf("metadata = %+v", metadata)
	}
	if strings.Contains(metadata.Namespace, key.ID) || strings.Contains(metadata.SchemaVersion, key.ID) {
		t.Fatalf("metadata exposes raw key id: %+v", metadata)
	}
}

func TestSearchCacheKeyIncludesDailyDietIDAndSortsTieBreakers(t *testing.T) {
	dailyDietID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	base := searchRequest([]search.SearchFilter{
		{FilterID: "same", Kind: search.SearchFilterKindFoodCategory, Include: true},
		{FilterID: "same", Kind: search.SearchFilterKindFoodCategory, Include: false},
	})
	withDailyDiet := base
	withDailyDiet.DailyDietID = &dailyDietID
	if BuildSearchCacheKey(base) == BuildSearchCacheKey(withDailyDiet) {
		t.Fatal("daily diet id did not change key")
	}

	inputID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	left := BuildSimilarityCacheKey([]search.SubstitutionInput{
		{FoodObjectID: inputID, Quantity: 2, Unit: "g"},
		{FoodObjectID: inputID, Quantity: 1, Unit: "g"},
		{FoodObjectID: inputID, Quantity: 1, Unit: "ml"},
	})
	right := BuildSimilarityCacheKey([]search.SubstitutionInput{
		{FoodObjectID: inputID, Quantity: 1, Unit: "ml"},
		{FoodObjectID: inputID, Quantity: 1, Unit: "g"},
		{FoodObjectID: inputID, Quantity: 2, Unit: "g"},
	})
	if left != right {
		t.Fatalf("similarity tie-breaker keys differ: %q != %q", left.String(), right.String())
	}
}

func TestCacheResponseMetadataContract(t *testing.T) {
	payload, err := json.Marshal(search.SearchResponse{
		TotalCount: 1,
		Page:       1,
		Cache: &search.CacheMetadata{
			Status:        search.CacheStatusMiss,
			Namespace:     string(RedisNamespaceSearch),
			SchemaVersion: SearchSchemaVersion,
			TTLSeconds:    int64(DefaultSearchTTL / time.Second),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"cache"`) || strings.Contains(string(payload), `"Cache"`) || !strings.Contains(string(payload), SearchSchemaVersion) {
		t.Fatalf("cache metadata payload = %s", payload)
	}
}

func TestGetOrLoadSearchResponseAttachesHitMissMetadataAndStoresWithTTL(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	req := searchRequest(nil)
	key := BuildSearchCacheKey(req)
	loadCalls := 0

	miss, err := GetOrLoadSearchResponse(ctx, store, req, DefaultSearchTTL, func(context.Context) (search.SearchResponse, error) {
		loadCalls++
		return search.SearchResponse{TotalCount: 2, Page: 1}, nil
	})
	if err != nil {
		t.Fatalf("miss GetOrLoadSearchResponse() error = %v", err)
	}
	if loadCalls != 1 || miss.Cache == nil || miss.Cache.Status != search.CacheStatusMiss {
		t.Fatalf("miss response loadCalls=%d cache=%+v", loadCalls, miss.Cache)
	}
	if store.ttls[key.String()] != DefaultSearchTTL {
		t.Fatalf("stored ttl = %v, want %v", store.ttls[key.String()], DefaultSearchTTL)
	}
	if strings.Contains(store.values[key.String()], `"Cache"`) {
		t.Fatalf("stored response included transient cache metadata: %s", store.values[key.String()])
	}

	hit, err := GetOrLoadSearchResponse(ctx, store, req, DefaultSearchTTL, func(context.Context) (search.SearchResponse, error) {
		loadCalls++
		return search.SearchResponse{}, nil
	})
	if err != nil {
		t.Fatalf("hit GetOrLoadSearchResponse() error = %v", err)
	}
	if loadCalls != 1 || hit.Cache == nil || hit.Cache.Status != search.CacheStatusHit || hit.TotalCount != 2 {
		t.Fatalf("hit response loadCalls=%d response=%+v", loadCalls, hit)
	}
}

// Implements DESIGN-009 TagManager and DESIGN-011 RedisCache in-flight invalidation verification.
func TestInFlightSearchMissCannotRepopulateAfterClassificationInvalidation(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	generation := &controlledClassificationGeneration{store: store}
	responseStore := SearchResponseStore{Store: store, Generation: generation}
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	req := searchRequest(nil)
	repository := &blockingCatalogRepository{started: started, release: release}

	go func() {
		_, err := search.NewCatalogService(repository, responseStore).Search(ctx, req)
		done <- err
	}()
	<-started
	generation.Advance()
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("in-flight search error = %v", err)
	}
	if _, hit, _, err := responseStore.GetSearchResponse(ctx, req); err != nil || hit {
		t.Fatalf("stale cache read hit=%v err=%v", hit, err)
	}
}

func TestSearchResponseStoreGetSetUsesDefaultTTLAndStripsTransientMetadata(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	responseStore := SearchResponseStore{Store: store}
	req := searchRequest(nil)
	key := searchCacheKeyForGeneration(BuildSearchCacheKey(req), 0)

	miss, hit, token, err := responseStore.GetSearchResponse(ctx, req)
	if err != nil || hit || miss.Cache != nil {
		t.Fatalf("initial get hit=%v err=%v response=%+v", hit, err, miss)
	}
	if stored, err := responseStore.SetSearchResponse(ctx, req, search.SearchResponse{
		TotalCount: 4,
		Page:       1,
		Cache:      &search.CacheMetadata{Status: search.CacheStatusMiss, Namespace: "search", SchemaVersion: SearchSchemaVersion, TTLSeconds: 300},
	}, token); err != nil || !stored {
		t.Fatalf("SetSearchResponse() error = %v", err)
	}
	if store.ttls[key.String()] != DefaultSearchTTL {
		t.Fatalf("ttl = %v, want %v", store.ttls[key.String()], DefaultSearchTTL)
	}
	if strings.Contains(store.values[key.String()], `"Cache"`) {
		t.Fatalf("stored response included transient cache metadata: %s", store.values[key.String()])
	}

	got, hit, _, err := responseStore.GetSearchResponse(ctx, req)
	if err != nil || !hit {
		t.Fatalf("cached get hit=%v err=%v", hit, err)
	}
	if got.TotalCount != 4 || got.Cache == nil || got.Cache.Status != search.CacheStatusHit || got.Cache.TTLSeconds != int64(DefaultSearchTTL/time.Second) {
		t.Fatalf("cached response = %+v", got)
	}
}

func TestSearchResponseStoreSimilarityCalculationUsesNamespaceSchemaAndTTL(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	responseStore := SearchResponseStore{Store: store, SimilarityTTL: 42 * time.Second}
	inputs := []search.SubstitutionInput{{
		FoodObjectID: uuid.MustParse("77777777-7777-4777-8777-777777777777"),
		Quantity:     100,
		Unit:         "g",
	}}
	calculation := search.SimilarityCalculation{
		Results:     []search.SimilarityResult{{ItemID: uuid.MustParse("88888888-8888-4888-8888-888888888888"), Score: 0.91}},
		Diagnostics: []search.SimilarityDiagnostic{{ItemID: uuid.MustParse("99999999-9999-4999-8999-999999999999"), Code: "below_threshold"}},
	}
	key := similarityCacheKeyForGeneration(BuildSimilarityCacheKey(inputs), 0)

	stored, err := responseStore.SetSimilarityCalculation(ctx, inputs, calculation, search.SimilarityCalculationCacheToken{})
	if err != nil || !stored {
		t.Fatalf("SetSimilarityCalculation() error = %v", err)
	}
	if key.Namespace != RedisNamespaceSimilarity || key.Version != SimilaritySchemaVersion+"-food-data-0" {
		t.Fatalf("similarity key = %+v", key)
	}
	if store.ttls[key.String()] != 42*time.Second {
		t.Fatalf("ttl = %v, want 42s", store.ttls[key.String()])
	}
	got, hit, _, err := responseStore.GetSimilarityCalculation(ctx, inputs)
	if err != nil || !hit {
		t.Fatalf("GetSimilarityCalculation() hit=%v err=%v", hit, err)
	}
	if len(got.Results) != 1 || got.Results[0].Score != 0.91 || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "below_threshold" {
		t.Fatalf("cached calculation = %+v", got)
	}
	metadata := responseStore.SimilarityCalculationCacheMetadata(inputs, search.CacheStatusMiss)
	if metadata == nil || metadata.Status != search.CacheStatusMiss || metadata.Namespace != string(RedisNamespaceSimilarity) || metadata.SchemaVersion != SimilaritySchemaVersion || metadata.TTLSeconds != 42 {
		t.Fatalf("similarity cache metadata = %+v", metadata)
	}
}

func TestGetOrLoadAutocompleteResponseAttachesMetadata(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	loadCalls := 0

	miss, err := GetOrLoadAutocompleteResponse(ctx, store, " tomato ", DefaultAutocompleteTTL, func(context.Context) (search.AutocompleteResponse, error) {
		loadCalls++
		return search.AutocompleteResponse{Items: []search.RankedAutocomplete{{Label: "Tomato", Rank: 1}}}, nil
	})
	if err != nil {
		t.Fatalf("miss GetOrLoadAutocompleteResponse() error = %v", err)
	}
	if miss.Cache == nil || miss.Cache.Status != search.CacheStatusMiss || miss.Cache.Namespace != string(RedisNamespaceAutocomplete) {
		t.Fatalf("miss cache metadata = %+v", miss.Cache)
	}

	hit, err := GetOrLoadAutocompleteResponse(ctx, store, "tomato", DefaultAutocompleteTTL, func(context.Context) (search.AutocompleteResponse, error) {
		loadCalls++
		return search.AutocompleteResponse{}, nil
	})
	if err != nil {
		t.Fatalf("hit GetOrLoadAutocompleteResponse() error = %v", err)
	}
	if loadCalls != 1 || hit.Cache == nil || hit.Cache.Status != search.CacheStatusHit || len(hit.Items) != 1 {
		t.Fatalf("hit response loadCalls=%d response=%+v", loadCalls, hit)
	}
}

func TestGetOrLoadFallsBackWhenRedisFails(t *testing.T) {
	ctx := context.Background()
	loadCalls := 0
	response, err := GetOrLoadSearchResponse(ctx, failingStore{err: errors.New("redis down")}, searchRequest(nil), DefaultSearchTTL, func(context.Context) (search.SearchResponse, error) {
		loadCalls++
		return search.SearchResponse{TotalCount: 5}, nil
	})
	if err != nil {
		t.Fatalf("GetOrLoadSearchResponse() error = %v", err)
	}
	if loadCalls != 1 || response.Cache == nil || response.Cache.Status != search.CacheStatusMiss || response.TotalCount != 5 {
		t.Fatalf("fallback response loadCalls=%d response=%+v", loadCalls, response)
	}
}

func TestGetOrLoadSimilarityResultsCachesAndFallsBack(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{values: map[string]string{}, ttls: map[string]time.Duration{}}
	input := []search.SubstitutionInput{{
		FoodObjectID: uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		Quantity:     1,
		Unit:         "gram",
	}}
	result := []search.SimilarityResult{{ItemID: uuid.MustParse("66666666-6666-6666-6666-666666666666"), Score: 0.9}}
	loadCalls := 0

	miss, missMetadata, err := GetOrLoadSimilarityResults(ctx, store, input, DefaultSimilarityTTL, func(context.Context) ([]search.SimilarityResult, error) {
		loadCalls++
		return result, nil
	})
	if err != nil {
		t.Fatalf("miss GetOrLoadSimilarityResults() error = %v", err)
	}
	if loadCalls != 1 || missMetadata.Status != search.CacheStatusMiss || len(miss) != 1 {
		t.Fatalf("miss loadCalls=%d metadata=%+v results=%+v", loadCalls, missMetadata, miss)
	}

	hit, hitMetadata, err := GetOrLoadSimilarityResults(ctx, store, input, DefaultSimilarityTTL, func(context.Context) ([]search.SimilarityResult, error) {
		loadCalls++
		return nil, nil
	})
	if err != nil {
		t.Fatalf("hit GetOrLoadSimilarityResults() error = %v", err)
	}
	if loadCalls != 1 || hitMetadata.Status != search.CacheStatusHit || len(hit) != 1 {
		t.Fatalf("hit loadCalls=%d metadata=%+v results=%+v", loadCalls, hitMetadata, hit)
	}
}

func TestSearchResponseStoreMetadataAndTTLSelection(t *testing.T) {
	req := searchRequest(nil)
	configured := SearchResponseStore{TTL: 42 * time.Second}
	metadata := configured.SearchResponseCacheMetadata(req, search.CacheStatusMiss)
	if metadata == nil || metadata.Status != search.CacheStatusMiss || metadata.TTLSeconds != 42 || metadata.Namespace != string(RedisNamespaceSearch) {
		t.Fatalf("configured metadata = %+v", metadata)
	}
	if got := configured.ttl(); got != 42*time.Second {
		t.Fatalf("configured ttl = %v", got)
	}
	if got := (SearchResponseStore{}).similarityTTL(); got != DefaultSimilarityTTL {
		t.Fatalf("default similarity ttl = %v", got)
	}
}

func TestGetOrLoadHelpersPropagateLoaderErrors(t *testing.T) {
	wantErr := errors.New("loader failed")
	if _, err := GetOrLoadSearchResponse(context.Background(), nil, searchRequest(nil), time.Minute, func(context.Context) (search.SearchResponse, error) {
		return search.SearchResponse{}, wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("search loader error = %v", err)
	}
	if _, err := GetOrLoadAutocompleteResponse(context.Background(), nil, "apple", time.Minute, func(context.Context) (search.AutocompleteResponse, error) {
		return search.AutocompleteResponse{}, wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("autocomplete loader error = %v", err)
	}
	results := []search.SimilarityResult{{Score: 0.5}}
	got, metadata, err := GetOrLoadSimilarityResults(context.Background(), nil, nil, time.Minute, func(context.Context) ([]search.SimilarityResult, error) {
		return results, wantErr
	})
	if !errors.Is(err, wantErr) || len(got) != 1 || metadata.Status != search.CacheStatusMiss {
		t.Fatalf("similarity loader results=%+v metadata=%+v err=%v", got, metadata, err)
	}
}

func TestCanonicalFilterIncludeTieBreaker(t *testing.T) {
	canonical := canonicalFilters([]search.SearchFilter{
		{FilterID: "same", Kind: search.SearchFilterKindAllergen, Include: true},
		{FilterID: "same", Kind: search.SearchFilterKindAllergen, Include: false},
		{FilterID: "alpha", Kind: search.SearchFilterKindAllergen, Include: true},
	})
	if len(canonical) != 3 || canonical[0].FilterID != "alpha" || canonical[1].Include || !canonical[2].Include {
		t.Fatalf("canonical filters = %+v", canonical)
	}
}

func TestStableHashPanicsForUnsupportedPayload(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("stableHash() did not panic for unsupported payload")
		}
	}()
	stableHash(make(chan int))
}

func searchRequest(filters []search.SearchFilter) search.SearchRequest {
	return search.SearchRequest{
		Query:   " Tomato soup ",
		Mode:    search.SearchModeCatalog,
		Filters: filters,
		Page:    1,
	}
}

type memoryStore struct {
	values map[string]string
	ttls   map[string]time.Duration
}

func (s *memoryStore) Get(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", redis.Nil
	}
	return value, nil
}

func (s *memoryStore) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	s.values[key] = value
	s.ttls[key] = ttl
	return nil
}

type failingStore struct {
	err error
}

type blockingCatalogRepository struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingCatalogRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	close(r.started)
	<-r.release
	return []repository.FoodItemEntity{{Name: "Old label", PhysicalState: repository.PhysicalStateSolid}}, 1, nil
}

type controlledClassificationGeneration struct {
	mu         sync.Mutex
	generation uint64
	store      RedisStore
}

func (g *controlledClassificationGeneration) Current(context.Context) (uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.generation, nil
}

func (g *controlledClassificationGeneration) Advance() {
	g.mu.Lock()
	g.generation++
	g.mu.Unlock()
}

func (g *controlledClassificationGeneration) SetIfCurrent(ctx context.Context, generation uint64, key, value string, ttl time.Duration) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.generation != generation {
		return false, nil
	}
	return true, g.store.Set(ctx, key, value, ttl)
}

func (s failingStore) Get(context.Context, string) (string, error) {
	return "", s.err
}

func (s failingStore) Set(context.Context, string, string, time.Duration) error {
	return s.err
}

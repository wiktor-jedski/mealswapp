package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// Implements DESIGN-011 RedisCache schema-version and TTL defaults.
const (
	// SearchSchemaVersion isolates Redis entries when search response shape changes.
	// Implements DESIGN-011 RedisCache schema-version isolation.
	SearchSchemaVersion = "search-response-v2"
	// AutocompleteSchemaVersion isolates Redis entries when autocomplete response shape changes.
	// Implements DESIGN-011 RedisCache schema-version isolation.
	AutocompleteSchemaVersion = "autocomplete-response-v1"
	// SimilaritySchemaVersion isolates Redis entries when similarity calculation shape changes.
	// Implements DESIGN-011 RedisCache schema-version isolation.
	SimilaritySchemaVersion = "similarity-calculation-v1"
	DefaultSearchTTL        = 5 * time.Minute
	DefaultAutocompleteTTL  = 2 * time.Minute
	DefaultSimilarityTTL    = 15 * time.Minute
)

// RedisNamespace identifies the Redis key family.
// Implements DESIGN-011 RedisCache namespace handling.
type RedisNamespace string

// Implements DESIGN-011 RedisCache namespace constants.
const (
	RedisNamespaceSearch       RedisNamespace = "search"
	RedisNamespaceAutocomplete RedisNamespace = "autocomplete"
	RedisNamespaceSimilarity   RedisNamespace = "similarity"
)

// RedisCacheKey is the structured server-side Redis key.
// Implements DESIGN-011 RedisCache.
type RedisCacheKey struct {
	Namespace RedisNamespace
	ID        string
	Version   string
}

// String renders the Redis storage key without exposing request PII.
// Implements DESIGN-011 RedisCache stable hash key format.
func (k RedisCacheKey) String() string {
	return string(k.Namespace) + ":" + k.Version + ":" + k.ID
}

// RedisStore is the narrow cache command boundary used by GetRedis and SetRedis.
// Implements DESIGN-011 RedisCache get/set behavior.
type RedisStore interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
}

// SearchResponseStore adapts RedisStore to the Catalog Search service cache boundary.
// Implements DESIGN-002 SearchController and DESIGN-011 RedisCache.
type SearchResponseStore struct {
	Store         RedisStore
	TTL           time.Duration
	SimilarityTTL time.Duration
}

// GetSearchResponse reads one cached SearchResponse.
// Implements DESIGN-011 RedisCache cache-hit behavior.
func (s SearchResponseStore) GetSearchResponse(ctx context.Context, req search.SearchRequest) (search.SearchResponse, bool, error) {
	ttl := s.ttl()
	key := BuildSearchCacheKey(req)
	response, hit, err := GetRedis[search.SearchResponse](ctx, s.Store, key)
	if err != nil || !hit {
		return response, hit, err
	}
	response.Cache = cacheMetadataPtr(key, search.CacheStatusHit, ttl)
	return response, true, nil
}

// SetSearchResponse stores one successful SearchResponse.
// Implements DESIGN-011 RedisCache cache-miss persistence behavior.
func (s SearchResponseStore) SetSearchResponse(ctx context.Context, req search.SearchRequest, response search.SearchResponse) error {
	ttl := s.ttl()
	key := BuildSearchCacheKey(req)
	return SetRedis(ctx, s.Store, key, responseWithoutCacheMetadata(response), ttl)
}

// SearchResponseCacheMetadata returns response metadata for a search cache request.
// Implements DESIGN-011 RedisCache response metadata.
func (s SearchResponseStore) SearchResponseCacheMetadata(req search.SearchRequest, status search.CacheStatus) *search.CacheMetadata {
	key := BuildSearchCacheKey(req)
	return cacheMetadataPtr(key, status, s.ttl())
}

// GetSimilarityCalculation reads one cached Substitution Search macro comparison payload.
// Implements DESIGN-011 RedisCache similarity calculation cache-hit behavior.
func (s SearchResponseStore) GetSimilarityCalculation(ctx context.Context, inputs []search.SubstitutionInput) (search.SimilarityCalculation, bool, error) {
	key := BuildSimilarityCacheKey(inputs)
	return GetRedis[search.SimilarityCalculation](ctx, s.Store, key)
}

// SetSimilarityCalculation stores one successful Substitution Search macro comparison payload.
// Implements DESIGN-011 RedisCache similarity calculation cache-miss persistence behavior.
func (s SearchResponseStore) SetSimilarityCalculation(ctx context.Context, inputs []search.SubstitutionInput, calculation search.SimilarityCalculation) error {
	key := BuildSimilarityCacheKey(inputs)
	return SetRedis(ctx, s.Store, key, calculation, s.similarityTTL())
}

// SimilarityCalculationCacheMetadata returns response metadata for a similarity cache request.
// Implements DESIGN-011 RedisCache response metadata.
func (s SearchResponseStore) SimilarityCalculationCacheMetadata(inputs []search.SubstitutionInput, status search.CacheStatus) *search.CacheMetadata {
	key := BuildSimilarityCacheKey(inputs)
	return cacheMetadataPtr(key, status, s.similarityTTL())
}

// ttl returns the configured search TTL or the default duration.
// Implements DESIGN-011 RedisCache cache expiration behavior.
func (s SearchResponseStore) ttl() time.Duration {
	if s.TTL <= 0 {
		return DefaultSearchTTL
	}
	return s.TTL
}

// similarityTTL returns the configured similarity TTL or the default duration.
// Implements DESIGN-011 RedisCache cache expiration behavior.
func (s SearchResponseStore) similarityTTL() time.Duration {
	if s.SimilarityTTL <= 0 {
		return DefaultSimilarityTTL
	}
	return s.SimilarityTTL
}

// GoRedisStore adapts go-redis clients to RedisStore.
// Implements DESIGN-011 RedisCache get/set behavior.
type GoRedisStore struct {
	Client redis.Cmdable
}

// Get retrieves a raw Redis value.
// Implements DESIGN-011 RedisCache get behavior.
func (s GoRedisStore) Get(ctx context.Context, key string) (string, error) {
	return s.Client.Get(ctx, key).Result()
}

// Set stores a raw Redis value with TTL.
// Implements DESIGN-011 RedisCache set behavior.
func (s GoRedisStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.Client.Set(ctx, key, value, ttl).Err()
}

// GetRedis retrieves a typed value from Redis. Redis misses and Redis failures are non-fatal.
// Implements DESIGN-011 RedisCache get behavior and redis_down fallback.
func GetRedis[T any](ctx context.Context, store RedisStore, key RedisCacheKey) (T, bool, error) {
	var zero T
	if store == nil {
		return zero, false, nil
	}
	raw, err := store.Get(ctx, key.String())
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, false, nil
		}
		return zero, false, err
	}
	var value T
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return zero, false, err
	}
	return value, true, nil
}

// SetRedis stores a typed value in Redis with a caller-selected TTL.
// Implements DESIGN-011 RedisCache set behavior.
func SetRedis[T any](ctx context.Context, store RedisStore, key RedisCacheKey, value T, ttl time.Duration) error {
	if store == nil || ttl <= 0 {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return store.Set(ctx, key.String(), string(payload), ttl)
}

// GetOrLoadSearchResponse returns cached search results or falls back to the source loader.
// Implements DESIGN-011 RedisCache cache-hit, cache-miss, and redis_down fallback behavior.
func GetOrLoadSearchResponse(ctx context.Context, store RedisStore, req search.SearchRequest, ttl time.Duration, load func(context.Context) (search.SearchResponse, error)) (search.SearchResponse, error) {
	key := BuildSearchCacheKey(req)
	if cached, hit, err := GetRedis[search.SearchResponse](ctx, store, key); err == nil && hit {
		cached.Cache = cacheMetadataPtr(key, search.CacheStatusHit, ttl)
		return cached, nil
	}

	response, err := load(ctx)
	if err != nil {
		return response, err
	}
	response.Cache = cacheMetadataPtr(key, search.CacheStatusMiss, ttl)
	_ = SetRedis(ctx, store, key, responseWithoutCacheMetadata(response), ttl)
	return response, nil
}

// GetOrLoadAutocompleteResponse returns cached autocomplete results or falls back to the source loader.
// Implements DESIGN-011 RedisCache autocomplete cache metadata and redis_down fallback behavior.
func GetOrLoadAutocompleteResponse(ctx context.Context, store RedisStore, query string, ttl time.Duration, load func(context.Context) (search.AutocompleteResponse, error)) (search.AutocompleteResponse, error) {
	key := BuildAutocompleteCacheKey(query)
	if cached, hit, err := GetRedis[search.AutocompleteResponse](ctx, store, key); err == nil && hit {
		cached.Cache = cacheMetadataPtr(key, search.CacheStatusHit, ttl)
		return cached, nil
	}

	response, err := load(ctx)
	if err != nil {
		return response, err
	}
	response.Cache = cacheMetadataPtr(key, search.CacheStatusMiss, ttl)
	_ = SetRedis(ctx, store, key, autocompleteWithoutCacheMetadata(response), ttl)
	return response, nil
}

// GetOrLoadSimilarityResults returns cached similarity results or falls back to the source loader.
// Implements DESIGN-011 RedisCache similarity calculation get/set and redis_down fallback behavior.
func GetOrLoadSimilarityResults(ctx context.Context, store RedisStore, inputs []search.SubstitutionInput, ttl time.Duration, load func(context.Context) ([]search.SimilarityResult, error)) ([]search.SimilarityResult, search.CacheMetadata, error) {
	key := BuildSimilarityCacheKey(inputs)
	if cached, hit, err := GetRedis[[]search.SimilarityResult](ctx, store, key); err == nil && hit {
		return cached, SearchCacheMetadata(key, search.CacheStatusHit, ttl), nil
	}

	results, err := load(ctx)
	if err != nil {
		return results, SearchCacheMetadata(key, search.CacheStatusMiss, ttl), err
	}
	_ = SetRedis(ctx, store, key, results, ttl)
	return results, SearchCacheMetadata(key, search.CacheStatusMiss, ttl), nil
}

// BuildSearchCacheKey builds a deterministic key from normalized search inputs.
// Implements DESIGN-011 RedisCache and DESIGN-002 QueryParser cache key contract.
func BuildSearchCacheKey(req search.SearchRequest) RedisCacheKey {
	normalized := canonicalSearchRequest(req)
	return RedisCacheKey{
		Namespace: RedisNamespaceSearch,
		ID:        stableHash(normalized),
		Version:   SearchSchemaVersion,
	}
}

// BuildAutocompleteCacheKey builds a deterministic key from autocomplete query text.
// Implements DESIGN-011 RedisCache autocomplete key contract.
func BuildAutocompleteCacheKey(query string) RedisCacheKey {
	return RedisCacheKey{
		Namespace: RedisNamespaceAutocomplete,
		ID:        stableHash(map[string]string{"query": normalizeSpaces(query)}),
		Version:   AutocompleteSchemaVersion,
	}
}

// BuildSimilarityCacheKey builds a deterministic key from substitution calculation inputs.
// Implements DESIGN-011 RedisCache similarity key contract.
func BuildSimilarityCacheKey(inputs []search.SubstitutionInput) RedisCacheKey {
	canonicalInputs := canonicalSubstitutionInputs(inputs)
	return RedisCacheKey{
		Namespace: RedisNamespaceSimilarity,
		ID:        stableHash(canonicalInputs),
		Version:   SimilaritySchemaVersion,
	}
}

// SearchCacheMetadata returns response-safe cache metadata.
// Implements DESIGN-011 RedisCache response metadata.
func SearchCacheMetadata(key RedisCacheKey, status search.CacheStatus, ttl time.Duration) search.CacheMetadata {
	return search.CacheMetadata{
		Status:        status,
		Namespace:     string(key.Namespace),
		SchemaVersion: key.Version,
		TTLSeconds:    int64(ttl / time.Second),
	}
}

// cacheMetadataPtr returns pointer metadata for response payloads.
// Implements DESIGN-011 RedisCache response metadata.
func cacheMetadataPtr(key RedisCacheKey, status search.CacheStatus, ttl time.Duration) *search.CacheMetadata {
	metadata := SearchCacheMetadata(key, status, ttl)
	return &metadata
}

// responseWithoutCacheMetadata removes transient cache metadata before persistence.
// Implements DESIGN-011 RedisCache cache payload normalization.
func responseWithoutCacheMetadata(response search.SearchResponse) search.SearchResponse {
	response.Cache = nil
	return response
}

// autocompleteWithoutCacheMetadata removes transient autocomplete cache metadata before persistence.
// Implements DESIGN-011 RedisCache cache payload normalization.
func autocompleteWithoutCacheMetadata(response search.AutocompleteResponse) search.AutocompleteResponse {
	response.Cache = nil
	return response
}

// canonicalSearch is the stable hash payload for search cache keys.
// Implements DESIGN-011 RedisCache stable key hashing.
type canonicalSearch struct {
	Query              string                  `json:"query"`
	Mode               search.SearchMode       `json:"mode"`
	Filters            []canonicalFilter       `json:"filters"`
	Page               int                     `json:"page"`
	SubstitutionInputs []canonicalSubstitution `json:"substitutionInputs"`
	DailyDietID        string                  `json:"dailyDietId,omitempty"`
}

// canonicalFilter is the stable hash payload for one search filter.
// Implements DESIGN-011 RedisCache stable key hashing.
type canonicalFilter struct {
	FilterID string                  `json:"filterId"`
	Kind     search.SearchFilterKind `json:"kind"`
	Include  bool                    `json:"include"`
}

// canonicalSubstitution is the stable hash payload for one substitution input.
// Implements DESIGN-011 RedisCache stable key hashing.
type canonicalSubstitution struct {
	FoodObjectID uuid.UUID `json:"foodObjectId"`
	Quantity     float64   `json:"quantity"`
	Unit         string    `json:"unit"`
}

// canonicalSearchRequest normalizes search request fields for key hashing.
// Implements DESIGN-011 RedisCache stable key hashing.
func canonicalSearchRequest(req search.SearchRequest) canonicalSearch {
	dailyDietID := ""
	if req.DailyDietID != nil {
		dailyDietID = req.DailyDietID.String()
	}
	return canonicalSearch{
		Query:              normalizeSpaces(req.Query),
		Mode:               req.Mode,
		Filters:            canonicalFilters(req.Filters),
		Page:               req.Page,
		SubstitutionInputs: canonicalSubstitutionInputs(req.SubstitutionInputs),
		DailyDietID:        dailyDietID,
	}
}

// canonicalFilters normalizes and sorts filters for deterministic cache keys.
// Implements DESIGN-011 RedisCache stable key hashing.
func canonicalFilters(filters []search.SearchFilter) []canonicalFilter {
	canonical := make([]canonicalFilter, 0, len(filters))
	for _, filter := range filters {
		canonical = append(canonical, canonicalFilter{
			FilterID: strings.ToLower(strings.TrimSpace(filter.FilterID)),
			Kind:     filter.Kind,
			Include:  filter.Include,
		})
	}
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].Kind != canonical[j].Kind {
			return canonical[i].Kind < canonical[j].Kind
		}
		if canonical[i].FilterID != canonical[j].FilterID {
			return canonical[i].FilterID < canonical[j].FilterID
		}
		return !canonical[i].Include && canonical[j].Include
	})
	return canonical
}

// canonicalSubstitutionInputs normalizes and sorts substitution inputs for deterministic cache keys.
// Implements DESIGN-011 RedisCache stable key hashing.
func canonicalSubstitutionInputs(inputs []search.SubstitutionInput) []canonicalSubstitution {
	canonical := make([]canonicalSubstitution, 0, len(inputs))
	for _, input := range inputs {
		canonical = append(canonical, canonicalSubstitution{
			FoodObjectID: input.FoodObjectID,
			Quantity:     input.Quantity,
			Unit:         strings.ToLower(strings.TrimSpace(input.Unit)),
		})
	}
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].FoodObjectID != canonical[j].FoodObjectID {
			return canonical[i].FoodObjectID.String() < canonical[j].FoodObjectID.String()
		}
		if canonical[i].Unit != canonical[j].Unit {
			return canonical[i].Unit < canonical[j].Unit
		}
		return canonical[i].Quantity < canonical[j].Quantity
	})
	return canonical
}

// normalizeSpaces lowercases and compacts whitespace for cache keys.
// Implements DESIGN-011 RedisCache stable key hashing.
func normalizeSpaces(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// stableHash serializes canonical payloads into non-PII cache key IDs.
// Implements DESIGN-011 RedisCache stable key hashing.
func stableHash(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

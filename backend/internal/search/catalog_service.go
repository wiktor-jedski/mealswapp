package search

import (
	"context"
	"slices"
	"sort"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// FoodCatalogRepository is the repository primitive used by Catalog Search orchestration.
// Implements DESIGN-002 SearchController.
type FoodCatalogRepository interface {
	Search(ctx context.Context, q repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error)
}

// SearchResponseCache stores successful Catalog Search responses by deterministic request key.
// Implements DESIGN-002 SearchController and DESIGN-011 RedisCache.
type SearchResponseCache interface {
	GetSearchResponse(context.Context, SearchRequest) (SearchResponse, bool, SearchResponseCacheToken, error)
	SetSearchResponse(context.Context, SearchRequest, SearchResponse, SearchResponseCacheToken) (bool, error)
}

// SearchResponseCacheToken preserves the shared generation observed before a cache miss.
// Implements DESIGN-011 RedisCache guarded cache writes.
type SearchResponseCacheToken struct {
	ClassificationGeneration uint64
}

// SearchResponseCacheMetadataProvider reports response-safe cache metadata without exposing cache keys.
// Implements DESIGN-011 RedisCache response metadata.
type SearchResponseCacheMetadataProvider interface {
	SearchResponseCacheMetadata(SearchRequest, CacheStatus) *CacheMetadata
}

// SimilarityCalculation carries cached macro comparison results and diagnostics.
// Implements DESIGN-003 CosineSimilarityCalculator and DESIGN-011 RedisCache.
type SimilarityCalculation struct {
	Results     []SimilarityResult
	Diagnostics []SimilarityDiagnostic
}

// SimilarityCalculationCache stores successful Substitution Search macro comparison payloads.
// Implements DESIGN-002 SearchController and DESIGN-011 RedisCache.
type SimilarityCalculationCache interface {
	GetSimilarityCalculation(context.Context, []SubstitutionInput) (SimilarityCalculation, bool, SimilarityCalculationCacheToken, error)
	SetSimilarityCalculation(context.Context, []SubstitutionInput, SimilarityCalculation, SimilarityCalculationCacheToken) (bool, error)
}

// SimilarityCalculationCacheToken guards writes against catalog mutations during calculation.
// Implements DESIGN-011 RedisCache shared food-data generation.
type SimilarityCalculationCacheToken struct{ FoodDataGeneration uint64 }

// SimilarityCalculationCacheMetadataProvider reports response-safe similarity cache metadata.
// Implements DESIGN-011 RedisCache response metadata.
type SimilarityCalculationCacheMetadataProvider interface {
	SimilarityCalculationCacheMetadata([]SubstitutionInput, CacheStatus) *CacheMetadata
}

// CatalogService orchestrates Catalog Search over filters, repository paging, and cache.
// Implements DESIGN-002 SearchController.
type CatalogService struct {
	repository FoodCatalogRepository
	cache      SearchResponseCache
}

// NewCatalogService creates Catalog Search orchestration.
// Implements DESIGN-002 SearchController.
func NewCatalogService(repository FoodCatalogRepository, cache SearchResponseCache) *CatalogService {
	return &CatalogService{repository: repository, cache: cache}
}

// Search executes Catalog Search and returns a response envelope payload.
// Implements DESIGN-002 SearchController.
func (s *CatalogService) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	prepared, err := PrepareSearchRequest(req, DailyDietDataUnavailable)
	if err != nil {
		return SearchResponse{}, err
	}
	if prepared.Rejection != nil {
		return SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: req.Page, SimilarityScores: []float64{}, Warnings: []string{}, Rejection: prepared.Rejection}, nil
	}
	if prepared.ParsedQuery.Strategy != SearchStrategyCatalog {
		return SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: req.Page, SimilarityScores: []float64{}, Warnings: []string{}, Rejection: &SearchRejection{Code: "rejected_search", Message: "search mode is not available for catalog results", Field: "mode"}}, nil
	}
	normalizedReq := req
	normalizedReq.Query = prepared.ParsedQuery.NormalizedText
	normalizedReq.Page = normalizedPage(req.Page)

	cacheWarnings := []string{}
	var cacheToken SearchResponseCacheToken
	if s.cache != nil {
		if cached, hit, token, err := s.cache.GetSearchResponse(ctx, normalizedReq); err == nil && hit {
			return cached, nil
		} else if err != nil {
			cacheWarnings = append(cacheWarnings, WarningCacheUnavailable)
		} else {
			cacheToken = token
		}
	}

	response, err := s.loadCatalog(ctx, prepared, normalizedReq)
	if err != nil {
		return SearchResponse{}, err
	}
	response.Warnings = append(response.Warnings, cacheWarnings...)
	if response.Rejection == nil && s.cache != nil {
		if stored, err := s.cache.SetSearchResponse(ctx, normalizedReq, responseWithoutCache(response), cacheToken); err != nil {
			response.Warnings = appendWarningOnce(response.Warnings, WarningCacheUnavailable)
		} else if stored {
			response.Cache = searchResponseCacheMetadata(s.cache, normalizedReq, CacheStatusMiss)
		}
	}
	return response, nil
}

// loadCatalog retrieves repository results and maps them to a search response.
// Implements DESIGN-002 SearchController.
func (s *CatalogService) loadCatalog(ctx context.Context, prepared PreparedSearch, req SearchRequest) (SearchResponse, error) {
	items, total, err := s.repository.Search(ctx, prepared.Filters.RepositoryQuery)
	if err != nil {
		return SearchResponse{}, err
	}
	sortCatalogItems(items)
	return SearchResponse{
		Items:            items,
		TotalCount:       total,
		Page:             req.Page,
		SimilarityScores: emptyScores(len(items)),
		Warnings:         catalogWarnings(prepared.Filters.ExclusionRules),
	}, nil
}

// sortCatalogItems applies name-first presentation ordering.
// Implements DESIGN-002 SearchController.
func sortCatalogItems(items []repository.FoodItemEntity) {
	// Active food names are unique in PostgreSQL; stable sorting preserves repository order for defensive fixtures.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
}

// catalogWarnings formats exclusion rules as response warnings.
// Implements DESIGN-002 FilterProcessor.
func catalogWarnings(rules []ExclusionRule) []string {
	if len(rules) == 0 {
		return []string{}
	}
	warnings := make([]string, 0, len(rules))
	for _, rule := range rules {
		warnings = append(warnings, "excluded "+string(rule.Kind)+" "+rule.FilterID)
	}
	return warnings
}

// emptyScores returns neutral catalog similarity scores.
// Implements DESIGN-002 SearchController.
func emptyScores(count int) []float64 {
	scores := make([]float64, count)
	return scores
}

// normalizedPage clamps invalid page values to the first page.
// Implements DESIGN-002 PaginationHandler.
func normalizedPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// responseWithoutCache removes transient cache metadata before storage.
// Implements DESIGN-011 RedisCache.
func responseWithoutCache(response SearchResponse) SearchResponse {
	response.Cache = nil
	return response
}

// searchResponseCacheMetadata returns optional metadata for successful cache writes.
// Implements DESIGN-011 RedisCache response metadata.
func searchResponseCacheMetadata(cache SearchResponseCache, req SearchRequest, status CacheStatus) *CacheMetadata {
	provider, ok := cache.(SearchResponseCacheMetadataProvider)
	if !ok {
		return nil
	}
	return provider.SearchResponseCacheMetadata(req, status)
}

// appendWarningOnce appends cache-degradation warning metadata only when absent.
// Implements DESIGN-011 RedisCache degraded response metadata.
func appendWarningOnce(warnings []string, warning string) []string {
	if slices.Contains(warnings, warning) {
		return warnings
	}
	return append(warnings, warning)
}

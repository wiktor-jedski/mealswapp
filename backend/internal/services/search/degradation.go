package search

import (
	"context"
	"errors"
	"strings"

	"mealswapp/backend/internal/http/apperrors"
)

var (
	ErrCacheMiss              = errors.New("cache miss")
	ErrSearchCacheUnavailable = errors.New("search cache unavailable")
	ErrSearchRepositoryDown   = errors.New("search repository unavailable")
	ErrSimilarityUnavailable  = errors.New("similarity unavailable")
)

type SearchExecutionRequest struct {
	Query         QueryInput
	Filters       FilterInput
	CacheKey      string
	UseCache      bool
	UseSimilarity bool
}

type SearchExecutionResult struct {
	Items            []any
	TotalCount       int
	Page             int
	PageSize         int
	SimilarityScores []float64
	Warnings         []string
	CacheHit         bool
	Degraded         bool
}

type SearchCache interface {
	Get(ctx context.Context, key string) (SearchExecutionResult, error)
	Set(ctx context.Context, key string, value SearchExecutionResult) error
}

type SearchRepository interface {
	Search(ctx context.Context, query RepositoryQuery) (SearchExecutionResult, error)
}

type SimilarityProcessor interface {
	ApplySimilarity(ctx context.Context, request ParsedQuery, result SearchExecutionResult) (SearchExecutionResult, error)
}

type Executor struct {
	cache      SearchCache
	repository SearchRepository
	similarity SimilarityProcessor
}

func NewExecutor(cache SearchCache, repository SearchRepository, similarity SimilarityProcessor) Executor {
	return Executor{cache: cache, repository: repository, similarity: similarity}
}

func (executor Executor) Search(ctx context.Context, request SearchExecutionRequest) (SearchExecutionResult, error) {
	parsed, err := ParseQuery(request.Query)
	if err != nil {
		return SearchExecutionResult{}, err
	}

	filterQuery, err := ApplyFilters(request.Filters)
	if err != nil {
		return SearchExecutionResult{}, err
	}

	var warnings []string
	if request.UseCache && executor.cache != nil && strings.TrimSpace(request.CacheKey) != "" {
		cached, err := executor.cache.Get(ctx, request.CacheKey)
		if err == nil {
			cached.CacheHit = true
			return normalizeExecutionResult(cached), nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			warnings = append(warnings, "cache_unavailable")
		}
	}

	if executor.repository == nil {
		return SearchExecutionResult{}, apperrors.DependencyUnavailable("Search repository unavailable")
	}

	result, err := executor.repository.Search(ctx, filterQuery)
	if err != nil {
		return SearchExecutionResult{}, apperrors.DependencyUnavailable("Search repository unavailable")
	}
	result.Warnings = append(warnings, result.Warnings...)

	if request.UseSimilarity && executor.similarity != nil {
		similar, err := executor.similarity.ApplySimilarity(ctx, parsed, result)
		if err != nil {
			result.Degraded = true
			result.Warnings = append(result.Warnings, "similarity_unavailable")
		} else {
			result = similar
			result.Warnings = append(warnings, result.Warnings...)
		}
	}

	result = normalizeExecutionResult(result)
	if request.UseCache && executor.cache != nil && strings.TrimSpace(request.CacheKey) != "" {
		_ = executor.cache.Set(ctx, request.CacheKey, result)
	}
	return result, nil
}

func normalizeExecutionResult(result SearchExecutionResult) SearchExecutionResult {
	if result.Items == nil {
		result.Items = []any{}
	}
	if result.SimilarityScores == nil {
		result.SimilarityScores = []float64{}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	return result
}

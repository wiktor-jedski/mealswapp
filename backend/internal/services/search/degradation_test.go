package search

import (
	"context"
	"testing"

	"mealswapp/backend/internal/http/apperrors"
)

func TestExecutorReturnsCacheHitWhenRedisAvailable(t *testing.T) {
	cache := &fakeSearchCache{
		result: SearchExecutionResult{
			Items:      []any{map[string]any{"id": "cached"}},
			TotalCount: 1,
		},
	}
	repository := &fakeSearchRepository{}

	result, err := NewExecutor(cache, repository, nil).Search(context.Background(), SearchExecutionRequest{
		Query:    QueryInput{Mode: ModeSingle, Query: "tofu"},
		Filters:  FilterInput{NormalizedSearch: "tofu"},
		CacheKey: "search-key",
		UseCache: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.CacheHit || repository.calls != 0 {
		t.Fatalf("expected cache hit without repository call, result=%#v calls=%d", result, repository.calls)
	}
}

func TestExecutorFallsBackToRepositoryWhenRedisFails(t *testing.T) {
	cache := &fakeSearchCache{getErr: ErrSearchCacheUnavailable}
	repository := &fakeSearchRepository{
		result: SearchExecutionResult{
			Items:      []any{map[string]any{"id": "db"}},
			TotalCount: 1,
		},
	}

	result, err := NewExecutor(cache, repository, nil).Search(context.Background(), SearchExecutionRequest{
		Query:    QueryInput{Mode: ModeSingle, Query: "tofu"},
		Filters:  FilterInput{NormalizedSearch: "tofu"},
		CacheKey: "search-key",
		UseCache: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.CacheHit {
		t.Fatalf("expected database result after cache failure, got %#v", result)
	}
	if repository.calls != 1 {
		t.Fatalf("expected one repository call, got %d", repository.calls)
	}
	if !containsString(result.Warnings, "cache_unavailable") {
		t.Fatalf("expected cache warning, got %#v", result.Warnings)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected database result to be offered back to cache, got %d calls", cache.setCalls)
	}
}

func TestExecutorReturnsStructuredDependencyErrorWhenRepositoryFails(t *testing.T) {
	cache := &fakeSearchCache{getErr: ErrCacheMiss}
	repository := &fakeSearchRepository{err: ErrSearchRepositoryDown}

	_, err := NewExecutor(cache, repository, nil).Search(context.Background(), SearchExecutionRequest{
		Query:    QueryInput{Mode: ModeSingle, Query: "tofu"},
		Filters:  FilterInput{NormalizedSearch: "tofu"},
		CacheKey: "search-key",
		UseCache: true,
	})

	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "dependency_unavailable" || appErr.Status != 503 || !appErr.Retryable {
		t.Fatalf("expected structured dependency error, got %#v", err)
	}
}

func TestExecutorDegradesToTextResultsWhenSimilarityFails(t *testing.T) {
	repository := &fakeSearchRepository{
		result: SearchExecutionResult{
			Items:      []any{map[string]any{"id": "text"}},
			TotalCount: 1,
		},
	}
	similarity := &fakeSimilarityProcessor{err: ErrSimilarityUnavailable}

	result, err := NewExecutor(nil, repository, similarity).Search(context.Background(), SearchExecutionRequest{
		Query: QueryInput{
			Mode:       ModeReplacement,
			Query:      "olive oil",
			SourceItem: "butter",
		},
		Filters:       FilterInput{NormalizedSearch: "olive oil"},
		UseSimilarity: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Degraded || !containsString(result.Warnings, "similarity_unavailable") {
		t.Fatalf("expected degraded text result with similarity warning, got %#v", result)
	}
	if len(result.Items) != 1 || similarity.calls != 1 {
		t.Fatalf("expected repository text items and one similarity attempt, got result=%#v calls=%d", result, similarity.calls)
	}
}

func TestExecutorReturnsEmptyArraysForEmptyDatabaseResults(t *testing.T) {
	repository := &fakeSearchRepository{}

	result, err := NewExecutor(nil, repository, nil).Search(context.Background(), SearchExecutionRequest{
		Query:   QueryInput{Mode: ModeSingle, Query: "missing"},
		Filters: FilterInput{NormalizedSearch: "missing"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Items == nil || result.SimilarityScores == nil || result.Warnings == nil {
		t.Fatalf("expected normalized empty arrays, got %#v", result)
	}
}

type fakeSearchCache struct {
	result   SearchExecutionResult
	getErr   error
	setCalls int
}

func (cache *fakeSearchCache) Get(ctx context.Context, key string) (SearchExecutionResult, error) {
	if cache.getErr != nil {
		return SearchExecutionResult{}, cache.getErr
	}
	return cache.result, nil
}

func (cache *fakeSearchCache) Set(ctx context.Context, key string, value SearchExecutionResult) error {
	cache.setCalls++
	return nil
}

type fakeSearchRepository struct {
	result SearchExecutionResult
	err    error
	calls  int
}

func (repository *fakeSearchRepository) Search(ctx context.Context, query RepositoryQuery) (SearchExecutionResult, error) {
	repository.calls++
	if repository.err != nil {
		return SearchExecutionResult{}, repository.err
	}
	return repository.result, nil
}

type fakeSimilarityProcessor struct {
	result SearchExecutionResult
	err    error
	calls  int
}

func (processor *fakeSimilarityProcessor) ApplySimilarity(ctx context.Context, request ParsedQuery, result SearchExecutionResult) (SearchExecutionResult, error) {
	processor.calls++
	if processor.err != nil {
		return SearchExecutionResult{}, processor.err
	}
	if processor.result.Items != nil {
		return processor.result, nil
	}
	result.SimilarityScores = []float64{0.95}
	return result, nil
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

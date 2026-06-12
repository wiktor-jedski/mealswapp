package search

import (
	"context"
)

// Searcher is implemented by concrete search services.
// Implements DESIGN-002 SearchController.
type Searcher interface {
	Search(context.Context, SearchRequest) (SearchResponse, error)
}

// SearchDispatcher routes API search requests to the service for the resolved strategy.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
type SearchDispatcher struct {
	catalog      Searcher
	substitution Searcher
}

// NewSearchDispatcher composes production search strategy dispatch.
// Implements DESIGN-002 SearchController.
func NewSearchDispatcher(catalog Searcher, substitution Searcher) *SearchDispatcher {
	return &SearchDispatcher{catalog: catalog, substitution: substitution}
}

// Search resolves the user request shape and delegates to the matching search service.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
func (d *SearchDispatcher) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	parsed, err := BuildParsedQuery(req)
	if err != nil {
		return SearchResponse{}, err
	}
	switch parsed.Strategy {
	case SearchStrategySubstitution:
		return d.substitution.Search(ctx, req)
	default:
		return d.catalog.Search(ctx, req)
	}
}

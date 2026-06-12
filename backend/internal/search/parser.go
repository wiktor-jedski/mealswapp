package search

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// PageSize is the deterministic backend search page size.
// Implements DESIGN-002 PaginationHandler.
const PageSize = 10

// BuildParsedQuery normalizes request text and resolves search strategy.
// Implements DESIGN-002 QueryParser.
func BuildParsedQuery(req SearchRequest) (ParsedQuery, error) {
	normalizedQuery, err := security.NormalizeInput(security.InputFieldSearchQuery, req.Query)
	if err != nil {
		return ParsedQuery{}, fmt.Errorf("query is invalid: %w", err)
	}
	pageText := strconv.Itoa(req.Page)
	if _, err := security.NormalizeInput(security.InputFieldPagination, pageText); err != nil {
		return ParsedQuery{}, fmt.Errorf("page is invalid: %w", err)
	}
	strategy, err := SelectStrategy(req)
	if err != nil {
		return ParsedQuery{}, err
	}
	limit, offset := Paginate(req.Page, PageSize)
	return ParsedQuery{
		NormalizedText: normalizedQuery.Value,
		Tokens:         strings.Fields(normalizedQuery.Value),
		Strategy:       strategy,
		Limit:          limit,
		Offset:         offset,
	}, nil
}

// SelectStrategy resolves the search operation from request shape.
// Implements DESIGN-002 QueryParser.
func SelectStrategy(req SearchRequest) (SearchStrategy, error) {
	if len(req.SubstitutionInputs) > 0 {
		return SearchStrategySubstitution, nil
	}
	if req.DailyDietID != nil {
		return SearchStrategyDailyDietAlternative, nil
	}
	switch req.Mode {
	case SearchModeCatalog:
		return SearchStrategyCatalog, nil
	case SearchModeSubstitution:
		return SearchStrategySubstitution, nil
	case SearchModeDailyDietAlternative:
		return SearchStrategyDailyDietAlternative, nil
	default:
		return "", fmt.Errorf("search mode is unsupported")
	}
}

// Paginate clamps page size to 10 and converts one-based pages to offsets.
// Implements DESIGN-002 PaginationHandler.
func Paginate(page int, pageSize int) (limit int, offset int) {
	if page < 1 {
		page = 1
	}
	_ = pageSize
	limit = PageSize
	return limit, (page - 1) * limit
}

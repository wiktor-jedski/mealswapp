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
	pageText := strconv.Itoa(req.Page)
	if _, err := security.NormalizeInput(security.InputFieldPagination, pageText); err != nil {
		return ParsedQuery{}, fmt.Errorf("page is invalid: %w", err)
	}
	strategy, err := SelectStrategy(req)
	if err != nil {
		return ParsedQuery{}, err
	}
	normalizedText, err := normalizeQueryForStrategy(req.Query, strategy)
	if err != nil {
		return ParsedQuery{}, err
	}
	limit, offset := Paginate(req.Page, PageSize)
	return ParsedQuery{
		NormalizedText: normalizedText,
		Tokens:         strings.Fields(normalizedText),
		Strategy:       strategy,
		Limit:          limit,
		Offset:         offset,
	}, nil
}

// normalizeQueryForStrategy applies query text requirements for each search strategy.
// Implements DESIGN-002 QueryParser.
func normalizeQueryForStrategy(query string, strategy SearchStrategy) (string, error) {
	if strings.TrimSpace(query) == "" && strategy == SearchStrategySubstitution {
		return "", nil
	}
	normalizedQuery, err := security.NormalizeInput(security.InputFieldSearchQuery, query)
	if err != nil {
		return "", fmt.Errorf("query is invalid: %w", err)
	}
	return normalizedQuery.Value, nil
}

// SelectStrategy resolves the search operation from the requested mode.
// Implements DESIGN-002 QueryParser.
func SelectStrategy(req SearchRequest) (SearchStrategy, error) {
	switch req.Mode {
	case SearchModeCatalog:
		return SearchStrategyCatalog, nil
	case SearchModeSubstitution:
		return SearchStrategySubstitution, nil
	case SearchModeDailyDiet:
		return SearchStrategyDailyDiet, nil
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

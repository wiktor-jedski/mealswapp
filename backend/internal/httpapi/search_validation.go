package httpapi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// ValidateSearchRequestBody validates the Phase 04 search request before service dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func ValidateSearchRequestBody(body map[string]any) error {
	query, ok := body["query"].(string)
	if !ok {
		return errors.New("query is required")
	}
	if _, err := security.NormalizeInput(security.InputFieldSearchQuery, query); err != nil {
		return errors.New("query is invalid")
	}
	mode, ok := body["mode"].(string)
	if !ok {
		return errors.New("mode is required")
	}
	if _, err := security.NormalizeInput(security.InputFieldSearchMode, mode); err != nil {
		return errors.New("mode is invalid")
	}
	if err := validateSearchPageValue(body["page"]); err != nil {
		return err
	}
	if err := validateSearchFilters(body["filters"]); err != nil {
		return err
	}
	if err := validateSubstitutionInputs(body["substitutionInputs"]); err != nil {
		return err
	}
	if dailyDietID, ok := body["dailyDietId"]; ok {
		id, ok := dailyDietID.(string)
		if !ok {
			return errors.New("daily diet id is invalid")
		}
		if _, err := security.NormalizeInput(security.InputFieldDailyDietID, id); err != nil {
			return errors.New("daily diet id is invalid")
		}
	}
	return nil
}

// ValidateAutocompleteQueryParams validates autocomplete query parameters before service dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-002 AutocompleteRanker.
func ValidateAutocompleteQueryParams(values map[string]string) error {
	query := values["query"]
	if query == "" {
		query = values["q"]
	}
	if _, err := security.NormalizeInput(security.InputFieldAutocompleteQuery, query); err != nil {
		return errors.New("autocomplete query is invalid")
	}
	page := values["page"]
	if page == "" {
		page = "1"
	}
	if _, err := security.NormalizeInput(security.InputFieldPagination, page); err != nil {
		return errors.New("page is invalid")
	}
	return nil
}

// ParseValidatedSearchRequestBody converts a validated JSON body into the search contract.
// Implements DESIGN-002 QueryParser and DESIGN-010 RequestValidator.
func ParseValidatedSearchRequestBody(body map[string]any) (search.SearchRequest, error) {
	req := search.SearchRequest{
		Query: body["query"].(string),
		Mode:  search.SearchMode(body["mode"].(string)),
		Page:  int(body["page"].(float64)),
	}
	if dailyDietID, ok := body["dailyDietId"].(string); ok {
		id, err := uuid.Parse(dailyDietID)
		if err != nil {
			return search.SearchRequest{}, errors.New("daily diet id is invalid")
		}
		req.DailyDietID = &id
	}
	filters, err := parseValidatedSearchFilters(body["filters"])
	if err != nil {
		return search.SearchRequest{}, err
	}
	req.Filters = filters
	inputs, err := parseValidatedSubstitutionInputs(body["substitutionInputs"])
	if err != nil {
		return search.SearchRequest{}, err
	}
	req.SubstitutionInputs = inputs
	return req, nil
}

// validateSearchPageValue validates one-based JSON page values.
// Implements DESIGN-010 RequestValidator and DESIGN-002 PaginationHandler.
func validateSearchPageValue(value any) error {
	switch page := value.(type) {
	case float64:
		if page != float64(int(page)) {
			return errors.New("page is invalid")
		}
		return validatePageString(strconv.Itoa(int(page)))
	case int:
		return validatePageString(strconv.Itoa(page))
	case string:
		return validatePageString(page)
	default:
		return errors.New("page is required")
	}
}

// validateSearchFilters validates Phase 04 filter array shape and kinds.
// Implements DESIGN-010 RequestValidator and DESIGN-002 FilterProcessor.
func validateSearchFilters(value any) error {
	if value == nil {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return errors.New("filters are invalid")
	}
	for _, item := range items {
		filter, ok := item.(map[string]any)
		if !ok {
			return errors.New("filter is invalid")
		}
		filterID, ok := filter["filterId"].(string)
		if !ok || strings.TrimSpace(filterID) == "" {
			return errors.New("filter id is invalid")
		}
		kind, ok := filter["kind"].(string)
		if !ok {
			return errors.New("filter kind is required")
		}
		if _, err := security.NormalizeInput(security.InputFieldSearchFilterKind, kind); err != nil {
			return errors.New("filter kind is invalid")
		}
		if _, ok := filter["include"].(bool); !ok {
			return errors.New("filter include is invalid")
		}
	}
	return nil
}

// validateSubstitutionInputs validates Phase 04 substitution input array shape.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func validateSubstitutionInputs(value any) error {
	if value == nil {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return errors.New("substitution inputs are invalid")
	}
	for _, item := range items {
		input, ok := item.(map[string]any)
		if !ok {
			return errors.New("substitution input is invalid")
		}
		foodObjectID, ok := input["foodObjectId"].(string)
		if !ok {
			return errors.New("food object id is invalid")
		}
		if _, err := uuid.Parse(foodObjectID); err != nil {
			return errors.New("food object id is invalid")
		}
		quantity, err := quantityString(input["quantity"])
		if err != nil {
			return err
		}
		if _, err := security.NormalizeInput(security.InputFieldSubstitutionQuantity, quantity); err != nil {
			return errors.New("substitution quantity is invalid")
		}
		unit, ok := input["unit"].(string)
		if !ok {
			return errors.New("substitution unit is required")
		}
		if _, err := security.NormalizeInput(security.InputFieldSubstitutionUnit, unit); err != nil {
			return errors.New("substitution unit is invalid")
		}
	}
	return nil
}

// validatePageString validates page text through the typed security normalizer.
// Implements DESIGN-010 RequestValidator and DESIGN-002 PaginationHandler.
func validatePageString(page string) error {
	if _, err := security.NormalizeInput(security.InputFieldPagination, page); err != nil {
		return errors.New("page is invalid")
	}
	return nil
}

// quantityString converts accepted JSON quantity representations to validator text.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func quantityString(value any) (string, error) {
	switch quantity := value.(type) {
	case float64:
		return strconv.FormatFloat(quantity, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(quantity), nil
	case string:
		return quantity, nil
	default:
		return "", fmt.Errorf("substitution quantity is invalid")
	}
}

// parseValidatedSearchFilters converts validated JSON filter values for conflict checks.
// Implements DESIGN-010 RequestValidator and DESIGN-002 FilterProcessor.
func parseValidatedSearchFilters(value any) ([]search.SearchFilter, error) {
	if value == nil {
		return nil, nil
	}
	items := value.([]any)
	filters := make([]search.SearchFilter, 0, len(items))
	for _, item := range items {
		filter := item.(map[string]any)
		filters = append(filters, search.SearchFilter{
			FilterID: filter["filterId"].(string),
			Kind:     search.SearchFilterKind(filter["kind"].(string)),
			Include:  filter["include"].(bool),
		})
	}
	return filters, nil
}

// parseValidatedSubstitutionInputs converts validated JSON substitution values for strategy checks.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func parseValidatedSubstitutionInputs(value any) ([]search.SubstitutionInput, error) {
	if value == nil {
		return nil, nil
	}
	items := value.([]any)
	inputs := make([]search.SubstitutionInput, 0, len(items))
	for _, item := range items {
		input := item.(map[string]any)
		foodObjectID, err := uuid.Parse(input["foodObjectId"].(string))
		if err != nil {
			return nil, errors.New("food object id is invalid")
		}
		quantity, err := quantityFloat(input["quantity"])
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, search.SubstitutionInput{
			FoodObjectID: foodObjectID,
			Quantity:     quantity,
			Unit:         input["unit"].(string),
		})
	}
	return inputs, nil
}

// quantityFloat converts validated JSON quantity values for strategy checks.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func quantityFloat(value any) (float64, error) {
	switch quantity := value.(type) {
	case float64:
		return quantity, nil
	case int:
		return float64(quantity), nil
	case string:
		parsed, err := strconv.ParseFloat(quantity, 64)
		if err != nil {
			return 0, errors.New("substitution quantity is invalid")
		}
		return parsed, nil
	default:
		return 0, errors.New("substitution quantity is invalid")
	}
}

package httpapi

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController defensive collection limits.
const (
	maxSearchFilters      = 20
	maxSubstitutionInputs = 20
)

// validatedSearchRequestBodyDTO represents the typed search request shape after route validation.
// Implements DESIGN-010 RequestValidator and DESIGN-002 QueryParser.
type validatedSearchRequestBodyDTO struct {
	Query              string                          `json:"query"`
	Mode               string                          `json:"mode"`
	Page               *int                            `json:"page"`
	Filters            []validatedSearchFilterDTO      `json:"filters"`
	SubstitutionInputs []validatedSubstitutionInputDTO `json:"substitutionInputs"`
	DailyDietID        *string                         `json:"dailyDietId"`
}

// validatedSearchFilterDTO represents one typed search filter from the request DTO.
// Implements DESIGN-010 RequestValidator and DESIGN-002 FilterProcessor.
type validatedSearchFilterDTO struct {
	FilterID string `json:"filterId"`
	Kind     string `json:"kind"`
	Include  *bool  `json:"include"`
}

// validatedSubstitutionInputDTO represents one typed substitution input from the request DTO.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
type validatedSubstitutionInputDTO struct {
	FoodObjectID string   `json:"foodObjectId"`
	Quantity     *float64 `json:"quantity"`
	Unit         string   `json:"unit"`
}

// ValidateSearchRequestBody validates the Phase 04 search request before service dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func ValidateSearchRequestBody(body map[string]any) error {
	query, ok := body["query"].(string)
	if !ok {
		return errors.New("query is required")
	}
	mode, ok := body["mode"].(string)
	if !ok {
		return errors.New("mode is required")
	}
	if _, err := security.NormalizeInput(security.InputFieldSearchMode, mode); err != nil {
		return errors.New("mode is invalid")
	}
	if err := validateSearchQueryForMode(mode, query); err != nil {
		return err
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
	if err := validateSearchModeShape(mode, body); err != nil {
		return err
	}
	return nil
}

// ValidateAutocompleteQueryParams validates autocomplete query parameters before service dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-002 AutocompleteRanker.
func ValidateAutocompleteQueryParams(values map[string]string) error {
	query := values["query"]
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
	dto, err := decodeValidatedSearchRequestBody(body)
	if err != nil {
		return search.SearchRequest{}, err
	}
	return searchRequestFromValidatedDTO(dto)
}

// searchRequestFromValidatedDTO maps the typed request DTO into the search contract.
// Implements DESIGN-002 QueryParser and DESIGN-010 RequestValidator.
func searchRequestFromValidatedDTO(dto validatedSearchRequestBodyDTO) (search.SearchRequest, error) {
	if strings.TrimSpace(dto.Mode) == "" {
		return search.SearchRequest{}, errors.New("mode is required")
	}
	if dto.Page == nil {
		return search.SearchRequest{}, errors.New("page is required")
	}
	if err := validateSearchQueryForMode(dto.Mode, dto.Query); err != nil {
		return search.SearchRequest{}, err
	}
	if err := validateSearchModeDTOShape(dto); err != nil {
		return search.SearchRequest{}, err
	}
	req := search.SearchRequest{
		Query: dto.Query,
		Mode:  search.SearchMode(dto.Mode),
		Page:  *dto.Page,
	}
	if dto.DailyDietID != nil {
		id, err := uuid.Parse(*dto.DailyDietID)
		if err != nil {
			return search.SearchRequest{}, errors.New("daily diet id is invalid")
		}
		req.DailyDietID = &id
	}
	filters, err := parseValidatedSearchFilterDTOs(dto.Filters)
	if err != nil {
		return search.SearchRequest{}, err
	}
	req.Filters = filters
	inputs, err := parseValidatedSubstitutionInputDTOs(dto.SubstitutionInputs)
	if err != nil {
		return search.SearchRequest{}, err
	}
	req.SubstitutionInputs = inputs
	return req, nil
}

// validateSearchQueryForMode applies mode-specific query requirements.
// Implements DESIGN-010 RequestValidator and DESIGN-002 QueryParser.
func validateSearchQueryForMode(mode string, query string) error {
	if strings.TrimSpace(query) == "" && search.SearchMode(mode) == search.SearchModeSubstitution {
		return nil
	}
	if _, err := security.NormalizeInput(security.InputFieldSearchQuery, query); err != nil {
		return errors.New("query is invalid")
	}
	return nil
}

// validateSearchModeShape validates that optional body fields match the requested search mode.
// Implements DESIGN-010 RequestValidator and DESIGN-002 QueryParser.
func validateSearchModeShape(mode string, body map[string]any) error {
	substitutionCount := substitutionInputCount(body["substitutionInputs"])
	hasDailyDietID := body["dailyDietId"] != nil
	switch search.SearchMode(mode) {
	case search.SearchModeCatalog:
		if substitutionCount > 0 || hasDailyDietID {
			return errors.New("catalog search body is invalid")
		}
	case search.SearchModeSubstitution:
		if substitutionCount == 0 || hasDailyDietID {
			return errors.New("substitution search body is invalid")
		}
	case search.SearchModeDailyDiet, search.SearchModeDailyDietAlternative:
		if substitutionCount > 0 || !hasDailyDietID {
			return errors.New("daily diet alternative search body is invalid")
		}
	}
	return nil
}

// validateSearchModeDTOShape validates typed search request DTO mode/body consistency.
// Implements DESIGN-010 RequestValidator and DESIGN-002 QueryParser.
func validateSearchModeDTOShape(dto validatedSearchRequestBodyDTO) error {
	switch search.SearchMode(dto.Mode) {
	case search.SearchModeCatalog:
		if len(dto.SubstitutionInputs) > 0 || dto.DailyDietID != nil {
			return errors.New("catalog search body is invalid")
		}
	case search.SearchModeSubstitution:
		if len(dto.SubstitutionInputs) == 0 || dto.DailyDietID != nil {
			return errors.New("substitution search body is invalid")
		}
	case search.SearchModeDailyDiet, search.SearchModeDailyDietAlternative:
		if len(dto.SubstitutionInputs) > 0 || dto.DailyDietID == nil {
			return errors.New("daily diet alternative search body is invalid")
		}
	}
	return nil
}

// substitutionInputCount returns the number of substitution inputs when the JSON shape is an array.
// Implements DESIGN-010 RequestValidator and DESIGN-002 QueryParser.
func substitutionInputCount(value any) int {
	items, ok := value.([]any)
	if !ok {
		return 0
	}
	return len(items)
}

// decodeValidatedSearchRequestBody converts the generic route body map into a typed DTO.
// Implements DESIGN-010 RequestValidator and DESIGN-002 QueryParser.
func decodeValidatedSearchRequestBody(body map[string]any) (validatedSearchRequestBodyDTO, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return validatedSearchRequestBodyDTO{}, err
	}
	var dto validatedSearchRequestBodyDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		return validatedSearchRequestBodyDTO{}, err
	}
	return dto, nil
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
	if len(items) > maxSearchFilters {
		return errors.New("too many filters")
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
	if len(items) > maxSubstitutionInputs {
		return errors.New("too many substitution inputs")
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
		quantity, ok := input["quantity"].(float64)
		if !ok {
			return errors.New("substitution quantity is invalid")
		}
		if _, err := security.NormalizeInput(security.InputFieldSubstitutionQuantity, strconv.FormatFloat(quantity, 'f', -1, 64)); err != nil {
			return errors.New("substitution quantity is invalid")
		}
		unit, ok := input["unit"].(string)
		if !ok {
			return errors.New("substitution unit is required")
		}
		if err := validateSubstitutionUnit(unit); err != nil {
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

// parseValidatedSearchFilterDTOs converts validated filter DTO values for conflict checks.
// Implements DESIGN-010 RequestValidator and DESIGN-002 FilterProcessor.
func parseValidatedSearchFilterDTOs(items []validatedSearchFilterDTO) ([]search.SearchFilter, error) {
	filters := make([]search.SearchFilter, 0, len(items))
	for _, filter := range items {
		if filter.FilterID == "" {
			return nil, errors.New("filter id is invalid")
		}
		if filter.Kind == "" {
			return nil, errors.New("filter kind is required")
		}
		if filter.Include == nil {
			return nil, errors.New("filter include is invalid")
		}
		filters = append(filters, search.SearchFilter{
			FilterID: filter.FilterID,
			Kind:     search.SearchFilterKind(filter.Kind),
			Include:  *filter.Include,
		})
	}
	return filters, nil
}

// parseValidatedSubstitutionInputDTOs converts validated substitution DTO values for strategy checks.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func parseValidatedSubstitutionInputDTOs(items []validatedSubstitutionInputDTO) ([]search.SubstitutionInput, error) {
	inputs := make([]search.SubstitutionInput, 0, len(items))
	for _, input := range items {
		foodObjectID, err := uuid.Parse(input.FoodObjectID)
		if err != nil {
			return nil, errors.New("food object id is invalid")
		}
		if input.Quantity == nil {
			return nil, errors.New("substitution quantity is invalid")
		}
		if input.Unit == "" {
			return nil, errors.New("substitution unit is required")
		}
		if err := validateSubstitutionUnit(input.Unit); err != nil {
			return nil, errors.New("substitution unit is invalid")
		}
		inputs = append(inputs, search.SubstitutionInput{
			FoodObjectID: foodObjectID,
			Quantity:     *input.Quantity,
			Unit:         input.Unit,
		})
	}
	return inputs, nil
}

// validateSubstitutionUnit validates canonical public API substitution units.
// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController.
func validateSubstitutionUnit(unit string) error {
	if _, err := security.NormalizeInput(security.InputFieldSubstitutionUnit, unit); err != nil {
		return err
	}
	switch unit {
	case "g", "ml", "oz", "fl_oz":
		return nil
	default:
		return errors.New("substitution unit is invalid")
	}
}

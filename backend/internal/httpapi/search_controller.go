package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// SearchService defines Catalog Search orchestration for HTTP handlers.
// Implements DESIGN-002 SearchController.
type SearchService interface {
	Search(context.Context, search.SearchRequest) (search.SearchResponse, error)
}

// AutocompleteService defines autocomplete orchestration for HTTP handlers.
// Implements DESIGN-002 SearchController.
type AutocompleteService interface {
	Autocomplete(context.Context, string, repository.RepositoryContext) (search.AutocompleteResponse, error)
}

// SearchHistoryAppender defines authenticated search-history persistence for public search routes.
// Implements DESIGN-008 SearchHistoryRepository.
type SearchHistoryAppender interface {
	AddHistory(ctx context.Context, userID uuid.UUID, query string, mode string, filtersHash string) (uuid.UUID, error)
}

// SearchController owns Catalog Search endpoint handlers.
// Implements DESIGN-002 SearchController and DESIGN-008 SearchHistoryRepository.
type SearchController struct {
	service      SearchService
	autocomplete AutocompleteService
	history      SearchHistoryAppender
}

// Implements DESIGN-002 SearchController compile-time route controller contract.
var _ Controller = (*SearchController)(nil)

// NewSearchController creates Catalog Search handlers.
// Implements DESIGN-002 SearchController.
func NewSearchController(service SearchService) *SearchController {
	return &SearchController{service: service}
}

// WithSearchHistoryAppender enables authenticated search-history persistence.
// Implements DESIGN-008 SearchHistoryRepository.
func (c *SearchController) WithSearchHistoryAppender(history SearchHistoryAppender) *SearchController {
	c.history = history
	return c
}

// WithAutocompleteService enables ranked autocomplete route exposure.
// Implements DESIGN-002 SearchController.
func (c *SearchController) WithAutocompleteService(autocomplete AutocompleteService) *SearchController {
	c.autocomplete = autocomplete
	return c
}

// Routes returns public search routes.
// Implements DESIGN-002 SearchController and DESIGN-008 SearchHistoryRepository.
func (c *SearchController) Routes() []RouteDefinition {
	routes := []RouteDefinition{{Method: fiber.MethodPost, Path: "/search", OptionalAuth: c.history != nil, ExemptCSRF: true, Validate: ValidateJSON(ValidateSearchRequestBody), RateLimit: &RateLimitRule{Scope: "endpoint", MaxRequests: 120, WindowSeconds: 60}, Handler: c.Search}}
	if c.autocomplete != nil {
		routes = append(routes, RouteDefinition{Method: fiber.MethodGet, Path: "/search/autocomplete", OptionalAuth: true, Validate: ValidateQuery(ValidateAutocompleteQueryParams), RateLimit: &RateLimitRule{Scope: "endpoint", MaxRequests: 240, WindowSeconds: 60}, Handler: c.Autocomplete})
	}
	return routes
}

// Search returns Catalog Search results in the shared response envelope.
// Implements DESIGN-002 SearchController.
func (c *SearchController) Search(ctx *fiber.Ctx) error {
	req, err := ParseSearchRequest(ctx)
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
	}
	response, err := c.service.Search(ctx.UserContext(), req)
	if err != nil {
		if errors.Is(err, search.ErrDailyDietIDRequired) {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
		}
		var similarityErr search.SimilarityUnavailableError
		if errors.As(err, &similarityErr) {
			return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "similarity_unavailable", Message: "service temporarily unavailable", Retryable: true, Cause: err}
		}
		return err
	}
	if response.Rejection != nil {
		appErr := AppError{HTTPStatus: fiber.StatusUnprocessableEntity, Category: "validation", Code: response.Rejection.Code, Message: response.Rejection.Message}
		appErr.RequestID = requestID(ctx)
		return ctx.Status(fiber.StatusUnprocessableEntity).JSON(Envelope{Status: "error", RequestID: appErr.RequestID, Data: map[string]any{"rejection": searchRejectionData(*response.Rejection)}, Error: &appErr})
	}
	if err := c.appendAuthenticatedHistory(ctx, req); err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: searchResponseData(response)})
}

// Autocomplete returns ranked food and meal suggestions in the shared response envelope.
// Implements DESIGN-002 SearchController.
func (c *SearchController) Autocomplete(ctx *fiber.Ctx) error {
	if c.autocomplete == nil {
		return fiber.ErrNotFound
	}
	query := ctx.Query("query")
	if query == "" {
		query = ctx.Query("q")
	}
	response, err := c.autocomplete.Autocomplete(ctx.UserContext(), query, repositoryContextFromAuth(ctx))
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: autocompleteResponseData(response)})
}

// repositoryContextFromAuth derives repository context from trusted authentication state.
// Implements DESIGN-002 SearchController.
func repositoryContextFromAuth(ctx *fiber.Ctx) repository.RepositoryContext {
	if user, ok := authenticatedUser(ctx); ok {
		return repository.RepositoryContext{UserID: &user.UserID}
	}
	return repository.RepositoryContext{}
}

// appendAuthenticatedHistory records completed searches for the server-derived user only.
// Implements DESIGN-008 SearchHistoryRepository.
func (c *SearchController) appendAuthenticatedHistory(ctx *fiber.Ctx, req search.SearchRequest) error {
	if c.history == nil {
		return nil
	}
	user, ok := authenticatedUser(ctx)
	if !ok {
		return nil
	}
	_, err := c.history.AddHistory(ctx.UserContext(), user.UserID, req.Query, string(req.Mode), searchFiltersHash(req.Filters))
	return err
}

// ParseSearchRequest converts a validated Fiber body to a backend SearchRequest.
// Implements DESIGN-002 SearchController.
func ParseSearchRequest(ctx *fiber.Ctx) (search.SearchRequest, error) {
	body := map[string]any{}
	if err := ctx.BodyParser(&body); err != nil {
		return search.SearchRequest{}, err
	}
	req := search.SearchRequest{
		Query:   body["query"].(string),
		Mode:    search.SearchMode(body["mode"].(string)),
		Page:    parsePage(body["page"]),
		Filters: parseFilters(body["filters"]),
	}
	if inputs, ok := body["substitutionInputs"]; ok {
		req.SubstitutionInputs = parseSubstitutionInputs(inputs)
	}
	if rawDailyDietID, ok := body["dailyDietId"].(string); ok && rawDailyDietID != "" {
		id, err := uuid.Parse(rawDailyDietID)
		if err != nil {
			return search.SearchRequest{}, err
		}
		req.DailyDietID = &id
	}
	return req, nil
}

// parsePage converts validated JSON page values to an integer.
// Implements DESIGN-002 PaginationHandler.
func parsePage(value any) int {
	switch page := value.(type) {
	case float64:
		return int(page)
	case int:
		return page
	case string:
		parsed, _ := strconv.Atoi(page)
		return parsed
	default:
		return 1
	}
}

// parseFilters converts validated JSON filter values to service filters.
// Implements DESIGN-002 FilterProcessor.
func parseFilters(value any) []search.SearchFilter {
	items, ok := value.([]any)
	if !ok {
		return []search.SearchFilter{}
	}
	filters := make([]search.SearchFilter, 0, len(items))
	for _, item := range items {
		filter := item.(map[string]any)
		filters = append(filters, search.SearchFilter{FilterID: filter["filterId"].(string), Kind: search.SearchFilterKind(filter["kind"].(string)), Include: filter["include"].(bool)})
	}
	return filters
}

// parseSubstitutionInputs converts validated JSON substitution values to service inputs.
// Implements DESIGN-002 SearchController.
func parseSubstitutionInputs(value any) []search.SubstitutionInput {
	items, ok := value.([]any)
	if !ok {
		return []search.SubstitutionInput{}
	}
	inputs := make([]search.SubstitutionInput, 0, len(items))
	for _, item := range items {
		input := item.(map[string]any)
		foodObjectID, _ := uuid.Parse(input["foodObjectId"].(string))
		inputs = append(inputs, search.SubstitutionInput{FoodObjectID: foodObjectID, Quantity: parseQuantity(input["quantity"]), Unit: input["unit"].(string)})
	}
	return inputs
}

// parseQuantity converts validated JSON quantities to float values.
// Implements DESIGN-002 SearchController.
func parseQuantity(value any) float64 {
	switch quantity := value.(type) {
	case float64:
		return quantity
	case int:
		return float64(quantity)
	case string:
		parsed, _ := strconv.ParseFloat(quantity, 64)
		return parsed
	default:
		return 0
	}
}

// searchResponseData maps service search output to the OpenAPI response shape.
// Implements DESIGN-002 SearchController.
func searchResponseData(response search.SearchResponse) map[string]any {
	data := map[string]any{
		"items":              foodItemsData(response.Items),
		"totalCount":         response.TotalCount,
		"page":               response.Page,
		"similarityScores":   response.SimilarityScores,
		"similarityMetadata": similarityMetadataData(response.SimilarityMetadata),
		"warnings":           response.Warnings,
	}
	if response.Cache != nil {
		data["cache"] = map[string]any{"status": string(response.Cache.Status), "namespace": response.Cache.Namespace, "schemaVersion": response.Cache.SchemaVersion, "ttlSeconds": response.Cache.TTLSeconds}
	}
	return data
}

// autocompleteResponseData maps autocomplete output to the OpenAPI response shape.
// Implements DESIGN-002 SearchController.
func autocompleteResponseData(response search.AutocompleteResponse) map[string]any {
	data := map[string]any{"items": autocompleteItemsData(response.Items)}
	if response.Cache != nil {
		data["cache"] = map[string]any{"status": string(response.Cache.Status), "namespace": response.Cache.Namespace, "schemaVersion": response.Cache.SchemaVersion, "ttlSeconds": response.Cache.TTLSeconds}
	}
	return data
}

// autocompleteItemsData maps ranked autocomplete entries to response items.
// Implements DESIGN-002 AutocompleteRanker.
func autocompleteItemsData(items []search.RankedAutocomplete) []map[string]any {
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, map[string]any{"itemId": item.ItemID, "label": item.Label, "exactMatch": item.ExactMatch, "levenshteinDistance": item.LevenshteinDistance, "length": item.Length, "rank": item.Rank})
	}
	return data
}

// foodItemsData maps repository food entities to response items.
// Implements DESIGN-002 SearchController.
func foodItemsData(items []repository.FoodItemEntity) []map[string]any {
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, map[string]any{"id": item.ID.String(), "name": item.Name, "physicalState": string(item.PhysicalState), "imageUrl": item.ImageURL})
	}
	return data
}

// similarityMetadataData maps similarity metadata to response items.
// Implements DESIGN-003 SimilarityIndicatorMapper.
func similarityMetadataData(metadata []search.SimilarityMetadata) []map[string]any {
	data := make([]map[string]any, 0, len(metadata))
	for _, item := range metadata {
		data = append(data, map[string]any{
			"itemId":           item.ItemID.String(),
			"score":            item.Score,
			"tier":             string(item.Tier),
			"colorHex":         item.ColorHex,
			"imageUrl":         item.ImageURL,
			"matchingQuantity": item.MatchingQuantity,
		})
	}
	return data
}

// searchRejectionData maps rejected search details to the error response shape.
// Implements DESIGN-002 SearchController.
func searchRejectionData(rejection search.SearchRejection) map[string]any {
	data := map[string]any{"code": rejection.Code, "message": rejection.Message}
	if rejection.Field != "" {
		data["field"] = rejection.Field
	}
	return data
}

// searchFiltersHash stores a stable non-PII fingerprint for filter context.
// Implements DESIGN-008 SearchHistoryRepository.
func searchFiltersHash(filters []search.SearchFilter) string {
	payload, err := json.Marshal(filters)
	if err != nil || len(payload) == 0 || string(payload) == "null" {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

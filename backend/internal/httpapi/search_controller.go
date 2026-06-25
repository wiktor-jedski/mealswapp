package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

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

// searchResponseDTO is the public HTTP payload for successful search responses.
// Implements DESIGN-002 SearchController.
type searchResponseDTO struct {
	Items              []foodObjectDTO         `json:"items"`
	TotalCount         int                     `json:"totalCount"`
	Page               int                     `json:"page"`
	SimilarityScores   []float64               `json:"similarityScores"`
	SimilarityMetadata []similarityMetadataDTO `json:"similarityMetadata"`
	SourceSummary      *sourceSummaryDTO       `json:"sourceSummary,omitempty"`
	Warnings           []string                `json:"warnings"`
	Cache              *searchCacheMetadataDTO `json:"cache,omitempty"`
}

// autocompleteResponseDTO is the public HTTP payload for successful autocomplete responses.
// Implements DESIGN-002 SearchController.
type autocompleteResponseDTO struct {
	Items []autocompleteItemDTO   `json:"items"`
	Cache *searchCacheMetadataDTO `json:"cache,omitempty"`
}

// foodObjectDTO is the narrow public Food Object search result.
// Implements DESIGN-002 SearchController.
type foodObjectDTO struct {
	ID                  string                     `json:"id"`
	Name                string                     `json:"name"`
	PhysicalState       string                     `json:"physicalState"`
	ImageURL            string                     `json:"imageUrl"`
	Classifications     []classificationSummaryDTO `json:"classifications"`
	PrimaryFoodCategory *classificationSummaryDTO  `json:"primaryFoodCategory"`
	Macros              macroProfileDTO            `json:"macros"`
	MacroBasis          string                     `json:"macroBasis"`
	Calories            float64                    `json:"calories"`
}

// classificationSummaryDTO is the public classification identity summary.
// Implements DESIGN-002 SearchController.
type classificationSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// macroProfileDTO is the public protein/carbohydrate/fat macro profile.
// Implements DESIGN-002 SearchController.
type macroProfileDTO struct {
	Protein       float64 `json:"protein"`
	Carbohydrates float64 `json:"carbohydrates"`
	Fat           float64 `json:"fat"`
}

// sourceSummaryDTO reports the user's substitution input totals without cross-basis mass/volume assumptions.
// Implements DESIGN-002 SearchController.
type sourceSummaryDTO struct {
	Macros           macroProfileDTO `json:"macros"`
	Calories         float64         `json:"calories"`
	TotalGrams       float64         `json:"totalGrams"`
	TotalMilliliters float64         `json:"totalMilliliters"`
}

// autocompleteItemDTO is one ranked autocomplete suggestion.
// Implements DESIGN-002 AutocompleteRanker.
type autocompleteItemDTO struct {
	ItemID              string `json:"itemId"`
	Label               string `json:"label"`
	ExactMatch          bool   `json:"exactMatch"`
	LevenshteinDistance int    `json:"levenshteinDistance"`
	Length              int    `json:"length"`
	Rank                int    `json:"rank"`
}

// similarityMetadataDTO is one public similarity display metadata entry.
// Implements DESIGN-003 SimilarityIndicatorMapper.
type similarityMetadataDTO struct {
	ItemID           string  `json:"itemId"`
	Score            float64 `json:"score"`
	Tier             string  `json:"tier"`
	ImageURL         string  `json:"imageUrl"`
	MatchingQuantity float64 `json:"matchingQuantity"`
}

// searchCacheMetadataDTO is cache metadata safe to expose over HTTP.
// Implements DESIGN-011 RedisCache response metadata.
type searchCacheMetadataDTO struct {
	Status        string `json:"status"`
	Namespace     string `json:"namespace"`
	SchemaVersion string `json:"schemaVersion"`
	TTLSeconds    int64  `json:"ttlSeconds"`
}

// searchRejectionDTO is the public rejected-search payload.
// Implements DESIGN-002 SearchController.
type searchRejectionDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
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
	data, err := envelopeData(searchResponseData(response))
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: data})
}

// Autocomplete returns ranked food and meal suggestions in the shared response envelope.
// Implements DESIGN-002 SearchController.
func (c *SearchController) Autocomplete(ctx *fiber.Ctx) error {
	if c.autocomplete == nil {
		return fiber.ErrNotFound
	}
	query := ctx.Query("query")
	response, err := c.autocomplete.Autocomplete(ctx.UserContext(), query, repositoryContextFromAuth(ctx))
	if err != nil {
		return err
	}
	data, err := envelopeData(autocompleteResponseData(response))
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: data})
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
	var dto validatedSearchRequestBodyDTO
	if err := ctx.BodyParser(&dto); err != nil {
		return search.SearchRequest{}, err
	}
	return searchRequestFromValidatedDTO(dto)
}

// searchResponseData maps service search output to the OpenAPI response DTO.
// Implements DESIGN-002 SearchController.
func searchResponseData(response search.SearchResponse) searchResponseDTO {
	return searchResponseDTO{
		Items:              foodItemsData(response.Items),
		TotalCount:         response.TotalCount,
		Page:               response.Page,
		SimilarityScores:   response.SimilarityScores,
		SimilarityMetadata: similarityMetadataData(response.SimilarityMetadata),
		SourceSummary:      sourceSummaryData(response.SourceSummary),
		Warnings:           response.Warnings,
		Cache:              searchCacheData(response.Cache),
	}
}

// autocompleteResponseData maps autocomplete output to the OpenAPI response DTO.
// Implements DESIGN-002 SearchController.
func autocompleteResponseData(response search.AutocompleteResponse) autocompleteResponseDTO {
	return autocompleteResponseDTO{
		Items: autocompleteItemsData(response.Items),
		Cache: searchCacheData(response.Cache),
	}
}

// autocompleteItemsData maps ranked autocomplete entries to response items.
// Implements DESIGN-002 AutocompleteRanker.
func autocompleteItemsData(items []search.RankedAutocomplete) []autocompleteItemDTO {
	data := make([]autocompleteItemDTO, 0, len(items))
	for _, item := range items {
		data = append(data, autocompleteItemDTO{ItemID: item.ItemID, Label: item.Label, ExactMatch: item.ExactMatch, LevenshteinDistance: item.LevenshteinDistance, Length: item.Length, Rank: item.Rank})
	}
	return data
}

// foodItemsData maps repository food entities to response items.
// Implements DESIGN-002 SearchController.
func foodItemsData(items []repository.FoodItemEntity) []foodObjectDTO {
	data := make([]foodObjectDTO, 0, len(items))
	for _, item := range items {
		data = append(data, foodItemData(item))
	}
	return data
}

// foodItemData maps one repository food entity to the public FoodObject DTO.
// Implements DESIGN-002 SearchController.
func foodItemData(item repository.FoodItemEntity) foodObjectDTO {
	return foodObjectDTO{
		ID:                  item.ID.String(),
		Name:                item.Name,
		PhysicalState:       string(item.PhysicalState),
		ImageURL:            item.ImageURL,
		Classifications:     classificationSummariesData(item.FoodCategories, item.CulinaryRoles),
		PrimaryFoodCategory: primaryFoodCategoryData(item.FoodCategories),
		Macros:              macroProfileData(item.MacrosPer100),
		MacroBasis:          macroBasisForState(item.PhysicalState),
		Calories:            search.CalculateCalories(item.MacrosPer100),
	}
}

// classificationSummariesData maps food categories and culinary roles to public classification summaries.
// Implements DESIGN-002 SearchController.
func classificationSummariesData(foodCategories []repository.ClassificationEntity, culinaryRoles []repository.ClassificationEntity) []classificationSummaryDTO {
	data := make([]classificationSummaryDTO, 0, len(foodCategories)+len(culinaryRoles))
	for _, category := range foodCategories {
		data = append(data, classificationSummaryDTO{ID: category.ID.String(), Name: category.Name, Kind: string(category.Kind)})
	}
	for _, role := range culinaryRoles {
		data = append(data, classificationSummaryDTO{ID: role.ID.String(), Name: role.Name, Kind: string(role.Kind)})
	}
	return data
}

// primaryFoodCategoryData returns the first food category summary or nil when absent.
// Implements DESIGN-002 SearchController.
func primaryFoodCategoryData(foodCategories []repository.ClassificationEntity) *classificationSummaryDTO {
	if len(foodCategories) == 0 {
		return nil
	}
	category := foodCategories[0]
	return &classificationSummaryDTO{ID: category.ID.String(), Name: category.Name, Kind: string(category.Kind)}
}

// macroProfileData maps repository macro values to the public macro profile.
// Implements DESIGN-002 SearchController.
func macroProfileData(macros repository.MacroValues) macroProfileDTO {
	return macroProfileDTO{Protein: macros.Protein, Carbohydrates: macros.Carbohydrates, Fat: macros.Fat}
}

// sourceSummaryData maps substitution input totals to the public response DTO.
// Implements DESIGN-002 SearchController.
func sourceSummaryData(summary *search.SubstitutionSourceSummary) *sourceSummaryDTO {
	if summary == nil {
		return nil
	}
	return &sourceSummaryDTO{
		Macros:           macroProfileData(summary.Macros),
		Calories:         summary.Calories,
		TotalGrams:       summary.TotalGrams,
		TotalMilliliters: summary.TotalMilliliters,
	}
}

// macroBasisForState derives the 100g/100ml macro basis from physical state.
// Implements DESIGN-002 SearchController.
func macroBasisForState(state repository.PhysicalState) string {
	if state == repository.PhysicalStateLiquid {
		return "100ml"
	}
	return "100g"
}

// similarityMetadataData maps similarity metadata to response items.
// Implements DESIGN-003 SimilarityIndicatorMapper.
func similarityMetadataData(metadata []search.SimilarityMetadata) []similarityMetadataDTO {
	data := make([]similarityMetadataDTO, 0, len(metadata))
	for _, item := range metadata {
		data = append(data, similarityMetadataDTO{ItemID: item.ItemID.String(), Score: item.Score, Tier: string(item.Tier), ImageURL: item.ImageURL, MatchingQuantity: item.MatchingQuantity})
	}
	return data
}

// searchCacheData maps internal cache metadata to the public HTTP DTO.
// Implements DESIGN-011 RedisCache response metadata.
func searchCacheData(cache *search.CacheMetadata) *searchCacheMetadataDTO {
	if cache == nil {
		return nil
	}
	return &searchCacheMetadataDTO{Status: string(cache.Status), Namespace: cache.Namespace, SchemaVersion: cache.SchemaVersion, TTLSeconds: cache.TTLSeconds}
}

// searchRejectionData maps rejected search details to the error response shape.
// Implements DESIGN-002 SearchController.
func searchRejectionData(rejection search.SearchRejection) searchRejectionDTO {
	return searchRejectionDTO{Code: rejection.Code, Message: rejection.Message, Field: rejection.Field}
}

// envelopeData converts typed route DTOs into the current shared envelope map.
// Implements DESIGN-017 GlobalExceptionHandler.
func envelopeData(dto any) (map[string]any, error) {
	payload, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("marshal response DTO: %w", err)
	}
	data := map[string]any{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("map response DTO: %w", err)
	}
	return data, nil
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

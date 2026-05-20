package handlers

import (
	"context"
	"net/http"
	"strings"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"
	"mealswapp/backend/internal/services/entitlements"
	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/gofiber/fiber/v2"
)

const maxSearchPageSize = 10

type SearchService interface {
	Search(ctx context.Context, request SearchRequest) (SearchResponse, error)
	Autocomplete(ctx context.Context, request AutocompleteRequest) ([]searchsvc.RankedAutocomplete, error)
}

type SearchHandler struct {
	service      SearchService
	usageLimiter SearchUsageLimiter
}

type SearchUsageLimiter interface {
	CheckAndRecord(ctx context.Context, accessToken string, mode searchsvc.Mode) (entitlements.Decision, error)
}

type SearchRequest struct {
	Query       string                      `json:"query"`
	Mode        searchsvc.Mode              `json:"mode"`
	Page        int                         `json:"page"`
	Filters     []searchsvc.TagFilter       `json:"filters"`
	Ingredients []searchsvc.IngredientInput `json:"ingredients"`
	SourceItem  string                      `json:"sourceItemId"`
	FilterQuery searchsvc.RepositoryQuery   `json:"-"`
}

type SearchResponse struct {
	Items            []any     `json:"items"`
	TotalCount       int       `json:"totalCount"`
	Page             int       `json:"page"`
	PageSize         int       `json:"pageSize"`
	SimilarityScores []float64 `json:"similarityScores"`
	Warnings         []string  `json:"warnings"`
}

type AutocompleteRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type searchRequestPayload struct {
	Query           string                      `json:"query"`
	Mode            searchsvc.Mode              `json:"mode"`
	Page            int                         `json:"page"`
	Filters         []searchsvc.TagFilter       `json:"filters"`
	Ingredients     []searchsvc.IngredientInput `json:"ingredients"`
	SourceItemID    string                      `json:"sourceItemId"`
	EnabledMacros   map[string]bool             `json:"enabledMacros"`
	DietaryTagIDs   []string                    `json:"dietaryTagIds"`
	AllergenTagIDs  []string                    `json:"allergenTagIds"`
	SourceProviders []string                    `json:"sourceProviders"`
}

func NewSearchHandler(service SearchService) SearchHandler {
	return SearchHandler{service: service}
}

func NewSearchHandlerWithUsageLimiter(service SearchService, usageLimiter SearchUsageLimiter) SearchHandler {
	return SearchHandler{service: service, usageLimiter: usageLimiter}
}

func (handler SearchHandler) Search(ctx *fiber.Ctx) error {
	payload, err := validation.DecodeJSON[searchRequestPayload](ctx)
	if err != nil {
		return err
	}

	payload.Query = strings.TrimSpace(payload.Query)
	if payload.Page == 0 {
		payload.Page = 1
	}
	if payload.Page < 1 {
		return apperrors.Validation("Search request validation failed", []map[string]string{{"field": "page", "code": "min"}})
	}
	if payload.Mode == "" {
		payload.Mode = searchsvc.ModeSingle
	}

	parsed, err := searchsvc.ParseQuery(searchsvc.QueryInput{
		Mode:        payload.Mode,
		Query:       payload.Query,
		SourceItem:  payload.SourceItemID,
		Ingredients: payload.Ingredients,
	})
	if err != nil {
		return err
	}
	if handler.usageLimiter != nil {
		decision, err := handler.usageLimiter.CheckAndRecord(ctx.Context(), bearerToken(ctx), parsed.Mode)
		if err != nil {
			return err
		}
		if !decision.Allowed {
			return apperrors.AppError{
				Category: apperrors.CategoryEntitlement,
				Code:     decision.Code,
				Message:  decision.Reason,
				Status:   http.StatusPaymentRequired,
				Fields:   decision.Entitlement,
			}
		}
	}

	filterQuery, err := searchsvc.ApplyFilters(searchsvc.FilterInput{
		TagFilters:       payload.Filters,
		EnabledMacros:    payload.EnabledMacros,
		DietaryTagIDs:    payload.DietaryTagIDs,
		AllergenTagIDs:   payload.AllergenTagIDs,
		SourceProviders:  payload.SourceProviders,
		NormalizedSearch: parsed.Query,
		Limit:            maxSearchPageSize,
		Offset:           (payload.Page - 1) * maxSearchPageSize,
	})
	if err != nil {
		return err
	}

	result, err := handler.service.Search(ctx.Context(), SearchRequest{
		Query:       parsed.Query,
		Mode:        parsed.Mode,
		Page:        payload.Page,
		Filters:     payload.Filters,
		Ingredients: parsed.Ingredients,
		SourceItem:  parsed.SourceItem,
		FilterQuery: filterQuery,
	})
	if err != nil {
		return err
	}
	result.Page = payload.Page
	if result.PageSize == 0 {
		result.PageSize = maxSearchPageSize
	}
	if result.Items == nil {
		result.Items = []any{}
	}
	if result.SimilarityScores == nil {
		result.SimilarityScores = []float64{}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}

	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler SearchHandler) Autocomplete(ctx *fiber.Ctx) error {
	query := strings.TrimSpace(ctx.Query("query", ctx.Query("q")))
	limit := ctx.QueryInt("limit", maxSearchPageSize)
	if limit < 1 || limit > maxSearchPageSize {
		return apperrors.Validation("Autocomplete request validation failed", []map[string]string{{"field": "limit", "code": "range"}})
	}
	if query == "" {
		return ctx.JSON(responses.Success([]searchsvc.RankedAutocomplete{}, requestID(ctx)))
	}

	result, err := handler.service.Autocomplete(ctx.Context(), AutocompleteRequest{Query: query, Limit: limit})
	if err != nil {
		return err
	}
	if result == nil {
		result = []searchsvc.RankedAutocomplete{}
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

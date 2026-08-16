package httpapi

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// FilterOptionReader defines backend-owned search filter-option reads.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
type FilterOptionReader interface {
	Options(context.Context, search.SearchMode) (search.FilterOptionsResponse, error)
}

// FilterOptionController exposes public read-only filter policy.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
type FilterOptionController struct {
	service FilterOptionReader
}

// filterOptionDTO is one localized-label-ready public filter option.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
type filterOptionDTO struct {
	FilterID       string                     `json:"filterId"`
	Kind           string                     `json:"kind"`
	Label          string                     `json:"label"`
	LabelKey       string                     `json:"labelKey,omitempty"`
	IncludeAllowed bool                       `json:"includeAllowed"`
	ExcludeAllowed bool                       `json:"excludeAllowed"`
	Excludes       []filterOptionReferenceDTO `json:"excludes"`
}

// filterOptionReferenceDTO is one projected backend policy dependency.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
type filterOptionReferenceDTO struct {
	FilterID string `json:"filterId"`
	Kind     string `json:"kind"`
}

// Implements DESIGN-009 TagManager compile-time route controller contract.
var _ Controller = (*FilterOptionController)(nil)

// NewFilterOptionController creates public filter-option handlers.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
func NewFilterOptionController(service FilterOptionReader) *FilterOptionController {
	return &FilterOptionController{service: service}
}

// Routes returns the anonymous read-only filter-option route.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
func (c *FilterOptionController) Routes() []RouteDefinition {
	return []RouteDefinition{{Method: fiber.MethodGet, Path: "/search/filter-options", Validate: ValidateQuery(ValidateFilterOptionQueryParams), RateLimit: &RateLimitRule{Scope: "endpoint", MaxRequests: 240, WindowSeconds: 60}, Handler: c.Get}}
}

// Get returns deterministic backend-owned filter options.
// Implements DESIGN-009 TagManager filter-option HTTP boundary.
func (c *FilterOptionController) Get(ctx *fiber.Ctx) error {
	response, err := c.service.Options(ctx.UserContext(), search.SearchMode(ctx.Query("mode")))
	if err != nil {
		return err
	}
	options := make([]filterOptionDTO, 0, len(response.Options))
	for _, option := range response.Options {
		excludes := make([]filterOptionReferenceDTO, 0, len(option.Excludes))
		for _, excluded := range option.Excludes {
			excludes = append(excludes, filterOptionReferenceDTO{FilterID: excluded.FilterID, Kind: string(excluded.Kind)})
		}
		options = append(options, filterOptionDTO{FilterID: option.FilterID, Kind: string(option.Kind), Label: option.Label, LabelKey: option.LabelKey, IncludeAllowed: option.IncludeAllowed, ExcludeAllowed: option.ExcludeAllowed, Excludes: excludes})
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"mode": response.Mode, "options": options}})
}

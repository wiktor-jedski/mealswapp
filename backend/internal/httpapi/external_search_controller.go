package httpapi

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/externaldata"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// ExternalSearchService is the read-only ARCH-012 orchestration dependency.
// Implements DESIGN-009 ExternalSearchProxy.
type ExternalSearchService interface {
	Search(context.Context, externaldata.ExternalSearchQuery) (externaldata.ExternalSearchResponse, error)
}

// WithExternalSearch registers the documented read-only admin search route.
// Implements DESIGN-009 AdminController and ExternalSearchProxy.
func (c *AdminController) WithExternalSearch(service ExternalSearchService, logs observability.LogSink) *AdminController {
	c.externalSearch = service
	validator := NewCurationRequestValidator(logs)
	c.routes = append(c.routes, AdminRouteDefinition{
		Method: fiber.MethodGet, Path: "/external-search", Handler: c.SearchExternal,
		Validate:  validator.ValidateExternalSearchQuery,
		RateLimit: &RateLimitRule{Scope: "user", MaxRequests: 30, WindowSeconds: 60},
	})
	return c
}

// SearchExternal returns normalized candidates without invoking mutation or audit persistence.
// Implements DESIGN-009 AdminController SearchExternal and ExternalSearchProxy.
func (c *AdminController) SearchExternal(ctx *fiber.Ctx) error {
	request, ok := NormalizedExternalSearchRequest(ctx)
	if !ok {
		return curationValidationError()
	}
	if c == nil || c.externalSearch == nil {
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true}
	}
	response, err := c.externalSearch.Search(ctx.UserContext(), externaldata.ExternalSearchQuery{
		Query: request.Query, Provider: request.Provider, Page: request.Page,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true, Cause: err}
	}
	return ctx.Status(fiber.StatusOK).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{
		"candidates": response.Candidates, "warnings": response.Warnings, "page": response.Page,
	}})
}

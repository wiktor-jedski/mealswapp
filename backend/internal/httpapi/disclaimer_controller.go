package httpapi

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/compliance"
)

// DisclaimerService defines disclaimer retrieval for HTTP handlers.
// Implements DESIGN-015 DisclaimerRenderer.
type DisclaimerService interface {
	GetDisclaimer(context.Context, string) (compliance.DisclaimerContent, error)
}

// DisclaimerController owns public disclaimer content routes.
// Implements DESIGN-015 DisclaimerRenderer.
type DisclaimerController struct {
	service DisclaimerService
}

// NewDisclaimerController creates disclaimer handlers.
// Implements DESIGN-015 DisclaimerRenderer.
func NewDisclaimerController(service DisclaimerService) *DisclaimerController {
	return &DisclaimerController{service: service}
}

// Routes returns public disclaimer content routes.
// Implements DESIGN-015 DisclaimerRenderer.
func (c *DisclaimerController) Routes() []RouteDefinition {
	return []RouteDefinition{{Method: fiber.MethodGet, Path: "/disclaimers", Handler: c.GetDisclaimer}}
}

// GetDisclaimer returns stable Markdown disclaimer content.
// Implements DESIGN-015 DisclaimerRenderer.
func (c *DisclaimerController) GetDisclaimer(ctx *fiber.Ctx) error {
	content, err := c.service.GetDisclaimer(ctx.UserContext(), ctx.Query("location", "login"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	ctx.Set("Cache-Control", "public, max-age=300")
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"location": content.Location, "version": content.Version, "markdown": content.Markdown, "fallback": content.Fallback, "alert": content.Alert}})
}

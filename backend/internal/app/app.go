package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
)

// New constructs the backend Fiber app from HTTP API dependencies.
//
// Implements DESIGN-010 RouteHandler app constructor seam.
func New(deps httpapi.Dependencies) *fiber.App {
	return httpapi.NewRouter(deps)
}

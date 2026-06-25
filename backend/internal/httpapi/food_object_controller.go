package httpapi

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// FoodObjectLookup defines public food-object detail lookup for UI hydration.
// Implements DESIGN-002 SearchController.
type FoodObjectLookup interface {
	GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.FoodItemEntity, error)
}

// FoodObjectController owns public food-object detail routes.
// Implements DESIGN-002 SearchController.
type FoodObjectController struct {
	lookup FoodObjectLookup
}

// Implements DESIGN-002 SearchController compile-time route controller contract.
var _ Controller = (*FoodObjectController)(nil)

// NewFoodObjectController creates public food-object detail handlers.
// Implements DESIGN-002 SearchController.
func NewFoodObjectController(lookup FoodObjectLookup) *FoodObjectController {
	return &FoodObjectController{lookup: lookup}
}

// Routes returns public food-object detail routes for frontend item hydration.
// Implements DESIGN-002 SearchController and DESIGN-010 RouteHandler.
func (c *FoodObjectController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/food-objects/:id", OptionalAuth: true, RateLimit: &RateLimitRule{Scope: "endpoint", MaxRequests: 240, WindowSeconds: 60}, Handler: c.GetFoodObject},
	}
}

// GetFoodObject returns one FoodObject DTO inside the shared response envelope.
// Implements DESIGN-002 SearchController.
func (c *FoodObjectController) GetFoodObject(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
	}
	item, err := c.lookup.GetByID(ctx.UserContext(), id, repositoryContextFromAuth(ctx))
	if err != nil {
		return err
	}
	data, err := envelopeData(foodItemData(item))
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: data})
}

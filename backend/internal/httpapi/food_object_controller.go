package httpapi

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// FoodObjectLookup defines public food-object detail lookup for UI hydration.
// Implements DESIGN-002 SearchController.
type FoodObjectLookup interface {
	GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.FoodItemEntity, error)
}

// MealObjectLookup defines public Meal detail lookup for Food Object hydration.
// Implements DESIGN-002 SearchController.
type MealObjectLookup interface {
	GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.MealEntity, error)
}

// FoodObjectController owns public food-object detail routes.
// Implements DESIGN-002 SearchController.
type FoodObjectController struct {
	lookup FoodObjectLookup
	meals  MealObjectLookup
}

// Implements DESIGN-002 SearchController compile-time route controller contract.
var _ Controller = (*FoodObjectController)(nil)

// NewFoodObjectController creates public food-object detail handlers.
// Implements DESIGN-002 SearchController.
func NewFoodObjectController(lookup FoodObjectLookup, meals ...MealObjectLookup) *FoodObjectController {
	controller := &FoodObjectController{lookup: lookup}
	if len(meals) > 0 {
		controller.meals = meals[0]
	}
	return controller
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
	objectType := repository.FoodObjectType(ctx.Query("objectType"))
	if objectType == repository.FoodObjectTypeMeal {
		if c.meals == nil {
			return dailyDietDependencyError()
		}
		meal, err := c.meals.GetByID(ctx.UserContext(), id, repositoryContextFromAuth(ctx))
		if err != nil {
			return err
		}
		data, err := envelopeData(mealObjectData(meal))
		if err != nil {
			return err
		}
		return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: data})
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

// mealObjectData projects one repository Meal into the shared Food Object DTO.
// Implements DESIGN-002 SearchController.
func mealObjectData(meal repository.MealEntity) foodObjectDTO {
	foodCategories := make([]repository.ClassificationEntity, 0, len(meal.Classifications))
	culinaryRoles := make([]repository.ClassificationEntity, 0, len(meal.Classifications))
	for _, classification := range meal.Classifications {
		if classification.Kind == repository.ClassificationKindFoodCategory {
			foodCategories = append(foodCategories, classification)
		} else if classification.Kind == repository.ClassificationKindCulinaryRole {
			culinaryRoles = append(culinaryRoles, classification)
		}
	}
	return foodObjectDTO{
		ID: meal.ID.String(), ObjectType: string(repository.FoodObjectTypeMeal), Name: meal.Name,
		PhysicalState: string(meal.PhysicalState), Classifications: classificationSummariesData(foodCategories, culinaryRoles),
		PrimaryFoodCategory: primaryFoodCategoryData(foodCategories), Macros: macroProfileData(meal.MacrosPer100),
		MacroBasis: macroBasisForState(meal.PhysicalState), Calories: search.CalculateCalories(meal.MacrosPer100),
	}
}

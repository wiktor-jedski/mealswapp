package httpapi

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/profile"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// ProfileService defines profile behavior for HTTP handlers.
// Implements DESIGN-008 ProfileController.
type ProfileService interface {
	GetProfile(context.Context, uuid.UUID) (profile.UserProfile, error)
	UpdatePreferences(context.Context, uuid.UUID, profile.UpdateRequest) (profile.UpdateResult, error)
}

// ProfileController owns profile and preference routes.
// Implements DESIGN-008 ProfileController.
type ProfileController struct {
	service     ProfileService
	dailyDiet   DailyDietService
	customItems CustomItemService
}

// Implements DESIGN-008 ProfileController compile-time route controller contract.
var _ Controller = (*ProfileController)(nil)

// NewProfileController creates authenticated profile handlers.
// Implements DESIGN-008 ProfileController.
func NewProfileController(service ProfileService, dailyDietServices ...DailyDietService) *ProfileController {
	controller := &ProfileController{service: service}
	if len(dailyDietServices) > 0 {
		controller.dailyDiet = dailyDietServices[0]
	}
	return controller
}

// WithCustomItems attaches authenticated custom-item behavior.
// Implements DESIGN-008 ProfileController custom-item routes.
func (c *ProfileController) WithCustomItems(service CustomItemService) *ProfileController {
	c.customItems = service
	return c
}

// Routes returns authenticated profile routes.
// Implements DESIGN-008 ProfileController.
func (c *ProfileController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/profile", RequiresAuth: true, Handler: c.GetProfile},
		{Method: fiber.MethodPut, Path: "/profile", RequiresAuth: true, RequiresCSRF: true, Validate: ValidateJSON(validateProfilePreferenceBody), Handler: c.UpdatePreferences},
		{Method: fiber.MethodGet, Path: "/daily-diets", RequiresAuth: true, Handler: c.ListDailyDiets},
		{Method: fiber.MethodPost, Path: "/daily-diets", RequiresAuth: true, RequiresCSRF: true, Validate: validateDailyDietCreate, Handler: c.CreateDailyDiet},
		{Method: fiber.MethodGet, Path: "/daily-diets/:dietId", RequiresAuth: true, Validate: ValidatePath("dietId", validateDailyDietID), Handler: c.GetDailyDiet},
		{Method: fiber.MethodPut, Path: "/daily-diets/:dietId", RequiresAuth: true, RequiresCSRF: true, Validate: func(ctx *fiber.Ctx) error {
			if err := validateDailyDietID(ctx.Params("dietId")); err != nil {
				return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
			}
			return validateDailyDietBody(ctx)
		}, Handler: c.ReplaceDailyDiet},
		{Method: fiber.MethodDelete, Path: "/daily-diets/:dietId", RequiresAuth: true, RequiresCSRF: true, Validate: ValidatePath("dietId", validateDailyDietID), Handler: c.DeleteDailyDiet},
		{Method: fiber.MethodPost, Path: "/custom-items", RequiresAuth: true, RequiresCSRF: true, Validate: validateCustomItemCreate, Handler: c.CreateCustomItem},
		{Method: fiber.MethodGet, Path: "/custom-items/:itemId", RequiresAuth: true, Validate: ValidatePath("itemId", validateCustomItemID), Handler: c.GetCustomItem},
		{Method: fiber.MethodPut, Path: "/custom-items/:itemId", RequiresAuth: true, RequiresCSRF: true, Validate: validateCustomItemUpdate, Handler: c.UpdateCustomItem},
		{Method: fiber.MethodDelete, Path: "/custom-items/:itemId", RequiresAuth: true, RequiresCSRF: true, Validate: ValidatePath("itemId", validateCustomItemID), Handler: c.DeleteCustomItem},
	}
}

// GetProfile returns the authenticated user's profile.
// Implements DESIGN-008 ProfileController.
func (c *ProfileController) GetProfile(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	profile, err := c.service.GetProfile(ctx.UserContext(), user.UserID)
	if err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: profileData(profile, false)})
}

// UpdatePreferences saves profile preferences for the authenticated user.
// Implements DESIGN-008 ProfileController.
func (c *ProfileController) UpdatePreferences(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	var req profilePreferenceRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	result, err := c.service.UpdatePreferences(ctx.UserContext(), user.UserID, profile.UpdateRequest{DisplayName: req.DisplayName, UnitSystem: repository.UnitSystem(req.UnitSystem), ThemePreference: req.ThemePreference})
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: profileData(result.Profile, result.RequiresUnitRecalculation)})
}

// profilePreferenceRequest carries mutable profile preference fields.
// Implements DESIGN-008 ProfileController.
type profilePreferenceRequest struct {
	DisplayName     *string `json:"displayName"`
	UnitSystem      string  `json:"unitSystem"`
	ThemePreference string  `json:"themePreference"`
}

// validateProfilePreferenceBody validates profile preference JSON before service dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateProfilePreferenceBody(body map[string]any) error {
	if unitSystem, ok := body["unitSystem"].(string); !ok || (unitSystem != string(repository.UnitSystemMetric) && unitSystem != string(repository.UnitSystemImperial)) {
		return errors.New("unit system is invalid")
	}
	if theme, ok := body["themePreference"].(string); !ok || (theme != "system" && theme != "light" && theme != "dark") {
		return errors.New("theme preference is invalid")
	}
	if displayName, ok := body["displayName"]; ok && displayName != nil {
		if _, ok := displayName.(string); !ok {
			return errors.New("display name is invalid")
		}
	}
	return nil
}

// profileData maps profile service data to HTTP envelope fields.
// Implements DESIGN-008 ProfileController.
func profileData(profile profile.UserProfile, requiresUnitRecalculation bool) map[string]any {
	return map[string]any{
		"userId":                    profile.UserID.String(),
		"displayName":               profile.DisplayName,
		"unitSystem":                string(profile.UnitSystem),
		"themePreference":           profile.ThemePreference,
		"requiresUnitRecalculation": requiresUnitRecalculation,
	}
}

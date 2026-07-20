package httpapi

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/dailydiet"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// DailyDietService defines authenticated saved-diet behavior for ProfileController.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
type DailyDietService interface {
	Create(context.Context, uuid.UUID, dailydiet.CreateRequest) (dailydiet.CreateResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (dailydiet.DailyDiet, error)
	List(context.Context, uuid.UUID) ([]dailydiet.DailyDiet, error)
	Replace(context.Context, uuid.UUID, uuid.UUID, dailydiet.ReplaceRequest) (dailydiet.DailyDiet, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

// CreateDailyDiet creates or replays an authenticated user's saved one-day diet.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (c *ProfileController) CreateDailyDiet(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	if c.dailyDiet == nil {
		return dailyDietDependencyError()
	}
	var req dailydiet.CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	req.IdempotencyKey = ctx.Get("Idempotency-Key")
	result, err := c.dailyDiet.Create(ctx.UserContext(), user.UserID, req)
	if err != nil {
		return dailyDietError(err)
	}
	return ctx.Status(result.Status).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: dailyDietData(result.Diet)})
}

// GetDailyDiet returns one authenticated user's saved one-day diet.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (c *ProfileController) GetDailyDiet(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	dietID, err := parseDailyDietID(ctx.Params("dietId"))
	if err != nil {
		return err
	}
	if c.dailyDiet == nil {
		return dailyDietDependencyError()
	}
	diet, err := c.dailyDiet.Get(ctx.UserContext(), user.UserID, dietID)
	if err != nil {
		return dailyDietError(err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: dailyDietData(diet)})
}

// ListDailyDiets returns all saved one-day diets owned by the authenticated user.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (c *ProfileController) ListDailyDiets(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	if c.dailyDiet == nil {
		return dailyDietDependencyError()
	}
	diets, err := c.dailyDiet.List(ctx.UserContext(), user.UserID)
	if err != nil {
		return dailyDietError(err)
	}
	data := make([]map[string]any, 0, len(diets))
	for _, diet := range diets {
		data = append(data, dailyDietData(diet))
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"diets": data}})
}

// ReplaceDailyDiet replaces one authenticated user's saved one-day diet.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (c *ProfileController) ReplaceDailyDiet(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	dietID, err := parseDailyDietID(ctx.Params("dietId"))
	if err != nil {
		return err
	}
	if c.dailyDiet == nil {
		return dailyDietDependencyError()
	}
	var req dailydiet.ReplaceRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	diet, err := c.dailyDiet.Replace(ctx.UserContext(), user.UserID, dietID, req)
	if err != nil {
		return dailyDietError(err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: dailyDietData(diet)})
}

// DeleteDailyDiet deletes one authenticated user's saved one-day diet.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (c *ProfileController) DeleteDailyDiet(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	dietID, err := parseDailyDietID(ctx.Params("dietId"))
	if err != nil {
		return err
	}
	if c.dailyDiet == nil {
		return dailyDietDependencyError()
	}
	if err := c.dailyDiet.Delete(ctx.UserContext(), user.UserID, dietID); err != nil {
		return dailyDietError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// validateDailyDietCreate validates the create header and JSON body.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateDailyDietCreate(ctx *fiber.Ctx) error {
	key := strings.TrimSpace(ctx.Get("Idempotency-Key"))
	if len(key) < 8 || len(key) > 255 {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	}
	return validateDailyDietBody(ctx)
}

// validateDailyDietBody validates a create or replacement JSON body.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateDailyDietBody(ctx *fiber.Ctx) error {
	body := map[string]any{}
	if err := ctx.BodyParser(&body); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	if err := validateDailyDietBodyMap(body); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return ctx.Next()
}

// validateDailyDietBodyMap rejects unknown fields and malformed quantities.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateDailyDietBodyMap(body map[string]any) error {
	if len(body) != 2 {
		return errors.New("daily diet body contains unsupported fields")
	}
	name, ok := body["name"].(string)
	if !ok || strings.TrimSpace(name) == "" || len([]rune(strings.TrimSpace(name))) > 120 || strings.ContainsRune(name, '\x00') {
		return errors.New("daily diet name is invalid")
	}
	rawEntries, ok := body["entries"].([]any)
	if !ok || len(rawEntries) == 0 || len(rawEntries) > 100 {
		return errors.New("daily diet entries are invalid")
	}
	positions := make(map[int]struct{}, len(rawEntries))
	for _, rawEntry := range rawEntries {
		entry, ok := rawEntry.(map[string]any)
		if !ok || len(entry) != 5 {
			return errors.New("daily diet meal entry is invalid")
		}
		foodObjectID, ok := entry["foodObjectId"].(string)
		if !ok {
			return errors.New("Food Object id is invalid")
		}
		parsedID, err := uuid.Parse(foodObjectID)
		if err != nil || parsedID == uuid.Nil {
			return errors.New("Food Object id is invalid")
		}
		foodObjectType, ok := entry["foodObjectType"].(string)
		if !ok || (foodObjectType != string(repository.FoodObjectTypeMeal) && foodObjectType != string(repository.FoodObjectTypeFoodItem)) {
			return errors.New("Food Object type is invalid")
		}
		quantity, ok := entry["quantity"].(float64)
		if !ok || math.IsNaN(quantity) || math.IsInf(quantity, 0) || quantity <= 0 || quantity > 1_000_000 {
			return errors.New("quantity is invalid")
		}
		unit, ok := entry["unit"].(string)
		if !ok || repository.ValidateQuantityUnit(unit) != nil {
			return errors.New("unit is invalid")
		}
		positionValue, ok := entry["position"].(float64)
		if !ok || math.Trunc(positionValue) != positionValue || positionValue < 0 || positionValue >= 100 {
			return errors.New("position is invalid")
		}
		position := int(positionValue)
		if _, exists := positions[position]; exists {
			return errors.New("positions must be unique")
		}
		positions[position] = struct{}{}
	}
	return nil
}

// validateDailyDietID validates a saved-diet path identifier.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateDailyDietID(value string) error {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return errors.New("daily diet id is invalid")
	}
	return nil
}

// parseDailyDietID parses a validated saved-diet path identifier.
// Implements DESIGN-008 ProfileController.
func parseDailyDietID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return id, nil
}

// dailyDietData maps a service projection without exposing ownership metadata.
// Implements DESIGN-008 ProfileController.
func dailyDietData(diet dailydiet.DailyDiet) map[string]any {
	entries := make([]map[string]any, 0, len(diet.Entries))
	for _, entry := range diet.Entries {
		foodObjectID, foodObjectType := entry.FoodObjectID, entry.FoodObjectType
		if foodObjectID == uuid.Nil && entry.MealID != uuid.Nil {
			foodObjectID, foodObjectType = entry.MealID, repository.FoodObjectTypeMeal
		}
		entries = append(entries, map[string]any{
			"id": entry.ID.String(), "foodObjectId": foodObjectID.String(), "foodObjectType": foodObjectType, "quantity": entry.Quantity,
			"unit": entry.Unit, "position": entry.Position,
		})
	}
	return map[string]any{
		"id": diet.ID.String(), "name": diet.Name, "entries": entries,
		"aggregateMacros": map[string]any{
			"protein": diet.AggregateMacros.Protein, "carbohydrates": diet.AggregateMacros.Carbohydrates,
			"fat": diet.AggregateMacros.Fat, "calories": diet.AggregateMacros.Calories,
		},
		"createdAt": diet.CreatedAt, "updatedAt": diet.UpdatedAt,
	}
}

// unauthorizedError returns the shared protected-route response.
// Implements DESIGN-006 JWTManager and DESIGN-008 ProfileController.
func unauthorizedError() AppError {
	return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
}

// dailyDietDependencyError reports an unavailable saved-diet dependency.
// Implements DESIGN-008 ProfileController and DESIGN-017 GlobalExceptionHandler.
func dailyDietDependencyError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "daily_diet_unavailable", Message: "daily diet service is unavailable", Retryable: true}
}

// dailyDietError maps saved-diet failures to stable API errors.
// Implements DESIGN-008 ProfileController and DESIGN-017 GlobalExceptionHandler.
func dailyDietError(err error) error {
	switch {
	case errors.Is(err, dailydiet.ErrMissingIdempotencyKey):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	case errors.Is(err, dailydiet.ErrIdempotencyConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "idempotency_key_conflict", Message: "Idempotency-Key was already used with a different request body"}
	case errors.Is(err, dailydiet.ErrDuplicateName):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "duplicate_daily_diet_name", Message: "A Daily Diet with this name already exists"}
	case repository.IsKind(err, repository.ErrorKindNotFound):
		return AppError{HTTPStatus: fiber.StatusNotFound, Category: "validation", Code: "not_found", Message: "resource not found"}
	case repository.IsKind(err, repository.ErrorKindConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "conflict", Message: "resource conflicts with existing data"}
	default:
		return err
	}
}

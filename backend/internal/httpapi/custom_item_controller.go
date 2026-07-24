package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// CustomItemService defines authenticated custom-item behavior for ProfileController.
// Implements DESIGN-008 ProfileController custom-item routes.
type CustomItemService interface {
	Create(context.Context, uuid.UUID, customitem.CreateRequest) (customitem.CreateResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (customitem.Item, error)
	Update(context.Context, uuid.UUID, uuid.UUID, customitem.Request) (customitem.Item, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

// CreateCustomItem creates or replays an authenticated user's private item.
// Implements DESIGN-008 ProfileController custom-item creation.
func (c *ProfileController) CreateCustomItem(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	if c.customItems == nil {
		return customItemDependencyError()
	}
	req, err := customItemRequest(ctx)
	if err != nil {
		return err
	}
	result, err := c.customItems.Create(ctx.UserContext(), user.UserID, customitem.CreateRequest{
		Request: req, IdempotencyKey: ctx.Get("Idempotency-Key"),
	})
	if err != nil {
		return customItemError(err)
	}
	return ctx.Status(result.Status).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: customItemData(result.Item)})
}

// GetCustomItem returns one private item only to its authenticated owner.
// Implements DESIGN-008 ProfileController custom-item read.
func (c *ProfileController) GetCustomItem(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	itemID, err := parseCustomItemID(ctx.Params("itemId"))
	if err != nil {
		return err
	}
	if c.customItems == nil {
		return customItemDependencyError()
	}
	item, err := c.customItems.Get(ctx.UserContext(), user.UserID, itemID)
	if err != nil {
		return customItemError(err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: customItemData(item)})
}

// UpdateCustomItem replaces one private item only for its authenticated owner.
// Implements DESIGN-008 ProfileController custom-item update.
func (c *ProfileController) UpdateCustomItem(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	itemID, err := parseCustomItemID(ctx.Params("itemId"))
	if err != nil {
		return err
	}
	if c.customItems == nil {
		return customItemDependencyError()
	}
	req, err := customItemRequest(ctx)
	if err != nil {
		return err
	}
	item, err := c.customItems.Update(ctx.UserContext(), user.UserID, itemID, req)
	if err != nil {
		return customItemError(err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: customItemData(item)})
}

// DeleteCustomItem soft-deletes one private item only for its authenticated owner.
// Implements DESIGN-008 ProfileController custom-item delete.
func (c *ProfileController) DeleteCustomItem(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return unauthorizedError()
	}
	itemID, err := parseCustomItemID(ctx.Params("itemId"))
	if err != nil {
		return err
	}
	if c.customItems == nil {
		return customItemDependencyError()
	}
	if err := c.customItems.Delete(ctx.UserContext(), user.UserID, itemID); err != nil {
		return customItemError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// validateCustomItemCreate validates the idempotency header and strict JSON body.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateCustomItemCreate(ctx *fiber.Ctx) error {
	key := strings.TrimSpace(ctx.Get("Idempotency-Key"))
	if len(key) < 8 || len(key) > 255 || strings.ContainsRune(key, '\x00') {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	}
	return validateCustomItemBody(ctx)
}

// validateCustomItemUpdate validates the path identifier and strict JSON body.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateCustomItemUpdate(ctx *fiber.Ctx) error {
	if err := validateCustomItemID(ctx.Params("itemId")); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return validateCustomItemBody(ctx)
}

// validateCustomItemBody rejects malformed, unknown, and client-owned fields before dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateCustomItemBody(ctx *fiber.Ctx) error {
	req, err := decodeCustomItemRequest(ctx.Body())
	if err != nil {
		return err
	}
	ctx.Locals("customItemRequest", req)
	return ctx.Next()
}

// decodeCustomItemRequest enforces required, non-null, known, and domain-valid fields.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func decodeCustomItemRequest(body []byte) (customitem.Request, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil || raw == nil {
		return customitem.Request{}, invalidCustomItemBodyError()
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return customitem.Request{}, invalidCustomItemBodyError()
	}
	allowed := map[string]struct{}{
		"name": {}, "physicalState": {}, "prepTimeMinutes": {}, "averageUnitWeightGrams": {},
		"averageServingVolumeMilliliters": {}, "densityGramsPerMilliliter": {}, "densitySourceProvider": {},
		"densitySourceFoodId": {}, "densitySourceKind": {}, "macrosPer100": {}, "micros": {},
		"foodCategoryIds": {}, "culinaryRoleIds": {}, "imageUrl": {},
	}
	for field, value := range raw {
		if _, ok := allowed[field]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return customitem.Request{}, invalidCustomItemBodyError()
		}
	}
	for _, field := range []string{"name", "physicalState", "macrosPer100", "micros"} {
		if _, ok := raw[field]; !ok {
			return customitem.Request{}, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
		}
	}
	var macroFields map[string]json.RawMessage
	if err := json.Unmarshal(raw["macrosPer100"], &macroFields); err != nil || len(macroFields) != 3 {
		return customitem.Request{}, invalidCustomItemBodyError()
	}
	for _, field := range []string{"protein", "carbohydrates", "fat"} {
		value, ok := macroFields[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return customitem.Request{}, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
		}
	}
	var req customitem.Request
	strict := json.NewDecoder(bytes.NewReader(body))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&req); err != nil {
		return customitem.Request{}, invalidCustomItemBodyError()
	}
	for _, field := range []string{"averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter"} {
		if _, present := raw[field]; present {
			var value float64
			if err := json.Unmarshal(raw[field], &value); err != nil || value <= 0 {
				return customitem.Request{}, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
			}
		}
	}
	if hasDuplicateUUID(req.FoodCategoryIDs) || hasDuplicateUUID(req.CulinaryRoleIDs) {
		return customitem.Request{}, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	normalized, err := customitem.ValidateRequest(req)
	if err != nil {
		return customitem.Request{}, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
	}
	return normalized, nil
}

// hasDuplicateUUID reports duplicate classification IDs rejected by OpenAPI uniqueItems.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func hasDuplicateUUID(ids []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

// customItemRequest returns the request validated by middleware or validates direct calls.
// Implements DESIGN-008 ProfileController custom-item request parsing.
func customItemRequest(ctx *fiber.Ctx) (customitem.Request, error) {
	if req, ok := ctx.Locals("customItemRequest").(customitem.Request); ok {
		return req, nil
	}
	return decodeCustomItemRequest(ctx.Body())
}

// validateCustomItemID validates a private-item path identifier.
// Implements DESIGN-010 RequestValidator and DESIGN-008 ProfileController.
func validateCustomItemID(value string) error {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return errors.New("custom item id is invalid")
	}
	return nil
}

// parseCustomItemID parses a validated path identifier into a UUID.
// Implements DESIGN-008 ProfileController custom-item routing.
func parseCustomItemID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return id, nil
}

// customItemData maps an owner-free service projection into the shared envelope.
// Implements DESIGN-008 ProfileController custom-item response.
func customItemData(item customitem.Item) map[string]any {
	return map[string]any{
		"id": item.ID, "name": item.Name, "physicalState": item.PhysicalState, "prepTimeMinutes": item.PrepTimeMinutes,
		"averageUnitWeightGrams": item.AverageUnitWeightGrams, "averageServingVolumeMilliliters": item.AverageServingVolumeMilliliters,
		"densityGramsPerMilliliter": item.DensityGramsPerMilliliter, "densitySourceProvider": item.DensitySourceProvider,
		"densitySourceFoodId": item.DensitySourceFoodID, "densitySourceKind": item.DensitySourceKind, "macrosPer100": item.MacrosPer100,
		"micros": item.Micros, "foodCategories": item.FoodCategories, "culinaryRoles": item.CulinaryRoles, "imageUrl": item.ImageURL,
	}
}

// invalidCustomItemBodyError returns the stable malformed-body response.
// Implements DESIGN-008 ProfileController and DESIGN-017 GlobalExceptionHandler.
func invalidCustomItemBodyError() AppError {
	return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
}

// customItemDependencyError reports an unavailable custom-item service.
// Implements DESIGN-008 ProfileController and DESIGN-017 GlobalExceptionHandler.
func customItemDependencyError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "custom_item_unavailable", Message: "custom item service is unavailable", Retryable: true}
}

// customItemError maps service/repository failures to user-safe API errors.
// Implements DESIGN-008 ProfileController and DESIGN-017 GlobalExceptionHandler.
func customItemError(err error) error {
	switch {
	case errors.Is(err, customitem.ErrMissingIdempotencyKey):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	case errors.Is(err, customitem.ErrIdempotencyConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "idempotency_key_conflict", Message: "Idempotency-Key was already used with a different request body"}
	case repository.IsKind(err, repository.ErrorKindNotFound):
		return AppError{HTTPStatus: fiber.StatusNotFound, Category: "validation", Code: "not_found", Message: "resource not found"}
	case repository.IsKind(err, repository.ErrorKindConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "conflict", Message: "resource conflicts with existing data"}
	case repository.IsKind(err, repository.ErrorKindValidation), repository.IsKind(err, repository.ErrorKindInvalidMicronutrientKey):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	default:
		return err
	}
}

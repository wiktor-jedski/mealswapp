package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/itemcurator"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// ManualItemService defines administrator-authored global food-item behavior.
// Implements DESIGN-009 ItemCurator admin routes.
type ManualItemService interface {
	Create(context.Context, repository.AdminMutationExecutor, uuid.UUID, string, itemcurator.Request) (itemcurator.CreateResult, error)
	Get(context.Context, uuid.UUID) (itemcurator.Item, error)
	Update(context.Context, repository.AdminMutationExecutor, uuid.UUID, itemcurator.Request) (itemcurator.MutationResult, error)
	Delete(context.Context, repository.AdminMutationExecutor, uuid.UUID) (itemcurator.MutationResult, error)
}

// ManualItemController handles the ItemCurator-specific admin route actions.
// Implements DESIGN-009 ItemCurator.
type ManualItemController struct {
	service ManualItemService
}

// NewManualItemAdminController composes manual item routes with the secure admin gateway.
// Implements DESIGN-009 AdminController and ItemCurator.
func NewManualItemAdminController(audit repository.AdminMutationAuditRepository, service ManualItemService) *AdminController {
	items := &ManualItemController{service: service}
	readLimit := RateLimitRule{Scope: "user", MaxRequests: 120, WindowSeconds: 60}
	mutationLimit := RateLimitRule{Scope: "user", MaxRequests: 30, WindowSeconds: 60}
	return NewAdminController(audit,
		AdminRouteDefinition{Method: fiber.MethodPost, Path: "/items", Mutation: items.Create, Validate: validateManualItemCreate, RateLimit: &mutationLimit, AuditAction: "manual_create", EntityType: "food_item"},
		AdminRouteDefinition{Method: fiber.MethodGet, Path: "/items/:itemId", Handler: items.Get, Validate: ValidatePath("itemId", validateManualItemID), RateLimit: &readLimit},
		AdminRouteDefinition{Method: fiber.MethodPut, Path: "/items/:itemId", Mutation: items.Update, Validate: validateManualItemUpdate, RateLimit: &mutationLimit, AuditAction: "manual_update", EntityType: "food_item"},
		AdminRouteDefinition{Method: fiber.MethodDelete, Path: "/items/:itemId", Mutation: items.Delete, Validate: ValidatePath("itemId", validateManualItemID), RateLimit: &mutationLimit, AuditAction: "manual_delete", EntityType: "food_item"},
	)
}

// Create creates or replays one global food item.
// Implements DESIGN-009 ItemCurator idempotent create.
func (c *ManualItemController) Create(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	admin, err := RequireAdmin(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	if c == nil || c.service == nil {
		return AdminMutationResult{}, manualItemDependencyError()
	}
	req, err := manualItemRequest(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	result, err := c.service.Create(ctx.UserContext(), tx, admin.UserID, ctx.Get("Idempotency-Key"), req)
	if err != nil {
		return AdminMutationResult{}, manualItemError(err)
	}
	id := result.Item.ID
	return AdminMutationResult{
		HTTPStatus: result.Status,
		Data:       manualItemData(result.Item),
		Audit: func() repository.AdminAuditChanges {
			if result.Replayed {
				return repository.AdminAuditChanges{Replayed: true}
			}
			return repository.AdminAuditChanges{EntityID: &id, After: manualItemAuditSnapshot(result.Item, true, false)}
		}(),
	}, nil
}

// Get returns one active global food item.
// Implements DESIGN-009 ItemCurator read behavior.
func (c *ManualItemController) Get(ctx *fiber.Ctx) error {
	id, err := parseManualItemID(ctx.Params("itemId"))
	if err != nil {
		return err
	}
	if c == nil || c.service == nil {
		return manualItemDependencyError()
	}
	item, err := c.service.Get(ctx.UserContext(), id)
	if err != nil {
		return manualItemError(err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: manualItemData(item)})
}

// Update replaces one active global food item.
// Implements DESIGN-009 ItemCurator update behavior.
func (c *ManualItemController) Update(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	id, err := parseManualItemID(ctx.Params("itemId"))
	if err != nil {
		return AdminMutationResult{}, err
	}
	if c == nil || c.service == nil {
		return AdminMutationResult{}, manualItemDependencyError()
	}
	req, err := manualItemRequest(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	result, err := c.service.Update(ctx.UserContext(), tx, id, req)
	if err != nil {
		return AdminMutationResult{}, manualItemError(err)
	}
	return AdminMutationResult{Data: manualItemData(result.After), Audit: repository.AdminAuditChanges{
		EntityID: &id, Before: manualItemAuditSnapshot(result.Before, true, false), After: manualItemAuditSnapshot(result.After, true, false),
	}}, nil
}

// Delete soft-deletes one active global food item.
// Implements DESIGN-009 ItemCurator soft-delete behavior.
func (c *ManualItemController) Delete(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	id, err := parseManualItemID(ctx.Params("itemId"))
	if err != nil {
		return AdminMutationResult{}, err
	}
	if c == nil || c.service == nil {
		return AdminMutationResult{}, manualItemDependencyError()
	}
	result, err := c.service.Delete(ctx.UserContext(), tx, id)
	if err != nil {
		return AdminMutationResult{}, manualItemError(err)
	}
	return AdminMutationResult{HTTPStatus: fiber.StatusNoContent, Audit: repository.AdminAuditChanges{
		EntityID: &id, Before: manualItemAuditSnapshot(result.Before, true, false), After: manualItemAuditSnapshot(result.Before, false, true),
	}}, nil
}

// validateManualItemCreate enforces a durable key and strict global-item body.
// Implements DESIGN-009 ItemCurator and DESIGN-010 RequestValidator.
func validateManualItemCreate(ctx *fiber.Ctx) error {
	key := strings.TrimSpace(ctx.Get("Idempotency-Key"))
	if len(key) < 8 || len(key) > 255 || strings.ContainsRune(key, '\x00') {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	}
	return validateManualItemBody(ctx)
}

// validateManualItemUpdate validates the item identity and strict replacement body.
// Implements DESIGN-009 ItemCurator and DESIGN-010 RequestValidator.
func validateManualItemUpdate(ctx *fiber.Ctx) error {
	if err := validateManualItemID(ctx.Params("itemId")); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return validateManualItemBody(ctx)
}

// validateManualItemBody reuses the complete food-field schema while rejecting ownership input.
// Implements DESIGN-009 ItemCurator global/private separation.
func validateManualItemBody(ctx *fiber.Ctx) error {
	if err := rejectDuplicateJSONKeys(ctx.Body()); err != nil {
		return invalidCustomItemBodyError()
	}
	req, err := decodeCustomItemRequest(ctx.Body())
	if err != nil {
		return err
	}
	ctx.Locals("manualItemRequest", itemcurator.Request(req))
	return ctx.Next()
}

// manualItemRequest returns the request approved by validation middleware.
// Implements DESIGN-009 ItemCurator typed handoff.
func manualItemRequest(ctx *fiber.Ctx) (itemcurator.Request, error) {
	if req, ok := ctx.Locals("manualItemRequest").(itemcurator.Request); ok {
		return req, nil
	}
	req, err := decodeCustomItemRequest(ctx.Body())
	return itemcurator.Request(req), err
}

// validateManualItemID validates a global-item path identifier.
// Implements DESIGN-009 ItemCurator request boundary.
func validateManualItemID(value string) error {
	_, err := parseManualItemID(value)
	return err
}

// parseManualItemID parses a global-item path identifier into a UUID.
// Implements DESIGN-009 ItemCurator request boundary.
func parseManualItemID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	return id, nil
}

// manualItemData maps the owner-free item projection into the shared envelope.
// Implements DESIGN-009 ItemCurator response boundary.
func manualItemData(item itemcurator.Item) map[string]any {
	return map[string]any{
		"id": item.ID, "name": item.Name, "physicalState": item.PhysicalState, "prepTimeMinutes": item.PrepTimeMinutes,
		"averageUnitWeightGrams": item.AverageUnitWeightGrams, "averageServingVolumeMilliliters": item.AverageServingVolumeMilliliters,
		"densityGramsPerMilliliter": item.DensityGramsPerMilliliter, "densitySourceProvider": item.DensitySourceProvider,
		"densitySourceFoodId": item.DensitySourceFoodID, "densitySourceKind": item.DensitySourceKind, "macrosPer100": item.MacrosPer100,
		"micros": item.Micros, "foodCategories": item.FoodCategories, "culinaryRoles": item.CulinaryRoles, "imageUrl": item.ImageURL,
	}
}

// manualItemAuditSnapshot emits only bounded enum/boolean curation state.
// Implements DESIGN-009 ItemCurator privacy-safe before/after snapshots.
func manualItemAuditSnapshot(item itemcurator.Item, active bool, deleted bool) []byte {
	fields := map[string]any{"active": active, "physicalState": item.PhysicalState}
	if deleted {
		fields["deleted"] = true
	}
	payload, _ := json.Marshal(fields)
	return payload
}

// manualItemDependencyError reports an unavailable curation service.
// Implements DESIGN-009 ItemCurator fail-closed behavior.
func manualItemDependencyError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "manual_item_unavailable", Message: "manual item service is unavailable", Retryable: true}
}

// manualItemError maps curation failures to user-safe API errors.
// Implements DESIGN-009 ItemCurator and DESIGN-017 GlobalExceptionHandler.
func manualItemError(err error) error {
	switch {
	case errors.Is(err, itemcurator.ErrMissingIdempotencyKey):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	case errors.Is(err, itemcurator.ErrIdempotencyConflict):
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

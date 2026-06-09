package httpapi

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

// UserDataService defines saved data and history behavior for HTTP handlers.
// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository.
type UserDataService interface {
	ListSaved(context.Context, uuid.UUID, *repository.SavedItemKind) ([]repository.SavedItem, error)
	DeleteSaved(context.Context, uuid.UUID, uuid.UUID, repository.SavedItemKind) error
	ListHistory(context.Context, uuid.UUID, int) ([]userdata.SearchHistoryEntry, error)
	ClearHistory(context.Context, uuid.UUID) error
}

// UserDataController owns saved-data and history routes.
// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository.
type UserDataController struct {
	service UserDataService
}

// NewUserDataController creates authenticated user-data handlers.
// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository.
func NewUserDataController(service UserDataService) *UserDataController {
	return &UserDataController{service: service}
}

// Routes returns authenticated saved-data and history routes.
// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository.
func (c *UserDataController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/saved-items", RequiresAuth: true, Validate: ValidateQuery(validateSavedItemsQuery), Handler: c.ListSaved},
		{Method: fiber.MethodDelete, Path: "/saved-items/:kind/:itemId", RequiresAuth: true, RequiresCSRF: true, Validate: validateDeleteSavedPath(), Handler: c.DeleteSaved},
		{Method: fiber.MethodGet, Path: "/search-history", RequiresAuth: true, Handler: c.ListHistory},
		{Method: fiber.MethodDelete, Path: "/search-history", RequiresAuth: true, RequiresCSRF: true, Handler: c.ClearHistory},
	}
}

// ListSaved returns saved items for the authenticated user.
// Implements DESIGN-008 SavedDataRepository.
func (c *UserDataController) ListSaved(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	kind, err := savedKindQuery(ctx.Query("kind"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	items, err := c.service.ListSaved(ctx.UserContext(), user.UserID, kind)
	if err != nil {
		return err
	}
	data := []map[string]any{}
	for _, item := range items {
		data = append(data, map[string]any{"id": item.ID.String(), "itemId": item.ItemID.String(), "kind": string(item.Kind)})
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"items": data}})
}

// DeleteSaved removes one saved item for the authenticated user.
// Implements DESIGN-008 SavedDataRepository.
func (c *UserDataController) DeleteSaved(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	itemID, err := uuid.Parse(ctx.Params("itemId"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	}
	kind := repository.SavedItemKind(ctx.Params("kind"))
	if err := c.service.DeleteSaved(ctx.UserContext(), user.UserID, itemID, kind); err != nil {
		return err
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// ListHistory returns latest search history for the authenticated user.
// Implements DESIGN-008 SearchHistoryRepository.
func (c *UserDataController) ListHistory(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	entries, err := c.service.ListHistory(ctx.UserContext(), user.UserID, 100)
	if err != nil {
		return err
	}
	data := []map[string]any{}
	for _, entry := range entries {
		data = append(data, map[string]any{"id": entry.ID.String(), "query": entry.Query, "mode": entry.Mode, "filtersHash": entry.FiltersHash})
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"history": data}})
}

// ClearHistory clears search history for the authenticated user.
// Implements DESIGN-008 SearchHistoryRepository.
func (c *UserDataController) ClearHistory(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	if err := c.service.ClearHistory(ctx.UserContext(), user.UserID); err != nil {
		return err
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// savedKindQuery validates optional saved-item filtering.
// Implements DESIGN-008 SavedDataRepository.
func savedKindQuery(value string) (*repository.SavedItemKind, error) {
	if value == "" {
		return nil, nil
	}
	kind := repository.SavedItemKind(value)
	if kind != repository.SavedItemKindFavorite && kind != repository.SavedItemKindSavedMeal && kind != repository.SavedItemKindSavedDiet {
		return nil, repository.NewError(repository.ErrorKindValidation, "saved item kind is invalid", nil)
	}
	return &kind, nil
}

// validateSavedItemsQuery validates saved-item filter query parameters.
// Implements DESIGN-010 RequestValidator.
func validateSavedItemsQuery(values map[string]string) error {
	if _, err := savedKindQuery(values["kind"]); err != nil {
		return err
	}
	return nil
}

// validateDeleteSavedPath validates saved-item delete route parameters before dispatch.
// Implements DESIGN-010 RequestValidator.
func validateDeleteSavedPath() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if _, err := savedKindQuery(ctx.Params("kind")); err != nil {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
		}
		if _, err := uuid.Parse(ctx.Params("itemId")); err != nil {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: errors.New("item id is invalid")}
		}
		return ctx.Next()
	}
}

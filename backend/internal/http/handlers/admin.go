package handlers

import (
	"context"
	"errors"
	"time"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/tag"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"
	"mealswapp/backend/internal/repositories"
	"mealswapp/backend/internal/services/externaldata"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AdminSummaryService interface {
	Summary(ctx context.Context, admin AdminContext) (AdminSummary, error)
}

type ExternalSearchService interface {
	SearchExternalFoods(ctx context.Context, query externaldata.ExternalSearchQuery) (externaldata.ExternalSearchResult, error)
}

type ItemCuratorService interface {
	List(ctx context.Context, query string, page int, limit int) (externaldata.ItemListResult, error)
	Get(ctx context.Context, id uuid.UUID) (food.FoodItemEntity, error)
	Create(ctx context.Context, item food.FoodItemEntity) (food.FoodItemEntity, error)
	Update(ctx context.Context, id uuid.UUID, item food.FoodItemEntity) (food.FoodItemEntity, error)
	Transition(ctx context.Context, id uuid.UUID, transition externaldata.CurationTransition) (food.FoodItemEntity, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type TagManagerService interface {
	List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error)
	Upsert(ctx context.Context, entity tag.TagEntity) (tag.TagEntity, error)
	Assign(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error
	Remove(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) error
	Merge(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID) error
}

type UserAdminService interface {
	List(ctx context.Context, query string, page int, limit int) (externaldata.UserAdminListResult, error)
	Detail(ctx context.Context, userID uuid.UUID) (externaldata.UserAdminDetail, error)
	Disable(ctx context.Context, userID uuid.UUID) (repositories.UserEntity, error)
	ResetLockout(ctx context.Context, userID uuid.UUID) error
	AuditHistory(ctx context.Context, userID uuid.UUID, page int, limit int) (externaldata.UserAuditHistory, error)
}

type AdminContext struct {
	UserID    uuid.UUID `json:"userId"`
	Role      string    `json:"role"`
	RequestID string    `json:"requestId"`
}

type AdminSummary struct {
	PendingImports   int       `json:"pendingImports"`
	PendingItems     int       `json:"pendingItems"`
	ActiveUsers      int       `json:"activeUsers"`
	RecentAuditCount int       `json:"recentAuditCount"`
	GeneratedAt      time.Time `json:"generatedAt"`
}

type AdminHandler struct {
	auth     AuthService
	summary  AdminSummaryService
	external ExternalSearchService
	items    ItemCuratorService
	tags     TagManagerService
	users    UserAdminService
	now      func() time.Time
}

func NewAdminHandler(auth AuthService, summary AdminSummaryService, external ExternalSearchService, items ItemCuratorService, tags TagManagerService, users ...UserAdminService) AdminHandler {
	handler := AdminHandler{auth: auth, summary: summary, external: external, now: time.Now}
	handler.items = items
	handler.tags = tags
	if len(users) > 0 {
		handler.users = users[0]
	}
	return handler
}

func (handler AdminHandler) RequireAdminMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		admin, err := handler.RequireAdmin(ctx)
		if err != nil {
			return err
		}
		ctx.Locals("adminContext", admin)
		return ctx.Next()
	}
}

func (handler AdminHandler) Summary(ctx *fiber.Ctx) error {
	admin, ok := ctx.Locals("adminContext").(AdminContext)
	if !ok {
		var err error
		admin, err = handler.RequireAdmin(ctx)
		if err != nil {
			return err
		}
	}
	if handler.summary != nil {
		result, err := handler.summary.Summary(ctx.Context(), admin)
		if err != nil {
			return err
		}
		return ctx.JSON(responses.Success(result, requestID(ctx)))
	}
	return ctx.JSON(responses.Success(AdminSummary{GeneratedAt: handler.now().UTC()}, requestID(ctx)))
}

func (handler AdminHandler) SearchExternal(ctx *fiber.Ctx) error {
	if handler.external == nil {
		return apperrors.DependencyUnavailable("External search is unavailable")
	}
	provider := externaldata.Provider(ctx.Query("provider", string(externaldata.ProviderAll)))
	page, err := validation.QueryInt(ctx, "page", 1, 1, 0)
	if err != nil {
		return err
	}
	pageSize, err := validation.QueryInt(ctx, "pageSize", 10, 1, 50)
	if err != nil {
		return err
	}
	query := externaldata.ExternalSearchQuery{
		Query:    ctx.Query("query"),
		Provider: provider,
		Page:     page,
		PageSize: pageSize,
	}
	result, err := handler.external.SearchExternalFoods(ctx.Context(), query)
	if err != nil {
		return externalSearchError(err)
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler AdminHandler) ListItems(ctx *fiber.Ctx) error {
	if handler.items == nil {
		return apperrors.DependencyUnavailable("Item curator is unavailable")
	}
	page, err := validation.QueryInt(ctx, "page", 1, 1, 0)
	if err != nil {
		return err
	}
	limit, err := validation.QueryInt(ctx, "pageSize", 10, 1, 50)
	if err != nil {
		return err
	}
	result, err := handler.items.List(ctx.Context(), ctx.Query("query"), page, limit)
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler AdminHandler) GetItem(ctx *fiber.Ctx) error {
	if handler.items == nil {
		return apperrors.DependencyUnavailable("Item curator is unavailable")
	}
	id, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	item, err := handler.items.Get(ctx.Context(), id)
	if err != nil {
		return err
	}
	return ctx.JSON(responses.Success(item, requestID(ctx)))
}

func (handler AdminHandler) CreateItem(ctx *fiber.Ctx) error {
	if handler.items == nil {
		return apperrors.DependencyUnavailable("Item curator is unavailable")
	}
	item, err := validation.DecodeJSON[food.FoodItemEntity](ctx)
	if err != nil {
		return err
	}
	created, err := handler.items.Create(ctx.Context(), item)
	if err != nil {
		return itemCuratorError(err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(responses.Success(created, requestID(ctx)))
}

func (handler AdminHandler) UpdateItem(ctx *fiber.Ctx) error {
	if handler.items == nil {
		return apperrors.DependencyUnavailable("Item curator is unavailable")
	}
	id, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	item, err := validation.DecodeJSON[food.FoodItemEntity](ctx)
	if err != nil {
		return err
	}
	updated, err := handler.items.Update(ctx.Context(), id, item)
	if err != nil {
		return itemCuratorError(err)
	}
	return ctx.JSON(responses.Success(updated, requestID(ctx)))
}

func (handler AdminHandler) TransitionItem(ctx *fiber.Ctx) error {
	if handler.items == nil {
		return apperrors.DependencyUnavailable("Item curator is unavailable")
	}
	id, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	updated, err := handler.items.Transition(ctx.Context(), id, externaldata.CurationTransition(ctx.Params("transition")))
	if err != nil {
		if errors.Is(err, externaldata.ErrItemCuratorInvalidTransition) {
			return apperrors.Validation("Invalid curation transition", nil)
		}
		return err
	}
	return ctx.JSON(responses.Success(updated, requestID(ctx)))
}

func (handler AdminHandler) DeleteItem(ctx *fiber.Ctx) error {
	if handler.items == nil {
		return apperrors.DependencyUnavailable("Item curator is unavailable")
	}
	id, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	if err := handler.items.Delete(ctx.Context(), id); err != nil {
		return err
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler AdminHandler) ListTags(ctx *fiber.Ctx) error {
	if handler.tags == nil {
		return apperrors.DependencyUnavailable("Tag manager is unavailable")
	}
	tags, err := handler.tags.List(ctx.Context(), tag.Kind(ctx.Query("kind")))
	if err != nil {
		return tagManagerError(err)
	}
	return ctx.JSON(responses.Success(tags, requestID(ctx)))
}

func (handler AdminHandler) UpsertTag(ctx *fiber.Ctx) error {
	if handler.tags == nil {
		return apperrors.DependencyUnavailable("Tag manager is unavailable")
	}
	entity, err := validation.DecodeJSON[tag.TagEntity](ctx)
	if err != nil {
		return err
	}
	updated, err := handler.tags.Upsert(ctx.Context(), entity)
	if err != nil {
		return tagManagerError(err)
	}
	return ctx.JSON(responses.Success(updated, requestID(ctx)))
}

func (handler AdminHandler) AssignTag(ctx *fiber.Ctx) error {
	if handler.tags == nil {
		return apperrors.DependencyUnavailable("Tag manager is unavailable")
	}
	foodItemID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	payload, err := validation.DecodeJSON[externaldata.TagAssignment](ctx)
	if err != nil {
		return err
	}
	if err := handler.tags.Assign(ctx.Context(), foodItemID, payload.TagID); err != nil {
		return tagManagerError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler AdminHandler) RemoveTag(ctx *fiber.Ctx) error {
	if handler.tags == nil {
		return apperrors.DependencyUnavailable("Tag manager is unavailable")
	}
	foodItemID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	tagID, err := validation.UUIDParam(ctx, "tagId")
	if err != nil {
		return err
	}
	if err := handler.tags.Remove(ctx.Context(), foodItemID, tagID); err != nil {
		return tagManagerError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler AdminHandler) DeactivateTag(ctx *fiber.Ctx) error {
	if handler.tags == nil {
		return apperrors.DependencyUnavailable("Tag manager is unavailable")
	}
	id, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	if err := handler.tags.Deactivate(ctx.Context(), id); err != nil {
		return tagManagerError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler AdminHandler) MergeTags(ctx *fiber.Ctx) error {
	if handler.tags == nil {
		return apperrors.DependencyUnavailable("Tag manager is unavailable")
	}
	payload, err := validation.DecodeJSON[externaldata.TagMergeRequest](ctx)
	if err != nil {
		return err
	}
	if err := handler.tags.Merge(ctx.Context(), payload.SourceID, payload.TargetID); err != nil {
		return tagManagerError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler AdminHandler) ListUsers(ctx *fiber.Ctx) error {
	if handler.users == nil {
		return apperrors.DependencyUnavailable("User admin is unavailable")
	}
	page, err := validation.QueryInt(ctx, "page", 1, 1, 0)
	if err != nil {
		return err
	}
	limit, err := validation.QueryInt(ctx, "pageSize", 10, 1, 50)
	if err != nil {
		return err
	}
	result, err := handler.users.List(ctx.Context(), ctx.Query("query"), page, limit)
	if err != nil {
		return userAdminError(err)
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler AdminHandler) GetUser(ctx *fiber.Ctx) error {
	if handler.users == nil {
		return apperrors.DependencyUnavailable("User admin is unavailable")
	}
	userID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	result, err := handler.users.Detail(ctx.Context(), userID)
	if err != nil {
		return userAdminError(err)
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler AdminHandler) DisableUser(ctx *fiber.Ctx) error {
	if handler.users == nil {
		return apperrors.DependencyUnavailable("User admin is unavailable")
	}
	userID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	user, err := handler.users.Disable(ctx.Context(), userID)
	if err != nil {
		return userAdminError(err)
	}
	return ctx.JSON(responses.Success(user, requestID(ctx)))
}

func (handler AdminHandler) ResetUserLockout(ctx *fiber.Ctx) error {
	if handler.users == nil {
		return apperrors.DependencyUnavailable("User admin is unavailable")
	}
	userID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	if err := handler.users.ResetLockout(ctx.Context(), userID); err != nil {
		return userAdminError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (handler AdminHandler) UserAuditHistory(ctx *fiber.Ctx) error {
	if handler.users == nil {
		return apperrors.DependencyUnavailable("User admin is unavailable")
	}
	userID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	page, err := validation.QueryInt(ctx, "page", 1, 1, 0)
	if err != nil {
		return err
	}
	limit, err := validation.QueryInt(ctx, "pageSize", 10, 1, 50)
	if err != nil {
		return err
	}
	result, err := handler.users.AuditHistory(ctx.Context(), userID, page, limit)
	if err != nil {
		return userAdminError(err)
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

func (handler AdminHandler) RequireAdmin(ctx *fiber.Ctx) (AdminContext, error) {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return AdminContext{}, err
	}
	if handler.auth == nil {
		return AdminContext{}, apperrors.Unauthorized("Unauthorized")
	}
	user, err := handler.auth.CurrentUser(ctx.Context(), CurrentUserCommand{AccessToken: token})
	if err != nil {
		return AdminContext{}, err
	}
	if user.Role != "admin" {
		return AdminContext{}, apperrors.Forbidden("Forbidden")
	}
	return AdminContext{UserID: user.ID, Role: user.Role, RequestID: requestID(ctx)}, nil
}

func externalSearchError(err error) error {
	var providerErr externaldata.ProviderError
	if !errors.As(err, &providerErr) {
		return err
	}
	switch providerErr.Kind {
	case externaldata.ProviderErrorInvalidQuery, externaldata.ProviderErrorBadPayload:
		return apperrors.Validation(providerErr.Message, nil)
	case externaldata.ProviderErrorRateLimited:
		return apperrors.RateLimited(providerErr.Message)
	case externaldata.ProviderErrorTimeout:
		return apperrors.Timeout(providerErr.Message)
	case externaldata.ProviderErrorUnavailable:
		return apperrors.DependencyUnavailable(providerErr.Message)
	default:
		return apperrors.DependencyUnavailable(providerErr.Message)
	}
}

func itemCuratorError(err error) error {
	if errors.Is(err, externaldata.ErrImportDraftInvalid) || errors.Is(err, externaldata.ErrImportConflict) || errors.Is(err, externaldata.ErrItemCuratorInvalidTransition) ||
		errors.Is(err, food.ErrMissingName) || errors.Is(err, food.ErrInvalidPhysicalState) || errors.Is(err, food.ErrUnsupportedServingUnit) ||
		errors.Is(err, food.ErrInvalidServingSize) || errors.Is(err, food.ErrInvalidCalories) || errors.Is(err, food.ErrInvalidMacros) ||
		errors.Is(err, food.ErrInvalidMicronutrients) || errors.Is(err, food.ErrInvalidPrepTime) || errors.Is(err, food.ErrInvalidUnitWeight) {
		return apperrors.Validation(err.Error(), nil)
	}
	return err
}

func tagManagerError(err error) error {
	if errors.Is(err, tag.ErrMissingName) || errors.Is(err, tag.ErrInvalidKind) || errors.Is(err, externaldata.ErrTagMergeInvalid) {
		return apperrors.Validation(err.Error(), nil)
	}
	return err
}

func userAdminError(err error) error {
	if errors.Is(err, externaldata.ErrUserAdminInvalidUser) {
		return apperrors.Validation(err.Error(), nil)
	}
	return err
}

package httpapi

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// AccountDeletionService defines account deletion request behavior for HTTP handlers.
// Implements DESIGN-008 AccountDeleter.
type AccountDeletionService interface {
	RequestDeletion(context.Context, uuid.UUID) (repository.DataDeletionRequest, error)
}

// AccountDeletionController owns authenticated account deletion routes.
// Implements DESIGN-008 AccountDeleter.
type AccountDeletionController struct {
	service  AccountDeletionService
	sessions *AuthSessionManager
}

// Implements DESIGN-008 AccountDeleter compile-time route controller contract.
var _ Controller = (*AccountDeletionController)(nil)

// NewAccountDeletionController creates account deletion handlers.
// Implements DESIGN-008 AccountDeleter.
func NewAccountDeletionController(service AccountDeletionService, sessions *AuthSessionManager) *AccountDeletionController {
	return &AccountDeletionController{service: service, sessions: sessions}
}

// Routes returns authenticated deletion routes.
// Implements DESIGN-008 AccountDeleter.
func (c *AccountDeletionController) Routes() []RouteDefinition {
	return []RouteDefinition{{Method: fiber.MethodDelete, Path: "/account", RequiresAuth: true, RequiresCSRF: true, RequiresAudit: true, Handler: c.DeleteAccount}}
}

// DeleteAccount requests deletion and clears authorization state.
// Implements DESIGN-008 AccountDeleter.
func (c *AccountDeletionController) DeleteAccount(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	request, err := c.service.RequestDeletion(ctx.UserContext(), user.UserID)
	if err != nil {
		return err
	}
	if err := c.sessions.ClearAuthenticatedCookies(ctx); err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"requestId": request.ID.String(), "status": request.Status}})
}

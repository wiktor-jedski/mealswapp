package handlers

import (
	"context"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AdminSummaryService interface {
	Summary(ctx context.Context, admin AdminContext) (AdminSummary, error)
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
	auth    AuthService
	summary AdminSummaryService
	now     func() time.Time
}

func NewAdminHandler(auth AuthService, summary AdminSummaryService) AdminHandler {
	return AdminHandler{auth: auth, summary: summary, now: time.Now}
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

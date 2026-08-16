package httpapi

import (
	"bytes"
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/useradmin"
)

// UserAdminService is the restricted service boundary exposed to HTTP.
// Implements DESIGN-009 UserAdminPanel.
type UserAdminService interface {
	Lookup(context.Context, useradmin.Actor, useradmin.LookupRequest) (useradmin.Page, error)
	RetryDeletion(context.Context, useradmin.Actor, uuid.UUID, uuid.UUID, repository.AdminMutationExecutor) (useradmin.RetryResult, error)
}

// UserAdminController defines documented routes below the shared admin gateway.
// Implements DESIGN-009 UserAdminPanel.
type UserAdminController struct {
	service UserAdminService
}

// NewUserAdminController creates restricted user-administration route definitions.
// Implements DESIGN-009 UserAdminPanel.
func NewUserAdminController(service UserAdminService) *UserAdminController {
	return &UserAdminController{service: service}
}

// AdminRoutes returns only lookup and deletion-retry routes.
// Implements DESIGN-009 UserAdminPanel.
func (c *UserAdminController) AdminRoutes() []AdminRouteDefinition {
	lookupRate := &RateLimitRule{Scope: "user", MaxRequests: 30, WindowSeconds: 60}
	retryRate := &RateLimitRule{Scope: "user", MaxRequests: 5, WindowSeconds: 60}
	return []AdminRouteDefinition{
		{Method: fiber.MethodGet, Path: "/users", Handler: c.Lookup, Validate: validateAdminUserLookup, RateLimit: lookupRate},
		{Method: fiber.MethodPost, Path: "/users/:userId/deletion-requests/:requestId/retry", Mutation: c.RetryDeletion, Validate: validateAdminDeletionRetry, RateLimit: retryRate, AuditAction: "retry_deletion", EntityType: "deletion_request"},
	}
}

// Lookup returns one exact match or one bounded page after a durable privacy-safe audit.
// Implements DESIGN-009 UserAdminPanel.
func (c *UserAdminController) Lookup(ctx *fiber.Ctx) error {
	admin, err := RequireAdmin(ctx)
	if err != nil {
		return err
	}
	if c.service == nil {
		return adminUserDependencyError()
	}
	request, ok := ctx.Locals("adminUserLookup").(useradmin.LookupRequest)
	if !ok {
		return adminUserValidationError()
	}
	page, err := c.service.Lookup(ctx.UserContext(), useradmin.Actor{UserID: admin.UserID, Role: admin.Role, RequestID: admin.RequestID}, request)
	if err != nil {
		return adminUserError(err)
	}
	data := map[string]any{"users": page.Users}
	if page.NextCursor != nil {
		data["nextCursor"] = page.NextCursor
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: admin.RequestID, Data: data})
}

// RetryDeletion queues one eligible scoped deletion failure and returns audit metadata.
// Implements DESIGN-009 UserAdminPanel.
func (c *UserAdminController) RetryDeletion(ctx *fiber.Ctx, tx repository.AdminMutationExecutor) (AdminMutationResult, error) {
	admin, err := RequireAdmin(ctx)
	if err != nil {
		return AdminMutationResult{}, err
	}
	if c.service == nil {
		return AdminMutationResult{}, adminUserDependencyError()
	}
	userID, userErr := uuid.Parse(ctx.Params("userId"))
	requestID, requestErr := uuid.Parse(ctx.Params("requestId"))
	if userErr != nil || requestErr != nil {
		return AdminMutationResult{}, adminUserValidationError()
	}
	result, err := c.service.RetryDeletion(ctx.UserContext(), useradmin.Actor{UserID: admin.UserID, Role: admin.Role, RequestID: admin.RequestID}, userID, requestID, tx)
	if err != nil {
		return AdminMutationResult{}, adminUserError(err)
	}
	if result.FailureCategory != "permanent" && result.FailureCategory != "unknown" && result.FailureCategory != "transient" {
		return AdminMutationResult{}, errors.New("invalid deletion retry category")
	}
	before := []byte(`{"status":"failed","failureCategory":"` + result.FailureCategory + `"}`)
	requestEntityID := result.RequestID
	return AdminMutationResult{
		Data:  map[string]any{"requestId": result.RequestID, "status": "pending"},
		Audit: repository.AdminAuditChanges{EntityID: &requestEntityID, Before: before, After: []byte(`{"status":"pending"}`)},
	}, nil
}

// validateAdminUserLookup rejects unknown, duplicate, conflicting, and unbounded query input.
// Implements DESIGN-009 UserAdminPanel.
func validateAdminUserLookup(ctx *fiber.Ctx) error {
	allowed := map[string]bool{"userId": true, "email": true, "cursor": true, "limit": true}
	seen := map[string]bool{}
	valid := true
	ctx.Context().QueryArgs().VisitAll(func(key []byte, _ []byte) {
		name := string(key)
		if !allowed[name] || seen[name] {
			valid = false
		}
		seen[name] = true
	})
	if !valid {
		return adminUserValidationError()
	}
	request := useradmin.LookupRequest{Email: ctx.Query("email")}
	if value := ctx.Query("userId"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return adminUserValidationError()
		}
		request.UserID = &id
	}
	if value := ctx.Query("cursor"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return adminUserValidationError()
		}
		request.Cursor = &id
	}
	if value := ctx.Query("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > useradmin.MaxPageSize {
			return adminUserValidationError()
		}
		request.Limit = limit
	}
	if request.UserID != nil && request.Email != "" || request.Cursor != nil && (request.UserID != nil || request.Email != "") {
		return adminUserValidationError()
	}
	ctx.Locals("adminUserLookup", request)
	return ctx.Next()
}

// validateAdminDeletionRetry accepts UUID scope and no client-controlled mutation fields.
// Implements DESIGN-009 UserAdminPanel.
func validateAdminDeletionRetry(ctx *fiber.Ctx) error {
	userID, userErr := uuid.Parse(ctx.Params("userId"))
	requestID, requestErr := uuid.Parse(ctx.Params("requestId"))
	if userErr != nil || requestErr != nil || userID == uuid.Nil || requestID == uuid.Nil || len(bytes.TrimSpace(ctx.Body())) != 0 {
		return adminUserValidationError()
	}
	return ctx.Next()
}

// adminUserError maps restricted service failures to safe, fixed API errors.
// Implements DESIGN-009 UserAdminPanel.
func adminUserError(err error) error {
	switch {
	case errors.Is(err, useradmin.ErrForbidden):
		return AppError{HTTPStatus: fiber.StatusForbidden, Category: "auth", Code: "forbidden", Message: "administrator access required"}
	case repository.IsKind(err, repository.ErrorKindValidation):
		return adminUserValidationError()
	case repository.IsKind(err, repository.ErrorKindNotFound):
		return AppError{HTTPStatus: fiber.StatusNotFound, Category: "validation", Code: "not_found", Message: "resource not found"}
	case repository.IsKind(err, repository.ErrorKindConnection), repository.IsKind(err, repository.ErrorKindRetryable), repository.IsKind(err, repository.ErrorKindCanceled):
		return adminUserDependencyError()
	default:
		return err
	}
}

// adminUserValidationError returns the fixed validation envelope for restricted administration.
// Implements DESIGN-009 UserAdminPanel.
func adminUserValidationError() AppError {
	return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
}

// adminUserDependencyError returns the fixed dependency envelope for restricted administration.
// Implements DESIGN-009 UserAdminPanel.
func adminUserDependencyError() AppError {
	return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true}
}

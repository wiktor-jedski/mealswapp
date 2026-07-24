package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// AdminContext is immutable server-derived authorization and correlation metadata.
// Implements DESIGN-009 AdminController.
type AdminContext struct {
	UserID    uuid.UUID
	Role      string
	RequestID string
}

// AdminMutationResult delays the client response until mutation and audit commit together.
// Implements DESIGN-009 AdminController fail-closed transactional audit boundary.
type AdminMutationResult struct {
	HTTPStatus  int
	Data        map[string]any
	Audit       repository.AdminAuditChanges
	AfterCommit func()
}

// AdminMutationHandler performs one transaction-scoped mutation and returns a deferred safe response.
// Implements DESIGN-009 AdminController fail-closed transactional audit boundary.
type AdminMutationHandler func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error)

// AdminRouteDefinition describes one explicitly registered route below /api/v1/admin.
// Implements DESIGN-009 AdminController and DESIGN-010 RouteHandler.
type AdminRouteDefinition struct {
	Method      string
	Path        string
	Handler     fiber.Handler
	Mutation    AdminMutationHandler
	Validate    fiber.Handler
	RateLimit   *RateLimitRule
	AuditAction string
	EntityType  string
}

// AdminController owns the versioned admin route group and transactional audit coordination.
// Implements DESIGN-009 AdminController.
type AdminController struct {
	audit          repository.AdminMutationAuditRepository
	routes         []AdminRouteDefinition
	now            func() time.Time
	externalSearch ExternalSearchService
	telemetry      *observability.AdminExternalTelemetry
}

// WithTelemetry adds bounded mutation and audit-failure observations.
// Implements DESIGN-014 MetricsCollector and LogAggregator.
func (c *AdminController) WithTelemetry(telemetry *observability.AdminExternalTelemetry) *AdminController {
	if c != nil {
		c.telemetry = telemetry
	}
	return c
}

// Implements DESIGN-009 AdminController compile-time route controller contract.
var _ Controller = (*AdminController)(nil)

// NewAdminController creates an allowlisted admin route group.
// Implements DESIGN-009 AdminController.
func NewAdminController(audit repository.AdminMutationAuditRepository, routes ...AdminRouteDefinition) *AdminController {
	return &AdminController{audit: audit, routes: routes, now: time.Now}
}

// Routes converts explicit admin definitions into the shared versioned gateway contract.
// Implements DESIGN-009 AdminController and DESIGN-010 RouteHandler.
func (c *AdminController) Routes() []RouteDefinition {
	routes := make([]RouteDefinition, 0, len(c.routes))
	seen := make([]AdminRouteDefinition, 0, len(c.routes))
	for _, route := range c.routes {
		c.validateRoute(route)
		for _, registered := range seen {
			if route.Method == registered.Method && adminRoutePathsCollide(route.Path, registered.Path) {
				panic("colliding admin route definition")
			}
		}
		seen = append(seen, route)
		handler := route.Handler
		mutation := isMutation(route.Method)
		if mutation {
			handler = c.transactionalMutation(route)
		}
		routes = append(routes, RouteDefinition{
			Method: route.Method, Path: "/admin" + route.Path, Handler: handler,
			RequiresAuth: true, RequiresAdmin: true, RequiresCSRF: mutation,
			RequiresAudit: mutation, Validate: route.Validate, RateLimit: route.RateLimit,
		})
	}
	return routes
}

// adminRoutePathsCollide reports whether two safe templates can match the same request path.
// Implements DESIGN-009 AdminController collision-free documented route allowlist.
func adminRoutePathsCollide(left string, right string) bool {
	leftSegments := strings.Split(left[1:], "/")
	rightSegments := strings.Split(right[1:], "/")
	if len(leftSegments) != len(rightSegments) {
		return false
	}
	for index := range leftSegments {
		leftParameter := strings.HasPrefix(leftSegments[index], ":")
		rightParameter := strings.HasPrefix(rightSegments[index], ":")
		if !leftParameter && !rightParameter && leftSegments[index] != rightSegments[index] {
			return false
		}
	}
	return true
}

// RequireAdmin returns only server-verified admin identity and request metadata.
// Implements DESIGN-009 AdminController.
func RequireAdmin(ctx *fiber.Ctx) (AdminContext, error) {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AdminContext{}, AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	if user.Role != string(repository.UserRoleAdmin) {
		return AdminContext{}, AppError{HTTPStatus: fiber.StatusForbidden, Category: "auth", Code: "forbidden", Message: "administrator access required"}
	}
	return AdminContext{UserID: user.UserID, Role: string(repository.UserRoleAdmin), RequestID: requestID(ctx)}, nil
}

// requireAdminRole enforces role authorization after verified cookie authentication.
// Implements DESIGN-009 AdminController.
func requireAdminRole(ctx *fiber.Ctx) error {
	if _, err := RequireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Next()
}

// validateRoute fails startup for admin routes that could bypass a required control.
// Implements DESIGN-009 AdminController secure route registration.
func (c *AdminController) validateRoute(route AdminRouteDefinition) {
	mutation := isMutation(route.Method)
	if route.Method != fiber.MethodGet && !mutation {
		panic("admin routes support only documented read and mutation methods")
	}
	if !isSafeAdminRoutePath(route.Path) || strings.HasPrefix(route.Path, "/admin") {
		panic("admin route must use a safe path relative to /api/v1/admin")
	}
	if route.RateLimit == nil || (route.RateLimit.Scope != "user" && route.RateLimit.Scope != "endpoint") {
		panic("admin routes require a user- or endpoint-scoped rate limit")
	}
	if mutation {
		if route.Mutation == nil || route.Handler != nil || route.Validate == nil || route.AuditAction != strings.TrimSpace(route.AuditAction) || route.EntityType != strings.TrimSpace(route.EntityType) || route.AuditAction == "" || route.EntityType == "" {
			panic("admin mutations require validation, one transactional handler, and fixed audit metadata")
		}
		return
	}
	if route.Handler == nil || route.Mutation != nil || route.AuditAction != "" || route.EntityType != "" {
		panic("admin reads require one read handler and no mutation audit metadata")
	}
}

// isSafeAdminRoutePath accepts only explicit literal segments and required named parameters.
// Implements DESIGN-009 AdminController documented route allowlist.
func isSafeAdminRoutePath(path string) bool {
	if path == "" || path == "/" || !strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return false
	}
	parameters := make(map[string]struct{})
	for _, segment := range strings.Split(path[1:], "/") {
		if segment == "" {
			return false
		}
		if strings.HasPrefix(segment, ":") {
			name := segment[1:]
			if !isAdminRouteIdentifier(name) {
				return false
			}
			if _, duplicate := parameters[name]; duplicate {
				return false
			}
			parameters[name] = struct{}{}
			continue
		}
		if !isAdminRouteLiteral(segment) {
			return false
		}
	}
	return true
}

// isAdminRouteLiteral validates one explicit kebab-case path segment.
// Implements DESIGN-009 AdminController documented route allowlist.
func isAdminRouteLiteral(segment string) bool {
	for index, char := range segment {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if index > 0 && ((char >= '0' && char <= '9') || char == '-') {
			continue
		}
		return false
	}
	return true
}

// isAdminRouteIdentifier validates one required lower-camel-case route parameter.
// Implements DESIGN-009 AdminController documented route allowlist.
func isAdminRouteIdentifier(identifier string) bool {
	for index, char := range identifier {
		if (char >= 'a' && char <= 'z') || (index > 0 && (char >= 'A' && char <= 'Z' || char >= '0' && char <= '9')) {
			continue
		}
		return false
	}
	return identifier != ""
}

// transactionalMutation joins one admin mutation to audit persistence before a response can commit.
// Implements DESIGN-009 AdminController fail-closed transactional audit boundary.
func (c *AdminController) transactionalMutation(route AdminRouteDefinition) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		admin, err := RequireAdmin(ctx)
		if err != nil {
			return err
		}
		if c.audit == nil {
			c.telemetry.AdminMutation(ctx.UserContext(), route.AuditAction, "audit_failed")
			return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true}
		}
		entry := repository.AdminAuditEntry{AdminUserID: admin.UserID, Action: route.AuditAction, EntityType: route.EntityType, RequestID: admin.RequestID, CreatedAt: c.now()}
		var result AdminMutationResult
		var response []byte
		status := fiber.StatusOK
		if err := c.audit.WithMutationAudit(ctx.UserContext(), entry, func(tx repository.AdminMutationExecutor) (repository.AdminAuditChanges, error) {
			var mutationErr error
			result, mutationErr = route.Mutation(ctx, tx)
			if mutationErr != nil {
				return result.Audit, mutationErr
			}
			status = result.HTTPStatus
			if status == 0 {
				status = fiber.StatusOK
			}
			if status < fiber.StatusOK || status >= fiber.StatusMultipleChoices {
				return result.Audit, errors.New("admin mutation response status must be successful")
			}
			if status == fiber.StatusNoContent {
				if len(result.Data) != 0 {
					return result.Audit, errors.New("admin no-content response cannot contain data")
				}
				return result.Audit, nil
			}
			response, mutationErr = json.Marshal(Envelope{Status: "ok", RequestID: admin.RequestID, Data: result.Data})
			if mutationErr != nil {
				return result.Audit, fmt.Errorf("encode admin mutation response: %w", mutationErr)
			}
			return result.Audit, nil
		}); err != nil {
			if errors.Is(err, repository.ErrAdminAuditPersistence) {
				c.telemetry.AdminMutation(ctx.UserContext(), route.AuditAction, "audit_failed")
				return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true, Cause: err}
			}
			c.telemetry.AdminMutation(ctx.UserContext(), route.AuditAction, "failed")
			return err
		}
		c.telemetry.AdminMutation(ctx.UserContext(), route.AuditAction, "succeeded")
		if result.AfterCommit != nil {
			result.AfterCommit()
		}
		if status == fiber.StatusNoContent {
			return ctx.SendStatus(fiber.StatusNoContent)
		}
		ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		return ctx.Status(status).Send(response)
	}
}

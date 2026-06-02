package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// observabilityFallbackWriter reports sink failures without calling the failed sink again.
// Implements DESIGN-014 LogAggregator.
var observabilityFallbackWriter io.Writer = os.Stderr

// requestIsTLS is replaceable in tests because Fiber's in-memory transport has no TLS connection state.
// Implements DESIGN-013 TLSEnforcer.
var requestIsTLS = func(ctx *fiber.Ctx) bool { return ctx.Context().IsTLS() }

// Dependencies provides infrastructure and cross-cutting services to the HTTP router.
// Implements DESIGN-010 RouteHandler dependency boundary.
type Dependencies struct {
	Config       config.Config
	PostgresPing func(context.Context) error
	RedisPing    func(context.Context) error
	Audit        security.AuditLogger
	Logs         observability.LogSink
	Metrics      observability.MetricsCollector
	Routes       []RouteDefinition
	CSRF         *CSRFManager
}

// GatewayContext is request-scoped gateway metadata.
// Implements DESIGN-010 RouteHandler.
type GatewayContext struct {
	RequestID string
	UserID    string
	StartedAt time.Time
	Deadline  time.Time
}

// AppError is a user-safe classified server error.
// Implements DESIGN-017 GlobalExceptionHandler.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Category   string `json:"category"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	RequestID  string `json:"requestId,omitempty"`
	Cause      error  `json:"-"`
}

// Error returns the safe application error code.
// Implements DESIGN-017 GlobalExceptionHandler.
func (e AppError) Error() string { return e.Code }

// Envelope is the shared JSON response wrapper returned by API handlers.
// Implements DESIGN-017 GlobalExceptionHandler.
type Envelope struct {
	Status    string         `json:"status"`
	RequestID string         `json:"requestId"`
	Data      map[string]any `json:"data,omitempty"`
	Error     *AppError      `json:"error,omitempty"`
}

// RouteDefinition describes a versioned service route and required gateway hooks.
// Implements DESIGN-010 RouteHandler.
type RouteDefinition struct {
	Method        string
	Path          string
	Handler       fiber.Handler
	RequiresAuth  bool
	RequiresCSRF  bool
	ExemptCSRF    bool
	RequiresAudit bool
	Validate      fiber.Handler
	RateLimit     *RateLimitRule
}

// RateLimitRule configures a scoped Fiber request limit.
// Implements DESIGN-010 RateLimiter.
type RateLimitRule struct {
	Scope         string
	MaxRequests   int
	WindowSeconds int
}

// rateLimitHandler enforces one endpoint rate-limit rule with Fiber middleware.
// Implements DESIGN-010 RateLimiter.
func rateLimitHandler(rule RateLimitRule) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        rule.MaxRequests,
		Expiration: time.Duration(rule.WindowSeconds) * time.Second,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return rateLimitKey(ctx, rule.Scope)
		},
		LimitReached: func(ctx *fiber.Ctx) error {
			return AppError{HTTPStatus: fiber.StatusTooManyRequests, Category: "auth", Code: "rate_limited", Message: "too many requests", Retryable: true}
		},
	})
}

// FailedLoginRule returns the Phase 03 reusable brute-force limit.
// Implements DESIGN-010 RateLimiter.
func FailedLoginRule() RateLimitRule {
	return RateLimitRule{Scope: "ip", MaxRequests: 10, WindowSeconds: 600}
}

// NewRouter constructs the Fiber application with the Phase 02 gateway stack.
// Implements DESIGN-010 RouteHandler.
func NewRouter(deps Dependencies) (*fiber.App, error) {
	if deps.Config.APITimeout <= 0 {
		return nil, errors.New("API timeout must be positive")
	}
	if len(deps.Config.AllowedOrigins) == 0 {
		return nil, errors.New("at least one allowed origin is required")
	}
	app := fiber.New(fiber.Config{ErrorHandler: writeError})
	app.Use(requestid.New())
	app.Use(gatewayContext(deps.Config.APITimeout))
	app.Use(cors(deps.Config.AllowedOrigins))
	app.Use(tlsEnforcer(deps.Config))
	app.Use(securityHeaders(deps.Config))
	app.Use(instrument(deps))
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	if deps.CSRF == nil {
		deps.CSRF = NewCSRFManager(deps.Config, deps.Audit)
	}
	app.Use(deps.CSRF.IssueToken)
	app.Get("/health", health)
	app.Get("/ready", ready(deps))
	v1 := app.Group("/api/v1")
	v1.Get("/health", health)
	v1.Get("/ready", ready(deps))
	v1.Get("/auth/csrf-token", csrfToken)
	registerV1Routes(v1, deps)
	return app, nil
}

// registerV1Routes composes route-specific gateway hooks.
// Implements DESIGN-010 RouteHandler.
func registerV1Routes(group fiber.Router, deps Dependencies) {
	for _, route := range deps.Routes {
		if isMutation(route.Method) && route.RequiresCSRF == route.ExemptCSRF {
			panic("mutations must declare exactly one CSRF policy")
		}
		handlers := []fiber.Handler{}
		if route.RequiresAuth {
			handlers = append(handlers, requireAuth)
		}
		if route.RequiresCSRF {
			handlers = append(handlers, deps.CSRF.Validate)
		}
		if route.Validate != nil {
			handlers = append(handlers, route.Validate)
		}
		if route.RateLimit != nil {
			handlers = append(handlers, rateLimitHandler(*route.RateLimit))
		}
		if route.RequiresAudit {
			handlers = append(handlers, requireAudit(deps.Audit))
		}
		handlers = append(handlers, route.Handler)
		group.Add(route.Method, route.Path, handlers...)
	}
}

// isMutation reports whether an HTTP method changes server state.
// Implements DESIGN-010 CSRFValidator.
func isMutation(method string) bool {
	return slices.Contains([]string{fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete}, method)
}

// gatewayContext attaches request metadata and enforces cancellation deadlines.
// Implements DESIGN-010 RouteHandler.
func gatewayContext(timeout time.Duration) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		started := time.Now()
		userCtx, cancel := context.WithTimeout(ctx.UserContext(), timeout)
		defer cancel()
		ctx.SetUserContext(userCtx)
		ctx.Locals("gateway", GatewayContext{RequestID: requestID(ctx), StartedAt: started, Deadline: started.Add(timeout)})
		err := ctx.Next()
		if errors.Is(userCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return AppError{HTTPStatus: fiber.StatusGatewayTimeout, Category: "timeout", Code: "request_timeout", Message: "request timed out", Retryable: true}
		}
		return err
	}
}

// securityHeaders writes the global browser security policy.
// Implements DESIGN-010 SecurityHeaderMiddleware.
func securityHeaders(cfg config.Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		ctx.Set("Content-Security-Policy", "default-src 'self'")
		ctx.Set("X-Frame-Options", "DENY")
		ctx.Set("X-Content-Type-Options", "nosniff")
		ctx.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if cfg.EnforceTLS {
			ctx.Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge))
		}
		return ctx.Next()
	}
}

// cors enforces the configured credentialed browser origins.
// Implements DESIGN-010 CORSHandler.
func cors(origins []string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		origin := ctx.Get("Origin")
		if origin != "" {
			if !slices.Contains(origins, origin) {
				return AppError{HTTPStatus: fiber.StatusForbidden, Category: "auth", Code: "cors_forbidden", Message: "origin is not allowed"}
			}
			ctx.Set("Access-Control-Allow-Origin", origin)
			ctx.Set("Access-Control-Allow-Credentials", "true")
			ctx.Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			ctx.Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if ctx.Method() == fiber.MethodOptions {
			return ctx.SendStatus(fiber.StatusNoContent)
		}
		return ctx.Next()
	}
}

// tlsEnforcer redirects deployed HTTP traffic to HTTPS.
// Implements DESIGN-013 TLSEnforcer.
func tlsEnforcer(cfg config.Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !cfg.EnforceTLS {
			return ctx.Next()
		}
		if !requestIsTLS(ctx) {
			return ctx.Redirect("https://"+ctx.Hostname()+ctx.OriginalURL(), fiber.StatusPermanentRedirect)
		}
		return ctx.Next()
	}
}

// requireAudit fails closed before dispatching security-sensitive mutations.
// Implements DESIGN-013 AuditLogger.
func requireAudit(audit security.AuditLogger) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		userID, err := auditUserID(ctx)
		if err != nil {
			return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required", Cause: err}
		}
		err = security.RecordAuditRequired(ctx.UserContext(), audit, security.AuditLogEntry{
			RequestID: requestID(ctx), UserID: userID, Action: "api.mutation", Resource: routeTemplate(ctx),
			Outcome: "attempt", IP: ctx.IP(), UserAgent: ctx.Get("User-Agent"), CreatedAt: time.Now(),
		})
		if err != nil {
			return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true, Cause: err}
		}
		return ctx.Next()
	}
}

// requireAuth is the Phase 02 protected-route hook replaced by Phase 03 JWT validation.
// Implements DESIGN-010 RouteHandler.
func requireAuth(ctx *fiber.Ctx) error {
	userID := strings.TrimSpace(ctx.Get("X-Test-User-ID"))
	if userID == "" {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	if _, err := uuid.Parse(userID); err != nil {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required", Cause: fmt.Errorf("parse authenticated user ID: %w", err)}
	}
	return ctx.Next()
}

// ValidateJSON returns middleware that parses and validates JSON payloads.
// Implements DESIGN-010 RequestValidator.
func ValidateJSON(validate func(map[string]any) error) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		body := map[string]any{}
		if err := ctx.BodyParser(&body); err != nil {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
		}
		if err := validate(body); err != nil {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
		}
		return ctx.Next()
	}
}

// ValidateQuery returns middleware that validates query parameters.
// Implements DESIGN-010 RequestValidator.
func ValidateQuery(validate func(map[string]string) error) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		values := map[string]string{}
		ctx.Context().QueryArgs().VisitAll(func(key []byte, value []byte) { values[string(key)] = string(value) })
		if err := validate(values); err != nil {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
		}
		return ctx.Next()
	}
}

// ValidatePath returns middleware that validates path parameters.
// Implements DESIGN-010 RequestValidator.
func ValidatePath(name string, validate func(string) error) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if err := validate(ctx.Params(name)); err != nil {
			return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
		}
		return ctx.Next()
	}
}

// health reports process liveness.
// Implements DESIGN-014 UptimeMonitor.
func health(ctx *fiber.Ctx) error {
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"service": "mealswapp-api"}})
}

// ready reports dependency readiness and metrics.
// Implements DESIGN-014 UptimeMonitor.
func ready(deps Dependencies) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		checks := map[string]string{}
		status := fiber.StatusOK
		for name, ping := range map[string]func(context.Context) error{"postgres": deps.PostgresPing, "redis": deps.RedisPing} {
			if ping == nil {
				continue
			}
			if err := ping(ctx.UserContext()); err != nil {
				checks[name], status = "unavailable", fiber.StatusServiceUnavailable
			} else {
				checks[name] = "ok"
			}
			health := 0.0
			if checks[name] == "ok" {
				health = 1
			}
			recordMetric(ctx, deps.Metrics, "dependency_health", health, "state", map[string]string{"dependency": name})
		}
		state := "ready"
		if status != fiber.StatusOK {
			state = "not_ready"
		}
		return ctx.Status(status).JSON(Envelope{Status: state, RequestID: requestID(ctx), Data: map[string]any{"checks": checks}})
	}
}

// instrument emits structured logs and low-cardinality HTTP metrics.
// Implements DESIGN-014 FiberLogger.
func instrument(deps Dependencies) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		started := time.Now()
		recordMetric(ctx, deps.Metrics, "http_concurrent_requests", 1, "requests", nil)
		defer recordMetric(ctx, deps.Metrics, "http_concurrent_requests", -1, "requests", nil)
		err := ctx.Next()
		status := ctx.Response().StatusCode()
		if err != nil {
			status = ClassifyServerError(err).HTTPStatus
			userID, userIDErr := auditUserID(ctx)
			if userIDErr != nil {
				reportObservabilityFailure("audit user ID", userIDErr)
			}
			security.RecordAuditBestEffort(ctx.UserContext(), deps.Audit, security.AuditLogEntry{RequestID: requestID(ctx), UserID: userID, Action: "api.error", Resource: routeTemplate(ctx), Outcome: "failure", IP: ctx.IP(), UserAgent: ctx.Get("User-Agent"), CreatedAt: time.Now()})
		}
		userID, userIDErr := auditUserID(ctx)
		if userIDErr != nil {
			reportObservabilityFailure("audit user ID", userIDErr)
		}
		outcome := "success"
		if status >= 400 {
			outcome = "failure"
		}
		security.RecordAuditBestEffort(ctx.UserContext(), deps.Audit, security.AuditLogEntry{RequestID: requestID(ctx), UserID: userID, Action: "api.request", Resource: routeTemplate(ctx), Outcome: outcome, IP: ctx.IP(), UserAgent: ctx.Get("User-Agent"), CreatedAt: time.Now()})
		labels := map[string]string{"route": routeTemplate(ctx), "status": strconv.Itoa(status)}
		recordMetric(ctx, deps.Metrics, "http_request_latency_seconds", time.Since(started).Seconds(), "seconds", labels)
		recordMetric(ctx, deps.Metrics, "http_response_total", 1, "responses", labels)
		if status >= 400 {
			recordMetric(ctx, deps.Metrics, "http_error_total", 1, "errors", labels)
		}
		if deps.Logs != nil {
			fields := map[string]any{"route": routeTemplate(ctx), "method": ctx.Method(), "status": status, "latencyMs": time.Since(started).Milliseconds()}
			if userID := ctx.Get("X-Test-User-ID"); userID != "" {
				fields["userId"] = userID
			}
			if err := deps.Logs.Log(ctx.UserContext(), observability.LogEvent{RequestID: requestID(ctx), Service: "api", Level: logLevel(status), Message: "http_request", Fields: fields, CreatedAt: time.Now()}); err != nil {
				reportObservabilityFailure("log", err)
			}
		}
		return err
	}
}

// auditUserID returns an optional authenticated user UUID for security audit metadata.
// Implements DESIGN-013 AuditLogger.
func auditUserID(ctx *fiber.Ctx) (*uuid.UUID, error) {
	value := strings.TrimSpace(ctx.Get("X-Test-User-ID"))
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse audit user ID: %w", err)
	}
	return &id, nil
}

// logLevel maps response statuses to structured log severity.
// Implements DESIGN-017 ErrorMessageMapper.
func logLevel(status int) string {
	switch {
	case status >= 500:
		return "error"
	case status >= 400:
		return "warn"
	default:
		return "info"
	}
}

// recordMetric writes one optional metric observation.
// Implements DESIGN-014 MetricsCollector.
func recordMetric(ctx *fiber.Ctx, metrics observability.MetricsCollector, name string, value float64, unit string, labels map[string]string) {
	if metrics != nil {
		if err := metrics.RecordMetric(ctx.UserContext(), observability.MetricPoint{Name: name, Value: value, Unit: unit, Labels: labels, ObservedAt: time.Now()}); err != nil {
			reportObservabilityFailure("metric", err)
		}
	}
}

// reportObservabilityFailure writes a non-recursive fallback diagnostic.
// Implements DESIGN-014 LogAggregator.
func reportObservabilityFailure(kind string, err error) {
	fmt.Fprintf(observabilityFallbackWriter, "observability %s sink failure: %v\n", kind, err)
}

// writeError serializes a safe classified error envelope.
// Implements DESIGN-017 GlobalExceptionHandler.
func writeError(ctx *fiber.Ctx, err error) error {
	appErr := ClassifyServerError(err)
	appErr.RequestID = requestID(ctx)
	return ctx.Status(appErr.HTTPStatus).JSON(Envelope{Status: "error", RequestID: appErr.RequestID, Error: &appErr})
}

// ClassifyServerError maps returned errors into safe response metadata.
// Implements DESIGN-017 GlobalExceptionHandler.
func ClassifyServerError(err error) AppError {
	var appErr AppError
	if errors.As(err, &appErr) {
		if appErr.HTTPStatus == 0 {
			appErr.HTTPStatus = fiber.StatusInternalServerError
		}
		return appErr
	}
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return AppError{HTTPStatus: fiberErr.Code, Category: "validation", Code: http.StatusText(fiberErr.Code), Message: fiberErr.Message}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return AppError{HTTPStatus: fiber.StatusGatewayTimeout, Category: "timeout", Code: "request_timeout", Message: "request timed out", Retryable: true}
	case repository.IsKind(err, repository.ErrorKindValidation):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
	case repository.IsKind(err, repository.ErrorKindNotFound):
		return AppError{HTTPStatus: fiber.StatusNotFound, Category: "validation", Code: "not_found", Message: "resource not found"}
	case repository.IsKind(err, repository.ErrorKindConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "conflict", Message: "resource conflicts with existing data"}
	case repository.IsKind(err, repository.ErrorKindConnection):
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "dependency_unavailable", Message: "service temporarily unavailable", Retryable: true}
	}
	return AppError{HTTPStatus: fiber.StatusInternalServerError, Category: "server", Code: "internal_error", Message: "internal server error", Cause: err}
}

// requestID reads request correlation metadata.
// Implements DESIGN-010 RouteHandler.
func requestID(ctx *fiber.Ctx) string {
	if id, ok := ctx.Locals("requestid").(string); ok {
		return id
	}
	return ""
}

// routeTemplate returns a low-cardinality route label.
// Implements DESIGN-014 MetricsCollector.
func routeTemplate(ctx *fiber.Ctx) string {
	route := ctx.Route()
	if route == nil || route.Path == "" || len(route.Handlers) == 0 || (route.Path == "/" && ctx.Path() != "/") {
		return "unmatched"
	}
	return route.Path
}

// rateLimitKey derives a scoped fixed-window counter key.
// Implements DESIGN-010 RateLimiter.
func rateLimitKey(ctx *fiber.Ctx, scope string) string {
	if scope == "user" {
		return scope + ":" + ctx.Get("X-Test-User-ID")
	}
	if scope == "endpoint" {
		return scope + ":" + ctx.Path()
	}
	return "ip:" + ctx.IP()
}

package http

import (
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/handlers"
	"mealswapp/backend/internal/http/middleware"
	"mealswapp/backend/internal/http/session"
	"mealswapp/backend/internal/observability"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type ServiceDependencies struct {
	Config                   config.Config
	ReadinessChecker         handlers.ReadinessChecker
	Metrics                  *observability.MetricsCollector
	AuthService              handlers.AuthService
	SessionManager           handlers.AuthCookieManager
	OAuthService             handlers.OAuthService
	AccountFlowService       handlers.AccountFlowService
	ProfileService           handlers.ProfileService
	SavedDataService         handlers.SavedDataService
	SearchService            handlers.SearchService
	SearchUsageLimiter       handlers.SearchUsageLimiter
	SubscriptionService      handlers.SubscriptionService
	StripeWebhookService     handlers.StripeWebhookService
	OptimizationService      handlers.OptimizationService
	OptimizationUserResolver handlers.OptimizationUserResolver
	AdminSummaryService      handlers.AdminSummaryService
	ExternalSearchService    handlers.ExternalSearchService
	ItemCuratorService       handlers.ItemCuratorService
	TagManagerService        handlers.TagManagerService
	UserAdminService         handlers.UserAdminService
}

type RouteDefinition struct {
	Method      string
	Path        string
	Version     string
	Handler     fiber.Handler
	Middlewares []fiber.Handler
}

type GatewayContext struct {
	RequestID string
	StartedAt time.Time
	Deadline  time.Time
}

func NewRouter(deps ServiceDependencies) *fiber.App {
	metrics := deps.Metrics
	if metrics == nil {
		metrics = observability.NewMetricsCollector()
	}
	deps.Metrics = metrics

	app := fiber.New(fiber.Config{
		AppName:      "mealswapp-api",
		ErrorHandler: GlobalExceptionHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(gatewayContextMiddleware(10 * time.Second))
	app.Use(middleware.TLSEnforcer(middleware.DefaultTLSPolicy(deps.Config.Environment)))
	app.Use(middleware.RequestLogger(middleware.DefaultRequestLoggerConfig()))
	app.Use(metrics.Middleware())
	app.Use(middleware.SecurityHeadersMiddleware(middleware.DefaultSecurityHeaders()))
	app.Use(middleware.CORSHandler(middleware.DefaultCORSConfig(deps.Config)))
	app.Use(middleware.CSRFValidator(middleware.DefaultCSRFConfig()))
	app.Use(middleware.RateLimiter(middleware.DefaultRateLimiterConfig()))

	health := handlers.NewHealthHandler(deps.Config, deps.ReadinessChecker, metrics)
	app.Get("/health", health.Health)
	app.Get("/ready", health.Ready)
	app.Get("/metrics", metrics.Handler)
	app.Get("/static/similarity/:file", handlers.SimilarityAsset)

	RegisterV1Routes(app, deps)
	app.Use(notFoundHandler)

	return app
}

func RegisterV1Routes(app *fiber.App, deps ServiceDependencies) {
	api := app.Group("/api/v1")
	metrics := deps.Metrics
	if metrics == nil {
		metrics = observability.NewMetricsCollector()
	}
	health := handlers.NewHealthHandler(deps.Config, deps.ReadinessChecker, metrics)

	routes := []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/health", Version: "v1", Handler: health.Health},
		{Method: fiber.MethodGet, Path: "/ready", Version: "v1", Handler: health.Ready},
	}
	if deps.AuthService != nil {
		sessionManager := deps.SessionManager
		if sessionManager == nil {
			sessionManager = session.NewManager(session.Config{Environment: deps.Config.Environment})
		}
		auth := handlers.NewAuthHandler(deps.AuthService, sessionManager)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/register", Version: "v1", Handler: auth.Register},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/login", Version: "v1", Handler: auth.Login},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/logout", Version: "v1", Handler: auth.Logout},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/refresh", Version: "v1", Handler: auth.Refresh},
			RouteDefinition{Method: fiber.MethodGet, Path: "/auth/me", Version: "v1", Handler: auth.CurrentUser},
		)
	}
	if deps.OAuthService != nil {
		oauth := handlers.NewOAuthHandler(deps.OAuthService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/oauth/:provider/start", Version: "v1", Handler: oauth.Start},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/oauth/:provider/callback", Version: "v1", Handler: oauth.Callback},
		)
	}
	if deps.AccountFlowService != nil {
		flows := handlers.NewAccountFlowHandler(deps.AccountFlowService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/password-reset/request", Version: "v1", Handler: flows.RequestPasswordReset},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/password-reset/confirm", Version: "v1", Handler: flows.ConfirmPasswordReset},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/email-verification/request", Version: "v1", Handler: flows.RequestEmailVerification},
			RouteDefinition{Method: fiber.MethodPost, Path: "/auth/email-verification/confirm", Version: "v1", Handler: flows.ConfirmEmailVerification},
		)
	}
	if deps.ProfileService != nil {
		profile := handlers.NewProfileHandler(deps.ProfileService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodGet, Path: "/profile", Version: "v1", Handler: profile.Get},
			RouteDefinition{Method: fiber.MethodPatch, Path: "/profile", Version: "v1", Handler: profile.Update},
		)
	}
	if deps.SavedDataService != nil {
		saved := handlers.NewSavedDataHandler(deps.SavedDataService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodGet, Path: "/saved-data", Version: "v1", Handler: saved.List},
			RouteDefinition{Method: fiber.MethodPost, Path: "/saved-data", Version: "v1", Handler: saved.Create},
			RouteDefinition{Method: fiber.MethodPatch, Path: "/saved-data/:id", Version: "v1", Handler: saved.Update},
			RouteDefinition{Method: fiber.MethodDelete, Path: "/saved-data/:id", Version: "v1", Handler: saved.Delete},
		)
	}
	if deps.SearchService != nil {
		search := handlers.NewSearchHandlerWithUsageLimiter(deps.SearchService, deps.SearchUsageLimiter)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodPost, Path: "/search", Version: "v1", Handler: search.Search},
			RouteDefinition{Method: fiber.MethodGet, Path: "/autocomplete", Version: "v1", Handler: search.Autocomplete},
		)
	}
	if deps.SubscriptionService != nil {
		subscription := handlers.NewSubscriptionHandler(deps.SubscriptionService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodGet, Path: "/subscription/status", Version: "v1", Handler: subscription.Status},
			RouteDefinition{Method: fiber.MethodPost, Path: "/subscription/checkout", Version: "v1", Handler: subscription.CreateCheckout},
			RouteDefinition{Method: fiber.MethodPost, Path: "/subscription/portal", Version: "v1", Handler: subscription.CreateCustomerPortal},
			RouteDefinition{Method: fiber.MethodGet, Path: "/subscription/entitlement", Version: "v1", Handler: subscription.Entitlement},
		)
	}
	if deps.StripeWebhookService != nil {
		webhook := handlers.NewStripeWebhookHandler(deps.StripeWebhookService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodPost, Path: "/webhooks/stripe", Version: "v1", Handler: webhook.Handle},
		)
	}
	if deps.OptimizationService != nil {
		optimization := handlers.NewOptimizationHandler(deps.OptimizationService, deps.OptimizationUserResolver)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodPost, Path: "/optimization/jobs", Version: "v1", Handler: optimization.Submit},
			RouteDefinition{Method: fiber.MethodGet, Path: "/optimization/jobs/:id", Version: "v1", Handler: optimization.GetJob},
		)
	}
	if deps.AuthService != nil {
		admin := handlers.NewAdminHandler(deps.AuthService, deps.AdminSummaryService, deps.ExternalSearchService, deps.ItemCuratorService, deps.TagManagerService, deps.UserAdminService)
		routes = append(routes,
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/summary", Version: "v1", Handler: admin.Summary, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/external-search", Version: "v1", Handler: admin.SearchExternal, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/items", Version: "v1", Handler: admin.ListItems, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/items", Version: "v1", Handler: admin.CreateItem, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/items/:id", Version: "v1", Handler: admin.GetItem, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPatch, Path: "/admin/items/:id", Version: "v1", Handler: admin.UpdateItem, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodDelete, Path: "/admin/items/:id", Version: "v1", Handler: admin.DeleteItem, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/tags", Version: "v1", Handler: admin.ListTags, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/tags", Version: "v1", Handler: admin.UpsertTag, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/tags/merge", Version: "v1", Handler: admin.MergeTags, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/tags/:id/deactivate", Version: "v1", Handler: admin.DeactivateTag, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/items/:id/tags", Version: "v1", Handler: admin.AssignTag, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodDelete, Path: "/admin/items/:id/tags/:tagId", Version: "v1", Handler: admin.RemoveTag, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/users", Version: "v1", Handler: admin.ListUsers, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/users/:id", Version: "v1", Handler: admin.GetUser, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/users/:id/disable", Version: "v1", Handler: admin.DisableUser, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/users/:id/reset-lockout", Version: "v1", Handler: admin.ResetUserLockout, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodGet, Path: "/admin/users/:id/audit", Version: "v1", Handler: admin.UserAuditHistory, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
			RouteDefinition{Method: fiber.MethodPost, Path: "/admin/items/:id/:transition", Version: "v1", Handler: admin.TransitionItem, Middlewares: []fiber.Handler{admin.RequireAdminMiddleware()}},
		)
	}

	for _, route := range routes {
		handlers := append(route.Middlewares, route.Handler)
		api.Add(route.Method, route.Path, handlers...)
	}
}

func ExtractGatewayContext(ctx *fiber.Ctx) GatewayContext {
	if gatewayContext, ok := ctx.Locals("gatewayContext").(GatewayContext); ok {
		return gatewayContext
	}

	now := time.Now().UTC()
	return GatewayContext{
		RequestID: requestID(ctx),
		StartedAt: now,
		Deadline:  now.Add(10 * time.Second),
	}
}

func gatewayContextMiddleware(timeout time.Duration) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		startedAt := time.Now().UTC()
		ctx.Locals("gatewayContext", GatewayContext{
			RequestID: requestID(ctx),
			StartedAt: startedAt,
			Deadline:  startedAt.Add(timeout),
		})

		return ctx.Next()
	}
}

func notFoundHandler(ctx *fiber.Ctx) error {
	return apperrors.NotFound("Route not found")
}

func requestID(ctx *fiber.Ctx) string {
	if value, ok := ctx.Locals("requestid").(string); ok {
		return value
	}

	return ctx.GetRespHeader("X-Request-ID")
}

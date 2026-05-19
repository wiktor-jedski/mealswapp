package http

import (
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/handlers"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type ServiceDependencies struct {
	Config config.Config
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
	app := fiber.New(fiber.Config{
		AppName:      "mealswapp-api",
		ErrorHandler: errorHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(gatewayContextMiddleware(10 * time.Second))

	health := handlers.NewHealthHandler(deps.Config)
	app.Get("/health", health.Health)
	app.Get("/ready", health.Ready)

	RegisterV1Routes(app, deps)
	app.Use(notFoundHandler)

	return app
}

func RegisterV1Routes(app *fiber.App, deps ServiceDependencies) {
	api := app.Group("/api/v1")
	health := handlers.NewHealthHandler(deps.Config)

	routes := []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/health", Version: "v1", Handler: health.Health},
		{Method: fiber.MethodGet, Path: "/ready", Version: "v1", Handler: health.Ready},
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
	return ctx.Status(fiber.StatusNotFound).JSON(responses.Failure("not_found", "Route not found", requestID(ctx)))
}

func errorHandler(ctx *fiber.Ctx, err error) error {
	if validationErr, ok := validation.AsValidationError(err); ok {
		return ctx.Status(fiber.StatusBadRequest).JSON(responses.ValidationFailure("Request validation failed", requestID(ctx), validationErr.Fields))
	}

	code := fiber.StatusInternalServerError
	message := "Internal server error"
	errorCode := "internal_error"

	if fiberErr, ok := err.(*fiber.Error); ok {
		code = fiberErr.Code
		message = fiberErr.Message
		errorCode = "request_failed"
	}

	return ctx.Status(code).JSON(responses.Failure(errorCode, message, requestID(ctx)))
}

func requestID(ctx *fiber.Ctx) string {
	if value, ok := ctx.Locals("requestid").(string); ok {
		return value
	}

	return ctx.GetRespHeader("X-Request-ID")
}

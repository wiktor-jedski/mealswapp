package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/mealswapp/mealswapp/backend/internal/config"
)

type Dependencies struct {
	Config       config.Config
	PostgresPing func(context.Context) error
	RedisPing    func(context.Context) error
}

// envelope is the shared JSON response wrapper returned by API handlers.
//
// Implements DESIGN-017 GlobalExceptionHandler response envelope shape.
type envelope struct {
	Status    string         `json:"status"`
	RequestID string         `json:"requestId"`
	Data      map[string]any `json:"data,omitempty"`
	Error     *apiError      `json:"error,omitempty"`
}

// apiError is the user-safe error payload embedded in response envelopes.
//
// Implements DESIGN-017 ErrorMessageMapper user-safe API error shape.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewRouter constructs the Fiber application with Phase 00 middleware and routes.
//
// Implements DESIGN-010 RouteHandler, RequestValidator, and DESIGN-017 GlobalExceptionHandler wiring.
func NewRouter(deps Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: writeError,
	})

	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(timeoutMiddleware(10 * time.Second))

	app.Get("/health", health)
	app.Get("/ready", ready(deps))

	v1 := app.Group("/api/v1")
	v1.Get("/health", health)
	v1.Get("/ready", ready(deps))

	app.Get("/panic-test", func(ctx *fiber.Ctx) error {
		panic("test panic")
	})

	return app
}

// health writes the process liveness response.
//
// Implements DESIGN-010 RouteHandler liveness endpoint.
func health(ctx *fiber.Ctx) error {
	return ctx.JSON(envelope{
		Status:    "ok",
		RequestID: requestID(ctx),
		Data: map[string]any{
			"service": "mealswapp-api",
		},
	})
}

// ready returns a handler that reports dependency readiness.
//
// Implements DESIGN-010 RouteHandler readiness endpoint with dependency pings.
func ready(deps Dependencies) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		checkCtx, cancel := context.WithTimeout(ctx.UserContext(), 2*time.Second)
		defer cancel()

		checks := map[string]string{}
		status := fiber.StatusOK
		if deps.PostgresPing != nil {
			if err := deps.PostgresPing(checkCtx); err != nil {
				status = fiber.StatusServiceUnavailable
				checks["postgres"] = "unavailable"
			} else {
				checks["postgres"] = "ok"
			}
		}
		if deps.RedisPing != nil {
			if err := deps.RedisPing(checkCtx); err != nil {
				status = fiber.StatusServiceUnavailable
				checks["redis"] = "unavailable"
			} else {
				checks["redis"] = "ok"
			}
		}

		bodyStatus := "ready"
		if status != fiber.StatusOK {
			bodyStatus = "not_ready"
		}

		return ctx.Status(status).JSON(envelope{
			Status:    bodyStatus,
			RequestID: requestID(ctx),
			Data: map[string]any{
				"checks": checks,
			},
		})
	}
}

// timeoutMiddleware attaches a request-scoped deadline to Fiber context.
//
// Implements DESIGN-010 RouteHandler 10-second request deadline.
func timeoutMiddleware(timeout time.Duration) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		reqCtx, cancel := context.WithTimeout(ctx.UserContext(), timeout)
		defer cancel()
		ctx.SetUserContext(reqCtx)
		return ctx.Next()
	}
}

// writeError converts Fiber and application errors into the shared error envelope.
//
// Implements DESIGN-017 GlobalExceptionHandler consistent error envelope.
func writeError(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	return ctx.Status(code).JSON(envelope{
		Status:    "error",
		RequestID: requestID(ctx),
		Error: &apiError{
			Code:    http.StatusText(code),
			Message: message,
		},
	})
}

// requestID returns the request ID set by middleware, if present.
//
// Implements DESIGN-010 RouteHandler request context metadata.
func requestID(ctx *fiber.Ctx) string {
	if id, ok := ctx.Locals("requestid").(string); ok {
		return id
	}
	return ""
}

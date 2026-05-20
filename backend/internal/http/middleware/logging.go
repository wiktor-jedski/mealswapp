package middleware

import (
	"log/slog"
	"os"
	"time"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/gofiber/fiber/v2"
)

type RequestLoggerConfig struct {
	Logger *slog.Logger
	Now    func() time.Time
}

func DefaultRequestLoggerConfig() RequestLoggerConfig {
	return RequestLoggerConfig{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		Now:    time.Now,
	}
}

func RequestLogger(config RequestLoggerConfig) fiber.Handler {
	if config.Logger == nil {
		config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	if config.Now == nil {
		config.Now = time.Now
	}

	return func(ctx *fiber.Ctx) error {
		startedAt := config.Now()
		err := ctx.Next()
		latency := config.Now().Sub(startedAt)

		attrs := []slog.Attr{
			slog.String("request_id", requestIDFromContext(ctx)),
			slog.String("method", ctx.Method()),
			slog.String("path", ctx.Path()),
			slog.String("route", routePattern(ctx)),
			slog.Int("status", responseStatus(ctx, err)),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("ip", ctx.IP()),
		}

		if err != nil {
			code, category := safeErrorMetadata(err)
			attrs = append(attrs, slog.String("error_code", code), slog.String("error_category", category))
			config.Logger.LogAttrs(ctx.Context(), slog.LevelError, "http_request", attrs...)
			return err
		}

		config.Logger.LogAttrs(ctx.Context(), slog.LevelInfo, "http_request", attrs...)
		return nil
	}
}

func responseStatus(ctx *fiber.Ctx, err error) int {
	if err == nil {
		status := ctx.Response().StatusCode()
		if status == 0 {
			return fiber.StatusOK
		}
		return status
	}

	if appErr, ok := apperrors.As(err); ok && appErr.Status != 0 {
		return appErr.Status
	}
	if fiberErr, ok := err.(*fiber.Error); ok {
		return fiberErr.Code
	}
	return fiber.StatusInternalServerError
}

func safeErrorMetadata(err error) (string, string) {
	if appErr, ok := apperrors.As(err); ok {
		return appErr.Code, string(appErr.Category)
	}
	if fiberErr, ok := err.(*fiber.Error); ok {
		appErr := apperrors.FromFiberError(fiberErr)
		return appErr.Code, string(appErr.Category)
	}
	return "internal_error", string(apperrors.CategoryServer)
}

func routePattern(ctx *fiber.Ctx) string {
	route := ctx.Route()
	if route == nil || route.Path == "" {
		return ctx.Path()
	}
	return route.Path
}

func requestIDFromContext(ctx *fiber.Ctx) string {
	if value, ok := ctx.Locals("requestid").(string); ok {
		return value
	}
	return ctx.GetRespHeader(fiber.HeaderXRequestID)
}

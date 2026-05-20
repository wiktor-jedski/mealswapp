package middleware

import (
	"mealswapp/backend/internal/config"
	"slices"

	"github.com/gofiber/fiber/v2"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

func DefaultCORSConfig(cfg config.Config) CORSConfig {
	origins := cfg.CORSOrigins
	if len(origins) == 0 && (cfg.Environment == "" || cfg.Environment == "local" || cfg.Environment == "test") {
		origins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	return CORSConfig{
		AllowedOrigins: origins,
		AllowedMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodPatch,
			fiber.MethodDelete,
			fiber.MethodOptions,
		},
		AllowedHeaders: []string{
			fiber.HeaderAccept,
			fiber.HeaderAuthorization,
			fiber.HeaderContentType,
			"X-CSRF-Token",
			fiber.HeaderXRequestID,
		},
		AllowCredentials: true,
	}
}

func CORSHandler(config CORSConfig) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		origin := ctx.Get(fiber.HeaderOrigin)
		ctx.Append(fiber.HeaderVary, fiber.HeaderOrigin)

		if origin == "" {
			return ctx.Next()
		}

		if !slices.Contains(config.AllowedOrigins, origin) {
			if ctx.Method() == fiber.MethodOptions {
				return fiber.ErrForbidden
			}
			return ctx.Next()
		}

		ctx.Set(fiber.HeaderAccessControlAllowOrigin, origin)
		ctx.Set(fiber.HeaderAccessControlAllowMethods, joinHeaderValues(config.AllowedMethods))
		ctx.Set(fiber.HeaderAccessControlAllowHeaders, joinHeaderValues(config.AllowedHeaders))
		if config.AllowCredentials {
			ctx.Set(fiber.HeaderAccessControlAllowCredentials, "true")
		}

		if ctx.Method() == fiber.MethodOptions {
			return ctx.SendStatus(fiber.StatusNoContent)
		}

		return ctx.Next()
	}
}

func joinHeaderValues(values []string) string {
	if len(values) == 0 {
		return ""
	}

	joined := values[0]
	for _, value := range values[1:] {
		joined += ", " + value
	}

	return joined
}

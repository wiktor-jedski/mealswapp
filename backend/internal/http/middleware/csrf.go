package middleware

import (
	"mealswapp/backend/internal/http/apperrors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type CSRFConfig struct {
	ProtectedPathPrefixes []string
	ExemptPathPrefixes    []string
	TokenHeader           string
	TokenCookie           string
	SessionCookie         string
}

func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		ProtectedPathPrefixes: []string{
			"/api/v1/auth/logout",
			"/api/v1/auth/refresh",
			"/api/v1/profile",
			"/api/v1/preferences",
			"/api/v1/saved",
			"/api/v1/account",
			"/api/v1/admin",
		},
		ExemptPathPrefixes: []string{
			"/api/v1/webhooks/stripe",
		},
		TokenHeader:   "X-CSRF-Token",
		TokenCookie:   "csrf_token",
		SessionCookie: "session_id",
	}
}

func CSRFValidator(config CSRFConfig) fiber.Handler {
	if config.TokenHeader == "" {
		config.TokenHeader = "X-CSRF-Token"
	}
	if config.TokenCookie == "" {
		config.TokenCookie = "csrf_token"
	}
	if config.SessionCookie == "" {
		config.SessionCookie = "session_id"
	}

	return func(ctx *fiber.Ctx) error {
		if isSafeMethod(ctx.Method()) || hasPathPrefix(ctx.Path(), config.ExemptPathPrefixes) {
			return ctx.Next()
		}

		if !hasPathPrefix(ctx.Path(), config.ProtectedPathPrefixes) && ctx.Cookies(config.SessionCookie) == "" {
			return ctx.Next()
		}

		headerToken := strings.TrimSpace(ctx.Get(config.TokenHeader))
		cookieToken := strings.TrimSpace(ctx.Cookies(config.TokenCookie))
		if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
			return apperrors.AppError{
				Category: apperrors.CategoryAuth,
				Code:     "csrf_failed",
				Message:  "CSRF token validation failed",
				Status:   fiber.StatusForbidden,
			}
		}

		return ctx.Next()
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return true
	default:
		return false
	}
}

func hasPathPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type TLSPolicy struct {
	Environment       string
	RedirectHTTP      bool
	HSTSMaxAgeSeconds int
}

func DefaultTLSPolicy(environment string) TLSPolicy {
	production := environment != "" && environment != "local" && environment != "test"
	return TLSPolicy{
		Environment:       environment,
		RedirectHTTP:      production,
		HSTSMaxAgeSeconds: 31536000,
	}
}

func TLSEnforcer(policy TLSPolicy) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !policy.RedirectHTTP {
			return ctx.Next()
		}
		if isHTTPS(ctx) {
			ctx.Set(fiber.HeaderStrictTransportSecurity, "max-age=31536000; includeSubDomains")
			return ctx.Next()
		}
		target := "https://" + ctx.Hostname() + ctx.OriginalURL()
		return ctx.Redirect(target, fiber.StatusPermanentRedirect)
	}
}

func isHTTPS(ctx *fiber.Ctx) bool {
	if strings.EqualFold(ctx.Protocol(), "https") {
		return true
	}
	if strings.EqualFold(ctx.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return strings.EqualFold(ctx.Get("X-Forwarded-Ssl"), "on")
}

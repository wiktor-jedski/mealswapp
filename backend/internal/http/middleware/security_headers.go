package middleware

import "github.com/gofiber/fiber/v2"

type SecurityHeaders struct {
	ContentSecurityPolicy string
	FrameOptions          string
	ContentTypeOptions    string
	ReferrerPolicy        string
	PermissionsPolicy     string
	CacheControl          string
}

func DefaultSecurityHeaders() SecurityHeaders {
	return SecurityHeaders{
		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'",
		FrameOptions:          "DENY",
		ContentTypeOptions:    "nosniff",
		ReferrerPolicy:        "no-referrer",
		PermissionsPolicy:     "geolocation=(), camera=(), microphone=()",
		CacheControl:          "no-store",
	}
}

func SecurityHeadersMiddleware(config SecurityHeaders) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		ctx.Set(fiber.HeaderContentSecurityPolicy, config.ContentSecurityPolicy)
		ctx.Set(fiber.HeaderXFrameOptions, config.FrameOptions)
		ctx.Set(fiber.HeaderXContentTypeOptions, config.ContentTypeOptions)
		ctx.Set(fiber.HeaderReferrerPolicy, config.ReferrerPolicy)
		ctx.Set("Permissions-Policy", config.PermissionsPolicy)
		ctx.Set(fiber.HeaderCacheControl, config.CacheControl)
		ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

		return ctx.Next()
	}
}

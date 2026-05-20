package middleware

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSecurityHeadersMiddlewareSetsHeadersOnSuccess(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeadersMiddleware(DefaultSecurityHeaders()))
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	res, err := app.Test(newRequest(t, http.MethodGet, "/ok"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	assertSecurityHeaders(t, res)
}

func TestSecurityHeadersMiddlewareSetsHeadersOnError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return ctx.Status(fiber.StatusInternalServerError).JSON(map[string]string{"error": "failed"})
		},
	})
	app.Use(SecurityHeadersMiddleware(DefaultSecurityHeaders()))
	app.Get("/fail", func(ctx *fiber.Ctx) error {
		return fiber.ErrInternalServerError
	})

	res, err := app.Test(newRequest(t, http.MethodGet, "/fail"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	assertSecurityHeaders(t, res)
}

func assertSecurityHeaders(t *testing.T, res *http.Response) {
	t.Helper()

	expected := map[string]string{
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"X-Frame-Options":         "DENY",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"Permissions-Policy":      "geolocation=(), camera=(), microphone=()",
		"Cache-Control":           "no-store",
	}

	for header, value := range expected {
		if got := res.Header.Get(header); got != value {
			t.Fatalf("expected %s %q, got %q", header, value, got)
		}
	}
}

func newRequest(t *testing.T, method string, path string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

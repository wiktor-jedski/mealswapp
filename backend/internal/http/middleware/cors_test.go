package middleware

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestCORSHandlerAllowsConfiguredOrigin(t *testing.T) {
	app := newCORSApp(CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{fiber.MethodGet, fiber.MethodOptions},
		AllowedHeaders:   []string{fiber.HeaderContentType, fiber.HeaderAuthorization},
		AllowCredentials: true,
	})

	req := newRequest(t, http.MethodGet, "/ok")
	req.Header.Set(fiber.HeaderOrigin, "https://app.example.com")

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "https://app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowCredentials); got != "true" {
		t.Fatalf("expected credentials allowed, got %q", got)
	}
	if got := res.Header.Get(fiber.HeaderVary); got != fiber.HeaderOrigin {
		t.Fatalf("expected Vary Origin, got %q", got)
	}
}

func TestCORSHandlerDoesNotCredentialUnlistedOrigin(t *testing.T) {
	app := newCORSApp(CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{fiber.MethodGet, fiber.MethodOptions},
		AllowedHeaders:   []string{fiber.HeaderContentType},
		AllowCredentials: true,
	})

	req := newRequest(t, http.MethodGet, "/ok")
	req.Header.Set(fiber.HeaderOrigin, "https://evil.example.com")

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowCredentials); got != "" {
		t.Fatalf("expected no credentials header, got %q", got)
	}
}

func TestCORSHandlerHandlesPreflight(t *testing.T) {
	app := newCORSApp(CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodOptions},
		AllowedHeaders:   []string{fiber.HeaderContentType, fiber.HeaderAuthorization},
		AllowCredentials: true,
	})

	req := newRequest(t, http.MethodOptions, "/ok")
	req.Header.Set(fiber.HeaderOrigin, "https://app.example.com")
	req.Header.Set(fiber.HeaderAccessControlRequestMethod, fiber.MethodPost)

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", res.StatusCode)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "https://app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowMethods); got != "GET, POST, OPTIONS" {
		t.Fatalf("expected configured methods, got %q", got)
	}
}

func TestCORSHandlerRejectsUnlistedPreflight(t *testing.T) {
	app := fiber.New()
	app.Use(CORSHandler(CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{fiber.MethodGet, fiber.MethodOptions},
		AllowedHeaders:   []string{fiber.HeaderContentType},
		AllowCredentials: true,
	}))
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})

	req := newRequest(t, http.MethodOptions, "/ok")
	req.Header.Set(fiber.HeaderOrigin, "https://evil.example.com")
	req.Header.Set(fiber.HeaderAccessControlRequestMethod, fiber.MethodGet)

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", res.StatusCode)
	}
	if got := res.Header.Get(fiber.HeaderAccessControlAllowCredentials); got != "" {
		t.Fatalf("expected no credentials header, got %q", got)
	}
}

func newCORSApp(config CORSConfig) *fiber.App {
	app := fiber.New()
	app.Use(CORSHandler(config))
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})
	return app
}

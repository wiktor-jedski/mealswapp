package middleware

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestTLSEnforcerRedirectsHTTPInProduction(t *testing.T) {
	app := fiber.New()
	app.Use(TLSEnforcer(DefaultTLSPolicy("production")))
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})

	req := newRequest(t, http.MethodGet, "/ok?x=1")
	req.Host = "api.example.com"
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("expected 308 redirect, got %d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "https://api.example.com/ok?x=1" {
		t.Fatalf("unexpected redirect location %q", location)
	}
}

func TestTLSEnforcerAllowsForwardedHTTPSAndSetsHSTS(t *testing.T) {
	app := fiber.New()
	app.Use(TLSEnforcer(DefaultTLSPolicy("production")))
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})

	req := newRequest(t, http.MethodGet, "/ok")
	req.Header.Set("X-Forwarded-Proto", "https")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if hsts := res.Header.Get(fiber.HeaderStrictTransportSecurity); hsts != "max-age=31536000; includeSubDomains" {
		t.Fatalf("unexpected HSTS header %q", hsts)
	}
}

func TestTLSEnforcerExemptsLocalAndTest(t *testing.T) {
	for _, env := range []string{"", "local", "test"} {
		app := fiber.New()
		app.Use(TLSEnforcer(DefaultTLSPolicy(env)))
		app.Get("/ok", func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(fiber.StatusOK)
		})

		res, err := app.Test(newRequest(t, http.MethodGet, "/ok"))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected local/test 200 for env %q, got %d", env, res.StatusCode)
		}
	}
}

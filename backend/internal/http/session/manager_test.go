package session

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestManagerSetsLocalCookieFlags(t *testing.T) {
	app := newSessionTestApp(NewManager(Config{Environment: "local"}))

	res, err := app.Test(newSessionRequest(t, http.MethodPost, "/login"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	cookies := strings.Join(res.Header.Values("Set-Cookie"), "\n")
	if !strings.Contains(cookies, "access_token=access") || !strings.Contains(cookies, "refresh_token=refresh") {
		t.Fatalf("expected auth cookies, got %s", cookies)
	}
	if !strings.Contains(cookies, "HttpOnly") || !strings.Contains(cookies, "SameSite=Lax") {
		t.Fatalf("expected local HttpOnly Lax cookies, got %s", cookies)
	}
	if strings.Contains(cookies, "Secure") {
		t.Fatalf("expected local cookies without Secure flag, got %s", cookies)
	}
}

func TestManagerSetsProductionCookieFlags(t *testing.T) {
	app := newSessionTestApp(NewManager(Config{Environment: "production", Domain: "mealswapp.example"}))

	res, err := app.Test(newSessionRequest(t, http.MethodPost, "/login"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	cookies := strings.Join(res.Header.Values("Set-Cookie"), "\n")
	if !strings.Contains(cookies, "secure") || !strings.Contains(cookies, "HttpOnly") || !strings.Contains(cookies, "SameSite=Strict") {
		t.Fatalf("expected production secure strict cookies, got %s", cookies)
	}
	if !strings.Contains(cookies, "domain=mealswapp.example") {
		t.Fatalf("expected configured domain, got %s", cookies)
	}
}

func TestManagerClearsAuthCookies(t *testing.T) {
	manager := NewManager(Config{Environment: "production"})
	app := fiber.New()
	app.Post("/logout", func(ctx *fiber.Ctx) error {
		manager.ClearAuthCookies(ctx)
		return ctx.SendStatus(fiber.StatusOK)
	})

	res, err := app.Test(newSessionRequest(t, http.MethodPost, "/logout"))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	cookies := strings.Join(res.Header.Values("Set-Cookie"), "\n")
	if !strings.Contains(cookies, "access_token=;") || !strings.Contains(cookies, "refresh_token=;") {
		t.Fatalf("expected cleared auth cookies, got %s", cookies)
	}
}

func newSessionTestApp(manager Manager) *fiber.App {
	app := fiber.New()
	app.Post("/login", func(ctx *fiber.Ctx) error {
		manager.SetAuthCookies(
			ctx,
			"access",
			"refresh",
			time.Date(2026, 5, 19, 12, 15, 0, 0, time.UTC),
			time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
		)
		return ctx.SendStatus(fiber.StatusOK)
	})
	return app
}

func newSessionRequest(t *testing.T, method string, path string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

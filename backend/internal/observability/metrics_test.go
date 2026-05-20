package observability

import (
	"io"
	"mealswapp/backend/internal/http/apperrors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestMetricsCollectorRecordsRequestLatencyAndErrors(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	collector := NewMetricsCollector()
	collector.now = func() time.Time {
		now = now.Add(15 * time.Millisecond)
		return now
	}

	app := fiber.New(fiber.Config{ErrorHandler: func(ctx *fiber.Ctx, err error) error {
		return ctx.SendStatus(statusFromError(err))
	}})
	app.Use(collector.Middleware())
	app.Get("/ok", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})
	app.Get("/fail", func(ctx *fiber.Ctx) error {
		return apperrors.DependencyUnavailable("database unavailable")
	})
	app.Get("/metrics", collector.Handler)

	mustRequest(t, app, http.MethodGet, "/ok", http.StatusOK)
	mustRequest(t, app, http.MethodGet, "/fail", http.StatusServiceUnavailable)
	res := mustRequest(t, app, http.MethodGet, "/metrics", http.StatusOK)
	defer res.Body.Close()

	body := readBody(t, res)
	assertMetricContains(t, body, "mealswapp_http_requests_total 2")
	assertMetricContains(t, body, "mealswapp_http_errors_total 1")
	assertMetricContains(t, body, "mealswapp_http_request_latency_ms_sum 30")
}

func TestMetricsCollectorExposesReadinessAndDependencyStatus(t *testing.T) {
	collector := NewMetricsCollector()
	collector.SetReadiness(false, map[string]bool{
		"database": false,
		"redis":    true,
	})

	app := fiber.New()
	app.Get("/metrics", collector.Handler)

	res := mustRequest(t, app, http.MethodGet, "/metrics", http.StatusOK)
	defer res.Body.Close()

	body := readBody(t, res)
	assertMetricContains(t, body, "mealswapp_health_ready 0")
	assertMetricContains(t, body, `mealswapp_dependency_status{name="database"} 0`)
	assertMetricContains(t, body, `mealswapp_dependency_status{name="redis"} 1`)
}

func assertMetricContains(t *testing.T, body string, metric string) {
	t.Helper()

	if !strings.Contains(body, metric) {
		t.Fatalf("expected metric %q in body:\n%s", metric, body)
	}
}

func mustRequest(t *testing.T, app *fiber.App, method string, path string, status int) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != status {
		res.Body.Close()
		t.Fatalf("expected status %d, got %d", status, res.StatusCode)
	}
	return res
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

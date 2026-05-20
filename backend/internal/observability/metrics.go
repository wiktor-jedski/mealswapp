package observability

import (
	"fmt"
	"mealswapp/backend/internal/http/apperrors"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type MetricsCollector struct {
	mu               sync.Mutex
	requestsTotal    int64
	errorsTotal      int64
	latencyTotalMS   int64
	readinessHealthy bool
	dependencies     map[string]bool
	now              func() time.Time
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		readinessHealthy: true,
		dependencies:     make(map[string]bool),
		now:              time.Now,
	}
}

func (collector *MetricsCollector) Middleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		startedAt := collector.now()
		err := ctx.Next()
		latency := collector.now().Sub(startedAt)
		status := ctx.Response().StatusCode()
		if err != nil {
			status = statusFromError(err)
		}

		collector.RecordRequest(status, latency)
		return err
	}
}

func (collector *MetricsCollector) RecordRequest(status int, latency time.Duration) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	collector.requestsTotal++
	collector.latencyTotalMS += latency.Milliseconds()
	if status >= 500 {
		collector.errorsTotal++
	}
}

func (collector *MetricsCollector) SetReadiness(healthy bool, dependencies map[string]bool) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	collector.readinessHealthy = healthy
	collector.dependencies = make(map[string]bool, len(dependencies))
	for name, healthy := range dependencies {
		collector.dependencies[name] = healthy
	}
}

func (collector *MetricsCollector) Handler(ctx *fiber.Ctx) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	var builder strings.Builder
	builder.WriteString("# TYPE mealswapp_http_requests_total counter\n")
	builder.WriteString(fmt.Sprintf("mealswapp_http_requests_total %d\n", collector.requestsTotal))
	builder.WriteString("# TYPE mealswapp_http_errors_total counter\n")
	builder.WriteString(fmt.Sprintf("mealswapp_http_errors_total %d\n", collector.errorsTotal))
	builder.WriteString("# TYPE mealswapp_http_request_latency_ms_sum counter\n")
	builder.WriteString(fmt.Sprintf("mealswapp_http_request_latency_ms_sum %d\n", collector.latencyTotalMS))
	builder.WriteString("# TYPE mealswapp_health_ready gauge\n")
	builder.WriteString(fmt.Sprintf("mealswapp_health_ready %d\n", boolGauge(collector.readinessHealthy)))
	builder.WriteString("# TYPE mealswapp_dependency_status gauge\n")
	for name, healthy := range collector.dependencies {
		builder.WriteString(fmt.Sprintf("mealswapp_dependency_status{name=%q} %d\n", name, boolGauge(healthy)))
	}

	ctx.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return ctx.SendString(builder.String())
}

func boolGauge(value bool) int {
	if value {
		return 1
	}
	return 0
}

func statusFromError(err error) int {
	if appErr, ok := apperrors.As(err); ok && appErr.Status != 0 {
		return appErr.Status
	}
	if fiberErr, ok := err.(*fiber.Error); ok {
		return fiberErr.Code
	}
	return fiber.StatusInternalServerError
}

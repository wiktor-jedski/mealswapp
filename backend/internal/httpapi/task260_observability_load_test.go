package httpapi

// Implements DESIGN-014 MetricsCollector/LogAggregator Task 260 admin and responsiveness gate.

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type task260NoopAudit struct{}

func (task260NoopAudit) Audit(context.Context, security.AuditLogEntry) error { return nil }

type task260BlockingTelemetrySink struct{ release chan struct{} }

func (s task260BlockingTelemetrySink) RecordMetric(context.Context, observability.MetricPoint) error {
	<-s.release
	return nil
}

func (s task260BlockingTelemetrySink) Log(context.Context, observability.LogEvent) error {
	<-s.release
	return nil
}

func TestTask260AuditFailureHasDistinctBoundedAdminOutcome(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	sink := &observability.MemorySink{}
	telemetry := observability.NewAdminExternalTelemetry(sink, sink)
	rule := RateLimitRule{Scope: "user", MaxRequests: 2, WindowSeconds: 60}
	controller := NewAdminController(&task258FailingAdminAudit{}, AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/task-260-audit", RateLimit: &rule, AuditAction: "manual_update", EntityType: "food_item",
		Validate: func(ctx *fiber.Ctx) error { return ctx.Next() },
		Mutation: func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{Data: map[string]any{"status": "updated"}}, nil
		},
	}).WithTelemetry(telemetry)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: &auditSink{}, Logs: sink, Metrics: sink, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/task-260-audit", nil)
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.StatusCode)
	}
	metrics, logs := sink.Snapshot()
	foundMetric, foundLog := false, false
	for _, point := range metrics {
		if point.Name == observability.MetricAdminMutationOutcomes && len(point.Labels) == 2 && point.Labels["operation"] == "manual_update" && point.Labels["outcome"] == "audit_failed" {
			foundMetric = true
		}
	}
	for _, event := range logs {
		if event.Message == "admin_mutation_outcome" && event.Fields["outcome"] == "audit_failed" {
			foundLog = true
		}
	}
	if !foundMetric || !foundLog {
		t.Fatalf("audit failure metric=%t log=%t metrics=%+v logs=%+v", foundMetric, foundLog, metrics, logs)
	}
}

func TestTask260TelemetryLoadLeavesSearchAndAuthRoutesResponsive(t *testing.T) {
	sink := &observability.MemorySink{}
	telemetry := observability.NewAdminExternalTelemetry(sink, sink)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Audit: task260NoopAudit{}, Logs: sink, Metrics: sink, Routes: []RouteDefinition{{
		Method: fiber.MethodGet, Path: "/search", Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusOK) },
	}}})

	const requests = 64
	done := make(chan error, requests*2)
	var load sync.WaitGroup
	load.Add(1)
	go func() {
		defer load.Done()
		for range 512 {
			telemetry.ProviderCall(context.Background(), "usda", "success", time.Millisecond)
			telemetry.AdminMutation(context.Background(), "manual_create", "succeeded")
		}
	}()
	for _, path := range []string{"/api/v1/search", "/api/v1/auth/csrf-token"} {
		for range requests {
			go func(path string) {
				response, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
				if err == nil {
					response.Body.Close()
					if response.StatusCode != fiber.StatusOK {
						err = fiber.ErrInternalServerError
					}
				}
				done <- err
			}(path)
		}
	}
	deadline := time.After(5 * time.Second)
	for range requests * 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-deadline:
			t.Fatal("search or auth became unresponsive under telemetry load")
		}
	}
	load.Wait()
}

func TestTask260BlockingTelemetrySinkCannotHoldAdminRequest(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	blocking := task260BlockingTelemetrySink{release: make(chan struct{})}
	defer close(blocking.release)
	rule := RateLimitRule{Scope: "user", MaxRequests: 2, WindowSeconds: 60}
	controller := NewAdminController(&adminAuditCoordinator{}, AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/task-260-blocking-sink", RateLimit: &rule, AuditAction: "manual_update", EntityType: "food_item",
		Validate: func(ctx *fiber.Ctx) error { return ctx.Next() },
		Mutation: func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{Data: map[string]any{"status": "updated"}}, nil
		},
	}).WithTelemetry(observability.NewAdminExternalTelemetry(blocking, blocking))
	routerSink := &observability.MemorySink{}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: &auditSink{}, Logs: routerSink, Metrics: routerSink, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/task-260-blocking-sink", nil)
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	done := make(chan error, 1)
	go func() {
		response, err := app.Test(request)
		if err == nil {
			response.Body.Close()
			if response.StatusCode != fiber.StatusOK {
				err = fiber.ErrInternalServerError
			}
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(750 * time.Millisecond):
		t.Fatal("blocking telemetry sink held the admin request")
	}
}

package httpapi

// Implements DESIGN-009 AdminController and DESIGN-014 MetricsCollector operational integration gate.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type task258FailingAdminAudit struct {
	entry      repository.AdminAuditEntry
	mutated    bool
	rolledBack bool
}

// WithMutationAudit models the production fail-closed transaction boundary for HTTP integration verification.
// Implements DESIGN-009 AdminController audit_write_failed state.
func (a *task258FailingAdminAudit) WithMutationAudit(_ context.Context, entry repository.AdminAuditEntry, mutate func(repository.AdminMutationExecutor) (repository.AdminAuditChanges, error)) error {
	a.entry = entry
	if _, err := mutate(nil); err != nil {
		return err
	}
	a.mutated = true
	a.rolledBack = true
	return errors.Join(repository.ErrAdminAuditPersistence, errors.New("database host secret.internal unavailable"))
}

// TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized proves that an
// admin mutation cannot report success when its audit fails, while operators
// retain request-correlated logs and bounded metrics without sensitive causes.
func TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	adminAudit := &task258FailingAdminAudit{}
	securityAudit := &auditSink{}
	telemetry := &observability.MemorySink{}
	rule := RateLimitRule{Scope: "user", MaxRequests: 2, WindowSeconds: 60}
	controller := NewAdminController(adminAudit, AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/task-258-fixture", RateLimit: &rule,
		AuditAction: "fixture.update", EntityType: "fixture",
		Validate: func(ctx *fiber.Ctx) error { return ctx.Next() },
		Mutation: func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{Data: map[string]any{"unsafeSuccess": true}, Audit: repository.AdminAuditChanges{After: []byte(`{"status":"published"}`)}}, nil
		},
	})
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: securityAudit, Logs: telemetry, Metrics: telemetry, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/task-258-fixture", strings.NewReader(`{"providerPayload":"api-key-secret"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", token)
	request.Header.Set("X-Request-ID", "client-spoofed-trace")
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	envelope := decodeEnvelope(t, response.Body)
	if response.StatusCode != fiber.StatusServiceUnavailable || envelope.Error == nil || envelope.Error.Code != "dependency_unavailable" || !envelope.Error.Retryable {
		t.Fatalf("audit failure response = %d %+v", response.StatusCode, envelope)
	}
	if !adminAudit.mutated || !adminAudit.rolledBack || adminAudit.entry.RequestID != envelope.RequestID || !isServerRequestID(envelope.RequestID, "client-spoofed-trace") {
		t.Fatalf("transaction/correlation audit=%+v response=%+v", adminAudit, envelope)
	}

	metrics, logs := telemetry.Snapshot()
	if !task258HasHTTPMetric(metrics, "http_response_total", "/api/v1/admin/task-258-fixture", fiber.StatusServiceUnavailable) ||
		!task258HasHTTPMetric(metrics, "http_error_total", "/api/v1/admin/task-258-fixture", fiber.StatusServiceUnavailable) {
		t.Fatalf("missing bounded failure metrics: %+v", metrics)
	}
	if len(logs) == 0 || logs[len(logs)-1].RequestID != envelope.RequestID || logs[len(logs)-1].Level != "error" {
		t.Fatalf("missing correlated error log: %+v", logs)
	}
	if len(securityAudit.entries) == 0 || securityAudit.entries[len(securityAudit.entries)-1].RequestID != envelope.RequestID {
		t.Fatalf("missing correlated security audit: %+v", securityAudit.entries)
	}
	payload, err := json.Marshal(struct {
		Envelope Envelope
		Metrics  []observability.MetricPoint
		Logs     []observability.LogEvent
		Audits   any
	}{Envelope: envelope, Metrics: metrics, Logs: logs, Audits: securityAudit.entries})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"api-key-secret", "secret.internal", "client-spoofed-trace", "unsafeSuccess"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("sensitive value %q reached response or telemetry: %s", forbidden, payload)
		}
	}
}

// task258HasHTTPMetric accepts only the fixed route and numeric status labels emitted by the gateway.
// Implements DESIGN-014 MetricsCollector low-cardinality HTTP metrics.
func task258HasHTTPMetric(points []observability.MetricPoint, name, route string, status int) bool {
	for _, point := range points {
		if point.Name == name && len(point.Labels) == 2 && point.Labels["route"] == route && point.Labels["status"] == strconv.Itoa(status) {
			return true
		}
	}
	return false
}

package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/handlers"
)

func TestAdminControllerAllowsAdminSummary(t *testing.T) {
	auth := newFakeAuthService()
	auth.user.Role = "admin"
	token := auth.issueTokens().AccessToken
	summary := fakeAdminSummaryService{
		result: handlers.AdminSummary{
			PendingImports:   3,
			PendingItems:     4,
			ActiveUsers:      12,
			RecentAuditCount: 5,
			GeneratedAt:      time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC),
		},
	}
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		AuthService:         auth,
		AdminSummaryService: summary,
	})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/summary", "", token, false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected admin summary 200, got %d", res.StatusCode)
	}
	data := dataMap(t, decodeEnvelope(t, res).Data)
	if data["pendingImports"] != float64(3) || data["activeUsers"] != float64(12) {
		t.Fatalf("unexpected admin summary: %#v", data)
	}
}

func TestAdminControllerRejectsNonAdminAndUnauthenticatedUsers(t *testing.T) {
	auth := newFakeAuthService()
	token := auth.issueTokens().AccessToken
	app := NewRouter(ServiceDependencies{
		Config:      config.Config{Environment: "test"},
		AuthService: auth,
	})

	forbidden := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/summary", "", token, false)
	defer forbidden.Body.Close()
	if forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("expected non-admin 403, got %d", forbidden.StatusCode)
	}
	forbiddenPayload := decodeEnvelope(t, forbidden)
	if forbiddenPayload.Error == nil || forbiddenPayload.Error.Code != "forbidden" || forbiddenPayload.Error.Message != "Forbidden" {
		t.Fatalf("expected audit-safe forbidden envelope, got %#v", forbiddenPayload.Error)
	}

	unauthorized := performJSONRequest(t, app, http.MethodGet, "/api/v1/admin/summary", "", "", false)
	defer unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated 401, got %d", unauthorized.StatusCode)
	}
	unauthorizedPayload := decodeEnvelope(t, unauthorized)
	if unauthorizedPayload.Error == nil || unauthorizedPayload.Error.Code != "unauthorized" {
		t.Fatalf("expected unauthorized envelope, got %#v", unauthorizedPayload.Error)
	}
}

type fakeAdminSummaryService struct {
	result handlers.AdminSummary
}

func (service fakeAdminSummaryService) Summary(ctx context.Context, admin handlers.AdminContext) (handlers.AdminSummary, error) {
	return service.result, nil
}

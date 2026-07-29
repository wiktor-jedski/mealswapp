package httpapi

// Implements DESIGN-009 AdminController catalog export authorization and safe-query verification.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type catalogExportService struct {
	includeDeleted []bool
	content        string
}

func (s *catalogExportService) Write(_ context.Context, output io.Writer, includeDeleted bool) (int, error) {
	s.includeDeleted = append(s.includeDeleted, includeDeleted)
	_, err := io.WriteString(output, s.content)
	return 1, err
}

func TestCatalogExportRouteIsAdminOnlyAndDefaultsToActive(t *testing.T) {
	cfg := testConfig()
	adminAuth, adminCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	userAuth, userCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleUser), nil)
	logs := &observability.MemorySink{}
	service := &catalogExportService{content: `{"schema":"mealswapp.global-catalog.v1","items":[]}` + "\n"}
	controller := NewGlobalCatalogExportAdminController(service)

	request := func(auth *JWTAuthenticator, cookies []*http.Cookie, path string) (int, string) {
		t.Helper()
		app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Logs: logs, Routes: controller.Routes()})
		req := httptest.NewRequest(fiber.MethodGet, path, nil)
		addCookies(req, cookies)
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, string(body)
	}
	if status, _ := request(nil, nil, "/api/v1/admin/catalog-export"); status != fiber.StatusUnauthorized {
		t.Fatalf("anonymous status = %d", status)
	}
	if status, _ := request(userAuth, userCookies, "/api/v1/admin/catalog-export"); status != fiber.StatusForbidden {
		t.Fatalf("ordinary user status = %d", status)
	}
	if status, body := request(adminAuth, adminCookies, "/api/v1/admin/catalog-export"); status != fiber.StatusOK || body != service.content {
		t.Fatalf("admin response = %d %q", status, body)
	}
	if status, _ := request(adminAuth, adminCookies, "/api/v1/admin/catalog-export?includeDeleted=true"); status != fiber.StatusOK {
		t.Fatalf("includeDeleted status = %d", status)
	}
	if len(service.includeDeleted) != 2 || service.includeDeleted[0] || !service.includeDeleted[1] {
		t.Fatalf("includeDeleted calls = %#v", service.includeDeleted)
	}
	for _, path := range []string{
		"/api/v1/admin/catalog-export?includeDeleted=yes",
		"/api/v1/admin/catalog-export?format=csv",
		"/api/v1/admin/catalog-export?includeDeleted=true&includeDeleted=false",
	} {
		if status, _ := request(adminAuth, adminCookies, path); status != fiber.StatusBadRequest {
			t.Fatalf("%s status = %d", path, status)
		}
	}
	for _, event := range logs.Logs {
		encoded := event.Message
		for _, value := range event.Fields {
			encoded += toString(value)
		}
		for _, forbidden := range []string{"cookie", "token", "email@", "private item", "catalog payload"} {
			if strings.Contains(strings.ToLower(encoded), forbidden) {
				t.Fatalf("unsafe log = %+v", event)
			}
		}
	}
}

func TestCatalogExportRouteShapeHasNoMutationOrAccountExportControls(t *testing.T) {
	routes := NewGlobalCatalogExportAdminController(&catalogExportService{}).Routes()
	if len(routes) != 1 {
		t.Fatalf("routes = %#v", routes)
	}
	route := routes[0]
	if route.Method != fiber.MethodGet || route.Path != "/admin/catalog-export" || !route.RequiresAuth || !route.RequiresAdmin ||
		route.RequiresCSRF || route.RequiresAudit {
		t.Fatalf("unsafe export route = %#v", route)
	}
}

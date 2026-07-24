package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/externaldata"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 AdminController and ExternalSearchProxy HTTP authorization and read-only verification.

type externalSearchServiceStub struct {
	queries  []externaldata.ExternalSearchQuery
	response externaldata.ExternalSearchResponse
	err      error
}

func (s *externalSearchServiceStub) Search(_ context.Context, query externaldata.ExternalSearchQuery) (externaldata.ExternalSearchResponse, error) {
	s.queries = append(s.queries, query)
	return s.response, s.err
}

func TestAdminExternalSearchForbidsNonAdminAndDoesNotAuditRead(t *testing.T) {
	cfg := testConfig()
	adminAuth, adminCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	userAuth, userCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleUser), nil)
	audit := &adminAuditCoordinator{}
	service := &externalSearchServiceStub{response: externaldata.ExternalSearchResponse{
		Candidates: []externaldata.ExternalCandidate{{
			Provider: "usda", ExternalID: "1", Name: "Apple", PhysicalState: repository.PhysicalStateSolid,
			MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 2, Fat: 3}, Micronutrients: repository.MicroValues{}, Warnings: []string{},
		}}, Warnings: []externaldata.ExternalDataWarning{}, Page: 2,
	}}
	controller := NewAdminController(audit).WithExternalSearch(service, nil)

	request := func(auth *JWTAuthenticator, cookies []*http.Cookie) (int, Envelope) {
		t.Helper()
		app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Routes: controller.Routes()})
		req := httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=apple&provider=all&page=2", nil)
		addCookies(req, cookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, decodeEnvelope(t, resp.Body)
	}

	status, body := request(userAuth, userCookies)
	if status != fiber.StatusForbidden || body.Error == nil || body.Error.Code != "forbidden" || len(service.queries) != 0 {
		t.Fatalf("non-admin response=%d %#v calls=%d", status, body, len(service.queries))
	}
	status, body = request(adminAuth, adminCookies)
	if status != fiber.StatusOK || body.Status != "ok" || len(service.queries) != 1 {
		t.Fatalf("admin response=%d %#v calls=%d", status, body, len(service.queries))
	}
	if got := service.queries[0]; got.Query != "apple" || got.Provider != "all" || got.Page != 2 || got.PageSize != 0 {
		t.Fatalf("typed handoff = %#v", got)
	}
	if audit.committed != 0 || len(audit.entries) != 0 {
		t.Fatalf("search wrote admin audit = %#v", audit.entries)
	}
}

func TestAdminExternalSearchSerializesBoundedWarningsWithContractKeys(t *testing.T) {
	cfg := testConfig()
	auth, cookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	service := &externalSearchServiceStub{response: externaldata.ExternalSearchResponse{
		Candidates: []externaldata.ExternalCandidate{},
		Warnings: []externaldata.ExternalDataWarning{
			{Provider: "usda", Code: externaldata.WarningUnavailable, Message: externaldata.WarningUnavailable},
			{Provider: "openfoodfacts", Code: externaldata.WarningRateLimited, Message: externaldata.WarningRateLimited},
		},
		Page: 1,
	}}
	controller := NewAdminController(nil).WithExternalSearch(service, nil)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Routes: controller.Routes()})
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=apple&provider=all&page=1", nil)
	addCookies(req, cookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := decodeEnvelope(t, resp.Body)
	warnings, ok := body.Data["warnings"].([]any)
	if resp.StatusCode != fiber.StatusOK || !ok || len(warnings) != 2 {
		t.Fatalf("warning response=%d %#v", resp.StatusCode, body.Data["warnings"])
	}
	for i, want := range []map[string]string{
		{"provider": "usda", "code": externaldata.WarningUnavailable, "message": externaldata.WarningUnavailable},
		{"provider": "openfoodfacts", "code": externaldata.WarningRateLimited, "message": externaldata.WarningRateLimited},
	} {
		warning, ok := warnings[i].(map[string]any)
		if !ok || len(warning) != len(want) {
			t.Fatalf("warning[%d] = %#v", i, warnings[i])
		}
		for key, value := range want {
			if warning[key] != value {
				t.Fatalf("warning[%d][%q] = %#v, want %q", i, key, warning[key], value)
			}
		}
	}
}

func TestAdminExternalSearchMapsDependencyFailureAndRequiresValidatedInput(t *testing.T) {
	service := &externalSearchServiceStub{err: errors.New("database detail must stay private")}
	controller := NewAdminController(nil).WithExternalSearch(service, nil)
	ctxApp := fiber.New(fiber.Config{ErrorHandler: writeError})
	ctxApp.Get("/direct", controller.SearchExternal)
	resp, err := ctxApp.Test(httptest.NewRequest(fiber.MethodGet, "/direct", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("missing validation status = %d", resp.StatusCode)
	}

	cfg := testConfig()
	auth, cookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Routes: controller.Routes()})
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=apple&provider=usda&page=1", nil)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := decodeEnvelope(t, resp.Body)
	if resp.StatusCode != fiber.StatusServiceUnavailable || body.Error == nil || body.Error.Code != "dependency_unavailable" || body.Error.Message == service.err.Error() {
		t.Fatalf("dependency response=%d %#v", resp.StatusCode, body)
	}

	service.err = context.Canceled
	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=apple&provider=usda&page=1", nil)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("canceled response status = %d", resp.StatusCode)
	}

	unavailable := NewAdminController(nil).WithExternalSearch(nil, nil)
	app = mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Routes: unavailable.Routes()})
	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=apple&provider=usda&page=1", nil)
	addCookies(req, cookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("missing service status = %d", resp.StatusCode)
	}
}

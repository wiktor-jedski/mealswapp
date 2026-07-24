package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/dataimporter"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 DataImporter HTTP confirmation verification.

type curatedImportServiceStub struct {
	adminID uuid.UUID
	key     string
	req     dataimporter.Request
	result  dataimporter.Result
	err     error
	calls   int
}

type curatedImportInvalidatorStub struct{ calls int }

func (s *curatedImportInvalidatorStub) Invalidate() { s.calls++ }

func (s *curatedImportServiceStub) Confirm(_ context.Context, _ repository.AdminMutationExecutor, adminID uuid.UUID, key string, req dataimporter.Request) (dataimporter.Result, error) {
	s.calls++
	s.adminID, s.key, s.req = adminID, key, req
	return s.result, s.err
}

func TestCuratedImportHTTPCommitsSafeResponseAndAudit(t *testing.T) {
	cfg := testConfig()
	adminID, foodID, importID := uuid.New(), uuid.New(), uuid.New()
	auth, cookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	audit := &adminAuditCoordinator{}
	invalidator := &curatedImportInvalidatorStub{}
	service := &curatedImportServiceStub{result: dataimporter.Result{ImportID: importID, FoodItemID: foodID, Name: "Curated tofu", PhysicalState: repository.PhysicalStateSolid}}
	controller := NewCuratedImportAdminController(audit, service, invalidator)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Audit: &auditSink{}, Routes: controller.Routes()})
	csrf, csrfCookies := fetchCSRFToken(t, app)
	body := `{"sourceProvider":"usda","externalId":"fdc-1","name":"Curated tofu","physicalState":"solid","macrosPer100":{"protein":18,"carbohydrates":4,"fat":8},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/imports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Idempotency-Key", "optional-natural-key")
	addCookies(req, cookies)
	addCookies(req, csrfCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)
	if resp.StatusCode != fiber.StatusCreated || service.calls != 1 || service.adminID != adminID || service.req.SourceProvider != "usda" || len(audit.entries) != 1 || audit.entries[0].EntityID == nil || *audit.entries[0].EntityID != foodID || invalidator.calls != 1 {
		t.Fatalf("status=%d envelope=%+v service=%+v audit=%+v", resp.StatusCode, envelope, service, audit.entries)
	}
	if strings.Contains(string(audit.entries[0].After), "usda") || string(audit.entries[0].After) != `{"physicalState":"solid","status":"imported"}` {
		t.Fatalf("unsafe audit=%s", audit.entries[0].After)
	}
	service.result.Replayed = true
	replayRequest := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/imports", strings.NewReader(body))
	replayRequest.Header.Set("Content-Type", "application/json")
	replayRequest.Header.Set("X-CSRF-Token", csrf)
	addCookies(replayRequest, cookies)
	addCookies(replayRequest, csrfCookies)
	replayResponse, err := app.Test(replayRequest)
	if err != nil {
		t.Fatal(err)
	}
	replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusCreated || len(audit.entries) != 1 || invalidator.calls != 1 {
		t.Fatalf("replay status=%d audits=%d invalidations=%d", replayResponse.StatusCode, len(audit.entries), invalidator.calls)
	}
}

func TestCuratedImportHTTPValidationAndConflictMapping(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{dataimporter.ErrIdempotencyConflict, "idempotency_key_conflict"},
		{dataimporter.ErrProviderConflict, "provider_identity_conflict"},
		{dataimporter.ErrNameConfirmation, "name_conflict_confirmation_required"},
	} {
		var appErr AppError
		if !errors.As(curatedImportError(test.err), &appErr) || appErr.HTTPStatus != fiber.StatusConflict || appErr.Code != test.code {
			t.Fatalf("error=%v mapped=%+v", test.err, appErr)
		}
	}
	app := fiber.New()
	app.Post("/", validateCuratedImport, func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) })
	for _, body := range []string{
		`{"name":"Missing fields"}`,
		`{"name":"Food","physicalState":"solid","macrosPer100":{"protein":0,"carbohydrates":0,"fat":0},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[],"unknown":true}`,
		`{"name":"Food","name":"Other","physicalState":"solid","macrosPer100":{"protein":0,"carbohydrates":0,"fat":0},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`,
	} {
		req := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 400 {
			resp.Body.Close()
			t.Fatalf("invalid body accepted: %s", body)
		}
		resp.Body.Close()
	}
}

func TestCuratedImportHTTPNormalizesAndRejectsAdversarialDraftsBeforeDispatch(t *testing.T) {
	validBody := func(name, imageURL, protein string) string {
		return `{"name":` + name + `,"physicalState":"solid","imageUrl":` + imageURL + `,"macrosPer100":{"protein":` + protein + `,"carbohydrates":0,"fat":0},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`
	}
	service := &curatedImportServiceStub{result: dataimporter.Result{ImportID: uuid.New(), FoodItemID: uuid.New(), Name: "Café au lait", PhysicalState: repository.PhysicalStateSolid}}
	app := fiber.New()
	app.Post("/", validateCuratedImport, func(ctx *fiber.Ctx) error {
		req := ctx.Locals("curatedImportRequest").(dataimporter.Request)
		service.calls++
		service.req = req
		return ctx.SendStatus(fiber.StatusNoContent)
	})

	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(validBody(`"  Cafe\u0301   au lait  "`, `" https://images.example.com/cafe.jpg "`, "10"))))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusNoContent || service.calls != 1 || service.req.Name != "Café au lait" || service.req.ImageURL != "https://images.example.com/cafe.jpg" {
		t.Fatalf("status=%d calls=%d normalized=%+v", response.StatusCode, service.calls, service.req)
	}

	for _, body := range []string{
		validBody(`"Oat\nMilk"`, `""`, "10"),
		validBody(`"Food"`, `"http://127.0.0.1/image"`, "10"),
		validBody(`"Food"`, `""`, "1e100"),
	} {
		response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(body)))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode < 400 {
			t.Fatalf("adversarial body accepted: %s", body)
		}
	}
	if service.calls != 1 {
		t.Fatalf("service calls=%d, want normalized request only", service.calls)
	}
}

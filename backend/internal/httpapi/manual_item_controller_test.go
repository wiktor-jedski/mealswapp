package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/itemcurator"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// fakeManualItemService verifies HTTP ownership and audit handoff without persistence coupling.
// Implements DESIGN-009 ItemCurator HTTP test boundary.
type fakeManualItemService struct {
	item      itemcurator.Item
	createErr error
	calls     []string
}

func (s *fakeManualItemService) Create(_ context.Context, _ repository.AdminMutationExecutor, _ uuid.UUID, key string, req itemcurator.Request) (itemcurator.CreateResult, error) {
	s.calls = append(s.calls, "create")
	if s.createErr != nil {
		return itemcurator.CreateResult{}, s.createErr
	}
	item := s.item
	item.Name, item.PhysicalState, item.MacrosPer100 = req.Name, req.PhysicalState, req.MacrosPer100
	return itemcurator.CreateResult{Item: item, Status: fiber.StatusCreated, Replayed: key == "replay-key-0001"}, nil
}

func (s *fakeManualItemService) Get(_ context.Context, _ uuid.UUID) (itemcurator.Item, error) {
	s.calls = append(s.calls, "get")
	return s.item, nil
}

func (s *fakeManualItemService) Update(_ context.Context, _ repository.AdminMutationExecutor, _ uuid.UUID, req itemcurator.Request) (itemcurator.MutationResult, error) {
	s.calls = append(s.calls, "update")
	after := s.item
	after.Name = req.Name
	return itemcurator.MutationResult{Before: s.item, After: after}, nil
}

func (s *fakeManualItemService) Delete(_ context.Context, _ repository.AdminMutationExecutor, _ uuid.UUID) (itemcurator.MutationResult, error) {
	s.calls = append(s.calls, "delete")
	return itemcurator.MutationResult{Before: s.item}, nil
}

// TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots verifies
// IT-ARCH-009-006, ARCH-009, DESIGN-009 ItemCurator, and SW-REQ-056.
func TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots(t *testing.T) {
	cfg := testConfig()
	adminID := uuid.New()
	authenticator, authCookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	itemID := uuid.New()
	service := &fakeManualItemService{item: itemcurator.Item{ID: itemID, Name: "Before", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}, Micros: repository.MicroValues{}, FoodCategories: []itemcurator.ClassificationSummary{}, CulinaryRoles: []itemcurator.ClassificationSummary{}}}
	audit := &adminAuditCoordinator{}
	controller := NewManualItemAdminController(audit, service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: &auditSink{}, Routes: controller.Routes()})
	csrf, csrfCookies := fetchCSRFToken(t, app)
	body := `{"name":"Manual tofu","physicalState":"solid","prepTimeMinutes":0,"macrosPer100":{"protein":10,"carbohydrates":2,"fat":3},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`

	create := manualItemHTTPRequest(t, app, fiber.MethodPost, "/api/v1/admin/items", body, authCookies, csrfCookies, csrf, "create-key-0001")
	if create.StatusCode != fiber.StatusCreated || audit.committed != 1 || len(audit.changes) != 1 || audit.changes[0].EntityID == nil || *audit.changes[0].EntityID != itemID || !strings.Contains(string(audit.changes[0].After), `"active":true`) {
		t.Fatalf("create status=%d audit=%+v", create.StatusCode, audit)
	}
	create.Body.Close()
	replay := manualItemHTTPRequest(t, app, fiber.MethodPost, "/api/v1/admin/items", body, authCookies, csrfCookies, csrf, "replay-key-0001")
	if replay.StatusCode != fiber.StatusCreated || audit.committed != 1 {
		t.Fatalf("replay status=%d audit commits=%d", replay.StatusCode, audit.committed)
	}
	replay.Body.Close()
	read := manualItemHTTPRequest(t, app, fiber.MethodGet, "/api/v1/admin/items/"+itemID.String(), "", authCookies, nil, "", "")
	if read.StatusCode != fiber.StatusOK {
		t.Fatalf("read status=%d", read.StatusCode)
	}
	read.Body.Close()
	update := manualItemHTTPRequest(t, app, fiber.MethodPut, "/api/v1/admin/items/"+itemID.String(), strings.Replace(body, "Manual tofu", "Manual tempeh", 1), authCookies, csrfCookies, csrf, "")
	if update.StatusCode != fiber.StatusOK || audit.committed != 2 || len(audit.changes[1].Before) == 0 || len(audit.changes[1].After) == 0 {
		t.Fatalf("update status=%d audit=%+v", update.StatusCode, audit)
	}
	update.Body.Close()
	deleted := manualItemHTTPRequest(t, app, fiber.MethodDelete, "/api/v1/admin/items/"+itemID.String(), "", authCookies, csrfCookies, csrf, "")
	if deleted.StatusCode != fiber.StatusNoContent || audit.committed != 3 || !strings.Contains(string(audit.changes[2].After), `"deleted":true`) {
		t.Fatalf("delete status=%d audit=%+v", deleted.StatusCode, audit)
	}
	deleted.Body.Close()
}

func TestManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	service := &fakeManualItemService{item: itemcurator.Item{ID: uuid.New(), PhysicalState: repository.PhysicalStateSolid}}
	controller := NewManualItemAdminController(&adminAuditCoordinator{}, service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: &auditSink{}, Routes: controller.Routes()})
	csrf, csrfCookies := fetchCSRFToken(t, app)
	valid := `{"name":"Manual tofu","physicalState":"solid","macrosPer100":{"protein":10,"carbohydrates":2,"fat":3},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`
	cases := []string{
		strings.Replace(valid, `"name":"Manual tofu"`, `"name":"First","name":"Second"`, 1),
		strings.Replace(valid, `"foodCategoryIds":[]`, `"foodCategoryIds":["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"]`, 1),
		strings.Replace(valid, `"protein":10`, `"protein":-1`, 1),
		strings.Replace(valid, `"physicalState":"solid"`, `"physicalState":"liquid"`, 1),
		strings.Replace(valid, `"micros":{}`, `"micros":{"bad\u0000key":1}`, 1),
		strings.Replace(valid, `"name":"Manual tofu"`, `"name":"Manual tofu","ownerId":"`+uuid.NewString()+`"`, 1),
		strings.Replace(valid, `"name":"Manual tofu"`, `"name":"Manual tofu","imageUrl":"ftp://example.test/a"`, 1),
	}
	for index, body := range cases {
		resp := manualItemHTTPRequest(t, app, fiber.MethodPost, "/api/v1/admin/items", body, authCookies, csrfCookies, csrf, "invalid-key-0001")
		if resp.StatusCode != fiber.StatusBadRequest {
			payload, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("invalid case %d status=%d body=%s", index, resp.StatusCode, payload)
		}
		resp.Body.Close()
	}
	service.createErr = itemcurator.ErrIdempotencyConflict
	conflict := manualItemHTTPRequest(t, app, fiber.MethodPost, "/api/v1/admin/items", valid, authCookies, csrfCookies, csrf, "conflict-key-0001")
	if conflict.StatusCode != fiber.StatusConflict {
		t.Fatalf("idempotency conflict status=%d", conflict.StatusCode)
	}
	conflict.Body.Close()
	service.createErr = repository.NewError(repository.ErrorKindConflict, "duplicate name", errors.New("database detail must not leak"))
	duplicate := manualItemHTTPRequest(t, app, fiber.MethodPost, "/api/v1/admin/items", valid, authCookies, csrfCookies, csrf, "duplicate-key-0001")
	if duplicate.StatusCode != fiber.StatusConflict {
		t.Fatalf("duplicate status=%d", duplicate.StatusCode)
	}
	duplicate.Body.Close()
	privatePath := manualItemHTTPRequest(t, app, fiber.MethodGet, "/api/v1/custom-items/"+service.item.ID.String(), "", authCookies, nil, "", "")
	if privatePath.StatusCode != fiber.StatusNotFound {
		t.Fatalf("manual controller exposed private route: %d", privatePath.StatusCode)
	}
	privatePath.Body.Close()
}

func manualItemHTTPRequest(t *testing.T, app *fiber.App, method string, path string, body string, authCookies []*http.Cookie, csrfCookies []*http.Cookie, csrf string, key string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	addCookies(req, authCookies)
	addCookies(req, csrfCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

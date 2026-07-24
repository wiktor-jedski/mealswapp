package httpapi

// Implements DESIGN-009 TagManager HTTP authorization, CRUD, audit, and invalidation verification.

import (
	"context"
	"io"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/tagmanager"
)

type classificationAdminMemoryRepository struct {
	items map[uuid.UUID]repository.ClassificationEntity
	inUse map[uuid.UUID]bool
}

func (r *classificationAdminMemoryRepository) List(_ context.Context, kind repository.ClassificationKind) ([]repository.ClassificationEntity, error) {
	items := []repository.ClassificationEntity{}
	for _, item := range r.items {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *classificationAdminMemoryRepository) GetByID(_ context.Context, id uuid.UUID) (repository.ClassificationEntity, error) {
	item, ok := r.items[id]
	if !ok {
		return repository.ClassificationEntity{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return item, nil
}

func (r *classificationAdminMemoryRepository) Create(_ context.Context, item repository.ClassificationEntity) (repository.ClassificationEntity, error) {
	for _, existing := range r.items {
		if existing.Kind == item.Kind && equalClassificationParent(existing.ParentID, item.ParentID) && strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(item.Name)) {
			return repository.ClassificationEntity{}, repository.NewError(repository.ErrorKindConflict, "duplicate", nil)
		}
	}
	item.ID = uuid.New()
	r.items[item.ID] = item
	return item, nil
}

func (r *classificationAdminMemoryRepository) Update(_ context.Context, item repository.ClassificationEntity) (repository.ClassificationEntity, error) {
	r.items[item.ID] = item
	return item, nil
}

func (r *classificationAdminMemoryRepository) SoftDelete(_ context.Context, id uuid.UUID) error {
	if r.inUse[id] {
		return repository.NewError(repository.ErrorKindConflict, "in use", nil)
	}
	delete(r.items, id)
	return nil
}

func equalClassificationParent(left, right *uuid.UUID) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

type classificationInvalidator struct{ calls int }

func (i *classificationInvalidator) Invalidate() { i.calls++ }

// TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation verifies
// IT-ARCH-009-005, ARCH-009, DESIGN-009 TagManager, and SW-REQ-057.
func TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation(t *testing.T) {
	cfg := testConfig()
	adminID := uuid.New()
	authenticator, authCookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	repo := &classificationAdminMemoryRepository{items: map[uuid.UUID]repository.ClassificationEntity{}, inUse: map[uuid.UUID]bool{}}
	invalidator := &classificationInvalidator{}
	audit := &adminAuditCoordinator{}
	tags := NewClassificationAdminController(tagmanager.NewService(repo), func(repository.AdminMutationExecutor) repository.ClassificationAdminRepository { return repo }, NewCurationRequestValidator(&observability.MemorySink{}), invalidator)
	controller := NewAdminController(audit, tags.AdminRoutes()...)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: &auditSink{}, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	request := func(method, path, body string) (int, Envelope) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		}
		if method != fiber.MethodGet {
			req.Header.Set("X-CSRF-Token", token)
		}
		addCookies(req, authCookies)
		addCookies(req, csrfCookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == fiber.StatusNoContent {
			_, _ = io.Copy(io.Discard, resp.Body)
			return resp.StatusCode, Envelope{}
		}
		return resp.StatusCode, decodeEnvelope(t, resp.Body)
	}

	status, body := request(fiber.MethodPost, "/api/v1/admin/classifications/food_category", `{"name":"Fruit"}`)
	if status != fiber.StatusCreated || invalidator.calls != 1 || len(audit.entries) != 1 {
		t.Fatalf("create status=%d body=%+v invalidations=%d audits=%d", status, body, invalidator.calls, len(audit.entries))
	}
	created := body.Data["classification"].(map[string]any)
	id := uuid.MustParse(created["id"].(string))
	if strings.Contains(string(audit.entries[0].After), "Fruit") || len(audit.entries[0].After) == 0 {
		t.Fatalf("unsafe create audit = %s", audit.entries[0].After)
	}
	status, body = request(fiber.MethodGet, "/api/v1/admin/classifications?kind=food_category", "")
	if status != fiber.StatusOK || len(body.Data["classifications"].([]any)) != 1 {
		t.Fatalf("list status=%d body=%+v", status, body)
	}
	status, _ = request(fiber.MethodPost, "/api/v1/admin/classifications/food_category", `{"name":" fruit "}`)
	if status != fiber.StatusConflict || invalidator.calls != 1 {
		t.Fatalf("duplicate status=%d invalidations=%d", status, invalidator.calls)
	}
	status, body = request(fiber.MethodPut, "/api/v1/admin/classifications/"+id.String(), `{"name":"Produce"}`)
	if status != fiber.StatusOK || body.Data["classification"].(map[string]any)["name"] != "Produce" || invalidator.calls != 2 {
		t.Fatalf("update status=%d body=%+v invalidations=%d", status, body, invalidator.calls)
	}
	repo.inUse[id] = true
	status, _ = request(fiber.MethodDelete, "/api/v1/admin/classifications/"+id.String(), "")
	if status != fiber.StatusConflict || invalidator.calls != 2 {
		t.Fatalf("in-use delete status=%d invalidations=%d", status, invalidator.calls)
	}
	repo.inUse[id] = false
	status, _ = request(fiber.MethodDelete, "/api/v1/admin/classifications/"+id.String(), "")
	if status != fiber.StatusNoContent || invalidator.calls != 3 || len(audit.entries) != 3 || len(audit.entries[1].Before) == 0 || len(audit.entries[1].After) == 0 || len(audit.entries[2].Before) == 0 || len(audit.entries[2].After) == 0 {
		t.Fatalf("delete status=%d invalidations=%d audits=%+v", status, invalidator.calls, audit.entries)
	}
}

func TestClassificationAdminHTTPRejectsNonAdminMutationAndAuditFailureInvalidation(t *testing.T) {
	cfg := testConfig()
	repo := &classificationAdminMemoryRepository{items: map[uuid.UUID]repository.ClassificationEntity{}, inUse: map[uuid.UUID]bool{}}
	invalidator := &classificationInvalidator{}
	tags := NewClassificationAdminController(tagmanager.NewService(repo), func(repository.AdminMutationExecutor) repository.ClassificationAdminRepository { return repo }, NewCurationRequestValidator(nil), invalidator)

	userAuth, userCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleUser), nil)
	userController := NewAdminController(&adminAuditCoordinator{}, tags.AdminRoutes()...)
	userApp := mustNewRouter(t, Dependencies{Config: cfg, Auth: userAuth, Audit: &auditSink{}, Routes: userController.Routes()})
	userToken, userCSRFCookies := fetchCSRFToken(t, userApp)
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/classifications/food_category", strings.NewReader(`{"name":"Denied"}`))
	req.Header.Set("X-CSRF-Token", userToken)
	addCookies(req, userCookies)
	addCookies(req, userCSRFCookies)
	resp, err := userApp.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden || len(repo.items) != 0 {
		t.Fatalf("non-admin status=%d items=%d", resp.StatusCode, len(repo.items))
	}

	adminAuth, adminCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	failingAudit := &adminAuditCoordinator{err: repository.ErrAdminAuditPersistence}
	adminController := NewAdminController(failingAudit, tags.AdminRoutes()...)
	adminApp := mustNewRouter(t, Dependencies{Config: cfg, Auth: adminAuth, Audit: &auditSink{}, Routes: adminController.Routes()})
	adminToken, adminCSRFCookies := fetchCSRFToken(t, adminApp)
	req = httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/classifications/food_category", strings.NewReader(`{"name":"Rollback"}`))
	req.Header.Set("X-CSRF-Token", adminToken)
	addCookies(req, adminCookies)
	addCookies(req, adminCSRFCookies)
	resp, err = adminApp.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable || invalidator.calls != 0 {
		t.Fatalf("audit failure status=%d invalidations=%d", resp.StatusCode, invalidator.calls)
	}
}

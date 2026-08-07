package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/micronutrient"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type micronutrientAdminMemoryRepository struct {
	entries map[string]repository.MicronutrientVocabularyEntry
	inUse   map[string]bool
}

func (r *micronutrientAdminMemoryRepository) ListAll(context.Context) ([]repository.MicronutrientVocabularyEntry, error) {
	entries := make([]repository.MicronutrientVocabularyEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries, nil
}
func (r *micronutrientAdminMemoryRepository) Get(_ context.Context, key string) (repository.MicronutrientVocabularyEntry, error) {
	entry, ok := r.entries[key]
	if !ok {
		return entry, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return entry, nil
}
func (r *micronutrientAdminMemoryRepository) Create(_ context.Context, entry repository.MicronutrientVocabularyEntry) (repository.MicronutrientVocabularyEntry, error) {
	r.entries[entry.Key] = entry
	return entry, nil
}
func (r *micronutrientAdminMemoryRepository) UpdateDisplayName(_ context.Context, key, value string) (repository.MicronutrientVocabularyEntry, error) {
	entry := r.entries[key]
	entry.DisplayName = value
	r.entries[key] = entry
	return entry, nil
}
func (r *micronutrientAdminMemoryRepository) UpdateUnit(_ context.Context, key, value string) (repository.MicronutrientVocabularyEntry, error) {
	if r.inUse[key] {
		return repository.MicronutrientVocabularyEntry{}, repository.NewError(repository.ErrorKindConflict, "in use", nil)
	}
	entry := r.entries[key]
	entry.Unit = value
	r.entries[key] = entry
	return entry, nil
}
func (r *micronutrientAdminMemoryRepository) SetActive(_ context.Context, key string, active bool) (repository.MicronutrientVocabularyEntry, error) {
	if !active && r.inUse[key] {
		return repository.MicronutrientVocabularyEntry{}, repository.NewError(repository.ErrorKindConflict, "in use", nil)
	}
	entry := r.entries[key]
	entry.Active = active
	r.entries[key] = entry
	return entry, nil
}

// TestMicronutrientAdminRoutesAreClosedAndHaveNoDelete proves the explicit security allowlist.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-009 AdminController.
func TestMicronutrientAdminRoutesAreClosedAndHaveNoDelete(t *testing.T) {
	controller := NewMicronutrientAdminController(micronutrient.NewService(nil), nil)
	routes := controller.AdminRoutes()
	if len(routes) != 6 {
		t.Fatalf("route count = %d", len(routes))
	}
	for _, route := range routes {
		if route.Method == fiber.MethodDelete {
			t.Fatalf("hard-delete route registered: %+v", route)
		}
		if route.RateLimit == nil || route.Method != fiber.MethodGet && (route.Validate == nil || route.Mutation == nil || route.AuditAction == "") {
			t.Fatalf("unprotected route: %+v", route)
		}
	}
}

// TestMicronutrientValidationRejectsMutableKeysAndBodies proves strict route validation.
func TestMicronutrientValidationRejectsMutableKeysAndBodies(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: func(ctx *fiber.Ctx, err error) error {
		var appErr AppError
		if errors.As(err, &appErr) {
			return ctx.SendStatus(appErr.HTTPStatus)
		}
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}})
	app.Post("/create", validateMicronutrientCreate, func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) })
	req := httptest.NewRequest(fiber.MethodPost, "/create", bytes.NewBufferString(`{"key":"Na","displayName":"Sodium","unit":"mg"}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(req)
	if err != nil || response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("alias response = %v, %v", response, err)
	}
	req = httptest.NewRequest(fiber.MethodPost, "/create", bytes.NewBufferString(`{"key":"Sodium","displayName":"Sodium","unit":"mg","active":false}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err = app.Test(req)
	if err != nil || response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("mutable active response = %v, %v", response, err)
	}
}

// TestMicronutrientAdminHTTPLifecycle proves authenticated CSRF-protected lifecycle, conflicts, audit, and no hard delete.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-009 AdminController.
func TestMicronutrientAdminHTTPLifecycle(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	repo := &micronutrientAdminMemoryRepository{entries: map[string]repository.MicronutrientVocabularyEntry{}, inUse: map[string]bool{}}
	audit := &adminAuditCoordinator{}
	vocabulary := NewMicronutrientAdminController(micronutrient.NewService(repo), func(repository.AdminMutationExecutor) repository.MicronutrientVocabularyAdminRepository { return repo })
	controller := NewAdminController(audit, vocabulary.AdminRoutes()...)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: &auditSink{}, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	request := func(method, path, body string) (int, Envelope) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		}
		if method != fiber.MethodGet && method != fiber.MethodDelete {
			req.Header.Set("X-CSRF-Token", token)
		}
		addCookies(req, authCookies)
		addCookies(req, csrfCookies)
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode == fiber.StatusNotFound {
			return response.StatusCode, Envelope{}
		}
		return response.StatusCode, decodeEnvelope(t, response.Body)
	}

	status, _ := request(fiber.MethodPost, "/api/v1/admin/micronutrients", `{"key":"VitaminK","displayName":"Vitamin K","unit":"mcg"}`)
	if status != fiber.StatusCreated {
		t.Fatalf("create status = %d", status)
	}
	status, body := request(fiber.MethodGet, "/api/v1/admin/micronutrients", "")
	if status != fiber.StatusOK || len(body.Data["micronutrients"].([]any)) != 1 {
		t.Fatalf("list = %d %+v", status, body)
	}
	// The live response projection must match the documented lower-camel-case API contract.
	encoded, err := json.Marshal(repository.MicronutrientVocabularyEntry{Key: "VitaminK", DisplayName: "Vitamin K", Unit: "mcg", Active: true})
	if err != nil || string(encoded) != `{"key":"VitaminK","displayName":"Vitamin K","unit":"mcg","active":true}` {
		t.Fatalf("micronutrient response JSON = %s, err=%v", encoded, err)
	}
	status, _ = request(fiber.MethodPut, "/api/v1/admin/micronutrients/VitaminK/display-name", `{"displayName":"Vitamin K1"}`)
	if status != fiber.StatusOK {
		t.Fatalf("display update status = %d", status)
	}
	status, _ = request(fiber.MethodPut, "/api/v1/admin/micronutrients/VitaminK/unit", `{"unit":"mg"}`)
	if status != fiber.StatusOK {
		t.Fatalf("unit update status = %d", status)
	}
	repo.inUse["VitaminK"] = true
	status, _ = request(fiber.MethodPost, "/api/v1/admin/micronutrients/VitaminK/deactivate", "")
	if status != fiber.StatusConflict || !repo.entries["VitaminK"].Active {
		t.Fatalf("in-use deactivate = %d %+v", status, repo.entries["VitaminK"])
	}
	repo.inUse["VitaminK"] = false
	status, _ = request(fiber.MethodPost, "/api/v1/admin/micronutrients/VitaminK/deactivate", "")
	if status != fiber.StatusOK || repo.entries["VitaminK"].Active {
		t.Fatalf("deactivate = %d %+v", status, repo.entries["VitaminK"])
	}
	status, _ = request(fiber.MethodPost, "/api/v1/admin/micronutrients/VitaminK/reactivate", "")
	if status != fiber.StatusOK || !repo.entries["VitaminK"].Active || len(audit.entries) != 5 {
		t.Fatalf("reactivate = %d %+v audits=%d", status, repo.entries["VitaminK"], len(audit.entries))
	}
	status, _ = request(fiber.MethodDelete, "/api/v1/admin/micronutrients/VitaminK", "")
	if status != fiber.StatusNotFound {
		t.Fatalf("hard delete status = %d", status)
	}
}

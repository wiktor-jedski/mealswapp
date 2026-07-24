package httpapi

// Implements DESIGN-009 AdminController authorization, gateway ordering, and audit-boundary verification.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type adminAuditCoordinator struct {
	entries   []repository.AdminAuditEntry
	changes   []repository.AdminAuditChanges
	committed int
	err       error
}

func (c *adminAuditCoordinator) WithMutationAudit(_ context.Context, entry repository.AdminAuditEntry, fn func(repository.AdminMutationExecutor) (repository.AdminAuditChanges, error)) error {
	changes, err := fn(nil)
	if err != nil {
		return err
	}
	if changes.Replayed {
		return nil
	}
	if c.err != nil {
		return fmt.Errorf("%w: %v", repository.ErrAdminAuditPersistence, c.err)
	}
	entry.EntityID, entry.Before, entry.After = changes.EntityID, changes.Before, changes.After
	c.entries = append(c.entries, entry)
	c.changes = append(c.changes, changes)
	c.committed++
	return nil
}

// TestAdminGatewayAuthorizationAllowlistAndSafeReadDegradation verifies IT-ARCH-009-001,
// ARCH-009, DESIGN-009 AdminController, and SW-REQ-054.
func TestAdminGatewayAuthorizationAllowlistAndSafeReadDegradation(t *testing.T) {
	cfg := testConfig()
	adminID := uuid.New()
	adminAuth, adminCookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	userAuth, userCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleUser), nil)
	rule := RateLimitRule{Scope: "user", MaxRequests: 20, WindowSeconds: 60}
	controller := NewAdminController(nil, AdminRouteDefinition{
		Method: fiber.MethodGet, Path: "/fixture", RateLimit: &rule,
		Handler: func(ctx *fiber.Ctx) error {
			admin, err := RequireAdmin(ctx)
			if err != nil || admin.UserID != adminID || admin.Role != "admin" || admin.RequestID == "" {
				return errors.New("admin context mismatch")
			}
			return ctx.JSON(Envelope{Status: "ok", RequestID: admin.RequestID, Data: map[string]any{"visible": true}})
		},
	})

	request := func(authenticator *JWTAuthenticator, cookies []*http.Cookie, headers map[string]string, path string) (int, Envelope) {
		t.Helper()
		app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: controller.Routes()})
		req := httptest.NewRequest(fiber.MethodGet, path, nil)
		for name, value := range headers {
			req.Header.Set(name, value)
		}
		addCookies(req, cookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, decodeEnvelope(t, resp.Body)
	}

	const clientRequestID = "client-controlled-request-id"
	status, body := request(nil, nil, map[string]string{"X-Role": "admin", "X-User-ID": adminID.String(), "X-Request-ID": clientRequestID}, "/api/v1/admin/fixture")
	if status != fiber.StatusUnauthorized || !isServerRequestID(body.RequestID, clientRequestID) {
		t.Fatalf("anonymous spoof = %d %+v", status, body)
	}
	status, body = request(userAuth, userCookies, map[string]string{"X-Role": "admin", "X-User-ID": adminID.String(), "X-Request-ID": clientRequestID}, "/api/v1/admin/fixture")
	if status != fiber.StatusForbidden || body.Error == nil || body.Error.Code != "forbidden" || !isServerRequestID(body.RequestID, clientRequestID) {
		t.Fatalf("non-admin spoof = %d %+v", status, body)
	}
	status, body = request(adminAuth, adminCookies, map[string]string{"X-Role": "user", "X-User-ID": uuid.NewString(), "X-Request-ID": clientRequestID}, "/api/v1/admin/fixture")
	if status != fiber.StatusOK || !isServerRequestID(body.RequestID, clientRequestID) {
		t.Fatalf("verified admin = %d %+v", status, body)
	}
	status, body = request(adminAuth, adminCookies, nil, "/api/v1/admin/undocumented")
	if status != fiber.StatusNotFound || body.RequestID == "" {
		t.Fatalf("undocumented route = %d %+v", status, body)
	}
}

func TestAdminMutationControlOrderAtomicAuditAndSanitizedEnvelopes(t *testing.T) {
	cfg := testConfig()
	adminID := uuid.New()
	authenticator, authCookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	securityAudit := &auditSink{}
	logs := &observability.MemorySink{}
	adminAudit := &adminAuditCoordinator{}
	validationCalls, mutationCalls := 0, 0
	rule := RateLimitRule{Scope: "endpoint", MaxRequests: 1, WindowSeconds: 60}
	controller := NewAdminController(adminAudit, AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/fixture", RateLimit: &rule, AuditAction: "fixture.update", EntityType: "fixture",
		Validate: func(ctx *fiber.Ctx) error {
			validationCalls++
			if !strings.Contains(string(ctx.Body()), `"valid":true`) {
				return curationValidationError()
			}
			return ctx.Next()
		},
		Mutation: func(_ *fiber.Ctx, _ repository.AdminMutationExecutor) (AdminMutationResult, error) {
			mutationCalls++
			return AdminMutationResult{Data: map[string]any{"updated": true}, Audit: repository.AdminAuditChanges{Before: []byte(`{"state":"before"}`), After: []byte(`{"state":"after"}`)}}, nil
		},
	})
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: securityAudit, Logs: logs, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	send := func(body, csrf string) (int, Envelope) {
		t.Helper()
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/fixture", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Role", "admin")
		req.Header.Set("X-Request-ID", "client-controlled-request-id")
		req.Header.Set("X-Provider-Payload", "raw-provider-secret")
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		addCookies(req, authCookies)
		addCookies(req, csrfCookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, decodeEnvelope(t, resp.Body)
	}

	status, _ := send(`{"valid":true}`, "")
	if status != fiber.StatusForbidden || validationCalls != 0 || mutationCalls != 0 {
		t.Fatalf("csrf order status=%d validation=%d mutation=%d", status, validationCalls, mutationCalls)
	}
	status, _ = send(`{"providerPayload":"raw-provider-secret"}`, token)
	if status != fiber.StatusBadRequest || validationCalls != 1 || mutationCalls != 0 {
		t.Fatalf("validation order status=%d validation=%d mutation=%d", status, validationCalls, mutationCalls)
	}
	status, body := send(`{"valid":true}`, token)
	if status != fiber.StatusOK || mutationCalls != 1 || adminAudit.committed != 1 || len(adminAudit.entries) != 1 {
		t.Fatalf("successful mutation status=%d body=%+v mutation=%d audit=%d", status, body, mutationCalls, adminAudit.committed)
	}
	if !isServerRequestID(body.RequestID, "client-controlled-request-id") || adminAudit.entries[0].RequestID != body.RequestID || adminAudit.entries[0].AdminUserID != adminID {
		t.Fatalf("correlation body=%+v audit=%+v", body, adminAudit.entries[0])
	}
	status, body = send(`{"valid":true}`, token)
	if status != fiber.StatusTooManyRequests || mutationCalls != 1 || body.Error == nil || body.Error.Code != "rate_limited" {
		t.Fatalf("rate order status=%d body=%+v mutation=%d", status, body, mutationCalls)
	}
	for _, event := range logs.Logs {
		if !isServerRequestID(event.RequestID, "client-controlled-request-id") {
			t.Fatalf("log has unsafe request ID: %+v", event)
		}
		encoded := event.Message
		for _, value := range event.Fields {
			encoded += " " + strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(toString(value)), "\n", ""), "\r", ""))
		}
		if strings.Contains(encoded, "raw-provider-secret") {
			t.Fatalf("raw provider payload reached logs: %+v", event)
		}
	}
	for _, entry := range securityAudit.entries {
		if !isServerRequestID(entry.RequestID, "client-controlled-request-id") || strings.Contains(entry.Action+entry.Resource+entry.Outcome, "raw-provider-secret") {
			t.Fatalf("unsafe or uncorrelated security audit: %+v", entry)
		}
	}
}

func TestAdminRouteRegistrationRejectsMissingControls(t *testing.T) {
	rule := RateLimitRule{Scope: "user", MaxRequests: 1, WindowSeconds: 60}
	validMutation := AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/fixture", RateLimit: &rule, AuditAction: "fixture.update", EntityType: "fixture",
		Validate: func(ctx *fiber.Ctx) error { return ctx.Next() },
		Mutation: func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{}, nil
		},
	}
	cases := []AdminRouteDefinition{
		{Method: fiber.MethodTrace, Path: "/fixture", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule},
		{Method: fiber.MethodGet, Path: "/fixture", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }},
		{Method: fiber.MethodGet, Path: "/admin/fixture", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule},
		{Method: fiber.MethodGet, Path: "/fixture", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, Mutation: validMutation.Mutation, RateLimit: &rule},
		{Method: fiber.MethodGet, Path: "/../fixture", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule},
		{Method: fiber.MethodGet, Path: "/*", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule},
		{Method: fiber.MethodGet, Path: "/fixture/:id?", Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule},
		{Method: fiber.MethodPost, Path: "/fixture", Mutation: validMutation.Mutation, Validate: validMutation.Validate, AuditAction: validMutation.AuditAction, EntityType: validMutation.EntityType},
		{Method: fiber.MethodPost, Path: "/fixture", RateLimit: &rule, Mutation: validMutation.Mutation, AuditAction: validMutation.AuditAction, EntityType: validMutation.EntityType},
		{Method: fiber.MethodPost, Path: "/fixture", RateLimit: &rule, Mutation: validMutation.Mutation, Validate: validMutation.Validate, EntityType: validMutation.EntityType},
		{Method: fiber.MethodPost, Path: "/fixture", RateLimit: &rule, Mutation: validMutation.Mutation, Validate: validMutation.Validate, AuditAction: " fixture.update", EntityType: validMutation.EntityType},
	}
	for index, route := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("case %d did not reject unsafe admin route", index)
				}
			}()
			_ = NewAdminController(nil, route).Routes()
		}()
	}

	read := func(path string) AdminRouteDefinition {
		return AdminRouteDefinition{Method: fiber.MethodGet, Path: path, Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule}
	}
	if routes := NewAdminController(nil, read("/fixture/:id")).Routes(); len(routes) != 1 || routes[0].Path != "/admin/fixture/:id" {
		t.Fatalf("documented parameter route = %#v", routes)
	}
	assertAdminRoutesPanic(t, read("/fixture/:id"), read("/fixture/:id"))
}

func TestAdminRouteRegistrationRejectsSemanticCollisionsInEitherOrder(t *testing.T) {
	rule := RateLimitRule{Scope: "user", MaxRequests: 1, WindowSeconds: 60}
	read := func(path string) AdminRouteDefinition {
		return AdminRouteDefinition{Method: fiber.MethodGet, Path: path, Handler: func(ctx *fiber.Ctx) error { return ctx.Next() }, RateLimit: &rule}
	}
	for _, routes := range [][]AdminRouteDefinition{
		{read("/fixture/:id"), read("/fixture/:name")},
		{read("/fixture/:name"), read("/fixture/:id")},
		{read("/fixture/search"), read("/fixture/:id")},
		{read("/fixture/:id"), read("/fixture/search")},
	} {
		assertAdminRoutesPanic(t, routes...)
	}
	if routes := NewAdminController(nil, read("/fixture/search"), read("/fixture/:id/details")).Routes(); len(routes) != 2 {
		t.Fatalf("non-overlapping admin routes = %#v", routes)
	}
}

func TestAdminRoutePathGrammarBranches(t *testing.T) {
	for _, path := range []string{"", "/", "fixture", "/fixture/", "/fixture//details", "/:id/:id"} {
		if isSafeAdminRoutePath(path) {
			t.Fatalf("isSafeAdminRoutePath(%q) = true", path)
		}
	}
	for _, path := range []string{"/fixture-2", "/fixture/:itemId/details"} {
		if !isSafeAdminRoutePath(path) {
			t.Fatalf("isSafeAdminRoutePath(%q) = false", path)
		}
	}
	if adminRoutePathsCollide("/fixture/search", "/fixture/details") {
		t.Fatal("distinct static routes collide")
	}
}

func assertAdminRoutesPanic(t *testing.T, routes ...AdminRouteDefinition) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("colliding admin routes did not fail startup")
		}
	}()
	_ = NewAdminController(nil, routes...).Routes()
}

func TestGatewayRegistrationRejectsAdminRouteWithoutAuthentication(t *testing.T) {
	app := fiber.New()
	defer func() {
		if recover() == nil {
			t.Fatal("admin route without authentication did not fail startup")
		}
	}()
	registerV1Routes(app.Group("/api/v1"), Dependencies{Routes: []RouteDefinition{{
		Method: fiber.MethodGet, Path: "/admin/fixture", RequiresAdmin: true, Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusOK) },
	}}})
}

func TestAdminMutationResponseWaitsForCommitAndPreservesDomainErrors(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	rule := RateLimitRule{Scope: "user", MaxRequests: 10, WindowSeconds: 60}
	securityAudit := &auditSink{}

	t.Run("direct missing context unauthorized", func(t *testing.T) {
		app := fiber.New(fiber.Config{ErrorHandler: writeError})
		app.Get("/", func(ctx *fiber.Ctx) error {
			_, err := RequireAdmin(ctx)
			return err
		})
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("missing admin context = %d", resp.StatusCode)
		}
	})

	t.Run("audit dependency required", func(t *testing.T) {
		controller := NewAdminController(nil, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{HTTPStatus: fiber.StatusCreated}, nil
		}))
		status, body := sendAdminMutation(t, cfg, authenticator, authCookies, securityAudit, controller)
		if status != fiber.StatusServiceUnavailable || body.Error == nil || body.Error.Code != "dependency_unavailable" {
			t.Fatalf("nil audit dependency = %d %+v", status, body)
		}
	})

	t.Run("wrapper rejects missing admin context", func(t *testing.T) {
		controller := NewAdminController(&adminAuditCoordinator{})
		app := fiber.New(fiber.Config{ErrorHandler: writeError})
		app.Post("/", controller.transactionalMutation(validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{}, nil
		})))
		resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("missing wrapper context = %d", resp.StatusCode)
		}
	})

	t.Run("domain error preserved", func(t *testing.T) {
		controller := NewAdminController(&adminAuditCoordinator{}, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{}, AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "conflict", Message: "resource conflicts with existing data"}
		}))
		status, body := sendAdminMutation(t, cfg, authenticator, authCookies, securityAudit, controller)
		if status != fiber.StatusConflict || body.Error == nil || body.Error.Code != "conflict" {
			t.Fatalf("domain error = %d %+v", status, body)
		}
	})

	t.Run("no content after commit", func(t *testing.T) {
		coordinator := &adminAuditCoordinator{}
		controller := NewAdminController(coordinator, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{HTTPStatus: fiber.StatusNoContent}, nil
		}))
		app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: securityAudit, Routes: controller.Routes()})
		token, csrfCookies := fetchCSRFToken(t, app)
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/fixture", strings.NewReader(`{"valid":true}`))
		req.Header.Set("X-CSRF-Token", token)
		addCookies(req, authCookies)
		addCookies(req, csrfCookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusNoContent || coordinator.committed != 1 {
			t.Fatalf("no-content mutation = %d committed=%d", resp.StatusCode, coordinator.committed)
		}
	})

	t.Run("created response after commit", func(t *testing.T) {
		coordinator := &adminAuditCoordinator{}
		controller := NewAdminController(coordinator, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{HTTPStatus: fiber.StatusCreated, Data: map[string]any{"created": true}}, nil
		}))
		status, body := sendAdminMutation(t, cfg, authenticator, authCookies, securityAudit, controller)
		if status != fiber.StatusCreated || coordinator.committed != 1 || body.Data["created"] != true {
			t.Fatalf("created mutation = %d committed=%d body=%+v", status, coordinator.committed, body)
		}
	})

	t.Run("response serialization failure rolls back", func(t *testing.T) {
		coordinator := &adminAuditCoordinator{}
		controller := NewAdminController(coordinator, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{Data: map[string]any{"unsupported": make(chan int)}}, nil
		}))
		status, body := sendAdminMutation(t, cfg, authenticator, authCookies, securityAudit, controller)
		if status != fiber.StatusInternalServerError || coordinator.committed != 0 || body.Error == nil || body.Error.Code != "internal_error" {
			t.Fatalf("serialization failure = %d committed=%d body=%+v", status, coordinator.committed, body)
		}
	})

	for _, status := range []int{fiber.StatusContinue, fiber.StatusMultipleChoices} {
		t.Run(fmt.Sprintf("invalid success status %d rolls back", status), func(t *testing.T) {
			coordinator := &adminAuditCoordinator{}
			controller := NewAdminController(coordinator, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
				return AdminMutationResult{HTTPStatus: status}, nil
			}))
			got, body := sendAdminMutation(t, cfg, authenticator, authCookies, securityAudit, controller)
			if got != fiber.StatusInternalServerError || coordinator.committed != 0 || body.Error == nil || body.Error.Code != "internal_error" {
				t.Fatalf("invalid status = %d committed=%d body=%+v", got, coordinator.committed, body)
			}
		})
	}

	t.Run("no content with data rolls back", func(t *testing.T) {
		coordinator := &adminAuditCoordinator{}
		controller := NewAdminController(coordinator, validAdminMutationRoute(rule, func(*fiber.Ctx, repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{HTTPStatus: fiber.StatusNoContent, Data: map[string]any{"unexpected": true}}, nil
		}))
		status, body := sendAdminMutation(t, cfg, authenticator, authCookies, securityAudit, controller)
		if status != fiber.StatusInternalServerError || coordinator.committed != 0 || body.Error == nil || body.Error.Code != "internal_error" {
			t.Fatalf("no-content data = %d committed=%d body=%+v", status, coordinator.committed, body)
		}
	})
}

func TestAdminRequestIDsAreServerGeneratedAndConcurrentUnique(t *testing.T) {
	cfg := testConfig()
	authenticator, cookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	rule := RateLimitRule{Scope: "user", MaxRequests: 1000, WindowSeconds: 60}
	controller := NewAdminController(nil, AdminRouteDefinition{Method: fiber.MethodGet, Path: "/fixture", RateLimit: &rule, Handler: func(ctx *fiber.Ctx) error {
		return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx)})
	}})
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Routes: controller.Routes()})

	const requests = 32
	ids := make(chan string, requests)
	errs := make(chan error, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/fixture", nil)
			req.Header.Set("X-Request-ID", "same-client-controlled-id")
			addCookies(req, cookies)
			resp, err := app.Test(req)
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				errs <- fmt.Errorf("status %d", resp.StatusCode)
				return
			}
			ids <- decodeEnvelope(t, resp.Body).RequestID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := make(map[string]struct{}, requests)
	for id := range ids {
		if !isServerRequestID(id, "same-client-controlled-id") {
			t.Fatalf("unsafe request ID %q", id)
		}
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("duplicate request ID %q", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != requests {
		t.Fatalf("unique request IDs = %d, want %d", len(seen), requests)
	}
}

func validAdminMutationRoute(rule RateLimitRule, mutation AdminMutationHandler) AdminRouteDefinition {
	return AdminRouteDefinition{
		Method: fiber.MethodPost, Path: "/fixture", RateLimit: &rule, AuditAction: "fixture.update", EntityType: "fixture",
		Validate: func(ctx *fiber.Ctx) error { return ctx.Next() }, Mutation: mutation,
	}
}

func sendAdminMutation(t *testing.T, cfg config.Config, authenticator *JWTAuthenticator, authCookies []*http.Cookie, securityAudit *auditSink, controller *AdminController) (int, Envelope) {
	t.Helper()
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: securityAudit, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/fixture", strings.NewReader(`{"valid":true}`))
	req.Header.Set("X-Request-ID", "client-controlled-request-id")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, authCookies)
	addCookies(req, csrfCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, decodeEnvelope(t, resp.Body)
}

// TestAdminMutationRollsBackWhenTransactionalAuditFails verifies IT-ARCH-009-003,
// ARCH-009, DESIGN-009 AdminController, and SW-REQ-055/SW-REQ-056/SW-REQ-057.
func TestAdminMutationRollsBackWhenTransactionalAuditFails(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleAdmin), nil)
	securityAudit := &auditSink{}
	adminAudit := &adminAuditCoordinator{err: errors.New("audit persistence unavailable")}
	rule := RateLimitRule{Scope: "user", MaxRequests: 2, WindowSeconds: 60}
	controller := NewAdminController(adminAudit, AdminRouteDefinition{
		Method: fiber.MethodDelete, Path: "/fixture/:id", RateLimit: &rule, AuditAction: "fixture.delete", EntityType: "fixture",
		Validate: func(ctx *fiber.Ctx) error { return ctx.Next() },
		Mutation: func(_ *fiber.Ctx, _ repository.AdminMutationExecutor) (AdminMutationResult, error) {
			return AdminMutationResult{HTTPStatus: fiber.StatusNoContent}, nil
		},
	})
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, Audit: securityAudit, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	req := httptest.NewRequest(fiber.MethodDelete, "/api/v1/admin/fixture/1", nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, authCookies)
	addCookies(req, csrfCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := decodeEnvelope(t, resp.Body)
	if resp.StatusCode != fiber.StatusServiceUnavailable || adminAudit.committed != 0 || body.Error == nil || body.Error.Code != "dependency_unavailable" || !isServerRequestID(body.RequestID, "client-controlled-request-id") {
		t.Fatalf("audit failure = %d committed=%d body=%+v", resp.StatusCode, adminAudit.committed, body)
	}
	if strings.Contains(string(mustJSON(t, body)), "audit persistence unavailable") {
		t.Fatalf("internal audit error leaked: %+v", body)
	}
}

func toString(value any) string { return fmt.Sprint(value) }

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func isServerRequestID(value string, clientValue string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value && value != clientValue && len(value) == 36
}

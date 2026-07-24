package httpapi

// Implements DESIGN-009 UserAdminPanel HTTP authorization, projection, validation, retry, and audit verification.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/useradmin"
)

type fakeUserAdminService struct {
	page          useradmin.Page
	lookupActor   useradmin.Actor
	lookupRequest useradmin.LookupRequest
	lookupCalls   int
	retryActor    useradmin.Actor
	retryUserID   uuid.UUID
	retryRequest  uuid.UUID
	retryCalls    int
	err           error
}

func (s *fakeUserAdminService) Lookup(_ context.Context, actor useradmin.Actor, request useradmin.LookupRequest) (useradmin.Page, error) {
	s.lookupActor, s.lookupRequest, s.lookupCalls = actor, request, s.lookupCalls+1
	return s.page, s.err
}

func (s *fakeUserAdminService) RetryDeletion(_ context.Context, actor useradmin.Actor, userID uuid.UUID, requestID uuid.UUID, _ repository.AdminMutationExecutor) (useradmin.RetryResult, error) {
	s.retryActor, s.retryUserID, s.retryRequest, s.retryCalls = actor, userID, requestID, s.retryCalls+1
	if s.err != nil {
		return useradmin.RetryResult{}, s.err
	}
	return useradmin.RetryResult{RequestID: requestID, FailureCategory: "permanent"}, nil
}

func TestUserAdminHTTPFailClosedProjectionAndBoundedValidation(t *testing.T) {
	cfg := testConfig()
	adminID := uuid.New()
	userID := uuid.New()
	adminAuth, adminCookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	userAuth, userCookies := testJWTAuthRole(t, cfg, uuid.New(), string(repository.UserRoleUser), nil)
	service := &fakeUserAdminService{page: useradmin.Page{Users: []useradmin.User{{ID: userID, Email: "approved@example.test", EmailVerified: true, CreatedAt: time.Now()}}}}
	controller := NewAdminController(nil, NewUserAdminController(service).AdminRoutes()...)
	logs := &observability.MemorySink{}

	send := func(auth *JWTAuthenticator, cookies []*http.Cookie, path string) (int, string) {
		t.Helper()
		app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Logs: logs, Routes: controller.Routes()})
		req := httptest.NewRequest(fiber.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer secret-token-material")
		req.Header.Set("X-Role", "admin")
		req.Header.Set("X-User-ID", adminID.String())
		addCookies(req, cookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(body)
	}

	status, _ := send(nil, nil, "/api/v1/admin/users")
	if status != fiber.StatusUnauthorized || service.lookupCalls != 0 {
		t.Fatalf("anonymous status=%d calls=%d", status, service.lookupCalls)
	}
	status, _ = send(userAuth, userCookies, "/api/v1/admin/users")
	if status != fiber.StatusForbidden || service.lookupCalls != 0 {
		t.Fatalf("non-admin status=%d calls=%d", status, service.lookupCalls)
	}
	status, body := send(adminAuth, adminCookies, "/api/v1/admin/users?limit=1")
	if status != fiber.StatusOK || service.lookupCalls != 1 || service.lookupActor.UserID != adminID || service.lookupActor.Role != "admin" || service.lookupRequest.Limit != 1 {
		t.Fatalf("admin status=%d calls=%d actor=%+v request=%+v body=%s", status, service.lookupCalls, service.lookupActor, service.lookupRequest, body)
	}
	if !strings.Contains(body, "approved@example.test") {
		t.Fatalf("approved projection missing: %s", body)
	}
	for _, forbidden := range []string{"password", "salt", "secret-token-material", "impersonat", "failureReason", "nextAttemptAt", "receipt", "role"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Fatalf("response exposed %q: %s", forbidden, body)
		}
	}

	for _, path := range []string{
		"/api/v1/admin/users?limit=26",
		"/api/v1/admin/users?limit=1&limit=2",
		"/api/v1/admin/users?unknown=value",
		"/api/v1/admin/users?userId=" + userID.String() + "&email=approved@example.test",
		"/api/v1/admin/users?cursor=" + uuid.NewString() + "&email=approved@example.test",
		"/api/v1/admin/users?userId=not-a-uuid",
	} {
		status, _ = send(adminAuth, adminCookies, path)
		if status != fiber.StatusBadRequest {
			t.Fatalf("invalid lookup %q status=%d", path, status)
		}
	}
	for _, path := range []string{"/api/v1/admin/users/" + userID.String() + "/impersonate", "/api/v1/admin/users/" + userID.String() + "/password", "/api/v1/admin/roles"} {
		status, _ = send(adminAuth, adminCookies, path)
		if status != fiber.StatusNotFound {
			t.Fatalf("forbidden capability path %q status=%d", path, status)
		}
	}
	for _, event := range logs.Logs {
		if strings.Contains(event.Message, "secret-token-material") {
			t.Fatalf("token reached logs: %+v", event)
		}
	}
}

// TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit verifies
// IT-ARCH-009-007, ARCH-009, DESIGN-009 UserAdminPanel, and
// SW-REQ-054/SW-REQ-073.
func TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit(t *testing.T) {
	cfg := testConfig()
	adminID, userID, requestID := uuid.New(), uuid.New(), uuid.New()
	auth, cookies := testJWTAuthRole(t, cfg, adminID, string(repository.UserRoleAdmin), nil)
	service := &fakeUserAdminService{}
	audit := &adminAuditCoordinator{}
	controller := NewAdminController(audit, NewUserAdminController(service).AdminRoutes()...)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: auth, Audit: &auditSink{}, Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	path := "/api/v1/admin/users/" + userID.String() + "/deletion-requests/" + requestID.String() + "/retry"

	send := func(requestPath string, csrf string, body string) (int, Envelope) {
		t.Helper()
		req := httptest.NewRequest(fiber.MethodPost, requestPath, strings.NewReader(body))
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		addCookies(req, cookies)
		addCookies(req, csrfCookies)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, decodeEnvelope(t, resp.Body)
	}

	status, _ := send(path, "", "")
	if status != fiber.StatusForbidden || service.retryCalls != 0 {
		t.Fatalf("missing csrf status=%d calls=%d", status, service.retryCalls)
	}
	status, _ = send(path, token, `{"role":"admin"}`)
	if status != fiber.StatusBadRequest || service.retryCalls != 0 {
		t.Fatalf("client mutation fields status=%d calls=%d", status, service.retryCalls)
	}
	status, body := send(path, token, "")
	if status != fiber.StatusOK || service.retryCalls != 1 || service.retryUserID != userID || service.retryRequest != requestID || audit.committed != 1 {
		t.Fatalf("retry status=%d body=%+v calls=%d audit=%d", status, body, service.retryCalls, audit.committed)
	}
	if len(audit.entries) != 1 || audit.entries[0].AdminUserID != adminID || audit.entries[0].Action != "retry_deletion" || audit.entries[0].EntityType != "deletion_request" || audit.changes[0].EntityID == nil || *audit.changes[0].EntityID != requestID {
		t.Fatalf("retry audit entry=%+v changes=%+v", audit.entries, audit.changes)
	}
	encoded := string(mustJSON(t, body))
	for _, forbidden := range []string{"permanent", "failureReason", "retryCount", "nextAttemptAt", "password", "token"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("retry response exposed %q: %s", forbidden, encoded)
		}
	}

	service.err = repository.NewError(repository.ErrorKindNotFound, "cross-scope internal deletion detail", nil)
	status, body = send("/api/v1/admin/users/"+uuid.NewString()+"/deletion-requests/"+requestID.String()+"/retry", token, "")
	if status != fiber.StatusNotFound || body.Error == nil || strings.Contains(string(mustJSON(t, body)), "cross-scope") || audit.committed != 1 {
		t.Fatalf("cross-scope status=%d body=%+v audit=%d", status, body, audit.committed)
	}
}

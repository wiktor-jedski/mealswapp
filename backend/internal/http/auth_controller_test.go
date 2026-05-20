package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/handlers"
	"mealswapp/backend/internal/http/responses"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuthControllerRegisterLoginCurrentRefreshLogoutFlow(t *testing.T) {
	auth := newFakeAuthService()
	app := NewRouter(ServiceDependencies{
		Config:      config.Config{Environment: "test"},
		AuthService: auth,
	})

	registerBody := `{
		"email":"user@example.com",
		"password":"correct-password",
		"displayName":"User",
		"acceptPrivacyPolicy":true,
		"acceptTerms":true,
		"acceptNutritionDisclaimer":true,
		"privacyPolicyVersion":"privacy-v1",
		"termsVersion":"terms-v1",
		"nutritionDisclaimerVersion":"nutrition-v1"
	}`
	registerRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/auth/register", registerBody, "", false)
	defer registerRes.Body.Close()
	if registerRes.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", registerRes.StatusCode)
	}
	registerPayload := decodeEnvelope(t, registerRes)
	if !registerPayload.Success {
		t.Fatalf("expected register success, got %#v", registerPayload)
	}

	loginRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/auth/login", `{"email":"user@example.com","password":"correct-password"}`, "", false)
	defer loginRes.Body.Close()
	if loginRes.StatusCode != http.StatusOK {
		t.Fatalf("expected login 200, got %d", loginRes.StatusCode)
	}
	if len(loginRes.Cookies()) < 2 {
		t.Fatalf("expected auth cookies on login, got %#v", loginRes.Cookies())
	}
	loginPayload := decodeEnvelope(t, loginRes)
	tokens := authResultTokens(t, loginPayload)

	currentUserRes := performJSONRequest(t, app, http.MethodGet, "/api/v1/auth/me", "", tokens.AccessToken, false)
	defer currentUserRes.Body.Close()
	if currentUserRes.StatusCode != http.StatusOK {
		t.Fatalf("expected current user 200, got %d", currentUserRes.StatusCode)
	}

	refreshRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/auth/refresh", `{"refreshToken":"`+tokens.RefreshToken+`"}`, "", true)
	defer refreshRes.Body.Close()
	if refreshRes.StatusCode != http.StatusOK {
		t.Fatalf("expected refresh 200, got %d", refreshRes.StatusCode)
	}
	refreshPayload := decodeEnvelope(t, refreshRes)
	refreshed := sessionTokens(t, refreshPayload.Data)
	if refreshed.AccessToken == tokens.AccessToken || refreshed.RefreshToken == tokens.RefreshToken {
		t.Fatalf("expected token rotation, got %#v", refreshed)
	}

	logoutRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/auth/logout", `{"refreshToken":"`+refreshed.RefreshToken+`"}`, refreshed.AccessToken, true)
	defer logoutRes.Body.Close()
	if logoutRes.StatusCode != http.StatusOK {
		t.Fatalf("expected logout 200, got %d", logoutRes.StatusCode)
	}
	if len(logoutRes.Cookies()) < 2 {
		t.Fatalf("expected auth cookies cleared on logout, got %#v", logoutRes.Cookies())
	}

	afterLogoutRes := performJSONRequest(t, app, http.MethodGet, "/api/v1/auth/me", "", refreshed.AccessToken, false)
	defer afterLogoutRes.Body.Close()
	if afterLogoutRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected current user unauthorized after logout, got %d", afterLogoutRes.StatusCode)
	}
}

func TestAuthControllerRegistrationFailsWithoutRequiredConsent(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config:      config.Config{Environment: "test"},
		AuthService: newFakeAuthService(),
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/auth/register", `{
		"email":"user@example.com",
		"password":"correct-password",
		"acceptPrivacyPolicy":true,
		"acceptTerms":false,
		"acceptNutritionDisclaimer":true,
		"privacyPolicyVersion":"privacy-v1",
		"termsVersion":"terms-v1",
		"nutritionDisclaimerVersion":"nutrition-v1"
	}`, "", false)
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
	payload := decodeEnvelope(t, res)
	if payload.Error == nil || payload.Error.Code != "consent_missing" {
		t.Fatalf("expected consent_missing error, got %#v", payload)
	}
}

func TestAuthControllerInvalidLoginAndMissingCurrentUserTokenFail(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config:      config.Config{Environment: "test"},
		AuthService: newFakeAuthService(),
	})

	loginRes := performJSONRequest(t, app, http.MethodPost, "/api/v1/auth/login", `{"email":"user@example.com","password":"wrong"}`, "", false)
	defer loginRes.Body.Close()
	if loginRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected invalid login 401, got %d", loginRes.StatusCode)
	}

	currentUserRes := performJSONRequest(t, app, http.MethodGet, "/api/v1/auth/me", "", "", false)
	defer currentUserRes.Body.Close()
	if currentUserRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected missing current-user token 401, got %d", currentUserRes.StatusCode)
	}
}

func performJSONRequest(t *testing.T, app interface {
	Test(*http.Request, ...int) (*http.Response, error)
}, method string, path string, body string, accessToken string, csrf bool) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if csrf {
		req.Header.Set("X-CSRF-Token", "csrf-token")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-token"})
	}

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func authResultTokens(t *testing.T, payload responses.Envelope) handlers.SessionTokens {
	t.Helper()

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", payload.Data)
	}
	return sessionTokens(t, data["tokens"])
}

func sessionTokens(t *testing.T, value any) handlers.SessionTokens {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var tokens handlers.SessionTokens
	if err := json.Unmarshal(raw, &tokens); err != nil {
		t.Fatal(err)
	}
	return tokens
}

type fakeAuthService struct {
	user          handlers.AuthUser
	accessTokens  map[string]bool
	refreshTokens map[string]bool
	nextToken     int
}

func newFakeAuthService() *fakeAuthService {
	return &fakeAuthService{
		user: handlers.AuthUser{
			ID:            uuid.New(),
			Email:         "user@example.com",
			DisplayName:   "User",
			EmailVerified: false,
			Role:          "user",
		},
		accessTokens:  make(map[string]bool),
		refreshTokens: make(map[string]bool),
	}
}

func (service *fakeAuthService) Register(ctx context.Context, command handlers.RegisterCommand) (handlers.AuthResult, error) {
	if !command.AcceptPrivacyPolicy || !command.AcceptTerms || !command.AcceptNutritionDisclaimer {
		return handlers.AuthResult{}, apperrors.AppError{
			Category: apperrors.CategoryValidation,
			Code:     "consent_missing",
			Message:  "Required consent is missing",
			Status:   http.StatusBadRequest,
		}
	}
	service.user.Email = command.Email
	return handlers.AuthResult{User: service.user, Tokens: service.issueTokens()}, nil
}

func (service *fakeAuthService) Login(ctx context.Context, command handlers.LoginCommand) (handlers.AuthResult, error) {
	if command.Email != service.user.Email || command.Password != "correct-password" {
		return handlers.AuthResult{}, apperrors.Unauthorized("Invalid credentials")
	}
	return handlers.AuthResult{User: service.user, Tokens: service.issueTokens()}, nil
}

func (service *fakeAuthService) Logout(ctx context.Context, command handlers.LogoutCommand) error {
	delete(service.accessTokens, command.AccessToken)
	delete(service.refreshTokens, command.RefreshToken)
	return nil
}

func (service *fakeAuthService) Refresh(ctx context.Context, command handlers.RefreshCommand) (handlers.SessionTokens, error) {
	if !service.refreshTokens[command.RefreshToken] {
		return handlers.SessionTokens{}, apperrors.Unauthorized("Invalid refresh token")
	}
	delete(service.refreshTokens, command.RefreshToken)
	return service.issueTokens(), nil
}

func (service *fakeAuthService) CurrentUser(ctx context.Context, command handlers.CurrentUserCommand) (handlers.AuthUser, error) {
	if !service.accessTokens[command.AccessToken] {
		return handlers.AuthUser{}, apperrors.Unauthorized("Unauthorized")
	}
	return service.user, nil
}

func (service *fakeAuthService) issueTokens() handlers.SessionTokens {
	service.nextToken++
	tokens := handlers.SessionTokens{
		AccessToken:      "access-token-" + uuid.NewString(),
		RefreshToken:     "refresh-token-" + uuid.NewString(),
		AccessExpiresAt:  time.Date(2026, 5, 19, 12, service.nextToken, 0, 0, time.UTC),
		RefreshExpiresAt: time.Date(2026, 5, 26, 12, service.nextToken, 0, 0, time.UTC),
	}
	service.accessTokens[tokens.AccessToken] = true
	service.refreshTokens[tokens.RefreshToken] = true
	return tokens
}

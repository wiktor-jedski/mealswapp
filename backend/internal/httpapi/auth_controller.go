package httpapi

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// AuthService defines core account-flow behavior for HTTP handlers.
// Implements DESIGN-006 AuthController.
type AuthService interface {
	Register(context.Context, string, string, auth.RegistrationConsent) (auth.AuthSession, error)
	Login(context.Context, string, string) (auth.AuthSession, error)
	Refresh(context.Context, string) (auth.AuthSession, error)
	Logout(context.Context, string) error
	MarkEmailVerified(context.Context, uuid.UUID) error
	RequestPasswordReset(context.Context, string) (string, error)
	ConsumePasswordReset(context.Context, string, string) error
}

// AuthController owns registration, login, logout, and refresh HTTP handlers.
// Implements DESIGN-006 AuthController.
type AuthController struct {
	service  AuthService
	sessions *AuthSessionManager
	logs     observability.LogSink
}

// Implements DESIGN-006 AuthController compile-time route controller contract.
var _ Controller = (*AuthController)(nil)

// NewAuthController creates account-flow HTTP handlers.
// Implements DESIGN-006 AuthController.
func NewAuthController(service AuthService, sessions *AuthSessionManager) *AuthController {
	return &AuthController{service: service, sessions: sessions}
}

// WithLogSink attaches structured warning logs for best-effort auth cleanup.
// Implements DESIGN-014 LogAggregator.
func (c *AuthController) WithLogSink(logs observability.LogSink) *AuthController {
	c.logs = logs
	return c
}

// Routes returns versioned auth routes with CSRF and rate-limit policies.
// Implements DESIGN-006 AuthController.
func (c *AuthController) Routes() []RouteDefinition {
	failedLoginRule := FailedLoginRule()
	return []RouteDefinition{
		{Method: fiber.MethodPost, Path: "/auth/register", ExemptCSRF: true, RequiresAudit: true, Validate: ValidateJSON(validateRegisterBody), Handler: c.Register},
		{Method: fiber.MethodPost, Path: "/auth/login", ExemptCSRF: true, RequiresAudit: true, Validate: ValidateJSON(validateLoginBody), RateLimit: &failedLoginRule, Handler: c.Login},
		{Method: fiber.MethodPost, Path: "/auth/logout", RequiresAuth: true, RequiresCSRF: true, Handler: c.Logout},
		{Method: fiber.MethodPost, Path: "/auth/refresh", ExemptCSRF: true, Handler: c.Refresh},
		{Method: fiber.MethodPost, Path: "/auth/verify-email", RequiresAuth: true, RequiresCSRF: true, RequiresAudit: true, Handler: c.VerifyEmail},
		{Method: fiber.MethodPost, Path: "/auth/password-reset/request", ExemptCSRF: true, Validate: ValidateJSON(validatePasswordResetRequestBody), Handler: c.RequestPasswordReset},
		{Method: fiber.MethodPost, Path: "/auth/password-reset/consume", ExemptCSRF: true, RequiresAudit: true, Validate: ValidateJSON(validatePasswordResetConsumeBody), Handler: c.ConsumePasswordReset},
	}
}

// Register creates an account and authenticated session.
// Implements DESIGN-006 AuthController.
func (c *AuthController) Register(ctx *fiber.Ctx) error {
	var req authCredentialRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	session, err := c.service.Register(ctx.UserContext(), req.Email, req.Password, auth.RegistrationConsent{PrivacyPolicyVersion: req.PrivacyPolicyVersion, TermsVersion: req.TermsVersion})
	if err != nil {
		return mapAuthError(ctx, err)
	}
	if err := c.sessions.SetAuthenticatedCookies(ctx, toHTTPAuthSession(session)); err != nil {
		return err
	}
	return ctx.Status(fiber.StatusCreated).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: authSessionData(session)})
}

// Login authenticates an account and writes session cookies.
// Implements DESIGN-006 AuthController.
func (c *AuthController) Login(ctx *fiber.Ctx) error {
	var req authCredentialRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	session, err := c.service.Login(ctx.UserContext(), req.Email, req.Password)
	if err != nil {
		return mapAuthError(ctx, err)
	}
	if err := c.sessions.SetAuthenticatedCookies(ctx, toHTTPAuthSession(session)); err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: authSessionData(session)})
}

// Logout revokes the refresh session and clears cookies.
// Implements DESIGN-006 AuthController.
func (c *AuthController) Logout(ctx *fiber.Ctx) error {
	if err := c.service.Logout(ctx.UserContext(), ctx.Cookies(c.sessions.cfg.Account.RefreshCookieName)); err != nil {
		return mapAuthError(ctx, err)
	}
	if err := c.sessions.ClearAuthenticatedCookies(ctx); err != nil {
		return err
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// Refresh rotates refresh tokens and writes new session cookies.
// Implements DESIGN-006 AuthController.
func (c *AuthController) Refresh(ctx *fiber.Ctx) error {
	session, err := c.service.Refresh(ctx.UserContext(), ctx.Cookies(c.sessions.cfg.Account.RefreshCookieName))
	if err != nil {
		if clearErr := c.sessions.ClearAuthenticatedCookies(ctx); clearErr != nil {
			c.warn(ctx, "auth_refresh_cookie_clear_failed", map[string]any{"error": clearErr.Error()})
		}
		return mapAuthError(ctx, err)
	}
	if err := c.sessions.SetAuthenticatedCookies(ctx, toHTTPAuthSession(session)); err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: authSessionData(session)})
}

// VerifyEmail updates the verified-login projection.
// Implements DESIGN-006 AuthController.
func (c *AuthController) VerifyEmail(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	if err := c.service.MarkEmailVerified(ctx.UserContext(), user.UserID); err != nil {
		return mapAuthError(ctx, err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"hasVerifiedLoginMethod": true}})
}

// RequestPasswordReset stores a reset token when an account exists and always returns generic success.
// Implements DESIGN-006 AuthController.
func (c *AuthController) RequestPasswordReset(ctx *fiber.Ctx) error {
	var req passwordResetRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	if _, err := c.service.RequestPasswordReset(ctx.UserContext(), req.Email); err != nil {
		return mapAuthError(ctx, err)
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"accepted": true}})
}

// ConsumePasswordReset changes the password and clears authorization state.
// Implements DESIGN-006 AuthController.
func (c *AuthController) ConsumePasswordReset(ctx *fiber.Ctx) error {
	var req passwordResetConsumeRequest
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	if err := c.service.ConsumePasswordReset(ctx.UserContext(), req.Token, req.NewPassword); err != nil {
		return mapAuthError(ctx, err)
	}
	if err := c.sessions.ClearAuthenticatedCookies(ctx); err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"reset": true}})
}

// authCredentialRequest carries register/login JSON fields.
// Implements DESIGN-006 AuthController.
type authCredentialRequest struct {
	Email                string `json:"email"`
	Password             string `json:"password"`
	PrivacyPolicyVersion string `json:"privacyPolicyVersion"`
	TermsVersion         string `json:"termsVersion"`
}

// passwordResetRequest carries reset request inputs.
// Implements DESIGN-006 AuthController.
type passwordResetRequest struct {
	Email string `json:"email"`
}

// passwordResetConsumeRequest carries reset consume inputs.
// Implements DESIGN-006 AuthController.
type passwordResetConsumeRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// toHTTPAuthSession maps core auth sessions to cookie writer input.
// Implements DESIGN-006 AuthController.
func toHTTPAuthSession(session auth.AuthSession) AuthSessionTokens {
	return AuthSessionTokens{AccessToken: session.AccessToken, RefreshToken: session.RefreshToken, AccessExpiresAt: session.AccessExpiresAt, RefreshExpiresAt: session.RefreshExpiresAt}
}

// authSessionData returns safe non-token auth response data.
// Implements DESIGN-006 AuthController.
func authSessionData(session auth.AuthSession) map[string]any {
	return map[string]any{"userId": session.UserID.String(), "role": session.Role, "hasVerifiedLoginMethod": session.HasVerifiedLoginMethod, "accessExpiresAt": session.AccessExpiresAt, "refreshExpiresAt": session.RefreshExpiresAt}
}

// mapAuthError maps core auth failures to safe gateway errors.
// Implements DESIGN-006 AuthController.
func mapAuthError(ctx *fiber.Ctx, err error) error {
	if errors.Is(err, auth.ErrInvalidCredentials) {
		return InvalidCredentialsError()
	}
	if errors.Is(err, auth.ErrSessionExpired) {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "session_expired", Message: "authentication required"}
	}
	if errors.Is(err, auth.ErrTokenReuseDetected) {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "token_reuse_detected", Message: "authentication required"}
	}
	if errors.Is(err, auth.ErrPasswordResetInvalid) {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "auth", Code: "password_reset_invalid", Message: "request validation failed"}
	}
	var locked *auth.AccountLocked
	if errors.As(err, &locked) {
		return AccountLockedError(ctx, locked.RetryAfter)
	}
	if err.Error() == "consent_missing" || err.Error() == "consent_version_stale" || err.Error() == "consent_version_invalid" {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: err.Error(), Message: "request validation failed"}
	}
	return err
}

// warn emits optional structured controller warnings.
// Implements DESIGN-014 LogAggregator.
func (c *AuthController) warn(ctx *fiber.Ctx, message string, fields map[string]any) {
	if c.logs == nil {
		return
	}
	if err := c.logs.Log(ctx.UserContext(), observability.LogEvent{RequestID: requestID(ctx), Service: "api", Level: "warning", Message: message, Fields: fields, CreatedAt: time.Now()}); err != nil {
		_, _ = observabilityFallbackWriter.Write([]byte(err.Error() + "\n"))
	}
}

// validateRegisterBody validates registration JSON before service dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-015 ConsentManager.
func validateRegisterBody(body map[string]any) error {
	if _, ok := body["email"].(string); !ok {
		return errors.New("email is required")
	}
	if _, ok := body["password"].(string); !ok {
		return errors.New("password is required")
	}
	privacy, ok := body["privacyPolicyVersion"].(string)
	if !ok {
		return errors.New("privacy policy version is required")
	}
	terms, ok := body["termsVersion"].(string)
	if !ok {
		return errors.New("terms version is required")
	}
	if _, err := security.NormalizeInput(security.InputFieldConsentVersion, privacy); err != nil {
		return err
	}
	_, err := security.NormalizeInput(security.InputFieldConsentVersion, terms)
	return err
}

// validateLoginBody validates credential JSON before service dispatch.
// Implements DESIGN-010 RequestValidator.
func validateLoginBody(body map[string]any) error {
	if _, ok := body["email"].(string); !ok {
		return errors.New("email is required")
	}
	if _, ok := body["password"].(string); !ok {
		return errors.New("password is required")
	}
	return nil
}

// validatePasswordResetRequestBody validates reset request JSON.
// Implements DESIGN-010 RequestValidator.
func validatePasswordResetRequestBody(body map[string]any) error {
	if email, ok := body["email"].(string); !ok || strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}
	return nil
}

// validatePasswordResetConsumeBody validates reset consume JSON.
// Implements DESIGN-010 RequestValidator.
func validatePasswordResetConsumeBody(body map[string]any) error {
	if token, ok := body["token"].(string); !ok || strings.TrimSpace(token) == "" {
		return errors.New("token is required")
	}
	if password, ok := body["newPassword"].(string); !ok || password == "" {
		return errors.New("new password is required")
	}
	return nil
}

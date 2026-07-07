package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// ErrOAuthProviderUnavailable means no configured provider gateway can serve the request.
// Implements DESIGN-006 OAuthHandler and DESIGN-017 ErrorMessageMapper.
var ErrOAuthProviderUnavailable = errors.New("OAuth provider gateway is not configured")

// OAuthService defines provider login behavior for HTTP handlers.
// Implements DESIGN-006 OAuthHandler.
type OAuthService interface {
	CompleteOAuth(context.Context, string, auth.OAuthProfile) (auth.OAuthResult, error)
}

// OAuthProviderGateway abstracts goth provider start and callback behavior.
// Implements DESIGN-006 OAuthHandler.
type OAuthProviderGateway interface {
	StartOAuth(context.Context, string, string) (string, error)
	CompleteOAuth(context.Context, string, map[string]string) (auth.OAuthProfile, error)
}

// OAuthController owns OAuth start/callback HTTP handlers.
// Implements DESIGN-006 OAuthHandler.
type OAuthController struct {
	service  OAuthService
	gateway  OAuthProviderGateway
	sessions *AuthSessionManager
	now      func() time.Time
}

// Implements DESIGN-006 OAuthHandler compile-time route controller contract.
var _ Controller = (*OAuthController)(nil)

// NewOAuthController creates goth-backed OAuth HTTP handlers.
// Implements DESIGN-006 OAuthHandler.
func NewOAuthController(service OAuthService, gateway OAuthProviderGateway, sessions *AuthSessionManager) *OAuthController {
	return &OAuthController{service: service, gateway: gateway, sessions: sessions, now: time.Now}
}

// Routes returns versioned OAuth routes with explicit CSRF policy.
// Implements DESIGN-006 OAuthHandler.
func (c *OAuthController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/auth/oauth/:provider/start", Validate: ValidatePath("provider", func(value string) error { _, err := normalizeOAuthProviderParam(value); return err }), Handler: c.StartOAuth},
		{Method: fiber.MethodGet, Path: "/auth/oauth/:provider/callback", Validate: ValidatePath("provider", func(value string) error { _, err := normalizeOAuthProviderParam(value); return err }), Handler: c.CompleteOAuth},
	}
}

// StartOAuth begins the provider authorization redirect.
// Implements DESIGN-006 OAuthHandler.
func (c *OAuthController) StartOAuth(ctx *fiber.Ctx) error {
	provider, err := normalizeOAuthProviderParam(ctx.Params("provider"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "oauth_provider_invalid", Message: "request validation failed"}
	}
	state, err := generateOAuthState()
	if err != nil {
		return err
	}
	location, err := c.gateway.StartOAuth(ctx.UserContext(), provider, state)
	if err != nil {
		return mapOAuthGatewayError(err)
	}
	c.setOAuthCookie(ctx, oauthStateCookieName, state)
	c.setOAuthCookie(ctx, oauthReturnCookieName, safeOAuthReturnPath(ctx.Query("return_to")))
	return ctx.Redirect(location, fiber.StatusFound)
}

// CompleteOAuth finishes provider login and writes authenticated cookies.
// Implements DESIGN-006 OAuthHandler.
func (c *OAuthController) CompleteOAuth(ctx *fiber.Ctx) error {
	provider, err := normalizeOAuthProviderParam(ctx.Params("provider"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "oauth_provider_invalid", Message: "request validation failed"}
	}
	returnPath := safeOAuthReturnPath(ctx.Cookies(oauthReturnCookieName))
	defer c.clearOAuthCookie(ctx, oauthStateCookieName)
	defer c.clearOAuthCookie(ctx, oauthReturnCookieName)
	if !validOAuthState(ctx.Cookies(oauthStateCookieName), ctx.Query("state")) {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "auth", Code: "oauth_state_invalid", Message: "request validation failed"}
	}
	profile, err := c.gateway.CompleteOAuth(ctx.UserContext(), provider, queryParams(ctx))
	if err != nil {
		return mapOAuthGatewayError(err)
	}
	result, err := c.service.CompleteOAuth(ctx.UserContext(), provider, profile)
	if err != nil {
		return mapOAuthError(err)
	}
	if err := c.sessions.SetAuthenticatedCookies(ctx, toHTTPAuthSession(result.Session)); err != nil {
		return err
	}
	return ctx.Redirect(strings.TrimRight(c.sessions.cfg.FrontendOrigin, "/")+returnPath, fiber.StatusFound)
}

// Implements DESIGN-006 OAuthHandler callback cookie names and lifetime.
const (
	oauthStateCookieName  = "mealswapp_oauth_state"
	oauthReturnCookieName = "mealswapp_oauth_return"
	oauthStateTTL         = 10 * time.Minute
)

// generateOAuthState creates an unguessable CSRF token for provider redirects.
// Implements DESIGN-006 OAuthHandler.
func generateOAuthState() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// validOAuthState verifies callback state without leaking token timing.
// Implements DESIGN-006 OAuthHandler.
func validOAuthState(expected string, actual string) bool {
	if expected == "" || actual == "" || len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

// safeOAuthReturnPath restricts OAuth return targets to relative frontend paths.
// Implements DESIGN-006 OAuthHandler.
func safeOAuthReturnPath(value string) string {
	if strings.TrimSpace(value) == "" {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "/"
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
		return "/"
	}
	parsed.Fragment = ""
	return parsed.RequestURI()
}

// setOAuthCookie stores short-lived OAuth callback metadata outside JavaScript.
// Implements DESIGN-006 OAuthHandler.
func (c *OAuthController) setOAuthCookie(ctx *fiber.Ctx, name string, value string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  c.now().Add(oauthStateTTL),
		MaxAge:   int(oauthStateTTL.Seconds()),
		HTTPOnly: true,
		Secure:   c.sessions.cfg.EnforceTLS,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/api/v1/auth/oauth",
	})
}

// clearOAuthCookie removes OAuth callback metadata after callback attempts.
// Implements DESIGN-006 OAuthHandler.
func (c *OAuthController) clearOAuthCookie(ctx *fiber.Ctx, name string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   c.sessions.cfg.EnforceTLS,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/api/v1/auth/oauth",
	})
}

// normalizeOAuthProviderParam validates route provider names.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 InputNormalizer.
func normalizeOAuthProviderParam(provider string) (string, error) {
	normalized, err := security.NormalizeInput(security.InputFieldOAuthProvider, provider)
	if err != nil {
		return "", err
	}
	return normalized.Value, nil
}

// queryParams copies callback query data at the HTTP trust boundary.
// Implements DESIGN-006 OAuthHandler.
func queryParams(ctx *fiber.Ctx) map[string]string {
	params := map[string]string{}
	ctx.Context().QueryArgs().VisitAll(func(key []byte, value []byte) {
		params[string(key)] = string(value)
	})
	return params
}

// mapOAuthError maps OAuth service failures to safe gateway errors.
// Implements DESIGN-006 OAuthHandler.
func mapOAuthError(err error) error {
	var linkRequired *auth.OAuthLinkRequired
	if errors.As(err, &linkRequired) {
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "auth", Code: "oauth_link_required", Message: "account linking required"}
	}
	if errors.Is(err, auth.ErrOAuthProviderMismatch) {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "auth", Code: "oauth_provider_mismatch", Message: "request validation failed"}
	}
	return err
}

// mapOAuthGatewayError maps provider setup/exchange failures to safe responses.
// Implements DESIGN-006 OAuthHandler and DESIGN-017 ErrorMessageMapper.
func mapOAuthGatewayError(err error) error {
	if errors.Is(err, ErrOAuthProviderUnavailable) {
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "oauth_provider_unavailable", Message: "sign-in provider is temporarily unavailable", Retryable: true}
	}
	return err
}

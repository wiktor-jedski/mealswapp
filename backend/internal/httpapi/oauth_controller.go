package httpapi

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

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
}

var _ Controller = (*OAuthController)(nil)

// NewOAuthController creates goth-backed OAuth HTTP handlers.
// Implements DESIGN-006 OAuthHandler.
func NewOAuthController(service OAuthService, gateway OAuthProviderGateway, sessions *AuthSessionManager) *OAuthController {
	return &OAuthController{service: service, gateway: gateway, sessions: sessions}
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
	location, err := c.gateway.StartOAuth(ctx.UserContext(), provider, requestID(ctx))
	if err != nil {
		return err
	}
	return ctx.Redirect(location, fiber.StatusFound)
}

// CompleteOAuth finishes provider login and writes authenticated cookies.
// Implements DESIGN-006 OAuthHandler.
func (c *OAuthController) CompleteOAuth(ctx *fiber.Ctx) error {
	provider, err := normalizeOAuthProviderParam(ctx.Params("provider"))
	if err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "oauth_provider_invalid", Message: "request validation failed"}
	}
	profile, err := c.gateway.CompleteOAuth(ctx.UserContext(), provider, queryParams(ctx))
	if err != nil {
		return err
	}
	result, err := c.service.CompleteOAuth(ctx.UserContext(), provider, profile)
	if err != nil {
		return mapOAuthError(err)
	}
	if err := c.sessions.SetAuthenticatedCookies(ctx, toHTTPAuthSession(result.Session)); err != nil {
		return err
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: authSessionData(result.Session)})
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

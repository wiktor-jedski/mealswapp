package httpapi

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
)

// AuthSessionTokens contains signed tokens and their browser-cookie expiries.
// Implements DESIGN-006 SessionManager.
type AuthSessionTokens struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// AuthSessionManager owns authenticated access and refresh cookies.
// Implements DESIGN-006 SessionManager.
type AuthSessionManager struct {
	cfg  config.Config
	csrf *CSRFManager
	now  func() time.Time
}

// NewAuthSessionManager creates authenticated cookie handling around CSRF session state.
// Implements DESIGN-006 SessionManager.
func NewAuthSessionManager(cfg config.Config, csrf *CSRFManager) *AuthSessionManager {
	if csrf == nil {
		csrf = NewCSRFManager(cfg, nil)
	}
	return &AuthSessionManager{cfg: cfg, csrf: csrf, now: time.Now}
}

// SetAuthenticatedCookies regenerates authorization state and writes token cookies.
// Implements DESIGN-006 SessionManager.
func (m *AuthSessionManager) SetAuthenticatedCookies(ctx *fiber.Ctx, tokens AuthSessionTokens) error {
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || !tokens.AccessExpiresAt.After(m.now()) || !tokens.RefreshExpiresAt.After(m.now()) {
		return AppError{HTTPStatus: fiber.StatusInternalServerError, Category: "auth", Code: "session_token_invalid", Message: "session token is invalid"}
	}
	if err := m.csrf.RegenerateAuthorizationState(ctx); err != nil {
		return err
	}
	m.setCookie(ctx, m.cfg.Account.AccessCookieName, tokens.AccessToken, tokens.AccessExpiresAt)
	m.setCookie(ctx, m.cfg.Account.RefreshCookieName, tokens.RefreshToken, tokens.RefreshExpiresAt)
	return nil
}

// ClearAuthenticatedCookies invalidates authorization state and clears token cookies.
// Implements DESIGN-006 SessionManager.
func (m *AuthSessionManager) ClearAuthenticatedCookies(ctx *fiber.Ctx) error {
	if err := m.csrf.InvalidateAuthorizationState(ctx); err != nil {
		return err
	}
	m.clearCookie(ctx, m.cfg.Account.AccessCookieName)
	m.clearCookie(ctx, m.cfg.Account.RefreshCookieName)
	return nil
}

// setCookie writes strict authenticated token cookies.
// Implements DESIGN-006 SessionManager.
func (m *AuthSessionManager) setCookie(ctx *fiber.Ctx, name string, value string, expiresAt time.Time) {
	maxAge := max(int(expiresAt.Sub(m.now()).Seconds()), 1)
	ctx.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HTTPOnly: true,
		Secure:   m.cfg.EnforceTLS,
		SameSite: fiber.CookieSameSiteStrictMode,
		Path:     "/",
	})
}

// clearCookie removes authenticated token cookies.
// Implements DESIGN-006 SessionManager.
func (m *AuthSessionManager) clearCookie(ctx *fiber.Ctx, name string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   m.cfg.EnforceTLS,
		SameSite: fiber.CookieSameSiteStrictMode,
		Path:     "/",
	})
}

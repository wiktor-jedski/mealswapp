package httpapi

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Implements DESIGN-010 CSRFValidator browser-session constants.
const (
	csrfTokenContextKey = "mealswapp.csrf.token"
	csrfCookieName      = "mealswapp_csrf"
	sessionCookieName   = "mealswapp_session"
)

// CSRFManager owns Fiber CSRF middleware and its authenticated session store.
// Implements DESIGN-010 CSRFValidator and DESIGN-006 SessionManager.
type CSRFManager struct {
	sessionStore *session.Store
	middleware   fiber.Handler
}

// NewCSRFManager configures session-bound synchronizer tokens for browser clients.
// Implements DESIGN-010 CSRFValidator and DESIGN-006 SessionManager.
func NewCSRFManager(cfg config.Config, audit security.AuditLogger) *CSRFManager {
	store := session.New(session.Config{
		Expiration:     30 * time.Minute,
		KeyLookup:      "cookie:" + sessionCookieName,
		CookieSecure:   cfg.EnforceTLS,
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
	})
	manager := &CSRFManager{sessionStore: store}
	manager.middleware = csrf.New(csrf.Config{
		KeyLookup:      "header:X-CSRF-Token",
		CookieName:     csrfCookieName,
		CookieSecure:   cfg.EnforceTLS,
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
		Expiration:     time.Hour,
		Session:        store,
		ContextKey:     csrfTokenContextKey,
		ErrorHandler: func(ctx *fiber.Ctx, _ error) error {
			security.RecordAuditBestEffort(ctx.UserContext(), audit, security.AuditLogEntry{
				RequestID: requestID(ctx), Action: "csrf.validate", Resource: routeTemplate(ctx),
				Outcome: "failure", IP: ctx.IP(), UserAgent: ctx.Get("User-Agent"), CreatedAt: time.Now(),
			})
			return AppError{HTTPStatus: fiber.StatusForbidden, Category: "auth", Code: "csrf_failed", Message: "csrf validation failed"}
		},
	})
	return manager
}

// IssueToken applies Fiber CSRF middleware to safe requests so the SPA can receive a token.
// Implements DESIGN-010 CSRFValidator.
func (m *CSRFManager) IssueToken(ctx *fiber.Ctx) error {
	if isMutation(ctx.Method()) {
		return ctx.Next()
	}
	return m.middleware(ctx)
}

// Validate applies Fiber CSRF middleware to route-scoped protected mutations.
// Implements DESIGN-010 CSRFValidator.
func (m *CSRFManager) Validate(ctx *fiber.Ctx) error {
	return m.middleware(ctx)
}

// RegenerateAuthorizationState rotates the Fiber session after login, refresh, or reset.
// Implements DESIGN-006 SessionManager.
func (m *CSRFManager) RegenerateAuthorizationState(ctx *fiber.Ctx) error {
	sess, err := m.sessionStore.Get(ctx)
	if err != nil {
		return err
	}
	return sess.Reset()
}

// InvalidateAuthorizationState destroys the Fiber session during logout.
// Implements DESIGN-006 SessionManager.
func (m *CSRFManager) InvalidateAuthorizationState(ctx *fiber.Ctx) error {
	sess, err := m.sessionStore.Get(ctx)
	if err != nil {
		return err
	}
	return sess.Destroy()
}

// csrfToken returns the Fiber-managed synchronizer token for safe SPA delivery.
// Implements DESIGN-006 AuthController.
func csrfToken(ctx *fiber.Ctx) error {
	token, ok := ctx.Locals(csrfTokenContextKey).(string)
	if !ok || token == "" {
		return errors.New("CSRF token missing from request context")
	}
	return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{"csrfToken": token}})
}

package httpapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-006 JWTManager authenticated request context key.
const authenticatedUserLocal = "authenticatedUser"

// AccessTokenValidator validates signed access-token claims.
// Implements DESIGN-006 JWTManager.
type AccessTokenValidator interface {
	ValidateAccessToken(context.Context, string) (auth.AccessTokenClaims, error)
}

// AuthenticatedUser is server-derived request identity metadata.
// Implements DESIGN-006 JWTManager.
type AuthenticatedUser struct {
	UserID                 uuid.UUID
	Role                   string
	HasVerifiedLoginMethod bool
	SessionID              uuid.UUID
	RefreshFamilyID        uuid.UUID
}

// JWTAuthenticator validates auth cookies against JWT and repository session state.
// Implements DESIGN-006 JWTManager.
type JWTAuthenticator struct {
	cfg      config.Config
	tokens   AccessTokenValidator
	sessions repository.SessionRepository
	now      func() time.Time
}

// NewJWTAuthenticator creates protected-route cookie authentication.
// Implements DESIGN-006 JWTManager.
func NewJWTAuthenticator(cfg config.Config, tokens AccessTokenValidator, sessions repository.SessionRepository) *JWTAuthenticator {
	return &JWTAuthenticator{cfg: cfg, tokens: tokens, sessions: sessions, now: time.Now}
}

// Authenticate validates cookies and returns server-derived identity claims.
// Implements DESIGN-006 JWTManager.
func (a *JWTAuthenticator) Authenticate(ctx context.Context, accessToken string, refreshToken string) (AuthenticatedUser, error) {
	if a == nil || a.tokens == nil || a.sessions == nil || accessToken == "" || refreshToken == "" {
		return AuthenticatedUser{}, errors.New("authentication required")
	}
	claims, err := a.tokens.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return AuthenticatedUser{}, err
	}
	session, err := a.sessions.GetSessionByRefreshTokenHash(ctx, auth.HashRefreshToken(refreshToken))
	if err != nil {
		return AuthenticatedUser{}, err
	}
	if session.ID != claims.SessionID || session.UserID != claims.UserID || session.RefreshFamilyID != claims.RefreshFamilyID {
		return AuthenticatedUser{}, errors.New("session claims mismatch")
	}
	if session.RevokedAt != nil || !session.RefreshExpiresAt.After(a.now()) {
		return AuthenticatedUser{}, errors.New("session is not active")
	}
	return AuthenticatedUser{UserID: claims.UserID, Role: claims.Role, HasVerifiedLoginMethod: claims.HasVerifiedLoginMethod, SessionID: claims.SessionID, RefreshFamilyID: claims.RefreshFamilyID}, nil
}

// requireAuth validates JWT cookies before protected handlers.
// Implements DESIGN-006 JWTManager.
func requireAuth(authenticator *JWTAuthenticator) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if authenticator == nil {
			return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
		}
		user, err := authenticator.Authenticate(ctx.UserContext(), ctx.Cookies(authenticator.cfg.Account.AccessCookieName), ctx.Cookies(authenticator.cfg.Account.RefreshCookieName))
		if err != nil {
			fmt.Printf("auth failed: %v (access: %q, refresh: %q)\n", err, ctx.Cookies(authenticator.cfg.Account.AccessCookieName), ctx.Cookies(authenticator.cfg.Account.RefreshCookieName))
			return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required", Cause: err}
		}
		ctx.Locals(authenticatedUserLocal, user)
		return ctx.Next()
	}
}

// optionalAuth attaches server-derived identity for public routes when valid cookies are present.
// Implements DESIGN-006 JWTManager and DESIGN-008 SearchHistoryRepository.
func optionalAuth(authenticator *JWTAuthenticator) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if authenticator == nil {
			return ctx.Next()
		}
		accessToken := ctx.Cookies(authenticator.cfg.Account.AccessCookieName)
		refreshToken := ctx.Cookies(authenticator.cfg.Account.RefreshCookieName)
		if accessToken == "" || refreshToken == "" {
			return ctx.Next()
		}
		user, err := authenticator.Authenticate(ctx.UserContext(), accessToken, refreshToken)
		if err == nil {
			ctx.Locals(authenticatedUserLocal, user)
		} else {
			fmt.Printf("optionalAuth failed: %v (access: %q, refresh: %q)\n", err, accessToken, refreshToken)
		}
		return ctx.Next()
	}
}

// authenticatedUser returns the server-derived request identity when present.
// Implements DESIGN-006 JWTManager.
func authenticatedUser(ctx *fiber.Ctx) (AuthenticatedUser, bool) {
	user, ok := ctx.Locals(authenticatedUserLocal).(AuthenticatedUser)
	return user, ok
}

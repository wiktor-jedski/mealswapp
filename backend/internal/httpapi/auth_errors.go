package httpapi

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
)

// InvalidCredentialsError returns the generic failed-login response.
// Implements DESIGN-006 AccountLockoutTracker.
func InvalidCredentialsError() AppError {
	return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "invalid_credentials", Message: auth.GenericInvalidCredentialMessage()}
}

// AccountLockedError returns locked-account retry metadata without exposing identity.
// Implements DESIGN-006 AccountLockoutTracker.
func AccountLockedError(ctx *fiber.Ctx, retryAfter time.Duration) AppError {
	seconds := max(int(retryAfter.Seconds()), 1)
	ctx.Set("Retry-After", strconv.Itoa(seconds))
	return AppError{HTTPStatus: fiber.StatusTooManyRequests, Category: "auth", Code: "account_locked", Message: auth.GenericInvalidCredentialMessage(), Retryable: true}
}

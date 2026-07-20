package httpapi

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
)

// InvalidCredentialsError returns the generic failed-login response.
// Implements DESIGN-006 AccountLockoutTracker.
func InvalidCredentialsError() AppError {
	return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "invalid_credentials", Message: auth.GenericInvalidCredentialMessage}
}

// AccountLockedError returns locked-account retry metadata without exposing identity.
// Implements DESIGN-006 AccountLockoutTracker.
func AccountLockedError(ctx *fiber.Ctx, retryAfter time.Duration) AppError {
	return retryableTooManyRequests(ctx, retryAfter, "auth", "account_locked", auth.GenericInvalidCredentialMessage)
}

// retryableTooManyRequests applies the shared 429 envelope and positive
// whole-second Retry-After contract.
// Implements DESIGN-004 JobStatusTracker, DESIGN-006 AccountLockoutTracker, and DESIGN-010 RateLimiter.
func retryableTooManyRequests(ctx *fiber.Ctx, retryAfter time.Duration, category, code, message string) AppError {
	ctx.Set(fiber.HeaderRetryAfter, strconv.FormatInt(max(int64(math.Ceil(retryAfter.Seconds())), 1), 10))
	return AppError{HTTPStatus: fiber.StatusTooManyRequests, Category: category, Code: code, Message: message, Retryable: true}
}

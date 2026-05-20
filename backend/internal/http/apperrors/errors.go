package apperrors

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type Category string

const (
	CategoryValidation  Category = "validation"
	CategoryAuth        Category = "auth"
	CategoryEntitlement Category = "entitlement"
	CategoryTimeout     Category = "timeout"
	CategoryServer      Category = "server"
	CategoryDependency  Category = "dependency"
	CategoryUnknown     Category = "unknown"
)

type AppError struct {
	Category  Category
	Code      string
	Message   string
	Retryable bool
	Status    int
	Cause     error
	Fields    any
}

func (err AppError) Error() string {
	if err.Cause != nil {
		return err.Cause.Error()
	}
	return err.Message
}

func (err AppError) Unwrap() error {
	return err.Cause
}

func Validation(message string, fields any) AppError {
	return AppError{Category: CategoryValidation, Code: "validation_error", Message: message, Status: http.StatusBadRequest, Fields: fields}
}

func Unauthorized(message string) AppError {
	return AppError{Category: CategoryAuth, Code: "unauthorized", Message: message, Status: http.StatusUnauthorized}
}

func Forbidden(message string) AppError {
	return AppError{Category: CategoryAuth, Code: "forbidden", Message: message, Status: http.StatusForbidden}
}

func EntitlementRequired(message string) AppError {
	return AppError{Category: CategoryEntitlement, Code: "entitlement_required", Message: message, Status: http.StatusPaymentRequired}
}

func NotFound(message string) AppError {
	return AppError{Category: CategoryUnknown, Code: "not_found", Message: message, Status: http.StatusNotFound}
}

func Conflict(message string) AppError {
	return AppError{Category: CategoryValidation, Code: "conflict", Message: message, Status: http.StatusConflict}
}

func RateLimited(message string) AppError {
	return AppError{Category: CategoryDependency, Code: "rate_limited", Message: message, Retryable: true, Status: http.StatusTooManyRequests}
}

func DependencyUnavailable(message string) AppError {
	return AppError{Category: CategoryDependency, Code: "dependency_unavailable", Message: message, Retryable: true, Status: http.StatusServiceUnavailable}
}

func Timeout(message string) AppError {
	return AppError{Category: CategoryTimeout, Code: "timeout", Message: message, Retryable: true, Status: http.StatusGatewayTimeout}
}

func Internal(cause error) AppError {
	return AppError{Category: CategoryServer, Code: "internal_error", Message: "Internal server error", Status: http.StatusInternalServerError, Cause: cause}
}

func FromFiberError(err *fiber.Error) AppError {
	switch err.Code {
	case fiber.StatusUnauthorized:
		return Unauthorized("Unauthorized")
	case fiber.StatusForbidden:
		return Forbidden("Forbidden")
	case fiber.StatusNotFound:
		return NotFound("Route not found")
	case fiber.StatusConflict:
		return Conflict("Conflict")
	case fiber.StatusTooManyRequests:
		return RateLimited("Too many requests")
	case fiber.StatusServiceUnavailable:
		return DependencyUnavailable("Service unavailable")
	case fiber.StatusGatewayTimeout:
		return Timeout("Request timed out")
	default:
		if err.Code >= 500 {
			return Internal(err)
		}
		return AppError{Category: CategoryUnknown, Code: "request_failed", Message: err.Message, Status: err.Code, Cause: err}
	}
}

func As(err error) (AppError, bool) {
	var appErr AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return AppError{}, false
}

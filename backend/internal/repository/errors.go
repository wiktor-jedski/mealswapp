package repository

import (
	"errors"
	"fmt"
)

// ErrAdminAuditPersistence identifies an audit write that must roll back its admin mutation.
// Implements DESIGN-009 AdminController fail-closed transactional audit boundary.
var ErrAdminAuditPersistence = errors.New("admin audit persistence failed")

// ErrorKind classifies repository failures for service and API mapping.
// Implements DESIGN-005 RepositoryInterfaces.
type ErrorKind string

// Implements DESIGN-005 RepositoryInterfaces.
const (
	// ErrorKindNotFound indicates that the requested record does not exist.
	ErrorKindNotFound ErrorKind = "not_found"
	// ErrorKindValidation indicates that repository input is invalid.
	ErrorKindValidation ErrorKind = "validation_error"
	// ErrorKindConflict indicates that a persistence constraint rejected the operation.
	ErrorKindConflict ErrorKind = "constraint_violation"
	// ErrorKindIdempotencyConflict indicates key reuse with a different normalized request body.
	ErrorKindIdempotencyConflict ErrorKind = "idempotency_conflict"
	// ErrorKindInvalidMicronutrientKey indicates an unsupported micronutrient vocabulary key.
	ErrorKindInvalidMicronutrientKey ErrorKind = "invalid_micronutrient_key"
	// ErrorKindUnitConversion indicates that a quantity cannot be converted as requested.
	ErrorKindUnitConversion ErrorKind = "unit_conversion_error"
	// ErrorKindConnection indicates a database communication failure.
	ErrorKindConnection ErrorKind = "connection_error"
	// ErrorKindRetryable indicates a transient database failure that may succeed when retried.
	ErrorKindRetryable ErrorKind = "retryable_error"
	// ErrorKindCanceled indicates that database work was canceled or timed out.
	ErrorKindCanceled ErrorKind = "canceled_error"
	// ErrorKindInternal indicates an unexpected database or schema failure.
	ErrorKindInternal ErrorKind = "internal_error"
	// ErrorKindRecipeCycle indicates that a recipe operation would introduce a cycle.
	ErrorKindRecipeCycle ErrorKind = "recipe_cycle_error"
)

// Error is a typed repository error with optional wrapped cause.
// Implements DESIGN-005 RepositoryInterfaces.
type Error struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

// Error returns the repository error kind, message, and optional wrapped cause.
// Implements DESIGN-005 RepositoryInterfaces.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return string(e.Kind) + ": " + e.Message
	}
	return string(e.Kind) + ": " + e.Message + ": " + e.Cause.Error()
}

// Unwrap returns the wrapped cause.
// Implements DESIGN-005 RepositoryInterfaces.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewError creates a typed repository error.
// Implements DESIGN-005 RepositoryInterfaces.
func NewError(kind ErrorKind, message string, cause error) *Error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}

// IsKind reports whether err is or wraps a repository error of kind.
// Implements DESIGN-005 RepositoryInterfaces.
func IsKind(err error, kind ErrorKind) bool {
	var repoErr *Error
	return errors.As(err, &repoErr) && repoErr.Kind == kind
}

// validationError creates a repository input-validation error.
// Implements DESIGN-005 RepositoryInterfaces.
func validationError(message string) error {
	return NewError(ErrorKindValidation, message, nil)
}

// unitConversionError creates a formatted repository unit-conversion error.
// Implements DESIGN-005 RepositoryInterfaces.
func unitConversionError(format string, args ...any) error {
	return NewError(ErrorKindUnitConversion, fmt.Sprintf(format, args...), nil)
}

package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 10
)

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Fields []FieldError
}

type Pagination struct {
	Page     int
	PageSize int
	Limit    int
	Offset   int
}

func (err ValidationError) Error() string {
	return "validation failed"
}

func (err ValidationError) HasErrors() bool {
	return len(err.Fields) > 0
}

func DecodeJSON[T any](ctx *fiber.Ctx) (T, error) {
	var payload T
	if len(ctx.Body()) == 0 {
		return payload, ValidationError{Fields: []FieldError{{
			Field:   "body",
			Code:    "required",
			Message: "Request body is required",
		}}}
	}

	if err := json.Unmarshal(ctx.Body(), &payload); err != nil {
		return payload, ValidationError{Fields: []FieldError{{
			Field:   "body",
			Code:    "malformed_json",
			Message: "Request body must be valid JSON",
		}}}
	}

	return payload, nil
}

func RequiredString(field string, value string) []FieldError {
	if strings.TrimSpace(value) == "" {
		return []FieldError{{Field: field, Code: "required", Message: fmt.Sprintf("%s is required", field)}}
	}

	return nil
}

func UUIDParam(ctx *fiber.Ctx, name string) (uuid.UUID, error) {
	value := strings.TrimSpace(ctx.Params(name))
	if value == "" {
		return uuid.Nil, ValidationError{Fields: []FieldError{{Field: name, Code: "required", Message: fmt.Sprintf("%s is required", name)}}}
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, ValidationError{Fields: []FieldError{{Field: name, Code: "invalid_uuid", Message: fmt.Sprintf("%s must be a valid UUID", name)}}}
	}

	return id, nil
}

func QueryInt(ctx *fiber.Ctx, name string, fallback int, min int, max int) (int, error) {
	value := strings.TrimSpace(ctx.Query(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, ValidationError{Fields: []FieldError{{Field: name, Code: "invalid_integer", Message: fmt.Sprintf("%s must be an integer", name)}}}
	}
	if parsed < min {
		return 0, ValidationError{Fields: []FieldError{{Field: name, Code: "too_small", Message: fmt.Sprintf("%s must be at least %d", name, min)}}}
	}
	if max > 0 && parsed > max {
		return 0, ValidationError{Fields: []FieldError{{Field: name, Code: "too_large", Message: fmt.Sprintf("%s must be at most %d", name, max)}}}
	}

	return parsed, nil
}

func PaginationFromQuery(ctx *fiber.Ctx) (Pagination, error) {
	page, err := QueryInt(ctx, "page", DefaultPage, 1, 0)
	if err != nil {
		return Pagination{}, err
	}
	pageSize, err := QueryInt(ctx, "pageSize", DefaultPageSize, 1, MaxPageSize)
	if err != nil {
		return Pagination{}, err
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
	}, nil
}

func Merge(errors ...[]FieldError) error {
	var fields []FieldError
	for _, errs := range errors {
		fields = append(fields, errs...)
	}
	if len(fields) == 0 {
		return nil
	}

	return ValidationError{Fields: fields}
}

func AsValidationError(err error) (ValidationError, bool) {
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		return validationErr, true
	}

	return ValidationError{}, false
}

package http

import (
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"

	"github.com/gofiber/fiber/v2"
)

func GlobalExceptionHandler(ctx *fiber.Ctx, err error) error {
	appErr := ClassifyServerError(err)
	return WriteErrorResponse(ctx, appErr)
}

func ClassifyServerError(err error) apperrors.AppError {
	if validationErr, ok := validation.AsValidationError(err); ok {
		return apperrors.Validation("Request validation failed", validationErr.Fields)
	}

	if appErr, ok := apperrors.As(err); ok {
		return appErr
	}

	if fiberErr, ok := err.(*fiber.Error); ok {
		return apperrors.FromFiberError(fiberErr)
	}

	return apperrors.Internal(err)
}

func WriteErrorResponse(ctx *fiber.Ctx, err apperrors.AppError) error {
	status := err.Status
	if status == 0 {
		status = fiber.StatusInternalServerError
	}

	envelope := responses.Failure(err.Code, err.Message, requestID(ctx))
	envelope.Error.Category = string(err.Category)
	envelope.Error.Retryable = err.Retryable
	envelope.Error.Fields = err.Fields

	return ctx.Status(status).JSON(envelope)
}

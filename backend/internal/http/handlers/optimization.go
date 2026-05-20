package handlers

import (
	"context"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/http/validation"
	"mealswapp/backend/internal/services/optimization"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OptimizationService interface {
	Submit(ctx context.Context, userID uuid.UUID, request optimization.DietOptimizationRequest) (optimization.SubmitResult, error)
	Get(ctx context.Context, jobID uuid.UUID) (optimization.OptimizationJob, bool, error)
}

type OptimizationUserResolver interface {
	UserIDFromAccessToken(ctx context.Context, accessToken string) (uuid.UUID, bool, error)
}

type OptimizationHandler struct {
	service  OptimizationService
	resolver OptimizationUserResolver
}

func NewOptimizationHandler(service OptimizationService, resolver OptimizationUserResolver) OptimizationHandler {
	return OptimizationHandler{service: service, resolver: resolver}
}

func (handler OptimizationHandler) Submit(ctx *fiber.Ctx) error {
	token, err := requiredBearerToken(ctx)
	if err != nil {
		return err
	}
	userID, err := handler.userID(ctx.Context(), token)
	if err != nil {
		return err
	}
	payload, err := validation.DecodeJSON[optimization.DietOptimizationRequest](ctx)
	if err != nil {
		return err
	}
	result, err := handler.service.Submit(ctx.Context(), userID, payload)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusAccepted).JSON(responses.Success(result, requestID(ctx)))
}

func (handler OptimizationHandler) GetJob(ctx *fiber.Ctx) error {
	jobID, err := validation.UUIDParam(ctx, "id")
	if err != nil {
		return err
	}
	job, ok, err := handler.service.Get(ctx.Context(), jobID)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.NotFound("Optimization job not found")
	}
	return ctx.JSON(responses.Success(job, requestID(ctx)))
}

func (handler OptimizationHandler) userID(ctx context.Context, accessToken string) (uuid.UUID, error) {
	if handler.resolver == nil {
		return uuid.Nil, nil
	}
	userID, ok, err := handler.resolver.UserIDFromAccessToken(ctx, accessToken)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, apperrors.Unauthorized("Unauthorized")
	}
	return userID, nil
}

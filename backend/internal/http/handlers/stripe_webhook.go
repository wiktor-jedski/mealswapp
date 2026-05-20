package handlers

import (
	"context"
	"errors"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/responses"
	"mealswapp/backend/internal/services/entitlements"

	"github.com/gofiber/fiber/v2"
)

type StripeWebhookService interface {
	Handle(ctx context.Context, signature string, payload []byte) (entitlements.ProcessedEvent, error)
}

type StripeWebhookHandler struct {
	service StripeWebhookService
}

func NewStripeWebhookHandler(service StripeWebhookService) StripeWebhookHandler {
	return StripeWebhookHandler{service: service}
}

func (handler StripeWebhookHandler) Handle(ctx *fiber.Ctx) error {
	result, err := handler.service.Handle(ctx.Context(), ctx.Get("Stripe-Signature"), append([]byte(nil), ctx.Body()...))
	if err != nil {
		if errors.Is(err, entitlements.ErrWebhookSignatureInvalid) {
			return apperrors.AppError{
				Category: apperrors.CategoryAuth,
				Code:     "webhook_signature_invalid",
				Message:  "Stripe webhook signature is invalid",
				Status:   fiber.StatusBadRequest,
			}
		}
		if errors.Is(err, entitlements.ErrWebhookEventInvalid) {
			return apperrors.Validation("Stripe webhook payload is invalid", []map[string]string{{"field": "body", "code": "invalid"}})
		}
		return apperrors.Internal(err)
	}
	return ctx.JSON(responses.Success(result, requestID(ctx)))
}

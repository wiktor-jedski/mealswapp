package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

// StripeWebhookProcessor defines verified provider webhook behavior.
// Implements DESIGN-007 StripeWebhookHandler.
type StripeWebhookProcessor interface {
	HandleWebhook(context.Context, subscription.WebhookRequest) (subscription.WebhookResult, error)
}

// StripeWebhookHandler owns Stripe signature verification, idempotency, and retry responses.
// Implements DESIGN-007 StripeWebhookHandler.
type StripeWebhookHandler struct {
	service StripeWebhookProcessor
	audit   security.AuditLogger
}

// Implements DESIGN-007 StripeWebhookHandler compile-time route controller contract.
var _ Controller = (*StripeWebhookHandler)(nil)

// NewStripeWebhookHandler creates Stripe webhook HTTP routes.
// Implements DESIGN-007 StripeWebhookHandler.
func NewStripeWebhookHandler(service StripeWebhookProcessor, audit security.AuditLogger) *StripeWebhookHandler {
	return &StripeWebhookHandler{service: service, audit: audit}
}

// Routes returns the unauthenticated provider webhook endpoint.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodPost, Path: "/billing/stripe/webhook", ExemptCSRF: true, Handler: h.Handle},
	}
}

// Handle processes a Stripe webhook with retry-aware status codes.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) Handle(ctx *fiber.Ctx) error {
	if h.service == nil {
		return AppError{HTTPStatus: fiber.StatusInternalServerError, Category: "dependency", Code: "stripe_webhook_unavailable", Message: "billing webhook is unavailable", Retryable: true}
	}
	result, err := h.service.HandleWebhook(ctx.UserContext(), subscription.WebhookRequest{
		Payload:    ctx.BodyRaw(),
		Signature:  ctx.Get("Stripe-Signature"),
		ReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		return h.webhookError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: map[string]any{
		"eventId":   result.EventID,
		"eventType": result.EventType,
		"duplicate": result.Duplicate,
	}})
}

// webhookError maps provider webhook failures to Stripe retry semantics.
// Implements DESIGN-007 StripeWebhookHandler and DESIGN-017 GlobalExceptionHandler.
func (h *StripeWebhookHandler) webhookError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, subscription.ErrWebhookInvalidSignature):
		security.RecordAuditBestEffort(ctx.UserContext(), h.audit, security.AuditLogEntry{
			RequestID: requestID(ctx),
			Action:    "stripe_webhook_signature_invalid",
			Resource:  "billing.stripe.webhook",
			Outcome:   "blocked",
			IP:        ctx.IP(),
			UserAgent: ctx.Get("User-Agent"),
			CreatedAt: time.Now().UTC(),
		})
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "security", Code: "webhook_signature_invalid", Message: "webhook signature is invalid"}
	case errors.Is(err, subscription.ErrWebhookInvalidPayload):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "webhook_payload_invalid", Message: "webhook payload is invalid"}
	default:
		return AppError{HTTPStatus: fiber.StatusInternalServerError, Category: "dependency", Code: "webhook_processing_failed", Message: "webhook processing failed", Retryable: true, Cause: err}
	}
}

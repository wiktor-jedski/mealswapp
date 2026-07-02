package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

// CheckoutCreator defines checkout creation behavior for HTTP handlers.
// Implements DESIGN-007 SubscriptionController.
type CheckoutCreator interface {
	CreateCheckout(context.Context, subscription.CheckoutRequest) (subscription.CheckoutResult, error)
}

// EntitlementStatusReader defines frontend-safe entitlement status reads.
// Implements DESIGN-007 SubscriptionController.
type EntitlementStatusReader interface {
	GetEntitlementStatus(context.Context, uuid.UUID) (entitlement.Status, error)
}

// SubscriptionController owns authenticated subscription and entitlement routes.
// Implements DESIGN-007 SubscriptionController.
type SubscriptionController struct {
	service CheckoutCreator
	status  EntitlementStatusReader
}

// Implements DESIGN-007 SubscriptionController compile-time route controller contract.
var _ Controller = (*SubscriptionController)(nil)

// NewSubscriptionController creates subscription checkout handlers.
// Implements DESIGN-007 SubscriptionController.
func NewSubscriptionController(service CheckoutCreator, status ...EntitlementStatusReader) *SubscriptionController {
	controller := &SubscriptionController{service: service}
	if len(status) > 0 {
		controller.status = status[0]
	}
	return controller
}

// Routes returns authenticated subscription checkout routes.
// Implements DESIGN-007 SubscriptionController.
func (c *SubscriptionController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{Method: fiber.MethodPost, Path: "/billing/checkout", RequiresAuth: true, RequiresCSRF: true, Validate: ValidateJSON(ValidateCheckoutCreateRequestBody), Handler: c.CreateCheckout},
		{Method: fiber.MethodGet, Path: "/billing/entitlement", RequiresAuth: true, Handler: c.GetEntitlement},
	}
}

// CreateCheckout starts or replays provider-hosted checkout creation.
// Implements DESIGN-007 SubscriptionController.
func (c *SubscriptionController) CreateCheckout(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	var req checkoutCreateRequestDTO
	if err := ctx.BodyParser(&req); err != nil {
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_json", Message: "invalid request body"}
	}
	if c.service == nil {
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "stripe_unavailable", Message: "billing provider is unavailable", Retryable: true}
	}
	result, err := c.service.CreateCheckout(ctx.UserContext(), subscription.CheckoutRequest{
		UserID:         user.UserID,
		IdempotencyKey: ctx.Get("Idempotency-Key"),
		Method:         ctx.Method(),
		Route:          "/billing/checkout",
		Plan:           req.Plan,
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
	})
	if err != nil {
		return checkoutError(err)
	}
	return ctx.Status(result.StatusCode).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: checkoutResponseData(result.Response)})
}

// GetEntitlement returns frontend-safe entitlement and billing state.
// Implements DESIGN-007 SubscriptionController.
func (c *SubscriptionController) GetEntitlement(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return AppError{HTTPStatus: fiber.StatusUnauthorized, Category: "auth", Code: "unauthorized", Message: "authentication required"}
	}
	if c.status == nil {
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "entitlement_unavailable", Message: "entitlement status is unavailable", Retryable: true}
	}
	status, err := c.status.GetEntitlementStatus(ctx.UserContext(), user.UserID)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusOK).JSON(Envelope{Status: "ok", RequestID: requestID(ctx), Data: entitlementStatusResponseData(status)})
}

// checkoutError maps checkout service errors to user-safe API errors.
// Implements DESIGN-007 SubscriptionController and DESIGN-017 GlobalExceptionHandler.
func checkoutError(err error) error {
	switch {
	case errors.Is(err, subscription.ErrMissingIdempotencyKey):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "idempotency_key_required", Message: "Idempotency-Key header is required"}
	case errors.Is(err, subscription.ErrIdempotencyConflict):
		return AppError{HTTPStatus: fiber.StatusConflict, Category: "validation", Code: "idempotency_key_conflict", Message: "Idempotency-Key was already used with a different request body"}
	case errors.Is(err, subscription.ErrInvalidPlan):
		return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "invalid_plan", Message: "plan is invalid"}
	case errors.Is(err, subscription.ErrStripeUnavailable):
		return AppError{HTTPStatus: fiber.StatusServiceUnavailable, Category: "dependency", Code: "stripe_unavailable", Message: "billing provider is unavailable", Retryable: true}
	default:
		return err
	}
}

// checkoutResponseData maps sanitized checkout response fields to the API envelope.
// Implements DESIGN-007 SubscriptionController.
func checkoutResponseData(response subscription.CheckoutResponse) map[string]any {
	return map[string]any{
		"checkoutSessionId": response.CheckoutSessionID,
		"checkoutUrl":       response.CheckoutURL,
		"plan":              response.Plan,
		"priceId":           response.PriceID,
		"amountCents":       response.AmountCents,
	}
}

// entitlementStatusResponseData maps sanitized entitlement state to the API envelope.
// Implements DESIGN-007 SubscriptionController.
func entitlementStatusResponseData(status entitlement.Status) map[string]any {
	data := map[string]any{
		"userId":               status.UserID.String(),
		"tier":                 status.Tier,
		"status":               status.EntitlementStatus,
		"allowedModes":         status.AllowedModes,
		"searchLimitPer24h":    status.SearchLimitPer24h,
		"usageUsed":            status.UsageUsed,
		"usageRemaining":       status.UsageRemaining,
		"usageWindowStartedAt": timeString(status.UsageWindowStartedAt),
		"trialExpiresAt":       timeString(status.TrialExpiresAt),
		"billingRecoveryState": status.BillingRecoveryState,
	}
	return data
}

// timeString returns RFC3339 UTC timestamps for optional response fields.
// Implements DESIGN-007 SubscriptionController.
func timeString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

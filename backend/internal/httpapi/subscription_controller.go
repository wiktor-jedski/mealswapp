package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
	"time"
)

// Implements DESIGN-007 SubscriptionController

// PaymentIntentRequest defines the request body for creating a checkout session.
// It explicitly excludes raw card fields to satisfy PCI scope requirements.
// Implements DESIGN-007 PaymentIntentRequest.
type PaymentIntentRequest struct {
	PriceID    string `json:"priceId"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

// Plan maps Stripe Price IDs to internal labels and amounts.
// Implements DESIGN-007 Subscription Pricing Tiers.
type Plan struct {
	Label    string
	AmountUS int // in cents
}

// CheckoutGateway creates external payment sessions.
// Implements DESIGN-007 Stripe checkout session creation.
type CheckoutGateway interface {
	CreateSession(ctx context.Context, userID uuid.UUID, priceID, successURL, cancelURL, idempotencyKey string) (string, error)
}

// IdempotencyRecord stores the result of a checkout creation for retries.
// Implements DESIGN-007 SubscriptionController.
type IdempotencyRecord struct {
	BodyHash string
	URL      string
}

// SubscriptionController manages billing routes.
// Implements DESIGN-007 SubscriptionController.
type SubscriptionController struct {
	billingConfig config.BillingConfig
	frontendUrl   string
	plans         map[string]Plan
	gateway       CheckoutGateway
	idemStore     sync.Map // map[string]IdempotencyRecord
	entManager    *subscription.EntitlementManager
	usageLimiter  *subscription.UsageLimiter
}

// NewSubscriptionController creates a controller and maps price IDs to SW-REQ-050 amounts.
// Implements DESIGN-007 SubscriptionController initialization.
func NewSubscriptionController(cfg config.Config, gateway CheckoutGateway, entManager *subscription.EntitlementManager, usageLimiter *subscription.UsageLimiter) *SubscriptionController {
	return &SubscriptionController{
		billingConfig: cfg.Billing,
		frontendUrl:   cfg.FrontendOrigin,
		plans: map[string]Plan{
			cfg.Billing.MonthlyPlanPriceID: {Label: "monthly", AmountUS: 300},
			cfg.Billing.AnnualPlanPriceID:  {Label: "annual", AmountUS: 2500},
		},
		gateway:      gateway,
		entManager:   entManager,
		usageLimiter: usageLimiter,
	}
}

// CreateCheckout handles creating a new checkout session.
// Implements DESIGN-007 CreateCheckout endpoint.
func (c *SubscriptionController) CreateCheckout(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	idempotencyKey := ctx.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Idempotency-Key header is required"})
	}

	var req PaymentIntentRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	bodyBytes, _ := json.Marshal(req)
	bodyHash := fmt.Sprintf("%x", sha256.Sum256(bodyBytes))

	if record, exists := c.idemStore.Load(idempotencyKey); exists {
		idem := record.(IdempotencyRecord)
		if idem.BodyHash != bodyHash {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Idempotency-Key reused with different body"})
		}
		return ctx.JSON(fiber.Map{"url": idem.URL})
	}

	if _, validPrice := c.plans[req.PriceID]; !validPrice {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid price ID"})
	}

	if err := c.ValidateRedirectURLs(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	url, err := c.gateway.CreateSession(ctx.Context(), user.UserID, req.PriceID, req.SuccessURL, req.CancelURL, idempotencyKey)
	if err != nil {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "billing service unavailable"})
	}

	c.idemStore.Store(idempotencyKey, IdempotencyRecord{
		BodyHash: bodyHash,
		URL:      url,
	})

	return ctx.JSON(fiber.Map{"url": url})
}

// ValidateRedirectURLs ensures the provided URLs match the configured origins.
// Implements DESIGN-007 Checkout success/cancel URL validation.
func (c *SubscriptionController) ValidateRedirectURLs(req PaymentIntentRequest) error {
	success, err := url.Parse(req.SuccessURL)
	if err != nil || success.Host == "" {
		return errors.New("invalid success URL")
	}
	cancel, err := url.Parse(req.CancelURL)
	if err != nil || cancel.Host == "" {
		return errors.New("invalid cancel URL")
	}

	frontend, _ := url.Parse(c.frontendUrl)
	if success.Host != frontend.Host || !strings.HasPrefix(req.SuccessURL, c.frontendUrl) {
		return errors.New("success URL must match frontend origin")
	}
	if cancel.Host != frontend.Host || !strings.HasPrefix(req.CancelURL, c.frontendUrl) {
		return errors.New("cancel URL must match frontend origin")
	}

	return nil
}

// Routes returns the endpoints provided by the SubscriptionController.
// Implements DESIGN-007 SubscriptionController.
func (c *SubscriptionController) Routes() []RouteDefinition {
	return []RouteDefinition{
		{
			Method:       "POST",
			Path:         "/api/v1/subscription/checkout",
			Handler:      c.CreateCheckout,
			RequiresAuth: true,
			RequiresCSRF: true,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/entitlements",
			Handler:      c.GetEntitlement,
			RequiresAuth: true,
			RequiresCSRF: false,
		},
	}
}

// GetEntitlement handles reading the user's entitlement and billing state.
// Implements DESIGN-007 SubscriptionController GetEntitlement.
func (c *SubscriptionController) GetEntitlement(ctx *fiber.Ctx) error {
	user, ok := authenticatedUser(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	ent, err := c.entManager.GetEntitlementState(ctx.Context(), user.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	usageRemaining, err := c.usageLimiter.GetUsageRemaining(ctx.Context(), &ent, "catalog", time.Now())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	allowedModes := []string{}
	if (ent.Tier == "trial" || ent.Tier == "paid") && ent.Status == "active" {
		allowedModes = []string{"catalog", "substitution:single", "substitution:multi", "daily_diet", "daily_diet_alternative"}
	} else {
		allowedModes = []string{"catalog", "substitution:single"}
	}

	var expiresAt string
	if ent.ExpiresAt != nil {
		expiresAt = ent.ExpiresAt.Format(time.RFC3339)
	}

	data := fiber.Map{
		"tier":              ent.Tier,
		"status":            ent.Status,
		"allowedModes":      allowedModes,
		"searchLimitPer24h": 3, // Hardcoded for free users as per spec
		"usageRemaining":    usageRemaining,
		"expiresAt":         expiresAt,
	}

	return ctx.JSON(fiber.Map{
		"status":    "ok",
		"requestId": ctx.Locals("requestId"), // fallback or omit if not present
		"data":      data,
	})
}

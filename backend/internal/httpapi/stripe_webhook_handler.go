package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
	"time"
)

// StripeWebhookHandler processes async payment events from Stripe.
// Implements DESIGN-007 StripeWebhookHandler.
type StripeWebhookHandler struct {
	webhookSecret string
	events        repository.StripeEventRepository
	entitlements  repository.EntitlementRepository
	audit         security.AuditLogger
}

// NewStripeWebhookHandler creates a new webhook handler.
// Implements DESIGN-007 StripeWebhookHandler.
func NewStripeWebhookHandler(cfg config.Config, events repository.StripeEventRepository, entitlements repository.EntitlementRepository, audit security.AuditLogger) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		webhookSecret: cfg.Billing.StripeWebhookSecret,
		events:        events,
		entitlements:  entitlements,
		audit:         audit,
	}
}

// Routes returns the endpoints provided by the StripeWebhookHandler.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) Routes() []RouteDefinition {
	return []RouteDefinition{
		{
			Method:       "POST",
			Path:         "/api/v1/billing/webhook",
			Handler:      h.Handle,
			RequiresAuth: false,
			RequiresCSRF: false,
			ExemptCSRF:   true,
		},
	}
}

// Handle verifies and processes incoming Stripe events.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) Handle(ctx *fiber.Ctx) error {
	payload := ctx.Body()
	sigHeader := ctx.Get("Stripe-Signature")
	
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, h.webhookSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		security.RecordAuditBestEffort(ctx.UserContext(), h.audit, security.AuditLogEntry{
			RequestID: requestID(ctx), Action: "api.billing.webhook", Resource: "webhook", Outcome: "failure", IP: ctx.IP(), UserAgent: ctx.Get("User-Agent"), CreatedAt: time.Now(),
		})
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	sanitizedPayload, _ := json.Marshal(map[string]string{
		"event_id":   event.ID,
		"event_type": string(event.Type),
	})

	inserted, err := h.events.InsertProcessedStripeEvent(ctx.Context(), repository.ProcessedStripeEvent{
		EventID:     event.ID,
		EventType:   string(event.Type),
		Outcome:     "success",
		Payload:     sanitizedPayload,
		ProcessedAt: time.Now(),
	})
	
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	
	if !inserted {
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "duplicate"})
	}

	if err := h.processEvent(ctx.Context(), event); err != nil {
		// Attempt to record failure in idempotency table if it fails processing
		sanitizedPayload, _ := json.Marshal(map[string]string{
			"event_id":   event.ID,
			"event_type": string(event.Type),
		})
		h.events.InsertProcessedStripeEvent(ctx.Context(), repository.ProcessedStripeEvent{
			EventID:     event.ID,
			EventType:   string(event.Type),
			Outcome:     "failed",
			Payload:     sanitizedPayload,
			ProcessedAt: time.Now(),
		})
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// processEvent dispatches supported stripe events.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) processEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}
		
		if session.ClientReferenceID == "" || session.Mode != stripe.CheckoutSessionModeSubscription {
			return nil
		}
		
		userID, err := uuid.Parse(session.ClientReferenceID)
		if err != nil {
			return err
		}
		
		return h.appendEntitlement(ctx, userID, "paid", "active", session.Customer.ID, session.Subscription.ID)

	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return err
		}
		return h.updateSubscriptionStatus(ctx, invoice.Subscription.ID, invoice.Customer.ID, "paid", "active")

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return err
		}
		return h.updateSubscriptionStatus(ctx, invoice.Subscription.ID, invoice.Customer.ID, "paid", "past_due")

	case "customer.subscription.deleted", "customer.subscription.canceled":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			return err
		}
		return h.updateSubscriptionStatus(ctx, subscription.ID, subscription.Customer.ID, "paid", "cancelled")
	}

	return nil
}

// appendEntitlement appends an active entitlement.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) appendEntitlement(ctx context.Context, userID uuid.UUID, tier, status, customerID, subscriptionID string) error {
	allowedModes := []string{"catalog", "substitution:single", "substitution:multi", "daily_diet", "daily_diet_alternative"}
	if tier == "free" || status != "active" {
		allowedModes = []string{"catalog", "substitution:single"}
	}
	
	ent := repository.Entitlement{
		UserID:               userID,
		Tier:                 tier,
		Status:               status,
		SearchLimitPer24h:    0, // unlimited searches for paid
		AllowedModes:         allowedModes,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: subscriptionID,
	}
	
	if tier == "free" {
		ent.SearchLimitPer24h = 3
	}

	return h.entitlements.AppendEntitlement(ctx, ent)
}

// updateSubscriptionStatus patches entitlement states.
// Implements DESIGN-007 StripeWebhookHandler.
func (h *StripeWebhookHandler) updateSubscriptionStatus(ctx context.Context, subscriptionID, customerID, tier, status string) error {
	var ent repository.Entitlement
	var err error

	if subscriptionID != "" {
		ent, err = h.entitlements.GetLatestByStripeSubscription(ctx, subscriptionID)
	} else if customerID != "" {
		ent, err = h.entitlements.GetLatestByStripeCustomer(ctx, customerID)
	} else {
		return errors.New("missing stripe identifiers")
	}

	if err != nil {
		if repository.IsKind(err, repository.ErrorKindNotFound) {
			return nil // ignore unknown subscriptions
		}
		return err
	}

	return h.appendEntitlement(ctx, ent.UserID, tier, status, ent.StripeCustomerID, ent.StripeSubscriptionID)
}

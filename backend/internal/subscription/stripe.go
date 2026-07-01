package subscription

import (
	"context"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
)

// StripeCheckoutGateway uses the official Stripe SDK to create checkout sessions.
// Implements DESIGN-007 CheckoutGateway.
type StripeCheckoutGateway struct {
	secretKey string
}

// NewStripeCheckoutGateway creates a new stripe checkout gateway.
// Implements DESIGN-007 CheckoutGateway.
func NewStripeCheckoutGateway(cfg config.Config) *StripeCheckoutGateway {
	return &StripeCheckoutGateway{
		secretKey: cfg.Billing.StripeSecretKey,
	}
}

// CreateSession creates a Stripe checkout session.
// Implements DESIGN-007 CheckoutGateway.
func (g *StripeCheckoutGateway) CreateSession(ctx context.Context, userID uuid.UUID, priceID, successURL, cancelURL, idempotencyKey string) (string, error) {
	stripe.Key = g.secretKey

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(userID.String()),
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	if idempotencyKey != "" {
		params.IdempotencyKey = stripe.String(idempotencyKey)
	}

	sess, err := session.New(params)
	if err != nil {
		return "", err
	}

	return sess.URL, nil
}

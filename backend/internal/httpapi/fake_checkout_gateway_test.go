package httpapi

// Implements DESIGN-007 SubscriptionController.

import (
	"context"

	"github.com/google/uuid"
)

// fakeCheckoutGateway implements CheckoutGateway for testing.
type fakeCheckoutGateway struct {
	sessions []PaymentIntentRequest
	urls     []string
	err      error
}

func (f *fakeCheckoutGateway) CreateSession(_ context.Context, _ uuid.UUID, priceID, successURL, cancelURL, idempotencyKey string) (string, error) {
	req := PaymentIntentRequest{PriceID: priceID, SuccessURL: successURL, CancelURL: cancelURL}
	if f.err != nil {
		return "", f.err
	}
	f.sessions = append(f.sessions, req)
	if len(f.urls) > 0 {
		return f.urls[len(f.sessions)-1], nil
	}
	return "https://checkout.stripe.com/pay/cs_test_123", nil
}

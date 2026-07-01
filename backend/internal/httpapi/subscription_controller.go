package httpapi

import (
	"errors"
	"net/url"
	"strings"

	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
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

// SubscriptionController manages billing routes.
// Implements DESIGN-007 SubscriptionController.
type SubscriptionController struct {
	billingConfig config.BillingConfig
	frontendUrl   string
	plans         map[string]Plan
}

// NewSubscriptionController creates a controller and maps price IDs to SW-REQ-050 amounts.
// Implements DESIGN-007 SubscriptionController initialization.
func NewSubscriptionController(cfg config.Config) *SubscriptionController {
	return &SubscriptionController{
		billingConfig: cfg.Billing,
		frontendUrl:   cfg.FrontendOrigin,
		plans: map[string]Plan{
			cfg.Billing.MonthlyPlanPriceID: {Label: "monthly", AmountUS: 300},
			cfg.Billing.AnnualPlanPriceID:  {Label: "annual", AmountUS: 2500},
		},
	}
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

package httpapi

import (
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strings"
)

// Implements DESIGN-007 SubscriptionController checkout request validation.
var allowedCheckoutRequestFields = []string{"plan", "successUrl", "cancelUrl"}

// Implements DESIGN-007 SubscriptionController raw-card-data rejection.
var rejectedCheckoutRequestFields = []string{"card", "cardNumber", "card[number]", "number", "cvc", "cvv", "expiry", "expMonth", "expYear", "paymentMethodData"}

// checkoutCreateRequestDTO is the server-owned checkout creation request shape.
// Implements DESIGN-007 SubscriptionController and SW-REQ-050 pricing tiers.
type checkoutCreateRequestDTO struct {
	Plan       string `json:"plan"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

// ValidateCheckoutCreateRequestBody rejects malformed checkout requests and raw card fields.
// Implements DESIGN-007 SubscriptionController.
func ValidateCheckoutCreateRequestBody(body map[string]any) error {
	for key := range body {
		normalized := strings.ToLower(strings.TrimSpace(key))
		for _, rejected := range rejectedCheckoutRequestFields {
			if normalized == strings.ToLower(rejected) {
				return errors.New("raw payment card data is not accepted")
			}
		}
		if !slices.Contains(allowedCheckoutRequestFields, key) {
			return errors.New("checkout request contains an unsupported field")
		}
	}
	dto, err := decodeCheckoutCreateRequestBody(body)
	if err != nil {
		return err
	}
	if dto.Plan != "monthly" && dto.Plan != "annual" {
		return errors.New("plan is invalid")
	}
	if err := validateCheckoutRequestRedirectURL(dto.SuccessURL); err != nil {
		return err
	}
	if err := validateCheckoutRequestRedirectURL(dto.CancelURL); err != nil {
		return err
	}
	return nil
}

// validateCheckoutRequestRedirectURL rejects relative or fragment-bearing checkout redirects.
// Implements DESIGN-007 SubscriptionController checkout redirect validation.
func validateCheckoutRequestRedirectURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Fragment != "" {
		return errors.New("checkout redirect URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("checkout redirect URL is invalid")
	}
	return nil
}

// decodeCheckoutCreateRequestBody converts a validated generic checkout body into the typed DTO.
// Implements DESIGN-007 SubscriptionController.
func decodeCheckoutCreateRequestBody(body map[string]any) (checkoutCreateRequestDTO, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return checkoutCreateRequestDTO{}, err
	}
	var dto checkoutCreateRequestDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		return checkoutCreateRequestDTO{}, err
	}
	return dto, nil
}

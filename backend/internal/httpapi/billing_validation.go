package httpapi

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

// Implements DESIGN-007 SubscriptionController checkout request validation.
var allowedCheckoutRequestFields = []string{"plan", "successUrl", "cancelUrl"}

// Implements DESIGN-007 SubscriptionController billing portal request validation.
var allowedBillingPortalRequestFields = []string{"returnUrl"}

// Implements DESIGN-007 SubscriptionController raw-card-data rejection.
var rejectedCheckoutRequestFields = []string{"card", "cardNumber", "card[number]", "number", "cvc", "cvv", "expiry", "expMonth", "expYear", "paymentMethodData"}

// checkoutCreateRequestDTO is the server-owned checkout creation request shape.
// Implements DESIGN-007 SubscriptionController and SW-REQ-050 pricing tiers.
type checkoutCreateRequestDTO struct {
	Plan       string `json:"plan"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

// billingPortalRequestDTO is the server-owned billing portal request shape.
// Implements DESIGN-007 SubscriptionController.
type billingPortalRequestDTO struct {
	ReturnURL string `json:"returnUrl"`
}

// ValidateCheckoutCreateRequestBodyForOrigin rejects malformed checkout requests and cross-origin redirects.
// Implements DESIGN-007 SubscriptionController checkout redirect validation.
func ValidateCheckoutCreateRequestBodyForOrigin(body map[string]any, allowedOrigin string) error {
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
	if err := validateCheckoutRequestRedirectURL(dto.SuccessURL, allowedOrigin); err != nil {
		return err
	}
	if err := validateCheckoutRequestRedirectURL(dto.CancelURL, allowedOrigin); err != nil {
		return err
	}
	return nil
}

// ValidateBillingPortalRequestBodyForOrigin rejects malformed billing portal requests and cross-origin return URLs.
// Implements DESIGN-007 SubscriptionController billing portal request validation.
func ValidateBillingPortalRequestBodyForOrigin(body map[string]any, allowedOrigin string) error {
	for key := range body {
		if !slices.Contains(allowedBillingPortalRequestFields, key) {
			return errors.New("billing portal request contains an unsupported field")
		}
	}
	dto, err := decodeBillingPortalRequestBody(body)
	if err != nil {
		return err
	}
	return validateCheckoutRequestRedirectURL(dto.ReturnURL, allowedOrigin)
}

// validateCheckoutRequestRedirectURL rejects relative, fragment-bearing, or cross-origin billing redirects.
// Implements DESIGN-007 SubscriptionController checkout redirect validation.
func validateCheckoutRequestRedirectURL(value string, allowedOrigin string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Fragment != "" {
		return errors.New("checkout redirect URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("checkout redirect URL is invalid")
	}
	if !sameOrigin(parsed, allowedOrigin) {
		return errors.New("checkout redirect URL origin is not allowed")
	}
	return nil
}

// decodeCheckoutCreateRequestBody converts a validated generic checkout body into the typed DTO.
// Implements DESIGN-007 SubscriptionController.
func decodeCheckoutCreateRequestBody(body map[string]any) (checkoutCreateRequestDTO, error) {
	plan, err := requiredStringField(body, "plan")
	if err != nil {
		return checkoutCreateRequestDTO{}, err
	}
	successURL, err := requiredStringField(body, "successUrl")
	if err != nil {
		return checkoutCreateRequestDTO{}, err
	}
	cancelURL, err := requiredStringField(body, "cancelUrl")
	if err != nil {
		return checkoutCreateRequestDTO{}, err
	}
	return checkoutCreateRequestDTO{Plan: plan, SuccessURL: successURL, CancelURL: cancelURL}, nil
}

// decodeBillingPortalRequestBody converts a validated generic portal body into the typed DTO.
// Implements DESIGN-007 SubscriptionController.
func decodeBillingPortalRequestBody(body map[string]any) (billingPortalRequestDTO, error) {
	returnURL, err := requiredStringField(body, "returnUrl")
	if err != nil {
		return billingPortalRequestDTO{}, err
	}
	return billingPortalRequestDTO{ReturnURL: returnURL}, nil
}

// requiredStringField extracts required JSON string fields without re-marshaling the body.
// Implements DESIGN-007 SubscriptionController billing request validation.
func requiredStringField(body map[string]any, field string) (string, error) {
	value, ok := body[field]
	if !ok {
		return "", errors.New("billing request is missing a required field")
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", errors.New("billing request field is invalid")
	}
	return text, nil
}

// sameOrigin checks redirect targets against the configured frontend origin.
// Implements DESIGN-007 SubscriptionController checkout redirect validation.
func sameOrigin(candidate *url.URL, allowedOrigin string) bool {
	allowed, err := url.Parse(allowedOrigin)
	if err != nil || allowed.Scheme == "" || allowed.Host == "" {
		return false
	}
	return strings.EqualFold(candidate.Scheme, allowed.Scheme) && strings.EqualFold(candidate.Host, allowed.Host)
}

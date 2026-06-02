package security

import (
	"errors"
	"net/mail"
	"strings"
)

// InputField identifies a supported field-specific normalization rule.
// Implements DESIGN-013 InputNormalizer.
type InputField string

// Implements DESIGN-013 InputNormalizer supported field types.
const (
	// InputFieldEmail selects email normalization and validation.
	InputFieldEmail InputField = "email"
)

// NormalizationResult reports accepted string normalization without exposing logs.
// Implements DESIGN-013 InputNormalizer.
type NormalizationResult struct {
	Value      string
	Changed    bool
	Violations []string
}

// NormalizeInput dispatches to the selected field-specific normalization rule.
// Implements DESIGN-013 InputNormalizer.
func NormalizeInput(field InputField, value string) (NormalizationResult, error) {
	switch field {
	case InputFieldEmail:
		return normalizeEmail(value)
	default:
		return NormalizationResult{}, errors.New("unsupported input field")
	}
}

// normalizeEmail trims and validates an email address without output escaping.
// Implements DESIGN-013 InputNormalizer.
func normalizeEmail(value string) (NormalizationResult, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return NormalizationResult{}, errors.New("email is required")
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return NormalizationResult{}, errors.New("email contains invalid characters")
	}
	address, err := mail.ParseAddress(trimmed)
	if err != nil || address.Address != trimmed {
		return NormalizationResult{}, errors.New("email is invalid")
	}
	result := NormalizationResult{Value: trimmed, Changed: trimmed != value}
	if result.Changed {
		result.Violations = []string{"whitespace_trimmed"}
	}
	return result, nil
}

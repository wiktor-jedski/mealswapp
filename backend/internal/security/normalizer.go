package security

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// InputField identifies a supported field-specific normalization rule.
// Implements DESIGN-013 InputNormalizer.
type InputField string

// Implements DESIGN-013 InputNormalizer supported field types.
const (
	// InputFieldEmail selects email normalization and validation.
	InputFieldEmail InputField = "email"
	// InputFieldPassword selects password policy validation.
	InputFieldPassword InputField = "password"
	// InputFieldDisplayName selects profile display-name normalization.
	InputFieldDisplayName InputField = "display_name"
	// InputFieldConsentVersion selects legal content version validation.
	InputFieldConsentVersion InputField = "consent_version"
	// InputFieldOAuthProvider selects OAuth provider-name validation.
	InputFieldOAuthProvider InputField = "oauth_provider"
	// InputFieldExportFormat selects account export-format validation.
	InputFieldExportFormat InputField = "export_format"
)

// Implements DESIGN-015 ConsentManager version identifier validation.
var consentVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{1,63}$`)

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
	case InputFieldPassword:
		return validatePassword(value, 12)
	case InputFieldDisplayName:
		return normalizeDisplayName(value)
	case InputFieldConsentVersion:
		return normalizeConsentVersion(value)
	case InputFieldOAuthProvider:
		return normalizeOAuthProvider(value)
	case InputFieldExportFormat:
		return normalizeExportFormat(value)
	default:
		return NormalizationResult{}, errors.New("unsupported input field")
	}
}

// ValidatePasswordPolicy validates a password against the configured minimum length.
// Implements DESIGN-006 PasswordHasher and DESIGN-013 InputNormalizer.
func ValidatePasswordPolicy(value string, minLength int) (NormalizationResult, error) {
	if minLength < 8 {
		return NormalizationResult{}, errors.New("password minimum length is invalid")
	}
	return validatePassword(value, minLength)
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

// validatePassword checks password complexity without trimming or returning the input.
// Implements DESIGN-006 PasswordHasher and DESIGN-013 InputNormalizer.
func validatePassword(value string, minLength int) (NormalizationResult, error) {
	if utf8.RuneCountInString(value) < minLength {
		return NormalizationResult{}, errors.New("password does not satisfy policy")
	}
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, r := range value {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsSpace(r):
			return NormalizationResult{}, errors.New("password does not satisfy policy")
		default:
			hasSymbol = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit || !hasSymbol {
		return NormalizationResult{}, errors.New("password does not satisfy policy")
	}
	return NormalizationResult{Value: value}, nil
}

// normalizeDisplayName trims, collapses internal spacing, and bounds display names.
// Implements DESIGN-008 ProfileController and DESIGN-013 InputNormalizer.
func normalizeDisplayName(value string) (NormalizationResult, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return NormalizationResult{Value: ""}, nil
	}
	parts := strings.Fields(trimmed)
	normalized := strings.Join(parts, " ")
	if utf8.RuneCountInString(normalized) > 80 {
		return NormalizationResult{}, errors.New("display name is too long")
	}
	if strings.ContainsRune(normalized, '\x00') {
		return NormalizationResult{}, errors.New("display name contains invalid characters")
	}
	changed := normalized != value
	result := NormalizationResult{Value: normalized, Changed: changed}
	if changed {
		result.Violations = []string{"whitespace_normalized"}
	}
	return result, nil
}

// normalizeConsentVersion validates legal-content version identifiers.
// Implements DESIGN-015 ConsentManager and DESIGN-013 InputNormalizer.
func normalizeConsentVersion(value string) (NormalizationResult, error) {
	trimmed := strings.TrimSpace(value)
	if !consentVersionPattern.MatchString(trimmed) {
		return NormalizationResult{}, errors.New("consent version is invalid")
	}
	result := NormalizationResult{Value: trimmed, Changed: trimmed != value}
	if result.Changed {
		result.Violations = []string{"whitespace_trimmed"}
	}
	return result, nil
}

// normalizeOAuthProvider accepts only supported external login providers.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 InputNormalizer.
func normalizeOAuthProvider(value string) (NormalizationResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized != "google" && normalized != "apple" {
		return NormalizationResult{}, errors.New("OAuth provider is unsupported")
	}
	result := NormalizationResult{Value: normalized, Changed: normalized != value}
	if result.Changed {
		result.Violations = []string{"provider_normalized"}
	}
	return result, nil
}

// normalizeExportFormat accepts account export formats supported by DESIGN-008.
// Implements DESIGN-008 DataExporter and DESIGN-013 InputNormalizer.
func normalizeExportFormat(value string) (NormalizationResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized != "json" && normalized != "csv" {
		return NormalizationResult{}, errors.New("export format is unsupported")
	}
	result := NormalizationResult{Value: normalized, Changed: normalized != value}
	if result.Changed {
		result.Violations = []string{"format_normalized"}
	}
	return result, nil
}

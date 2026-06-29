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
	// InputFieldSearchQuery selects search request query normalization.
	InputFieldSearchQuery InputField = "search_query"
	// InputFieldAutocompleteQuery selects autocomplete query normalization.
	InputFieldAutocompleteQuery InputField = "autocomplete_query"
	// InputFieldSearchMode selects search mode validation.
	InputFieldSearchMode InputField = "search_mode"
	// InputFieldPagination selects pagination query validation.
	InputFieldPagination InputField = "pagination"
	// InputFieldSearchFilterKind selects search filter kind validation.
	InputFieldSearchFilterKind InputField = "search_filter_kind"
	// InputFieldSubstitutionQuantity selects substitution quantity validation.
	InputFieldSubstitutionQuantity InputField = "substitution_quantity"
	// InputFieldSubstitutionUnit selects substitution unit validation.
	InputFieldSubstitutionUnit InputField = "substitution_unit"
	// InputFieldDailyDietID selects daily-diet identifier validation.
	InputFieldDailyDietID InputField = "daily_diet_id"
)

// Implements DESIGN-015 ConsentManager version identifier validation.
var consentVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{1,63}$`)

// Implements DESIGN-002 SearchController and DESIGN-013 InputNormalizer bounds.
const (
	MaxSearchQueryLength       = 200
	MaxAutocompleteQueryLength = 120
	MaxSearchPage              = 10000
)

// Implements DESIGN-002 FilterProcessor and DESIGN-013 InputNormalizer supported search filter kind tokens.
const (
	searchFilterKindFoodCategory  = "food_category"
	searchFilterKindCulinaryRole  = "culinary_role"
	searchFilterKindPhysicalState = "physical_state"
	searchFilterKindAllergen      = "allergen"
	searchFilterKindDietaryPreset = "dietary_preset"
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
	case InputFieldSearchQuery:
		return normalizeSearchQuery(value, MaxSearchQueryLength)
	case InputFieldAutocompleteQuery:
		return normalizeSearchQuery(value, MaxAutocompleteQueryLength)
	case InputFieldSearchMode:
		return normalizeSearchMode(value)
	case InputFieldPagination:
		return normalizeSearchPage(value)
	case InputFieldSearchFilterKind:
		return normalizeSearchFilterKind(value)
	case InputFieldSubstitutionQuantity:
		return normalizeSubstitutionQuantity(value)
	case InputFieldSubstitutionUnit:
		return normalizeSubstitutionUnit(value)
	case InputFieldDailyDietID:
		return normalizeDailyDietID(value)
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

// normalizeSearchQuery trims, lowercases, collapses spaces, and bounds search text.
// Implements DESIGN-002 SearchController and DESIGN-013 InputNormalizer.
func normalizeSearchQuery(value string, maxRunes int) (NormalizationResult, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return NormalizationResult{}, errors.New("search query is required")
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return NormalizationResult{}, errors.New("search query contains invalid characters")
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(trimmed), " "))
	if utf8.RuneCountInString(normalized) > maxRunes {
		return NormalizationResult{}, errors.New("search query is too long")
	}
	result := NormalizationResult{Value: normalized, Changed: normalized != value}
	if result.Changed {
		result.Violations = []string{"query_normalized"}
	}
	return result, nil
}

// normalizeSearchMode accepts only the search strategies supported by DESIGN-002.
// Implements DESIGN-002 SearchController and DESIGN-013 InputNormalizer.
func normalizeSearchMode(value string) (NormalizationResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "catalog", "substitution", "daily_diet_alternative":
		result := NormalizationResult{Value: normalized, Changed: normalized != value}
		if result.Changed {
			result.Violations = []string{"mode_normalized"}
		}
		return result, nil
	default:
		return NormalizationResult{}, errors.New("search mode is unsupported")
	}
}

// normalizeSearchPage validates a one-based search page within defensive bounds.
// Implements DESIGN-002 PaginationHandler and DESIGN-013 InputNormalizer.
func normalizeSearchPage(value string) (NormalizationResult, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return NormalizationResult{}, errors.New("page is required")
	}
	for _, r := range trimmed {
		if !unicode.IsDigit(r) {
			return NormalizationResult{}, errors.New("page is invalid")
		}
	}
	if strings.HasPrefix(trimmed, "0") && trimmed != "0" {
		return NormalizationResult{}, errors.New("page is invalid")
	}
	page := 0
	for _, r := range trimmed {
		page = page*10 + int(r-'0')
		if page > MaxSearchPage {
			return NormalizationResult{}, errors.New("page is too large")
		}
	}
	if page < 1 {
		return NormalizationResult{}, errors.New("page is invalid")
	}
	result := NormalizationResult{Value: trimmed, Changed: trimmed != value}
	if result.Changed {
		result.Violations = []string{"whitespace_trimmed"}
	}
	return result, nil
}

// normalizeSearchFilterKind accepts the filter kinds supported by DESIGN-002.
// Implements DESIGN-002 FilterProcessor and DESIGN-013 InputNormalizer.
func normalizeSearchFilterKind(value string) (NormalizationResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case searchFilterKindFoodCategory, searchFilterKindCulinaryRole, searchFilterKindPhysicalState, searchFilterKindAllergen, searchFilterKindDietaryPreset:
		result := NormalizationResult{Value: normalized, Changed: normalized != value}
		if result.Changed {
			result.Violations = []string{"filter_kind_normalized"}
		}
		return result, nil
	default:
		return NormalizationResult{}, errors.New("search filter kind is unsupported")
	}
}

// normalizeSubstitutionQuantity validates a positive decimal quantity without locale parsing.
// Implements DESIGN-002 SearchController and DESIGN-013 InputNormalizer.
func normalizeSubstitutionQuantity(value string) (NormalizationResult, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return NormalizationResult{}, errors.New("substitution quantity is required")
	}
	seenDecimal := false
	seenDigit := false
	for _, r := range trimmed {
		switch {
		case unicode.IsDigit(r):
			seenDigit = true
		case r == '.' && !seenDecimal:
			seenDecimal = true
		default:
			return NormalizationResult{}, errors.New("substitution quantity is invalid")
		}
	}
	if !seenDigit || trimmed == "0" || strings.Trim(trimmed, "0.") == "" {
		return NormalizationResult{}, errors.New("substitution quantity must be positive")
	}
	result := NormalizationResult{Value: trimmed, Changed: trimmed != value}
	if result.Changed {
		result.Violations = []string{"whitespace_trimmed"}
	}
	return result, nil
}

// normalizeSubstitutionUnit validates a compact unit token for substitution inputs.
// Implements DESIGN-002 SearchController and DESIGN-013 InputNormalizer.
func normalizeSubstitutionUnit(value string) (NormalizationResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || utf8.RuneCountInString(normalized) > 32 {
		return NormalizationResult{}, errors.New("substitution unit is invalid")
	}
	for _, r := range normalized {
		if !unicode.IsLetter(r) && r != '_' && r != '-' {
			return NormalizationResult{}, errors.New("substitution unit is invalid")
		}
	}
	result := NormalizationResult{Value: normalized, Changed: normalized != value}
	if result.Changed {
		result.Violations = []string{"unit_normalized"}
	}
	return result, nil
}

// normalizeDailyDietID validates UUID-shaped daily-diet search identifiers.
// Implements DESIGN-002 SearchController and DESIGN-013 InputNormalizer.
func normalizeDailyDietID(value string) (NormalizationResult, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if len(trimmed) != 36 {
		return NormalizationResult{}, errors.New("daily diet id is invalid")
	}
	for i, r := range trimmed {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return NormalizationResult{}, errors.New("daily diet id is invalid")
			}
			continue
		}
		if !unicode.IsDigit(r) && (r < 'a' || r > 'f') {
			return NormalizationResult{}, errors.New("daily diet id is invalid")
		}
	}
	result := NormalizationResult{Value: trimmed, Changed: trimmed != value}
	if result.Changed {
		result.Violations = []string{"daily_diet_id_normalized"}
	}
	return result, nil
}

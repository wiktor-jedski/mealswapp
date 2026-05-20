package security

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInputRejected = errors.New("input rejected")

type SanitizationResult struct {
	Value      string
	Changed    bool
	Violations []string
}

type SanitizerRule struct {
	Field     string
	MaxRunes  int
	AllowHTML bool
}

func SanitizeInput(field string, value string) (SanitizationResult, error) {
	switch field {
	case "search_term":
		return sanitizeWithRule(value, SanitizerRule{Field: field, MaxRunes: 128})
	case "profile_name":
		return sanitizeWithRule(value, SanitizerRule{Field: field, MaxRunes: 80})
	case "admin_import":
		return sanitizeWithRule(value, SanitizerRule{Field: field, MaxRunes: 255})
	default:
		return sanitizeWithRule(value, SanitizerRule{Field: field, MaxRunes: 255})
	}
}

func SanitizeSearchTerm(value string) (SanitizationResult, error) {
	return SanitizeInput("search_term", value)
}

func SanitizeProfileName(value string) (SanitizationResult, error) {
	return SanitizeInput("profile_name", value)
}

func SanitizeAdminImportField(value string) (SanitizationResult, error) {
	return SanitizeInput("admin_import", value)
}

func sanitizeWithRule(value string, rule SanitizerRule) (SanitizationResult, error) {
	trimmed := strings.TrimSpace(value)
	result := SanitizationResult{
		Value:   trimmed,
		Changed: trimmed != value,
	}

	if !utf8.ValidString(trimmed) {
		result.Violations = append(result.Violations, "invalid_utf8")
	}
	if rule.MaxRunes > 0 && utf8.RuneCountInString(trimmed) > rule.MaxRunes {
		result.Violations = append(result.Violations, "too_long")
	}
	if !rule.AllowHTML && containsUnsafeText(trimmed) {
		result.Violations = append(result.Violations, "unsafe_html")
	}

	if len(result.Violations) > 0 {
		return result, ErrInputRejected
	}

	return result, nil
}

func containsUnsafeText(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "javascript:") {
		return true
	}
	if strings.Contains(lower, "<") || strings.Contains(lower, ">") {
		return true
	}
	if strings.Contains(lower, "onerror=") || strings.Contains(lower, "onclick=") || strings.Contains(lower, "onload=") {
		return true
	}
	return false
}

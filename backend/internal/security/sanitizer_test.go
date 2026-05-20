package security

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeSearchTermTrimsWhitespace(t *testing.T) {
	result, err := SanitizeSearchTerm("  chickpea pasta  ")
	if err != nil {
		t.Fatal(err)
	}

	if result.Value != "chickpea pasta" || !result.Changed {
		t.Fatalf("expected trimmed changed result, got %#v", result)
	}
}

func TestSanitizeInputRejectsLengthLimitsByRune(t *testing.T) {
	_, err := SanitizeProfileName(strings.Repeat("a", 81))
	if !errors.Is(err, ErrInputRejected) {
		t.Fatalf("expected input rejected, got %v", err)
	}
}

func TestSanitizeInputRejectsUnsafeHTMLAndScriptPayloads(t *testing.T) {
	cases := []string{
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(1)>",
		"javascript:alert(1)",
	}

	for _, value := range cases {
		t.Run(value, func(t *testing.T) {
			result, err := SanitizeAdminImportField(value)
			if !errors.Is(err, ErrInputRejected) {
				t.Fatalf("expected input rejected, got %v", err)
			}
			if len(result.Violations) == 0 || result.Violations[0] != "unsafe_html" {
				t.Fatalf("expected unsafe_html violation, got %#v", result.Violations)
			}
		})
	}
}

func TestSanitizeInputPreservesUnicodeText(t *testing.T) {
	input := "Żurek z tofu i crème fraîche"
	result, err := SanitizeSearchTerm(input)
	if err != nil {
		t.Fatal(err)
	}

	if result.Value != input || result.Changed {
		t.Fatalf("expected unicode text preserved, got %#v", result)
	}
}

func TestSanitizeInputReportsMultipleViolations(t *testing.T) {
	result, err := SanitizeSearchTerm("<script>" + strings.Repeat("x", 130) + "</script>")
	if !errors.Is(err, ErrInputRejected) {
		t.Fatalf("expected input rejected, got %v", err)
	}

	if len(result.Violations) != 2 {
		t.Fatalf("expected too_long and unsafe_html violations, got %#v", result.Violations)
	}
}

package security

// Implements DESIGN-013 InputNormalizer curation boundary verification.

import (
	"strings"
	"testing"
)

func TestCurationTextNormalizationBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		field     InputField
		value     string
		want      string
		changed   bool
		wantError bool
	}{
		{name: "item NFC and whitespace", field: InputFieldCurationItemName, value: "  Cafe\u0301   au lait  ", want: "Café au lait", changed: true},
		{name: "classification Unicode", field: InputFieldCurationClassificationName, value: "Żywność – świeża", want: "Żywność – świeża"},
		{name: "classification punctuation", field: InputFieldCurationClassificationName, value: "Fruit & Vegetables", want: "Fruit & Vegetables"},
		{name: "classification maximum", field: InputFieldCurationClassificationName, value: strings.Repeat("a", 120), want: strings.Repeat("a", 120)},
		{name: "item maximum", field: InputFieldCurationItemName, value: strings.Repeat("a", MaxCurationItemNameLength), want: strings.Repeat("a", MaxCurationItemNameLength)},
		{name: "item too long", field: InputFieldCurationItemName, value: strings.Repeat("a", MaxCurationItemNameLength+1), wantError: true},
		{name: "classification too long", field: InputFieldCurationClassificationName, value: strings.Repeat("a", 121), wantError: true},
		{name: "empty item", field: InputFieldCurationItemName, value: "  ", wantError: true},
		{name: "emoji name", field: InputFieldCurationItemName, value: "Apple 🍎", wantError: true},
		{name: "name control", field: InputFieldCurationItemName, value: "Apple\nPie", wantError: true},
		{name: "external query", field: InputFieldExternalQuery, value: "  Crème   fraîche ", want: "Crème fraîche", changed: true},
		{name: "external query too long", field: InputFieldExternalQuery, value: strings.Repeat("q", MaxExternalQueryLength+1), wantError: true},
		{name: "provider text optional", field: InputFieldProviderText, value: "  Brand   description  ", want: "Brand description", changed: true},
		{name: "provider text control", field: InputFieldProviderText, value: "secret\ttext", wantError: true},
		{name: "provider text bidi control", field: InputFieldProviderText, value: "secret\u202etext", wantError: true},
		{name: "provider text too long", field: InputFieldProviderText, value: strings.Repeat("x", MaxProviderTextLength+1), wantError: true},
		{name: "malformed UTF-8", field: InputFieldProviderText, value: string([]byte{'b', 'a', 'd', 0xff}), wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NormalizeInput(tc.field, tc.value)
			if tc.wantError {
				if err == nil {
					t.Fatalf("NormalizeInput(%q) accepted %+v", tc.value, result)
				}
				return
			}
			if err != nil || result.Value != tc.want || result.Changed != tc.changed {
				t.Fatalf("NormalizeInput(%q) = %+v, %v", tc.value, result, err)
			}
		})
	}
}

func TestCurationProviderIdentifierAndUnitNormalization(t *testing.T) {
	tests := []struct {
		field     InputField
		value     string
		want      string
		wantError bool
	}{
		{InputFieldCurationProvider, " USDA ", "usda", false},
		{InputFieldCurationProvider, "OpenFoodFacts", "openfoodfacts", false},
		{InputFieldCurationProvider, "open-food-facts", "openfoodfacts", false},
		{InputFieldCurationProvider, "all", "", true},
		{InputFieldExternalProvider, "all", "all", false},
		{InputFieldCurationProvider, "other", "", true},
		{InputFieldProviderIdentifier, " fdc:123/4 ", "fdc:123/4", false},
		{InputFieldProviderIdentifier, "id with spaces", "", true},
		{InputFieldProviderIdentifier, "é", "", true},
		{InputFieldProviderIdentifier, strings.Repeat("x", MaxProviderIdentifierLength+1), "", true},
		{InputFieldServingUnit, "grams", "g", false},
		{InputFieldServingUnit, "millilitres", "ml", false},
		{InputFieldServingUnit, "ounces", "oz", false},
		{InputFieldServingUnit, "fluid ounces", "fl_oz", false},
		{InputFieldServingUnit, "portion", "serving", false},
		{InputFieldServingUnit, "cup", "", true},
	}
	for _, tc := range tests {
		result, err := NormalizeInput(tc.field, tc.value)
		if tc.wantError {
			if err == nil {
				t.Fatalf("%s %q accepted %+v", tc.field, tc.value, result)
			}
			continue
		}
		if err != nil || result.Value != tc.want {
			t.Fatalf("%s %q = %+v, %v", tc.field, tc.value, result, err)
		}
	}
}

func TestCurationImageURLSafety(t *testing.T) {
	tests := []struct {
		value     string
		want      string
		wantError bool
	}{
		{"", "", false},
		{" https://images.example.com/food/a.jpg?size=2 ", "https://images.example.com/food/a.jpg?size=2", false},
		{"https://images.example.com:443/a.jpg", "https://images.example.com:443/a.jpg", false},
		{"http://images.example.com/a.jpg", "", true},
		{"https://user:secret@images.example.com/a.jpg", "", true},
		{"https://localhost/a.jpg", "", true},
		{"https://localhost./a.jpg", "", true},
		{"https://food.internal/a.jpg", "", true},
		{"https://2130706433/a.jpg", "", true},
		{"https://127.0.0.1/a.jpg", "", true},
		{"https://10.0.0.1/a.jpg", "", true},
		{"https://[::1]/a.jpg", "", true},
		{"https://images.example.com:0/a.jpg", "", true},
		{"https://images.example.com:65536/a.jpg", "", true},
		{"https://images.example.com:invalid/a.jpg", "", true},
		{"https://é.example.com/a.jpg", "", true},
		{"https://images.example.com/a.jpg#fragment", "", true},
		{"https://images.example.com/a%00.jpg", "", true},
		{"data:image/png;base64,AA", "", true},
		{strings.Repeat("x", MaxImageURLLength+1), "", true},
	}
	for _, tc := range tests {
		result, err := NormalizeInput(InputFieldImageURL, tc.value)
		if tc.wantError {
			if err == nil {
				t.Fatalf("image URL %q accepted %+v", tc.value, result)
			}
			continue
		}
		if err != nil || result.Value != tc.want {
			t.Fatalf("image URL %q = %+v, %v", tc.value, result, err)
		}
	}
}

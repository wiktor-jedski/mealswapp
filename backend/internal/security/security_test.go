package security

// Implements DESIGN-013 EncryptionService, InputNormalizer, and AuditLogger verification.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type keys struct {
	active  string
	entries map[string][]byte
	err     error
}

func (k keys) ActiveKey(context.Context) (string, []byte, error) {
	return k.active, k.entries[k.active], k.err
}
func (k keys) Key(_ context.Context, version string) ([]byte, error) {
	key, ok := k.entries[version]
	if !ok {
		return nil, errors.New("missing key")
	}
	return key, nil
}
func (k keys) ActiveLookupKey(context.Context) (string, []byte, error) {
	return k.active, k.entries[k.active], k.err
}
func (k keys) LookupKey(_ context.Context, version string) ([]byte, error) {
	key, ok := k.entries[version]
	if !ok {
		return nil, errors.New("missing key")
	}
	return key, nil
}

func TestEncryptionRoundTripRotationAndFailures(t *testing.T) {
	ctx := context.Background()
	loader := keys{active: "v2", entries: map[string][]byte{"v1": []byte("11111111111111111111111111111111"), "v2": []byte("22222222222222222222222222222222")}}
	service := NewEncryptionService(loader)
	first, err := service.EncryptPII(ctx, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	second, _ := service.EncryptPII(ctx, []byte("secret"))
	if string(first.Nonce) == string(second.Nonce) || string(first.Ciphertext) == "secret" {
		t.Fatal("nonce reuse or plaintext envelope")
	}
	plain, err := service.DecryptPII(ctx, first)
	if err != nil || string(plain) != "secret" {
		t.Fatalf("decrypt = %q, %v", plain, err)
	}
	first.Ciphertext[0] ^= 1
	if _, err := service.DecryptPII(ctx, first); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err := NewEncryptionService(keys{active: "bad", entries: map[string][]byte{"bad": []byte("short")}}).EncryptPII(ctx, nil); err == nil {
		t.Fatal("short key accepted")
	}
	if _, err := service.DecryptPII(ctx, EncryptionEnvelope{KeyVersion: "missing"}); err == nil {
		t.Fatal("missing key accepted")
	}
	wrongKey := NewEncryptionService(keys{entries: map[string][]byte{"v2": []byte("33333333333333333333333333333333")}})
	if _, err := wrongKey.DecryptPII(ctx, first); err == nil {
		t.Fatal("wrong key accepted")
	}
	short := NewEncryptionService(keys{entries: map[string][]byte{"short": []byte("short")}})
	if _, err := short.DecryptPII(ctx, EncryptionEnvelope{KeyVersion: "short"}); err == nil {
		t.Fatal("short decrypt key accepted")
	}
	service.randomness = strings.NewReader("")
	if _, err := service.EncryptPII(ctx, nil); err == nil {
		t.Fatal("randomness failure accepted")
	}
	if _, err := NewEncryptionService(keys{err: errors.New("down")}).EncryptPII(ctx, nil); err == nil {
		t.Fatal("active key failure accepted")
	}
}

func TestLookupDigestDeterminismRotationAndFailures(t *testing.T) {
	ctx := context.Background()
	loader := keys{active: "lookup-v2", entries: map[string][]byte{
		"lookup-v1": []byte("11111111111111111111111111111111"),
		"lookup-v2": []byte("22222222222222222222222222222222"),
	}}
	service := NewLookupDigestService(loader)
	first, err := service.DigestForWrite(ctx, []byte("user@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.DigestForWrite(ctx, []byte("user@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.KeyVersion != "lookup-v2" || first.Value == "user@example.test" {
		t.Fatalf("digest = %+v, repeat = %+v", first, second)
	}
	oldVersion, err := service.DigestForVersion(ctx, "lookup-v1", []byte("user@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	if oldVersion.Value == first.Value {
		t.Fatalf("rotation digest did not change: %+v", oldVersion)
	}
	wrongKey := NewLookupDigestService(keys{active: "lookup-v2", entries: map[string][]byte{
		"lookup-v2": []byte("33333333333333333333333333333333"),
	}})
	wrongDigest, err := wrongKey.DigestForWrite(ctx, []byte("user@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	if wrongDigest.Value == first.Value {
		t.Fatal("wrong lookup key produced matching digest")
	}
	if _, err := service.DigestForVersion(ctx, "missing", []byte("user@example.test")); err == nil {
		t.Fatal("missing lookup key accepted")
	}
	if _, err := NewLookupDigestService(keys{active: "short", entries: map[string][]byte{"short": []byte("short")}}).DigestForWrite(ctx, nil); err == nil {
		t.Fatal("short lookup key accepted")
	}
	if _, err := NewLookupDigestService(keys{err: errors.New("down")}).DigestForWrite(ctx, nil); err == nil {
		t.Fatal("active lookup key failure accepted")
	}
}

func TestNormalizeInput(t *testing.T) {
	result, err := NormalizeInput(InputFieldEmail, " user@example.com ")
	if err != nil || !result.Changed || len(result.Violations) != 1 {
		t.Fatalf("normalize = %+v, %v", result, err)
	}
	password, err := ValidatePasswordPolicy("StrongerPassword1!", 12)
	if err != nil || password.Value != "StrongerPassword1!" {
		t.Fatalf("password normalize = %+v, %v", password, err)
	}
	displayName, err := NormalizeInput(InputFieldDisplayName, " Ada   Lovelace ")
	if err != nil || displayName.Value != "Ada Lovelace" || !displayName.Changed {
		t.Fatalf("display name normalize = %+v, %v", displayName, err)
	}
	consent, err := NormalizeInput(InputFieldConsentVersion, " privacy-2026-06 ")
	if err != nil || consent.Value != "privacy-2026-06" || !consent.Changed {
		t.Fatalf("consent version normalize = %+v, %v", consent, err)
	}
	provider, err := NormalizeInput(InputFieldOAuthProvider, " Google ")
	if err != nil || provider.Value != "google" || !provider.Changed {
		t.Fatalf("provider normalize = %+v, %v", provider, err)
	}
	format, err := NormalizeInput(InputFieldExportFormat, " CSV ")
	if err != nil || format.Value != "csv" || !format.Changed {
		t.Fatalf("format normalize = %+v, %v", format, err)
	}
	searchQuery, err := NormalizeInput(InputFieldSearchQuery, "  Fresh   TOMATO  ")
	if err != nil || searchQuery.Value != "fresh tomato" || !searchQuery.Changed {
		t.Fatalf("search query normalize = %+v, %v", searchQuery, err)
	}
	autocompleteQuery, err := NormalizeInput(InputFieldAutocompleteQuery, " Lent ")
	if err != nil || autocompleteQuery.Value != "lent" || !autocompleteQuery.Changed {
		t.Fatalf("autocomplete query normalize = %+v, %v", autocompleteQuery, err)
	}
	searchMode, err := NormalizeInput(InputFieldSearchMode, " Substitution ")
	if err != nil || searchMode.Value != "substitution" || !searchMode.Changed {
		t.Fatalf("search mode normalize = %+v, %v", searchMode, err)
	}
	page, err := NormalizeInput(InputFieldPagination, " 12 ")
	if err != nil || page.Value != "12" || !page.Changed {
		t.Fatalf("page normalize = %+v, %v", page, err)
	}
	filterKind, err := NormalizeInput(InputFieldSearchFilterKind, " Allergen ")
	if err != nil || filterKind.Value != "allergen" || !filterKind.Changed {
		t.Fatalf("filter kind normalize = %+v, %v", filterKind, err)
	}
	quantity, err := NormalizeInput(InputFieldSubstitutionQuantity, " 12.5 ")
	if err != nil || quantity.Value != "12.5" || !quantity.Changed {
		t.Fatalf("quantity normalize = %+v, %v", quantity, err)
	}
	unit, err := NormalizeInput(InputFieldSubstitutionUnit, " Gram ")
	if err != nil || unit.Value != "gram" || !unit.Changed {
		t.Fatalf("unit normalize = %+v, %v", unit, err)
	}
	dailyDietID, err := NormalizeInput(InputFieldDailyDietID, " 2D4A5F20-C55F-4BA7-9751-779E682F7063 ")
	if err != nil || dailyDietID.Value != "2d4a5f20-c55f-4ba7-9751-779e682f7063" || !dailyDietID.Changed {
		t.Fatalf("daily diet id normalize = %+v, %v", dailyDietID, err)
	}
	for _, input := range []struct {
		field InputField
		value string
	}{
		{InputFieldEmail, ""},
		{InputFieldEmail, "\x00"},
		{InputFieldEmail, "bad"},
		{InputFieldPassword, "too-short"},
		{InputFieldPassword, "lowercasepassword1!"},
		{InputFieldPassword, "UPPERCASEPASSWORD1!"},
		{InputFieldPassword, "NoDigitsHere!"},
		{InputFieldPassword, "No symbol 1"},
		{InputFieldDisplayName, strings.Repeat("a", 81)},
		{InputFieldDisplayName, "Ada\x00Lovelace"},
		{InputFieldConsentVersion, "bad version"},
		{InputFieldOAuthProvider, "github"},
		{InputFieldExportFormat, "xml"},
		{InputFieldSearchQuery, "   "},
		{InputFieldSearchQuery, strings.Repeat("a", MaxSearchQueryLength+1)},
		{InputFieldSearchQuery, "tomato\x00"},
		{InputFieldAutocompleteQuery, strings.Repeat("a", MaxAutocompleteQueryLength+1)},
		{InputFieldSearchMode, "meal_plan"},
		{InputFieldPagination, "0"},
		{InputFieldPagination, "-1"},
		{InputFieldPagination, "1.5"},
		{InputFieldPagination, "10001"},
		{InputFieldSearchFilterKind, "brand"},
		{InputFieldSubstitutionQuantity, "0"},
		{InputFieldSubstitutionQuantity, "-1"},
		{InputFieldSubstitutionQuantity, "1,5"},
		{InputFieldSubstitutionUnit, ""},
		{InputFieldSubstitutionUnit, "grams/ml"},
		{InputFieldDailyDietID, "not-a-uuid"},
		{"unknown", "value"},
	} {
		if _, err := NormalizeInput(input.field, input.value); err == nil {
			t.Fatalf("accepted %+v", input)
		}
	}
	if _, err := ValidatePasswordPolicy("StrongerPassword1!", 7); err == nil || strings.Contains(err.Error(), "StrongerPassword1") {
		t.Fatalf("password policy error = %v", err)
	}
	emptyDisplayName, err := NormalizeInput(InputFieldDisplayName, "   ")
	if err != nil || emptyDisplayName.Value != "" {
		t.Fatalf("empty display name = %+v, %v", emptyDisplayName, err)
	}
}

type failingAudit struct{ count int }

func (a *failingAudit) Audit(context.Context, AuditLogEntry) error {
	a.count++
	return errors.New("down")
}

func TestAuditPolicies(t *testing.T) {
	audit := &failingAudit{}
	if RecordAuditRequired(context.Background(), audit, AuditLogEntry{}) == nil {
		t.Fatal("mutation did not fail closed")
	}
	RecordAuditBestEffort(context.Background(), audit, AuditLogEntry{})
	RecordAuditBestEffort(context.Background(), nil, AuditLogEntry{})
	if audit.count != 2 || RecordAuditRequired(context.Background(), nil, AuditLogEntry{}) == nil {
		t.Fatalf("audit count = %d", audit.count)
	}
}

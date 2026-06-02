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

func TestNormalizeInput(t *testing.T) {
	result, err := NormalizeInput(InputFieldEmail, " user@example.com ")
	if err != nil || !result.Changed || len(result.Violations) != 1 {
		t.Fatalf("normalize = %+v, %v", result, err)
	}
	for _, input := range []struct {
		field InputField
		value string
	}{{InputFieldEmail, ""}, {InputFieldEmail, "\x00"}, {InputFieldEmail, "bad"}, {"unknown", "value"}} {
		if _, err := NormalizeInput(input.field, input.value); err == nil {
			t.Fatalf("accepted %+v", input)
		}
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

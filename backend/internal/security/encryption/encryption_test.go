package encryption

import (
	"bytes"
	"errors"
	"testing"
)

func TestServiceEncryptsAndDecryptsAES256GCMEnvelope(t *testing.T) {
	service, err := NewService(bytes.Repeat([]byte{1}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	service.random = bytes.NewReader(bytes.Repeat([]byte{2}, 12))

	envelope, err := service.Encrypt([]byte("person@example.com"), []byte("users.email"))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Algorithm != AlgorithmAES256GCM || envelope.KeyVersion != "v1" || envelope.Ciphertext == "" || envelope.Nonce == "" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
	if envelope.Ciphertext == "person@example.com" {
		t.Fatal("ciphertext must not contain plaintext")
	}

	plaintext, err := service.Decrypt(envelope, []byte("users.email"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "person@example.com" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}

func TestServiceRejectsBadKeyAndTamperedCiphertext(t *testing.T) {
	if _, err := NewService([]byte("short"), "v1"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected invalid key error, got %v", err)
	}

	service, err := NewService(bytes.Repeat([]byte{1}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	service.random = bytes.NewReader(bytes.Repeat([]byte{2}, 12))
	envelope, err := service.Encrypt([]byte("secret"), nil)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Ciphertext = envelope.Ciphertext[:len(envelope.Ciphertext)-2] + "AA"

	if _, err := service.Decrypt(envelope, nil); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("expected invalid ciphertext error, got %v", err)
	}
}

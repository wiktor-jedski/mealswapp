// Phase: phase-01 | Task: 17 | Architecture: ARCH-013 | Design: EncryptionService

package middleware

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "a" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	plaintext := []byte("Hello, World!")

	ciphertext, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := service.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match original: got %q, want %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptToBase64DecryptFromBase64(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "b" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	plaintext := []byte("Secret data with UTF-8: 你好世界")

	encrypted, err := service.EncryptToBase64(plaintext)
	if err != nil {
		t.Fatalf("EncryptToBase64 failed: %v", err)
	}

	decrypted, err := service.DecryptFromBase64(encrypted)
	if err != nil {
		t.Fatalf("DecryptFromBase64 failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match original: got %q, want %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "c" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.Encrypt([]byte{})
	if err != ErrNilInput {
		t.Errorf("Expected ErrNilInput for empty plaintext, got: %v", err)
	}
}

func TestEncryptNilPlaintext(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "d" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.Encrypt(nil)
	if err != ErrNilInput {
		t.Errorf("Expected ErrNilInput for nil plaintext, got: %v", err)
	}
}

func TestDecryptNilCiphertext(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "e" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.Decrypt(nil)
	if err != ErrNilInput {
		t.Errorf("Expected ErrNilInput for nil ciphertext, got: %v", err)
	}
}

func TestDecryptDataTooShort(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "f" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.Decrypt([]byte("short"))
	if err != ErrDataTooShort {
		t.Errorf("Expected ErrDataTooShort, got: %v", err)
	}
}

func TestDecryptTamperedData(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "g" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	plaintext := []byte("Original message")
	ciphertext, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = service.Decrypt(ciphertext)
	if err != ErrAuthenticationFailed {
		t.Errorf("Expected ErrAuthenticationFailed for tampered data, got: %v", err)
	}
}

func TestNewServiceEmptyKey(t *testing.T) {
	_, err := NewService(Config{
		MasterKey: "",
	})
	if err != ErrKeyNotInitialized {
		t.Errorf("Expected ErrKeyNotInitialized, got: %v", err)
	}
}

func TestNewServiceInvalidKey(t *testing.T) {
	_, err := NewService(Config{
		MasterKey: "invalid-base64!!!",
	})
	if err != ErrInvalidKeySize {
		t.Errorf("Expected ErrInvalidKeySize, got: %v", err)
	}
}

func TestNewServiceWrongKeySize(t *testing.T) {
	shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, err := NewService(Config{
		MasterKey: shortKey,
	})
	if err != ErrInvalidKeySize {
		t.Errorf("Expected ErrInvalidKeySize for wrong key size, got: %v", err)
	}
}

func TestEncryptDifferentPlaintexts(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "h" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	testCases := []string{
		"",
		"a",
		"Hello",
		"Longer text with multiple words and symbols !@#$%^&*()",
		"Unicode: 日本語 中文 한국어",
		"Binary-like: \x00\x01\x02\x03\xff\xfe\xfd",
	}

	for _, tc := range testCases {
		plaintext := []byte(tc)
		ciphertext, err := service.Encrypt(plaintext)
		if err != nil {
			t.Errorf("Encrypt failed for %q: %v", tc, err)
			continue
		}

		decrypted, err := service.Decrypt(ciphertext)
		if err != nil {
			t.Errorf("Decrypt failed for %q: %v", tc, err)
			continue
		}

		if string(decrypted) != string(plaintext) {
			t.Errorf("Roundtrip failed for %q: got %q, want %q", tc, string(decrypted), string(plaintext))
		}
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	masterKey := base64.StdEncoding.EncodeToString(make([]byte, KeySize256))
	for i := range masterKey {
		masterKey = masterKey[:i] + "i" + masterKey[i+1:]
	}

	service, err := NewService(Config{
		MasterKey: masterKey,
		NonceSize: 12,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	plaintext := []byte("Same plaintext")

	ct1, _ := service.Encrypt(plaintext)
	ct2, _ := service.Encrypt(plaintext)

	if string(ct1) == string(ct2) {
		t.Error("Encrypt should produce different ciphertexts due to random nonce")
	}
}

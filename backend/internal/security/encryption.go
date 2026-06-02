package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// EncryptionEnvelope stores authenticated encrypted PII with its key version.
// Implements DESIGN-013 EncryptionService.
type EncryptionEnvelope struct {
	KeyVersion string
	Nonce      []byte
	Ciphertext []byte
}

// KeyLoader resolves versioned encryption keys from local fixtures or Secret Manager adapters.
// Implements DESIGN-013 EncryptionService.
type KeyLoader interface {
	ActiveKey(context.Context) (string, []byte, error)
	Key(context.Context, string) ([]byte, error)
}

// EncryptionService encrypts and decrypts PII envelopes.
// Implements DESIGN-013 EncryptionService.
type EncryptionService struct {
	keys       KeyLoader
	randomness io.Reader
}

// NewEncryptionService creates an envelope encryption service.
// Implements DESIGN-013 EncryptionService.
func NewEncryptionService(keys KeyLoader) *EncryptionService {
	return &EncryptionService{keys: keys, randomness: rand.Reader}
}

// EncryptPII encrypts plaintext with the active AES-256-GCM key.
// Implements DESIGN-013 EncryptionService.
func (s *EncryptionService) EncryptPII(ctx context.Context, plaintext []byte) (EncryptionEnvelope, error) {
	version, key, err := s.keys.ActiveKey(ctx)
	if err != nil {
		return EncryptionEnvelope{}, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return EncryptionEnvelope{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(s.randomness, nonce); err != nil {
		return EncryptionEnvelope{}, err
	}
	return EncryptionEnvelope{KeyVersion: version, Nonce: nonce, Ciphertext: gcm.Seal(nil, nonce, plaintext, nil)}, nil
}

// DecryptPII decrypts and authenticates an encryption envelope.
// Implements DESIGN-013 EncryptionService.
func (s *EncryptionService) DecryptPII(ctx context.Context, envelope EncryptionEnvelope) ([]byte, error) {
	key, err := s.keys.Key(ctx, envelope.KeyVersion)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, envelope.Nonce, envelope.Ciphertext, nil)
}

// newGCM constructs an AES-256-GCM authenticated cipher.
// Implements DESIGN-013 EncryptionService.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256 key must contain 32 bytes")
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	return gcm, nil
}

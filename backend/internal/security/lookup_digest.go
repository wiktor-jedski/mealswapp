package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// LookupDigest stores deterministic keyed lookup material for encrypted PII.
// Implements DESIGN-013 EncryptionService.
type LookupDigest struct {
	KeyVersion string
	Value      string
}

// LookupKeyLoader resolves versioned HMAC keys for deterministic PII lookup.
// Implements DESIGN-013 EncryptionService.
type LookupKeyLoader interface {
	ActiveLookupKey(context.Context) (string, []byte, error)
	LookupKey(context.Context, string) ([]byte, error)
}

// LookupDigestService derives versioned HMAC-SHA-256 lookup digests.
// Implements DESIGN-013 EncryptionService.
type LookupDigestService struct {
	keys LookupKeyLoader
}

// NewLookupDigestService creates a deterministic PII lookup digest service.
// Implements DESIGN-013 EncryptionService.
func NewLookupDigestService(keys LookupKeyLoader) *LookupDigestService {
	return &LookupDigestService{keys: keys}
}

// DigestForWrite derives a digest with the active lookup key version.
// Implements DESIGN-013 EncryptionService.
func (s *LookupDigestService) DigestForWrite(ctx context.Context, normalized []byte) (LookupDigest, error) {
	version, key, err := s.keys.ActiveLookupKey(ctx)
	if err != nil {
		return LookupDigest{}, err
	}
	value, err := keyedDigest(key, normalized)
	if err != nil {
		return LookupDigest{}, err
	}
	return LookupDigest{KeyVersion: version, Value: value}, nil
}

// DigestForVersion derives a digest with a specific lookup key version.
// Implements DESIGN-013 EncryptionService.
func (s *LookupDigestService) DigestForVersion(ctx context.Context, version string, normalized []byte) (LookupDigest, error) {
	key, err := s.keys.LookupKey(ctx, version)
	if err != nil {
		return LookupDigest{}, err
	}
	value, err := keyedDigest(key, normalized)
	if err != nil {
		return LookupDigest{}, err
	}
	return LookupDigest{KeyVersion: version, Value: value}, nil
}

// keyedDigest derives a hex-encoded HMAC-SHA-256 digest.
// Implements DESIGN-013 EncryptionService.
func keyedDigest(key []byte, normalized []byte) (string, error) {
	if len(key) < 32 {
		return "", errors.New("lookup HMAC key must contain at least 32 bytes")
	}
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(normalized); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

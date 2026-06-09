package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
	"golang.org/x/crypto/argon2"
)

// PasswordHashParams controls Argon2id password hashing cost.
// Implements DESIGN-006 PasswordHasher.
type PasswordHashParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
	SaltLength  int
	MinLength   int
}

// PasswordHasher hashes and verifies passwords with Argon2id.
// Implements DESIGN-006 PasswordHasher.
type PasswordHasher struct {
	params     PasswordHashParams
	randomness io.Reader
}

// DefaultPasswordHashParams returns conservative local defaults.
// Implements DESIGN-006 PasswordHasher.
func DefaultPasswordHashParams() PasswordHashParams {
	return PasswordHashParams{MemoryKiB: 64 * 1024, Iterations: 3, Parallelism: 2, KeyLength: 32, SaltLength: 16, MinLength: 12}
}

// NewPasswordHasher creates an Argon2id password hasher.
// Implements DESIGN-006 PasswordHasher.
func NewPasswordHasher(params PasswordHashParams) (*PasswordHasher, error) {
	if params.MemoryKiB < 19*1024 || params.Iterations == 0 || params.Parallelism == 0 || params.KeyLength < 16 || params.SaltLength < 16 || params.MinLength < 8 {
		return nil, errors.New("password hash parameters are invalid")
	}
	return &PasswordHasher{params: params, randomness: rand.Reader}, nil
}

// NewDefaultPasswordHasher creates a hasher with default Argon2id parameters.
// Implements DESIGN-006 PasswordHasher.
func NewDefaultPasswordHasher() *PasswordHasher {
	hasher, err := NewPasswordHasher(DefaultPasswordHashParams())
	if err != nil {
		panic(err)
	}
	return hasher
}

// HashPassword validates policy, creates a unique salt, and hashes a password.
// Implements DESIGN-006 PasswordHasher.
func (h *PasswordHasher) HashPassword(password string) (string, string, error) {
	if _, err := security.ValidatePasswordPolicy(password, h.params.MinLength); err != nil {
		return "", "", err
	}
	salt := make([]byte, h.params.SaltLength)
	if _, err := io.ReadFull(h.randomness, salt); err != nil {
		return "", "", err
	}
	hash := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.MemoryKiB, h.params.Parallelism, h.params.KeyLength)
	encodedHash := fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s", h.params.MemoryKiB, h.params.Iterations, h.params.Parallelism, base64.RawStdEncoding.EncodeToString(hash))
	return encodedHash, base64.RawStdEncoding.EncodeToString(salt), nil
}

// VerifyPassword verifies a password against an encoded Argon2id hash and salt.
// Implements DESIGN-006 PasswordHasher.
func (h *PasswordHasher) VerifyPassword(password string, encodedHash string, encodedSalt string) bool {
	params, expectedHash, err := parseEncodedHash(encodedHash)
	if err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(encodedSalt)
	if err != nil || len(salt) == 0 {
		return false
	}
	actualHash := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, uint32(len(expectedHash)))
	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

// TestPasswordHash returns deterministic fixture-safe credentials for repository tests only.
// Implements DESIGN-006 PasswordHasher.
func TestPasswordHash() (string, string) {
	return "argon2id$v=19$m=19456,t=1,p=1$u67X4pB7vrPK0wZMLU3SXg", "dGVzdC1maXh0dXJlLXNhbHQ"
}

// parseEncodedHash parses this package's Argon2id hash format.
// Implements DESIGN-006 PasswordHasher.
func parseEncodedHash(encodedHash string) (PasswordHashParams, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return PasswordHashParams{}, nil, errors.New("password hash is malformed")
	}
	params, err := parseHashParams(parts[2])
	if err != nil {
		return PasswordHashParams{}, nil, err
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(hash) < 16 {
		return PasswordHashParams{}, nil, errors.New("password hash is malformed")
	}
	params.KeyLength = uint32(len(hash))
	return params, hash, nil
}

// parseHashParams parses Argon2 cost metadata.
// Implements DESIGN-006 PasswordHasher.
func parseHashParams(value string) (PasswordHashParams, error) {
	params := PasswordHashParams{}
	for _, part := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			return PasswordHashParams{}, errors.New("password hash parameters are malformed")
		}
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return PasswordHashParams{}, errors.New("password hash parameters are malformed")
		}
		switch key {
		case "m":
			params.MemoryKiB = uint32(parsed)
		case "t":
			params.Iterations = uint32(parsed)
		case "p":
			if parsed > 255 {
				return PasswordHashParams{}, errors.New("password hash parameters are malformed")
			}
			params.Parallelism = uint8(parsed)
		default:
			return PasswordHashParams{}, errors.New("password hash parameters are malformed")
		}
	}
	if params.MemoryKiB == 0 || params.Iterations == 0 || params.Parallelism == 0 {
		return PasswordHashParams{}, errors.New("password hash parameters are malformed")
	}
	return params, nil
}

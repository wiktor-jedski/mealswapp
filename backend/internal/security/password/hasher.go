package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

var (
	ErrWeakPassword = errors.New("weak password")
	ErrInvalidHash  = errors.New("invalid password hash")
)

type Parameters struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParameters() Parameters {
	return Parameters{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

type Hasher struct {
	params Parameters
}

func NewHasher(params Parameters) Hasher {
	if params.Memory == 0 {
		params.Memory = DefaultParameters().Memory
	}
	if params.Iterations == 0 {
		params.Iterations = DefaultParameters().Iterations
	}
	if params.Parallelism == 0 {
		params.Parallelism = DefaultParameters().Parallelism
	}
	if params.SaltLength == 0 {
		params.SaltLength = DefaultParameters().SaltLength
	}
	if params.KeyLength == 0 {
		params.KeyLength = DefaultParameters().KeyLength
	}
	return Hasher{params: params}
}

func (hasher Hasher) Hash(password string) (string, error) {
	if err := ValidateStrength(password); err != nil {
		return "", err
	}

	salt := make([]byte, hasher.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, hasher.params.Iterations, hasher.params.Memory, hasher.params.Parallelism, hasher.params.KeyLength)
	return encodeHash(hasher.params, salt, key), nil
}

func (hasher Hasher) Verify(password string, encodedHash string) (bool, error) {
	params, salt, expectedKey, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	actualKey := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, uint32(len(expectedKey)))
	if subtle.ConstantTimeCompare(actualKey, expectedKey) == 1 {
		return true, nil
	}
	return false, nil
}

func (hasher Hasher) NeedsRehash(encodedHash string) (bool, error) {
	params, _, key, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	return params.Memory != hasher.params.Memory ||
		params.Iterations != hasher.params.Iterations ||
		params.Parallelism != hasher.params.Parallelism ||
		uint32(len(key)) != hasher.params.KeyLength, nil
}

func ValidateStrength(password string) error {
	if len([]rune(password)) < 12 {
		return ErrWeakPassword
	}

	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSymbol = true
		}
	}

	if !hasLower || !hasUpper || !hasDigit || !hasSymbol {
		return ErrWeakPassword
	}
	return nil
}

func encodeHash(params Parameters, salt []byte, key []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func decodeHash(encodedHash string) (Parameters, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return Parameters{}, nil, nil, ErrInvalidHash
	}

	params, err := parseParameters(parts[3])
	if err != nil {
		return Parameters{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Parameters{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Parameters{}, nil, nil, ErrInvalidHash
	}

	return params, salt, key, nil
}

func parseParameters(value string) (Parameters, error) {
	var params Parameters
	for _, part := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			return Parameters{}, ErrInvalidHash
		}
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return Parameters{}, ErrInvalidHash
		}

		switch key {
		case "m":
			params.Memory = uint32(parsed)
		case "t":
			params.Iterations = uint32(parsed)
		case "p":
			if parsed > 255 {
				return Parameters{}, ErrInvalidHash
			}
			params.Parallelism = uint8(parsed)
		default:
			return Parameters{}, ErrInvalidHash
		}
	}

	if params.Memory == 0 || params.Iterations == 0 || params.Parallelism == 0 {
		return Parameters{}, ErrInvalidHash
	}
	return params, nil
}

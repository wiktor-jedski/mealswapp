// Phase: phase-01 | Task: 10 | Architecture: ARCH-013 | Design: EncryptionService

package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

type KeySize int

const (
	KeySize256 KeySize = 32
)

type EncryptionMode int

const (
	ModeGCM EncryptionMode = iota
)

type Service struct {
	gcm     cipher.AEAD
	keySize KeySize
}

type EncryptedData struct {
	Ciphertext []byte
	Nonce      []byte
	Version    int
}

type ServiceOption func(*Service)

type Config struct {
	MasterKey string
	NonceSize int
}

var (
	ErrInvalidKeySize       = errors.New("encryption: invalid key size")
	ErrInvalidNonceSize     = errors.New("encryption: invalid nonce size")
	ErrEncryptionFailed     = errors.New("encryption: encryption operation failed")
	ErrDecryptionFailed     = errors.New("encryption: decryption operation failed")
	ErrAuthenticationFailed = errors.New("encryption: authentication failed - data may be tampered")
	ErrNilInput             = errors.New("encryption: nil or empty input data")
	ErrKeyNotInitialized    = errors.New("encryption: service not initialized with encryption key")
	ErrDataTooShort         = errors.New("encryption: encrypted data too short")
	ErrUnsupportedVersion   = errors.New("encryption: unsupported encrypted data version")
)

func WithMode(mode EncryptionMode) ServiceOption {
	return func(s *Service) {
		_ = mode
	}
}

func WithKeySize(size KeySize) ServiceOption {
	return func(s *Service) {
		s.keySize = size
	}
}

func NewService(config Config) (*Service, error) {
	if config.MasterKey == "" {
		return nil, ErrKeyNotInitialized
	}

	keyBytes, err := base64.StdEncoding.DecodeString(config.MasterKey)
	if err != nil {
		return nil, ErrInvalidKeySize
	}

	if len(keyBytes) != int(KeySize256) {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	nonceSize := config.NonceSize
	if nonceSize == 0 {
		nonceSize = 12
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, nonceSize)
	if err != nil {
		return nil, err
	}

	return &Service{gcm: gcm, keySize: KeySize256}, nil
}

func (s *Service) generateRandomBytes(size int) []byte {
	bytes := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		panic("encryption: failed to generate random bytes")
	}
	return bytes
}

func (s *Service) encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func (s *Service) decodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

func (s *Service) Encrypt(plaintext []byte) ([]byte, error) {
	if plaintext == nil || len(plaintext) == 0 {
		return nil, ErrNilInput
	}

	nonce := s.generateRandomBytes(s.gcm.NonceSize())

	ciphertext := s.gcm.Seal(
		make([]byte, 0, len(plaintext)+s.gcm.Overhead()),
		nonce,
		plaintext,
		nil,
	)

	return ciphertext, nil
}

func (s *Service) EncryptToBase64(plaintext []byte) (string, error) {
	ciphertext, err := s.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return s.encodeBase64(ciphertext), nil
}

func (s *Service) Decrypt(ciphertext []byte) ([]byte, error) {
	if ciphertext == nil {
		return nil, ErrNilInput
	}

	nonceSize := s.gcm.NonceSize()
	overhead := s.gcm.Overhead()

	if len(ciphertext) < nonceSize+overhead {
		return nil, ErrDataTooShort
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := s.gcm.Open(
		make([]byte, 0, len(encrypted)),
		nonce,
		encrypted,
		nil,
	)
	if err != nil {
		return nil, ErrAuthenticationFailed
	}

	return plaintext, nil
}

func (s *Service) DecryptFromBase64(encoded string) ([]byte, error) {
	ciphertext, err := s.decodeBase64(encoded)
	if err != nil {
		return nil, err
	}
	return s.Decrypt(ciphertext)
}

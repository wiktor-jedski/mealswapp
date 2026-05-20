package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const AlgorithmAES256GCM = "AES-256-GCM"

var (
	ErrInvalidKey        = errors.New("encryption key must be 32 bytes")
	ErrInvalidEnvelope   = errors.New("encryption envelope is invalid")
	ErrUnsupportedAlgo   = errors.New("encryption algorithm is unsupported")
	ErrInvalidCiphertext = errors.New("ciphertext authentication failed")
)

type Envelope struct {
	KeyVersion string `json:"keyVersion"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	Algorithm  string `json:"algorithm"`
}

type Service struct {
	key        []byte
	keyVersion string
	random     io.Reader
}

func NewService(key []byte, keyVersion string) (Service, error) {
	if len(key) != 32 {
		return Service{}, ErrInvalidKey
	}
	return Service{key: append([]byte(nil), key...), keyVersion: keyVersion, random: rand.Reader}, nil
}

func (service Service) Encrypt(plaintext []byte, additionalData []byte) (Envelope, error) {
	block, err := aes.NewCipher(service.key)
	if err != nil {
		return Envelope{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Envelope{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(service.random, nonce); err != nil {
		return Envelope{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, additionalData)
	return Envelope{
		KeyVersion: service.keyVersion,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Algorithm:  AlgorithmAES256GCM,
	}, nil
}

func (service Service) Decrypt(envelope Envelope, additionalData []byte) ([]byte, error) {
	if envelope.Algorithm != AlgorithmAES256GCM || envelope.KeyVersion == "" {
		return nil, ErrUnsupportedAlgo
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, ErrInvalidEnvelope
	}
	block, err := aes.NewCipher(service.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, ErrInvalidEnvelope
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}
	return plaintext, nil
}

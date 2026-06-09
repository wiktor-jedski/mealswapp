package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SigningKeyLoader resolves versioned JWT signing keys.
// Implements DESIGN-006 JWTManager.
type SigningKeyLoader interface {
	ActiveSigningKey(context.Context) (string, []byte, error)
	SigningKey(context.Context, string) ([]byte, error)
}

// JWTManager creates and validates access JWTs plus refresh-token hashes.
// Implements DESIGN-006 JWTManager.
type JWTManager struct {
	keys       SigningKeyLoader
	randomness io.Reader
	now        func() time.Time
}

// AccessTokenClaims are the trusted identity claims carried in access tokens.
// Implements DESIGN-006 JWTManager.
type AccessTokenClaims struct {
	UserID                 uuid.UUID
	Role                   string
	HasVerifiedLoginMethod bool
	SessionID              uuid.UUID
	RefreshFamilyID        uuid.UUID
	ExpiresAt              time.Time
	IssuedAt               time.Time
	KeyVersion             string
}

// RefreshToken stores opaque refresh-token material and its hash.
// Implements DESIGN-006 JWTManager.
type RefreshToken struct {
	Plaintext string
	Hash      string
}

// RefreshFamilyState describes refresh-token reuse detection results.
// Implements DESIGN-006 JWTManager.
type RefreshFamilyState string

// Implements DESIGN-006 JWTManager.
const (
	RefreshFamilyStateAccepted      RefreshFamilyState = "accepted"
	RefreshFamilyStateReuseDetected RefreshFamilyState = "reuse_detected"
	RefreshFamilyStateFamilyRevoked RefreshFamilyState = "family_revoked"
)

// NewJWTManager creates a token manager with versioned signing keys.
// Implements DESIGN-006 JWTManager.
func NewJWTManager(keys SigningKeyLoader) *JWTManager {
	return &JWTManager{keys: keys, randomness: rand.Reader, now: time.Now}
}

// CreateAccessToken signs access-token claims with the active key version.
// Implements DESIGN-006 JWTManager.
func (m *JWTManager) CreateAccessToken(ctx context.Context, claims AccessTokenClaims) (string, error) {
	if claims.UserID == uuid.Nil || claims.SessionID == uuid.Nil || claims.RefreshFamilyID == uuid.Nil || strings.TrimSpace(claims.Role) == "" {
		return "", errors.New("access token claims are incomplete")
	}
	if claims.ExpiresAt.IsZero() || !claims.ExpiresAt.After(m.now()) {
		return "", errors.New("access token expiry is invalid")
	}
	version, key, err := m.keys.ActiveSigningKey(ctx)
	if err != nil {
		return "", err
	}
	if len(key) < 32 {
		return "", errors.New("JWT signing key must contain at least 32 bytes")
	}
	issuedAt := claims.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = m.now()
	}
	payload := jwtPayload{
		Subject:                claims.UserID.String(),
		Role:                   claims.Role,
		HasVerifiedLoginMethod: claims.HasVerifiedLoginMethod,
		SessionID:              claims.SessionID.String(),
		RefreshFamilyID:        claims.RefreshFamilyID.String(),
		ExpiresAt:              claims.ExpiresAt.Unix(),
		IssuedAt:               issuedAt.Unix(),
	}
	header := jwtHeader{Algorithm: "HS256", Type: "JWT", KeyID: version}
	return signJWT(header, payload, key)
}

// ValidateAccessToken authenticates and parses an access token.
// Implements DESIGN-006 JWTManager.
func (m *JWTManager) ValidateAccessToken(ctx context.Context, token string) (AccessTokenClaims, error) {
	header, payload, signingInput, signature, err := parseJWT(token)
	if err != nil {
		return AccessTokenClaims{}, err
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" || strings.TrimSpace(header.KeyID) == "" {
		return AccessTokenClaims{}, errors.New("access token header is invalid")
	}
	key, err := m.keys.SigningKey(ctx, header.KeyID)
	if err != nil {
		return AccessTokenClaims{}, err
	}
	if len(key) < 32 {
		return AccessTokenClaims{}, errors.New("JWT signing key must contain at least 32 bytes")
	}
	expected := jwtSignature(signingInput, key)
	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return AccessTokenClaims{}, errors.New("access token signature is invalid")
	}
	claims, err := payload.toClaims(header.KeyID)
	if err != nil {
		return AccessTokenClaims{}, err
	}
	if !claims.ExpiresAt.After(m.now()) {
		return AccessTokenClaims{}, errors.New("access token expired")
	}
	return claims, nil
}

// CreateRefreshToken creates opaque refresh token material and its SHA-256 hash.
// Implements DESIGN-006 JWTManager.
func (m *JWTManager) CreateRefreshToken() (RefreshToken, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(m.randomness, raw); err != nil {
		return RefreshToken{}, err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	return RefreshToken{Plaintext: plain, Hash: HashRefreshToken(plain)}, nil
}

// HashRefreshToken hashes opaque refresh-token material for persistence.
// Implements DESIGN-006 JWTManager.
func HashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// VerifyRefreshTokenHash compares refresh-token material with a persisted hash.
// Implements DESIGN-006 JWTManager.
func VerifyRefreshTokenHash(plain string, expectedHash string) bool {
	actual := HashRefreshToken(plain)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}

// DecideRefreshFamilyState detects reuse and family revocation states.
// Implements DESIGN-006 JWTManager.
func DecideRefreshFamilyState(presentedHash string, currentHash string, tokenRevoked bool, familyRevoked bool) RefreshFamilyState {
	if familyRevoked {
		return RefreshFamilyStateFamilyRevoked
	}
	if tokenRevoked || subtle.ConstantTimeCompare([]byte(presentedHash), []byte(currentHash)) != 1 {
		return RefreshFamilyStateReuseDetected
	}
	return RefreshFamilyStateAccepted
}

// jwtHeader stores protected JWT metadata.
// Implements DESIGN-006 JWTManager.
type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

// jwtPayload stores access-token identity claims.
// Implements DESIGN-006 JWTManager.
type jwtPayload struct {
	Subject                string `json:"sub"`
	Role                   string `json:"role"`
	HasVerifiedLoginMethod bool   `json:"verified"`
	SessionID              string `json:"sid"`
	RefreshFamilyID        string `json:"rfid"`
	ExpiresAt              int64  `json:"exp"`
	IssuedAt               int64  `json:"iat"`
}

// toClaims validates required payload fields.
// Implements DESIGN-006 JWTManager.
func (p jwtPayload) toClaims(keyVersion string) (AccessTokenClaims, error) {
	userID, err := uuid.Parse(p.Subject)
	if err != nil {
		return AccessTokenClaims{}, errors.New("access token subject is invalid")
	}
	sessionID, err := uuid.Parse(p.SessionID)
	if err != nil {
		return AccessTokenClaims{}, errors.New("access token session is invalid")
	}
	refreshFamilyID, err := uuid.Parse(p.RefreshFamilyID)
	if err != nil {
		return AccessTokenClaims{}, errors.New("access token refresh family is invalid")
	}
	if strings.TrimSpace(p.Role) == "" || p.ExpiresAt == 0 || p.IssuedAt == 0 {
		return AccessTokenClaims{}, errors.New("access token claims are incomplete")
	}
	return AccessTokenClaims{UserID: userID, Role: p.Role, HasVerifiedLoginMethod: p.HasVerifiedLoginMethod, SessionID: sessionID, RefreshFamilyID: refreshFamilyID, ExpiresAt: time.Unix(p.ExpiresAt, 0), IssuedAt: time.Unix(p.IssuedAt, 0), KeyVersion: keyVersion}, nil
}

// signJWT serializes and signs a JWT.
// Implements DESIGN-006 JWTManager.
func signJWT(header jwtHeader, payload jwtPayload, key []byte) (string, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := jwtSignature([]byte(signingInput), key)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// parseJWT validates the JWT wire shape and decodes metadata.
// Implements DESIGN-006 JWTManager.
func parseJWT(token string) (jwtHeader, jwtPayload, []byte, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, jwtPayload{}, nil, nil, errors.New("access token is malformed")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, jwtPayload{}, nil, nil, errors.New("access token header is malformed")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, jwtPayload{}, nil, nil, errors.New("access token payload is malformed")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, jwtPayload{}, nil, nil, errors.New("access token signature is malformed")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtHeader{}, jwtPayload{}, nil, nil, errors.New("access token header is malformed")
	}
	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return jwtHeader{}, jwtPayload{}, nil, nil, errors.New("access token payload is malformed")
	}
	return header, payload, []byte(parts[0] + "." + parts[1]), signature, nil
}

// jwtSignature signs the JWT signing input.
// Implements DESIGN-006 JWTManager.
func jwtSignature(signingInput []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(signingInput)
	return mac.Sum(nil)
}

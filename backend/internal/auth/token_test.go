package auth

// Implements DESIGN-006 JWTManager verification.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type signingKeys struct {
	active  string
	entries map[string][]byte
	err     error
}

func (k signingKeys) ActiveSigningKey(context.Context) (string, []byte, error) {
	return k.active, k.entries[k.active], k.err
}

func (k signingKeys) SigningKey(_ context.Context, version string) ([]byte, error) {
	key, ok := k.entries[version]
	if !ok {
		return nil, errors.New("missing key")
	}
	return key, nil
}

// TestJWTManagerAccessTokens verifies DESIGN-006 JWTManager access-token lifecycle.
func TestJWTManagerAccessTokens(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	manager := NewJWTManager(signingKeys{active: "jwt-v2", entries: map[string][]byte{
		"jwt-v1": []byte("11111111111111111111111111111111"),
		"jwt-v2": []byte("22222222222222222222222222222222"),
	}})
	manager.now = func() time.Time { return now }
	claims := AccessTokenClaims{
		UserID:                 uuid.New(),
		Role:                   "admin",
		HasVerifiedLoginMethod: true,
		SessionID:              uuid.New(),
		RefreshFamilyID:        uuid.New(),
		ExpiresAt:              now.Add(15 * time.Minute),
	}
	token, err := manager.CreateAccessToken(ctx, claims)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	parsed, err := manager.ValidateAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if parsed.UserID != claims.UserID || parsed.Role != "admin" || !parsed.HasVerifiedLoginMethod || parsed.KeyVersion != "jwt-v2" {
		t.Fatalf("claims = %#v", parsed)
	}
	manager.now = func() time.Time { return now.Add(16 * time.Minute) }
	if _, err := manager.ValidateAccessToken(ctx, token); err == nil {
		t.Fatal("ValidateAccessToken() accepted expired token")
	}
}

// TestJWTManagerRejectsInvalidAccessTokens verifies DESIGN-006 JWTManager fail-closed validation.
func TestJWTManagerRejectsInvalidAccessTokens(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	manager := NewJWTManager(signingKeys{active: "jwt-v1", entries: map[string][]byte{"jwt-v1": []byte("11111111111111111111111111111111")}})
	manager.now = func() time.Time { return now }
	validClaims := AccessTokenClaims{UserID: uuid.New(), Role: "user", SessionID: uuid.New(), RefreshFamilyID: uuid.New(), ExpiresAt: now.Add(time.Minute)}
	token, err := manager.CreateAccessToken(ctx, validClaims)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1][:len(parts[1])-1] + "x." + parts[2]
	for _, candidate := range []string{"", "bad", "a.b.c", tampered} {
		if _, err := manager.ValidateAccessToken(ctx, candidate); err == nil {
			t.Fatalf("ValidateAccessToken() accepted %q", candidate)
		}
	}
	wrongKey := NewJWTManager(signingKeys{active: "jwt-v1", entries: map[string][]byte{"jwt-v1": []byte("22222222222222222222222222222222")}})
	wrongKey.now = func() time.Time { return now }
	if _, err := wrongKey.ValidateAccessToken(ctx, token); err == nil {
		t.Fatal("ValidateAccessToken() accepted wrong signature")
	}
	if _, err := manager.CreateAccessToken(ctx, AccessTokenClaims{Role: "user", ExpiresAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("CreateAccessToken() accepted missing claims")
	}
	if _, err := manager.CreateAccessToken(ctx, AccessTokenClaims{UserID: uuid.New(), Role: "user", SessionID: uuid.New(), RefreshFamilyID: uuid.New(), ExpiresAt: now.Add(-time.Minute)}); err == nil {
		t.Fatal("CreateAccessToken() accepted expired claims")
	}
	shortKey := NewJWTManager(signingKeys{active: "short", entries: map[string][]byte{"short": []byte("short")}})
	shortKey.now = func() time.Time { return now }
	if _, err := shortKey.CreateAccessToken(ctx, validClaims); err == nil {
		t.Fatal("CreateAccessToken() accepted short key")
	}
	down := NewJWTManager(signingKeys{err: errors.New("down")})
	down.now = func() time.Time { return now }
	if _, err := down.CreateAccessToken(ctx, validClaims); err == nil {
		t.Fatal("CreateAccessToken() accepted key loader failure")
	}
}

// TestJWTManagerRefreshTokens verifies DESIGN-006 JWTManager refresh-token hashing and reuse state.
func TestJWTManagerRefreshTokens(t *testing.T) {
	manager := NewJWTManager(signingKeys{})
	first, err := manager.CreateRefreshToken()
	if err != nil {
		t.Fatalf("CreateRefreshToken() error = %v", err)
	}
	second, err := manager.CreateRefreshToken()
	if err != nil {
		t.Fatalf("CreateRefreshToken() second error = %v", err)
	}
	if first.Plaintext == second.Plaintext || first.Hash == first.Plaintext || !VerifyRefreshTokenHash(first.Plaintext, first.Hash) {
		t.Fatalf("refresh tokens = %+v %+v", first, second)
	}
	if VerifyRefreshTokenHash("wrong", first.Hash) {
		t.Fatal("VerifyRefreshTokenHash() accepted wrong token")
	}
	if DecideRefreshFamilyState(first.Hash, first.Hash, false, false) != RefreshFamilyStateAccepted {
		t.Fatal("matching active token was not accepted")
	}
	if DecideRefreshFamilyState(second.Hash, first.Hash, false, false) != RefreshFamilyStateReuseDetected {
		t.Fatal("mismatched token did not trigger reuse detection")
	}
	if DecideRefreshFamilyState(first.Hash, first.Hash, true, false) != RefreshFamilyStateReuseDetected {
		t.Fatal("revoked token did not trigger reuse detection")
	}
	if DecideRefreshFamilyState(first.Hash, first.Hash, false, true) != RefreshFamilyStateFamilyRevoked {
		t.Fatal("revoked family was not reported")
	}
	manager.randomness = strings.NewReader("")
	if _, err := manager.CreateRefreshToken(); err == nil {
		t.Fatal("CreateRefreshToken() accepted randomness failure")
	}
}

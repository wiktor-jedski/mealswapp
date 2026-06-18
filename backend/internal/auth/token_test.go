package auth

// Implements DESIGN-006 JWTManager verification.

import (
	"context"
	"encoding/base64"
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

func TestJWTManagerRemainingValidationPaths(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	key := []byte("11111111111111111111111111111111")
	payload := jwtPayload{Subject: uuid.NewString(), Role: "user", SessionID: uuid.NewString(), RefreshFamilyID: uuid.NewString(), ExpiresAt: now.Add(time.Minute).Unix(), IssuedAt: now.Unix()}

	for _, header := range []jwtHeader{
		{Algorithm: "bad", Type: "JWT", KeyID: "jwt-v1"},
		{Algorithm: "HS256", Type: "bad", KeyID: "jwt-v1"},
		{Algorithm: "HS256", Type: "JWT"},
	} {
		token, err := signJWT(header, payload, key)
		if err != nil {
			t.Fatal(err)
		}
		manager := NewJWTManager(signingKeys{entries: map[string][]byte{"jwt-v1": key}})
		manager.now = func() time.Time { return now }
		if _, err := manager.ValidateAccessToken(ctx, token); err == nil {
			t.Fatalf("ValidateAccessToken() accepted header %+v", header)
		}
	}

	valid, err := signJWT(jwtHeader{Algorithm: "HS256", Type: "JWT", KeyID: "missing"}, payload, key)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewJWTManager(signingKeys{entries: map[string][]byte{}})
	manager.now = func() time.Time { return now }
	if _, err := manager.ValidateAccessToken(ctx, valid); err == nil {
		t.Fatal("ValidateAccessToken() accepted missing key")
	}
	short, err := signJWT(jwtHeader{Algorithm: "HS256", Type: "JWT", KeyID: "short"}, payload, []byte("short"))
	if err != nil {
		t.Fatal(err)
	}
	manager = NewJWTManager(signingKeys{entries: map[string][]byte{"short": []byte("short")}})
	manager.now = func() time.Time { return now }
	if _, err := manager.ValidateAccessToken(ctx, short); err == nil {
		t.Fatal("ValidateAccessToken() accepted short key")
	}

	malformed := []string{
		"!.e30.c2ln",
		"e30.!.c2ln",
		"e30.e30.!",
		base64.RawURLEncoding.EncodeToString([]byte("{")) + ".e30.c2ln",
		"e30." + base64.RawURLEncoding.EncodeToString([]byte("{")) + ".c2ln",
	}
	for _, token := range malformed {
		if _, _, _, _, err := parseJWT(token); err == nil {
			t.Fatalf("parseJWT() accepted %q", token)
		}
	}

	invalidPayloads := []jwtPayload{
		{Subject: "bad", SessionID: uuid.NewString(), RefreshFamilyID: uuid.NewString(), Role: "user", ExpiresAt: 1, IssuedAt: 1},
		{Subject: uuid.NewString(), SessionID: "bad", RefreshFamilyID: uuid.NewString(), Role: "user", ExpiresAt: 1, IssuedAt: 1},
		{Subject: uuid.NewString(), SessionID: uuid.NewString(), RefreshFamilyID: "bad", Role: "user", ExpiresAt: 1, IssuedAt: 1},
		{Subject: uuid.NewString(), SessionID: uuid.NewString(), RefreshFamilyID: uuid.NewString()},
	}
	for _, candidate := range invalidPayloads {
		if _, err := candidate.toClaims("jwt-v1"); err == nil {
			t.Fatalf("toClaims() accepted %+v", candidate)
		}
	}
	invalidClaimsToken, err := signJWT(jwtHeader{Algorithm: "HS256", Type: "JWT", KeyID: "jwt-v1"}, invalidPayloads[0], key)
	if err != nil {
		t.Fatal(err)
	}
	manager = NewJWTManager(signingKeys{entries: map[string][]byte{"jwt-v1": key}})
	manager.now = func() time.Time { return now }
	if _, err := manager.ValidateAccessToken(ctx, invalidClaimsToken); err == nil {
		t.Fatal("ValidateAccessToken() accepted invalid signed claims")
	}
}

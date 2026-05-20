package token

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestManagerIssuesAndValidatesAccessToken(t *testing.T) {
	manager := testManager()
	userID := uuid.New()

	pair, err := manager.IssuePair(UserClaims{UserID: userID, Email: "user@example.com", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}

	claims, err := manager.ValidateAccess(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != userID.String() || claims.Email != "user@example.com" || claims.Role != "user" || claims.TokenType != TokenTypeAccess {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestManagerRejectsExpiredTokens(t *testing.T) {
	manager := testManager()
	pair, err := manager.IssuePair(UserClaims{UserID: uuid.New(), Email: "user@example.com", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}

	manager.now = func() time.Time {
		return time.Date(2026, 5, 19, 12, 16, 0, 0, time.UTC)
	}

	_, err = manager.ValidateAccess(pair.AccessToken)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected expired token, got %v", err)
	}
}

func TestManagerRotatesRefreshTokenAndInvalidatesOldToken(t *testing.T) {
	manager := testManager()
	pair, err := manager.IssuePair(UserClaims{UserID: uuid.New(), Email: "user@example.com", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := manager.RotateRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == pair.RefreshToken || rotated.AccessToken == pair.AccessToken {
		t.Fatal("expected rotated tokens")
	}

	_, err = manager.ValidateRefresh(pair.RefreshToken)
	if !errors.Is(err, ErrRevokedToken) {
		t.Fatalf("expected old refresh token revoked, got %v", err)
	}
}

func TestManagerRejectsMalformedClaims(t *testing.T) {
	manager := testManager()
	claims := Claims{
		Email:     "user@example.com",
		Role:      "user",
		TokenType: TokenTypeAccess,
		ID:        uuid.NewString(),
		IssuedAt:  manager.now().Unix(),
		ExpiresAt: manager.now().Add(time.Minute).Unix(),
	}
	token, err := manager.sign(claims)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.ValidateAccess(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestManagerRejectsTamperedSignature(t *testing.T) {
	manager := testManager()
	pair, err := manager.IssuePair(UserClaims{UserID: uuid.New(), Email: "user@example.com", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(pair.AccessToken, ".")
	var claims Claims
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	claims.Role = "admin"
	payload, err = json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	parts[1] = base64.RawURLEncoding.EncodeToString(payload)

	_, err = manager.ValidateAccess(strings.Join(parts, "."))
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func testManager() *Manager {
	manager := NewManager([]byte("test-secret-with-enough-length"), 15*time.Minute, 7*24*time.Hour)
	manager.now = func() time.Time {
		return time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	}
	return &manager
}

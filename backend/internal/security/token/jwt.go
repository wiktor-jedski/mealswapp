package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
	ErrRevokedToken = errors.New("revoked token")
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type UserClaims struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

type Claims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"typ"`
	ID        string `json:"jti"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time

	mu      sync.Mutex
	revoked map[string]struct{}
}

func NewManager(secret []byte, accessTTL time.Duration, refreshTTL time.Duration) Manager {
	return Manager{
		secret:     append([]byte(nil), secret...),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
		revoked:    make(map[string]struct{}),
	}
}

func (manager *Manager) IssuePair(user UserClaims) (TokenPair, error) {
	now := manager.now().UTC()
	accessExpiresAt := now.Add(manager.accessTTL)
	refreshExpiresAt := now.Add(manager.refreshTTL)

	accessToken, err := manager.issue(user, TokenTypeAccess, now, accessExpiresAt)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, err := manager.issue(user, TokenTypeRefresh, now, refreshExpiresAt)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (manager *Manager) ValidateAccess(token string) (Claims, error) {
	return manager.validate(token, TokenTypeAccess)
}

func (manager *Manager) ValidateRefresh(token string) (Claims, error) {
	return manager.validate(token, TokenTypeRefresh)
}

func (manager *Manager) RotateRefresh(refreshToken string) (TokenPair, error) {
	claims, err := manager.ValidateRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, err
	}

	manager.RevokeID(claims.ID)
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}

	return manager.IssuePair(UserClaims{UserID: userID, Email: claims.Email, Role: claims.Role})
}

func (manager *Manager) Revoke(token string) error {
	claims, err := manager.parse(token)
	if err != nil {
		return err
	}
	manager.RevokeID(claims.ID)
	return nil
}

func (manager *Manager) RevokeID(id string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.revoked[id] = struct{}{}
}

func (manager *Manager) issue(user UserClaims, tokenType string, issuedAt time.Time, expiresAt time.Time) (string, error) {
	claims := Claims{
		Subject:   user.UserID.String(),
		Email:     user.Email,
		Role:      user.Role,
		TokenType: tokenType,
		ID:        uuid.NewString(),
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}
	return manager.sign(claims)
}

func (manager *Manager) validate(token string, tokenType string) (Claims, error) {
	claims, err := manager.parse(token)
	if err != nil {
		return Claims{}, err
	}
	if claims.TokenType != tokenType {
		return Claims{}, ErrInvalidToken
	}
	if manager.isRevoked(claims.ID) {
		return Claims{}, ErrRevokedToken
	}
	if !manager.now().UTC().Before(time.Unix(claims.ExpiresAt, 0)) {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

func (manager *Manager) parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signed := parts[0] + "." + parts[1]
	expectedSignature := manager.signature(signed)
	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !hmac.Equal(actualSignature, expectedSignature) {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !claims.validShape() {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (manager *Manager) sign(claims Claims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signed := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	return signed + "." + base64.RawURLEncoding.EncodeToString(manager.signature(signed)), nil
}

func (manager *Manager) signature(signed string) []byte {
	mac := hmac.New(sha256.New, manager.secret)
	mac.Write([]byte(signed))
	return mac.Sum(nil)
}

func (manager *Manager) isRevoked(id string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	_, revoked := manager.revoked[id]
	return revoked
}

func (claims Claims) validShape() bool {
	if claims.Subject == "" || claims.TokenType == "" || claims.ID == "" || claims.ExpiresAt == 0 || claims.IssuedAt == 0 {
		return false
	}
	if claims.TokenType != TokenTypeAccess && claims.TokenType != TokenTypeRefresh {
		return false
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return false
	}
	return true
}

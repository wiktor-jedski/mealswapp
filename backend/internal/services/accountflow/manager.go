package accountflow

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenUsed    = errors.New("token already used")
)

type TokenRecord struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type TokenStore interface {
	StorePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetPasswordReset(ctx context.Context, tokenHash string) (TokenRecord, error)
	MarkPasswordResetUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
	StoreEmailVerification(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetEmailVerification(ctx context.Context, tokenHash string) (TokenRecord, error)
	MarkEmailVerificationUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
	UpdatePasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type Manager struct {
	store     TokenStore
	hasher    PasswordHasher
	now       func() time.Time
	resetTTL  time.Duration
	verifyTTL time.Duration
}

func NewManager(store TokenStore, hasher PasswordHasher) Manager {
	return Manager{
		store:     store,
		hasher:    hasher,
		now:       time.Now,
		resetTTL:  time.Hour,
		verifyTTL: 24 * time.Hour,
	}
}

func (manager Manager) CreatePasswordReset(ctx context.Context, userID uuid.UUID) (string, error) {
	token, tokenHash, err := newToken()
	if err != nil {
		return "", err
	}
	return token, manager.store.StorePasswordReset(ctx, userID, tokenHash, manager.now().UTC().Add(manager.resetTTL))
}

func (manager Manager) ConsumePasswordReset(ctx context.Context, token string, newPassword string) error {
	tokenHash := hashToken(token)
	record, err := manager.store.GetPasswordReset(ctx, tokenHash)
	if err != nil {
		return err
	}
	if err := manager.ensureUsable(record); err != nil {
		return err
	}

	passwordHash, err := manager.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := manager.store.UpdatePasswordHash(ctx, record.UserID, passwordHash); err != nil {
		return err
	}
	return manager.store.MarkPasswordResetUsed(ctx, tokenHash, manager.now().UTC())
}

func (manager Manager) CreateEmailVerification(ctx context.Context, userID uuid.UUID) (string, error) {
	token, tokenHash, err := newToken()
	if err != nil {
		return "", err
	}
	return token, manager.store.StoreEmailVerification(ctx, userID, tokenHash, manager.now().UTC().Add(manager.verifyTTL))
}

func (manager Manager) ConsumeEmailVerification(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	record, err := manager.store.GetEmailVerification(ctx, tokenHash)
	if err != nil {
		return err
	}
	if err := manager.ensureUsable(record); err != nil {
		return err
	}

	if err := manager.store.MarkEmailVerified(ctx, record.UserID); err != nil {
		return err
	}
	return manager.store.MarkEmailVerificationUsed(ctx, tokenHash, manager.now().UTC())
}

func (manager Manager) ensureUsable(record TokenRecord) error {
	if record.UsedAt != nil {
		return apperrors.Conflict("Token already used")
	}
	if !manager.now().UTC().Before(record.ExpiresAt) {
		return apperrors.AppError{Category: apperrors.CategoryValidation, Code: "token_expired", Message: "Token expired", Status: 400}
	}
	return nil
}

func newToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

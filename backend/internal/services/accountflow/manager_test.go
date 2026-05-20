package accountflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
)

func TestCreateAndConsumePasswordReset(t *testing.T) {
	store := newFakeTokenStore()
	manager := NewManager(store, fakeHasher{})
	userID := uuid.New()

	token, err := manager.CreatePasswordReset(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected reset token")
	}

	if err := manager.ConsumePasswordReset(context.Background(), token, "NewPassword1!"); err != nil {
		t.Fatal(err)
	}
	if store.passwordUserID != userID || store.passwordHash != "hashed:NewPassword1!" {
		t.Fatalf("expected password update, got user=%s hash=%q", store.passwordUserID, store.passwordHash)
	}
}

func TestPasswordResetRejectsExpiredAndUsedTokens(t *testing.T) {
	store := newFakeTokenStore()
	manager := NewManager(store, fakeHasher{})
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	userID := uuid.New()

	token, err := manager.CreatePasswordReset(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour + time.Second)
	err = manager.ConsumePasswordReset(context.Background(), token, "NewPassword1!")
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "token_expired" {
		t.Fatalf("expected token_expired, got %v", err)
	}

	now = time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	token, err = manager.CreatePasswordReset(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ConsumePasswordReset(context.Background(), token, "NewPassword1!"); err != nil {
		t.Fatal(err)
	}
	err = manager.ConsumePasswordReset(context.Background(), token, "NewPassword1!")
	appErr, ok = apperrors.As(err)
	if !ok || appErr.Code != "conflict" {
		t.Fatalf("expected used-token conflict, got %v", err)
	}
}

func TestCreateAndConsumeEmailVerification(t *testing.T) {
	store := newFakeTokenStore()
	manager := NewManager(store, fakeHasher{})
	userID := uuid.New()

	token, err := manager.CreateEmailVerification(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ConsumeEmailVerification(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if store.verifiedUserID != userID {
		t.Fatalf("expected verified user %s, got %s", userID, store.verifiedUserID)
	}
}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password required")
	}
	return "hashed:" + password, nil
}

type fakeTokenStore struct {
	passwordResets     map[string]TokenRecord
	emailVerifications map[string]TokenRecord
	passwordUserID     uuid.UUID
	passwordHash       string
	verifiedUserID     uuid.UUID
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{
		passwordResets:     make(map[string]TokenRecord),
		emailVerifications: make(map[string]TokenRecord),
	}
}

func (store *fakeTokenStore) StorePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	store.passwordResets[tokenHash] = TokenRecord{UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (store *fakeTokenStore) GetPasswordReset(ctx context.Context, tokenHash string) (TokenRecord, error) {
	return store.passwordResets[tokenHash], nil
}

func (store *fakeTokenStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	record := store.passwordResets[tokenHash]
	record.UsedAt = &usedAt
	store.passwordResets[tokenHash] = record
	return nil
}

func (store *fakeTokenStore) StoreEmailVerification(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	store.emailVerifications[tokenHash] = TokenRecord{UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (store *fakeTokenStore) GetEmailVerification(ctx context.Context, tokenHash string) (TokenRecord, error) {
	return store.emailVerifications[tokenHash], nil
}

func (store *fakeTokenStore) MarkEmailVerificationUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	record := store.emailVerifications[tokenHash]
	record.UsedAt = &usedAt
	store.emailVerifications[tokenHash] = record
	return nil
}

func (store *fakeTokenStore) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	store.verifiedUserID = userID
	return nil
}

func (store *fakeTokenStore) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	store.passwordUserID = userID
	store.passwordHash = passwordHash
	return nil
}

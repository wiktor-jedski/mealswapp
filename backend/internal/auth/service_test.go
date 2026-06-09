package auth

// Implements DESIGN-006 AuthController verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type authKeyLoader struct {
	activeEncryption string
	activeLookup     string
	encryption       map[string][]byte
	lookup           map[string][]byte
}

func (l authKeyLoader) ActiveKey(context.Context) (string, []byte, error) {
	return l.activeEncryption, l.encryption[l.activeEncryption], nil
}

func (l authKeyLoader) Key(_ context.Context, version string) ([]byte, error) {
	key, ok := l.encryption[version]
	if !ok {
		return nil, errors.New("missing encryption key")
	}
	return key, nil
}

func (l authKeyLoader) ActiveLookupKey(context.Context) (string, []byte, error) {
	return l.activeLookup, l.lookup[l.activeLookup], nil
}

func (l authKeyLoader) LookupKey(_ context.Context, version string) ([]byte, error) {
	key, ok := l.lookup[version]
	if !ok {
		return nil, errors.New("missing lookup key")
	}
	return key, nil
}

type memoryIdentityRepository struct {
	byDigest map[repository.LookupDigest]repository.EncryptedAuthUser
	byID     map[uuid.UUID]repository.EncryptedAuthUser
	oauth    map[string]repository.EncryptedOAuthIdentity
	created  []repository.EncryptedAuthUser
}

func (r *memoryIdentityRepository) CreateUser(_ context.Context, user repository.EncryptedAuthUser) (uuid.UUID, error) {
	if r.byDigest == nil {
		r.byDigest = map[repository.LookupDigest]repository.EncryptedAuthUser{}
	}
	if r.byID == nil {
		r.byID = map[uuid.UUID]repository.EncryptedAuthUser{}
	}
	user.ID = uuid.New()
	r.byDigest[user.NormalizedEmailDigest] = user
	r.byID[user.ID] = user
	r.created = append(r.created, user)
	return user.ID, nil
}

func (r *memoryIdentityRepository) GetUserByNormalizedEmailDigest(_ context.Context, digest repository.LookupDigest) (repository.EncryptedAuthUser, error) {
	user, ok := r.byDigest[digest]
	if !ok {
		return repository.EncryptedAuthUser{}, repository.NewError(repository.ErrorKindNotFound, "user not found", nil)
	}
	return user, nil
}

func (r *memoryIdentityRepository) UpsertOAuthIdentity(_ context.Context, identity repository.EncryptedOAuthIdentity) (uuid.UUID, error) {
	if r.oauth == nil {
		r.oauth = map[string]repository.EncryptedOAuthIdentity{}
	}
	identity.ID = uuid.New()
	r.oauth[identity.Provider+"|"+identity.ProviderUserIDDigest.Value] = identity
	return identity.ID, nil
}

func (r *memoryIdentityRepository) GetOAuthIdentity(_ context.Context, provider string, digest repository.LookupDigest) (repository.EncryptedOAuthIdentity, error) {
	identity, ok := r.oauth[provider+"|"+digest.Value]
	if !ok {
		return repository.EncryptedOAuthIdentity{}, repository.NewError(repository.ErrorKindNotFound, "oauth identity not found", nil)
	}
	return identity, nil
}

func (r *memoryIdentityRepository) GetEncryptedUserByID(_ context.Context, id uuid.UUID) (repository.EncryptedAuthUser, error) {
	user, ok := r.byID[id]
	if !ok {
		return repository.EncryptedAuthUser{}, repository.NewError(repository.ErrorKindNotFound, "user not found", nil)
	}
	return user, nil
}

type memorySessionRepository struct {
	byHash        map[string]repository.UserSession
	created       []repository.UserSession
	revoked       map[uuid.UUID]bool
	revokedFamily map[uuid.UUID]bool
}

func (r *memorySessionRepository) CreateSession(_ context.Context, session repository.UserSession) (uuid.UUID, error) {
	if r.byHash == nil {
		r.byHash = map[string]repository.UserSession{}
	}
	session.ID = uuid.New()
	session.CreatedAt = time.Now()
	r.byHash[session.RefreshTokenHash] = session
	r.created = append(r.created, session)
	return session.ID, nil
}

func (r *memorySessionRepository) GetSessionByRefreshTokenHash(_ context.Context, hash string) (repository.UserSession, error) {
	session, ok := r.byHash[hash]
	if !ok {
		return repository.UserSession{}, repository.NewError(repository.ErrorKindNotFound, "session not found", nil)
	}
	return session, nil
}

func (r *memorySessionRepository) RevokeSession(_ context.Context, sessionID uuid.UUID) error {
	if r.revoked == nil {
		r.revoked = map[uuid.UUID]bool{}
	}
	r.revoked[sessionID] = true
	for hash, session := range r.byHash {
		if session.ID == sessionID {
			now := time.Now()
			session.RevokedAt = &now
			r.byHash[hash] = session
		}
	}
	return nil
}

func (r *memorySessionRepository) RevokeSessionFamily(_ context.Context, familyID uuid.UUID) error {
	if r.revokedFamily == nil {
		r.revokedFamily = map[uuid.UUID]bool{}
	}
	r.revokedFamily[familyID] = true
	return nil
}

func (r *memorySessionRepository) RevokeUserSessions(_ context.Context, userID uuid.UUID) error {
	if r.revoked == nil {
		r.revoked = map[uuid.UUID]bool{}
	}
	for hash, session := range r.byHash {
		if session.UserID == userID {
			r.revoked[session.ID] = true
			now := time.Now()
			session.RevokedAt = &now
			r.byHash[hash] = session
		}
	}
	return nil
}

type memoryVerificationRepository struct {
	verified        map[uuid.UUID]bool
	passwordUpdates map[uuid.UUID]string
}

func (r *memoryVerificationRepository) MarkEmailVerified(_ context.Context, userID uuid.UUID) error {
	if r.verified == nil {
		r.verified = map[uuid.UUID]bool{}
	}
	r.verified[userID] = true
	return nil
}

func (r *memoryVerificationRepository) UpdatePassword(_ context.Context, userID uuid.UUID, passwordHash string, passwordSalt string) error {
	if r.passwordUpdates == nil {
		r.passwordUpdates = map[uuid.UUID]string{}
	}
	r.passwordUpdates[userID] = passwordHash + ":" + passwordSalt
	return nil
}

type memoryResetTokenRepository struct {
	tokens map[string]repository.PasswordResetToken
}

func (r *memoryResetTokenRepository) CreatePasswordResetToken(_ context.Context, token repository.PasswordResetToken) error {
	if r.tokens == nil {
		r.tokens = map[string]repository.PasswordResetToken{}
	}
	r.tokens[token.TokenHash] = token
	return nil
}

func (r *memoryResetTokenRepository) ConsumePasswordResetToken(_ context.Context, tokenHash string, usedAt time.Time) (repository.PasswordResetToken, error) {
	token, ok := r.tokens[tokenHash]
	if !ok || token.UsedAt != nil || !token.ExpiresAt.After(usedAt) {
		return repository.PasswordResetToken{}, repository.NewError(repository.ErrorKindNotFound, "reset not found", nil)
	}
	token.UsedAt = &usedAt
	r.tokens[tokenHash] = token
	return token, nil
}

type memoryTrialHook struct {
	called []uuid.UUID
}

func (h *memoryTrialHook) ActivateFirstLoginTrial(_ context.Context, userID uuid.UUID) error {
	h.called = append(h.called, userID)
	return nil
}

// TestCoreAuthServiceLoginRefreshAndReuse verifies DESIGN-006 AuthController service composition.
func TestCoreAuthServiceLoginRefreshAndReuse(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	hasher, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12})
	if err != nil {
		t.Fatal(err)
	}
	passwordHash, passwordSalt, err := hasher.HashPassword("StrongerPassword1!")
	if err != nil {
		t.Fatal(err)
	}
	keys := authKeyLoader{
		activeEncryption: "pii-v1",
		activeLookup:     "lookup-v1",
		encryption:       map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")},
		lookup:           map[string][]byte{"lookup-v1": []byte("22222222222222222222222222222222")},
	}
	digest, err := security.NewLookupDigestService(keys).DigestForWrite(ctx, []byte("user@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	user := repository.EncryptedAuthUser{ID: userID, NormalizedEmailDigest: repository.LookupDigest{KeyVersion: digest.KeyVersion, Value: digest.Value}, EmailVerified: true, Role: repository.UserRoleAdmin, PasswordHash: &passwordHash, PasswordSalt: &passwordSalt}
	identities := &memoryIdentityRepository{byDigest: map[repository.LookupDigest]repository.EncryptedAuthUser{user.NormalizedEmailDigest: user}, byID: map[uuid.UUID]repository.EncryptedAuthUser{userID: user}}
	sessions := &memorySessionRepository{}
	verification := &memoryVerificationRepository{}
	resetTokens := &memoryResetTokenRepository{}
	lockouts := &memoryLockoutRepository{state: repository.AccountLockoutState{UserID: userID}}
	lockout := NewAccountLockoutTracker(lockouts)
	lockout.now = func() time.Time { return now }
	manager := NewJWTManager(signingKeys{active: "jwt-v1", entries: map[string][]byte{"jwt-v1": []byte("33333333333333333333333333333333")}})
	manager.now = func() time.Time { return now }
	service := NewCoreAuthService(CoreAuthDependencies{Config: CoreAuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour}, Identities: identities, Sessions: sessions, Verification: verification, ResetTokens: resetTokens, Lockout: lockout, Hasher: hasher, Tokens: manager, Encryption: security.NewEncryptionService(keys), Digests: security.NewLookupDigestService(keys)})
	service.now = func() time.Time { return now }

	if _, err := service.Login(ctx, "user@example.test", "WrongPassword1!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password err = %v, want invalid credentials", err)
	}
	if lockouts.state.FailedLoginCount != 1 {
		t.Fatalf("failure count = %d, want 1", lockouts.state.FailedLoginCount)
	}
	session, err := service.Login(ctx, " user@example.test ", "StrongerPassword1!")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if session.UserID != userID || session.Role != "admin" || !session.HasVerifiedLoginMethod || session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatalf("login session = %#v", session)
	}
	if lockouts.state.FailedLoginCount != 0 || len(sessions.created) != 1 {
		t.Fatalf("post-login lockout/session = %#v %d", lockouts.state, len(sessions.created))
	}
	refreshed, err := service.Refresh(ctx, session.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.RefreshToken == session.RefreshToken || len(sessions.created) != 2 || !sessions.revoked[sessions.created[0].ID] {
		t.Fatalf("refresh rotation failed refreshed=%#v created=%d revoked=%v", refreshed, len(sessions.created), sessions.revoked)
	}
	revoked := sessions.created[1]
	reusedHash := HashRefreshToken(refreshed.RefreshToken)
	nowTime := now
	revoked.RevokedAt = &nowTime
	sessions.byHash[reusedHash] = revoked
	if _, err := service.Refresh(ctx, refreshed.RefreshToken); !errors.Is(err, ErrTokenReuseDetected) {
		t.Fatalf("reuse err = %v, want token reuse", err)
	}
	if !sessions.revokedFamily[revoked.RefreshFamilyID] {
		t.Fatal("refresh reuse did not revoke family")
	}
	if err := service.MarkEmailVerified(ctx, userID); err != nil || !verification.verified[userID] {
		t.Fatalf("MarkEmailVerified() err=%v verified=%v", err, verification.verified[userID])
	}
	resetToken, err := service.RequestPasswordReset(ctx, "user@example.test")
	if err != nil || resetToken == "" {
		t.Fatalf("RequestPasswordReset() token=%q err=%v", resetToken, err)
	}
	if len(resetTokens.tokens) != 1 {
		t.Fatalf("reset tokens = %#v", resetTokens.tokens)
	}
	if _, ok := resetTokens.tokens[resetToken]; ok {
		t.Fatal("reset token repository stored plaintext token")
	}
	if _, ok := resetTokens.tokens[HashRefreshToken(resetToken)]; !ok {
		t.Fatal("reset token repository did not store hashed token")
	}
	if missingToken, err := service.RequestPasswordReset(ctx, "missing@example.test"); err != nil || missingToken != "" {
		t.Fatalf("missing reset token=%q err=%v", missingToken, err)
	}
	if err := service.ConsumePasswordReset(ctx, resetToken, "NewPassword1!"); err != nil {
		t.Fatalf("ConsumePasswordReset() error = %v", err)
	}
	if verification.passwordUpdates[userID] == "" || !sessions.revoked[sessions.created[0].ID] {
		t.Fatalf("reset side effects password=%q revoked=%v", verification.passwordUpdates[userID], sessions.revoked)
	}
	if err := service.ConsumePasswordReset(ctx, resetToken, "NewPassword1!"); !errors.Is(err, ErrPasswordResetInvalid) {
		t.Fatalf("reuse reset err = %v, want invalid", err)
	}
}

// TestCoreAuthServiceOAuthBoundary verifies DESIGN-006 OAuthHandler service composition.
func TestCoreAuthServiceOAuthBoundary(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	keys := authKeyLoader{
		activeEncryption: "pii-v1",
		activeLookup:     "lookup-v1",
		encryption:       map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")},
		lookup:           map[string][]byte{"lookup-v1": []byte("22222222222222222222222222222222")},
	}
	manager := NewJWTManager(signingKeys{active: "jwt-v1", entries: map[string][]byte{"jwt-v1": []byte("33333333333333333333333333333333")}})
	manager.now = func() time.Time { return now }
	digests := security.NewLookupDigestService(keys)
	identities := &memoryIdentityRepository{byDigest: map[repository.LookupDigest]repository.EncryptedAuthUser{}, byID: map[uuid.UUID]repository.EncryptedAuthUser{}, oauth: map[string]repository.EncryptedOAuthIdentity{}}
	sessions := &memorySessionRepository{}
	verification := &memoryVerificationRepository{}
	trials := &memoryTrialHook{}
	service := NewCoreAuthService(CoreAuthDependencies{Config: CoreAuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour}, Identities: identities, Sessions: sessions, Verification: verification, OAuthTrial: trials, Tokens: manager, Encryption: security.NewEncryptionService(keys), Digests: digests})
	service.now = func() time.Time { return now }

	existingDigest, err := digests.DigestForWrite(ctx, []byte("google-user-1"))
	if err != nil {
		t.Fatal(err)
	}
	existingUser := repository.EncryptedAuthUser{ID: uuid.New(), EmailVerified: false, Role: repository.UserRoleUser}
	identities.byID[existingUser.ID] = existingUser
	identities.oauth["google|"+existingDigest.Value] = repository.EncryptedOAuthIdentity{UserID: existingUser.ID, Provider: "google", ProviderUserIDDigest: repository.LookupDigest{KeyVersion: existingDigest.KeyVersion, Value: existingDigest.Value}}
	result, err := service.CompleteOAuth(ctx, "google", OAuthProfile{Provider: "google", ProviderUserID: "google-user-1", Email: "linked@example.test", EmailVerified: true})
	if err != nil {
		t.Fatalf("CompleteOAuth() existing error = %v", err)
	}
	if result.Session.UserID != existingUser.ID || !result.Session.HasVerifiedLoginMethod || !verification.verified[existingUser.ID] || len(trials.called) != 0 {
		t.Fatalf("existing oauth result=%#v verified=%v trials=%d", result, verification.verified, len(trials.called))
	}
	if _, err := service.CompleteOAuth(ctx, "apple", OAuthProfile{Provider: "google", ProviderUserID: "google-user-2", Email: "wrong@example.test"}); !errors.Is(err, ErrOAuthProviderMismatch) {
		t.Fatalf("provider mismatch err = %v", err)
	}

	emailDigest, err := digests.DigestForWrite(ctx, []byte("match@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	matchedUser := repository.EncryptedAuthUser{ID: uuid.New(), NormalizedEmailDigest: repository.LookupDigest{KeyVersion: emailDigest.KeyVersion, Value: emailDigest.Value}, Role: repository.UserRoleUser}
	identities.byDigest[matchedUser.NormalizedEmailDigest] = matchedUser
	identities.byID[matchedUser.ID] = matchedUser
	if _, err := service.CompleteOAuth(ctx, "google", OAuthProfile{Provider: "google", ProviderUserID: "google-user-3", Email: "match@example.test"}); err == nil {
		t.Fatal("CompleteOAuth() accepted email match without explicit linking")
	} else {
		var linkRequired *OAuthLinkRequired
		if !errors.As(err, &linkRequired) || linkRequired.UserID != matchedUser.ID {
			t.Fatalf("link required err = %#v", err)
		}
	}

	created, err := service.CompleteOAuth(ctx, "apple", OAuthProfile{Provider: "apple", ProviderUserID: "apple-user-1", Email: "new-oauth@example.test", EmailVerified: true})
	if err != nil {
		t.Fatalf("CompleteOAuth() create error = %v", err)
	}
	if !created.CreatedUser || !created.Linked || !created.Session.HasVerifiedLoginMethod || len(trials.called) != 1 || trials.called[0] != created.Session.UserID {
		t.Fatalf("created oauth result=%#v trials=%v", created, trials.called)
	}
	appleDigest, err := digests.DigestForWrite(ctx, []byte("apple-user-1"))
	if err != nil {
		t.Fatal(err)
	}
	stored := identities.oauth["apple|"+appleDigest.Value]
	if stored.UserID != created.Session.UserID || stored.ProviderUserIDDigest.Value != appleDigest.Value || string(stored.ProviderUserID.Ciphertext) == "apple-user-1" {
		t.Fatalf("stored oauth identity = %#v", stored)
	}
	if err := service.LinkOAuthIdentity(ctx, matchedUser.ID, "google", OAuthProfile{Provider: "google", ProviderUserID: "google-user-3", Email: "match@example.test", EmailVerified: true}); err != nil {
		t.Fatalf("LinkOAuthIdentity() error = %v", err)
	}
	if !verification.verified[matchedUser.ID] {
		t.Fatal("explicit OAuth link did not update verified projection")
	}
}

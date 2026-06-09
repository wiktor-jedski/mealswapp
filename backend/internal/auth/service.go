package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// CoreAuthConfig carries token lifetimes for core account flows.
// Implements DESIGN-006 AuthController.
type CoreAuthConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// EncryptedIdentityRepository loads encrypted account identity for authentication.
// Implements DESIGN-006 AuthController.
type EncryptedIdentityRepository interface {
	GetUserByNormalizedEmailDigest(context.Context, repository.LookupDigest) (repository.EncryptedAuthUser, error)
	GetEncryptedUserByID(context.Context, uuid.UUID) (repository.EncryptedAuthUser, error)
}

// CoreAuthService composes registration, login, refresh, and logout behavior.
// Implements DESIGN-006 AuthController.
type CoreAuthService struct {
	cfg          CoreAuthConfig
	registration *RegistrationService
	identities   EncryptedIdentityRepository
	sessions     repository.SessionRepository
	verification repository.AccountVerificationRepository
	resetTokens  repository.PasswordResetTokenRepository
	oauthTrial   OAuthTrialHook
	lockout      *AccountLockoutTracker
	hasher       *PasswordHasher
	tokens       *JWTManager
	encryption   *security.EncryptionService
	digests      *security.LookupDigestService
	now          func() time.Time
}

// CoreAuthDependencies groups core auth collaborators.
// Implements DESIGN-006 AuthController.
type CoreAuthDependencies struct {
	Config       CoreAuthConfig
	Registration *RegistrationService
	Identities   EncryptedIdentityRepository
	Sessions     repository.SessionRepository
	Verification repository.AccountVerificationRepository
	ResetTokens  repository.PasswordResetTokenRepository
	OAuthTrial   OAuthTrialHook
	Lockout      *AccountLockoutTracker
	Hasher       *PasswordHasher
	Tokens       *JWTManager
	Encryption   *security.EncryptionService
	Digests      *security.LookupDigestService
}

// AuthSession contains issued session tokens for browser cookies.
// Implements DESIGN-006 AuthController.
type AuthSession struct {
	UserID                 uuid.UUID
	AccessToken            string
	RefreshToken           string
	AccessExpiresAt        time.Time
	RefreshExpiresAt       time.Time
	HasVerifiedLoginMethod bool
	Role                   string
}

// NewCoreAuthService creates the composed account-flow service.
// Implements DESIGN-006 AuthController.
func NewCoreAuthService(deps CoreAuthDependencies) *CoreAuthService {
	return &CoreAuthService{cfg: deps.Config, registration: deps.Registration, identities: deps.Identities, sessions: deps.Sessions, verification: deps.Verification, resetTokens: deps.ResetTokens, oauthTrial: deps.OAuthTrial, lockout: deps.Lockout, hasher: deps.Hasher, tokens: deps.Tokens, encryption: deps.Encryption, digests: deps.Digests, now: time.Now}
}

// Register creates an encrypted user with consent and returns authenticated cookies.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) Register(ctx context.Context, email string, password string, consent RegistrationConsent) (AuthSession, error) {
	normalizedEmail, err := security.NormalizeInput(security.InputFieldEmail, email)
	if err != nil {
		return AuthSession{}, err
	}
	passwordHash, passwordSalt, err := s.hasher.HashPassword(password)
	if err != nil {
		return AuthSession{}, err
	}
	encryptedEmail, err := s.encryption.EncryptPII(ctx, []byte(normalizedEmail.Value))
	if err != nil {
		return AuthSession{}, err
	}
	digest, err := s.digests.DigestForWrite(ctx, []byte(normalizedEmail.Value))
	if err != nil {
		return AuthSession{}, err
	}
	userID, err := s.registration.Register(ctx, repository.EncryptedAuthUser{
		Email:                 toRepositoryEncryptedField(encryptedEmail),
		NormalizedEmailDigest: toRepositoryLookupDigest(digest),
		Role:                  repository.UserRoleUser,
		PasswordHash:          &passwordHash,
		PasswordSalt:          &passwordSalt,
	}, consent)
	if err != nil {
		return AuthSession{}, err
	}
	return s.createSession(ctx, repository.EncryptedAuthUser{ID: userID, Role: repository.UserRoleUser}, uuid.New())
}

// Login validates credentials, lockout state, and returns authenticated cookies.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) Login(ctx context.Context, email string, password string) (AuthSession, error) {
	normalizedEmail, err := security.NormalizeInput(security.InputFieldEmail, email)
	if err != nil {
		return AuthSession{}, ErrInvalidCredentials
	}
	digest, err := s.digests.DigestForWrite(ctx, []byte(normalizedEmail.Value))
	if err != nil {
		return AuthSession{}, err
	}
	user, err := s.identities.GetUserByNormalizedEmailDigest(ctx, toRepositoryLookupDigest(digest))
	if err != nil {
		return AuthSession{}, ErrInvalidCredentials
	}
	state, err := s.lockout.Check(ctx, user.ID)
	if err != nil {
		return AuthSession{}, err
	}
	if state.Locked() {
		return AuthSession{}, &AccountLocked{RetryAfter: state.RetryAfter}
	}
	if user.PasswordHash == nil || user.PasswordSalt == nil || !s.hasher.VerifyPassword(password, *user.PasswordHash, *user.PasswordSalt) {
		state, err := s.lockout.RecordFailure(ctx, user.ID)
		if err != nil {
			return AuthSession{}, err
		}
		if state.Locked() {
			return AuthSession{}, &AccountLocked{RetryAfter: state.RetryAfter}
		}
		return AuthSession{}, ErrInvalidCredentials
	}
	if err := s.lockout.RecordSuccess(ctx, user.ID); err != nil {
		return AuthSession{}, err
	}
	return s.createSession(ctx, user, uuid.New())
}

// Refresh rotates refresh tokens and revokes the family on reuse.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) Refresh(ctx context.Context, refreshToken string) (AuthSession, error) {
	hash := HashRefreshToken(refreshToken)
	session, err := s.sessions.GetSessionByRefreshTokenHash(ctx, hash)
	if err != nil {
		return AuthSession{}, ErrSessionExpired
	}
	if session.RevokedAt != nil || !session.RefreshExpiresAt.After(s.now()) {
		_ = s.sessions.RevokeSessionFamily(ctx, session.RefreshFamilyID)
		return AuthSession{}, ErrTokenReuseDetected
	}
	user, err := s.identities.GetEncryptedUserByID(ctx, session.UserID)
	if err != nil {
		return AuthSession{}, err
	}
	if err := s.sessions.RevokeSession(ctx, session.ID); err != nil {
		return AuthSession{}, err
	}
	return s.createSession(ctx, user, session.RefreshFamilyID)
}

// Logout revokes the current refresh session if one is present.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	session, err := s.sessions.GetSessionByRefreshTokenHash(ctx, HashRefreshToken(refreshToken))
	if err != nil {
		return nil
	}
	return s.sessions.RevokeSession(ctx, session.ID)
}

// MarkEmailVerified updates the verified-login projection.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return s.verification.MarkEmailVerified(ctx, userID)
}

// RequestPasswordReset stores a hashed single-use reset token, returning generic success for missing users.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	normalizedEmail, err := security.NormalizeInput(security.InputFieldEmail, email)
	if err != nil {
		return "", nil
	}
	digest, err := s.digests.DigestForWrite(ctx, []byte(normalizedEmail.Value))
	if err != nil {
		return "", err
	}
	user, err := s.identities.GetUserByNormalizedEmailDigest(ctx, toRepositoryLookupDigest(digest))
	if err != nil {
		return "", nil
	}
	reset, err := s.tokens.CreateRefreshToken()
	if err != nil {
		return "", err
	}
	if err := s.resetTokens.CreatePasswordResetToken(ctx, repository.PasswordResetToken{TokenHash: reset.Hash, UserID: user.ID, ExpiresAt: s.now().Add(time.Hour)}); err != nil {
		return "", err
	}
	return reset.Plaintext, nil
}

// ConsumePasswordReset validates a reset token, updates password, and revokes sessions.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) ConsumePasswordReset(ctx context.Context, plainToken string, newPassword string) error {
	token, err := s.resetTokens.ConsumePasswordResetToken(ctx, HashRefreshToken(plainToken), s.now())
	if err != nil {
		return ErrPasswordResetInvalid
	}
	passwordHash, passwordSalt, err := s.hasher.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.verification.UpdatePassword(ctx, token.UserID, passwordHash, passwordSalt); err != nil {
		return err
	}
	return s.sessions.RevokeUserSessions(ctx, token.UserID)
}

// createSession creates access and refresh tokens plus persisted refresh metadata.
// Implements DESIGN-006 AuthController.
func (s *CoreAuthService) createSession(ctx context.Context, user repository.EncryptedAuthUser, familyID uuid.UUID) (AuthSession, error) {
	refresh, err := s.tokens.CreateRefreshToken()
	if err != nil {
		return AuthSession{}, err
	}
	now := s.now()
	accessExpiresAt := now.Add(s.cfg.AccessTokenTTL)
	refreshExpiresAt := now.Add(s.cfg.RefreshTokenTTL)
	sessionID, err := s.sessions.CreateSession(ctx, repository.UserSession{UserID: user.ID, RefreshTokenHash: refresh.Hash, RefreshFamilyID: familyID, AccessExpiresAt: accessExpiresAt, RefreshExpiresAt: refreshExpiresAt})
	if err != nil {
		return AuthSession{}, err
	}
	role := string(user.Role)
	if role == "" {
		role = string(repository.UserRoleUser)
	}
	access, err := s.tokens.CreateAccessToken(ctx, AccessTokenClaims{UserID: user.ID, Role: role, HasVerifiedLoginMethod: user.EmailVerified, SessionID: sessionID, RefreshFamilyID: familyID, ExpiresAt: accessExpiresAt})
	if err != nil {
		return AuthSession{}, err
	}
	return AuthSession{UserID: user.ID, AccessToken: access, RefreshToken: refresh.Plaintext, AccessExpiresAt: accessExpiresAt, RefreshExpiresAt: refreshExpiresAt, HasVerifiedLoginMethod: user.EmailVerified, Role: role}, nil
}

// toRepositoryEncryptedField maps security envelopes to repository envelopes.
// Implements DESIGN-013 EncryptionService.
func toRepositoryEncryptedField(field security.EncryptionEnvelope) repository.EncryptedField {
	return repository.EncryptedField{KeyVersion: field.KeyVersion, Nonce: field.Nonce, Ciphertext: field.Ciphertext}
}

// toRepositoryLookupDigest maps security lookup digests to repository lookup material.
// Implements DESIGN-013 EncryptionService.
func toRepositoryLookupDigest(digest security.LookupDigest) repository.LookupDigest {
	return repository.LookupDigest{KeyVersion: digest.KeyVersion, Value: digest.Value}
}

// ErrInvalidCredentials is the generic credential failure.
// Implements DESIGN-006 AuthController.
var ErrInvalidCredentials = errors.New("invalid_credentials")

// ErrSessionExpired means cookies must be cleared and login is required.
// Implements DESIGN-006 AuthController.
var ErrSessionExpired = errors.New("session_expired")

// ErrTokenReuseDetected means a refresh family has been revoked.
// Implements DESIGN-006 AuthController.
var ErrTokenReuseDetected = errors.New("token_reuse_detected")

// ErrPasswordResetInvalid means the reset token is missing, expired, or used.
// Implements DESIGN-006 AuthController.
var ErrPasswordResetInvalid = errors.New("password_reset_invalid")

// AccountLocked carries safe retry metadata for locked accounts.
// Implements DESIGN-006 AuthController.
type AccountLocked struct {
	RetryAfter time.Duration
}

// Error returns a stable safe lockout code.
// Implements DESIGN-006 AuthController.
func (e *AccountLocked) Error() string { return "account_locked" }

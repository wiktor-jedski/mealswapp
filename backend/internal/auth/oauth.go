package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// OAuthProfile contains normalized provider claims from the goth callback.
// Implements DESIGN-006 OAuthHandler.
type OAuthProfile struct {
	Provider       string
	ProviderUserID string
	Email          string
	DisplayName    string
	EmailVerified  bool
}

// OAuthTrialHook activates first-login trial behavior without owning entitlement state.
// Implements DESIGN-006 OAuthHandler and ARCH-007 trial boundary.
type OAuthTrialHook interface {
	ActivateFirstLoginTrial(context.Context, uuid.UUID) error
}

// OAuthIdentityStore persists encrypted OAuth identities and account rows.
// Implements DESIGN-006 OAuthHandler.
type OAuthIdentityStore interface {
	CreateUser(context.Context, repository.EncryptedAuthUser) (uuid.UUID, error)
	GetUserByNormalizedEmailDigest(context.Context, repository.LookupDigest) (repository.EncryptedAuthUser, error)
	GetEncryptedUserByID(context.Context, uuid.UUID) (repository.EncryptedAuthUser, error)
	UpsertOAuthIdentity(context.Context, repository.EncryptedOAuthIdentity) (uuid.UUID, error)
	GetOAuthIdentity(context.Context, string, repository.LookupDigest) (repository.EncryptedOAuthIdentity, error)
}

// OAuthResult reports the account session plus whether a first account row was created.
// Implements DESIGN-006 OAuthHandler.
type OAuthResult struct {
	Session     AuthSession
	CreatedUser bool
	Linked      bool
}

// CompleteOAuth creates or reuses an OAuth-linked account.
// Implements DESIGN-006 OAuthHandler.
func (s *CoreAuthService) CompleteOAuth(ctx context.Context, expectedProvider string, profile OAuthProfile) (OAuthResult, error) {
	normalized, err := normalizeOAuthProfile(expectedProvider, profile)
	if err != nil {
		return OAuthResult{}, err
	}
	store, err := s.oauthStore()
	if err != nil {
		return OAuthResult{}, err
	}
	providerDigest, err := s.digests.DigestForWrite(ctx, []byte(normalized.ProviderUserID))
	if err != nil {
		return OAuthResult{}, err
	}
	identity, err := store.GetOAuthIdentity(ctx, normalized.Provider, toRepositoryLookupDigest(providerDigest))
	if err == nil {
		user, err := store.GetEncryptedUserByID(ctx, identity.UserID)
		if err != nil {
			return OAuthResult{}, err
		}
		if normalized.EmailVerified && !user.EmailVerified {
			if err := s.verification.MarkEmailVerified(ctx, user.ID); err != nil {
				return OAuthResult{}, err
			}
			user.EmailVerified = true
		}
		session, err := s.createSession(ctx, user, uuid.New())
		if err != nil {
			return OAuthResult{}, err
		}
		return OAuthResult{Session: session}, nil
	}
	if !repository.IsKind(err, repository.ErrorKindNotFound) {
		return OAuthResult{}, err
	}
	emailDigest, encryptedEmail, err := s.emailLookupAndEnvelope(ctx, normalized.Email)
	if err != nil {
		return OAuthResult{}, err
	}
	if existing, err := store.GetUserByNormalizedEmailDigest(ctx, emailDigest); err == nil {
		return OAuthResult{}, &OAuthLinkRequired{UserID: existing.ID}
	} else if !repository.IsKind(err, repository.ErrorKindNotFound) {
		return OAuthResult{}, err
	}
	userID, err := store.CreateUser(ctx, repository.EncryptedAuthUser{
		Email:                 encryptedEmail,
		NormalizedEmailDigest: emailDigest,
		EmailVerified:         normalized.EmailVerified,
		Role:                  repository.UserRoleUser,
	})
	if err != nil {
		return OAuthResult{}, err
	}
	if err := s.linkOAuthIdentity(ctx, store, userID, normalized, toRepositoryLookupDigest(providerDigest)); err != nil {
		return OAuthResult{}, err
	}
	if s.oauthTrial != nil {
		if err := s.oauthTrial.ActivateFirstLoginTrial(ctx, userID); err != nil {
			return OAuthResult{}, err
		}
	}
	session, err := s.createSession(ctx, repository.EncryptedAuthUser{ID: userID, EmailVerified: normalized.EmailVerified, Role: repository.UserRoleUser}, uuid.New())
	if err != nil {
		return OAuthResult{}, err
	}
	return OAuthResult{Session: session, CreatedUser: true, Linked: true}, nil
}

// LinkOAuthIdentity explicitly links an OAuth profile to an authenticated user.
// Implements DESIGN-006 OAuthHandler.
func (s *CoreAuthService) LinkOAuthIdentity(ctx context.Context, userID uuid.UUID, expectedProvider string, profile OAuthProfile) error {
	if userID == uuid.Nil {
		return errors.New("user id is required")
	}
	normalized, err := normalizeOAuthProfile(expectedProvider, profile)
	if err != nil {
		return err
	}
	store, err := s.oauthStore()
	if err != nil {
		return err
	}
	digest, err := s.digests.DigestForWrite(ctx, []byte(normalized.ProviderUserID))
	if err != nil {
		return err
	}
	if err := s.linkOAuthIdentity(ctx, store, userID, normalized, toRepositoryLookupDigest(digest)); err != nil {
		return err
	}
	if normalized.EmailVerified {
		return s.verification.MarkEmailVerified(ctx, userID)
	}
	return nil
}

// linkOAuthIdentity encrypts provider-owned identity fields before persistence.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
func (s *CoreAuthService) linkOAuthIdentity(ctx context.Context, store OAuthIdentityStore, userID uuid.UUID, profile OAuthProfile, providerDigest repository.LookupDigest) error {
	encryptedProviderID, err := s.encryption.EncryptPII(ctx, []byte(profile.ProviderUserID))
	if err != nil {
		return err
	}
	encryptedEmail, err := s.encryption.EncryptPII(ctx, []byte(profile.Email))
	if err != nil {
		return err
	}
	_, err = store.UpsertOAuthIdentity(ctx, repository.EncryptedOAuthIdentity{
		UserID:               userID,
		Provider:             profile.Provider,
		ProviderUserID:       toRepositoryEncryptedField(encryptedProviderID),
		ProviderUserIDDigest: providerDigest,
		Email:                toRepositoryEncryptedField(encryptedEmail),
	})
	return err
}

// emailLookupAndEnvelope prepares encrypted account email persistence material.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
func (s *CoreAuthService) emailLookupAndEnvelope(ctx context.Context, email string) (repository.LookupDigest, repository.EncryptedField, error) {
	emailDigest, err := s.digests.DigestForWrite(ctx, []byte(email))
	if err != nil {
		return repository.LookupDigest{}, repository.EncryptedField{}, err
	}
	encryptedEmail, err := s.encryption.EncryptPII(ctx, []byte(email))
	if err != nil {
		return repository.LookupDigest{}, repository.EncryptedField{}, err
	}
	return toRepositoryLookupDigest(emailDigest), toRepositoryEncryptedField(encryptedEmail), nil
}

// oauthStore checks that the configured identity repository supports OAuth operations.
// Implements DESIGN-006 OAuthHandler.
func (s *CoreAuthService) oauthStore() (OAuthIdentityStore, error) {
	store, ok := s.identities.(OAuthIdentityStore)
	if !ok {
		return nil, errors.New("OAuth identity store is not configured")
	}
	return store, nil
}

// normalizeOAuthProfile validates provider claims at the trust boundary.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 InputNormalizer.
func normalizeOAuthProfile(expectedProvider string, profile OAuthProfile) (OAuthProfile, error) {
	expected, err := security.NormalizeInput(security.InputFieldOAuthProvider, expectedProvider)
	if err != nil {
		return OAuthProfile{}, err
	}
	provider, err := security.NormalizeInput(security.InputFieldOAuthProvider, profile.Provider)
	if err != nil {
		return OAuthProfile{}, err
	}
	if provider.Value != expected.Value {
		return OAuthProfile{}, ErrOAuthProviderMismatch
	}
	email, err := security.NormalizeInput(security.InputFieldEmail, profile.Email)
	if err != nil {
		return OAuthProfile{}, err
	}
	providerUserID := strings.TrimSpace(profile.ProviderUserID)
	if providerUserID == "" || strings.ContainsRune(providerUserID, '\x00') {
		return OAuthProfile{}, errors.New("OAuth provider user id is invalid")
	}
	return OAuthProfile{Provider: provider.Value, ProviderUserID: providerUserID, Email: email.Value, DisplayName: strings.TrimSpace(profile.DisplayName), EmailVerified: profile.EmailVerified}, nil
}

// OAuthLinkRequired means an OAuth email matched an account that must be linked explicitly.
// Implements DESIGN-006 OAuthHandler.
type OAuthLinkRequired struct {
	UserID uuid.UUID
}

// Error returns a stable safe account-linking code.
// Implements DESIGN-006 OAuthHandler.
func (e *OAuthLinkRequired) Error() string { return "oauth_link_required" }

// ErrOAuthProviderMismatch means the callback provider did not match the requested provider.
// Implements DESIGN-006 OAuthHandler.
var ErrOAuthProviderMismatch = errors.New("oauth_provider_mismatch")

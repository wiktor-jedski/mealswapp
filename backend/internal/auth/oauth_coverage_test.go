package auth

// Implements DESIGN-006 OAuthHandler trust-boundary verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

func TestNormalizeOAuthProfileRejectsInvalidClaims(t *testing.T) {
	cases := []struct {
		expected string
		profile  OAuthProfile
	}{
		{"bad", OAuthProfile{Provider: "google", ProviderUserID: "id", Email: "user@example.test"}},
		{"google", OAuthProfile{Provider: "bad", ProviderUserID: "id", Email: "user@example.test"}},
		{"google", OAuthProfile{Provider: "apple", ProviderUserID: "id", Email: "user@example.test"}},
		{"google", OAuthProfile{Provider: "google", ProviderUserID: "id", Email: "bad"}},
		{"google", OAuthProfile{Provider: "google", Email: "user@example.test"}},
		{"google", OAuthProfile{Provider: "google", ProviderUserID: "bad\x00id", Email: "user@example.test"}},
	}
	for _, tc := range cases {
		if _, err := normalizeOAuthProfile(tc.expected, tc.profile); err == nil {
			t.Fatalf("claims accepted: expected=%q profile=%+v", tc.expected, tc.profile)
		}
	}
	profile, err := normalizeOAuthProfile("google", OAuthProfile{Provider: " GOOGLE ", ProviderUserID: " id ", Email: " USER@example.test ", DisplayName: " Ada "})
	if err != nil || profile.Provider != "google" || profile.ProviderUserID != "id" || profile.Email != "USER@example.test" || profile.DisplayName != "Ada" {
		t.Fatalf("normalized profile=%+v err=%v", profile, err)
	}
}

func TestOAuthConfigurationAndLinkValidation(t *testing.T) {
	service := NewCoreAuthService(CoreAuthDependencies{Identities: &oauthUnsupportedIdentityRepository{}})
	if _, err := service.oauthStore(); err == nil {
		t.Fatal("oauthStore() accepted unsupported repository")
	}
	if err := service.LinkOAuthIdentity(context.Background(), uuid.Nil, "google", OAuthProfile{}); err == nil {
		t.Fatal("LinkOAuthIdentity() accepted nil user")
	}
	if (&OAuthLinkRequired{UserID: uuid.New()}).Error() != "oauth_link_required" {
		t.Fatal("OAuthLinkRequired.Error() changed")
	}
}

func TestOAuthDependencyFailures(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("dependency failed")
	profile := OAuthProfile{Provider: "google", ProviderUserID: "provider-user", Email: "user@example.test", EmailVerified: true}

	unsupported := NewCoreAuthService(CoreAuthDependencies{Identities: &oauthUnsupportedIdentityRepository{}})
	if _, err := unsupported.CompleteOAuth(ctx, "google", profile); err == nil {
		t.Fatal("CompleteOAuth() accepted unsupported identity repository")
	}
	if err := unsupported.LinkOAuthIdentity(ctx, uuid.New(), "bad", profile); err == nil {
		t.Fatal("LinkOAuthIdentity() accepted provider mismatch")
	}
	if err := unsupported.LinkOAuthIdentity(ctx, uuid.New(), "google", profile); err == nil {
		t.Fatal("LinkOAuthIdentity() accepted unsupported identity repository")
	}

	service, identities, sessions, verification, _, _, user := newCoreAuthFailureFixture(t)
	identities.oauthGetErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() identity lookup error = %v", err)
	}
	identities.oauthGetErr = nil
	digest, err := service.digests.DigestForWrite(ctx, []byte(profile.ProviderUserID))
	if err != nil {
		t.Fatal(err)
	}
	identities.oauth = map[string]repository.EncryptedOAuthIdentity{"google|" + digest.Value: {UserID: user.ID}}
	identities.byIDErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() user lookup error = %v", err)
	}
	identities.byIDErr = nil
	verification.err = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() verification error = %v", err)
	}
	verification.err = nil
	sessions.createErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() existing session error = %v", err)
	}

	service, identities, sessions, _, _, _, _ = newCoreAuthFailureFixture(t)
	identities.byDigest = map[repository.LookupDigest]repository.EncryptedAuthUser{}
	identities.byDigestErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() email lookup error = %v", err)
	}
	identities.byDigestErr = nil
	identities.createErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() create user error = %v", err)
	}
	identities.createErr = nil
	identities.upsertErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() link error = %v", err)
	}
	identities.upsertErr = nil
	identities.byDigest = map[repository.LookupDigest]repository.EncryptedAuthUser{}
	service.oauthTrial = &memoryTrialHook{err: wantErr}
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() trial error = %v", err)
	}
	service.oauthTrial = nil
	identities.byDigest = map[repository.LookupDigest]repository.EncryptedAuthUser{}
	sessions.createErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() new session error = %v", err)
	}

	service, identities, _, verification, _, _, user = newCoreAuthFailureFixture(t)
	identities.upsertErr = wantErr
	if err := service.LinkOAuthIdentity(ctx, user.ID, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("LinkOAuthIdentity() persistence error = %v", err)
	}
	identities.upsertErr = nil
	verification.err = wantErr
	if err := service.LinkOAuthIdentity(ctx, user.ID, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("LinkOAuthIdentity() verification error = %v", err)
	}
	profile.EmailVerified = false
	verification.err = nil
	if err := service.LinkOAuthIdentity(ctx, user.ID, "google", profile); err != nil {
		t.Fatalf("LinkOAuthIdentity() unverified error = %v", err)
	}

	service, identities, _, _, _, _, user = newCoreAuthFailureFixture(t)
	service.digests = security.NewLookupDigestService(authKeyLoader{lookupErr: wantErr})
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() provider digest error = %v", err)
	}
	if err := service.LinkOAuthIdentity(ctx, user.ID, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("LinkOAuthIdentity() digest error = %v", err)
	}
	service.digests = security.NewLookupDigestService(authKeyLoader{activeLookup: "lookup-v1", lookup: map[string][]byte{"lookup-v1": []byte("22222222222222222222222222222222")}})
	service.encryption = security.NewEncryptionService(authKeyLoader{encryptionErr: wantErr})
	identities.oauth = map[string]repository.EncryptedOAuthIdentity{}
	identities.byDigest = map[repository.LookupDigest]repository.EncryptedAuthUser{}
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() email encryption error = %v", err)
	}
	digest, err = service.digests.DigestForWrite(ctx, []byte(profile.ProviderUserID))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.linkOAuthIdentity(ctx, identities, user.ID, profile, toRepositoryLookupDigest(digest)); !errors.Is(err, wantErr) {
		t.Fatalf("linkOAuthIdentity() encryption error = %v", err)
	}
	if _, _, err := service.emailLookupAndEnvelope(ctx, profile.Email); !errors.Is(err, wantErr) {
		t.Fatalf("emailLookupAndEnvelope() encryption error = %v", err)
	}
	service.digests = security.NewLookupDigestService(authKeyLoader{lookupErr: wantErr})
	if _, _, err := service.emailLookupAndEnvelope(ctx, profile.Email); !errors.Is(err, wantErr) {
		t.Fatalf("emailLookupAndEnvelope() digest error = %v", err)
	}

	service, identities, sessions, _, _, _, user = newCoreAuthFailureFixture(t)
	identities.byDigest = map[repository.LookupDigest]repository.EncryptedAuthUser{}
	sessions.createErr = wantErr
	if _, err := service.CompleteOAuth(ctx, "google", profile); !errors.Is(err, wantErr) {
		t.Fatalf("CompleteOAuth() final session error = %v", err)
	}

	baseKeys := authKeyLoader{activeEncryption: "pii-v1", encryption: map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")}}
	service.encryption = security.NewEncryptionService(&secondEncryptionFailureKeyLoader{authKeyLoader: baseKeys, err: wantErr})
	digest = security.LookupDigest{KeyVersion: "lookup-v1", Value: "digest"}
	if err := service.linkOAuthIdentity(ctx, identities, user.ID, profile, toRepositoryLookupDigest(digest)); !errors.Is(err, wantErr) {
		t.Fatalf("linkOAuthIdentity() email encryption error = %v", err)
	}
}

type oauthUnsupportedIdentityRepository struct{}

type secondEncryptionFailureKeyLoader struct {
	authKeyLoader
	calls int
	err   error
}

func (l *secondEncryptionFailureKeyLoader) ActiveKey(ctx context.Context) (string, []byte, error) {
	l.calls++
	if l.calls == 2 {
		return "", nil, l.err
	}
	return l.authKeyLoader.ActiveKey(ctx)
}

func (*oauthUnsupportedIdentityRepository) GetUserByNormalizedEmailDigest(context.Context, repository.LookupDigest) (repository.EncryptedAuthUser, error) {
	return repository.EncryptedAuthUser{}, errors.New("unused")
}

func (*oauthUnsupportedIdentityRepository) GetEncryptedUserByID(context.Context, uuid.UUID) (repository.EncryptedAuthUser, error) {
	return repository.EncryptedAuthUser{}, errors.New("unused")
}

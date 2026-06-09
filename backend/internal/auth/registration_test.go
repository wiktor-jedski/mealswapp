package auth

// Implements DESIGN-015 ConsentManager verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type fakeRegistrationRepository struct {
	called  bool
	err     error
	userID  uuid.UUID
	privacy string
	terms   string
}

func (r *fakeRegistrationRepository) CreateUserWithConsent(_ context.Context, _ repository.EncryptedAuthUser, privacyVersion string, termsVersion string) (uuid.UUID, error) {
	r.called = true
	r.privacy = privacyVersion
	r.terms = termsVersion
	if r.err != nil {
		return uuid.Nil, r.err
	}
	if r.userID == uuid.Nil {
		r.userID = uuid.New()
	}
	return r.userID, nil
}

// TestRegistrationServiceConsentGate verifies DESIGN-015 ConsentManager registration gating.
func TestRegistrationServiceConsentGate(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRegistrationRepository{}
	service := NewRegistrationService(repo, "privacy-v1", "terms-v1")
	user := repository.EncryptedAuthUser{
		Email:                 repository.EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("nonce"), Ciphertext: []byte("ciphertext")},
		NormalizedEmailDigest: repository.LookupDigest{KeyVersion: "lookup-v1", Value: "digest"},
	}
	if _, err := service.Register(ctx, user, RegistrationConsent{}); err == nil || err.Error() != "consent_missing" || repo.called {
		t.Fatalf("missing consent err=%v called=%v", err, repo.called)
	}
	if _, err := service.Register(ctx, user, RegistrationConsent{PrivacyPolicyVersion: "old", TermsVersion: "terms-v1"}); err == nil || err.Error() != "consent_version_stale" || repo.called {
		t.Fatalf("stale consent err=%v called=%v", err, repo.called)
	}
	if _, err := service.Register(ctx, user, RegistrationConsent{PrivacyPolicyVersion: "bad version", TermsVersion: "terms-v1"}); err == nil || err.Error() != "consent_version_invalid" || repo.called {
		t.Fatalf("invalid consent err=%v called=%v", err, repo.called)
	}
	userID, err := service.Register(ctx, user, RegistrationConsent{PrivacyPolicyVersion: "privacy-v1", TermsVersion: "terms-v1"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if userID == uuid.Nil || !repo.called || repo.privacy != "privacy-v1" || repo.terms != "terms-v1" {
		t.Fatalf("registration result id=%s repo=%+v", userID, repo)
	}
	repo.called = false
	userID, err = service.Register(ctx, user, RegistrationConsent{PrivacyPolicyVersion: " privacy-v1 ", TermsVersion: " terms-v1 "})
	if err != nil || userID == uuid.Nil || !repo.called || repo.privacy != "privacy-v1" || repo.terms != "terms-v1" {
		t.Fatalf("normalized registration id=%s err=%v repo=%+v", userID, err, repo)
	}
}

// TestRegistrationServicePropagatesRepositoryFailure verifies DESIGN-015 ConsentManager rollback failures surface.
func TestRegistrationServicePropagatesRepositoryFailure(t *testing.T) {
	repo := &fakeRegistrationRepository{err: errors.New("duplicate")}
	service := NewRegistrationService(repo, "privacy-v1", "terms-v1")
	_, err := service.Register(context.Background(), repository.EncryptedAuthUser{}, RegistrationConsent{PrivacyPolicyVersion: "privacy-v1", TermsVersion: "terms-v1"})
	if err == nil {
		t.Fatal("Register() error = nil, want repository failure")
	}
}

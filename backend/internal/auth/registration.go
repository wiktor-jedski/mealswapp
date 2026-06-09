package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// RegistrationConsent captures explicitly accepted legal versions.
// Implements DESIGN-015 ConsentManager.
type RegistrationConsent struct {
	PrivacyPolicyVersion string
	TermsVersion         string
}

// RegistrationService gates account creation on current consent.
// Implements DESIGN-015 ConsentManager.
type RegistrationService struct {
	repo                  repository.RegistrationRepository
	currentPrivacyVersion string
	currentTermsVersion   string
}

// NewRegistrationService creates a consent-gated registration service.
// Implements DESIGN-015 ConsentManager.
func NewRegistrationService(repo repository.RegistrationRepository, currentPrivacyVersion string, currentTermsVersion string) *RegistrationService {
	return &RegistrationService{repo: repo, currentPrivacyVersion: currentPrivacyVersion, currentTermsVersion: currentTermsVersion}
}

// Register creates an account only when the current legal versions were accepted.
// Implements DESIGN-015 ConsentManager.
func (s *RegistrationService) Register(ctx context.Context, user repository.EncryptedAuthUser, consent RegistrationConsent) (uuid.UUID, error) {
	privacyVersion, err := security.NormalizeInput(security.InputFieldConsentVersion, consent.PrivacyPolicyVersion)
	if err != nil {
		if consent.PrivacyPolicyVersion == "" {
			return uuid.Nil, errors.New("consent_missing")
		}
		return uuid.Nil, errors.New("consent_version_invalid")
	}
	termsVersion, err := security.NormalizeInput(security.InputFieldConsentVersion, consent.TermsVersion)
	if err != nil {
		if consent.TermsVersion == "" {
			return uuid.Nil, errors.New("consent_missing")
		}
		return uuid.Nil, errors.New("consent_version_invalid")
	}
	if privacyVersion.Value != s.currentPrivacyVersion || termsVersion.Value != s.currentTermsVersion {
		return uuid.Nil, errors.New("consent_version_stale")
	}
	return s.repo.CreateUserWithConsent(ctx, user, privacyVersion.Value, termsVersion.Value)
}

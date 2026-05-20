package consent

import (
	"context"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

type RequiredVersions struct {
	PrivacyPolicy       string
	Terms               string
	NutritionDisclaimer string
}

type RegistrationConsent struct {
	UserID                     uuid.UUID
	AcceptPrivacyPolicy        bool
	AcceptTerms                bool
	AcceptNutritionDisclaimer  bool
	PrivacyPolicyVersion       string
	TermsVersion               string
	NutritionDisclaimerVersion string
	IPAddress                  string
	UserAgent                  string
	AcceptedAt                 time.Time
}

type Manager struct {
	repository repositories.ConsentRepository
	required   RequiredVersions
	now        func() time.Time
}

func NewManager(repository repositories.ConsentRepository, required RequiredVersions) Manager {
	return Manager{
		repository: repository,
		required:   required,
		now:        time.Now,
	}
}

func (manager Manager) RecordRegistrationConsent(ctx context.Context, input RegistrationConsent) (uuid.UUID, error) {
	if err := manager.RequireRegistrationConsent(input); err != nil {
		return uuid.Nil, err
	}

	acceptedAt := input.AcceptedAt
	if acceptedAt.IsZero() {
		acceptedAt = manager.now().UTC()
	}

	return manager.repository.Record(ctx, repositories.ConsentRecordEntity{
		UserID:                     input.UserID,
		PrivacyPolicyVersion:       input.PrivacyPolicyVersion,
		TermsVersion:               input.TermsVersion,
		NutritionDisclaimerVersion: input.NutritionDisclaimerVersion,
		AcceptedAt:                 acceptedAt,
		IPAddress:                  input.IPAddress,
		UserAgent:                  input.UserAgent,
	})
}

func (manager Manager) RequireRegistrationConsent(input RegistrationConsent) error {
	missing := manager.missingConsentFields(input)
	if len(missing) > 0 {
		return apperrors.AppError{
			Category: apperrors.CategoryValidation,
			Code:     "consent_missing",
			Message:  "Required consent is missing",
			Status:   400,
			Fields:   map[string]any{"missing": missing},
		}
	}

	return nil
}

func (manager Manager) HasRequiredConsent(ctx context.Context, userID uuid.UUID) (bool, error) {
	return manager.repository.HasRequiredConsent(ctx, userID, manager.required.PrivacyPolicy, manager.required.Terms, manager.required.NutritionDisclaimer)
}

func (manager Manager) missingConsentFields(input RegistrationConsent) []string {
	var missing []string
	if !input.AcceptPrivacyPolicy || input.PrivacyPolicyVersion != manager.required.PrivacyPolicy {
		missing = append(missing, "privacy_policy")
	}
	if !input.AcceptTerms || input.TermsVersion != manager.required.Terms {
		missing = append(missing, "terms")
	}
	if !input.AcceptNutritionDisclaimer || input.NutritionDisclaimerVersion != manager.required.NutritionDisclaimer {
		missing = append(missing, "nutrition_disclaimer")
	}
	return missing
}

package consent

import (
	"context"
	"testing"
	"time"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestRequireRegistrationConsentBlocksMissingConsent(t *testing.T) {
	manager := NewManager(&fakeConsentRepository{}, testVersions())

	err := manager.RequireRegistrationConsent(RegistrationConsent{
		AcceptPrivacyPolicy:        true,
		PrivacyPolicyVersion:       "privacy-v1",
		AcceptTerms:                false,
		TermsVersion:               "terms-v1",
		AcceptNutritionDisclaimer:  true,
		NutritionDisclaimerVersion: "nutrition-v1",
	})

	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("expected app error, got %v", err)
	}
	if appErr.Code != "consent_missing" || appErr.Status != 400 {
		t.Fatalf("unexpected consent error: %#v", appErr)
	}
}

func TestRecordRegistrationConsentPersistsMetadata(t *testing.T) {
	repo := &fakeConsentRepository{}
	manager := NewManager(repo, testVersions())
	manager.now = func() time.Time {
		return time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	}
	userID := uuid.New()

	id, err := manager.RecordRegistrationConsent(context.Background(), RegistrationConsent{
		UserID:                     userID,
		AcceptPrivacyPolicy:        true,
		AcceptTerms:                true,
		AcceptNutritionDisclaimer:  true,
		PrivacyPolicyVersion:       "privacy-v1",
		TermsVersion:               "terms-v1",
		NutritionDisclaimerVersion: "nutrition-v1",
		IPAddress:                  "203.0.113.10",
		UserAgent:                  "test-agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	if id == uuid.Nil {
		t.Fatal("expected consent id")
	}
	if repo.record.UserID != userID || repo.record.IPAddress != "203.0.113.10" || repo.record.UserAgent != "test-agent" {
		t.Fatalf("expected consent metadata persisted, got %#v", repo.record)
	}
	if repo.record.AcceptedAt.IsZero() {
		t.Fatal("expected accepted timestamp")
	}
}

func TestHasRequiredConsentChecksCurrentVersions(t *testing.T) {
	userID := uuid.New()
	repo := &fakeConsentRepository{hasConsent: true}
	manager := NewManager(repo, testVersions())

	hasConsent, err := manager.HasRequiredConsent(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}

	if !hasConsent {
		t.Fatal("expected required consent")
	}
	if repo.checkedUserID != userID || repo.checkedPrivacy != "privacy-v1" || repo.checkedTerms != "terms-v1" || repo.checkedNutrition != "nutrition-v1" {
		t.Fatalf("expected current versions checked, got %#v", repo)
	}
}

func testVersions() RequiredVersions {
	return RequiredVersions{
		PrivacyPolicy:       "privacy-v1",
		Terms:               "terms-v1",
		NutritionDisclaimer: "nutrition-v1",
	}
}

type fakeConsentRepository struct {
	record           repositories.ConsentRecordEntity
	hasConsent       bool
	checkedUserID    uuid.UUID
	checkedPrivacy   string
	checkedTerms     string
	checkedNutrition string
}

func (repo *fakeConsentRepository) Record(ctx context.Context, record repositories.ConsentRecordEntity) (uuid.UUID, error) {
	repo.record = record
	return uuid.New(), nil
}

func (repo *fakeConsentRepository) HasRequiredConsent(ctx context.Context, userID uuid.UUID, privacyVersion string, termsVersion string, nutritionDisclaimerVersion string) (bool, error) {
	repo.checkedUserID = userID
	repo.checkedPrivacy = privacyVersion
	repo.checkedTerms = termsVersion
	repo.checkedNutrition = nutritionDisclaimerVersion
	return repo.hasConsent, nil
}

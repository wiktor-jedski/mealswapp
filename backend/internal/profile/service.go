package profile

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Service owns profile reads and preference updates.
// Implements DESIGN-008 PreferenceManager.
type Service struct {
	repo       repository.EncryptedUserProfileRepository
	encryption *security.EncryptionService
}

// NewService creates profile preference behavior.
// Implements DESIGN-008 PreferenceManager.
func NewService(repo repository.EncryptedUserProfileRepository, encryption *security.EncryptionService) *Service {
	return &Service{repo: repo, encryption: encryption}
}

// UserProfile is the decrypted service-boundary profile.
// Implements DESIGN-008 PreferenceManager.
type UserProfile struct {
	UserID          uuid.UUID
	DisplayName     string
	UnitSystem      repository.UnitSystem
	ThemePreference string
}

// UpdateRequest carries mutable profile preference fields.
// Implements DESIGN-008 PreferenceManager.
type UpdateRequest struct {
	DisplayName     *string
	UnitSystem      repository.UnitSystem
	ThemePreference string
}

// UpdateResult returns saved preferences and recalculation guidance.
// Implements DESIGN-008 PreferenceManager.
type UpdateResult struct {
	Profile                   UserProfile
	RequiresUnitRecalculation bool
}

// GetProfile returns a default profile after first authentication.
// Implements DESIGN-008 PreferenceManager.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	profile, err := s.repo.GetOrCreateEncryptedProfile(ctx, userID)
	if err != nil {
		return UserProfile{}, err
	}
	return s.decryptProfile(ctx, profile)
}

// UpdatePreferences stores encrypted display name and non-PII preferences.
// Implements DESIGN-008 PreferenceManager.
func (s *Service) UpdatePreferences(ctx context.Context, userID uuid.UUID, req UpdateRequest) (UpdateResult, error) {
	if userID == uuid.Nil {
		return UpdateResult{}, errors.New("user id is required")
	}
	if req.UnitSystem != repository.UnitSystemMetric && req.UnitSystem != repository.UnitSystemImperial {
		return UpdateResult{}, errors.New("unit system is invalid")
	}
	if req.ThemePreference != "system" && req.ThemePreference != "light" && req.ThemePreference != "dark" {
		return UpdateResult{}, errors.New("theme preference is invalid")
	}
	previous, err := s.repo.GetOrCreateEncryptedProfile(ctx, userID)
	if err != nil {
		return UpdateResult{}, err
	}
	var displayName *repository.EncryptedField
	if req.DisplayName != nil {
		normalized, err := security.NormalizeInput(security.InputFieldDisplayName, *req.DisplayName)
		if err != nil {
			return UpdateResult{}, err
		}
		if normalized.Value != "" {
			encrypted, err := s.encryption.EncryptPII(ctx, []byte(normalized.Value))
			if err != nil {
				return UpdateResult{}, err
			}
			field := repository.EncryptedField{KeyVersion: encrypted.KeyVersion, Nonce: encrypted.Nonce, Ciphertext: encrypted.Ciphertext}
			displayName = &field
		}
	} else {
		displayName = previous.DisplayName
	}
	updated, err := s.repo.UpdateEncryptedProfile(ctx, repository.EncryptedUserProfile{UserID: userID, DisplayName: displayName, UnitSystem: req.UnitSystem, ThemePreference: req.ThemePreference})
	if err != nil {
		return UpdateResult{}, err
	}
	decrypted, err := s.decryptProfile(ctx, updated)
	if err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{Profile: decrypted, RequiresUnitRecalculation: previous.UnitSystem != updated.UnitSystem}, nil
}

// decryptProfile decrypts display-name PII only at the service boundary.
// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService.
func (s *Service) decryptProfile(ctx context.Context, stored repository.EncryptedUserProfile) (UserProfile, error) {
	displayName := ""
	if stored.DisplayName != nil {
		plain, err := s.encryption.DecryptPII(ctx, security.EncryptionEnvelope{KeyVersion: stored.DisplayName.KeyVersion, Nonce: stored.DisplayName.Nonce, Ciphertext: stored.DisplayName.Ciphertext})
		if err != nil {
			return UserProfile{}, err
		}
		displayName = string(plain)
	}
	return UserProfile{UserID: stored.UserID, DisplayName: displayName, UnitSystem: stored.UnitSystem, ThemePreference: stored.ThemePreference}, nil
}

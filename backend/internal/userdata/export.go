package userdata

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// ExportIdentityRepository loads encrypted account identity for export.
// Implements DESIGN-008 DataExporter.
type ExportIdentityRepository interface {
	GetEncryptedUserByID(context.Context, uuid.UUID) (repository.EncryptedAuthUser, error)
}

// ExportService builds account export payloads.
// Implements DESIGN-008 DataExporter.
type ExportService struct {
	identity   ExportIdentityRepository
	profiles   repository.EncryptedUserProfileRepository
	saved      repository.SavedItemRepository
	history    repository.EncryptedSearchHistoryRepository
	consent    repository.ConsentRepository
	encryption *security.EncryptionService
	diets      repository.DailyDietRepository
}

// NewExportService creates account export behavior.
// Implements DESIGN-008 DataExporter.
func NewExportService(identity ExportIdentityRepository, profiles repository.EncryptedUserProfileRepository, saved repository.SavedItemRepository, history repository.EncryptedSearchHistoryRepository, consent repository.ConsentRepository, encryption *security.EncryptionService, diets ...repository.DailyDietRepository) *ExportService {
	var dietRepository repository.DailyDietRepository
	if len(diets) > 0 {
		dietRepository = diets[0]
	}
	return &ExportService{identity: identity, profiles: profiles, saved: saved, history: history, consent: consent, encryption: encryption, diets: dietRepository}
}

// ExportPayload is a serialized account export response.
// Implements DESIGN-008 DataExporter.
type ExportPayload struct {
	Format      string
	ContentType string
	Filename    string
	Body        []byte
}

// ExportBundle contains decrypted account export data at the export boundary.
// Implements DESIGN-008 DataExporter.
type ExportBundle struct {
	User        ExportUser             `json:"user"`
	Consent     []ExportConsent        `json:"consent"`
	SavedItems  []repository.SavedItem `json:"savedItems"`
	SavedDiets  []repository.SavedDiet `json:"savedDiets"`
	History     []SearchHistoryEntry   `json:"history"`
	CustomItems []ExportCustomItem     `json:"customItems"`
}

// ExportUser contains decrypted user/profile fields for export.
// Implements DESIGN-008 DataExporter.
type ExportUser struct {
	UserID          uuid.UUID             `json:"userId"`
	Email           string                `json:"email"`
	Role            repository.UserRole   `json:"role"`
	DisplayName     string                `json:"displayName"`
	UnitSystem      repository.UnitSystem `json:"unitSystem"`
	ThemePreference string                `json:"themePreference"`
}

// ExportConsent contains accepted legal versions.
// Implements DESIGN-015 ConsentManager.
type ExportConsent struct {
	PrivacyPolicyVersion string `json:"privacyPolicyVersion"`
	TermsVersion         string `json:"termsVersion"`
}

// ExportCustomItem reserves a typed user-owned custom item export contract.
// Implements DESIGN-008 DataExporter.
type ExportCustomItem struct {
	ID string `json:"id"`
}

// BuildExport serializes account data as JSON or CSV.
// Implements DESIGN-008 DataExporter.
func (s *ExportService) BuildExport(ctx context.Context, userID uuid.UUID, format string) (ExportPayload, error) {
	normalized, err := security.NormalizeInput(security.InputFieldExportFormat, format)
	if err != nil {
		return ExportPayload{}, err
	}
	bundle, err := s.buildBundle(ctx, userID)
	if err != nil {
		return ExportPayload{}, err
	}
	if normalized.Value == "json" {
		body, err := json.Marshal(bundle)
		if err != nil {
			return ExportPayload{}, err
		}
		return ExportPayload{Format: "json", ContentType: "application/json", Filename: "mealswapp-export.json", Body: body}, nil
	}
	body := encodeCSV(bundle)
	return ExportPayload{Format: "csv", ContentType: "text/csv", Filename: "mealswapp-export.csv", Body: body}, nil
}

// buildBundle gathers and decrypts account export data.
// Implements DESIGN-008 DataExporter and DESIGN-013 EncryptionService.
func (s *ExportService) buildBundle(ctx context.Context, userID uuid.UUID) (ExportBundle, error) {
	user, err := s.identity.GetEncryptedUserByID(ctx, userID)
	if err != nil {
		return ExportBundle{}, err
	}
	email, err := decryptField(ctx, s.encryption, user.Email)
	if err != nil {
		return ExportBundle{}, err
	}
	profile, err := s.profiles.GetOrCreateEncryptedProfile(ctx, userID)
	if err != nil {
		return ExportBundle{}, err
	}
	displayName := ""
	if profile.DisplayName != nil {
		displayName, err = decryptField(ctx, s.encryption, *profile.DisplayName)
		if err != nil {
			return ExportBundle{}, err
		}
	}
	saved, err := s.saved.ListItems(ctx, userID, nil)
	if err != nil {
		return ExportBundle{}, err
	}
	encryptedHistory, err := s.history.ListEncryptedHistory(ctx, userID, 100)
	if err != nil {
		return ExportBundle{}, err
	}
	history := make([]SearchHistoryEntry, 0, len(encryptedHistory))
	for _, entry := range encryptedHistory {
		query, err := decryptField(ctx, s.encryption, entry.Query)
		if err != nil {
			return ExportBundle{}, err
		}
		history = append(history, SearchHistoryEntry{ID: entry.ID, Query: query, Mode: entry.Mode, FiltersHash: entry.FiltersHash})
	}
	consentRecords, err := s.consent.ListConsent(ctx, userID)
	if err != nil {
		return ExportBundle{}, err
	}
	consent := make([]ExportConsent, 0, len(consentRecords))
	for _, record := range consentRecords {
		consent = append(consent, ExportConsent{PrivacyPolicyVersion: record.PrivacyPolicyVersion, TermsVersion: record.TermsVersion})
	}
	diets := []repository.SavedDiet{}
	if s.diets != nil {
		diets, err = s.diets.List(ctx, userID)
		if err != nil {
			return ExportBundle{}, err
		}
	}
	role := user.Role
	if role == "" {
		role = repository.UserRoleUser
	}
	return ExportBundle{
		User:    ExportUser{UserID: userID, Email: email, Role: role, DisplayName: displayName, UnitSystem: profile.UnitSystem, ThemePreference: profile.ThemePreference},
		Consent: consent, SavedItems: saved, SavedDiets: diets, History: history, CustomItems: []ExportCustomItem{},
	}, nil
}

// decryptField decrypts one repository encrypted field.
// Implements DESIGN-013 EncryptionService.
func decryptField(ctx context.Context, encryption *security.EncryptionService, field repository.EncryptedField) (string, error) {
	plain, err := encryption.DecryptPII(ctx, security.EncryptionEnvelope{KeyVersion: field.KeyVersion, Nonce: field.Nonce, Ciphertext: field.Ciphertext})
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// encodeCSV writes separate CSV sections into one downloadable file.
// Implements DESIGN-008 DataExporter.
func encodeCSV(bundle ExportBundle) []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	rows := [][]string{
		{"section", "field", "value"},
		{"user", "userId", bundle.User.UserID.String()},
		{"user", "email", bundle.User.Email},
		{"user", "displayName", bundle.User.DisplayName},
		{"user", "unitSystem", string(bundle.User.UnitSystem)},
		{"user", "themePreference", bundle.User.ThemePreference},
	}
	for _, item := range bundle.SavedItems {
		rows = append(rows, []string{"savedItems", string(item.Kind), item.ItemID.String()})
	}
	for _, diet := range bundle.SavedDiets {
		rows = append(rows, []string{"savedDiets", diet.Name, diet.ID.String()})
		for _, entry := range diet.Entries {
			rows = append(rows, []string{"savedDietMeals", diet.ID.String(), entry.MealID.String()})
		}
	}
	for _, entry := range bundle.History {
		rows = append(rows, []string{"history", entry.Mode, entry.Query})
	}
	for _, record := range bundle.Consent {
		rows = append(rows, []string{"consent", record.PrivacyPolicyVersion, record.TermsVersion})
	}
	// placeholder for user-owned custom items in the future
	rows = append(rows, []string{"customItems", "count", "0"})
	_ = writer.WriteAll(rows)
	return buf.Bytes()
}

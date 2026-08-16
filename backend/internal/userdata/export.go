package userdata

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
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
	identity    ExportIdentityRepository
	profiles    repository.EncryptedUserProfileRepository
	saved       repository.SavedItemRepository
	history     repository.EncryptedSearchHistoryRepository
	consent     repository.ConsentRepository
	encryption  *security.EncryptionService
	diets       repository.DailyDietRepository
	customItems CustomItemExporter
}

// CustomItemExporter loads API-safe private items for one authenticated owner.
// Implements DESIGN-008 DataExporter owner-scoped custom-item export.
type CustomItemExporter interface {
	List(context.Context, uuid.UUID) ([]customitem.Item, error)
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

// WithCustomItems attaches the owner-scoped custom-item export source.
// Implements DESIGN-008 DataExporter owner-scoped custom-item export.
func (s *ExportService) WithCustomItems(items CustomItemExporter) *ExportService {
	s.customItems = items
	return s
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
	User        ExportUser            `json:"user"`
	Consent     []ExportConsent       `json:"consent"`
	SavedItems  []ExportSavedItem     `json:"savedItems"`
	SavedDiets  []ExportSavedDiet     `json:"savedDiets"`
	History     []ExportSearchHistory `json:"history"`
	CustomItems []ExportCustomItem    `json:"customItems"`
}

// ExportSavedItem is a portable saved-data projection without repository ownership.
// Implements DESIGN-008 DataExporter.
type ExportSavedItem struct {
	ID        uuid.UUID `json:"id"`
	ItemID    uuid.UUID `json:"itemId"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}

// ExportSavedDiet is an API-safe saved-diet projection without owner identity.
// Implements DESIGN-008 DataExporter.
type ExportSavedDiet struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Entries   []ExportSavedDietEntry `json:"entries"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// ExportSavedDietEntry is an API-safe saved-diet entry projection.
// Implements DESIGN-008 DataExporter.
type ExportSavedDietEntry struct {
	ID             uuid.UUID `json:"id"`
	FoodObjectID   uuid.UUID `json:"foodObjectId"`
	FoodObjectType string    `json:"foodObjectType"`
	Quantity       float64   `json:"quantity"`
	Unit           string    `json:"unit"`
	Position       int       `json:"position"`
}

// ExportSearchHistory is a portable decrypted history projection without repository ownership.
// Implements DESIGN-008 DataExporter.
type ExportSearchHistory struct {
	ID          uuid.UUID `json:"id"`
	Query       string    `json:"query"`
	Mode        string    `json:"mode"`
	FiltersHash string    `json:"filtersHash"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ExportCustomItem is a portable private-item projection without repository ownership.
// Implements DESIGN-008 DataExporter.
type ExportCustomItem struct {
	ID                              uuid.UUID                     `json:"id"`
	Name                            string                        `json:"name"`
	PhysicalState                   string                        `json:"physicalState"`
	PrepTimeMinutes                 int                           `json:"prepTimeMinutes"`
	AverageUnitWeightGrams          float64                       `json:"averageUnitWeightGrams,omitempty"`
	AverageServingVolumeMilliliters float64                       `json:"averageServingVolumeMilliliters,omitempty"`
	DensityGramsPerMilliliter       float64                       `json:"densityGramsPerMilliliter,omitempty"`
	DensitySourceProvider           string                        `json:"densitySourceProvider,omitempty"`
	DensitySourceFoodID             string                        `json:"densitySourceFoodId,omitempty"`
	DensitySourceKind               string                        `json:"densitySourceKind,omitempty"`
	MacrosPer100                    ExportMacros                  `json:"macrosPer100"`
	Micros                          map[string]float64            `json:"micros"`
	FoodCategories                  []ExportClassificationSummary `json:"foodCategories"`
	CulinaryRoles                   []ExportClassificationSummary `json:"culinaryRoles"`
	ImageURL                        string                        `json:"imageUrl,omitempty"`
}

// ExportMacros contains portable macronutrients per 100 grams or milliliters.
// Implements DESIGN-008 DataExporter.
type ExportMacros struct {
	Protein       float64 `json:"protein"`
	Carbohydrates float64 `json:"carbohydrates"`
	Fat           float64 `json:"fat"`
}

// ExportClassificationSummary contains portable classification content without hierarchy state.
// Implements DESIGN-008 DataExporter.
type ExportClassificationSummary struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Kind string    `json:"kind"`
}

// ExportUser contains decrypted user/profile fields for export.
// Implements DESIGN-008 DataExporter.
type ExportUser struct {
	UserID          uuid.UUID `json:"userId"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	DisplayName     string    `json:"displayName"`
	UnitSystem      string    `json:"unitSystem"`
	ThemePreference string    `json:"themePreference"`
}

// ExportConsent contains accepted legal versions.
// Implements DESIGN-015 ConsentManager.
type ExportConsent struct {
	PrivacyPolicyVersion string `json:"privacyPolicyVersion"`
	TermsVersion         string `json:"termsVersion"`
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
	body, err := encodeCSV(bundle)
	if err != nil {
		return ExportPayload{}, err
	}
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
	savedRecords, err := s.saved.ListItems(ctx, userID, nil)
	if err != nil {
		return ExportBundle{}, err
	}
	saved := make([]ExportSavedItem, 0, len(savedRecords))
	for _, item := range savedRecords {
		saved = append(saved, ExportSavedItem{
			ID: item.ID, ItemID: item.ItemID, Kind: string(item.Kind), CreatedAt: item.CreatedAt,
		})
	}
	encryptedHistory, err := s.history.ListEncryptedHistory(ctx, userID, 100)
	if err != nil {
		return ExportBundle{}, err
	}
	history := make([]ExportSearchHistory, 0, len(encryptedHistory))
	for _, entry := range encryptedHistory {
		query, err := decryptField(ctx, s.encryption, entry.Query)
		if err != nil {
			return ExportBundle{}, err
		}
		history = append(history, ExportSearchHistory{
			ID: entry.ID, Query: query, Mode: entry.Mode, FiltersHash: entry.FiltersHash, CreatedAt: entry.CreatedAt,
		})
	}
	consentRecords, err := s.consent.ListConsent(ctx, userID)
	if err != nil {
		return ExportBundle{}, err
	}
	consent := make([]ExportConsent, 0, len(consentRecords))
	for _, record := range consentRecords {
		consent = append(consent, ExportConsent{PrivacyPolicyVersion: record.PrivacyPolicyVersion, TermsVersion: record.TermsVersion})
	}
	diets := []ExportSavedDiet{}
	if s.diets != nil {
		savedDiets, listErr := s.diets.List(ctx, userID)
		err = listErr
		if err != nil {
			return ExportBundle{}, err
		}
		for _, diet := range savedDiets {
			entries := make([]ExportSavedDietEntry, 0, len(diet.Entries))
			for _, entry := range diet.Entries {
				objectID, objectType := entry.FoodObjectID, entry.FoodObjectType
				if objectID == uuid.Nil {
					objectID, objectType = entry.MealID, repository.FoodObjectTypeMeal
				}
				entries = append(entries, ExportSavedDietEntry{
					ID: entry.ID, FoodObjectID: objectID, FoodObjectType: string(objectType),
					Quantity: entry.Quantity, Unit: entry.Unit, Position: entry.Position,
				})
			}
			diets = append(diets, ExportSavedDiet{
				ID: diet.ID, Name: diet.Name, Entries: entries,
				CreatedAt: diet.CreatedAt, UpdatedAt: diet.UpdatedAt,
			})
		}
	}
	customItems := []ExportCustomItem{}
	if s.customItems != nil {
		items, listErr := s.customItems.List(ctx, userID)
		err = listErr
		if err != nil {
			return ExportBundle{}, err
		}
		for _, item := range items {
			customItems = append(customItems, exportCustomItem(item))
		}
	}
	role := user.Role
	if role == "" {
		role = repository.UserRoleUser
	}
	return ExportBundle{
		User:    ExportUser{UserID: userID, Email: email, Role: string(role), DisplayName: displayName, UnitSystem: string(profile.UnitSystem), ThemePreference: profile.ThemePreference},
		Consent: consent, SavedItems: saved, SavedDiets: diets, History: history, CustomItems: customItems,
	}, nil
}

// exportCustomItem copies an API-safe private item into the export-only boundary.
// Implements DESIGN-008 DataExporter.
func exportCustomItem(item customitem.Item) ExportCustomItem {
	classifications := func(items []customitem.ClassificationSummary) []ExportClassificationSummary {
		result := make([]ExportClassificationSummary, 0, len(items))
		for _, item := range items {
			result = append(result, ExportClassificationSummary{ID: item.ID, Name: item.Name, Kind: string(item.Kind)})
		}
		return result
	}
	micros := make(map[string]float64, len(item.Micros))
	for key, value := range item.Micros {
		micros[key] = value
	}
	return ExportCustomItem{
		ID: item.ID, Name: item.Name, PhysicalState: string(item.PhysicalState), PrepTimeMinutes: item.PrepTimeMinutes,
		AverageUnitWeightGrams: item.AverageUnitWeightGrams, AverageServingVolumeMilliliters: item.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: item.DensityGramsPerMilliliter, DensitySourceProvider: item.DensitySourceProvider,
		DensitySourceFoodID: item.DensitySourceFoodID, DensitySourceKind: item.DensitySourceKind,
		MacrosPer100: ExportMacros{Protein: item.MacrosPer100.Protein, Carbohydrates: item.MacrosPer100.Carbohydrates, Fat: item.MacrosPer100.Fat},
		Micros:       micros, FoodCategories: classifications(item.FoodCategories), CulinaryRoles: classifications(item.CulinaryRoles),
		ImageURL: item.ImageURL,
	}
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
func encodeCSV(bundle ExportBundle) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	rows := [][]string{
		{"section", "field", "value"},
		{"user", "userId", bundle.User.UserID.String()},
		{"user", "email", bundle.User.Email},
		{"user", "role", bundle.User.Role},
		{"user", "displayName", bundle.User.DisplayName},
		{"user", "unitSystem", bundle.User.UnitSystem},
		{"user", "themePreference", bundle.User.ThemePreference},
	}
	for _, item := range bundle.SavedItems {
		row, err := exportCSVRow("savedItems", item.ID.String(), item)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for _, diet := range bundle.SavedDiets {
		row, err := exportCSVRow("savedDiets", diet.ID.String(), diet)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for _, entry := range bundle.History {
		row, err := exportCSVRow("history", entry.ID.String(), entry)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for _, record := range bundle.Consent {
		row, err := exportCSVRow("consent", record.PrivacyPolicyVersion, record)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for _, item := range bundle.CustomItems {
		row, err := exportCSVRow("customItems", item.ID.String(), item)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for _, section := range []struct {
		name  string
		count int
	}{
		{name: "savedItems", count: len(bundle.SavedItems)},
		{name: "savedDiets", count: len(bundle.SavedDiets)},
		{name: "history", count: len(bundle.History)},
		{name: "consent", count: len(bundle.Consent)},
		{name: "customItems", count: len(bundle.CustomItems)},
	} {
		if section.count == 0 {
			rows = append(rows, []string{section.name, "count", "0"})
		}
	}
	writer.WriteAll(rows)
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// exportCSVRow serializes one owner-free projection as an escaped CSV JSON cell.
// Implements DESIGN-008 DataExporter.
func exportCSVRow(section, field string, value any) ([]string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return []string{section, field, string(payload)}, nil
}

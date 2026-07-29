package userdata

// Implements DESIGN-008 DataExporter verification.

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type memoryExportRepository struct {
	user    repository.EncryptedAuthUser
	profile repository.EncryptedUserProfile
	saved   []repository.SavedItem
	history []repository.EncryptedSearchHistoryEntry
	consent []repository.ConsentRecord
	errAt   string
}

func TestEncodeCSVCustomItemsEmptyAndNonemptyExact(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	base := ExportBundle{User: ExportUser{UserID: userID, Email: "a@example.test", Role: "user", UnitSystem: "metric", ThemePreference: "system"}}
	tests := []struct {
		name  string
		items []ExportCustomItem
		rows  [][]string
	}{
		{name: "empty", rows: [][]string{{"customItems", "count", "0"}}},
		{name: "nonempty", items: []ExportCustomItem{{ID: itemID, Name: "Tofu", PhysicalState: "solid", MacrosPer100: ExportMacros{Protein: 1}, Micros: map[string]float64{}, FoodCategories: []ExportClassificationSummary{{ID: uuid.New(), Name: "Child", Kind: "food_category"}}, CulinaryRoles: []ExportClassificationSummary{}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle := base
			bundle.CustomItems = test.items
			actual, err := encodeCSV(bundle)
			if err != nil {
				t.Fatal(err)
			}
			rows := [][]string{
				{"section", "field", "value"},
				{"user", "userId", userID.String()},
				{"user", "email", "a@example.test"},
				{"user", "role", "user"},
				{"user", "displayName", ""},
				{"user", "unitSystem", "metric"},
				{"user", "themePreference", "system"},
			}
			for _, item := range test.items {
				payload, err := json.Marshal(item)
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, []string{"customItems", item.ID.String(), string(payload)})
			}
			rows = append(rows,
				[]string{"savedItems", "count", "0"},
				[]string{"savedDiets", "count", "0"},
				[]string{"history", "count", "0"},
				[]string{"consent", "count", "0"},
			)
			rows = append(rows, test.rows...)
			var expected bytes.Buffer
			writer := csv.NewWriter(&expected)
			writer.WriteAll(rows)
			if !bytes.Equal(actual, expected.Bytes()) {
				t.Fatalf("CSV bytes = %q, want %q", actual, expected.Bytes())
			}
			if strings.Contains(string(actual), "parentId") {
				t.Fatalf("CSV leaked classification hierarchy: %s", actual)
			}
		})
	}
	if _, err := encodeCSV(ExportBundle{CustomItems: []ExportCustomItem{{MacrosPer100: ExportMacros{Protein: math.NaN()}}}}); err == nil {
		t.Fatal("invalid custom item CSV projection accepted")
	}
}

type memoryExportDiets struct {
	diets []repository.SavedDiet
}

type memoryExportCustomItems struct {
	items  []customitem.Item
	userID uuid.UUID
	err    error
}

func (r *memoryExportCustomItems) List(_ context.Context, userID uuid.UUID) ([]customitem.Item, error) {
	r.userID = userID
	return r.items, r.err
}

func (r memoryExportDiets) Create(context.Context, uuid.UUID, repository.SavedDiet) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (r memoryExportDiets) Get(context.Context, uuid.UUID, uuid.UUID) (repository.SavedDiet, error) {
	return repository.SavedDiet{}, nil
}
func (r memoryExportDiets) List(context.Context, uuid.UUID) ([]repository.SavedDiet, error) {
	return r.diets, nil
}
func (r memoryExportDiets) Replace(context.Context, uuid.UUID, repository.SavedDiet) error {
	return nil
}
func (r memoryExportDiets) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (r *memoryExportRepository) GetEncryptedUserByID(_ context.Context, userID uuid.UUID) (repository.EncryptedAuthUser, error) {
	if r.errAt == "identity" {
		return repository.EncryptedAuthUser{}, errors.New("identity failed")
	}
	r.user.ID = userID
	return r.user, nil
}

func (r *memoryExportRepository) GetOrCreateEncryptedProfile(context.Context, uuid.UUID) (repository.EncryptedUserProfile, error) {
	if r.errAt == "profile" {
		return repository.EncryptedUserProfile{}, errors.New("profile failed")
	}
	return r.profile, nil
}

func (r *memoryExportRepository) UpdateEncryptedProfile(context.Context, repository.EncryptedUserProfile) (repository.EncryptedUserProfile, error) {
	return repository.EncryptedUserProfile{}, nil
}

func (r *memoryExportRepository) SaveItem(context.Context, uuid.UUID, uuid.UUID, repository.SavedItemKind) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) RemoveItem(context.Context, uuid.UUID, uuid.UUID, repository.SavedItemKind) error {
	return nil
}

func (r *memoryExportRepository) ListItems(context.Context, uuid.UUID, *repository.SavedItemKind) ([]repository.SavedItem, error) {
	if r.errAt == "saved" {
		return nil, errors.New("saved failed")
	}
	return r.saved, nil
}

func (r *memoryExportRepository) AddEncryptedHistory(context.Context, repository.EncryptedSearchHistoryEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) ListEncryptedHistory(context.Context, uuid.UUID, int) ([]repository.EncryptedSearchHistoryEntry, error) {
	if r.errAt == "history" {
		return nil, errors.New("history failed")
	}
	return r.history, nil
}

func (r *memoryExportRepository) RecordConsent(context.Context, repository.ConsentRecord) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) HasRequiredConsent(context.Context, uuid.UUID, string, string) (bool, error) {
	return true, nil
}

func (r *memoryExportRepository) ListConsent(context.Context, uuid.UUID) ([]repository.ConsentRecord, error) {
	if r.errAt == "consent" {
		return nil, errors.New("consent failed")
	}
	return r.consent, nil
}

// TestExportServiceBuildsJSONAndCSV verifies DESIGN-008 DataExporter serialization.
func TestExportServiceBuildsJSONAndCSV(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	createdAt := time.Date(2026, 7, 29, 12, 30, 0, 0, time.UTC)
	encryption := security.NewEncryptionService(keyLoader{active: "pii-v1", entries: map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")}})
	encrypt := func(value string) repository.EncryptedField {
		field, err := encryption.EncryptPII(ctx, []byte(value))
		if err != nil {
			t.Fatal(err)
		}
		return repository.EncryptedField{KeyVersion: field.KeyVersion, Nonce: field.Nonce, Ciphertext: field.Ciphertext}
	}
	display := encrypt("Ada")
	repo := &memoryExportRepository{
		user:    repository.EncryptedAuthUser{Email: encrypt("ada@example.test"), Role: repository.UserRoleUser},
		profile: repository.EncryptedUserProfile{UserID: userID, DisplayName: &display, UnitSystem: repository.UnitSystemMetric, ThemePreference: "dark"},
		saved:   []repository.SavedItem{{ID: uuid.New(), UserID: userID, ItemID: uuid.New(), Kind: repository.SavedItemKindFavorite, CreatedAt: createdAt}},
		history: []repository.EncryptedSearchHistoryEntry{{ID: uuid.New(), UserID: userID, Query: encrypt("tomato"), Mode: "search", FiltersHash: "hash", CreatedAt: createdAt}},
		consent: []repository.ConsentRecord{{UserID: userID, PrivacyPolicyVersion: "privacy-v1", TermsVersion: "terms-v1"}},
	}
	dietID := uuid.New()
	diets := memoryExportDiets{diets: []repository.SavedDiet{{ID: dietID, UserID: userID, Name: "Training Day", Entries: []repository.SavedDietMealEntry{{ID: uuid.New(), MealID: uuid.New(), Quantity: 1, Unit: "g", Position: 0}}, CreatedAt: createdAt, UpdatedAt: createdAt}}}
	customID := uuid.New()
	customs := &memoryExportCustomItems{items: []customitem.Item{{
		ID: customID, Name: "Private tofu", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 2, Fat: 4}, Micros: repository.MicroValues{"Sodium": 3},
		FoodCategories: []customitem.ClassificationSummary{{ID: uuid.New(), Name: "Child", Kind: repository.ClassificationKindFoodCategory}},
	}}}
	service := NewExportService(repo, repo, repo, repo, repo, encryption, diets).WithCustomItems(customs)
	payload, err := service.BuildExport(ctx, userID, "json")
	if err != nil {
		t.Fatalf("BuildExport(json) error = %v", err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(payload.Body, &bundle); err != nil {
		t.Fatalf("decode export json: %v", err)
	}
	if bundle.User.Email != "ada@example.test" || bundle.User.DisplayName != "Ada" || len(bundle.SavedItems) != 1 || len(bundle.SavedDiets) != 1 || len(bundle.History) != 1 || len(bundle.CustomItems) != 1 || bundle.CustomItems[0].ID != customID || bundle.CustomItems[0].Name != "Private tofu" || customs.userID != userID {
		t.Fatalf("json bundle = %#v", bundle)
	}
	if strings.Contains(string(payload.Body), "ownerId") || strings.Contains(string(payload.Body), "parentId") || strings.Contains(string(payload.Body), "global") {
		t.Fatalf("JSON export leaked ownership/global data: %s", payload.Body)
	}
	var rawBundle map[string]any
	if err := json.Unmarshal(payload.Body, &rawBundle); err != nil {
		t.Fatalf("decode raw export json: %v", err)
	}
	if _, ok := rawBundle["format"]; ok {
		t.Fatalf("json bundle leaked transport format: %s", payload.Body)
	}
	assertOwnerFreeExport(t, rawBundle)
	savedDietsJSON, err := json.Marshal(rawBundle["savedDiets"])
	if err != nil || strings.Contains(string(savedDietsJSON), "UserID") || strings.Contains(string(savedDietsJSON), "userId") {
		t.Fatalf("saved-diet export leaked owner identity: %s err=%v", savedDietsJSON, err)
	}
	csvPayload, err := service.BuildExport(ctx, userID, "csv")
	if err != nil {
		t.Fatalf("BuildExport(csv) error = %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(csvPayload.Body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	foundCustom := false
	foundHistory := false
	foundDiet := false
	for _, row := range rows {
		if len(row) == 3 && row[0] == "customItems" && row[1] == customID.String() {
			var exported ExportCustomItem
			if err := json.Unmarshal([]byte(row[2]), &exported); err != nil || !reflect.DeepEqual(exported, bundle.CustomItems[0]) {
				t.Fatalf("CSV custom item = %#v err=%v", exported, err)
			}
			foundCustom = true
		}
		if len(row) == 3 && row[0] == "history" {
			var exported ExportSearchHistory
			if err := json.Unmarshal([]byte(row[2]), &exported); err != nil || exported.Query != "tomato" {
				t.Fatalf("CSV history = %#v err=%v", exported, err)
			}
			foundHistory = true
		}
		if len(row) == 3 && row[0] == "savedDiets" && row[1] == dietID.String() {
			var exported ExportSavedDiet
			if err := json.Unmarshal([]byte(row[2]), &exported); err != nil || exported.Name != "Training Day" {
				t.Fatalf("CSV saved diet = %#v err=%v", exported, err)
			}
			foundDiet = true
		}
	}
	if !foundCustom || !foundHistory || !foundDiet || strings.Contains(string(csvPayload.Body), "ownerId") || strings.Contains(string(csvPayload.Body), "parentId") || strings.Contains(string(csvPayload.Body), "global") {
		t.Fatalf("CSV custom item missing or leaked data: %s", csvPayload.Body)
	}
	secondJSON, err := service.BuildExport(ctx, userID, "json")
	if err != nil || !bytes.Equal(payload.Body, secondJSON.Body) {
		t.Fatalf("JSON export is not deterministic: err=%v", err)
	}
	secondCSV, err := service.BuildExport(ctx, userID, "csv")
	if err != nil || !bytes.Equal(csvPayload.Body, secondCSV.Body) {
		t.Fatalf("CSV export is not deterministic: err=%v", err)
	}
	if _, err := service.BuildExport(ctx, userID, "xml"); err == nil {
		t.Fatal("BuildExport() accepted unsupported format")
	}
}

func TestExportBundleSerializationBoundaryHasNoPersistenceTypes(t *testing.T) {
	repositoryPath := reflect.TypeOf(repository.SavedItem{}).PkgPath()
	customItemPath := reflect.TypeOf(customitem.Item{}).PkgPath()
	seen := map[reflect.Type]bool{}
	var inspect func(reflect.Type)
	inspect = func(value reflect.Type) {
		for value.Kind() == reflect.Pointer || value.Kind() == reflect.Slice || value.Kind() == reflect.Array || value.Kind() == reflect.Map {
			value = value.Elem()
		}
		if seen[value] {
			return
		}
		seen[value] = true
		if value.PkgPath() == repositoryPath || value.PkgPath() == customItemPath {
			t.Errorf("serialization boundary reaches persistence/API type %s", value)
			return
		}
		if value.Kind() != reflect.Struct || value.PkgPath() != reflect.TypeOf(ExportBundle{}).PkgPath() {
			return
		}
		for index := 0; index < value.NumField(); index++ {
			inspect(value.Field(index).Type)
		}
	}
	inspect(reflect.TypeOf(ExportBundle{}))
}

func assertOwnerFreeExport(t *testing.T, value any) {
	t.Helper()
	var walk func(any, []string)
	walk = func(current any, path []string) {
		switch typed := current.(type) {
		case []any:
			for index, item := range typed {
				walk(item, append(path, fmt.Sprint(index)))
			}
		case map[string]any:
			for key, item := range typed {
				normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
				topLevelIdentity := len(path) == 1 && path[0] == "user" && key == "userId"
				if !topLevelIdentity && (normalized == "userid" || normalized == "ownerid" || normalized == "owner") {
					t.Errorf("nested ownership key %q at %s", key, strings.Join(path, "."))
				}
				walk(item, append(path, key))
			}
		}
	}
	walk(value, nil)
}

func TestExportServicePropagatesValidationDependencyAndDecryptionErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	key := []byte("11111111111111111111111111111111")
	encryption := security.NewEncryptionService(keyLoader{active: "pii-v1", entries: map[string][]byte{"pii-v1": key}})
	encrypt := func(value string) repository.EncryptedField {
		field, err := encryption.EncryptPII(ctx, []byte(value))
		if err != nil {
			t.Fatal(err)
		}
		return repository.EncryptedField{KeyVersion: field.KeyVersion, Nonce: field.Nonce, Ciphertext: field.Ciphertext}
	}
	valid := func() *memoryExportRepository {
		return &memoryExportRepository{
			user:    repository.EncryptedAuthUser{Email: encrypt("ada@example.test")},
			profile: repository.EncryptedUserProfile{UserID: userID, UnitSystem: repository.UnitSystemMetric, ThemePreference: "system"},
		}
	}
	if _, err := NewExportService(valid(), valid(), valid(), valid(), valid(), encryption).BuildExport(ctx, userID, ""); err == nil {
		t.Fatal("empty export format accepted")
	}
	for _, stage := range []string{"identity", "profile", "saved", "history", "consent"} {
		repo := valid()
		repo.errAt = stage
		if _, err := NewExportService(repo, repo, repo, repo, repo, encryption).BuildExport(ctx, userID, "json"); err == nil {
			t.Fatalf("%s failure ignored", stage)
		}
	}
	badEmail := valid()
	badEmail.user.Email.KeyVersion = "missing"
	if _, err := NewExportService(badEmail, badEmail, badEmail, badEmail, badEmail, encryption).BuildExport(ctx, userID, "json"); err == nil {
		t.Fatal("email decryption failure ignored")
	}
	badDisplay := valid()
	display := encrypt("Ada")
	display.KeyVersion = "missing"
	badDisplay.profile.DisplayName = &display
	if _, err := NewExportService(badDisplay, badDisplay, badDisplay, badDisplay, badDisplay, encryption).BuildExport(ctx, userID, "json"); err == nil {
		t.Fatal("display-name decryption failure ignored")
	}
	badHistory := valid()
	query := encrypt("apple")
	query.KeyVersion = "missing"
	badHistory.history = []repository.EncryptedSearchHistoryEntry{{Query: query}}
	if _, err := NewExportService(badHistory, badHistory, badHistory, badHistory, badHistory, encryption).BuildExport(ctx, userID, "json"); err == nil {
		t.Fatal("history decryption failure ignored")
	}
	defaultRole := valid()
	payload, err := NewExportService(defaultRole, defaultRole, defaultRole, defaultRole, defaultRole, encryption).BuildExport(ctx, userID, "json")
	if err != nil {
		t.Fatal(err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(payload.Body, &bundle); err != nil || bundle.User.Role != "user" {
		t.Fatalf("default role bundle=%+v err=%v", bundle, err)
	}
}

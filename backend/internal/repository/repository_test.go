package repository

// Implements DESIGN-005 RepositoryInterfaces.
// Implements DESIGN-005 MacroNormalizer.
// Implements DESIGN-005 UnitConverter.

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

type contractFoodRepository struct{}

func (contractFoodRepository) GetByID(context.Context, uuid.UUID, RepositoryContext) (FoodItemEntity, error) {
	return FoodItemEntity{}, nil
}
func (contractFoodRepository) Search(context.Context, RepositoryQuery) ([]FoodItemEntity, int, error) {
	return nil, 0, nil
}
func (contractFoodRepository) Create(context.Context, FoodItemEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractFoodRepository) Update(context.Context, FoodItemEntity) error { return nil }
func (contractFoodRepository) Delete(context.Context, uuid.UUID) error      { return nil }

type contractMealRepository struct{}

func (contractMealRepository) GetByID(context.Context, uuid.UUID, RepositoryContext) (MealEntity, error) {
	return MealEntity{}, nil
}
func (contractMealRepository) Search(context.Context, RepositoryQuery) ([]MealEntity, int, error) {
	return nil, 0, nil
}
func (contractMealRepository) CalculateMacros(context.Context, uuid.UUID) (MacroValues, error) {
	return MacroValues{}, nil
}
func (contractMealRepository) Create(context.Context, MealEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractMealRepository) Update(context.Context, MealEntity) error { return nil }
func (contractMealRepository) Delete(context.Context, uuid.UUID) error  { return nil }

type contractClassificationRepository struct{}

func (contractClassificationRepository) List(context.Context, ClassificationKind) ([]ClassificationEntity, error) {
	return nil, nil
}
func (contractClassificationRepository) Upsert(context.Context, ClassificationEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractClassificationRepository) IsInUse(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}
func (contractClassificationRepository) SoftDelete(context.Context, uuid.UUID) error { return nil }

type contractVocabularyRepository struct{}

func (contractVocabularyRepository) ListActive(context.Context) ([]MicronutrientVocabularyEntry, error) {
	return nil, nil
}
func (contractVocabularyRepository) IsAllowed(context.Context, string) (bool, error) {
	return false, nil
}
func (contractVocabularyRepository) Upsert(context.Context, MicronutrientVocabularyEntry) error {
	return nil
}

type contractUserProfileRepository struct{}

func (contractUserProfileRepository) GetOrCreate(context.Context, uuid.UUID) (UserProfile, error) {
	return UserProfile{}, nil
}
func (contractUserProfileRepository) UpdateProfile(context.Context, UserProfile) (PreferenceUpdateResult, error) {
	return PreferenceUpdateResult{}, nil
}

type contractEncryptedUserProfileRepository struct{}

func (contractEncryptedUserProfileRepository) GetOrCreateEncryptedProfile(context.Context, uuid.UUID) (EncryptedUserProfile, error) {
	return EncryptedUserProfile{}, nil
}
func (contractEncryptedUserProfileRepository) UpdateEncryptedProfile(context.Context, EncryptedUserProfile) (EncryptedUserProfile, error) {
	return EncryptedUserProfile{}, nil
}

type contractSavedItemRepository struct{}

func (contractSavedItemRepository) SaveItem(context.Context, uuid.UUID, uuid.UUID, SavedItemKind) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractSavedItemRepository) RemoveItem(context.Context, uuid.UUID, uuid.UUID, SavedItemKind) error {
	return nil
}
func (contractSavedItemRepository) ListItems(context.Context, uuid.UUID, *SavedItemKind) ([]SavedItem, error) {
	return nil, nil
}

type contractSearchHistoryRepository struct{}

func (contractSearchHistoryRepository) AddHistory(context.Context, SearchHistoryEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractSearchHistoryRepository) ListHistory(context.Context, uuid.UUID, int) ([]SearchHistoryEntry, error) {
	return nil, nil
}
func (contractSearchHistoryRepository) ClearHistory(context.Context, uuid.UUID) error { return nil }

type contractEncryptedSearchHistoryRepository struct{}

func (contractEncryptedSearchHistoryRepository) AddEncryptedHistory(context.Context, EncryptedSearchHistoryEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractEncryptedSearchHistoryRepository) ListEncryptedHistory(context.Context, uuid.UUID, int) ([]EncryptedSearchHistoryEntry, error) {
	return nil, nil
}

type contractAuthUserRepository struct{}

func (contractAuthUserRepository) CreateUser(context.Context, AuthUser) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractAuthUserRepository) GetUserByID(context.Context, uuid.UUID) (AuthUser, error) {
	return AuthUser{}, nil
}
func (contractAuthUserRepository) GetUserByNormalizedEmail(context.Context, string) (AuthUser, error) {
	return AuthUser{}, nil
}
func (contractAuthUserRepository) UpdateUserState(context.Context, AuthUser) error { return nil }

type contractAccountDeletionRepository struct{}

func (contractAccountDeletionRepository) DeleteUserAccount(context.Context, uuid.UUID) error {
	return nil
}

type contractOAuthIdentityRepository struct{}

func (contractOAuthIdentityRepository) UpsertOAuthIdentity(context.Context, OAuthIdentity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractOAuthIdentityRepository) GetOAuthIdentity(context.Context, string, string) (OAuthIdentity, error) {
	return OAuthIdentity{}, nil
}

type contractSessionRepository struct{}

func (contractSessionRepository) CreateSession(context.Context, UserSession) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractSessionRepository) GetSessionByRefreshTokenHash(context.Context, string) (UserSession, error) {
	return UserSession{}, nil
}
func (contractSessionRepository) RevokeSession(context.Context, uuid.UUID) error       { return nil }
func (contractSessionRepository) RevokeSessionFamily(context.Context, uuid.UUID) error { return nil }
func (contractSessionRepository) RevokeUserSessions(context.Context, uuid.UUID) error  { return nil }

type contractPasswordResetTokenRepository struct{}

func (contractPasswordResetTokenRepository) CreatePasswordResetToken(context.Context, PasswordResetToken) error {
	return nil
}
func (contractPasswordResetTokenRepository) ConsumePasswordResetToken(context.Context, string, time.Time) (PasswordResetToken, error) {
	return PasswordResetToken{}, nil
}

type contractEntitlementRepository struct{}

func (contractEntitlementRepository) AppendEntitlement(context.Context, Entitlement) error {
	return nil
}
func (contractEntitlementRepository) GetLatest(context.Context, uuid.UUID) (Entitlement, error) {
	return Entitlement{}, nil
}

type contractUsageRepository struct{}

func (contractUsageRepository) RecordUsage(context.Context, uuid.UUID, string, time.Time) (UsageWindow, error) {
	return UsageWindow{}, nil
}
func (contractUsageRepository) RecordUsageWithinLimit(context.Context, uuid.UUID, string, time.Time, time.Time, int) (UsageWindow, bool, error) {
	return UsageWindow{}, false, nil
}
func (contractUsageRepository) GetUsageSince(context.Context, uuid.UUID, string, time.Time) (UsageWindow, error) {
	return UsageWindow{}, nil
}

type contractTrialRepository struct{}

func (contractTrialRepository) ListExpiredTrials(context.Context, time.Time) ([]Entitlement, error) {
	return nil, nil
}

type contractStripeEventRepository struct{}

func (contractStripeEventRepository) InsertProcessedStripeEvent(context.Context, ProcessedStripeEvent) (bool, error) {
	return false, nil
}
func (contractStripeEventRepository) ProcessStripeWebhookEvent(context.Context, ProcessedStripeEvent, *Entitlement) (bool, error) {
	return false, nil
}
func (contractStripeEventRepository) InsertStripeDeadLetter(context.Context, StripeDeadLetter) error {
	return nil
}

type contractCheckoutIdempotencyRepository struct{}

func (contractCheckoutIdempotencyRepository) GetCheckoutIdempotency(context.Context, uuid.UUID, string, string, string) (CheckoutIdempotencyRecord, error) {
	return CheckoutIdempotencyRecord{}, nil
}
func (contractCheckoutIdempotencyRepository) StoreCheckoutIdempotency(context.Context, CheckoutIdempotencyRecord) error {
	return nil
}

type contractConsentRepository struct{}

func (contractConsentRepository) RecordConsent(context.Context, ConsentRecord) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractConsentRepository) HasRequiredConsent(context.Context, uuid.UUID, string, string) (bool, error) {
	return false, nil
}
func (contractConsentRepository) ListConsent(context.Context, uuid.UUID) ([]ConsentRecord, error) {
	return nil, nil
}

type contractDeletionRequestRepository struct{}

func (contractDeletionRequestRepository) RequestDeletion(context.Context, uuid.UUID) (DataDeletionRequest, error) {
	return DataDeletionRequest{}, nil
}
func (contractDeletionRequestRepository) UpdateDeletionStatus(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (contractDeletionRequestRepository) ListDeletionAudit(context.Context, uuid.UUID) ([]DataDeletionAuditEntry, error) {
	return nil, nil
}
func (contractDeletionRequestRepository) ClaimDeletionRequests(context.Context, time.Time, int) ([]DataDeletionRequest, error) {
	return nil, nil
}
func (contractDeletionRequestRepository) RecordDeletionFailure(context.Context, uuid.UUID, string, string, *time.Time) error {
	return nil
}
func (contractDeletionRequestRepository) CompleteDeletionRequest(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}

type contractCuratedImportRepository struct{}

func (contractCuratedImportRepository) UpsertCuratedImport(context.Context, CuratedImport) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractCuratedImportRepository) FindCuratedImport(context.Context, string, string) (CuratedImport, error) {
	return CuratedImport{}, nil
}

type contractAdminAuditRepository struct{}

func (contractAdminAuditRepository) PersistAuditEntry(context.Context, AdminAuditEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (contractAdminAuditRepository) WithAudit(context.Context, AdminAuditEntry, func(sqlExecutor) error) error {
	return nil
}
func (contractAdminAuditRepository) ListAuditForEntity(context.Context, string, uuid.UUID) ([]AdminAuditEntry, error) {
	return nil, nil
}

var (
	_ FoodItemRepository                = contractFoodRepository{}
	_ MealRepository                    = contractMealRepository{}
	_ ClassificationRepository          = contractClassificationRepository{}
	_ MicronutrientVocabularyRepository = contractVocabularyRepository{}
	_ UserProfileRepository             = contractUserProfileRepository{}
	_ EncryptedUserProfileRepository    = contractEncryptedUserProfileRepository{}
	_ SavedItemRepository               = contractSavedItemRepository{}
	_ SearchHistoryRepository           = contractSearchHistoryRepository{}
	_ EncryptedSearchHistoryRepository  = contractEncryptedSearchHistoryRepository{}
	_ AuthUserRepository                = contractAuthUserRepository{}
	_ AccountDeletionRepository         = contractAccountDeletionRepository{}
	_ OAuthIdentityRepository           = contractOAuthIdentityRepository{}
	_ SessionRepository                 = contractSessionRepository{}
	_ PasswordResetTokenRepository      = contractPasswordResetTokenRepository{}
	_ SavedItemRepository               = (*PostgresSavedDataRepository)(nil)
	_ SearchHistoryRepository           = (*PostgresSavedDataRepository)(nil)
	_ EntitlementRepository             = contractEntitlementRepository{}
	_ UsageRepository                   = contractUsageRepository{}
	_ TrialRepository                   = contractTrialRepository{}
	_ StripeEventRepository             = contractStripeEventRepository{}
	_ EntitlementRepository             = (*PostgresEntitlementRepository)(nil)
	_ UsageRepository                   = (*PostgresEntitlementRepository)(nil)
	_ TrialRepository                   = (*PostgresEntitlementRepository)(nil)
	_ StripeEventRepository             = (*PostgresEntitlementRepository)(nil)
	_ ConsentRepository                 = contractConsentRepository{}
	_ DeletionRequestRepository         = contractDeletionRequestRepository{}
	_ ConsentRepository                 = (*PostgresComplianceRepository)(nil)
	_ DeletionRequestRepository         = (*PostgresComplianceRepository)(nil)
	_ CuratedImportRepository           = contractCuratedImportRepository{}
	_ AdminAuditRepository              = contractAdminAuditRepository{}
	_ CuratedImportRepository           = (*PostgresAdminImportAuditRepository)(nil)
	_ AdminAuditRepository              = (*PostgresAdminImportAuditRepository)(nil)
)

func TestRepositoryErrorKind(t *testing.T) {
	cause := errors.New("driver failed")
	err := NewError(ErrorKindConnection, "connect", cause)
	if got := err.Error(); got != "connection_error: connect: driver failed" {
		t.Fatalf("Error() = %q, want wrapped message", got)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("repository error does not wrap cause")
	}
	if !IsKind(err, ErrorKindConnection) {
		t.Fatalf("IsKind() = false, want true")
	}
	if IsKind(err, ErrorKindNotFound) {
		t.Fatalf("IsKind() = true for wrong kind")
	}

	plain := NewError(ErrorKindNotFound, "missing", nil)
	if got := plain.Error(); got != "not_found: missing" {
		t.Fatalf("plain Error() = %q, want unwrapped message", got)
	}
	if plain.Unwrap() != nil {
		t.Fatalf("plain Unwrap() != nil")
	}

	var nilErr *Error
	if got := nilErr.Error(); got != "" {
		t.Fatalf("nil Error() = %q, want empty string", got)
	}
	if nilErr.Unwrap() != nil {
		t.Fatalf("nil Unwrap() != nil")
	}
}

func TestNormalizeMacros(t *testing.T) {
	got, err := NormalizeMacros(MacroValues{Protein: 12.5, Carbohydrates: 10, Fat: 1}, 250, PhysicalStateSolid)
	if err != nil {
		t.Fatalf("NormalizeMacros() error = %v", err)
	}
	want := MacroValues{Protein: 5, Carbohydrates: 4, Fat: 0.4}
	if got != want {
		t.Fatalf("NormalizeMacros() = %#v, want %#v", got, want)
	}

	got, err = NormalizeMacros(MacroValues{Protein: 3, Carbohydrates: 9, Fat: 0}, 300, PhysicalStateLiquid)
	if err != nil {
		t.Fatalf("NormalizeMacros() liquid error = %v", err)
	}
	want = MacroValues{Protein: 1, Carbohydrates: 3, Fat: 0}
	if got != want {
		t.Fatalf("NormalizeMacros() liquid = %#v, want %#v", got, want)
	}
}

func TestNormalizeMacrosRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		macros   MacroValues
		quantity float64
		state    PhysicalState
	}{
		{name: "zero quantity", macros: MacroValues{}, quantity: 0, state: PhysicalStateSolid},
		{name: "negative macro", macros: MacroValues{Protein: -1}, quantity: 100, state: PhysicalStateSolid},
		{name: "non finite macro", macros: MacroValues{Protein: math.NaN()}, quantity: 100, state: PhysicalStateSolid},
		{name: "solid macros exceed mass", macros: MacroValues{Protein: 51, Carbohydrates: 50}, quantity: 100, state: PhysicalStateSolid},
		{name: "bad state", macros: MacroValues{}, quantity: 100, state: "frozen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NormalizeMacros(tt.macros, tt.quantity, tt.state); !IsKind(err, ErrorKindValidation) {
				t.Fatalf("NormalizeMacros() error = %v, want validation kind", err)
			}
		})
	}
}

func TestValidateMacrosPer100AllowsDenseLiquids(t *testing.T) {
	err := ValidateMacrosPer100(MacroValues{Protein: 60, Carbohydrates: 50}, PhysicalStateLiquid)
	if err != nil {
		t.Fatalf("ValidateMacrosPer100() dense liquid error = %v", err)
	}
}

func TestValidateMicronutrientKeys(t *testing.T) {
	vocab := []MicronutrientVocabularyEntry{
		{Key: "Sodium", Active: true},
		{Key: "Calcium", Active: false},
	}

	if err := ValidateMicronutrientKeys(MicroValues{"Sodium": 1}, vocab); err != nil {
		t.Fatalf("ValidateMicronutrientKeys() error = %v", err)
	}
	if err := ValidateMicronutrientKeys(MicroValues{"Calcium": 1}, vocab); !IsKind(err, ErrorKindInvalidMicronutrientKey) {
		t.Fatalf("ValidateMicronutrientKeys() error = %v, want invalid micronutrient kind", err)
	}
	if err := ValidateMicronutrientKeys(MicroValues{"Na": 1}, vocab); !IsKind(err, ErrorKindInvalidMicronutrientKey) {
		t.Fatalf("ValidateMicronutrientKeys() alias error = %v, want invalid micronutrient kind", err)
	}
}

func TestScaleMacros(t *testing.T) {
	got := ScaleMacros(MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}, 33.3333, 100)
	want := MacroValues{Protein: 3.3333, Carbohydrates: 6.6667, Fat: 1.6667}
	if got != want {
		t.Fatalf("ScaleMacros() = %#v, want %#v", got, want)
	}
	if got := ScaleMacros(MacroValues{Protein: 10}, 1, 0); got != (MacroValues{}) {
		t.Fatalf("ScaleMacros() zero basis = %#v, want zero value", got)
	}
}

func TestConvertUnit(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		from string
		to   string
		want float64
	}{
		{name: "grams to ounces", in: 28.349523125, from: "g", to: "oz", want: 1},
		{name: "ounces to grams", in: 1, from: "oz", to: "g", want: 28.3495},
		{name: "milliliters to fluid ounces", in: 29.5735295625, from: "ml", to: "fl_oz", want: 1},
		{name: "fluid ounces to milliliters", in: 1, from: "fl_oz", to: "ml", want: 29.5735},
		{name: "same unit", in: 2.12345, from: "g", to: "g", want: 2.12345},
		{name: "same serving unit", in: 1, from: "serving", to: "serving", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertUnit(tt.in, tt.from, tt.to)
			if err != nil {
				t.Fatalf("ConvertUnit() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ConvertUnit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertUnitRejectsUnsupportedAndNegativeValues(t *testing.T) {
	if _, err := ConvertUnit(1, "g", "ml"); !IsKind(err, ErrorKindUnitConversion) {
		t.Fatalf("ConvertUnit() error = %v, want unit conversion kind", err)
	}
	if _, err := ConvertUnit(-1, "g", "oz"); !IsKind(err, ErrorKindUnitConversion) {
		t.Fatalf("ConvertUnit() negative error = %v, want unit conversion kind", err)
	}
	for _, unit := range []string{"grams", "fl oz", "servings", "cup"} {
		if _, err := ConvertUnit(1, unit, unit); !IsKind(err, ErrorKindUnitConversion) {
			t.Fatalf("ConvertUnit(%q) error = %v, want unit conversion kind", unit, err)
		}
	}
}

func TestConvertServingToBase(t *testing.T) {
	quantity, unit, err := ConvertServingToBase(2, 125, 0, PhysicalStateSolid)
	if err != nil {
		t.Fatalf("ConvertServingToBase() error = %v", err)
	}
	if quantity != 250 || unit != "g" {
		t.Fatalf("ConvertServingToBase() = %v %s, want 250 g", quantity, unit)
	}

	quantity, unit, err = ConvertServingToBase(1.5, 0, 200, PhysicalStateLiquid)
	if err != nil {
		t.Fatalf("ConvertServingToBase() liquid error = %v", err)
	}
	if quantity != 300 || unit != "ml" {
		t.Fatalf("ConvertServingToBase() liquid = %v %s, want 300 ml", quantity, unit)
	}
}

func TestConvertServingToBaseRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		servings float64
		weight   float64
		volume   float64
		state    PhysicalState
		kind     ErrorKind
	}{
		{name: "negative servings", servings: -1, weight: 1, state: PhysicalStateSolid, kind: ErrorKindUnitConversion},
		{name: "missing weight", servings: 1, weight: 0, state: PhysicalStateSolid, kind: ErrorKindUnitConversion},
		{name: "missing liquid volume", servings: 1, volume: 0, state: PhysicalStateLiquid, kind: ErrorKindUnitConversion},
		{name: "bad state", servings: 1, weight: 1, state: "powder", kind: ErrorKindValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ConvertServingToBase(tt.servings, tt.weight, tt.volume, tt.state)
			if !IsKind(err, tt.kind) {
				t.Fatalf("ConvertServingToBase() error = %v, want %s", err, tt.kind)
			}
		})
	}
}

func TestValidateFoodDensity(t *testing.T) {
	tests := []struct {
		name string
		item FoodItemEntity
	}{
		{name: "negative serving volume", item: FoodItemEntity{PhysicalState: PhysicalStateLiquid, AverageServingVolumeMilliliters: -1}},
		{name: "negative density", item: FoodItemEntity{PhysicalState: PhysicalStateLiquid, DensityGramsPerMilliliter: -1}},
		{name: "solid liquid metadata", item: FoodItemEntity{PhysicalState: PhysicalStateSolid, DensityGramsPerMilliliter: 1}},
		{name: "provenance without density", item: FoodItemEntity{PhysicalState: PhysicalStateLiquid, DensitySourceKind: "manual"}},
		{name: "invalid provenance kind", item: FoodItemEntity{PhysicalState: PhysicalStateLiquid, DensityGramsPerMilliliter: 1, DensitySourceKind: "guessed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateFoodDensity(tt.item); !IsKind(err, ErrorKindValidation) {
				t.Fatalf("validateFoodDensity() error = %v, want validation", err)
			}
		})
	}
	for _, kind := range []string{"imported", "manual", "estimated"} {
		if err := validateFoodDensity(FoodItemEntity{PhysicalState: PhysicalStateLiquid, DensityGramsPerMilliliter: 1, DensitySourceKind: kind}); err != nil {
			t.Fatalf("validateFoodDensity() kind %q error = %v", kind, err)
		}
	}
}

func TestIngredientMassGrams(t *testing.T) {
	if got, err := ingredientMassGrams(125, FoodItemEntity{PhysicalState: PhysicalStateSolid}); got != 125 || err != nil {
		t.Fatalf("ingredientMassGrams() solid = %v, %v", got, err)
	}
	if got, err := ingredientMassGrams(125, FoodItemEntity{PhysicalState: PhysicalStateLiquid, DensityGramsPerMilliliter: 1.2}); got != 150 || err != nil {
		t.Fatalf("ingredientMassGrams() liquid = %v, %v", got, err)
	}
	if got, err := ingredientMassGrams(125, FoodItemEntity{PhysicalState: PhysicalStateLiquid}); got != 0 || !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ingredientMassGrams() missing density = %v, %v", got, err)
	}
}

func TestRemainingRepositoryCoverageBranches(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()

	if err := ValidateMacrosPer100(MacroValues{}, "frozen"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ValidateMacrosPer100() state error = %v", err)
	}
	if err := NewPostgresFoodItemRepository(nil).validateFoodItem(ctx, FoodItemEntity{PhysicalState: PhysicalStateSolid}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateFoodItem() name error = %v", err)
	}
	if err := NewPostgresFoodItemRepository(nil).validateFoodItem(ctx, FoodItemEntity{Name: "Bad Density", PhysicalState: PhysicalStateSolid, DensityGramsPerMilliliter: 1}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateFoodItem() density error = %v", err)
	}
	if err := NewPostgresMealRepository(nil).validateMeal(ctx, MealEntity{Type: MealTypeSingle, PhysicalState: PhysicalStateSolid}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateMeal() name error = %v", err)
	}
	if err := NewPostgresMealRepository(nil).validateMeal(ctx, MealEntity{Type: "snack", Name: "Bad Type", PhysicalState: PhysicalStateSolid}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateMeal() type error = %v", err)
	}
	if macros, available, err := NewPostgresMealRepository(nil).calculateCompositeMacros(ctx, nil); err == nil || available || macros != (MacroValues{}) {
		t.Fatalf("calculateCompositeMacros() empty = %#v, %v, %v", macros, available, err)
	}
	liquidValues := foodFixtureValues(itemID)
	liquidValues[2] = PhysicalStateLiquid
	if macros, available, err := NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: liquidValues}, rows: &fakeRows{}}).calculateCompositeMacros(ctx, []RecipeIngredientEntity{{FoodItemID: itemID, Quantity: 100, Unit: "ml"}}); !IsKind(err, ErrorKindValidation) || available || macros != (MacroValues{}) {
		t.Fatalf("calculateCompositeMacros() missing density = %#v, %v, %v", macros, available, err)
	}
	if meals, total, err := NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{}}).Search(ctx, RepositoryQuery{Offset: -1}); err != nil || len(meals) != 0 || total != 0 {
		t.Fatalf("Search() negative offset = %#v, %v, %v", meals, total, err)
	}
	repo := NewPostgresSavedDataRepository(&fakeSQLExecutor{
		rowList: []fakeRow{{values: foodFixtureValues(itemID)}, {err: errors.New("insert failed")}},
		rows:    &fakeRows{},
	})
	if _, err := repo.SaveItem(ctx, userID, itemID, SavedItemKindFavorite); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("SaveItem() insert error = %v", err)
	}
}

func TestMealSearchSkipsPageAndHydrationBeyondTotal(t *testing.T) {
	db := &fakeSQLExecutor{row: fakeRow{values: []any{0}}, rows: &fakeRows{}}
	meals, total, err := NewPostgresMealRepository(db).Search(context.Background(), RepositoryQuery{Limit: 100, Offset: 100})
	if err != nil || total != 0 || len(meals) != 0 {
		t.Fatalf("Search() = %d meals, total %d, %v", len(meals), total, err)
	}
	if db.queryCalls != 0 {
		t.Fatalf("page query calls = %d, want 0 beyond total", db.queryCalls)
	}
}

func TestMealSearchAppliesPaginationInSQL(t *testing.T) {
	db := &fakeSQLExecutor{row: fakeRow{values: []any{250}}, rows: &fakeRows{}}
	meals, total, err := NewPostgresMealRepository(db).Search(context.Background(), RepositoryQuery{Limit: 100, Offset: 100})
	if err != nil || total != 250 || len(meals) != 0 {
		t.Fatalf("Search() = %d meals, total %d, %v", len(meals), total, err)
	}
	if db.queryCalls != 1 || len(db.queryArgs) != 1 {
		t.Fatalf("page query calls = %d args=%v, want one", db.queryCalls, db.queryArgs)
	}
	args := db.queryArgs[0]
	if len(args) != 7 || args[5] != 100 || args[6] != 100 {
		t.Fatalf("page query args = %#v, want limit=100 offset=100", args)
	}
}

func foodFixtureValues(id uuid.UUID) []any {
	now := time.Now()
	return []any{id, "Fixture", PhysicalStateSolid, 0, (*float64)(nil), (*float64)(nil), (*float64)(nil), (*string)(nil), (*string)(nil), (*string)(nil), 0.0, 0.0, 0.0, []byte(`{}`), (*string)(nil), (*time.Time)(nil), now, now}
}

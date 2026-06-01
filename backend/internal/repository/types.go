// Package repository defines domain persistence contracts for Mealswapp.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PhysicalState identifies the storage basis for food and meal quantities.
// Implements DESIGN-005 FoodItemEntity.
type PhysicalState string

// Implements DESIGN-005 FoodItemEntity.
const (
	PhysicalStateSolid  PhysicalState = "solid"
	PhysicalStateLiquid PhysicalState = "liquid"
)

// UnitSystem identifies caller-facing unit conversion preferences.
// Implements DESIGN-005 UnitConverter.
type UnitSystem string

// Implements DESIGN-005 UnitConverter.
const (
	UnitSystemMetric   UnitSystem = "metric"
	UnitSystemImperial UnitSystem = "imperial"
)

// MacroValues stores protein, carbohydrates, and fat values.
// Implements DESIGN-005 MacroNormalizer.
type MacroValues struct {
	Protein       float64
	Carbohydrates float64
	Fat           float64
}

// MicroValues stores micronutrient values by canonical vocabulary key.
// Implements DESIGN-005 MicronutrientVocabulary.
type MicroValues map[string]float64

// MicronutrientVocabularyEntry stores one canonical micronutrient definition.
// Implements DESIGN-005 MicronutrientVocabulary.
type MicronutrientVocabularyEntry struct {
	Key         string
	DisplayName string
	Unit        string
	Active      bool
}

// TagKind identifies category and functionality tag groups.
// Implements DESIGN-005 TagEntity.
type TagKind string

// Implements DESIGN-005 TagEntity.
const (
	TagKindCategory      TagKind = "category"
	TagKindFunctionality TagKind = "functionality"
)

// TagEntity stores global tag identity and optional hierarchy.
// Implements DESIGN-005 TagEntity.
type TagEntity struct {
	ID       uuid.UUID
	Name     string
	Kind     TagKind
	ParentID *uuid.UUID
}

// FoodItemEntity stores normalized food item data owned by repositories.
// Implements DESIGN-005 FoodItemEntity.
type FoodItemEntity struct {
	ID                              uuid.UUID
	Name                            string
	PhysicalState                   PhysicalState
	PrepTimeMinutes                 int
	AverageUnitWeightGrams          float64
	AverageServingVolumeMilliliters float64
	DensityGramsPerMilliliter       float64
	DensitySourceProvider           string
	DensitySourceFoodID             string
	DensitySourceKind               string
	MacrosPer100                    MacroValues
	Micros                          MicroValues
	CategoryTags                    []TagEntity
	FunctionalityTags               []TagEntity
	ImageURL                        string
	DeletedAt                       *time.Time
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
}

// MealType identifies opaque single and composite meals.
// Implements DESIGN-005 MealEntity.
type MealType string

// Implements DESIGN-005 MealEntity.
const (
	MealTypeSingle    MealType = "single"
	MealTypeComposite MealType = "composite"
)

// RecipeIngredientEntity stores one recipe ingredient quantity.
// Implements DESIGN-005 RecipeEntity.
type RecipeIngredientEntity struct {
	FoodItemID uuid.UUID
	Quantity   float64
	Unit       string
	Position   int
}

// MealEntity stores opaque single and composite meal data.
// Implements DESIGN-005 MealEntity.
type MealEntity struct {
	ID                        uuid.UUID
	Type                      MealType
	Name                      string
	RecipeItems               []RecipeIngredientEntity
	PhysicalState             PhysicalState
	PrepTimeMinutes           int
	AverageUnitWeightGrams    float64
	MacrosPer100              MacroValues
	NormalizedMacrosAvailable bool
	Tags                      []TagEntity
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// RepositoryContext carries caller scoping and conversion preferences.
// Implements DESIGN-005 RepositoryInterfaces.
type RepositoryContext struct {
	UserID         *uuid.UUID
	UnitSystem     UnitSystem
	IncludeDeleted bool
}

// RepositoryQuery carries deterministic search and filter inputs.
// Implements DESIGN-005 RepositoryInterfaces.
type RepositoryQuery struct {
	RepositoryContext
	Name             string
	CategoryTagIDs   []uuid.UUID
	FunctionalityIDs []uuid.UUID
	MaxPrepMinutes   *int
	Limit            int
	Offset           int
}

// UserRole identifies supported repository-owned user roles.
// Implements DESIGN-006 AuthUser.
type UserRole string

// Implements DESIGN-006 AuthUser.
const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// AuthUser stores repository-owned authentication identity data.
// Implements DESIGN-006 AuthUser.
type AuthUser struct {
	ID            uuid.UUID
	Email         string
	EmailVerified bool
	Role          UserRole
	PasswordHash  *string
	PasswordSalt  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OAuthIdentity stores a provider-specific login identity linked to a user.
// Implements DESIGN-006 OAuthHandler.
type OAuthIdentity struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Provider       string
	ProviderUserID string
	Email          string
	CreatedAt      time.Time
}

// UserProfile stores user-scoped preferences and profile data.
// Implements DESIGN-008 PreferenceManager.
type UserProfile struct {
	UserID          uuid.UUID
	DisplayName     string
	UnitSystem      UnitSystem
	ThemePreference string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SavedItemKind identifies user-scoped saved data categories.
// Implements DESIGN-008 SavedDataRepository.
type SavedItemKind string

// Implements DESIGN-008 SavedDataRepository.
const (
	SavedItemKindFavorite  SavedItemKind = "favorite"
	SavedItemKindSavedMeal SavedItemKind = "saved_meal"
	SavedItemKindSavedDiet SavedItemKind = "saved_diet"
)

// SavedItem stores one user-owned favorite, saved meal, or saved diet reference.
// Implements DESIGN-008 SavedDataRepository.
type SavedItem struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ItemID    uuid.UUID
	Kind      SavedItemKind
	CreatedAt time.Time
}

// SearchHistoryEntry stores one user-owned search history record.
// Implements DESIGN-008 SearchHistoryRepository.
type SearchHistoryEntry struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Query       string
	Mode        string
	FiltersHash string
	CreatedAt   time.Time
}

// PreferenceUpdateResult returns saved preferences and recalculation guidance.
// Implements DESIGN-008 PreferenceManager.
type PreferenceUpdateResult struct {
	Profile                   UserProfile
	RequiresUnitRecalculation bool
}

// Entitlement stores subscription tier state.
// Implements DESIGN-007 EntitlementManager.
type Entitlement struct {
	UserID               uuid.UUID
	Tier                 string
	Status               string
	SearchLimitPer24h    int
	AllowedModes         []string
	ExpiresAt            *time.Time
	StripeCustomerID     string
	StripeSubscriptionID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// UsageWindow stores feature usage for a bounded entitlement window.
// Implements DESIGN-007 UsageLimiter.
type UsageWindow struct {
	UserID      uuid.UUID
	Feature     string
	StartedAt   time.Time
	SearchCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProcessedStripeEvent stores webhook idempotency metadata.
// Implements DESIGN-007 StripeWebhookHandler.
type ProcessedStripeEvent struct {
	EventID     string
	EventType   string
	Outcome     string
	Payload     []byte
	ProcessedAt time.Time
}

// AdminAuditEntry stores auditable administrative mutations.
// Implements DESIGN-009 AdminController.
type AdminAuditEntry struct {
	ID          uuid.UUID
	AdminUserID uuid.UUID
	Action      string
	EntityType  string
	EntityID    *uuid.UUID
	Before      []byte
	After       []byte
	RequestID   string
	CreatedAt   time.Time
}

// ConsentRecord stores accepted legal consent versions.
// Implements DESIGN-015 ConsentManager.
type ConsentRecord struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	PrivacyPolicyVersion string
	TermsVersion         string
	AcceptedAt           time.Time
}

// DataDeletionRequest stores auditable account-erasure workflow state.
// Implements DESIGN-015 DataRetentionPolicy.
type DataDeletionRequest struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Status        string
	RequestedAt   time.Time
	CompletedAt   *time.Time
	FailureReason string
}

// DataDeletionAuditEntry stores deletion status transitions.
// Implements DESIGN-015 DataRetentionPolicy.
type DataDeletionAuditEntry struct {
	ID         uuid.UUID
	RequestID  uuid.UUID
	FromStatus string
	ToStatus   string
	Note       string
	CreatedAt  time.Time
}

// CuratedImport stores external curation metadata.
// Implements DESIGN-009 DataImporter.
type CuratedImport struct {
	ID             uuid.UUID
	SourceProvider string
	ExternalID     string
	FoodItemID     *uuid.UUID
	Status         string
	ConflictReason string
	RawPayload     []byte
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FoodItemRepository defines food item persistence behavior.
// Implements DESIGN-005 RepositoryInterfaces.
type FoodItemRepository interface {
	GetByID(ctx context.Context, id uuid.UUID, rc RepositoryContext) (FoodItemEntity, error)
	Search(ctx context.Context, q RepositoryQuery) ([]FoodItemEntity, int, error)
	Create(ctx context.Context, item FoodItemEntity) (uuid.UUID, error)
	Update(ctx context.Context, item FoodItemEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MealRepository defines meal and recipe persistence behavior.
// Implements DESIGN-005 RepositoryInterfaces.
type MealRepository interface {
	GetByID(ctx context.Context, id uuid.UUID, rc RepositoryContext) (MealEntity, error)
	Search(ctx context.Context, q RepositoryQuery) ([]MealEntity, int, error)
	CalculateMacros(ctx context.Context, mealID uuid.UUID) (MacroValues, error)
	Create(ctx context.Context, meal MealEntity) (uuid.UUID, error)
	Update(ctx context.Context, meal MealEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TagRepository defines tag persistence behavior.
// Implements DESIGN-005 RepositoryInterfaces.
type TagRepository interface {
	List(ctx context.Context, kind TagKind) ([]TagEntity, error)
	Upsert(ctx context.Context, tag TagEntity) (uuid.UUID, error)
	IsInUse(ctx context.Context, id uuid.UUID) (bool, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// MicronutrientVocabularyRepository defines micronutrient vocabulary persistence behavior.
// Implements DESIGN-005 RepositoryInterfaces.
type MicronutrientVocabularyRepository interface {
	ListActive(ctx context.Context) ([]MicronutrientVocabularyEntry, error)
	IsAllowed(ctx context.Context, key string) (bool, error)
	Upsert(ctx context.Context, entry MicronutrientVocabularyEntry) error
}

// UserProfileRepository defines user profile and preference persistence behavior.
// Implements DESIGN-008 PreferenceManager.
type UserProfileRepository interface {
	GetOrCreate(ctx context.Context, userID uuid.UUID) (UserProfile, error)
	UpdateProfile(ctx context.Context, profile UserProfile) (PreferenceUpdateResult, error)
}

// SavedItemRepository defines saved-item persistence behavior.
// Implements DESIGN-008 SavedDataRepository.
type SavedItemRepository interface {
	SaveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) (uuid.UUID, error)
	RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) error
	ListItems(ctx context.Context, userID uuid.UUID, kind *SavedItemKind) ([]SavedItem, error)
}

// SearchHistoryRepository defines search-history persistence behavior.
// Implements DESIGN-008 SearchHistoryRepository.
type SearchHistoryRepository interface {
	AddHistory(ctx context.Context, entry SearchHistoryEntry) (uuid.UUID, error)
	ListHistory(ctx context.Context, userID uuid.UUID, limit int) ([]SearchHistoryEntry, error)
}

// EntitlementRepository defines entitlement-state persistence behavior.
// Implements DESIGN-007 EntitlementManager.
type EntitlementRepository interface {
	AppendEntitlement(ctx context.Context, entitlement Entitlement) error
	GetLatest(ctx context.Context, userID uuid.UUID) (Entitlement, error)
}

// UsageRepository defines rolling usage persistence behavior.
// Implements DESIGN-007 UsageLimiter.
type UsageRepository interface {
	RecordUsage(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (UsageWindow, error)
	GetUsageSince(ctx context.Context, userID uuid.UUID, feature string, since time.Time) (UsageWindow, error)
}

// TrialRepository defines trial-expiry persistence behavior.
// Implements DESIGN-007 TrialTracker.
type TrialRepository interface {
	ListExpiredTrials(ctx context.Context, now time.Time) ([]Entitlement, error)
}

// StripeEventRepository defines Stripe webhook idempotency persistence behavior.
// Implements DESIGN-007 StripeWebhookHandler.
type StripeEventRepository interface {
	InsertProcessedStripeEvent(ctx context.Context, event ProcessedStripeEvent) (bool, error)
}

// ConsentRepository defines consent persistence behavior.
// Implements DESIGN-015 ConsentManager.
type ConsentRepository interface {
	RecordConsent(ctx context.Context, record ConsentRecord) (uuid.UUID, error)
	HasRequiredConsent(ctx context.Context, userID uuid.UUID, privacyVersion string, termsVersion string) (bool, error)
}

// DeletionRequestRepository defines account-deletion workflow persistence behavior.
// Implements DESIGN-015 DataRetentionPolicy.
type DeletionRequestRepository interface {
	RequestDeletion(ctx context.Context, userID uuid.UUID) (DataDeletionRequest, error)
	UpdateDeletionStatus(ctx context.Context, requestID uuid.UUID, status string, note string) error
	ListDeletionAudit(ctx context.Context, requestID uuid.UUID) ([]DataDeletionAuditEntry, error)
}

// CuratedImportRepository defines curated external-import persistence behavior.
// Implements DESIGN-009 DataImporter.
type CuratedImportRepository interface {
	UpsertCuratedImport(ctx context.Context, item CuratedImport) (uuid.UUID, error)
	FindCuratedImport(ctx context.Context, provider string, externalID string) (CuratedImport, error)
}

// AdminAuditRepository defines administrative mutation audit persistence behavior.
// Implements DESIGN-009 AdminController.
type AdminAuditRepository interface {
	PersistAuditEntry(ctx context.Context, entry AdminAuditEntry) (uuid.UUID, error)
	WithAudit(ctx context.Context, entry AdminAuditEntry, fn func(sqlExecutor) error) error
	ListAuditForEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]AdminAuditEntry, error)
}

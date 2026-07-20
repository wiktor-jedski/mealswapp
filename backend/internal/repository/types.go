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

// ClassificationKind identifies Food Category and Culinary Role classification groups.
// Implements DESIGN-005 ClassificationEntity.
type ClassificationKind string

// Implements DESIGN-005 ClassificationEntity.
const (
	ClassificationKindFoodCategory ClassificationKind = "food_category"
	ClassificationKindCulinaryRole ClassificationKind = "culinary_role"
)

// ClassificationEntity stores global classification identity and optional hierarchy.
// Implements DESIGN-005 ClassificationEntity.
type ClassificationEntity struct {
	ID       uuid.UUID
	Name     string
	Kind     ClassificationKind
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
	FoodCategories                  []ClassificationEntity
	CulinaryRoles                   []ClassificationEntity
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
	Classifications           []ClassificationEntity
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
	Name                    string
	FoodCategoryIDs         []uuid.UUID
	ExcludedFoodCategoryIDs []uuid.UUID
	CulinaryRoleIDs         []uuid.UUID
	ExcludedCulinaryRoleIDs []uuid.UUID
	AllergenIDs             []uuid.UUID
	ExcludedAllergenIDs     []uuid.UUID
	AllergenKeys            []string
	ExcludedAllergenKeys    []string
	FoodObjectTypes         []PhysicalState
	ExcludedFoodObjectTypes []PhysicalState
	MaxPrepMinutes          *int
	Limit                   int
	Offset                  int
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

// EncryptedField stores one encrypted PII envelope at the repository boundary.
// Implements DESIGN-013 EncryptionService.
type EncryptedField struct {
	KeyVersion string
	Nonce      []byte
	Ciphertext []byte
}

// LookupDigest stores deterministic keyed lookup material for encrypted PII.
// Implements DESIGN-013 EncryptionService.
type LookupDigest struct {
	KeyVersion string
	Value      string
}

// EncryptedAuthUser stores encrypted account identity data.
// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService.
type EncryptedAuthUser struct {
	ID                    uuid.UUID
	Email                 EncryptedField
	NormalizedEmailDigest LookupDigest
	EmailVerified         bool
	Role                  UserRole
	PasswordHash          *string
	PasswordSalt          *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
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

// EncryptedOAuthIdentity stores encrypted provider identity data.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
type EncryptedOAuthIdentity struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	Provider             string
	ProviderUserID       EncryptedField
	ProviderUserIDDigest LookupDigest
	Email                EncryptedField
	CreatedAt            time.Time
}

// UserSession stores refresh-token rotation metadata for an authenticated user.
// Implements DESIGN-006 AuthController.
type UserSession struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
	RefreshFamilyID  uuid.UUID
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

// PasswordResetToken stores a hashed password-reset token.
// Implements DESIGN-006 AuthController.
type PasswordResetToken struct {
	TokenHash string
	UserID    uuid.UUID
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// AccountLockoutState stores persisted failed-login lockout metadata.
// Implements DESIGN-006 AccountLockoutTracker.
type AccountLockoutState struct {
	UserID           uuid.UUID
	FailedLoginCount int
	LockedUntil      *time.Time
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

// EncryptedUserProfile stores encrypted profile PII with non-PII preferences.
// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService.
type EncryptedUserProfile struct {
	UserID          uuid.UUID
	DisplayName     *EncryptedField
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

// SavedDiet stores one user-owned daily diet and its ordered meal entries.
// Implements DESIGN-008 SavedDataRepository.
type SavedDiet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	Entries   []SavedDietMealEntry
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SavedDietMealEntry stores one positive, canonically-unitized meal quantity.
// Implements DESIGN-008 SavedDataRepository.
type SavedDietMealEntry struct {
	ID          uuid.UUID
	SavedDietID uuid.UUID
	MealID      uuid.UUID
	Quantity    float64
	Unit        string
	Position    int
	CreatedAt   time.Time
}

// DailyDiet is the descriptive alias used by the Phase 07 API boundary.
// Implements DESIGN-008 SavedDataRepository.
type DailyDiet = SavedDiet

// DailyDietMealEntry is the descriptive alias used by the Phase 07 API boundary.
// Implements DESIGN-008 SavedDataRepository.
type DailyDietMealEntry = SavedDietMealEntry

// DailyDietCreateResponse is the immutable successful create projection stored for exact replay.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
type DailyDietCreateResponse struct {
	ID              uuid.UUID                      `json:"id"`
	Name            string                         `json:"name"`
	Entries         []DailyDietCreateResponseEntry `json:"entries"`
	AggregateMacros DailyDietCreateResponseMacros  `json:"aggregateMacros"`
	CreatedAt       time.Time                      `json:"createdAt"`
	UpdatedAt       time.Time                      `json:"updatedAt"`
}

// DailyDietCreateResponseEntry is one immutable entry in a create response.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
type DailyDietCreateResponseEntry struct {
	ID       uuid.UUID `json:"id"`
	MealID   uuid.UUID `json:"mealId"`
	Quantity float64   `json:"quantity"`
	Unit     string    `json:"unit"`
	Position int       `json:"position"`
}

// DailyDietCreateResponseMacros is the immutable aggregate returned by create.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
type DailyDietCreateResponseMacros struct {
	Protein       float64 `json:"protein"`
	Carbohydrates float64 `json:"carbohydrates"`
	Fat           float64 `json:"fat"`
	Calories      float64 `json:"calories"`
}

// DailyDietCreateClaim is one typed, user-scoped create mutation claim.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
type DailyDietCreateClaim struct {
	UserID     uuid.UUID
	Key        string
	BodyHash   string
	Diet       SavedDiet
	Response   DailyDietCreateResponse
	StatusCode int
}

// DailyDietCreateClaimResult returns either the newly persisted or original immutable response.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
type DailyDietCreateClaimResult struct {
	Response   DailyDietCreateResponse
	StatusCode int
	Replayed   bool
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

// EncryptedSearchHistoryEntry stores encrypted search-history query text.
// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService.
type EncryptedSearchHistoryEntry struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Query       EncryptedField
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

// StripeDeadLetter stores sanitized webhook failure metadata without raw payment payloads.
// Implements DESIGN-007 StripeWebhookHandler dead-letter persistence.
type StripeDeadLetter struct {
	EventID              string
	EventType            string
	FailureCategory      string
	ErrorMessage         string
	PayloadSHA256        string
	StripeCustomerID     string
	StripeSubscriptionID string
	UserID               *uuid.UUID
	CreatedAt            time.Time
}

// CheckoutIdempotencyRecord stores one completed checkout creation response.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
type CheckoutIdempotencyRecord struct {
	UserID       uuid.UUID
	Method       string
	Route        string
	Key          string
	BodyHash     string
	StatusCode   int
	ResponseBody []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          string
	RequestedAt     time.Time
	CompletedAt     *time.Time
	FailureReason   string
	FailureCategory string
	RetryCount      int
	NextAttemptAt   *time.Time
	ReceiptID       *uuid.UUID
	ReceiptIssuedAt *time.Time
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

// ClassificationRepository defines classification persistence behavior.
// Implements DESIGN-005 RepositoryInterfaces.
type ClassificationRepository interface {
	List(ctx context.Context, kind ClassificationKind) ([]ClassificationEntity, error)
	Upsert(ctx context.Context, classification ClassificationEntity) (uuid.UUID, error)
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

// EncryptedUserProfileRepository defines encrypted profile PII persistence behavior.
// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService.
type EncryptedUserProfileRepository interface {
	GetOrCreateEncryptedProfile(ctx context.Context, userID uuid.UUID) (EncryptedUserProfile, error)
	UpdateEncryptedProfile(ctx context.Context, profile EncryptedUserProfile) (EncryptedUserProfile, error)
}

// SavedItemRepository defines saved-item persistence behavior.
// Implements DESIGN-008 SavedDataRepository.
type SavedItemRepository interface {
	SaveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) (uuid.UUID, error)
	RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) error
	ListItems(ctx context.Context, userID uuid.UUID, kind *SavedItemKind) ([]SavedItem, error)
}

// DailyDietRepository defines user-scoped saved daily-diet persistence.
// Implements DESIGN-008 SavedDataRepository.
type DailyDietRepository interface {
	Create(ctx context.Context, userID uuid.UUID, diet SavedDiet) (uuid.UUID, error)
	Get(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) (SavedDiet, error)
	List(ctx context.Context, userID uuid.UUID) ([]SavedDiet, error)
	Replace(ctx context.Context, userID uuid.UUID, diet SavedDiet) error
	Delete(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) error
}

// DailyDietMutationRepository adds atomic create/idempotency and ownership-aware delete behavior.
// Implements DESIGN-008 SavedDataRepository and ProfileController.
type DailyDietMutationRepository interface {
	DailyDietRepository
	GetDailyDietCreateClaim(ctx context.Context, userID uuid.UUID, key string, bodyHash string) (DailyDietCreateClaimResult, error)
	ClaimDailyDietCreate(ctx context.Context, claim DailyDietCreateClaim) (DailyDietCreateClaimResult, error)
	DeleteIfOwned(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) (deleted bool, exists bool, err error)
}

// SearchHistoryRepository defines search-history persistence behavior.
// Implements DESIGN-008 SearchHistoryRepository.
type SearchHistoryRepository interface {
	AddHistory(ctx context.Context, entry SearchHistoryEntry) (uuid.UUID, error)
	ListHistory(ctx context.Context, userID uuid.UUID, limit int) ([]SearchHistoryEntry, error)
	ClearHistory(ctx context.Context, userID uuid.UUID) error
}

// EncryptedSearchHistoryRepository defines encrypted history query persistence behavior.
// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService.
type EncryptedSearchHistoryRepository interface {
	AddEncryptedHistory(ctx context.Context, entry EncryptedSearchHistoryEntry) (uuid.UUID, error)
	ListEncryptedHistory(ctx context.Context, userID uuid.UUID, limit int) ([]EncryptedSearchHistoryEntry, error)
}

// AuthUserRepository defines Phase 03-facing user identity persistence behavior.
// Implements DESIGN-006 AuthController.
type AuthUserRepository interface {
	CreateUser(ctx context.Context, user AuthUser) (uuid.UUID, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (AuthUser, error)
	GetUserByNormalizedEmail(ctx context.Context, normalizedEmail string) (AuthUser, error)
	UpdateUserState(ctx context.Context, user AuthUser) error
}

// AccountDeletionRepository defines production account row deletion behavior.
// Implements DESIGN-008 AccountDeleter.
type AccountDeletionRepository interface {
	DeleteUserAccount(ctx context.Context, userID uuid.UUID) error
}

// OAuthIdentityRepository defines Phase 03-facing OAuth identity persistence behavior.
// Implements DESIGN-006 AuthController.
type OAuthIdentityRepository interface {
	UpsertOAuthIdentity(ctx context.Context, identity OAuthIdentity) (uuid.UUID, error)
	GetOAuthIdentity(ctx context.Context, provider string, providerUserID string) (OAuthIdentity, error)
}

// SessionRepository defines Phase 03-facing session persistence behavior.
// Implements DESIGN-006 AuthController.
type SessionRepository interface {
	CreateSession(ctx context.Context, session UserSession) (uuid.UUID, error)
	GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (UserSession, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeSessionFamily(ctx context.Context, refreshFamilyID uuid.UUID) error
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
}

// PasswordResetTokenRepository defines Phase 03-facing reset-token persistence behavior.
// Implements DESIGN-006 AuthController.
type PasswordResetTokenRepository interface {
	CreatePasswordResetToken(ctx context.Context, token PasswordResetToken) error
	ConsumePasswordResetToken(ctx context.Context, tokenHash string, usedAt time.Time) (PasswordResetToken, error)
}

// AccountVerificationRepository defines verification and password mutation behavior.
// Implements DESIGN-006 AuthController.
type AccountVerificationRepository interface {
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, passwordSalt string) error
}

// AccountLockoutRepository defines failed-login persistence behavior.
// Implements DESIGN-006 AccountLockoutTracker.
type AccountLockoutRepository interface {
	GetLockoutState(ctx context.Context, userID uuid.UUID) (AccountLockoutState, error)
	RecordFailedLogin(ctx context.Context, userID uuid.UUID, threshold int, lockedUntil time.Time, now time.Time) (AccountLockoutState, error)
	ResetFailedLogins(ctx context.Context, userID uuid.UUID) (AccountLockoutState, error)
}

// RegistrationRepository defines transactional account creation with consent.
// Implements DESIGN-015 ConsentManager.
type RegistrationRepository interface {
	CreateUserWithConsent(ctx context.Context, user EncryptedAuthUser, privacyVersion string, termsVersion string) (uuid.UUID, error)
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
	RecordUsageWithinLimit(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time, since time.Time, limit int) (UsageWindow, bool, error)
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
	ProcessStripeWebhookEvent(ctx context.Context, event ProcessedStripeEvent, entitlement *Entitlement) (bool, error)
	InsertStripeDeadLetter(ctx context.Context, entry StripeDeadLetter) error
}

// CheckoutIdempotencyRepository defines checkout creation idempotency persistence.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
type CheckoutIdempotencyRepository interface {
	GetCheckoutIdempotency(ctx context.Context, userID uuid.UUID, method string, route string, key string) (CheckoutIdempotencyRecord, error)
	StoreCheckoutIdempotency(ctx context.Context, record CheckoutIdempotencyRecord) error
}

// ConsentRepository defines consent persistence behavior.
// Implements DESIGN-015 ConsentManager.
type ConsentRepository interface {
	RecordConsent(ctx context.Context, record ConsentRecord) (uuid.UUID, error)
	HasRequiredConsent(ctx context.Context, userID uuid.UUID, privacyVersion string, termsVersion string) (bool, error)
	ListConsent(ctx context.Context, userID uuid.UUID) ([]ConsentRecord, error)
}

// DeletionRequestRepository defines account-deletion workflow persistence behavior.
// Implements DESIGN-015 DataRetentionPolicy.
type DeletionRequestRepository interface {
	RequestDeletion(ctx context.Context, userID uuid.UUID) (DataDeletionRequest, error)
	UpdateDeletionStatus(ctx context.Context, requestID uuid.UUID, status string, note string) error
	ListDeletionAudit(ctx context.Context, requestID uuid.UUID) ([]DataDeletionAuditEntry, error)
	ClaimDeletionRequests(ctx context.Context, now time.Time, limit int) ([]DataDeletionRequest, error)
	RecordDeletionFailure(ctx context.Context, requestID uuid.UUID, category string, note string, nextAttemptAt *time.Time) error
	CompleteDeletionRequest(ctx context.Context, requestID uuid.UUID, receiptID uuid.UUID, completedAt time.Time) error
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

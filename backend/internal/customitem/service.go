// Package customitem owns authenticated user-created food-item behavior.
package customitem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-008 ProfileController custom-item creation errors.
var (
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different body")
)

// Request contains only client-editable custom-item fields.
// Implements DESIGN-008 ProfileController custom-item mutation contract.
type Request struct {
	Name                            string                   `json:"name"`
	PhysicalState                   repository.PhysicalState `json:"physicalState"`
	PrepTimeMinutes                 int                      `json:"prepTimeMinutes"`
	AverageUnitWeightGrams          float64                  `json:"averageUnitWeightGrams,omitempty"`
	AverageServingVolumeMilliliters float64                  `json:"averageServingVolumeMilliliters,omitempty"`
	DensityGramsPerMilliliter       float64                  `json:"densityGramsPerMilliliter,omitempty"`
	DensitySourceProvider           string                   `json:"densitySourceProvider,omitempty"`
	DensitySourceFoodID             string                   `json:"densitySourceFoodId,omitempty"`
	DensitySourceKind               string                   `json:"densitySourceKind,omitempty"`
	MacrosPer100                    repository.MacroValues   `json:"macrosPer100"`
	Micros                          repository.MicroValues   `json:"micros"`
	FoodCategoryIDs                 []uuid.UUID              `json:"foodCategoryIds"`
	CulinaryRoleIDs                 []uuid.UUID              `json:"culinaryRoleIds"`
	ImageURL                        string                   `json:"imageUrl,omitempty"`
}

// CreateRequest carries the cross-phase idempotency scope beside the normalized body.
// Implements DESIGN-008 ProfileController custom-item creation.
type CreateRequest struct {
	Request
	IdempotencyKey string
}

// ClassificationSummary is the hierarchy-free public classification projection.
// Implements DESIGN-008 ProfileController and DataExporter classification contract.
type ClassificationSummary struct {
	ID   uuid.UUID                     `json:"id"`
	Name string                        `json:"name"`
	Kind repository.ClassificationKind `json:"kind"`
}

// Item is the API/export-safe custom-item projection without ownership or hierarchy metadata.
// Implements DESIGN-008 ProfileController and DataExporter.
type Item struct {
	ID                              uuid.UUID                `json:"id"`
	Name                            string                   `json:"name"`
	PhysicalState                   repository.PhysicalState `json:"physicalState"`
	PrepTimeMinutes                 int                      `json:"prepTimeMinutes"`
	AverageUnitWeightGrams          float64                  `json:"averageUnitWeightGrams,omitempty"`
	AverageServingVolumeMilliliters float64                  `json:"averageServingVolumeMilliliters,omitempty"`
	DensityGramsPerMilliliter       float64                  `json:"densityGramsPerMilliliter,omitempty"`
	DensitySourceProvider           string                   `json:"densitySourceProvider,omitempty"`
	DensitySourceFoodID             string                   `json:"densitySourceFoodId,omitempty"`
	DensitySourceKind               string                   `json:"densitySourceKind,omitempty"`
	MacrosPer100                    repository.MacroValues   `json:"macrosPer100"`
	Micros                          repository.MicroValues   `json:"micros"`
	FoodCategories                  []ClassificationSummary  `json:"foodCategories"`
	CulinaryRoles                   []ClassificationSummary  `json:"culinaryRoles"`
	ImageURL                        string                   `json:"imageUrl,omitempty"`
}

// CreateResult carries a created or replayed custom item.
// Implements DESIGN-008 ProfileController custom-item creation.
type CreateResult struct {
	Item     Item
	Status   int
	Replayed bool
}

// Service coordinates owner-scoped persistence and create idempotency.
// Implements DESIGN-008 ProfileController custom-item behavior.
type Service struct {
	items     repository.CustomFoodItemRepository
	telemetry *observability.AdminExternalTelemetry
}

// WithTelemetry adds owner-free lifecycle outcome observations.
// Implements DESIGN-014 MetricsCollector.
func (s *Service) WithTelemetry(telemetry *observability.AdminExternalTelemetry) *Service {
	if s != nil {
		s.telemetry = telemetry
	}
	return s
}

// NewService creates authenticated custom-item behavior.
// Implements DESIGN-008 ProfileController custom-item behavior.
func NewService(items repository.CustomFoodItemRepository) *Service {
	return &Service{items: items}
}

// Create persists or replays one owner-scoped custom item.
// Implements DESIGN-008 ProfileController custom-item creation.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (result CreateResult, err error) {
	defer func() { s.recordLifecycle(ctx, "create", lifecycleOutcome(result.Replayed, err)) }()
	if userID == uuid.Nil {
		return CreateResult{}, validationError("user id is required")
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if len(key) < 8 || len(key) > 255 || strings.ContainsRune(key, '\x00') {
		return CreateResult{}, ErrMissingIdempotencyKey
	}
	normalized, err := ValidateRequest(req.Request)
	if err != nil {
		return CreateResult{}, err
	}
	if s == nil || s.items == nil {
		return CreateResult{}, repository.NewError(repository.ErrorKindConnection, "custom item service is unavailable", nil)
	}
	bodyHash, err := requestHash(normalized)
	if err != nil {
		return CreateResult{}, err
	}
	claimResult, err := s.items.ClaimCreate(ctx, repository.CustomFoodItemCreateClaim{
		UserID: userID, Key: key, BodyHash: bodyHash, Item: toEntity(userID, uuid.Nil, normalized),
	}, func(entity repository.CustomFoodItemEntity) ([]byte, error) {
		return json.Marshal(fromEntity(entity))
	})
	if err != nil {
		if repository.IsKind(err, repository.ErrorKindIdempotencyConflict) {
			return CreateResult{}, ErrIdempotencyConflict
		}
		return CreateResult{}, err
	}
	return createResultFromClaim(claimResult)
}

// Get loads one item only for its authenticated owner.
// Implements DESIGN-008 ProfileController custom-item read.
func (s *Service) Get(ctx context.Context, userID, itemID uuid.UUID) (item Item, err error) {
	defer func() { s.recordLifecycle(ctx, "get", lifecycleOutcome(false, err)) }()
	if err := validateIdentity(userID, itemID); err != nil {
		return Item{}, err
	}
	if s == nil || s.items == nil {
		return Item{}, repository.NewError(repository.ErrorKindConnection, "custom item service is unavailable", nil)
	}
	entity, err := s.items.GetByID(ctx, userID, itemID, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
	if err != nil {
		return Item{}, err
	}
	return fromEntity(entity), nil
}

// Update replaces one item only for its authenticated owner.
// Implements DESIGN-008 ProfileController custom-item update.
func (s *Service) Update(ctx context.Context, userID, itemID uuid.UUID, req Request) (item Item, err error) {
	defer func() { s.recordLifecycle(ctx, "update", lifecycleOutcome(false, err)) }()
	if err := validateIdentity(userID, itemID); err != nil {
		return Item{}, err
	}
	if s == nil || s.items == nil {
		return Item{}, repository.NewError(repository.ErrorKindConnection, "custom item service is unavailable", nil)
	}
	normalized, err := ValidateRequest(req)
	if err != nil {
		return Item{}, err
	}
	if err := s.items.Update(ctx, toEntity(userID, itemID, normalized)); err != nil {
		return Item{}, err
	}
	return s.Get(ctx, userID, itemID)
}

// Delete soft-deletes one item only for its authenticated owner.
// Implements DESIGN-008 ProfileController custom-item delete.
func (s *Service) Delete(ctx context.Context, userID, itemID uuid.UUID) (err error) {
	defer func() { s.recordLifecycle(ctx, "delete", lifecycleOutcome(false, err)) }()
	if err := validateIdentity(userID, itemID); err != nil {
		return err
	}
	if s == nil || s.items == nil {
		return repository.NewError(repository.ErrorKindConnection, "custom item service is unavailable", nil)
	}
	return s.items.Delete(ctx, userID, itemID)
}

// List loads all active items for one owner.
// Implements DESIGN-008 DataExporter owner-scoped custom-item export.
func (s *Service) List(ctx context.Context, userID uuid.UUID) (result []Item, err error) {
	defer func() { s.recordLifecycle(ctx, "list", lifecycleOutcome(false, err)) }()
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	if s == nil || s.items == nil {
		return nil, repository.NewError(repository.ErrorKindConnection, "custom item service is unavailable", nil)
	}
	entities, err := s.items.List(ctx, userID, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(entities))
	for _, entity := range entities {
		items = append(items, fromEntity(entity))
	}
	return items, nil
}

// recordLifecycle emits one owner-free bounded lifecycle outcome.
// Implements DESIGN-014 MetricsCollector.
func (s *Service) recordLifecycle(ctx context.Context, operation, outcome string) {
	if s != nil {
		s.telemetry.CustomItemLifecycle(ctx, operation, outcome)
	}
}

// lifecycleOutcome maps service errors to the closed observability vocabulary.
// Implements DESIGN-014 MetricsCollector.
func lifecycleOutcome(replayed bool, err error) string {
	if err == nil {
		if replayed {
			return "replayed"
		}
		return "succeeded"
	}
	switch {
	case errors.Is(err, ErrMissingIdempotencyKey), repository.IsKind(err, repository.ErrorKindValidation):
		return "validation_failed"
	case errors.Is(err, ErrIdempotencyConflict), repository.IsKind(err, repository.ErrorKindIdempotencyConflict), repository.IsKind(err, repository.ErrorKindConflict):
		return "conflict"
	case repository.IsKind(err, repository.ErrorKindNotFound):
		return "not_found"
	case repository.IsKind(err, repository.ErrorKindConnection):
		return "dependency_failed"
	default:
		return "error"
	}
}

// createResultFromClaim decodes the repository's immutable owner-free response.
// Implements DESIGN-008 ProfileController custom-item idempotency replay.
func createResultFromClaim(claim repository.CustomFoodItemCreateClaimResult) (CreateResult, error) {
	var item Item
	if err := json.Unmarshal(claim.ResponseBody, &item); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Item: item, Status: claim.StatusCode, Replayed: claim.Replayed}, nil
}

// ValidateRequest validates and canonicalizes client-editable fields before mutation.
// Implements DESIGN-008 ProfileController custom-item request normalization.
func ValidateRequest(req Request) (Request, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.DensitySourceProvider = strings.TrimSpace(req.DensitySourceProvider)
	req.DensitySourceFoodID = strings.TrimSpace(req.DensitySourceFoodID)
	req.DensitySourceKind = strings.TrimSpace(req.DensitySourceKind)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	if !validText(req.Name, 200, true) {
		return Request{}, validationError("custom item name is invalid")
	}
	if req.PhysicalState != repository.PhysicalStateSolid && req.PhysicalState != repository.PhysicalStateLiquid {
		return Request{}, validationError("custom item physical state is invalid")
	}
	if req.PrepTimeMinutes < 0 || !validNonnegative(req.MacrosPer100.Protein) || !validNonnegative(req.MacrosPer100.Carbohydrates) || !validNonnegative(req.MacrosPer100.Fat) {
		return Request{}, validationError("custom item nutrition values are invalid")
	}
	if req.PhysicalState == repository.PhysicalStateSolid && req.MacrosPer100.Protein+req.MacrosPer100.Carbohydrates+req.MacrosPer100.Fat > 100 {
		return Request{}, validationError("solid macro values per 100 g cannot exceed 100 g")
	}
	if !validOptionalPositive(req.AverageUnitWeightGrams) || !validOptionalPositive(req.AverageServingVolumeMilliliters) || !validOptionalPositive(req.DensityGramsPerMilliliter) {
		return Request{}, validationError("custom item physical measures are invalid")
	}
	if !validText(req.DensitySourceProvider, 200, false) || !validText(req.DensitySourceFoodID, 200, false) || !validText(req.DensitySourceKind, 20, false) || !validText(req.ImageURL, 2048, false) {
		return Request{}, validationError("custom item text field is too long")
	}
	if req.ImageURL != "" {
		parsed, err := url.ParseRequestURI(req.ImageURL)
		if err != nil || (parsed.Scheme != "" && !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
			return Request{}, validationError("custom item image url is invalid")
		}
	}
	if err := validateDensity(req); err != nil {
		return Request{}, err
	}
	if len(req.FoodCategoryIDs) > 100 || len(req.CulinaryRoleIDs) > 100 {
		return Request{}, validationError("too many custom item classifications")
	}
	for _, ids := range [][]uuid.UUID{req.FoodCategoryIDs, req.CulinaryRoleIDs} {
		if slices.Contains(ids, uuid.Nil) {
			return Request{}, validationError("custom item classification id is invalid")
		}
	}
	for key, value := range req.Micros {
		if !validText(strings.TrimSpace(key), 120, true) || !validNonnegative(value) {
			return Request{}, validationError("custom item micronutrients are invalid")
		}
	}
	req.FoodCategoryIDs = normalizedIDs(req.FoodCategoryIDs)
	req.CulinaryRoleIDs = normalizedIDs(req.CulinaryRoleIDs)
	if req.FoodCategoryIDs == nil {
		req.FoodCategoryIDs = []uuid.UUID{}
	}
	if req.CulinaryRoleIDs == nil {
		req.CulinaryRoleIDs = []uuid.UUID{}
	}
	if req.Micros == nil {
		req.Micros = repository.MicroValues{}
	}
	return req, nil
}

// validNonnegative reports whether a client number is finite and nonnegative.
// Implements DESIGN-008 ProfileController custom-item request validation.
func validNonnegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

// validOptionalPositive validates zero-as-absent physical measures.
// Implements DESIGN-008 ProfileController custom-item request validation.
func validOptionalPositive(value float64) bool {
	return value == 0 || validNonnegative(value) && value > 0
}

// validText rejects PostgreSQL-incompatible NULs and enforces character limits.
// Implements DESIGN-008 ProfileController custom-item request validation.
func validText(value string, maxRunes int, required bool) bool {
	return (!required || value != "") && !strings.ContainsRune(value, '\x00') && len([]rune(value)) <= maxRunes
}

// validateDensity enforces solid/liquid density and provenance invariants.
// Implements DESIGN-005 FoodItemEntity liquid-density validation.
func validateDensity(req Request) error {
	if req.PhysicalState == repository.PhysicalStateSolid {
		if req.AverageServingVolumeMilliliters != 0 || req.DensityGramsPerMilliliter != 0 || req.DensitySourceProvider != "" || req.DensitySourceFoodID != "" || req.DensitySourceKind != "" {
			return validationError("solid custom items cannot contain liquid density fields")
		}
		return nil
	}
	if req.DensityGramsPerMilliliter <= 0 || (req.DensitySourceKind != "imported" && req.DensitySourceKind != "manual" && req.DensitySourceKind != "estimated") {
		return validationError("liquid custom item density is invalid")
	}
	if req.DensitySourceKind == "imported" && ((req.DensitySourceProvider != "usda" && req.DensitySourceProvider != "openfoodfacts") || req.DensitySourceFoodID == "") {
		return validationError("imported liquid density requires trusted provider evidence")
	}
	return nil
}

// normalizedIDs returns sorted, de-duplicated classification identifiers.
// Implements DESIGN-008 ProfileController custom-item request normalization.
func normalizedIDs(ids []uuid.UUID) []uuid.UUID {
	result := slices.Clone(ids)
	slices.SortFunc(result, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(result)
}

// requestHash produces the stable normalized body identity used for idempotency.
// Implements DESIGN-008 ProfileController custom-item idempotency.
func requestHash(req Request) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// validateIdentity rejects ownerless or unidentified service operations.
// Implements DESIGN-008 ProfileController server-derived ownership.
func validateIdentity(userID, itemID uuid.UUID) error {
	if userID == uuid.Nil || itemID == uuid.Nil {
		return validationError("custom item identity is required")
	}
	return nil
}

// validationError creates a repository-compatible validation classification.
// Implements DESIGN-008 ProfileController structured error mapping.
func validationError(message string) error {
	return repository.NewError(repository.ErrorKindValidation, message, nil)
}

// toEntity adds the server-derived owner to client-editable fields.
// Implements DESIGN-008 ProfileController server-derived ownership.
func toEntity(ownerID, itemID uuid.UUID, req Request) repository.CustomFoodItemEntity {
	classifications := func(ids []uuid.UUID, kind repository.ClassificationKind) []repository.ClassificationEntity {
		result := make([]repository.ClassificationEntity, 0, len(ids))
		for _, id := range ids {
			result = append(result, repository.ClassificationEntity{ID: id, Kind: kind})
		}
		return result
	}
	return repository.CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: repository.FoodItemEntity{
		ID: itemID, Name: req.Name, PhysicalState: req.PhysicalState, PrepTimeMinutes: req.PrepTimeMinutes,
		AverageUnitWeightGrams: req.AverageUnitWeightGrams, AverageServingVolumeMilliliters: req.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: req.DensityGramsPerMilliliter, DensitySourceProvider: req.DensitySourceProvider,
		DensitySourceFoodID: req.DensitySourceFoodID, DensitySourceKind: req.DensitySourceKind, MacrosPer100: req.MacrosPer100,
		Micros: req.Micros, FoodCategories: classifications(req.FoodCategoryIDs, repository.ClassificationKindFoodCategory),
		CulinaryRoles: classifications(req.CulinaryRoleIDs, repository.ClassificationKindCulinaryRole), ImageURL: req.ImageURL,
	}}
}

// fromEntity removes ownership metadata from the API/export projection.
// Implements DESIGN-008 ProfileController and DataExporter private-item projection.
func fromEntity(entity repository.CustomFoodItemEntity) Item {
	foodCategories := classificationSummaries(entity.FoodCategories)
	culinaryRoles := classificationSummaries(entity.CulinaryRoles)
	micros := entity.Micros
	if micros == nil {
		micros = repository.MicroValues{}
	}
	return Item{
		ID: entity.ID, Name: entity.Name, PhysicalState: entity.PhysicalState, PrepTimeMinutes: entity.PrepTimeMinutes,
		AverageUnitWeightGrams: entity.AverageUnitWeightGrams, AverageServingVolumeMilliliters: entity.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: entity.DensityGramsPerMilliliter, DensitySourceProvider: entity.DensitySourceProvider,
		DensitySourceFoodID: entity.DensitySourceFoodID, DensitySourceKind: entity.DensitySourceKind, MacrosPer100: entity.MacrosPer100,
		Micros: micros, FoodCategories: foodCategories, CulinaryRoles: culinaryRoles, ImageURL: entity.ImageURL,
	}
}

// classificationSummaries strips repository-only hierarchy metadata at the public boundary.
// Implements DESIGN-008 ProfileController and DataExporter classification contract.
func classificationSummaries(entities []repository.ClassificationEntity) []ClassificationSummary {
	result := make([]ClassificationSummary, 0, len(entities))
	for _, entity := range entities {
		result = append(result, ClassificationSummary{ID: entity.ID, Name: entity.Name, Kind: entity.Kind})
	}
	return result
}

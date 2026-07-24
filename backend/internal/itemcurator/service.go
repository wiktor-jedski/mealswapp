// Package itemcurator owns administrator-authored global food-item behavior.
package itemcurator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 ItemCurator idempotency errors.
var (
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different body")
)

// Request contains administrator-editable global food-item fields.
// Implements DESIGN-009 ItemCurator request boundary.
type Request = customitem.Request

// ClassificationSummary is the hierarchy-free administration projection.
// Implements DESIGN-009 ItemCurator response boundary.
type ClassificationSummary struct {
	ID   uuid.UUID                     `json:"id"`
	Name string                        `json:"name"`
	Kind repository.ClassificationKind `json:"kind"`
}

// Item is the ownerless global-item administration projection.
// Implements DESIGN-009 ItemCurator global/private separation.
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

// CreateResult carries a newly created or replayed global item.
// Implements DESIGN-009 ItemCurator idempotent create.
type CreateResult struct {
	Item     Item
	Status   int
	Replayed bool
}

// MutationResult carries authoritative before/after state for auditing.
// Implements DESIGN-009 ItemCurator audit snapshots.
type MutationResult struct {
	Before Item
	After  Item
}

// Store is the global-only persistence boundary used by ItemCurator.
// Implements DESIGN-009 ItemCurator global/private separation.
type Store interface {
	GetByID(context.Context, uuid.UUID, bool) (repository.FoodItemEntity, error)
	GetByIDInMutation(context.Context, repository.AdminMutationExecutor, uuid.UUID, bool) (repository.FoodItemEntity, error)
	ClaimCreate(context.Context, repository.AdminMutationExecutor, repository.ManualFoodItemCreateClaim, repository.ManualFoodItemResponseEncoder) (repository.ManualFoodItemCreateClaimResult, error)
	Update(context.Context, repository.AdminMutationExecutor, repository.FoodItemEntity) error
	Delete(context.Context, repository.AdminMutationExecutor, uuid.UUID) error
}

// Service coordinates global-item validation, idempotency, and audit state.
// Implements DESIGN-009 ItemCurator.
type Service struct {
	items Store
}

// NewService creates manual global-item behavior.
// Implements DESIGN-009 ItemCurator.
func NewService(items Store) *Service { return &Service{items: items} }

// Create persists or replays one ownerless global item in the audit transaction.
// Implements DESIGN-009 ItemCurator idempotent create.
func (s *Service) Create(ctx context.Context, tx repository.AdminMutationExecutor, adminID uuid.UUID, key string, req Request) (CreateResult, error) {
	if adminID == uuid.Nil {
		return CreateResult{}, validationError("admin user id is required")
	}
	key = strings.TrimSpace(key)
	if len(key) < 8 || len(key) > 255 || strings.ContainsRune(key, '\x00') {
		return CreateResult{}, ErrMissingIdempotencyKey
	}
	normalized, err := customitem.ValidateRequest(req)
	if err != nil {
		return CreateResult{}, err
	}
	if s == nil || s.items == nil || tx == nil {
		return CreateResult{}, repository.NewError(repository.ErrorKindConnection, "manual item service is unavailable", nil)
	}
	bodyHash, err := requestHash(normalized)
	if err != nil {
		return CreateResult{}, err
	}
	claim, err := s.items.ClaimCreate(ctx, tx, repository.ManualFoodItemCreateClaim{AdminUserID: adminID, Key: key, BodyHash: bodyHash, Item: toEntity(uuid.Nil, normalized)}, func(entity repository.FoodItemEntity) ([]byte, error) {
		return json.Marshal(fromEntity(entity))
	})
	if err != nil {
		if repository.IsKind(err, repository.ErrorKindIdempotencyConflict) {
			return CreateResult{}, ErrIdempotencyConflict
		}
		return CreateResult{}, err
	}
	var item Item
	if err := json.Unmarshal(claim.ResponseBody, &item); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Item: item, Status: claim.StatusCode, Replayed: claim.Replayed}, nil
}

// Get loads one active global item.
// Implements DESIGN-009 ItemCurator read behavior.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Item, error) {
	if id == uuid.Nil {
		return Item{}, validationError("food item id is required")
	}
	if s == nil || s.items == nil {
		return Item{}, repository.NewError(repository.ErrorKindConnection, "manual item service is unavailable", nil)
	}
	item, err := s.items.GetByID(ctx, id, false)
	if err != nil {
		return Item{}, err
	}
	return fromEntity(item), nil
}

// Update replaces one active global item and returns authoritative audit state.
// Implements DESIGN-009 ItemCurator update behavior.
func (s *Service) Update(ctx context.Context, tx repository.AdminMutationExecutor, id uuid.UUID, req Request) (MutationResult, error) {
	if id == uuid.Nil {
		return MutationResult{}, validationError("food item id is required")
	}
	normalized, err := customitem.ValidateRequest(req)
	if err != nil {
		return MutationResult{}, err
	}
	if s == nil || s.items == nil || tx == nil {
		return MutationResult{}, repository.NewError(repository.ErrorKindConnection, "manual item service is unavailable", nil)
	}
	before, err := s.items.GetByIDInMutation(ctx, tx, id, false)
	if err != nil {
		return MutationResult{}, err
	}
	if err := s.items.Update(ctx, tx, toEntity(id, normalized)); err != nil {
		return MutationResult{}, err
	}
	after, err := s.items.GetByIDInMutation(ctx, tx, id, false)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Before: fromEntity(before), After: fromEntity(after)}, nil
}

// Delete soft-deletes one active global item and returns authoritative audit state.
// Implements DESIGN-009 ItemCurator soft-delete behavior.
func (s *Service) Delete(ctx context.Context, tx repository.AdminMutationExecutor, id uuid.UUID) (MutationResult, error) {
	if id == uuid.Nil {
		return MutationResult{}, validationError("food item id is required")
	}
	if s == nil || s.items == nil || tx == nil {
		return MutationResult{}, repository.NewError(repository.ErrorKindConnection, "manual item service is unavailable", nil)
	}
	before, err := s.items.GetByIDInMutation(ctx, tx, id, false)
	if err != nil {
		return MutationResult{}, err
	}
	if err := s.items.Delete(ctx, tx, id); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Before: fromEntity(before)}, nil
}

// requestHash produces stable normalized body identity for retries.
// Implements DESIGN-009 ItemCurator idempotent create.
func requestHash(req Request) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// toEntity maps validated administrator input to the ownerless global model.
// Implements DESIGN-009 ItemCurator global/private separation.
func toEntity(id uuid.UUID, req Request) repository.FoodItemEntity {
	classifications := func(ids []uuid.UUID, kind repository.ClassificationKind) []repository.ClassificationEntity {
		result := make([]repository.ClassificationEntity, 0, len(ids))
		for _, classificationID := range ids {
			result = append(result, repository.ClassificationEntity{ID: classificationID, Kind: kind})
		}
		return result
	}
	return repository.FoodItemEntity{
		ID: id, Name: req.Name, PhysicalState: req.PhysicalState, PrepTimeMinutes: req.PrepTimeMinutes,
		AverageUnitWeightGrams: req.AverageUnitWeightGrams, AverageServingVolumeMilliliters: req.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: req.DensityGramsPerMilliliter, DensitySourceProvider: req.DensitySourceProvider,
		DensitySourceFoodID: req.DensitySourceFoodID, DensitySourceKind: req.DensitySourceKind, MacrosPer100: req.MacrosPer100,
		Micros: req.Micros, FoodCategories: classifications(req.FoodCategoryIDs, repository.ClassificationKindFoodCategory),
		CulinaryRoles: classifications(req.CulinaryRoleIDs, repository.ClassificationKindCulinaryRole), ImageURL: req.ImageURL,
	}
}

// fromEntity maps global persistence state to an owner-free administration response.
// Implements DESIGN-009 ItemCurator global/private separation.
func fromEntity(entity repository.FoodItemEntity) Item {
	classifications := func(values []repository.ClassificationEntity) []ClassificationSummary {
		result := make([]ClassificationSummary, 0, len(values))
		for _, value := range values {
			result = append(result, ClassificationSummary{ID: value.ID, Name: value.Name, Kind: value.Kind})
		}
		return result
	}
	micros := entity.Micros
	if micros == nil {
		micros = repository.MicroValues{}
	}
	return Item{
		ID: entity.ID, Name: entity.Name, PhysicalState: entity.PhysicalState, PrepTimeMinutes: entity.PrepTimeMinutes,
		AverageUnitWeightGrams: entity.AverageUnitWeightGrams, AverageServingVolumeMilliliters: entity.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: entity.DensityGramsPerMilliliter, DensitySourceProvider: entity.DensitySourceProvider,
		DensitySourceFoodID: entity.DensitySourceFoodID, DensitySourceKind: entity.DensitySourceKind, MacrosPer100: entity.MacrosPer100,
		Micros: micros, FoodCategories: classifications(entity.FoodCategories), CulinaryRoles: classifications(entity.CulinaryRoles), ImageURL: entity.ImageURL,
	}
}

// validationError creates a repository-compatible validation classification.
// Implements DESIGN-009 ItemCurator structured error mapping.
func validationError(message string) error {
	return repository.NewError(repository.ErrorKindValidation, message, nil)
}

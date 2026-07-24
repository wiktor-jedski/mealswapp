// Package dataimporter owns administrator-confirmed external food imports.
package dataimporter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/curation"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 DataImporter stable conflict outcomes.
var (
	ErrMissingIdempotencyKey = errors.New("idempotency key is required when provider identity is absent")
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different body")
	ErrProviderConflict      = errors.New("provider identity conflicts with an existing import")
	ErrNameConfirmation      = errors.New("normalized name conflict requires explicit confirmation")
)

// Request is the editable curated draft plus confirmation metadata.
// Implements DESIGN-009 DataImporter CuratedItemDraft.
type Request struct {
	SourceProvider      string `json:"sourceProvider,omitempty"`
	ExternalID          string `json:"externalId,omitempty"`
	ConfirmNameConflict bool   `json:"confirmNameConflict,omitempty"`
	customitem.Request
}

// Result identifies the durable import and immediately searchable global food item.
// Implements DESIGN-009 DataImporter confirmation result.
type Result struct {
	ImportID      uuid.UUID                `json:"importId"`
	FoodItemID    uuid.UUID                `json:"foodItemId"`
	Name          string                   `json:"name"`
	PhysicalState repository.PhysicalState `json:"physicalState"`
	Merged        bool                     `json:"merged"`
	Replayed      bool                     `json:"replayed"`
}

// Store persists confirmation state in the admin gateway transaction.
// Implements DESIGN-009 DataImporter persistence boundary.
type Store interface {
	ConfirmCuratedImport(context.Context, repository.AdminMutationExecutor, repository.CuratedImportConfirmation) (repository.CuratedImportConfirmationResult, error)
}

// Service validates editable drafts and coordinates durable confirmation.
// Implements DESIGN-009 DataImporter.
type Service struct {
	store     Store
	telemetry *observability.AdminExternalTelemetry
}

// NewService creates curated-import behavior.
// Implements DESIGN-009 DataImporter.
func NewService(store Store) *Service { return &Service{store: store} }

// WithTelemetry adds bounded curated-import outcome observations.
// Implements DESIGN-014 MetricsCollector.
func (s *Service) WithTelemetry(telemetry *observability.AdminExternalTelemetry) *Service {
	if s != nil {
		s.telemetry = telemetry
	}
	return s
}

// Confirm validates, hashes, and persists or replays one curated draft.
// Implements DESIGN-009 DataImporter confirmation workflow.
func (s *Service) Confirm(ctx context.Context, tx repository.AdminMutationExecutor, adminID uuid.UUID, idempotencyKey string, req Request) (result Result, err error) {
	defer func() {
		if s != nil && err != nil {
			s.telemetry.ImportOutcome(ctx, importTelemetryProvider(req.SourceProvider), importTelemetryOutcome(result, err))
		}
	}()
	req, err = NormalizeRequest(ctx, req)
	if err != nil {
		return Result{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if req.SourceProvider == "" && (len(idempotencyKey) < 8 || len(idempotencyKey) > 255 || strings.ContainsRune(idempotencyKey, '\x00')) {
		return Result{}, ErrMissingIdempotencyKey
	}
	item, err := customitem.ValidateRequest(req.Request)
	if err != nil {
		return Result{}, err
	}
	if adminID == uuid.Nil || tx == nil || s == nil || s.store == nil {
		return Result{}, repository.NewError(repository.ErrorKindConnection, "curated import service is unavailable", nil)
	}
	req.Request = item
	bodyHash, err := requestHash(req)
	if err != nil {
		return Result{}, err
	}
	confirmed, err := s.store.ConfirmCuratedImport(ctx, tx, repository.CuratedImportConfirmation{
		AdminUserID: adminID, IdempotencyKey: idempotencyKey, BodyHash: bodyHash,
		SourceProvider: req.SourceProvider, ExternalID: req.ExternalID, ConfirmNameConflict: req.ConfirmNameConflict,
		Item: toEntity(item),
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrCuratedImportIdentityConflict):
			return Result{}, ErrProviderConflict
		case errors.Is(err, repository.ErrCuratedImportNameConfirmationRequired):
			return Result{}, ErrNameConfirmation
		case repository.IsKind(err, repository.ErrorKindIdempotencyConflict):
			return Result{}, ErrIdempotencyConflict
		default:
			return Result{}, err
		}
	}
	return Result{ImportID: confirmed.ImportID, FoodItemID: confirmed.Item.ID, Name: confirmed.Item.Name, PhysicalState: confirmed.Item.PhysicalState, Merged: confirmed.Merged, Replayed: confirmed.Replayed}, nil
}

// RecordCommittedOutcome emits a success only after mutation and audit commit together.
// Implements DESIGN-014 MetricsCollector and DESIGN-009 DataImporter fail-closed audit behavior.
func (s *Service) RecordCommittedOutcome(ctx context.Context, provider string, result Result) {
	if s != nil {
		s.telemetry.ImportOutcome(ctx, importTelemetryProvider(provider), importTelemetryOutcome(result, nil))
	}
}

// importTelemetryProvider maps source identity to a closed provider label.
// Implements DESIGN-014 MetricsCollector.
func importTelemetryProvider(provider string) string {
	if provider == "usda" || provider == "openfoodfacts" {
		return provider
	}
	return "manual"
}

// importTelemetryOutcome maps confirmation state to a closed outcome label.
// Implements DESIGN-014 MetricsCollector.
func importTelemetryOutcome(result Result, err error) string {
	if err == nil {
		if result.Replayed {
			return "replayed"
		}
		if result.Merged {
			return "merged"
		}
		return "created"
	}
	switch {
	case errors.Is(err, ErrMissingIdempotencyKey), repository.IsKind(err, repository.ErrorKindValidation):
		return "validation_failed"
	case errors.Is(err, ErrIdempotencyConflict):
		return "idempotency_conflict"
	case errors.Is(err, ErrProviderConflict):
		return "provider_conflict"
	case errors.Is(err, ErrNameConfirmation):
		return "name_conflict"
	case repository.IsKind(err, repository.ErrorKindConnection):
		return "dependency_failed"
	default:
		return "error"
	}
}

// NormalizeRequest applies the typed curation trust boundary and returns only canonical values.
// Implements DESIGN-009 DataImporter and DESIGN-013 InputNormalizer.
func NormalizeRequest(ctx context.Context, req Request) (Request, error) {
	if !validMicronutrientValues(req.Micros) {
		return Request{}, validationError("curated item is invalid")
	}
	normalized, err := curation.NewInputNormalizer(nil).NormalizeItem(ctx, curation.ItemRequest{
		Name: req.Name, PhysicalState: req.PhysicalState, PrepTimeMinutes: req.PrepTimeMinutes,
		AverageUnitWeightGrams: req.AverageUnitWeightGrams, AverageServingVolumeMilliliters: req.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: req.DensityGramsPerMilliliter, DensitySourceProvider: req.DensitySourceProvider,
		DensitySourceFoodID: req.DensitySourceFoodID, DensitySourceKind: req.DensitySourceKind, ImageURL: req.ImageURL,
		SourceProvider: req.SourceProvider, ExternalID: req.ExternalID, MacrosPer100: req.MacrosPer100,
	})
	if err != nil {
		return Request{}, validationError("curated item is invalid")
	}
	req.Name, req.PhysicalState, req.PrepTimeMinutes = normalized.Name, normalized.PhysicalState, normalized.PrepTimeMinutes
	req.AverageUnitWeightGrams, req.AverageServingVolumeMilliliters = normalized.AverageUnitWeightGrams, normalized.AverageServingVolumeMilliliters
	req.DensityGramsPerMilliliter, req.DensitySourceProvider = normalized.DensityGramsPerMilliliter, normalized.DensitySourceProvider
	req.DensitySourceFoodID, req.DensitySourceKind, req.ImageURL = normalized.DensitySourceFoodID, normalized.DensitySourceKind, normalized.ImageURL
	req.SourceProvider, req.ExternalID, req.MacrosPer100 = normalized.SourceProvider, normalized.ExternalID, normalized.MacrosPer100
	return req, nil
}

// validMicronutrientValues applies curation bounds while preserving repository vocabulary keys.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-013 InputNormalizer.
func validMicronutrientValues(values repository.MicroValues) bool {
	if len(values) > 200 {
		return false
	}
	for _, value := range values {
		if value < 0 || value > curation.MaxCurationNutritionValue || math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

// requestHash creates stable identity from the normalized editable draft.
// Implements DESIGN-009 DataImporter exact replay.
func requestHash(req Request) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// toEntity maps the editable draft into the ownerless global food model.
// Implements DESIGN-009 DataImporter ARCH-005 persistence mapping.
func toEntity(req customitem.Request) repository.FoodItemEntity {
	classifications := func(ids []uuid.UUID, kind repository.ClassificationKind) []repository.ClassificationEntity {
		result := make([]repository.ClassificationEntity, 0, len(ids))
		for _, id := range ids {
			result = append(result, repository.ClassificationEntity{ID: id, Kind: kind})
		}
		return result
	}
	return repository.FoodItemEntity{
		Name: req.Name, PhysicalState: req.PhysicalState, PrepTimeMinutes: req.PrepTimeMinutes,
		AverageUnitWeightGrams: req.AverageUnitWeightGrams, AverageServingVolumeMilliliters: req.AverageServingVolumeMilliliters,
		DensityGramsPerMilliliter: req.DensityGramsPerMilliliter, DensitySourceProvider: req.DensitySourceProvider,
		DensitySourceFoodID: req.DensitySourceFoodID, DensitySourceKind: req.DensitySourceKind, MacrosPer100: req.MacrosPer100,
		Micros: req.Micros, FoodCategories: classifications(req.FoodCategoryIDs, repository.ClassificationKindFoodCategory),
		CulinaryRoles: classifications(req.CulinaryRoleIDs, repository.ClassificationKindCulinaryRole), ImageURL: req.ImageURL,
	}
}

// validationError returns a repository-compatible draft validation failure.
// Implements DESIGN-009 DataImporter structured validation.
func validationError(message string) error {
	return repository.NewError(repository.ErrorKindValidation, message, nil)
}

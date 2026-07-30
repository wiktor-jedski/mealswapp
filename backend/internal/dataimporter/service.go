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
	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 DataImporter stable conflict outcomes.
var (
	// ErrMissingIdempotencyKey indicates a provider-less import without an idempotency key.
	ErrMissingIdempotencyKey = errors.New("idempotency key is required when provider identity is absent")
	// ErrIdempotencyConflict indicates reuse of an idempotency key with a different request.
	ErrIdempotencyConflict = errors.New("idempotency key reused with different body")
	// ErrProviderConflict indicates a provider identity already belongs to another import.
	ErrProviderConflict = errors.New("provider identity conflicts with an existing import")
	// ErrNameConfirmation indicates a normalized-name conflict awaiting explicit confirmation.
	ErrNameConfirmation = errors.New("normalized name conflict requires explicit confirmation")
	// ErrExternalRecordEvidence indicates missing, malformed, stale, or mismatched server evidence.
	ErrExternalRecordEvidence = errors.New("external record evidence is invalid")
)

// Request is the editable curated draft plus confirmation metadata.
// Implements DESIGN-009 DataImporter CuratedItemDraft.
type Request struct {
	ExternalRecordToken string                    `json:"externalRecordToken,omitempty"`
	ConfirmNameConflict bool                      `json:"confirmNameConflict,omitempty"`
	SelectedRecord      providerregistry.Identity `json:"-"`
	customitem.Request
}

// Result identifies the durable import and immediately searchable global food item.
// Implements DESIGN-009 DataImporter confirmation result.
type Result struct {
	ImportID       uuid.UUID                `json:"importId"`
	FoodItemID     uuid.UUID                `json:"foodItemId"`
	Name           string                   `json:"name"`
	PhysicalState  repository.PhysicalState `json:"physicalState"`
	Merged         bool                     `json:"merged"`
	Replayed       bool                     `json:"replayed"`
	SourceProvider string                   `json:"-"`
}

// Store persists confirmation state in the admin gateway transaction.
// Implements DESIGN-009 DataImporter persistence boundary.
type Store interface {
	ConfirmCuratedImport(context.Context, repository.AdminMutationExecutor, repository.CuratedImportConfirmation) (repository.CuratedImportConfirmationResult, error)
}

// EvidenceResolver resolves opaque search-result references to canonical server records.
// Implements DESIGN-012 DataNormalizer trusted external provenance.
type EvidenceResolver interface {
	ResolveContext(context.Context, string) (providerregistry.Identity, error)
}

// Service validates editable drafts and coordinates durable confirmation.
// Implements DESIGN-009 DataImporter.
type Service struct {
	store     Store
	evidence  EvidenceResolver
	telemetry *observability.AdminExternalTelemetry
}

// NewService creates curated-import behavior.
// Implements DESIGN-009 DataImporter.
func NewService(store Store, evidence ...EvidenceResolver) *Service {
	service := &Service{store: store}
	if len(evidence) > 0 {
		service.evidence = evidence[0]
	}
	return service
}

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
			s.telemetry.ImportOutcome(ctx, importTelemetryProvider(req.SelectedRecord.Provider), importTelemetryOutcome(result, err))
		}
	}()
	req, err = NormalizeRequest(ctx, req)
	if err != nil {
		return Result{}, err
	}
	identity, err := s.resolveRecord(ctx, req)
	if err != nil {
		return Result{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if identity.Provider == "" && (len(idempotencyKey) < 8 || len(idempotencyKey) > 255 || strings.ContainsRune(idempotencyKey, '\x00')) {
		return Result{}, ErrMissingIdempotencyKey
	}
	item, err := validateImportedItem(req.Request, identity)
	if err != nil {
		return Result{}, err
	}
	if adminID == uuid.Nil || tx == nil || s == nil || s.store == nil {
		return Result{}, repository.NewError(repository.ErrorKindConnection, "curated import service is unavailable", nil)
	}
	req.Request = item
	bodyHash, err := requestHash(req, identity)
	if err != nil {
		return Result{}, err
	}
	confirmed, err := s.store.ConfirmCuratedImport(ctx, tx, repository.CuratedImportConfirmation{
		AdminUserID: adminID, IdempotencyKey: idempotencyKey, BodyHash: bodyHash,
		SourceProvider: identity.Provider, ExternalID: identity.ExternalID, ConfirmNameConflict: req.ConfirmNameConflict,
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
	return Result{ImportID: confirmed.ImportID, FoodItemID: confirmed.Item.ID, Name: confirmed.Item.Name, PhysicalState: confirmed.Item.PhysicalState, Merged: confirmed.Merged, Replayed: confirmed.Replayed, SourceProvider: identity.Provider}, nil
}

// resolveRecord accepts trusted internal records or resolves an opaque client selection.
// Implements DESIGN-012 DataNormalizer exact selected-record identity.
func (s *Service) resolveRecord(ctx context.Context, req Request) (providerregistry.Identity, error) {
	identity := req.SelectedRecord
	if req.ExternalRecordToken != "" {
		if s == nil || s.evidence == nil {
			return providerregistry.Identity{}, ErrExternalRecordEvidence
		}
		resolved, err := s.evidence.ResolveContext(ctx, req.ExternalRecordToken)
		if err != nil || identity.Provider != "" && identity != resolved {
			return providerregistry.Identity{}, ErrExternalRecordEvidence
		}
		identity = resolved
	}
	if (identity.Provider == "") != (identity.ExternalID == "") {
		return providerregistry.Identity{}, ErrExternalRecordEvidence
	}
	if identity.Provider == "" {
		return identity, nil
	}
	canonical, err := providerregistry.Default().Normalize(identity.Provider, identity.ExternalID)
	if err != nil || canonical != identity {
		return providerregistry.Identity{}, ErrExternalRecordEvidence
	}
	return canonical, nil
}

// validateImportedItem keeps manual entry authority separate from trusted import evidence.
// Implements DESIGN-012 DataNormalizer imported density provenance.
func validateImportedItem(req customitem.Request, identity providerregistry.Identity) (customitem.Request, error) {
	kind := req.DensitySourceKind
	if kind == "imported" {
		if identity.Provider == "" {
			return customitem.Request{}, ErrExternalRecordEvidence
		}
		req.DensitySourceKind = "manual"
	}
	normalized, err := customitem.ValidateRequest(req)
	if err != nil {
		return customitem.Request{}, err
	}
	if kind == "imported" {
		normalized.DensitySourceKind = "imported"
		normalized.DensitySourceProvider = identity.Provider
		normalized.DensitySourceFoodID = identity.ExternalID
	}
	return normalized, nil
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
	if canonical, err := providerregistry.Default().NormalizeProvider(provider); err == nil {
		return canonical
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
		MacrosPer100: req.MacrosPer100,
	})
	if err != nil {
		return Request{}, validationError("curated item is invalid")
	}
	req.Name, req.PhysicalState, req.PrepTimeMinutes = normalized.Name, normalized.PhysicalState, normalized.PrepTimeMinutes
	req.AverageUnitWeightGrams, req.AverageServingVolumeMilliliters = normalized.AverageUnitWeightGrams, normalized.AverageServingVolumeMilliliters
	req.DensityGramsPerMilliliter, req.DensitySourceProvider = normalized.DensityGramsPerMilliliter, normalized.DensitySourceProvider
	req.DensitySourceFoodID, req.DensitySourceKind, req.ImageURL = normalized.DensitySourceFoodID, normalized.DensitySourceKind, normalized.ImageURL
	req.MacrosPer100 = normalized.MacrosPer100
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
func requestHash(req Request, identity providerregistry.Identity) (string, error) {
	req.ExternalRecordToken = ""
	req.SelectedRecord = providerregistry.Identity{}
	payload, err := json.Marshal(struct {
		Request
		SourceProvider string `json:"sourceProvider,omitempty"`
		ExternalID     string `json:"externalId,omitempty"`
	}{Request: req, SourceProvider: identity.Provider, ExternalID: identity.ExternalID})
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

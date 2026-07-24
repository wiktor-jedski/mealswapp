package repository

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-009 DataImporter conflict outcomes.
var (
	ErrCuratedImportIdentityConflict         = errors.New("curated import identity conflicts with an existing confirmation")
	ErrCuratedImportNameConfirmationRequired = errors.New("curated import normalized name requires explicit confirmation")
)

// Implements DESIGN-009 DataImporter natural-identity lock.
//
//go:embed sql/curated_import_lock_identity.sql
var curatedImportLockIdentitySQL string

// Implements DESIGN-009 DataImporter natural-identity replay lookup.
//
//go:embed sql/curated_import_find_for_update.sql
var curatedImportFindForUpdateSQL string

// Implements DESIGN-009 DataImporter immutable confirmation insert.
//
//go:embed sql/curated_import_insert.sql
var curatedImportInsertSQL string

// Implements DESIGN-009 DataImporter normalized-name conflict lookup.
//
//go:embed sql/curated_import_food_by_name_for_update.sql
var curatedImportFoodByNameForUpdateSQL string

// Implements DESIGN-009 DataImporter normalized-name serialization.
//
//go:embed sql/curated_import_lock_name.sql
var curatedImportLockNameSQL string

// Implements DESIGN-009 DataImporter idempotency claim.
//
//go:embed sql/curated_import_create_claim.sql
var curatedImportCreateClaimSQL string

// Implements DESIGN-009 DataImporter idempotency replay lookup.
//
//go:embed sql/curated_import_create_claim_get.sql
var curatedImportCreateClaimGetSQL string

// Implements DESIGN-009 DataImporter idempotency response completion.
//
//go:embed sql/curated_import_create_claim_complete.sql
var curatedImportCreateClaimCompleteSQL string

// ConfirmCuratedImport creates or replays one curated import inside the gateway-owned transaction.
// Implements DESIGN-009 DataImporter transactional confirmation.
func (r *PostgresAdminImportAuditRepository) ConfirmCuratedImport(ctx context.Context, tx AdminMutationExecutor, claim CuratedImportConfirmation) (CuratedImportConfirmationResult, error) {
	if err := validateCuratedImportConfirmation(claim, tx); err != nil {
		return CuratedImportConfirmationResult{}, err
	}
	claim.SourceProvider = strings.ToLower(strings.TrimSpace(claim.SourceProvider))
	claim.ExternalID = strings.TrimSpace(claim.ExternalID)
	claim.IdempotencyKey = strings.TrimSpace(claim.IdempotencyKey)
	if claim.SourceProvider != "" {
		return confirmNaturalCuratedImport(ctx, tx, claim)
	}
	return confirmIdempotentCuratedImport(ctx, tx, claim)
}

// confirmNaturalCuratedImport serializes provider identity and rejects body drift.
// Implements DESIGN-009 DataImporter provider/external-ID natural uniqueness.
func confirmNaturalCuratedImport(ctx context.Context, tx AdminMutationExecutor, claim CuratedImportConfirmation) (CuratedImportConfirmationResult, error) {
	if err := tx.QueryRow(ctx, curatedImportLockIdentitySQL, claim.SourceProvider, claim.ExternalID).Scan(new(any)); err != nil {
		return CuratedImportConfirmationResult{}, mapPostgresError(err, "lock curated import identity")
	}
	existing, err := scanCuratedImport(tx.QueryRow(ctx, curatedImportFindForUpdateSQL, claim.SourceProvider, claim.ExternalID))
	if err == nil {
		if existing.Status != "imported" || curatedImportBodyHash(existing.RawPayload) != claim.BodyHash || existing.FoodItemID == nil || *existing.FoodItemID == uuid.Nil {
			return CuratedImportConfirmationResult{}, ErrCuratedImportIdentityConflict
		}
		replay, replayErr := curatedImportReplayFromPayload(existing.RawPayload)
		if replayErr != nil || replay.ImportID != existing.ID || replay.FoodItemID != *existing.FoodItemID {
			return CuratedImportConfirmationResult{}, NewError(ErrorKindInternal, "curated import replay is invalid", replayErr)
		}
		return replay.result(true), nil
	}
	if !IsKind(err, ErrorKindNotFound) {
		return CuratedImportConfirmationResult{}, err
	}
	return persistCuratedImport(ctx, tx, claim, claim.SourceProvider, claim.ExternalID)
}

// confirmIdempotentCuratedImport uses the shared durable mutation-key standard when no natural identity exists.
// Implements DESIGN-009 DataImporter cross-phase idempotency.
func confirmIdempotentCuratedImport(ctx context.Context, tx AdminMutationExecutor, claim CuratedImportConfirmation) (CuratedImportConfirmationResult, error) {
	var adminID uuid.UUID
	err := tx.QueryRow(ctx, curatedImportCreateClaimSQL, claim.AdminUserID, claim.IdempotencyKey, claim.BodyHash).Scan(&adminID)
	if err == nil {
		provider, externalID := syntheticCuratedImportIdentity(claim.AdminUserID, claim.IdempotencyKey)
		result, persistErr := persistCuratedImport(ctx, tx, claim, provider, externalID)
		if persistErr != nil {
			return CuratedImportConfirmationResult{}, persistErr
		}
		payload, marshalErr := json.Marshal(newCuratedImportReplay(result))
		if marshalErr != nil {
			return CuratedImportConfirmationResult{}, marshalErr
		}
		var status int
		var stored []byte
		if completeErr := tx.QueryRow(ctx, curatedImportCreateClaimCompleteSQL, claim.AdminUserID, claim.IdempotencyKey, claim.BodyHash, payload).Scan(&status, &stored); completeErr != nil {
			return CuratedImportConfirmationResult{}, mapPostgresError(completeErr, "complete curated import claim")
		}
		if status != 201 || !json.Valid(stored) {
			return CuratedImportConfirmationResult{}, NewError(ErrorKindInternal, "curated import claim completion is invalid", nil)
		}
		replay, replayErr := decodeCuratedImportReplay(stored)
		if replayErr != nil {
			return CuratedImportConfirmationResult{}, replayErr
		}
		return replay.result(false), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return CuratedImportConfirmationResult{}, mapPostgresError(err, "claim curated import")
	}
	var bodyHash string
	var status int
	var payload []byte
	if err := tx.QueryRow(ctx, curatedImportCreateClaimGetSQL, claim.AdminUserID, claim.IdempotencyKey).Scan(&bodyHash, &status, &payload); err != nil {
		return CuratedImportConfirmationResult{}, mapPostgresError(err, "read curated import claim")
	}
	if bodyHash != claim.BodyHash {
		return CuratedImportConfirmationResult{}, NewError(ErrorKindIdempotencyConflict, "idempotency key reused with different body", nil)
	}
	if status != 201 {
		return CuratedImportConfirmationResult{}, NewError(ErrorKindInternal, "curated import claim is incomplete", nil)
	}
	replay, err := decodeCuratedImportReplay(payload)
	if err != nil {
		return CuratedImportConfirmationResult{}, err
	}
	return replay.result(true), nil
}

// persistCuratedImport validates once, resolves name conflicts, and links immutable import metadata.
// Implements DESIGN-009 DataImporter optimized validation and confirmation.
func persistCuratedImport(ctx context.Context, tx AdminMutationExecutor, claim CuratedImportConfirmation, provider, externalID string) (CuratedImportConfirmationResult, error) {
	if err := validateFoodItemWithExecutor(ctx, tx, claim.Item); err != nil {
		return CuratedImportConfirmationResult{}, err
	}
	// Lock order is import identity/idempotency key, then canonical food name.
	if err := tx.QueryRow(ctx, curatedImportLockNameSQL, claim.Item.Name).Scan(new(any)); err != nil {
		return CuratedImportConfirmationResult{}, mapPostgresError(err, "lock curated import name")
	}
	var foodID uuid.UUID
	nameErr := tx.QueryRow(ctx, curatedImportFoodByNameForUpdateSQL, claim.Item.Name).Scan(&foodID)
	merged := nameErr == nil
	if nameErr != nil && !errors.Is(nameErr, pgx.ErrNoRows) {
		return CuratedImportConfirmationResult{}, mapPostgresError(nameErr, "find curated import name conflict")
	}
	if merged && !claim.ConfirmNameConflict {
		return CuratedImportConfirmationResult{}, ErrCuratedImportNameConfirmationRequired
	}
	if merged {
		claim.Item.ID = foodID
		if err := updateValidatedCuratedFood(ctx, tx, claim.Item); err != nil {
			return CuratedImportConfirmationResult{}, err
		}
	} else {
		var err error
		foodID, err = createValidatedCuratedFood(ctx, tx, claim.Item)
		if err != nil {
			return CuratedImportConfirmationResult{}, err
		}
	}
	item, err := getManualFoodByID(ctx, tx, foodID, false)
	if err != nil {
		return CuratedImportConfirmationResult{}, err
	}
	result := CuratedImportConfirmationResult{ImportID: uuid.New(), Item: item, Merged: merged}
	replay := newCuratedImportReplay(result)
	payload, err := json.Marshal(curatedImportMetadata{BodyHash: claim.BodyHash, Response: replay})
	if err != nil {
		return CuratedImportConfirmationResult{}, err
	}
	var importID uuid.UUID
	if err := tx.QueryRow(ctx, curatedImportInsertSQL, result.ImportID, provider, externalID, foodID, payload).Scan(&importID); err != nil {
		return CuratedImportConfirmationResult{}, mapPostgresError(err, "insert curated import")
	}
	if importID != result.ImportID {
		return CuratedImportConfirmationResult{}, NewError(ErrorKindInternal, "curated import identity changed during insert", nil)
	}
	return result, nil
}

// createValidatedCuratedFood persists a draft after the workflow's single validation pass.
// Implements DESIGN-009 DataImporter optimized validation.
func createValidatedCuratedFood(ctx context.Context, tx AdminMutationExecutor, item FoodItemEntity) (uuid.UUID, error) {
	var id uuid.UUID
	if err := tx.QueryRow(ctx, foodCreateSQL, item.Name, string(item.PhysicalState), item.PrepTimeMinutes, nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters), nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider), nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind), item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat, marshalMicros(item.Micros), nullableString(item.ImageURL)).Scan(&id); err != nil {
		return uuid.Nil, mapPostgresError(err, "create curated food item")
	}
	if err := replaceFoodClassificationsWithExecutor(ctx, tx, id, item.FoodCategories, item.CulinaryRoles); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// updateValidatedCuratedFood merges an explicitly confirmed draft after one validation pass.
// Implements DESIGN-009 DataImporter explicit conflict confirmation.
func updateValidatedCuratedFood(ctx context.Context, tx AdminMutationExecutor, item FoodItemEntity) error {
	result, err := tx.Exec(ctx, foodUpdateSQL, item.ID, item.Name, string(item.PhysicalState), item.PrepTimeMinutes, nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters), nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider), nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind), item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat, marshalMicros(item.Micros), nullableString(item.ImageURL))
	if err != nil {
		return mapPostgresError(err, "update curated food item")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "food item not found", nil)
	}
	return replaceFoodClassificationsWithExecutor(ctx, tx, item.ID, item.FoodCategories, item.CulinaryRoles)
}

// curatedImportReplay is the immutable response identity stored for key-based replays.
// Implements DESIGN-009 DataImporter idempotency response.
type curatedImportReplay struct {
	ImportID      uuid.UUID     `json:"importId"`
	FoodItemID    uuid.UUID     `json:"foodItemId"`
	Name          string        `json:"name"`
	PhysicalState PhysicalState `json:"physicalState"`
	Merged        bool          `json:"merged"`
}

// curatedImportMetadata stores natural-identity replay input and output together.
// Implements DESIGN-009 DataImporter exact replay.
type curatedImportMetadata struct {
	BodyHash string              `json:"bodyHash"`
	Response curatedImportReplay `json:"response"`
}

// newCuratedImportReplay projects only immutable response-owned fields.
// Implements DESIGN-009 DataImporter exact replay.
func newCuratedImportReplay(result CuratedImportConfirmationResult) curatedImportReplay {
	return curatedImportReplay{ImportID: result.ImportID, FoodItemID: result.Item.ID, Name: result.Item.Name, PhysicalState: result.Item.PhysicalState, Merged: result.Merged}
}

// decodeCuratedImportReplay validates a durable key-based response DTO.
// Implements DESIGN-009 DataImporter exact replay.
func decodeCuratedImportReplay(payload []byte) (curatedImportReplay, error) {
	var replay curatedImportReplay
	if json.Unmarshal(payload, &replay) != nil || !validCuratedImportReplay(replay) {
		return curatedImportReplay{}, NewError(ErrorKindInternal, "curated import replay is invalid", nil)
	}
	return replay, nil
}

// curatedImportReplayFromPayload validates a durable natural-identity response DTO.
// Implements DESIGN-009 DataImporter exact replay.
func curatedImportReplayFromPayload(payload []byte) (curatedImportReplay, error) {
	var metadata curatedImportMetadata
	if json.Unmarshal(payload, &metadata) != nil || !validCuratedImportReplay(metadata.Response) {
		return curatedImportReplay{}, NewError(ErrorKindInternal, "curated import metadata is invalid", nil)
	}
	return metadata.Response, nil
}

// validCuratedImportReplay checks fields required to reconstruct the public response.
// Implements DESIGN-009 DataImporter exact replay.
func validCuratedImportReplay(replay curatedImportReplay) bool {
	return replay.ImportID != uuid.Nil && replay.FoodItemID != uuid.Nil && strings.TrimSpace(replay.Name) != "" && (replay.PhysicalState == PhysicalStateSolid || replay.PhysicalState == PhysicalStateLiquid)
}

// result reconstructs a confirmation without loading mutable food state.
// Implements DESIGN-009 DataImporter exact replay.
func (replay curatedImportReplay) result(replayed bool) CuratedImportConfirmationResult {
	return CuratedImportConfirmationResult{
		ImportID: replay.ImportID,
		Item:     FoodItemEntity{ID: replay.FoodItemID, Name: replay.Name, PhysicalState: replay.PhysicalState},
		Merged:   replay.Merged,
		Replayed: replayed,
	}
}

// curatedImportBodyHash reads the closed metadata persisted for natural replay comparison.
// Implements DESIGN-009 DataImporter exact replay.
func curatedImportBodyHash(payload []byte) string {
	var metadata curatedImportMetadata
	if json.Unmarshal(payload, &metadata) != nil {
		return ""
	}
	return metadata.BodyHash
}

// syntheticCuratedImportIdentity keeps key-based imports unique without representing the key as provider data.
// Implements DESIGN-009 DataImporter identity fallback.
func syntheticCuratedImportIdentity(adminID uuid.UUID, key string) (string, string) {
	sum := uuid.NewSHA1(adminID, []byte(key))
	return "idempotency", sum.String()
}

// validateCuratedImportConfirmation rejects malformed identity and hash state before SQL.
// Implements DESIGN-009 DataImporter confirmation validation.
func validateCuratedImportConfirmation(claim CuratedImportConfirmation, tx AdminMutationExecutor) error {
	if tx == nil || claim.AdminUserID == uuid.Nil {
		return validationError("curated import transaction and admin are required")
	}
	provider, externalID := strings.ToLower(strings.TrimSpace(claim.SourceProvider)), strings.TrimSpace(claim.ExternalID)
	if (provider == "") != (externalID == "") || (provider != "" && provider != "usda" && provider != "openfoodfacts") {
		return validationError("curated import provider identity is invalid")
	}
	if provider == "" && (len(strings.TrimSpace(claim.IdempotencyKey)) < 8 || len(claim.IdempotencyKey) > 255 || strings.ContainsRune(claim.IdempotencyKey, '\x00')) {
		return validationError("curated import idempotency key is invalid")
	}
	hash, err := hex.DecodeString(claim.BodyHash)
	if err != nil || len(hash) != 32 {
		return validationError("curated import body hash must be sha256")
	}
	return nil
}

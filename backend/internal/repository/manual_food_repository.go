package repository

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-009 ItemCurator idempotent create claim.
//
//go:embed sql/manual_food_create_claim.sql
var manualFoodCreateClaimSQL string

// Implements DESIGN-009 ItemCurator idempotent create claim lookup.
//
//go:embed sql/manual_food_create_claim_get.sql
var manualFoodCreateClaimGetSQL string

// Implements DESIGN-009 ItemCurator idempotent create completion.
//
//go:embed sql/manual_food_create_claim_complete.sql
var manualFoodCreateClaimCompleteSQL string

// PostgresManualFoodItemRepository persists administrator-authored global food items.
// Implements DESIGN-009 ItemCurator global/private separation.
type PostgresManualFoodItemRepository struct {
	db sqlExecutor
}

// NewPostgresManualFoodItemRepository creates a global-item curation repository.
// Implements DESIGN-009 ItemCurator.
func NewPostgresManualFoodItemRepository(db sqlExecutor) *PostgresManualFoodItemRepository {
	return &PostgresManualFoodItemRepository{db: db}
}

// GetByID loads one global item and never queries the private custom-item table.
// Implements DESIGN-009 ItemCurator global/private separation.
func (r *PostgresManualFoodItemRepository) GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (FoodItemEntity, error) {
	if id == uuid.Nil {
		return FoodItemEntity{}, validationError("food item id is required")
	}
	return getManualFoodByID(ctx, r.db, id, includeDeleted)
}

// GetByIDInMutation loads one global item from the gateway-owned transaction.
// Implements DESIGN-009 ItemCurator transactional audit snapshots.
func (r *PostgresManualFoodItemRepository) GetByIDInMutation(ctx context.Context, tx AdminMutationExecutor, id uuid.UUID, includeDeleted bool) (FoodItemEntity, error) {
	if id == uuid.Nil {
		return FoodItemEntity{}, validationError("food item id is required")
	}
	return getManualFoodByID(ctx, tx, id, includeDeleted)
}

// ClaimCreate creates one global item or replays its transactionally stored response.
// Implements DESIGN-009 ItemCurator idempotent global-item create.
func (r *PostgresManualFoodItemRepository) ClaimCreate(ctx context.Context, tx AdminMutationExecutor, claim ManualFoodItemCreateClaim, encode ManualFoodItemResponseEncoder) (ManualFoodItemCreateClaimResult, error) {
	if err := validateManualFoodCreateClaim(claim, encode); err != nil {
		return ManualFoodItemCreateClaimResult{}, err
	}
	_, claimErr := scanManualFoodCreateClaim(tx.QueryRow(ctx, manualFoodCreateClaimSQL, claim.AdminUserID, claim.Key, claim.BodyHash))
	if claimErr == nil {
		id, err := createManualFoodItem(ctx, tx, claim.Item)
		if err != nil {
			return ManualFoodItemCreateClaimResult{}, err
		}
		item, err := getManualFoodByID(ctx, tx, id, false)
		if err != nil {
			return ManualFoodItemCreateClaimResult{}, err
		}
		payload, err := encode(item)
		if err != nil || len(payload) == 0 || !json.Valid(payload) {
			return ManualFoodItemCreateClaimResult{}, NewError(ErrorKindInternal, "manual food create response is invalid", err)
		}
		completed, err := scanManualFoodCreateClaim(tx.QueryRow(ctx, manualFoodCreateClaimCompleteSQL, claim.AdminUserID, claim.Key, claim.BodyHash, payload))
		if err != nil {
			return ManualFoodItemCreateClaimResult{}, err
		}
		return ManualFoodItemCreateClaimResult{ResponseBody: completed.responseBody, StatusCode: completed.statusCode}, nil
	}
	if !IsKind(claimErr, ErrorKindNotFound) {
		return ManualFoodItemCreateClaimResult{}, claimErr
	}
	existing, err := scanManualFoodCreateClaim(tx.QueryRow(ctx, manualFoodCreateClaimGetSQL, claim.AdminUserID, claim.Key))
	if err != nil {
		return ManualFoodItemCreateClaimResult{}, err
	}
	if existing.bodyHash != claim.BodyHash {
		return ManualFoodItemCreateClaimResult{}, NewError(ErrorKindIdempotencyConflict, "idempotency key reused with different body", nil)
	}
	if existing.statusCode != 201 || len(existing.responseBody) == 0 || !json.Valid(existing.responseBody) {
		return ManualFoodItemCreateClaimResult{}, NewError(ErrorKindInternal, "manual food create claim is incomplete", nil)
	}
	return ManualFoodItemCreateClaimResult{ResponseBody: existing.responseBody, StatusCode: existing.statusCode, Replayed: true}, nil
}

// Update replaces one active global item inside the audit transaction.
// Implements DESIGN-009 ItemCurator update behavior.
func (r *PostgresManualFoodItemRepository) Update(ctx context.Context, tx AdminMutationExecutor, item FoodItemEntity) error {
	if item.ID == uuid.Nil {
		return validationError("food item id is required")
	}
	if err := validateFoodItemWithExecutor(ctx, tx, item); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, foodUpdateSQL, item.ID, item.Name, string(item.PhysicalState), item.PrepTimeMinutes, nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters), nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider), nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind), item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat, marshalMicros(item.Micros), nullableString(item.ImageURL))
	if err != nil {
		return mapPostgresError(err, "update manual food item")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "food item not found", nil)
	}
	return replaceFoodClassificationsWithExecutor(ctx, tx, item.ID, item.FoodCategories, item.CulinaryRoles)
}

// Delete soft-deletes one active global item inside the audit transaction.
// Implements DESIGN-009 ItemCurator soft-delete behavior.
func (r *PostgresManualFoodItemRepository) Delete(ctx context.Context, tx AdminMutationExecutor, id uuid.UUID) error {
	if id == uuid.Nil {
		return validationError("food item id is required")
	}
	result, err := tx.Exec(ctx, foodSoftDeleteSQL, id)
	if err != nil {
		return mapPostgresError(err, "delete manual food item")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "food item not found", nil)
	}
	return nil
}

// createManualFoodItem persists one ownerless global row and its classifications.
// Implements DESIGN-009 ItemCurator global/private separation.
func createManualFoodItem(ctx context.Context, tx sqlExecutor, item FoodItemEntity) (uuid.UUID, error) {
	if err := validateFoodItemWithExecutor(ctx, tx, item); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, foodCreateSQL, item.Name, string(item.PhysicalState), item.PrepTimeMinutes, nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters), nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider), nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind), item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat, marshalMicros(item.Micros), nullableString(item.ImageURL)).Scan(&id); err != nil {
		return uuid.Nil, mapPostgresError(err, "create manual food item")
	}
	if err := replaceFoodClassificationsWithExecutor(ctx, tx, id, item.FoodCategories, item.CulinaryRoles); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// getManualFoodByID hydrates one global row using the supplied executor.
// Implements DESIGN-009 ItemCurator read and audit snapshot behavior.
func getManualFoodByID(ctx context.Context, db sqlExecutor, id uuid.UUID, includeDeleted bool) (FoodItemEntity, error) {
	item, err := getFoodByIDWithExecutor(ctx, db, id, includeDeleted)
	if err != nil {
		return FoodItemEntity{}, err
	}
	if err := hydrateFoodClassificationsWithExecutor(ctx, db, &item); err != nil {
		return FoodItemEntity{}, err
	}
	return item, nil
}

// manualFoodCreateRecord is the validated internal idempotency row.
// Implements DESIGN-009 ItemCurator idempotent create.
type manualFoodCreateRecord struct {
	bodyHash     string
	statusCode   int
	responseBody []byte
}

// scanManualFoodCreateClaim validates one administrator-scoped idempotency row.
// Implements DESIGN-009 ItemCurator idempotent create.
func scanManualFoodCreateClaim(row pgx.Row) (manualFoodCreateRecord, error) {
	var userID uuid.UUID
	var method, route, key, bodyHash string
	var statusCode int
	var responseBody []byte
	var createdAt, updatedAt time.Time
	if err := row.Scan(&userID, &method, &route, &key, &bodyHash, &statusCode, &responseBody, &createdAt, &updatedAt); err != nil {
		return manualFoodCreateRecord{}, mapPostgresError(err, "scan manual food create claim")
	}
	if userID == uuid.Nil || method != "POST" || route != "/admin/items" || strings.TrimSpace(key) == "" || strings.TrimSpace(bodyHash) == "" {
		return manualFoodCreateRecord{}, NewError(ErrorKindInternal, "manual food create claim scope is invalid", nil)
	}
	return manualFoodCreateRecord{bodyHash: bodyHash, statusCode: statusCode, responseBody: responseBody}, nil
}

// validateManualFoodCreateClaim rejects malformed claim identity and encoders.
// Implements DESIGN-009 ItemCurator idempotent create.
func validateManualFoodCreateClaim(claim ManualFoodItemCreateClaim, encode ManualFoodItemResponseEncoder) error {
	if claim.AdminUserID == uuid.Nil {
		return validationError("admin user id is required")
	}
	if len(strings.TrimSpace(claim.Key)) < 8 || len(claim.Key) > 255 || strings.ContainsRune(claim.Key, '\x00') {
		return validationError("manual food create idempotency key is invalid")
	}
	hash, err := hex.DecodeString(claim.BodyHash)
	if err != nil || len(hash) != 32 {
		return validationError("manual food create body hash must be sha256")
	}
	if encode == nil {
		return validationError("manual food create response encoder is required")
	}
	return nil
}

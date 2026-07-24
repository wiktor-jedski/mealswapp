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

// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item create query.
//
//go:embed sql/custom_food_create.sql
var customFoodCreateSQL string

// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item read query.
//
//go:embed sql/custom_food_get_by_id.sql
var customFoodGetByIDSQL string

// Implements DESIGN-008 DataExporter owner-scoped custom-item list query.
//
//go:embed sql/custom_food_list.sql
var customFoodListSQL string

// Implements DESIGN-008 ProfileController durable custom-item create claim query.
//
//go:embed sql/custom_food_create_claim.sql
var customFoodCreateClaimSQL string

// Implements DESIGN-008 ProfileController durable custom-item create claim read query.
//
//go:embed sql/custom_food_create_claim_get.sql
var customFoodCreateClaimGetSQL string

// Implements DESIGN-008 ProfileController durable custom-item create response query.
//
//go:embed sql/custom_food_create_claim_complete.sql
var customFoodCreateClaimCompleteSQL string

// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item update query.
//
//go:embed sql/custom_food_update.sql
var customFoodUpdateSQL string

// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item soft-delete query.
//
//go:embed sql/custom_food_soft_delete.sql
var customFoodSoftDeleteSQL string

// Implements DESIGN-005 FoodItemEntity custom-item classification replacement query.
//
//go:embed sql/custom_food_clear_classifications.sql
var customFoodClearClassificationsSQL string

// Implements DESIGN-005 FoodItemEntity custom-item classification assignment query.
//
//go:embed sql/custom_food_attach_classification.sql
var customFoodAttachClassificationSQL string

// Implements DESIGN-005 FoodItemEntity custom-item classification hydration query.
//
//go:embed sql/custom_food_list_classifications.sql
var customFoodListClassificationsSQL string

// PostgresCustomFoodItemRepository persists private food items separately from the curated catalog.
// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item persistence.
type PostgresCustomFoodItemRepository struct {
	db transactionalExecutor
}

// Implements DESIGN-005 FoodItemEntity compile-time custom repository contract.
var _ CustomFoodItemRepository = (*PostgresCustomFoodItemRepository)(nil)

// NewPostgresCustomFoodItemRepository creates a PostgreSQL-backed private food-item repository.
// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item persistence.
func NewPostgresCustomFoodItemRepository(db transactionalExecutor) *PostgresCustomFoodItemRepository {
	return &PostgresCustomFoodItemRepository{db: db}
}

// GetByID loads a private item only when it belongs to ownerID.
// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item read.
func (r *PostgresCustomFoodItemRepository) GetByID(ctx context.Context, ownerID uuid.UUID, id uuid.UUID, rc RepositoryContext) (CustomFoodItemEntity, error) {
	if err := validateCustomFoodIdentity(ownerID, id); err != nil {
		return CustomFoodItemEntity{}, err
	}
	item, err := scanFoodItem(r.db.QueryRow(ctx, customFoodGetByIDSQL, ownerID, id, rc.IncludeDeleted))
	if err != nil {
		if IsKind(err, ErrorKindValidation) {
			return CustomFoodItemEntity{}, err
		}
		return CustomFoodItemEntity{}, mapPostgresError(err, "custom food item not found")
	}
	if err := r.hydrateClassifications(ctx, &item); err != nil {
		return CustomFoodItemEntity{}, err
	}
	convertFoodItemForUnitSystem(&item, rc.UnitSystem)
	return CustomFoodItemEntity{FoodItemEntity: item, OwnerID: ownerID}, nil
}

// List loads private items belonging only to ownerID in deterministic order.
// Implements DESIGN-008 DataExporter owner-scoped custom-item export.
func (r *PostgresCustomFoodItemRepository) List(ctx context.Context, ownerID uuid.UUID, rc RepositoryContext) ([]CustomFoodItemEntity, error) {
	if ownerID == uuid.Nil {
		return nil, validationError("custom food item owner id is required")
	}
	rows, err := r.db.Query(ctx, customFoodListSQL, ownerID, rc.IncludeDeleted)
	if err != nil {
		return nil, mapPostgresError(err, "list custom food items")
	}
	baseItems := make([]FoodItemEntity, 0)
	for rows.Next() {
		item, err := scanFoodItem(rows)
		if err != nil {
			rows.Close()
			return nil, mapPostgresError(err, "scan custom food item")
		}
		baseItems = append(baseItems, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, mapPostgresError(err, "iterate custom food items")
	}
	rows.Close()

	items := make([]CustomFoodItemEntity, 0, len(baseItems))
	for _, item := range baseItems {
		if err := r.hydrateClassifications(ctx, &item); err != nil {
			return nil, err
		}
		convertFoodItemForUnitSystem(&item, rc.UnitSystem)
		items = append(items, CustomFoodItemEntity{FoodItemEntity: item, OwnerID: ownerID})
	}
	return items, nil
}

// ClaimCreate atomically serializes a scoped key, creates one item, and stores its response.
// Implements DESIGN-008 ProfileController durable custom-item creation.
func (r *PostgresCustomFoodItemRepository) ClaimCreate(ctx context.Context, claim CustomFoodItemCreateClaim, encode CustomFoodItemResponseEncoder) (CustomFoodItemCreateClaimResult, error) {
	if err := validateCustomFoodCreateClaim(claim, encode); err != nil {
		return CustomFoodItemCreateClaimResult{}, err
	}
	var result CustomFoodItemCreateClaimResult
	err := withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		_, claimErr := scanCustomFoodCreateClaim(db.QueryRow(ctx, customFoodCreateClaimSQL, claim.UserID, claim.Key, claim.BodyHash))
		if claimErr == nil {
			itemID, err := createCustomFoodItemInTransaction(ctx, db, claim.Item)
			if err != nil {
				return err
			}
			item, err := NewPostgresCustomFoodItemRepository(db).GetByID(ctx, claim.UserID, itemID, RepositoryContext{UnitSystem: UnitSystemMetric})
			if err != nil {
				return err
			}
			payload, err := encode(item)
			if err != nil || len(payload) == 0 || !json.Valid(payload) {
				return NewError(ErrorKindInternal, "custom food create response is invalid", err)
			}
			completed, err := scanCustomFoodCreateClaim(db.QueryRow(ctx, customFoodCreateClaimCompleteSQL, claim.UserID, claim.Key, claim.BodyHash, payload))
			if err != nil {
				return err
			}
			result = CustomFoodItemCreateClaimResult{ResponseBody: completed.responseBody, StatusCode: completed.statusCode}
			return nil
		}
		if !IsKind(claimErr, ErrorKindNotFound) {
			return claimErr
		}
		existing, err := scanCustomFoodCreateClaim(db.QueryRow(ctx, customFoodCreateClaimGetSQL, claim.UserID, claim.Key))
		if err != nil {
			return err
		}
		if existing.bodyHash != claim.BodyHash {
			return NewError(ErrorKindIdempotencyConflict, "idempotency key reused with different body", nil)
		}
		if existing.statusCode != 201 || len(existing.responseBody) == 0 || !json.Valid(existing.responseBody) {
			return NewError(ErrorKindInternal, "custom food create claim is incomplete", nil)
		}
		result = CustomFoodItemCreateClaimResult{ResponseBody: existing.responseBody, StatusCode: existing.statusCode, Replayed: true}
		return nil
	})
	return result, err
}

// Create validates and persists a private food item for its mandatory owner.
// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item create.
func (r *PostgresCustomFoodItemRepository) Create(ctx context.Context, item CustomFoodItemEntity) (uuid.UUID, error) {
	if item.OwnerID == uuid.Nil {
		return uuid.Nil, validationError("custom food item owner id is required")
	}
	if err := NewPostgresFoodItemRepository(r.db).validateFoodItem(ctx, item.FoodItemEntity); err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	err := withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		var err error
		id, err = createCustomFoodItemInTransaction(ctx, db, item)
		return err
	})
	return id, err
}

// createCustomFoodItemInTransaction persists one item and its classifications without committing independently.
// Implements DESIGN-008 ProfileController atomic custom-item creation.
func createCustomFoodItemInTransaction(ctx context.Context, db transactionalExecutor, item CustomFoodItemEntity) (uuid.UUID, error) {
	if item.OwnerID == uuid.Nil {
		return uuid.Nil, validationError("custom food item owner id is required")
	}
	if err := NewPostgresFoodItemRepository(db).validateFoodItem(ctx, item.FoodItemEntity); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	if err := db.QueryRow(ctx, customFoodCreateSQL,
		item.OwnerID, item.Name, string(item.PhysicalState), item.PrepTimeMinutes,
		nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters),
		nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider),
		nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind),
		item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat,
		marshalMicros(item.Micros), nullableString(item.ImageURL),
	).Scan(&id); err != nil {
		return uuid.Nil, mapPostgresError(err, "create custom food item")
	}
	if err := NewPostgresCustomFoodItemRepository(db).replaceClassifications(ctx, id, item.FoodCategories, item.CulinaryRoles); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// customFoodCreateRecord is the validated internal idempotency row.
// Implements DESIGN-008 ProfileController durable custom-item creation.
type customFoodCreateRecord struct {
	bodyHash     string
	statusCode   int
	responseBody []byte
}

// scanCustomFoodCreateClaim reads one scoped custom-item mutation claim.
// Implements DESIGN-008 ProfileController durable custom-item creation.
func scanCustomFoodCreateClaim(row pgx.Row) (customFoodCreateRecord, error) {
	var userID uuid.UUID
	var method, route, key, bodyHash string
	var statusCode int
	var responseBody []byte
	var createdAt, updatedAt time.Time
	if err := row.Scan(&userID, &method, &route, &key, &bodyHash, &statusCode, &responseBody, &createdAt, &updatedAt); err != nil {
		return customFoodCreateRecord{}, mapPostgresError(err, "scan custom food create claim")
	}
	if userID == uuid.Nil || method != "POST" || route != "/custom-items" || strings.TrimSpace(key) == "" || strings.TrimSpace(bodyHash) == "" {
		return customFoodCreateRecord{}, NewError(ErrorKindInternal, "custom food create claim scope is invalid", nil)
	}
	return customFoodCreateRecord{bodyHash: bodyHash, statusCode: statusCode, responseBody: responseBody}, nil
}

// validateCustomFoodCreateClaim rejects malformed atomic mutation inputs.
// Implements DESIGN-008 ProfileController durable custom-item creation.
func validateCustomFoodCreateClaim(claim CustomFoodItemCreateClaim, encode CustomFoodItemResponseEncoder) error {
	if claim.UserID == uuid.Nil || claim.Item.OwnerID != claim.UserID {
		return validationError("custom food create owner is invalid")
	}
	if len(strings.TrimSpace(claim.Key)) < 8 || len(claim.Key) > 255 || strings.ContainsRune(claim.Key, '\x00') {
		return validationError("custom food create idempotency key is invalid")
	}
	hash, err := hex.DecodeString(claim.BodyHash)
	if err != nil || len(hash) != 32 {
		return validationError("custom food create body hash must be sha256")
	}
	if encode == nil {
		return validationError("custom food create response encoder is required")
	}
	return nil
}

// Update replaces a private food item only when its ID belongs to its owner.
// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item update.
func (r *PostgresCustomFoodItemRepository) Update(ctx context.Context, item CustomFoodItemEntity) error {
	if err := validateCustomFoodIdentity(item.OwnerID, item.ID); err != nil {
		return err
	}
	if err := NewPostgresFoodItemRepository(r.db).validateFoodItem(ctx, item.FoodItemEntity); err != nil {
		return err
	}

	return withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		result, err := db.Exec(ctx, customFoodUpdateSQL,
			item.OwnerID, item.ID, item.Name, string(item.PhysicalState), item.PrepTimeMinutes,
			nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters),
			nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider),
			nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind),
			item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat,
			marshalMicros(item.Micros), nullableString(item.ImageURL),
		)
		if err != nil {
			return mapPostgresError(err, "update custom food item")
		}
		if result.RowsAffected() == 0 {
			return NewError(ErrorKindNotFound, "custom food item not found", nil)
		}
		return NewPostgresCustomFoodItemRepository(db).replaceClassifications(ctx, item.ID, item.FoodCategories, item.CulinaryRoles)
	})
}

// Delete soft-deletes a private food item only when it belongs to ownerID.
// Implements DESIGN-005 FoodItemEntity owner-scoped custom-item delete.
func (r *PostgresCustomFoodItemRepository) Delete(ctx context.Context, ownerID uuid.UUID, id uuid.UUID) error {
	if err := validateCustomFoodIdentity(ownerID, id); err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, customFoodSoftDeleteSQL, ownerID, id)
	if err != nil {
		return mapPostgresError(err, "delete custom food item")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "custom food item not found", nil)
	}
	return nil
}

// hydrateClassifications loads global classification identities assigned to a private item.
// Implements DESIGN-005 FoodItemEntity custom-item classification hydration.
func (r *PostgresCustomFoodItemRepository) hydrateClassifications(ctx context.Context, item *FoodItemEntity) error {
	rows, err := r.db.Query(ctx, customFoodListClassificationsSQL, item.ID)
	if err != nil {
		return mapPostgresError(err, "load custom food classifications")
	}
	defer rows.Close()

	item.FoodCategories = nil
	item.CulinaryRoles = nil
	for rows.Next() {
		var classification ClassificationEntity
		if err := rows.Scan(&classification.ID, &classification.Name, &classification.Kind, &classification.ParentID); err != nil {
			return mapPostgresError(err, "scan custom food classification")
		}
		switch classification.Kind {
		case ClassificationKindFoodCategory:
			item.FoodCategories = append(item.FoodCategories, classification)
		case ClassificationKindCulinaryRole:
			item.CulinaryRoles = append(item.CulinaryRoles, classification)
		}
	}
	if err := rows.Err(); err != nil {
		return mapPostgresError(err, "iterate custom food classifications")
	}
	return nil
}

// replaceClassifications atomically replaces a private item's classification assignments.
// Implements DESIGN-005 FoodItemEntity custom-item classification persistence.
func (r *PostgresCustomFoodItemRepository) replaceClassifications(ctx context.Context, itemID uuid.UUID, foodCategories []ClassificationEntity, culinaryRoles []ClassificationEntity) error {
	if _, err := r.db.Exec(ctx, customFoodClearClassificationsSQL, itemID); err != nil {
		return mapPostgresError(err, "clear custom food classifications")
	}
	for _, classifications := range [][]ClassificationEntity{foodCategories, culinaryRoles} {
		for _, classification := range classifications {
			if _, err := r.db.Exec(ctx, customFoodAttachClassificationSQL, itemID, classification.ID); err != nil {
				return mapPostgresError(err, "replace custom food classifications")
			}
		}
	}
	return nil
}

// validateCustomFoodIdentity rejects ownerless or unidentified private-item operations before database access.
// Implements DESIGN-005 FoodItemEntity mandatory custom-item ownership.
func validateCustomFoodIdentity(ownerID uuid.UUID, itemID uuid.UUID) error {
	if ownerID == uuid.Nil {
		return validationError("custom food item owner id is required")
	}
	if itemID == uuid.Nil {
		return validationError("custom food item id is required")
	}
	return nil
}

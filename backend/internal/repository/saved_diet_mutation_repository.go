package repository

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-008 SavedDataRepository fixed daily-diet create scope.
const (
	dailyDietCreateMethod = "POST"
	dailyDietCreateRoute  = "/daily-diets"
)

// Implements DESIGN-008 SavedDataRepository typed daily-diet create claim query.
//
//go:embed sql/daily_diet_create_claim.sql
var dailyDietCreateClaimSQL string

// Implements DESIGN-008 SavedDataRepository typed daily-diet create claim read query.
//
//go:embed sql/daily_diet_create_claim_get.sql
var dailyDietCreateClaimGetSQL string

// Implements DESIGN-008 SavedDataRepository atomic daily-diet parent query.
//
//go:embed sql/saved_diet_create_snapshot.sql
var savedDietCreateSnapshotSQL string

// Implements DESIGN-008 SavedDataRepository atomic daily-diet entry query.
//
//go:embed sql/saved_diet_entry_insert_snapshot.sql
var savedDietEntryInsertSnapshotSQL string

// Implements DESIGN-008 SavedDataRepository ownership-aware deletion query.
//
//go:embed sql/saved_diet_exists.sql
var savedDietExistsSQL string

// Implements DESIGN-008 SavedDataRepository atomic mutation contract.
var _ DailyDietMutationRepository = (*PostgresSavedDataRepository)(nil)

// GetDailyDietCreateClaim loads and validates an immutable response for an exact request hash.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func (r *PostgresSavedDataRepository) GetDailyDietCreateClaim(ctx context.Context, userID uuid.UUID, key string, bodyHash string) (DailyDietCreateClaimResult, error) {
	if err := validateDailyDietCreateScope(userID, key, bodyHash); err != nil {
		return DailyDietCreateClaimResult{}, err
	}
	record, err := scanDailyDietCreateClaim(r.db.QueryRow(ctx, dailyDietCreateClaimGetSQL, userID, dailyDietCreateMethod, dailyDietCreateRoute, key))
	if err != nil {
		return DailyDietCreateClaimResult{}, err
	}
	if record.bodyHash != bodyHash {
		return DailyDietCreateClaimResult{}, NewError(ErrorKindConflict, "idempotency key reused with different body", nil)
	}
	return DailyDietCreateClaimResult{Response: record.response, StatusCode: record.statusCode, Replayed: true}, nil
}

// ClaimDailyDietCreate atomically claims a key and writes the exact immutable response snapshot.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func (r *PostgresSavedDataRepository) ClaimDailyDietCreate(ctx context.Context, claim DailyDietCreateClaim) (DailyDietCreateClaimResult, error) {
	if err := validateDailyDietCreateClaim(claim); err != nil {
		return DailyDietCreateClaimResult{}, err
	}
	payload, err := json.Marshal(claim.Response)
	if err != nil {
		return DailyDietCreateClaimResult{}, validationError("daily diet create response is invalid")
	}

	var result DailyDietCreateClaimResult
	err = withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		record, claimErr := scanDailyDietCreateClaim(db.QueryRow(ctx, dailyDietCreateClaimSQL, claim.UserID, dailyDietCreateMethod, dailyDietCreateRoute, claim.Key, claim.BodyHash, claim.StatusCode, payload))
		if claimErr == nil {
			if err := createSavedDietSnapshot(ctx, db, claim); err != nil {
				return err
			}
			result = DailyDietCreateClaimResult{Response: record.response, StatusCode: record.statusCode}
			return nil
		}
		if !IsKind(claimErr, ErrorKindNotFound) {
			return claimErr
		}

		existing, existingErr := scanDailyDietCreateClaim(db.QueryRow(ctx, dailyDietCreateClaimGetSQL, claim.UserID, dailyDietCreateMethod, dailyDietCreateRoute, claim.Key))
		if existingErr != nil {
			return existingErr
		}
		if existing.bodyHash != claim.BodyHash {
			return NewError(ErrorKindConflict, "idempotency key reused with different body", nil)
		}
		result = DailyDietCreateClaimResult{Response: existing.response, StatusCode: existing.statusCode, Replayed: true}
		return nil
	})
	return result, err
}

// DeleteIfOwned deletes a diet and reports whether an existing row belonged to another user.
// Implements DESIGN-008 SavedDataRepository and ProfileController.
func (r *PostgresSavedDataRepository) DeleteIfOwned(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) (bool, bool, error) {
	if userID == uuid.Nil {
		return false, false, validationError("user id is required")
	}
	if dietID == uuid.Nil {
		return false, false, validationError("saved diet id is required")
	}
	var deleted, exists bool
	err := withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		tag, err := db.Exec(ctx, savedDietDeleteSQL, dietID, userID)
		if err != nil {
			return mapPostgresError(err, "delete saved diet")
		}
		deleted = tag.RowsAffected() > 0
		if deleted {
			exists = true
			return nil
		}
		if err := db.QueryRow(ctx, savedDietExistsSQL, dietID).Scan(&exists); err != nil {
			return mapPostgresError(err, "check saved diet ownership")
		}
		return nil
	})
	return deleted, exists, err
}

// dailyDietCreateRecord is the validated internal persistence row.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
type dailyDietCreateRecord struct {
	bodyHash   string
	statusCode int
	response   DailyDietCreateResponse
}

// scanDailyDietCreateClaim decodes only the canonical persisted response shape.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func scanDailyDietCreateClaim(row pgx.Row) (dailyDietCreateRecord, error) {
	var userID uuid.UUID
	var method, route, key, bodyHash string
	var statusCode int
	var payload []byte
	var createdAt, updatedAt time.Time
	if err := row.Scan(&userID, &method, &route, &key, &bodyHash, &statusCode, &payload, &createdAt, &updatedAt); err != nil {
		return dailyDietCreateRecord{}, mapPostgresError(err, "scan daily diet create claim")
	}
	if method != dailyDietCreateMethod || route != dailyDietCreateRoute || statusCode != 201 {
		return dailyDietCreateRecord{}, NewError(ErrorKindInternal, "daily diet create claim scope is invalid", nil)
	}
	response, err := decodeDailyDietCreateResponse(payload)
	if err != nil {
		return dailyDietCreateRecord{}, err
	}
	if err := validateDailyDietCreateScope(userID, key, bodyHash); err != nil {
		return dailyDietCreateRecord{}, NewError(ErrorKindInternal, "daily diet create claim is invalid", err)
	}
	return dailyDietCreateRecord{bodyHash: bodyHash, statusCode: statusCode, response: response}, nil
}

// decodeDailyDietCreateResponse rejects unknown, trailing, dual, and legacy response bodies.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func decodeDailyDietCreateResponse(payload []byte) (DailyDietCreateResponse, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var response DailyDietCreateResponse
	if err := decoder.Decode(&response); err != nil {
		return DailyDietCreateResponse{}, NewError(ErrorKindInternal, "daily diet create response is invalid", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return DailyDietCreateResponse{}, NewError(ErrorKindInternal, "daily diet create response has trailing data", nil)
	}
	var required struct {
		AggregateMacros *struct {
			Protein       *float64 `json:"protein"`
			Carbohydrates *float64 `json:"carbohydrates"`
			Fat           *float64 `json:"fat"`
			Calories      *float64 `json:"calories"`
		} `json:"aggregateMacros"`
	}
	if err := json.Unmarshal(payload, &required); err != nil || required.AggregateMacros == nil || required.AggregateMacros.Protein == nil || required.AggregateMacros.Carbohydrates == nil || required.AggregateMacros.Fat == nil || required.AggregateMacros.Calories == nil {
		return DailyDietCreateResponse{}, NewError(ErrorKindInternal, "daily diet create response macros are incomplete", err)
	}
	if err := validateDailyDietCreateResponse(response); err != nil {
		return DailyDietCreateResponse{}, NewError(ErrorKindInternal, "daily diet create response is invalid", err)
	}
	return response, nil
}

// validateDailyDietCreateClaim checks that persistence input and response describe one exact resource.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func validateDailyDietCreateClaim(claim DailyDietCreateClaim) error {
	if err := validateDailyDietCreateScope(claim.UserID, claim.Key, claim.BodyHash); err != nil {
		return err
	}
	if claim.StatusCode != 201 {
		return validationError("daily diet create status must be 201")
	}
	if err := validateSavedDietInput(claim.UserID, claim.Diet, false); err != nil {
		return err
	}
	if claim.Diet.ID != claim.Response.ID || claim.Diet.UserID != claim.UserID || !claim.Diet.CreatedAt.Equal(claim.Response.CreatedAt) || !claim.Diet.UpdatedAt.Equal(claim.Response.UpdatedAt) {
		return validationError("daily diet create identities do not match")
	}
	if err := validateDailyDietCreateResponse(claim.Response); err != nil {
		return err
	}
	if len(claim.Diet.Entries) != len(claim.Response.Entries) {
		return validationError("daily diet create entries do not match")
	}
	for index, entry := range claim.Diet.Entries {
		responseEntry := claim.Response.Entries[index]
		entryID, entryType := savedDietEntryFoodObject(entry)
		responseID, responseType := responseEntryFoodObject(responseEntry)
		if entry.ID != responseEntry.ID || entry.SavedDietID != claim.Diet.ID || entryID != responseID || entryType != responseType || entry.Quantity != responseEntry.Quantity || entry.Unit != responseEntry.Unit || entry.Position != responseEntry.Position {
			return validationError("daily diet create entries do not match")
		}
	}
	return nil
}

// validateDailyDietCreateScope checks the fixed user, key, and SHA-256 request scope.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func validateDailyDietCreateScope(userID uuid.UUID, key string, bodyHash string) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if len(strings.TrimSpace(key)) < 8 || len(key) > 255 {
		return validationError("idempotency key is invalid")
	}
	decoded, err := hex.DecodeString(bodyHash)
	if err != nil || len(decoded) != 32 {
		return validationError("body hash must be sha256")
	}
	return nil
}

// validateDailyDietCreateResponse checks the exact immutable response domain shape.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func validateDailyDietCreateResponse(response DailyDietCreateResponse) error {
	if response.ID == uuid.Nil || strings.TrimSpace(response.Name) == "" || response.CreatedAt.IsZero() || response.UpdatedAt.IsZero() || len(response.Entries) == 0 || len(response.Entries) > 100 {
		return validationError("daily diet create response is incomplete")
	}
	positions := make(map[int]struct{}, len(response.Entries))
	for _, entry := range response.Entries {
		objectID, objectType := responseEntryFoodObject(entry)
		if entry.ID == uuid.Nil || objectID == uuid.Nil || (objectType != FoodObjectTypeMeal && objectType != FoodObjectTypeFoodItem) || entry.Quantity <= 0 || math.IsNaN(entry.Quantity) || math.IsInf(entry.Quantity, 0) || ValidateQuantityUnit(entry.Unit) != nil || entry.Position < 0 || entry.Position >= 100 {
			return validationError("daily diet create response entry is invalid")
		}
		if _, exists := positions[entry.Position]; exists {
			return validationError("daily diet create response positions are duplicated")
		}
		positions[entry.Position] = struct{}{}
	}
	macros := response.AggregateMacros
	for _, value := range []float64{macros.Protein, macros.Carbohydrates, macros.Fat, macros.Calories} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return validationError("daily diet create response macros are invalid")
		}
	}
	return nil
}

// createSavedDietSnapshot writes response-identical parent, entries, and saved-item rows.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func createSavedDietSnapshot(ctx context.Context, db transactionalExecutor, claim DailyDietCreateClaim) error {
	if _, err := db.Exec(ctx, savedDietCreateSnapshotSQL, claim.Diet.ID, claim.UserID, claim.Diet.Name, claim.Diet.CreatedAt, claim.Diet.UpdatedAt); err != nil {
		return mapPostgresError(err, "create saved diet")
	}
	for _, entry := range claim.Diet.Entries {
		objectID, objectType := savedDietEntryFoodObject(entry)
		if _, err := db.Exec(ctx, savedDietEntryInsertSnapshotSQL, entry.ID, claim.Diet.ID, objectID, objectType, entry.Quantity, entry.Unit, entry.Position, entry.CreatedAt); err != nil {
			return mapPostgresError(err, "create saved diet entry")
		}
	}
	return ensureSavedDietItem(ctx, db, claim.UserID, claim.Diet.ID)
}

func savedDietEntryFoodObject(entry SavedDietMealEntry) (uuid.UUID, FoodObjectType) {
	if entry.FoodObjectID != uuid.Nil {
		return entry.FoodObjectID, entry.FoodObjectType
	}
	return entry.MealID, FoodObjectTypeMeal
}

func responseEntryFoodObject(entry DailyDietCreateResponseEntry) (uuid.UUID, FoodObjectType) {
	if entry.FoodObjectID != uuid.Nil {
		return entry.FoodObjectID, entry.FoodObjectType
	}
	return entry.MealID, FoodObjectTypeMeal
}

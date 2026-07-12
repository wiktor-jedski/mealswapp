package repository

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-008 ProfileController daily-diet idempotency claim query.
//
//go:embed sql/checkout_idempotency_claim.sql
var checkoutIdempotencyClaimSQL string

// Implements DESIGN-008 ProfileController daily-diet idempotency lock query.
//
//go:embed sql/checkout_idempotency_get_for_update.sql
var checkoutIdempotencyGetForUpdateSQL string

// Implements DESIGN-008 SavedDataRepository atomic daily-diet parent query.
//
//go:embed sql/saved_diet_create_with_id.sql
var savedDietCreateWithIDSQL string

// Implements DESIGN-008 SavedDataRepository ownership-aware deletion query.
//
//go:embed sql/saved_diet_exists.sql
var savedDietExistsSQL string

// Implements DESIGN-008 SavedDataRepository and ProfileController atomic mutation contract.
var _ DailyDietMutationRepository = (*PostgresSavedDataRepository)(nil)

// CreateWithIdempotency claims the request key and persists the complete daily-diet write atomically.
// Implements DESIGN-008 SavedDataRepository and ProfileController.
func (r *PostgresSavedDataRepository) CreateWithIdempotency(ctx context.Context, userID uuid.UUID, diet SavedDiet, record CheckoutIdempotencyRecord) (AtomicDailyDietMutationResult, error) {
	if err := validateSavedDietInput(userID, diet, false); err != nil {
		return AtomicDailyDietMutationResult{}, err
	}
	entries, err := normalizeSavedDietEntries(diet.Entries)
	if err != nil {
		return AtomicDailyDietMutationResult{}, err
	}
	if record.UserID != userID || record.Method != "POST" || record.Route != "/daily-diets" {
		return AtomicDailyDietMutationResult{}, validationError("daily diet idempotency scope is invalid")
	}
	if err := validateCheckoutIdempotencyRecord(record); err != nil {
		return AtomicDailyDietMutationResult{}, err
	}
	claimedDietID, err := dailyDietIDFromIdempotencyResponse(record.ResponseBody)
	if err != nil {
		return AtomicDailyDietMutationResult{}, err
	}
	if diet.ID != uuid.Nil && diet.ID != claimedDietID {
		return AtomicDailyDietMutationResult{}, validationError("daily diet idempotency id does not match diet id")
	}
	diet.ID = claimedDietID

	var result AtomicDailyDietMutationResult
	err = withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		claimed, claimErr := scanCheckoutIdempotencyRecord(db.QueryRow(ctx, checkoutIdempotencyClaimSQL, record.UserID, record.Method, record.Route, record.Key, record.BodyHash, record.StatusCode, record.ResponseBody))
		if claimErr == nil {
			if err := createSavedDietRows(ctx, db, userID, diet, entries); err != nil {
				return err
			}
			result = AtomicDailyDietMutationResult{DietID: diet.ID, Idempotency: claimed}
			return nil
		}
		if !IsKind(claimErr, ErrorKindNotFound) {
			return claimErr
		}

		existing, existingErr := scanCheckoutIdempotencyRecord(db.QueryRow(ctx, checkoutIdempotencyGetForUpdateSQL, record.UserID, record.Method, record.Route, record.Key))
		if existingErr != nil {
			if IsKind(existingErr, ErrorKindNotFound) {
				return NewError(ErrorKindRetryable, "daily diet idempotency claim disappeared", existingErr)
			}
			return existingErr
		}
		if existing.BodyHash != record.BodyHash {
			return NewError(ErrorKindConflict, "idempotency key reused with different body", nil)
		}
		existingDietID, err := dailyDietIDFromIdempotencyResponse(existing.ResponseBody)
		if err != nil {
			return err
		}
		result = AtomicDailyDietMutationResult{DietID: existingDietID, Idempotency: existing, Replayed: true}
		return nil
	})
	if err != nil {
		return AtomicDailyDietMutationResult{}, err
	}
	return result, nil
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

// createSavedDietRows writes the parent, ordered entries, and saved-item index in the caller's transaction.
// Implements DESIGN-008 SavedDataRepository.
func createSavedDietRows(ctx context.Context, db transactionalExecutor, userID uuid.UUID, diet SavedDiet, entries []SavedDietMealEntry) error {
	if err := db.QueryRow(ctx, savedDietCreateWithIDSQL, diet.ID, userID, normalizeSavedDietName(diet.Name)).Scan(&diet.ID); err != nil {
		return mapPostgresError(err, "create saved diet")
	}
	if err := replaceSavedDietEntries(ctx, db, diet.ID, entries); err != nil {
		return err
	}
	return ensureSavedDietItem(ctx, db, userID, diet.ID)
}

// scanCheckoutIdempotencyRecord scans one durable mutation idempotency row.
// Implements DESIGN-008 ProfileController daily-diet idempotency.
func scanCheckoutIdempotencyRecord(row pgx.Row) (CheckoutIdempotencyRecord, error) {
	var record CheckoutIdempotencyRecord
	err := row.Scan(
		&record.UserID,
		&record.Method,
		&record.Route,
		&record.Key,
		&record.BodyHash,
		&record.StatusCode,
		&record.ResponseBody,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return CheckoutIdempotencyRecord{}, mapPostgresError(err, "scan checkout idempotency")
	}
	return record, nil
}

// dailyDietIDFromIdempotencyResponse extracts the durable resource reference.
// Implements DESIGN-008 ProfileController daily-diet idempotency.
func dailyDietIDFromIdempotencyResponse(payload []byte) (uuid.UUID, error) {
	var response struct {
		DailyDietID uuid.UUID `json:"dailyDietId"`
		ID          uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return uuid.Nil, NewError(ErrorKindInternal, "daily diet idempotency response is invalid", err)
	}
	if response.DailyDietID != uuid.Nil {
		return response.DailyDietID, nil
	}
	if response.ID != uuid.Nil {
		return response.ID, nil
	}
	return uuid.Nil, NewError(ErrorKindInternal, "daily diet idempotency response has no diet id", errors.New("missing daily diet id"))
}

package repository

import (
	"context"
	_ "embed"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-009 UserAdminPanel bounded lookup queries.
//
//go:embed sql/admin_user_list.sql
var adminUserListSQL string

// Implements DESIGN-009 UserAdminPanel exact user-id lookup query.
//
//go:embed sql/admin_user_get_by_id.sql
var adminUserGetByIDSQL string

// Implements DESIGN-009 UserAdminPanel exact encrypted-email lookup query.
//
//go:embed sql/admin_user_get_by_digest.sql
var adminUserGetByDigestSQL string

// Implements DESIGN-009 UserAdminPanel locked retry query.
//
//go:embed sql/admin_deletion_retry.sql
var adminDeletionRetrySQL string

// PostgresAdminUserRepository persists restricted user-administration operations.
// Implements DESIGN-009 UserAdminPanel.
type PostgresAdminUserRepository struct {
	db sqlExecutor
}

// Implements DESIGN-009 UserAdminPanel compile-time repository contract.
var _ AdminUserRepository = (*PostgresAdminUserRepository)(nil)

// NewPostgresAdminUserRepository creates restricted user-administration persistence.
// Implements DESIGN-009 UserAdminPanel.
func NewPostgresAdminUserRepository(db sqlExecutor) *PostgresAdminUserRepository {
	return &PostgresAdminUserRepository{db: db}
}

// LookupAdminUsers returns an exact result or one bounded deterministic page.
// Implements DESIGN-009 UserAdminPanel.
func (r *PostgresAdminUserRepository) LookupAdminUsers(ctx context.Context, lookup AdminUserLookup) ([]AdminUserRecord, error) {
	if err := validateAdminUserLookup(lookup); err != nil {
		return nil, err
	}
	query, args := adminUserListSQL, []any{lookup.AfterID, lookup.Limit}
	if lookup.UserID != nil {
		query, args = adminUserGetByIDSQL, []any{*lookup.UserID}
	} else if lookup.EmailDigest != nil {
		query, args = adminUserGetByDigestSQL, []any{lookup.EmailDigest.KeyVersion, lookup.EmailDigest.Value}
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, mapPostgresError(err, "lookup administrative users")
	}
	defer rows.Close()
	users := []AdminUserRecord{}
	for rows.Next() {
		user, err := scanAdminUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate administrative users")
	}
	return users, nil
}

// RetryAdminDeletion atomically claims one eligible failure in the supplied audit transaction.
// Implements DESIGN-009 UserAdminPanel.
func (r *PostgresAdminUserRepository) RetryAdminDeletion(ctx context.Context, tx AdminMutationExecutor, userID uuid.UUID, requestID uuid.UUID) (AdminDeletionRetry, error) {
	if tx == nil || userID == uuid.Nil || requestID == uuid.Nil {
		return AdminDeletionRetry{}, validationError("transaction, user id, and deletion request id are required")
	}
	var retry AdminDeletionRetry
	err := tx.QueryRow(ctx, adminDeletionRetrySQL, requestID, userID).Scan(&retry.RequestID, &retry.FailureCategory, &retry.RetryCount)
	if err != nil {
		return AdminDeletionRetry{}, mapPostgresError(err, "retry account deletion")
	}
	return retry, nil
}

// validateAdminUserLookup enforces one exact selector or one bounded page.
// Implements DESIGN-009 UserAdminPanel.
func validateAdminUserLookup(lookup AdminUserLookup) error {
	if lookup.Limit < 1 || lookup.Limit > 26 {
		return validationError("administrative user lookup limit is invalid")
	}
	exact := 0
	if lookup.UserID != nil {
		if *lookup.UserID == uuid.Nil {
			return validationError("administrative user id is invalid")
		}
		exact++
	}
	if lookup.EmailDigest != nil {
		if err := validateLookupDigest(*lookup.EmailDigest); err != nil {
			return err
		}
		exact++
	}
	if exact > 1 || exact == 1 && lookup.AfterID != nil {
		return validationError("administrative user lookup scope is invalid")
	}
	return nil
}

// scanAdminUser reads only the restricted encrypted projection.
// Implements DESIGN-009 UserAdminPanel.
func scanAdminUser(row pgx.Row) (AdminUserRecord, error) {
	var user AdminUserRecord
	var requestID *uuid.UUID
	var status *string
	var failureCategory *string
	var retryCount *int
	var requestedAt *time.Time
	if err := row.Scan(&user.ID, &user.Email.KeyVersion, &user.Email.Nonce, &user.Email.Ciphertext, &user.EmailVerified, &user.CreatedAt, &requestID, &status, &failureCategory, &retryCount, &requestedAt); err != nil {
		return AdminUserRecord{}, mapPostgresError(err, "scan administrative user")
	}
	if requestID != nil && status != nil && retryCount != nil && requestedAt != nil {
		user.Deletion = &AdminDeletionSummary{RequestID: *requestID, Status: *status, RetryCount: *retryCount, RequestedAt: *requestedAt}
		if failureCategory != nil {
			user.Deletion.FailureCategory = *failureCategory
		}
	}
	return user, nil
}

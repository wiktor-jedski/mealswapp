package repository

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// adminBootstrapLockSQL serializes bootstrap attempts.
// Implements DESIGN-009 AdminController operator bootstrap queries.
//
//go:embed sql/admin_bootstrap_lock.sql
var adminBootstrapLockSQL string

// adminBootstrapTargetByDigestSQL resolves the normalized encrypted-email lookup.
// Implements DESIGN-009 AdminController operator bootstrap queries.
//
//go:embed sql/admin_bootstrap_target_by_digest.sql
var adminBootstrapTargetByDigestSQL string

// adminBootstrapTargetByIDSQL resolves the advanced UUID selector.
// Implements DESIGN-009 AdminController operator bootstrap queries.
//
//go:embed sql/admin_bootstrap_target_by_id.sql
var adminBootstrapTargetByIDSQL string

// adminBootstrapExistingSQL enforces first-administrator-only creation.
// Implements DESIGN-009 AdminController operator bootstrap queries.
//
//go:embed sql/admin_bootstrap_existing.sql
var adminBootstrapExistingSQL string

// adminBootstrapPromoteSQL persists the role transition.
// Implements DESIGN-009 AdminController operator bootstrap queries.
//
//go:embed sql/admin_bootstrap_promote.sql
var adminBootstrapPromoteSQL string

// adminBootstrapAuditSQL persists truthful operator-origin evidence.
// Implements DESIGN-009 AdminController operator bootstrap queries.
//
//go:embed sql/admin_bootstrap_audit.sql
var adminBootstrapAuditSQL string

// PostgresAdministratorBootstrapRepository owns the serialized bootstrap transaction.
// Implements DESIGN-009 AdminController operator-only bootstrap.
type PostgresAdministratorBootstrapRepository struct {
	db                 transactionalExecutor
	validateCredential PasswordCredentialValidator
}

// Implements DESIGN-009 AdminController compile-time bootstrap repository contract.
var _ AdministratorBootstrapRepository = (*PostgresAdministratorBootstrapRepository)(nil)

// NewPostgresAdministratorBootstrapRepository creates a PostgreSQL bootstrap repository.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func NewPostgresAdministratorBootstrapRepository(db transactionalExecutor, validateCredential PasswordCredentialValidator) *PostgresAdministratorBootstrapRepository {
	return &PostgresAdministratorBootstrapRepository{db: db, validateCredential: validateCredential}
}

// BootstrapAdministrator promotes one verified credentialed account and writes its operator audit atomically.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func (r *PostgresAdministratorBootstrapRepository) BootstrapAdministrator(ctx context.Context, selector AdministratorBootstrapSelector, requestID string) (result AdministratorBootstrapResult, err error) {
	if err := validateAdministratorBootstrapSelector(selector); err != nil {
		return result, err
	}
	if r.validateCredential == nil {
		return result, validationError("password credential validator is required")
	}
	if _, err := uuid.Parse(requestID); err != nil {
		return result, validationError("bootstrap request id is required")
	}
	err = withTransaction(ctx, r.db, func(tx transactionalExecutor) error {
		if _, err := tx.Exec(ctx, adminBootstrapLockSQL); err != nil {
			return mapPostgresError(err, "lock administrator bootstrap")
		}
		target, err := loadAdministratorBootstrapTarget(ctx, tx, selector)
		if err != nil {
			return err
		}
		if !target.verified {
			return ErrAdministratorBootstrapUnverified
		}
		hasPassword := target.passwordHash != nil && target.passwordSalt != nil && r.validateCredential(*target.passwordHash, *target.passwordSalt)
		if !target.hasOAuthCredential && !hasPassword {
			return ErrAdministratorBootstrapNoCredential
		}
		var existing uuid.UUID
		err = tx.QueryRow(ctx, adminBootstrapExistingSQL).Scan(&existing)
		switch {
		case err == nil && existing == target.id:
			result = AdministratorBootstrapResult{UserID: target.id, Replayed: true}
			return nil
		case err == nil:
			return ErrAdministratorAlreadyExists
		case !errors.Is(err, pgx.ErrNoRows):
			return mapPostgresError(err, "load existing administrator")
		}
		if _, err := tx.Exec(ctx, adminBootstrapPromoteSQL, target.id); err != nil {
			return mapPostgresError(err, "promote administrator")
		}
		result.UserID = target.id
		var auditID uuid.UUID
		if err := tx.QueryRow(ctx, adminBootstrapAuditSQL, target.id, requestID).Scan(&auditID, &result.AuditedAt); err != nil {
			return fmt.Errorf("%w: %w", ErrAdminAuditPersistence, mapPostgresError(err, "audit administrator bootstrap"))
		}
		result.AuditID = &auditID
		return nil
	})
	return result, err
}

// administratorBootstrapTarget is the transaction-locked eligibility projection.
// Implements DESIGN-009 AdminController operator-only bootstrap.
type administratorBootstrapTarget struct {
	id                 uuid.UUID
	role               UserRole
	verified           bool
	passwordHash       *string
	passwordSalt       *string
	hasOAuthCredential bool
}

// loadAdministratorBootstrapTarget resolves and locks exactly one selected account.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func loadAdministratorBootstrapTarget(ctx context.Context, tx transactionalExecutor, selector AdministratorBootstrapSelector) (administratorBootstrapTarget, error) {
	if selector.UserID != nil {
		return scanAdministratorBootstrapTarget(tx.QueryRow(ctx, adminBootstrapTargetByIDSQL, *selector.UserID))
	}
	canonical, canonicalErr := scanAdministratorBootstrapTarget(tx.QueryRow(ctx, adminBootstrapTargetByDigestSQL, selector.EmailDigest.KeyVersion, selector.EmailDigest.Value))
	if selector.LegacyEmailDigest == nil {
		return canonical, canonicalErr
	}
	legacy, legacyErr := scanAdministratorBootstrapTarget(tx.QueryRow(ctx, adminBootstrapTargetByDigestSQL, selector.LegacyEmailDigest.KeyVersion, selector.LegacyEmailDigest.Value))
	if canonicalErr == nil && legacyErr == nil {
		if canonical.id != legacy.id {
			return administratorBootstrapTarget{}, ErrCanonicalEmailCollision
		}
		return canonical, nil
	}
	if canonicalErr == nil {
		if !errors.Is(legacyErr, ErrAdministratorBootstrapTargetNotFound) {
			return administratorBootstrapTarget{}, legacyErr
		}
		return canonical, nil
	}
	if !errors.Is(canonicalErr, ErrAdministratorBootstrapTargetNotFound) {
		return administratorBootstrapTarget{}, canonicalErr
	}
	if legacyErr != nil {
		return administratorBootstrapTarget{}, legacyErr
	}
	if err := NewPostgresEncryptedIdentityRepository(tx).ReindexUserEmailDigest(ctx, legacy.id, *selector.EmailDigest); err != nil {
		if IsKind(err, ErrorKindConflict) {
			return administratorBootstrapTarget{}, ErrCanonicalEmailCollision
		}
		return administratorBootstrapTarget{}, err
	}
	return legacy, nil
}

// scanAdministratorBootstrapTarget reads a locked bootstrap eligibility projection.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func scanAdministratorBootstrapTarget(row pgx.Row) (administratorBootstrapTarget, error) {
	var target administratorBootstrapTarget
	if err := row.Scan(&target.id, &target.role, &target.verified, &target.passwordHash, &target.passwordSalt, &target.hasOAuthCredential); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return target, ErrAdministratorBootstrapTargetNotFound
		}
		return target, mapPostgresError(err, "load administrator bootstrap target")
	}
	return target, nil
}

// validateAdministratorBootstrapSelector requires exactly one valid selector.
// Implements DESIGN-009 AdminController operator-only bootstrap.
func validateAdministratorBootstrapSelector(selector AdministratorBootstrapSelector) error {
	if (selector.UserID == nil) == (selector.EmailDigest == nil) {
		return validationError("exactly one bootstrap selector is required")
	}
	if selector.UserID != nil && *selector.UserID == uuid.Nil {
		return validationError("bootstrap user id is required")
	}
	if selector.EmailDigest != nil {
		if err := validateLookupDigest(*selector.EmailDigest); err != nil {
			return err
		}
	}
	if selector.LegacyEmailDigest != nil {
		if selector.EmailDigest == nil || *selector.LegacyEmailDigest == *selector.EmailDigest {
			return validationError("legacy bootstrap email selector is invalid")
		}
		return validateLookupDigest(*selector.LegacyEmailDigest)
	}
	return nil
}

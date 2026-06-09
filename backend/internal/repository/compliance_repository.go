package repository

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-015 ConsentManager record query.
//
//go:embed sql/consent_record.sql
var consentRecordSQL string

// Implements DESIGN-015 ConsentManager required-consent query.
//
//go:embed sql/consent_has_required.sql
var consentHasRequiredSQL string

// Implements DESIGN-015 ConsentManager account-export list query.
//
//go:embed sql/consent_list_by_user.sql
var consentListByUserSQL string

// Implements DESIGN-015 DataRetentionPolicy request query.
//
//go:embed sql/deletion_request.sql
var deletionRequestSQL string

// Implements DESIGN-015 DataRetentionPolicy status query.
//
//go:embed sql/deletion_get_status.sql
var deletionGetStatusSQL string

// Implements DESIGN-015 DataRetentionPolicy status-update query.
//
//go:embed sql/deletion_update_status.sql
var deletionUpdateStatusSQL string

// Implements DESIGN-015 DataRetentionPolicy transition-audit query.
//
//go:embed sql/deletion_audit_insert.sql
var deletionAuditInsertSQL string

// Implements DESIGN-015 DataRetentionPolicy audit-list query.
//
//go:embed sql/deletion_audit_list.sql
var deletionAuditListSQL string

// Implements DESIGN-015 DataRetentionPolicy worker claim query.
//
//go:embed sql/deletion_claim.sql
var deletionClaimSQL string

// Implements DESIGN-015 DataRetentionPolicy categorized failure query.
//
//go:embed sql/deletion_fail.sql
var deletionFailSQL string

// Implements DESIGN-015 DataRetentionPolicy pseudonymous receipt query.
//
//go:embed sql/deletion_complete.sql
var deletionCompleteSQL string

// Implements DESIGN-009 AdminController curated-import upsert query.
//
//go:embed sql/curated_import_upsert.sql
var curatedImportUpsertSQL string

// Implements DESIGN-009 AdminController curated-import lookup query.
//
//go:embed sql/curated_import_find.sql
var curatedImportFindSQL string

// Implements DESIGN-009 AdminController audit insert query.
//
//go:embed sql/admin_audit_insert.sql
var adminAuditInsertSQL string

// Implements DESIGN-009 AdminController audit-list query.
//
//go:embed sql/admin_audit_list_for_entity.sql
var adminAuditListForEntitySQL string

// PostgresComplianceRepository persists consent and deletion workflow records.
// Implements DESIGN-015 DataRetentionPolicy.
type PostgresComplianceRepository struct {
	db transactionalExecutor
}

// Implements DESIGN-015 ConsentManager compile-time repository contract.
var _ ConsentRepository = (*PostgresComplianceRepository)(nil)

// Implements DESIGN-015 DataRetentionPolicy compile-time repository contract.
var _ DeletionRequestRepository = (*PostgresComplianceRepository)(nil)

// NewPostgresComplianceRepository creates a PostgreSQL-backed compliance repository.
// Implements DESIGN-015 DataRetentionPolicy.
func NewPostgresComplianceRepository(db transactionalExecutor) *PostgresComplianceRepository {
	return &PostgresComplianceRepository{db: db}
}

// RecordConsent stores one accepted privacy and terms version.
// Implements DESIGN-015 ConsentManager.
func (r *PostgresComplianceRepository) RecordConsent(ctx context.Context, record ConsentRecord) (uuid.UUID, error) {
	if err := validateConsentRecord(record); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, consentRecordSQL, record.UserID, record.PrivacyPolicyVersion, record.TermsVersion).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "record consent")
	}
	return id, nil
}

// HasRequiredConsent reports whether the user accepted the required legal versions.
// Implements DESIGN-015 ConsentManager.
func (r *PostgresComplianceRepository) HasRequiredConsent(ctx context.Context, userID uuid.UUID, privacyVersion string, termsVersion string) (bool, error) {
	if userID == uuid.Nil {
		return false, validationError("user id is required")
	}
	if strings.TrimSpace(privacyVersion) == "" || strings.TrimSpace(termsVersion) == "" {
		return false, validationError("consent versions are required")
	}
	var exists bool
	err := r.db.QueryRow(ctx, consentHasRequiredSQL, userID, privacyVersion, termsVersion).Scan(&exists)
	if err != nil {
		return false, mapPostgresError(err, "check consent")
	}
	return exists, nil
}

// ListConsent returns accepted consent records for account export.
// Implements DESIGN-015 ConsentManager.
func (r *PostgresComplianceRepository) ListConsent(ctx context.Context, userID uuid.UUID) ([]ConsentRecord, error) {
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	rows, err := r.db.Query(ctx, consentListByUserSQL, userID)
	if err != nil {
		return nil, mapPostgresError(err, "list consent")
	}
	defer rows.Close()
	records := []ConsentRecord{}
	for rows.Next() {
		var record ConsentRecord
		if err := rows.Scan(&record.ID, &record.UserID, &record.PrivacyPolicyVersion, &record.TermsVersion, &record.AcceptedAt); err != nil {
			return nil, mapPostgresError(err, "scan consent")
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate consent")
	}
	return records, nil
}

// RequestDeletion creates or returns the active deletion request for a user.
// Implements DESIGN-015 DataRetentionPolicy.
func (r *PostgresComplianceRepository) RequestDeletion(ctx context.Context, userID uuid.UUID) (DataDeletionRequest, error) {
	if userID == uuid.Nil {
		return DataDeletionRequest{}, validationError("user id is required")
	}
	row := r.db.QueryRow(ctx, deletionRequestSQL, userID)
	return scanDeletionRequest(row)
}

// UpdateDeletionStatus stores a deletion status transition and audit note atomically.
// Implements DESIGN-015 DataRetentionPolicy.
func (r *PostgresComplianceRepository) UpdateDeletionStatus(ctx context.Context, requestID uuid.UUID, status string, note string) error {
	if requestID == uuid.Nil {
		return validationError("request id is required")
	}
	if !validDeletionStatus(status) {
		return validationError("deletion status is invalid")
	}
	return withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		var previous string
		if err := db.QueryRow(ctx, deletionGetStatusSQL, requestID).Scan(&previous); err != nil {
			return mapPostgresError(err, "load deletion request")
		}
		if !validDeletionTransition(previous, status) {
			return NewError(ErrorKindConflict, "deletion status transition is invalid", nil)
		}
		result, err := db.Exec(ctx, deletionUpdateStatusSQL, requestID, status, note)
		if err != nil {
			return mapPostgresError(err, "update deletion request")
		}
		if result.RowsAffected() == 0 {
			return NewError(ErrorKindNotFound, "deletion request not found", nil)
		}
		_, err = db.Exec(ctx, deletionAuditInsertSQL, requestID, previous, status, note)
		if err != nil {
			return mapPostgresError(err, "audit deletion transition")
		}
		return nil
	})
}

// ListDeletionAudit returns exportable deletion audit records.
// Implements DESIGN-015 DataRetentionPolicy.
func (r *PostgresComplianceRepository) ListDeletionAudit(ctx context.Context, requestID uuid.UUID) ([]DataDeletionAuditEntry, error) {
	if requestID == uuid.Nil {
		return nil, validationError("request id is required")
	}
	rows, err := r.db.Query(ctx, deletionAuditListSQL, requestID)
	if err != nil {
		return nil, mapPostgresError(err, "list deletion audit")
	}
	defer rows.Close()
	entries := []DataDeletionAuditEntry{}
	for rows.Next() {
		entry, err := scanDeletionAuditEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate deletion audit")
	}
	return entries, nil
}

// ClaimDeletionRequests atomically claims pending or retryable deletion work.
// Implements DESIGN-015 DataRetentionPolicy.
func (r *PostgresComplianceRepository) ClaimDeletionRequests(ctx context.Context, now time.Time, limit int) ([]DataDeletionRequest, error) {
	if limit <= 0 {
		limit = 1
	}
	rows, err := r.db.Query(ctx, deletionClaimSQL, now, limit)
	if err != nil {
		return nil, mapPostgresError(err, "claim deletion requests")
	}
	defer rows.Close()
	requests := []DataDeletionRequest{}
	for rows.Next() {
		request, err := scanDeletionRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate deletion claims")
	}
	return requests, nil
}

// RecordDeletionFailure stores sanitized failure category and retry metadata.
// Implements DESIGN-015 DataRetentionPolicy.
func (r *PostgresComplianceRepository) RecordDeletionFailure(ctx context.Context, requestID uuid.UUID, category string, note string, nextAttemptAt *time.Time) error {
	if requestID == uuid.Nil {
		return validationError("request id is required")
	}
	if category != "transient" && category != "permanent" && category != "unknown" {
		return validationError("failure category is invalid")
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, deletionFailSQL, requestID, category, note, nextAttemptAt).Scan(&id)
	if err != nil {
		return mapPostgresError(err, "record deletion failure")
	}
	_, err = r.db.Exec(ctx, deletionAuditInsertSQL, requestID, "processing", "failed", category)
	if err != nil {
		return mapPostgresError(err, "audit deletion failure")
	}
	return nil
}

// CompleteDeletionRequest stores a pseudonymous deletion receipt.
// Implements DESIGN-015 DataRetentionPolicy.
func (r *PostgresComplianceRepository) CompleteDeletionRequest(ctx context.Context, requestID uuid.UUID, receiptID uuid.UUID, completedAt time.Time) error {
	if requestID == uuid.Nil || receiptID == uuid.Nil {
		return validationError("request and receipt ids are required")
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, deletionCompleteSQL, requestID, receiptID, completedAt).Scan(&id)
	if err != nil {
		return mapPostgresError(err, "complete deletion request")
	}
	_, err = r.db.Exec(ctx, deletionAuditInsertSQL, requestID, "processing", "completed", "completed")
	if err != nil {
		return mapPostgresError(err, "audit deletion completion")
	}
	return nil
}

// PostgresAdminImportAuditRepository persists curated imports and admin audit records.
// Implements DESIGN-009 AdminController.
type PostgresAdminImportAuditRepository struct {
	db transactionalExecutor
}

// Implements DESIGN-009 AdminController compile-time repository contract.
var _ AdminAuditRepository = (*PostgresAdminImportAuditRepository)(nil)

// Implements DESIGN-009 DataImporter compile-time repository contract.
var _ CuratedImportRepository = (*PostgresAdminImportAuditRepository)(nil)

// NewPostgresAdminImportAuditRepository creates a PostgreSQL-backed admin repository.
// Implements DESIGN-009 AdminController.
func NewPostgresAdminImportAuditRepository(db transactionalExecutor) *PostgresAdminImportAuditRepository {
	return &PostgresAdminImportAuditRepository{db: db}
}

// UpsertCuratedImport stores external curation metadata and returns the stable import id.
// Implements DESIGN-009 AdminController.
func (r *PostgresAdminImportAuditRepository) UpsertCuratedImport(ctx context.Context, item CuratedImport) (uuid.UUID, error) {
	if err := validateCuratedImport(item); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, curatedImportUpsertSQL, item.SourceProvider, item.ExternalID, item.FoodItemID, item.Status, item.ConflictReason, normalizedJSONPayload(item.RawPayload)).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "upsert curated import")
	}
	return id, nil
}

// FindCuratedImport returns an import by external source identity.
// Implements DESIGN-009 AdminController.
func (r *PostgresAdminImportAuditRepository) FindCuratedImport(ctx context.Context, provider string, externalID string) (CuratedImport, error) {
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(externalID) == "" {
		return CuratedImport{}, validationError("provider and external id are required")
	}
	row := r.db.QueryRow(ctx, curatedImportFindSQL, provider, externalID)
	return scanCuratedImport(row)
}

// PersistAuditEntry stores one admin audit record.
// Implements DESIGN-009 AdminController.
func (r *PostgresAdminImportAuditRepository) PersistAuditEntry(ctx context.Context, entry AdminAuditEntry) (uuid.UUID, error) {
	if err := validateAdminAuditEntry(entry); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, adminAuditInsertSQL, entry.AdminUserID, entry.Action, entry.EntityType, entry.EntityID, nullableJSONPayload(entry.Before), nullableJSONPayload(entry.After), entry.RequestID).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "persist admin audit")
	}
	return id, nil
}

// WithAudit runs a mutation and persists audit in the same transaction.
// Implements DESIGN-009 AdminController.
func (r *PostgresAdminImportAuditRepository) WithAudit(ctx context.Context, entry AdminAuditEntry, fn func(sqlExecutor) error) error {
	if fn == nil {
		return validationError("audit mutation is required")
	}
	return withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		if err := fn(db); err != nil {
			return err
		}
		_, err := NewPostgresAdminImportAuditRepository(db).PersistAuditEntry(ctx, entry)
		return err
	})
}

// ListAuditForEntity returns exportable audit entries for one entity.
// Implements DESIGN-009 AdminController.
func (r *PostgresAdminImportAuditRepository) ListAuditForEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]AdminAuditEntry, error) {
	if strings.TrimSpace(entityType) == "" || entityID == uuid.Nil {
		return nil, validationError("entity type and id are required")
	}
	rows, err := r.db.Query(ctx, adminAuditListForEntitySQL, entityType, entityID)
	if err != nil {
		return nil, mapPostgresError(err, "list admin audit")
	}
	defer rows.Close()
	entries := []AdminAuditEntry{}
	for rows.Next() {
		entry, err := scanAdminAuditEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate admin audit")
	}
	return entries, nil
}

// scanDeletionRequest reads an account-deletion request from a PostgreSQL row.
// Implements DESIGN-015 DataRetentionPolicy.
func scanDeletionRequest(row pgx.Row) (DataDeletionRequest, error) {
	var request DataDeletionRequest
	if err := row.Scan(&request.ID, &request.UserID, &request.Status, &request.RequestedAt, &request.CompletedAt, &request.FailureReason, &request.FailureCategory, &request.RetryCount, &request.NextAttemptAt, &request.ReceiptID, &request.ReceiptIssuedAt); err != nil {
		return DataDeletionRequest{}, mapPostgresError(err, "scan deletion request")
	}
	return request, nil
}

// scanDeletionAuditEntry reads a deletion audit record from a PostgreSQL row.
// Implements DESIGN-015 DataRetentionPolicy.
func scanDeletionAuditEntry(row pgx.Row) (DataDeletionAuditEntry, error) {
	var entry DataDeletionAuditEntry
	if err := row.Scan(&entry.ID, &entry.RequestID, &entry.FromStatus, &entry.ToStatus, &entry.Note, &entry.CreatedAt); err != nil {
		return DataDeletionAuditEntry{}, mapPostgresError(err, "scan deletion audit")
	}
	return entry, nil
}

// scanCuratedImport reads curated import metadata from a PostgreSQL row.
// Implements DESIGN-009 AdminController.
func scanCuratedImport(row pgx.Row) (CuratedImport, error) {
	var item CuratedImport
	if err := row.Scan(&item.ID, &item.SourceProvider, &item.ExternalID, &item.FoodItemID, &item.Status, &item.ConflictReason, &item.RawPayload, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return CuratedImport{}, mapPostgresError(err, "scan curated import")
	}
	return item, nil
}

// scanAdminAuditEntry reads an administrative audit record from a PostgreSQL row.
// Implements DESIGN-009 AdminController.
func scanAdminAuditEntry(row pgx.Row) (AdminAuditEntry, error) {
	var entry AdminAuditEntry
	if err := row.Scan(&entry.ID, &entry.AdminUserID, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.Before, &entry.After, &entry.RequestID, &entry.CreatedAt); err != nil {
		return AdminAuditEntry{}, mapPostgresError(err, "scan admin audit")
	}
	return entry, nil
}

// validateConsentRecord checks required consent fields.
// Implements DESIGN-015 ConsentManager.
func validateConsentRecord(record ConsentRecord) error {
	if record.UserID == uuid.Nil {
		return validationError("user id is required")
	}
	if strings.TrimSpace(record.PrivacyPolicyVersion) == "" || strings.TrimSpace(record.TermsVersion) == "" {
		return validationError("consent versions are required")
	}
	return nil
}

// validDeletionStatus reports whether status is supported by the deletion workflow.
// Implements DESIGN-015 DataRetentionPolicy.
func validDeletionStatus(status string) bool {
	return status == "pending" || status == "processing" || status == "completed" || status == "failed"
}

// validDeletionTransition reports whether the deletion workflow allows a state change.
// Implements DESIGN-015 DataRetentionPolicy.
func validDeletionTransition(from string, to string) bool {
	switch from {
	case "pending":
		return to == "processing"
	case "processing":
		return to == "completed" || to == "failed"
	case "failed":
		return to == "processing"
	default:
		return false
	}
}

// validateCuratedImport checks curated import identity, state, and payload fields.
// Implements DESIGN-009 AdminController.
func validateCuratedImport(item CuratedImport) error {
	if strings.TrimSpace(item.SourceProvider) == "" || strings.TrimSpace(item.ExternalID) == "" {
		return validationError("provider and external id are required")
	}
	if item.Status != "draft" && item.Status != "imported" && item.Status != "conflict" && item.Status != "rejected" {
		return validationError("curated import status is invalid")
	}
	if !json.Valid(normalizedJSONPayload(item.RawPayload)) {
		return validationError("raw payload must be valid json")
	}
	return nil
}

// validateAdminAuditEntry checks required administrative audit fields.
// Implements DESIGN-009 AdminController.
func validateAdminAuditEntry(entry AdminAuditEntry) error {
	if entry.AdminUserID == uuid.Nil {
		return validationError("admin user id is required")
	}
	if strings.TrimSpace(entry.Action) == "" || strings.TrimSpace(entry.EntityType) == "" {
		return validationError("audit action and entity type are required")
	}
	if entry.Before != nil && !json.Valid(entry.Before) {
		return validationError("before snapshot must be valid json")
	}
	if entry.After != nil && !json.Valid(entry.After) {
		return validationError("after snapshot must be valid json")
	}
	return nil
}

// normalizedJSONPayload replaces an empty JSON payload with an empty object.
// Implements DESIGN-009 AdminController.
func normalizedJSONPayload(payload []byte) []byte {
	if len(payload) == 0 {
		return []byte(`{}`)
	}
	return payload
}

// nullableJSONPayload converts an empty JSON payload to a SQL null value.
// Implements DESIGN-009 AdminController.
func nullableJSONPayload(payload []byte) any {
	if len(payload) == 0 {
		return nil
	}
	return payload
}

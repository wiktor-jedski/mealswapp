package repository

import (
	"context"
	_ "embed"
	"time"

	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-012 DataNormalizer deployment-safe external evidence storage.
//
//go:embed sql/record_evidence_store.sql
var storeRecordEvidenceSQL string

// Implements DESIGN-012 DataNormalizer expired evidence cleanup.
//
//go:embed sql/record_evidence_cleanup.sql
var cleanupRecordEvidenceSQL string

// Implements DESIGN-012 DataNormalizer fail-closed external evidence.
//
//go:embed sql/record_evidence_resolve.sql
var resolveRecordEvidenceSQL string

// PostgresRecordEvidenceRepository coordinates evidence across API instances.
// Implements DESIGN-012 DataNormalizer deployment-safe provenance coordination.
type PostgresRecordEvidenceRepository struct{ db transactionalExecutor }

// NewPostgresRecordEvidenceRepository creates PostgreSQL-backed record evidence storage.
// Implements DESIGN-012 DataNormalizer deployment-safe provenance coordination.
func NewPostgresRecordEvidenceRepository(db transactionalExecutor) *PostgresRecordEvidenceRepository {
	return &PostgresRecordEvidenceRepository{db: db}
}

// StoreRecordEvidence persists one opaque server-issued token until its expiry.
// Implements DESIGN-012 DataNormalizer exact selected-record provenance.
func (r *PostgresRecordEvidenceRepository) StoreRecordEvidence(ctx context.Context, token, provider, externalID string, expiresAt time.Time) error {
	if r == nil || r.db == nil {
		return NewError(ErrorKindConnection, "record evidence repository is unavailable", nil)
	}
	err := withTransaction(ctx, r.db, func(tx transactionalExecutor) error {
		if _, err := tx.Exec(ctx, cleanupRecordEvidenceSQL, expiresAt); err != nil {
			return mapPostgresError(err, "cleanup record evidence")
		}
		if _, err := tx.Exec(ctx, storeRecordEvidenceSQL, token, provider, externalID, expiresAt); err != nil {
			return mapPostgresError(err, "store record evidence")
		}
		return nil
	})
	return err
}

// ResolveRecordEvidence reads one live token, preserving safe idempotent retries until expiry.
// Implements DESIGN-012 DataNormalizer fail-closed external evidence.
func (r *PostgresRecordEvidenceRepository) ResolveRecordEvidence(ctx context.Context, token string, now time.Time) (string, string, error) {
	if r == nil || r.db == nil {
		return "", "", NewError(ErrorKindConnection, "record evidence repository is unavailable", nil)
	}
	var provider, externalID string
	err := r.db.QueryRow(ctx, resolveRecordEvidenceSQL, token, now).Scan(&provider, &externalID)
	if err == pgx.ErrNoRows {
		return "", "", NewError(ErrorKindNotFound, "record evidence not found", nil)
	}
	return provider, externalID, mapPostgresError(err, "resolve record evidence")
}

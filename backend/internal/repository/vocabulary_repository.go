package repository

import (
	"context"
	_ "embed"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-005 MicronutrientVocabulary active vocabulary query.
//
//go:embed sql/vocabulary_list_active.sql
var vocabularyListActiveSQL string

// Implements DESIGN-005 MicronutrientVocabulary allowed-key query.
//
//go:embed sql/vocabulary_is_allowed.sql
var vocabularyIsAllowedSQL string

// Implements DESIGN-005 MicronutrientVocabulary upsert query.
//
//go:embed sql/vocabulary_upsert.sql
var vocabularyUpsertSQL string

// Implements DESIGN-005 MicronutrientVocabulary administrator listing.
//
//go:embed sql/vocabulary_list_all.sql
var vocabularyListAllSQL string

// Implements DESIGN-005 MicronutrientVocabulary administrator lookup.
//
//go:embed sql/vocabulary_get.sql
var vocabularyGetSQL string

// Implements DESIGN-005 MicronutrientVocabulary retry-safe creation.
//
//go:embed sql/vocabulary_create.sql
var vocabularyCreateSQL string

// Implements DESIGN-005 MicronutrientVocabulary display-name update.
//
//go:embed sql/vocabulary_update_display_name.sql
var vocabularyUpdateDisplayNameSQL string

// Implements DESIGN-005 MicronutrientVocabulary safe unit update.
//
//go:embed sql/vocabulary_update_unit.sql
var vocabularyUpdateUnitSQL string

// Implements DESIGN-005 MicronutrientVocabulary lifecycle.
//
//go:embed sql/vocabulary_set_active.sql
var vocabularySetActiveSQL string

// Implements DESIGN-005 MicronutrientVocabulary item usage guard.
//
//go:embed sql/vocabulary_is_in_use.sql
var vocabularyIsInUseSQL string

// Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
//
//go:embed sql/vocabulary_lock_usage_tables.sql
var vocabularyLockUsageTablesSQL string

// Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
//
//go:embed sql/vocabulary_lock_item_write_tables.sql
var vocabularyLockItemWriteTablesSQL string

// PostgresMicronutrientVocabularyRepository persists canonical micronutrient keys.
// Implements DESIGN-005 MicronutrientVocabulary.
type PostgresMicronutrientVocabularyRepository struct {
	db sqlExecutor
}

// Implements DESIGN-005 MicronutrientVocabulary compile-time repository contract.
var _ MicronutrientVocabularyRepository = (*PostgresMicronutrientVocabularyRepository)(nil)

// Implements DESIGN-005 MicronutrientVocabulary compile-time administrator contract.
var _ MicronutrientVocabularyAdminRepository = (*PostgresMicronutrientVocabularyRepository)(nil)

// NewPostgresMicronutrientVocabularyRepository creates a PostgreSQL-backed vocabulary repository.
// Implements DESIGN-005 MicronutrientVocabulary.
func NewPostgresMicronutrientVocabularyRepository(db sqlExecutor) *PostgresMicronutrientVocabularyRepository {
	return &PostgresMicronutrientVocabularyRepository{db: db}
}

// ListActive returns active canonical vocabulary entries.
// Implements DESIGN-005 MicronutrientVocabulary.
func (r *PostgresMicronutrientVocabularyRepository) ListActive(ctx context.Context) ([]MicronutrientVocabularyEntry, error) {
	rows, err := r.db.Query(ctx, vocabularyListActiveSQL)
	if err != nil {
		return nil, mapPostgresError(err, "list active micronutrients")
	}
	defer rows.Close()

	entries := []MicronutrientVocabularyEntry{}
	for rows.Next() {
		var entry MicronutrientVocabularyEntry
		if err := rows.Scan(&entry.Key, &entry.DisplayName, &entry.Unit, &entry.Active); err != nil {
			return nil, mapPostgresError(err, "scan micronutrient")
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate micronutrients")
	}
	return entries, nil
}

// IsAllowed reports whether key exists as an active canonical vocabulary key.
// Implements DESIGN-005 MicronutrientVocabulary.
func (r *PostgresMicronutrientVocabularyRepository) IsAllowed(ctx context.Context, key string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, vocabularyIsAllowedSQL, key).Scan(&allowed)
	return allowed, mapPostgresError(err, "check micronutrient key")
}

// Upsert creates or updates a vocabulary entry.
// Implements DESIGN-005 MicronutrientVocabulary.
func (r *PostgresMicronutrientVocabularyRepository) Upsert(ctx context.Context, entry MicronutrientVocabularyEntry) error {
	if entry.Key == "" || entry.DisplayName == "" || entry.Unit == "" {
		return validationError("micronutrient key, display name, and unit are required")
	}
	_, err := r.db.Exec(ctx, vocabularyUpsertSQL, entry.Key, entry.DisplayName, entry.Unit, entry.Active)
	return mapPostgresError(err, "upsert micronutrient")
}

// ListAll returns active and inactive entries in deterministic canonical-key order.
// Implements DESIGN-005 MicronutrientVocabulary administrator listing.
func (r *PostgresMicronutrientVocabularyRepository) ListAll(ctx context.Context) ([]MicronutrientVocabularyEntry, error) {
	rows, err := r.db.Query(ctx, vocabularyListAllSQL)
	if err != nil {
		return nil, mapPostgresError(err, "list micronutrients")
	}
	defer rows.Close()
	entries := []MicronutrientVocabularyEntry{}
	for rows.Next() {
		var entry MicronutrientVocabularyEntry
		if err := rows.Scan(&entry.Key, &entry.DisplayName, &entry.Unit, &entry.Active); err != nil {
			return nil, mapPostgresError(err, "scan micronutrient")
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate micronutrients")
	}
	return entries, nil
}

// Get returns one canonical entry, including inactive entries.
// Implements DESIGN-005 MicronutrientVocabulary administrator management.
func (r *PostgresMicronutrientVocabularyRepository) Get(ctx context.Context, key string) (MicronutrientVocabularyEntry, error) {
	return r.scanEntry(r.db.QueryRow(ctx, vocabularyGetSQL, key), "get micronutrient")
}

// Create inserts a canonical entry or returns an exact existing replay.
// Implements DESIGN-005 MicronutrientVocabulary retry-safe creation.
func (r *PostgresMicronutrientVocabularyRepository) Create(ctx context.Context, entry MicronutrientVocabularyEntry) (MicronutrientVocabularyEntry, error) {
	result, err := r.scanEntry(r.db.QueryRow(ctx, vocabularyCreateSQL, entry.Key, entry.DisplayName, entry.Unit), "create micronutrient")
	if errors.Is(err, pgx.ErrNoRows) {
		return MicronutrientVocabularyEntry{}, NewError(ErrorKindConflict, "micronutrient key already exists with different values", nil)
	}
	return result, err
}

// UpdateDisplayName changes only the administrator-facing label.
// Implements DESIGN-005 MicronutrientVocabulary immutable canonical keys.
func (r *PostgresMicronutrientVocabularyRepository) UpdateDisplayName(ctx context.Context, key, displayName string) (MicronutrientVocabularyEntry, error) {
	return r.scanEntry(r.db.QueryRow(ctx, vocabularyUpdateDisplayNameSQL, key, displayName), "update micronutrient display name")
}

// UpdateUnit changes an unused entry's unit.
// Implements DESIGN-005 MicronutrientVocabulary in-use safeguard.
func (r *PostgresMicronutrientVocabularyRepository) UpdateUnit(ctx context.Context, key, unit string) (MicronutrientVocabularyEntry, error) {
	if err := r.lockUsageTables(ctx); err != nil {
		return MicronutrientVocabularyEntry{}, err
	}
	inUse, err := r.isInUse(ctx, key)
	if err != nil {
		return MicronutrientVocabularyEntry{}, err
	}
	if inUse {
		return MicronutrientVocabularyEntry{}, NewError(ErrorKindConflict, "micronutrient is in use", nil)
	}
	return r.scanEntry(r.db.QueryRow(ctx, vocabularyUpdateUnitSQL, key, unit), "update micronutrient unit")
}

// SetActive deactivates only unused entries and always permits reactivation.
// Implements DESIGN-005 MicronutrientVocabulary lifecycle.
func (r *PostgresMicronutrientVocabularyRepository) SetActive(ctx context.Context, key string, active bool) (MicronutrientVocabularyEntry, error) {
	if !active {
		if err := r.lockUsageTables(ctx); err != nil {
			return MicronutrientVocabularyEntry{}, err
		}
		inUse, err := r.isInUse(ctx, key)
		if err != nil {
			return MicronutrientVocabularyEntry{}, err
		}
		if inUse {
			return MicronutrientVocabularyEntry{}, NewError(ErrorKindConflict, "micronutrient is in use", nil)
		}
	}
	return r.scanEntry(r.db.QueryRow(ctx, vocabularySetActiveSQL, key, active), "set micronutrient active state")
}

// lockUsageTables serializes guarded changes with global and private food writes.
// Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
func (r *PostgresMicronutrientVocabularyRepository) lockUsageTables(ctx context.Context) error {
	return lockMicronutrientUsageTables(ctx, r.db)
}

// lockMicronutrientUsageTables serializes vocabulary guards with every item write.
// Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
func lockMicronutrientUsageTables(ctx context.Context, db sqlExecutor) error {
	_, err := db.Exec(ctx, vocabularyLockUsageTablesSQL)
	return mapPostgresError(err, "lock micronutrient usage")
}

// lockMicronutrientItemWriteTables coordinates item DML with vocabulary guards.
// Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
func lockMicronutrientItemWriteTables(ctx context.Context, db sqlExecutor) error {
	_, err := db.Exec(ctx, vocabularyLockItemWriteTablesSQL)
	return mapPostgresError(err, "lock micronutrient item writes")
}

// isInUse reports whether either food-item store contains the canonical key.
// Implements DESIGN-005 MicronutrientVocabulary global/private usage guard.
func (r *PostgresMicronutrientVocabularyRepository) isInUse(ctx context.Context, key string) (bool, error) {
	var inUse bool
	err := r.db.QueryRow(ctx, vocabularyIsInUseSQL, key).Scan(&inUse)
	return inUse, mapPostgresError(err, "check micronutrient usage")
}

// scanEntry maps one canonical vocabulary row without exposing database diagnostics.
// Implements DESIGN-005 MicronutrientVocabulary repository projection.
func (r *PostgresMicronutrientVocabularyRepository) scanEntry(row pgx.Row, operation string) (MicronutrientVocabularyEntry, error) {
	var entry MicronutrientVocabularyEntry
	err := row.Scan(&entry.Key, &entry.DisplayName, &entry.Unit, &entry.Active)
	return entry, mapPostgresError(err, operation)
}

package repository

import (
	"context"
	_ "embed"
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

// PostgresMicronutrientVocabularyRepository persists canonical micronutrient keys.
// Implements DESIGN-005 MicronutrientVocabulary.
type PostgresMicronutrientVocabularyRepository struct {
	db sqlExecutor
}

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

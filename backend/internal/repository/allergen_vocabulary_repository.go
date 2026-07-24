package repository

import (
	"context"
	_ "embed"
)

// Implements DESIGN-009 TagManager active allergen vocabulary query.
//
//go:embed sql/allergen_vocabulary_list.sql
var allergenVocabularyListSQL string

// AllergenVocabularyEntry is one active backend-owned allergen filter identity.
// Implements DESIGN-009 TagManager filter-option vocabulary.
type AllergenVocabularyEntry struct {
	Key      string
	Name     string
	LabelKey string
}

// AllergenVocabularyRepository defines active persisted allergen vocabulary reads.
// Implements DESIGN-009 TagManager filter-option vocabulary.
type AllergenVocabularyRepository interface {
	ListActive(ctx context.Context) ([]AllergenVocabularyEntry, error)
}

// PostgresAllergenVocabularyRepository reads persisted allergen filter identities.
// Implements DESIGN-009 TagManager filter-option vocabulary.
type PostgresAllergenVocabularyRepository struct {
	db sqlExecutor
}

// Implements DESIGN-009 TagManager compile-time allergen vocabulary contract.
var _ AllergenVocabularyRepository = (*PostgresAllergenVocabularyRepository)(nil)

// NewPostgresAllergenVocabularyRepository creates a PostgreSQL-backed allergen vocabulary reader.
// Implements DESIGN-009 TagManager filter-option vocabulary.
func NewPostgresAllergenVocabularyRepository(db sqlExecutor) *PostgresAllergenVocabularyRepository {
	return &PostgresAllergenVocabularyRepository{db: db}
}

// ListActive returns active allergen identities in deterministic display order.
// Implements DESIGN-009 TagManager filter-option vocabulary.
func (r *PostgresAllergenVocabularyRepository) ListActive(ctx context.Context) ([]AllergenVocabularyEntry, error) {
	rows, err := r.db.Query(ctx, allergenVocabularyListSQL)
	if err != nil {
		return nil, mapPostgresError(err, "list allergen vocabulary")
	}
	defer rows.Close()

	entries := []AllergenVocabularyEntry{}
	for rows.Next() {
		var entry AllergenVocabularyEntry
		if err := rows.Scan(&entry.Key, &entry.Name, &entry.LabelKey); err != nil {
			return nil, mapPostgresError(err, "scan allergen vocabulary")
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate allergen vocabulary")
	}
	return entries, nil
}

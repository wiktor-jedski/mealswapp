package postgres

import (
	"context"

	"mealswapp/backend/internal/domain/micronutrient"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MicronutrientVocabularyRepository struct {
	db DBTX
}

func NewMicronutrientVocabularyRepository(pool *pgxpool.Pool) MicronutrientVocabularyRepository {
	return NewMicronutrientVocabularyRepositoryWithDB(pool)
}

func NewMicronutrientVocabularyRepositoryWithDB(db DBTX) MicronutrientVocabularyRepository {
	return MicronutrientVocabularyRepository{db: db}
}

func (repo MicronutrientVocabularyRepository) ListActive(ctx context.Context) ([]micronutrient.Entry, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT key, display_name, unit, active
		FROM micronutrient_vocabulary
		WHERE active = true
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []micronutrient.Entry
	for rows.Next() {
		var entry micronutrient.Entry
		if err := rows.Scan(&entry.Key, &entry.DisplayName, &entry.Unit, &entry.Active); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (repo MicronutrientVocabularyRepository) IsAllowed(ctx context.Context, key string) (bool, error) {
	var allowed bool
	err := repo.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM micronutrient_vocabulary
			WHERE key = $1 AND active = true
		)
	`, key).Scan(&allowed)
	if err != nil {
		return false, err
	}

	return allowed, nil
}

func (repo MicronutrientVocabularyRepository) Upsert(ctx context.Context, entry micronutrient.Entry) error {
	if err := entry.Validate(); err != nil {
		return err
	}

	_, err := repo.db.Exec(ctx, `
		INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key)
		DO UPDATE SET display_name = excluded.display_name, unit = excluded.unit, active = excluded.active, updated_at = now()
	`, entry.Key, entry.DisplayName, entry.Unit, entry.Active)
	return err
}

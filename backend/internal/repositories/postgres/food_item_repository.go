package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FoodItemRepository struct {
	db DBTX
}

func NewFoodItemRepository(pool *pgxpool.Pool) FoodItemRepository {
	return NewFoodItemRepositoryWithDB(pool)
}

func NewFoodItemRepositoryWithDB(db DBTX) FoodItemRepository {
	return FoodItemRepository{db: db}
}

func (repo FoodItemRepository) Create(ctx context.Context, item food.FoodItemEntity) (uuid.UUID, error) {
	if err := item.Validate(); err != nil {
		return uuid.Nil, err
	}

	micros, err := json.Marshal(item.Micros)
	if err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	err = repo.db.QueryRow(ctx, `
		INSERT INTO food_items (
			name,
			physical_state,
			serving_unit,
			serving_size,
			calories_per_100,
			protein_grams_per_100,
			carbs_grams_per_100,
			fat_grams_per_100,
			micronutrients,
			source_provider,
			source_external_id,
			source_url,
			source_imported_at,
			curation_state,
			image_url,
			prep_time_minutes,
			average_unit_weight_grams
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, coalesce(nullif($14, ''), 'draft'), $15, $16, $17)
		RETURNING id
	`, item.Name, item.PhysicalState, item.ServingUnit, item.ServingSize, item.CaloriesPer100, item.MacrosPer100.ProteinGrams, item.MacrosPer100.CarbsGrams, item.MacrosPer100.FatGrams, micros, nullableString(item.Source.Provider), nullableString(item.Source.ExternalID), nullableString(item.Source.ProviderURL), item.Source.ImportedAt, item.Source.CurationState, nullableString(item.ImageURL), item.PrepTimeMinutes, item.AverageUnitWeightGrams).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (repo FoodItemRepository) GetByID(ctx context.Context, id uuid.UUID, _ repositories.RepositoryContext) (food.FoodItemEntity, error) {
	var item food.FoodItemEntity
	var micros []byte
	var sourceProvider *string
	var sourceExternalID *string
	var sourceURL *string
	err := repo.db.QueryRow(ctx, `
		SELECT
			id, name, physical_state, serving_unit, serving_size, calories_per_100,
			protein_grams_per_100, carbs_grams_per_100, fat_grams_per_100,
			micronutrients, source_provider, source_external_id, source_url, source_imported_at,
			curation_state, coalesce(image_url, ''), prep_time_minutes, average_unit_weight_grams,
			created_at, updated_at
		FROM food_items
		WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.PhysicalState, &item.ServingUnit, &item.ServingSize, &item.CaloriesPer100, &item.MacrosPer100.ProteinGrams, &item.MacrosPer100.CarbsGrams, &item.MacrosPer100.FatGrams, &micros, &sourceProvider, &sourceExternalID, &sourceURL, &item.Source.ImportedAt, &item.Source.CurationState, &item.ImageURL, &item.PrepTimeMinutes, &item.AverageUnitWeightGrams, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return food.FoodItemEntity{}, err
	}
	if err := json.Unmarshal(micros, &item.Micros); err != nil {
		return food.FoodItemEntity{}, err
	}
	item.Source.Provider = stringValue(sourceProvider)
	item.Source.ExternalID = stringValue(sourceExternalID)
	item.Source.ProviderURL = stringValue(sourceURL)

	return item, nil
}

func (repo FoodItemRepository) Search(ctx context.Context, q repositories.FoodItemQuery) ([]food.FoodItemEntity, int, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	rows, err := repo.db.Query(ctx, `
		SELECT id
		FROM food_items
		WHERE ($1 = '' OR normalized_name LIKE '%' || lower(trim($1)) || '%')
		ORDER BY normalized_name ASC
		LIMIT $2 OFFSET $3
	`, q.Text, limit, q.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []food.FoodItemEntity
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		item, err := repo.GetByID(ctx, id, repositories.RepositoryContext{})
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := repo.db.QueryRow(ctx, `SELECT count(*) FROM food_items WHERE ($1 = '' OR normalized_name LIKE '%' || lower(trim($1)) || '%')`, q.Text).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (repo FoodItemRepository) Update(ctx context.Context, item food.FoodItemEntity) error {
	if item.ID == uuid.Nil {
		return errors.New("food item id is required")
	}
	if err := item.Validate(); err != nil {
		return err
	}
	micros, err := json.Marshal(item.Micros)
	if err != nil {
		return err
	}

	tag, err := repo.db.Exec(ctx, `
		UPDATE food_items
		SET name = $2,
			physical_state = $3,
			serving_unit = $4,
			serving_size = $5,
			calories_per_100 = $6,
			protein_grams_per_100 = $7,
			carbs_grams_per_100 = $8,
			fat_grams_per_100 = $9,
			micronutrients = $10,
			image_url = $11,
			prep_time_minutes = $12,
			average_unit_weight_grams = $13,
			updated_at = now()
		WHERE id = $1
	`, item.ID, item.Name, item.PhysicalState, item.ServingUnit, item.ServingSize, item.CaloriesPer100, item.MacrosPer100.ProteinGrams, item.MacrosPer100.CarbsGrams, item.MacrosPer100.FatGrams, micros, nullableString(item.ImageURL), item.PrepTimeMinutes, item.AverageUnitWeightGrams)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo FoodItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM food_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

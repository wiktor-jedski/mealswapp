package postgres

import (
	"context"

	"mealswapp/backend/internal/domain/tag"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	db DBTX
}

type FoodItemTagFilter = repositories.FoodItemTagFilter

func NewTagRepository(pool *pgxpool.Pool) TagRepository {
	return NewTagRepositoryWithDB(pool)
}

func NewTagRepositoryWithDB(db DBTX) TagRepository {
	return TagRepository{db: db}
}

func (repo TagRepository) Upsert(ctx context.Context, entity tag.TagEntity) (uuid.UUID, error) {
	if err := entity.Validate(); err != nil {
		return uuid.Nil, err
	}

	active := entity.Active
	if !active {
		active = true
	}

	var id uuid.UUID
	err := repo.db.QueryRow(ctx, `
		INSERT INTO tags (name, kind, parent_id, active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (kind, normalized_name)
		DO UPDATE SET parent_id = excluded.parent_id, active = excluded.active, updated_at = now()
		RETURNING id
	`, entity.Name, entity.Kind, entity.ParentID, active).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (repo TagRepository) List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT id, name, kind, parent_id, active, created_at, updated_at
		FROM tags
		WHERE kind = $1
		ORDER BY normalized_name ASC
	`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []tag.TagEntity
	for rows.Next() {
		var entity tag.TagEntity
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.Kind, &entity.ParentID, &entity.Active, &entity.CreatedAt, &entity.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func (repo TagRepository) AttachToFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	_, err := repo.db.Exec(ctx, `
		INSERT INTO food_item_tags (food_item_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, foodItemID, tagID)
	return err
}

func (repo TagRepository) RemoveFromFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	result, err := repo.db.Exec(ctx, `
		DELETE FROM food_item_tags
		WHERE food_item_id = $1 AND tag_id = $2
	`, foodItemID, tagID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}

	return nil
}

func (repo TagRepository) QueryFoodItemIDs(ctx context.Context, filter FoodItemTagFilter) ([]uuid.UUID, error) {
	return repo.queryFoodItemIDs(ctx, filter.IncludeTagIDs, filter.ExcludeTagIDs)
}

func (repo TagRepository) queryFoodItemIDs(ctx context.Context, includeTagIDs []uuid.UUID, excludeTagIDs []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT fi.id
		FROM food_items fi
		WHERE (
			cardinality($1::uuid[]) = 0
			OR NOT EXISTS (
				SELECT 1
				FROM unnest($1::uuid[]) include_tag(id)
				WHERE NOT EXISTS (
					SELECT 1
					FROM food_item_tags fit
					WHERE fit.food_item_id = fi.id
					AND fit.tag_id = include_tag.id
				)
			)
		)
		AND (
			cardinality($2::uuid[]) = 0
			OR NOT EXISTS (
				SELECT 1
				FROM food_item_tags fit
				WHERE fit.food_item_id = fi.id
				AND fit.tag_id = ANY($2::uuid[])
			)
		)
		ORDER BY fi.normalized_name ASC
	`, includeTagIDs, excludeTagIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

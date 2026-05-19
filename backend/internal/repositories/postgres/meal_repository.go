package postgres

import (
	"context"
	"errors"
	"fmt"

	"mealswapp/backend/internal/domain/meal"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MealRepository struct {
	db   DBTX
	pool *pgxpool.Pool
}

func NewMealRepository(pool *pgxpool.Pool) MealRepository {
	return NewMealRepositoryWithDB(pool, pool)
}

func NewMealRepositoryWithDB(db DBTX, pool *pgxpool.Pool) MealRepository {
	return MealRepository{db: db, pool: pool}
}

func (repo MealRepository) Create(ctx context.Context, entity meal.MealEntity) (uuid.UUID, error) {
	if err := entity.Validate(); err != nil {
		return uuid.Nil, err
	}

	if repo.pool == nil {
		var id uuid.UUID
		err := repo.db.QueryRow(ctx, `
			INSERT INTO meals (user_id, name, meal_type)
			VALUES ($1, $2, $3)
			RETURNING id
		`, entity.UserID, entity.Name, entity.Type).Scan(&id)
		if err != nil {
			return uuid.Nil, err
		}
		if err := insertMealItems(ctx, repo.db, id, entity.Items); err != nil {
			return uuid.Nil, err
		}
		return id, nil
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO meals (user_id, name, meal_type)
		VALUES ($1, $2, $3)
		RETURNING id
	`, entity.UserID, entity.Name, entity.Type).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	if err := insertMealItems(ctx, tx, id, entity.Items); err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (repo MealRepository) GetByID(ctx context.Context, id uuid.UUID) (meal.MealEntity, error) {
	var entity meal.MealEntity
	err := repo.db.QueryRow(ctx, `
		SELECT id, user_id, name, meal_type, created_at, updated_at
		FROM meals
		WHERE id = $1
	`, id).Scan(&entity.ID, &entity.UserID, &entity.Name, &entity.Type, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		return meal.MealEntity{}, err
	}

	items, err := repo.listItems(ctx, id)
	if err != nil {
		return meal.MealEntity{}, err
	}
	entity.Items = items

	return entity, nil
}

func (repo MealRepository) Update(ctx context.Context, entity meal.MealEntity) error {
	if entity.ID == uuid.Nil {
		return errors.New("meal id is required")
	}
	if err := entity.Validate(); err != nil {
		return err
	}

	if repo.pool == nil {
		tag, err := repo.db.Exec(ctx, `
			UPDATE meals
			SET name = $2, meal_type = $3, updated_at = now()
			WHERE id = $1
		`, entity.ID, entity.Name, entity.Type)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return pgx.ErrNoRows
		}
		if _, err := repo.db.Exec(ctx, `DELETE FROM meal_items WHERE meal_id = $1`, entity.ID); err != nil {
			return err
		}
		return insertMealItems(ctx, repo.db, entity.ID, entity.Items)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE meals
		SET name = $2, meal_type = $3, updated_at = now()
		WHERE id = $1
	`, entity.ID, entity.Name, entity.Type)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `DELETE FROM meal_items WHERE meal_id = $1`, entity.ID); err != nil {
		return err
	}

	if err := insertMealItems(ctx, tx, entity.ID, entity.Items); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo MealRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM meals WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}

	return nil
}

func (repo MealRepository) listItems(ctx context.Context, mealID uuid.UUID) ([]meal.MealItemEntity, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT id, meal_id, food_item_id, quantity, unit, position
		FROM meal_items
		WHERE meal_id = $1
		ORDER BY position ASC
	`, mealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []meal.MealItemEntity
	for rows.Next() {
		var item meal.MealItemEntity
		if err := rows.Scan(&item.ID, &item.MealID, &item.FoodItemID, &item.Quantity, &item.Unit, &item.Position); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

type mealItemInserter interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func insertMealItems(ctx context.Context, tx mealItemInserter, mealID uuid.UUID, items []meal.MealItemEntity) error {
	for index, item := range items {
		position := item.Position
		if position == 0 && index > 0 {
			position = index
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO meal_items (meal_id, food_item_id, quantity, unit, position)
			VALUES ($1, $2, $3, $4, $5)
		`, mealID, item.FoodItemID, item.Quantity, item.Unit, position)
		if err != nil {
			return fmt.Errorf("insert meal item %d: %w", index, err)
		}
	}

	return nil
}

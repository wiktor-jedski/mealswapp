package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/recipe"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecipeRepository struct {
	db   DBTX
	pool *pgxpool.Pool
}

type recipeTotals struct {
	Calories float64
	Macros   food.MacroValues
}

func NewRecipeRepository(pool *pgxpool.Pool) RecipeRepository {
	return NewRecipeRepositoryWithDB(pool, pool)
}

func NewRecipeRepositoryWithDB(db DBTX, pool *pgxpool.Pool) RecipeRepository {
	return RecipeRepository{db: db, pool: pool}
}

func (repo RecipeRepository) Create(ctx context.Context, entity recipe.RecipeEntity) (uuid.UUID, error) {
	totals, err := repo.CalculateTotals(ctx, entity.Ingredients)
	if err != nil {
		return uuid.Nil, err
	}
	entity.CaloriesTotal = totals.Calories
	entity.MacrosTotal = totals.Macros

	if err := entity.Validate(); err != nil {
		return uuid.Nil, err
	}

	if repo.pool == nil {
		var id uuid.UUID
		err := repo.db.QueryRow(ctx, `
			INSERT INTO recipes (
				user_id,
				name,
				calories_total,
				protein_grams_total,
				carbs_grams_total,
				fat_grams_total,
				source_provider,
				source_external_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`, entity.UserID, entity.Name, entity.CaloriesTotal, entity.MacrosTotal.ProteinGrams, entity.MacrosTotal.CarbsGrams, entity.MacrosTotal.FatGrams, nullableString(entity.SourceProvider), nullableString(entity.SourceID)).Scan(&id)
		if err != nil {
			return uuid.Nil, err
		}
		if err := insertRecipeIngredients(ctx, repo.db, id, entity.Ingredients); err != nil {
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
		INSERT INTO recipes (
			user_id,
			name,
			calories_total,
			protein_grams_total,
			carbs_grams_total,
			fat_grams_total,
			source_provider,
			source_external_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, entity.UserID, entity.Name, entity.CaloriesTotal, entity.MacrosTotal.ProteinGrams, entity.MacrosTotal.CarbsGrams, entity.MacrosTotal.FatGrams, nullableString(entity.SourceProvider), nullableString(entity.SourceID)).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	if err := insertRecipeIngredients(ctx, tx, id, entity.Ingredients); err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (repo RecipeRepository) GetByID(ctx context.Context, id uuid.UUID) (recipe.RecipeEntity, error) {
	var entity recipe.RecipeEntity
	err := repo.db.QueryRow(ctx, `
		SELECT
			id,
			user_id,
			name,
			calories_total,
			protein_grams_total,
			carbs_grams_total,
			fat_grams_total,
			coalesce(source_provider, ''),
			coalesce(source_external_id, ''),
			created_at,
			updated_at
		FROM recipes
		WHERE id = $1
	`, id).Scan(
		&entity.ID,
		&entity.UserID,
		&entity.Name,
		&entity.CaloriesTotal,
		&entity.MacrosTotal.ProteinGrams,
		&entity.MacrosTotal.CarbsGrams,
		&entity.MacrosTotal.FatGrams,
		&entity.SourceProvider,
		&entity.SourceID,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		return recipe.RecipeEntity{}, err
	}

	ingredients, err := repo.listIngredients(ctx, id)
	if err != nil {
		return recipe.RecipeEntity{}, err
	}
	entity.Ingredients = ingredients

	return entity, nil
}

func (repo RecipeRepository) Update(ctx context.Context, entity recipe.RecipeEntity) error {
	if entity.ID == uuid.Nil {
		return errors.New("recipe id is required")
	}

	totals, err := repo.CalculateTotals(ctx, entity.Ingredients)
	if err != nil {
		return err
	}
	entity.CaloriesTotal = totals.Calories
	entity.MacrosTotal = totals.Macros

	if err := entity.Validate(); err != nil {
		return err
	}

	if repo.pool == nil {
		tag, err := repo.db.Exec(ctx, `
			UPDATE recipes
			SET
				name = $2,
				calories_total = $3,
				protein_grams_total = $4,
				carbs_grams_total = $5,
				fat_grams_total = $6,
				source_provider = $7,
				source_external_id = $8,
				updated_at = now()
			WHERE id = $1
		`, entity.ID, entity.Name, entity.CaloriesTotal, entity.MacrosTotal.ProteinGrams, entity.MacrosTotal.CarbsGrams, entity.MacrosTotal.FatGrams, nullableString(entity.SourceProvider), nullableString(entity.SourceID))
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return pgx.ErrNoRows
		}
		if _, err := repo.db.Exec(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = $1`, entity.ID); err != nil {
			return err
		}
		return insertRecipeIngredients(ctx, repo.db, entity.ID, entity.Ingredients)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE recipes
		SET
			name = $2,
			calories_total = $3,
			protein_grams_total = $4,
			carbs_grams_total = $5,
			fat_grams_total = $6,
			source_provider = $7,
			source_external_id = $8,
			updated_at = now()
		WHERE id = $1
	`, entity.ID, entity.Name, entity.CaloriesTotal, entity.MacrosTotal.ProteinGrams, entity.MacrosTotal.CarbsGrams, entity.MacrosTotal.FatGrams, nullableString(entity.SourceProvider), nullableString(entity.SourceID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = $1`, entity.ID); err != nil {
		return err
	}

	if err := insertRecipeIngredients(ctx, tx, entity.ID, entity.Ingredients); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo RecipeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}

	return nil
}

func (repo RecipeRepository) CalculateTotals(ctx context.Context, ingredients []recipe.RecipeIngredientEntity) (recipeTotals, error) {
	var totals recipeTotals
	for _, ingredient := range ingredients {
		if err := ingredient.Validate(); err != nil {
			return recipeTotals{}, err
		}

		var calories float64
		var protein float64
		var carbs float64
		var fat float64
		var servingSize float64
		err := repo.db.QueryRow(ctx, `
			SELECT calories_per_100, protein_grams_per_100, carbs_grams_per_100, fat_grams_per_100, serving_size
			FROM food_items
			WHERE id = $1
		`, ingredient.FoodItemID).Scan(&calories, &protein, &carbs, &fat, &servingSize)
		if err != nil {
			return recipeTotals{}, err
		}

		scale := ingredient.Quantity / 100
		switch ingredient.Unit {
		case "piece", "serving":
			scale = (ingredient.Quantity * servingSize) / 100
		}

		totals.Calories += calories * scale
		totals.Macros.ProteinGrams += protein * scale
		totals.Macros.CarbsGrams += carbs * scale
		totals.Macros.FatGrams += fat * scale
	}

	totals.Calories = round3(totals.Calories)
	totals.Macros.ProteinGrams = round3(totals.Macros.ProteinGrams)
	totals.Macros.CarbsGrams = round3(totals.Macros.CarbsGrams)
	totals.Macros.FatGrams = round3(totals.Macros.FatGrams)

	return totals, nil
}

func (repo RecipeRepository) listIngredients(ctx context.Context, recipeID uuid.UUID) ([]recipe.RecipeIngredientEntity, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT id, recipe_id, food_item_id, quantity, unit, position
		FROM recipe_ingredients
		WHERE recipe_id = $1
		ORDER BY position ASC
	`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ingredients []recipe.RecipeIngredientEntity
	for rows.Next() {
		var ingredient recipe.RecipeIngredientEntity
		if err := rows.Scan(&ingredient.ID, &ingredient.RecipeID, &ingredient.FoodItemID, &ingredient.Quantity, &ingredient.Unit, &ingredient.Position); err != nil {
			return nil, err
		}
		ingredients = append(ingredients, ingredient)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ingredients, nil
}

type recipeIngredientInserter interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func insertRecipeIngredients(ctx context.Context, tx recipeIngredientInserter, recipeID uuid.UUID, ingredients []recipe.RecipeIngredientEntity) error {
	for index, ingredient := range ingredients {
		position := ingredient.Position
		if position == 0 && index > 0 {
			position = index
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, food_item_id, quantity, unit, position)
			VALUES ($1, $2, $3, $4, $5)
		`, recipeID, ingredient.FoodItemID, ingredient.Quantity, ingredient.Unit, position)
		if err != nil {
			return fmt.Errorf("insert recipe ingredient %d: %w", index, err)
		}
	}

	return nil
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

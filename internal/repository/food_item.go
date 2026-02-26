// Phase: phase-01 | Task: 6 | Architecture: ARCH-005 | Design: FoodItemEntity
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"mealswapp/internal/models"
)

type foodItemRepository struct {
	db *pgxpool.Pool
}

func NewFoodItemRepository(db *pgxpool.Pool) FoodItemRepository {
	return &foodItemRepository{db: db}
}

func (r *foodItemRepository) Create(ctx context.Context, item *models.FoodItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	macrosJSON, err := json.Marshal(item.Macros)
	if err != nil {
		return fmt.Errorf("failed to marshal macros: %w", err)
	}

	microsJSON, err := json.Marshal(item.Micros)
	if err != nil {
		return fmt.Errorf("failed to marshal micros: %w", err)
	}

	query := `
		INSERT INTO food_items (
			id, name, physical_state, prep_time, average_unit_weight,
			macros, micros, image_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.Exec(ctx, query,
		item.ID,
		item.Name,
		item.PhysicalState,
		item.PrepTime,
		item.AverageUnitWeight,
		macrosJSON,
		microsJSON,
		item.ImageURL,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert food item: %w", err)
	}

	if len(item.CategoryTags) > 0 {
		for _, tag := range item.CategoryTags {
			_, err = tx.Exec(ctx,
				`INSERT INTO food_item_category_tags (food_item_id, tag_id) VALUES ($1, $2)
				 ON CONFLICT DO NOTHING`,
				item.ID, tag.ID)
			if err != nil {
				return fmt.Errorf("failed to insert category tag: %w", err)
			}
		}
	}

	if len(item.FunctionalityTags) > 0 {
		for _, tag := range item.FunctionalityTags {
			_, err = tx.Exec(ctx,
				`INSERT INTO food_item_functionality_tags (food_item_id, tag_id) VALUES ($1, $2)
				 ON CONFLICT DO NOTHING`,
				item.ID, tag.ID)
			if err != nil {
				return fmt.Errorf("failed to insert functionality tag: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *foodItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FoodItem, error) {
	query := `
		SELECT id, name, physical_state, prep_time, average_unit_weight,
		       macros, micros, image_url, created_at, updated_at
		FROM food_items WHERE id = $1
	`

	var item models.FoodItem
	var macrosJSON, microsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.PhysicalState,
		&item.PrepTime,
		&item.AverageUnitWeight,
		&macrosJSON,
		&microsJSON,
		&item.ImageURL,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("food item not found")
		}
		return nil, fmt.Errorf("failed to get food item: %w", err)
	}

	if err := json.Unmarshal(macrosJSON, &item.Macros); err != nil {
		return nil, fmt.Errorf("failed to unmarshal macros: %w", err)
	}
	if err := json.Unmarshal(microsJSON, &item.Micros); err != nil {
		return nil, fmt.Errorf("failed to unmarshal micros: %w", err)
	}

	categoryTags, err := r.getTagsByFoodItemID(ctx, id, "category")
	if err != nil {
		return nil, err
	}
	item.CategoryTags = categoryTags

	functionalityTags, err := r.getTagsByFoodItemID(ctx, id, "functionality")
	if err != nil {
		return nil, err
	}
	item.FunctionalityTags = functionalityTags

	return &item, nil
}

func (r *foodItemRepository) getTagsByFoodItemID(ctx context.Context, foodItemID uuid.UUID, tagType string) ([]models.Tag, error) {
	var query string
	var rows pgx.Rows
	var err error

	if tagType == "category" {
		query = `
			SELECT t.id, t.name, t.slug, t.type, t.description, t.color_hex, t.icon_url, t.created_at, t.updated_at
			FROM tags t
			JOIN food_item_category_tags fct ON t.id = fct.tag_id
			WHERE fct.food_item_id = $1
		`
	} else {
		query = `
			SELECT t.id, t.name, t.slug, t.type, t.description, t.color_hex, t.icon_url, t.created_at, t.updated_at
			FROM tags t
			JOIN food_item_functionality_tags fft ON t.id = fft.tag_id
			WHERE fft.food_item_id = $1
		`
	}

	rows, err = r.db.Query(ctx, query, foodItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Slug,
			&tag.Type,
			&tag.Description,
			&tag.ColorHex,
			&tag.IconURL,
			&tag.CreatedAt,
			&tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (r *foodItemRepository) List(ctx context.Context, query models.FoodItemQuery) ([]*models.FoodItem, int64, error) {
	baseQuery := `FROM food_items WHERE 1=1`
	var args []interface{}
	argIndex := 1

	if len(query.IDs) > 0 {
		baseQuery += fmt.Sprintf(" AND id = ANY($%d)", argIndex)
		args = append(args, query.IDs)
		argIndex++
	}

	if query.Name != nil {
		baseQuery += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
		args = append(args, "%"+*query.Name+"%")
		argIndex++
	}

	if query.PhysicalState != nil {
		baseQuery += fmt.Sprintf(" AND physical_state = $%d", argIndex)
		args = append(args, *query.PhysicalState)
		argIndex++
	}

	if len(query.CategoryTagIDs) > 0 {
		baseQuery += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM food_item_category_tags fct
			WHERE fct.food_item_id = food_items.id
			AND fct.tag_id = ANY($%d)
		)`, argIndex)
		args = append(args, query.CategoryTagIDs)
		argIndex++
	}

	if len(query.FunctionalityTagIDs) > 0 {
		baseQuery += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM food_item_functionality_tags fft
			WHERE fft.food_item_id = food_items.id
			AND fft.tag_id = ANY($%d)
		)`, argIndex)
		args = append(args, query.FunctionalityTagIDs)
		argIndex++
	}

	if query.MinProtein != nil {
		baseQuery += fmt.Sprintf(" AND (macros->>'protein')::float >= $%d", argIndex)
		args = append(args, *query.MinProtein)
		argIndex++
	}

	if query.MaxProtein != nil {
		baseQuery += fmt.Sprintf(" AND (macros->>'protein')::float <= $%d", argIndex)
		args = append(args, *query.MaxProtein)
		argIndex++
	}

	if query.MinCarbs != nil {
		baseQuery += fmt.Sprintf(" AND (macros->>'carbs')::float >= $%d", argIndex)
		args = append(args, *query.MinCarbs)
		argIndex++
	}

	if query.MaxCarbs != nil {
		baseQuery += fmt.Sprintf(" AND (macros->>'carbs')::float <= $%d", argIndex)
		args = append(args, *query.MaxCarbs)
		argIndex++
	}

	if query.MinFat != nil {
		baseQuery += fmt.Sprintf(" AND (macros->>'fat')::float >= $%d", argIndex)
		args = append(args, *query.MinFat)
		argIndex++
	}

	if query.MaxFat != nil {
		baseQuery += fmt.Sprintf(" AND (macros->>'fat')::float <= $%d", argIndex)
		args = append(args, *query.MaxFat)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count food items: %w", err)
	}

	sortColumn := "name"
	switch query.SortBy {
	case "prep_time", "created_at", "protein", "carbs", "fat":
		if query.SortBy == "protein" || query.SortBy == "carbs" || query.SortBy == "fat" {
			sortColumn = fmt.Sprintf("macros->>'%s'", query.SortBy)
		} else {
			sortColumn = query.SortBy
		}
	}

	sortOrder := "ASC"
	if strings.ToUpper(query.SortOrder) == "DESC" {
		sortOrder = "DESC"
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	offset := (query.Page - 1) * query.PageSize

	selectQuery := fmt.Sprintf(`
		SELECT id, name, physical_state, prep_time, average_unit_weight,
		       macros, micros, image_url, created_at, updated_at
		%s ORDER BY %s %s LIMIT $%d OFFSET $%d
	`, baseQuery, sortColumn, sortOrder, argIndex, argIndex+1)

	args = append(args, query.PageSize, offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list food items: %w", err)
	}
	defer rows.Close()

	var items []*models.FoodItem
	for rows.Next() {
		var item models.FoodItem
		var macrosJSON, microsJSON []byte

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.PhysicalState,
			&item.PrepTime,
			&item.AverageUnitWeight,
			&macrosJSON,
			&microsJSON,
			&item.ImageURL,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan food item: %w", err)
		}

		if err := json.Unmarshal(macrosJSON, &item.Macros); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal macros: %w", err)
		}
		if err := json.Unmarshal(microsJSON, &item.Micros); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal micros: %w", err)
		}

		categoryTags, err := r.getTagsByFoodItemID(ctx, item.ID, "category")
		if err != nil {
			return nil, 0, err
		}
		item.CategoryTags = categoryTags

		functionalityTags, err := r.getTagsByFoodItemID(ctx, item.ID, "functionality")
		if err != nil {
			return nil, 0, err
		}
		item.FunctionalityTags = functionalityTags

		items = append(items, &item)
	}

	return items, total, nil
}

func (r *foodItemRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var setClauses []string
	var args []interface{}
	argIndex := 1

	if name, ok := updates["name"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	if physicalState, ok := updates["physical_state"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("physical_state = $%d", argIndex))
		args = append(args, physicalState)
		argIndex++
	}

	if prepTime, ok := updates["prep_time"].(int); ok {
		setClauses = append(setClauses, fmt.Sprintf("prep_time = $%d", argIndex))
		args = append(args, prepTime)
		argIndex++
	}

	if avgUnitWeight, ok := updates["average_unit_weight"].(float64); ok {
		setClauses = append(setClauses, fmt.Sprintf("average_unit_weight = $%d", argIndex))
		args = append(args, avgUnitWeight)
		argIndex++
	}

	if macros, ok := updates["macros"].(models.Macros); ok {
		macrosJSON, err := json.Marshal(macros)
		if err != nil {
			return fmt.Errorf("failed to marshal macros: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("macros = $%d", argIndex))
		args = append(args, macrosJSON)
		argIndex++
	}

	if micros, ok := updates["micros"].(models.Micros); ok {
		microsJSON, err := json.Marshal(micros)
		if err != nil {
			return fmt.Errorf("failed to marshal micros: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("micros = $%d", argIndex))
		args = append(args, microsJSON)
		argIndex++
	}

	if imageURL, ok := updates["image_url"].(*string); ok {
		setClauses = append(setClauses, fmt.Sprintf("image_url = $%d", argIndex))
		args = append(args, imageURL)
		argIndex++
	}

	if categoryTagIDs, ok := updates["category_tag_ids"].([]uuid.UUID); ok {
		_, err := tx.Exec(ctx, "DELETE FROM food_item_category_tags WHERE food_item_id = $1", id)
		if err != nil {
			return fmt.Errorf("failed to delete category tags: %w", err)
		}

		for _, tagID := range categoryTagIDs {
			_, err = tx.Exec(ctx,
				`INSERT INTO food_item_category_tags (food_item_id, tag_id) VALUES ($1, $2)`,
				id, tagID)
			if err != nil {
				return fmt.Errorf("failed to insert category tag: %w", err)
			}
		}
	}

	if functionalityTagIDs, ok := updates["functionality_tag_ids"].([]uuid.UUID); ok {
		_, err := tx.Exec(ctx, "DELETE FROM food_item_functionality_tags WHERE food_item_id = $1", id)
		if err != nil {
			return fmt.Errorf("failed to delete functionality tags: %w", err)
		}

		for _, tagID := range functionalityTagIDs {
			_, err = tx.Exec(ctx,
				`INSERT INTO food_item_functionality_tags (food_item_id, tag_id) VALUES ($1, $2)`,
				id, tagID)
			if err != nil {
				return fmt.Errorf("failed to insert functionality tag: %w", err)
			}
		}
	}

	if len(setClauses) > 0 {
		setClauses = append(setClauses, "updated_at = NOW()")
		query := fmt.Sprintf("UPDATE food_items SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIndex)
		args = append(args, id)

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to update food item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *foodItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM food_items WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete food item: %w", err)
	}
	return nil
}

func (r *foodItemRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM food_items WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

func (r *foodItemRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.FoodItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, name, physical_state, prep_time, average_unit_weight,
		       macros, micros, image_url, created_at, updated_at
		FROM food_items WHERE id = ANY($1)
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get food items by IDs: %w", err)
	}
	defer rows.Close()

	var items []*models.FoodItem
	for rows.Next() {
		var item models.FoodItem
		var macrosJSON, microsJSON []byte

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.PhysicalState,
			&item.PrepTime,
			&item.AverageUnitWeight,
			&macrosJSON,
			&microsJSON,
			&item.ImageURL,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan food item: %w", err)
		}

		if err := json.Unmarshal(macrosJSON, &item.Macros); err != nil {
			return nil, fmt.Errorf("failed to unmarshal macros: %w", err)
		}
		if err := json.Unmarshal(microsJSON, &item.Micros); err != nil {
			return nil, fmt.Errorf("failed to unmarshal micros: %w", err)
		}

		categoryTags, err := r.getTagsByFoodItemID(ctx, item.ID, "category")
		if err != nil {
			return nil, err
		}
		item.CategoryTags = categoryTags

		functionalityTags, err := r.getTagsByFoodItemID(ctx, item.ID, "functionality")
		if err != nil {
			return nil, err
		}
		item.FunctionalityTags = functionalityTags

		items = append(items, &item)
	}

	return items, nil
}

func (r *foodItemRepository) Count(ctx context.Context, query models.FoodItemQuery) (int64, error) {
	baseQuery := `FROM food_items WHERE 1=1`
	var args []interface{}
	argIndex := 1

	if len(query.IDs) > 0 {
		baseQuery += fmt.Sprintf(" AND id = ANY($%d)", argIndex)
		args = append(args, query.IDs)
		argIndex++
	}

	if query.Name != nil {
		baseQuery += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
		args = append(args, "%"+*query.Name+"%")
		argIndex++
	}

	if query.PhysicalState != nil {
		baseQuery += fmt.Sprintf(" AND physical_state = $%d", argIndex)
		args = append(args, *query.PhysicalState)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count food items: %w", err)
	}

	return total, nil
}

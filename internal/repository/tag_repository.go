// Phase: phase-01 | Task: 5 | Architecture: ARCH-005 | Design: TagEntity

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mealswapp/internal/models"
)

type pgTagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) TagRepository {
	return &pgTagRepository{db: db}
}

func (r *pgTagRepository) Create(ctx context.Context, input models.TagCreateInput) (*models.Tag, error) {
	query := `
		INSERT INTO tags (id, name, slug, type, description, color_hex, icon_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, slug, type, description, color_hex, icon_url, created_at, updated_at
	`

	tag := &models.Tag{
		ID:          generateUUID(),
		Name:        input.Name,
		Slug:        generateSlug(input.Name),
		Type:        input.Type,
		Description: input.Description,
		ColorHex:    input.ColorHex,
		IconURL:     input.IconURL,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if tag.ColorHex == "" {
		tag.ColorHex = getDefaultColorForType(input.Type)
	}

	err := r.db.QueryRowContext(ctx, query,
		tag.ID, tag.Name, tag.Slug, tag.Type, tag.Description,
		tag.ColorHex, tag.IconURL, tag.CreatedAt, tag.UpdatedAt,
	).Scan(
		&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
		&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return tag, nil
}

func (r *pgTagRepository) Update(ctx context.Context, id string, input models.TagUpdateInput) (*models.Tag, error) {
	var updates []string
	var args []interface{}
	argCount := 1

	if input.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *input.Name)
		argCount++
	}
	if input.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, *input.Description)
		argCount++
	}
	if input.ColorHex != nil {
		updates = append(updates, fmt.Sprintf("color_hex = $%d", argCount))
		args = append(args, *input.ColorHex)
		argCount++
	}
	if input.IconURL != nil {
		updates = append(updates, fmt.Sprintf("icon_url = $%d", argCount))
		args = append(args, *input.IconURL)
		argCount++
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now().UTC())
	argCount++

	args = append(args, id)

	query := fmt.Sprintf("UPDATE tags SET %s WHERE id = $%d RETURNING id, name, slug, type, description, color_hex, icon_url, created_at, updated_at",
		strings.Join(updates, ", "), argCount)

	tag := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
		&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	return tag, nil
}

func (r *pgTagRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM tags WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tag not found")
	}

	return nil
}

func (r *pgTagRepository) GetByID(ctx context.Context, id string) (*models.Tag, error) {
	query := `
		SELECT id, name, slug, type, description, color_hex, icon_url, created_at, updated_at
		FROM tags WHERE id = $1
	`

	tag := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
		&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	return tag, nil
}

func (r *pgTagRepository) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	query := `
		SELECT id, name, slug, type, description, color_hex, icon_url, created_at, updated_at
		FROM tags WHERE slug = $1
	`

	tag := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
		&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tag by slug: %w", err)
	}

	return tag, nil
}

func (r *pgTagRepository) GetByIDs(ctx context.Context, ids []string) ([]models.Tag, error) {
	if len(ids) == 0 {
		return []models.Tag{}, nil
	}

	query := `
		SELECT id, name, slug, type, description, color_hex, icon_url, created_at, updated_at
		FROM tags WHERE id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags by ids: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		tag := models.Tag{}
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
			&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

func (r *pgTagRepository) GetByType(ctx context.Context, tagType models.TagType) ([]models.Tag, error) {
	query := `
		SELECT id, name, slug, type, description, color_hex, icon_url, created_at, updated_at
		FROM tags WHERE type = $1 ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tagType)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags by type: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		tag := models.Tag{}
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
			&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

func (r *pgTagRepository) GetCategoryTags(ctx context.Context) ([]models.Tag, error) {
	return r.GetByType(ctx, models.TagTypeCategory)
}

func (r *pgTagRepository) GetFunctionalityTags(ctx context.Context) ([]models.Tag, error) {
	return r.GetByType(ctx, models.TagTypeFunctionality)
}

func (r *pgTagRepository) List(ctx context.Context, filter models.TagFilter) (*models.TagListResult, error) {
	baseQuery := "FROM tags WHERE 1=1"
	var args []interface{}
	argCount := 1

	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			placeholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, t)
			argCount++
		}
		baseQuery += fmt.Sprintf(" AND type IN (%s)", strings.Join(placeholders, ", "))
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Search+"%")
		argCount++
	}

	if filter.Slug != "" {
		baseQuery += fmt.Sprintf(" AND slug = $%d", argCount)
		args = append(args, filter.Slug)
		argCount++
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count tags: %w", err)
	}

	orderBy := "name"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	orderDir := "ASC"
	if filter.OrderDir != "" {
		orderDir = filter.OrderDir
	}

	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	selectQuery := fmt.Sprintf("SELECT id, name, slug, type, description, color_hex, icon_url, created_at, updated_at %s ORDER BY %s %s LIMIT %d OFFSET %d",
		baseQuery, orderBy, orderDir, limit, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		tag := models.Tag{}
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
			&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	hasMore := (offset + len(tags)) < total

	return &models.TagListResult{
		Tags:    tags,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

func (r *pgTagRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tag exists: %w", err)
	}
	return exists, nil
}

func (r *pgTagRepository) ExistsBySlug(ctx context.Context, slug string, excludeID string) (bool, error) {
	var exists bool
	var err error

	if excludeID != "" {
		err = r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tags WHERE slug = $1 AND id != $2)", slug, excludeID).Scan(&exists)
	} else {
		err = r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tags WHERE slug = $1)", slug).Scan(&exists)
	}

	if err != nil {
		return false, fmt.Errorf("failed to check slug exists: %w", err)
	}
	return exists, nil
}

func (r *pgTagRepository) CountByType(ctx context.Context, tagType models.TagType) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags WHERE type = $1", tagType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tags by type: %w", err)
	}
	return count, nil
}

func (r *pgTagRepository) GetTagsForFoodItem(ctx context.Context, foodItemID string) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.type, t.description, t.color_hex, t.icon_url, t.created_at, t.updated_at
		FROM tags t
		JOIN food_item_tags fit ON t.id = fit.tag_id
		WHERE fit.food_item_id = $1
		ORDER BY t.name
	`

	rows, err := r.db.QueryContext(ctx, query, foodItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags for food item: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		tag := models.Tag{}
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
			&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

func (r *pgTagRepository) GetTagsForMeal(ctx context.Context, mealID string) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.type, t.description, t.color_hex, t.icon_url, t.created_at, t.updated_at
		FROM tags t
		JOIN meal_tags mt ON t.id = mt.tag_id
		WHERE mt.meal_id = $1
		ORDER BY t.name
	`

	rows, err := r.db.QueryContext(ctx, query, mealID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags for meal: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		tag := models.Tag{}
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description,
			&tag.ColorHex, &tag.IconURL, &tag.CreatedAt, &tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

func (r *pgTagRepository) AssignTagsToFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO food_item_tags (food_item_id, tag_id, assigned_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, tagID := range tagIDs {
		_, err := stmt.ExecContext(ctx, foodItemID, tagID, now)
		if err != nil {
			return fmt.Errorf("failed to assign tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *pgTagRepository) RemoveTagsFromFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(tagIDs))
	args := []interface{}{foodItemID}
	for i, tagID := range tagIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, tagID)
	}

	query := fmt.Sprintf("DELETE FROM food_item_tags WHERE food_item_id = $1 AND tag_id IN (%s)", strings.Join(placeholders, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to remove tags from food item: %w", err)
	}

	return nil
}

func (r *pgTagRepository) AssignTagsToMeal(ctx context.Context, mealID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO meal_tags (meal_id, tag_id, assigned_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, tagID := range tagIDs {
		_, err := stmt.ExecContext(ctx, mealID, tagID, now)
		if err != nil {
			return fmt.Errorf("failed to assign tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *pgTagRepository) RemoveTagsFromMeal(ctx context.Context, mealID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(tagIDs))
	args := []interface{}{mealID}
	for i, tagID := range tagIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, tagID)
	}

	query := fmt.Sprintf("DELETE FROM meal_tags WHERE meal_id = $1 AND tag_id IN (%s)", strings.Join(placeholders, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to remove tags from meal: %w", err)
	}

	return nil
}

func generateUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", newRandomString(8), newRandomString(4), newRandomString(4), newRandomString(4), newRandomString(12))
}

func newRandomString(length int) string {
	const charset = "0123456789abcdef"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	var result []rune
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	return strings.Trim(string(result), "-")
}

func getDefaultColorForType(tagType models.TagType) string {
	switch tagType {
	case models.TagTypeCategory:
		return "#3B82F6"
	case models.TagTypeFunctionality:
		return "#10B981"
	default:
		return "#6B7280"
	}
}

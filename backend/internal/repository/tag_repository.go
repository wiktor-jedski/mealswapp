package repository

import (
	"context"
	_ "embed"

	"github.com/google/uuid"
)

// Implements DESIGN-005 TagEntity active tag query.
//
//go:embed sql/tag_list.sql
var tagListSQL string

// Implements DESIGN-005 TagEntity child-tag upsert query.
//
//go:embed sql/tag_upsert_child.sql
var tagUpsertChildSQL string

// Implements DESIGN-005 TagEntity root-tag upsert query.
//
//go:embed sql/tag_upsert_root.sql
var tagUpsertRootSQL string

// Implements DESIGN-005 TagEntity usage query.
//
//go:embed sql/tag_is_in_use.sql
var tagIsInUseSQL string

// Implements DESIGN-005 TagEntity soft-delete query.
//
//go:embed sql/tag_soft_delete.sql
var tagSoftDeleteSQL string

// PostgresTagRepository persists category and functionality tags in PostgreSQL.
// Implements DESIGN-005 TagEntity.
type PostgresTagRepository struct {
	db sqlExecutor
}

// NewPostgresTagRepository creates a PostgreSQL-backed tag repository.
// Implements DESIGN-005 TagEntity.
func NewPostgresTagRepository(db sqlExecutor) *PostgresTagRepository {
	return &PostgresTagRepository{db: db}
}

// List returns active tags of the requested kind in deterministic hierarchy order.
// Implements DESIGN-005 TagEntity.
func (r *PostgresTagRepository) List(ctx context.Context, kind TagKind) ([]TagEntity, error) {
	rows, err := r.db.Query(ctx, tagListSQL, string(kind))
	if err != nil {
		return nil, mapPostgresError(err, "list tags")
	}
	defer rows.Close()

	tags := []TagEntity{}
	for rows.Next() {
		var tag TagEntity
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Kind, &tag.ParentID); err != nil {
			return nil, mapPostgresError(err, "scan tag")
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate tags")
	}
	return tags, nil
}

// Upsert creates or updates an active tag by kind, parent, and normalized name.
// Implements DESIGN-005 TagEntity.
func (r *PostgresTagRepository) Upsert(ctx context.Context, tag TagEntity) (uuid.UUID, error) {
	if tag.Kind != TagKindCategory && tag.Kind != TagKindFunctionality {
		return uuid.Nil, validationError("tag kind must be category or functionality")
	}
	if tag.Name == "" {
		return uuid.Nil, validationError("tag name is required")
	}

	var id uuid.UUID
	if tag.ParentID != nil {
		err := r.db.QueryRow(ctx, tagUpsertChildSQL, tag.Name, string(tag.Kind), tag.ParentID).Scan(&id)
		return id, mapPostgresError(err, "upsert child tag")
	}

	err := r.db.QueryRow(ctx, tagUpsertRootSQL, tag.Name, string(tag.Kind), tag.ParentID).Scan(&id)
	return id, mapPostgresError(err, "upsert tag")
}

// IsInUse reports whether a tag is attached to any food item or meal.
// Implements DESIGN-005 TagEntity.
func (r *PostgresTagRepository) IsInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	var inUse bool
	err := r.db.QueryRow(ctx, tagIsInUseSQL, id).Scan(&inUse)
	return inUse, mapPostgresError(err, "check tag usage")
}

// SoftDelete marks an unused tag deleted and rejects destructive changes for in-use tags.
// Implements DESIGN-005 TagEntity.
func (r *PostgresTagRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	inUse, err := r.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return NewError(ErrorKindConflict, "tag is in use", nil)
	}
	result, err := r.db.Exec(ctx, tagSoftDeleteSQL, id)
	if err != nil {
		return mapPostgresError(err, "delete tag")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "tag not found", nil)
	}
	return nil
}

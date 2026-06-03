package repository

import (
	"context"
	_ "embed"

	"github.com/google/uuid"
)

// Implements DESIGN-005 ClassificationEntity active classification query.
//
//go:embed sql/classification_list.sql
var classificationListSQL string

// Implements DESIGN-005 ClassificationEntity child-classification upsert query.
//
//go:embed sql/classification_upsert_child.sql
var classificationUpsertChildSQL string

// Implements DESIGN-005 ClassificationEntity root-classification upsert query.
//
//go:embed sql/classification_upsert_root.sql
var classificationUpsertRootSQL string

// Implements DESIGN-005 ClassificationEntity usage query.
//
//go:embed sql/classification_is_in_use.sql
var classificationIsInUseSQL string

// Implements DESIGN-005 ClassificationEntity soft-delete query.
//
//go:embed sql/classification_soft_delete.sql
var classificationSoftDeleteSQL string

// PostgresClassificationRepository persists Food Category and Culinary Role classifications in PostgreSQL.
// Implements DESIGN-005 ClassificationEntity.
type PostgresClassificationRepository struct {
	db sqlExecutor
}

// NewPostgresClassificationRepository creates a PostgreSQL-backed classification repository.
// Implements DESIGN-005 ClassificationEntity.
func NewPostgresClassificationRepository(db sqlExecutor) *PostgresClassificationRepository {
	return &PostgresClassificationRepository{db: db}
}

// List returns active classifications of the requested kind in deterministic hierarchy order.
// Implements DESIGN-005 ClassificationEntity.
func (r *PostgresClassificationRepository) List(ctx context.Context, kind ClassificationKind) ([]ClassificationEntity, error) {
	rows, err := r.db.Query(ctx, classificationListSQL, string(kind))
	if err != nil {
		return nil, mapPostgresError(err, "list classifications")
	}
	defer rows.Close()

	classifications := []ClassificationEntity{}
	for rows.Next() {
		var classification ClassificationEntity
		if err := rows.Scan(&classification.ID, &classification.Name, &classification.Kind, &classification.ParentID); err != nil {
			return nil, mapPostgresError(err, "scan classification")
		}
		classifications = append(classifications, classification)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate classifications")
	}
	return classifications, nil
}

// Upsert creates or updates an active classification by kind, parent, and normalized name.
// Implements DESIGN-005 ClassificationEntity.
func (r *PostgresClassificationRepository) Upsert(ctx context.Context, classification ClassificationEntity) (uuid.UUID, error) {
	if classification.Kind != ClassificationKindFoodCategory && classification.Kind != ClassificationKindCulinaryRole {
		return uuid.Nil, validationError("classification kind must be food_category or culinary_role")
	}
	if classification.Name == "" {
		return uuid.Nil, validationError("classification name is required")
	}

	var id uuid.UUID
	if classification.ParentID != nil {
		err := r.db.QueryRow(ctx, classificationUpsertChildSQL, classification.Name, string(classification.Kind), classification.ParentID).Scan(&id)
		return id, mapPostgresError(err, "upsert child classification")
	}

	err := r.db.QueryRow(ctx, classificationUpsertRootSQL, classification.Name, string(classification.Kind), classification.ParentID).Scan(&id)
	return id, mapPostgresError(err, "upsert classification")
}

// IsInUse reports whether a classification is attached to any food item or meal.
// Implements DESIGN-005 ClassificationEntity.
func (r *PostgresClassificationRepository) IsInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	var inUse bool
	err := r.db.QueryRow(ctx, classificationIsInUseSQL, id).Scan(&inUse)
	return inUse, mapPostgresError(err, "check classification usage")
}

// SoftDelete marks an unused classification deleted and rejects destructive changes for in-use classifications.
// Implements DESIGN-005 ClassificationEntity.
func (r *PostgresClassificationRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	inUse, err := r.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return NewError(ErrorKindConflict, "classification is in use", nil)
	}
	result, err := r.db.Exec(ctx, classificationSoftDeleteSQL, id)
	if err != nil {
		return mapPostgresError(err, "delete classification")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "classification not found", nil)
	}
	return nil
}

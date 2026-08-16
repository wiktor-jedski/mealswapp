package repository

import (
	"context"
	_ "embed"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Implements DESIGN-005 ClassificationEntity active classification query.
//
//go:embed sql/classification_list.sql
var classificationListSQL string

// Implements DESIGN-009 TagManager classification lookup query.
//
//go:embed sql/classification_get_by_id.sql
var classificationGetByIDSQL string

// Implements DESIGN-009 TagManager classification create query.
//
//go:embed sql/classification_create.sql
var classificationCreateSQL string

// Implements DESIGN-009 TagManager classification update query.
//
//go:embed sql/classification_update.sql
var classificationUpdateSQL string

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

// Implements DESIGN-005 ClassificationEntity compile-time repository contract.
var _ ClassificationRepository = (*PostgresClassificationRepository)(nil)

// Implements DESIGN-009 TagManager compile-time admin repository contract.
var _ ClassificationAdminRepository = (*PostgresClassificationRepository)(nil)

// NewPostgresClassificationRepository creates a PostgreSQL-backed classification repository.
// Implements DESIGN-005 ClassificationEntity.
func NewPostgresClassificationRepository(db sqlExecutor) *PostgresClassificationRepository {
	return &PostgresClassificationRepository{db: db}
}

// List returns active classifications of the requested kind in deterministic hierarchy order.
// Implements DESIGN-005 ClassificationEntity.
func (r *PostgresClassificationRepository) List(ctx context.Context, kind ClassificationKind) ([]ClassificationEntity, error) {
	if !validClassificationKind(kind) {
		return nil, validationError("classification kind must be food_category or culinary_role")
	}
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

// GetByID returns one active global classification.
// Implements DESIGN-009 TagManager.
func (r *PostgresClassificationRepository) GetByID(ctx context.Context, id uuid.UUID) (ClassificationEntity, error) {
	if id == uuid.Nil {
		return ClassificationEntity{}, validationError("classification id is required")
	}
	var classification ClassificationEntity
	err := r.db.QueryRow(ctx, classificationGetByIDSQL, id).Scan(&classification.ID, &classification.Name, &classification.Kind, &classification.ParentID)
	return classification, mapPostgresError(err, "get classification")
}

// Create inserts one new global classification and rejects normalized duplicates.
// Implements DESIGN-009 TagManager.
func (r *PostgresClassificationRepository) Create(ctx context.Context, classification ClassificationEntity) (ClassificationEntity, error) {
	if err := validateClassificationMutation(classification, false); err != nil {
		return ClassificationEntity{}, err
	}
	err := r.db.QueryRow(ctx, classificationCreateSQL, classification.Name, string(classification.Kind), classification.ParentID).Scan(&classification.ID, &classification.Name, &classification.Kind, &classification.ParentID)
	return classification, mapClassificationError(err, "create classification")
}

// Update renames or reparents one active global classification without changing its kind.
// Implements DESIGN-009 TagManager.
func (r *PostgresClassificationRepository) Update(ctx context.Context, classification ClassificationEntity) (ClassificationEntity, error) {
	if err := validateClassificationMutation(classification, true); err != nil {
		return ClassificationEntity{}, err
	}
	err := r.db.QueryRow(ctx, classificationUpdateSQL, classification.ID, classification.Name, classification.ParentID).Scan(&classification.ID, &classification.Name, &classification.Kind, &classification.ParentID)
	return classification, mapClassificationError(err, "update classification")
}

// Upsert creates or updates an active classification by kind, parent, and normalized name.
// Implements DESIGN-005 ClassificationEntity.
func (r *PostgresClassificationRepository) Upsert(ctx context.Context, classification ClassificationEntity) (uuid.UUID, error) {
	if err := validateClassificationMutation(classification, false); err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	if classification.ParentID != nil {
		err := r.db.QueryRow(ctx, classificationUpsertChildSQL, classification.Name, string(classification.Kind), classification.ParentID).Scan(&id)
		return id, mapPostgresError(err, "upsert child classification")
	}

	err := r.db.QueryRow(ctx, classificationUpsertRootSQL, classification.Name, string(classification.Kind), classification.ParentID).Scan(&id)
	return id, mapPostgresError(err, "upsert classification")
}

// validateClassificationMutation enforces the repository trust boundary before SQL dispatch.
// Implements DESIGN-009 TagManager.
func validateClassificationMutation(classification ClassificationEntity, requireID bool) error {
	if !validClassificationKind(classification.Kind) {
		return validationError("classification kind must be food_category or culinary_role")
	}
	if classification.Name == "" {
		return validationError("classification name is required")
	}
	if requireID && classification.ID == uuid.Nil {
		return validationError("classification id is required")
	}
	if classification.ParentID != nil && *classification.ParentID == classification.ID {
		return NewError(ErrorKindConflict, "classification hierarchy cycle", nil)
	}
	return nil
}

// validClassificationKind limits global administration to the two DESIGN-009 kinds.
// Implements DESIGN-009 TagManager.
func validClassificationKind(kind ClassificationKind) bool {
	return kind == ClassificationKindFoodCategory || kind == ClassificationKindCulinaryRole
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
		return mapClassificationError(err, "delete classification")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "classification not found", nil)
	}
	return nil
}

// mapClassificationError preserves conflict semantics for database-enforced cycle and use guards.
// Implements DESIGN-009 TagManager.
func mapClassificationError(err error, fallback string) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && (postgresError.ConstraintName == "classification_hierarchy_cycle" || postgresError.ConstraintName == "classification_in_use") {
		return NewError(ErrorKindConflict, fallback, err)
	}
	return mapPostgresError(err, fallback)
}

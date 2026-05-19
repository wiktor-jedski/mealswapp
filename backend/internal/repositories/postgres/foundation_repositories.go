package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct{ db DBTX }
type PreferenceRepository struct{ db DBTX }
type EntitlementRepository struct{ db DBTX }
type SavedDataRepository struct{ db DBTX }
type AuditLogRepository struct{ db DBTX }
type ImportRepository struct{ db DBTX }

func NewUserRepositoryWithDB(db DBTX) UserRepository             { return UserRepository{db: db} }
func NewPreferenceRepositoryWithDB(db DBTX) PreferenceRepository { return PreferenceRepository{db: db} }
func NewEntitlementRepositoryWithDB(db DBTX) EntitlementRepository {
	return EntitlementRepository{db: db}
}
func NewSavedDataRepositoryWithDB(db DBTX) SavedDataRepository { return SavedDataRepository{db: db} }
func NewAuditLogRepositoryWithDB(db DBTX) AuditLogRepository   { return AuditLogRepository{db: db} }
func NewImportRepositoryWithDB(db DBTX) ImportRepository       { return ImportRepository{db: db} }

func (repo UserRepository) Create(ctx context.Context, user repositories.UserEntity) (uuid.UUID, error) {
	var id uuid.UUID
	err := repo.db.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash, role, disabled)
		VALUES ($1, $2, $3, coalesce(nullif($4, ''), 'user'), $5)
		RETURNING id
	`, user.Email, user.DisplayName, user.PasswordHash, user.Role, user.Disabled).Scan(&id)
	return id, err
}

func (repo UserRepository) GetByID(ctx context.Context, id uuid.UUID) (repositories.UserEntity, error) {
	var user repositories.UserEntity
	err := repo.db.QueryRow(ctx, `
		SELECT id, email, display_name, password_hash, role, disabled, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Role, &user.Disabled, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (repo UserRepository) Update(ctx context.Context, user repositories.UserEntity) error {
	if user.ID == uuid.Nil {
		return errors.New("user id is required")
	}
	tag, err := repo.db.Exec(ctx, `
		UPDATE users SET email = $2, display_name = $3, password_hash = $4, role = $5, disabled = $6, updated_at = now()
		WHERE id = $1
	`, user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Role, user.Disabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo PreferenceRepository) Upsert(ctx context.Context, pref repositories.PreferenceEntity) error {
	enabled, err := json.Marshal(pref.EnabledMacros)
	if err != nil {
		return err
	}
	if pref.ExcludedTagIDs == nil {
		pref.ExcludedTagIDs = []uuid.UUID{}
	}
	if pref.DietaryFilterIDs == nil {
		pref.DietaryFilterIDs = []uuid.UUID{}
	}
	_, err = repo.db.Exec(ctx, `
		INSERT INTO user_preferences (user_id, theme, default_search_mode, enabled_macros, excluded_tag_ids, dietary_filter_ids)
		VALUES ($1, coalesce(nullif($2, ''), 'system'), coalesce(nullif($3, ''), 'single'), $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET theme = excluded.theme, default_search_mode = excluded.default_search_mode, enabled_macros = excluded.enabled_macros, excluded_tag_ids = excluded.excluded_tag_ids, dietary_filter_ids = excluded.dietary_filter_ids, updated_at = now()
	`, pref.UserID, pref.Theme, pref.DefaultSearchMode, enabled, pref.ExcludedTagIDs, pref.DietaryFilterIDs)
	return err
}

func (repo PreferenceRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.PreferenceEntity, error) {
	var pref repositories.PreferenceEntity
	var enabled []byte
	err := repo.db.QueryRow(ctx, `
		SELECT user_id, theme, default_search_mode, enabled_macros, excluded_tag_ids, dietary_filter_ids, updated_at
		FROM user_preferences WHERE user_id = $1
	`, userID).Scan(&pref.UserID, &pref.Theme, &pref.DefaultSearchMode, &enabled, &pref.ExcludedTagIDs, &pref.DietaryFilterIDs, &pref.UpdatedAt)
	if err != nil {
		return repositories.PreferenceEntity{}, err
	}
	if err := json.Unmarshal(enabled, &pref.EnabledMacros); err != nil {
		return repositories.PreferenceEntity{}, err
	}
	return pref, nil
}

func (repo PreferenceRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM user_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo EntitlementRepository) Upsert(ctx context.Context, ent repositories.EntitlementEntity) error {
	_, err := repo.db.Exec(ctx, `
		INSERT INTO entitlements (user_id, plan, status, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET plan = excluded.plan, status = excluded.status, expires_at = excluded.expires_at, updated_at = now()
	`, ent.UserID, ent.Plan, ent.Status, ent.ExpiresAt)
	return err
}

func (repo EntitlementRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error) {
	var ent repositories.EntitlementEntity
	err := repo.db.QueryRow(ctx, `SELECT user_id, plan, status, expires_at, updated_at FROM entitlements WHERE user_id = $1`, userID).
		Scan(&ent.UserID, &ent.Plan, &ent.Status, &ent.ExpiresAt, &ent.UpdatedAt)
	return ent, err
}

func (repo EntitlementRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM entitlements WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo SavedDataRepository) Create(ctx context.Context, saved repositories.SavedDataEntity) (uuid.UUID, error) {
	var id uuid.UUID
	err := repo.db.QueryRow(ctx, `
		INSERT INTO saved_data (user_id, kind, label, payload)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, saved.UserID, saved.Kind, saved.Label, saved.Payload).Scan(&id)
	return id, err
}

func (repo SavedDataRepository) GetByID(ctx context.Context, id uuid.UUID) (repositories.SavedDataEntity, error) {
	var saved repositories.SavedDataEntity
	err := repo.db.QueryRow(ctx, `SELECT id, user_id, kind, label, payload, created_at FROM saved_data WHERE id = $1`, id).
		Scan(&saved.ID, &saved.UserID, &saved.Kind, &saved.Label, &saved.Payload, &saved.CreatedAt)
	return saved, err
}

func (repo SavedDataRepository) Update(ctx context.Context, saved repositories.SavedDataEntity) error {
	tag, err := repo.db.Exec(ctx, `UPDATE saved_data SET kind = $2, label = $3, payload = $4, updated_at = now() WHERE id = $1`, saved.ID, saved.Kind, saved.Label, saved.Payload)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo SavedDataRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM saved_data WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo AuditLogRepository) Create(ctx context.Context, entry repositories.AuditLogEntity) (uuid.UUID, error) {
	var id uuid.UUID
	err := repo.db.QueryRow(ctx, `INSERT INTO audit_logs (actor_id, action, target, metadata) VALUES ($1, $2, $3, $4) RETURNING id`, entry.ActorID, entry.Action, entry.Target, entry.Metadata).Scan(&id)
	return id, err
}

func (repo AuditLogRepository) GetByID(ctx context.Context, id uuid.UUID) (repositories.AuditLogEntity, error) {
	var entry repositories.AuditLogEntity
	err := repo.db.QueryRow(ctx, `SELECT id, actor_id, action, target, metadata, created_at FROM audit_logs WHERE id = $1`, id).
		Scan(&entry.ID, &entry.ActorID, &entry.Action, &entry.Target, &entry.Metadata, &entry.CreatedAt)
	return entry, err
}

func (repo AuditLogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM audit_logs WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo ImportRepository) Create(ctx context.Context, record repositories.ImportEntity) (uuid.UUID, error) {
	var id uuid.UUID
	err := repo.db.QueryRow(ctx, `INSERT INTO import_records (provider, external_id, status, payload) VALUES ($1, $2, $3, $4) RETURNING id`, record.Provider, record.ExternalID, record.Status, record.Payload).Scan(&id)
	return id, err
}

func (repo ImportRepository) GetByID(ctx context.Context, id uuid.UUID) (repositories.ImportEntity, error) {
	var record repositories.ImportEntity
	err := repo.db.QueryRow(ctx, `SELECT id, provider, external_id, status, payload, created_at, updated_at FROM import_records WHERE id = $1`, id).
		Scan(&record.ID, &record.Provider, &record.ExternalID, &record.Status, &record.Payload, &record.CreatedAt, &record.UpdatedAt)
	return record, err
}

func (repo ImportRepository) Update(ctx context.Context, record repositories.ImportEntity) error {
	tag, err := repo.db.Exec(ctx, `UPDATE import_records SET status = $2, payload = $3, updated_at = now() WHERE id = $1`, record.ID, record.Status, record.Payload)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (repo ImportRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := repo.db.Exec(ctx, `DELETE FROM import_records WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

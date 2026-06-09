package repository

import (
	"context"
	_ "embed"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-008 PreferenceManager get-or-create query.
//
//go:embed sql/profile_get_or_create.sql
var profileGetOrCreateSQL string

// Implements DESIGN-008 PreferenceManager unit-system query.
//
//go:embed sql/profile_get_unit_system.sql
var profileGetUnitSystemSQL string

// Implements DESIGN-008 PreferenceManager update query.
//
//go:embed sql/profile_update.sql
var profileUpdateSQL string

// Implements DESIGN-008 SavedDataRepository save query.
//
//go:embed sql/saved_item_save.sql
var savedItemSaveSQL string

// Implements DESIGN-008 SavedDataRepository remove query.
//
//go:embed sql/saved_item_remove.sql
var savedItemRemoveSQL string

// Implements DESIGN-008 SavedDataRepository list query.
//
//go:embed sql/saved_item_list.sql
var savedItemListSQL string

// Implements DESIGN-008 SearchHistoryRepository add query.
//
//go:embed sql/search_history_add.sql
var searchHistoryAddSQL string

// Implements DESIGN-008 SearchHistoryRepository list query.
//
//go:embed sql/search_history_list.sql
var searchHistoryListSQL string

// Implements DESIGN-008 SearchHistoryRepository clear query.
//
//go:embed sql/search_history_clear.sql
var searchHistoryClearSQL string

// PostgresUserProfileRepository persists user profiles and preferences in PostgreSQL.
// Implements DESIGN-008 PreferenceManager.
type PostgresUserProfileRepository struct {
	db sqlExecutor
}

// Implements DESIGN-008 PreferenceManager compile-time repository contract.
var _ UserProfileRepository = (*PostgresUserProfileRepository)(nil)

// NewPostgresUserProfileRepository creates a PostgreSQL-backed profile repository.
// Implements DESIGN-008 PreferenceManager.
func NewPostgresUserProfileRepository(db sqlExecutor) *PostgresUserProfileRepository {
	return &PostgresUserProfileRepository{db: db}
}

// GetOrCreate returns the user's profile, creating default preferences on first access.
// Implements DESIGN-008 PreferenceManager.
func (r *PostgresUserProfileRepository) GetOrCreate(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	if userID == uuid.Nil {
		return UserProfile{}, validationError("user id is required")
	}
	row := r.db.QueryRow(ctx, profileGetOrCreateSQL, userID)
	return scanUserProfile(row)
}

// UpdateProfile persists display, unit, and theme preferences with a recalculation hint.
// Implements DESIGN-008 PreferenceManager.
func (r *PostgresUserProfileRepository) UpdateProfile(ctx context.Context, profile UserProfile) (PreferenceUpdateResult, error) {
	if profile.UserID == uuid.Nil {
		return PreferenceUpdateResult{}, validationError("user id is required")
	}
	if profile.UnitSystem != UnitSystemMetric && profile.UnitSystem != UnitSystemImperial {
		return PreferenceUpdateResult{}, validationError("unit system is invalid")
	}
	if profile.ThemePreference != "system" && profile.ThemePreference != "light" && profile.ThemePreference != "dark" {
		return PreferenceUpdateResult{}, validationError("theme preference is invalid")
	}

	var previous UnitSystem
	err := r.db.QueryRow(ctx, profileGetUnitSystemSQL, profile.UserID).Scan(&previous)
	if err != nil {
		if err == pgx.ErrNoRows {
			return PreferenceUpdateResult{}, NewError(ErrorKindNotFound, "profile not found", err)
		}
		return PreferenceUpdateResult{}, mapPostgresError(err, "load profile preferences")
	}

	row := r.db.QueryRow(ctx, profileUpdateSQL, profile.UserID, profile.DisplayName, string(profile.UnitSystem), profile.ThemePreference)
	updated, err := scanUserProfile(row)
	if err != nil {
		return PreferenceUpdateResult{}, err
	}
	return PreferenceUpdateResult{Profile: updated, RequiresUnitRecalculation: previous != updated.UnitSystem}, nil
}

// PostgresSavedDataRepository persists saved items and search history in PostgreSQL.
// Implements DESIGN-008 SavedDataRepository.
type PostgresSavedDataRepository struct {
	db transactionalExecutor
}

// Implements DESIGN-008 SavedDataRepository compile-time repository contract.
var _ SavedItemRepository = (*PostgresSavedDataRepository)(nil)

// Implements DESIGN-008 SearchHistoryRepository compile-time repository contract.
var _ SearchHistoryRepository = (*PostgresSavedDataRepository)(nil)

// NewPostgresSavedDataRepository creates a PostgreSQL-backed saved data repository.
// Implements DESIGN-008 SavedDataRepository.
func NewPostgresSavedDataRepository(db transactionalExecutor) *PostgresSavedDataRepository {
	return &PostgresSavedDataRepository{db: db}
}

// SaveItem stores a server-scoped saved item idempotently.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) SaveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) (uuid.UUID, error) {
	if err := validateSavedItemInput(userID, itemID, kind); err != nil {
		return uuid.Nil, err
	}
	if err := r.validateSavedItemTarget(ctx, itemID, kind); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, savedItemSaveSQL, userID, itemID, string(kind)).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "save item")
	}
	return id, nil
}

// RemoveItem removes a saved item for one user without affecting other users.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) error {
	if err := validateSavedItemInput(userID, itemID, kind); err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, savedItemRemoveSQL, userID, itemID, string(kind))
	if err != nil {
		return mapPostgresError(err, "remove saved item")
	}
	if tag.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "saved item not found", nil)
	}
	return nil
}

// ListItems returns saved items scoped to one user, optionally filtered by kind.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) ListItems(ctx context.Context, userID uuid.UUID, kind *SavedItemKind) ([]SavedItem, error) {
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	if kind != nil && !validSavedItemKind(*kind) {
		return nil, validationError("saved item kind is invalid")
	}
	rows, err := r.db.Query(ctx, savedItemListSQL, userID, savedItemKindFilter(kind))
	if err != nil {
		return nil, mapPostgresError(err, "list saved items")
	}
	defer rows.Close()

	items := []SavedItem{}
	for rows.Next() {
		item, err := scanSavedItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate saved items")
	}
	return items, nil
}

// AddHistory stores one search history entry for the server-supplied user.
// Implements DESIGN-008 SearchHistoryRepository.
func (r *PostgresSavedDataRepository) AddHistory(ctx context.Context, entry SearchHistoryEntry) (uuid.UUID, error) {
	if entry.UserID == uuid.Nil {
		return uuid.Nil, validationError("user id is required")
	}
	if strings.TrimSpace(entry.Query) == "" {
		return uuid.Nil, validationError("history query is required")
	}
	if strings.TrimSpace(entry.Mode) == "" {
		return uuid.Nil, validationError("history mode is required")
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, searchHistoryAddSQL, entry.UserID, entry.Query, entry.Mode, entry.FiltersHash).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "add search history")
	}
	return id, nil
}

// ListHistory returns recent search history scoped to one user.
// Implements DESIGN-008 SearchHistoryRepository.
func (r *PostgresSavedDataRepository) ListHistory(ctx context.Context, userID uuid.UUID, limit int) ([]SearchHistoryEntry, error) {
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, searchHistoryListSQL, userID, limit)
	if err != nil {
		return nil, mapPostgresError(err, "list search history")
	}
	defer rows.Close()

	entries := []SearchHistoryEntry{}
	for rows.Next() {
		entry, err := scanSearchHistoryEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate search history")
	}
	return entries, nil
}

// ClearHistory removes all search history entries for one user.
// Implements DESIGN-008 SearchHistoryRepository.
func (r *PostgresSavedDataRepository) ClearHistory(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if _, err := r.db.Exec(ctx, searchHistoryClearSQL, userID); err != nil {
		return mapPostgresError(err, "clear search history")
	}
	return nil
}

// validateSavedItemTarget verifies that a saved-item target exists for its kind.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) validateSavedItemTarget(ctx context.Context, itemID uuid.UUID, kind SavedItemKind) error {
	switch kind {
	case SavedItemKindFavorite:
		_, err := NewPostgresFoodItemRepository(r.db).GetByID(ctx, itemID, RepositoryContext{})
		return err
	case SavedItemKindSavedMeal:
		_, err := NewPostgresMealRepository(r.db).GetByID(ctx, itemID, RepositoryContext{})
		return err
	case SavedItemKindSavedDiet:
		return validationError("saved diet persistence is deferred until Phase 07")
	default:
		return validationError("saved item kind is invalid")
	}
}

// scanUserProfile reads a user profile from a PostgreSQL row.
// Implements DESIGN-008 PreferenceManager.
func scanUserProfile(row pgx.Row) (UserProfile, error) {
	var profile UserProfile
	if err := row.Scan(&profile.UserID, &profile.DisplayName, &profile.UnitSystem, &profile.ThemePreference, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
		return UserProfile{}, mapPostgresError(err, "scan user profile")
	}
	return profile, nil
}

// scanSavedItem reads a saved item from a PostgreSQL row.
// Implements DESIGN-008 SavedDataRepository.
func scanSavedItem(rows pgx.Rows) (SavedItem, error) {
	var item SavedItem
	if err := rows.Scan(&item.ID, &item.UserID, &item.ItemID, &item.Kind, &item.CreatedAt); err != nil {
		return SavedItem{}, mapPostgresError(err, "scan saved item")
	}
	return item, nil
}

// scanSearchHistoryEntry reads a search-history entry from a PostgreSQL row.
// Implements DESIGN-008 SearchHistoryRepository.
func scanSearchHistoryEntry(rows pgx.Rows) (SearchHistoryEntry, error) {
	var entry SearchHistoryEntry
	if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Query, &entry.Mode, &entry.FiltersHash, &entry.CreatedAt); err != nil {
		return SearchHistoryEntry{}, mapPostgresError(err, "scan search history")
	}
	return entry, nil
}

// validateSavedItemInput checks saved-item identity and kind fields.
// Implements DESIGN-008 SavedDataRepository.
func validateSavedItemInput(userID uuid.UUID, itemID uuid.UUID, kind SavedItemKind) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if itemID == uuid.Nil {
		return validationError("item id is required")
	}
	if !validSavedItemKind(kind) {
		return validationError("saved item kind is invalid")
	}
	return nil
}

// validSavedItemKind reports whether a saved-item kind is supported.
// Implements DESIGN-008 SavedDataRepository.
func validSavedItemKind(kind SavedItemKind) bool {
	return kind == SavedItemKindFavorite || kind == SavedItemKindSavedMeal || kind == SavedItemKindSavedDiet
}

// savedItemKindFilter maps a list filter to its persisted saved-item kind.
// Implements DESIGN-008 SavedDataRepository.
func savedItemKindFilter(kind *SavedItemKind) string {
	if kind == nil {
		return ""
	}
	return string(*kind)
}

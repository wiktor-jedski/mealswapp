package repository

import (
	"context"
	_ "embed"
	"math"
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

// Implements DESIGN-008 SavedDataRepository saved-diet parent insert query.
//
//go:embed sql/saved_diet_create.sql
var savedDietCreateSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet parent read query.
//
//go:embed sql/saved_diet_get.sql
var savedDietGetSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet list query.
//
//go:embed sql/saved_diet_list.sql
var savedDietListSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet parent replacement query.
//
//go:embed sql/saved_diet_update.sql
var savedDietUpdateSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet delete query.
//
//go:embed sql/saved_diet_delete.sql
var savedDietDeleteSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet entry list query.
//
//go:embed sql/saved_diet_entry_list.sql
var savedDietEntryListSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet entry insert query.
//
//go:embed sql/saved_diet_entry_insert.sql
var savedDietEntryInsertSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet entry replacement query.
//
//go:embed sql/saved_diet_entry_clear.sql
var savedDietEntryClearSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet saved-item query.
//
//go:embed sql/saved_diet_saved_item.sql
var savedDietSavedItemSQL string

// Implements DESIGN-008 SavedDataRepository saved-diet target validation query.
//
//go:embed sql/saved_diet_target_exists.sql
var savedDietTargetExistsSQL string

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

// Implements DESIGN-008 SavedDataRepository compile-time saved-diet contract.
var _ DailyDietRepository = (*PostgresSavedDataRepository)(nil)

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
	if kind == SavedItemKindSavedDiet {
		if err := r.validateSavedDietSavedItemTarget(ctx, userID, itemID); err != nil {
			return uuid.Nil, err
		}
	} else if err := r.validateSavedItemTarget(ctx, itemID, kind); err != nil {
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

// Create stores a user-owned saved daily diet and its ordered meal entries in one transaction.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) Create(ctx context.Context, userID uuid.UUID, diet SavedDiet) (uuid.UUID, error) {
	if err := validateSavedDietInput(userID, diet, false); err != nil {
		return uuid.Nil, err
	}
	entries, err := normalizeSavedDietEntries(diet.Entries)
	if err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	err = withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		name := normalizeSavedDietName(diet.Name)
		if err := db.QueryRow(ctx, savedDietCreateSQL, userID, name).Scan(&id); err != nil {
			return mapPostgresError(err, "create saved diet")
		}
		if err := replaceSavedDietEntries(ctx, db, id, entries); err != nil {
			return err
		}
		var savedItemID uuid.UUID
		if err := db.QueryRow(ctx, savedDietSavedItemSQL, userID, id).Scan(&savedItemID); err != nil {
			return mapPostgresError(err, "save saved diet item")
		}
		return nil
	})
	return id, err
}

// Get returns one saved daily diet only when it belongs to userID.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) Get(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) (SavedDiet, error) {
	if userID == uuid.Nil {
		return SavedDiet{}, validationError("user id is required")
	}
	if dietID == uuid.Nil {
		return SavedDiet{}, validationError("saved diet id is required")
	}
	return r.getSavedDiet(ctx, r.db, userID, dietID)
}

// List returns all saved daily diets owned by userID in deterministic recency order.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) List(ctx context.Context, userID uuid.UUID) ([]SavedDiet, error) {
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	rows, err := r.db.Query(ctx, savedDietListSQL, userID)
	if err != nil {
		return nil, mapPostgresError(err, "list saved diets")
	}
	defer rows.Close()

	diets := []SavedDiet{}
	for rows.Next() {
		var diet SavedDiet
		if err := scanSavedDiet(rows, &diet); err != nil {
			return nil, err
		}
		diet, err = r.loadSavedDietEntries(ctx, r.db, diet)
		if err != nil {
			return nil, err
		}
		diets = append(diets, diet)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate saved diets")
	}
	return diets, nil
}

// Replace atomically replaces the parent name and all ordered meal entries for one owned diet.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) Replace(ctx context.Context, userID uuid.UUID, diet SavedDiet) error {
	if err := validateSavedDietInput(userID, diet, true); err != nil {
		return err
	}
	entries, err := normalizeSavedDietEntries(diet.Entries)
	if err != nil {
		return err
	}
	return withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		tag, err := db.Exec(ctx, savedDietUpdateSQL, diet.ID, userID, normalizeSavedDietName(diet.Name))
		if err != nil {
			return mapPostgresError(err, "replace saved diet")
		}
		if tag.RowsAffected() == 0 {
			return NewError(ErrorKindNotFound, "saved diet not found", nil)
		}
		if err := replaceSavedDietEntries(ctx, db, diet.ID, entries); err != nil {
			return err
		}
		if err := ensureSavedDietItem(ctx, db, userID, diet.ID); err != nil {
			return err
		}
		return nil
	})
}

// Delete removes one owned saved diet; database triggers remove its saved-item reference and entries.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) Delete(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if dietID == uuid.Nil {
		return validationError("saved diet id is required")
	}
	tag, err := r.db.Exec(ctx, savedDietDeleteSQL, dietID, userID)
	if err != nil {
		return mapPostgresError(err, "delete saved diet")
	}
	if tag.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "saved diet not found", nil)
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
		return validationError("saved diet owner is required")
	default:
		return validationError("saved item kind is invalid")
	}
}

// validateSavedDietSavedItemTarget verifies that a saved diet belongs to the saved-item owner.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) validateSavedDietSavedItemTarget(ctx context.Context, userID uuid.UUID, dietID uuid.UUID) error {
	var exists bool
	if err := r.db.QueryRow(ctx, savedDietTargetExistsSQL, dietID, userID).Scan(&exists); err != nil {
		return mapPostgresError(err, "validate saved diet target")
	}
	if !exists {
		return NewError(ErrorKindValidation, "saved diet target does not exist for user", nil)
	}
	return nil
}

// getSavedDiet loads a parent and its deterministic entry order using one ownership predicate.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) getSavedDiet(ctx context.Context, db sqlExecutor, userID uuid.UUID, dietID uuid.UUID) (SavedDiet, error) {
	var diet SavedDiet
	if err := scanSavedDiet(db.QueryRow(ctx, savedDietGetSQL, dietID, userID), &diet); err != nil {
		return SavedDiet{}, err
	}
	return r.loadSavedDietEntries(ctx, db, diet)
}

// loadSavedDietEntries hydrates ordered meal-entry rows for a saved diet.
// Implements DESIGN-008 SavedDataRepository.
func (r *PostgresSavedDataRepository) loadSavedDietEntries(ctx context.Context, db sqlExecutor, diet SavedDiet) (SavedDiet, error) {
	rows, err := db.Query(ctx, savedDietEntryListSQL, diet.ID)
	if err != nil {
		return SavedDiet{}, mapPostgresError(err, "list saved diet entries")
	}
	defer rows.Close()
	diet.Entries = []SavedDietMealEntry{}
	for rows.Next() {
		var entry SavedDietMealEntry
		if err := scanSavedDietEntry(rows, &entry); err != nil {
			return SavedDiet{}, err
		}
		diet.Entries = append(diet.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return SavedDiet{}, mapPostgresError(err, "iterate saved diet entries")
	}
	return diet, nil
}

// replaceSavedDietEntries replaces all entries inside the caller's transaction.
// Implements DESIGN-008 SavedDataRepository.
func replaceSavedDietEntries(ctx context.Context, db transactionalExecutor, dietID uuid.UUID, entries []SavedDietMealEntry) error {
	if _, err := db.Exec(ctx, savedDietEntryClearSQL, dietID); err != nil {
		return mapPostgresError(err, "clear saved diet entries")
	}
	for _, entry := range entries {
		if _, err := db.Exec(ctx, savedDietEntryInsertSQL, dietID, entry.MealID, entry.Quantity, entry.Unit, entry.Position); err != nil {
			return mapPostgresError(err, "replace saved diet entries")
		}
	}
	return nil
}

// ensureSavedDietItem keeps the polymorphic saved-item index complete for a persisted diet.
// Implements DESIGN-008 SavedDataRepository.
func ensureSavedDietItem(ctx context.Context, db transactionalExecutor, userID uuid.UUID, dietID uuid.UUID) error {
	var savedItemID uuid.UUID
	if err := db.QueryRow(ctx, savedDietSavedItemSQL, userID, dietID).Scan(&savedItemID); err != nil {
		return mapPostgresError(err, "save saved diet item")
	}
	return nil
}

// scanSavedDiet scans one saved-diet parent row.
// Implements DESIGN-008 SavedDataRepository.
func scanSavedDiet(row pgx.Row, diet *SavedDiet) error {
	if err := row.Scan(&diet.ID, &diet.UserID, &diet.Name, &diet.CreatedAt, &diet.UpdatedAt); err != nil {
		return mapPostgresError(err, "scan saved diet")
	}
	return nil
}

// scanSavedDietEntry scans one ordered saved-diet meal-entry row.
// Implements DESIGN-008 SavedDataRepository.
func scanSavedDietEntry(rows pgx.Rows, entry *SavedDietMealEntry) error {
	if err := rows.Scan(&entry.ID, &entry.SavedDietID, &entry.MealID, &entry.Quantity, &entry.Unit, &entry.Position, &entry.CreatedAt); err != nil {
		return mapPostgresError(err, "scan saved diet entry")
	}
	return nil
}

// validateSavedDietInput validates identity and entry fields before a transaction starts.
// Implements DESIGN-008 SavedDataRepository.
func validateSavedDietInput(userID uuid.UUID, diet SavedDiet, replacing bool) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if replacing && diet.ID == uuid.Nil {
		return validationError("saved diet id is required")
	}
	if strings.ContainsRune(diet.Name, '\x00') {
		return validationError("saved diet name contains invalid characters")
	}
	for _, entry := range diet.Entries {
		if entry.MealID == uuid.Nil {
			return validationError("saved diet meal id is required")
		}
		if entry.Quantity <= 0 || math.IsNaN(entry.Quantity) || math.IsInf(entry.Quantity, 0) {
			return validationError("saved diet meal quantity must be finite and positive")
		}
		if ValidateQuantityUnit(entry.Unit) != nil {
			return validationError("saved diet meal unit is invalid")
		}
	}
	return nil
}

// normalizeSavedDietEntries assigns positions when callers provide an array without explicit order.
// Implements DESIGN-008 SavedDataRepository.
func normalizeSavedDietEntries(entries []SavedDietMealEntry) ([]SavedDietMealEntry, error) {
	result := append([]SavedDietMealEntry(nil), entries...)
	if len(result) > 1 {
		allDefault := true
		for _, entry := range result {
			if entry.Position != 0 {
				allDefault = false
				break
			}
		}
		if allDefault {
			for i := range result {
				result[i].Position = i
			}
		}
	}
	seen := make(map[int]struct{}, len(result))
	for _, entry := range result {
		if entry.Position < 0 {
			return nil, validationError("saved diet meal position must be non-negative")
		}
		if _, ok := seen[entry.Position]; ok {
			return nil, validationError("saved diet meal positions must be unique")
		}
		seen[entry.Position] = struct{}{}
	}
	return result, nil
}

// normalizeSavedDietName supplies a stable default while rejecting only invalid NUL data.
// Implements DESIGN-008 SavedDataRepository.
func normalizeSavedDietName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Daily Diet"
	}
	return strings.TrimSpace(name)
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

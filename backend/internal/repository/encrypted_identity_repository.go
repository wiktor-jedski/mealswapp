package repository

import (
	"context"
	_ "embed"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService encrypted create query.
//
//go:embed sql/encrypted_user_create.sql
var encryptedUserCreateSQL string

// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService encrypted lookup query.
//
//go:embed sql/encrypted_user_get_by_digest.sql
var encryptedUserGetByDigestSQL string

// Implements DESIGN-006 AuthController encrypted user-by-id query.
//
//go:embed sql/encrypted_user_get_by_id.sql
var encryptedUserGetByIDSQL string

// Implements DESIGN-013 EncryptionService lookup digest reindex query.
//
//go:embed sql/encrypted_user_update_digest.sql
var encryptedUserUpdateDigestSQL string

// Implements DESIGN-008 AccountDeleter production account delete query.
//
//go:embed sql/user_delete_account.sql
var userDeleteAccountSQL string

// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService encrypted upsert query.
//
//go:embed sql/encrypted_oauth_identity_upsert.sql
var encryptedOAuthIdentityUpsertSQL string

// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService encrypted lookup query.
//
//go:embed sql/encrypted_oauth_identity_get.sql
var encryptedOAuthIdentityGetSQL string

// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService encrypted display-name query.
//
//go:embed sql/encrypted_profile_get_or_create.sql
var encryptedProfileGetOrCreateSQL string

// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService encrypted display-name query.
//
//go:embed sql/encrypted_profile_update.sql
var encryptedProfileUpdateSQL string

// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService encrypted query insert.
//
//go:embed sql/encrypted_search_history_add.sql
var encryptedSearchHistoryAddSQL string

// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService encrypted query list.
//
//go:embed sql/encrypted_search_history_list.sql
var encryptedSearchHistoryListSQL string

// PostgresEncryptedIdentityRepository persists encrypted account PII in PostgreSQL.
// Implements DESIGN-013 EncryptionService.
type PostgresEncryptedIdentityRepository struct {
	db sqlExecutor
}

var _ AccountDeletionRepository = (*PostgresEncryptedIdentityRepository)(nil)
var _ EncryptedSearchHistoryRepository = (*PostgresEncryptedIdentityRepository)(nil)
var _ EncryptedUserProfileRepository = (*PostgresEncryptedIdentityRepository)(nil)

// NewPostgresEncryptedIdentityRepository creates an encrypted PII repository.
// Implements DESIGN-013 EncryptionService.
func NewPostgresEncryptedIdentityRepository(db sqlExecutor) *PostgresEncryptedIdentityRepository {
	return &PostgresEncryptedIdentityRepository{db: db}
}

// CreateUser stores encrypted account email and deterministic lookup digest.
// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) CreateUser(ctx context.Context, user EncryptedAuthUser) (uuid.UUID, error) {
	if err := validateEncryptedAuthUser(user); err != nil {
		return uuid.Nil, err
	}
	role := user.Role
	if role == "" {
		role = UserRoleUser
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, encryptedUserCreateSQL, user.Email.KeyVersion, user.Email.Nonce, user.Email.Ciphertext, user.NormalizedEmailDigest.KeyVersion, user.NormalizedEmailDigest.Value, string(role), user.EmailVerified, user.PasswordHash, user.PasswordSalt).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "create encrypted user")
	}
	return id, nil
}

// GetUserByNormalizedEmailDigest loads encrypted account identity by HMAC lookup digest.
// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) GetUserByNormalizedEmailDigest(ctx context.Context, digest LookupDigest) (EncryptedAuthUser, error) {
	if err := validateLookupDigest(digest); err != nil {
		return EncryptedAuthUser{}, err
	}
	row := r.db.QueryRow(ctx, encryptedUserGetByDigestSQL, digest.KeyVersion, digest.Value)
	return scanEncryptedAuthUser(row)
}

// GetEncryptedUserByID loads encrypted account identity by user ID.
// Implements DESIGN-006 AuthController.
func (r *PostgresEncryptedIdentityRepository) GetEncryptedUserByID(ctx context.Context, userID uuid.UUID) (EncryptedAuthUser, error) {
	if userID == uuid.Nil {
		return EncryptedAuthUser{}, validationError("user id is required")
	}
	row := r.db.QueryRow(ctx, encryptedUserGetByIDSQL, userID)
	return scanEncryptedAuthUser(row)
}

// ReindexUserEmailDigest updates lookup material after deriving a new key-version digest.
// Implements DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) ReindexUserEmailDigest(ctx context.Context, userID uuid.UUID, digest LookupDigest) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if err := validateLookupDigest(digest); err != nil {
		return err
	}
	var updated uuid.UUID
	err := r.db.QueryRow(ctx, encryptedUserUpdateDigestSQL, userID, digest.KeyVersion, digest.Value).Scan(&updated)
	if err != nil {
		return mapPostgresError(err, "reindex encrypted user email digest")
	}
	return nil
}

// DeleteUserAccount removes a user row and cascaded user-owned production data.
// Implements DESIGN-008 AccountDeleter.
func (r *PostgresEncryptedIdentityRepository) DeleteUserAccount(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if _, err := r.db.Exec(ctx, userDeleteAccountSQL, userID); err != nil {
		return mapPostgresError(err, "delete user account")
	}
	return nil
}

// UpsertOAuthIdentity stores encrypted OAuth provider identity fields.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) UpsertOAuthIdentity(ctx context.Context, identity EncryptedOAuthIdentity) (uuid.UUID, error) {
	if err := validateEncryptedOAuthIdentity(identity); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, encryptedOAuthIdentityUpsertSQL, identity.UserID, identity.Provider, identity.ProviderUserID.KeyVersion, identity.ProviderUserID.Nonce, identity.ProviderUserID.Ciphertext, identity.ProviderUserIDDigest.KeyVersion, identity.ProviderUserIDDigest.Value, identity.Email.KeyVersion, identity.Email.Nonce, identity.Email.Ciphertext).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "upsert encrypted oauth identity")
	}
	return id, nil
}

// GetOAuthIdentity loads encrypted provider identity by deterministic digest.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) GetOAuthIdentity(ctx context.Context, provider string, digest LookupDigest) (EncryptedOAuthIdentity, error) {
	if strings.TrimSpace(provider) == "" {
		return EncryptedOAuthIdentity{}, validationError("provider is required")
	}
	if err := validateLookupDigest(digest); err != nil {
		return EncryptedOAuthIdentity{}, err
	}
	row := r.db.QueryRow(ctx, encryptedOAuthIdentityGetSQL, provider, digest.KeyVersion, digest.Value)
	return scanEncryptedOAuthIdentity(row)
}

// GetOrCreateEncryptedProfile returns encrypted display-name profile data.
// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) GetOrCreateEncryptedProfile(ctx context.Context, userID uuid.UUID) (EncryptedUserProfile, error) {
	if userID == uuid.Nil {
		return EncryptedUserProfile{}, validationError("user id is required")
	}
	row := r.db.QueryRow(ctx, encryptedProfileGetOrCreateSQL, userID)
	return scanEncryptedUserProfile(row)
}

// UpdateEncryptedProfile stores encrypted display-name PII with preferences.
// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) UpdateEncryptedProfile(ctx context.Context, profile EncryptedUserProfile) (EncryptedUserProfile, error) {
	if profile.UserID == uuid.Nil {
		return EncryptedUserProfile{}, validationError("user id is required")
	}
	if profile.DisplayName != nil && !validEncryptedField(*profile.DisplayName) {
		return EncryptedUserProfile{}, validationError("display name envelope is invalid")
	}
	if profile.UnitSystem != UnitSystemMetric && profile.UnitSystem != UnitSystemImperial {
		return EncryptedUserProfile{}, validationError("unit system is invalid")
	}
	if profile.ThemePreference != "system" && profile.ThemePreference != "light" && profile.ThemePreference != "dark" {
		return EncryptedUserProfile{}, validationError("theme preference is invalid")
	}
	keyVersion, nonce, ciphertext := nullableEncryptedField(profile.DisplayName)
	row := r.db.QueryRow(ctx, encryptedProfileUpdateSQL, profile.UserID, keyVersion, nonce, ciphertext, string(profile.UnitSystem), profile.ThemePreference)
	return scanEncryptedUserProfile(row)
}

// AddEncryptedHistory stores encrypted search-history query text.
// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) AddEncryptedHistory(ctx context.Context, entry EncryptedSearchHistoryEntry) (uuid.UUID, error) {
	if entry.UserID == uuid.Nil {
		return uuid.Nil, validationError("user id is required")
	}
	if !validEncryptedField(entry.Query) {
		return uuid.Nil, validationError("history query envelope is invalid")
	}
	if strings.TrimSpace(entry.Mode) == "" {
		return uuid.Nil, validationError("history mode is required")
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, encryptedSearchHistoryAddSQL, entry.UserID, entry.Query.KeyVersion, entry.Query.Nonce, entry.Query.Ciphertext, entry.Mode, entry.FiltersHash).Scan(&id)
	if err != nil {
		return uuid.Nil, mapPostgresError(err, "add encrypted search history")
	}
	return id, nil
}

// ListEncryptedHistory returns encrypted recent search history scoped to one user.
// Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService.
func (r *PostgresEncryptedIdentityRepository) ListEncryptedHistory(ctx context.Context, userID uuid.UUID, limit int) ([]EncryptedSearchHistoryEntry, error) {
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, encryptedSearchHistoryListSQL, userID, limit)
	if err != nil {
		return nil, mapPostgresError(err, "list encrypted search history")
	}
	defer rows.Close()
	entries := []EncryptedSearchHistoryEntry{}
	for rows.Next() {
		var entry EncryptedSearchHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Query.KeyVersion, &entry.Query.Nonce, &entry.Query.Ciphertext, &entry.Mode, &entry.FiltersHash, &entry.CreatedAt); err != nil {
			return nil, mapPostgresError(err, "scan encrypted search history")
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate encrypted search history")
	}
	return entries, nil
}

// validateEncryptedAuthUser checks required encrypted account fields.
// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService.
func validateEncryptedAuthUser(user EncryptedAuthUser) error {
	if !validEncryptedField(user.Email) {
		return validationError("email envelope is invalid")
	}
	if err := validateLookupDigest(user.NormalizedEmailDigest); err != nil {
		return err
	}
	if user.Role != "" && user.Role != UserRoleUser && user.Role != UserRoleAdmin {
		return validationError("user role is invalid")
	}
	if (user.PasswordHash == nil) != (user.PasswordSalt == nil) {
		return validationError("password hash and salt must be provided together")
	}
	return nil
}

// validateEncryptedOAuthIdentity checks required encrypted OAuth fields.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
func validateEncryptedOAuthIdentity(identity EncryptedOAuthIdentity) error {
	if identity.UserID == uuid.Nil {
		return validationError("user id is required")
	}
	if strings.TrimSpace(identity.Provider) == "" {
		return validationError("provider is required")
	}
	if !validEncryptedField(identity.ProviderUserID) || !validEncryptedField(identity.Email) {
		return validationError("oauth identity envelope is invalid")
	}
	if err := validateLookupDigest(identity.ProviderUserIDDigest); err != nil {
		return err
	}
	return nil
}

// validateLookupDigest checks deterministic lookup material.
// Implements DESIGN-013 EncryptionService.
func validateLookupDigest(digest LookupDigest) error {
	if strings.TrimSpace(digest.KeyVersion) == "" || strings.TrimSpace(digest.Value) == "" {
		return validationError("lookup digest is required")
	}
	return nil
}

// validEncryptedField reports whether an encrypted field envelope is complete.
// Implements DESIGN-013 EncryptionService.
func validEncryptedField(field EncryptedField) bool {
	return strings.TrimSpace(field.KeyVersion) != "" && len(field.Nonce) > 0 && len(field.Ciphertext) > 0
}

// nullableEncryptedField maps optional encrypted fields to nullable SQL arguments.
// Implements DESIGN-013 EncryptionService.
func nullableEncryptedField(field *EncryptedField) (*string, []byte, []byte) {
	if field == nil {
		return nil, nil, nil
	}
	return &field.KeyVersion, field.Nonce, field.Ciphertext
}

// scanEncryptedAuthUser reads encrypted account identity from PostgreSQL.
// Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService.
func scanEncryptedAuthUser(row pgx.Row) (EncryptedAuthUser, error) {
	var user EncryptedAuthUser
	if err := row.Scan(&user.ID, &user.Email.KeyVersion, &user.Email.Nonce, &user.Email.Ciphertext, &user.NormalizedEmailDigest.KeyVersion, &user.NormalizedEmailDigest.Value, &user.EmailVerified, &user.Role, &user.PasswordHash, &user.PasswordSalt, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return EncryptedAuthUser{}, mapPostgresError(err, "scan encrypted user")
	}
	return user, nil
}

// scanEncryptedOAuthIdentity reads encrypted OAuth identity from PostgreSQL.
// Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService.
func scanEncryptedOAuthIdentity(row pgx.Row) (EncryptedOAuthIdentity, error) {
	var identity EncryptedOAuthIdentity
	if err := row.Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID.KeyVersion, &identity.ProviderUserID.Nonce, &identity.ProviderUserID.Ciphertext, &identity.ProviderUserIDDigest.KeyVersion, &identity.ProviderUserIDDigest.Value, &identity.Email.KeyVersion, &identity.Email.Nonce, &identity.Email.Ciphertext, &identity.CreatedAt); err != nil {
		return EncryptedOAuthIdentity{}, mapPostgresError(err, "scan encrypted oauth identity")
	}
	return identity, nil
}

// scanEncryptedUserProfile reads encrypted profile data from PostgreSQL.
// Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService.
func scanEncryptedUserProfile(row pgx.Row) (EncryptedUserProfile, error) {
	var profile EncryptedUserProfile
	var keyVersion *string
	var nonce []byte
	var ciphertext []byte
	if err := row.Scan(&profile.UserID, &keyVersion, &nonce, &ciphertext, &profile.UnitSystem, &profile.ThemePreference, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
		return EncryptedUserProfile{}, mapPostgresError(err, "scan encrypted profile")
	}
	if keyVersion != nil {
		profile.DisplayName = &EncryptedField{KeyVersion: *keyVersion, Nonce: nonce, Ciphertext: ciphertext}
	}
	return profile, nil
}

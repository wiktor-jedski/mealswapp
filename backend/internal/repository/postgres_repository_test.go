package repository

// Implements DESIGN-005 ClassificationEntity.
// Implements DESIGN-005 MicronutrientVocabulary.

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
)

const testDatabaseURL = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"

// Implements DESIGN-005 RepositoryInterfaces integration fixtures.
//
//go:embed sql/testdata/advisory_lock.sql
var testAdvisoryLockSQL string

//go:embed sql/testdata/advisory_unlock.sql
var testAdvisoryUnlockSQL string

//go:embed sql/testdata/user_create.sql
var testUserCreateSQL string

//go:embed sql/testdata/oauth_user_create.sql
var testOAuthUserCreateSQL string

//go:embed sql/testdata/oauth_identity_create.sql
var testOAuthIdentityCreateSQL string

//go:embed sql/testdata/invalid_password_pair_create.sql
var testInvalidPasswordPairCreateSQL string

//go:embed sql/testdata/food_fixture_create.sql
var testFoodFixtureCreateSQL string

//go:embed sql/testdata/food_classification_fixture_create.sql
var testFoodClassificationFixtureCreateSQL string

//go:embed sql/testdata/inactive_vocabulary_upsert.sql
var testInactiveVocabularyUpsertSQL string

//go:embed sql/testdata/food_exists_by_name.sql
var testFoodExistsByNameSQL string

//go:embed sql/testdata/user_delete.sql
var testUserDeleteSQL string

//go:embed sql/testdata/entitlement_count_by_user.sql
var testEntitlementCountByUserSQL string

//go:embed sql/testdata/stripe_dead_letter_get.sql
var testStripeDeadLetterGetSQL string

//go:embed sql/testdata/food_name_fixture_create.sql
var testFoodNameFixtureCreateSQL string

//go:embed sql/testdata/collision_food_create.sql
var testCollisionFoodCreateSQL string

//go:embed sql/testdata/collision_meal_create.sql
var testCollisionMealCreateSQL string

//go:embed sql/testdata/collision_ingredient_create.sql
var testCollisionIngredientCreateSQL string

//go:embed sql/testdata/invalid_liquid_without_density_create.sql
var testInvalidLiquidWithoutDensityCreateSQL string

//go:embed sql/testdata/invalid_liquid_without_density_kind_create.sql
var testInvalidLiquidWithoutDensityKindCreateSQL string

func openRepositoryTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("MEALSWAPP_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = testDatabaseURL
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres unavailable: %v", err)
	}
	if _, err := pool.Exec(ctx, testAdvisoryLockSQL); err != nil {
		pool.Close()
		t.Fatalf("acquire repository test database lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), testAdvisoryUnlockSQL)
	})

	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		pool.Close()
		t.Fatalf("resolve migration dir: %v", err)
	}
	if err := migrations.Run(ctx, pool, "down", migrationDir); err != nil {
		pool.Close()
		t.Fatalf("reset migrations down: %v", err)
	}
	if err := migrations.Run(ctx, pool, "up", migrationDir); err != nil {
		pool.Close()
		t.Fatalf("apply migrations up: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func createRepositoryUser(t *testing.T, ctx context.Context, db *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := db.QueryRow(ctx, testUserCreateSQL, email).Scan(&id); err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return id
}

func TestUserIdentitySchemaSupportsOAuthOnlyUsers(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()

	var userID uuid.UUID
	if err := db.QueryRow(ctx, testOAuthUserCreateSQL).Scan(&userID); err != nil {
		t.Fatalf("insert OAuth-only user: %v", err)
	}
	if _, err := db.Exec(ctx, testOAuthIdentityCreateSQL, userID); err != nil {
		t.Fatalf("insert OAuth identity: %v", err)
	}

	if _, err := db.Exec(ctx, testInvalidPasswordPairCreateSQL); err == nil {
		t.Fatal("insert user with hash but no salt error = nil, want constraint violation")
	}
}

func TestPostgresEncryptedIdentityRepository(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresEncryptedIdentityRepository(db)

	email := EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("email-nonce"), Ciphertext: []byte("email-ciphertext")}
	digest := LookupDigest{KeyVersion: "lookup-v1", Value: "email-digest"}
	hash := "fixture-hash"
	salt := "fixture-salt"
	userID, err := repo.CreateUser(ctx, EncryptedAuthUser{
		Email:                 email,
		NormalizedEmailDigest: digest,
		Role:                  UserRoleUser,
		PasswordHash:          &hash,
		PasswordSalt:          &salt,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := repo.CreateUser(ctx, EncryptedAuthUser{Email: email, NormalizedEmailDigest: digest}); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("CreateUser() duplicate digest error = %v, want conflict", err)
	}
	storedUser, err := repo.GetUserByNormalizedEmailDigest(ctx, digest)
	if err != nil {
		t.Fatalf("GetUserByNormalizedEmailDigest() error = %v", err)
	}
	if storedUser.ID != userID || storedUser.Email.KeyVersion != "pii-v1" || storedUser.NormalizedEmailDigest != digest {
		t.Fatalf("stored user = %#v", storedUser)
	}
	storedByID, err := repo.GetEncryptedUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("GetEncryptedUserByID() error = %v", err)
	}
	if storedByID.ID != userID || storedByID.NormalizedEmailDigest != digest {
		t.Fatalf("stored user by id = %#v", storedByID)
	}
	newDigest := LookupDigest{KeyVersion: "lookup-v2", Value: "email-digest-v2"}
	if err := repo.ReindexUserEmailDigest(ctx, userID, newDigest); err != nil {
		t.Fatalf("ReindexUserEmailDigest() error = %v", err)
	}
	if _, err := repo.GetUserByNormalizedEmailDigest(ctx, digest); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("old digest lookup error = %v, want not found", err)
	}
	storedUser, err = repo.GetUserByNormalizedEmailDigest(ctx, newDigest)
	if err != nil {
		t.Fatalf("new digest lookup error = %v", err)
	}
	if storedUser.ID != userID || storedUser.NormalizedEmailDigest != newDigest {
		t.Fatalf("reindexed user = %#v", storedUser)
	}
	providerID := EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("provider-nonce"), Ciphertext: []byte("provider-ciphertext")}
	providerDigest := LookupDigest{KeyVersion: "lookup-v1", Value: "provider-digest"}
	oauthID, err := repo.UpsertOAuthIdentity(ctx, EncryptedOAuthIdentity{
		UserID:               userID,
		Provider:             "google",
		ProviderUserID:       providerID,
		ProviderUserIDDigest: providerDigest,
		Email:                email,
	})
	if err != nil {
		t.Fatalf("UpsertOAuthIdentity() error = %v", err)
	}
	storedOAuth, err := repo.GetOAuthIdentity(ctx, "google", providerDigest)
	if err != nil {
		t.Fatalf("GetOAuthIdentity() error = %v", err)
	}
	if storedOAuth.ID != oauthID || storedOAuth.ProviderUserIDDigest != providerDigest || storedOAuth.ProviderUserID.KeyVersion != "pii-v1" {
		t.Fatalf("stored oauth = %#v", storedOAuth)
	}
	var legacyProviderID, legacyOAuthEmail string
	if err := db.QueryRow(ctx, "SELECT provider_user_id, email FROM oauth_identities WHERE id = $1", oauthID).Scan(&legacyProviderID, &legacyOAuthEmail); err != nil {
		t.Fatalf("select legacy oauth fields: %v", err)
	}
	if !strings.HasPrefix(legacyProviderID, "encrypted:") || legacyOAuthEmail != "encrypted" {
		t.Fatalf("legacy oauth columns = %q, %q", legacyProviderID, legacyOAuthEmail)
	}

	createdProfile, err := repo.GetOrCreateEncryptedProfile(ctx, userID)
	if err != nil {
		t.Fatalf("GetOrCreateEncryptedProfile() first call error = %v", err)
	}
	if createdProfile.UserID != userID || createdProfile.UnitSystem != UnitSystemMetric || createdProfile.ThemePreference != "system" {
		t.Fatalf("created encrypted profile = %#v", createdProfile)
	}
	displayName := EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("display-nonce"), Ciphertext: []byte("display-ciphertext")}
	profile, err := repo.UpdateEncryptedProfile(ctx, EncryptedUserProfile{
		UserID:          userID,
		DisplayName:     &displayName,
		UnitSystem:      UnitSystemImperial,
		ThemePreference: "dark",
	})
	if err != nil {
		t.Fatalf("UpdateEncryptedProfile() error = %v", err)
	}
	if profile.DisplayName == nil || profile.DisplayName.KeyVersion != "pii-v1" || profile.UnitSystem != UnitSystemImperial {
		t.Fatalf("encrypted profile = %#v", profile)
	}
	storedProfile, err := repo.GetOrCreateEncryptedProfile(ctx, userID)
	if err != nil {
		t.Fatalf("GetOrCreateEncryptedProfile() error = %v", err)
	}
	if storedProfile.DisplayName == nil || storedProfile.DisplayName.KeyVersion != "pii-v1" || storedProfile.ThemePreference != "dark" {
		t.Fatalf("stored encrypted profile = %#v", storedProfile)
	}
	var legacyDisplayName string
	if err := db.QueryRow(ctx, "SELECT coalesce(display_name, '') FROM user_profiles WHERE user_id = $1", userID).Scan(&legacyDisplayName); err != nil {
		t.Fatalf("select legacy display name: %v", err)
	}
	if legacyDisplayName != "encrypted" {
		t.Fatalf("legacy display name = %q", legacyDisplayName)
	}

	historyID, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{
		UserID:      userID,
		Query:       EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("query-nonce"), Ciphertext: []byte("query-ciphertext")},
		Mode:        "food",
		FiltersHash: "hash",
	})
	if err != nil {
		t.Fatalf("AddEncryptedHistory() error = %v", err)
	}
	var legacyQuery string
	if err := db.QueryRow(ctx, "SELECT query FROM search_history WHERE id = $1", historyID).Scan(&legacyQuery); err != nil {
		t.Fatalf("select legacy query: %v", err)
	}
	if legacyQuery != "encrypted" {
		t.Fatalf("legacy query = %q", legacyQuery)
	}
	encryptedHistory, err := repo.ListEncryptedHistory(ctx, userID, 100)
	if err != nil {
		t.Fatalf("ListEncryptedHistory() error = %v", err)
	}
	if len(encryptedHistory) != 1 || encryptedHistory[0].ID != historyID || encryptedHistory[0].Query.KeyVersion != "pii-v1" {
		t.Fatalf("encrypted history = %#v", encryptedHistory)
	}
	if _, err := NewPostgresComplianceRepository(db).RequestDeletion(ctx, userID); err != nil {
		t.Fatalf("RequestDeletion() identity lockout error = %v", err)
	}
	if _, err := repo.GetEncryptedUserByID(ctx, userID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetEncryptedUserByID() pending deletion error = %v, want not found", err)
	}
	if _, err := repo.GetUserByNormalizedEmailDigest(ctx, newDigest); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetUserByNormalizedEmailDigest() pending deletion error = %v, want not found", err)
	}
}

func TestPostgresEncryptedIdentityRepositoryEncryptedHistoryLatest100(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresEncryptedIdentityRepository(db)
	userID := createRepositoryUser(t, ctx, db, "history-cap@example.test")
	otherUserID := createRepositoryUser(t, ctx, db, "history-cap-other@example.test")
	baseCreatedAt := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	duplicateCiphertext := []byte("duplicate-query-ciphertext")

	insertedIDs := make([]uuid.UUID, 0, 102)
	for i := 0; i < 102; i++ {
		ciphertext := []byte(fmt.Sprintf("query-ciphertext-%03d", i))
		if i == 100 || i == 101 {
			ciphertext = duplicateCiphertext
		}
		id, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{
			UserID: userID,
			Query: EncryptedField{
				KeyVersion: "pii-v1",
				Nonce:      []byte(fmt.Sprintf("query-nonce-%03d", i)),
				Ciphertext: ciphertext,
			},
			Mode:        "food",
			FiltersHash: fmt.Sprintf("filters-%03d", i),
		})
		if err != nil {
			t.Fatalf("AddEncryptedHistory(%d) error = %v", i, err)
		}
		if _, err := db.Exec(ctx, "UPDATE search_history SET created_at = $1 WHERE id = $2", baseCreatedAt.Add(time.Duration(i)*time.Second), id); err != nil {
			t.Fatalf("set created_at for history %d: %v", i, err)
		}
		insertedIDs = append(insertedIDs, id)
	}
	finalID, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{
		UserID: userID,
		Query:  EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("query-nonce-102"), Ciphertext: []byte("query-ciphertext-102")},
		Mode:   "food",
	})
	if err != nil {
		t.Fatalf("AddEncryptedHistory() final user row error = %v", err)
	}
	insertedIDs = append(insertedIDs, finalID)
	otherID, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{
		UserID: otherUserID,
		Query:  EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("other-query-nonce"), Ciphertext: []byte("other-query-ciphertext")},
		Mode:   "food",
	})
	if err != nil {
		t.Fatalf("AddEncryptedHistory() other user error = %v", err)
	}

	var persistedCount int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM search_history WHERE user_id = $1", userID).Scan(&persistedCount); err != nil {
		t.Fatalf("count capped history: %v", err)
	}
	if persistedCount != 100 {
		t.Fatalf("persisted history count = %d, want 100", persistedCount)
	}
	var oldestExists bool
	if err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM search_history WHERE id = $1)", insertedIDs[0]).Scan(&oldestExists); err != nil {
		t.Fatalf("check oldest history row: %v", err)
	}
	if oldestExists {
		t.Fatalf("oldest history row %s was not pruned", insertedIDs[0])
	}
	history, err := repo.ListEncryptedHistory(ctx, userID, 0)
	if err != nil {
		t.Fatalf("ListEncryptedHistory() error = %v", err)
	}
	if len(history) != 100 {
		t.Fatalf("history length = %d, want 100", len(history))
	}
	if history[0].ID != finalID || history[99].ID != insertedIDs[3] {
		t.Fatalf("history ordering/latest range first=%s last=%s, want first=%s last=%s", history[0].ID, history[99].ID, finalID, insertedIDs[3])
	}
	duplicateCount := 0
	for i, entry := range history {
		if entry.UserID != userID {
			t.Fatalf("history[%d].UserID = %s, want %s", i, entry.UserID, userID)
		}
		if entry.ID == otherID {
			t.Fatalf("history[%d] includes other user's row %s", i, otherID)
		}
		if i > 0 && history[i-1].CreatedAt.Before(entry.CreatedAt) {
			t.Fatalf("history order at %d: %s before %s", i, history[i-1].CreatedAt, entry.CreatedAt)
		}
		if string(entry.Query.Ciphertext) == string(duplicateCiphertext) {
			duplicateCount++
		}
	}
	if duplicateCount != 2 {
		t.Fatalf("duplicate encrypted query count = %d, want 2", duplicateCount)
	}
	otherHistory, err := repo.ListEncryptedHistory(ctx, otherUserID, 100)
	if err != nil {
		t.Fatalf("ListEncryptedHistory() other user error = %v", err)
	}
	if len(otherHistory) != 1 || otherHistory[0].ID != otherID {
		t.Fatalf("other user history = %#v, want only %s", otherHistory, otherID)
	}
}

func TestPostgresAccountLockoutRepository(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresAccountLockoutRepository(db)
	userID := createRepositoryUser(t, ctx, db, "lockout@example.test")
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	lockedUntil := now.Add(15 * time.Minute)

	state, err := repo.GetLockoutState(ctx, userID)
	if err != nil {
		t.Fatalf("GetLockoutState() error = %v", err)
	}
	if state.FailedLoginCount != 0 || state.LockedUntil != nil {
		t.Fatalf("initial state = %#v", state)
	}
	for i := 1; i <= 4; i++ {
		state, err = repo.RecordFailedLogin(ctx, userID, 5, lockedUntil, now)
		if err != nil {
			t.Fatalf("RecordFailedLogin(%d) error = %v", i, err)
		}
		if state.FailedLoginCount != i || state.LockedUntil != nil {
			t.Fatalf("failure %d state = %#v", i, state)
		}
	}
	state, err = repo.RecordFailedLogin(ctx, userID, 5, lockedUntil, now)
	if err != nil {
		t.Fatalf("RecordFailedLogin(lock) error = %v", err)
	}
	if state.FailedLoginCount != 5 || state.LockedUntil == nil || !state.LockedUntil.Equal(lockedUntil) {
		t.Fatalf("locked state = %#v", state)
	}
	state, err = repo.ResetFailedLogins(ctx, userID)
	if err != nil {
		t.Fatalf("ResetFailedLogins() error = %v", err)
	}
	if state.FailedLoginCount != 0 || state.LockedUntil != nil {
		t.Fatalf("reset state = %#v", state)
	}

	expiredLock := now.Add(-time.Minute)
	if _, err := db.Exec(ctx, `UPDATE users SET failed_login_count = 5, locked_until = $2 WHERE id = $1`, userID, expiredLock); err != nil {
		t.Fatalf("seed expired lock: %v", err)
	}
	state, err = repo.RecordFailedLogin(ctx, userID, 5, lockedUntil, now)
	if err != nil {
		t.Fatalf("RecordFailedLogin(expired) error = %v", err)
	}
	if state.FailedLoginCount != 1 || state.LockedUntil != nil {
		t.Fatalf("expired lock restart state = %#v", state)
	}
}

func TestPostgresRegistrationRepositoryCreatesUserWithConsentTransactionally(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresRegistrationRepository(db)
	user := EncryptedAuthUser{
		Email:                 EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("nonce"), Ciphertext: []byte("ciphertext")},
		NormalizedEmailDigest: LookupDigest{KeyVersion: "lookup-v1", Value: "registration-digest"},
		Role:                  UserRoleUser,
	}

	userID, err := repo.CreateUserWithConsent(ctx, user, "privacy-v1", "terms-v1")
	if err != nil {
		t.Fatalf("CreateUserWithConsent() error = %v", err)
	}
	hasConsent, err := NewPostgresComplianceRepository(db).HasRequiredConsent(ctx, userID, "privacy-v1", "terms-v1")
	if err != nil {
		t.Fatalf("HasRequiredConsent() error = %v", err)
	}
	if !hasConsent {
		t.Fatal("registration consent was not persisted")
	}
	if _, err := repo.CreateUserWithConsent(ctx, user, "privacy-v1", "terms-v1"); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("CreateUserWithConsent() duplicate error = %v, want conflict", err)
	}
	var consentCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM consent_records WHERE privacy_policy_version = 'privacy-v1' AND terms_version = 'terms-v1'`).Scan(&consentCount); err != nil {
		t.Fatalf("count consent records: %v", err)
	}
	if consentCount != 1 {
		t.Fatalf("duplicate registration committed consent rows = %d", consentCount)
	}
}

func TestPostgresSessionRepository(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresSessionRepository(db)
	userID := createRepositoryUser(t, ctx, db, "session@example.test")
	familyID := uuid.New()
	accessExpiresAt := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)
	refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Second)

	sessionID, err := repo.CreateSession(ctx, UserSession{
		UserID:           userID,
		RefreshTokenHash: "refresh-hash-1",
		RefreshFamilyID:  familyID,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	session, err := repo.GetSessionByRefreshTokenHash(ctx, "refresh-hash-1")
	if err != nil {
		t.Fatalf("GetSessionByRefreshTokenHash() error = %v", err)
	}
	if session.ID != sessionID || session.UserID != userID || session.RefreshFamilyID != familyID || session.RevokedAt != nil {
		t.Fatalf("session = %#v", session)
	}
	if err := repo.RevokeSession(ctx, sessionID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	session, err = repo.GetSessionByRefreshTokenHash(ctx, "refresh-hash-1")
	if err != nil {
		t.Fatalf("GetSessionByRefreshTokenHash() after revoke error = %v", err)
	}
	if session.RevokedAt == nil {
		t.Fatalf("session was not revoked: %#v", session)
	}
	secondID, err := repo.CreateSession(ctx, UserSession{
		UserID:           userID,
		RefreshTokenHash: "refresh-hash-2",
		RefreshFamilyID:  familyID,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	})
	if err != nil {
		t.Fatalf("CreateSession() second error = %v", err)
	}
	if err := repo.RevokeSessionFamily(ctx, familyID); err != nil {
		t.Fatalf("RevokeSessionFamily() error = %v", err)
	}
	var revokedAt *time.Time
	if err := db.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, secondID).Scan(&revokedAt); err != nil {
		t.Fatalf("select revoked family session: %v", err)
	}
	if revokedAt == nil {
		t.Fatal("session family was not revoked")
	}
}

func TestPostgresAccountVerificationRepository(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresAccountVerificationRepository(db)
	sessionRepo := NewPostgresSessionRepository(db)
	userID := createRepositoryUser(t, ctx, db, "verify-reset@example.test")
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.MarkEmailVerified(ctx, userID); err != nil {
		t.Fatalf("MarkEmailVerified() error = %v", err)
	}
	var verified bool
	if err := db.QueryRow(ctx, `SELECT email_verified FROM users WHERE id = $1`, userID).Scan(&verified); err != nil {
		t.Fatalf("select verified: %v", err)
	}
	if !verified {
		t.Fatal("email_verified was not updated")
	}
	if err := repo.CreatePasswordResetToken(ctx, PasswordResetToken{TokenHash: "reset-hash", UserID: userID, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("CreatePasswordResetToken() error = %v", err)
	}
	token, err := repo.ConsumePasswordResetToken(ctx, "reset-hash", now)
	if err != nil {
		t.Fatalf("ConsumePasswordResetToken() error = %v", err)
	}
	if token.UserID != userID || token.UsedAt == nil || token.TokenHash != "reset-hash" {
		t.Fatalf("consumed token = %#v", token)
	}
	if _, err := repo.ConsumePasswordResetToken(ctx, "reset-hash", now); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("ConsumePasswordResetToken() reuse error = %v, want not found", err)
	}
	if err := repo.CreatePasswordResetToken(ctx, PasswordResetToken{TokenHash: "expired-reset-hash", UserID: userID, ExpiresAt: now.Add(-time.Minute)}); err != nil {
		t.Fatalf("CreatePasswordResetToken() expired seed error = %v", err)
	}
	if _, err := repo.ConsumePasswordResetToken(ctx, "expired-reset-hash", now); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("ConsumePasswordResetToken() expired error = %v, want not found", err)
	}
	if err := repo.UpdatePassword(ctx, userID, "new-hash", "new-salt"); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	var passwordHash string
	if err := db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&passwordHash); err != nil {
		t.Fatalf("select password hash: %v", err)
	}
	if passwordHash != "new-hash" {
		t.Fatalf("password hash = %q", passwordHash)
	}
	sessionID, err := sessionRepo.CreateSession(ctx, UserSession{UserID: userID, RefreshTokenHash: "reset-session", RefreshFamilyID: uuid.New(), AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := sessionRepo.RevokeUserSessions(ctx, userID); err != nil {
		t.Fatalf("RevokeUserSessions() error = %v", err)
	}
	var revokedAt *time.Time
	if err := db.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, sessionID).Scan(&revokedAt); err != nil {
		t.Fatalf("select revoked session: %v", err)
	}
	if revokedAt == nil {
		t.Fatal("password reset did not revoke sessions")
	}
}

func TestPostgresClassificationRepositoryUpsertListAndSoftDelete(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresClassificationRepository(db)

	rootID, err := repo.Upsert(ctx, ClassificationEntity{Name: "Fruit", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("Upsert() root error = %v", err)
	}
	sameRootID, err := repo.Upsert(ctx, ClassificationEntity{Name: " fruit ", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("Upsert() duplicate root error = %v", err)
	}
	if sameRootID != rootID {
		t.Fatalf("duplicate root ID = %s, want %s", sameRootID, rootID)
	}

	childID, err := repo.Upsert(ctx, ClassificationEntity{Name: "Citrus", Kind: ClassificationKindFoodCategory, ParentID: &rootID})
	if err != nil {
		t.Fatalf("Upsert() child error = %v", err)
	}

	classifications, err := repo.List(ctx, ClassificationKindFoodCategory)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(classifications) != 2 {
		t.Fatalf("List() length = %d, want 2: %#v", len(classifications), classifications)
	}

	inUse, err := repo.IsInUse(ctx, childID)
	if err != nil {
		t.Fatalf("IsInUse() error = %v", err)
	}
	if inUse {
		t.Fatalf("IsInUse() = true for unattached classification")
	}
	if err := repo.SoftDelete(ctx, childID); err != nil {
		t.Fatalf("SoftDelete() unused error = %v", err)
	}

	classifications, err = repo.List(ctx, ClassificationKindFoodCategory)
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(classifications) != 1 {
		t.Fatalf("List() after delete length = %d, want 1", len(classifications))
	}
}

func TestPostgresClassificationRepositoryInUseSafeguard(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresClassificationRepository(db)

	classificationID, err := repo.Upsert(ctx, ClassificationEntity{Name: "Protein", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if _, err := db.Exec(ctx, testFoodFixtureCreateSQL); err != nil {
		t.Fatalf("create food fixture: %v", err)
	}
	if _, err := db.Exec(ctx, testFoodClassificationFixtureCreateSQL, classificationID); err != nil {
		t.Fatalf("attach classification fixture: %v", err)
	}

	inUse, err := repo.IsInUse(ctx, classificationID)
	if err != nil {
		t.Fatalf("IsInUse() error = %v", err)
	}
	if !inUse {
		t.Fatalf("IsInUse() = false for attached classification")
	}
	if err := repo.SoftDelete(ctx, classificationID); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("SoftDelete() error = %v, want conflict", err)
	}
	if err := repo.SoftDelete(ctx, uuid.New()); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("SoftDelete() missing error = %v, want not found", err)
	}
}

func TestPostgresClassificationRepositoryValidation(t *testing.T) {
	repo := NewPostgresClassificationRepository(nil)
	if _, err := repo.Upsert(context.Background(), ClassificationEntity{Name: "x", Kind: "bad"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Upsert() invalid kind error = %v, want validation", err)
	}
	if _, err := repo.Upsert(context.Background(), ClassificationEntity{Kind: ClassificationKindFoodCategory}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Upsert() missing name error = %v, want validation", err)
	}
}

func TestPostgresVocabularyRepository(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresMicronutrientVocabularyRepository(db)

	if err := repo.Upsert(ctx, MicronutrientVocabularyEntry{Key: "Magnesium", DisplayName: "Magnesium", Unit: "mg", Active: true}); err != nil {
		t.Fatalf("Upsert() active error = %v", err)
	}
	if err := repo.Upsert(ctx, MicronutrientVocabularyEntry{Key: "Legacy", DisplayName: "Legacy", Unit: "mg", Active: false}); err != nil {
		t.Fatalf("Upsert() inactive error = %v", err)
	}

	allowed, err := repo.IsAllowed(ctx, "Magnesium")
	if err != nil {
		t.Fatalf("IsAllowed() active error = %v", err)
	}
	if !allowed {
		t.Fatalf("IsAllowed() active = false")
	}
	allowed, err = repo.IsAllowed(ctx, "Legacy")
	if err != nil {
		t.Fatalf("IsAllowed() inactive error = %v", err)
	}
	if allowed {
		t.Fatalf("IsAllowed() inactive = true")
	}
	allowed, err = repo.IsAllowed(ctx, "Missing")
	if err != nil {
		t.Fatalf("IsAllowed() missing error = %v", err)
	}
	if allowed {
		t.Fatalf("IsAllowed() missing = true")
	}

	entries, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}
	foundMagnesium := false
	for _, entry := range entries {
		if entry.Key == "Legacy" {
			t.Fatalf("ListActive() included inactive entry")
		}
		if entry.Key == "Magnesium" {
			foundMagnesium = true
		}
	}
	if !foundMagnesium {
		t.Fatalf("ListActive() missing Magnesium entry")
	}
}

func TestPostgresVocabularyRepositoryValidation(t *testing.T) {
	repo := NewPostgresMicronutrientVocabularyRepository(nil)
	if err := repo.Upsert(context.Background(), MicronutrientVocabularyEntry{Key: "x"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Upsert() invalid entry error = %v, want validation", err)
	}
}

func TestPostgresFoodItemRepositoryCRUDHydrationAndConversion(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	classificationRepo := NewPostgresClassificationRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)

	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Protein", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create food_category: %v", err)
	}
	functionalityID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Quick", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("create culinary_role: %v", err)
	}

	id, err := foodRepo.Create(ctx, FoodItemEntity{
		Name:                   "Tofu",
		PhysicalState:          PhysicalStateSolid,
		PrepTimeMinutes:        5,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           MacroValues{Protein: 8, Carbohydrates: 2, Fat: 4},
		Micros:                 MicroValues{"Sodium": 7},
		FoodCategories:         []ClassificationEntity{{ID: categoryID, Kind: ClassificationKindFoodCategory}},
		CulinaryRoles:          []ClassificationEntity{{ID: functionalityID, Kind: ClassificationKindCulinaryRole}},
		ImageURL:               "https://example.test/tofu.jpg",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	item, err := foodRepo.GetByID(ctx, id, RepositoryContext{UnitSystem: UnitSystemMetric})
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if item.Name != "Tofu" || item.MacrosPer100.Protein != 8 || item.Micros["Sodium"] != 7 {
		t.Fatalf("GetByID() item = %#v", item)
	}
	if len(item.FoodCategories) != 1 || item.FoodCategories[0].ID != categoryID {
		t.Fatalf("GetByID() food_category classifications = %#v", item.FoodCategories)
	}
	if len(item.CulinaryRoles) != 1 || item.CulinaryRoles[0].ID != functionalityID {
		t.Fatalf("GetByID() culinary_role classifications = %#v", item.CulinaryRoles)
	}

	imperial, err := foodRepo.GetByID(ctx, id, RepositoryContext{UnitSystem: UnitSystemImperial})
	if err != nil {
		t.Fatalf("GetByID() imperial error = %v", err)
	}
	if imperial.AverageUnitWeightGrams != 3.5274 {
		t.Fatalf("imperial average unit weight = %v, want 3.5274 oz", imperial.AverageUnitWeightGrams)
	}

	item.Name = "Firm Tofu"
	item.MacrosPer100.Protein = 9
	item.FoodCategories = nil
	if err := foodRepo.Update(ctx, item); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := foodRepo.GetByID(ctx, id, RepositoryContext{UnitSystem: UnitSystemMetric})
	if err != nil {
		t.Fatalf("GetByID() updated error = %v", err)
	}
	if updated.Name != "Firm Tofu" || updated.MacrosPer100.Protein != 9 {
		t.Fatalf("updated item = %#v", updated)
	}
	if len(updated.FoodCategories) != 0 || len(updated.CulinaryRoles) != 1 {
		t.Fatalf("updated classifications = food_category %#v culinary_role %#v", updated.FoodCategories, updated.CulinaryRoles)
	}

	if err := foodRepo.Delete(ctx, id); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := foodRepo.GetByID(ctx, id, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetByID() deleted error = %v, want not found", err)
	}
	deleted, err := foodRepo.GetByID(ctx, id, RepositoryContext{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("GetByID() include deleted error = %v", err)
	}
	if deleted.DeletedAt == nil {
		t.Fatalf("deleted item DeletedAt = nil")
	}
}

func TestPostgresFoodItemRepositoryValidationAndConflicts(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	classificationRepo := NewPostgresClassificationRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)

	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Category", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create food_category: %v", err)
	}
	functionalityID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Function", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("create culinary_role: %v", err)
	}

	valid := FoodItemEntity{
		Name:           "Apple",
		PhysicalState:  PhysicalStateSolid,
		MacrosPer100:   MacroValues{Protein: 1, Carbohydrates: 2, Fat: 3},
		Micros:         MicroValues{"Sodium": 1},
		FoodCategories: []ClassificationEntity{{ID: categoryID, Kind: ClassificationKindFoodCategory}},
	}
	if _, err := foodRepo.Create(ctx, valid); err != nil {
		t.Fatalf("Create() valid error = %v", err)
	}
	if _, err := foodRepo.Create(ctx, valid); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("Create() duplicate error = %v, want conflict", err)
	}

	invalidCases := []struct {
		name string
		item FoodItemEntity
		kind ErrorKind
	}{
		{name: "missing name", item: FoodItemEntity{PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}}, kind: ErrorKindValidation},
		{name: "invalid physical state", item: FoodItemEntity{Name: "Bad State", PhysicalState: "gas", MacrosPer100: MacroValues{}}, kind: ErrorKindValidation},
		{name: "negative prep", item: FoodItemEntity{Name: "Bad Prep", PhysicalState: PhysicalStateSolid, PrepTimeMinutes: -1, MacrosPer100: MacroValues{}}, kind: ErrorKindValidation},
		{name: "negative macro", item: FoodItemEntity{Name: "Bad Macro", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: -1}}, kind: ErrorKindValidation},
		{name: "solid macros exceed mass", item: FoodItemEntity{Name: "Impossible Solid", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 51, Carbohydrates: 50}}, kind: ErrorKindValidation},
		{name: "invalid micronutrient", item: FoodItemEntity{Name: "Bad Micro", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, Micros: MicroValues{"Na": 1}}, kind: ErrorKindInvalidMicronutrientKey},
		{name: "inactive micronutrient", item: FoodItemEntity{Name: "Inactive Micro", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, Micros: MicroValues{"Legacy": 1}}, kind: ErrorKindInvalidMicronutrientKey},
		{name: "missing classification", item: FoodItemEntity{Name: "Bad Classification", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, FoodCategories: []ClassificationEntity{{ID: uuid.New(), Kind: ClassificationKindFoodCategory}}}, kind: ErrorKindValidation},
		{name: "wrong classification kind", item: FoodItemEntity{Name: "Wrong Classification", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, FoodCategories: []ClassificationEntity{{ID: functionalityID, Kind: ClassificationKindFoodCategory}}}, kind: ErrorKindValidation},
	}
	if _, err := db.Exec(ctx, testInactiveVocabularyUpsertSQL); err != nil {
		t.Fatalf("insert inactive vocabulary: %v", err)
	}
	for _, tt := range invalidCases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := foodRepo.Create(ctx, tt.item); !IsKind(err, tt.kind) {
				t.Fatalf("Create() error = %v, want %s", err, tt.kind)
			}
		})
	}

	var invalidInserted bool
	if err := db.QueryRow(ctx, testFoodExistsByNameSQL, "Bad Classification").Scan(&invalidInserted); err != nil {
		t.Fatalf("check validation rollback: %v", err)
	}
	if invalidInserted {
		t.Fatalf("invalid food item was inserted despite validation failure")
	}
}

func TestPostgresFoodItemRepositoryRequiresLiquidDensity(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	foodRepo := NewPostgresFoodItemRepository(db)

	if _, err := db.Exec(ctx, testInvalidLiquidWithoutDensityCreateSQL); err == nil {
		t.Fatal("direct liquid insert without density error = nil, want constraint violation")
	}
	if _, err := db.Exec(ctx, testInvalidLiquidWithoutDensityKindCreateSQL); err == nil {
		t.Fatal("direct liquid insert without density source kind error = nil, want constraint violation")
	}

	for _, kind := range []string{"manual", "estimated", "imported"} {
		item := FoodItemEntity{
			Name:                      "Liquid " + kind,
			PhysicalState:             PhysicalStateLiquid,
			DensityGramsPerMilliliter: 1,
			DensitySourceKind:         kind,
			MacrosPer100:              MacroValues{},
		}
		if kind == "imported" {
			item.DensitySourceProvider = "fixture"
			item.DensitySourceFoodID = "liquid-" + kind
		}
		id, err := foodRepo.Create(ctx, item)
		if err != nil {
			t.Fatalf("Create() %s density error = %v", kind, err)
		}
		item.ID = id
		item.DensityGramsPerMilliliter = 0
		item.DensitySourceKind = ""
		item.DensitySourceProvider = ""
		item.DensitySourceFoodID = ""
		if err := foodRepo.Update(ctx, item); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("Update() %s without density error = %v, want validation", kind, err)
		}
	}
}

func TestPostgresFoodItemRepositorySearch(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	classificationRepo := NewPostgresClassificationRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)

	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Fruit", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create food_category: %v", err)
	}
	functionalityID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Breakfast", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("create culinary_role: %v", err)
	}
	excludedCategoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Snack", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create excluded food_category: %v", err)
	}
	excludedRoleID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Dessert", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("create excluded culinary_role: %v", err)
	}

	appleID, err := foodRepo.Create(ctx, FoodItemEntity{
		Name:            "Apple",
		PhysicalState:   PhysicalStateSolid,
		PrepTimeMinutes: 1,
		MacrosPer100:    MacroValues{Protein: 0.3, Carbohydrates: 14, Fat: 0.2},
		FoodCategories:  []ClassificationEntity{{ID: categoryID}},
		CulinaryRoles:   []ClassificationEntity{{ID: functionalityID}},
	})
	if err != nil {
		t.Fatalf("create apple: %v", err)
	}
	if _, err := foodRepo.Create(ctx, FoodItemEntity{
		Name:            "Banana",
		PhysicalState:   PhysicalStateSolid,
		PrepTimeMinutes: 2,
		MacrosPer100:    MacroValues{Protein: 1.1, Carbohydrates: 23, Fat: 0.3},
		FoodCategories:  []ClassificationEntity{{ID: categoryID}, {ID: excludedCategoryID}},
	}); err != nil {
		t.Fatalf("create banana: %v", err)
	}
	appleJuiceID, err := foodRepo.Create(ctx, FoodItemEntity{
		Name:                            "Apple Juice",
		PhysicalState:                   PhysicalStateLiquid,
		DensityGramsPerMilliliter:       1.04,
		DensitySourceKind:               "estimated",
		AverageServingVolumeMilliliters: 250,
		MacrosPer100:                    MacroValues{Protein: 0.1, Carbohydrates: 11, Fat: 0},
		FoodCategories:                  []ClassificationEntity{{ID: categoryID}},
		CulinaryRoles:                   []ClassificationEntity{{ID: excludedRoleID}},
	})
	if err != nil {
		t.Fatalf("create apple juice: %v", err)
	}

	maxPrep := 1
	items, total, err := foodRepo.Search(ctx, RepositoryQuery{
		RepositoryContext: RepositoryContext{UnitSystem: UnitSystemMetric},
		Name:              "Ap",
		FoodCategoryIDs:   []uuid.UUID{categoryID},
		CulinaryRoleIDs:   []uuid.UUID{functionalityID},
		MaxPrepMinutes:    &maxPrep,
		Limit:             10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != appleID {
		t.Fatalf("Search() total=%d items=%#v, want apple only", total, items)
	}
	items, total, err = foodRepo.Search(ctx, RepositoryQuery{Name: "Juice", Limit: 10})
	if err != nil {
		t.Fatalf("Search() infix food name error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != appleJuiceID {
		t.Fatalf("Search() infix food name total=%d items=%#v, want apple juice", total, items)
	}
	items, total, err = foodRepo.Search(ctx, RepositoryQuery{
		RepositoryContext:       RepositoryContext{UnitSystem: UnitSystemMetric},
		FoodCategoryIDs:         []uuid.UUID{categoryID},
		ExcludedFoodCategoryIDs: []uuid.UUID{excludedCategoryID},
		ExcludedCulinaryRoleIDs: []uuid.UUID{excludedRoleID},
		ExcludedAllergenIDs:     []uuid.UUID{excludedCategoryID},
		FoodObjectTypes:         []PhysicalState{PhysicalStateSolid},
		ExcludedFoodObjectTypes: []PhysicalState{PhysicalStateLiquid},
		Limit:                   10,
	})
	if err != nil {
		t.Fatalf("Search() exclusion filters error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != appleID {
		t.Fatalf("Search() exclusion filters total=%d items=%#v, want apple only", total, items)
	}

	if err := foodRepo.Delete(ctx, appleID); err != nil {
		t.Fatalf("Delete() apple error = %v", err)
	}
	items, total, err = foodRepo.Search(ctx, RepositoryQuery{Name: "Ap", CulinaryRoleIDs: []uuid.UUID{functionalityID}, Limit: 10})
	if err != nil {
		t.Fatalf("Search() deleted exclusion error = %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("Search() deleted exclusion total=%d items=%#v, want none", total, items)
	}

	items, total, err = foodRepo.Search(ctx, RepositoryQuery{FoodCategoryIDs: []uuid.UUID{excludedCategoryID}, Limit: 1, Offset: 99})
	if err != nil {
		t.Fatalf("Search() past final row error = %v", err)
	}
	if total != 1 || len(items) != 0 {
		t.Fatalf("Search() past final row total=%d items=%#v, want empty page with total 1", total, items)
	}
}

func TestPostgresMealRepositorySearch(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	classificationRepo := NewPostgresClassificationRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)
	mealRepo := NewPostgresMealRepository(db)

	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Breakfast", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create food_category: %v", err)
	}
	functionalityID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Quick", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("create culinary_role: %v", err)
	}

	_, err = foodRepo.Create(ctx, FoodItemEntity{
		Name:                   "Oat Bowl",
		PhysicalState:          PhysicalStateSolid,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           MacroValues{Protein: 5, Carbohydrates: 27, Fat: 3},
	})
	if err != nil {
		t.Fatalf("create oat food: %v", err)
	}
	_, err = foodRepo.Create(ctx, FoodItemEntity{
		Name:                   "Berry Bowl",
		PhysicalState:          PhysicalStateSolid,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           MacroValues{Protein: 1, Carbohydrates: 14, Fat: 0.3},
	})
	if err != nil {
		t.Fatalf("create berry food: %v", err)
	}

	oatMealID, err := mealRepo.Create(ctx, MealEntity{
		Type:                   MealTypeSingle,
		Name:                   "Oat Bowl",
		PhysicalState:          PhysicalStateSolid,
		PrepTimeMinutes:        3,
		Classifications:        []ClassificationEntity{{ID: categoryID}, {ID: functionalityID}},
		RecipeItems:            nil,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           MacroValues{Protein: 5, Carbohydrates: 27, Fat: 3},
	})
	if err != nil {
		t.Fatalf("create oat meal: %v", err)
	}
	berryMealID, err := mealRepo.Create(ctx, MealEntity{
		Type:                   MealTypeSingle,
		Name:                   "Berry Bowl",
		PhysicalState:          PhysicalStateSolid,
		PrepTimeMinutes:        8,
		Classifications:        []ClassificationEntity{{ID: categoryID}},
		AverageUnitWeightGrams: 100,
		MacrosPer100:           MacroValues{Protein: 1, Carbohydrates: 14, Fat: 0.3},
	})
	if err != nil {
		t.Fatalf("create berry meal: %v", err)
	}

	maxPrep := 5
	meals, total, err := mealRepo.Search(ctx, RepositoryQuery{
		Name:            "Oat",
		FoodCategoryIDs: []uuid.UUID{categoryID},
		CulinaryRoleIDs: []uuid.UUID{functionalityID},
		MaxPrepMinutes:  &maxPrep,
		Limit:           10,
	})
	if err != nil {
		t.Fatalf("Search() filtered error = %v", err)
	}
	if total != 1 || len(meals) != 1 || meals[0].ID != oatMealID {
		t.Fatalf("Search() filtered total=%d meals=%#v, want oat only", total, meals)
	}

	meals, total, err = mealRepo.Search(ctx, RepositoryQuery{Name: "Bowl", Limit: 10})
	if err != nil {
		t.Fatalf("Search() infix meal name error = %v", err)
	}
	if total != 2 || len(meals) != 2 || meals[0].ID != berryMealID || meals[1].ID != oatMealID {
		t.Fatalf("Search() infix meal name total=%d meals=%#v, want berry then oat", total, meals)
	}

	meals, total, err = mealRepo.Search(ctx, RepositoryQuery{Name: "Berry", Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("Search() page error = %v", err)
	}
	if total != 1 || len(meals) != 1 || meals[0].ID != berryMealID {
		t.Fatalf("Search() page total=%d meals=%#v, want berry", total, meals)
	}

	meals, total, err = mealRepo.Search(ctx, RepositoryQuery{Name: "x' OR 1=1 --", Limit: 10})
	if err != nil {
		t.Fatalf("Search() parameterized text error = %v", err)
	}
	if total != 0 || len(meals) != 0 {
		t.Fatalf("Search() parameterized text total=%d meals=%#v, want none", total, meals)
	}

	if err := mealRepo.Delete(ctx, oatMealID); err != nil {
		t.Fatalf("Delete() oat meal error = %v", err)
	}
	meals, total, err = mealRepo.Search(ctx, RepositoryQuery{Name: "Oat", Limit: 10})
	if err != nil {
		t.Fatalf("Search() deleted exclusion error = %v", err)
	}
	if total != 0 || len(meals) != 0 {
		t.Fatalf("Search() deleted exclusion total=%d meals=%#v, want none", total, meals)
	}

	meals, total, err = mealRepo.Search(ctx, RepositoryQuery{RepositoryContext: RepositoryContext{IncludeDeleted: true}, Name: "Oat", Limit: 10})
	if err != nil {
		t.Fatalf("Search() include deleted error = %v", err)
	}
	if total != 1 || len(meals) != 1 || meals[0].ID != oatMealID {
		t.Fatalf("Search() include deleted total=%d meals=%#v, want deleted oat", total, meals)
	}
}

func TestPostgresUserDataRepositories(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	profileRepo := NewPostgresUserProfileRepository(db)
	savedRepo := NewPostgresSavedDataRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)
	mealRepo := NewPostgresMealRepository(db)

	userID := createRepositoryUser(t, ctx, db, "profile@example.test")
	otherUserID := createRepositoryUser(t, ctx, db, "other-profile@example.test")

	profile, err := profileRepo.GetOrCreate(ctx, userID)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if profile.UserID != userID || profile.UnitSystem != UnitSystemMetric || profile.ThemePreference != "system" {
		t.Fatalf("default profile = %#v", profile)
	}

	result, err := profileRepo.UpdateProfile(ctx, UserProfile{
		UserID:          userID,
		DisplayName:     "  Ada  ",
		UnitSystem:      UnitSystemImperial,
		ThemePreference: "dark",
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if !result.RequiresUnitRecalculation || result.Profile.DisplayName != "Ada" || result.Profile.UnitSystem != UnitSystemImperial {
		t.Fatalf("preference update = %#v", result)
	}
	result, err = profileRepo.UpdateProfile(ctx, UserProfile{
		UserID:          userID,
		DisplayName:     "Ada",
		UnitSystem:      UnitSystemImperial,
		ThemePreference: "light",
	})
	if err != nil {
		t.Fatalf("UpdateProfile() same unit error = %v", err)
	}
	if result.RequiresUnitRecalculation {
		t.Fatalf("same unit update requested recalculation")
	}

	foodID, err := foodRepo.Create(ctx, FoodItemEntity{Name: "Saved Apple", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 1}})
	if err != nil {
		t.Fatalf("create food: %v", err)
	}
	mealID, err := mealRepo.Create(ctx, MealEntity{Type: MealTypeSingle, Name: "Saved Meal", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 1}})
	if err != nil {
		t.Fatalf("create meal: %v", err)
	}

	favoriteID, err := savedRepo.SaveItem(ctx, userID, foodID, SavedItemKindFavorite)
	if err != nil {
		t.Fatalf("SaveItem() favorite error = %v", err)
	}
	duplicateID, err := savedRepo.SaveItem(ctx, userID, foodID, SavedItemKindFavorite)
	if err != nil {
		t.Fatalf("SaveItem() duplicate error = %v", err)
	}
	if duplicateID != favoriteID {
		t.Fatalf("duplicate saved id = %s, want %s", duplicateID, favoriteID)
	}
	if _, err := savedRepo.SaveItem(ctx, userID, mealID, SavedItemKindSavedMeal); err != nil {
		t.Fatalf("SaveItem() saved meal error = %v", err)
	}
	if _, err := savedRepo.SaveItem(ctx, otherUserID, foodID, SavedItemKindFavorite); err != nil {
		t.Fatalf("SaveItem() other user error = %v", err)
	}

	favoriteKind := SavedItemKindFavorite
	items, err := savedRepo.ListItems(ctx, userID, &favoriteKind)
	if err != nil {
		t.Fatalf("ListItems() favorites error = %v", err)
	}
	if len(items) != 1 || items[0].UserID != userID || items[0].ItemID != foodID {
		t.Fatalf("favorite items = %#v", items)
	}
	allItems, err := savedRepo.ListItems(ctx, userID, nil)
	if err != nil {
		t.Fatalf("ListItems() all error = %v", err)
	}
	if len(allItems) != 2 {
		t.Fatalf("all saved items length = %d, want 2: %#v", len(allItems), allItems)
	}

	historyID, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{UserID: userID, Query: "  apple  ", Mode: "food", FiltersHash: "abc"})
	if err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}
	if historyID == uuid.Nil {
		t.Fatalf("AddHistory() id is nil")
	}
	if _, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{UserID: otherUserID, Query: "banana", Mode: "food"}); err != nil {
		t.Fatalf("AddHistory() other user error = %v", err)
	}
	history, err := savedRepo.ListHistory(ctx, userID, 0)
	if err != nil {
		t.Fatalf("ListHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].UserID != userID || history[0].Query != "apple" || history[0].FiltersHash != "abc" {
		t.Fatalf("history = %#v", history)
	}
	if err := savedRepo.ClearHistory(ctx, userID); err != nil {
		t.Fatalf("ClearHistory() error = %v", err)
	}
	history, err = savedRepo.ListHistory(ctx, userID, 10)
	if err != nil {
		t.Fatalf("ListHistory() after clear error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("history after clear = %#v", history)
	}

	if err := savedRepo.RemoveItem(ctx, userID, foodID, SavedItemKindFavorite); err != nil {
		t.Fatalf("RemoveItem() error = %v", err)
	}
	if err := savedRepo.RemoveItem(ctx, userID, foodID, SavedItemKindFavorite); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("RemoveItem() missing error = %v, want not found", err)
	}

	if _, err := db.Exec(ctx, testUserDeleteSQL, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	history, err = savedRepo.ListHistory(ctx, userID, 10)
	if err != nil {
		t.Fatalf("ListHistory() after user delete error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("history after cascade = %#v, want empty", history)
	}
	allItems, err = savedRepo.ListItems(ctx, userID, nil)
	if err != nil {
		t.Fatalf("ListItems() after user delete error = %v", err)
	}
	if len(allItems) != 0 {
		t.Fatalf("items after cascade = %#v, want empty", allItems)
	}
}

func TestPostgresUserDataRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	userID := uuid.New()
	itemID := uuid.New()
	now := time.Now()
	profileValues := []any{userID, "Ada", UnitSystemMetric, "system", now, now}
	savedValues := []any{uuid.New(), userID, itemID, SavedItemKindFavorite, now}
	historyValues := []any{uuid.New(), userID, "apple", "food", "hash", now}

	profileRepo := NewPostgresUserProfileRepository(&fakeSQLExecutor{})
	if _, err := profileRepo.GetOrCreate(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetOrCreate() nil user error = %v, want validation", err)
	}
	profileRepo = NewPostgresUserProfileRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := profileRepo.GetOrCreate(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetOrCreate() scan error = %v, want connection", err)
	}
	if _, err := NewPostgresUserProfileRepository(&fakeSQLExecutor{}).UpdateProfile(ctx, UserProfile{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdateProfile() nil user error = %v, want validation", err)
	}
	if _, err := NewPostgresUserProfileRepository(&fakeSQLExecutor{}).UpdateProfile(ctx, UserProfile{UserID: userID, UnitSystem: "bad", ThemePreference: "system"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdateProfile() bad unit error = %v, want validation", err)
	}
	if _, err := NewPostgresUserProfileRepository(&fakeSQLExecutor{}).UpdateProfile(ctx, UserProfile{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "bad"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdateProfile() bad theme error = %v, want validation", err)
	}
	profileRepo = NewPostgresUserProfileRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if _, err := profileRepo.UpdateProfile(ctx, UserProfile{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "system"}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("UpdateProfile() missing error = %v, want not found", err)
	}
	profileRepo = NewPostgresUserProfileRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := profileRepo.UpdateProfile(ctx, UserProfile{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "system"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdateProfile() load error = %v, want connection", err)
	}
	profileRepo = NewPostgresUserProfileRepository(&fakeSQLExecutor{rowList: []fakeRow{{values: []any{UnitSystemMetric}}, {values: profileValues}}})
	if _, err := profileRepo.UpdateProfile(ctx, UserProfile{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "system"}); err != nil {
		t.Fatalf("UpdateProfile() fake success error = %v", err)
	}
	profileRepo = NewPostgresUserProfileRepository(&fakeSQLExecutor{rowList: []fakeRow{{values: []any{UnitSystemMetric}}, {err: scanErr}}})
	if _, err := profileRepo.UpdateProfile(ctx, UserProfile{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "system"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdateProfile() update scan error = %v, want connection", err)
	}

	savedRepo := NewPostgresSavedDataRepository(&fakeSQLExecutor{})
	if _, err := savedRepo.SaveItem(ctx, uuid.Nil, itemID, SavedItemKindFavorite); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("SaveItem() nil user error = %v, want validation", err)
	}
	if _, err := savedRepo.SaveItem(ctx, userID, uuid.Nil, SavedItemKindFavorite); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("SaveItem() nil item error = %v, want validation", err)
	}
	if _, err := savedRepo.SaveItem(ctx, userID, itemID, "bad"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("SaveItem() bad kind error = %v, want validation", err)
	}
	if err := savedRepo.validateSavedItemTarget(ctx, itemID, "bad"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateSavedItemTarget() bad kind error = %v, want validation", err)
	}
	if _, err := savedRepo.SaveItem(ctx, userID, itemID, SavedItemKindSavedDiet); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("SaveItem() saved diet error = %v, want validation", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if _, err := savedRepo.SaveItem(ctx, userID, itemID, SavedItemKindFavorite); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("SaveItem() missing favorite target error = %v, want not found", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := savedRepo.SaveItem(ctx, userID, itemID, SavedItemKindSavedDiet); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("SaveItem() deferred saved diet error = %v, want validation", err)
	}
	if err := NewPostgresSavedDataRepository(&fakeSQLExecutor{execErr: queryErr}).RemoveItem(ctx, userID, itemID, SavedItemKindFavorite); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RemoveItem() exec error = %v, want connection", err)
	}
	if err := NewPostgresSavedDataRepository(&fakeSQLExecutor{}).RemoveItem(ctx, uuid.Nil, itemID, SavedItemKindFavorite); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RemoveItem() validation error = %v, want validation", err)
	}

	badKind := SavedItemKind("bad")
	if _, err := savedRepo.ListItems(ctx, uuid.Nil, nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListItems() nil user error = %v, want validation", err)
	}
	if _, err := savedRepo.ListItems(ctx, userID, &badKind); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListItems() bad kind error = %v, want validation", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := savedRepo.ListItems(ctx, userID, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListItems() query error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := savedRepo.ListItems(ctx, userID, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListItems() scan error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, err := savedRepo.ListItems(ctx, userID, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListItems() rows error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, values: savedValues}})
	if items, err := savedRepo.ListItems(ctx, userID, nil); err != nil || len(items) != 1 {
		t.Fatalf("ListItems() fake success items=%#v err=%v", items, err)
	}

	if _, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AddHistory() nil user error = %v, want validation", err)
	}
	if _, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{UserID: userID}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AddHistory() missing query error = %v, want validation", err)
	}
	if _, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{UserID: userID, Query: "x"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AddHistory() missing mode error = %v, want validation", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{UserID: userID, Query: "x", Mode: "food"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("AddHistory() insert error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{row: fakeRow{values: []any{uuid.New()}}})
	if _, err := savedRepo.AddHistory(ctx, SearchHistoryEntry{UserID: userID, Query: "x", Mode: "food"}); err != nil {
		t.Fatalf("AddHistory() fake success error = %v", err)
	}
	if _, err := savedRepo.ListHistory(ctx, uuid.Nil, 10); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListHistory() nil user error = %v, want validation", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := savedRepo.ListHistory(ctx, userID, 10); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListHistory() query error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := savedRepo.ListHistory(ctx, userID, 10); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListHistory() scan error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, err := savedRepo.ListHistory(ctx, userID, 10); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListHistory() rows error = %v, want connection", err)
	}
	savedRepo = NewPostgresSavedDataRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, values: historyValues}})
	if entries, err := savedRepo.ListHistory(ctx, userID, 10); err != nil || len(entries) != 1 {
		t.Fatalf("ListHistory() fake success entries=%#v err=%v", entries, err)
	}
}

func TestPostgresEncryptedIdentityRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	userID := uuid.New()
	now := time.Now()
	field := EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("nonce"), Ciphertext: []byte("ciphertext")}
	digest := LookupDigest{KeyVersion: "lookup-v1", Value: "digest"}
	userValues := []any{userID, field.KeyVersion, field.Nonce, field.Ciphertext, digest.KeyVersion, digest.Value, false, UserRoleUser, (*string)(nil), (*string)(nil), now, now}
	oauthValues := []any{uuid.New(), userID, "google", field.KeyVersion, field.Nonce, field.Ciphertext, digest.KeyVersion, digest.Value, field.KeyVersion, field.Nonce, field.Ciphertext, now}
	profileValues := []any{userID, &field.KeyVersion, field.Nonce, field.Ciphertext, UnitSystemMetric, "system", now, now}

	repo := NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{})
	if _, err := repo.CreateUser(ctx, EncryptedAuthUser{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateUser() invalid envelope error = %v, want validation", err)
	}
	hash := "hash"
	if _, err := repo.CreateUser(ctx, EncryptedAuthUser{Email: field, NormalizedEmailDigest: digest, PasswordHash: &hash}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateUser() password pair error = %v, want validation", err)
	}
	if _, err := repo.CreateUser(ctx, EncryptedAuthUser{Email: field, NormalizedEmailDigest: digest, Role: "owner"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateUser() bad role error = %v, want validation", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.CreateUser(ctx, EncryptedAuthUser{Email: field, NormalizedEmailDigest: digest}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CreateUser() scan error = %v, want connection", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{values: []any{userID}}})
	if _, err := repo.CreateUser(ctx, EncryptedAuthUser{Email: field, NormalizedEmailDigest: digest}); err != nil {
		t.Fatalf("CreateUser() fake success error = %v", err)
	}

	if _, err := repo.GetUserByNormalizedEmailDigest(ctx, LookupDigest{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetUserByNormalizedEmailDigest() invalid digest error = %v, want validation", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if _, err := repo.GetUserByNormalizedEmailDigest(ctx, digest); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetUserByNormalizedEmailDigest() missing error = %v, want not found", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{values: userValues}})
	if _, err := repo.GetUserByNormalizedEmailDigest(ctx, digest); err != nil {
		t.Fatalf("GetUserByNormalizedEmailDigest() fake success error = %v", err)
	}
	if _, err := repo.GetEncryptedUserByID(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetEncryptedUserByID() nil user error = %v, want validation", err)
	}
	if _, err := repo.GetEncryptedUserByID(ctx, userID); err != nil {
		t.Fatalf("GetEncryptedUserByID() fake success error = %v", err)
	}
	if err := repo.ReindexUserEmailDigest(ctx, uuid.Nil, digest); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ReindexUserEmailDigest() nil user error = %v, want validation", err)
	}
	if err := repo.ReindexUserEmailDigest(ctx, userID, LookupDigest{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ReindexUserEmailDigest() invalid digest error = %v, want validation", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if err := repo.ReindexUserEmailDigest(ctx, userID, digest); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("ReindexUserEmailDigest() missing error = %v, want not found", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{values: []any{userID}}})
	if err := repo.ReindexUserEmailDigest(ctx, userID, digest); err != nil {
		t.Fatalf("ReindexUserEmailDigest() fake success error = %v", err)
	}

	if _, err := repo.UpsertOAuthIdentity(ctx, EncryptedOAuthIdentity{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpsertOAuthIdentity() invalid error = %v, want validation", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.UpsertOAuthIdentity(ctx, EncryptedOAuthIdentity{UserID: userID, Provider: "google", ProviderUserID: field, ProviderUserIDDigest: digest, Email: field}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpsertOAuthIdentity() scan error = %v, want connection", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{values: []any{uuid.New()}}})
	if _, err := repo.UpsertOAuthIdentity(ctx, EncryptedOAuthIdentity{UserID: userID, Provider: "google", ProviderUserID: field, ProviderUserIDDigest: digest, Email: field}); err != nil {
		t.Fatalf("UpsertOAuthIdentity() fake success error = %v", err)
	}
	if _, err := repo.GetOAuthIdentity(ctx, "", digest); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetOAuthIdentity() provider error = %v, want validation", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{values: oauthValues}})
	if _, err := repo.GetOAuthIdentity(ctx, "google", digest); err != nil {
		t.Fatalf("GetOAuthIdentity() fake success error = %v", err)
	}

	if _, err := repo.UpdateEncryptedProfile(ctx, EncryptedUserProfile{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdateEncryptedProfile() nil user error = %v, want validation", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{values: profileValues}})
	if _, err := repo.UpdateEncryptedProfile(ctx, EncryptedUserProfile{UserID: userID, DisplayName: &field, UnitSystem: UnitSystemMetric, ThemePreference: "system"}); err != nil {
		t.Fatalf("UpdateEncryptedProfile() fake success error = %v", err)
	}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{row: fakeRow{err: queryErr}})
	if _, err := repo.UpdateEncryptedProfile(ctx, EncryptedUserProfile{UserID: userID, UnitSystem: UnitSystemMetric, ThemePreference: "system"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdateEncryptedProfile() scan error = %v, want connection", err)
	}

	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AddEncryptedHistory() invalid error = %v, want validation", err)
	}
	historyID := uuid.New()
	historyTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{historyID}}, execErrs: []error{nil}}}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{tx: historyTx})
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: field, Mode: "food"}); err != nil {
		t.Fatalf("AddEncryptedHistory() fake success error = %v", err)
	}
	if historyTx.execN != 1 {
		t.Fatalf("AddEncryptedHistory() prune calls = %d, want 1", historyTx.execN)
	}

	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{beginErr: queryErr})
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: field, Mode: "food"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("AddEncryptedHistory() begin error = %v, want connection", err)
	}
	insertTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{err: queryErr}}}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{tx: insertTx})
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: field, Mode: "food"}); !IsKind(err, ErrorKindConnection) || !insertTx.rolledBack {
		t.Fatalf("AddEncryptedHistory() insert error = %v rolledBack=%t, want connection and rollback", err, insertTx.rolledBack)
	}
	pruneTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{historyID}}, execErr: queryErr}}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{tx: pruneTx})
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: field, Mode: "food"}); !IsKind(err, ErrorKindConnection) || !pruneTx.rolledBack {
		t.Fatalf("AddEncryptedHistory() prune error = %v rolledBack=%t, want connection and rollback", err, pruneTx.rolledBack)
	}
	commitTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{historyID}}}, commitErr: queryErr}
	repo = NewPostgresEncryptedIdentityRepository(&fakeSQLExecutor{tx: commitTx})
	if _, err := repo.AddEncryptedHistory(ctx, EncryptedSearchHistoryEntry{UserID: userID, Query: field, Mode: "food"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("AddEncryptedHistory() commit error = %v, want connection", err)
	}
}

func TestPostgresAccountLockoutRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	scanErr := errors.New("scan failed")
	userID := uuid.New()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	lockedUntil := now.Add(15 * time.Minute)
	values := []any{3, (*time.Time)(nil)}

	repo := NewPostgresAccountLockoutRepository(&fakeSQLExecutor{})
	if _, err := repo.GetLockoutState(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetLockoutState() nil user error = %v, want validation", err)
	}
	repo = NewPostgresAccountLockoutRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if _, err := repo.GetLockoutState(ctx, userID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetLockoutState() missing error = %v, want not found", err)
	}
	repo = NewPostgresAccountLockoutRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.GetLockoutState(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetLockoutState() scan error = %v, want connection", err)
	}
	repo = NewPostgresAccountLockoutRepository(&fakeSQLExecutor{row: fakeRow{values: values}})
	if _, err := repo.GetLockoutState(ctx, userID); err != nil {
		t.Fatalf("GetLockoutState() fake success error = %v", err)
	}

	if _, err := repo.RecordFailedLogin(ctx, uuid.Nil, 5, lockedUntil, now); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordFailedLogin() nil user error = %v, want validation", err)
	}
	if _, err := repo.RecordFailedLogin(ctx, userID, 0, lockedUntil, now); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordFailedLogin() bad threshold error = %v, want validation", err)
	}
	if _, err := repo.RecordFailedLogin(ctx, userID, 5, now, lockedUntil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordFailedLogin() bad time error = %v, want validation", err)
	}
	repo = NewPostgresAccountLockoutRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.RecordFailedLogin(ctx, userID, 5, lockedUntil, now); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordFailedLogin() scan error = %v, want connection", err)
	}

	if _, err := repo.ResetFailedLogins(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ResetFailedLogins() nil user error = %v, want validation", err)
	}
	repo = NewPostgresAccountLockoutRepository(&fakeSQLExecutor{row: fakeRow{values: []any{0, (*time.Time)(nil)}}})
	if _, err := repo.ResetFailedLogins(ctx, userID); err != nil {
		t.Fatalf("ResetFailedLogins() fake success error = %v", err)
	}
}

func TestPostgresSessionRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	scanErr := errors.New("scan failed")
	userID := uuid.New()
	sessionID := uuid.New()
	familyID := uuid.New()
	now := time.Now()
	values := []any{sessionID, userID, "hash", familyID, now.Add(time.Minute), now.Add(time.Hour), (*time.Time)(nil), now}
	repo := NewPostgresSessionRepository(&fakeSQLExecutor{})

	if _, err := repo.CreateSession(ctx, UserSession{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateSession() invalid identity error = %v, want validation", err)
	}
	if _, err := repo.CreateSession(ctx, UserSession{UserID: userID, RefreshTokenHash: "hash", RefreshFamilyID: familyID, AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(time.Minute)}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CreateSession() invalid expiry error = %v, want validation", err)
	}
	repo = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.CreateSession(ctx, UserSession{UserID: userID, RefreshTokenHash: "hash", RefreshFamilyID: familyID, AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour)}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CreateSession() scan error = %v, want connection", err)
	}
	repo = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{values: []any{sessionID}}})
	if _, err := repo.CreateSession(ctx, UserSession{UserID: userID, RefreshTokenHash: "hash", RefreshFamilyID: familyID, AccessExpiresAt: now.Add(time.Minute), RefreshExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("CreateSession() fake success error = %v", err)
	}

	if _, err := repo.GetSessionByRefreshTokenHash(ctx, ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetSessionByRefreshTokenHash() blank error = %v, want validation", err)
	}
	repo = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if _, err := repo.GetSessionByRefreshTokenHash(ctx, "hash"); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetSessionByRefreshTokenHash() missing error = %v, want not found", err)
	}
	repo = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{values: values}})
	if _, err := repo.GetSessionByRefreshTokenHash(ctx, "hash"); err != nil {
		t.Fatalf("GetSessionByRefreshTokenHash() fake success error = %v", err)
	}

	if err := repo.RevokeSession(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RevokeSession() nil error = %v, want validation", err)
	}
	repo = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{err: pgx.ErrNoRows}})
	if err := repo.RevokeSession(ctx, sessionID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("RevokeSession() missing error = %v, want not found", err)
	}
	repo = NewPostgresSessionRepository(&fakeSQLExecutor{row: fakeRow{values: []any{sessionID}}})
	if err := repo.RevokeSession(ctx, sessionID); err != nil {
		t.Fatalf("RevokeSession() fake success error = %v", err)
	}
	if err := repo.RevokeSessionFamily(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RevokeSessionFamily() nil error = %v, want validation", err)
	}
	if err := repo.RevokeSessionFamily(ctx, familyID); err != nil {
		t.Fatalf("RevokeSessionFamily() fake success error = %v", err)
	}
}

func TestPostgresEntitlementRepository(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresEntitlementRepository(db)
	userID := createRepositoryUser(t, ctx, db, "entitled@example.test")
	otherUserID := createRepositoryUser(t, ctx, db, "trial@example.test")
	expiredAt := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)

	if err := repo.AppendEntitlement(ctx, Entitlement{
		UserID:            userID,
		Tier:              "free",
		Status:            "active",
		SearchLimitPer24h: 3,
		AllowedModes:      []string{"catalog"},
	}); err != nil {
		t.Fatalf("AppendEntitlement() free error = %v", err)
	}
	if err := repo.AppendEntitlement(ctx, Entitlement{
		UserID:               userID,
		Tier:                 "paid",
		Status:               "active",
		SearchLimitPer24h:    0,
		AllowedModes:         []string{"catalog", "substitution"},
		StripeCustomerID:     "cus_fixture",
		StripeSubscriptionID: "sub_fixture",
	}); err != nil {
		t.Fatalf("AppendEntitlement() paid error = %v", err)
	}
	if err := repo.AppendEntitlement(ctx, Entitlement{
		UserID:            otherUserID,
		Tier:              "trial",
		Status:            "active",
		SearchLimitPer24h: 0,
		AllowedModes:      []string{"catalog", "substitution"},
		ExpiresAt:         &expiredAt,
	}); err != nil {
		t.Fatalf("AppendEntitlement() trial error = %v", err)
	}

	latest, err := repo.GetLatest(ctx, userID)
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if latest.Tier != "paid" || latest.Status != "active" || latest.StripeCustomerID != "cus_fixture" || len(latest.AllowedModes) != 2 {
		t.Fatalf("latest entitlement = %#v", latest)
	}

	var count int
	if err := db.QueryRow(ctx, testEntitlementCountByUserSQL, userID).Scan(&count); err != nil {
		t.Fatalf("count entitlements: %v", err)
	}
	if count != 2 {
		t.Fatalf("entitlement history count = %d, want 2", count)
	}

	windowStart := time.Now().UTC().Truncate(24 * time.Hour)
	window, err := repo.RecordUsage(ctx, userID, " search ", windowStart)
	if err != nil {
		t.Fatalf("RecordUsage() first error = %v", err)
	}
	if window.Feature != "search" || window.SearchCount != 1 {
		t.Fatalf("first usage window = %#v", window)
	}
	window, err = repo.RecordUsage(ctx, userID, "search", windowStart)
	if err != nil {
		t.Fatalf("RecordUsage() second error = %v", err)
	}
	if window.SearchCount != 2 {
		t.Fatalf("second usage window count = %d, want 2", window.SearchCount)
	}
	if _, err := repo.RecordUsage(ctx, userID, "search", windowStart.Add(-25*time.Hour)); err != nil {
		t.Fatalf("RecordUsage() old occurrence error = %v", err)
	}
	usage, err := repo.GetUsageSince(ctx, userID, "search", windowStart.Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetUsageSince() error = %v", err)
	}
	if usage.SearchCount != 2 {
		t.Fatalf("GetUsageSince() count = %d, want 2", usage.SearchCount)
	}
	limited, recorded, err := repo.RecordUsageWithinLimit(ctx, userID, "search", windowStart.Add(time.Minute), windowStart.Add(-time.Hour), 3)
	if err != nil {
		t.Fatalf("RecordUsageWithinLimit() third error = %v", err)
	}
	if !recorded || limited.SearchCount != 3 {
		t.Fatalf("RecordUsageWithinLimit() third window=%#v recorded=%v, want count 3 recorded", limited, recorded)
	}
	limited, recorded, err = repo.RecordUsageWithinLimit(ctx, userID, "search", windowStart.Add(2*time.Minute), windowStart.Add(-time.Hour), 3)
	if err != nil {
		t.Fatalf("RecordUsageWithinLimit() capped error = %v", err)
	}
	if recorded || limited.SearchCount != 3 {
		t.Fatalf("RecordUsageWithinLimit() capped window=%#v recorded=%v, want count 3 not recorded", limited, recorded)
	}

	expired, err := repo.ListExpiredTrials(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("ListExpiredTrials() error = %v", err)
	}
	if len(expired) != 1 || expired[0].UserID != otherUserID {
		t.Fatalf("expired trials = %#v", expired)
	}
	if err := repo.AppendEntitlement(ctx, Entitlement{
		UserID:            otherUserID,
		Tier:              "paid",
		Status:            "active",
		SearchLimitPer24h: 0,
		AllowedModes:      []string{"catalog", "substitution", "daily_diet", "daily_diet_alternative"},
	}); err != nil {
		t.Fatalf("AppendEntitlement() paid after trial error = %v", err)
	}
	expired, err = repo.ListExpiredTrials(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("ListExpiredTrials() after paid error = %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expired trials after newer paid entitlement = %#v, want none", expired)
	}

	inserted, err := repo.InsertProcessedStripeEvent(ctx, ProcessedStripeEvent{
		EventID:   "evt_fixture",
		EventType: "checkout.session.completed",
		Outcome:   "success",
		Payload:   []byte(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("InsertProcessedStripeEvent() error = %v", err)
	}
	if !inserted {
		t.Fatalf("InsertProcessedStripeEvent() inserted=false, want true")
	}
	inserted, err = repo.InsertProcessedStripeEvent(ctx, ProcessedStripeEvent{
		EventID:   "evt_fixture",
		EventType: "checkout.session.completed",
		Outcome:   "success",
		Payload:   []byte(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("InsertProcessedStripeEvent() duplicate error = %v", err)
	}
	if inserted {
		t.Fatalf("InsertProcessedStripeEvent() duplicate inserted=true, want false")
	}

	rawStripePayload := `{"latest_invoice":{"card":{"last4":"4242"}},"customer_email":"payer@example.test"}`
	payloadHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := repo.InsertStripeDeadLetter(ctx, StripeDeadLetter{
		EventID:              "evt_dead_letter",
		EventType:            "invoice.payment_failed",
		FailureCategory:      "webhook_processing_failed",
		ErrorMessage:         "database write failed",
		PayloadSHA256:        payloadHash,
		StripeCustomerID:     "cus_dead_letter",
		StripeSubscriptionID: "sub_dead_letter",
		UserID:               &userID,
	}); err != nil {
		t.Fatalf("InsertStripeDeadLetter() error = %v", err)
	}
	var eventID, eventType, failureCategory, errorMessage, storedPayloadHash, customerID, subscriptionID string
	var storedUserID uuid.UUID
	if err := db.QueryRow(ctx, testStripeDeadLetterGetSQL, "evt_dead_letter").Scan(&eventID, &eventType, &failureCategory, &errorMessage, &storedPayloadHash, &customerID, &subscriptionID, &storedUserID); err != nil {
		t.Fatalf("get stripe dead letter: %v", err)
	}
	if eventID != "evt_dead_letter" || eventType != "invoice.payment_failed" || failureCategory != "webhook_processing_failed" || storedPayloadHash != payloadHash {
		t.Fatalf("dead letter metadata = %q %q %q %q", eventID, eventType, failureCategory, storedPayloadHash)
	}
	if customerID != "cus_dead_letter" || subscriptionID != "sub_dead_letter" || storedUserID != userID {
		t.Fatalf("dead letter provider ids = %q %q %s", customerID, subscriptionID, storedUserID)
	}
	persistedDeadLetter := strings.Join([]string{eventID, eventType, failureCategory, errorMessage, storedPayloadHash, customerID, subscriptionID}, " ")
	if strings.Contains(persistedDeadLetter, "4242") || strings.Contains(persistedDeadLetter, "payer@example.test") || strings.Contains(persistedDeadLetter, rawStripePayload) {
		t.Fatalf("dead letter persisted raw payment data: %q", persistedDeadLetter)
	}
}

func TestPostgresStripeWebhookDuplicateTransactionDoesNotAppend(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresEntitlementRepository(db)
	userID := createRepositoryUser(t, ctx, db, "stripe-duplicate@example.test")
	firstEntitlement := Entitlement{
		UserID:               userID,
		Tier:                 "paid",
		Status:               "active",
		SearchLimitPer24h:    0,
		AllowedModes:         []string{"catalog", "substitution", "daily_diet_alternative"},
		StripeCustomerID:     "cus_first",
		StripeSubscriptionID: "sub_first",
	}

	inserted, err := repo.ProcessStripeWebhookEvent(ctx, ProcessedStripeEvent{
		EventID:   "evt_duplicate_tx",
		EventType: "checkout.session.completed",
		Outcome:   "success",
		Payload:   []byte(`{"id":"evt_duplicate_tx"}`),
	}, &firstEntitlement)
	if err != nil || !inserted {
		t.Fatalf("ProcessStripeWebhookEvent() first inserted=%v err=%v, want insert", inserted, err)
	}
	var beforeCount int
	if err := db.QueryRow(ctx, testEntitlementCountByUserSQL, userID).Scan(&beforeCount); err != nil {
		t.Fatalf("count entitlements before duplicate: %v", err)
	}

	duplicateEntitlement := firstEntitlement
	duplicateEntitlement.Status = "cancelled"
	duplicateEntitlement.StripeCustomerID = "cus_duplicate"
	duplicateEntitlement.StripeSubscriptionID = "sub_duplicate"
	inserted, err = repo.ProcessStripeWebhookEvent(ctx, ProcessedStripeEvent{
		EventID:   "evt_duplicate_tx",
		EventType: "checkout.session.completed",
		Outcome:   "success",
		Payload:   []byte(`{"id":"evt_duplicate_tx"}`),
	}, &duplicateEntitlement)
	if err != nil || inserted {
		t.Fatalf("ProcessStripeWebhookEvent() duplicate inserted=%v err=%v, want duplicate success", inserted, err)
	}

	var afterCount int
	if err := db.QueryRow(ctx, testEntitlementCountByUserSQL, userID).Scan(&afterCount); err != nil {
		t.Fatalf("count entitlements after duplicate: %v", err)
	}
	if afterCount != beforeCount {
		t.Fatalf("duplicate webhook entitlement count = %d, want %d", afterCount, beforeCount)
	}
	latest, err := repo.GetLatest(ctx, userID)
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if latest.Status != "active" || latest.StripeCustomerID != "cus_first" || latest.StripeSubscriptionID != "sub_first" {
		t.Fatalf("duplicate webhook changed latest entitlement: %#v", latest)
	}
}

func TestPostgresEntitlementRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	userID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(time.Hour)
	entitlementValues := []any{userID, "trial", "active", 0, []string{"catalog"}, &expiresAt, "", "", now, now}
	windowValues := []any{userID, "search", now, 1, now, now}
	validEntitlement := Entitlement{UserID: userID, Tier: "free", Status: "active", SearchLimitPer24h: 3, AllowedModes: []string{"catalog"}}
	repo := NewPostgresEntitlementRepository(&fakeSQLExecutor{})

	invalidEntitlements := []Entitlement{
		{},
		{UserID: userID, Tier: "bad", Status: "active", AllowedModes: []string{"catalog"}},
		{UserID: userID, Tier: "free", Status: "bad", AllowedModes: []string{"catalog"}},
		{UserID: userID, Tier: "free", Status: "active", SearchLimitPer24h: -1, AllowedModes: []string{"catalog"}},
		{UserID: userID, Tier: "free", Status: "active"},
		{UserID: userID, Tier: "free", Status: "active", AllowedModes: []string{" "}},
		{UserID: userID, Tier: "free", Status: "active", AllowedModes: []string{"catalog", "catalog"}},
		{UserID: userID, Tier: "trial", Status: "active", AllowedModes: []string{"catalog"}},
	}
	for _, entitlement := range invalidEntitlements {
		if err := repo.AppendEntitlement(ctx, entitlement); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("AppendEntitlement(%#v) error = %v, want validation", entitlement, err)
		}
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{execErr: queryErr})
	if err := repo.AppendEntitlement(ctx, validEntitlement); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("AppendEntitlement() exec error = %v, want connection", err)
	}

	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{})
	if _, err := repo.GetLatest(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetLatest() nil user error = %v, want validation", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.GetLatest(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetLatest() scan error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{row: fakeRow{values: entitlementValues}})
	if entitlement, err := repo.GetLatest(ctx, userID); err != nil || entitlement.UserID != userID {
		t.Fatalf("GetLatest() fake success entitlement=%#v err=%v", entitlement, err)
	}

	if _, err := repo.RecordUsage(ctx, uuid.Nil, "search", now); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsage() nil user error = %v, want validation", err)
	}
	if _, err := repo.RecordUsage(ctx, userID, " ", now); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsage() blank feature error = %v, want validation", err)
	}
	if _, err := repo.RecordUsage(ctx, userID, "search", time.Time{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsage() zero start error = %v, want validation", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.RecordUsage(ctx, userID, "search", now); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordUsage() scan error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{row: fakeRow{values: windowValues}})
	if window, err := repo.RecordUsage(ctx, userID, "search", now); err != nil || window.SearchCount != 1 {
		t.Fatalf("RecordUsage() fake success window=%#v err=%v", window, err)
	}
	if _, _, err := repo.RecordUsageWithinLimit(ctx, uuid.Nil, "search", now, now.Add(-time.Hour), 3); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsageWithinLimit() nil user error = %v, want validation", err)
	}
	if _, _, err := repo.RecordUsageWithinLimit(ctx, userID, " ", now, now.Add(-time.Hour), 3); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsageWithinLimit() blank feature error = %v, want validation", err)
	}
	if _, _, err := repo.RecordUsageWithinLimit(ctx, userID, "search", time.Time{}, now.Add(-time.Hour), 3); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsageWithinLimit() zero occurrence error = %v, want validation", err)
	}
	if _, _, err := repo.RecordUsageWithinLimit(ctx, userID, "search", now, time.Time{}, 3); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsageWithinLimit() zero since error = %v, want validation", err)
	}
	if _, _, err := repo.RecordUsageWithinLimit(ctx, userID, "search", now, now.Add(-time.Hour), 0); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordUsageWithinLimit() non-positive limit error = %v, want validation", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErr: queryErr}}})
	if _, _, err := repo.RecordUsageWithinLimit(ctx, userID, "search", now, now.Add(-time.Hour), 3); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordUsageWithinLimit() lock error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{err: scanErr}}}})
	if _, _, err := repo.RecordUsageWithinLimit(ctx, userID, "search", now, now.Add(-time.Hour), 3); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordUsageWithinLimit() current usage scan error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rowList: []fakeRow{
		{values: []any{userID, "search", now.Add(-time.Hour), 0, now, now}},
		{values: windowValues},
	}}}})
	if window, recorded, err := repo.RecordUsageWithinLimit(ctx, userID, "search", now, now.Add(-time.Hour), 3); err != nil || !recorded || window.SearchCount != 1 {
		t.Fatalf("RecordUsageWithinLimit() fake success window=%#v recorded=%v err=%v", window, recorded, err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{userID, "search", now.Add(-time.Hour), 3, now, now}}}}})
	if window, recorded, err := repo.RecordUsageWithinLimit(ctx, userID, "search", now, now.Add(-time.Hour), 3); err != nil || recorded || window.SearchCount != 3 {
		t.Fatalf("RecordUsageWithinLimit() fake capped window=%#v recorded=%v err=%v", window, recorded, err)
	}
	if _, err := repo.GetUsageSince(ctx, uuid.Nil, "search", now); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetUsageSince() nil user error = %v, want validation", err)
	}
	if _, err := repo.GetUsageSince(ctx, userID, " ", now); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetUsageSince() blank feature error = %v, want validation", err)
	}
	if _, err := repo.GetUsageSince(ctx, userID, "search", time.Time{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetUsageSince() zero since error = %v, want validation", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.GetUsageSince(ctx, userID, "search", now); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetUsageSince() scan error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{row: fakeRow{values: windowValues}})
	if window, err := repo.GetUsageSince(ctx, userID, "search", now); err != nil || window.SearchCount != 1 {
		t.Fatalf("GetUsageSince() fake success window=%#v err=%v", window, err)
	}

	if _, err := repo.ListExpiredTrials(ctx, time.Time{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListExpiredTrials() zero now error = %v, want validation", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := repo.ListExpiredTrials(ctx, now); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListExpiredTrials() query error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := repo.ListExpiredTrials(ctx, now); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListExpiredTrials() scan error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, err := repo.ListExpiredTrials(ctx, now); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListExpiredTrials() rows error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, values: entitlementValues}})
	if entitlements, err := repo.ListExpiredTrials(ctx, now); err != nil || len(entitlements) != 1 {
		t.Fatalf("ListExpiredTrials() fake success entitlements=%#v err=%v", entitlements, err)
	}

	invalidEvents := []ProcessedStripeEvent{
		{},
		{EventID: "evt", Outcome: "success"},
		{EventID: "evt", EventType: "checkout", Outcome: "bad"},
		{EventID: "evt", EventType: "checkout", Outcome: "success", Payload: []byte(`{`)},
	}
	for _, event := range invalidEvents {
		if _, err := repo.InsertProcessedStripeEvent(ctx, event); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("InsertProcessedStripeEvent(%#v) error = %v, want validation", event, err)
		}
	}
	duplicateErr := &pgconn.PgError{Code: "23505"}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{execErr: duplicateErr})
	if inserted, err := repo.InsertProcessedStripeEvent(ctx, ProcessedStripeEvent{EventID: "evt", EventType: "checkout", Outcome: "success"}); err != nil || inserted {
		t.Fatalf("InsertProcessedStripeEvent() duplicate inserted=%v err=%v", inserted, err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{execErr: queryErr})
	if _, err := repo.InsertProcessedStripeEvent(ctx, ProcessedStripeEvent{EventID: "evt", EventType: "checkout", Outcome: "success"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("InsertProcessedStripeEvent() exec error = %v, want connection", err)
	}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 1")}})
	if inserted, err := repo.InsertProcessedStripeEvent(ctx, ProcessedStripeEvent{EventID: "evt", EventType: "checkout", Outcome: "success"}); err != nil || !inserted {
		t.Fatalf("InsertProcessedStripeEvent() fake success inserted=%v err=%v", inserted, err)
	}

	tx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErrs: []error{nil, nil}, execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 1"), pgconn.NewCommandTag("INSERT 1")}}}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: tx})
	inserted, err := repo.ProcessStripeWebhookEvent(ctx, ProcessedStripeEvent{EventID: "evt_tx", EventType: "checkout", Outcome: "success"}, &validEntitlement)
	if err != nil || !inserted || tx.execN != 2 {
		t.Fatalf("ProcessStripeWebhookEvent() inserted=%v execN=%d err=%v, want event and entitlement writes", inserted, tx.execN, err)
	}

	tx = &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErrs: []error{nil}, execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0")}}}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: tx})
	inserted, err = repo.ProcessStripeWebhookEvent(ctx, ProcessedStripeEvent{EventID: "evt_tx", EventType: "checkout", Outcome: "success"}, &validEntitlement)
	if err != nil || inserted || tx.execN != 1 {
		t.Fatalf("ProcessStripeWebhookEvent() duplicate inserted=%v execN=%d err=%v, want no entitlement write", inserted, tx.execN, err)
	}

	tx = &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErrs: []error{nil, queryErr}, execTags: []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 1")}}}
	repo = NewPostgresEntitlementRepository(&fakeSQLExecutor{tx: tx})
	if _, err := repo.ProcessStripeWebhookEvent(ctx, ProcessedStripeEvent{EventID: "evt_tx", EventType: "checkout", Outcome: "success"}, &validEntitlement); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ProcessStripeWebhookEvent() entitlement write error = %v, want connection", err)
	}
	if !tx.rolledBack {
		t.Fatal("ProcessStripeWebhookEvent() write failure did not roll back transaction")
	}
}

func TestPostgresComplianceAndAdminRepositories(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	complianceRepo := NewPostgresComplianceRepository(db)
	adminRepo := NewPostgresAdminImportAuditRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)
	userID := createRepositoryUser(t, ctx, db, "compliance@example.test")
	adminID := createRepositoryUser(t, ctx, db, "admin@example.test")

	consentID, err := complianceRepo.RecordConsent(ctx, ConsentRecord{
		UserID:               userID,
		PrivacyPolicyVersion: "privacy-v1",
		TermsVersion:         "terms-v1",
	})
	if err != nil {
		t.Fatalf("RecordConsent() error = %v", err)
	}
	duplicateConsentID, err := complianceRepo.RecordConsent(ctx, ConsentRecord{
		UserID:               userID,
		PrivacyPolicyVersion: "privacy-v1",
		TermsVersion:         "terms-v1",
	})
	if err != nil {
		t.Fatalf("RecordConsent() duplicate error = %v", err)
	}
	if duplicateConsentID != consentID {
		t.Fatalf("duplicate consent id = %s, want %s", duplicateConsentID, consentID)
	}
	hasConsent, err := complianceRepo.HasRequiredConsent(ctx, userID, "privacy-v1", "terms-v1")
	if err != nil {
		t.Fatalf("HasRequiredConsent() error = %v", err)
	}
	if !hasConsent {
		t.Fatalf("HasRequiredConsent() = false, want true")
	}
	consentRecords, err := complianceRepo.ListConsent(ctx, userID)
	if err != nil {
		t.Fatalf("ListConsent() error = %v", err)
	}
	if len(consentRecords) != 1 || consentRecords[0].ID != consentID || consentRecords[0].PrivacyPolicyVersion != "privacy-v1" {
		t.Fatalf("consent records = %#v", consentRecords)
	}

	deletion, err := complianceRepo.RequestDeletion(ctx, userID)
	if err != nil {
		t.Fatalf("RequestDeletion() error = %v", err)
	}
	if deletion.UserID != userID || deletion.Status != "pending" {
		t.Fatalf("deletion request = %#v", deletion)
	}
	sameDeletion, err := complianceRepo.RequestDeletion(ctx, userID)
	if err != nil {
		t.Fatalf("RequestDeletion() duplicate error = %v", err)
	}
	if sameDeletion.ID != deletion.ID {
		t.Fatalf("duplicate deletion id = %s, want %s", sameDeletion.ID, deletion.ID)
	}
	if err := complianceRepo.UpdateDeletionStatus(ctx, deletion.ID, "processing", "worker started"); err != nil {
		t.Fatalf("UpdateDeletionStatus() processing error = %v", err)
	}
	if err := complianceRepo.UpdateDeletionStatus(ctx, deletion.ID, "completed", "done"); err != nil {
		t.Fatalf("UpdateDeletionStatus() completed error = %v", err)
	}
	deletionAudit, err := complianceRepo.ListDeletionAudit(ctx, deletion.ID)
	if err != nil {
		t.Fatalf("ListDeletionAudit() error = %v", err)
	}
	if len(deletionAudit) != 3 || deletionAudit[0].ToStatus != "pending" || deletionAudit[2].ToStatus != "completed" {
		t.Fatalf("deletion audit = %#v", deletionAudit)
	}

	foodID, err := foodRepo.Create(ctx, FoodItemEntity{Name: "Imported Pear", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 1}})
	if err != nil {
		t.Fatalf("create food: %v", err)
	}
	importID, err := adminRepo.UpsertCuratedImport(ctx, CuratedImport{
		SourceProvider: "usda",
		ExternalID:     "fdc-1",
		FoodItemID:     &foodID,
		Status:         "imported",
		RawPayload:     []byte(`{"name":"Imported Pear"}`),
	})
	if err != nil {
		t.Fatalf("UpsertCuratedImport() error = %v", err)
	}
	conflictID, err := adminRepo.UpsertCuratedImport(ctx, CuratedImport{
		SourceProvider: "usda",
		ExternalID:     "fdc-1",
		FoodItemID:     &foodID,
		Status:         "conflict",
		ConflictReason: "duplicate normalized name",
		RawPayload:     []byte(`{"name":"Imported Pear"}`),
	})
	if err != nil {
		t.Fatalf("UpsertCuratedImport() conflict error = %v", err)
	}
	if conflictID != importID {
		t.Fatalf("curated import upsert id = %s, want %s", conflictID, importID)
	}
	imported, err := adminRepo.FindCuratedImport(ctx, "usda", "fdc-1")
	if err != nil {
		t.Fatalf("FindCuratedImport() error = %v", err)
	}
	if imported.Status != "conflict" || imported.ConflictReason == "" {
		t.Fatalf("curated import = %#v", imported)
	}

	auditID, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{
		AdminUserID: adminID,
		Action:      "update_food",
		EntityType:  "food_item",
		EntityID:    &foodID,
		Before:      []byte(`{"name":"Old"}`),
		After:       []byte(`{"name":"Imported Pear"}`),
		RequestID:   "req-1",
	})
	if err != nil {
		t.Fatalf("PersistAuditEntry() error = %v", err)
	}
	if auditID == uuid.Nil {
		t.Fatalf("PersistAuditEntry() id is nil")
	}
	auditEntries, err := adminRepo.ListAuditForEntity(ctx, "food_item", foodID)
	if err != nil {
		t.Fatalf("ListAuditForEntity() error = %v", err)
	}
	if len(auditEntries) != 1 || auditEntries[0].RequestID != "req-1" {
		t.Fatalf("audit entries = %#v", auditEntries)
	}

	rollbackName := "Rollback Pear"
	err = adminRepo.WithAudit(ctx, AdminAuditEntry{AdminUserID: uuid.New(), Action: "", EntityType: "food_item"}, func(tx sqlExecutor) error {
		_, insertErr := tx.Exec(ctx, testFoodNameFixtureCreateSQL, rollbackName)
		return insertErr
	})
	if !IsKind(err, ErrorKindValidation) {
		t.Fatalf("WithAudit() invalid audit error = %v, want validation", err)
	}
	var exists bool
	if err := db.QueryRow(ctx, testFoodExistsByNameSQL, rollbackName).Scan(&exists); err != nil {
		t.Fatalf("check rollback: %v", err)
	}
	if exists {
		t.Fatalf("mutation committed despite audit failure")
	}
}

func TestPostgresComplianceRepositoryDeletionTransitions(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresComplianceRepository(db)
	userID := createRepositoryUser(t, ctx, db, "deletion-transitions@example.test")

	request, err := repo.RequestDeletion(ctx, userID)
	if err != nil {
		t.Fatalf("RequestDeletion() error = %v", err)
	}
	assertTransition := func(from string, to string, wantKind ErrorKind) {
		t.Helper()
		err := repo.UpdateDeletionStatus(ctx, request.ID, to, from+" to "+to)
		if wantKind == "" && err != nil {
			t.Fatalf("UpdateDeletionStatus(%s -> %s) error = %v", from, to, err)
		}
		if wantKind != "" && !IsKind(err, wantKind) {
			t.Fatalf("UpdateDeletionStatus(%s -> %s) error = %v, want %s", from, to, err, wantKind)
		}
	}

	assertTransition("pending", "pending", ErrorKindConflict)
	assertTransition("pending", "completed", ErrorKindConflict)
	assertTransition("pending", "failed", ErrorKindConflict)
	assertTransition("pending", "processing", "")
	assertTransition("processing", "processing", ErrorKindConflict)
	assertTransition("processing", "pending", ErrorKindConflict)
	assertTransition("processing", "failed", "")
	assertTransition("failed", "failed", ErrorKindConflict)
	assertTransition("failed", "completed", ErrorKindConflict)
	assertTransition("failed", "pending", ErrorKindConflict)
	assertTransition("failed", "processing", "")
	assertTransition("processing", "completed", "")
	assertTransition("completed", "pending", ErrorKindConflict)
	assertTransition("completed", "processing", ErrorKindConflict)
	assertTransition("completed", "completed", ErrorKindConflict)
	assertTransition("completed", "failed", ErrorKindConflict)

	audit, err := repo.ListDeletionAudit(ctx, request.ID)
	if err != nil {
		t.Fatalf("ListDeletionAudit() error = %v", err)
	}
	if len(audit) != 5 {
		t.Fatalf("ListDeletionAudit() length = %d, want initial request plus four legal transitions: %#v", len(audit), audit)
	}
}

func TestPostgresComplianceRepositoryDeletionHardening(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresComplianceRepository(db)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	firstUserID := createRepositoryUser(t, ctx, db, "deletion-claim-1@example.test")
	secondUserID := createRepositoryUser(t, ctx, db, "deletion-claim-2@example.test")
	first, err := repo.RequestDeletion(ctx, firstUserID)
	if err != nil {
		t.Fatalf("RequestDeletion() first error = %v", err)
	}
	second, err := repo.RequestDeletion(ctx, secondUserID)
	if err != nil {
		t.Fatalf("RequestDeletion() second error = %v", err)
	}
	claimed, err := repo.ClaimDeletionRequests(ctx, now, 2)
	if err != nil {
		t.Fatalf("ClaimDeletionRequests() error = %v", err)
	}
	if len(claimed) != 2 || claimed[0].Status != "processing" || claimed[1].Status != "processing" {
		t.Fatalf("claimed = %#v", claimed)
	}
	nextAttempt := now.Add(time.Hour)
	if err := repo.RecordDeletionFailure(ctx, first.ID, "transient", "database temporarily unavailable", &nextAttempt); err != nil {
		t.Fatalf("RecordDeletionFailure() transient error = %v", err)
	}
	claimed, err = repo.ClaimDeletionRequests(ctx, now, 2)
	if err != nil {
		t.Fatalf("ClaimDeletionRequests() before retry error = %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("claimed before retry = %#v", claimed)
	}
	claimed, err = repo.ClaimDeletionRequests(ctx, nextAttempt.Add(time.Second), 2)
	if err != nil {
		t.Fatalf("ClaimDeletionRequests() retry error = %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != first.ID || claimed[0].RetryCount != 1 {
		t.Fatalf("retry claim = %#v", claimed)
	}
	if err := repo.RecordDeletionFailure(ctx, first.ID, "permanent", "provider policy", nil); err != nil {
		t.Fatalf("RecordDeletionFailure() permanent error = %v", err)
	}
	claimed, err = repo.ClaimDeletionRequests(ctx, nextAttempt.Add(2*time.Hour), 2)
	if err != nil {
		t.Fatalf("ClaimDeletionRequests() after permanent error = %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("permanent failure was retryable: %#v", claimed)
	}
	receiptID := uuid.New()
	if err := repo.CompleteDeletionRequest(ctx, second.ID, receiptID, now); err != nil {
		t.Fatalf("CompleteDeletionRequest() error = %v", err)
	}
	var storedReceipt uuid.UUID
	var storedUserID *uuid.UUID
	if err := db.QueryRow(ctx, "SELECT receipt_id, NULL::uuid FROM data_deletion_requests WHERE id = $1", second.ID).Scan(&storedReceipt, &storedUserID); err != nil {
		t.Fatalf("select receipt: %v", err)
	}
	if storedReceipt != receiptID || storedUserID != nil {
		t.Fatalf("receipt/user = %s/%v", storedReceipt, storedUserID)
	}

	thirdUserID := createRepositoryUser(t, ctx, db, "deletion-claim-3@example.test")
	fourthUserID := createRepositoryUser(t, ctx, db, "deletion-claim-4@example.test")
	if _, err := repo.RequestDeletion(ctx, thirdUserID); err != nil {
		t.Fatalf("RequestDeletion() third error = %v", err)
	}
	if _, err := repo.RequestDeletion(ctx, fourthUserID); err != nil {
		t.Fatalf("RequestDeletion() fourth error = %v", err)
	}
	var wg sync.WaitGroup
	claimedIDs := make(chan uuid.UUID, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claims, err := repo.ClaimDeletionRequests(ctx, now, 1)
			if err != nil {
				errs <- err
				return
			}
			if len(claims) != 1 {
				errs <- errors.New("worker did not claim exactly one request")
				return
			}
			claimedIDs <- claims[0].ID
		}()
	}
	wg.Wait()
	close(errs)
	close(claimedIDs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent claim error = %v", err)
		}
	}
	seen := map[uuid.UUID]bool{}
	for id := range claimedIDs {
		seen[id] = true
	}
	if len(seen) != 2 {
		t.Fatalf("concurrent claims were not distinct: %v", seen)
	}
}

func TestPostgresComplianceAndAdminRepositoryValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	userID := uuid.New()
	adminID := uuid.New()
	entityID := uuid.New()
	requestID := uuid.New()
	now := time.Now()
	deletionValues := []any{requestID, userID, "pending", now, (*time.Time)(nil), "", "", 0, (*time.Time)(nil), (*uuid.UUID)(nil), (*time.Time)(nil)}
	deletionAuditValues := []any{uuid.New(), requestID, "pending", "processing", "note", now}
	importValues := []any{uuid.New(), "usda", "fdc-1", &entityID, "conflict", "duplicate", []byte(`{"x":1}`), now, now}
	auditValues := []any{uuid.New(), adminID, "update", "food_item", &entityID, []byte(`{"before":true}`), []byte(`{"after":true}`), "req-1", now}

	complianceRepo := NewPostgresComplianceRepository(&fakeSQLExecutor{})
	if _, err := complianceRepo.RecordConsent(ctx, ConsentRecord{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordConsent() nil user error = %v, want validation", err)
	}
	if _, err := complianceRepo.RecordConsent(ctx, ConsentRecord{UserID: userID}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RecordConsent() missing versions error = %v, want validation", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := complianceRepo.RecordConsent(ctx, ConsentRecord{UserID: userID, PrivacyPolicyVersion: "p", TermsVersion: "t"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RecordConsent() scan error = %v, want connection", err)
	}
	if _, err := NewPostgresComplianceRepository(&fakeSQLExecutor{}).HasRequiredConsent(ctx, uuid.Nil, "p", "t"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("HasRequiredConsent() nil user error = %v, want validation", err)
	}
	if _, err := NewPostgresComplianceRepository(&fakeSQLExecutor{}).HasRequiredConsent(ctx, userID, "", "t"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("HasRequiredConsent() missing version error = %v, want validation", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := complianceRepo.HasRequiredConsent(ctx, userID, "p", "t"); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("HasRequiredConsent() scan error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{values: []any{true}}})
	if ok, err := complianceRepo.HasRequiredConsent(ctx, userID, "p", "t"); err != nil || !ok {
		t.Fatalf("HasRequiredConsent() fake success ok=%v err=%v", ok, err)
	}

	if _, err := complianceRepo.RequestDeletion(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("RequestDeletion() nil user error = %v, want validation", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := complianceRepo.RequestDeletion(ctx, userID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("RequestDeletion() scan error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{row: fakeRow{values: deletionValues}})
	if request, err := complianceRepo.RequestDeletion(ctx, userID); err != nil || request.ID != requestID {
		t.Fatalf("RequestDeletion() fake success request=%#v err=%v", request, err)
	}

	if err := complianceRepo.UpdateDeletionStatus(ctx, uuid.Nil, "pending", ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdateDeletionStatus() nil request error = %v, want validation", err)
	}
	if err := complianceRepo.UpdateDeletionStatus(ctx, requestID, "bad", ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpdateDeletionStatus() bad status error = %v, want validation", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{err: queryErr}}}})
	if err := complianceRepo.UpdateDeletionStatus(ctx, requestID, "processing", ""); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdateDeletionStatus() load error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{"pending"}}, execErr: queryErr}}})
	if err := complianceRepo.UpdateDeletionStatus(ctx, requestID, "processing", ""); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdateDeletionStatus() update error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{"pending"}}, execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 0")}}}})
	if err := complianceRepo.UpdateDeletionStatus(ctx, requestID, "processing", ""); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("UpdateDeletionStatus() missing error = %v, want not found", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{"pending"}}, execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")}, execErrs: []error{nil, queryErr}}}})
	if err := complianceRepo.UpdateDeletionStatus(ctx, requestID, "processing", ""); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpdateDeletionStatus() audit error = %v, want connection", err)
	}
	if _, err := complianceRepo.ListDeletionAudit(ctx, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListDeletionAudit() nil request error = %v, want validation", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := complianceRepo.ListDeletionAudit(ctx, requestID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListDeletionAudit() query error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := complianceRepo.ListDeletionAudit(ctx, requestID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListDeletionAudit() scan error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, err := complianceRepo.ListDeletionAudit(ctx, requestID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListDeletionAudit() rows error = %v, want connection", err)
	}
	complianceRepo = NewPostgresComplianceRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, values: deletionAuditValues}})
	if entries, err := complianceRepo.ListDeletionAudit(ctx, requestID); err != nil || len(entries) != 1 {
		t.Fatalf("ListDeletionAudit() fake success entries=%#v err=%v", entries, err)
	}

	adminRepo := NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{})
	if _, err := adminRepo.UpsertCuratedImport(ctx, CuratedImport{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpsertCuratedImport() missing identity error = %v, want validation", err)
	}
	if _, err := adminRepo.UpsertCuratedImport(ctx, CuratedImport{SourceProvider: "usda", ExternalID: "1", Status: "bad"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpsertCuratedImport() bad status error = %v, want validation", err)
	}
	if _, err := adminRepo.UpsertCuratedImport(ctx, CuratedImport{SourceProvider: "usda", ExternalID: "1", Status: "draft", RawPayload: []byte(`{`)}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("UpsertCuratedImport() bad payload error = %v, want validation", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := adminRepo.UpsertCuratedImport(ctx, CuratedImport{SourceProvider: "usda", ExternalID: "1", Status: "draft"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("UpsertCuratedImport() scan error = %v, want connection", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{})
	if _, err := adminRepo.FindCuratedImport(ctx, "", "1"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("FindCuratedImport() validation error = %v, want validation", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := adminRepo.FindCuratedImport(ctx, "usda", "1"); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("FindCuratedImport() scan error = %v, want connection", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{row: fakeRow{values: importValues}})
	if item, err := adminRepo.FindCuratedImport(ctx, "usda", "1"); err != nil || item.SourceProvider != "usda" {
		t.Fatalf("FindCuratedImport() fake success item=%#v err=%v", item, err)
	}

	if _, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("PersistAuditEntry() nil admin error = %v, want validation", err)
	}
	if _, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{AdminUserID: adminID}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("PersistAuditEntry() missing fields error = %v, want validation", err)
	}
	if _, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "x", EntityType: "food", Before: []byte(`{`)}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("PersistAuditEntry() bad before error = %v, want validation", err)
	}
	if _, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "x", EntityType: "food", After: []byte(`{`)}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("PersistAuditEntry() bad after error = %v, want validation", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "x", EntityType: "food"}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("PersistAuditEntry() scan error = %v, want connection", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{row: fakeRow{values: []any{uuid.New()}}})
	if _, err := adminRepo.PersistAuditEntry(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "x", EntityType: "food", Before: []byte(`{}`), After: []byte(`{}`)}); err != nil {
		t.Fatalf("PersistAuditEntry() fake success error = %v", err)
	}
	if err := adminRepo.WithAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "x", EntityType: "food"}, nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("WithAudit() nil mutation error = %v, want validation", err)
	}
	if err := adminRepo.WithAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "x", EntityType: "food"}, func(sqlExecutor) error { return queryErr }); !errors.Is(err, queryErr) {
		t.Fatalf("WithAudit() mutation error = %v, want raw query error", err)
	}
	if _, err := adminRepo.ListAuditForEntity(ctx, "", entityID); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ListAuditForEntity() validation error = %v, want validation", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := adminRepo.ListAuditForEntity(ctx, "food", entityID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListAuditForEntity() query error = %v, want connection", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := adminRepo.ListAuditForEntity(ctx, "food", entityID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListAuditForEntity() scan error = %v, want connection", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, err := adminRepo.ListAuditForEntity(ctx, "food", entityID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListAuditForEntity() rows error = %v, want connection", err)
	}
	adminRepo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, values: auditValues}})
	if entries, err := adminRepo.ListAuditForEntity(ctx, "food", entityID); err != nil || len(entries) != 1 {
		t.Fatalf("ListAuditForEntity() fake success entries=%#v err=%v", entries, err)
	}
}

func TestPostgresFoodItemRepositoryUpdateAndDeleteMissing(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	foodRepo := NewPostgresFoodItemRepository(db)
	missingID := uuid.New()

	if err := foodRepo.Update(ctx, FoodItemEntity{ID: uuid.Nil}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Update() missing id error = %v, want validation", err)
	}
	if err := foodRepo.Update(ctx, FoodItemEntity{ID: missingID, Name: "Bad Functionality", PhysicalState: PhysicalStateSolid, CulinaryRoles: []ClassificationEntity{{ID: uuid.Nil}}}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Update() invalid culinary_role classification error = %v, want validation", err)
	}
	if err := foodRepo.Update(ctx, FoodItemEntity{ID: missingID, Name: "Missing", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("Update() missing row error = %v, want not found", err)
	}
	if err := foodRepo.Delete(ctx, missingID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("Delete() missing row error = %v, want not found", err)
	}
}

func TestPostgresMealRepositorySingleRecipeAndMacros(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	foodRepo := NewPostgresFoodItemRepository(db)
	mealRepo := NewPostgresMealRepository(db)
	classificationRepo := NewPostgresClassificationRepository(db)

	classificationID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Dinner", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create meal classification: %v", err)
	}
	riceID, err := foodRepo.Create(ctx, FoodItemEntity{
		Name:                   "Rice",
		PhysicalState:          PhysicalStateSolid,
		AverageUnitWeightGrams: 50,
		MacrosPer100:           MacroValues{Protein: 2, Carbohydrates: 30, Fat: 1},
	})
	if err != nil {
		t.Fatalf("create rice: %v", err)
	}
	beansID, err := foodRepo.Create(ctx, FoodItemEntity{
		Name:                   "Beans",
		PhysicalState:          PhysicalStateSolid,
		AverageUnitWeightGrams: 40,
		MacrosPer100:           MacroValues{Protein: 10, Carbohydrates: 20, Fat: 2},
	})
	if err != nil {
		t.Fatalf("create beans: %v", err)
	}

	singleID, err := mealRepo.Create(ctx, MealEntity{
		Type:                   MealTypeSingle,
		Name:                   "Restaurant Rice Bowl",
		PhysicalState:          PhysicalStateSolid,
		PrepTimeMinutes:        5,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           MacroValues{Protein: 2, Carbohydrates: 30, Fat: 1},
		Classifications:        []ClassificationEntity{{ID: classificationID}},
	})
	if err != nil {
		t.Fatalf("Create() single error = %v", err)
	}
	single, err := mealRepo.GetByID(ctx, singleID, RepositoryContext{UnitSystem: UnitSystemImperial})
	if err != nil {
		t.Fatalf("GetByID() single error = %v", err)
	}
	if single.Type != MealTypeSingle || single.Name != "Restaurant Rice Bowl" || len(single.Classifications) != 1 {
		t.Fatalf("single meal = %#v", single)
	}
	if single.AverageUnitWeightGrams != 3.5274 {
		t.Fatalf("single imperial weight = %v, want 3.5274", single.AverageUnitWeightGrams)
	}
	singleMacros, err := mealRepo.CalculateMacros(ctx, singleID)
	if err != nil {
		t.Fatalf("CalculateMacros() single error = %v", err)
	}
	if singleMacros != (MacroValues{Protein: 2, Carbohydrates: 30, Fat: 1}) {
		t.Fatalf("single macros = %#v", singleMacros)
	}

	recipeID, err := mealRepo.Create(ctx, MealEntity{
		Type:          MealTypeComposite,
		Name:          "Rice and Beans",
		PhysicalState: PhysicalStateSolid,
		RecipeItems: []RecipeIngredientEntity{
			{FoodItemID: riceID, Quantity: 150, Unit: "g", Position: 0},
			{FoodItemID: beansID, Quantity: 2, Unit: "serving", Position: 1},
		},
		Classifications: []ClassificationEntity{{ID: classificationID}},
	})
	if err != nil {
		t.Fatalf("Create() recipe error = %v", err)
	}
	recipe, err := mealRepo.GetByID(ctx, recipeID, RepositoryContext{})
	if err != nil {
		t.Fatalf("GetByID() recipe error = %v", err)
	}
	if recipe.Type != MealTypeComposite || len(recipe.RecipeItems) != 2 || len(recipe.Classifications) != 1 {
		t.Fatalf("composite meal = %#v", recipe)
	}
	recipeMacros, err := mealRepo.CalculateMacros(ctx, recipeID)
	if err != nil {
		t.Fatalf("CalculateMacros() recipe error = %v", err)
	}
	wantRecipeMacros := MacroValues{Protein: 4.7826, Carbohydrates: 26.5217, Fat: 1.3478}
	if recipeMacros != wantRecipeMacros {
		t.Fatalf("recipe macros = %#v, want %#v", recipeMacros, wantRecipeMacros)
	}

	recipe.RecipeItems = []RecipeIngredientEntity{{FoodItemID: riceID, Quantity: 1, Unit: "oz", Position: 0}}
	if err := mealRepo.Update(ctx, recipe); err != nil {
		t.Fatalf("Update() recipe error = %v", err)
	}
	updatedMacros, err := mealRepo.CalculateMacros(ctx, recipeID)
	if err != nil {
		t.Fatalf("CalculateMacros() updated recipe error = %v", err)
	}
	if updatedMacros != (MacroValues{Protein: 2, Carbohydrates: 30.0002, Fat: 1}) {
		t.Fatalf("updated recipe macros = %#v", updatedMacros)
	}
}

func TestPostgresMealRepositoryLiquidRecipeNormalizationAndUnits(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	foodRepo := NewPostgresFoodItemRepository(db)
	mealRepo := NewPostgresMealRepository(db)

	solidID, err := foodRepo.Create(ctx, FoodItemEntity{Name: "Solid Ingredient", PhysicalState: PhysicalStateSolid, AverageUnitWeightGrams: 50, MacrosPer100: MacroValues{Protein: 10}})
	if err != nil {
		t.Fatalf("create solid ingredient: %v", err)
	}
	liquidID, err := foodRepo.Create(ctx, FoodItemEntity{Name: "Liquid Ingredient", PhysicalState: PhysicalStateLiquid, AverageServingVolumeMilliliters: 125, DensityGramsPerMilliliter: 0.8, DensitySourceKind: "manual", MacrosPer100: MacroValues{Protein: 8}})
	if err != nil {
		t.Fatalf("create liquid ingredient: %v", err)
	}

	invalid := []RecipeIngredientEntity{
		{FoodItemID: solidID, Quantity: 100, Unit: "ml"},
		{FoodItemID: solidID, Quantity: 1, Unit: "fl_oz"},
		{FoodItemID: liquidID, Quantity: 100, Unit: "g"},
		{FoodItemID: liquidID, Quantity: 1, Unit: "oz"},
	}
	for _, ingredient := range invalid {
		_, err := mealRepo.Create(ctx, MealEntity{Type: MealTypeComposite, Name: "Invalid " + ingredient.Unit, PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{ingredient}})
		if !IsKind(err, ErrorKindUnitConversion) {
			t.Fatalf("Create() cross-basis %s error = %v, want unit conversion", ingredient.Unit, err)
		}
	}

	valid := []RecipeIngredientEntity{
		{FoodItemID: liquidID, Quantity: 100, Unit: "ml"},
		{FoodItemID: liquidID, Quantity: 1, Unit: "fl_oz"},
		{FoodItemID: liquidID, Quantity: 1, Unit: "serving"},
	}
	for _, ingredient := range valid {
		id, err := mealRepo.Create(ctx, MealEntity{Type: MealTypeComposite, Name: "Liquid Recipe " + ingredient.Unit, PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{ingredient}})
		if err != nil {
			t.Fatalf("Create() liquid %s error = %v", ingredient.Unit, err)
		}
		meal, err := mealRepo.GetByID(ctx, id, RepositoryContext{})
		if err != nil {
			t.Fatalf("GetByID() liquid %s error = %v", ingredient.Unit, err)
		}
		wantProtein := 10.0
		if ingredient.Unit == "fl_oz" {
			wantProtein = 10.0001
		}
		if !meal.NormalizedMacrosAvailable || meal.MacrosPer100 != (MacroValues{Protein: wantProtein}) {
			t.Fatalf("GetByID() liquid %s meal = %#v", ingredient.Unit, meal)
		}
	}
}

func TestPostgresMealRepositoryValidationAndCycles(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	foodRepo := NewPostgresFoodItemRepository(db)
	mealRepo := NewPostgresMealRepository(db)

	foodID, err := foodRepo.Create(ctx, FoodItemEntity{Name: "Egg", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 13, Carbohydrates: 1, Fat: 11}})
	if err != nil {
		t.Fatalf("create food: %v", err)
	}

	invalidMeals := []struct {
		name string
		meal MealEntity
		kind ErrorKind
	}{
		{name: "bad type", meal: MealEntity{Type: "snack", PhysicalState: PhysicalStateSolid}, kind: ErrorKindValidation},
		{name: "bad state", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: "gas"}, kind: ErrorKindValidation},
		{name: "negative prep", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid, PrepTimeMinutes: -1}, kind: ErrorKindValidation},
		{name: "missing name", meal: MealEntity{Type: MealTypeSingle, PhysicalState: PhysicalStateSolid}, kind: ErrorKindValidation},
		{name: "single with ingredients", meal: MealEntity{Type: MealTypeSingle, Name: "Bad", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}, kind: ErrorKindValidation},
		{name: "composite without ingredients", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid}, kind: ErrorKindValidation},
		{name: "missing ingredient food", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: uuid.New(), Quantity: 1, Unit: "g"}}}, kind: ErrorKindNotFound},
		{name: "invalid ingredient unit", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "cup"}}}, kind: ErrorKindUnitConversion},
		{name: "zero quantity", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 0, Unit: "g"}}}, kind: ErrorKindValidation},
		{name: "duplicate position", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g", Position: 0}, {FoodItemID: foodID, Quantity: 1, Unit: "g", Position: 0}}}, kind: ErrorKindValidation},
		{name: "nil meal classification", meal: MealEntity{Type: MealTypeComposite, Name: "Bad", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}, Classifications: []ClassificationEntity{{ID: uuid.Nil}}}, kind: ErrorKindValidation},
	}
	for _, tt := range invalidMeals {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := mealRepo.Create(ctx, tt.meal); !IsKind(err, tt.kind) {
				t.Fatalf("Create() error = %v, want %s", err, tt.kind)
			}
		})
	}

	collisionID := uuid.New()
	if _, err := db.Exec(ctx, testCollisionFoodCreateSQL, collisionID); err != nil {
		t.Fatalf("create collision food: %v", err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin collision fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, testCollisionMealCreateSQL, collisionID); err != nil {
		t.Fatalf("create collision meal: %v", err)
	}
	if _, err := tx.Exec(ctx, testCollisionIngredientCreateSQL, collisionID); err != nil {
		t.Fatalf("create collision ingredient: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("create collision fixture: %v", err)
	}
	if err := mealRepo.Update(ctx, MealEntity{ID: collisionID, Type: MealTypeComposite, Name: "Collision Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: collisionID, Quantity: 1, Unit: "g"}}}); err != nil {
		t.Fatalf("Update() UUID collision error = %v", err)
	}

	if err := mealRepo.Delete(ctx, collisionID); err != nil {
		t.Fatalf("Delete() meal error = %v", err)
	}
	if _, err := mealRepo.GetByID(ctx, collisionID, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetByID() deleted meal error = %v, want not found", err)
	}
	if _, err := mealRepo.GetByID(ctx, collisionID, RepositoryContext{IncludeDeleted: true}); err != nil {
		t.Fatalf("GetByID() include deleted meal error = %v", err)
	}
}

func TestPostgresMealRepositoryErrorBranches(t *testing.T) {
	ctx := context.Background()
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	execErr := errors.New("exec failed")
	commitErr := errors.New("commit failed")
	now := time.Now()
	mealID := uuid.New()
	foodID := uuid.New()
	mealValues := []any{mealID, MealTypeSingle, "Opaque Meal", PhysicalStateSolid, 0, floatPtr(100), floatPtr(1), floatPtr(2), floatPtr(3), now, now}
	recipeValues := []any{mealID, MealTypeComposite, "Composite Meal", PhysicalStateLiquid, 0, floatPtr(250), (*float64)(nil), (*float64)(nil), (*float64)(nil), now, now}
	foodValues := []any{foodID, "Ingredient", PhysicalStateSolid, 0, floatPtr(100), (*float64)(nil), (*float64)(nil), (*string)(nil), (*string)(nil), (*string)(nil), 1.0, 2.0, 3.0, []byte(`{}`), (*string)(nil), (*time.Time)(nil), now, now}

	repo := NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.GetByID(ctx, mealID, RepositoryContext{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetByID() scan error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() query error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() scan error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() rows error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{
		rowsList:  []pgx.Rows{&fakeRows{next: true, values: []any{mealID}}},
		queryErrs: []error{nil, queryErr},
	})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() hydrate error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: recipeValues}, queryErr: queryErr})
	if _, err := repo.GetByID(ctx, mealID, RepositoryContext{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetByID() ingredient query error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: mealValues}, rows: &fakeRows{err: queryErr}})
	if _, err := repo.GetByID(ctx, mealID, RepositoryContext{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetByID() classification rows error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.CalculateMacros(ctx, mealID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CalculateMacros() get error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: mealValues}, rows: &fakeRows{}})
	if macros, err := repo.CalculateMacros(ctx, mealID); err != nil || macros != (MacroValues{Protein: 1, Carbohydrates: 2, Fat: 3}) {
		t.Fatalf("CalculateMacros() single = %#v, %v", macros, err)
	}

	ingredientValues := []any{foodID, 1.0, "g", 0}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{rowList: []fakeRow{{values: recipeValues}, {err: scanErr}}, rowsList: []pgx.Rows{&fakeRows{next: true, values: ingredientValues}, &fakeRows{}}})
	if _, err := repo.CalculateMacros(ctx, mealID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("CalculateMacros() recipe food error = %v, want connection", err)
	}

	cupIngredientValues := []any{foodID, 1.0, "cup", 0}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{rowList: []fakeRow{{values: recipeValues}, {values: foodValues}}, rowsList: []pgx.Rows{&fakeRows{next: true, values: cupIngredientValues}, &fakeRows{}, &fakeRows{}}})
	if _, err := repo.CalculateMacros(ctx, mealID); !IsKind(err, ErrorKindUnitConversion) {
		t.Fatalf("CalculateMacros() recipe unit error = %v, want unit conversion", err)
	}

	badMealValues := []any{mealID, MealType("bad"), "Bad Meal", PhysicalStateSolid, 0, (*float64)(nil), (*float64)(nil), (*float64)(nil), (*float64)(nil), now, now}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: badMealValues}, rows: &fakeRows{}})
	if _, err := repo.CalculateMacros(ctx, mealID); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("CalculateMacros() bad type error = %v, want validation", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, rows: &fakeRows{}, beginErr: execErr})
	if _, err := repo.Create(ctx, MealEntity{Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() begin error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{err: scanErr}}}})
	if _, err := repo.Create(ctx, MealEntity{Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() insert error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{mealID}}, execErrs: []error{nil, execErr}}}})
	if _, err := repo.Create(ctx, MealEntity{Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() replace ingredient error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, rows: &fakeRows{}, tx: &fakeTx{commitErr: commitErr, fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{mealID}}}}})
	if _, err := repo.Create(ctx, MealEntity{Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() commit error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErr: execErr}}})
	if err := repo.Update(ctx, MealEntity{ID: mealID, Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Update() exec error = %v, want connection", err)
	}

	if err := repo.Update(ctx, MealEntity{ID: uuid.Nil}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Update() nil id error = %v, want validation", err)
	}

	if err := repo.Update(ctx, MealEntity{ID: mealID, Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: "steam"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Update() validation error = %v, want validation", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 0")}}}})
	if err := repo.Update(ctx, MealEntity{ID: mealID, Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("Update() missing error = %v, want not found", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{rowList: []fakeRow{{values: foodValues}, {values: []any{true}}}, rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")}, execErrs: []error{nil, nil, execErr}}}})
	if err := repo.Update(ctx, MealEntity{ID: mealID, Type: MealTypeComposite, Name: "Composite Meal", PhysicalState: PhysicalStateSolid, RecipeItems: []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}, Classifications: []ClassificationEntity{{ID: uuid.New()}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Update() replace classification error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{execErr: execErr})
	if err := repo.Delete(ctx, mealID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Delete() exec error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 0")}})
	if err := repo.Delete(ctx, mealID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("Delete() missing error = %v, want not found", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if err := repo.validateMealClassifications(ctx, []ClassificationEntity{{ID: uuid.New()}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("validateMealClassifications() scan error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{values: []any{false}}})
	if err := repo.validateMealClassifications(ctx, []ClassificationEntity{{ID: uuid.New()}}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateMealClassifications() missing error = %v, want validation", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{execErr: execErr})
	if err := repo.replaceIngredients(ctx, mealID, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceIngredients() clear error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{execErrs: []error{nil, execErr}})
	if err := repo.replaceIngredients(ctx, mealID, []RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 1, Unit: "g"}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceIngredients() insert error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{execErr: execErr})
	if err := repo.replaceMealClassifications(ctx, mealID, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceMealClassifications() clear error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{execErrs: []error{nil, execErr}})
	if err := repo.replaceMealClassifications(ctx, mealID, []ClassificationEntity{{ID: uuid.New()}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceMealClassifications() insert error = %v, want connection", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := repo.loadIngredients(ctx, mealID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("loadIngredients() query error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := repo.loadIngredients(ctx, mealID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("loadIngredients() scan error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, err := repo.loadIngredients(ctx, mealID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("loadIngredients() rows error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{queryErr: queryErr})
	if err := repo.hydrateMealClassifications(ctx, &MealEntity{ID: mealID}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateMealClassifications() query error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if err := repo.hydrateMealClassifications(ctx, &MealEntity{ID: mealID}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateMealClassifications() scan error = %v, want connection", err)
	}
	repo = NewPostgresMealRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if err := repo.hydrateMealClassifications(ctx, &MealEntity{ID: mealID}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateMealClassifications() rows error = %v, want connection", err)
	}

	if _, err := ingredientBasisQuantity(RecipeIngredientEntity{Quantity: 1, Unit: "ml"}, FoodItemEntity{PhysicalState: PhysicalStateLiquid}); err != nil {
		t.Fatalf("ingredientBasisQuantity() ml error = %v", err)
	}
	if got, err := ingredientBasisQuantity(RecipeIngredientEntity{Quantity: 1, Unit: "fl_oz"}, FoodItemEntity{PhysicalState: PhysicalStateLiquid}); err != nil || got != 29.5735 {
		t.Fatalf("ingredientBasisQuantity() fl_oz = %v, %v; want 29.5735 nil", got, err)
	}
	if _, err := ingredientBasisQuantity(RecipeIngredientEntity{Quantity: 1, Unit: "g"}, FoodItemEntity{PhysicalState: "frozen"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ingredientBasisQuantity() invalid state error = %v, want validation", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{})
	if err := repo.validateIngredients(ctx, uuid.Nil, []RecipeIngredientEntity{{FoodItemID: uuid.Nil, Quantity: 1, Unit: "g"}}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateIngredients() missing food id error = %v, want validation", err)
	}

	repo = NewPostgresMealRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if err := repo.validateMeal(ctx, MealEntity{Type: MealTypeSingle, Name: "Impossible", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 101}}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("validateMeal() macro error = %v, want validation", err)
	}

	meal := MealEntity{PhysicalState: PhysicalStateLiquid, AverageUnitWeightGrams: 250}
	convertMealForUnitSystem(&meal, UnitSystemImperial)
	if meal.AverageUnitWeightGrams != 8.4535 {
		t.Fatalf("convertMealForUnitSystem() liquid = %v, want 8.4535", meal.AverageUnitWeightGrams)
	}

}

func TestPostgresFoodItemRepositoryErrorBranches(t *testing.T) {
	ctx := context.Background()
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	execErr := errors.New("exec failed")
	now := time.Now()
	foodID := uuid.New()
	foodValues := []any{
		foodID,
		"Water",
		PhysicalStateLiquid,
		0,
		floatPtr(250),
		floatPtr(250),
		floatPtr(1),
		stringPtr("usda"),
		stringPtr("water"),
		stringPtr("imported"),
		0.0,
		0.0,
		0.0,
		[]byte(`{}`),
		stringPtr("https://example.test/water.jpg"),
		(*time.Time)(nil),
		now,
		now,
	}

	repo := NewPostgresFoodItemRepository(&fakeSQLExecutor{row: fakeRow{values: foodValues}, queryErr: queryErr})
	if _, err := repo.GetByID(ctx, foodID, RepositoryContext{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetByID() hydrate query error = %v, want connection", err)
	}

	invalidFoodValues := append([]any{}, foodValues...)
	invalidFoodValues[13] = []byte(`[`)
	invalidFoodValues[14] = (*string)(nil)
	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{row: fakeRow{values: invalidFoodValues}})
	if _, err := repo.GetByID(ctx, foodID, RepositoryContext{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetByID() invalid micros error = %v, want validation", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() query error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() count error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() scan error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() rows error = %v, want connection", err)
	}

	searchValues := append([]any{}, foodValues...)
	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{row: fakeRow{values: []any{1}}, rowsList: []pgx.Rows{&fakeRows{next: true, values: searchValues}, nil}, queryErrs: []error{nil, queryErr}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{RepositoryContext: RepositoryContext{UnitSystem: UnitSystemImperial}, Limit: -1, Offset: -1}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Search() hydrate error = %v, want connection", err)
	}

	invalidSearchValues := append([]any{}, searchValues...)
	invalidSearchValues[13] = []byte(`[`)
	invalidSearchValues[14] = (*string)(nil)
	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{row: fakeRow{values: []any{1}}, rows: &fakeRows{next: true, values: invalidSearchValues}})
	if _, _, err := repo.Search(ctx, RepositoryQuery{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Search() invalid micros error = %v, want validation", err)
	}

	failedCreateTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{foodID}}, execErr: execErr}}
	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{}, tx: failedCreateTx})
	validWater := FoodItemEntity{Name: "Water", PhysicalState: PhysicalStateLiquid, DensityGramsPerMilliliter: 1, DensitySourceKind: "manual", MacrosPer100: MacroValues{}}
	if _, err := repo.Create(ctx, validWater); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() replace classifications error = %v, want connection", err)
	}
	if !failedCreateTx.rolledBack {
		t.Fatal("Create() replace classifications error did not roll back transaction")
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{err: scanErr}}}})
	if _, err := repo.Create(ctx, validWater); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() insert scan error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErr: execErr}}})
	validWater.ID = foodID
	if err := repo.Update(ctx, validWater); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Update() exec error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")}, execErrs: []error{nil, execErr}}}})
	if err := repo.Update(ctx, validWater); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Update() replace classifications error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{execErr: execErr})
	if err := repo.Delete(ctx, foodID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Delete() exec error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if _, err := repo.Create(ctx, FoodItemEntity{Name: "Bad Classification Scan", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, FoodCategories: []ClassificationEntity{{ID: uuid.New()}}}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Create() classification scan error = %v, want connection", err)
	}

	if _, err := NewPostgresFoodItemRepository(nil).Create(ctx, FoodItemEntity{Name: "Nil Classification", PhysicalState: PhysicalStateSolid, FoodCategories: []ClassificationEntity{{ID: uuid.Nil}}}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Create() nil classification error = %v, want validation", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{queryErr: queryErr})
	if err := repo.validateMicronutrients(ctx, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("validateMicronutrients() query error = %v, want connection", err)
	}

	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{execErrs: []error{nil, execErr}})
	if err := repo.replaceFoodClassifications(ctx, foodID, []ClassificationEntity{{ID: uuid.New()}}, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceFoodClassifications() insert error = %v, want connection", err)
	}

	itemForHydration := FoodItemEntity{ID: foodID}
	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if err := repo.hydrateFoodClassifications(ctx, &itemForHydration); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateFoodClassifications() scan error = %v, want connection", err)
	}
	repo = NewPostgresFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{err: queryErr}})
	if err := repo.hydrateFoodClassifications(ctx, &itemForHydration); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateFoodClassifications() rows error = %v, want connection", err)
	}

	noMicrosValues := append([]any{}, foodValues...)
	noMicrosValues[13] = []byte{}
	noMicrosValues[14] = (*string)(nil)
	item, err := scanFoodItem(fakeRow{values: noMicrosValues})
	if err != nil {
		t.Fatalf("scanFoodItem() no micros error = %v", err)
	}
	if item.Micros == nil || item.ImageURL != "" {
		t.Fatalf("scanFoodItem() no micros item = %#v", item)
	}
	item, err = scanFoodItem(&fakeRows{values: noMicrosValues})
	if err != nil {
		t.Fatalf("scanFoodItem() rows no micros error = %v", err)
	}
	if item.Micros == nil || item.ImageURL != "" {
		t.Fatalf("scanFoodItem() rows no micros item = %#v", item)
	}

	convertFoodItemForUnitSystem(&item, UnitSystemImperial)
	if item.AverageServingVolumeMilliliters != 8.4535 {
		t.Fatalf("liquid imperial serving volume = %v, want 8.4535", item.AverageServingVolumeMilliliters)
	}
}

func TestMapPostgresError(t *testing.T) {
	if err := mapPostgresError(nil, "ok"); err != nil {
		t.Fatalf("mapPostgresError(nil) = %v, want nil", err)
	}
	if err := mapPostgresError(pgx.ErrNoRows, "missing"); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("mapPostgresError(no rows) = %v, want not found", err)
	}
	if err := mapPostgresError(errors.New("network"), "query"); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("mapPostgresError(generic) = %v, want connection", err)
	}
	tests := []struct {
		code string
		kind ErrorKind
	}{
		{code: "23505", kind: ErrorKindConflict},
		{code: "23502", kind: ErrorKindValidation},
		{code: "23503", kind: ErrorKindValidation},
		{code: "23514", kind: ErrorKindValidation},
		{code: "22001", kind: ErrorKindValidation},
		{code: "22003", kind: ErrorKindValidation},
		{code: "22P02", kind: ErrorKindValidation},
		{code: "40001", kind: ErrorKindRetryable},
		{code: "40P01", kind: ErrorKindRetryable},
		{code: "57014", kind: ErrorKindCanceled},
		{code: "08006", kind: ErrorKindConnection},
		{code: "42501", kind: ErrorKindInternal},
		{code: "42P01", kind: ErrorKindInternal},
		{code: "99999", kind: ErrorKindInternal},
	}
	for _, tt := range tests {
		if err := mapPostgresError(&pgconn.PgError{Code: tt.code}, "query"); !IsKind(err, tt.kind) {
			t.Fatalf("mapPostgresError(%s) = %v, want %s", tt.code, err, tt.kind)
		}
	}
}

func TestPostgresRepositoryErrorBranches(t *testing.T) {
	queryErr := errors.New("query failed")
	scanErr := errors.New("scan failed")
	rowsErr := errors.New("rows failed")
	execErr := errors.New("exec failed")

	classificationRepo := NewPostgresClassificationRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := classificationRepo.List(context.Background(), ClassificationKindFoodCategory); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("classification List() query error = %v, want connection", err)
	}

	classificationRepo = NewPostgresClassificationRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := classificationRepo.List(context.Background(), ClassificationKindFoodCategory); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("classification List() scan error = %v, want connection", err)
	}

	classificationRepo = NewPostgresClassificationRepository(&fakeSQLExecutor{rows: &fakeRows{err: rowsErr}})
	if _, err := classificationRepo.List(context.Background(), ClassificationKindFoodCategory); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("classification List() rows error = %v, want connection", err)
	}

	classificationRepo = NewPostgresClassificationRepository(&fakeSQLExecutor{row: fakeRow{values: []any{false}}, execErr: execErr})
	if err := classificationRepo.SoftDelete(context.Background(), uuid.New()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("classification SoftDelete() exec error = %v, want connection", err)
	}

	classificationRepo = NewPostgresClassificationRepository(&fakeSQLExecutor{row: fakeRow{err: scanErr}})
	if err := classificationRepo.SoftDelete(context.Background(), uuid.New()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("classification SoftDelete() usage error = %v, want connection", err)
	}

	vocabRepo := NewPostgresMicronutrientVocabularyRepository(&fakeSQLExecutor{queryErr: queryErr})
	if _, err := vocabRepo.ListActive(context.Background()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("vocab ListActive() query error = %v, want connection", err)
	}

	vocabRepo = NewPostgresMicronutrientVocabularyRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: scanErr}})
	if _, err := vocabRepo.ListActive(context.Background()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("vocab ListActive() scan error = %v, want connection", err)
	}

	vocabRepo = NewPostgresMicronutrientVocabularyRepository(&fakeSQLExecutor{rows: &fakeRows{err: rowsErr}})
	if _, err := vocabRepo.ListActive(context.Background()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("vocab ListActive() rows error = %v, want connection", err)
	}
}

type fakeSQLExecutor struct {
	rows      pgx.Rows
	rowsList  []pgx.Rows
	row       fakeRow
	rowList   []fakeRow
	queryErr  error
	queryErrs []error
	queryN    int
	execErr   error
	execErrs  []error
	execTags  []pgconn.CommandTag
	execN     int
	rowN      int
	beginErr  error
	tx        *fakeTx
}

func (e *fakeSQLExecutor) Begin(context.Context) (pgx.Tx, error) {
	if e.beginErr != nil {
		return nil, e.beginErr
	}
	if e.tx != nil {
		return e.tx, nil
	}
	return &fakeTx{}, nil
}

func (e *fakeSQLExecutor) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if len(e.execErrs) > 0 {
		index := e.execN
		if index >= len(e.execErrs) {
			index = len(e.execErrs) - 1
		}
		err := e.execErrs[index]
		classification := pgconn.CommandTag{}
		if index < len(e.execTags) {
			classification = e.execTags[index]
		}
		e.execN++
		return classification, err
	}
	if len(e.execTags) > 0 {
		return e.execTags[0], e.execErr
	}
	return pgconn.CommandTag{}, e.execErr
}

func (e *fakeSQLExecutor) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if len(e.queryErrs) > 0 {
		index := e.queryN
		if index >= len(e.queryErrs) {
			index = len(e.queryErrs) - 1
		}
		e.queryN++
		if e.queryErrs[index] != nil {
			return nil, e.queryErrs[index]
		}
		if index < len(e.rowsList) {
			return e.rowsList[index], nil
		}
	}
	if e.queryErr != nil {
		return nil, e.queryErr
	}
	if len(e.rowsList) > 0 {
		return e.rowsList[0], nil
	}
	return e.rows, nil
}

func (e *fakeSQLExecutor) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(e.rowList) > 0 {
		index := e.rowN
		if index >= len(e.rowList) {
			index = len(e.rowList) - 1
		}
		e.rowN++
		return e.rowList[index]
	}
	return e.row
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		if value == nil {
			continue
		}
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type fakeRows struct {
	next    bool
	scanErr error
	err     error
	values  []any
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	next := r.next
	r.next = false
	return next
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i, value := range r.values {
		if value == nil {
			continue
		}
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

type fakeTx struct {
	fakeSQLExecutor
	commitErr   error
	rollbackErr error
	rolledBack  bool
}

func (t *fakeTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }
func (t *fakeTx) Commit(context.Context) error          { return t.commitErr }
func (t *fakeTx) Rollback(context.Context) error {
	t.rolledBack = true
	return t.rollbackErr
}
func (t *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *fakeTx) Conn() *pgx.Conn { return nil }

func floatPtr(value float64) *float64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

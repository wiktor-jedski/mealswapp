package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/adminbootstrap"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
	"github.com/wiktor-jedski/mealswapp/backend/internal/testdatabase"
)

// Implements DESIGN-009 AdminController private interactive email input and safe output.
func TestRunReadsEmailInteractivelyAndEmitsOnlySafeMetadata(t *testing.T) {
	sensitive := "Admin.Secret@Example.test"
	userID, auditID := uuid.New(), uuid.New()
	var captured adminbootstrap.Request
	execute := func(_ context.Context, request adminbootstrap.Request) (repository.AdministratorBootstrapResult, error) {
		captured = request
		return repository.AdministratorBootstrapResult{UserID: userID, AuditID: &auditID, AuditedAt: time.Now()}, nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--environment", "development"}, strings.NewReader(sensitive+"\n"), &stdout, &stderr, execute); code != 0 {
		t.Fatalf("run() code = %d stderr=%q", code, stderr.String())
	}
	if captured.Email != sensitive || !strings.Contains(stderr.String(), "Administrator email:") {
		t.Fatalf("captured=%#v stderr=%q", captured, stderr.String())
	}
	combined := stdout.String() + stderr.String()
	for _, forbidden := range []string{sensitive, "postgres://", "password", "hash", "salt", "token", "cookie", "secret-key"} {
		if strings.Contains(strings.ToLower(combined), strings.ToLower(forbidden)) {
			t.Fatalf("output contains forbidden %q: %q", forbidden, combined)
		}
	}
	if !strings.Contains(stdout.String(), userID.String()) || !strings.Contains(stdout.String(), auditID.String()) || !strings.Contains(stdout.String(), "actor=operator") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunSupportsUUIDAndSafeFailures(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		name string
		args []string
		err  error
		code int
	}{
		{"uuid", []string{"--environment", "production", "--confirm-production", "--user-id", id.String()}, nil, 0},
		{"bad uuid", []string{"--environment", "development", "--user-id", "private@example.test"}, nil, 2},
		{"two selectors", []string{"--environment", "development", "--user-id", id.String(), "--email", "private@example.test"}, nil, 2},
		{"other admin", []string{"--environment", "development", "--user-id", id.String()}, repository.ErrAdministratorAlreadyExists, 1},
		{"unknown internal", []string{"--environment", "development", "--user-id", id.String()}, errors.New("postgres://user:pass@host/db secret"), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &stdout, &stderr, func(context.Context, adminbootstrap.Request) (repository.AdministratorBootstrapResult, error) {
				return repository.AdministratorBootstrapResult{UserID: id, Replayed: true}, tc.err
			})
			if code != tc.code {
				t.Fatalf("code = %d, want %d", code, tc.code)
			}
			combined := stdout.String() + stderr.String()
			for _, forbidden := range []string{"private@example.test", "user:pass", "secret"} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("output leaked %q: %q", forbidden, combined)
				}
			}
		})
	}
}

func TestRunRejectsMissingOrMalformedArguments(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		stdin string
	}{
		{args: []string{}},
		{args: []string{"--environment"}},
		{args: []string{"--environment", "development", "extra"}},
		{args: []string{"--environment", "development"}},
	} {
		var stderr bytes.Buffer
		if code := run(tc.args, strings.NewReader(tc.stdin), ioDiscard{}, &stderr, nil); code != 2 {
			t.Fatalf("run(%q) code = %d", tc.args, code)
		}
	}
}

func TestSafeBootstrapErrorsAndLookupKeyConfiguration(t *testing.T) {
	for err, want := range map[error]string{
		repository.ErrAdministratorBootstrapTargetNotFound: "target not found",
		repository.ErrAdministratorBootstrapUnverified:     "target is not verified",
		repository.ErrAdministratorBootstrapNoCredential:   "target has no usable credential",
		repository.ErrAdministratorAlreadyExists:           "another administrator exists",
		repository.ErrCanonicalEmailCollision:              "email identity conflict",
		errors.New("sensitive internal error"):             "request rejected",
	} {
		if got := safeBootstrapError(err); got != want {
			t.Fatalf("safeBootstrapError(%v) = %q, want %q", err, got, want)
		}
	}

	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")
	loader, err := newLookupKeyLoader("development")
	if err != nil {
		t.Fatalf("newLookupKeyLoader(development) error = %v", err)
	}
	version, active, err := loader.ActiveLookupKey(context.Background())
	if err != nil || version != "local-v1" || len(active) != 32 {
		t.Fatalf("active version=%q len=%d err=%v", version, len(active), err)
	}
	loaded, err := loader.LookupKey(context.Background(), version)
	if err != nil || !bytes.Equal(active, loaded) {
		t.Fatalf("lookup key mismatch err=%v", err)
	}
	if _, err := newLookupKeyLoader("production"); err == nil {
		t.Fatal("newLookupKeyLoader(production) accepted missing key")
	}
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", strings.Repeat("k", 40))
	if loader, err = newLookupKeyLoader("production"); err != nil || len(loader.key) != 32 {
		t.Fatalf("production loader len=%d err=%v", len(loader.key), err)
	}
}

func TestExecuteBootstrapReturnsOnlySafeCompositionFailures(t *testing.T) {
	t.Setenv("MEALSWAPP_CLP_VERSION", "invalid")
	if _, err := executeBootstrap(context.Background(), adminbootstrap.Request{}); err == nil || err.Error() != "configuration unavailable" {
		t.Fatalf("configuration error = %v", err)
	}

	t.Setenv("MEALSWAPP_CLP_VERSION", "1.17.11")
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "short")
	if _, err := executeBootstrap(context.Background(), adminbootstrap.Request{}); err == nil || err.Error() != "lookup configuration unavailable" {
		t.Fatalf("lookup configuration error = %v", err)
	}

	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")
	t.Setenv("MEALSWAPP_DATABASE_URL", "postgres://127.0.0.1:1/mealswapp?connect_timeout=1")
	if _, err := executeBootstrap(context.Background(), adminbootstrap.Request{}); err == nil || err.Error() != "database unavailable" {
		t.Fatalf("database error = %v", err)
	}
}

func TestExecuteBootstrapComposesSuccessfulRuntime(t *testing.T) {
	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatal(err)
	}
	db := testdatabase.Reset(t, migrationDir)
	databaseURL := os.Getenv(testdatabase.EnvironmentVariable)
	if databaseURL == "" {
		databaseURL = testdatabase.DefaultURL
	}
	t.Setenv("MEALSWAPP_DATABASE_URL", databaseURL)
	t.Setenv("MEALSWAPP_ENV", "development")
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")

	email := "Runtime.Admin@Example.TEST"
	loader, _ := newLookupKeyLoader("development")
	digest, err := security.NewLookupDigestService(loader).DigestForWrite(context.Background(), []byte(strings.ToLower(email)))
	if err != nil {
		t.Fatal(err)
	}
	var userID uuid.UUID
	if err := db.QueryRow(context.Background(), `
		INSERT INTO users (
			email_key_version, email_nonce, email_ciphertext,
			normalized_email_lookup_key_version, normalized_email_digest,
			email_verified, password_hash, password_salt
		)
		VALUES ('test-v1', '\x01', '\x01', $1, $2, true,
			'argon2id$v=19$m=19456,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA',
			'MDEyMzQ1Njc4OWFiY2RlZg')
		RETURNING id
	`, digest.KeyVersion, digest.Value).Scan(&userID); err != nil {
		t.Fatalf("create runtime bootstrap user: %v", err)
	}
	request := adminbootstrap.Request{TargetEnvironment: "development", Email: email, RequestID: uuid.NewString()}
	result, err := executeBootstrap(context.Background(), request)
	if err != nil || result.UserID != userID || result.AuditID == nil || *result.AuditID == uuid.Nil || result.Replayed {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	replay, err := executeBootstrap(context.Background(), request)
	if err != nil || !replay.Replayed || replay.UserID != userID {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
}

func TestExecuteBootstrapReindexesLegacyMixedCaseEmail(t *testing.T) {
	db, databaseURL := resetBootstrapCommandDatabase(t)
	t.Setenv("MEALSWAPP_DATABASE_URL", databaseURL)
	t.Setenv("MEALSWAPP_ENV", "development")
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")

	email := "Legacy.Admin@Example.TEST"
	loader, _ := newLookupKeyLoader("development")
	legacy, err := security.NewLookupDigestService(loader).DigestForWrite(context.Background(), []byte(email))
	if err != nil {
		t.Fatal(err)
	}
	userID := insertBootstrapCommandUser(t, db, legacy, bootstrapCommandValidHash, bootstrapCommandValidSalt)
	result, err := executeBootstrap(context.Background(), adminbootstrap.Request{TargetEnvironment: "development", Email: email, RequestID: uuid.NewString()})
	if err != nil || result.UserID != userID {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	canonical, _ := security.NewLookupDigestService(loader).DigestForWrite(context.Background(), []byte(strings.ToLower(email)))
	var stored string
	if err := db.QueryRow(context.Background(), `SELECT normalized_email_digest FROM users WHERE id = $1`, userID).Scan(&stored); err != nil || stored != canonical.Value {
		t.Fatalf("stored digest=%q want=%q err=%v", stored, canonical.Value, err)
	}
}

func TestExecuteBootstrapRefusesCanonicalEmailCollision(t *testing.T) {
	db, databaseURL := resetBootstrapCommandDatabase(t)
	t.Setenv("MEALSWAPP_DATABASE_URL", databaseURL)
	t.Setenv("MEALSWAPP_ENV", "development")
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")
	loader, _ := newLookupKeyLoader("development")
	digests := security.NewLookupDigestService(loader)
	canonical, _ := digests.DigestForWrite(context.Background(), []byte("collision@example.test"))
	legacy, _ := digests.DigestForWrite(context.Background(), []byte("Collision@Example.TEST"))
	insertBootstrapCommandUser(t, db, canonical, bootstrapCommandValidHash, bootstrapCommandValidSalt)
	insertBootstrapCommandUser(t, db, legacy, bootstrapCommandValidHash, bootstrapCommandValidSalt)

	_, err := executeBootstrap(context.Background(), adminbootstrap.Request{TargetEnvironment: "development", Email: "Collision@Example.TEST", RequestID: uuid.NewString()})
	if !errors.Is(err, repository.ErrCanonicalEmailCollision) {
		t.Fatalf("collision error=%v", err)
	}
	var administrators int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE role = 'admin'`).Scan(&administrators); err != nil || administrators != 0 {
		t.Fatalf("administrators=%d err=%v", administrators, err)
	}
}

func TestExecuteBootstrapRejectsMalformedPasswordCredential(t *testing.T) {
	db, databaseURL := resetBootstrapCommandDatabase(t)
	t.Setenv("MEALSWAPP_DATABASE_URL", databaseURL)
	t.Setenv("MEALSWAPP_ENV", "development")
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")
	loader, _ := newLookupKeyLoader("development")
	cases := []struct {
		name string
		hash string
		salt string
	}{
		{"malformed parameters", "argon2id$v=19$m=bad,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA", bootstrapCommandValidSalt},
		{"weak parameters", "argon2id$v=19$m=1,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA", bootstrapCommandValidSalt},
		{"invalid hash base64", "argon2id$v=19$m=19456,t=1,p=1$not base64", bootstrapCommandValidSalt},
		{"short hash", "argon2id$v=19$m=19456,t=1,p=1$c2hvcnQ", bootstrapCommandValidSalt},
		{"invalid salt base64", bootstrapCommandValidHash, "not base64"},
		{"short salt", bootstrapCommandValidHash, "c2hvcnQ"},
	}
	for index, tc := range cases {
		digest, _ := security.NewLookupDigestService(loader).DigestForWrite(context.Background(), []byte(fmt.Sprintf("malformed-%d@example.test", index)))
		userID := insertBootstrapCommandUser(t, db, digest, tc.hash, tc.salt)
		_, err := executeBootstrap(context.Background(), adminbootstrap.Request{TargetEnvironment: "development", UserID: &userID, RequestID: uuid.NewString()})
		if !errors.Is(err, repository.ErrAdministratorBootstrapNoCredential) {
			t.Fatalf("%s error=%v, want unusable credential", tc.name, err)
		}
	}
	hasher, err := auth.NewPasswordHasher(auth.PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12})
	if err != nil {
		t.Fatal(err)
	}
	validHash, validSalt, err := hasher.HashPassword("StrongerPassword1!")
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := security.NewLookupDigestService(loader).DigestForWrite(context.Background(), []byte("valid-generated@example.test"))
	userID := insertBootstrapCommandUser(t, db, digest, validHash, validSalt)
	result, err := executeBootstrap(context.Background(), adminbootstrap.Request{TargetEnvironment: "development", UserID: &userID, RequestID: uuid.NewString()})
	if err != nil || result.UserID != userID || result.AuditID == nil {
		t.Fatalf("generated credential result=%+v err=%v", result, err)
	}
}

const bootstrapCommandValidHash = "argon2id$v=19$m=19456,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA"
const bootstrapCommandValidSalt = "MDEyMzQ1Njc4OWFiY2RlZg"

func resetBootstrapCommandDatabase(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatal(err)
	}
	db := testdatabase.Reset(t, migrationDir)
	databaseURL := os.Getenv(testdatabase.EnvironmentVariable)
	if databaseURL == "" {
		databaseURL = testdatabase.DefaultURL
	}
	return db, databaseURL
}

func insertBootstrapCommandUser(t *testing.T, db *pgxpool.Pool, digest security.LookupDigest, hash string, salt string) uuid.UUID {
	t.Helper()
	var userID uuid.UUID
	if err := db.QueryRow(context.Background(), `
		INSERT INTO users (
			email_key_version, email_nonce, email_ciphertext,
			normalized_email_lookup_key_version, normalized_email_digest,
			email_verified, password_hash, password_salt
		)
		VALUES ('test-v1', '\x01', '\x01', $1, $2, true, $3, $4)
		RETURNING id
	`, digest.KeyVersion, digest.Value, hash, salt).Scan(&userID); err != nil {
		t.Fatalf("create bootstrap command user: %v", err)
	}
	return userID
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

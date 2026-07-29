package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const bootstrapTestHash = "argon2id$v=19$m=19456,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA"
const bootstrapRequestID = "27608020-0000-4000-8000-000000000001"

func bootstrapSelector(digest string) AdministratorBootstrapSelector {
	value := LookupDigest{KeyVersion: "lookup-v1", Value: digest}
	return AdministratorBootstrapSelector{EmailDigest: &value}
}

func validateBootstrapTestCredential(hash string, salt string) bool {
	return hash == bootstrapTestHash && salt == "MDEyMzQ1Njc4OWFiY2RlZg"
}

// Implements DESIGN-009 AdminController initial bootstrap, replay, and refusal behavior.
func TestPostgresAdministratorBootstrapLifecycle(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	targetID := createBootstrapUserOnDB(t, db, true, "target", true)
	repo := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential)

	result, err := repo.BootstrapAdministrator(ctx, bootstrapSelector("target"), bootstrapRequestID)
	if err != nil {
		t.Fatalf("BootstrapAdministrator() error = %v", err)
	}
	if result.UserID != targetID || result.AuditID == nil || *result.AuditID == uuid.Nil || result.AuditedAt.IsZero() || result.Replayed {
		t.Fatalf("result = %#v", result)
	}
	var role, actorKind, action string
	var actorID *uuid.UUID
	if err := db.QueryRow(ctx, `
		SELECT u.role, a.actor_kind, a.action, a.admin_user_id
		FROM users u
		JOIN admin_audit_entries a ON a.entity_id = u.id
		WHERE u.id = $1
	`, targetID).Scan(&role, &actorKind, &action, &actorID); err != nil {
		t.Fatalf("load bootstrap state: %v", err)
	}
	if role != "admin" || actorKind != "operator" || action != "bootstrap_administrator" || actorID != nil {
		t.Fatalf("role=%q actor=%q action=%q actorID=%v", role, actorKind, action, actorID)
	}
	readback, err := NewPostgresAdminImportAuditRepository(db).ListAuditForEntity(ctx, "user", targetID)
	if err != nil {
		t.Fatalf("ListAuditForEntity() bootstrap readback error = %v", err)
	}
	if len(readback) != 1 {
		t.Fatalf("ListAuditForEntity() bootstrap readback = %#v", readback)
	}
	if readback[0].ActorKind != AdminAuditActorOperator || readback[0].AdminUserID != nil || readback[0].Action != "bootstrap_administrator" {
		t.Fatalf("shared audit readback lost operator attribution: %#v", readback[0])
	}

	replay, err := repo.BootstrapAdministrator(ctx, bootstrapSelector("target"), bootstrapRequestID)
	if err != nil || !replay.Replayed || replay.UserID != targetID || replay.AuditID != nil {
		t.Fatalf("replay = %#v err=%v", replay, err)
	}
	var audits int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM admin_audit_entries WHERE action = 'bootstrap_administrator'`).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audits=%d err=%v", audits, err)
	}

	otherID := createBootstrapUserOnDB(t, db, true, "other", true)
	if _, err := repo.BootstrapAdministrator(ctx, AdministratorBootstrapSelector{UserID: &otherID}, bootstrapRequestID); !errors.Is(err, ErrAdministratorAlreadyExists) {
		t.Fatalf("other bootstrap error = %v", err)
	}
}

// Implements DESIGN-009 AdminController target existence and credential gates.
func TestPostgresAdministratorBootstrapRejectsInvalidTargets(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential)
	createBootstrapUserOnDB(t, db, false, "unverified", true)
	createBootstrapUserOnDB(t, db, true, "credentialless", false)

	for _, tc := range []struct {
		name     string
		selector AdministratorBootstrapSelector
		want     error
	}{
		{"missing", bootstrapSelector("missing"), ErrAdministratorBootstrapTargetNotFound},
		{"unverified", bootstrapSelector("unverified"), ErrAdministratorBootstrapUnverified},
		{"credentialless", bootstrapSelector("credentialless"), ErrAdministratorBootstrapNoCredential},
		{"empty", AdministratorBootstrapSelector{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.BootstrapAdministrator(ctx, tc.selector, bootstrapRequestID)
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if tc.want == nil && !IsKind(err, ErrorKindValidation) {
				t.Fatalf("error = %v, want validation", err)
			}
		})
	}
}

// Implements DESIGN-009 AdminController verified OAuth Login Method eligibility.
func TestPostgresAdministratorBootstrapAcceptsEncryptedOAuthCredential(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	targetID := createBootstrapUserOnDB(t, db, true, "oauth", false)
	if _, err := db.Exec(ctx, `
		INSERT INTO oauth_identities (
			user_id, provider, provider_user_id, email,
			provider_user_id_key_version, provider_user_id_nonce, provider_user_id_ciphertext,
			provider_user_id_lookup_key_version, provider_user_id_digest,
			email_key_version, email_nonce, email_ciphertext
		)
		VALUES ($1, 'google', 'legacy-required', 'legacy-required@example.test',
			'test-v1', '\x01', '\x01', 'lookup-v1', 'oauth-provider-digest',
			'test-v1', '\x01', '\x01')
	`, targetID); err != nil {
		t.Fatalf("create encrypted OAuth identity: %v", err)
	}
	result, err := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential).BootstrapAdministrator(ctx, bootstrapSelector("oauth"), bootstrapRequestID)
	if err != nil || result.UserID != targetID {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

// Implements DESIGN-009 AdminController legacy mixed-case digest reindexing.
func TestPostgresAdministratorBootstrapReindexesLegacyEmailDigest(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	targetID := createBootstrapUserOnDB(t, db, true, "legacy-mixed-case", true)
	canonical := LookupDigest{KeyVersion: "lookup-v1", Value: "canonical-lower-case"}
	legacy := LookupDigest{KeyVersion: "lookup-v1", Value: "legacy-mixed-case"}
	result, err := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential).BootstrapAdministrator(ctx, AdministratorBootstrapSelector{EmailDigest: &canonical, LegacyEmailDigest: &legacy}, bootstrapRequestID)
	if err != nil || result.UserID != targetID {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var stored string
	if err := db.QueryRow(ctx, `SELECT normalized_email_digest FROM users WHERE id = $1`, targetID).Scan(&stored); err != nil || stored != canonical.Value {
		t.Fatalf("stored digest=%q err=%v", stored, err)
	}
}

// Implements DESIGN-009 AdminController canonical email collision refusal.
func TestPostgresAdministratorBootstrapRejectsCanonicalEmailCollision(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	createBootstrapUserOnDB(t, db, true, "canonical-lower-case", true)
	createBootstrapUserOnDB(t, db, true, "legacy-mixed-case", true)
	canonical := LookupDigest{KeyVersion: "lookup-v1", Value: "canonical-lower-case"}
	legacy := LookupDigest{KeyVersion: "lookup-v1", Value: "legacy-mixed-case"}
	_, err := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential).BootstrapAdministrator(ctx, AdministratorBootstrapSelector{EmailDigest: &canonical, LegacyEmailDigest: &legacy}, bootstrapRequestID)
	if !errors.Is(err, ErrCanonicalEmailCollision) {
		t.Fatalf("error=%v, want canonical collision", err)
	}
	var administrators int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = 'admin'`).Scan(&administrators); err != nil || administrators != 0 {
		t.Fatalf("administrators=%d err=%v", administrators, err)
	}
}

// Implements DESIGN-009 AdminController serialized concurrent bootstrap.
func TestPostgresAdministratorBootstrapSerializesConcurrentAttempts(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	createBootstrapUserOnDB(t, db, true, "concurrent", true)
	repo := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential)

	results := make(chan AdministratorBootstrapResult, 2)
	errs := make(chan error, 2)
	var start sync.WaitGroup
	start.Add(2)
	for range 2 {
		go func() {
			start.Done()
			start.Wait()
			result, err := repo.BootstrapAdministrator(ctx, bootstrapSelector("concurrent"), uuid.NewString())
			results <- result
			errs <- err
		}()
	}
	var created, replayed int
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent error = %v", err)
		}
		if (<-results).Replayed {
			replayed++
		} else {
			created++
		}
	}
	if created != 1 || replayed != 1 {
		t.Fatalf("created=%d replayed=%d", created, replayed)
	}
	var audits int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM admin_audit_entries WHERE action = 'bootstrap_administrator'`).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audits=%d err=%v", audits, err)
	}
}

// Implements DESIGN-009 AdminController fail-closed audit transaction.
func TestPostgresAdministratorBootstrapRollsBackWhenAuditFails(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	targetID := createBootstrapUserOnDB(t, db, true, "rollback", true)
	if _, err := db.Exec(ctx, `
		CREATE OR REPLACE FUNCTION fail_bootstrap_audit() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.action = 'bootstrap_administrator' THEN RAISE EXCEPTION 'forced audit failure'; END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER fail_bootstrap_audit BEFORE INSERT ON admin_audit_entries
		FOR EACH ROW EXECUTE FUNCTION fail_bootstrap_audit();
	`); err != nil {
		t.Fatalf("create audit failure trigger: %v", err)
	}
	_, err := NewPostgresAdministratorBootstrapRepository(db, validateBootstrapTestCredential).BootstrapAdministrator(ctx, bootstrapSelector("rollback"), bootstrapRequestID)
	if !errors.Is(err, ErrAdminAuditPersistence) {
		t.Fatalf("bootstrap error = %v", err)
	}
	var role string
	if err := db.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, targetID).Scan(&role); err != nil || role != "user" {
		t.Fatalf("role=%q err=%v", role, err)
	}
}

func TestPostgresAdministratorBootstrapValidationAndDatabaseFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	digest := LookupDigest{KeyVersion: "lookup-v1", Value: "digest"}
	for _, selector := range []AdministratorBootstrapSelector{
		{},
		{UserID: &id, EmailDigest: &digest},
		{UserID: func() *uuid.UUID { value := uuid.Nil; return &value }()},
		{EmailDigest: &LookupDigest{}},
		{LegacyEmailDigest: &digest},
		{EmailDigest: &digest, LegacyEmailDigest: &digest},
	} {
		if _, err := NewPostgresAdministratorBootstrapRepository(&fakeSQLExecutor{}, validateBootstrapTestCredential).BootstrapAdministrator(ctx, selector, bootstrapRequestID); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("selector %#v error = %v", selector, err)
		}
	}
	if _, err := NewPostgresAdministratorBootstrapRepository(&fakeSQLExecutor{}, validateBootstrapTestCredential).BootstrapAdministrator(ctx, AdministratorBootstrapSelector{UserID: &id}, ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("empty request error = %v", err)
	}
	if _, err := NewPostgresAdministratorBootstrapRepository(&fakeSQLExecutor{}, nil).BootstrapAdministrator(ctx, AdministratorBootstrapSelector{UserID: &id}, bootstrapRequestID); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil validator error = %v", err)
	}

	dbErr := errors.New("database unavailable")
	hash, salt := bootstrapTestHash, "MDEyMzQ1Njc4OWFiY2RlZg"
	target := fakeRow{values: []any{id, UserRoleUser, true, &hash, &salt, false}}
	noAdmin := fakeRow{err: pgx.ErrNoRows}
	audit := fakeRow{values: []any{uuid.New(), time.Now()}}
	for _, tc := range []struct {
		name string
		tx   *fakeTx
	}{
		{"lock", &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErrs: []error{dbErr}}}},
		{"target", &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rowList: []fakeRow{{err: dbErr}}}}},
		{"existing", &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rowList: []fakeRow{target, {err: dbErr}}}}},
		{"promote", &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErrs: []error{nil, dbErr}, rowList: []fakeRow{target, noAdmin}}}},
		{"audit", &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rowList: []fakeRow{target, noAdmin, {err: dbErr}}}}},
		{"commit", &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rowList: []fakeRow{target, noAdmin, audit}}, commitErr: dbErr}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPostgresAdministratorBootstrapRepository(&fakeSQLExecutor{tx: tc.tx}, validateBootstrapTestCredential).BootstrapAdministrator(ctx, AdministratorBootstrapSelector{UserID: &id}, bootstrapRequestID)
			if err == nil {
				t.Fatal("error = nil")
			}
		})
	}
}

func createBootstrapUserOnDB(t *testing.T, db transactionalExecutor, verified bool, digest string, credential bool) uuid.UUID {
	t.Helper()
	var hash, salt any
	if credential {
		hash, salt = bootstrapTestHash, "MDEyMzQ1Njc4OWFiY2RlZg"
	}
	var id uuid.UUID
	if err := db.QueryRow(context.Background(), `
		INSERT INTO users (
			email_key_version, email_nonce, email_ciphertext,
			normalized_email_lookup_key_version, normalized_email_digest,
			email_verified, password_hash, password_salt
		)
		VALUES ('test-v1', '\x01', '\x01', 'lookup-v1', $1, $2, $3, $4)
		RETURNING id
	`, digest, verified, hash, salt).Scan(&id); err != nil {
		t.Fatalf("create bootstrap user: %v", err)
	}
	return id
}

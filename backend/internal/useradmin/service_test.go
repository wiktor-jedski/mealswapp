package useradmin

// Implements DESIGN-009 UserAdminPanel projection, authorization, decryption, pagination, and audit verification.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type memoryAdminUsers struct {
	records         []repository.AdminUserRecord
	recordsByDigest map[string][]repository.AdminUserRecord
	lookup          repository.AdminUserLookup
	lookupCalls     int
	reindexedUserID uuid.UUID
	reindexedDigest repository.LookupDigest
	reindexErr      error
	retry           repository.AdminDeletionRetry
	retryErr        error
	retryCalls      int
	retryUserID     uuid.UUID
	retryID         uuid.UUID
}

func (r *memoryAdminUsers) LookupAdminUsers(_ context.Context, lookup repository.AdminUserLookup) ([]repository.AdminUserRecord, error) {
	r.lookup, r.lookupCalls = lookup, r.lookupCalls+1
	if lookup.EmailDigest != nil && r.recordsByDigest != nil {
		return append([]repository.AdminUserRecord(nil), r.recordsByDigest[lookup.EmailDigest.Value]...), nil
	}
	return append([]repository.AdminUserRecord(nil), r.records...), nil
}

func (r *memoryAdminUsers) ReindexUserEmailDigest(_ context.Context, userID uuid.UUID, digest repository.LookupDigest) error {
	r.reindexedUserID, r.reindexedDigest = userID, digest
	return r.reindexErr
}

func (r *memoryAdminUsers) RetryAdminDeletion(_ context.Context, _ repository.AdminMutationExecutor, userID uuid.UUID, requestID uuid.UUID) (repository.AdminDeletionRetry, error) {
	r.retryCalls, r.retryUserID, r.retryID = r.retryCalls+1, userID, requestID
	return r.retry, r.retryErr
}

type noopAdminTx struct{}

func (noopAdminTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (noopAdminTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (noopAdminTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }

type recordingDecrypter struct {
	calls int
	err   error
}

func (d *recordingDecrypter) DecryptPII(_ context.Context, envelope security.EncryptionEnvelope) ([]byte, error) {
	d.calls++
	if d.err != nil {
		return nil, d.err
	}
	return append([]byte(nil), envelope.Ciphertext...), nil
}

type recordingDigester struct {
	input  []byte
	inputs [][]byte
	values map[string]string
	err    error
}

func (d *recordingDigester) DigestForWrite(_ context.Context, input []byte) (security.LookupDigest, error) {
	d.input = append([]byte(nil), input...)
	d.inputs = append(d.inputs, append([]byte(nil), input...))
	value := "safe-digest"
	if configured, ok := d.values[string(input)]; ok {
		value = configured
	}
	return security.LookupDigest{KeyVersion: "lookup-v1", Value: value}, d.err
}

type recordingLookupAudit struct {
	entries []repository.AdminAuditEntry
	err     error
}

func (a *recordingLookupAudit) PersistAuditEntry(_ context.Context, entry repository.AdminAuditEntry) (uuid.UUID, error) {
	if a.err != nil {
		return uuid.Nil, a.err
	}
	a.entries = append(a.entries, entry)
	return uuid.New(), nil
}

func TestLookupProjectsOnlyApprovedFieldsWithBoundedPaginationAndAudit(t *testing.T) {
	adminID := uuid.New()
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	requestedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	repo := &memoryAdminUsers{records: []repository.AdminUserRecord{
		{ID: ids[0], Email: repository.EncryptedField{KeyVersion: "v1", Ciphertext: []byte("one@example.test")}, EmailVerified: true, CreatedAt: requestedAt},
		{ID: ids[1], Email: repository.EncryptedField{KeyVersion: "v1", Ciphertext: []byte("two@example.test")}, CreatedAt: requestedAt, Deletion: &repository.AdminDeletionSummary{RequestID: uuid.New(), Status: "failed", FailureCategory: "permanent", RetryCount: 1, RequestedAt: requestedAt}},
		{ID: ids[2], Email: repository.EncryptedField{KeyVersion: "v1", Ciphertext: []byte("lookahead@example.test")}},
	}}
	decrypt := &recordingDecrypter{}
	audit := &recordingLookupAudit{}
	service := NewService(repo, audit, decrypt, &recordingDigester{})
	service.now = func() time.Time { return requestedAt }

	page, err := service.Lookup(context.Background(), Actor{UserID: adminID, Role: "admin", RequestID: "request-252"}, LookupRequest{Limit: 2})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(page.Users) != 2 || page.NextCursor == nil || *page.NextCursor != ids[1] || decrypt.calls != 2 {
		t.Fatalf("page=%+v decrypt calls=%d", page, decrypt.calls)
	}
	if page.Users[1].Deletion == nil || page.Users[1].Deletion.FailureCategory != "permanent" || page.Users[0].Email != "one@example.test" {
		t.Fatalf("approved projection mismatch: %+v", page.Users)
	}
	if repo.lookup.Limit != 3 || repo.lookup.UserID != nil || repo.lookup.EmailDigest != nil {
		t.Fatalf("repository lookup = %+v, want bounded lookahead", repo.lookup)
	}
	if len(audit.entries) != 1 || audit.entries[0].AdminUserID == nil || *audit.entries[0].AdminUserID != adminID || audit.entries[0].ActorKind != repository.AdminAuditActorAdministrator || audit.entries[0].Action != "lookup_users" || audit.entries[0].EntityID != nil || len(audit.entries[0].Before)+len(audit.entries[0].After) != 0 {
		t.Fatalf("lookup audit = %+v", audit.entries)
	}
	encoded := strings.Join([]string{page.Users[0].Email, page.Users[1].Email, page.Users[1].Deletion.Status, page.Users[1].Deletion.FailureCategory}, " ")
	for _, forbidden := range []string{"password", "token", "failure_reason", "next_attempt_at", "lease", "receipt"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("projection contains forbidden %q: %s", forbidden, encoded)
		}
	}
}

func TestLookupExactEmailNormalizesDigestAndAuditsEntity(t *testing.T) {
	userID := uuid.New()
	repo := &memoryAdminUsers{records: []repository.AdminUserRecord{{ID: userID, Email: repository.EncryptedField{KeyVersion: "v1", Ciphertext: []byte("user@example.com")}}}}
	digester := &recordingDigester{}
	audit := &recordingLookupAudit{}
	service := NewService(repo, audit, &recordingDecrypter{}, digester)

	page, err := service.Lookup(context.Background(), Actor{UserID: uuid.New(), Role: "admin", RequestID: "request-exact"}, LookupRequest{Email: " User@Example.com "})
	if err != nil || len(page.Users) != 1 {
		t.Fatalf("Lookup() page=%+v error=%v", page, err)
	}
	if len(digester.inputs) != 2 || string(digester.inputs[0]) != "user@example.com" || string(digester.inputs[1]) != "User@Example.com" || repo.lookup.EmailDigest == nil || repo.lookup.EmailDigest.Value != "safe-digest" || repo.lookup.Limit != 1 {
		t.Fatalf("exact lookup=%+v digest inputs=%q", repo.lookup, digester.inputs)
	}
	if len(audit.entries) != 1 || audit.entries[0].EntityID == nil || *audit.entries[0].EntityID != userID {
		t.Fatalf("exact audit = %+v", audit.entries)
	}
}

func TestLookupExactEmailReindexesLegacyMixedCaseDigest(t *testing.T) {
	userID := uuid.New()
	repo := &memoryAdminUsers{recordsByDigest: map[string][]repository.AdminUserRecord{
		"legacy-digest": {{ID: userID, Email: repository.EncryptedField{KeyVersion: "v1", Ciphertext: []byte("User@Example.com")}}},
	}}
	digester := &recordingDigester{values: map[string]string{"user@example.com": "canonical-digest", "User@Example.com": "legacy-digest"}}
	service := NewService(repo, &recordingLookupAudit{}, &recordingDecrypter{}, digester)

	page, err := service.Lookup(context.Background(), Actor{UserID: uuid.New(), Role: "admin", RequestID: "request-legacy"}, LookupRequest{Email: " User@Example.com "})
	if err != nil || len(page.Users) != 1 || page.Users[0].ID != userID {
		t.Fatalf("Lookup() page=%+v error=%v", page, err)
	}
	if repo.lookupCalls != 2 || repo.reindexedUserID != userID || repo.reindexedDigest.Value != "canonical-digest" {
		t.Fatalf("legacy resolution calls=%d user=%s digest=%+v", repo.lookupCalls, repo.reindexedUserID, repo.reindexedDigest)
	}
}

func TestLookupExactEmailRefusesCanonicalLegacyCollision(t *testing.T) {
	canonicalID, legacyID := uuid.New(), uuid.New()
	repo := &memoryAdminUsers{recordsByDigest: map[string][]repository.AdminUserRecord{
		"canonical-digest": {{ID: canonicalID}},
		"legacy-digest":    {{ID: legacyID}},
	}}
	digester := &recordingDigester{values: map[string]string{"user@example.com": "canonical-digest", "User@Example.com": "legacy-digest"}}
	service := NewService(repo, &recordingLookupAudit{}, &recordingDecrypter{}, digester)

	page, err := service.Lookup(context.Background(), Actor{UserID: uuid.New(), Role: "admin", RequestID: "request-collision"}, LookupRequest{Email: "User@Example.com"})
	if !errors.Is(err, repository.ErrCanonicalEmailCollision) || len(page.Users) != 0 || repo.reindexedUserID != uuid.Nil {
		t.Fatalf("Lookup() page=%+v error=%v reindexed=%s", page, err, repo.reindexedUserID)
	}
}

func TestLookupFailsClosedBeforeUnauthorizedDecryptionAndOnAuditFailure(t *testing.T) {
	repo := &memoryAdminUsers{records: []repository.AdminUserRecord{{ID: uuid.New(), Email: repository.EncryptedField{KeyVersion: "v1", Ciphertext: []byte("private@example.test")}}}}
	decrypt := &recordingDecrypter{}
	audit := &recordingLookupAudit{}
	service := NewService(repo, audit, decrypt, &recordingDigester{})

	for _, actor := range []Actor{{UserID: uuid.New(), Role: "user", RequestID: "request"}, {UserID: uuid.New(), Role: "admin"}, {Role: "admin", RequestID: "request"}} {
		if _, err := service.Lookup(context.Background(), actor, LookupRequest{}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("unauthorized Lookup() error = %v", err)
		}
	}
	if repo.lookupCalls != 0 || decrypt.calls != 0 {
		t.Fatalf("unauthorized boundary repository=%d decrypt=%d", repo.lookupCalls, decrypt.calls)
	}

	audit.err = errors.New("audit unavailable with private detail")
	page, err := service.Lookup(context.Background(), Actor{UserID: uuid.New(), Role: "admin", RequestID: "request"}, LookupRequest{})
	if err == nil || len(page.Users) != 0 || decrypt.calls != 1 {
		t.Fatalf("audit failure page=%+v err=%v decrypt=%d", page, err, decrypt.calls)
	}
}

func TestLookupRejectsUnboundedAndConflictingScopes(t *testing.T) {
	service := NewService(&memoryAdminUsers{}, &recordingLookupAudit{}, &recordingDecrypter{}, &recordingDigester{})
	actor := Actor{UserID: uuid.New(), Role: "admin", RequestID: "request"}
	id, cursor := uuid.New(), uuid.New()
	for _, request := range []LookupRequest{{Limit: -1}, {Limit: MaxPageSize + 1}, {UserID: &id, Email: "user@example.test"}, {UserID: &id, Cursor: &cursor}, {Email: "user@example.test", Cursor: &cursor}, {Email: "invalid"}} {
		if _, err := service.Lookup(context.Background(), actor, request); !repository.IsKind(err, repository.ErrorKindValidation) {
			t.Fatalf("Lookup(%+v) error = %v, want validation", request, err)
		}
	}
}

func TestRetryDeletionEnforcesAuthorizationScopeAndForwardsLegalClaim(t *testing.T) {
	userID, requestID := uuid.New(), uuid.New()
	repo := &memoryAdminUsers{retry: repository.AdminDeletionRetry{RequestID: requestID, FailureCategory: "unknown", RetryCount: 2}}
	service := NewService(repo, &recordingLookupAudit{}, &recordingDecrypter{}, &recordingDigester{})
	admin := Actor{UserID: uuid.New(), Role: "admin", RequestID: "request-retry"}

	result, err := service.RetryDeletion(context.Background(), admin, userID, requestID, noopAdminTx{})
	if err != nil || result.RequestID != requestID || result.FailureCategory != "unknown" || repo.retryCalls != 1 || repo.retryUserID != userID || repo.retryID != requestID {
		t.Fatalf("RetryDeletion() result=%+v err=%v repo=%+v", result, err, repo)
	}
	for _, actor := range []Actor{{UserID: uuid.New(), Role: "user", RequestID: "request"}, {UserID: uuid.New(), Role: "admin"}} {
		if _, err := service.RetryDeletion(context.Background(), actor, userID, requestID, noopAdminTx{}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("unauthorized retry error=%v", err)
		}
	}
	for _, input := range []struct {
		userID    uuid.UUID
		requestID uuid.UUID
		tx        repository.AdminMutationExecutor
	}{{requestID: requestID, tx: noopAdminTx{}}, {userID: userID, tx: noopAdminTx{}}, {userID: userID, requestID: requestID}} {
		if _, err := service.RetryDeletion(context.Background(), admin, input.userID, input.requestID, input.tx); !repository.IsKind(err, repository.ErrorKindValidation) {
			t.Fatalf("invalid retry error=%v", err)
		}
	}
	repo.retryErr = repository.NewError(repository.ErrorKindNotFound, "scope mismatch", nil)
	if _, err := service.RetryDeletion(context.Background(), admin, uuid.New(), requestID, noopAdminTx{}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-scope retry error=%v", err)
	}
}

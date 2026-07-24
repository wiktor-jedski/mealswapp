package repository

// Implements DESIGN-009 UserAdminPanel persistence, legal transition, scope, concurrency, and audit verification.

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAdminUserLookupIsExactBoundedAndPrivacyMinimized(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresAdminUserRepository(db)
	emails := []string{"lookup-a@example.test", "lookup-b@example.test", "lookup-c@example.test"}
	ids := make([]uuid.UUID, 0, len(emails))
	idsByEmail := make(map[string]uuid.UUID, len(emails))
	for _, email := range emails {
		id := createRepositoryUser(t, ctx, db, email)
		ids = append(ids, id)
		idsByEmail[email] = id
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

	page, err := repo.LookupAdminUsers(ctx, AdminUserLookup{Limit: 2})
	if err != nil || len(page) != 2 || page[0].ID != ids[0] || page[1].ID != ids[1] {
		t.Fatalf("first page=%+v err=%v", page, err)
	}
	next, err := repo.LookupAdminUsers(ctx, AdminUserLookup{AfterID: &page[1].ID, Limit: 2})
	if err != nil || len(next) != 1 || next[0].ID != ids[2] {
		t.Fatalf("next page=%+v err=%v", next, err)
	}
	exactID, err := repo.LookupAdminUsers(ctx, AdminUserLookup{UserID: &ids[1], Limit: 1})
	if err != nil || len(exactID) != 1 || exactID[0].ID != ids[1] {
		t.Fatalf("id lookup=%+v err=%v", exactID, err)
	}
	exactEmail, err := repo.LookupAdminUsers(ctx, AdminUserLookup{EmailDigest: &LookupDigest{KeyVersion: "test-v1", Value: emails[1]}, Limit: 1})
	if err != nil || len(exactEmail) != 1 || exactEmail[0].ID != idsByEmail[emails[1]] {
		t.Fatalf("email lookup=%+v err=%v", exactEmail, err)
	}
	if exactEmail[0].Email.KeyVersion != "test-v1" || string(exactEmail[0].Email.Ciphertext) != emails[1] {
		t.Fatalf("encrypted projection mismatch: %+v", exactEmail[0])
	}

	for _, lookup := range []AdminUserLookup{{Limit: 0}, {Limit: 27}, {UserID: &ids[0], EmailDigest: &LookupDigest{KeyVersion: "v", Value: "digest"}, Limit: 1}, {UserID: &ids[0], AfterID: &ids[1], Limit: 1}} {
		if _, err := repo.LookupAdminUsers(ctx, lookup); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("LookupAdminUsers(%+v) error=%v, want validation", lookup, err)
		}
	}
}

func TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	adminID := createRepositoryUser(t, ctx, db, "retry-admin@example.test")
	audit := NewPostgresAdminImportAuditRepository(db)

	cases := []struct {
		name       string
		category   string
		retryCount int
		wantOK     bool
	}{
		{name: "permanent", category: "permanent", wantOK: true},
		{name: "unknown", category: "unknown", wantOK: true},
		{name: "exhausted", category: "transient", retryCount: 3, wantOK: true},
		{name: "retryable transient", category: "transient", retryCount: 2},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID := createRepositoryUser(t, ctx, db, fmt.Sprintf("retry-%d@example.test", index))
			requestID := createFailedDeletionFixture(t, ctx, db, userID, tc.category, tc.retryCount)
			err := audit.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "retry_deletion", EntityType: "deletion_request", RequestID: uuid.NewString(), CreatedAt: time.Now()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
				retry, err := NewPostgresAdminUserRepository(tx).RetryAdminDeletion(ctx, tx, userID, requestID)
				if err != nil {
					return AdminAuditChanges{}, err
				}
				return AdminAuditChanges{EntityID: &retry.RequestID, Before: []byte(`{"status":"failed","failureCategory":"` + retry.FailureCategory + `"}`), After: []byte(`{"status":"pending"}`)}, nil
			})
			if tc.wantOK && err != nil {
				t.Fatalf("legal retry error=%v", err)
			}
			if !tc.wantOK && !IsKind(err, ErrorKindNotFound) {
				t.Fatalf("illegal retry error=%v, want not found", err)
			}
			var status string
			var category *string
			var retryCount int
			if err := db.QueryRow(ctx, "SELECT status, failure_category, retry_count FROM data_deletion_requests WHERE id = $1", requestID).Scan(&status, &category, &retryCount); err != nil {
				t.Fatal(err)
			}
			if tc.wantOK && (status != "pending" || category != nil || retryCount != 0) {
				t.Fatalf("retried state status=%s category=%v retry=%d", status, category, retryCount)
			}
			if !tc.wantOK && (status != "failed" || category == nil || *category != tc.category || retryCount != tc.retryCount) {
				t.Fatalf("rejected state changed status=%s category=%v retry=%d", status, category, retryCount)
			}
		})
	}

	ownerID := createRepositoryUser(t, ctx, db, "retry-owner@example.test")
	otherID := createRepositoryUser(t, ctx, db, "retry-other@example.test")
	requestID := createFailedDeletionFixture(t, ctx, db, ownerID, "permanent", 0)
	err := audit.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "retry_deletion", EntityType: "deletion_request", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		_, err := NewPostgresAdminUserRepository(tx).RetryAdminDeletion(ctx, tx, otherID, requestID)
		return AdminAuditChanges{}, err
	})
	if !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("cross-scope retry error=%v, want not found", err)
	}
}

// TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits verifies
// IT-ARCH-009-007, ARCH-009, DESIGN-009 UserAdminPanel, and SW-REQ-054/SW-REQ-073.
func TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	adminID := createRepositoryUser(t, ctx, db, "concurrent-admin@example.test")
	userID := createRepositoryUser(t, ctx, db, "concurrent-user@example.test")
	requestID := createFailedDeletionFixture(t, ctx, db, userID, "unknown", 0)
	audit := NewPostgresAdminImportAuditRepository(db)

	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			ready.Done()
			<-start
			results <- audit.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "retry_deletion", EntityType: "deletion_request", RequestID: uuid.NewString(), CreatedAt: time.Now()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
				retry, err := NewPostgresAdminUserRepository(tx).RetryAdminDeletion(ctx, tx, userID, requestID)
				if err != nil {
					return AdminAuditChanges{}, err
				}
				return AdminAuditChanges{EntityID: &retry.RequestID, Before: []byte(`{"status":"failed","failureCategory":"unknown"}`), After: []byte(`{"status":"pending"}`)}, nil
			})
		}()
	}
	ready.Wait()
	close(start)
	successes, misses := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case IsKind(err, ErrorKindNotFound):
			misses++
		default:
			t.Fatalf("concurrent retry error=%v", err)
		}
	}
	if successes != 1 || misses != 1 {
		t.Fatalf("concurrent outcomes success=%d miss=%d", successes, misses)
	}
	var deletionAudits, adminAudits int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM data_deletion_audit_entries WHERE request_id = $1 AND from_status = 'failed' AND to_status = 'pending' AND note = 'admin_retry'", requestID).Scan(&deletionAudits); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, "SELECT count(*) FROM admin_audit_entries WHERE entity_type = 'deletion_request' AND entity_id = $1 AND action = 'retry_deletion'", requestID).Scan(&adminAudits); err != nil {
		t.Fatal(err)
	}
	if deletionAudits != 1 || adminAudits != 1 {
		t.Fatalf("audit counts deletion=%d admin=%d", deletionAudits, adminAudits)
	}
}

func createFailedDeletionFixture(t *testing.T, ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, category string, retryCount int) uuid.UUID {
	t.Helper()
	request, err := NewPostgresComplianceRepository(db).RequestDeletion(ctx, userID)
	if err != nil {
		t.Fatalf("RequestDeletion() error=%v", err)
	}
	if _, err := db.Exec(ctx, "UPDATE data_deletion_requests SET status = 'failed', failure_category = $2, failure_reason = 'fixture_internal_detail', retry_count = $3, next_attempt_at = NULL WHERE id = $1", request.ID, category, retryCount); err != nil {
		t.Fatalf("seed failed deletion: %v", err)
	}
	return request.ID
}

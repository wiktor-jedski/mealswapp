package userdata

// Implements DESIGN-008 AccountDeleter verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryDeletionRepository struct {
	request   repository.DataDeletionRequest
	claimed   []repository.DataDeletionRequest
	failures  []string
	nextRetry *time.Time
	completed uuid.UUID
}

func (r *memoryDeletionRepository) RequestDeletion(context.Context, uuid.UUID) (repository.DataDeletionRequest, error) {
	return r.request, nil
}
func (r *memoryDeletionRepository) UpdateDeletionStatus(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (r *memoryDeletionRepository) ListDeletionAudit(context.Context, uuid.UUID) ([]repository.DataDeletionAuditEntry, error) {
	return nil, nil
}
func (r *memoryDeletionRepository) ClaimDeletionRequests(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error) {
	return r.claimed, nil
}
func (r *memoryDeletionRepository) RecordDeletionFailure(_ context.Context, _ uuid.UUID, category string, note string, nextAttemptAt *time.Time) error {
	r.failures = append(r.failures, category+":"+note)
	r.nextRetry = nextAttemptAt
	return nil
}
func (r *memoryDeletionRepository) CompleteDeletionRequest(_ context.Context, _ uuid.UUID, receiptID uuid.UUID, _ time.Time) error {
	r.completed = receiptID
	return nil
}

type memoryDeletionSessions struct {
	revoked uuid.UUID
}

func (s *memoryDeletionSessions) CreateSession(context.Context, repository.UserSession) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *memoryDeletionSessions) GetSessionByRefreshTokenHash(context.Context, string) (repository.UserSession, error) {
	return repository.UserSession{}, nil
}
func (s *memoryDeletionSessions) RevokeSession(context.Context, uuid.UUID) error       { return nil }
func (s *memoryDeletionSessions) RevokeSessionFamily(context.Context, uuid.UUID) error { return nil }
func (s *memoryDeletionSessions) RevokeUserSessions(_ context.Context, userID uuid.UUID) error {
	s.revoked = userID
	return nil
}

type memoryAccountDeletionRepository struct {
	deleted uuid.UUID
	err     error
}

func (r *memoryAccountDeletionRepository) DeleteUserAccount(_ context.Context, userID uuid.UUID) error {
	if r.err != nil {
		return r.err
	}
	r.deleted = userID
	return nil
}

type memoryCachePurger struct {
	err error
}

func (p memoryCachePurger) PurgeUser(context.Context, uuid.UUID) error { return p.err }

// TestAccountDeletionService verifies DESIGN-008 AccountDeleter service behavior.
func TestAccountDeletionService(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	requestID := uuid.New()
	receiptID := uuid.New()
	requests := &memoryDeletionRepository{request: repository.DataDeletionRequest{ID: requestID, UserID: userID, Status: "pending", RequestedAt: time.Now()}}
	sessions := &memoryDeletionSessions{}
	accounts := &memoryAccountDeletionRepository{}
	service := NewAccountDeletionService(requests, sessions, accounts, nil)
	request, err := service.RequestDeletion(ctx, userID)
	if err != nil || request.ID != requestID || sessions.revoked != userID {
		t.Fatalf("RequestDeletion() request=%#v err=%v revoked=%s", request, err, sessions.revoked)
	}
	if err := service.ExecuteDeletion(ctx, request, receiptID, time.Now()); err != nil {
		t.Fatalf("ExecuteDeletion() error = %v", err)
	}
	if accounts.deleted != userID || requests.completed != receiptID {
		t.Fatalf("execution deleted=%s completed=%s", accounts.deleted, requests.completed)
	}
	failingCache := NewAccountDeletionService(requests, sessions, accounts, memoryCachePurger{err: errors.New("cache down")})
	if err := failingCache.ExecuteDeletion(ctx, request, uuid.New(), time.Now()); err != nil {
		t.Fatalf("ExecuteDeletion() cache failure path error = %v", err)
	}
	if len(requests.failures) == 0 || requests.failures[len(requests.failures)-1] != "transient:cache_purge_failed" {
		t.Fatalf("cache failure metadata = %#v", requests.failures)
	}
}

// TestAccountDeletionServiceProcessesClaimedWork verifies DESIGN-015 retry orchestration.
func TestAccountDeletionServiceProcessesClaimedWork(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	request := repository.DataDeletionRequest{ID: uuid.New(), UserID: userID, Status: "processing", RetryCount: 1}
	requests := &memoryDeletionRepository{claimed: []repository.DataDeletionRequest{request}}
	accounts := &memoryAccountDeletionRepository{err: repository.NewError(repository.ErrorKindRetryable, "temporary", nil)}
	service := NewAccountDeletionService(requests, &memoryDeletionSessions{}, accounts, nil)
	claimed, err := service.ProcessDueDeletionRequests(ctx, now, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ProcessDueDeletionRequests() claimed=%#v err=%v", claimed, err)
	}
	if len(requests.failures) != 1 || requests.failures[0] != "transient:dependency_unavailable" {
		t.Fatalf("failure metadata = %#v", requests.failures)
	}
	if requests.nextRetry == nil || !requests.nextRetry.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("next retry = %v", requests.nextRetry)
	}

	requests.claimed = []repository.DataDeletionRequest{{ID: uuid.New(), Status: "processing"}}
	requests.failures = nil
	requests.nextRetry = nil
	accounts.err = nil
	if _, err := service.ProcessDueDeletionRequests(ctx, now, 1); err != nil {
		t.Fatalf("missing user processing error = %v", err)
	}
	if len(requests.failures) != 1 || requests.failures[0] != "permanent:missing_user_id" || requests.nextRetry != nil {
		t.Fatalf("missing user metadata = %#v retry=%v", requests.failures, requests.nextRetry)
	}
}

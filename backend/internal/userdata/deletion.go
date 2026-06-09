package userdata

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// CachePurger removes user-scoped cache keys during account deletion.
// Implements DESIGN-008 AccountDeleter.
type CachePurger interface {
	PurgeUser(context.Context, uuid.UUID) error
}

// AccountDeletionService owns account deletion requests and execution.
// Implements DESIGN-008 AccountDeleter.
type AccountDeletionService struct {
	requests repository.DeletionRequestRepository
	sessions repository.SessionRepository
	accounts repository.AccountDeletionRepository
	cache    CachePurger
}

// NewAccountDeletionService creates account deletion behavior.
// Implements DESIGN-008 AccountDeleter.
func NewAccountDeletionService(requests repository.DeletionRequestRepository, sessions repository.SessionRepository, accounts repository.AccountDeletionRepository, cache CachePurger) *AccountDeletionService {
	return &AccountDeletionService{requests: requests, sessions: sessions, accounts: accounts, cache: cache}
}

// RequestDeletion accepts an authenticated deletion request and revokes sessions.
// Implements DESIGN-008 AccountDeleter.
func (s *AccountDeletionService) RequestDeletion(ctx context.Context, userID uuid.UUID) (repository.DataDeletionRequest, error) {
	request, err := s.requests.RequestDeletion(ctx, userID)
	if err != nil {
		return repository.DataDeletionRequest{}, err
	}
	if err := s.sessions.RevokeUserSessions(ctx, userID); err != nil {
		return repository.DataDeletionRequest{}, err
	}
	return request, nil
}

// ExecuteDeletion deletes production account data and records a pseudonymous receipt.
// Implements DESIGN-008 AccountDeleter.
func (s *AccountDeletionService) ExecuteDeletion(ctx context.Context, request repository.DataDeletionRequest, receiptID uuid.UUID, completedAt time.Time) error {
	if err := s.sessions.RevokeUserSessions(ctx, request.UserID); err != nil {
		return err
	}
	if err := s.accounts.DeleteUserAccount(ctx, request.UserID); err != nil {
		return err
	}
	if s.cache != nil {
		if err := s.cache.PurgeUser(ctx, request.UserID); err != nil {
			return s.requests.RecordDeletionFailure(ctx, request.ID, "transient", "cache_purge_failed", nil)
		}
	}
	return s.requests.CompleteDeletionRequest(ctx, request.ID, receiptID, completedAt)
}

// ProcessDueDeletionRequests claims and executes due deletion work.
// Implements DESIGN-008 AccountDeleter and DESIGN-015 DataRetentionPolicy.
func (s *AccountDeletionService) ProcessDueDeletionRequests(ctx context.Context, now time.Time, limit int) ([]repository.DataDeletionRequest, error) {
	if now.IsZero() {
		now = time.Now()
	}
	claimed, err := s.requests.ClaimDeletionRequests(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	for _, request := range claimed {
		if request.UserID == uuid.Nil {
			if err := s.requests.RecordDeletionFailure(ctx, request.ID, "permanent", "missing_user_id", nil); err != nil {
				return claimed, err
			}
			continue
		}
		if err := s.ExecuteDeletion(ctx, request, uuid.New(), now); err != nil {
			category, note := classifyDeletionFailure(err)
			nextAttemptAt := nextDeletionAttempt(now, request.RetryCount, category)
			if err := s.requests.RecordDeletionFailure(ctx, request.ID, category, note, nextAttemptAt); err != nil {
				return claimed, err
			}
		}
	}
	return claimed, nil
}

// classifyDeletionFailure maps internal deletion failures to sanitized categories.
// Implements DESIGN-015 DataRetentionPolicy.
func classifyDeletionFailure(err error) (string, string) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "transient", "deadline_or_cancellation"
	}
	var repoErr *repository.Error
	if errors.As(err, &repoErr) {
		switch repoErr.Kind {
		case repository.ErrorKindConnection, repository.ErrorKindRetryable, repository.ErrorKindCanceled:
			return "transient", "dependency_unavailable"
		case repository.ErrorKindValidation, repository.ErrorKindConflict:
			return "permanent", "invalid_deletion_state"
		default:
			return "unknown", "repository_failure"
		}
	}
	return "unknown", "deletion_failed"
}

// nextDeletionAttempt schedules exponential retry only for non-exhausted transient failures.
// Implements DESIGN-015 DataRetentionPolicy.
func nextDeletionAttempt(now time.Time, retryCount int, category string) *time.Time {
	if category != "transient" || retryCount+1 >= 3 {
		return nil
	}
	delay := time.Duration(1<<retryCount) * time.Minute
	next := now.Add(delay)
	return &next
}

// Package useradmin implements restricted administrative user lookup and deletion retry.
package useradmin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// ErrForbidden identifies a caller that is not a verified administrator.
// Implements DESIGN-009 UserAdminPanel.
var ErrForbidden = errors.New("administrator access required")

// Implements DESIGN-009 UserAdminPanel bounded enumeration policy.
const (
	// DefaultPageSize is the user lookup page size used when no limit is supplied.
	DefaultPageSize = 20
	// MaxPageSize is the largest accepted user lookup page.
	MaxPageSize = 25
)

// Actor contains only server-derived authorization and correlation metadata.
// Implements DESIGN-009 UserAdminPanel.
type Actor struct {
	UserID    uuid.UUID
	Role      string
	RequestID string
}

// LookupRequest selects one exact account or one bounded page.
// Implements DESIGN-009 UserAdminPanel.
type LookupRequest struct {
	UserID *uuid.UUID
	Email  string
	Cursor *uuid.UUID
	Limit  int
}

// User is the approved privacy-minimized administration projection.
// Implements DESIGN-009 UserAdminPanel.
type User struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
	Deletion      *Deletion `json:"deletion,omitempty"`
}

// Deletion exposes workflow state without reasons, leases, receipts, or other internals.
// Implements DESIGN-009 UserAdminPanel.
type Deletion struct {
	RequestID       uuid.UUID `json:"requestId"`
	Status          string    `json:"status"`
	FailureCategory string    `json:"failureCategory,omitempty"`
	RetryCount      int       `json:"retryCount"`
	RequestedAt     time.Time `json:"requestedAt"`
}

// Page is one bounded deterministic user lookup page.
// Implements DESIGN-009 UserAdminPanel.
type Page struct {
	Users      []User     `json:"users"`
	NextCursor *uuid.UUID `json:"nextCursor,omitempty"`
}

// RetryResult contains safe retry response and fixed-code audit metadata.
// Implements DESIGN-009 UserAdminPanel.
type RetryResult struct {
	RequestID       uuid.UUID
	FailureCategory string
}

// piiDecrypter limits the service to decryption only.
// Implements DESIGN-009 UserAdminPanel authorized decryption boundary.
type piiDecrypter interface {
	DecryptPII(context.Context, security.EncryptionEnvelope) ([]byte, error)
}

// lookupDigester limits exact email search to keyed lookup derivation.
// Implements DESIGN-009 UserAdminPanel authorized lookup boundary.
type lookupDigester interface {
	DigestForWrite(context.Context, []byte) (security.LookupDigest, error)
}

// lookupAuditor persists a privacy-safe record before lookup data is released.
// Implements DESIGN-009 UserAdminPanel.
type lookupAuditor interface {
	PersistAuditEntry(context.Context, repository.AdminAuditEntry) (uuid.UUID, error)
}

// Service owns authorization, projection, bounded lookup, decryption, and retry policy.
// Implements DESIGN-009 UserAdminPanel.
type Service struct {
	users     repository.AdminUserRepository
	audit     lookupAuditor
	decrypter piiDecrypter
	digester  lookupDigester
	now       func() time.Time
}

// NewService creates restricted user-administration behavior.
// Implements DESIGN-009 UserAdminPanel.
func NewService(users repository.AdminUserRepository, audit lookupAuditor, decrypter piiDecrypter, digester lookupDigester) *Service {
	return &Service{users: users, audit: audit, decrypter: decrypter, digester: digester, now: time.Now}
}

// Lookup returns only the approved projection and audits before releasing plaintext.
// Implements DESIGN-009 UserAdminPanel.
func (s *Service) Lookup(ctx context.Context, actor Actor, request LookupRequest) (Page, error) {
	if err := authorize(actor); err != nil {
		return Page{}, err
	}
	if s.users == nil || s.audit == nil || s.decrypter == nil || s.digester == nil {
		return Page{}, repository.NewError(repository.ErrorKindConnection, "user administration dependency unavailable", nil)
	}
	lookup, legacyDigest, exact, err := s.lookupRequest(ctx, request)
	if err != nil {
		return Page{}, err
	}
	var records []repository.AdminUserRecord
	if lookup.EmailDigest != nil {
		record, resolveErr := repository.ResolveCanonicalEmailIdentity(
			ctx,
			*lookup.EmailDigest,
			legacyDigest,
			s.lookupUserByEmailDigest,
			func(record repository.AdminUserRecord) uuid.UUID { return record.ID },
			s.users.ReindexUserEmailDigest,
		)
		if resolveErr != nil {
			if !repository.IsKind(resolveErr, repository.ErrorKindNotFound) {
				return Page{}, resolveErr
			}
		} else {
			records = []repository.AdminUserRecord{record}
		}
	} else {
		records, err = s.users.LookupAdminUsers(ctx, lookup)
		if err != nil {
			return Page{}, err
		}
	}
	hasNext := !exact && len(records) > lookup.Limit-1
	if hasNext {
		records = records[:lookup.Limit-1]
	}
	users := make([]User, 0, len(records))
	for _, record := range records {
		plain, err := s.decrypter.DecryptPII(ctx, security.EncryptionEnvelope{KeyVersion: record.Email.KeyVersion, Nonce: record.Email.Nonce, Ciphertext: record.Email.Ciphertext})
		if err != nil {
			return Page{}, err
		}
		projected := User{ID: record.ID, Email: string(plain), EmailVerified: record.EmailVerified, CreatedAt: record.CreatedAt}
		if record.Deletion != nil {
			projected.Deletion = &Deletion{RequestID: record.Deletion.RequestID, Status: record.Deletion.Status, FailureCategory: record.Deletion.FailureCategory, RetryCount: record.Deletion.RetryCount, RequestedAt: record.Deletion.RequestedAt}
		}
		users = append(users, projected)
	}
	var entityID *uuid.UUID
	if exact && len(records) == 1 {
		id := records[0].ID
		entityID = &id
	}
	adminUserID := actor.UserID
	if _, err := s.audit.PersistAuditEntry(ctx, repository.AdminAuditEntry{ActorKind: repository.AdminAuditActorAdministrator, AdminUserID: &adminUserID, Action: "lookup_users", EntityType: "user", EntityID: entityID, RequestID: actor.RequestID, CreatedAt: s.now()}); err != nil {
		return Page{}, err
	}
	page := Page{Users: users}
	if hasNext && len(records) > 0 {
		cursor := records[len(records)-1].ID
		page.NextCursor = &cursor
	}
	return page, nil
}

// RetryDeletion claims one eligible failure inside the gateway audit transaction.
// Implements DESIGN-009 UserAdminPanel.
func (s *Service) RetryDeletion(ctx context.Context, actor Actor, userID uuid.UUID, requestID uuid.UUID, tx repository.AdminMutationExecutor) (RetryResult, error) {
	if err := authorize(actor); err != nil {
		return RetryResult{}, err
	}
	if s.users == nil || userID == uuid.Nil || requestID == uuid.Nil || tx == nil {
		return RetryResult{}, repository.NewError(repository.ErrorKindValidation, "scoped deletion retry is invalid", nil)
	}
	retry, err := s.users.RetryAdminDeletion(ctx, tx, userID, requestID)
	if err != nil {
		return RetryResult{}, err
	}
	return RetryResult{RequestID: retry.RequestID, FailureCategory: retry.FailureCategory}, nil
}

// lookupRequest normalizes selectors and adds one private lookahead row for pagination.
// Implements DESIGN-009 UserAdminPanel.
func (s *Service) lookupRequest(ctx context.Context, request LookupRequest) (repository.AdminUserLookup, *repository.LookupDigest, bool, error) {
	if request.UserID != nil && request.Email != "" || request.Cursor != nil && (request.UserID != nil || request.Email != "") {
		return repository.AdminUserLookup{}, nil, false, repository.NewError(repository.ErrorKindValidation, "user lookup scope is invalid", nil)
	}
	limit := request.Limit
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit < 1 || limit > MaxPageSize {
		return repository.AdminUserLookup{}, nil, false, repository.NewError(repository.ErrorKindValidation, "user lookup limit is invalid", nil)
	}
	lookup := repository.AdminUserLookup{UserID: request.UserID, AfterID: request.Cursor, Limit: limit + 1}
	exact := request.UserID != nil || request.Email != ""
	var legacyDigest *repository.LookupDigest
	if request.Email != "" {
		normalized, err := security.NormalizeInput(security.InputFieldEmail, request.Email)
		if err != nil {
			return repository.AdminUserLookup{}, nil, false, repository.NewError(repository.ErrorKindValidation, "user lookup email is invalid", nil)
		}
		digest, err := s.digester.DigestForWrite(ctx, []byte(normalized.Value))
		if err != nil {
			return repository.AdminUserLookup{}, nil, false, err
		}
		lookup.EmailDigest = &repository.LookupDigest{KeyVersion: digest.KeyVersion, Value: digest.Value}
		legacyValue := strings.TrimSpace(request.Email)
		if legacyValue != normalized.Value {
			legacy, err := s.digester.DigestForWrite(ctx, []byte(legacyValue))
			if err != nil {
				return repository.AdminUserLookup{}, nil, false, err
			}
			legacyDigest = &repository.LookupDigest{KeyVersion: legacy.KeyVersion, Value: legacy.Value}
		}
	}
	if exact {
		lookup.Limit = 1
	}
	return lookup, legacyDigest, exact, nil
}

// lookupUserByEmailDigest adapts exact administration lookup to the shared collision-safe resolver.
// Implements DESIGN-009 UserAdminPanel and DESIGN-013 InputNormalizer.
func (s *Service) lookupUserByEmailDigest(ctx context.Context, digest repository.LookupDigest) (repository.AdminUserRecord, error) {
	records, err := s.users.LookupAdminUsers(ctx, repository.AdminUserLookup{EmailDigest: &digest, Limit: 1})
	if err != nil {
		return repository.AdminUserRecord{}, err
	}
	if len(records) == 0 {
		return repository.AdminUserRecord{}, repository.NewError(repository.ErrorKindNotFound, "administrative user not found", nil)
	}
	return records[0], nil
}

// authorize enforces the service-level verified-admin boundary.
// Implements DESIGN-009 UserAdminPanel.
func authorize(actor Actor) error {
	if actor.UserID == uuid.Nil || actor.Role != string(repository.UserRoleAdmin) || actor.RequestID == "" {
		return ErrForbidden
	}
	return nil
}

package externaldata

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// RecordEvidenceTTL bounds how long an external-search result can authorize an import.
// Implements DESIGN-012 DataNormalizer stale provider-evidence rejection.
const RecordEvidenceTTL = 15 * time.Minute

// Implements DESIGN-012 DataNormalizer fail-closed external evidence.
var (
	// ErrRecordEvidenceInvalid identifies malformed, unknown, or expired record evidence.
	// Implements DESIGN-012 DataNormalizer fail-closed external evidence.
	ErrRecordEvidenceInvalid = errors.New("external record evidence is invalid")
	// ErrRecordEvidenceUnavailable identifies a shared evidence backend outage.
	// Implements DESIGN-012 DataNormalizer retryable evidence dependency failure.
	ErrRecordEvidenceUnavailable = errors.New("external record evidence is unavailable")
)

// recordEvidence binds a canonical server-selected identity to its expiry.
// Implements DESIGN-012 DataNormalizer short-lived selected-record evidence.
type recordEvidence struct {
	identity  providerregistry.Identity
	expiresAt time.Time
}

// RecordEvidenceBackend is the deployment-shared persistence seam for evidence.
// Implements DESIGN-012 DataNormalizer deployment-safe provenance coordination.
type RecordEvidenceBackend interface {
	StoreRecordEvidence(context.Context, string, string, string, time.Time) error
	ResolveRecordEvidence(context.Context, string, time.Time) (string, string, error)
}

// RecordEvidenceStore issues and resolves opaque references to server-normalized search records.
// Implements DESIGN-012 DataNormalizer trusted external provenance boundary.
type RecordEvidenceStore struct {
	mu       sync.Mutex
	records  map[string]recordEvidence
	now      func() time.Time
	ttl      time.Duration
	registry *providerregistry.Registry
	backend  RecordEvidenceBackend
}

// NewRecordEvidenceStore creates short-lived evidence storage, optionally backed by shared persistence.
// Implements DESIGN-012 DataNormalizer selected external-record boundary.
func NewRecordEvidenceStore(registry *providerregistry.Registry, backend ...RecordEvidenceBackend) *RecordEvidenceStore {
	store := &RecordEvidenceStore{records: map[string]recordEvidence{}, now: time.Now, ttl: RecordEvidenceTTL, registry: registry}
	if len(backend) > 0 {
		store.backend = backend[0]
	}
	return store
}

// Register records a canonical server-selected provider identity and returns an opaque token.
// Implements DESIGN-012 DataNormalizer exact selected-record provenance.
func (s *RecordEvidenceStore) Register(provider, externalID string) (string, error) {
	return s.RegisterContext(context.Background(), provider, externalID)
}

// RegisterContext records evidence in the deployment-shared backend when configured.
// Implements DESIGN-012 DataNormalizer exact selected-record provenance.
func (s *RecordEvidenceStore) RegisterContext(ctx context.Context, provider, externalID string) (string, error) {
	if s == nil || s.registry == nil {
		return "", ErrRecordEvidenceInvalid
	}
	identity, err := s.registry.Normalize(provider, externalID)
	if err != nil {
		return "", ErrRecordEvidenceInvalid
	}
	now := s.now()
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", ErrRecordEvidenceInvalid
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	if s.backend != nil {
		if err := s.backend.StoreRecordEvidence(ctx, token, identity.Provider, identity.ExternalID, now.Add(s.ttl)); err != nil {
			return "", classifyRecordEvidenceBackendError(err)
		}
		return token, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, evidence := range s.records {
		if !evidence.expiresAt.After(now) {
			delete(s.records, key)
		}
	}
	s.records[token] = recordEvidence{identity: identity, expiresAt: now.Add(s.ttl)}
	return token, nil
}

// Resolve returns only canonical identity from live server-issued evidence.
// Implements DESIGN-012 DataNormalizer malformed, unknown, and stale evidence rejection.
func (s *RecordEvidenceStore) Resolve(token string) (providerregistry.Identity, error) {
	return s.ResolveContext(context.Background(), token)
}

// ResolveContext resolves evidence from the deployment-shared backend when configured.
// Implements DESIGN-012 DataNormalizer stale and unknown evidence rejection.
func (s *RecordEvidenceStore) ResolveContext(ctx context.Context, token string) (providerregistry.Identity, error) {
	if s == nil || s.registry == nil || len(token) != 32 {
		return providerregistry.Identity{}, ErrRecordEvidenceInvalid
	}
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		return providerregistry.Identity{}, ErrRecordEvidenceInvalid
	}
	if s.backend != nil {
		provider, externalID, err := s.backend.ResolveRecordEvidence(ctx, token, s.now())
		if err != nil {
			return providerregistry.Identity{}, classifyRecordEvidenceBackendError(err)
		}
		identity, err := s.registry.Normalize(provider, externalID)
		if err != nil {
			return providerregistry.Identity{}, ErrRecordEvidenceInvalid
		}
		return identity, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	evidence, exists := s.records[token]
	if !exists || !evidence.expiresAt.After(s.now()) {
		delete(s.records, token)
		return providerregistry.Identity{}, ErrRecordEvidenceInvalid
	}
	identity, err := s.registry.Normalize(evidence.identity.Provider, evidence.identity.ExternalID)
	if err != nil || identity != evidence.identity {
		return providerregistry.Identity{}, ErrRecordEvidenceInvalid
	}
	return identity, nil
}

// classifyRecordEvidenceBackendError maps shared evidence dependency failures to API-safe classes.
// Implements DESIGN-012 DataNormalizer retryable shared-evidence dependency classification.
func classifyRecordEvidenceBackendError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if repository.IsKind(err, repository.ErrorKindConnection) ||
		repository.IsKind(err, repository.ErrorKindCanceled) ||
		repository.IsKind(err, repository.ErrorKindRetryable) ||
		repository.IsKind(err, repository.ErrorKindInternal) ||
		errors.Is(err, ErrRecordEvidenceUnavailable) {
		return ErrRecordEvidenceUnavailable
	}
	return ErrRecordEvidenceInvalid
}

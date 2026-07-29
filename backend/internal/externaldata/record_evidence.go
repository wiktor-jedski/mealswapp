package externaldata

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
)

// RecordEvidenceTTL bounds how long an external-search result can authorize an import.
// Implements DESIGN-012 DataNormalizer stale provider-evidence rejection.
const RecordEvidenceTTL = 15 * time.Minute

// Implements DESIGN-012 DataNormalizer fail-closed external evidence.
var (
	// ErrRecordEvidenceInvalid identifies malformed, unknown, or expired record evidence.
	// Implements DESIGN-012 DataNormalizer fail-closed external evidence.
	ErrRecordEvidenceInvalid = errors.New("external record evidence is invalid")
)

// recordEvidence binds a canonical server-selected identity to its expiry.
// Implements DESIGN-012 DataNormalizer short-lived selected-record evidence.
type recordEvidence struct {
	identity  providerregistry.Identity
	expiresAt time.Time
}

// RecordEvidenceStore issues and resolves opaque references to server-normalized search records.
// Implements DESIGN-012 DataNormalizer trusted external provenance boundary.
type RecordEvidenceStore struct {
	mu       sync.Mutex
	records  map[string]recordEvidence
	now      func() time.Time
	ttl      time.Duration
	registry *providerregistry.Registry
}

// NewRecordEvidenceStore creates a process-local, short-lived evidence store.
// Implements DESIGN-012 DataNormalizer selected external-record boundary.
func NewRecordEvidenceStore(registry *providerregistry.Registry) *RecordEvidenceStore {
	return &RecordEvidenceStore{records: map[string]recordEvidence{}, now: time.Now, ttl: RecordEvidenceTTL, registry: registry}
}

// Register records a canonical server-selected provider identity and returns an opaque token.
// Implements DESIGN-012 DataNormalizer exact selected-record provenance.
func (s *RecordEvidenceStore) Register(provider, externalID string) (string, error) {
	if s == nil || s.registry == nil {
		return "", ErrRecordEvidenceInvalid
	}
	identity, err := s.registry.Normalize(provider, externalID)
	if err != nil {
		return "", ErrRecordEvidenceInvalid
	}
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return "", ErrRecordEvidenceInvalid
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
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
	if s == nil || len(token) != 32 {
		return providerregistry.Identity{}, ErrRecordEvidenceInvalid
	}
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		return providerregistry.Identity{}, ErrRecordEvidenceInvalid
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

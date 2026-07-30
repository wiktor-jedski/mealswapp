package externaldata

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type evidenceBackendStub struct {
	provider, externalID, token string
	expiresAt                   time.Time
	err, storeErr               error
}

func (b *evidenceBackendStub) StoreRecordEvidence(_ context.Context, token, provider, externalID string, expiresAt time.Time) error {
	if b.storeErr != nil {
		return b.storeErr
	}
	b.token, b.provider, b.externalID, b.expiresAt = token, provider, externalID, expiresAt
	return nil
}

func (b *evidenceBackendStub) ResolveRecordEvidence(_ context.Context, token string, now time.Time) (string, string, error) {
	if b.err != nil {
		return "", "", b.err
	}
	if token != b.token || !b.expiresAt.After(now) {
		return "", "", ErrRecordEvidenceInvalid
	}
	return b.provider, b.externalID, nil
}

// Implements DESIGN-012 DataNormalizer retryable shared-evidence dependency verification.
func TestRecordEvidenceStoreClassifiesBackendOutageAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "database outage", err: repository.NewError(repository.ErrorKindConnection, "database down", nil), want: ErrRecordEvidenceUnavailable},
		{name: "database cancellation", err: repository.NewError(repository.ErrorKindCanceled, "query canceled", nil), want: ErrRecordEvidenceUnavailable},
		{name: "database retryable failure", err: repository.NewError(repository.ErrorKindRetryable, "serialization failure", nil), want: ErrRecordEvidenceUnavailable},
		{name: "cancellation", err: context.Canceled, want: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := NewRecordEvidenceStore(providerregistry.Default(), &evidenceBackendStub{token: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", err: test.err})
			if _, err := store.ResolveContext(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"); !errors.Is(err, test.want) {
				t.Fatalf("ResolveContext() error=%v, want %v", err, test.want)
			}
		})
	}
}

// Implements DESIGN-012 DataNormalizer retryable shared-evidence registration failure verification.
func TestRecordEvidenceStoreClassifiesRegistrationBackendFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "database outage", err: repository.NewError(repository.ErrorKindConnection, "database down", nil), want: ErrRecordEvidenceUnavailable},
		{name: "database cancellation", err: repository.NewError(repository.ErrorKindCanceled, "query canceled", nil), want: ErrRecordEvidenceUnavailable},
		{name: "database retryable failure", err: repository.NewError(repository.ErrorKindRetryable, "serialization failure", nil), want: ErrRecordEvidenceUnavailable},
		{name: "cancellation", err: context.Canceled, want: context.Canceled},
		{name: "invalid token state", err: errors.New("invalid evidence"), want: ErrRecordEvidenceInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := NewRecordEvidenceStore(providerregistry.Default(), &evidenceBackendStub{storeErr: test.err})
			if _, err := store.RegisterContext(context.Background(), "usda", "171265"); !errors.Is(err, test.want) {
				t.Fatalf("RegisterContext() error=%v, want %v", err, test.want)
			}
		})
	}
}

// Implements DESIGN-012 DataNormalizer trusted external provenance verification.
func TestRecordEvidenceStoreResolvesExactCanonicalIdentityAndRejectsInvalidEvidence(t *testing.T) {
	store := NewRecordEvidenceStore(providerregistry.Default())
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	token, err := store.Register(" USDA ", " 00171265 ")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := store.Resolve(token)
	if err != nil || identity != (providerregistry.Identity{Provider: "usda", ExternalID: "171265"}) {
		t.Fatalf("Resolve() = %#v, %v", identity, err)
	}
	for _, invalid := range []string{"", "malformed", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		if _, err := store.Resolve(invalid); err == nil {
			t.Fatalf("Resolve(%q) accepted invalid evidence", invalid)
		}
	}
	now = now.Add(RecordEvidenceTTL)
	if _, err := store.Resolve(token); err == nil {
		t.Fatal("expired evidence accepted")
	}
}

// Implements DESIGN-012 DataNormalizer deployment-shared evidence verification.
func TestRecordEvidenceStoreDelegatesToSharedBackend(t *testing.T) {
	backend := &evidenceBackendStub{}
	store := NewRecordEvidenceStore(providerregistry.Default(), backend)
	store.now = func() time.Time { return time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC) }
	token, err := store.RegisterContext(context.Background(), "usda", "00171265")
	if err != nil || token != backend.token || backend.provider != "usda" || backend.externalID != "171265" {
		t.Fatalf("shared registration token=%q backend=%+v err=%v", token, backend, err)
	}
	identity, err := store.ResolveContext(context.Background(), token)
	if err != nil || identity != (providerregistry.Identity{Provider: "usda", ExternalID: "171265"}) {
		t.Fatalf("shared resolution identity=%+v err=%v", identity, err)
	}
}

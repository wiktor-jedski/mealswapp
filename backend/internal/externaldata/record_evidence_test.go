package externaldata

import (
	"testing"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
)

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

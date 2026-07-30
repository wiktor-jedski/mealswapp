package repository

import (
	"context"
	"testing"
	"time"
)

// Implements DESIGN-012 DataNormalizer PostgreSQL evidence storage verification.
func TestPostgresRecordEvidenceStoreAndResolve(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresRecordEvidenceRepository(db)
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	token := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if err := repo.StoreRecordEvidence(ctx, token, "usda", "171265", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	provider, externalID, err := repo.ResolveRecordEvidence(ctx, token, now)
	if err != nil || provider != "usda" || externalID != "171265" {
		t.Fatalf("resolved evidence=(%q, %q), err=%v", provider, externalID, err)
	}
	if _, _, err := repo.ResolveRecordEvidence(ctx, token, now.Add(2*time.Minute)); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("expired evidence error=%v", err)
	}
}

// Implements DESIGN-012 DataNormalizer PostgreSQL evidence cleanup and retry verification.
func TestPostgresRecordEvidenceStoreCleansExpiredRowsAndPreservesRetryIdentity(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresRecordEvidenceRepository(db)
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	if err := repo.StoreRecordEvidence(ctx, "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "usda", "1", now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	token := "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"
	if err := repo.StoreRecordEvidence(ctx, token, "openfoodfacts", "001234", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.StoreRecordEvidence(ctx, token, "usda", "2", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	provider, externalID, err := repo.ResolveRecordEvidence(ctx, token, now)
	if err != nil || provider != "openfoodfacts" || externalID != "001234" {
		t.Fatalf("retry identity=(%q, %q), err=%v", provider, externalID, err)
	}
	if _, _, err := repo.ResolveRecordEvidence(ctx, "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", now); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("expired cleanup error=%v", err)
	}
}

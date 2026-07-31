package repository

import (
	"context"
	"testing"
	"time"
)

// TestPostgresMicronutrientUsageGuardSerializesItemWriteBoundary proves that
// the vocabulary guard and item-write boundary contend on the same lock.
// Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
func TestPostgresMicronutrientUsageGuardSerializesItemWriteBoundary(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	guard, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer guard.Rollback(ctx)
	if err := lockMicronutrientUsageTables(ctx, guard); err != nil {
		t.Fatalf("guard lock: %v", err)
	}

	write, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer write.Rollback(ctx)
	short, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	if err := lockMicronutrientItemWriteTables(short, write); err == nil {
		t.Fatal("item-write boundary acquired vocabulary guard lock concurrently")
	}
}

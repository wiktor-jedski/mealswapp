package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"mealswapp/backend/internal/domain/micronutrient"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMicronutrientVocabularyRepositoryValidatesSeededValues(t *testing.T) {
	databaseURL := os.Getenv("MEALSWAPP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MEALSWAPP_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool, "0005_micronutrient_vocabulary.down.sql")
	applyMigration(t, ctx, pool, "0005_micronutrient_vocabulary.up.sql")
	defer applyMigration(t, ctx, pool, "0005_micronutrient_vocabulary.down.sql")

	repo := NewMicronutrientVocabularyRepository(pool)
	active, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := micronutrient.ValidateKeys(map[string]float64{"Sodium": 25}, active); err != nil {
		t.Fatalf("expected seeded sodium key to be accepted, got %v", err)
	}

	if err := micronutrient.ValidateKeys(map[string]float64{"Na": 25}, active); !errors.Is(err, micronutrient.ErrUnknownKey) {
		t.Fatalf("expected alias Na to be rejected, got %v", err)
	}

	if err := repo.Upsert(ctx, micronutrient.Entry{Key: "BadUnit", DisplayName: "Bad Unit", Unit: "grams", Active: true}); !errors.Is(err, micronutrient.ErrInvalidUnit) {
		t.Fatalf("expected invalid unit error, got %v", err)
	}

	allowed, err := repo.IsAllowed(ctx, "Calcium")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected seeded calcium key to be allowed")
	}
}

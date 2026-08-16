package repository

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
)

// Implements DESIGN-009 TagManager allergen vocabulary verification.
func TestPostgresAllergenVocabularyRepositoryListsOnlyActiveEntries(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresAllergenVocabularyRepository(db)

	entries, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
		if entry.Name == "" || entry.LabelKey == "" {
			t.Fatalf("entry is not localized-label-ready: %#v", entry)
		}
	}
	want := []string{"animal_product", "dairy", "egg", "gluten", "meat", "peanut", "tree_nut"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("ListActive() keys = %#v, want %#v", keys, want)
	}

	if _, err := db.Exec(ctx, "UPDATE allergen_vocabulary SET deleted_at = now() WHERE key = $1", "dairy"); err != nil {
		t.Fatalf("deactivate allergen: %v", err)
	}
	entries, err = repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive() after deactivation error = %v", err)
	}
	for _, entry := range entries {
		if entry.Key == "dairy" {
			t.Fatal("ListActive() returned inactive dairy entry")
		}
	}

	if _, err := db.Exec(ctx, "UPDATE allergen_vocabulary SET deleted_at = now()"); err != nil {
		t.Fatalf("deactivate vocabulary: %v", err)
	}
	entries, err = repo.ListActive(ctx)
	if err != nil || entries == nil || len(entries) != 0 {
		t.Fatalf("ListActive() empty = %#v, %v; want non-nil empty slice", entries, err)
	}
}

// Implements DESIGN-009 TagManager inactive allergen state preservation across migration replay.
func TestAllergenVocabularyMigrationReplayPreservesInactiveDairy(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, "UPDATE allergen_vocabulary SET deleted_at = now() WHERE key = $1", "dairy"); err != nil {
		t.Fatalf("deactivate dairy: %v", err)
	}
	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatalf("resolve migration dir: %v", err)
	}
	if err := migrations.Run(ctx, db, "up", migrationDir); err != nil {
		t.Fatalf("repeat migrations up: %v", err)
	}

	var inactive bool
	if err := db.QueryRow(ctx, "SELECT deleted_at IS NOT NULL FROM allergen_vocabulary WHERE key = $1", "dairy").Scan(&inactive); err != nil {
		t.Fatalf("read dairy state: %v", err)
	}
	if !inactive {
		t.Fatal("dairy became active after repeat migrations up")
	}
	entries, err := NewPostgresAllergenVocabularyRepository(db).ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive() after repeat migrations up error = %v", err)
	}
	for _, entry := range entries {
		if entry.Key == "dairy" {
			t.Fatal("ListActive() returned dairy after repeat migrations up")
		}
	}
}

// Implements DESIGN-009 TagManager allergen dependency failure verification.
func TestPostgresAllergenVocabularyRepositoryClassifiesFailures(t *testing.T) {
	dependencyErr := errors.New("database details")
	repo := NewPostgresAllergenVocabularyRepository(&fakeSQLExecutor{queryErr: dependencyErr})
	if _, err := repo.ListActive(context.Background()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListActive() query error = %v, want connection", err)
	}
	repo = NewPostgresAllergenVocabularyRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: dependencyErr}})
	if _, err := repo.ListActive(context.Background()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListActive() scan error = %v, want connection", err)
	}
	repo = NewPostgresAllergenVocabularyRepository(&fakeSQLExecutor{rows: &fakeRows{err: dependencyErr}})
	if _, err := repo.ListActive(context.Background()); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("ListActive() iteration error = %v, want connection", err)
	}
}

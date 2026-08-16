package micronutrient

import (
	"context"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryRepository struct {
	entries map[string]repository.MicronutrientVocabularyEntry
	inUse   map[string]bool
}

func (r *memoryRepository) ListAll(context.Context) ([]repository.MicronutrientVocabularyEntry, error) {
	result := make([]repository.MicronutrientVocabularyEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, entry)
	}
	return result, nil
}
func (r *memoryRepository) Get(_ context.Context, key string) (repository.MicronutrientVocabularyEntry, error) {
	entry, ok := r.entries[key]
	if !ok {
		return entry, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return entry, nil
}
func (r *memoryRepository) Create(_ context.Context, entry repository.MicronutrientVocabularyEntry) (repository.MicronutrientVocabularyEntry, error) {
	if existing, ok := r.entries[entry.Key]; ok {
		if existing == entry {
			return existing, nil
		}
		return entry, repository.NewError(repository.ErrorKindConflict, "conflict", nil)
	}
	r.entries[entry.Key] = entry
	return entry, nil
}
func (r *memoryRepository) UpdateDisplayName(_ context.Context, key, value string) (repository.MicronutrientVocabularyEntry, error) {
	entry := r.entries[key]
	entry.DisplayName = value
	r.entries[key] = entry
	return entry, nil
}
func (r *memoryRepository) UpdateUnit(_ context.Context, key, value string) (repository.MicronutrientVocabularyEntry, error) {
	if r.inUse[key] {
		return repository.MicronutrientVocabularyEntry{}, repository.NewError(repository.ErrorKindConflict, "in use", nil)
	}
	entry := r.entries[key]
	entry.Unit = value
	r.entries[key] = entry
	return entry, nil
}
func (r *memoryRepository) SetActive(_ context.Context, key string, active bool) (repository.MicronutrientVocabularyEntry, error) {
	if !active && r.inUse[key] {
		return repository.MicronutrientVocabularyEntry{}, repository.NewError(repository.ErrorKindConflict, "in use", nil)
	}
	entry := r.entries[key]
	entry.Active = active
	r.entries[key] = entry
	return entry, nil
}

// TestServiceLifecycle proves immutable-key validation, exact replay, guarded changes, and reactivation.
// Implements DESIGN-005 MicronutrientVocabulary.
func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := &memoryRepository{entries: map[string]repository.MicronutrientVocabularyEntry{}, inUse: map[string]bool{}}
	service := NewService(repo)
	created, err := service.Create(ctx, repo, repository.MicronutrientVocabularyEntry{Key: "VitaminK", DisplayName: "Vitamin K", Unit: "mcg"})
	if err != nil || !created.Active {
		t.Fatalf("Create() = %+v, %v", created, err)
	}
	replayed, err := service.Create(ctx, repo, repository.MicronutrientVocabularyEntry{Key: "VitaminK", DisplayName: "Vitamin K", Unit: "mcg"})
	if err != nil || replayed != created || len(repo.entries) != 1 {
		t.Fatalf("exact replay = %+v, %v, count %d", replayed, err, len(repo.entries))
	}
	before, renamed, err := service.UpdateDisplayName(ctx, repo, "VitaminK", "Vitamin K1")
	if err != nil || before.Key != renamed.Key || renamed.DisplayName != "Vitamin K1" {
		t.Fatalf("UpdateDisplayName() = %+v %+v %v", before, renamed, err)
	}
	repo.inUse["VitaminK"] = true
	if _, _, err := service.UpdateUnit(ctx, repo, "VitaminK", "mg"); !repository.IsKind(err, repository.ErrorKindConflict) {
		t.Fatalf("in-use unit error = %v", err)
	}
	if _, _, err := service.SetActive(ctx, repo, "VitaminK", false); !repository.IsKind(err, repository.ErrorKindConflict) {
		t.Fatalf("in-use deactivate error = %v", err)
	}
	repo.inUse["VitaminK"] = false
	_, inactive, err := service.SetActive(ctx, repo, "VitaminK", false)
	if err != nil || inactive.Active {
		t.Fatalf("deactivate = %+v, %v", inactive, err)
	}
	_, active, err := service.SetActive(ctx, repo, "VitaminK", true)
	if err != nil || !active.Active {
		t.Fatalf("reactivate = %+v, %v", active, err)
	}
}

// TestValidationRejectsAliasesAndUnsupportedUnits proves the closed canonical contract.
func TestValidationRejectsAliasesAndUnsupportedUnits(t *testing.T) {
	for _, key := range []string{"Na", "sodium", "Vitamin C", ""} {
		if err := ValidateKey(key); err == nil {
			t.Fatalf("ValidateKey(%q) accepted", key)
		}
	}
	for _, unit := range []string{"µg", "kg", "MG", ""} {
		if err := ValidateUnit(unit); err == nil {
			t.Fatalf("ValidateUnit(%q) accepted", unit)
		}
	}
	for _, name := range []string{"", " Sodium", "Sodium\n"} {
		if err := ValidateDisplayName(name); err == nil {
			t.Fatalf("ValidateDisplayName(%q) accepted", name)
		}
	}
}

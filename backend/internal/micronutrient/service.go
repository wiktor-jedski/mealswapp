// Package micronutrient owns canonical micronutrient vocabulary administration.
package micronutrient

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-005 MicronutrientVocabulary canonical key validation.
var canonicalKey = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{2,119}$`)

// Service validates canonical vocabulary changes before persistence.
// Implements DESIGN-005 MicronutrientVocabulary administrator management.
type Service struct {
	reader repository.MicronutrientVocabularyAdminRepository
}

// NewService creates a canonical vocabulary manager.
// Implements DESIGN-005 MicronutrientVocabulary.
func NewService(reader repository.MicronutrientVocabularyAdminRepository) *Service {
	return &Service{reader: reader}
}

// List returns active and inactive canonical entries.
// Implements DESIGN-005 MicronutrientVocabulary administrator listing.
func (s *Service) List(ctx context.Context) ([]repository.MicronutrientVocabularyEntry, error) {
	return s.reader.ListAll(ctx)
}

// Create validates and creates one active canonical entry.
// Implements DESIGN-005 MicronutrientVocabulary retry-safe creation.
func (s *Service) Create(ctx context.Context, repo repository.MicronutrientVocabularyAdminRepository, entry repository.MicronutrientVocabularyEntry) (repository.MicronutrientVocabularyEntry, error) {
	if err := ValidateKey(entry.Key); err != nil {
		return repository.MicronutrientVocabularyEntry{}, err
	}
	if err := ValidateDisplayName(entry.DisplayName); err != nil {
		return repository.MicronutrientVocabularyEntry{}, err
	}
	if err := ValidateUnit(entry.Unit); err != nil {
		return repository.MicronutrientVocabularyEntry{}, err
	}
	entry.Active = true
	return repo.Create(ctx, entry)
}

// UpdateDisplayName changes only the display label and returns audit snapshots.
// Implements DESIGN-005 MicronutrientVocabulary immutable canonical keys.
func (s *Service) UpdateDisplayName(ctx context.Context, repo repository.MicronutrientVocabularyAdminRepository, key, displayName string) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
	if err := ValidateKey(key); err != nil {
		return repository.MicronutrientVocabularyEntry{}, repository.MicronutrientVocabularyEntry{}, err
	}
	if err := ValidateDisplayName(displayName); err != nil {
		return repository.MicronutrientVocabularyEntry{}, repository.MicronutrientVocabularyEntry{}, err
	}
	before, err := repo.Get(ctx, key)
	if err != nil {
		return before, repository.MicronutrientVocabularyEntry{}, err
	}
	after, err := repo.UpdateDisplayName(ctx, key, displayName)
	return before, after, err
}

// UpdateUnit changes an unused entry's canonical measurement unit.
// Implements DESIGN-005 MicronutrientVocabulary in-use safeguard.
func (s *Service) UpdateUnit(ctx context.Context, repo repository.MicronutrientVocabularyAdminRepository, key, unit string) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
	if err := ValidateKey(key); err != nil {
		return repository.MicronutrientVocabularyEntry{}, repository.MicronutrientVocabularyEntry{}, err
	}
	if err := ValidateUnit(unit); err != nil {
		return repository.MicronutrientVocabularyEntry{}, repository.MicronutrientVocabularyEntry{}, err
	}
	before, err := repo.Get(ctx, key)
	if err != nil {
		return before, repository.MicronutrientVocabularyEntry{}, err
	}
	after, err := repo.UpdateUnit(ctx, key, unit)
	return before, after, err
}

// SetActive deactivates unused entries or reactivates an existing entry.
// Implements DESIGN-005 MicronutrientVocabulary lifecycle.
func (s *Service) SetActive(ctx context.Context, repo repository.MicronutrientVocabularyAdminRepository, key string, active bool) (repository.MicronutrientVocabularyEntry, repository.MicronutrientVocabularyEntry, error) {
	if err := ValidateKey(key); err != nil {
		return repository.MicronutrientVocabularyEntry{}, repository.MicronutrientVocabularyEntry{}, err
	}
	before, err := repo.Get(ctx, key)
	if err != nil {
		return before, repository.MicronutrientVocabularyEntry{}, err
	}
	after, err := repo.SetActive(ctx, key, active)
	return before, after, err
}

// ValidateKey enforces immutable canonical-key syntax.
// Implements DESIGN-005 MicronutrientVocabulary canonical key validation.
func ValidateKey(value string) error {
	if !canonicalKey.MatchString(value) {
		return repository.NewError(repository.ErrorKindValidation, "invalid canonical micronutrient key", nil)
	}
	return nil
}

// ValidateDisplayName enforces a concise normalized human-readable label.
// Implements DESIGN-005 MicronutrientVocabulary display-name validation.
func ValidateDisplayName(value string) error {
	if value == "" || value != strings.TrimSpace(value) || len([]rune(value)) > 120 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return repository.NewError(repository.ErrorKindValidation, "invalid micronutrient display name", nil)
	}
	return nil
}

// ValidateUnit closes persisted units over the supported mass vocabulary.
// Implements DESIGN-005 MicronutrientVocabulary unit validation.
func ValidateUnit(value string) error {
	if value != "g" && value != "mg" && value != "mcg" {
		return repository.NewError(repository.ErrorKindValidation, "invalid micronutrient unit", nil)
	}
	return nil
}

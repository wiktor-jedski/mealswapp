package micronutrient

import (
	"errors"
	"testing"
)

func TestEntryValidationRejectsInvalidUnit(t *testing.T) {
	entry := Entry{Key: "Sodium", DisplayName: "Sodium", Unit: "grams", Active: true}

	if err := entry.Validate(); !errors.Is(err, ErrInvalidUnit) {
		t.Fatalf("expected invalid unit error, got %v", err)
	}
}

func TestValidateKeysAcceptsActiveVocabularyValues(t *testing.T) {
	vocabulary := []Entry{{Key: "Sodium", DisplayName: "Sodium", Unit: UnitMilligram, Active: true}}

	if err := ValidateKeys(map[string]float64{"Sodium": 10}, vocabulary); err != nil {
		t.Fatalf("expected known key to be accepted, got %v", err)
	}
}

func TestValidateKeysRejectsUnknownKeysAndInactiveEntries(t *testing.T) {
	vocabulary := []Entry{{Key: "Sodium", DisplayName: "Sodium", Unit: UnitMilligram, Active: false}}

	if err := ValidateKeys(map[string]float64{"Sodium": 10}, vocabulary); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("expected unknown key error for inactive entry, got %v", err)
	}
}

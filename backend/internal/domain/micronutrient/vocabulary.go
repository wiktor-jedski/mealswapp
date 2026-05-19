package micronutrient

import (
	"errors"
	"strings"
)

type Unit string

const (
	UnitMilligram         Unit = "mg"
	UnitMicrogram         Unit = "mcg"
	UnitInternationalUnit Unit = "IU"
)

type Entry struct {
	Key         string
	DisplayName string
	Unit        Unit
	Active      bool
}

var (
	ErrMissingKey         = errors.New("micronutrient key is required")
	ErrMissingDisplayName = errors.New("micronutrient display name is required")
	ErrInvalidUnit        = errors.New("invalid micronutrient unit")
	ErrUnknownKey         = errors.New("unknown micronutrient key")
)

func (entry Entry) Validate() error {
	if strings.TrimSpace(entry.Key) == "" {
		return ErrMissingKey
	}
	if strings.TrimSpace(entry.DisplayName) == "" {
		return ErrMissingDisplayName
	}
	if !entry.Unit.Valid() {
		return ErrInvalidUnit
	}

	return nil
}

func (unit Unit) Valid() bool {
	switch unit {
	case UnitMilligram, UnitMicrogram, UnitInternationalUnit:
		return true
	default:
		return false
	}
}

func ValidateKeys(values map[string]float64, vocabulary []Entry) error {
	allowed := make(map[string]struct{}, len(vocabulary))
	for _, entry := range vocabulary {
		if entry.Active {
			allowed[entry.Key] = struct{}{}
		}
	}

	for key := range values {
		if _, ok := allowed[key]; !ok {
			return ErrUnknownKey
		}
	}

	return nil
}

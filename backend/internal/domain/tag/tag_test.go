package tag

import (
	"errors"
	"testing"
)

func TestTagValidationAcceptsSupportedKinds(t *testing.T) {
	for _, kind := range []Kind{KindDiet, KindAllergen, KindFunctionality, KindCuration} {
		entity := TagEntity{Name: "High protein", Kind: kind}
		if err := entity.Validate(); err != nil {
			t.Fatalf("expected %s to be valid, got %v", kind, err)
		}
	}
}

func TestTagValidationRejectsMissingName(t *testing.T) {
	entity := TagEntity{Name: " ", Kind: KindDiet}

	if err := entity.Validate(); !errors.Is(err, ErrMissingName) {
		t.Fatalf("expected missing name error, got %v", err)
	}
}

func TestTagValidationRejectsInvalidKind(t *testing.T) {
	entity := TagEntity{Name: "Unknown", Kind: "category"}

	if err := entity.Validate(); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected invalid kind error, got %v", err)
	}
}

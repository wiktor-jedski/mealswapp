package tag

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindDiet          Kind = "diet"
	KindAllergen      Kind = "allergen"
	KindFunctionality Kind = "functionality"
	KindCuration      Kind = "curation"
)

type TagEntity struct {
	ID        uuid.UUID
	Name      string
	Kind      Kind
	ParentID  *uuid.UUID
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrMissingName = errors.New("tag name is required")
	ErrInvalidKind = errors.New("unsupported tag kind")
)

func (tag TagEntity) Validate() error {
	if strings.TrimSpace(tag.Name) == "" {
		return ErrMissingName
	}

	if !tag.Kind.Valid() {
		return ErrInvalidKind
	}

	return nil
}

func (kind Kind) Valid() bool {
	switch kind {
	case KindDiet, KindAllergen, KindFunctionality, KindCuration:
		return true
	default:
		return false
	}
}

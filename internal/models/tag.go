// Phase: phase-01 | Task: 1 | Architecture: ARCH-005 | Design: TagEntity

package models

import (
	"time"
)

type TagType string

const (
	TagTypeCategory      TagType = "category"
	TagTypeFunctionality TagType = "functionality"
)

type Tag struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Type        TagType   `json:"type" db:"type"`
	Description string    `json:"description" db:"description"`
	ColorHex    string    `json:"color_hex" db:"color_hex"`
	IconURL     string    `json:"icon_url" db:"icon_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type TagCreateInput struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Type        TagType `json:"type" validate:"required,oneof=category functionality"`
	Description string  `json:"description" validate:"max=500"`
	ColorHex    string  `json:"color_hex" validate:"omitempty,hexcolor|len=7"`
	IconURL     string  `json:"icon_url" validate:"omitempty,url"`
}

type TagUpdateInput struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	ColorHex    *string `json:"color_hex,omitempty" validate:"omitempty,hexcolor|len=7"`
	IconURL     *string `json:"icon_url,omitempty" validate:"omitempty,url"`
}

type TagFilter struct {
	Types    []TagType `json:"types,omitempty"`
	Search   string    `json:"search,omitempty"`
	Slug     string    `json:"slug,omitempty"`
	Limit    int       `json:"limit,omitempty"`
	Offset   int       `json:"offset,omitempty"`
	OrderBy  string    `json:"order_by,omitempty"`
	OrderDir string    `json:"order_dir,omitempty"`
}

type TagListResult struct {
	Tags    []Tag `json:"tags"`
	Total   int   `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"has_more"`
}

type TagValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

const (
	ErrCodeTagNotFound      = "TAG_NOT_FOUND"
	ErrCodeTagAlreadyExists = "TAG_ALREADY_EXISTS"
	ErrCodeTagInvalidInput  = "TAG_INVALID_INPUT"
	ErrCodeTagInUse         = "TAG_IN_USE"
	ErrCodeTagDatabaseError = "TAG_DATABASE_ERROR"
	ErrCodeTagValidation    = "TAG_VALIDATION_ERROR"
)

package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/curation"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// Implements DESIGN-009 normalized curation request handoff.
const (
	normalizedExternalSearchLocal = "curation.normalizedExternalSearch"
	normalizedItemLocal           = "curation.normalizedItem"
	normalizedClassificationLocal = "curation.normalizedClassification"
)

// CurationRequestValidator validates Phase 08 curation inputs before dispatch.
// Implements DESIGN-010 RequestValidator and DESIGN-013 InputNormalizer.
type CurationRequestValidator struct {
	normalizer *curation.InputNormalizer
}

// NewCurationRequestValidator creates reusable HTTP validation with metadata-only logs.
// Implements DESIGN-010 RequestValidator and DESIGN-013 InputNormalizer.
func NewCurationRequestValidator(logs observability.LogSink) *CurationRequestValidator {
	return &CurationRequestValidator{normalizer: curation.NewInputNormalizer(logs)}
}

// ValidateExternalSearchQuery rejects ambiguous query input and stores its normalized typed value.
// Implements DESIGN-009 ExternalSearchProxy and DESIGN-013 InputNormalizer.
func (v *CurationRequestValidator) ValidateExternalSearchQuery(ctx *fiber.Ctx) error {
	values := map[string]string{}
	duplicate := false
	ctx.Context().QueryArgs().VisitAll(func(key []byte, value []byte) {
		name := string(key)
		if _, exists := values[name]; exists {
			duplicate = true
		}
		values[name] = string(value)
	})
	if duplicate || len(values) != 3 {
		v.inputNormalizer().RecordRejection(ctx.UserContext(), curation.RejectionFieldExternalSearchQuery)
		return curationValidationError()
	}
	page, err := strconv.Atoi(values["page"])
	if err != nil {
		v.inputNormalizer().RecordRejection(ctx.UserContext(), curation.RejectionFieldPagination)
		return curationValidationError()
	}
	normalized, err := v.inputNormalizer().NormalizeExternalSearch(ctx.UserContext(), curation.ExternalSearchRequest{
		Query: values["query"], Provider: values["provider"], Page: page,
	})
	if err != nil {
		return curationValidationError()
	}
	ctx.Locals(normalizedExternalSearchLocal, normalized)
	return ctx.Next()
}

// ValidateItemBody strictly decodes an item and stores its normalized typed value.
// Implements DESIGN-009 DataImporter and ItemCurator and DESIGN-013 InputNormalizer.
func (v *CurationRequestValidator) ValidateItemBody(ctx *fiber.Ctx) error {
	var req curation.ItemRequest
	if err := decodeStrictBody(ctx.Body(), &req); err != nil || validateRequiredMacros(ctx.Body()) != nil {
		v.inputNormalizer().RecordRejection(ctx.UserContext(), curation.RejectionFieldItemBody)
		return curationValidationError()
	}
	normalized, err := v.inputNormalizer().NormalizeItem(ctx.UserContext(), req)
	if err != nil {
		return curationValidationError()
	}
	ctx.Locals(normalizedItemLocal, normalized)
	return ctx.Next()
}

// ValidateClassificationBody strictly decodes a classification and stores its normalized typed value.
// Implements DESIGN-009 TagManager and DESIGN-013 InputNormalizer.
func (v *CurationRequestValidator) ValidateClassificationBody(ctx *fiber.Ctx) error {
	var req curation.ClassificationRequest
	if err := decodeStrictBody(ctx.Body(), &req); err != nil {
		v.inputNormalizer().RecordRejection(ctx.UserContext(), curation.RejectionFieldClassificationBody)
		return curationValidationError()
	}
	normalized, err := v.inputNormalizer().NormalizeClassification(ctx.UserContext(), req)
	if err != nil {
		return curationValidationError()
	}
	ctx.Locals(normalizedClassificationLocal, normalized)
	return ctx.Next()
}

// NormalizedExternalSearchRequest returns the only search value approved for provider dispatch.
// Implements DESIGN-009 ExternalSearchProxy typed handoff.
func NormalizedExternalSearchRequest(ctx *fiber.Ctx) (curation.ExternalSearchRequest, bool) {
	req, ok := ctx.Locals(normalizedExternalSearchLocal).(curation.ExternalSearchRequest)
	return req, ok
}

// NormalizedCurationItemRequest returns the only item value approved for repository dispatch.
// Implements DESIGN-009 DataImporter and ItemCurator typed handoff.
func NormalizedCurationItemRequest(ctx *fiber.Ctx) (curation.ItemRequest, bool) {
	req, ok := ctx.Locals(normalizedItemLocal).(curation.ItemRequest)
	return req, ok
}

// NormalizedCurationClassificationRequest returns the only classification value approved for repository dispatch.
// Implements DESIGN-009 TagManager typed handoff.
func NormalizedCurationClassificationRequest(ctx *fiber.Ctx) (curation.ClassificationRequest, bool) {
	req, ok := ctx.Locals(normalizedClassificationLocal).(curation.ClassificationRequest)
	return req, ok
}

// inputNormalizer supplies a no-log normalizer for a nil validator.
// Implements DESIGN-013 InputNormalizer defensive HTTP composition.
func (v *CurationRequestValidator) inputNormalizer() *curation.InputNormalizer {
	if v == nil || v.normalizer == nil {
		return curation.NewInputNormalizer(nil)
	}
	return v.normalizer
}

// curationValidationError returns the generic HTTP boundary error.
// Implements DESIGN-010 RequestValidator structured validation failures.
func curationValidationError() AppError {
	return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed"}
}

// decodeStrictBody rejects non-objects, malformed UTF-8, duplicate/unknown fields, and type mismatches.
// Implements DESIGN-010 RequestValidator typed curation request decoding.
func decodeStrictBody(body []byte, target any) error {
	trimmed := bytes.TrimSpace(body)
	if !utf8.Valid(body) || len(trimmed) == 0 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return errors.New("request body must be one UTF-8 JSON object")
	}
	if err := rejectDuplicateJSONKeys(trimmed); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body contains trailing data")
	}
	return nil
}

// validateRequiredMacros rejects null, missing, or non-object macro fields before typed dispatch.
// Implements DESIGN-005 MacroValues and DESIGN-013 InputNormalizer.
func validateRequiredMacros(body []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return err
	}
	raw, ok := object["macrosPer100"]
	trimmed := bytes.TrimSpace(raw)
	if !ok || len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("macrosPer100 must be an object")
	}
	var macros map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &macros); err != nil {
		return err
	}
	for _, name := range []string{"protein", "carbohydrates", "fat"} {
		value, exists := macros[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return errors.New("macro fields are required")
		}
	}
	return nil
}

// rejectDuplicateJSONKeys rejects ambiguity at every JSON object nesting level.
// Implements DESIGN-010 RequestValidator strict JSON decoding.
func rejectDuplicateJSONKeys(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("request body contains trailing data")
	}
	return nil
}

// scanJSONValue recursively consumes one value while tracking each object's keys.
// Implements DESIGN-010 RequestValidator strict JSON decoding.
func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is invalid")
			}
			if _, duplicate := seen[key]; duplicate {
				return errors.New("request body contains a duplicate field")
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return errors.New("request body contains an invalid JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}

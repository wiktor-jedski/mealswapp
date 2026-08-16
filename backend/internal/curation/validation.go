// Package curation owns reusable validation contracts for administrator curation flows.
package curation

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// ExternalSearchRequest is the normalized input shared by external food providers.
// Implements DESIGN-012 USDAClient and OpenFoodFactsClient request boundary.
type ExternalSearchRequest struct {
	Query    string `json:"query"`
	Provider string `json:"provider"`
	Page     int    `json:"page"`
}

// ItemRequest contains provider and administrator-editable item fields validated before dispatch.
// Implements DESIGN-009 DataImporter and ItemCurator request boundary.
type ItemRequest struct {
	Name                            string                   `json:"name"`
	PhysicalState                   repository.PhysicalState `json:"physicalState"`
	PrepTimeMinutes                 int                      `json:"prepTimeMinutes,omitempty"`
	AverageUnitWeightGrams          float64                  `json:"averageUnitWeightGrams,omitempty"`
	AverageServingVolumeMilliliters float64                  `json:"averageServingVolumeMilliliters,omitempty"`
	DensityGramsPerMilliliter       float64                  `json:"densityGramsPerMilliliter,omitempty"`
	DensitySourceProvider           string                   `json:"densitySourceProvider,omitempty"`
	DensitySourceFoodID             string                   `json:"densitySourceFoodId,omitempty"`
	DensitySourceKind               string                   `json:"densitySourceKind,omitempty"`
	ImageURL                        string                   `json:"imageUrl,omitempty"`
	ServingUnit                     string                   `json:"servingUnit,omitempty"`
	ServingQuantity                 float64                  `json:"servingQuantity,omitempty"`
	SourceProvider                  string                   `json:"sourceProvider,omitempty"`
	ExternalID                      string                   `json:"externalId,omitempty"`
	ProviderText                    string                   `json:"providerText,omitempty"`
	MacrosPer100                    repository.MacroValues   `json:"macrosPer100"`
	Micronutrients                  repository.MicroValues   `json:"micronutrients"`
}

// ClassificationRequest is the normalized administrator-authored classification input.
// Implements DESIGN-009 TagManager request boundary.
type ClassificationRequest struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parentId,omitempty"`
}

// RejectionField is the closed metadata vocabulary for pre-normalization decoding failures.
// Implements DESIGN-013 InputNormalizer metadata-only logging.
type RejectionField string

// Implements DESIGN-005 persisted curation bounds and DESIGN-013 bounded input validation.
const (
	// MaxCurationNutritionValue is the largest value accepted by persisted numeric(12,4) nutrition fields.
	// Implements DESIGN-005 FoodItemEntity and DESIGN-013 InputNormalizer.
	MaxCurationNutritionValue = 99_999_999.9999
	// MaxCurationServingQuantity matches the documented client quantity ceiling.
	// Implements DESIGN-004 quantity bounds and DESIGN-013 InputNormalizer.
	MaxCurationServingQuantity = 1_000_000
)

// Implements DESIGN-013 InputNormalizer metadata-only decoding failure categories.
const (
	RejectionFieldExternalSearchQuery RejectionField = "external_search_query"
	RejectionFieldPagination          RejectionField = "pagination"
	RejectionFieldItemBody            RejectionField = "curation_item_body"
	RejectionFieldClassificationBody  RejectionField = "curation_classification_body"
)

// InputNormalizer applies typed curation rules and emits only field-level metadata.
// Implements DESIGN-013 InputNormalizer normalized/input_rejected metadata policy.
type InputNormalizer struct {
	logs observability.LogSink
}

// NewInputNormalizer creates a curation normalizer with an optional structured log sink.
// Implements DESIGN-013 InputNormalizer metadata-only logging.
func NewInputNormalizer(logs observability.LogSink) *InputNormalizer {
	return &InputNormalizer{logs: logs}
}

// RecordRejection records a field category when typed decoding fails before normalization.
// Implements DESIGN-013 InputNormalizer input_rejected metadata policy.
func (n *InputNormalizer) RecordRejection(ctx context.Context, field RejectionField) {
	n.log(ctx, string(field), "rejected", false, 0)
}

// NormalizeExternalSearch validates a provider query before any outbound request.
// Implements DESIGN-012 ExternalSearchQuery and DESIGN-013 InputNormalizer.
func (n *InputNormalizer) NormalizeExternalSearch(ctx context.Context, req ExternalSearchRequest) (ExternalSearchRequest, error) {
	query, err := n.normalize(ctx, security.InputFieldExternalQuery, req.Query)
	if err != nil {
		return ExternalSearchRequest{}, err
	}
	provider, err := n.normalize(ctx, security.InputFieldExternalProvider, req.Provider)
	if err != nil {
		return ExternalSearchRequest{}, err
	}
	page, err := n.normalize(ctx, security.InputFieldPagination, strconv.Itoa(req.Page))
	if err != nil {
		return ExternalSearchRequest{}, errors.New("external search page is invalid")
	}
	parsedPage, _ := strconv.Atoi(page)
	return ExternalSearchRequest{Query: query, Provider: provider, Page: parsedPage}, nil
}

// NormalizeItem validates and canonicalizes one curation item before provider or repository dispatch.
// Implements DESIGN-009 DataImporter and ItemCurator and DESIGN-013 InputNormalizer.
func (n *InputNormalizer) NormalizeItem(ctx context.Context, req ItemRequest) (ItemRequest, error) {
	var err error
	if req.Name, err = n.normalize(ctx, security.InputFieldCurationItemName, req.Name); err != nil {
		return ItemRequest{}, err
	}
	if req.ImageURL, err = n.normalize(ctx, security.InputFieldImageURL, req.ImageURL); err != nil {
		return ItemRequest{}, err
	}
	if req.ServingUnit != "" {
		if req.ServingUnit, err = n.normalize(ctx, security.InputFieldServingUnit, req.ServingUnit); err != nil {
			return ItemRequest{}, err
		}
	}
	if req.SourceProvider != "" {
		if req.SourceProvider, err = n.normalize(ctx, security.InputFieldCurationProvider, req.SourceProvider); err != nil {
			return ItemRequest{}, err
		}
	}
	if req.ExternalID != "" {
		if req.ExternalID, err = n.normalize(ctx, security.InputFieldProviderIdentifier, req.ExternalID); err != nil {
			return ItemRequest{}, err
		}
	}
	if req.ProviderText, err = n.normalize(ctx, security.InputFieldProviderText, req.ProviderText); err != nil {
		return ItemRequest{}, err
	}
	if req.DensitySourceProvider != "" {
		if req.DensitySourceProvider, err = n.normalize(ctx, security.InputFieldCurationProvider, req.DensitySourceProvider); err != nil {
			return ItemRequest{}, err
		}
	}
	if req.DensitySourceFoodID != "" {
		if req.DensitySourceFoodID, err = n.normalize(ctx, security.InputFieldProviderIdentifier, req.DensitySourceFoodID); err != nil {
			return ItemRequest{}, err
		}
	}
	if req.DensitySourceKind != "" {
		if req.DensitySourceKind, err = n.normalize(ctx, security.InputFieldProviderText, req.DensitySourceKind); err != nil {
			return ItemRequest{}, err
		}
		req.DensitySourceKind = strings.ToLower(req.DensitySourceKind)
	}
	if (req.SourceProvider == "") != (req.ExternalID == "") {
		n.log(ctx, "provider_identity", "rejected", false, 0)
		return ItemRequest{}, errors.New("provider and provider identifier must be supplied together")
	}
	if !validMacrosWithinBounds(req.MacrosPer100) {
		n.log(ctx, "macros_per_100", "rejected", false, 0)
		return ItemRequest{}, errors.New("curation macro values are invalid")
	}
	if req.PrepTimeMinutes < 0 || float64(req.PrepTimeMinutes) > MaxCurationNutritionValue ||
		!validOptionalCurationMeasure(req.AverageUnitWeightGrams) ||
		!validOptionalCurationMeasure(req.AverageServingVolumeMilliliters) ||
		!validOptionalCurationMeasure(req.DensityGramsPerMilliliter) {
		n.log(ctx, "physical_measures", "rejected", false, 0)
		return ItemRequest{}, errors.New("curation physical measures are invalid")
	}
	if err := repository.ValidateMacrosPer100(req.MacrosPer100, req.PhysicalState); err != nil {
		n.log(ctx, "macros_per_100", "rejected", false, 0)
		return ItemRequest{}, errors.New("curation macro values are invalid")
	}
	if req.ServingQuantity < 0 || req.ServingQuantity > MaxCurationServingQuantity || math.IsNaN(req.ServingQuantity) || math.IsInf(req.ServingQuantity, 0) {
		n.log(ctx, "serving_quantity", "rejected", false, 0)
		return ItemRequest{}, errors.New("curation serving quantity is invalid")
	}
	if (req.ServingUnit == "") != (req.ServingQuantity == 0) {
		n.log(ctx, "serving", "rejected", false, 0)
		return ItemRequest{}, errors.New("curation serving unit and quantity must be supplied together")
	}
	if err := validateMicronutrients(req.Micronutrients); err != nil {
		n.log(ctx, "micronutrients", "rejected", false, 0)
		return ItemRequest{}, err
	}
	if req.Micronutrients == nil {
		req.Micronutrients = repository.MicroValues{}
	}
	return req, nil
}

// NormalizeClassification validates a classification name before repository dispatch.
// Implements DESIGN-009 TagManager and DESIGN-013 InputNormalizer.
func (n *InputNormalizer) NormalizeClassification(ctx context.Context, req ClassificationRequest) (ClassificationRequest, error) {
	name, err := n.normalize(ctx, security.InputFieldCurationClassificationName, req.Name)
	if err != nil {
		return ClassificationRequest{}, err
	}
	return ClassificationRequest{Name: name, ParentID: req.ParentID}, nil
}

// normalize delegates one typed field and records no raw value or error text.
// Implements DESIGN-013 InputNormalizer metadata-only logging.
func (n *InputNormalizer) normalize(ctx context.Context, field security.InputField, value string) (string, error) {
	result, err := security.NormalizeInput(field, value)
	if err != nil {
		n.log(ctx, string(field), "rejected", false, 0)
		return "", err
	}
	if result.Changed {
		n.log(ctx, string(field), "normalized", true, len(result.Violations))
	}
	return result.Value, nil
}

// log emits bounded categorical metadata and deliberately omits user/provider values.
// Implements DESIGN-013 InputNormalizer normalized/input_rejected metadata policy.
func (n *InputNormalizer) log(ctx context.Context, field string, outcome string, changed bool, violationCount int) {
	if n == nil || n.logs == nil || !allowedLogField(field) || (outcome != "normalized" && outcome != "rejected") {
		return
	}
	_ = n.logs.Log(ctx, observability.LogEvent{
		Service: "api", Level: "info", Message: "curation_input_validation", CreatedAt: time.Now(),
		Fields: map[string]any{"field": field, "outcome": outcome, "changed": changed, "violationCount": violationCount},
	})
}

// allowedLogField closes validation metadata over non-PII categories.
// Implements DESIGN-013 InputNormalizer metadata-only logging.
func allowedLogField(field string) bool {
	switch field {
	case string(security.InputFieldCurationItemName), string(security.InputFieldCurationClassificationName),
		string(security.InputFieldExternalQuery), string(security.InputFieldExternalProvider),
		string(security.InputFieldCurationProvider), string(security.InputFieldProviderIdentifier),
		string(security.InputFieldImageURL), string(security.InputFieldServingUnit),
		string(security.InputFieldProviderText), string(security.InputFieldPagination),
		"provider_identity", "macros_per_100", "serving_quantity", "serving", "micronutrients",
		"physical_measures",
		string(RejectionFieldExternalSearchQuery), string(RejectionFieldItemBody), string(RejectionFieldClassificationBody):
		return true
	default:
		return false
	}
}

// validMacrosWithinBounds enforces the persisted numeric(12,4) nutrition range.
// Implements DESIGN-005 MacroNormalizer and DESIGN-013 InputNormalizer.
func validMacrosWithinBounds(values repository.MacroValues) bool {
	return values.Protein <= MaxCurationNutritionValue && values.Carbohydrates <= MaxCurationNutritionValue && values.Fat <= MaxCurationNutritionValue
}

// validOptionalCurationMeasure bounds finite zero-as-absent persisted measures.
// Implements DESIGN-005 FoodItemEntity and DESIGN-013 InputNormalizer.
func validOptionalCurationMeasure(value float64) bool {
	return value >= 0 && value <= MaxCurationNutritionValue && !math.IsNaN(value) && !math.IsInf(value, 0)
}

// validateMicronutrients rejects malformed canonical keys and non-finite or negative values.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-013 InputNormalizer.
func validateMicronutrients(values repository.MicroValues) error {
	if len(values) > 200 {
		return errors.New("too many curation micronutrients")
	}
	for key, value := range values {
		if key == "" || utf8.RuneCountInString(key) > 120 || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > MaxCurationNutritionValue {
			return errors.New("curation micronutrients are invalid")
		}
		for _, r := range key {
			if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '_' {
				return errors.New("curation micronutrients are invalid")
			}
		}
		if strings.HasPrefix(key, "_") || strings.HasSuffix(key, "_") {
			return errors.New("curation micronutrients are invalid")
		}
	}
	return nil
}

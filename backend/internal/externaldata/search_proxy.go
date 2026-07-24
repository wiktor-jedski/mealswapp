package externaldata

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// DefaultExternalSearchPageSize bounds each selected provider page and the resulting response.
// Implements DESIGN-009 ExternalSearchProxy pagination.
const DefaultExternalSearchPageSize = 20

// ExternalCandidate is the bounded, non-persisted projection shown to administrators.
// Implements DESIGN-009 ExternalSearchProxy ExternalCandidate.
type ExternalCandidate struct {
	Provider       string                   `json:"provider"`
	ExternalID     string                   `json:"externalId"`
	Name           string                   `json:"name"`
	PhysicalState  repository.PhysicalState `json:"physicalState"`
	MacrosPer100   repository.MacroValues   `json:"macrosPer100"`
	Micronutrients repository.MicroValues   `json:"micronutrients"`
	ImageURL       string                   `json:"imageUrl,omitempty"`
	Warnings       []string                 `json:"warnings"`
}

// ExternalSearchResponse contains only normalized candidates and closed warning values.
// Implements DESIGN-009 ExternalSearchProxy result shaping.
type ExternalSearchResponse struct {
	Candidates []ExternalCandidate   `json:"candidates"`
	Warnings   []ExternalDataWarning `json:"warnings"`
	Page       int                   `json:"page"`
}

// ExternalSearchProxy orchestrates ARCH-012 without exposing persistence or audit dependencies.
// Implements DESIGN-009 ExternalSearchProxy.
type ExternalSearchProxy struct {
	providers  ProviderSet
	limits     *RateLimitHandler
	normalizer *DataNormalizer
}

// NewExternalSearchProxy creates a read-only provider orchestration boundary.
// Implements DESIGN-009 ExternalSearchProxy.
func NewExternalSearchProxy(providers ProviderSet, limits *RateLimitHandler, normalizer *DataNormalizer) *ExternalSearchProxy {
	return &ExternalSearchProxy{providers: providers, limits: limits, normalizer: normalizer}
}

// Search queries selected providers, normalizes one workflow snapshot, and returns deterministic bounded data.
// Implements DESIGN-009 ExternalSearchProxy and SW-REQ-055.
func (p *ExternalSearchProxy) Search(ctx context.Context, query ExternalSearchQuery) (ExternalSearchResponse, error) {
	if ctx == nil {
		return ExternalSearchResponse{}, errors.New("external search context is required")
	}
	if p == nil || p.normalizer == nil {
		return ExternalSearchResponse{}, errors.New("external search normalizer is required")
	}
	query.PageSize = DefaultExternalSearchPageSize
	records, warnings, err := searchExternalRecords(ctx, query, p.providers, p.limits)
	if err != nil {
		return ExternalSearchResponse{}, err
	}
	candidates, normalizationWarnings, err := p.normalizer.NormalizeRecordsWithWarnings(ctx, records)
	if err != nil {
		return ExternalSearchResponse{}, err
	}
	warnings = append(warnings, normalizationWarnings...)
	response := ExternalSearchResponse{Candidates: make([]ExternalCandidate, 0, len(candidates)), Warnings: warnings, Page: query.Page}
	for _, candidate := range candidates {
		candidate.Warnings = sortedUniqueStrings(candidate.Warnings)
		response.Candidates = append(response.Candidates, ExternalCandidate{
			Provider: candidate.Provider, ExternalID: candidate.ExternalID, Name: candidate.Name,
			PhysicalState: candidate.PhysicalState, MacrosPer100: candidate.MacrosPer100,
			Micronutrients: candidate.Micros, ImageURL: candidate.ImageURL, Warnings: candidate.Warnings,
		})
	}
	sort.Slice(response.Candidates, func(i, j int) bool {
		left, right := response.Candidates[i], response.Candidates[j]
		leftName, rightName := strings.ToLower(left.Name), strings.ToLower(right.Name)
		if leftName != rightName {
			return leftName < rightName
		}
		if left.Provider != right.Provider {
			return left.Provider < right.Provider
		}
		return left.ExternalID < right.ExternalID
	})
	response.Warnings = sortedUniqueWarnings(response.Warnings)
	return response, nil
}

// sortedUniqueWarnings canonicalizes bounded provider outcomes.
// Implements DESIGN-009 ExternalSearchProxy deterministic warning shaping.
func sortedUniqueWarnings(warnings []ExternalDataWarning) []ExternalDataWarning {
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Provider != warnings[j].Provider {
			return warnings[i].Provider < warnings[j].Provider
		}
		return warnings[i].Code < warnings[j].Code
	})
	result := warnings[:0]
	for _, warning := range warnings {
		if len(result) == 0 || result[len(result)-1].Provider != warning.Provider || result[len(result)-1].Code != warning.Code {
			result = append(result, warning)
		}
	}
	if result == nil {
		return []ExternalDataWarning{}
	}
	return result
}

// sortedUniqueStrings canonicalizes the closed candidate-warning set.
// Implements DESIGN-009 ExternalSearchProxy deterministic result shaping.
func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

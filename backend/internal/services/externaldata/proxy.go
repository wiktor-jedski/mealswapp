package externaldata

import (
	"context"
	"errors"
	"strings"

	"mealswapp/backend/internal/domain/micronutrient"
)

const ProviderAll Provider = "all"

type SearchClient interface {
	Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error)
}

type ExternalSearchProxy struct {
	usda       SearchClient
	off        SearchClient
	vocabulary []micronutrient.Entry
}

type ExternalSearchResult struct {
	Candidates []NormalizedFoodCandidate `json:"candidates"`
	Warnings   []ExternalDataWarning     `json:"warnings,omitempty"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"pageSize"`
}

func NewExternalSearchProxy(usda SearchClient, off SearchClient, vocabulary []micronutrient.Entry) ExternalSearchProxy {
	return ExternalSearchProxy{usda: usda, off: off, vocabulary: vocabulary}
}

func (proxy ExternalSearchProxy) SearchExternalFoods(ctx context.Context, query ExternalSearchQuery) (ExternalSearchResult, error) {
	query.Provider = Provider(strings.ToLower(strings.TrimSpace(string(query.Provider))))
	if query.Provider == "" {
		query.Provider = ProviderAll
	}
	if err := validateExternalQuery(query, query.Provider); err != nil {
		return ExternalSearchResult{}, err
	}
	if query.Provider != ProviderUSDA && query.Provider != ProviderOpenFoodFacts && query.Provider != ProviderAll {
		return ExternalSearchResult{}, ProviderError{Provider: query.Provider, Kind: ProviderErrorInvalidQuery, Message: "Unsupported external provider"}
	}

	result := ExternalSearchResult{Page: query.Page, PageSize: query.PageSize}
	providers := proxy.providers(query.Provider)
	for _, provider := range providers {
		records, err := proxy.searchProvider(ctx, provider, query)
		if err != nil {
			if query.Provider == ProviderAll {
				result.Warnings = append(result.Warnings, providerWarning(provider, err))
				continue
			}
			return ExternalSearchResult{}, err
		}
		for _, record := range records {
			candidate, err := NormalizeExternalRecord(record, proxy.vocabulary)
			if err != nil {
				result.Warnings = append(result.Warnings, providerWarning(provider, err))
				continue
			}
			result.Candidates = append(result.Candidates, candidate)
			result.Warnings = append(result.Warnings, candidate.Warnings...)
		}
	}

	return result, nil
}

func (proxy ExternalSearchProxy) providers(provider Provider) []Provider {
	if provider == ProviderAll {
		return []Provider{ProviderUSDA, ProviderOpenFoodFacts}
	}
	return []Provider{provider}
}

func (proxy ExternalSearchProxy) searchProvider(ctx context.Context, provider Provider, query ExternalSearchQuery) ([]ExternalFoodRecord, error) {
	providerQuery := query
	providerQuery.Provider = provider
	switch provider {
	case ProviderUSDA:
		if proxy.usda == nil {
			return nil, ProviderError{Provider: provider, Kind: ProviderErrorUnavailable, Message: "USDA client unavailable", Retryable: true}
		}
		return proxy.usda.Search(ctx, providerQuery)
	case ProviderOpenFoodFacts:
		if proxy.off == nil {
			return nil, ProviderError{Provider: provider, Kind: ProviderErrorUnavailable, Message: "OpenFoodFacts client unavailable", Retryable: true}
		}
		return proxy.off.Search(ctx, providerQuery)
	default:
		return nil, ProviderError{Provider: provider, Kind: ProviderErrorInvalidQuery, Message: "Unsupported external provider"}
	}
}

func providerWarning(provider Provider, err error) ExternalDataWarning {
	var providerErr ProviderError
	if errors.As(err, &providerErr) {
		return ExternalDataWarning{Provider: providerErr.Provider, Code: string(providerErr.Kind), Message: providerErr.Message}
	}
	return ExternalDataWarning{Provider: provider, Code: "normalization_incomplete", Message: err.Error()}
}

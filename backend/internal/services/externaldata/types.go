package externaldata

import (
	"fmt"
	"time"
)

type Provider string

const (
	ProviderUSDA Provider = "usda"
)

type ExternalSearchQuery struct {
	Query    string
	Provider Provider
	Page     int
	PageSize int
}

type ExternalFoodRecord struct {
	Provider    Provider           `json:"provider"`
	ExternalID  string             `json:"externalId"`
	Name        string             `json:"name"`
	ServingSize *float64           `json:"servingSize,omitempty"`
	ServingUnit string             `json:"servingUnit,omitempty"`
	Nutrients   map[string]float64 `json:"nutrients"`
	ImageURL    string             `json:"imageUrl,omitempty"`
	RawPayload  []byte             `json:"rawPayload,omitempty"`
}

type ProviderRateLimit struct {
	Provider     Provider
	Remaining    int
	ResetAt      time.Time
	BackoffUntil *time.Time
}

type ProviderErrorKind string

const (
	ProviderErrorInvalidQuery ProviderErrorKind = "invalid_query"
	ProviderErrorUnavailable  ProviderErrorKind = "provider_unavailable"
	ProviderErrorRateLimited  ProviderErrorKind = "provider_rate_limited"
	ProviderErrorTimeout      ProviderErrorKind = "timeout"
	ProviderErrorBadPayload   ProviderErrorKind = "invalid_external_payload"
)

type ProviderError struct {
	Provider  Provider
	Kind      ProviderErrorKind
	Message   string
	Retryable bool
	Cause     error
}

func (err ProviderError) Error() string {
	if err.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", err.Provider, err.Kind, err.Cause)
	}
	return fmt.Sprintf("%s: %s: %s", err.Provider, err.Kind, err.Message)
}

func (err ProviderError) Unwrap() error {
	return err.Cause
}

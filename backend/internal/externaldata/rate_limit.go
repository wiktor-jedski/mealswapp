package externaldata

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Implements DESIGN-012 RateLimitHandler provider quota tracking and retry orchestration.
const (
	MaxProviderRetries    = 3
	WarningRateLimited    = "provider_rate_limited"
	WarningUnavailable    = "provider_unavailable"
	WarningTimeout        = "timeout"
	WarningRetryExhausted = "retry_exhausted"
)

// ProviderRateLimit is the bounded state maintained independently per provider.
// Implements DESIGN-012 ProviderRateLimit.
type ProviderRateLimit struct {
	Provider     string
	Remaining    int
	ResetAt      time.Time
	BackoffUntil time.Time
}

// rateState stores mutable provider quota state.
// Implements DESIGN-012 RateLimitHandler.
type rateState struct{ ProviderRateLimit }

// RateLimitHandler owns quota state, retry windows, and bounded backoff decisions.
// Implements DESIGN-012 RateLimitHandler.
type RateLimitHandler struct {
	mu        sync.Mutex
	now       func() time.Time
	jitter    func(time.Duration) time.Duration
	states    map[string]rateState
	base      time.Duration
	deadline  time.Duration
	sleep     func(context.Context, time.Duration) error
	telemetry *observability.AdminExternalTelemetry
}

// WithTelemetry adds privacy-safe provider call, retry, and quota observations.
// Implements DESIGN-014 MetricsCollector.
func (h *RateLimitHandler) WithTelemetry(telemetry *observability.AdminExternalTelemetry) *RateLimitHandler {
	if h != nil {
		h.mu.Lock()
		h.telemetry = telemetry
		h.mu.Unlock()
	}
	return h
}

// NewRateLimitHandler creates an isolated handler. Clock and jitter are injectable for deterministic tests.
// Implements DESIGN-012 RateLimitHandler.
func NewRateLimitHandler(now func() time.Time, jitter func(time.Duration) time.Duration) *RateLimitHandler {
	if now == nil {
		now = time.Now
	}
	if jitter == nil {
		jitter = func(d time.Duration) time.Duration { return time.Duration(rand.Int63n(int64(d) + 1)) }
	}
	return &RateLimitHandler{now: now, jitter: jitter, states: make(map[string]rateState), base: 100 * time.Millisecond, deadline: 5 * time.Second, sleep: contextSleep}
}

// Configure sets the bounded deadline and sleep implementation used for each provider call.
// Implements DESIGN-012 RateLimitHandler configured call deadlines.
func (h *RateLimitHandler) Configure(deadline time.Duration, sleep func(context.Context, time.Duration) error) error {
	if deadline <= 0 || deadline > time.Minute {
		return errors.New("invalid provider deadline")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deadline = deadline
	if sleep != nil {
		h.sleep = sleep
	}
	return nil
}

// contextSleep waits without bypassing caller cancellation.
// Implements DESIGN-012 RateLimitHandler cancellation.
func contextSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// CheckRateLimit returns a copy of provider state, so callers cannot race or mutate it.
// Implements DESIGN-012 CheckRateLimit.
func (h *RateLimitHandler) CheckRateLimit(provider string) ProviderRateLimit {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.states[provider].ProviderRateLimit
}

// RecordRateLimit records only bounded standard response headers.
// Implements DESIGN-012 RecordRateLimit.
func (h *RateLimitHandler) RecordRateLimit(provider string, headers http.Header) error {
	if strings.TrimSpace(provider) == "" {
		return errors.New("provider required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.states[provider]
	s.Provider = provider
	if v, err := strconv.Atoi(headers.Get("X-RateLimit-Remaining")); err == nil && v >= 0 {
		s.Remaining = v
	}
	if v, err := strconv.ParseInt(headers.Get("X-RateLimit-Reset"), 10, 64); err == nil && v >= 0 {
		s.ResetAt = time.Unix(v, 0)
	}
	h.states[provider] = s
	return nil
}

// blocked checks quota and backoff windows for one provider.
// Implements DESIGN-012 RateLimitHandler.
func (h *RateLimitHandler) blocked(provider string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.states[provider]
	return h.now().Before(s.BackoffUntil) || s.Remaining == 0 && h.now().Before(s.ResetAt)
}

// backoff computes bounded exponential backoff with injected jitter.
// Implements DESIGN-012 RateLimitHandler.
func (h *RateLimitHandler) backoff(provider string, attempt int) time.Duration {
	d := h.base * time.Duration(1<<attempt)
	d += h.jitter(d)
	h.mu.Lock()
	s := h.states[provider]
	s.Provider = provider
	s.BackoffUntil = h.now().Add(d)
	h.states[provider] = s
	h.mu.Unlock()
	return d
}

// ResultProvider returns records and bounded response headers in one call, avoiding shared response state.
// Implements DESIGN-012 RateLimitHandler response-header updates.
type ResultProvider interface {
	SearchResult(context.Context, ExternalSearchQuery) (ProviderResult, error)
}

// ProviderResult contains only projected records and safe response metadata.
// Implements DESIGN-012 ProviderRateLimit.
type ProviderResult struct {
	Records []ExternalFoodRecord
	Headers http.Header
}

// projectRateLimitHeaders discards all provider response metadata except the quota fields consumed here.
// Implements DESIGN-012 ProviderRateLimit bounded response metadata.
func projectRateLimitHeaders(headers http.Header) http.Header {
	projected := make(http.Header, 2)
	for _, name := range []string{"X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if value := headers.Get(name); value != "" {
			projected.Set(name, value)
		}
	}
	return projected
}

// ProviderSet names the independently orchestrated external providers.
// Implements DESIGN-012 RateLimitHandler.
type ProviderSet struct {
	USDA          ResultProvider
	OpenFoodFacts ResultProvider
}

// ExternalDataWarning is a bounded provider outcome safe for API and telemetry use.
// Implements DESIGN-012 ExternalDataWarning.
type ExternalDataWarning struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// NormalizedFoodCandidate is the repository-shaped, non-persisted curation candidate.
// Implements DESIGN-012 DataNormalizer NormalizedFoodCandidate.
type NormalizedFoodCandidate struct {
	Provider                        string
	ExternalID                      string
	Name                            string
	PhysicalState                   repository.PhysicalState
	ServingSize                     float64
	ServingUnit                     string
	PackageSize                     float64
	PackageUnit                     string
	AverageUnitWeightGrams          float64
	AverageServingVolumeMilliliters float64
	DensityGramsPerMilliliter       float64
	DensitySourceProvider           string
	DensitySourceFoodID             string
	DensitySourceKind               DensitySourceKind
	MacrosPer100                    repository.MacroValues
	Micros                          repository.MicroValues
	Nutrients                       map[string]float64
	ImageURL                        string
	Warnings                        []string
}

// SearchExternalFoods queries providers independently, retries transient failures, and preserves partial success.
// Implements DESIGN-012 SearchExternalFoods and RateLimitHandler.
func SearchExternalFoods(ctx context.Context, query ExternalSearchQuery, providers ProviderSet, limits *RateLimitHandler) ([]NormalizedFoodCandidate, []ExternalDataWarning, error) {
	records, warnings, err := searchExternalRecords(ctx, query, providers, limits)
	if err != nil {
		return nil, warnings, err
	}
	all := make([]NormalizedFoodCandidate, 0, len(records))
	for _, record := range records {
		all = append(all, NormalizedFoodCandidate{Provider: record.Provider, ExternalID: record.ExternalID, Name: record.Name, Nutrients: record.Nutrients, ImageURL: record.ImageURL})
	}
	return all, warnings, nil
}

// searchExternalRecords preserves bounded provider projections for the DESIGN-009 ExternalSearchProxy normalization workflow.
// Implements DESIGN-009 ExternalSearchProxy and DESIGN-012 SearchExternalFoods.
func searchExternalRecords(ctx context.Context, query ExternalSearchQuery, providers ProviderSet, limits *RateLimitHandler) ([]ExternalFoodRecord, []ExternalDataWarning, error) {
	query, err := validateExternalSearchQuery(query)
	if err != nil {
		return nil, nil, err
	}
	if limits == nil {
		limits = NewRateLimitHandler(nil, nil)
	}
	selected := []struct {
		name string
		p    ResultProvider
	}{{"usda", providers.USDA}, {"openfoodfacts", providers.OpenFoodFacts}}
	if query.Provider != "all" {
		selected = selected[:0]
		if query.Provider == "usda" {
			selected = append(selected, struct {
				name string
				p    ResultProvider
			}{"usda", providers.USDA})
		} else if query.Provider == "openfoodfacts" {
			selected = append(selected, struct {
				name string
				p    ResultProvider
			}{"openfoodfacts", providers.OpenFoodFacts})
		}
	}
	var all []ExternalFoodRecord
	var warnings []ExternalDataWarning
	for _, item := range selected {
		if err := ctx.Err(); err != nil {
			return all, warnings, err
		}
		if item.p == nil {
			warnings = append(warnings, ExternalDataWarning{item.name, WarningUnavailable, WarningUnavailable})
			continue
		}
		if limits.blocked(item.name) {
			limits.telemetrySnapshot().ProviderQuota(ctx, item.name, "blocked")
			warnings = append(warnings, ExternalDataWarning{item.name, WarningRateLimited, WarningRateLimited})
			continue
		}
		providerQuery := query
		providerQuery.Provider = item.name
		var result ProviderResult
		var err error
		attempts := 0
		for attempt := 0; ; attempt++ {
			attempts = attempt + 1
			limits.mu.Lock()
			sleep, deadline, now, telemetry := limits.sleep, limits.deadline, limits.now, limits.telemetry
			limits.mu.Unlock()
			if attempt > 0 {
				telemetry.ProviderRetry(ctx, item.name, "scheduled")
				d := limits.backoff(item.name, attempt-1)
				if err = sleep(ctx, d); err != nil {
					telemetry.ProviderRetry(ctx, item.name, "canceled")
					return all, warnings, err
				}
			}
			startedAt := now()
			callCtx, cancel := context.WithTimeout(ctx, deadline)
			result, err = item.p.SearchResult(callCtx, providerQuery)
			cancel()
			telemetry.ProviderCall(ctx, item.name, providerTelemetryOutcome(err), now().Sub(startedAt))
			_ = limits.RecordRateLimit(item.name, result.Headers)
			telemetry.ProviderQuota(ctx, item.name, limits.quotaState(item.name, result.Headers))
			if ctxErr := ctx.Err(); ctxErr != nil {
				return all, warnings, ctxErr
			}
			if errors.Is(err, context.Canceled) {
				return all, warnings, err
			}
			if err == nil {
				break
			}
			if limits.blocked(item.name) {
				break
			}
			var pe *ProviderError
			if !errors.As(err, &pe) || !pe.Retryable || attempt >= MaxProviderRetries {
				break
			}
		}
		if err != nil {
			code := WarningUnavailable
			var pe *ProviderError
			if errors.As(err, &pe) {
				if pe.Code == ProviderErrorRateLimited {
					code = WarningRateLimited
				}
				if pe.Code == ProviderErrorTimeout {
					code = WarningTimeout
				}
				if pe.Retryable && attempts > MaxProviderRetries && pe.Code != ProviderErrorRateLimited && pe.Code != ProviderErrorTimeout {
					code = WarningRetryExhausted
					limits.telemetrySnapshot().ProviderRetry(ctx, item.name, "exhausted")
				}
			}
			warnings = append(warnings, ExternalDataWarning{item.name, code, code})
			continue
		}
		records := result.Records
		if len(records) > query.PageSize {
			records = records[:query.PageSize]
		}
		all = append(all, records...)
	}
	return all, warnings, nil
}

// telemetrySnapshot reads the optional telemetry dependency under the handler lock.
// Implements DESIGN-014 MetricsCollector.
func (h *RateLimitHandler) telemetrySnapshot() *observability.AdminExternalTelemetry {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.telemetry
}

// quotaState maps response state to a closed metric vocabulary.
// Implements DESIGN-014 MetricsCollector low-cardinality provider quota state.
func (h *RateLimitHandler) quotaState(_ string, headers http.Header) string {
	remainingHeader := headers.Get("X-RateLimit-Remaining")
	resetHeader := headers.Get("X-RateLimit-Reset")
	if remainingHeader == "" {
		return "unknown"
	}
	remaining, err := strconv.Atoi(remainingHeader)
	if err != nil || remaining < 0 {
		return "unknown"
	}
	if resetHeader != "" {
		reset, err := strconv.ParseInt(resetHeader, 10, 64)
		if err != nil || reset < 0 {
			return "unknown"
		}
	}
	if remaining == 0 {
		return "exhausted"
	}
	return "available"
}

// providerTelemetryOutcome removes causes and maps only closed provider categories.
// Implements DESIGN-014 MetricsCollector privacy-safe provider outcomes.
func providerTelemetryOutcome(err error) string {
	if err == nil {
		return "success"
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		switch providerErr.Code {
		case ProviderErrorInvalidInput, ProviderErrorNotConfigured, ProviderErrorRejected, ProviderErrorRateLimited, ProviderErrorUnavailable, ProviderErrorInvalidPayload, ProviderErrorResponseTooLarge, ProviderErrorTimeout, ProviderErrorCanceled:
			return string(providerErr.Code)
		default:
			return "error"
		}
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "error"
}

// validateExternalSearchQuery applies the provider-neutral bounds before selection or outbound I/O.
// Implements DESIGN-012 SearchExternalFoods input validation.
func validateExternalSearchQuery(query ExternalSearchQuery) (ExternalSearchQuery, error) {
	normalizedQuery, err := security.NormalizeInput(security.InputFieldExternalQuery, query.Query)
	if err != nil {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	normalizedProvider, err := security.NormalizeInput(security.InputFieldExternalProvider, query.Provider)
	if err != nil {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	if _, err := security.NormalizeInput(security.InputFieldPagination, strconv.Itoa(query.Page)); err != nil {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	maxPageSize := MaxOpenFoodFactsPageSize
	if normalizedProvider.Value == "usda" {
		maxPageSize = MaxUSDAPageSize
	}
	if query.PageSize < 1 || query.PageSize > maxPageSize {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	query.Query, query.Provider = normalizedQuery.Value, normalizedProvider.Value
	return query, nil
}

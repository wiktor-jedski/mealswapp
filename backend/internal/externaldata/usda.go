// Package externaldata owns bounded clients for external food-data providers.
package externaldata

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Implements DESIGN-012 USDAClient defensive provider bounds.
const (
	USDAAPIKeyEnvironment = "MEALSWAPP_USDA_API_KEY"
	DefaultUSDAEndpoint   = "https://api.nal.usda.gov/fdc/v1/foods/search"
	MaxUSDAPageSize       = 200
	defaultUSDADeadline   = 5 * time.Second
	defaultUSDABodyLimit  = int64(2 << 20)
	maxUSDABodyLimit      = defaultUSDABodyLimit
)

// ExternalSearchQuery is the provider-neutral, one-based search request.
// Implements DESIGN-012 ExternalSearchQuery.
type ExternalSearchQuery struct {
	Query    string
	Provider string
	Page     int
	PageSize int
}

// ExternalFoodPortion preserves provider volume evidence and its measured gram weight.
// Implements DESIGN-012 USDAClient payload parsing for later DataNormalizer density derivation.
type ExternalFoodPortion struct {
	Amount     float64
	Unit       string
	GramWeight float64
}

// ExternalFoodRecord is the loss-bounded USDA projection consumed by normalization.
// Implements DESIGN-012 ExternalFoodRecord.
type ExternalFoodRecord struct {
	Provider    string
	ExternalID  string
	Name        string
	ServingSize *float64
	ServingUnit string
	PackageSize *float64
	PackageUnit string
	Nutrients   map[string]float64
	Portions    []ExternalFoodPortion
	ImageURL    string
	RawPayload  json.RawMessage
}

// ProviderErrorCode is a closed, secret-safe provider failure category.
// Implements DESIGN-012 USDAClient provider diagnostics.
type ProviderErrorCode string

// Implements DESIGN-012 USDAClient provider status mapping.
const (
	ProviderErrorInvalidInput     ProviderErrorCode = "invalid_input"
	ProviderErrorNotConfigured    ProviderErrorCode = "not_configured"
	ProviderErrorRejected         ProviderErrorCode = "provider_rejected"
	ProviderErrorRateLimited      ProviderErrorCode = "provider_rate_limited"
	ProviderErrorUnavailable      ProviderErrorCode = "provider_unavailable"
	ProviderErrorInvalidPayload   ProviderErrorCode = "invalid_external_payload"
	ProviderErrorResponseTooLarge ProviderErrorCode = "provider_response_too_large"
	ProviderErrorTimeout          ProviderErrorCode = "timeout"
	ProviderErrorCanceled         ProviderErrorCode = "canceled"
)

// ProviderError reports bounded diagnostics without URLs, credentials, or payloads.
// Implements DESIGN-012 USDAClient safe provider diagnostics.
type ProviderError struct {
	Code       ProviderErrorCode
	HTTPStatus int
	Retryable  bool
	provider   string
	cause      error
}

// Error returns a stable diagnostic suitable for logs and upstream mapping.
// Implements DESIGN-012 USDAClient safe provider diagnostics.
func (e *ProviderError) Error() string {
	provider := e.provider
	if provider == "" {
		provider = "usda"
	}
	return provider + " request failed: " + string(e.Code)
}

// Unwrap preserves cancellation and deadline matching without exposing the cause text.
// Implements DESIGN-012 USDAClient cancellation and deadline propagation.
func (e *ProviderError) Unwrap() error { return e.cause }

// USDAConfig configures the FoodData Central boundary.
// Implements DESIGN-012 USDAClient configured deadlines and response limits.
type USDAConfig struct {
	APIKey       string
	Endpoint     string
	Deadline     time.Duration
	MaxBodyBytes int64
	HTTPClient   *http.Client
	Logs         observability.LogSink
}

// USDAClient performs one bounded FoodData Central search request.
// Implements DESIGN-012 USDAClient.
type USDAClient struct {
	apiKey       string
	endpoint     *url.URL
	deadline     time.Duration
	maxBodyBytes int64
	httpClient   *http.Client
	logs         observability.LogSink
}

// LoadUSDAAPIKey loads the API key from the process environment without adding it to diagnostics.
// Implements DESIGN-012 USDAClient credential loading.
func LoadUSDAAPIKey() (string, error) {
	key := strings.TrimSpace(os.Getenv(USDAAPIKeyEnvironment))
	if key == "" || strings.ContainsAny(key, "\x00\r\n") {
		return "", &ProviderError{Code: ProviderErrorNotConfigured}
	}
	return key, nil
}

// NewUSDAClient validates immutable provider configuration.
// Implements DESIGN-012 USDAClient configured request boundary.
func NewUSDAClient(cfg USDAConfig) (*USDAClient, error) {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	if cfg.APIKey == "" || strings.ContainsAny(cfg.APIKey, "\x00\r\n") {
		return nil, &ProviderError{Code: ProviderErrorNotConfigured}
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultUSDAEndpoint
	}
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Scheme != "https" && endpoint.Scheme != "http" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	if cfg.Deadline == 0 {
		cfg.Deadline = defaultUSDADeadline
	}
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = defaultUSDABodyLimit
	}
	if cfg.Deadline < 0 || cfg.MaxBodyBytes < 1 || cfg.MaxBodyBytes > maxUSDABodyLimit {
		return nil, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	httpClient := *cfg.HTTPClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &USDAClient{apiKey: cfg.APIKey, endpoint: endpoint, deadline: cfg.Deadline, maxBodyBytes: cfg.MaxBodyBytes, httpClient: &httpClient, logs: cfg.Logs}, nil
}

// Search validates input before network access, enforces a deadline and body limit, and projects USDA records.
// Implements DESIGN-012 USDAClient Search.
func (c *USDAClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error) {
	result, err := c.SearchResult(ctx, query)
	return result.Records, err
}

// SearchResult preserves bounded quota headers alongside every HTTP response outcome.
// Implements DESIGN-012 USDAClient response-header updates.
func (c *USDAClient) SearchResult(ctx context.Context, query ExternalSearchQuery) (ProviderResult, error) {
	validated, err := validateUSDAQuery(query)
	if err != nil {
		return ProviderResult{}, err
	}
	requestURL := *c.endpoint
	values := requestURL.Query()
	values.Set("api_key", c.apiKey)
	values.Set("query", validated.Query)
	values.Set("pageNumber", strconv.Itoa(validated.Page))
	values.Set("pageSize", strconv.Itoa(validated.PageSize))
	requestURL.RawQuery = values.Encode()

	requestCtx, cancel := context.WithTimeout(ctx, c.deadline)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return ProviderResult{}, c.failure(ctx, ProviderErrorInvalidInput, 0, false, nil)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ProviderResult{}, c.transportFailure(ctx, requestCtx, err, 0)
	}
	defer resp.Body.Close()
	result := ProviderResult{Headers: projectRateLimitHeaders(resp.Header)}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		code, retryable := mapUSDAStatus(resp.StatusCode)
		return result, c.failure(ctx, code, resp.StatusCode, retryable, nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBodyBytes+1))
	if err != nil {
		return result, c.transportFailure(ctx, requestCtx, err, resp.StatusCode)
	}
	if int64(len(body)) > c.maxBodyBytes {
		return result, c.failure(ctx, ProviderErrorResponseTooLarge, resp.StatusCode, false, nil)
	}
	records, err := decodeUSDASearch(body)
	if err != nil {
		return result, c.failure(ctx, ProviderErrorInvalidPayload, resp.StatusCode, false, nil)
	}
	result.Records = records
	return result, nil
}

// validateUSDAQuery applies defensive query and pagination bounds before network access.
// Implements DESIGN-012 USDAClient request construction.
func validateUSDAQuery(query ExternalSearchQuery) (ExternalSearchQuery, error) {
	normalizedQuery, err := security.NormalizeInput(security.InputFieldExternalQuery, query.Query)
	if err != nil {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	normalizedProvider, err := security.NormalizeInput(security.InputFieldExternalProvider, query.Provider)
	if err != nil || normalizedProvider.Value != "usda" {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	if _, err := security.NormalizeInput(security.InputFieldPagination, strconv.Itoa(query.Page)); err != nil {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	if query.PageSize < 1 || query.PageSize > MaxUSDAPageSize {
		return ExternalSearchQuery{}, &ProviderError{Code: ProviderErrorInvalidInput}
	}
	query.Query, query.Provider = normalizedQuery.Value, normalizedProvider.Value
	return query, nil
}

// usdaSearchPayload captures the required FoodData Central search envelope.
// Implements DESIGN-012 USDAClient payload parsing.
type usdaSearchPayload struct {
	TotalHits   *int              `json:"totalHits"`
	CurrentPage *int              `json:"currentPage"`
	TotalPages  *int              `json:"totalPages"`
	Foods       []json.RawMessage `json:"foods"`
}

// usdaFood captures provider fields required for external-record projection.
// Implements DESIGN-012 USDAClient payload parsing.
type usdaFood struct {
	FDCID       int            `json:"fdcId"`
	Description string         `json:"description"`
	ServingSize *float64       `json:"servingSize"`
	ServingUnit string         `json:"servingSizeUnit"`
	Nutrients   []usdaNutrient `json:"foodNutrients"`
	Measures    []usdaMeasure  `json:"foodMeasures"`
}

// usdaNutrient captures one named and unit-qualified USDA nutrient value.
// Implements DESIGN-012 USDAClient payload parsing.
type usdaNutrient struct {
	Name  string  `json:"nutrientName"`
	Unit  string  `json:"unitName"`
	Value float64 `json:"value"`
}

// usdaMeasure captures portion amounts with provider-measured gram weights.
// Implements DESIGN-012 USDAClient volume-portion parsing.
type usdaMeasure struct {
	DisseminationText string          `json:"disseminationText"`
	GramWeight        float64         `json:"gramWeight"`
	Amount            float64         `json:"amount"`
	MeasureUnit       usdaMeasureUnit `json:"measureUnit"`
}

// usdaMeasureUnit captures USDA's preferred and fallback portion-unit labels.
// Implements DESIGN-012 USDAClient volume-portion parsing.
type usdaMeasureUnit struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

// decodeUSDASearch rejects malformed or partial payloads and produces deterministic records.
// Implements DESIGN-012 USDAClient payload parsing.
func decodeUSDASearch(body []byte) ([]ExternalFoodRecord, error) {
	var payload usdaSearchPayload
	if err := json.Unmarshal(body, &payload); err != nil || payload.TotalHits == nil || payload.CurrentPage == nil || payload.TotalPages == nil || payload.Foods == nil || *payload.TotalHits < 0 || *payload.CurrentPage < 0 || *payload.TotalPages < 0 {
		return nil, errors.New("incomplete USDA search payload")
	}
	records := make([]ExternalFoodRecord, 0, len(payload.Foods))
	for _, raw := range payload.Foods {
		var food usdaFood
		if err := json.Unmarshal(raw, &food); err != nil {
			return nil, errors.New("malformed USDA food")
		}
		if food.FDCID < 1 || strings.TrimSpace(food.Description) == "" || food.Nutrients == nil || (food.ServingSize == nil) != (strings.TrimSpace(food.ServingUnit) == "") || food.ServingSize != nil && (!finitePositive(*food.ServingSize)) {
			return nil, errors.New("incomplete USDA food")
		}
		nutrients := make(map[string]float64, len(food.Nutrients))
		for _, nutrient := range food.Nutrients {
			name, unit := strings.TrimSpace(nutrient.Name), strings.TrimSpace(nutrient.Unit)
			key := name + " (" + unit + ")"
			if name == "" || unit == "" || math.IsNaN(nutrient.Value) || math.IsInf(nutrient.Value, 0) || nutrient.Value < 0 {
				return nil, errors.New("invalid USDA nutrient")
			}
			if _, duplicate := nutrients[key]; duplicate {
				return nil, errors.New("duplicate USDA nutrient")
			}
			nutrients[key] = nutrient.Value
		}
		portions := make([]ExternalFoodPortion, 0, len(food.Measures))
		for _, measure := range food.Measures {
			unit := strings.TrimSpace(measure.MeasureUnit.Abbreviation)
			if unit == "" {
				unit = strings.TrimSpace(measure.MeasureUnit.Name)
			}
			if unit == "" {
				unit = strings.TrimSpace(measure.DisseminationText)
			}
			if !finitePositive(measure.Amount) || !finitePositive(measure.GramWeight) || unit == "" {
				return nil, errors.New("invalid USDA portion")
			}
			portions = append(portions, ExternalFoodPortion{Amount: measure.Amount, Unit: unit, GramWeight: measure.GramWeight})
		}
		sort.Slice(portions, func(i, j int) bool {
			if portions[i].Unit != portions[j].Unit {
				return portions[i].Unit < portions[j].Unit
			}
			if portions[i].Amount != portions[j].Amount {
				return portions[i].Amount < portions[j].Amount
			}
			return portions[i].GramWeight < portions[j].GramWeight
		})
		records = append(records, ExternalFoodRecord{Provider: "usda", ExternalID: strconv.Itoa(food.FDCID), Name: strings.TrimSpace(food.Description), ServingSize: food.ServingSize, ServingUnit: strings.TrimSpace(food.ServingUnit), Nutrients: nutrients, Portions: portions, RawPayload: append(json.RawMessage(nil), raw...)})
	}
	return records, nil
}

// finitePositive validates provider quantities used as serving or portion evidence.
// Implements DESIGN-012 USDAClient payload parsing.
func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

// mapUSDAStatus maps HTTP status codes to the closed provider failure vocabulary.
// Implements DESIGN-012 USDAClient provider diagnostics.
func mapUSDAStatus(status int) (ProviderErrorCode, bool) {
	switch {
	case status == http.StatusTooManyRequests:
		return ProviderErrorRateLimited, true
	case status == http.StatusRequestTimeout || status >= http.StatusInternalServerError:
		return ProviderErrorUnavailable, true
	case status >= http.StatusBadRequest:
		return ProviderErrorRejected, false
	default:
		return ProviderErrorUnavailable, true
	}
}

// transportFailure preserves safe context sentinels while discarding URL-bearing causes.
// Implements DESIGN-012 USDAClient safe provider diagnostics.
func (c *USDAClient) transportFailure(ctx context.Context, requestCtx context.Context, transportErr error, status int) error {
	if errors.Is(requestCtx.Err(), context.DeadlineExceeded) || errors.Is(transportErr, context.DeadlineExceeded) {
		return c.failure(ctx, ProviderErrorTimeout, status, true, context.DeadlineExceeded)
	}
	if errors.Is(requestCtx.Err(), context.Canceled) || errors.Is(transportErr, context.Canceled) {
		return c.failure(ctx, ProviderErrorCanceled, status, false, context.Canceled)
	}
	return c.failure(ctx, ProviderErrorUnavailable, status, true, nil)
}

// failure emits only bounded categorical metadata and returns a safe provider error.
// Implements DESIGN-012 USDAClient safe provider diagnostics.
func (c *USDAClient) failure(ctx context.Context, code ProviderErrorCode, status int, retryable bool, cause error) error {
	err := &ProviderError{Code: code, HTTPStatus: status, Retryable: retryable, cause: cause}
	if c.logs != nil {
		_ = c.logs.Log(ctx, observability.LogEvent{Service: "api", Level: "warn", Message: "external_provider_failure", Fields: map[string]any{"provider": "usda", "code": string(code), "status": status, "retryable": retryable}, CreatedAt: time.Now()})
	}
	return err
}

// Implements DESIGN-012 USDAClient compile-time provider error contract.
var _ error = (*ProviderError)(nil)

package externaldata

// Implements DESIGN-012 OpenFoodFactsClient defensive provider boundary.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Implements DESIGN-012 OpenFoodFactsClient request and response bounds.
const (
	DefaultOpenFoodFactsEndpoint  = "https://world.openfoodfacts.org/cgi/search.pl"
	MaxOpenFoodFactsPageSize      = 100
	defaultOpenFoodFactsDeadline  = 5 * time.Second
	defaultOpenFoodFactsBodyLimit = int64(2 << 20)
	maxOpenFoodFactsBodyLimit     = defaultOpenFoodFactsBodyLimit
	openFoodFactsFields           = "code,product_name,serving_quantity,serving_quantity_unit,product_quantity,product_quantity_unit,image_front_url,nutriments"
)

// OpenFoodFactsConfig configures the public OpenFoodFacts search boundary.
// Implements DESIGN-012 OpenFoodFactsClient caller identification and response limits.
type OpenFoodFactsConfig struct {
	CallerID     string
	Endpoint     string
	Deadline     time.Duration
	MaxBodyBytes int64
	HTTPClient   *http.Client
	Logs         observability.LogSink
}

// OpenFoodFactsClient performs one bounded OpenFoodFacts text-search request.
// Implements DESIGN-012 OpenFoodFactsClient.
type OpenFoodFactsClient struct {
	callerID     string
	endpoint     *url.URL
	deadline     time.Duration
	maxBodyBytes int64
	httpClient   *http.Client
	logs         observability.LogSink
}

// NewOpenFoodFactsClient validates immutable provider configuration and disables redirects.
// Implements DESIGN-012 OpenFoodFactsClient configured request boundary.
func NewOpenFoodFactsClient(cfg OpenFoodFactsConfig) (*OpenFoodFactsClient, error) {
	cfg.CallerID = strings.TrimSpace(cfg.CallerID)
	if !validCallerID(cfg.CallerID) {
		return nil, openFoodFactsError(ProviderErrorNotConfigured, 0, false, nil)
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultOpenFoodFactsEndpoint
	}
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Scheme != "https" && endpoint.Scheme != "http" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, openFoodFactsError(ProviderErrorInvalidInput, 0, false, nil)
	}
	if cfg.Deadline == 0 {
		cfg.Deadline = defaultOpenFoodFactsDeadline
	}
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = defaultOpenFoodFactsBodyLimit
	}
	if cfg.Deadline < 0 || cfg.MaxBodyBytes < 1 || cfg.MaxBodyBytes > maxOpenFoodFactsBodyLimit {
		return nil, openFoodFactsError(ProviderErrorInvalidInput, 0, false, nil)
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	httpClient := *cfg.HTTPClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &OpenFoodFactsClient{callerID: cfg.CallerID, endpoint: endpoint, deadline: cfg.Deadline, maxBodyBytes: cfg.MaxBodyBytes, httpClient: &httpClient, logs: cfg.Logs}, nil
}

// Search validates before I/O, applies a deadline and body cap, and returns a loss-bounded projection.
// Implements DESIGN-012 OpenFoodFactsClient Search.
func (c *OpenFoodFactsClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error) {
	result, err := c.SearchResult(ctx, query)
	return result.Records, err
}

// SearchResult preserves bounded quota headers alongside every HTTP response outcome.
// Implements DESIGN-012 OpenFoodFactsClient response-header updates.
func (c *OpenFoodFactsClient) SearchResult(ctx context.Context, query ExternalSearchQuery) (ProviderResult, error) {
	validated, err := validateOpenFoodFactsQuery(query)
	if err != nil {
		return ProviderResult{}, err
	}
	requestURL := *c.endpoint
	values := requestURL.Query()
	values.Set("action", "process")
	values.Set("fields", openFoodFactsFields)
	values.Set("json", "1")
	values.Set("page", strconv.Itoa(validated.Page))
	values.Set("page_size", strconv.Itoa(validated.PageSize))
	values.Set("search_simple", "1")
	values.Set("search_terms", validated.Query)
	requestURL.RawQuery = values.Encode()

	requestCtx, cancel := context.WithTimeout(ctx, c.deadline)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return ProviderResult{}, c.failure(ctx, ProviderErrorInvalidInput, 0, false, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.callerID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ProviderResult{}, c.transportFailure(ctx, requestCtx, err, 0)
	}
	defer resp.Body.Close()
	result := ProviderResult{Headers: projectRateLimitHeaders(resp.Header)}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		code, retryable := mapOpenFoodFactsStatus(resp.StatusCode)
		return result, c.failure(ctx, code, resp.StatusCode, retryable, nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBodyBytes+1))
	if err != nil {
		return result, c.transportFailure(ctx, requestCtx, err, resp.StatusCode)
	}
	if int64(len(body)) > c.maxBodyBytes {
		return result, c.failure(ctx, ProviderErrorResponseTooLarge, resp.StatusCode, false, nil)
	}
	records, dropped, err := decodeOpenFoodFactsSearch(body)
	if err != nil {
		return result, c.failure(ctx, ProviderErrorInvalidPayload, resp.StatusCode, false, nil)
	}
	if dropped > 0 {
		c.logDropped(ctx, dropped)
	}
	result.Records = records
	return result, nil
}

// validateOpenFoodFactsQuery applies shared text/page validation and provider-specific page-size bounds.
// Implements DESIGN-012 OpenFoodFactsClient request construction.
func validateOpenFoodFactsQuery(query ExternalSearchQuery) (ExternalSearchQuery, error) {
	normalizedQuery, err := security.NormalizeInput(security.InputFieldExternalQuery, query.Query)
	if err != nil {
		return ExternalSearchQuery{}, openFoodFactsError(ProviderErrorInvalidInput, 0, false, nil)
	}
	normalizedProvider, err := security.NormalizeInput(security.InputFieldExternalProvider, query.Provider)
	if err != nil || normalizedProvider.Value != "openfoodfacts" {
		return ExternalSearchQuery{}, openFoodFactsError(ProviderErrorInvalidInput, 0, false, nil)
	}
	if _, err := security.NormalizeInput(security.InputFieldPagination, strconv.Itoa(query.Page)); err != nil || query.PageSize < 1 || query.PageSize > MaxOpenFoodFactsPageSize {
		return ExternalSearchQuery{}, openFoodFactsError(ProviderErrorInvalidInput, 0, false, nil)
	}
	query.Query, query.Provider = normalizedQuery.Value, normalizedProvider.Value
	return query, nil
}

// openFoodFactsSearchPayload captures the required legacy text-search envelope.
// Implements DESIGN-012 OpenFoodFactsClient payload parsing.
type openFoodFactsSearchPayload struct {
	Count     *int                   `json:"count"`
	Page      *int                   `json:"page"`
	PageCount *int                   `json:"page_count"`
	PageSize  *int                   `json:"page_size"`
	Products  []openFoodFactsProduct `json:"products"`
}

// openFoodFactsProduct captures only fields required by downstream normalization.
// Implements DESIGN-012 OpenFoodFactsClient payload parsing without raw-payload retention.
type openFoodFactsProduct struct {
	Code                json.RawMessage            `json:"code"`
	Name                json.RawMessage            `json:"product_name"`
	ServingQuantity     json.RawMessage            `json:"serving_quantity"`
	ServingQuantityUnit json.RawMessage            `json:"serving_quantity_unit"`
	ProductQuantity     json.RawMessage            `json:"product_quantity"`
	ProductQuantityUnit json.RawMessage            `json:"product_quantity_unit"`
	ImageURL            json.RawMessage            `json:"image_front_url"`
	Nutrients           map[string]json.RawMessage `json:"nutriments"`
}

// decodeOpenFoodFactsSearch rejects malformed envelopes and drops malformed product candidates.
// Implements DESIGN-012 OpenFoodFactsClient payload parsing.
func decodeOpenFoodFactsSearch(body []byte) ([]ExternalFoodRecord, int, error) {
	var payload openFoodFactsSearchPayload
	if err := json.Unmarshal(body, &payload); err != nil || payload.Count == nil || payload.Page == nil || payload.PageCount == nil || payload.PageSize == nil || payload.Products == nil || *payload.Count < 0 || *payload.Page < 1 || *payload.PageCount < 0 || *payload.PageSize < 1 {
		return nil, 0, errors.New("incomplete OpenFoodFacts search payload")
	}
	records := make([]ExternalFoodRecord, 0, len(payload.Products))
	dropped := 0
	for _, product := range payload.Products {
		record, ok := projectOpenFoodFactsProduct(product)
		if !ok {
			dropped++
			continue
		}
		records = append(records, record)
	}
	return records, dropped, nil
}

// projectOpenFoodFactsProduct validates one candidate and discards all unselected provider bytes.
// Implements DESIGN-012 OpenFoodFactsClient deterministic ExternalFoodRecord projection.
func projectOpenFoodFactsProduct(product openFoodFactsProduct) (ExternalFoodRecord, bool) {
	code, ok := decodeJSONString(product.Code)
	if !ok {
		return ExternalFoodRecord{}, false
	}
	identifier, err := security.NormalizeInput(security.InputFieldProviderIdentifier, code)
	if err != nil {
		return ExternalFoodRecord{}, false
	}
	name, ok := decodeJSONString(product.Name)
	if !ok {
		return ExternalFoodRecord{}, false
	}
	normalizedName, err := security.NormalizeInput(security.InputFieldProviderText, name)
	if err != nil || product.Nutrients == nil {
		return ExternalFoodRecord{}, false
	}
	nutrients := make(map[string]float64, len(product.Nutrients))
	for rawKey, rawValue := range product.Nutrients {
		key := strings.TrimSpace(rawKey)
		if key == "" || utf8.RuneCountInString(key) > 128 || containsUnsafeProviderText(key) {
			return ExternalFoodRecord{}, false
		}
		token := bytes.TrimSpace(rawValue)
		if len(token) == 0 {
			return ExternalFoodRecord{}, false
		}
		if token[0] == '"' {
			if key == "label" || strings.HasSuffix(key, "_unit") {
				continue
			}
			return ExternalFoodRecord{}, false
		}
		if token[0] != '-' && (token[0] < '0' || token[0] > '9') {
			return ExternalFoodRecord{}, false
		}
		var value float64
		if err := json.Unmarshal(token, &value); err != nil {
			return ExternalFoodRecord{}, false
		}
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return ExternalFoodRecord{}, false
		}
		nutrients[key] = value
	}

	servingSize, servingUnit := projectOpenFoodFactsQuantity(product.ServingQuantity, product.ServingQuantityUnit)
	packageSize, packageUnit := projectOpenFoodFactsQuantity(product.ProductQuantity, product.ProductQuantityUnit)

	imageURL := ""
	if len(product.ImageURL) > 0 {
		if value, imageOK := decodeJSONString(product.ImageURL); imageOK {
			if normalizedImage, imageErr := security.NormalizeInput(security.InputFieldImageURL, value); imageErr == nil {
				imageURL = normalizedImage.Value
			}
		}
	}
	return ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: identifier.Value, Name: normalizedName.Value, ServingSize: servingSize, ServingUnit: servingUnit, PackageSize: packageSize, PackageUnit: packageUnit, Nutrients: nutrients, ImageURL: imageURL}, true
}

// projectOpenFoodFactsQuantity canonicalizes optional serving or package quantity pairs.
// Implements DESIGN-012 DataNormalizer package and serving conversion inputs.
func projectOpenFoodFactsQuantity(rawQuantity json.RawMessage, rawUnit json.RawMessage) (*float64, string) {
	if len(rawQuantity) == 0 && len(rawUnit) == 0 {
		return nil, ""
	}
	var quantity float64
	unit, ok := decodeJSONString(rawUnit)
	if json.Unmarshal(rawQuantity, &quantity) != nil || !finitePositive(quantity) || !ok {
		return nil, ""
	}
	normalized, err := security.NormalizeInput(security.InputFieldServingUnit, unit)
	if err != nil {
		return nil, ""
	}
	return &quantity, normalized.Value
}

// decodeJSONString accepts one non-empty JSON string and trims provider padding.
// Implements DESIGN-012 OpenFoodFactsClient payload parsing.
func decodeJSONString(raw json.RawMessage) (string, bool) {
	var value string
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

// containsUnsafeProviderText rejects controls and invalid UTF-8 in provider-owned map keys.
// Implements DESIGN-012 OpenFoodFactsClient payload parsing.
func containsUnsafeProviderText(value string) bool {
	if !utf8.ValidString(value) {
		return true
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return true
		}
	}
	return false
}

// validCallerID enforces a bounded visible custom User-Agent before outbound access.
// Implements DESIGN-012 OpenFoodFactsClient required caller identification.
func validCallerID(value string) bool {
	return value != "" && len(value) <= 256 && !containsUnsafeProviderText(value)
}

// mapOpenFoodFactsStatus maps response statuses into the shared provider vocabulary.
// Implements DESIGN-012 OpenFoodFactsClient provider diagnostics.
func mapOpenFoodFactsStatus(status int) (ProviderErrorCode, bool) {
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

// transportFailure preserves only safe context sentinels and discards URL-bearing causes.
// Implements DESIGN-012 OpenFoodFactsClient safe provider diagnostics.
func (c *OpenFoodFactsClient) transportFailure(ctx context.Context, requestCtx context.Context, transportErr error, status int) error {
	if errors.Is(requestCtx.Err(), context.DeadlineExceeded) || errors.Is(transportErr, context.DeadlineExceeded) {
		return c.failure(ctx, ProviderErrorTimeout, status, true, context.DeadlineExceeded)
	}
	if errors.Is(requestCtx.Err(), context.Canceled) || errors.Is(transportErr, context.Canceled) {
		return c.failure(ctx, ProviderErrorCanceled, status, false, context.Canceled)
	}
	return c.failure(ctx, ProviderErrorUnavailable, status, true, nil)
}

// failure emits only categorical metadata and returns a provider-safe error.
// Implements DESIGN-012 OpenFoodFactsClient safe provider diagnostics.
func (c *OpenFoodFactsClient) failure(ctx context.Context, code ProviderErrorCode, status int, retryable bool, cause error) error {
	err := openFoodFactsError(code, status, retryable, cause)
	if c.logs != nil {
		_ = c.logs.Log(ctx, observability.LogEvent{Service: "api", Level: "warn", Message: "external_provider_failure", Fields: map[string]any{"provider": "openfoodfacts", "code": string(code), "status": status, "retryable": retryable}, CreatedAt: time.Now()})
	}
	return err
}

// logDropped records a bounded count without product values or payload bytes.
// Implements DESIGN-012 OpenFoodFactsClient safe provider diagnostics.
func (c *OpenFoodFactsClient) logDropped(ctx context.Context, count int) {
	if c.logs != nil {
		_ = c.logs.Log(ctx, observability.LogEvent{Service: "api", Level: "warn", Message: "external_provider_payload_dropped", Fields: map[string]any{"provider": "openfoodfacts", "count": count}, CreatedAt: time.Now()})
	}
}

// openFoodFactsError constructs the shared error without retaining provider-controlled data.
// Implements DESIGN-012 OpenFoodFactsClient safe provider diagnostics.
func openFoodFactsError(code ProviderErrorCode, status int, retryable bool, cause error) *ProviderError {
	return &ProviderError{Code: code, HTTPStatus: status, Retryable: retryable, provider: "openfoodfacts", cause: cause}
}

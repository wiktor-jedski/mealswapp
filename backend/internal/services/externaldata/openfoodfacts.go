package externaldata

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultOpenFoodFactsBaseURL = "https://world.openfoodfacts.org"

type OpenFoodFactsClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	rateLimit  ProviderRateLimit
}

type OpenFoodFactsClientOption func(*OpenFoodFactsClient)

func NewOpenFoodFactsClient(options ...OpenFoodFactsClientOption) *OpenFoodFactsClient {
	client := &OpenFoodFactsClient{
		baseURL:    defaultOpenFoodFactsBaseURL,
		httpClient: http.DefaultClient,
		timeout:    5 * time.Second,
		rateLimit:  ProviderRateLimit{Provider: ProviderOpenFoodFacts, Remaining: -1},
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func WithOpenFoodFactsHTTPClient(httpClient *http.Client) OpenFoodFactsClientOption {
	return func(client *OpenFoodFactsClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithOpenFoodFactsBaseURL(baseURL string) OpenFoodFactsClientOption {
	return func(client *OpenFoodFactsClient) {
		if strings.TrimSpace(baseURL) != "" {
			client.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func WithOpenFoodFactsTimeout(timeout time.Duration) OpenFoodFactsClientOption {
	return func(client *OpenFoodFactsClient) {
		if timeout > 0 {
			client.timeout = timeout
		}
	}
}

func (client *OpenFoodFactsClient) RateLimit() ProviderRateLimit {
	return client.rateLimit
}

func (client *OpenFoodFactsClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error) {
	if err := validateExternalQuery(query, ProviderOpenFoodFacts); err != nil {
		return nil, err
	}

	requestURL, err := client.searchURL(query)
	if err != nil {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorInvalidQuery, Message: "Invalid OpenFoodFacts search URL", Cause: err}
	}
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorInvalidQuery, Message: "Invalid OpenFoodFacts request", Cause: err}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MealSwapp/1.0 (+https://mealswapp.local)")

	res, err := client.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorTimeout, Message: "OpenFoodFacts request timed out", Retryable: true, Cause: err}
		}
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorUnavailable, Message: "OpenFoodFacts unavailable", Retryable: true, Cause: err}
	}
	defer res.Body.Close()
	client.recordRateLimit(res.Header)

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorRateLimited, Message: "OpenFoodFacts rate limited", Retryable: true}
	}
	if res.StatusCode >= 500 {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorUnavailable, Message: "OpenFoodFacts unavailable", Retryable: true}
	}
	if res.StatusCode >= 400 {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorInvalidQuery, Message: "OpenFoodFacts rejected request"}
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorUnavailable, Message: "OpenFoodFacts response unavailable", Retryable: true, Cause: err}
	}
	var payload openFoodFactsSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorBadPayload, Message: "OpenFoodFacts response malformed", Cause: err}
	}
	return parseOpenFoodFactsProducts(payload.Products)
}

func (client *OpenFoodFactsClient) searchURL(query ExternalSearchQuery) (string, error) {
	endpoint, err := url.Parse(client.baseURL + "/cgi/search.pl")
	if err != nil {
		return "", err
	}
	params := endpoint.Query()
	params.Set("search_terms", strings.TrimSpace(query.Query))
	params.Set("page", strconv.Itoa(query.Page))
	params.Set("page_size", strconv.Itoa(query.PageSize))
	params.Set("search_simple", "1")
	params.Set("action", "process")
	params.Set("json", "1")
	params.Set("fields", "code,product_name,serving_quantity,serving_size,nutriments,image_url")
	endpoint.RawQuery = params.Encode()
	return endpoint.String(), nil
}

func (client *OpenFoodFactsClient) recordRateLimit(header http.Header) {
	limit := ProviderRateLimit{Provider: ProviderOpenFoodFacts, Remaining: -1}
	if remaining, err := strconv.Atoi(header.Get("X-RateLimit-Remaining")); err == nil {
		limit.Remaining = remaining
	}
	if reset, err := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64); err == nil && reset > 0 {
		limit.ResetAt = time.Unix(reset, 0).UTC()
	}
	if retryAfter, err := strconv.Atoi(header.Get("Retry-After")); err == nil && retryAfter > 0 {
		backoff := time.Now().UTC().Add(time.Duration(retryAfter) * time.Second)
		limit.BackoffUntil = &backoff
	}
	client.rateLimit = limit
}

type openFoodFactsSearchResponse struct {
	Products []openFoodFactsProduct `json:"products"`
}

type openFoodFactsProduct struct {
	Code            string                  `json:"code"`
	ProductName     string                  `json:"product_name"`
	ServingQuantity *float64                `json:"serving_quantity"`
	ServingSize     string                  `json:"serving_size"`
	Nutriments      openFoodFactsNutriments `json:"nutriments"`
	ImageURL        string                  `json:"image_url"`
}

type openFoodFactsNutriments map[string]float64

func parseOpenFoodFactsProducts(products []openFoodFactsProduct) ([]ExternalFoodRecord, error) {
	records := make([]ExternalFoodRecord, 0, len(products))
	for _, product := range products {
		code := strings.TrimSpace(product.Code)
		name := strings.TrimSpace(product.ProductName)
		if code == "" || name == "" {
			continue
		}
		raw, err := json.Marshal(product)
		if err != nil {
			return nil, ProviderError{Provider: ProviderOpenFoodFacts, Kind: ProviderErrorBadPayload, Message: "OpenFoodFacts product payload malformed", Cause: err}
		}
		servingSize, servingUnit := parseOpenFoodFactsServing(product.ServingQuantity, product.ServingSize)
		records = append(records, ExternalFoodRecord{
			Provider:    ProviderOpenFoodFacts,
			ExternalID:  code,
			Name:        name,
			ServingSize: servingSize,
			ServingUnit: servingUnit,
			Nutrients:   parseOpenFoodFactsNutrients(product.Nutriments),
			ImageURL:    strings.TrimSpace(product.ImageURL),
			RawPayload:  raw,
		})
	}
	return records, nil
}

func parseOpenFoodFactsServing(quantity *float64, label string) (*float64, string) {
	if quantity != nil && *quantity > 0 {
		return quantity, servingUnitFromLabel(label)
	}
	return nil, servingUnitFromLabel(label)
}

func servingUnitFromLabel(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(normalized, "ml"):
		return "ml"
	case strings.Contains(normalized, "g"):
		return "g"
	case strings.Contains(normalized, "serving"):
		return "serving"
	default:
		return ""
	}
}

func parseOpenFoodFactsNutrients(nutrients openFoodFactsNutriments) map[string]float64 {
	values := make(map[string]float64)
	for key, value := range nutrients {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" || strings.HasSuffix(name, "_unit") || strings.HasSuffix(name, "_value") {
			continue
		}
		values[name] = value
	}
	return values
}

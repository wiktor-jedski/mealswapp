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

const defaultUSDABaseURL = "https://api.nal.usda.gov/fdc/v1"

type USDAClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
	rateLimit  ProviderRateLimit
}

type USDAClientOption func(*USDAClient)

func NewUSDAClient(apiKey string, options ...USDAClientOption) *USDAClient {
	client := &USDAClient{
		baseURL:    defaultUSDABaseURL,
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
		timeout:    5 * time.Second,
		rateLimit:  ProviderRateLimit{Provider: ProviderUSDA, Remaining: -1},
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func WithUSDAHTTPClient(httpClient *http.Client) USDAClientOption {
	return func(client *USDAClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithUSDABaseURL(baseURL string) USDAClientOption {
	return func(client *USDAClient) {
		if strings.TrimSpace(baseURL) != "" {
			client.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func WithUSDATimeout(timeout time.Duration) USDAClientOption {
	return func(client *USDAClient) {
		if timeout > 0 {
			client.timeout = timeout
		}
	}
}

func (client *USDAClient) RateLimit() ProviderRateLimit {
	return client.rateLimit
}

func (client *USDAClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error) {
	if err := validateExternalQuery(query); err != nil {
		return nil, err
	}

	requestURL, err := client.searchURL(query)
	if err != nil {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorInvalidQuery, Message: "Invalid USDA search URL", Cause: err}
	}
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorInvalidQuery, Message: "Invalid USDA request", Cause: err}
	}
	req.Header.Set("Accept", "application/json")

	res, err := client.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorTimeout, Message: "USDA request timed out", Retryable: true, Cause: err}
		}
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorUnavailable, Message: "USDA unavailable", Retryable: true, Cause: err}
	}
	defer res.Body.Close()
	client.recordRateLimit(res.Header)

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorRateLimited, Message: "USDA rate limited", Retryable: true}
	}
	if res.StatusCode >= 500 {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorUnavailable, Message: "USDA unavailable", Retryable: true}
	}
	if res.StatusCode >= 400 {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorInvalidQuery, Message: "USDA rejected request"}
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorUnavailable, Message: "USDA response unavailable", Retryable: true, Cause: err}
	}
	var payload usdaSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorBadPayload, Message: "USDA response malformed", Cause: err}
	}
	return parseUSDAFoods(payload.Foods)
}

func (client *USDAClient) searchURL(query ExternalSearchQuery) (string, error) {
	endpoint, err := url.Parse(client.baseURL + "/foods/search")
	if err != nil {
		return "", err
	}
	params := endpoint.Query()
	params.Set("query", strings.TrimSpace(query.Query))
	params.Set("pageNumber", strconv.Itoa(query.Page))
	params.Set("pageSize", strconv.Itoa(query.PageSize))
	if client.apiKey != "" {
		params.Set("api_key", client.apiKey)
	}
	endpoint.RawQuery = params.Encode()
	return endpoint.String(), nil
}

func (client *USDAClient) recordRateLimit(header http.Header) {
	limit := ProviderRateLimit{Provider: ProviderUSDA, Remaining: -1}
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

func validateExternalQuery(query ExternalSearchQuery) error {
	if strings.TrimSpace(query.Query) == "" {
		return ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorInvalidQuery, Message: "Search query is required"}
	}
	if query.Page < 1 {
		return ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorInvalidQuery, Message: "Page must be positive"}
	}
	if query.PageSize < 1 || query.PageSize > 50 {
		return ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorInvalidQuery, Message: "Page size must be between 1 and 50"}
	}
	return nil
}

type usdaSearchResponse struct {
	Foods []usdaFood `json:"foods"`
}

type usdaFood struct {
	FDCID       int            `json:"fdcId"`
	Description string         `json:"description"`
	ServingSize *float64       `json:"servingSize"`
	ServingUnit string         `json:"servingSizeUnit"`
	Nutrients   []usdaNutrient `json:"foodNutrients"`
}

type usdaNutrient struct {
	Name  string  `json:"nutrientName"`
	Value float64 `json:"value"`
	Unit  string  `json:"unitName"`
}

func parseUSDAFoods(foods []usdaFood) ([]ExternalFoodRecord, error) {
	records := make([]ExternalFoodRecord, 0, len(foods))
	for _, food := range foods {
		if food.FDCID == 0 || strings.TrimSpace(food.Description) == "" {
			continue
		}
		raw, err := json.Marshal(food)
		if err != nil {
			return nil, ProviderError{Provider: ProviderUSDA, Kind: ProviderErrorBadPayload, Message: "USDA food payload malformed", Cause: err}
		}
		records = append(records, ExternalFoodRecord{
			Provider:    ProviderUSDA,
			ExternalID:  strconv.Itoa(food.FDCID),
			Name:        strings.TrimSpace(food.Description),
			ServingSize: food.ServingSize,
			ServingUnit: strings.TrimSpace(food.ServingUnit),
			Nutrients:   parseUSDANutrients(food.Nutrients),
			RawPayload:  raw,
		})
	}
	return records, nil
}

func parseUSDANutrients(nutrients []usdaNutrient) map[string]float64 {
	values := make(map[string]float64)
	for _, nutrient := range nutrients {
		name := strings.ToLower(strings.TrimSpace(nutrient.Name))
		if name == "" {
			continue
		}
		values[name] = nutrient.Value
	}
	return values
}

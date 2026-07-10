package app

// Implements DESIGN-010 RouteHandler app constructor verification.

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// TestNewBuildsRouter proves that app router is built,
// /health is reachable and returns OK health response
// TestNewBuildsRouter verifies DESIGN-010 RouteHandler app constructor behavior.
func TestNewBuildsRouter(t *testing.T) {
	server, err := New(httpapi.Dependencies{Config: config.Config{APITimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173"}}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	resp, err := server.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

type fakeProductionPostgres struct{}

func (fakeProductionPostgres) Ping(context.Context) error { return nil }
func (fakeProductionPostgres) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("transaction not available")
}
func (fakeProductionPostgres) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("exec not available")
}
func (fakeProductionPostgres) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("query not available")
}
func (fakeProductionPostgres) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeProductionRow{}
}

type fakeProductionRow struct{}

func (fakeProductionRow) Scan(...any) error { return errors.New("row not available") }

// TestNewProductionExposesProductionRoutes verifies DESIGN-010 RouteHandler production composition.
func TestNewProductionExposesProductionRoutes(t *testing.T) {
	cfg := config.Config{
		APITimeout:     time.Second,
		AllowedOrigins: []string{"http://localhost:5173"},
		FrontendOrigin: "http://localhost:5173",
		Environment:    "development",
		Account: config.AccountConfig{
			AccessTokenTTL:              15 * time.Minute,
			RefreshTokenTTL:             7 * 24 * time.Hour,
			AccessCookieName:            "__Host-test_access",
			RefreshCookieName:           "__Host-test_refresh",
			CurrentPrivacyPolicyVersion: "privacy-v1",
			CurrentTermsVersion:         "terms-v1",
		},
	}
	server, err := NewProduction(cfg, fakeProductionPostgres{}, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	checks := []struct {
		method string
		path   string
		body   string
	}{
		{fiber.MethodGet, "/api/v1/auth/csrf-token", ""},
		{fiber.MethodGet, "/api/v1/disclaimers?location=login", ""},
		{fiber.MethodGet, "/api/v1/auth/oauth/google/start", ""},
		{fiber.MethodPost, "/api/v1/auth/register", `{"bad":true}`},
		{fiber.MethodGet, "/api/v1/profile", ""},
		{fiber.MethodGet, "/api/v1/account/export", ""},
		{fiber.MethodDelete, "/api/v1/account", ""},
		{fiber.MethodPost, "/api/v1/search", `{"query":"milk","mode":"catalog","page":1,"filters":[]}`},
		{fiber.MethodGet, "/api/v1/search/autocomplete?query=milk", ""},
		{fiber.MethodGet, "/api/v1/food-objects/71000000-0000-4000-8000-000000000001", ""},
		{fiber.MethodPost, "/api/v1/billing/stripe/webhook", `{"bad":true}`},
	}
	for _, check := range checks {
		req := httptest.NewRequest(check.method, check.path, strings.NewReader(check.body))
		if check.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := server.Test(req)
		if err != nil {
			t.Fatalf("%s %s error = %v", check.method, check.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == fiber.StatusNotFound {
			t.Fatalf("%s %s returned 404; route is not composed", check.method, check.path)
		}
	}
}

// TestNewProductionSearchRouteBlocksAnonymousSubstitutionBeforeCatalog verifies DESIGN-002 and DESIGN-007 production search composition.
func TestNewProductionSearchRouteBlocksAnonymousSubstitutionBeforeCatalog(t *testing.T) {
	cfg := config.Config{
		APITimeout:     time.Second,
		AllowedOrigins: []string{"http://localhost:5173"},
		FrontendOrigin: "http://localhost:5173",
		Environment:    "development",
		Account: config.AccountConfig{
			AccessTokenTTL:              15 * time.Minute,
			RefreshTokenTTL:             7 * 24 * time.Hour,
			AccessCookieName:            "__Host-test_access",
			RefreshCookieName:           "__Host-test_refresh",
			CurrentPrivacyPolicyVersion: "privacy-v1",
			CurrentTermsVersion:         "terms-v1",
		},
	}
	server, err := NewProduction(cfg, fakeProductionPostgres{}, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	body := `{"query":"milk","mode":"substitution","page":1,"filters":[],"substitutionInputs":[{"foodObjectId":"60000000-0000-4000-8000-000000000001","quantity":100,"unit":"g"}]}`

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("response = %d body=%s", resp.StatusCode, responseBody)
	}
	if strings.Contains(string(responseBody), "search mode is not available for catalog results") || strings.Contains(string(responseBody), `"field":"mode"`) {
		t.Fatalf("substitution request still reached catalog-only rejection: %s", responseBody)
	}
	if !strings.Contains(string(responseBody), `"code":"entitlement_denied"`) || !strings.Contains(string(responseBody), `"feature":"single_substitution"`) {
		t.Fatalf("substitution request did not stop at entitlement gate: %s", responseBody)
	}
}

func TestLocalKeyLoaderValidationAndInterfaces(t *testing.T) {
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")
	if _, err := newLocalKeyLoader("production"); err == nil {
		t.Fatal("production key loader accepted a missing secret")
	}
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "short")
	if _, err := newLocalKeyLoader("development"); err == nil {
		t.Fatal("key loader accepted a short secret")
	}
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "12345678901234567890123456789012-extra")
	loader, err := newLocalKeyLoader("production")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	version, key, err := loader.ActiveKey(ctx)
	if err != nil || version != "local-v1" || len(key) != 32 {
		t.Fatalf("ActiveKey() version=%q len=%d err=%v", version, len(key), err)
	}
	for name, load := range map[string]func() (string, []byte, error){
		"lookup":  func() (string, []byte, error) { return loader.ActiveLookupKey(ctx) },
		"signing": func() (string, []byte, error) { return loader.ActiveSigningKey(ctx) },
	} {
		gotVersion, gotKey, err := load()
		if err != nil || gotVersion != version || string(gotKey) != string(key) {
			t.Fatalf("%s key version=%q key=%q err=%v", name, gotVersion, gotKey, err)
		}
	}
	if _, err := loader.Key(ctx, "missing"); err == nil {
		t.Fatal("Key() accepted unknown version")
	}
	if got, err := loader.LookupKey(ctx, version); err != nil || string(got) != string(key) {
		t.Fatalf("LookupKey() key=%q err=%v", got, err)
	}
	if got, err := loader.SigningKey(ctx, version); err != nil || string(got) != string(key) {
		t.Fatalf("SigningKey() key=%q err=%v", got, err)
	}
}

func TestUnavailableOAuthGatewayAndRedisPurgerFailClosed(t *testing.T) {
	gateway := unavailableOAuthGateway{}
	if _, err := gateway.StartOAuth(context.Background(), "google", "state"); err == nil {
		t.Fatal("StartOAuth() did not fail closed")
	}
	if _, err := gateway.CompleteOAuth(context.Background(), "google", nil); err == nil {
		t.Fatal("CompleteOAuth() did not fail closed")
	}
	if err := (redisCachePurger{}).PurgeUser(context.Background(), uuid.New()); err != nil {
		t.Fatalf("nil Redis purge error = %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	defer client.Close()
	if err := (redisCachePurger{client: client}).PurgeUser(context.Background(), uuid.New()); err == nil {
		t.Fatal("Redis purge connection failure ignored")
	}
}

func TestGoogleOAuthGatewayConfiguration(t *testing.T) {
	missing := NewGoogleOAuthGateway(config.OAuthConfig{})
	if _, err := missing.StartOAuth(context.Background(), "google", "state"); err == nil {
		t.Fatal("missing Google config did not fail closed")
	}

	gateway := NewGoogleOAuthGateway(config.OAuthConfig{
		GoogleClientID:     "google-client-id",
		GoogleClientSecret: "google-client-secret",
		GoogleCallbackURL:  "http://localhost:8080/api/v1/auth/oauth/google/callback",
	})
	if _, err := gateway.StartOAuth(context.Background(), "apple", "state"); err == nil {
		t.Fatal("Apple OAuth unexpectedly enabled")
	}
	location, err := gateway.StartOAuth(context.Background(), "google", "state-191")
	if err != nil {
		t.Fatalf("StartOAuth() error = %v", err)
	}
	if !strings.Contains(location, "accounts.google.com") || !strings.Contains(location, "state=state-191") || strings.Contains(location, "google-client-secret") {
		t.Fatalf("unexpected Google auth URL: %s", location)
	}
}

func TestNewProductionRejectsMissingProductionSecret(t *testing.T) {
	t.Setenv("MEALSWAPP_LOCAL_SECRET_KEY", "")
	if _, err := NewProduction(config.Config{Environment: "production"}, fakeProductionPostgres{}, nil, observability.JSONSink{Writer: io.Discard}); err == nil {
		t.Fatal("NewProduction() accepted missing production secret")
	}
}

func TestNewProductionReadinessWithOptionalRedis(t *testing.T) {
	cfg := config.Config{
		APITimeout:     time.Second,
		AllowedOrigins: []string{"http://localhost:5173"},
		FrontendOrigin: "http://localhost:5173",
		Environment:    "development",
		Account: config.AccountConfig{
			AccessCookieName:            "__Host-test_access",
			RefreshCookieName:           "__Host-test_refresh",
			CurrentPrivacyPolicyVersion: "privacy-v1",
			CurrentTermsVersion:         "terms-v1",
		},
	}
	server, err := NewProduction(cfg, fakeProductionPostgres{}, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("readiness without redis = %d", resp.StatusCode)
	}

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: time.Millisecond})
	defer client.Close()
	server, err = NewProduction(cfg, fakeProductionPostgres{}, client, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = server.Test(httptest.NewRequest(fiber.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("readiness with unavailable redis = %d", resp.StatusCode)
	}
}

type autocompleteFoodRepository struct{}

func (autocompleteFoodRepository) GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.FoodItemEntity, error) {
	return repository.FoodItemEntity{}, errors.New("unused")
}
func (autocompleteFoodRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	items := []repository.FoodItemEntity{{ID: uuid.New(), Name: "Apple"}}
	return items, len(items), nil
}
func (autocompleteFoodRepository) Create(context.Context, repository.FoodItemEntity) (uuid.UUID, error) {
	return uuid.Nil, errors.New("unused")
}
func (autocompleteFoodRepository) Update(context.Context, repository.FoodItemEntity) error {
	return errors.New("unused")
}
func (autocompleteFoodRepository) Delete(context.Context, uuid.UUID) error {
	return errors.New("unused")
}

type autocompleteMealRepository struct{}

func (autocompleteMealRepository) GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.MealEntity, error) {
	return repository.MealEntity{}, errors.New("unused")
}
func (autocompleteMealRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.MealEntity, int, error) {
	return nil, 0, nil
}
func (autocompleteMealRepository) CalculateMacros(context.Context, uuid.UUID) (repository.MacroValues, error) {
	return repository.MacroValues{}, errors.New("unused")
}
func (autocompleteMealRepository) Create(context.Context, repository.MealEntity) (uuid.UUID, error) {
	return uuid.Nil, errors.New("unused")
}
func (autocompleteMealRepository) Update(context.Context, repository.MealEntity) error {
	return errors.New("unused")
}
func (autocompleteMealRepository) Delete(context.Context, uuid.UUID) error {
	return errors.New("unused")
}

type memoryRedisStore struct{ values map[string]string }

func (s *memoryRedisStore) Get(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", redis.Nil
	}
	return value, nil
}
func (s *memoryRedisStore) Set(_ context.Context, key string, value string, _ time.Duration) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func TestSearchAutocompleteAdapterCaching(t *testing.T) {
	service := search.NewAutocompleteService(autocompleteFoodRepository{}, autocompleteMealRepository{})
	adapter := searchAutocompleteAdapter{service: service}
	response, err := adapter.Autocomplete(context.Background(), "app", repository.RepositoryContext{})
	if err != nil || len(response.Items) == 0 || response.Cache != nil {
		t.Fatalf("uncached autocomplete response=%+v err=%v", response, err)
	}

	store := &memoryRedisStore{values: map[string]string{}}
	adapter.cache = store
	response, err = adapter.Autocomplete(context.Background(), "app", repository.RepositoryContext{})
	if err != nil || response.Cache == nil || response.Cache.Status != search.CacheStatusMiss {
		t.Fatalf("cache miss response=%+v err=%v", response, err)
	}
	response, err = adapter.Autocomplete(context.Background(), "app", repository.RepositoryContext{})
	if err != nil || response.Cache == nil || response.Cache.Status != search.CacheStatusHit {
		t.Fatalf("cache hit response=%+v err=%v", response, err)
	}
}

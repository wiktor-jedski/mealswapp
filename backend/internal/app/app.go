package app

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/compliance"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/profile"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

// New constructs the backend Fiber app from HTTP API dependencies.
// Implements DESIGN-010 RouteHandler app constructor seam.
func New(deps httpapi.Dependencies) (*fiber.App, error) {
	return httpapi.NewRouter(deps)
}

// NewProduction composes the Phase 03 API routes with PostgreSQL-backed services.
// Implements DESIGN-010 RouteHandler and DESIGN-006 AuthController production bootstrap.
func NewProduction(cfg config.Config, pg postgresStore, redisClient *redis.Client, telemetry observability.JSONSink) (*fiber.App, error) {
	keys, err := newLocalKeyLoader(cfg.Environment)
	if err != nil {
		return nil, err
	}
	encryption := security.NewEncryptionService(keys)
	digests := security.NewLookupDigestService(keys)
	tokens := auth.NewJWTManager(keys)
	sessions := repository.NewPostgresSessionRepository(pg)
	csrf := httpapi.NewCSRFManager(cfg, repository.NewPostgresSecurityAuditRepository(pg))
	sessionManager := httpapi.NewAuthSessionManager(cfg, csrf)
	identities := repository.NewPostgresEncryptedIdentityRepository(pg)
	verification := repository.NewPostgresAccountVerificationRepository(pg)
	authService := auth.NewCoreAuthService(auth.CoreAuthDependencies{
		Config: auth.CoreAuthConfig{AccessTokenTTL: cfg.Account.AccessTokenTTL, RefreshTokenTTL: cfg.Account.RefreshTokenTTL},
		Registration: auth.NewRegistrationService(
			repository.NewPostgresRegistrationRepository(pg),
			cfg.Account.CurrentPrivacyPolicyVersion,
			cfg.Account.CurrentTermsVersion,
		),
		Identities: identities, Sessions: sessions, Verification: verification, ResetTokens: verification,
		Lockout: auth.NewAccountLockoutTracker(repository.NewPostgresAccountLockoutRepository(pg)),
		Hasher:  auth.NewDefaultPasswordHasher(), Tokens: tokens, Encryption: encryption, Digests: digests,
	})
	savedRepo := repository.NewPostgresSavedDataRepository(pg)
	foodRepo := repository.NewPostgresFoodItemRepository(pg)
	mealRepo := repository.NewPostgresMealRepository(pg)
	complianceRepo := repository.NewPostgresComplianceRepository(pg)
	var searchResponseCache search.SearchResponseCache
	var similarityCache search.SimilarityCalculationCache
	var redisStore cache.RedisStore
	if redisClient != nil {
		redisStore = cache.GoRedisStore{Client: redisClient}
		searchResponseCache = cache.SearchResponseStore{Store: redisStore}
		similarityCache = cache.SearchResponseStore{Store: redisStore}
	}
	userDataService := userdata.NewService(savedRepo, identities, savedRepo, encryption)
	controllers := []httpapi.Controller{
		httpapi.NewAuthController(authService, sessionManager).WithLogSink(telemetry),
		httpapi.NewOAuthController(authService, unavailableOAuthGateway{}, sessionManager),
		httpapi.NewProfileController(profile.NewService(identities, encryption)),
		httpapi.NewSearchController(search.NewSearchDispatcher(
			search.NewCatalogService(foodRepo, searchResponseCache),
			search.NewSubstitutionService(foodRepo, searchResponseCache, similarityCache),
		)).WithAutocompleteService(searchAutocompleteAdapter{
			service: search.NewAutocompleteService(foodRepo, mealRepo),
			cache:   redisStore,
			ttl:     cache.DefaultAutocompleteTTL,
		}).WithSearchHistoryAppender(userDataService),
		httpapi.NewFoodObjectController(foodRepo),
		httpapi.NewUserDataController(userDataService),
		httpapi.NewExportController(userdata.NewExportService(identities, identities, savedRepo, identities, complianceRepo, encryption)),
		httpapi.NewAccountDeletionController(userdata.NewAccountDeletionService(complianceRepo, sessions, identities, redisCachePurger{client: redisClient}), sessionManager),
		httpapi.NewDisclaimerController(compliance.NewDisclaimerService(nil)),
	}
	routes := []httpapi.RouteDefinition{}
	for _, controller := range controllers {
		routes = append(routes, controller.Routes()...)
	}
	deps := httpapi.Dependencies{
		Config: cfg,
		PostgresPing: func(ctx context.Context) error {
			return pg.Ping(ctx)
		},
		RedisPing: func(ctx context.Context) error {
			if redisClient == nil {
				return nil
			}
			return redisClient.Ping(ctx).Err()
		},
		Audit: repository.NewPostgresSecurityAuditRepository(pg), Logs: telemetry, Metrics: telemetry,
		CSRF: csrf, Auth: httpapi.NewJWTAuthenticator(cfg, tokens, sessions), Routes: routes,
	}
	return New(deps)
}

// searchAutocompleteAdapter adds optional Redis metadata around autocomplete ranking.
// Implements DESIGN-002 SearchController and DESIGN-011 RedisCache.
type searchAutocompleteAdapter struct {
	service search.AutocompleteService
	cache   cache.RedisStore
	ttl     time.Duration
}

// Autocomplete returns ranked suggestions with cache metadata when Redis is configured.
// Implements DESIGN-002 SearchController.
func (a searchAutocompleteAdapter) Autocomplete(ctx context.Context, query string, rc repository.RepositoryContext) (search.AutocompleteResponse, error) {
	load := func(loadCtx context.Context) (search.AutocompleteResponse, error) {
		items, err := a.service.Autocomplete(loadCtx, query, rc)
		return search.AutocompleteResponse{Items: items}, err
	}
	if a.cache == nil {
		return load(ctx)
	}
	ttl := a.ttl
	if ttl <= 0 {
		ttl = cache.DefaultAutocompleteTTL
	}
	return cache.GetOrLoadAutocompleteResponse(ctx, a.cache, query, ttl, load)
}

// unavailableOAuthGateway exposes OAuth routes without fabricating provider behavior.
// Implements DESIGN-006 OAuthHandler production provider boundary.
type unavailableOAuthGateway struct{}

// Implements DESIGN-006 OAuthHandler compile-time provider gateway contract.
var _ httpapi.OAuthProviderGateway = unavailableOAuthGateway{}

// StartOAuth fails closed until Google or Apple provider credentials are configured.
// Implements DESIGN-006 OAuthHandler.
func (unavailableOAuthGateway) StartOAuth(context.Context, string, string) (string, error) {
	return "", errors.New("OAuth provider gateway is not configured")
}

// CompleteOAuth fails closed until Google or Apple provider credentials are configured.
// Implements DESIGN-006 OAuthHandler.
func (unavailableOAuthGateway) CompleteOAuth(context.Context, string, map[string]string) (auth.OAuthProfile, error) {
	return auth.OAuthProfile{}, errors.New("OAuth provider gateway is not configured")
}

// postgresStore is the shared PostgreSQL repository/readiness boundary.
// Implements DESIGN-005 RepositoryInterfaces.
type postgresStore interface {
	Ping(context.Context) error
	Begin(context.Context) (pgx.Tx, error)
	repositorySQLExecutor
}

// repositorySQLExecutor is satisfied by database.Pool.
// Implements DESIGN-005 RepositoryInterfaces.
type repositorySQLExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// localKeyLoader resolves local auth, encryption, and lookup keys from the environment.
// Implements DESIGN-013 EncryptionService and DESIGN-006 JWTManager.
type localKeyLoader struct {
	version string
	key     []byte
}

// Implements DESIGN-006 JWTManager compile-time signing key contract.
var _ auth.SigningKeyLoader = localKeyLoader{}

// Implements DESIGN-013 EncryptionService compile-time encryption key contract.
var _ security.KeyLoader = localKeyLoader{}

// Implements DESIGN-013 EncryptionService compile-time lookup key contract.
var _ security.LookupKeyLoader = localKeyLoader{}

// newLocalKeyLoader creates local key material for Phase 03 account flows.
// Implements DESIGN-013 EncryptionService and DESIGN-006 JWTManager.
func newLocalKeyLoader(environment string) (localKeyLoader, error) {
	key := os.Getenv("MEALSWAPP_LOCAL_SECRET_KEY")
	if key == "" {
		if environment == "production" {
			return localKeyLoader{}, errors.New("production requires MEALSWAPP_LOCAL_SECRET_KEY")
		}
		key = "dev-local-secret-key-32-bytes-ok!"
	}
	if len([]byte(key)) < 32 {
		return localKeyLoader{}, errors.New("MEALSWAPP_LOCAL_SECRET_KEY must contain at least 32 bytes")
	}
	return localKeyLoader{version: "local-v1", key: []byte(key)[:32]}, nil
}

// ActiveKey returns the active AES-256-GCM key.
// Implements DESIGN-013 EncryptionService.
func (l localKeyLoader) ActiveKey(context.Context) (string, []byte, error) {
	return l.version, l.key, nil
}

// Key returns a versioned AES-256-GCM key.
// Implements DESIGN-013 EncryptionService.
func (l localKeyLoader) Key(_ context.Context, version string) ([]byte, error) {
	if version != l.version {
		return nil, errors.New("local key version is unavailable")
	}
	return l.key, nil
}

// ActiveLookupKey returns the active deterministic lookup key.
// Implements DESIGN-013 EncryptionService.
func (l localKeyLoader) ActiveLookupKey(context.Context) (string, []byte, error) {
	return l.version, l.key, nil
}

// LookupKey returns a versioned deterministic lookup key.
// Implements DESIGN-013 EncryptionService.
func (l localKeyLoader) LookupKey(ctx context.Context, version string) ([]byte, error) {
	return l.Key(ctx, version)
}

// ActiveSigningKey returns the active JWT signing key.
// Implements DESIGN-006 JWTManager.
func (l localKeyLoader) ActiveSigningKey(context.Context) (string, []byte, error) {
	return l.version, l.key, nil
}

// SigningKey returns a versioned JWT signing key.
// Implements DESIGN-006 JWTManager.
func (l localKeyLoader) SigningKey(ctx context.Context, version string) ([]byte, error) {
	return l.Key(ctx, version)
}

// redisCachePurger removes user cache entries when a Redis client is configured.
// Implements DESIGN-008 AccountDeleter.
type redisCachePurger struct {
	client *redis.Client
}

// Implements DESIGN-008 AccountDeleter compile-time cache purge contract.
var _ userdata.CachePurger = redisCachePurger{}

// PurgeUser deletes the current user cache prefix best-effort.
// Implements DESIGN-008 AccountDeleter.
func (p redisCachePurger) PurgeUser(ctx context.Context, userID uuid.UUID) error {
	if p.client == nil {
		return nil
	}
	return p.client.Del(ctx, "user:"+userID.String()).Err()
}

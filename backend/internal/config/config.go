package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Implements DESIGN-010 RequestValidator local development defaults.
const (
	defaultHTTPPort             = "8080"
	defaultDatabaseURL          = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"
	defaultRedisURL             = "redis://localhost:6379/0"
	defaultEnvironment          = "development"
	defaultFrontendOrigin       = "http://localhost:5173"
	defaultAPITimeout           = 10 * time.Second
	defaultHSTSMaxAge           = 31536000
	defaultAccessTokenTTL       = 15 * time.Minute
	defaultRefreshTokenTTL      = 7 * 24 * time.Hour
	defaultEmailVerificationTTL = 24 * time.Hour
	defaultPasswordResetTTL     = time.Hour
)

// Config contains the environment-backed settings for the API and worker.
// Implements DESIGN-010 RequestValidator shared gateway configuration inputs.
type Config struct {
	HTTPPort       string
	DatabaseURL    string
	RedisURL       string
	Environment    string
	FrontendOrigin string
	AllowedOrigins []string
	APITimeout     time.Duration
	TrustedProxy   bool
	EnforceTLS     bool
	HSTSMaxAge     int
	TLSMinVersion  string
	Account        AccountConfig
	Billing        BillingConfig
}

// BillingConfig contains Stripe billing settings.
// Implements DESIGN-007 SubscriptionController configuration.
type BillingConfig struct {
	StripeSecretKey     string
	StripeWebhookSecret string
	MonthlyPlanPriceID  string
	AnnualPlanPriceID   string
	CheckoutSuccessURL  string
	CheckoutCancelURL   string
}

// AccountConfig contains authentication and account-flow settings.
// Implements DESIGN-006 AuthController and DESIGN-013 InputNormalizer.
type AccountConfig struct {
	PasswordMinLength           int
	AccessTokenTTL              time.Duration
	RefreshTokenTTL             time.Duration
	AccessCookieName            string
	RefreshCookieName           string
	CurrentPrivacyPolicyVersion string
	CurrentTermsVersion         string
	DisclaimerFallbackVersion   string
	EmailVerificationTTL        time.Duration
	PasswordResetTTL            time.Duration
}

// Load reads Mealswapp configuration from the environment and applies local defaults.
// Implements DESIGN-010 RequestValidator environment-backed config loading.
func Load() (Config, error) {
	cfg := Config{
		HTTPPort:       env("MEALSWAPP_HTTP_PORT", defaultHTTPPort),
		DatabaseURL:    env("MEALSWAPP_DATABASE_URL", defaultDatabaseURL),
		RedisURL:       env("MEALSWAPP_REDIS_URL", defaultRedisURL),
		Environment:    env("MEALSWAPP_ENV", defaultEnvironment),
		FrontendOrigin: env("MEALSWAPP_FRONTEND_ORIGIN", defaultFrontendOrigin),
	}
	cfg.AllowedOrigins = splitCSV(env("MEALSWAPP_ALLOWED_ORIGINS", cfg.FrontendOrigin))
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, errors.New("MEALSWAPP_ALLOWED_ORIGINS must contain at least one origin")
	}
	var err error
	if cfg.APITimeout, err = time.ParseDuration(env("MEALSWAPP_API_TIMEOUT", defaultAPITimeout.String())); err != nil || cfg.APITimeout <= 0 {
		return Config{}, errors.New("MEALSWAPP_API_TIMEOUT must be a positive duration")
	}
	if cfg.TrustedProxy, err = strconv.ParseBool(env("MEALSWAPP_TRUST_PROXY", "false")); err != nil {
		return Config{}, errors.New("MEALSWAPP_TRUST_PROXY must be a boolean")
	}
	if cfg.TrustedProxy {
		return Config{}, errors.New("MEALSWAPP_TRUST_PROXY=true is deferred until Phase 09 trusted ingress enforcement")
	}
	if cfg.EnforceTLS, err = strconv.ParseBool(env("MEALSWAPP_ENFORCE_TLS", "false")); err != nil {
		return Config{}, errors.New("MEALSWAPP_ENFORCE_TLS must be a boolean")
	}
	if cfg.HSTSMaxAge, err = strconv.Atoi(env("MEALSWAPP_HSTS_MAX_AGE", strconv.Itoa(defaultHSTSMaxAge))); err != nil || cfg.HSTSMaxAge < 0 {
		return Config{}, errors.New("MEALSWAPP_HSTS_MAX_AGE must be a non-negative integer")
	}
	if cfg.TLSMinVersion = env("MEALSWAPP_TLS_MIN_VERSION", "1.3"); cfg.TLSMinVersion != "1.3" {
		return Config{}, errors.New("MEALSWAPP_TLS_MIN_VERSION must be 1.3")
	}
	if cfg.Account, err = loadAccountConfig(); err != nil {
		return Config{}, err
	}
	if cfg.Billing, err = loadBillingConfig(cfg.FrontendOrigin); err != nil {
		return Config{}, err
	}

	if cfg.Environment == "production" {
		if os.Getenv("MEALSWAPP_DATABASE_URL") == "" || os.Getenv("MEALSWAPP_REDIS_URL") == "" {
			return Config{}, errors.New("production requires MEALSWAPP_DATABASE_URL and MEALSWAPP_REDIS_URL")
		}
		if cfg.Account.CurrentPrivacyPolicyVersion == "dev-privacy-v1" || cfg.Account.CurrentTermsVersion == "dev-terms-v1" {
			return Config{}, errors.New("production requires current consent versions")
		}
		if strings.HasPrefix(cfg.Account.AccessCookieName, "dev_") || strings.HasPrefix(cfg.Account.RefreshCookieName, "dev_") {
			return Config{}, errors.New("production requires non-development auth cookie names")
		}
		if strings.HasPrefix(cfg.Billing.StripeSecretKey, "sk_test_") || cfg.Billing.StripeSecretKey == "" || cfg.Billing.StripeSecretKey == "sk_test_dummy" {
			return Config{}, errors.New("production requires live Stripe secret key")
		}
		if cfg.Billing.StripeWebhookSecret == "whsec_dummy" || cfg.Billing.StripeWebhookSecret == "" {
			return Config{}, errors.New("production requires live Stripe webhook secret")
		}
		if strings.Contains(cfg.Billing.MonthlyPlanPriceID, "dummy") || strings.Contains(cfg.Billing.AnnualPlanPriceID, "dummy") {
			return Config{}, errors.New("production requires live Stripe price IDs")
		}
		cfg.EnforceTLS = true
	}
	if err := requireURLScheme("MEALSWAPP_DATABASE_URL", cfg.DatabaseURL, "postgres", "postgresql"); err != nil {
		return Config{}, err
	}
	if err := requireURLScheme("MEALSWAPP_REDIS_URL", cfg.RedisURL, "redis", "rediss"); err != nil {
		return Config{}, err
	}
	if err := requireURLScheme("MEALSWAPP_FRONTEND_ORIGIN", cfg.FrontendOrigin, "http", "https"); err != nil {
		return Config{}, err
	}
	for _, origin := range cfg.AllowedOrigins {
		if err := requireURLScheme("MEALSWAPP_ALLOWED_ORIGINS", origin, "http", "https"); err != nil {
			return Config{}, err
		}
	}

	return cfg, nil
}

// loadAccountConfig validates authentication and account-flow settings.
// Implements DESIGN-006 AuthController and DESIGN-013 InputNormalizer.
func loadAccountConfig() (AccountConfig, error) {
	minLength, err := strconv.Atoi(env("MEALSWAPP_PASSWORD_MIN_LENGTH", "12"))
	if err != nil || minLength < 8 {
		return AccountConfig{}, errors.New("MEALSWAPP_PASSWORD_MIN_LENGTH must be an integer of at least 8")
	}
	accessTTL, err := positiveDuration("MEALSWAPP_ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	if err != nil {
		return AccountConfig{}, err
	}
	refreshTTL, err := positiveDuration("MEALSWAPP_REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
	if err != nil {
		return AccountConfig{}, err
	}
	if refreshTTL <= accessTTL {
		return AccountConfig{}, errors.New("MEALSWAPP_REFRESH_TOKEN_TTL must be longer than MEALSWAPP_ACCESS_TOKEN_TTL")
	}
	emailVerificationTTL, err := positiveDuration("MEALSWAPP_EMAIL_VERIFICATION_TTL", defaultEmailVerificationTTL)
	if err != nil {
		return AccountConfig{}, err
	}
	passwordResetTTL, err := positiveDuration("MEALSWAPP_PASSWORD_RESET_TTL", defaultPasswordResetTTL)
	if err != nil {
		return AccountConfig{}, err
	}
	cfg := AccountConfig{
		PasswordMinLength:           minLength,
		AccessTokenTTL:              accessTTL,
		RefreshTokenTTL:             refreshTTL,
		AccessCookieName:            env("MEALSWAPP_ACCESS_COOKIE_NAME", "mealswapp_access"),
		RefreshCookieName:           env("MEALSWAPP_REFRESH_COOKIE_NAME", "mealswapp_refresh"),
		CurrentPrivacyPolicyVersion: env("MEALSWAPP_PRIVACY_POLICY_VERSION", "dev-privacy-v1"),
		CurrentTermsVersion:         env("MEALSWAPP_TERMS_VERSION", "dev-terms-v1"),
		DisclaimerFallbackVersion:   env("MEALSWAPP_DISCLAIMER_FALLBACK_VERSION", "dev-disclaimer-v1"),
		EmailVerificationTTL:        emailVerificationTTL,
		PasswordResetTTL:            passwordResetTTL,
	}
	if strings.TrimSpace(cfg.AccessCookieName) == "" || strings.TrimSpace(cfg.RefreshCookieName) == "" || cfg.AccessCookieName == cfg.RefreshCookieName {
		return AccountConfig{}, errors.New("auth cookie names must be present and distinct")
	}
	if strings.TrimSpace(cfg.CurrentPrivacyPolicyVersion) == "" || strings.TrimSpace(cfg.CurrentTermsVersion) == "" || strings.TrimSpace(cfg.DisclaimerFallbackVersion) == "" {
		return AccountConfig{}, errors.New("account legal content versions are required")
	}
	return cfg, nil
}

// loadBillingConfig validates Stripe sandbox and production billing settings.
// Implements DESIGN-007 SubscriptionController configuration.
func loadBillingConfig(frontendOrigin string) (BillingConfig, error) {
	cfg := BillingConfig{
		StripeSecretKey:     env("MEALSWAPP_STRIPE_SECRET_KEY", "sk_test_dummy"),
		StripeWebhookSecret: env("MEALSWAPP_STRIPE_WEBHOOK_SECRET", "whsec_dummy"),
		MonthlyPlanPriceID:  env("MEALSWAPP_STRIPE_MONTHLY_PRICE_ID", "price_dummy_monthly"),
		AnnualPlanPriceID:   env("MEALSWAPP_STRIPE_ANNUAL_PRICE_ID", "price_dummy_annual"),
		CheckoutSuccessURL:  env("MEALSWAPP_STRIPE_SUCCESS_URL", frontendOrigin+"/billing/success"),
		CheckoutCancelURL:   env("MEALSWAPP_STRIPE_CANCEL_URL", frontendOrigin+"/billing/cancel"),
	}
	if err := requireURLScheme("MEALSWAPP_STRIPE_SUCCESS_URL", cfg.CheckoutSuccessURL, "http", "https"); err != nil {
		return BillingConfig{}, err
	}
	if err := requireURLScheme("MEALSWAPP_STRIPE_CANCEL_URL", cfg.CheckoutCancelURL, "http", "https"); err != nil {
		return BillingConfig{}, err
	}
	return cfg, nil
}

// positiveDuration parses a positive duration from an environment variable.
// Implements DESIGN-006 AuthController token lifetime validation.
func positiveDuration(key string, fallback time.Duration) (time.Duration, error) {
	value, err := time.ParseDuration(env(key, fallback.String()))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return value, nil
}

// splitCSV parses comma-separated gateway settings.
// Implements DESIGN-010 RequestValidator allowed-origin parsing.
func splitCSV(value string) []string {
	values := []string{}
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

// env returns the configured environment value or the provided fallback.
// Implements DESIGN-010 RequestValidator defaulting for local development.
func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// requireURLScheme verifies that a configured URL has a supported scheme and host.
// Implements DESIGN-010 RequestValidator environment-backed config validation.
func requireURLScheme(key string, value string, schemes ...string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid URL", key)
	}
	if slices.Contains(schemes, parsed.Scheme) {
		return nil
	}
	return fmt.Errorf("%s must use one of these schemes: %v", key, schemes)
}

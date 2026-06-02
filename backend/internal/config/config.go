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
	defaultHTTPPort       = "8080"
	defaultDatabaseURL    = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"
	defaultRedisURL       = "redis://localhost:6379/0"
	defaultEnvironment    = "development"
	defaultFrontendOrigin = "http://localhost:5173"
	defaultAPITimeout     = 10 * time.Second
	defaultHSTSMaxAge     = 31536000
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

	if cfg.Environment == "production" {
		if os.Getenv("MEALSWAPP_DATABASE_URL") == "" || os.Getenv("MEALSWAPP_REDIS_URL") == "" {
			return Config{}, errors.New("production requires MEALSWAPP_DATABASE_URL and MEALSWAPP_REDIS_URL")
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

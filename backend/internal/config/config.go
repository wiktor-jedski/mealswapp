package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
)

// Implements DESIGN-010 RequestValidator local development defaults.
const (
	defaultHTTPPort       = "8080"
	defaultDatabaseURL    = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"
	defaultRedisURL       = "redis://localhost:6379/0"
	defaultEnvironment    = "development"
	defaultFrontendOrigin = "http://localhost:5173"
)

// Config contains the environment-backed settings for the API and worker.
// Implements DESIGN-010 RequestValidator shared gateway configuration inputs.
type Config struct {
	HTTPPort       string
	DatabaseURL    string
	RedisURL       string
	Environment    string
	FrontendOrigin string
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

	if cfg.Environment == "production" {
		if os.Getenv("MEALSWAPP_DATABASE_URL") == "" || os.Getenv("MEALSWAPP_REDIS_URL") == "" {
			return Config{}, errors.New("production requires MEALSWAPP_DATABASE_URL and MEALSWAPP_REDIS_URL")
		}
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

	return cfg, nil
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

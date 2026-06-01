package config

// Implements DESIGN-010 RequestValidator configuration verification.

import "testing"

// TestLoadUsesDevelopmentDefaults proves that config will use default values
// if no other values are passed.
// TestLoadUsesDevelopmentDefaults verifies DESIGN-010 RequestValidator config defaults.
func TestLoadUsesDevelopmentDefaults(t *testing.T) {
	t.Setenv("MEALSWAPP_HTTP_PORT", "")
	t.Setenv("MEALSWAPP_DATABASE_URL", "")
	t.Setenv("MEALSWAPP_REDIS_URL", "")
	t.Setenv("MEALSWAPP_ENV", "")
	t.Setenv("MEALSWAPP_FRONTEND_ORIGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPPort != defaultHTTPPort {
		t.Fatalf("HTTPPort = %q, want %q", cfg.HTTPPort, defaultHTTPPort)
	}
	if cfg.Environment != defaultEnvironment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, defaultEnvironment)
	}
}

// TestLoadRequiresProductionDependencyURLs proves that config will not load
// if prod env lacks valid URLs.
// TestLoadRequiresProductionDependencyURLs verifies DESIGN-010 RequestValidator production guards.
func TestLoadRequiresProductionDependencyURLs(t *testing.T) {
	t.Setenv("MEALSWAPP_ENV", "production")
	t.Setenv("MEALSWAPP_DATABASE_URL", "")
	t.Setenv("MEALSWAPP_REDIS_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want production dependency URL error")
	}
}

// TestLoadAcceptsProductionDependencyURLs proves that config app will load
// if all necessary URLs are passed.
// TestLoadAcceptsProductionDependencyURLs verifies DESIGN-010 RequestValidator production overrides.
func TestLoadAcceptsProductionDependencyURLs(t *testing.T) {
	t.Setenv("MEALSWAPP_ENV", "production")
	t.Setenv("MEALSWAPP_DATABASE_URL", "postgres://example")
	t.Setenv("MEALSWAPP_REDIS_URL", "redis://example:6379/0")
	t.Setenv("MEALSWAPP_HTTP_PORT", "9090")
	t.Setenv("MEALSWAPP_FRONTEND_ORIGIN", "https://example.test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPPort != "9090" || cfg.FrontendOrigin != "https://example.test" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

// TestLoadRejectsInvalidRedisURL checks if the string is indeed a valid redis URL.
// TestLoadRejectsInvalidRedisURL verifies DESIGN-010 RequestValidator Redis URL validation.
func TestLoadRejectsInvalidRedisURL(t *testing.T) {
	t.Setenv("MEALSWAPP_REDIS_URL", "not a redis url")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want Redis URL validation error")
	}
}

// TestLoadRejectsInvalidDatabaseURL checks if the string is indeed a db URL.
// TestLoadRejectsInvalidDatabaseURL verifies DESIGN-010 RequestValidator database URL validation.
func TestLoadRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("MEALSWAPP_DATABASE_URL", "https://example.test")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want database URL validation error")
	}
}

// TestLoadRejectsInvalidFrontendOrigin verifies DESIGN-010 RequestValidator frontend origin validation.
func TestLoadRejectsInvalidFrontendOrigin(t *testing.T) {
	t.Setenv("MEALSWAPP_FRONTEND_ORIGIN", "not a frontend origin")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want frontend origin validation error")
	}
}

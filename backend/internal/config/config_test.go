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
	if cfg.HSTSMaxAge != defaultHSTSMaxAge {
		t.Fatalf("HSTSMaxAge = %d, want %d", cfg.HSTSMaxAge, defaultHSTSMaxAge)
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

// TestLoadRejectsInvalidGatewaySettings verifies DESIGN-010 RequestValidator gateway value validation.
func TestLoadRejectsInvalidGatewaySettings(t *testing.T) {
	for key, value := range map[string]string{
		"MEALSWAPP_API_TIMEOUT":     "bad",
		"MEALSWAPP_TRUST_PROXY":     "bad",
		"MEALSWAPP_ENFORCE_TLS":     "bad",
		"MEALSWAPP_HSTS_MAX_AGE":    "-1",
		"MEALSWAPP_ALLOWED_ORIGINS": "bad",
		"MEALSWAPP_TLS_MIN_VERSION": "1.2",
	} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted %s=%q", key, value)
			}
		})
	}
}

// TestLoadRejectsEmptyAllowedOrigins verifies DESIGN-010 CORSHandler origin-list validation.
func TestLoadRejectsEmptyAllowedOrigins(t *testing.T) {
	t.Setenv("MEALSWAPP_ALLOWED_ORIGINS", ", ")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an empty allowed-origin list")
	}
}

// TestLoadRejectsTrustedProxyUntilIngressExists verifies DESIGN-013 TLSEnforcer deployment deferral.
func TestLoadRejectsTrustedProxyUntilIngressExists(t *testing.T) {
	t.Setenv("MEALSWAPP_TRUST_PROXY", "true")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted trusted-proxy mode before Phase 09 ingress enforcement")
	}
}

// TestLoadAcceptsHSTSMaxAgeOverride verifies DESIGN-010 SecurityHeaderMiddleware HSTS configuration.
func TestLoadAcceptsHSTSMaxAgeOverride(t *testing.T) {
	t.Setenv("MEALSWAPP_HSTS_MAX_AGE", "0")
	cfg, err := Load()
	if err != nil || cfg.HSTSMaxAge != 0 {
		t.Fatalf("Load() = %+v, %v", cfg, err)
	}
}

// TestLoadAcceptsMultipleOrigins verifies DESIGN-010 CORSHandler origin-list parsing.
func TestLoadAcceptsMultipleOrigins(t *testing.T) {
	t.Setenv("MEALSWAPP_ALLOWED_ORIGINS", "https://one.test, https://two.test")
	cfg, err := Load()
	if err != nil || len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("Load() = %+v, %v", cfg, err)
	}
}

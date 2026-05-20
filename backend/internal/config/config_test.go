package config

import "testing"

func TestLoadResolvesDependencyURLsFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "redis://example")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://staging.example.com")

	cfg := Load()

	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("expected database URL from env, got %q", cfg.DatabaseURL)
	}

	if cfg.RedisURL != "redis://example" {
		t.Fatalf("expected redis URL from env, got %q", cfg.RedisURL)
	}

	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "https://app.example.com" || cfg.CORSOrigins[1] != "https://staging.example.com" {
		t.Fatalf("expected CORS origins from env, got %#v", cfg.CORSOrigins)
	}
}

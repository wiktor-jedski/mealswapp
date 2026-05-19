package config

import "testing"

func TestLoadResolvesDependencyURLsFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "redis://example")

	cfg := Load()

	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("expected database URL from env, got %q", cfg.DatabaseURL)
	}

	if cfg.RedisURL != "redis://example" {
		t.Fatalf("expected redis URL from env, got %q", cfg.RedisURL)
	}
}

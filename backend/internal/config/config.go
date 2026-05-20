package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	APIAddr         string
	Environment     string
	DatabaseURL     string
	RedisURL        string
	CORSOrigins     []string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		APIAddr:         valueOrDefault("API_ADDR", ":8080"),
		Environment:     valueOrDefault("APP_ENV", "local"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisURL:        os.Getenv("REDIS_URL"),
		CORSOrigins:     splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		ShutdownTimeout: 5 * time.Second,
	}
}

func valueOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}

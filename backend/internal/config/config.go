package config

import (
	"os"
	"time"
)

type Config struct {
	APIAddr         string
	Environment     string
	DatabaseURL     string
	RedisURL        string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		APIAddr:         valueOrDefault("API_ADDR", ":8080"),
		Environment:     valueOrDefault("APP_ENV", "local"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisURL:        os.Getenv("REDIS_URL"),
		ShutdownTimeout: 5 * time.Second,
	}
}

func valueOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

package main

import (
	"context"
	"log/slog"
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/seed"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := seed.Apply(ctx, pool); err != nil {
		slog.Error("seed database", "error", err)
		os.Exit(1)
	}

	slog.Info("seed data applied")
}

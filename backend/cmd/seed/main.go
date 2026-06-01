package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/seed"
)

// main loads configuration, connects to PostgreSQL, and runs development seeding.
// Implements DESIGN-005 MicronutrientVocabulary.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer conn.Close(ctx)

	if err := seed.Run(ctx, conn); err != nil {
		log.Fatalf("run seed: %v", err)
	}
}

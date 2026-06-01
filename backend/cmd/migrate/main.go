package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
)

// main runs the requested database migration direction.
// Implements DESIGN-005 RepositoryInterfaces migration command bootstrap.
func main() {
	// determine direction, default up
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if direction != "up" && direction != "down" {
		log.Fatalf("usage: go run ./cmd/migrate [up|down]")
	}

	// load env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// create root context
	ctx := context.Background()

	// create live database connection
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer conn.Close(ctx)

	// now we can migrate
	if err := migrations.Run(ctx, conn, direction, "../database/migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
}

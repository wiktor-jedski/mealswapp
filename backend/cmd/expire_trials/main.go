// Package main provides the trial expiry command.
// Implements DESIGN-007 TrialTracker.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/database"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// main runs the expiry of trials idempotently.
// Implements DESIGN-007 TrialTracker.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pg.Close()

	entRepo := repository.NewPostgresEntitlementRepository(pg)
	tracker := entitlement.NewTrialTracker(entRepo, entRepo, time.Now)

	if err := tracker.ExpireTrials(ctx); err != nil {
		log.Fatalf("failed to expire trials: %v", err)
	}
	log.Println("successfully expired trials")
}

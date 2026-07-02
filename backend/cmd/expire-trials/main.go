package main

import (
	"context"
	"log"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/database"
	"github.com/wiktor-jedski/mealswapp/backend/internal/entitlement"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// main runs the trial-expiry downgrade command.
// Implements DESIGN-007 TrialTracker expiry command entrypoint.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	pg, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pg.Close()

	repo := repository.NewPostgresEntitlementRepository(pg)
	if err := runExpireTrials(ctx, repo, time.Now().UTC()); err != nil {
		log.Fatalf("expire trials: %v", err)
	}
	log.Print("trial expiry completed")
}

// runExpireTrials executes the command logic with an injected repository for tests.
// Implements DESIGN-007 TrialTracker expiry command entrypoint.
func runExpireTrials(ctx context.Context, repo interface {
	repository.EntitlementRepository
	repository.TrialRepository
}, now time.Time) error {
	return entitlement.NewTrialTracker(repo, repo).ExpireTrials(ctx, now)
}

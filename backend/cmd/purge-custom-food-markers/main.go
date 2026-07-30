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
	"github.com/wiktor-jedski/mealswapp/backend/internal/maintenance"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// main runs the production retry-marker maintenance scheduler.
// Implements DESIGN-008 AccountDeleter marker retention scheduler.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	repo := repository.NewPostgresCustomFoodItemRepository(db)
	if err := maintenance.RunDeletedCustomFoodCreateKeyPurger(ctx, repo, 15*time.Minute); err != nil {
		log.Fatalf("purge custom food markers: %v", err)
	}
}

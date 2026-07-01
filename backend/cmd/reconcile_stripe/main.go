// Package main provides the Stripe reconciliation command.
// Implements DESIGN-007 EntitlementManager.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/database"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

// main runs the reconciliation of Stripe subscriptions.
// Implements DESIGN-007 EntitlementManager.
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
	manager := subscription.NewEntitlementManager(entRepo)
	gateway := subscription.NewStripeCheckoutGateway(cfg)

	if err := manager.ReconcileStripeEntitlements(ctx, gateway); err != nil {
		log.Fatalf("failed to reconcile stripe entitlements: %v", err)
	}
	log.Println("successfully reconciled stripe entitlements")
}

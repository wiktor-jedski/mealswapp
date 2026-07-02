package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/database"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/subscription"
)

// main runs the hourly Stripe entitlement reconciliation job.
// Implements DESIGN-007 StripeWebhookHandler hourly reconciliation job.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pg, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pg.Close()

	service := subscription.NewReconciliationService(
		subscription.NewStripeSubscriptionGateway(cfg.Billing.StripeSecretKey, nil),
		repository.NewPostgresEntitlementRepository(pg),
		observability.JSONSink{Writer: os.Stdout},
	)
	if err := service.RunHourly(ctx); err != nil {
		log.Fatalf("stripe reconciliation stopped: %v", err)
	}
}

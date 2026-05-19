package main

import (
	"context"
	"errors"
	"log/slog"
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/worker"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	idle := os.Getenv("WORKER_IDLE") == "true"
	if err := worker.New(cfg).Run(ctx, idle); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker failed", "error", err)
		os.Exit(1)
	}
}

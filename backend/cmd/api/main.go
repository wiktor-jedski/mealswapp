package main

import (
	"context"
	"log/slog"
	"mealswapp/backend/internal/config"
	apihttp "mealswapp/backend/internal/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Load()
	app := apihttp.NewRouter(apihttp.ServiceDependencies{Config: cfg})

	errs := make(chan error, 1)
	go func() {
		errs <- app.Listen(cfg.APIAddr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errs:
		if err != nil {
			slog.Error("api server failed", "error", err)
			os.Exit(1)
		}
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			slog.Error("api server shutdown failed", "error", err)
			os.Exit(1)
		}
	}
}

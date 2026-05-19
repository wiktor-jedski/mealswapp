package worker

import (
	"context"
	"log/slog"
	"time"

	"mealswapp/backend/internal/config"
)

type Worker struct {
	config config.Config
}

type HookStatus struct {
	Name   string
	Status string
}

func New(cfg config.Config) Worker {
	return Worker{config: cfg}
}

func (w Worker) Initialize(ctx context.Context) ([]HookStatus, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return []HookStatus{
		{Name: "redis", Status: hookStatus(w.config.RedisURL)},
		{Name: "jobs", Status: "noop"},
	}, nil
}

func (w Worker) Run(ctx context.Context, idle bool) error {
	statuses, err := w.Initialize(ctx)
	if err != nil {
		return err
	}

	for _, status := range statuses {
		slog.Info("worker hook initialized", "name", status.Name, "status", status.Status)
	}

	if !idle {
		return nil
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			slog.Debug("worker idle heartbeat")
		}
	}
}

func hookStatus(value string) string {
	if value == "" {
		return "not_configured"
	}

	return "configured"
}

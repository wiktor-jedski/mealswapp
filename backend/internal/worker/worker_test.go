package worker

import (
	"context"
	"testing"

	"mealswapp/backend/internal/config"
)

func TestInitializeReportsNoopHooks(t *testing.T) {
	w := New(config.Config{RedisURL: "redis://localhost:6379/0"})

	statuses, err := w.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 hook statuses, got %d", len(statuses))
	}

	if statuses[0].Name != "redis" || statuses[0].Status != "configured" {
		t.Fatalf("unexpected redis status: %#v", statuses[0])
	}

	if statuses[1].Name != "jobs" || statuses[1].Status != "noop" {
		t.Fatalf("unexpected jobs status: %#v", statuses[1])
	}
}

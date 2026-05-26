package worker

// Implements DESIGN-004 JobQueueManager worker lifecycle verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/mealswapp/mealswapp/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// TestRunReturnsPingError verifies DESIGN-004 JobQueueManager Redis startup failure behavior.
func TestRunReturnsPingError(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	defer client.Close()

	err := Run(context.Background(), config.Config{Environment: "test"}, client)
	if err == nil {
		t.Fatal("Run() error = nil, want ping error")
	}
}

// TestRunAfterPingStopsWhenContextIsCanceled verifies DESIGN-004 JobQueueManager graceful shutdown behavior.
func TestRunAfterPingStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runAfterPing(ctx, config.Config{Environment: "test"}, func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("runAfterPing() error = %v", err)
	}
}

// TestRunAfterPingReturnsPingError verifies DESIGN-004 JobQueueManager ping failure behavior.
func TestRunAfterPingReturnsPingError(t *testing.T) {
	expected := errors.New("redis down")

	err := runAfterPing(context.Background(), config.Config{Environment: "test"}, func(context.Context) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("runAfterPing() error = %v, want %v", err, expected)
	}
}

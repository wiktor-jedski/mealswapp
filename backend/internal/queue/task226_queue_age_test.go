package queue

// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector Task 226 queue-age verification.

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestTask226StatsSeparateWaitingAndPendingAges(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	ctx := context.Background()
	if err := manager.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	now := time.Now()
	pendingID := addTask226Delivery(t, ctx, client, manager.Config().Stream, now.Add(-10*time.Second))
	pendingJob, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if pendingJob.EntryID != pendingID {
		t.Fatalf("pending entry = %q, want %q", pendingJob.EntryID, pendingID)
	}
	addTask226Delivery(t, ctx, client, manager.Config().Stream, now.Add(-time.Second))
	time.Sleep(20 * time.Millisecond)

	stats, err := manager.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.QueueDepth != 2 || stats.PendingDepth != 1 {
		t.Fatalf("depths = %#v, want total 2 and pending 1", stats)
	}
	if stats.OldestQueuedAge < 900*time.Millisecond || stats.OldestQueuedAge > 3*time.Second {
		t.Fatalf("oldest queued age = %s, want waiting entry age near 1s", stats.OldestQueuedAge)
	}
	if stats.OldestPendingAge < 10*time.Millisecond || stats.OldestPendingAge > time.Second {
		t.Fatalf("oldest pending age = %s, want Redis idle duration near 20ms", stats.OldestPendingAge)
	}
}

func TestTask226StatsPopulationAndClockSkewFixtures(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		manager := newIntegrationQueue(t, client)
		stats := task226Stats(t, ctx, manager)
		if stats.QueueDepth != 0 || stats.PendingDepth != 0 || stats.OldestQueuedAge != 0 || stats.OldestPendingAge != 0 {
			t.Fatalf("empty stats = %#v, want zero values", stats)
		}
	})

	t.Run("queued only", func(t *testing.T) {
		manager := newIntegrationQueue(t, client)
		if err := manager.Bootstrap(ctx); err != nil {
			t.Fatalf("Bootstrap() error = %v", err)
		}
		addTask226Delivery(t, ctx, client, manager.Config().Stream, time.Now().Add(-2*time.Second))
		stats := task226Stats(t, ctx, manager)
		if stats.QueueDepth != 1 || stats.PendingDepth != 0 || stats.OldestQueuedAge < 1900*time.Millisecond || stats.OldestPendingAge != 0 {
			t.Fatalf("queued-only stats = %#v", stats)
		}
	})

	t.Run("pending only uses Redis idle", func(t *testing.T) {
		manager := newIntegrationQueue(t, client)
		if err := manager.Bootstrap(ctx); err != nil {
			t.Fatalf("Bootstrap() error = %v", err)
		}
		now := time.Now()
		addTask226Delivery(t, ctx, client, manager.Config().Stream, now.Add(-20*time.Second))
		if _, err := manager.Reserve(ctx); err != nil {
			t.Fatalf("first Reserve() error = %v", err)
		}
		time.Sleep(30 * time.Millisecond)
		addTask226Delivery(t, ctx, client, manager.Config().Stream, now.Add(-19*time.Second))
		if _, err := manager.Reserve(ctx); err != nil {
			t.Fatalf("second Reserve() error = %v", err)
		}
		stats := task226Stats(t, ctx, manager)
		if stats.QueueDepth != 2 || stats.PendingDepth != 2 || stats.OldestQueuedAge != 0 {
			t.Fatalf("pending-only populations = %#v", stats)
		}
		if stats.OldestPendingAge < 25*time.Millisecond || stats.OldestPendingAge > time.Second {
			t.Fatalf("oldest pending age = %s, want longest Redis idle rather than 20s stream age", stats.OldestPendingAge)
		}
	})

	t.Run("future Redis stream clock clamps waiting age", func(t *testing.T) {
		manager := newIntegrationQueue(t, client)
		if err := manager.Bootstrap(ctx); err != nil {
			t.Fatalf("Bootstrap() error = %v", err)
		}
		addTask226Delivery(t, ctx, client, manager.Config().Stream, time.Now().Add(time.Minute))
		stats := task226Stats(t, ctx, manager)
		if stats.QueueDepth != 1 || stats.PendingDepth != 0 || stats.OldestQueuedAge != 0 || stats.OldestPendingAge != 0 {
			t.Fatalf("skewed-clock stats = %#v, want nonnegative zero age", stats)
		}
	})
}

func task226Stats(t *testing.T, ctx context.Context, manager *JobQueueManager) QueueStats {
	t.Helper()
	stats, err := manager.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	return stats
}

func addTask226Delivery(t *testing.T, ctx context.Context, client redis.UniversalClient, stream string, createdAt time.Time) string {
	t.Helper()
	entryID := strconv.FormatInt(createdAt.UnixMilli(), 10) + "-0"
	result, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		ID:     entryID,
		Values: []string{jobIDField, uuid.NewString(), enqueuedAtField, strconv.FormatInt(createdAt.UnixMilli(), 10)},
	}).Result()
	if err != nil {
		t.Fatalf("XAdd(%q) error = %v", entryID, err)
	}
	return result
}

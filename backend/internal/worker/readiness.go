package worker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Implements DESIGN-014 UptimeMonitor and DESIGN-004 JobQueueManager worker readiness.
const (
	// WorkerHeartbeatKey stores dedicated optimizer worker readiness heartbeats.
	WorkerHeartbeatKey      = "mealswapp:optimization:worker:heartbeat:v1"
	workerHeartbeatInterval = 5 * time.Second
	workerHeartbeatTTL      = 15 * time.Second
)

// OptimizationWorkerPing reports whether at least one dedicated optimization
// worker has refreshed its Redis heartbeat recently.
// Implements DESIGN-014 UptimeMonitor.
func OptimizationWorkerPing(redisClient redis.UniversalClient) func(context.Context) error {
	return func(ctx context.Context) error {
		if redisClient == nil {
			return errors.New("optimization worker Redis client is required")
		}
		minimum := strconv.FormatInt(time.Now().Add(-workerHeartbeatTTL).UnixMilli(), 10)
		count, err := redisClient.ZCount(ctx, WorkerHeartbeatKey, minimum, "+inf").Result()
		if err != nil {
			return fmt.Errorf("read optimization worker heartbeat: %w", err)
		}
		if count == 0 {
			return errors.New("optimization worker heartbeat unavailable")
		}
		return nil
	}
}

// startWorkerHeartbeat starts the dedicated worker's bounded Redis liveness marker.
// Implements DESIGN-014 UptimeMonitor.
func startWorkerHeartbeat(ctx context.Context, redisClient redis.UniversalClient, consumer string) (func(), error) {
	if redisClient == nil {
		return nil, errors.New("worker Redis client is required")
	}
	if err := writeWorkerHeartbeat(ctx, redisClient, consumer); err != nil {
		return nil, err
	}
	heartbeatCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(workerHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				_ = writeWorkerHeartbeat(heartbeatCtx, redisClient, consumer)
			}
		}
	}()
	return func() {
		cancel()
		<-done
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
		defer cleanupCancel()
		_, _ = redisClient.ZRem(cleanupCtx, WorkerHeartbeatKey, consumer).Result()
	}, nil
}

// writeWorkerHeartbeat refreshes one non-PII worker member and removes stale members.
// Implements DESIGN-014 UptimeMonitor.
func writeWorkerHeartbeat(ctx context.Context, redisClient redis.UniversalClient, consumer string) error {
	minimum := strconv.FormatInt(time.Now().Add(-workerHeartbeatTTL).UnixMilli(), 10)
	if err := redisClient.ZRemRangeByScore(ctx, WorkerHeartbeatKey, "-inf", minimum).Err(); err != nil {
		return err
	}
	if err := redisClient.ZAdd(ctx, WorkerHeartbeatKey, redis.Z{Score: float64(time.Now().UnixMilli()), Member: consumer}).Err(); err != nil {
		return err
	}
	return redisClient.Expire(ctx, WorkerHeartbeatKey, workerHeartbeatTTL).Err()
}

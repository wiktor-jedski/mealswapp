// Package worker contains the optimization admission and processing boundaries.
// Implements DESIGN-004 JobStatusTracker.
package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
)

// Implements DESIGN-004 JobStatusTracker admission defaults.
const (
	// DefaultOptimizationRateLimit bounds newly accepted jobs per fixed hour.
	DefaultOptimizationRateLimit = int64(10)
	// DefaultOptimizationActiveTTL bounds a leaked active-job reservation.
	DefaultOptimizationActiveTTL = time.Hour
)

// OptimizationAdmissionStatus identifies one atomic admission outcome.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationAdmissionStatus string

// Implements DESIGN-004 JobStatusTracker admission outcomes.
const (
	// OptimizationAdmissionAcquired means this request owns the user's active slot.
	OptimizationAdmissionAcquired OptimizationAdmissionStatus = "acquired"
	// OptimizationAdmissionReplay means an identical in-flight request owns the slot.
	OptimizationAdmissionReplay OptimizationAdmissionStatus = "replay"
	// OptimizationAdmissionConflict means the idempotency key was reused with another body.
	OptimizationAdmissionConflict OptimizationAdmissionStatus = "conflict"
	// OptimizationAdmissionActive means the user already has another active job.
	OptimizationAdmissionActive OptimizationAdmissionStatus = "active"
	// OptimizationAdmissionRateLimited means the fixed-hour allowance is exhausted.
	OptimizationAdmissionRateLimited OptimizationAdmissionStatus = "rate_limited"
)

// OptimizationAdmissionRequest contains server-derived admission identity.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationAdmissionRequest struct {
	UserID         uuid.UUID
	JobID          uuid.UUID
	IdempotencyKey string
	BodyHash       string
	CountRate      bool
}

// OptimizationAdmissionDecision reports whether and how submission may proceed.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationAdmissionDecision struct {
	Status     OptimizationAdmissionStatus
	JobID      uuid.UUID
	RetryAfter time.Duration
}

// OptimizationAdmissionGate reserves and releases per-user optimization capacity.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationAdmissionGate interface {
	Acquire(context.Context, OptimizationAdmissionRequest) (OptimizationAdmissionDecision, error)
	Release(context.Context, uuid.UUID, uuid.UUID) error
}

// OptimizationAdmissionConfig configures the Go/Redis admission policy.
// Implements DESIGN-004 JobStatusTracker.
type OptimizationAdmissionConfig struct {
	RateLimit int64
	ActiveTTL time.Duration
	Now       func() time.Time
}

// RedisOptimizationAdmissionGate enforces one active job and a fixed-hour rate limit.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
type RedisOptimizationAdmissionGate struct {
	client redis.UniversalClient
	config OptimizationAdmissionConfig
}

// NewRedisOptimizationAdmissionGate creates a fail-closed Redis admission gate.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
func NewRedisOptimizationAdmissionGate(client redis.UniversalClient, config OptimizationAdmissionConfig) *RedisOptimizationAdmissionGate {
	if config.RateLimit == 0 {
		config.RateLimit = DefaultOptimizationRateLimit
	}
	if config.ActiveTTL == 0 {
		config.ActiveTTL = DefaultOptimizationActiveTTL
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &RedisOptimizationAdmissionGate{client: client, config: config}
}

// Acquire atomically reserves the single active slot, then counts one fixed-hour acceptance.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
func (g *RedisOptimizationAdmissionGate) Acquire(ctx context.Context, req OptimizationAdmissionRequest) (OptimizationAdmissionDecision, error) {
	if err := g.validate(req); err != nil {
		return OptimizationAdmissionDecision{}, err
	}
	now := g.config.Now().UTC()
	keyHash := hashAdmissionValue(req.IdempotencyKey)
	value := req.JobID.String() + "|" + keyHash + "|" + req.BodyHash
	activeKey := optimizationAdmissionActiveKey(req.UserID)
	acquired, err := g.client.SetNX(ctx, activeKey, value, g.config.ActiveTTL).Result()
	if err != nil {
		return OptimizationAdmissionDecision{}, admissionUnavailable("reserve active optimization job", err)
	}
	if !acquired {
		return g.existingDecision(ctx, activeKey, keyHash, req.BodyHash)
	}
	if !req.CountRate {
		return OptimizationAdmissionDecision{Status: OptimizationAdmissionAcquired, JobID: req.JobID}, nil
	}

	hourStart := now.Truncate(time.Hour)
	rateKey := optimizationAdmissionRateKey(req.UserID, hourStart)
	var count *redis.IntCmd
	_, err = g.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		count = pipe.Incr(ctx, rateKey)
		pipe.ExpireAt(ctx, rateKey, hourStart.Add(time.Hour))
		return nil
	})
	if err != nil {
		_ = g.Release(context.WithoutCancel(ctx), req.UserID, req.JobID)
		return OptimizationAdmissionDecision{}, admissionUnavailable("count optimization admission", err)
	}
	if count.Val() > g.config.RateLimit {
		if err := g.Release(context.WithoutCancel(ctx), req.UserID, req.JobID); err != nil {
			return OptimizationAdmissionDecision{}, err
		}
		return OptimizationAdmissionDecision{Status: OptimizationAdmissionRateLimited, RetryAfter: hourStart.Add(time.Hour).Sub(now)}, nil
	}
	return OptimizationAdmissionDecision{Status: OptimizationAdmissionAcquired, JobID: req.JobID}, nil
}

// Release deletes the active slot only when it still belongs to the supplied job.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
func (g *RedisOptimizationAdmissionGate) Release(ctx context.Context, userID, jobID uuid.UUID) error {
	if g == nil || g.client == nil || userID == uuid.Nil || jobID == uuid.Nil {
		return admissionUnavailable("release active optimization job", errors.New("admission dependencies and identifiers are required"))
	}
	key := optimizationAdmissionActiveKey(userID)
	for attempts := 0; attempts < 3; attempts++ {
		err := g.client.Watch(ctx, func(tx *redis.Tx) error {
			value, err := tx.Get(ctx, key).Result()
			if errors.Is(err, redis.Nil) {
				return nil
			}
			if err != nil {
				return err
			}
			parts := strings.Split(value, "|")
			if len(parts) != 3 || parts[0] != jobID.String() {
				return nil
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, key)
				return nil
			})
			return err
		}, key)
		if !errors.Is(err, redis.TxFailedErr) {
			if err != nil {
				return admissionUnavailable("release active optimization job", err)
			}
			return nil
		}
	}
	return admissionUnavailable("release active optimization job", redis.TxFailedErr)
}

// existingDecision classifies a reservation held by an in-flight request.
// Implements DESIGN-004 JobStatusTracker.
func (g *RedisOptimizationAdmissionGate) existingDecision(ctx context.Context, activeKey, keyHash, bodyHash string) (OptimizationAdmissionDecision, error) {
	value, err := g.client.Get(ctx, activeKey).Result()
	if err != nil {
		return OptimizationAdmissionDecision{}, admissionUnavailable("read active optimization job", err)
	}
	parts := strings.Split(value, "|")
	if len(parts) != 3 {
		return OptimizationAdmissionDecision{}, admissionUnavailable("read active optimization job", errors.New("active reservation is malformed"))
	}
	jobID, err := uuid.Parse(parts[0])
	if err != nil {
		return OptimizationAdmissionDecision{}, admissionUnavailable("read active optimization job", err)
	}
	if parts[1] == keyHash {
		if parts[2] != bodyHash {
			return OptimizationAdmissionDecision{Status: OptimizationAdmissionConflict, JobID: jobID}, nil
		}
		return OptimizationAdmissionDecision{Status: OptimizationAdmissionReplay, JobID: jobID}, nil
	}
	ttl, err := g.client.TTL(ctx, activeKey).Result()
	if err != nil {
		return OptimizationAdmissionDecision{}, admissionUnavailable("read active optimization TTL", err)
	}
	return OptimizationAdmissionDecision{Status: OptimizationAdmissionActive, JobID: jobID, RetryAfter: ttl}, nil
}

// validate checks fail-closed configuration and server-derived identifiers.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RequestValidator.
func (g *RedisOptimizationAdmissionGate) validate(req OptimizationAdmissionRequest) error {
	if g == nil || g.client == nil || g.config.RateLimit <= 0 || g.config.ActiveTTL <= 0 || g.config.Now == nil {
		return admissionUnavailable("validate optimization admission", errors.New("admission configuration is invalid"))
	}
	if req.UserID == uuid.Nil || req.JobID == uuid.Nil || strings.TrimSpace(req.IdempotencyKey) == "" {
		return errors.New("optimization admission identifiers are required")
	}
	if len(req.BodyHash) != sha256.Size*2 {
		return errors.New("optimization admission body hash is invalid")
	}
	if _, err := hex.DecodeString(req.BodyHash); err != nil {
		return errors.New("optimization admission body hash is invalid")
	}
	return nil
}

// optimizationAdmissionActiveKey derives a pseudonymous per-user slot key.
// Implements DESIGN-004 JobStatusTracker.
func optimizationAdmissionActiveKey(userID uuid.UUID) string {
	return "mealswapp:optimization:admission:active:v1:" + hashAdmissionValue(userID.String())
}

// optimizationAdmissionRateKey derives a pseudonymous fixed-hour counter key.
// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter.
func optimizationAdmissionRateKey(userID uuid.UUID, hour time.Time) string {
	return fmt.Sprintf("mealswapp:optimization:admission:rate:v1:%s:%d", hashAdmissionValue(userID.String()), hour.Unix())
}

// hashAdmissionValue pseudonymizes admission key material with SHA-256.
// Implements DESIGN-004 JobStatusTracker.
func hashAdmissionValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// admissionUnavailable preserves fail-closed queue-unavailable classification.
// Implements DESIGN-004 JobStatusTracker.
func admissionUnavailable(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", queue.ErrQueueUnavailable, operation, err)
}

// Implements DESIGN-004 JobStatusTracker compile-time admission contract.
var _ OptimizationAdmissionGate = (*RedisOptimizationAdmissionGate)(nil)

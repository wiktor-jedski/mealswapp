package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Implements DESIGN-004 JobStatusTracker and DESIGN-010 RateLimiter admission verification.
func TestRedisOptimizationAdmissionEnforcesActiveAndFixedHourLimits(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	now := time.Now().UTC().Truncate(time.Second)
	gate := NewRedisOptimizationAdmissionGate(client, OptimizationAdmissionConfig{RateLimit: 10, ActiveTTL: time.Hour, Now: func() time.Time { return now }})
	userID := uuid.New()
	activeKey := optimizationAdmissionActiveKey(userID)
	rateKey := optimizationAdmissionRateKey(userID, now.Truncate(time.Hour))
	t.Cleanup(func() { _ = client.Del(context.Background(), activeKey, rateKey).Err() })
	if strings.Contains(activeKey, userID.String()) {
		t.Fatal("active admission key exposes the raw user ID")
	}

	firstJob := uuid.New()
	first := OptimizationAdmissionRequest{UserID: userID, JobID: firstJob, IdempotencyKey: "admission-key-1", BodyHash: admissionTestHash("body-1"), CountRate: true}
	decision, err := gate.Acquire(context.Background(), first)
	if err != nil || decision.Status != OptimizationAdmissionAcquired {
		t.Fatalf("first Acquire() = %+v, %v", decision, err)
	}
	replay, err := gate.Acquire(context.Background(), OptimizationAdmissionRequest{UserID: userID, JobID: uuid.New(), IdempotencyKey: first.IdempotencyKey, BodyHash: first.BodyHash})
	if err != nil || replay.Status != OptimizationAdmissionReplay || replay.JobID != firstJob {
		t.Fatalf("replay Acquire() = %+v, %v", replay, err)
	}
	conflict, err := gate.Acquire(context.Background(), OptimizationAdmissionRequest{UserID: userID, JobID: uuid.New(), IdempotencyKey: first.IdempotencyKey, BodyHash: admissionTestHash("changed")})
	if err != nil || conflict.Status != OptimizationAdmissionConflict {
		t.Fatalf("conflict Acquire() = %+v, %v", conflict, err)
	}
	active, err := gate.Acquire(context.Background(), OptimizationAdmissionRequest{UserID: userID, JobID: uuid.New(), IdempotencyKey: "another-key", BodyHash: admissionTestHash("body-2")})
	if err != nil || active.Status != OptimizationAdmissionActive || active.RetryAfter <= 0 {
		t.Fatalf("active Acquire() = %+v, %v", active, err)
	}
	if err := gate.Release(context.Background(), userID, uuid.New()); err != nil {
		t.Fatalf("wrong-owner Release() error = %v", err)
	}
	if exists, _ := client.Exists(context.Background(), activeKey).Result(); exists != 1 {
		t.Fatal("wrong job released the active slot")
	}
	if err := gate.Release(context.Background(), userID, firstJob); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	for accepted := 2; accepted <= 10; accepted++ {
		jobID := uuid.New()
		decision, err = gate.Acquire(context.Background(), OptimizationAdmissionRequest{UserID: userID, JobID: jobID, IdempotencyKey: uuid.NewString(), BodyHash: admissionTestHash(uuid.NewString()), CountRate: true})
		if err != nil || decision.Status != OptimizationAdmissionAcquired {
			t.Fatalf("Acquire() number %d = %+v, %v", accepted, decision, err)
		}
		if err := gate.Release(context.Background(), userID, jobID); err != nil {
			t.Fatalf("Release() number %d error = %v", accepted, err)
		}
	}
	limitedJob := uuid.New()
	limited, err := gate.Acquire(context.Background(), OptimizationAdmissionRequest{UserID: userID, JobID: limitedJob, IdempotencyKey: uuid.NewString(), BodyHash: admissionTestHash("limited"), CountRate: true})
	wantRetryAfter := now.Truncate(time.Hour).Add(time.Hour).Sub(now)
	if err != nil || limited.Status != OptimizationAdmissionRateLimited || limited.RetryAfter != wantRetryAfter {
		t.Fatalf("limited Acquire() = %+v, %v", limited, err)
	}
	if exists, _ := client.Exists(context.Background(), activeKey).Result(); exists != 0 {
		t.Fatal("rate-limited admission retained an active slot")
	}
}

func admissionTestHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

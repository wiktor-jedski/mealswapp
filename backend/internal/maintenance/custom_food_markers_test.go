package maintenance

import (
	"context"
	"errors"
	"testing"
	"time"
)

type markerRepo struct {
	calls int
	err   error
}

func (r *markerRepo) PurgeExpiredDeletedCustomFoodCreateKeys(context.Context) error {
	r.calls++
	return r.err
}

// Implements DESIGN-008 AccountDeleter marker retention scheduler verification.
func TestRunDeletedCustomFoodCreateKeyPurgerRunsImmediatelyAndOnTick(t *testing.T) {
	repo := &markerRepo{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunDeletedCustomFoodCreateKeyPurger(ctx, repo, time.Millisecond) }()
	deadline := time.After(time.Second)
	for repo.calls < 2 {
		select {
		case <-deadline:
			t.Fatal("scheduled marker purge did not run")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// Implements DESIGN-008 AccountDeleter marker retention scheduler verification.
func TestRunDeletedCustomFoodCreateKeyPurgerPropagatesErrors(t *testing.T) {
	want := errors.New("purge failed")
	if err := RunDeletedCustomFoodCreateKeyPurger(context.Background(), &markerRepo{err: want}, time.Hour); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
}

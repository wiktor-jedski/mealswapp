package maintenance

import (
	"context"
	"errors"
	"testing"
	"time"
)

type markerRepo struct {
	calls  int
	err    error
	called chan struct{}
}

func (r *markerRepo) PurgeExpiredDeletedCustomFoodCreateKeys(context.Context) error {
	r.calls++
	if r.called != nil {
		select {
		case r.called <- struct{}{}:
		default:
		}
	}
	return r.err
}

// Implements DESIGN-008 AccountDeleter marker retention scheduler verification.
func TestRunDeletedCustomFoodCreateKeyPurgerRunsImmediatelyAndOnTick(t *testing.T) {
	repo := &markerRepo{called: make(chan struct{}, 4)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunDeletedCustomFoodCreateKeyPurger(ctx, repo, time.Millisecond) }()
	select {
	case <-repo.called:
	case <-time.After(time.Second):
		t.Fatal("immediate marker purge did not run")
	}
	select {
	case <-repo.called:
	case <-time.After(time.Second):
		t.Fatal("scheduled marker purge did not run")
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

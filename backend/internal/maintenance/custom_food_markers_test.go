package maintenance

import (
	"context"
	"errors"
	"testing"
	"time"
)

type markerRepo struct {
	calls   int
	err     error
	called  chan struct{}
	results []error
}

func (r *markerRepo) PurgeExpiredDeletedCustomFoodCreateKeys(context.Context) error {
	r.calls++
	if r.called != nil {
		select {
		case r.called <- struct{}{}:
		default:
		}
	}
	if len(r.results) > 0 {
		err := r.results[0]
		r.results = r.results[1:]
		return err
	}
	return r.err
}

// Implements DESIGN-008 AccountDeleter failure-isolated maintenance verification.
func TestRunDeletedCustomFoodCreateKeyPurgerRetriesAfterFailure(t *testing.T) {
	repo := &markerRepo{called: make(chan struct{}, 4), results: []error{errors.New("temporary"), nil}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunDeletedCustomFoodCreateKeyPurger(ctx, repo, time.Millisecond) }()
	for range 2 {
		select {
		case <-repo.called:
		case <-time.After(time.Second):
			t.Fatal("maintenance retry did not run")
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if repo.calls < 2 {
		t.Fatalf("calls=%d", repo.calls)
	}
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
	repo := &markerRepo{err: want, called: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunDeletedCustomFoodCreateKeyPurger(ctx, repo, time.Millisecond) }()
	select {
	case <-repo.called:
	case <-time.After(time.Second):
		t.Fatal("purge did not run")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("scheduler error=%v", err)
	}
}

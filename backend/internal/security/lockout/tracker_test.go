package lockout

import (
	"testing"
	"time"
)

func TestTrackerLocksAccountAfterThreshold(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	tracker := NewTracker(Config{
		AccountFailureThreshold: 2,
		AccountLockoutWindow:    15 * time.Minute,
		IPFailureThreshold:      10,
		IPWindow:                10 * time.Minute,
		Now:                     func() time.Time { return now },
	})

	tracker.RecordFailure("user@example.com", "203.0.113.10")
	state := tracker.RecordFailure("user@example.com", "203.0.113.10")

	if state.LockedUntil == nil || state.RetryAfter != 15*time.Minute {
		t.Fatalf("expected account lockout, got %#v", state)
	}
}

func TestTrackerLockoutExpires(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	tracker := NewTracker(Config{
		AccountFailureThreshold: 1,
		AccountLockoutWindow:    15 * time.Minute,
		Now:                     func() time.Time { return now },
	})

	tracker.RecordFailure("user@example.com", "203.0.113.10")
	now = now.Add(15*time.Minute + time.Second)

	_, locked := tracker.IsLocked("user@example.com", "203.0.113.10")
	if locked {
		t.Fatal("expected lockout to expire")
	}
}

func TestTrackerResetsOnSuccess(t *testing.T) {
	tracker := NewTracker(Config{AccountFailureThreshold: 2})

	tracker.RecordFailure("user@example.com", "203.0.113.10")
	tracker.RecordSuccess("user@example.com", "203.0.113.10")
	state := tracker.State("user@example.com", "203.0.113.10")

	if state.AccountFailures != 0 || state.IPFailures != 0 || state.LockedUntil != nil {
		t.Fatalf("expected reset state, got %#v", state)
	}
}

func TestResetAccountClearsAccountLockoutOnly(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	tracker := NewTracker(Config{AccountFailureThreshold: 2, IPFailureThreshold: 2, Now: func() time.Time { return now }})

	tracker.RecordFailure("user@example.com", "203.0.113.10")
	tracker.RecordFailure("user@example.com", "203.0.113.10")
	tracker.ResetAccount("user@example.com")
	state := tracker.State("user@example.com", "203.0.113.10")

	if state.AccountFailures != 0 || state.IPFailures != 2 {
		t.Fatalf("expected account reset with IP state preserved, got %#v", state)
	}
}

func TestTrackerLocksIPWindow(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	tracker := NewTracker(Config{
		AccountFailureThreshold: 100,
		IPFailureThreshold:      2,
		IPWindow:                10 * time.Minute,
		Now:                     func() time.Time { return now },
	})

	tracker.RecordFailure("first@example.com", "203.0.113.10")
	state := tracker.RecordFailure("second@example.com", "203.0.113.10")

	if state.IPFailures != 2 || state.RetryAfter != 10*time.Minute {
		t.Fatalf("expected IP lockout state, got %#v", state)
	}
}

func TestTrackerIPWindowExpires(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	tracker := NewTracker(Config{
		AccountFailureThreshold: 100,
		IPFailureThreshold:      2,
		IPWindow:                10 * time.Minute,
		Now:                     func() time.Time { return now },
	})

	tracker.RecordFailure("first@example.com", "203.0.113.10")
	tracker.RecordFailure("second@example.com", "203.0.113.10")
	now = now.Add(10*time.Minute + time.Second)
	state := tracker.State("third@example.com", "203.0.113.10")

	if state.IPFailures != 0 || state.RetryAfter != 0 {
		t.Fatalf("expected IP window to expire, got %#v", state)
	}
}

func TestPublicFailureDoesNotRevealAccountState(t *testing.T) {
	normal := PublicFailureFor(State{})
	locked := PublicFailureFor(State{RetryAfter: 15 * time.Minute})

	if normal.Message != locked.Message || normal.Message != "Invalid credentials" {
		t.Fatalf("expected generic auth message, got normal=%#v locked=%#v", normal, locked)
	}
	if locked.RetryAfterSeconds != 900 {
		t.Fatalf("expected retry metadata without account-state wording, got %#v", locked)
	}
}

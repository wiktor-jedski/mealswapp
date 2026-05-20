package lockout

import (
	"time"
)

type State struct {
	AccountFailures int
	IPFailures      int
	LockedUntil     *time.Time
	RetryAfter      time.Duration
}

type PublicFailure struct {
	Message           string
	RetryAfterSeconds int
}

type Config struct {
	AccountFailureThreshold int
	AccountLockoutWindow    time.Duration
	IPFailureThreshold      int
	IPWindow                time.Duration
	Now                     func() time.Time
}

type Tracker struct {
	config   Config
	accounts map[string]accountState
	ips      map[string]ipState
}

type accountState struct {
	failures    int
	lockedUntil *time.Time
}

type ipState struct {
	failures int
	resetAt  time.Time
}

func NewTracker(config Config) *Tracker {
	if config.AccountFailureThreshold == 0 {
		config.AccountFailureThreshold = 5
	}
	if config.AccountLockoutWindow == 0 {
		config.AccountLockoutWindow = 15 * time.Minute
	}
	if config.IPFailureThreshold == 0 {
		config.IPFailureThreshold = 10
	}
	if config.IPWindow == 0 {
		config.IPWindow = 10 * time.Minute
	}
	if config.Now == nil {
		config.Now = time.Now
	}

	return &Tracker{
		config:   config,
		accounts: make(map[string]accountState),
		ips:      make(map[string]ipState),
	}
}

func (tracker *Tracker) State(accountKey string, ip string) State {
	now := tracker.config.Now()
	account := tracker.accounts[accountKey]
	ipStatus := tracker.ipStatus(ip, now)

	state := State{
		AccountFailures: account.failures,
		IPFailures:      ipStatus.failures,
	}

	if account.lockedUntil != nil && now.Before(*account.lockedUntil) {
		lockedUntil := *account.lockedUntil
		state.LockedUntil = &lockedUntil
		state.RetryAfter = lockedUntil.Sub(now)
	}
	if ipStatus.failures >= tracker.config.IPFailureThreshold && now.Before(ipStatus.resetAt) {
		retryAfter := ipStatus.resetAt.Sub(now)
		if state.RetryAfter == 0 || retryAfter > state.RetryAfter {
			state.RetryAfter = retryAfter
		}
		if state.LockedUntil == nil {
			lockedUntil := ipStatus.resetAt
			state.LockedUntil = &lockedUntil
		}
	}

	return state
}

func (tracker *Tracker) IsLocked(accountKey string, ip string) (State, bool) {
	state := tracker.State(accountKey, ip)
	return state, state.RetryAfter > 0
}

func (tracker *Tracker) RecordFailure(accountKey string, ip string) State {
	now := tracker.config.Now()

	account := tracker.accounts[accountKey]
	if account.lockedUntil == nil || !now.Before(*account.lockedUntil) {
		account.failures++
		if account.failures >= tracker.config.AccountFailureThreshold {
			lockedUntil := now.Add(tracker.config.AccountLockoutWindow)
			account.lockedUntil = &lockedUntil
		}
	}
	tracker.accounts[accountKey] = account

	ipStatus := tracker.ipStatus(ip, now)
	ipStatus.failures++
	tracker.ips[ip] = ipStatus

	return tracker.State(accountKey, ip)
}

func (tracker *Tracker) RecordSuccess(accountKey string, ip string) {
	delete(tracker.accounts, accountKey)
	delete(tracker.ips, ip)
}

func (tracker *Tracker) ResetAccount(accountKey string) {
	delete(tracker.accounts, accountKey)
}

func (tracker *Tracker) ipStatus(ip string, now time.Time) ipState {
	status := tracker.ips[ip]
	if status.resetAt.IsZero() || !now.Before(status.resetAt) {
		return ipState{resetAt: now.Add(tracker.config.IPWindow)}
	}
	return status
}

func PublicFailureFor(state State) PublicFailure {
	retryAfterSeconds := 0
	if state.RetryAfter > 0 {
		retryAfterSeconds = int(state.RetryAfter.Seconds())
		if retryAfterSeconds < 1 {
			retryAfterSeconds = 1
		}
	}

	return PublicFailure{
		Message:           "Invalid credentials",
		RetryAfterSeconds: retryAfterSeconds,
	}
}

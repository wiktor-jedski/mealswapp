package retention

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultPolicyCoversRequiredDataClasses(t *testing.T) {
	policy := DefaultPolicy()

	if err := policy.Validate(); err != nil {
		t.Fatalf("expected default policy to validate: %v", err)
	}
	if policy.BackupRetentionDays != 30 {
		t.Fatalf("expected 30-day backup retention, got %d", policy.BackupRetentionDays)
	}
	assertRule(t, policy, DataClassSessions, 30*time.Minute)
	assertRule(t, policy, DataClassExports, 7*24*time.Hour)
	assertRule(t, policy, DataClassOptimizationJobs, time.Hour)
	assertRule(t, policy, DataClassAuditLogs, 7*365*24*time.Hour)
}

func TestServiceEnforcesDryRunCutoffs(t *testing.T) {
	store := &fakeRetentionStore{counts: map[DataClass]int{
		DataClassSessions:         2,
		DataClassSearchHistory:    3,
		DataClassExports:          4,
		DataClassDeletedAccounts:  1,
		DataClassAuditLogs:        5,
		DataClassImportRecords:    6,
		DataClassOptimizationJobs: 7,
	}}
	service := NewService(store)
	service.now = func() time.Time {
		return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	}

	results, err := service.Enforce(context.Background(), DefaultPolicy(), true)
	if err != nil {
		t.Fatalf("unexpected enforce error: %v", err)
	}

	if len(results) != len(DefaultPolicy().Rules) {
		t.Fatalf("expected one result per rule, got %#v", results)
	}
	if !results[0].DryRun || results[0].DataClass != DataClassSessions || results[0].Deleted != 2 {
		t.Fatalf("unexpected first result: %#v", results[0])
	}
	if !store.calls[DataClassSessions].Equal(time.Date(2026, 5, 20, 11, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected session cutoff: %s", store.calls[DataClassSessions])
	}
	if !store.calls[DataClassDeletedAccounts].Equal(time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("deleted-account cleanup should be immediate, got %s", store.calls[DataClassDeletedAccounts])
	}
}

func TestPolicyValidationRejectsMissingRulesAndBadBackupRetention(t *testing.T) {
	policy := DefaultPolicy()
	policy.BackupRetentionDays = 31
	if !errors.Is(policy.Validate(), ErrInvalidBackupRetention) {
		t.Fatalf("expected invalid backup retention error")
	}

	policy = DefaultPolicy()
	policy.Rules = policy.Rules[:len(policy.Rules)-1]
	if !errors.Is(policy.Validate(), ErrMissingRule) {
		t.Fatalf("expected missing rule error")
	}
}

func assertRule(t *testing.T, policy Policy, dataClass DataClass, retainFor time.Duration) {
	t.Helper()
	for _, rule := range policy.Rules {
		if rule.DataClass == dataClass {
			if rule.RetainFor != retainFor {
				t.Fatalf("expected %s retention %s, got %s", dataClass, retainFor, rule.RetainFor)
			}
			return
		}
	}
	t.Fatalf("missing rule for %s", dataClass)
}

type fakeRetentionStore struct {
	counts map[DataClass]int
	calls  map[DataClass]time.Time
}

func (store *fakeRetentionStore) DeleteBefore(ctx context.Context, dataClass DataClass, cutoff time.Time, dryRun bool) (int, error) {
	if !dryRun {
		return 0, errors.New("expected dry run")
	}
	if store.calls == nil {
		store.calls = make(map[DataClass]time.Time)
	}
	store.calls[dataClass] = cutoff
	return store.counts[dataClass], nil
}

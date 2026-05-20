package retention

import (
	"context"
	"time"
)

type DataClass string

const (
	DataClassSessions         DataClass = "sessions"
	DataClassSearchHistory    DataClass = "search_history"
	DataClassExports          DataClass = "exports"
	DataClassDeletedAccounts  DataClass = "deleted_accounts"
	DataClassAuditLogs        DataClass = "audit_logs"
	DataClassImportRecords    DataClass = "import_records"
	DataClassOptimizationJobs DataClass = "optimization_jobs"
)

type Rule struct {
	DataClass DataClass
	RetainFor time.Duration
}

type Policy struct {
	Rules               []Rule
	BackupRetentionDays int
}

type CleanupResult struct {
	DataClass DataClass
	Cutoff    time.Time
	Deleted   int
	DryRun    bool
}

type Store interface {
	DeleteBefore(ctx context.Context, dataClass DataClass, cutoff time.Time, dryRun bool) (int, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) Service {
	return Service{store: store, now: time.Now}
}

func DefaultPolicy() Policy {
	return Policy{
		BackupRetentionDays: 30,
		Rules: []Rule{
			{DataClass: DataClassSessions, RetainFor: 30 * time.Minute},
			{DataClass: DataClassSearchHistory, RetainFor: 365 * 24 * time.Hour},
			{DataClass: DataClassExports, RetainFor: 7 * 24 * time.Hour},
			{DataClass: DataClassDeletedAccounts, RetainFor: 0},
			{DataClass: DataClassAuditLogs, RetainFor: 7 * 365 * 24 * time.Hour},
			{DataClass: DataClassImportRecords, RetainFor: 365 * 24 * time.Hour},
			{DataClass: DataClassOptimizationJobs, RetainFor: time.Hour},
		},
	}
}

func (policy Policy) Validate() error {
	if policy.BackupRetentionDays != 30 {
		return ErrInvalidBackupRetention
	}
	seen := map[DataClass]bool{}
	for _, rule := range policy.Rules {
		if rule.DataClass == "" || rule.RetainFor < 0 {
			return ErrInvalidRule
		}
		seen[rule.DataClass] = true
	}
	for _, required := range requiredDataClasses() {
		if !seen[required] {
			return ErrMissingRule
		}
	}
	return nil
}

func (service Service) Enforce(ctx context.Context, policy Policy, dryRun bool) ([]CleanupResult, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	now := service.now().UTC()
	results := make([]CleanupResult, 0, len(policy.Rules))
	for _, rule := range policy.Rules {
		cutoff := now.Add(-rule.RetainFor)
		deleted, err := service.store.DeleteBefore(ctx, rule.DataClass, cutoff, dryRun)
		if err != nil {
			return results, err
		}
		results = append(results, CleanupResult{DataClass: rule.DataClass, Cutoff: cutoff, Deleted: deleted, DryRun: dryRun})
	}
	return results, nil
}

func requiredDataClasses() []DataClass {
	return []DataClass{
		DataClassSessions,
		DataClassSearchHistory,
		DataClassExports,
		DataClassDeletedAccounts,
		DataClassAuditLogs,
		DataClassImportRecords,
		DataClassOptimizationJobs,
	}
}

package repository

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Implements DESIGN-007 EntitlementManager append query.
//
//go:embed sql/entitlement_append.sql
var entitlementAppendSQL string

// Implements DESIGN-007 EntitlementManager latest-state query.
//
//go:embed sql/entitlement_get_latest.sql
var entitlementGetLatestSQL string

// Implements DESIGN-007 UsageLimiter record query.
//
//go:embed sql/usage_window_record.sql
var usageWindowRecordSQL string

// Implements DESIGN-007 UsageLimiter aggregate query.
//
//go:embed sql/usage_window_get_since.sql
var usageWindowGetSinceSQL string

// Implements DESIGN-007 TrialTracker expired-trials query.
//
//go:embed sql/entitlement_list_expired_trials.sql
var entitlementListExpiredTrialsSQL string

// Implements DESIGN-007 StripeWebhookHandler idempotency query.
//
//go:embed sql/processed_stripe_event_insert.sql
var processedStripeEventInsertSQL string

// PostgresEntitlementRepository persists subscription and usage state in PostgreSQL.
// Implements DESIGN-007 EntitlementManager.
type PostgresEntitlementRepository struct {
	db sqlExecutor
}

var _ EntitlementRepository = (*PostgresEntitlementRepository)(nil)
var _ StripeEventRepository = (*PostgresEntitlementRepository)(nil)
var _ TrialRepository = (*PostgresEntitlementRepository)(nil)
var _ UsageRepository = (*PostgresEntitlementRepository)(nil)

// NewPostgresEntitlementRepository creates a PostgreSQL-backed entitlement repository.
// Implements DESIGN-007 EntitlementManager.
func NewPostgresEntitlementRepository(db sqlExecutor) *PostgresEntitlementRepository {
	return &PostgresEntitlementRepository{db: db}
}

// AppendEntitlement appends entitlement state without deleting previous history.
// Implements DESIGN-007 EntitlementManager.
func (r *PostgresEntitlementRepository) AppendEntitlement(ctx context.Context, entitlement Entitlement) error {
	if err := validateEntitlement(entitlement); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, entitlementAppendSQL, entitlement.UserID, entitlement.Tier, entitlement.Status, entitlement.SearchLimitPer24h, entitlement.AllowedModes, entitlement.ExpiresAt, entitlement.StripeCustomerID, entitlement.StripeSubscriptionID)
	if err != nil {
		return mapPostgresError(err, "append entitlement")
	}
	return nil
}

// GetLatest returns the most recent entitlement state for one user.
// Implements DESIGN-007 EntitlementManager.
func (r *PostgresEntitlementRepository) GetLatest(ctx context.Context, userID uuid.UUID) (Entitlement, error) {
	if userID == uuid.Nil {
		return Entitlement{}, validationError("user id is required")
	}
	row := r.db.QueryRow(ctx, entitlementGetLatestSQL, userID)
	return scanEntitlement(row)
}

// RecordUsage records usage at one occurrence time.
// Implements DESIGN-007 UsageLimiter.
func (r *PostgresEntitlementRepository) RecordUsage(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (UsageWindow, error) {
	if userID == uuid.Nil {
		return UsageWindow{}, validationError("user id is required")
	}
	if strings.TrimSpace(feature) == "" {
		return UsageWindow{}, validationError("feature is required")
	}
	if occurredAt.IsZero() {
		return UsageWindow{}, validationError("occurred at is required")
	}
	row := r.db.QueryRow(ctx, usageWindowRecordSQL, userID, feature, occurredAt)
	return scanUsageWindow(row)
}

// GetUsageSince returns usage accumulated since a caller-supplied cutoff.
// Implements DESIGN-007 UsageLimiter.
func (r *PostgresEntitlementRepository) GetUsageSince(ctx context.Context, userID uuid.UUID, feature string, since time.Time) (UsageWindow, error) {
	if userID == uuid.Nil {
		return UsageWindow{}, validationError("user id is required")
	}
	if strings.TrimSpace(feature) == "" {
		return UsageWindow{}, validationError("feature is required")
	}
	if since.IsZero() {
		return UsageWindow{}, validationError("since is required")
	}
	var window UsageWindow
	err := r.db.QueryRow(ctx, usageWindowGetSinceSQL, userID, feature, since).Scan(&window.UserID, &window.Feature, &window.StartedAt, &window.SearchCount, &window.CreatedAt, &window.UpdatedAt)
	if err != nil {
		return UsageWindow{}, mapPostgresError(err, "get usage since")
	}
	return window, nil
}

// ListExpiredTrials returns active trial entitlements expired by now.
// Implements DESIGN-007 TrialTracker.
func (r *PostgresEntitlementRepository) ListExpiredTrials(ctx context.Context, now time.Time) ([]Entitlement, error) {
	if now.IsZero() {
		return nil, validationError("now is required")
	}
	rows, err := r.db.Query(ctx, entitlementListExpiredTrialsSQL, now)
	if err != nil {
		return nil, mapPostgresError(err, "list expired trials")
	}
	defer rows.Close()

	entitlements := []Entitlement{}
	for rows.Next() {
		entitlement, err := scanEntitlement(rows)
		if err != nil {
			return nil, err
		}
		entitlements = append(entitlements, entitlement)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate expired trials")
	}
	return entitlements, nil
}

// InsertProcessedStripeEvent stores webhook idempotency metadata and reports duplicates.
// Implements DESIGN-007 StripeWebhookHandler.
func (r *PostgresEntitlementRepository) InsertProcessedStripeEvent(ctx context.Context, event ProcessedStripeEvent) (bool, error) {
	if strings.TrimSpace(event.EventID) == "" {
		return false, validationError("event id is required")
	}
	if strings.TrimSpace(event.EventType) == "" {
		return false, validationError("event type is required")
	}
	if event.Outcome != "success" && event.Outcome != "duplicate" && event.Outcome != "failed" {
		return false, validationError("event outcome is invalid")
	}
	payload := event.Payload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	if !json.Valid(payload) {
		return false, validationError("event payload must be valid json")
	}
	_, err := r.db.Exec(ctx, processedStripeEventInsertSQL, event.EventID, event.EventType, event.Outcome, payload)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil
		}
		return false, mapPostgresError(err, "insert processed stripe event")
	}
	return true, nil
}

// scanEntitlement reads an entitlement from a PostgreSQL row.
// Implements DESIGN-007 EntitlementManager.
func scanEntitlement(row pgx.Row) (Entitlement, error) {
	var entitlement Entitlement
	if err := row.Scan(
		&entitlement.UserID,
		&entitlement.Tier,
		&entitlement.Status,
		&entitlement.SearchLimitPer24h,
		&entitlement.AllowedModes,
		&entitlement.ExpiresAt,
		&entitlement.StripeCustomerID,
		&entitlement.StripeSubscriptionID,
		&entitlement.CreatedAt,
		&entitlement.UpdatedAt,
	); err != nil {
		return Entitlement{}, mapPostgresError(err, "scan entitlement")
	}
	return entitlement, nil
}

// scanUsageWindow reads a usage-window aggregate from a PostgreSQL row.
// Implements DESIGN-007 UsageLimiter.
func scanUsageWindow(row pgx.Row) (UsageWindow, error) {
	var window UsageWindow
	if err := row.Scan(&window.UserID, &window.Feature, &window.StartedAt, &window.SearchCount, &window.CreatedAt, &window.UpdatedAt); err != nil {
		return UsageWindow{}, mapPostgresError(err, "scan usage window")
	}
	return window, nil
}

// validateEntitlement checks entitlement state and tier invariants.
// Implements DESIGN-007 EntitlementManager.
func validateEntitlement(entitlement Entitlement) error {
	if entitlement.UserID == uuid.Nil {
		return validationError("user id is required")
	}
	if entitlement.Tier != "free" && entitlement.Tier != "trial" && entitlement.Tier != "paid" {
		return validationError("entitlement tier is invalid")
	}
	if entitlement.Status != "active" && entitlement.Status != "expired" && entitlement.Status != "past_due" && entitlement.Status != "cancelled" {
		return validationError("entitlement status is invalid")
	}
	if entitlement.SearchLimitPer24h < 0 {
		return validationError("search limit is invalid")
	}
	if len(entitlement.AllowedModes) == 0 {
		return validationError("allowed modes are required")
	}
	seenModes := map[string]struct{}{}
	for _, mode := range entitlement.AllowedModes {
		if mode != "catalog" && mode != "substitution" && mode != "daily_diet_alternative" {
			return validationError("allowed mode is invalid")
		}
		if _, exists := seenModes[mode]; exists {
			return validationError("allowed modes must be unique")
		}
		seenModes[mode] = struct{}{}
	}
	if entitlement.Tier == "trial" && entitlement.ExpiresAt == nil {
		return validationError("trial expiry is required")
	}
	return nil
}

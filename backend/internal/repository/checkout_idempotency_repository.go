package repository

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// Implements DESIGN-007 SubscriptionController checkout idempotency query.
//
//go:embed sql/checkout_idempotency_get.sql
var checkoutIdempotencyGetSQL string

// Implements DESIGN-007 SubscriptionController checkout idempotency query.
//
//go:embed sql/checkout_idempotency_store.sql
var checkoutIdempotencyStoreSQL string

// Implements DESIGN-004 JobStatusTracker publication acknowledgement update.
//
//go:embed sql/checkout_idempotency_update_response.sql
var checkoutIdempotencyUpdateResponseSQL string

// PostgresCheckoutIdempotencyRepository persists checkout idempotency responses.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
type PostgresCheckoutIdempotencyRepository struct {
	db sqlExecutor
}

// Implements DESIGN-007 SubscriptionController compile-time repository contract.
var _ CheckoutIdempotencyRepository = (*PostgresCheckoutIdempotencyRepository)(nil)

// NewPostgresCheckoutIdempotencyRepository creates a PostgreSQL checkout idempotency store.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func NewPostgresCheckoutIdempotencyRepository(db sqlExecutor) *PostgresCheckoutIdempotencyRepository {
	return &PostgresCheckoutIdempotencyRepository{db: db}
}

// GetCheckoutIdempotency loads a stored checkout response for a scoped idempotency key.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func (r *PostgresCheckoutIdempotencyRepository) GetCheckoutIdempotency(ctx context.Context, userID uuid.UUID, method string, route string, key string) (CheckoutIdempotencyRecord, error) {
	if err := validateCheckoutIdempotencyScope(userID, method, route, key); err != nil {
		return CheckoutIdempotencyRecord{}, err
	}
	var record CheckoutIdempotencyRecord
	err := r.db.QueryRow(ctx, checkoutIdempotencyGetSQL, userID, method, route, key).Scan(
		&record.UserID,
		&record.Method,
		&record.Route,
		&record.Key,
		&record.BodyHash,
		&record.StatusCode,
		&record.ResponseBody,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return CheckoutIdempotencyRecord{}, mapPostgresError(err, "get checkout idempotency")
	}
	return record, nil
}

// StoreCheckoutIdempotency stores the first completed checkout response for exact retries.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func (r *PostgresCheckoutIdempotencyRepository) StoreCheckoutIdempotency(ctx context.Context, record CheckoutIdempotencyRecord) error {
	if err := validateCheckoutIdempotencyRecord(record); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, checkoutIdempotencyStoreSQL, record.UserID, record.Method, record.Route, record.Key, record.BodyHash, record.StatusCode, record.ResponseBody)
	if err != nil {
		return mapPostgresError(err, "store checkout idempotency")
	}
	return nil
}

// UpdateCheckoutIdempotencyResponse replaces only the response attached to an
// existing, body-matched durable claim.
// Implements DESIGN-004 JobStatusTracker publication acknowledgement persistence.
func (r *PostgresCheckoutIdempotencyRepository) UpdateCheckoutIdempotencyResponse(ctx context.Context, record CheckoutIdempotencyRecord) error {
	if err := validateCheckoutIdempotencyRecord(record); err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, checkoutIdempotencyUpdateResponseSQL, record.UserID, record.Method, record.Route, record.Key, record.BodyHash, record.StatusCode, record.ResponseBody)
	if err != nil {
		return mapPostgresError(err, "update checkout idempotency response")
	}
	if tag.RowsAffected() != 1 {
		return NewError(ErrorKindInternal, "idempotency claim changed before response update", nil)
	}
	return nil
}

// validateCheckoutIdempotencyRecord checks the persisted idempotency response shape.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func validateCheckoutIdempotencyRecord(record CheckoutIdempotencyRecord) error {
	if err := validateCheckoutIdempotencyScope(record.UserID, record.Method, record.Route, record.Key); err != nil {
		return err
	}
	if strings.TrimSpace(record.BodyHash) == "" {
		return validationError("body hash is required")
	}
	if record.StatusCode < 100 || record.StatusCode > 599 {
		return validationError("status code is invalid")
	}
	if len(record.ResponseBody) == 0 || !json.Valid(record.ResponseBody) {
		return validationError("response body must be valid json")
	}
	return nil
}

// validateCheckoutIdempotencyScope checks stable checkout idempotency key scope.
// Implements DESIGN-007 SubscriptionController checkout idempotency.
func validateCheckoutIdempotencyScope(userID uuid.UUID, method string, route string, key string) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if strings.TrimSpace(method) == "" {
		return validationError("method is required")
	}
	if strings.TrimSpace(route) == "" {
		return validationError("route is required")
	}
	if strings.TrimSpace(key) == "" {
		return validationError("idempotency key is required")
	}
	return nil
}

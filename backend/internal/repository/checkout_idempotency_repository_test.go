package repository

// Implements DESIGN-007 SubscriptionController checkout idempotency repository verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type checkoutIdempotencyDB struct {
	execArgs []any
	err      error
}

func (db *checkoutIdempotencyDB) Exec(_ context.Context, _ string, arguments ...any) (pgconn.CommandTag, error) {
	db.execArgs = arguments
	return pgconn.NewCommandTag("INSERT 1"), db.err
}

func (db *checkoutIdempotencyDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unused")
}

func (db *checkoutIdempotencyDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return checkoutIdempotencyRow{}
}

type checkoutIdempotencyRow struct{}

func (checkoutIdempotencyRow) Scan(...any) error {
	return pgx.ErrNoRows
}

func TestPostgresCheckoutIdempotencyRepositoryStoresValidResponse(t *testing.T) {
	db := &checkoutIdempotencyDB{}
	repo := NewPostgresCheckoutIdempotencyRepository(db)
	userID := uuid.New()
	record := CheckoutIdempotencyRecord{
		UserID:       userID,
		Method:       "POST",
		Route:        "/billing/checkout",
		Key:          "checkout-123",
		BodyHash:     "hash",
		StatusCode:   200,
		ResponseBody: []byte(`{"checkoutSessionId":"cs_test_123","checkoutUrl":"https://checkout.stripe.test/session"}`),
	}

	if err := repo.StoreCheckoutIdempotency(context.Background(), record); err != nil {
		t.Fatalf("StoreCheckoutIdempotency() error = %v", err)
	}
	if len(db.execArgs) != 7 || db.execArgs[0] != userID || db.execArgs[6] == nil {
		t.Fatalf("exec args = %#v", db.execArgs)
	}
}

func TestPostgresCheckoutIdempotencyRepositoryRejectsInvalidResponse(t *testing.T) {
	db := &checkoutIdempotencyDB{}
	repo := NewPostgresCheckoutIdempotencyRepository(db)
	err := repo.StoreCheckoutIdempotency(context.Background(), CheckoutIdempotencyRecord{
		UserID:       uuid.New(),
		Method:       "POST",
		Route:        "/billing/checkout",
		Key:          "checkout-123",
		BodyHash:     "hash",
		StatusCode:   200,
		ResponseBody: []byte(`not json`),
	})
	if !IsKind(err, ErrorKindValidation) {
		t.Fatalf("StoreCheckoutIdempotency() error = %v", err)
	}
	if len(db.execArgs) != 0 {
		t.Fatalf("invalid record executed SQL with args = %#v", db.execArgs)
	}
}

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
	tag      string
}

func (db *checkoutIdempotencyDB) Exec(_ context.Context, _ string, arguments ...any) (pgconn.CommandTag, error) {
	db.execArgs = arguments
	if db.tag == "" {
		db.tag = "INSERT 1"
	}
	return pgconn.NewCommandTag(db.tag), db.err
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

// Implements DESIGN-004 JobStatusTracker publication acknowledgement persistence.
func TestPostgresCheckoutIdempotencyRepositoryUpdatesBodyMatchedResponse(t *testing.T) {
	db := &checkoutIdempotencyDB{}
	repo := NewPostgresCheckoutIdempotencyRepository(db)
	record := CheckoutIdempotencyRecord{
		UserID: uuid.New(), Method: "POST", Route: "/optimization/jobs", Key: "optimization-key", BodyHash: "canonical-hash", StatusCode: 202,
		ResponseBody: []byte(`{"jobId":"00000000-0000-0000-0000-000000000001","status":"queued","pollUrl":"/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000001","publicationState":"published"}`),
	}
	if err := repo.UpdateCheckoutIdempotencyResponse(context.Background(), record); err != nil {
		t.Fatalf("UpdateCheckoutIdempotencyResponse() error = %v", err)
	}
	if len(db.execArgs) != 7 || db.execArgs[4] != record.BodyHash || db.execArgs[5] != 202 || string(db.execArgs[6].([]byte)) != string(record.ResponseBody) {
		t.Fatalf("update args = %#v", db.execArgs)
	}
}

// Implements DESIGN-004 JobStatusTracker publication acknowledgement persistence.
func TestPostgresCheckoutIdempotencyRepositoryRejectsMissingResponseUpdateTarget(t *testing.T) {
	db := &checkoutIdempotencyDB{tag: "UPDATE 0"}
	err := NewPostgresCheckoutIdempotencyRepository(db).UpdateCheckoutIdempotencyResponse(context.Background(), CheckoutIdempotencyRecord{
		UserID: uuid.New(), Method: "POST", Route: "/optimization/jobs", Key: "optimization-key", BodyHash: "canonical-hash", StatusCode: 202, ResponseBody: []byte(`{}`),
	})
	if !IsKind(err, ErrorKindInternal) {
		t.Fatalf("UpdateCheckoutIdempotencyResponse() error = %v, want internal consistency failure", err)
	}
}

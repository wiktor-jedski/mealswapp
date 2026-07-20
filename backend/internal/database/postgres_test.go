package database

// Implements DESIGN-005 RepositoryInterfaces PostgreSQL boundary verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakePool struct {
	pingErr error
	closed  bool
}

func (p *fakePool) Ping(context.Context) error {
	return p.pingErr
}

func (p *fakePool) Close() {
	p.closed = true
}

func (p *fakePool) Begin(context.Context) (pgx.Tx, error) {
	return nil, p.pingErr
}

func (p *fakePool) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), p.pingErr
}

func (p *fakePool) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, p.pingErr
}

func (p *fakePool) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

type fakeRow struct{}

func (fakeRow) Scan(...any) error { return nil }

// TestOpenRejectsInvalidURL proves that Open fails with invalid URL
// TestOpenRejectsInvalidURL verifies DESIGN-005 RepositoryInterfaces invalid PostgreSQL URL handling.
func TestOpenRejectsInvalidURL(t *testing.T) {
	if _, err := Open(context.Background(), "not a postgres url"); err == nil {
		t.Fatal("Open() error = nil, want invalid URL error")
	}
}

// TestOpenAcceptsValidURL proves that Open creates a pg pool with valid URL.
// TestOpenAcceptsValidURL verifies DESIGN-005 RepositoryInterfaces valid PostgreSQL URL handling.
func TestOpenAcceptsValidURL(t *testing.T) {
	pool, err := Open(context.Background(), "postgres://mealswapp:mealswapp@localhost:5432/mealswapp_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	pool.Close()
}

// TestPoolPingAndClose verifies that Pool delegates Ping and Close to the underlying pool
// TestPoolPingAndClose verifies DESIGN-005 RepositoryInterfaces PostgreSQL pool wrapper behavior.
func TestPoolPingAndClose(t *testing.T) {
	expected := errors.New("down")
	fake := &fakePool{pingErr: expected}
	pool := &Pool{pool: fake}

	if err := pool.Ping(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("Ping() error = %v, want %v", err, expected)
	}

	pool.Close()
	if !fake.closed {
		t.Fatal("Close() did not close underlying pool")
	}
	if _, err := pool.Exec(context.Background(), "sql"); !errors.Is(err, expected) {
		t.Fatalf("Exec() error = %v", err)
	}
	if _, err := pool.Query(context.Background(), "sql"); !errors.Is(err, expected) {
		t.Fatalf("Query() error = %v", err)
	}
	if err := pool.QueryRow(context.Background(), "sql").Scan(); err != nil {
		t.Fatalf("QueryRow().Scan() error = %v", err)
	}
	if _, err := pool.Begin(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("Begin() error = %v, want %v", err, expected)
	}
}

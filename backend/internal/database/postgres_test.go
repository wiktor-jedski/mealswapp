package database

// Implements DESIGN-005 RepositoryInterfaces PostgreSQL boundary verification.

import (
	"context"
	"errors"
	"testing"
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

// TestOpenRejectsInvalidURL verifies DESIGN-005 RepositoryInterfaces invalid PostgreSQL URL handling.
func TestOpenRejectsInvalidURL(t *testing.T) {
	if _, err := Open(context.Background(), "not a postgres url"); err == nil {
		t.Fatal("Open() error = nil, want invalid URL error")
	}
}

// TestOpenAcceptsValidURL verifies DESIGN-005 RepositoryInterfaces valid PostgreSQL URL handling.
func TestOpenAcceptsValidURL(t *testing.T) {
	pool, err := Open(context.Background(), "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	pool.Close()
}

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
}

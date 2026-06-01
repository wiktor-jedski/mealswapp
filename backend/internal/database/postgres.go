package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pool describes the PostgreSQL pool operations used by the application.
// Implements DESIGN-005 RepositoryInterfaces PostgreSQL connection boundary.
type pool interface {
	Ping(context.Context) error
	Close()
}

// Pool wraps the PostgreSQL connection pool used by repositories and readiness checks.
// Implements DESIGN-005 RepositoryInterfaces PostgreSQL connection boundary.
type Pool struct {
	pool pool
}

// Open creates a PostgreSQL pool from the configured database URL.
// Implements DESIGN-005 RepositoryInterfaces database connection factory.
func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Pool{pool: pool}, nil
}

// Ping verifies that the PostgreSQL pool can reach the database.
// Implements DESIGN-010 RouteHandler readiness dependency check.
func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close releases the PostgreSQL pool resources.
// Implements DESIGN-005 RepositoryInterfaces database resource cleanup.
func (p *Pool) Close() {
	p.pool.Close()
}

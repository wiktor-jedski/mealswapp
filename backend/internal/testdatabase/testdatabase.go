// Package testdatabase provides a fail-closed PostgreSQL boundary for integration tests.
package testdatabase

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
)

// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
const DefaultURL = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp_test?sslmode=disable"

// EnvironmentVariable overrides the dedicated integration-test database without reusing application configuration.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
const EnvironmentVariable = "MEALSWAPP_TEST_DATABASE_URL"

// Open connects to an explicitly test-named database, creating the default local database when absent.
// It fails the test before connecting if configuration could target development or production data.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func Open(t testing.TB) *pgxpool.Pool {
	t.Helper()
	configured := configuredURL()
	databaseURL, databaseName, err := validateURL(configured)
	if err != nil {
		t.Fatalf("refuse unsafe integration test database: %v", err)
	}
	ctx := context.Background()
	if err := ensureDatabase(ctx, databaseURL, databaseName); err != nil {
		t.Skipf("test postgres unavailable: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("test postgres unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("test postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func configuredURL() string {
	if configured := os.Getenv(EnvironmentVariable); configured != "" {
		return configured
	}
	return DefaultURL
}

// Reset serializes destructive integration tests and returns a freshly migrated test schema.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func Reset(t testing.TB, migrationDirectory string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool := Open(t)
	if _, err := pool.Exec(ctx, "SELECT pg_advisory_lock(9010101)"); err != nil {
		t.Fatalf("acquire test database lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "SELECT pg_advisory_unlock(9010101)")
	})
	// Bootstrap a newly created blank database before exercising the rollback path.
	if err := migrations.Run(ctx, pool, "up", migrationDirectory); err != nil {
		t.Fatalf("bootstrap test migrations up: %v", err)
	}
	if err := migrations.Run(ctx, pool, "down", migrationDirectory); err != nil {
		t.Fatalf("reset test migrations down: %v", err)
	}
	if err := migrations.Run(ctx, pool, "up", migrationDirectory); err != nil {
		t.Fatalf("apply test migrations up: %v", err)
	}
	return pool
}

// validateURL accepts only PostgreSQL database names ending in _test.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func validateURL(raw string) (string, string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", "", errors.New("URL scheme must be postgres or postgresql")
	}
	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if databaseName == "" || strings.Contains(databaseName, "/") || !strings.HasSuffix(databaseName, "_test") {
		return "", "", errors.New("database name must end in _test")
	}
	return raw, databaseName, nil
}

// ensureDatabase creates a missing local test database through PostgreSQL's maintenance database.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func ensureDatabase(ctx context.Context, databaseURL, databaseName string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err == nil {
		err = pool.Ping(ctx)
		pool.Close()
	}
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "3D000" {
		return err
	}
	parsed, parseErr := url.Parse(databaseURL)
	if parseErr != nil {
		return parseErr
	}
	parsed.Path = "/postgres"
	parsed.RawPath = ""
	connection, connectErr := pgx.Connect(ctx, parsed.String())
	if connectErr != nil {
		return connectErr
	}
	defer connection.Close(ctx)
	_, createErr := connection.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize())
	if errors.As(createErr, &postgresError) && postgresError.Code == "42P04" {
		return nil
	}
	return createErr
}

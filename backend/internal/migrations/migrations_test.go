package migrations

// Implements DESIGN-005 RepositoryInterfaces schema migration verification.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeExecutor struct {
	sqls []string
	err  error
}

func (e *fakeExecutor) Exec(_ context.Context, stringSQL string, _ ...any) (pgconn.CommandTag, error) {
	e.sqls = append(e.sqls, stringSQL)
	return pgconn.CommandTag{}, e.err
}

// TestRunExecutesUpMigrationsInAscendingOrder verifies DESIGN-005 RepositoryInterfaces up migration ordering.
func TestRunExecutesUpMigrationsInAscendingOrder(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "002_second.up.sql", "select 2;")
	writeMigration(t, dir, "001_first.up.sql", "select 1;")
	writeMigration(t, dir, "003_empty.up.sql", "   \n\t")

	exec := &fakeExecutor{}
	if err := Run(context.Background(), exec, "up", dir); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := []string{"select 1;", "select 2;"}
	if len(exec.sqls) != len(want) {
		t.Fatalf("executed SQL count = %d, want %d: %#v", len(exec.sqls), len(want), exec.sqls)
	}
	for i, sql := range want {
		if exec.sqls[i] != sql {
			t.Fatalf("executed SQL[%d] = %q, want %q", i, exec.sqls[i], sql)
		}
	}
}

// TestRunExecutesDownMigrationsInDescendingOrder verifies DESIGN-005 RepositoryInterfaces down migration ordering.
func TestRunExecutesDownMigrationsInDescendingOrder(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_first.down.sql", "drop 1;")
	writeMigration(t, dir, "002_second.down.sql", "drop 2;")

	exec := &fakeExecutor{}
	if err := Run(context.Background(), exec, "down", dir); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := []string{"drop 2;", "drop 1;"}
	for i, sql := range want {
		if exec.sqls[i] != sql {
			t.Fatalf("executed SQL[%d] = %q, want %q", i, exec.sqls[i], sql)
		}
	}
}

// TestRunRejectsMissingMigrations verifies DESIGN-005 RepositoryInterfaces missing migration handling.
func TestRunRejectsMissingMigrations(t *testing.T) {
	if err := Run(context.Background(), &fakeExecutor{}, "up", t.TempDir()); err == nil {
		t.Fatal("Run() error = nil, want missing migrations error")
	}
}

// TestRunReturnsGlobError verifies DESIGN-005 RepositoryInterfaces invalid migration pattern handling.
func TestRunReturnsGlobError(t *testing.T) {
	if err := Run(context.Background(), &fakeExecutor{}, "up", "["); err == nil {
		t.Fatal("Run() error = nil, want glob error")
	}
}

// TestRunReturnsReadFileError verifies DESIGN-005 RepositoryInterfaces migration file read error handling.
func TestRunReturnsReadFileError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "missing.sql"), filepath.Join(dir, "001_missing.up.sql")); err != nil {
		t.Fatalf("create broken migration symlink: %v", err)
	}

	if err := Run(context.Background(), &fakeExecutor{}, "up", dir); err == nil {
		t.Fatal("Run() error = nil, want file read error")
	}
}

// TestRunWrapsExecError verifies DESIGN-005 RepositoryInterfaces migration execution error handling.
func TestRunWrapsExecError(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_fail.up.sql", "select fail;")

	expected := errors.New("database down")
	err := Run(context.Background(), &fakeExecutor{err: expected}, "up", dir)
	if !errors.Is(err, expected) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, expected)
	}
}

func writeMigration(t *testing.T, dir string, name string, sql string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(sql), 0o600); err != nil {
		t.Fatalf("write migration %s: %v", name, err)
	}
}

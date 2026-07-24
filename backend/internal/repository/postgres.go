package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// sqlExecutor describes the PostgreSQL operations used by repositories and transactions.
// Implements DESIGN-005 RepositoryInterfaces.
type sqlExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// txStarter describes an executor that can start a PostgreSQL transaction.
// Implements DESIGN-005 RepositoryInterfaces.
type txStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// transactionalExecutor describes repository operations with required transaction support.
// Implements DESIGN-005 RepositoryInterfaces.
type transactionalExecutor interface {
	sqlExecutor
	txStarter
}

// withTransaction runs fn in a required database transaction.
// Implements DESIGN-005 RepositoryInterfaces.
func withTransaction(ctx context.Context, db transactionalExecutor, fn func(transactionalExecutor) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err, "begin transaction")
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return mapPostgresError(err, "commit transaction")
	}
	return nil
}

// mapPostgresError maps PostgreSQL failures to stable repository error kinds.
// Implements DESIGN-005 RepositoryInterfaces.
func mapPostgresError(err error, fallback string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return NewError(ErrorKindNotFound, fallback, err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return NewError(ErrorKindConflict, fallback, err)
		case "23502", "23503", "23514", "22001", "22003", "22021", "22P02":
			return NewError(ErrorKindValidation, fallback, err)
		case "40001", "40P01":
			return NewError(ErrorKindRetryable, fallback, err)
		case "57014":
			return NewError(ErrorKindCanceled, fallback, err)
		}
		if len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
			return NewError(ErrorKindConnection, fallback, err)
		}
		return NewError(ErrorKindInternal, fallback, err)
	}

	return NewError(ErrorKindConnection, fallback, err)
}

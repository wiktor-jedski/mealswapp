package seed

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Implements DESIGN-005 MicronutrientVocabulary deterministic development fixtures.
//
//go:embed development.sql
var developmentSQL string

// beginner describes a database connection that can start the seed transaction.
// Implements DESIGN-005 MicronutrientVocabulary seed transaction boundary.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Run inserts deterministic development and test fixtures.
// Implements DESIGN-005 MicronutrientVocabulary.
func Run(ctx context.Context, db beginner) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, developmentSQL); err != nil {
		return fmt.Errorf("execute development seed: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}

package migrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// executor describes the SQL operation required to run migrations.
// Implements DESIGN-005 RepositoryInterfaces schema migration execution.
type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Run finds applicable migration files and executes them.
// Implements DESIGN-005 RepositoryInterfaces schema migration execution.
func Run(ctx context.Context, conn executor, direction string, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*."+direction+".sql"))
	if err != nil {
		return err
	}
	slices.Sort(files)
	if direction == "down" {
		slices.Reverse(files)
	}

	if len(files) == 0 {
		return fmt.Errorf("no %s migrations found in %s", direction, dir)
	}

	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(sql)) == "" {
			continue
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
	}
	return nil
}

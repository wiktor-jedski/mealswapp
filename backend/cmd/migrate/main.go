package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/mealswapp/mealswapp/backend/internal/config"
)

func main() {
	// Implements DESIGN-005 RepositoryInterfaces migration command bootstrap.
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if direction != "up" && direction != "down" {
		log.Fatalf("usage: go run ./cmd/migrate [up|down]")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer conn.Close(ctx)

	if err := runMigrations(ctx, conn, direction, "../database/migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
}

func runMigrations(ctx context.Context, conn *pgx.Conn, direction string, dir string) error {
	// Implements DESIGN-005 RepositoryInterfaces schema migration execution.
	files, err := filepath.Glob(filepath.Join(dir, "*."+direction+".sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	if direction == "down" {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
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

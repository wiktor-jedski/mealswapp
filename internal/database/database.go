// Phase: phase-01 | Task: 6 | Architecture: ARCH-005 | Design: FoodItemEntity
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
)

var (
	Pool   *pgxpool.Pool
	config *pgxpool.Config
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func DefaultConfig() *Config {
	return &Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "mealswapp"),
		Password: getEnv("DB_PASSWORD", "dev"),
		DBName:   getEnv("DB_NAME", "mealswapp"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) ToURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}

func Connect(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	url := cfg.ToURL()

	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	Pool = pool
	config = poolConfig

	return pool, nil
}

func ConnectFromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	return Connect(ctx, DefaultConfig())
}

func GetPool() *pgxpool.Pool {
	return Pool
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

func OpenDB(cfg *Config) (*stdlib.DB, error) {
	connConfig, err := pgxpool.ParseConfig(cfg.ToURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	db := stdlib.OpenDBFromPool(connConfig)
	return db, nil
}

// Phase: phase-01 | Task: 15 | Architecture: ARCH-005 | Design: Database
// RunMigrations executes all SQL migration files in the migrations directory
func RunMigrations(ctx context.Context) error {
	if Pool == nil {
		return fmt.Errorf("database pool not initialized")
	}

	migrationsDir := filepath.Join("internal", "database", "migrations")

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}

	sort.Strings(sqlFiles)

	for _, filename := range sqlFiles {
		filePath := filepath.Join(migrationsDir, filename)
		sql, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		if len(sql) == 0 {
			continue
		}

		_, err = Pool.Exec(ctx, string(sql))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		fmt.Printf("Executed migration: %s\n", filename)
	}

	return nil
}

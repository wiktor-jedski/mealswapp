// Phase: phase-01 | Task: 6 | Architecture: ARCH-005 | Design: FoodItemEntity
package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
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

func RunMigrations(ctx context.Context) error {
	return nil
}

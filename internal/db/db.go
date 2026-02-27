package db

import (
	"context"
	"fmt"

	"github.com/CentraGlobal/backend-payment-go/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a new pgx connection pool from the given DatabaseConfig.
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
	)
	return pgxpool.New(ctx, dsn)
}

// NewARIPool creates a new pgx connection pool from the given ARIDBConfig.
func NewARIPool(ctx context.Context, cfg config.ARIDBConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
	)
	return pgxpool.New(ctx, dsn)
}

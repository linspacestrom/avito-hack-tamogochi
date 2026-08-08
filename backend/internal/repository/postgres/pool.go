package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const disableSSLMode = "disable"

func NewPool(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	port, err := strconv.ParseUint(cfg.Port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	poolConfig.ConnConfig.Host = cfg.Host
	poolConfig.ConnConfig.Port = uint16(port)
	poolConfig.ConnConfig.User = cfg.Username
	poolConfig.ConnConfig.Password = cfg.Password
	poolConfig.ConnConfig.Database = cfg.Database

	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	if cfg.SSLMode == disableSSLMode {
		poolConfig.ConnConfig.TLSConfig = nil
	}

	poolConfig.ConnConfig.ConnectTimeout = cfg.Timeout
	poolConfig.MaxConns = cfg.MaxOpenConns
	poolConfig.MinConns = cfg.MinOpenConns
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

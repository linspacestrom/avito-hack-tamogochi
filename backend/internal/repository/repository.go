package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/config"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
)

const disableSSLMode = "disable"

func New(cfg config.DatabaseConfig) (*postgres.Repository, *manager.Manager, error) {
	pool, err := newPool(cfg)

	if err != nil {
		return nil, nil, fmt.Errorf("create postgres pool: %w", err)
	}

	trManager := manager.Must(trmpgx.NewDefaultFactory(pool))
	repo := postgres.New(pool, trmpgx.DefaultCtxGetter)

	return repo, trManager, nil
}

func newPool(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
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
	poolConfig.MinConns = cfg.MaxIdleConns
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

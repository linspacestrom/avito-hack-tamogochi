package postgres

import (
	"context"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: необходимо поменять на другой builder
//var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type Repository struct {
	db     *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func New(db *pgxpool.Pool, getter *trmpgx.CtxGetter) *Repository {
	return &Repository{db: db, getter: getter}
}

func (r *Repository) Close() {
	r.db.Close()
}

func (r *Repository) GetConn(ctx context.Context) trmpgx.Tr {
	return r.getter.DefaultTrOrDB(ctx, r.db)
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

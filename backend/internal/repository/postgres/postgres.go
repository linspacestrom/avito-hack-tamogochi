package postgres

import (
	"context"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	User    *UserRepository
	Pet     *PetRepository
	Session *SessionRepository
	Event   *EventRepository
}

func New(db *pgxpool.Pool) *Repository {
	r := &Repository{
		db:     db,
		getter: trmpgx.DefaultCtxGetter,
	}
	r.User = NewUserRepository(r)
	r.Pet = NewPetRepository(r)
	r.Session = NewSessionRepository(r)
	r.Event = NewEventRepository(r)
	return r
}

func (r *Repository) Close() {
	r.db.Close()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

func (r *Repository) GetConn(ctx context.Context) trmpgx.Tr {
	return r.getter.DefaultTrOrDB(ctx, r.db)
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type SessionRepository struct {
	repo *Repository
}

var _ service.SessionRepository = (*SessionRepository)(nil)

func NewSessionRepository(repo *Repository) *SessionRepository {
	return &SessionRepository{repo: repo}
}

func (r *SessionRepository) CreateSession(
	ctx context.Context,
	session entity.Session,
) (*entity.Session, error) {
	row, err := db.New(r.repo.GetConn(ctx)).CreateRefreshSession(
		ctx,
		db.CreateRefreshSessionParams{
			UserID:    session.UserID,
			TokenHash: session.TokenHash,
			ExpiresAt: pgtype.Timestamptz{
				Time:  session.ExpiresAt,
				Valid: true,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create refresh session: %w", err)
	}

	result := sessionFromDB(row)
	return &result, nil
}

func (r *SessionRepository) DeleteSession(
	ctx context.Context,
	tokenHash []byte,
) error {
	deleted, err := db.New(r.repo.GetConn(ctx)).DeleteRefreshSession(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("delete refresh session: %w", err)
	}
	if deleted == 0 {
		return apperror.ErrSessionNotFound
	}

	return nil
}

func (r *SessionRepository) GetSession(
	ctx context.Context,
	tokenHash []byte,
) (*entity.Session, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetRefreshSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrSessionNotFound
		}

		return nil, fmt.Errorf("get refresh session: %w", err)
	}

	result := sessionFromDB(row)
	return &result, nil
}

func sessionFromDB(session db.RefreshSession) entity.Session {
	return entity.Session{
		ID:        session.ID,
		UserID:    session.UserID,
		TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt.Time,
		CreatedAt: session.CreatedAt.Time,
	}
}

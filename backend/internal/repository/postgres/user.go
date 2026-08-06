package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository struct {
	repo *Repository
}

var _ service.UserRepository = (*UserRepository)(nil)

func NewUserRepository(repo *Repository) *UserRepository {
	return &UserRepository{
		repo: repo,
	}
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	user entity.User,
) (*entity.User, error) {
	row, err := db.New(r.repo.GetConn(ctx)).CreateUser(ctx, db.CreateUserParams{
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		PasswordHash: user.PasswordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperror.ErrEmailTaken
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	result := userFromDB(row)
	return &result, nil
}

func (r *UserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*entity.User, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	result := userFromDB(row)
	return &result, nil
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*entity.User, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	result := userFromDB(row)
	return &result, nil
}

func userFromDB(user db.User) entity.User {
	return entity.User{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		Status:       user.Status,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}
}

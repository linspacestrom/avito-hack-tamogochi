package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user entity.User) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session entity.Session) (*entity.Session, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
	GetSession(ctx context.Context, tokenHash []byte) (*entity.Session, error)
}

type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type TokenProvider interface {
	Issue(userID uuid.UUID) (entity.TokenPair, error)
	ParseAccess(token string) (uuid.UUID, error)
	HashRefresh(token string) []byte
}

type AuthService struct {
	users      UserRepository
	sessions   SessionRepository
	transactor Transactor
	tokens     TokenProvider
}

func NewAuthService(
	users UserRepository,
	sessions SessionRepository,
	transactor Transactor,
	tokens TokenProvider,
) *AuthService {
	return &AuthService{
		users:      users,
		sessions:   sessions,
		transactor: transactor,
		tokens:     tokens,
	}
}

func (s *AuthService) Register(
	ctx context.Context,
	email string,
	password string,
	displayName string,
) (*entity.AuthResult, error) {
	email, displayName, err := entity.ValidateRegistrationData(
		email,
		password,
		displayName,
	)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var result entity.AuthResult
	err = s.transactor.Do(ctx, func(txCtx context.Context) error {
		user, createErr := s.users.CreateUser(txCtx, entity.User{
			Email:        email,
			PasswordHash: string(passwordHash),
			DisplayName:  displayName,
		})
		if createErr != nil {
			return createErr
		}

		pair, issueErr := s.tokens.Issue(user.ID)
		if issueErr != nil {
			return issueErr
		}

		if _, createErr = s.sessions.CreateSession(txCtx, entity.Session{
			UserID:    user.ID,
			TokenHash: s.tokens.HashRefresh(pair.RefreshToken),
			ExpiresAt: pair.RefreshExpiresAt,
		}); createErr != nil {
			return createErr
		}

		result = entity.AuthResult{User: *user, Tokens: pair}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	return &result, nil
}

func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (*entity.AuthResult, error) {
	email, err := entity.NormalizeEmail(email)
	if err != nil || password == "" {
		return nil, apperror.ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperror.ErrUserNotFound) {
			return nil, apperror.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, apperror.ErrInvalidCredentials
	}
	if user.Status != "active" {
		return nil, apperror.ErrUserBlocked
	}

	pair, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}
	if _, err = s.sessions.CreateSession(ctx, entity.Session{
		UserID:    user.ID,
		TokenHash: s.tokens.HashRefresh(pair.RefreshToken),
		ExpiresAt: pair.RefreshExpiresAt,
	}); err != nil {
		return nil, fmt.Errorf("store refresh session: %w", err)
	}

	return &entity.AuthResult{User: *user, Tokens: pair}, nil
}

func (s *AuthService) Refresh(
	ctx context.Context,
	refreshToken string,
) (*entity.TokenPair, error) {
	if refreshToken == "" {
		return nil, apperror.ErrInvalidToken
	}

	oldHash := s.tokens.HashRefresh(refreshToken)
	var result entity.TokenPair

	err := s.transactor.Do(ctx, func(txCtx context.Context) error {
		session, getErr := s.sessions.GetSession(txCtx, oldHash)
		if getErr != nil {
			if errors.Is(getErr, apperror.ErrSessionNotFound) {
				return apperror.ErrInvalidToken
			}
			return getErr
		}
		if !time.Now().Before(session.ExpiresAt) {
			return apperror.ErrExpiredToken
		}

		user, getErr := s.users.GetByID(txCtx, session.UserID)
		if getErr != nil {
			return getErr
		}
		if user.Status != "active" {
			return apperror.ErrUserBlocked
		}

		if deleteErr := s.sessions.DeleteSession(txCtx, oldHash); deleteErr != nil {
			return deleteErr
		}

		pair, issueErr := s.tokens.Issue(user.ID)
		if issueErr != nil {
			return issueErr
		}
		if _, createErr := s.sessions.CreateSession(txCtx, entity.Session{
			UserID:    user.ID,
			TokenHash: s.tokens.HashRefresh(pair.RefreshToken),
			ExpiresAt: pair.RefreshExpiresAt,
		}); createErr != nil {
			return createErr
		}

		result = pair
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	return &result, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return apperror.ErrInvalidToken
	}

	if err := s.sessions.DeleteSession(ctx, s.tokens.HashRefresh(refreshToken)); err != nil {
		if errors.Is(err, apperror.ErrSessionNotFound) {
			return apperror.ErrInvalidToken
		}
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	if user.Status != "active" {
		return nil, apperror.ErrUserBlocked
	}

	return user, nil
}

func (s *AuthService) Authenticate(
	ctx context.Context,
	accessToken string,
) (uuid.UUID, error) {
	userID, err := s.tokens.ParseAccess(accessToken)
	if err != nil {
		return uuid.Nil, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrUserNotFound) {
			return uuid.Nil, apperror.ErrInvalidToken
		}
		return uuid.Nil, fmt.Errorf("authenticate user: %w", err)
	}
	if user.Status != "active" {
		return uuid.Nil, apperror.ErrUserBlocked
	}

	return userID, nil
}

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
	"github.com/google/uuid"
)

const maxRequestBodySize = 1 << 20

type AuthService interface {
	Register(ctx context.Context, email string, password string, displayName string) (*entity.AuthResult, error)
	Login(ctx context.Context, email string, password string) (*entity.AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*entity.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (*entity.User, error)
}

type Handler struct {
	log        *slog.Logger
	auth       AuthService
	rewards    RewardsService
	dailyCycle DailyCycleService
	tasks      TasksService
}

func New(
	log *slog.Logger,
	auth AuthService,
	rewards RewardsService,
	dailyCycle DailyCycleService,
	tasks TasksService,
) *Handler {
	return &Handler{log: log, auth: auth, rewards: rewards, dailyCycle: dailyCycle, tasks: tasks}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode HTTP response", slog.Any("error", err))
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperror.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid email")
	case errors.Is(err, apperror.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, "password must be 8-72 characters and contain an uppercase letter and a special character")
	case errors.Is(err, apperror.ErrInvalidDisplayName):
		writeError(w, http.StatusBadRequest, "display name must not be empty")
	case errors.Is(err, apperror.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already exists")
	case errors.Is(err, apperror.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, apperror.ErrInvalidToken),
		errors.Is(err, apperror.ErrExpiredToken),
		errors.Is(err, apperror.ErrSessionNotFound):
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
	case errors.Is(err, apperror.ErrUserBlocked):
		writeError(w, http.StatusForbidden, "user is blocked")
	case errors.Is(err, apperror.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	default:
		h.log.Error("request failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

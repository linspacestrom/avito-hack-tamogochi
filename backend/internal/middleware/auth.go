package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dto"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(ctx context.Context, accessToken string) (uuid.UUID, error)
}

type userIDContextKey struct{}

func RequireAuth(
	auth Authenticator,
	log *slog.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w)
				return
			}

			userID, err := auth.Authenticate(r.Context(), parts[1])
			if err != nil {
				writeAuthError(w, log, err)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(uuid.UUID)
	return userID, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	writeAuthResponse(w, http.StatusUnauthorized, "authentication required")
}

func writeAuthError(w http.ResponseWriter, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, apperror.ErrInvalidToken),
		errors.Is(err, apperror.ErrExpiredToken):
		writeAuthResponse(w, http.StatusUnauthorized, "invalid or expired token")
	case errors.Is(err, apperror.ErrUserBlocked):
		writeAuthResponse(w, http.StatusForbidden, "user is blocked")
	default:
		log.Error("authentication failed", slog.Any("error", err))
		writeAuthResponse(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeAuthResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: message,
	})
}

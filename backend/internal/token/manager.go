package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const refreshTokenBytes = 32

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func New(
	secret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) (*Manager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth secret must contain at least 32 bytes")
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, fmt.Errorf("token TTL values must be positive")
	}

	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}, nil
}

func (m *Manager) Issue(userID uuid.UUID) (entity.TokenPair, error) {
	now := m.now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		ID:        uuid.NewString(),
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(m.secret)
	if err != nil {
		return entity.TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}

	refreshBytes := make([]byte, refreshTokenBytes)
	if _, err = rand.Read(refreshBytes); err != nil {
		return entity.TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	return entity.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     base64.RawURLEncoding.EncodeToString(refreshBytes),
		RefreshExpiresAt: now.Add(m.refreshTTL),
	}, nil
}

func (m *Manager) ParseAccess(accessToken string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(
		accessToken,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, apperror.ErrInvalidToken
			}
			return m.secret, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !parsed.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, apperror.ErrExpiredToken
		}
		return uuid.Nil, apperror.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, apperror.ErrInvalidToken
	}

	return userID, nil
}

func (m *Manager) HashRefresh(refreshToken string) []byte {
	hash := sha256.Sum256([]byte(refreshToken))
	return hash[:]
}

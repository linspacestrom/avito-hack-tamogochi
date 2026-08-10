package leaderboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	reader Reader
}

// NewService creates the read API for both leaderboard views.
func NewService(reader Reader) (*Service, error) {
	if reader == nil {
		return nil, fmt.Errorf("%w: reader is required", ErrValidation)
	}
	return &Service{reader: reader}, nil
}

// ListPetLevels returns one deterministic page ordered by pet level.
func (s *Service) ListPetLevels(ctx context.Context, page Page) ([]PetLevelEntry, error) {
	normalized, err := normalizePage(page)
	if err != nil {
		return nil, err
	}
	return s.reader.ListPetLevels(ctx, normalized)
}

// ListGameScores returns one deterministic page ordered by best game score.
func (s *Service) ListGameScores(ctx context.Context, page Page) ([]GameScoreEntry, error) {
	normalized, err := normalizePage(page)
	if err != nil {
		return nil, err
	}
	return s.reader.ListGameScores(ctx, normalized)
}

// GetUserPositions returns the authenticated user's optional position in each view.
func (s *Service) GetUserPositions(ctx context.Context, userID uuid.UUID) (UserPositions, error) {
	if userID == uuid.Nil {
		return UserPositions{}, fmt.Errorf("%w: user ID is required", ErrValidation)
	}
	return s.reader.GetUserPositions(ctx, userID)
}

func normalizePage(page Page) (Page, error) {
	if page.Limit < 0 || page.Limit > MaxPageSize || page.Offset < 0 {
		return Page{}, fmt.Errorf("%w: invalid page", ErrValidation)
	}
	if page.Limit == 0 {
		page.Limit = DefaultPageSize
	}
	return page, nil
}

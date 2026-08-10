package leaderboard

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestServiceNormalizesDefaultPage(t *testing.T) {
	reader := &readerStub{}
	service, err := NewService(reader)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err = service.ListPetLevels(context.Background(), Page{}); err != nil {
		t.Fatalf("ListPetLevels() error = %v", err)
	}
	if reader.petPage != (Page{Limit: DefaultPageSize}) {
		t.Fatalf("pet page = %+v, want default limit", reader.petPage)
	}

	if _, err = service.ListGameScores(context.Background(), Page{Limit: 5, Offset: 10}); err != nil {
		t.Fatalf("ListGameScores() error = %v", err)
	}
	if reader.gamePage != (Page{Limit: 5, Offset: 10}) {
		t.Fatalf("game page = %+v", reader.gamePage)
	}
}

func TestServiceRejectsInvalidPageAndUser(t *testing.T) {
	service, err := NewService(&readerStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	invalidPages := []Page{
		{Limit: -1},
		{Limit: MaxPageSize + 1},
		{Offset: -1},
	}
	for _, page := range invalidPages {
		if _, err = service.ListPetLevels(context.Background(), page); !errors.Is(err, ErrValidation) {
			t.Fatalf("ListPetLevels(%+v) error = %v, want ErrValidation", page, err)
		}
	}
	if _, err = service.GetUserPositions(context.Background(), uuid.Nil); !errors.Is(err, ErrValidation) {
		t.Fatalf("GetUserPositions() error = %v, want ErrValidation", err)
	}
}

type readerStub struct {
	petPage  Page
	gamePage Page
}

func (s *readerStub) ListPetLevels(
	_ context.Context,
	page Page,
) ([]PetLevelEntry, error) {
	s.petPage = page
	return []PetLevelEntry{}, nil
}

func (s *readerStub) ListGameScores(
	_ context.Context,
	page Page,
) ([]GameScoreEntry, error) {
	s.gamePage = page
	return []GameScoreEntry{}, nil
}

func (*readerStub) GetUserPositions(
	context.Context,
	uuid.UUID,
) (UserPositions, error) {
	return UserPositions{}, nil
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/leaderboard"
	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/google/uuid"
)

type LeaderboardRepository struct {
	repo *Repository
}

var _ leaderboard.ProjectionWriter = (*LeaderboardRepository)(nil)
var _ leaderboard.Reader = (*LeaderboardRepository)(nil)
var _ leaderboard.ProjectionEventStore = (*EventRepository)(nil)

// NewLeaderboardRepository creates PostgreSQL persistence for leaderboard
// projections and ranked reads.
func NewLeaderboardRepository(repo *Repository) *LeaderboardRepository {
	return &LeaderboardRepository{repo: repo}
}

// InsertPet initializes the level projection for a user's first pet.
func (r *LeaderboardRepository) InsertPet(
	ctx context.Context,
	pet leaderboard.PetProjection,
) (bool, error) {
	rows, err := db.New(r.repo.GetConn(ctx)).InsertLeaderboardPet(
		ctx,
		db.InsertLeaderboardPetParams{
			UserID:               pet.UserID,
			PetID:                pet.PetID,
			PetName:              pet.PetName,
			Species:              pet.Species,
			LevelReachedPosition: pet.ReachedPosition,
		},
	)
	if err != nil {
		return false, fmt.Errorf("insert leaderboard pet: %w", err)
	}
	return rows == 1, nil
}

// AdvancePetLevel applies a strictly increasing level and its achievement position.
func (r *LeaderboardRepository) AdvancePetLevel(
	ctx context.Context,
	pet leaderboard.PetProjection,
) (bool, error) {
	rows, err := db.New(r.repo.GetConn(ctx)).AdvanceLeaderboardPetLevel(
		ctx,
		db.AdvanceLeaderboardPetLevelParams{
			UserID:               pet.UserID,
			PetID:                pet.PetID,
			Level:                pet.Level,
			LevelReachedPosition: pet.ReachedPosition,
		},
	)
	if err != nil {
		return false, fmt.Errorf("advance leaderboard pet level: %w", err)
	}
	return rows == 1, nil
}

// UpsertGameScore stores a score only when it improves the user's current record.
func (r *LeaderboardRepository) UpsertGameScore(
	ctx context.Context,
	score leaderboard.GameScoreProjection,
) (bool, error) {
	rows, err := db.New(r.repo.GetConn(ctx)).UpsertLeaderboardGameScore(
		ctx,
		db.UpsertLeaderboardGameScoreParams{
			UserID:           score.UserID,
			SessionID:        score.SessionID,
			BestScore:        score.BestScore,
			AchievedPosition: score.AchievedPosition,
		},
	)
	if err != nil {
		return false, fmt.Errorf("upsert leaderboard game score: %w", err)
	}
	return rows == 1, nil
}

// RecordEventFailure quarantines a validated reference to an unprojectable event.
func (r *LeaderboardRepository) RecordEventFailure(
	ctx context.Context,
	failure leaderboard.EventFailure,
) error {
	rows, err := db.New(r.repo.GetConn(ctx)).RecordLeaderboardEventFailure(
		ctx,
		db.RecordLeaderboardEventFailureParams{
			Reason:         string(failure.Reason),
			EventID:        failure.EventID,
			UserID:         failure.UserID,
			GlobalPosition: failure.GlobalPosition,
			EventType:      string(failure.EventType),
			SchemaVersion:  failure.SchemaVersion,
		},
	)
	if err != nil {
		return fmt.Errorf("record leaderboard event failure: %w", err)
	}
	if rows != 1 {
		return errors.New("leaderboard event failure does not match a stored event")
	}
	return nil
}

// ListPetLevels reads one deterministic page of the pet-level ranking.
func (r *LeaderboardRepository) ListPetLevels(
	ctx context.Context,
	page leaderboard.Page,
) ([]leaderboard.PetLevelEntry, error) {
	rows, err := db.New(r.repo.GetConn(ctx)).ListPetLevelLeaderboard(
		ctx,
		db.ListPetLevelLeaderboardParams{
			PageOffset: page.Offset,
			PageSize:   page.Limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list pet level leaderboard: %w", err)
	}
	entries := make([]leaderboard.PetLevelEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, leaderboard.PetLevelEntry{
			Rank:        row.Rank,
			UserID:      row.UserID,
			DisplayName: row.DisplayName,
			PetID:       row.PetID,
			PetName:     row.PetName,
			Species:     row.Species,
			Level:       row.Level,
		})
	}
	return entries, nil
}

// ListGameScores reads one deterministic page of the game-score ranking.
func (r *LeaderboardRepository) ListGameScores(
	ctx context.Context,
	page leaderboard.Page,
) ([]leaderboard.GameScoreEntry, error) {
	rows, err := db.New(r.repo.GetConn(ctx)).ListGameScoreLeaderboard(
		ctx,
		db.ListGameScoreLeaderboardParams{
			PageOffset: page.Offset,
			PageSize:   page.Limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list game score leaderboard: %w", err)
	}
	entries := make([]leaderboard.GameScoreEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, leaderboard.GameScoreEntry{
			Rank:        row.Rank,
			UserID:      row.UserID,
			DisplayName: row.DisplayName,
			BestScore:   row.BestScore,
		})
	}
	return entries, nil
}

// GetUserPositions reads both optional ranks from one PostgreSQL snapshot.
func (r *LeaderboardRepository) GetUserPositions(
	ctx context.Context,
	userID uuid.UUID,
) (leaderboard.UserPositions, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetUserLeaderboardPositions(ctx, userID)
	if err != nil {
		return leaderboard.UserPositions{}, fmt.Errorf("get user leaderboard positions: %w", err)
	}
	return leaderboard.UserPositions{
		PetLevelRank:  optionalRank(row.PetLevelRank),
		GameScoreRank: optionalRank(row.GameScoreRank),
	}, nil
}

func optionalRank(rank int64) *int64 {
	if rank == 0 {
		return nil
	}
	return &rank
}

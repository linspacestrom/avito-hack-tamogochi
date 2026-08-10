package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/dailysummary"
	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type DailySummaryRepository struct {
	repo *Repository
}

var _ dailysummary.CheckpointRepository = (*DailySummaryRepository)(nil)
var _ dailysummary.EventFailureRepository = (*DailySummaryRepository)(nil)
var _ dailysummary.EventReader = (*EventRepository)(nil)

// NewDailySummaryRepository creates PostgreSQL persistence for checkpoints and
// quarantined Daily Summary events.
func NewDailySummaryRepository(repo *Repository) *DailySummaryRepository {
	return &DailySummaryRepository{repo: repo}
}

// InsertCheckpoint initializes a user's checkpoint without overwriting an
// existing one. The returned value reports whether a row was inserted.
func (r *DailySummaryRepository) InsertCheckpoint(
	ctx context.Context,
	checkpoint dailysummary.Checkpoint,
) (bool, error) {
	rows, err := db.New(r.repo.GetConn(ctx)).InsertDailySummaryCheckpoint(
		ctx,
		db.InsertDailySummaryCheckpointParams{
			UserID: checkpoint.UserID,
			LastCheckInAt: pgtype.Timestamptz{
				Time:  checkpoint.LastCheckInAt,
				Valid: true,
			},
			LastEventPosition: checkpoint.LastEventPosition,
		},
	)
	if err != nil {
		return false, fmt.Errorf("insert daily summary checkpoint: %w", err)
	}

	return rows == 1, nil
}

// GetCheckpointForUpdate loads and locks a user's checkpoint for the duration of
// the current transaction, serializing concurrent check-ins.
func (r *DailySummaryRepository) GetCheckpointForUpdate(
	ctx context.Context,
	userID uuid.UUID,
) (dailysummary.Checkpoint, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetDailySummaryCheckpointForUpdate(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dailysummary.Checkpoint{}, dailysummary.ErrCheckpointNotFound
	}
	if err != nil {
		return dailysummary.Checkpoint{}, fmt.Errorf("get daily summary checkpoint: %w", err)
	}
	if !row.LastCheckInAt.Valid {
		return dailysummary.Checkpoint{}, errors.New("daily summary checkpoint has invalid timestamp")
	}

	return dailysummary.Checkpoint{
		UserID:            row.UserID,
		LastCheckInAt:     row.LastCheckInAt.Time,
		LastEventPosition: row.LastEventPosition,
	}, nil
}

// UpdateCheckpoint advances the last check-in time and processed event position.
func (r *DailySummaryRepository) UpdateCheckpoint(
	ctx context.Context,
	checkpoint dailysummary.Checkpoint,
) error {
	rows, err := db.New(r.repo.GetConn(ctx)).UpdateDailySummaryCheckpoint(
		ctx,
		db.UpdateDailySummaryCheckpointParams{
			UserID: checkpoint.UserID,
			LastCheckInAt: pgtype.Timestamptz{
				Time:  checkpoint.LastCheckInAt,
				Valid: true,
			},
			LastEventPosition: checkpoint.LastEventPosition,
		},
	)
	if err != nil {
		return fmt.Errorf("update daily summary checkpoint: %w", err)
	}
	if rows != 1 {
		return dailysummary.ErrCheckpointNotFound
	}

	return nil
}

// RecordEventFailure stores a validated reference to an event that could not be
// included in a summary, allowing the checkpoint to advance safely.
func (r *DailySummaryRepository) RecordEventFailure(
	ctx context.Context,
	failure dailysummary.EventFailure,
) error {
	recorded, err := db.New(r.repo.GetConn(ctx)).RecordDailySummaryEventFailure(
		ctx,
		db.RecordDailySummaryEventFailureParams{
			UserID:         failure.UserID,
			EventID:        failure.EventID,
			GlobalPosition: failure.GlobalPosition,
			EventType:      string(failure.EventType),
			SchemaVersion:  failure.SchemaVersion,
			Reason:         string(failure.Reason),
		},
	)
	if err != nil {
		return fmt.Errorf("record daily summary event failure: %w", err)
	}
	if !recorded {
		return errors.New("daily summary event failure does not match a stored event")
	}

	return nil
}

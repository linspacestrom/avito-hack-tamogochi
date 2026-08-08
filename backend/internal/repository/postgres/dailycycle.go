package postgres

import (
	"context"
	"errors"
	"time"

	db "github.com/NBx03/avito-hack-tamagotchi/backend/internal/repository/postgres/sqlc"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/jackc/pgx/v5"
)

// GetProgress returns nil, nil (not an error) if the user has never claimed a daily reward —
// that's the normal state for anyone who hasn't opened the daily-reward screen yet.
func (r *RewardsRepository) GetProgress(
	ctx context.Context,
	userID string,
) (*rewards.UserDailyRewardProgress, error) {
	id, err := stringToUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := db.New(r.repo.GetConn(ctx)).GetDailyRewardProgress(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := dailyRewardProgressFromDB(row)
	return &result, nil
}

func (r *RewardsRepository) GetRewardForDay(
	ctx context.Context,
	dayNumber int,
) (*rewards.RewardDefinition, error) {
	row, err := db.New(r.repo.GetConn(ctx)).GetRewardForDay(ctx, int32(dayNumber))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, rewards.ErrRewardNotFound
	}
	if err != nil {
		return nil, err
	}

	result := rewardDefinitionFromDB(row)
	return &result, nil
}

func (r *RewardsRepository) UpsertProgress(
	ctx context.Context,
	progress rewards.UserDailyRewardProgress,
) error {
	userID, err := stringToUUID(progress.UserID)
	if err != nil {
		return err
	}

	return db.New(r.repo.GetConn(ctx)).UpsertDailyRewardProgress(ctx, db.UpsertDailyRewardProgressParams{
		UserID:         userID,
		CurrentDay:     int32(progress.CurrentDay),
		CycleStartedAt: timeToTimestamptz(progress.CycleStartedAt),
		LastClaimedAt:  timePtrToTimestamptz(progress.LastClaimedAt),
	})
}

func (r *RewardsRepository) LogClaim(
	ctx context.Context,
	userID string,
	dayNumber int,
	rewardDefinitionID string,
	claimedAt time.Time,
) error {
	uID, err := stringToUUID(userID)
	if err != nil {
		return err
	}
	rewardID, err := stringToUUID(rewardDefinitionID)
	if err != nil {
		return err
	}

	return db.New(r.repo.GetConn(ctx)).LogDailyRewardClaim(ctx, db.LogDailyRewardClaimParams{
		UserID:             uID,
		DayNumber:          int32(dayNumber),
		RewardDefinitionID: rewardID,
		ClaimedAt:          timeToTimestamptz(claimedAt),
	})
}

func dailyRewardProgressFromDB(p db.UserDailyRewardProgress) rewards.UserDailyRewardProgress {
	return rewards.UserDailyRewardProgress{
		UserID:         uuidToString(p.UserID),
		CurrentDay:     int(p.CurrentDay),
		CycleStartedAt: p.CycleStartedAt.Time,
		LastClaimedAt:  timestamptzToTimePtr(p.LastClaimedAt),
	}
}

package postgres

import (
	"context"
	"errors"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/jackc/pgx/v5"
)

// GetProgress returns nil, nil (not an error) if the user has never claimed a daily reward —
// that's the normal state for anyone who hasn't opened the daily-reward screen yet.
func (r *RewardsRepository) GetProgress(
	ctx context.Context,
	userID string,
) (*rewards.UserDailyRewardProgress, error) {
	const query = `
		SELECT user_id::text, current_day, cycle_started_at, last_claimed_at
		FROM user_daily_reward_progress
		WHERE user_id = $1`

	var p rewards.UserDailyRewardProgress
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&p.UserID,
		&p.CurrentDay,
		&p.CycleStartedAt,
		&p.LastClaimedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *RewardsRepository) GetRewardForDay(
	ctx context.Context,
	dayNumber int,
) (*rewards.RewardDefinition, error) {
	const query = `
		SELECT rd.id::text, rd.code, rd.title, rd.description, rd.required_level,
			rd.validity_days, rd.is_active, rd.reward_type, rd.value
		FROM daily_reward_cycle drc
		JOIN reward_definitions rd ON rd.id = drc.reward_definition_id
		WHERE drc.day_number = $1`

	var def rewards.RewardDefinition
	err := r.db.QueryRow(ctx, query, dayNumber).Scan(
		&def.ID,
		&def.Code,
		&def.Title,
		&def.Description,
		&def.RequiredLevel,
		&def.ValidityDays,
		&def.IsActive,
		&def.RewardType,
		&def.Value,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, rewards.ErrRewardNotFound
	}
	if err != nil {
		return nil, err
	}

	return &def, nil
}

func (r *RewardsRepository) UpsertProgress(
	ctx context.Context,
	progress rewards.UserDailyRewardProgress,
) error {
	const query = `
		INSERT INTO user_daily_reward_progress (user_id, current_day, cycle_started_at, last_claimed_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			current_day = EXCLUDED.current_day,
			cycle_started_at = EXCLUDED.cycle_started_at,
			last_claimed_at = EXCLUDED.last_claimed_at`

	_, err := r.db.Exec(ctx, query,
		progress.UserID,
		progress.CurrentDay,
		progress.CycleStartedAt,
		progress.LastClaimedAt,
	)
	return err
}

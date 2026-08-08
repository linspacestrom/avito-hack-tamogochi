package postgres

import (
	"context"
	"errors"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/rewards"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolationCode = "23505"

// RewardsRepository is a standalone implementation using raw pgx queries — it is not yet
// wired into the shared Repository struct (that struct is being rebuilt in PR #10) and does
// not yet use sqlc (sqlc.yaml lands in the same PR). Once #10 merges, this should move onto
// the same sqlc-generated pattern as the rest of the repositories, and get a Rewards field
// on the shared Repository struct.
type RewardsRepository struct {
	db *pgxpool.Pool
}

func NewRewardsRepository(db *pgxpool.Pool) *RewardsRepository {
	return &RewardsRepository{db: db}
}

func (r *RewardsRepository) GetRewardDefinitionByCode(
	ctx context.Context,
	code string,
) (*rewards.RewardDefinition, error) {
	const query = `
		SELECT id::text, code, title, description, required_level, validity_days, is_active,
			reward_type, value
		FROM reward_definitions
		WHERE code = $1`

	var def rewards.RewardDefinition
	err := r.db.QueryRow(ctx, query, code).Scan(
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

func (r *RewardsRepository) CreateUserReward(
	ctx context.Context,
	reward rewards.UserReward,
) (*rewards.UserReward, error) {
	const query = `
		INSERT INTO user_rewards (user_id, reward_definition_id, source_event_id, status, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, issued_at`

	err := r.db.QueryRow(ctx, query,
		reward.UserID,
		reward.RewardDefinitionID,
		reward.SourceEventID,
		reward.Status,
		reward.ExpiresAt,
	).Scan(&reward.ID, &reward.IssuedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, rewards.ErrAlreadyIssued
		}
		return nil, err
	}

	return &reward, nil
}

func (r *RewardsRepository) ListUserRewards(
	ctx context.Context,
	userID string,
) ([]rewards.UserReward, error) {
	const query = `
		SELECT id::text, user_id::text, reward_definition_id::text, source_event_id::text,
			status, issued_at, expires_at, redeemed_at
		FROM user_rewards
		WHERE user_id = $1
		ORDER BY issued_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []rewards.UserReward
	for rows.Next() {
		var ur rewards.UserReward
		if err := rows.Scan(
			&ur.ID,
			&ur.UserID,
			&ur.RewardDefinitionID,
			&ur.SourceEventID,
			&ur.Status,
			&ur.IssuedAt,
			&ur.ExpiresAt,
			&ur.RedeemedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, ur)
	}

	return result, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}

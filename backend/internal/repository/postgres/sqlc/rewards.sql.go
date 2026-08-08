// ⚠️ Hand-authored, not real sqlc output — see the note in models.go.
// source: rewards.sql

package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const getRewardDefinitionByCode = `-- name: GetRewardDefinitionByCode :one
SELECT id, code, title, description, required_level, validity_days, is_active, reward_type, value
FROM reward_definitions
WHERE code = $1
`

func (q *Queries) GetRewardDefinitionByCode(ctx context.Context, code string) (RewardDefinition, error) {
	row := q.db.QueryRow(ctx, getRewardDefinitionByCode, code)
	var i RewardDefinition
	err := row.Scan(
		&i.ID,
		&i.Code,
		&i.Title,
		&i.Description,
		&i.RequiredLevel,
		&i.ValidityDays,
		&i.IsActive,
		&i.RewardType,
		&i.Value,
	)
	return i, err
}

const createUserReward = `-- name: CreateUserReward :one
INSERT INTO user_rewards (user_id, reward_definition_id, source_event_id, status, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, reward_definition_id, source_event_id, status, issued_at, expires_at, redeemed_at
`

type CreateUserRewardParams struct {
	UserID             uuid.UUID
	RewardDefinitionID uuid.UUID
	SourceEventID      pgtype.UUID
	Status             string
	ExpiresAt          pgtype.Timestamptz
}

func (q *Queries) CreateUserReward(ctx context.Context, arg CreateUserRewardParams) (UserReward, error) {
	row := q.db.QueryRow(ctx, createUserReward,
		arg.UserID,
		arg.RewardDefinitionID,
		arg.SourceEventID,
		arg.Status,
		arg.ExpiresAt,
	)
	var i UserReward
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.RewardDefinitionID,
		&i.SourceEventID,
		&i.Status,
		&i.IssuedAt,
		&i.ExpiresAt,
		&i.RedeemedAt,
	)
	return i, err
}

const listUserRewards = `-- name: ListUserRewards :many
SELECT id, user_id, reward_definition_id, source_event_id, status, issued_at, expires_at, redeemed_at
FROM user_rewards
WHERE user_id = $1
ORDER BY issued_at DESC
`

func (q *Queries) ListUserRewards(ctx context.Context, userID uuid.UUID) ([]UserReward, error) {
	rows, err := q.db.Query(ctx, listUserRewards, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []UserReward{}
	for rows.Next() {
		var i UserReward
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.RewardDefinitionID,
			&i.SourceEventID,
			&i.Status,
			&i.IssuedAt,
			&i.ExpiresAt,
			&i.RedeemedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

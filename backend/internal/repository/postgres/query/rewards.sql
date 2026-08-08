-- name: GetRewardDefinitionByCode :one
SELECT id, code, title, description, required_level, validity_days, is_active, reward_type, value
FROM reward_definitions
WHERE code = $1;

-- name: CreateUserReward :one
INSERT INTO user_rewards (user_id, reward_definition_id, source_event_id, status, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, reward_definition_id, source_event_id, status, issued_at, expires_at, redeemed_at;

-- name: ListUserRewards :many
SELECT id, user_id, reward_definition_id, source_event_id, status, issued_at, expires_at, redeemed_at
FROM user_rewards
WHERE user_id = $1
ORDER BY issued_at DESC;

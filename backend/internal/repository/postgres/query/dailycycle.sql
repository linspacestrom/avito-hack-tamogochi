-- name: GetDailyRewardProgress :one
SELECT user_id, current_day, cycle_started_at, last_claimed_at
FROM user_daily_reward_progress
WHERE user_id = $1;

-- name: GetRewardForDay :one
SELECT rd.id, rd.code, rd.title, rd.description, rd.required_level, rd.validity_days,
    rd.is_active, rd.reward_type, rd.value
FROM daily_reward_cycle drc
JOIN reward_definitions rd ON rd.id = drc.reward_definition_id
WHERE drc.day_number = $1;

-- name: UpsertDailyRewardProgress :exec
INSERT INTO user_daily_reward_progress (user_id, current_day, cycle_started_at, last_claimed_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    current_day = EXCLUDED.current_day,
    cycle_started_at = EXCLUDED.cycle_started_at,
    last_claimed_at = EXCLUDED.last_claimed_at;

-- name: LogDailyRewardClaim :exec
INSERT INTO daily_reward_claim_log (user_id, day_number, reward_definition_id, claimed_at)
VALUES ($1, $2, $3, $4);

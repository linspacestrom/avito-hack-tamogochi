-- name: InsertDailySummaryCheckpoint :execrows
INSERT INTO daily_summary_checkpoints (
    user_id,
    last_check_in_at,
    last_event_position
)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO NOTHING;

-- name: GetDailySummaryCheckpointForUpdate :one
SELECT user_id, last_check_in_at, last_event_position
FROM daily_summary_checkpoints
WHERE user_id = $1
FOR UPDATE;

-- name: UpdateDailySummaryCheckpoint :execrows
UPDATE daily_summary_checkpoints
SET last_check_in_at = $2,
    last_event_position = $3,
    updated_at = now()
WHERE user_id = $1;

-- name: RecordDailySummaryEventFailure :one
SELECT app_api.record_daily_summary_event_failure(
    sqlc.arg(user_id)::UUID,
    sqlc.arg(event_id)::UUID,
    sqlc.arg(global_position)::BIGINT,
    sqlc.arg(event_type)::TEXT,
    sqlc.arg(schema_version)::INTEGER,
    sqlc.arg(reason)::TEXT
);

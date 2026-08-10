-- Serializes workers that update the same projection and checkpoint.
-- The fixed hash seed keeps the same projection name mapped to one 64-bit lock key.
-- name: LockProjection :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(projection_name)::TEXT, 0));

-- Returns the last event position processed by a projection.
-- name: GetProjectionCheckpoint :one
SELECT COALESCE((
    SELECT last_position
    FROM projection_checkpoints
    WHERE projection_name = $1
), 0)::BIGINT AS last_position;

-- Advances a projection checkpoint after its read model is updated.
-- name: SaveProjectionCheckpoint :exec
SELECT app_api.save_projection_checkpoint(
    sqlc.arg(projection_name)::TEXT,
    sqlc.arg(last_position)::BIGINT
);

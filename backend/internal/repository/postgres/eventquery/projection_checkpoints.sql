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

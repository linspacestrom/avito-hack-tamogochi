-- name: CreateRefreshSession :one
INSERT INTO refresh_sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, created_at;

-- name: GetRefreshSessionByTokenHash :one
SELECT id, user_id, token_hash, expires_at, created_at
FROM refresh_sessions
WHERE token_hash = $1;

-- name: DeleteRefreshSession :execrows
DELETE FROM refresh_sessions
WHERE token_hash = $1;
